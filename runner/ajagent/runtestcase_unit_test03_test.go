package ajagent

import (
	"encoding/json"
	"fmt"
	"local/runner/utils"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func Test_runTestCase_IE_MissingRunArgs(t *testing.T) {
	th := NewTestHarness(t)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		// accept the incoming connection from your agent runner
		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Host reads events off the socket
		decoder := json.NewDecoder(conn)
		for {
			var event utils.AgentEventSpec
			if err := decoder.Decode(&event); err != nil {
				break // Connection closed or EOF reached
			}
			fmt.Printf("--> [SOCKET EVENT STREAM] Type: %-7s | Status: %-20s | Detail: %s\n",
				event.EvenType, event.Status, event.Details)
			// t.Logf("[HOST RECEIVED EVENT] Type: %s | Status: %s | Detail: %s", event.EvenType, event.Status, event.Details)
		}
	}()

	testSpec := utils.AgentExecSpec{
		SubmissionID:     "sub001",
		HaltOnFirstError: false,
		LogLimitKB:       1,
		TimeoutSec:       45,
		TestSetPath:      "artifacts/ts001",
		CompileArgs:      []string{"/usr/bin/g++", "-std=c++17", "-Wall", "-Wextra", "-o", "artifacts/main", "artifacts/main3.cpp"},
		RunArgs:          []string{""},
	}

	// find & connect to event stream socket
	testStreamConn, err := net.Dial("unix", os.Getenv("STREAM_SOCKET_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer testStreamConn.Close()

	// an encoder to auto append newlines
	streamEnconder := json.NewEncoder(testStreamConn)

	entries, err := os.ReadDir(os.Getenv("TESTSET_PATH"))
	if err != nil {
		log.Fatal(err)
	}

	// compilation step
	if len(testSpec.CompileArgs) > 0 {
		cmd := exec.Command(testSpec.CompileArgs[0], testSpec.CompileArgs[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Compilation failed: %v\nOutput: %s", err, string(output))
		}
	}

	// 8. Run the program & iterate over given testset
	for _, ts := range entries {

		if !ts.IsDir() {
			continue
		}

		testcaseDir := filepath.Join(testSpec.TestSetPath, ts.Name())
		inputPath := filepath.Join(testcaseDir, "in.txt")
		expectedPath := filepath.Join(testcaseDir, "out.txt")

		eventStatus := runTestCase(testSpec, inputPath, expectedPath)

		// stream events
		if err := streamEnconder.Encode(eventStatus); err != nil {
			log.Printf("Failed to write to event stream pipeline: %v", err)
			break
		}

		// continue unless HaltOnFirstError is True & no major errors (OLE IE)
		if testSpec.HaltOnFirstError && eventStatus.EvenType == "ERROR" {
			break
		}
	}
}
