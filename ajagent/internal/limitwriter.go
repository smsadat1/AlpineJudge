package internal

import (
	"bytes"
	"context"
	"sync"
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
	limit        int64
	written      int64
	buf          bytes.Buffer
	cancel       context.CancelFunc
	mu           sync.Mutex
	limitReached bool
}

func (w *LimitExceededWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := len(p)

	if w.written+int64(n) >= w.limit {
		if !w.limitReached {
			w.limitReached = true

			// write only upto limit
			allowed := w.limit - w.written
			if allowed > 0 {
				w.buf.Write(p[:allowed])
				w.written += allowed
			}

			// cancel context immediately
			if w.cancel != nil {
				w.cancel()
			}
		}
		// Return n so the caller doesn't panic, but process is being killed in background
		return n, nil
	}

	w.written += int64(n)
	return w.buf.Write(p)
}

func (w *LimitExceededWriter) LimitReached() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.limitReached
}
