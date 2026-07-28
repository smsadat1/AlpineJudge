package internal

import (
	"fmt"
	"os/exec"
	"syscall"
)

// signalHandler inspects process signals and maps them to appropriate judge status codes.
// Returns (status, details, isSignal)
func signalHandler(err error) (status string, details string, isSignal bool) {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return verdictRE, err.Error(), false
	}

	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return verdictRE, err.Error(), false
	}

	if waitStatus.Signaled() {
		switch waitStatus.Signal() {
		case syscall.SIGPIPE:
			return verdictOLE, "Output limit exceeded (SIGPIPE)", true
		case syscall.SIGABRT:
			return verdictRE, "Aborted (SIGABRT)", true
		case syscall.SIGILL:
			return verdictRE, "Illegal instruction (SIGILL)", true
		case syscall.SIGSEGV:
			return verdictRE, "Segmentation fault (SIGSEGV)", true
		case syscall.SIGFPE:
			return verdictRE, "Floating point exception (SIGFPE / Division by Zero)", true
		default:
			return verdictRE, fmt.Sprintf("Terminated by signal %v", waitStatus.Signal()), true
		}
	}

	return verdictRE,
		fmt.Sprintf("Exited with status %d", waitStatus.ExitStatus()),
		false
}
