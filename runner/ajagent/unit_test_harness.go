package ajagent

import (
	"local/runner/utils"
	"net"
	"os"
	"path/filepath"
	"testing"
)

type TestHarness struct {
	SocketPath  string
	TestsetPath string
	Listener    net.Listener
	TestSpec    utils.AgentExecSpec
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

func (th *TestHarness) CloseTestHarness() {
	if th.Listener != nil {
		_ = th.Listener.Close()
	}
	_ = os.Remove(th.SocketPath)
}
