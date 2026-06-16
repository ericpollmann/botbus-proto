package hubclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

// Verify: an idle (never-sending) SSE connection's goroutine actually exits
// when ctx is cancelled, despite Scan() blocking on the read.
func TestSubscribeIdleCancelTerminates(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if fl, ok := w.(http.Flusher); ok {
			fmt.Fprint(w, ": ok\n\n")
			fl.Flush()
		}
		close(started)
		<-r.Context().Done() // hold the conn open, never send a frame
	}))
	defer srv.Close()

	before := runtime.NumGoroutine()
	c := NewHTTPClient(srv.URL, "botbus.ai")
	ctx, cancel := context.WithCancel(context.Background())
	out, err := c.Subscribe(ctx, "abc123", "")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	<-started
	cancel() // cancel while the scan loop is blocked on an idle conn

	// out must close (goroutine returned) within a reasonable window.
	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("unexpected frame on idle stream")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("goroutine did not exit on ctx cancel (idle blocking-read leak)")
	}
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()
	t.Logf("goroutines before=%d after=%d", before, after)
}
