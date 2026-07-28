package internal

import (
	"bytes"
)

/*
IMPORTANT

DO NOT REFACTOR LimitExceededWriter without understanding its limitations. (Ruined an entire night trying to refactor it)

LimitExceededWriter only limits data that has already reached ajagent from cmd writer.
It cannot prevent log spam still buffered inside the submitted process or the C runtime.

If the submitted program buffers stdout/stderr instead of flushing immediately,
those bytes never reach this writer before process termination.

This behavior is intentional and ajagent might send WA in such case.

The actual protection is that containerd limits how far it can go and
container is destroyed before the remaining buffered logs can be flushed.

Do not assume this writer alone provides complete log-size enforcement.
*/
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
