package ajagent

import (
	"testing"
)

// HaltOnFirstError = true
func Test_RunnerAgent_Integration_HFE(t *testing.T) {
	// socketPath := "artifacts/agent.sock"
	// t.Setenv("STREAM_SOCKET_PATH", socketPath)
	// t.Setenv("CONFIG_PATH", "artifacts/execspec3.json")
	// t.Setenv("TESTSET_PATH", "artifacts/ts001")

	// _ = os.Remove(socketPath)

	// listener, err := net.Listen("unix", socketPath)
	// if err != nil {
	// 	t.Fatalf("Failed to create socket listener: %v", err)
	// }
	// defer listener.Close()

	// serverDone := make(chan struct{})
	// counter := 0

	// go func() {
	// 	defer close(serverDone)

	// 	for {
	// 		// Keep accepting connections if ajagent reconnects per event
	// 		conn, err := listener.Accept()
	// 		if err != nil {
	// 			return // Listener closed by main thread
	// 		}

	// 		decoder := json.NewDecoder(conn)
	// 		for {
	// 			var event utils.AgentEventSpec
	// 			if err := decoder.Decode(&event); err != nil {
	// 				// Socket closed by client or EOF, break inner loop to accept next connection if any
	// 				break
	// 			}
	// 			counter++
	// 			if counter == 4 {
	// 				conn.Close()
	// 				return // All 4 events received!
	// 			}
	// 		}
	// 		conn.Close()
	// 	}
	// }()

	// // Run agent synchronous execution
	// go ajagent.RunnerAgent()

	// // Wait for server goroutine to finish capturing 4 events
	// <-serverDone

	// if counter != 4 {
	// 	t.Errorf("Supposed to stop at 4th test, but got counter = %d", counter)
	// }
}
