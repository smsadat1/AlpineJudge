package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"utils"

	ajagent "ajagent/pkg"
)

func Test_RunnerAgent_Integration_Ok(t *testing.T) {
	th := NewTestHarness(t, CodeOk)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
		decoder := json.NewDecoder(conn)
		counter := 0
		for {
			counter++
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, fmt.Sprintf("Running test %v", counter), event.Status)
		}
	}()

	ajagent.RunnerAgent()
	<-testServerDone
}

func Test_RunnerAgent_Integration_HFE(t *testing.T) {
	th := NewTestHarness(t, CodeWrong)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	th.TestSpec.HaltOnFirstError = true

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
		decoder := json.NewDecoder(conn)
		counter := 0
		for {
			counter++
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, fmt.Sprintf("Wrong answer in test %v", counter), event.Status)
		}
	}()

	ajagent.RunnerAgent()
	<-testServerDone
}

func Test_RunnerAgent_Integration_Abort(t *testing.T) {
	th := NewTestHarness(t, CodeAbrt)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
		decoder := json.NewDecoder(conn)
		for {
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Runtime error", event.Status)
		}
	}()

	ajagent.RunnerAgent()
	<-testServerDone
}

func Test_RunnerAgent_Integration_FPE(t *testing.T) {
	th := NewTestHarness(t, CodeDivByZero)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
		decoder := json.NewDecoder(conn)
		for {
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Runtime error", event.Status)
		}
	}()

	ajagent.RunnerAgent()
	<-testServerDone
}

func Test_RunnerAgent_Integration_OLE(t *testing.T) {
	th := NewTestHarness(t, CodeLogSpam)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
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

	ajagent.RunnerAgent()
	<-testServerDone
}

func Test_RunnerAgent_Integration_Segfault(t *testing.T) {
	th := NewTestHarness(t, CodeSegfault)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
		decoder := json.NewDecoder(conn)
		for {
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Runtime error", event.Status)
		}
	}()

	ajagent.RunnerAgent()
	<-testServerDone
}

func Test_RunnerAgent_Integration_IllInstruction(t *testing.T) {
	th := NewTestHarness(t, CodeIll)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
		decoder := json.NewDecoder(conn)
		for {
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Runtime error", event.Status)
		}
	}()

	ajagent.RunnerAgent()
	<-testServerDone
}

func Test_RunnerAgent_Integration_TLE(t *testing.T) {
	th := NewTestHarness(t, CodeSleep)
	th.InitHarnessTestSpec()
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// send execspec
		if err := json.NewEncoder(conn).Encode(&th.TestSpec); err != nil {
			t.Errorf("Failed sending execspec JSON: %v", err)
		}

		// read response
		decoder := json.NewDecoder(conn)
		for {
			var event utils.Event
			if err := decoder.Decode(&event); err != nil {
				break
			}
			th.Assert(t, "INFO", event.Type)
			th.Assert(t, "Time limit exceeded", event.Status)
		}
	}()

	ajagent.RunnerAgent()
	<-testServerDone
}
