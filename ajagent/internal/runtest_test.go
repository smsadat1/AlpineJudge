package internal

import (
	"ajagent/tests"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"utils"
)

func run(t *testing.T, th *tests.TestHarness) {

	t.Helper()
	testsetPath := os.Getenv("TESTSET_PATH")

	for i := 1; ; i++ {
		input := filepath.Join(testsetPath, fmt.Sprintf("%03din.txt", i))
		output := filepath.Join(testsetPath, fmt.Sprintf("%03dout.txt", i))

		if _, err := os.Stat(input); os.IsNotExist(err) {
			break
		}

		runInfo := runTestCase(th.TestSpec, input, output, i)
		sendEvent(
			th.StreamConn,
			INFO,
			runInfo.Verdict, runInfo.Stdout, runInfo.Stderr,
			runInfo.Details,
		)
	}

}

func Test_runTestOK(t *testing.T) {

	th := tests.NewTestHarness(t, tests.CodeOk)
	th.InitHarnessTestSpec()

	testServerDone := make(chan struct{})
	th.Connect(t)

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
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Output matched expectation", event.Details)
		}
	}()

	th.Compile(t)
	run(t, th)
	th.CloseTestHarness()
	th.StreamConn.Close()
	<-testServerDone
}

func Test_runTestWA(t *testing.T) {
	th := tests.NewTestHarness(t, tests.CodeWrong)
	th.InitHarnessTestSpec()

	testServerDone := make(chan struct{})
	th.Connect(t)

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
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Output mismatch against expected testcase answers", event.Details)
		}
	}()

	th.Compile(t)
	run(t, th)
	th.CloseTestHarness()
	th.StreamConn.Close()
	<-testServerDone
}

func Test_runTestOLE(t *testing.T) {
	th := tests.NewTestHarness(t, tests.CodeLogSpam)
	th.InitHarnessTestSpec()

	testServerDone := make(chan struct{})
	th.Connect(t)

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
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Output limit exceeded", event.Status)
		}
	}()

	th.Compile(t)
	run(t, th)
	th.CloseTestHarness()
	th.StreamConn.Close()
	<-testServerDone
}
