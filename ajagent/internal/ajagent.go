package internal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"utils"
)

/*
ajagent is the "in-container agent" mentioned everywhere in the source.

It goes online during warm container creation phase, stays connected to socket to recieve instruction payload.
Since socket needs to be open from very first stage at container creation phase.
Every containers get assigned with slotID and gets dedicated socket right at that phase.
ajagent connects to that socket injected as env var from OCI specOpts during container creation.
The instruction payload (AgentExecSpec) contains submitted code path and relevant testset path with Compile & Runtime args.
ajagent just reads the instruction payload and follows it.

All the containers get /tmp/ruuner of host mounted as /workspace. But only touches relevant code submissions and testsets decided by upstream.
*/
func RunnerAgent() {

	// 1. Find & connect to event stream socket
	socketPath := os.Getenv("STREAM_SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/tmp/ajagent.sock" // Safe default (this shouldn't happen | if happen then there's more to investigate... maybe upstream)
	}

	streamConn, err := net.Dial("unix", socketPath)
	if err != nil {
		// can't stream event when socket connection itself fails, for this case log will appear in contInfo later after container exits
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

	// 3. Unmrashal execSpec data (the instruction payload)
	var execSpec utils.AgentExecSpec
	if err := json.Unmarshal(payload, &execSpec); err != nil {
		sendEvent(
			streamConn,
			FATAL, verdictIE, "", "",
			fmt.Sprintf("Failed to unmarshal execspec: %v\n", err),
		)
		sendResult(streamConn, verdictIE)
		return
	}

	// 4. Compilation stage (if any | args supplied from instruction payload over unix socket)
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
			sendResult(streamConn, "Compilation error")
			return
		}
	}

	// 5. Find & Read given testset directory (recieved from instruction payload over unix socket)
	testsetPath := execSpec.TestSetPath

	testCount := 0

	for i := 1; ; i++ {
		input := filepath.Join(testsetPath, fmt.Sprintf("%03din.txt", i))
		output := filepath.Join(testsetPath, fmt.Sprintf("%03dout.txt", i))

		_, err := os.Stat(input)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break // exit loop when testset ends
			}

			// real file error
			sendEvent(
				streamConn,
				ERROR,
				verdictIE, "", "",
				fmt.Sprintf("%v", err),
			)
			sendResult(streamConn, verdictIE)
			break // break here to prevent infinite loop
		}

		testCount = i
		runInfo := runTestCase(execSpec, input, output, i)
		sendEvent(
			streamConn,
			INFO,
			runInfo.Verdict, runInfo.Stdout, runInfo.Stderr,
			runInfo.Details,
		)
		if runInfo.Verdict == verdictRE ||
			runInfo.Verdict == verdictTLE ||
			runInfo.Verdict == verdictOLE ||
			runInfo.Verdict == verdictMLE ||
			runInfo.Verdict == verdictIE {

			sendResult(streamConn, runInfo.Verdict)
			break
		}

		// when HaltOnFirstError is true, stop right after recieving first WA
		if execSpec.HaltOnFirstError && runInfo.Verdict != verdictOK {
			sendResult(streamConn, verdictWA)
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
		sendResult(streamConn, verdictIE)
	}

	// everything went well, all tests passed without any issues
	sendResult(streamConn, verdictAC)
}
