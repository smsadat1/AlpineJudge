package utils

import (
	"bufio"
	"context"
	"io"
	"log"
	"shared"

	"github.com/containerd/containerd"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EngineDeps struct {
	Client    *containerd.Client
	Rmq       *shared.RMQManager
	S3        *shared.S3Manager
	JobQueue  string
	SSEQueue  string
	Namespace string
}

type ExecRules struct {
	// system
	RunnerID     string
	SubmissionID string
	ContainerID  string
	Image        string
	CompileArgs  []string // agent
	RunArgs      []string // agent
	TestID       string

	// environment
	EventQueueName string

	// rules
	PerTestMemoryLimitMB uint64
	PerTestTimeoutsec    uint32
	PerTestLogLimitKB    uint32
}

// execution specification for in-container agent
type AgentExecSpec struct {

	// system
	SubmissionID     string `json:"submission_id"`
	HaltOnFirstError bool   `json:"halt_on_first_error"`

	// resource
	LogLimitKB    uint32 `json:"log_limit_kb"`
	TimeoutSec    uint32 `json:"timeout_sec"`
	MemoryLimitMB uint64 `json:"memory_limit_mb"`

	// specifications
	TestSetPath string   `json:"testset_path"`
	CompileArgs []string `json:"compile_args"`
	RunArgs     []string `json:"run_args"`
}

// Deprecated: use Event (UESP) instead.
type AgentEventSpec struct {
	EvenType string
	Status   string
	Details  string
}

// ================================ UESP ================================\

/*
UESP (Unified Event Streaming Protocol) defines the communication
protocol between the in-container agent and the Execution Manager
over a Unix domain socket.

Every message transmitted through the socket is represented as an
Event.
*/
type Event struct {
	Type    string
	Status  string
	Stdout  string
	Stderr  string
	Details string
}

// RMQ specific event strcuture
type RMQPayload struct {
	Type    string
	Status  string
	Details string
}

// ========================================================================

// stream real time logs from container
func StreamContainerLogsToRMQ(
	ctx context.Context, queuename string, reader io.Reader, rmqm shared.RMQManager, msg amqp.Publishing,
) {
	scanner := bufio.NewScanner(reader)
	// scanner.Err()
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return // exit streaming immediately if the timeout hit or context cancelled
		default:
			// clone/update the body on each scan line to send the actual stdout log chunk
			clonedMsg := msg
			clonedMsg.Body = []byte(scanner.Text())
			_ = rmqm.Publish(ctx, queuename, clonedMsg)
		}
	}

	// check for errors after the loop ends
	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning input: %v", err)
		return
	}
}
