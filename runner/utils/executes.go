package utils

import (
	"bufio"
	"context"
	"io"
	"shared"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ExecRules struct {
	// system
	ContainerID string
	Image       string
	CompileArgs []string // agent
	RunArgs     []string // agent
	TestID      string

	// environment
	CodePathHost         string            // oci | ok
	CodePathContainer    string            // oci | ok
	TestsetPathHost      string            // oci | ok
	TestsetPathContainer string            // oci | ok
	Env                  map[string]string // only "CONFIG_PATH=/workspace/execspec.json" | ok
	OutStreamQueueName   string
	ErrStreamQueueName   string

	// rules
	MemoryLimitMB  uint64  // oci | ok
	PidLimit       int64   // oci | ok
	CpuQuota       float64 // oci | ok
	NoNewPrivilege bool    // oci | ok
	ReadOnlyRootfs bool    // oci | ok
	Timeoutsec     uint32  // agent + oci (t+extra)
	LogLimitKB     uint32  // agent	| ok
}

// execution specification for in-container agent
type AgentExecSpec struct {
	// resource
	LogLimitKB uint32
	TimeoutSec uint32

	// specifications
	TestSetPath string
	CompileArgs []string
	RunArgs     []string
}

// stream real time logs from container
func StreamContainerLogsToRMQ(
	ctx context.Context, queuename string, reader io.Reader, rmqm shared.RMQManager, msg amqp.Publishing,
) {
	scanner := bufio.NewScanner(reader)
	scanner.Err()
	for scanner.Scan() {
		rmqm.Publish(ctx, queuename, msg)
	}
}
