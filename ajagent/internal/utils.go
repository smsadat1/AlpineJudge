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
	verdictTLE string = "Time limit exceeded"   // not acceptable
	verdictRE  string = "Runtime error"         // not acceptable
	verdictMLE string = "Memory limit exceeded" // not acceptable
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

// CAN NOT afford to fail (must succeed or as backup contInfo can be investigated later after container exits to find out issues)
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

	// succeed or move on, can't wait during live stream
	_ = streamEnconder.Encode(evt)
}

/*
Convenient wrapper over sendEvent() for result stream
This may look redundant but Type RESULT is what signals client that event stream has ended
So it's sent over unix socket no matter the verdict
Only top level AjAgent should call it, not internal runtime manager runTestCase(). Calling it during runtime defeats it's purpose
*/
func sendResult(conn net.Conn, status string) {
	eventType := "RESULT"
	// no more stdout stderr with result, all stdout stderr sent during execution stream
	sendEvent(conn, eventType, status, "", "", "")
}
