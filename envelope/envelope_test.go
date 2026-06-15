package envelope

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	e := Envelope{
		V: 1, ID: "01jxx", From: "myth-compiler",
		To: []string{"myth-boss"}, Kind: "chat", Scope: "module",
		Channel: "abc", Subject: "hi", Body: "hello world",
	}
	raw, err := Encode(e)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.From != e.From || got.Body != e.Body || got.Kind != e.Kind {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	if len(got.To) != 1 || got.To[0] != "myth-boss" {
		t.Fatalf("To not preserved: %+v", got.To)
	}
	if got.V != e.V || got.ID != e.ID || got.Scope != e.Scope ||
		got.Channel != e.Channel || got.Subject != e.Subject {
		t.Fatalf("scalar field not preserved: got %+v want %+v", got, e)
	}
}

func TestNewIDProperties(t *testing.T) {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	inSet := func(s string) bool {
		for _, r := range s {
			found := false
			for _, a := range alphabet {
				if r == a {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if len(id) != 26 {
			t.Fatalf("len=%d want 26: %q", len(id), id)
		}
		if !inSet(id) {
			t.Fatalf("char outside crockford alphabet: %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id: %q", id)
		}
		seen[id] = true
	}
}

func TestParseOrWrap(t *testing.T) {
	raw, _ := Encode(Envelope{V: 1, From: "a", Kind: KindChat, Body: "x"})
	e, wrapped := ParseOrWrap("a", "fromchan", raw)
	if wrapped {
		t.Fatal("valid envelope should not be wrapped")
	}
	if e.From != "a" {
		t.Fatalf("From=%q", e.From)
	}
	e, wrapped = ParseOrWrap("eric", "fromchan", []byte("just typing here"))
	if !wrapped {
		t.Fatal("plain text should be wrapped")
	}
	if e.Kind != KindChat || e.Body != "just typing here" || e.From != "eric" {
		t.Fatalf("bad wrap: %+v", e)
	}
	if e.Channel != "fromchan" {
		t.Fatalf("Channel=%q", e.Channel)
	}
	if e.ID == "" {
		t.Fatal("wrapped envelope must get an id")
	}
}
