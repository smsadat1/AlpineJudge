package ajagent

import (
	"encoding/json"
	"local/runner/ajagent"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"utils"
)

type TestHarness struct {
	SocketPath     string
	TestsetPath    string
	Listener       net.Listener
	TestSpec       utils.AgentExecSpec
	streamEnconder *json.Encoder
}

/*
creates directories, cleans up old artifacts, sets up environment
variables, and starts a Unix domain socket listener bound to the current test lifecycle.
*/
func NewTestHarness(t *testing.T) *TestHarness {
	t.Helper() // Marks this function as a test helper so log line numbers point to your actual test

	artifactsDir := "artifacts"
	sockPath := filepath.Join(artifactsDir, "agent.sock")
	testsetPath := filepath.Join(artifactsDir, "ts001")

	// 1. Set environment variables for the test process
	t.Setenv("STREAM_SOCKET_PATH", sockPath)
	t.Setenv("TESTSET_PATH", testsetPath)

	// 2. Ensure clean directories
	if err := os.MkdirAll(testsetPath, 0755); err != nil {
		t.Fatalf("Harness: failed to create artifacts dir: %v", err)
	}

	// 3. Remove stale socket file if left behind from a previous run
	_ = os.Remove(sockPath)

	// 4. Start the socket listener
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("Harness: failed to create socket listener: %v", err)
	}

	h := &TestHarness{
		SocketPath:  sockPath,
		TestsetPath: testsetPath,
		Listener:    listener,
	}

	// 5. Register automatic teardown with testing.T
	// t.Cleanup runs automatically when the test (and all its subtests) completes!
	t.Cleanup(func() {
		h.CloseTestHarness()
	})

	return h
}

func (th *TestHarness) InitHarnessTestSpec() {
	th.TestSpec = utils.AgentExecSpec{
		SubmissionID:     "sub001",
		HaltOnFirstError: false,
		LogLimitKB:       1,
		TimeoutSec:       45,
		TestSetPath:      "artifacts/ts001",
		CompileArgs:      []string{"/usr/bin/g++", "-std=c++17", "-Wall", "-Wextra", "-o", "artifacts/main", ""},
		RunArgs:          []string{"./artifacts/main"},
	}
}

func (th *TestHarness) assert(t *testing.T, expected string, recieved string) {
	if expected != recieved {
		t.Errorf("Expected: %v | Received: %v", expected, recieved)
	}
}

func (th *TestHarness) connect(t *testing.T) {
	t.Helper()
	// find & connect to event stream socket
	testStreamConn, err := net.Dial("unix", os.Getenv("STREAM_SOCKET_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer testStreamConn.Close()

	// an encoder to auto append newlines
	th.streamEnconder = json.NewEncoder(testStreamConn)

}

func (th *TestHarness) compile(t *testing.T) {

	t.Helper()

	if len(th.TestSpec.CompileArgs) > 0 {
		cmd := exec.Command(th.TestSpec.CompileArgs[0], th.TestSpec.CompileArgs[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Compilation failed: %v\nOutput: %s", err, string(output))
		}
	}
}

func (th *TestHarness) run(t *testing.T) {

	t.Helper()

	entries, err := os.ReadDir(os.Getenv("TESTSET_PATH"))
	if err != nil {
		t.Fatal(err)
	}

	for _, ts := range entries {

		if !ts.IsDir() {
			continue
		}

		testcaseDir := filepath.Join(th.TestSpec.TestSetPath, ts.Name())
		inputPath := filepath.Join(testcaseDir, "in.txt")
		expectedPath := filepath.Join(testcaseDir, "out.txt")

		eventStatus := ajagent.RunTestCase(th.TestSpec, inputPath, expectedPath)

		// stream events
		if err := th.streamEnconder.Encode(eventStatus); err != nil {
			log.Printf("Failed to write to event stream pipeline: %v", err)
			break
		}

		// continue unless HaltOnFirstError is True & no major errors (OLE IE)
		if th.TestSpec.HaltOnFirstError && eventStatus.EvenType == "ERROR" {
			break
		}
	}
}

func (th *TestHarness) CloseTestHarness() {

	if th.Listener != nil {
		_ = th.Listener.Close()
	}
	_ = os.Remove(th.SocketPath)
}
