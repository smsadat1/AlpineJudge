package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"utils"
)

/*
The core function that runs the code at runtime and generates most verdicts (AC, WA, RE, TLE, OLE and sometimes IE)
In some cases, ajagent itself generates verdict (CE & IE)
*/
func runTestCase(
	spec utils.AgentExecSpec, inputPath string, expectedPath string, testCount int,
) runtimeInfo {

	// 1. Context per testcase
	tcCtx, tcCancel := context.WithTimeout(
		context.Background(),
		time.Duration(spec.TimeoutSec)*time.Second,
	)
	defer tcCancel()

	// 2. Open input
	stdin, err := os.Open(inputPath)
	if err != nil {
		return runtimeInfo{
			Verdict: verdictIE,
			Stdout:  "", Stderr: "",
			Details: err.Error(),
		}
	}
	defer stdin.Close()

	// 3. Expected result
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		return runtimeInfo{
			Verdict: verdictIE,
			Stdout:  "", Stderr: "",
			Details: err.Error(),
		}
	}

	// 4. Check runtime args existence
	if len(spec.RunArgs) == 0 {
		return runtimeInfo{
			Verdict: verdictIE,
			Stdout:  "", Stderr: "",
			Details: "Missing execution arguments in run specifications",
		}
	}

	// 5. Run test with resource limit enforcements
	cmd := exec.CommandContext(tcCtx, spec.RunArgs[0], spec.RunArgs[1:]...)
	cmd.Stdin = stdin
	cmd.Stdin = stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Isolate process groups (Linux/Unix)
	}
	stdout := &LimitExceededWriter{limit: int64(spec.LogLimitKB) * 1000}
	stderr := &LimitExceededWriter{limit: int64(spec.LogLimitKB) * 1000}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return runtimeInfo{
			Verdict: verdictIE,
			Stdout:  stdout.buf.String(), Stderr: stdout.buf.String(),
			Details: err.Error(),
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
		// TLE is sent when runtime is more than given time limit for test
		return runtimeInfo{
			Verdict: verdictTLE,
			Stdout:  stdout.buf.String(), Stderr: stderr.buf.String(),
			Details: "Code tried to run for too long",
		}
	case runErr := <-done:

		if stdout.LimitReached() || stderr.LimitReached() {
			return runtimeInfo{
				Verdict: verdictOLE,
				Stdout:  stdout.buf.String(), Stderr: stderr.buf.String(),
				Details: "Code attempted to write too much",
			}
		}

		if runErr != nil {
			status, details, signal := signalHandler(spec, cmd, runErr)
			if signal {
				return runtimeInfo{
					Verdict: status,
					Stdout:  stdout.buf.String(), Stderr: stderr.buf.String(),
					Details: fmt.Sprint(details),
				}
			}

			// Fallback for manual non-zero exits (e.g. exit(1) or return 1 from main)
			return runtimeInfo{
				Verdict: verdictRE,
				Stdout:  stdout.buf.String(), Stderr: stderr.buf.String(),
				Details: fmt.Sprint(details),
			}
		}
	}

	// Output evaluation
	actualOut := strings.TrimSpace(stdout.buf.String())
	wantedOut := strings.TrimSpace(string(expected))

	if actualOut != wantedOut {
		return runtimeInfo{
			Verdict: verdictWA + " in test " + fmt.Sprint(testCount),
			Stdout:  stdout.buf.String(), Stderr: stderr.buf.String(),
			Details: "Output mismatch against expected testcase answers",
		}
	}

	return runtimeInfo{
		Verdict: verdictOK + " " + fmt.Sprint(testCount),
		Stdout:  stdout.buf.String(), Stderr: stderr.buf.String(),
		Details: "Output matched expectation",
	}
}
