package internal

import (
	"fmt"
	"os/exec"
	"syscall"
	"utils"
)

/*
signalHandler() inspects process signals and maps them to appropriate judge status codes.
Returns (status, details, isSignal)
Useful for debugging purposes
*/
func signalHandler(
	spec utils.AgentExecSpec, cmd *exec.Cmd, err error,
) (status string, details string, isSignal bool) {
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

		case syscall.SIGKILL:

			var peakMemBytes uint64
			if sysUsage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
				peakMemBytes = uint64(sysUsage.Maxrss) * 1024
			}
			// check if SIGKILL was caused by exceeding the allowed memory ceiling (using ~95% threshold to account for OS buffer/cgroup padding)
			memoryThreshold := float64(spec.MemoryLimitMB) * 0.95

			if float64(peakMemBytes) >= memoryThreshold {
				return verdictMLE,
					fmt.Sprintf("Memory Used: [ %d MB / %d MB]", peakMemBytes/(1024*1024), spec.MemoryLimitMB),
					true
			}
			// if memory was normal, it was terminated by agent's per-testcase timeout timer
			return verdictTLE, "Time Limit Exceeded", true

		default:
			return verdictRE, fmt.Sprintf("Terminated by signal %v", waitStatus.Signal()), true
		}
	}

	return verdictRE,
		fmt.Sprintf("Exited with status %d", waitStatus.ExitStatus()),
		false
}
