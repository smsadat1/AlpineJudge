package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"utils"
)

// in-container agent to run execution
func RunnerAgent() {

	// 1. Find & connect to event stream socket
	socketPath := os.Getenv("STREAM_SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/tmp/ajagent.sock" // Safe default
	}

	streamConn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("Fatal: unable to connect to host stream socket: %v", err)
	}
	defer streamConn.Close()

	// 2. Get instruction data over unix socket
	reader := bufio.NewReader(streamConn)
	payload, err := reader.ReadBytes('\n') // read till delimiter
	if err != nil {
		sendEvent(
			streamConn,
			FATAL, verdictIE, "", "",
			fmt.Sprintf("Failed to recieved execspec: %v\n", err),
		)
		return
	}

	// 3. Unmrashal execSpec data
	var execSpec utils.AgentExecSpec
	if err := json.Unmarshal(payload, &execSpec); err != nil {
		sendEvent(
			streamConn,
			FATAL, verdictIE, "", "",
			fmt.Sprintf("Failed to unmarshal execspec: %v\n", err),
		)
		return
	}

	// 4. Compilation stage (if any)
	if len(execSpec.CompileArgs) > 0 {
		cmd := exec.Command(execSpec.CompileArgs[0], execSpec.CompileArgs[1:]...)
		stdout := &LimitExceededWriter{limit: int64(execSpec.LogLimitKB) * 1000}
		stderr := &LimitExceededWriter{limit: int64(execSpec.LogLimitKB) * 1000}

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Run(); err != nil {
			sendEvent(
				streamConn,
				ERROR, verdictCE,
				stdout.buf.String(), stderr.buf.String(), "",
			)
			return
		}
	}

	// 5. Find & Read given testset directory (Priority: env var TESTSET_PATH -> fallback to execSpec.TestSetPath)
	testsetPath := os.Getenv("TESTSET_PATH")
	if testsetPath == "" {
		testsetPath = execSpec.TestSetPath
	}
	testCount := 0

	for i := 1; ; i++ {
		input := filepath.Join(testsetPath, fmt.Sprintf("%03din.txt", i))
		output := filepath.Join(testsetPath, fmt.Sprintf("%03dout.txt", i))

		if _, err := os.Stat(input); os.IsNotExist(err) {
			break
		}
		testCount = i
		runInfo := runTestCase(execSpec, input, output, i)
		sendEvent(
			streamConn,
			INFO,
			runInfo.Verdict, runInfo.Stdout, runInfo.Stderr,
			runInfo.Details,
		)
		if runInfo.Verdict == verdictRE || runInfo.Verdict == verdictTLE || runInfo.Verdict == verdictOLE {
			break
		}
		if runInfo.Verdict == verdictWA && execSpec.HaltOnFirstError {
			break
		}
	}

	if testCount == 0 {
		sendEvent(
			streamConn,
			ERROR,
			verdictIE, "", "",
			fmt.Sprintf("No valid testcases or in.txt files found in '%s'", testsetPath),
		)
	}
}
