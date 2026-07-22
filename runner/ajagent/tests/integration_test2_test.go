package ajagent

import (
	"encoding/json"
	"local/runner/ajagent"
	"log"
	"net"
	"os"
	"testing"
	"utils"
)

func Test_RunnerAgent_Integration_CE(t *testing.T) {

	t.Setenv("STREAM_SOCKET_PATH", "artifacts/agent.sock")
	t.Setenv("CONFIG_PATH", "artifacts/execspec2.json")
	t.Setenv("TESTSET_PATH", "artifacts/ts001")

	th := NewTestHarness(t)

	// remove stale socket file if left behind & start listener
	serverDone := make(chan struct{})
	_ = os.Remove(os.Getenv("STREAM_SOCKET_PATH"))

	listener, err := net.Listen("unix", os.Getenv("STREAM_SOCKET_PATH"))
	if err != nil {
		log.Fatalf("Failed to create socket listener: %v", err)
	}

	go func() {
		defer close(serverDone)

		// accept the incoming connection from your agent runner
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		counter := 0

		// Host reads events off the socket
		decoder := json.NewDecoder(conn)
		for {
			var event utils.AgentEventSpec
			if err := decoder.Decode(&event); err != nil {
				break // Connection closed or EOF reached
			}
			counter++

			// CE is the second event (comes after PE)
			if counter == 2 {
				th.assert(t, "ERROR", event.EvenType)
				th.assert(t, "Compilation error", event.Status)
			}
		}
	}()

	ajagent.RunnerAgent()
}
