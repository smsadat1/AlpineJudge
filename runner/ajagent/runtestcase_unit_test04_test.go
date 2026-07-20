package ajagent

import (
	"encoding/json"
	"local/runner/utils"
	"testing"
)

func Test_runTestCase_RE_Segfault(t *testing.T) {
	th := NewTestHarness(t)
	th.InitHarnessTestSpec()
	th.TestSpec.CompileArgs[6] = "artifacts/main6.cpp"

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
			th.assert(t, "ERROR", event.EvenType)
			th.assert(t, "Runtime Errror", event.Status)
			th.assert(t, "Segmentation fault (SIGSEGV)", event.Details)
		}
	}()

	th.connect(t)
	th.compile(t)
	th.run(t)

	<-testServerDone
}
