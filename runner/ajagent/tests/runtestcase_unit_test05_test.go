package ajagent

import (
	"encoding/json"
	"fmt"
	"testing"
	"utils"
)

func Test_runTestCase_RE_IllegalInstruction(t *testing.T) {
	th := NewTestHarness(t)
	th.InitHarnessTestSpec()
	th.TestSpec.CompileArgs[6] = "artifacts/main7.cpp"

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

	th.connect(t)
	th.compile(t)
	th.run(t)

	<-testServerDone
}
