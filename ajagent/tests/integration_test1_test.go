package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"utils"

	ajagent "ajagent/pkg"
)

func Test_RunnerAgent_Integration_Ok(t *testing.T) {
	t.Setenv("CONFIG_PATH", "artifacts/execspec1.json")
	th := NewTestHarness(t, CodeOk)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
	t.Setenv("CONFIG_PATH", "artifacts/execspec2.json")
	th := NewTestHarness(t, CodeWrong)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
	t.Setenv("CONFIG_PATH", "artifacts/execspec1.json")
	th := NewTestHarness(t, CodeAbrt)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
	t.Setenv("CONFIG_PATH", "artifacts/execspec1.json")
	th := NewTestHarness(t, CodeDivByZero)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
	t.Setenv("CONFIG_PATH", "artifacts/execspec1.json")
	th := NewTestHarness(t, CodeLogSpam)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
	t.Setenv("CONFIG_PATH", "artifacts/execspec1.json")
	th := NewTestHarness(t, CodeSegfault)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
	t.Setenv("CONFIG_PATH", "artifacts/execspec1.json")
	th := NewTestHarness(t, CodeIll)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
	t.Setenv("CONFIG_PATH", "artifacts/execspec1.json")
	th := NewTestHarness(t, CodeSleep)
	testServerDone := make(chan struct{})

	go func() {
		defer close(testServerDone)

		conn, err := th.Listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

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
