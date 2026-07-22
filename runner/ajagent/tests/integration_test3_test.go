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

// HaltOnFirstError = true
func Test_RunnerAgent_Integration_HFE(t *testing.T) {

	t.Setenv("STREAM_SOCKET_PATH", "artifacts/agent.sock")
	t.Setenv("CONFIG_PATH", "artifacts/execspec3.json")
	t.Setenv("TESTSET_PATH", "artifacts/ts001")

	// remove stale socket file if left behind & start listener
	serverDone := make(chan struct{})
	_ = os.Remove(os.Getenv("STREAM_SOCKET_PATH"))

	listener, err := net.Listen("unix", os.Getenv("STREAM_SOCKET_PATH"))
	if err != nil {
		log.Fatalf("Failed to create socket listener: %v", err)
	}

	counter := 0
	go func() {
		defer close(serverDone)

		// accept the incoming connection from your agent runner
		conn, err := listener.Accept()
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
			counter++
		}
		if counter != 5 {
			t.Error("Supposed to stop at 5th test")
		}
	}()

	ajagent.RunnerAgent()
}
