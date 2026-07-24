package ajagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"utils"
)

var (
	OK  = "Running test"          // acceptable
	WA  = "Wrong answer"          // acceptable
	TLE = "Time limit exceeded"   // not acceptable
	MLE = "Memory limit exceeded" // not acceptable
	IE  = "Internal error"        // not acceptable
	CE  = "Compilation error"     // not acceptable
	OLE = "Output limit exceeded" // not acceptable
	RE  = "Runtime error"         // not acceptable
)
var ErrorOLE = errors.New(OLE)

type LimitExceededWriter struct {
	buf          bytes.Buffer
	limit        int64
	limitReached bool
}

func (w *LimitExceededWriter) Write(p []byte) (int, error) {

	// Defensively block mem alloc before it reaches the buffer
	if int64(w.buf.Len())+int64(len(p)) > w.limit {
		// Write only up to the remaining capacity to capture partial logs for debugging
		w.limitReached = true
		remaining := w.limit - int64(w.buf.Len())
		if remaining > 0 {
			w.buf.Write(p[:remaining])
		}
		return len(p), ErrorOLE
	}
	return w.buf.Write(p)
}

func (w *LimitExceededWriter) LimitReached() bool {
	return w.limitReached
}

// signalHandler inspects process signals and maps them to appropriate judge status codes.
// Returns (status, details, isSignal)
func signalHandler(err error) (status string, details string, isSignal bool) {

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return "", "", false
	}

	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return "", "", false // Exited with non-zero status, but not killed by signal
	}

	sig := waitStatus.Signal()

	switch sig {
	case syscall.SIGPIPE:
		return OLE, "Output limit exceeded (SIGPIPE / Broken Pipe)", true
	case syscall.SIGABRT:
		return RE, "Aborted (SIGABRT / Assertion failed)", true
	case syscall.SIGILL:
		return RE, "Illegal instruction error (SIGILL)", true
	case syscall.SIGSEGV:
		return RE, "Segmentation fault (SIGSEGV)", true
	case syscall.SIGFPE:
		return RE, "Floating point exception (SIGFPE / Division by Zero)", true
	default:
		return RE, fmt.Sprintf("Terminated by signal: %v", sig), true
	}

}

func RunTestCase(
	spec utils.AgentExecSpec, inputPath string, expectedPath string, testCount int,
) utils.AgentEventSpec {

	// per testcase context & timeout
	tcCtx, tcCancel := context.WithTimeout(
		context.Background(),
		time.Duration(spec.TimeoutSec)*time.Second,
	)
	defer tcCancel()

	stdin, err := os.Open(inputPath)
	if err != nil {
		return utils.AgentEventSpec{
			EvenType: "ERROR",
			Status:   IE,
			Details:  err.Error(),
		}
	}
	defer stdin.Close()

	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		return utils.AgentEventSpec{
			EvenType: "ERROR",
			Status:   IE,
			Details:  err.Error(),
		}
	}

	if len(spec.RunArgs) == 0 {
		return utils.AgentEventSpec{
			EvenType: "ERROR",
			Status:   IE,
			Details:  "Missing execution arguments in run specifications",
		}
	}

	cmd := exec.CommandContext(tcCtx, spec.RunArgs[0], spec.RunArgs[1:]...)
	cmd.Stdin = stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // Isolate process groups (Linux/Unix)

	stdout := &LimitExceededWriter{limit: int64(spec.LogLimitKB) * 1000}
	stderr := &LimitExceededWriter{limit: int64(spec.LogLimitKB) * 1000}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return utils.AgentEventSpec{
			EvenType: "ERROR",
			Status:   IE,
			Details:  err.Error(),
		}
	}

	// Wait for completion via channel to prevent stream blockages from locking execution loop
	defer func() {
		if cmd.Process != nil {
			// Kill group (-PID)
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()

	// Wait for completion via channel to prevent stream blockages from locking execution loop
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-tcCtx.Done():
		// Force terminate process tree immediately on timeout deadline expiration
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return utils.AgentEventSpec{
			EvenType: "ERROR",
			Status:   TLE,
			Details:  "Process exceeded strict execution runtime limitations",
		}
	case runErr := <-done:
		// First priority: check output buffer cap
		if stdout.LimitReached() {
			return utils.AgentEventSpec{
				EvenType: "ERROR",
				Status:   OLE,
				Details:  "Output limit exceeded: program produced too much output",
			}
		}

		// Second priority: check if program crashed or killed
		if runErr != nil {
			status, details, signal := signalHandler(runErr)
			if signal {
				return utils.AgentEventSpec{
					EvenType: "ERROR",
					Status:   status,
					Details:  details,
				}
			}

			// Fallback for manual non-zero exits (e.g. exit(1) or return 1 from main)
			return utils.AgentEventSpec{
				EvenType: "ERROR",
				Status:   RE, // Runtime errors (like segmentation faults  or borken pipes)
				Details:  fmt.Sprintf("Runtime Exception: %v | Stderr: %s", runErr, stderr.buf.String()),
			}
		}
	}

	// Output evaluation
	actualOut := strings.TrimSpace(stdout.buf.String())
	wantedOut := strings.TrimSpace(string(expected))

	if actualOut != wantedOut {
		return utils.AgentEventSpec{
			EvenType: "ACCEPT",
			Status:   WA,
			Details:  "Output mismatch against expected testcase answers",
		}
	}

	return utils.AgentEventSpec{
		EvenType: "ACCEPT",
		Status:   OK + " " + fmt.Sprint(testCount),
		Details:  "Output matched expectation",
	}
}

