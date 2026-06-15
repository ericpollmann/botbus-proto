// Package keys mints opaque capability keys for the routing fabric. A key is
// 128 bits of cryptographic randomness rendered as 26 lowercase Crockford
// base32 chars — the same alphabet botbus uses for channel ids. The key is the
// auth: whoever holds it can act as the bound agent, so it is never logged or
// placed in an envelope.
package keys

import (
	"crypto/rand"
	"encoding/base32"
)

var enc = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

// New mints a fresh capability key.
func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("keys: crypto/rand failed: " + err.Error())
	}
	return enc.EncodeToString(b[:])
}
