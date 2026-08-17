package tests

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"utils"
)

/*
	create
	/tmp/{submission_id}/agent.sock /tmp/{submission_id}/ts001/ /tmp/{submission_id}/execspec.json
	use this for runTestCase()
	skip testing memory spam & fork bomb
*/

const CodeOk = `
	#include <iostream>

	int main(int argc, char const *argv[])
	{
	    int a, b = 0;
	    std::cin >> a >> b;
	    std::cout << a+b << std::endl;
	}
`
const CodeWrong = `
	#include <iostream>

	int main(int argc, char const *argv[])
	{
	    int a, b = 0;
	    std::cin >> a >> b;
	    std::cout << a*b << std::endl;
	}
`
const CodeLogSpam = `
	#include <iostream>

	int main(int argc, char const *argv[])
	{
	    for (int i = 0; i < 1000000; i++)
	    {
	        std::cout << "LOL " << std::endl;
	    }
	    return 0;
	}
`
const CodeDivByZero = `
	#include <iostream>

	int main()
	{
	    double meh = 55 / 0;
	    std::cout << meh;
	    return 0;
	}
`
const CodeSegfault = `
	#include <unistd.h>

	int main(int argc, char const *argv[])
	{
	    int *p = NULL;
	    p[1] = 69;
	    return 0;
	}
`
const CodeAbrt = `
	#include <cassert>

	int main(int argc, char const *argv[])
	{
	    int x = 9;
	    assert(x == 10);
	    return 0;
	}
`
const CodeSleep = `
	#include <unistd.h>
	
	int main(int argc, char const *argv[])
	{
	    sleep(100000000);
	    return 0;
	}
	
`
const CodeIll = `
	int main()
	{
	    __builtin_trap();
	    return 0;
	}
`

type TestHarness struct {
	SocketPath     string
	TestsetPath    string
	Listener       net.Listener
	TestSpec       utils.AgentExecSpec
	streamEnconder *json.Encoder
	StreamConn     net.Conn
}

func NewTestHarness(t *testing.T, testcode string) *TestHarness {
	t.Helper()

	artifactsDir := "artifacts"
	// unique sockets to avoid collisions
	sockPath := filepath.Join(t.TempDir(), "agent.sock")
	testsetPath := filepath.Join(artifactsDir, "ts001")

	// set test env vars
	t.Setenv("STREAM_SOCKET_PATH", sockPath)

	if err := os.MkdirAll(testsetPath, 0755); err != nil {
		t.Fatalf("Harness: failed to create artifacts dir: %v", err)
	}

	// create test code file
	if err := os.WriteFile("artifacts/main.cpp", []byte(testcode), 0666); err != nil {
		t.Fatalf("Harness: failed to create test file: %v", err)
	}

	// prepare unix socket (remove stale socket first)
	_ = os.Remove(sockPath)
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("Harness: failed to create socket listener: %v", err)
	}

	h := &TestHarness{
		SocketPath:  sockPath,
		TestsetPath: testsetPath,
		Listener:    listener,
	}

	// auto cleanup
	t.Cleanup(func() {
		h.CloseTestHarness()
	})

	return h
}

func (th *TestHarness) InitHarnessTestSpec() {
	th.TestSpec = utils.AgentExecSpec{
		SubmissionID:     "sub001",
		HaltOnFirstError: true,

		LogLimitKB:    512,
		TimeoutSec:    45,
		MemoryLimitMB: 512,

		TestSetPath: "artifacts/ts001",
		CompileArgs: []string{"/usr/bin/g++", "-std=c++17", "-Wall", "-Wextra", "-o", "artifacts/main", "artifacts/main.cpp"},
		RunArgs:     []string{"./artifacts/main"},
	}
}

// only use for unit test | using it in integration test will cause double socket connection and test will hand
func (th *TestHarness) Connect(t *testing.T) {
	t.Helper()
	// find & connect to event stream socket
	testStreamConn, err := net.Dial("unix", os.Getenv("STREAM_SOCKET_PATH"))
	if err != nil {
		log.Fatal(err)
	}

	// an encoder to auto append newlines
	th.streamEnconder = json.NewEncoder(testStreamConn)
	th.StreamConn = testStreamConn
}

func (th *TestHarness) Compile(t *testing.T) {

	t.Helper()

	if len(th.TestSpec.CompileArgs) > 0 {
		cmd := exec.Command(th.TestSpec.CompileArgs[0], th.TestSpec.CompileArgs[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Compilation failed: %v\nOutput: %s", err, string(output))
		}
	}
}

func (th *TestHarness) Assert(t *testing.T, expected string, recieved string) {
	if expected != recieved {
		t.Errorf("Expected: %v | Received: %v", expected, recieved)
	}
}

func (th *TestHarness) CloseTestHarness() {

	if th.Listener != nil {
		_ = th.Listener.Close()
	}
	_ = os.Remove(th.SocketPath)
}
