package keys

import (
	"regexp"
	"testing"
)

func TestNewFormatAndUniqueness(t *testing.T) {
	re := regexp.MustCompile(`^[0-9abcdefghjkmnpqrstvwxyz]{26}$`)
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		k := New()
		if !re.MatchString(k) {
			t.Fatalf("key %q does not match Crockford-base32 26-char format", k)
		}
		if seen[k] {
			t.Fatalf("duplicate key minted: %q", k)
		}
		seen[k] = true
	}
}
