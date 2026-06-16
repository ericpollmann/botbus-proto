package hubclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFakeImplementsInterface(t *testing.T) {
	var _ HubClient = NewFake()
}

func TestFakePublishAndDrain(t *testing.T) {
	ctx := context.Background()
	f := NewFake()
	if err := f.Publish(ctx, "chanA", "hello"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	got := f.Published("chanA")
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("Published=%v", got)
	}
}

func TestFakeMint(t *testing.T) {
	f := NewFake()
	c1, err := f.MintChannel(context.Background())
	if err != nil {
		t.Fatalf("MintChannel: %v", err)
	}
	c2, _ := f.MintChannel(context.Background())
	if c1 == "" || c1 == c2 {
		t.Fatalf("mint should return unique non-empty ids: %q %q", c1, c2)
	}
}

func TestHTTPPublishSetsHostAndBody(t *testing.T) {
	var gotHost, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "botbus.ai")
	if err := c.Publish(context.Background(), "abc123", "router: hello"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !strings.HasPrefix(gotHost, "abc123.") {
		t.Fatalf("Host=%q, want abc123.* subdomain", gotHost)
	}
	if gotBody != "router: hello" {
		t.Fatalf("body=%q", gotBody)
	}
}

func TestHTTPMint(t *testing.T) {
	// Mirror the real hub's servNew shape: it returns a full URL with a
	// trailing newline ("https://<id>.<domain>/\n"), NOT a bare id. The client
	// must strip scheme + "."+domain suffix + trailing slash to recover the id.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Host, "new.") {
			t.Errorf("mint Host=%q, want new.* subdomain", r.Host)
		}
		fmt.Fprint(w, "https://abcdefghjkmnpqrstvwxyz0123.botbus.ai/\n")
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL, "botbus.ai")
	id, err := c.MintChannel(context.Background())
	if err != nil {
		t.Fatalf("MintChannel: %v", err)
	}
	if id != "abcdefghjkmnpqrstvwxyz0123" {
		t.Fatalf("id=%q, want bare channel id", id)
	}
}

func TestHTTPSubscribeReceivesFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl, _ := w.(http.Flusher)
		fmt.Fprint(w, "id: 8.deadbeef\ndata: alice: hi there\n\n")
		fl.Flush()
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "botbus.ai")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	frames, err := c.Subscribe(ctx, "abc123", "")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	select {
	case fr := <-frames:
		if fr.Name != "alice" || fr.Body != "hi there" {
			t.Fatalf("frame=%+v", fr)
		}
		if fr.Resume != "8.deadbeef" {
			t.Fatalf("resume=%q", fr.Resume)
		}
	case <-ctx.Done():
		t.Fatal("no frame received")
	}
}
