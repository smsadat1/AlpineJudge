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

func Test_runTestCase_RE_IllegalInstruction(t *testing.T) {
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

	th.InitHarnessTestSpec()
	th.TestSpec.CompileArgs[6] = "artifacts/main7.cpp"

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
	if len(th.TestSpec.CompileArgs) > 0 {
		cmd := exec.Command(th.TestSpec.CompileArgs[0], th.TestSpec.CompileArgs[1:]...)
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

		testcaseDir := filepath.Join(th.TestSpec.TestSetPath, ts.Name())
		inputPath := filepath.Join(testcaseDir, "in.txt")
		expectedPath := filepath.Join(testcaseDir, "out.txt")

		eventStatus := runTestCase(th.TestSpec, inputPath, expectedPath)

		// stream events
		if err := streamEnconder.Encode(eventStatus); err != nil {
			log.Printf("Failed to write to event stream pipeline: %v", err)
			break
		}

		// continue unless HaltOnFirstError is True & no major errors (OLE IE)
		if th.TestSpec.HaltOnFirstError && eventStatus.EvenType == "ERROR" {
			break
		}
	}
}
