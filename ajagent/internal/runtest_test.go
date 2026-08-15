package internal

import (
	"ajagent/tests"
	"encoding/json"
	"testing"
	"utils"
)

func run(t *testing.T, th *tests.TestHarness) {

	t.Helper()
	testsetPath := "../tests/artifacts/ts001"

	for i := 1; ; i++ {
		input, output, exists := findTestcaseFiles(testsetPath, i)
		if !exists {
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

		count := 0
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
			count++
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Output matched expectation", event.Details)
		}
		t.Logf("TOTAL EVENTS ASSERTED: %d", count)
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

		count := 0
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
			count++
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Output mismatch against expected testcase answers", event.Details)
		}
		t.Logf("TOTAL EVENTS ASSERTED: %d", count)
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

		count := 0
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
			count++
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Output limit exceeded", event.Status)
		}
		t.Logf("TOTAL EVENTS ASSERTED: %d", count)
	}()

	th.Compile(t)
	run(t, th)
	th.CloseTestHarness()
	th.StreamConn.Close()
	<-testServerDone
}
