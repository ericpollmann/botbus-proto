package hubclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Exercise MintChannel's string-trimming against shapes the spec flags as risky.
func TestMintEdgeCases(t *testing.T) {
	cases := []struct{ name, resp, want string }{
		{"canonical_https", "https://abcdefghjkmnpqrstvwxyz0123.botbus.ai/\n", "abcdefghjkmnpqrstvwxyz0123"},
		{"http_scheme", "http://abcdefghjkmnpqrstvwxyz0123.botbus.ai/\n", "abcdefghjkmnpqrstvwxyz0123"},
		{"no_trailing_slash", "https://abcdefghjkmnpqrstvwxyz0123.botbus.ai\n", "abcdefghjkmnpqrstvwxyz0123"},
		{"leading_space", "  https://abcdefghjkmnpqrstvwxyz0123.botbus.ai/\n", "abcdefghjkmnpqrstvwxyz0123"},
		{"no_scheme_bare_host", "abcdefghjkmnpqrstvwxyz0123.botbus.ai/\n", "abcdefghjkmnpqrstvwxyz0123"},
		{"bare_id", "abcdefghjkmnpqrstvwxyz0123\n", "abcdefghjkmnpqrstvwxyz0123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tc.resp)
			}))
			defer srv.Close()
			c := NewHTTPClient(srv.URL, "botbus.ai")
			id, err := c.MintChannel(context.Background())
			if err != nil {
				t.Fatalf("MintChannel: %v", err)
			}
			if id != tc.want {
				t.Errorf("resp=%q -> id=%q, want %q", tc.resp, id, tc.want)
			}
		})
	}
}
