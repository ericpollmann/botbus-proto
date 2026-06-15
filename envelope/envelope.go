// Package envelope defines the fabric message envelope carried inside botbus
// message bodies. The botbus hub remains byte-agnostic; envelopes are a
// client-side convention.
package envelope

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"time"
)

const crockford = "0123456789abcdefghjkmnpqrstvwxyz"

// NewID returns a 26-char Crockford-base32 ULID-style id: a 48-bit millisecond
// timestamp prefix (lexicographically sortable) followed by 80 random bits.
func NewID() string {
	var b [16]byte
	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	_, _ = rand.Read(b[6:])
	return encodeID(b[:])
}

// encodeID renders 16 bytes (128 bits) as 26 Crockford base32 chars (130 bits,
// top 2 bits zero) via big-int base conversion.
func encodeID(b []byte) string {
	n := new(big.Int).SetBytes(b)
	base := big.NewInt(32)
	var out [26]byte
	mod := new(big.Int)
	for i := 25; i >= 0; i-- {
		n.DivMod(n, base, mod)
		out[i] = crockford[mod.Int64()]
	}
	return string(out[:])
}

// Envelope is the JSON payload placed in a botbus message body.
type Envelope struct {
	V       int      `json:"v"`
	ID      string   `json:"id"`
	TS      string   `json:"ts,omitempty"`
	From    string   `json:"from"`
	To      []string `json:"to,omitempty"`
	Kind    string   `json:"kind"`
	Scope   string   `json:"scope,omitempty"`
	Channel string   `json:"channel,omitempty"`
	Subject string   `json:"subject,omitempty"`
	Body    string   `json:"body"`
}

// Kind constants.
const (
	KindChat          = "chat"
	KindDM            = "dm"
	KindTask          = "task"
	KindEscalate      = "escalate"
	KindStatus        = "status"
	KindReviewRequest = "review_request"
	KindCouncilResult = "council_result"
	KindControl       = "control"
	KindBatch         = "batch"
)

// Encode serializes an envelope to JSON bytes.
func Encode(e Envelope) ([]byte, error) { return json.Marshal(e) }

// Decode parses JSON bytes into an envelope.
func Decode(b []byte) (Envelope, error) {
	var e Envelope
	err := json.Unmarshal(b, &e)
	return e, err
}

// ParseOrWrap tries to parse b as an envelope. If b is not valid envelope JSON
// (a human typing into a channel, or a non-fabric client), it wraps the raw
// text as a chat envelope from `sender` originating on `fromChannel`.
// The bool reports whether wrapping occurred.
func ParseOrWrap(sender, fromChannel string, b []byte) (Envelope, bool) {
	e, err := Decode(b)
	if err == nil && e.V != 0 && e.Kind != "" {
		return e, false
	}
	return Envelope{
		V:       1,
		ID:      NewID(),
		From:    sender,
		Kind:    KindChat,
		Channel: fromChannel,
		Body:    string(b),
	}, true
}
