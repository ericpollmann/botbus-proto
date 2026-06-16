// Package hubclient abstracts the botbus hub's public HTTP surface (mint,
// publish, subscribe-with-resume) so the router never imports hub internals.
package hubclient

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Frame is one received message from a subscribed channel.
type Frame struct {
	Name   string // sender name (left of "name: body")
	Body   string // message body
	Resume string // resume token to persist (the SSE id: field)
}

// HubClient is the hub surface the router depends on.
type HubClient interface {
	MintChannel(ctx context.Context) (string, error)
	Publish(ctx context.Context, channel, body string) error
	Subscribe(ctx context.Context, channel, resume string) (<-chan Frame, error)
}

// Fake is an in-memory HubClient for tests.
type Fake struct {
	mu      sync.Mutex
	seq     int
	pubs    map[string][]string
	streams map[string]chan Frame
}

// NewFake constructs a Fake.
func NewFake() *Fake {
	return &Fake{pubs: map[string][]string{}, streams: map[string]chan Frame{}}
}

// MintChannel returns a unique fake channel id.
func (f *Fake) MintChannel(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	return fmt.Sprintf("fakechan%d", f.seq), nil
}

// Publish records a publish and forwards it to any subscriber of `channel`.
func (f *Fake) Publish(ctx context.Context, channel, body string) error {
	f.mu.Lock()
	f.pubs[channel] = append(f.pubs[channel], body)
	ch := f.streams[channel]
	f.mu.Unlock()
	if ch != nil {
		name, payload := splitNameBody(body)
		select {
		case ch <- Frame{Name: name, Body: payload}:
		default:
		}
	}
	return nil
}

// Published returns the bodies published to a channel (test helper).
func (f *Fake) Published(channel string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.pubs[channel]))
	copy(out, f.pubs[channel])
	return out
}

// Subscribe returns a buffered channel that Publish feeds.
func (f *Fake) Subscribe(ctx context.Context, channel, resume string) (<-chan Frame, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan Frame, 256)
	f.streams[channel] = ch
	return ch, nil
}

// Inject pushes a frame to a channel's subscriber as if it arrived from the hub.
func (f *Fake) Inject(channel string, fr Frame) {
	f.mu.Lock()
	ch := f.streams[channel]
	f.mu.Unlock()
	if ch != nil {
		ch <- fr
	}
}

// splitNameBody splits "name: body" (botbus text convention). If no delimiter,
// name is empty and the whole string is the body.
func splitNameBody(s string) (name, body string) {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == ':' && s[i+1] == ' ' {
			return s[:i], s[i+2:]
		}
	}
	return "", s
}

// compile-time guards: both implementations satisfy HubClient.
var (
	_ HubClient = (*Fake)(nil)
	_ HubClient = (*HTTPClient)(nil)
)

// HTTPClient talks to a real botbus hub over HTTP/SSE.
type HTTPClient struct {
	base   string
	domain string
	hc     *http.Client
}

// NewHTTPClient constructs an HTTPClient. base is the dialable origin; domain is
// the apex used to build the `<channel>.<domain>` Host header the hub routes on.
func NewHTTPClient(base, domain string) *HTTPClient {
	return &HTTPClient{
		base:   strings.TrimRight(base, "/"),
		domain: domain,
		hc:     &http.Client{},
	}
}

func (c *HTTPClient) hostFor(channel string) string {
	return channel + "." + c.domain
}

// MintChannel GETs the hub mint endpoint (new.<domain>) and returns the id.
//
// The real hub (servNew in ui.go) responds with a full channel URL plus a
// trailing newline — "https://<id>.<domain>/\n" — NOT a bare id, so we strip
// the scheme, the "."+domain suffix, and the trailing slash to recover the id
// (mirroring the hub's own e2eMint test helper).
func (c *HTTPClient) MintChannel(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/", nil)
	if err != nil {
		return "", err
	}
	req.Host = "new." + c.domain
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(b))
	id = strings.TrimPrefix(strings.TrimPrefix(id, "https://"), "http://")
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimSuffix(id, "."+c.domain)
	if id == "" {
		return "", fmt.Errorf("hubclient: empty mint response")
	}
	return id, nil
}

// Publish POSTs `body` to the channel (Host-header addressed). The hub replies
// 204 No Content on success (servPublish in transport.go).
func (c *HTTPClient) Publish(ctx context.Context, channel, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Host = c.hostFor(channel)
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hubclient: publish status %d", resp.StatusCode)
	}
	return nil
}

// Subscribe opens an SSE stream and parses frames until ctx is cancelled. The
// hub addresses channels by Host subdomain, gates SSE on Accept:
// text/event-stream, and resumes gap-only from the Last-Event-ID header (see
// servSSE in transport.go). Each event is `id: <token>\n` then one-or-more
// `data: <line>\n` then a blank line.
func (c *HTTPClient) Subscribe(ctx context.Context, channel, resume string) (<-chan Frame, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Host = c.hostFor(channel)
	req.Header.Set("Accept", "text/event-stream")
	if resume != "" {
		req.Header.Set("Last-Event-ID", resume)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	out := make(chan Frame, 256)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
		var id string
		var data []string
		emit := func() {
			if len(data) == 0 {
				return
			}
			name, body := splitNameBody(strings.Join(data, "\n"))
			select {
			case out <- Frame{Name: name, Body: body, Resume: id}:
			case <-ctx.Done():
			}
			id = ""
			data = nil
		}
		for sc.Scan() {
			line := sc.Text()
			switch {
			case line == "":
				emit()
			case strings.HasPrefix(line, "id: "):
				id = line[len("id: "):]
			case strings.HasPrefix(line, "data: "):
				data = append(data, line[len("data: "):])
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
	return out, nil
}
