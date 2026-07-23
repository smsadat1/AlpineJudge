package ajagent

import (
	"encoding/json"
	"local/runner/ajagent"
	"net"
	"os"
	"testing"
	"utils"
)

// HaltOnFirstError = true
func Test_RunnerAgent_Integration_HFE(t *testing.T) {
	socketPath := "artifacts/agent.sock"
	t.Setenv("STREAM_SOCKET_PATH", socketPath)
	t.Setenv("CONFIG_PATH", "artifacts/execspec3.json")
	t.Setenv("TESTSET_PATH", "artifacts/ts001")

	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan struct{})
	counter := 0

	go func() {
		defer close(serverDone) // Signals main thread when connection closes & reads finish

		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(conn)
		for {
			var event utils.AgentEventSpec
			if err := decoder.Decode(&event); err != nil {
				break // EOF when agent closes connection
			}
			counter++
		}
	}()

	// Run the agent
	ajagent.RunnerAgent()

	// 🚨 WAIT FOR SOCKET READER TO FINISH PROCESSING ALL BUFFERS
	<-serverDone

	// Now evaluate counter on the main thread safely
	if counter != 4 {
		t.Errorf("Supposed to stop at 4th test, but got counter = %d", counter)
	}
}
