package internal

import (
	"encoding/json"
	"errors"
	"net"
	"time"
	"utils"
)

const (
	verdictOK  string = "Running test"          // acceptable
	verdictAC  string = "Accepted"              // acceptable
	verdictWA  string = "Wrong answer"          // acceptable
	verdictRE  string = "Runtime error"         // not acceptable
	verdictTLE string = "Time limit exceeded"   // not acceptable
	verdictOLE string = "Output limit exceeded" // not acceptable
	verdictCE  string = "Compilation error"     // not acceptable
	verdictIE  string = "Internal error"        // not acceptable
)

const (
	FATAL string = "FATAL"
	ERROR string = "ERROR"
	INFO  string = "INFO"
)

type runtimeInfo struct {
	Verdict string
	Stdout  string
	Stderr  string
	Details string
}

var ErrorOLE = errors.New(verdictOLE)

// Config defines resource limits per test run
type SandboxLimits struct {
	MaxMemoryBytes uint64
	MaxProcesses   uint64
	Timeout        time.Duration
}

// Default limits for running compiled binary
var DefaultLimits = SandboxLimits{
	MaxMemoryBytes: 128 * 1024 * 1024, // 128 MB
	MaxProcesses:   1,                 // No extra forks
	Timeout:        2 * time.Second,
}

func sendEvent(
	conn net.Conn, evntype string, status string, stdout string, stderr string, msg string,
) {
	// an encoder to auto append newlines
	streamEnconder := json.NewEncoder(conn)

	evt := utils.Event{
		Type:    evntype,
		Status:  status,
		Stdout:  stdout,
		Stderr:  stderr,
		Details: msg,
	}
	_ = streamEnconder.Encode(evt)
}
