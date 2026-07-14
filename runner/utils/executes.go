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
	Args        []string

	// environment
	CodePathHost         string
	CodePathContainer    string
	TestsetPathHost      string
	TestsetPathContainer string
	Env                  map[string]string
	OutStreamQueueName   string
	ErrStreamQueueName   string

	// rules
	MemoryLimitMB  uint64
	PidLimit       int64
	CpuQuota       float64
	NoNewPrivilege bool
	ReadOnlyRootfs bool
	Timeoutsec     uint32
}

// execution specification for in-container agent
type AgentExecSpec struct {
	// execution
	HasCompilePhase bool
	CompileArgs     []string
	RunArgs         []string
	TestsetPath     string

	// resource
	LogLimitKB uint64
	Timeoutsec uint32
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