// in-container agent to run execution
func RunnerAgent() {

	// 1. Find & connect to event stream socket
	streamConn, err := net.Dial("unix", os.Getenv("STREAM_SOCKET_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer streamConn.Close()

	eventStatus := utils.AgentEventSpec{}

	// an encoder to auto append newlines
	streamEnconder := json.NewEncoder(streamConn)

	// 2. Find & load /workspace/execspec.json to spec
	jsonData, err := os.ReadFile(os.Getenv("CONFIG_PATH"))
	if err != nil {
		// stream events
		eventStatus.EvenType = "ERROR"
		eventStatus.Status = IE
		eventStatus.Details = "Failed to load execspec: " + err.Error() + "\n"
		if err := streamEnconder.Encode(eventStatus); err != nil {
			log.Printf("Failed to write to event stream pipeline: %v", err)
			return
		}
	}

	// 4. Unmarshal from JSON to Spec
	var execSpec utils.AgentExecSpec
	if err := json.Unmarshal(jsonData, &execSpec); err != nil {
		log.Fatalf("Failed to unmarshal execspec %v\n", err)
	}

	// 5. Create context
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(execSpec.TimeoutSec)*time.Second,
	)
	defer cancel()

	// 6. Compilation stage (if any)
	if len(execSpec.CompileArgs) > 0 {
		cmd := exec.CommandContext(ctx, execSpec.CompileArgs[0], execSpec.CompileArgs[1:]...)
		stdout := &LimitExceededWriter{limit: int64(execSpec.LogLimitKB) * 1000}
		stderr := &LimitExceededWriter{limit: int64(execSpec.LogLimitKB) * 1000}

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Run(); err != nil {
			// stream events
			eventStatus.EvenType = "ERROR"
			eventStatus.Status = CE
			eventStatus.Details = "STDOUT: " + stdout.buf.String() + "STDERR: " + stderr.buf.String()
			if err := streamEnconder.Encode(eventStatus); err != nil {
				log.Printf("Failed to write to event stream pipeline: %v", err)
				return
			}
			return
		}
	}

	// 7. Find & Read given testset
	entries, err := os.ReadDir(os.Getenv("TESTSET_PATH"))
	if err != nil {
		log.Fatal(err)
	}

	// 8. Run the program & iterate over given testset
	testCount := 0
	for _, ts := range entries {
		testCount++
		if !ts.IsDir() {
			continue
		}

		testcaseDir := filepath.Join(execSpec.TestSetPath, ts.Name())
		inputPath := filepath.Join(testcaseDir, "in.txt")
		expectedPath := filepath.Join(testcaseDir, "out.txt")

		eventStatus := RunTestCase(execSpec, inputPath, expectedPath, testCount)

		// stream events
		if err := streamEnconder.Encode(eventStatus); err != nil {
			log.Printf("Failed to write to event stream pipeline: %v", err)
			break
		}

		// continue unless HaltOnFirstError is True & no major errors (OLE IE RE)
		if execSpec.HaltOnFirstError && eventStatus.Status != OK {
			break
		}
	}
}
