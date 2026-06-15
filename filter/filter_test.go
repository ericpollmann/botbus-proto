package filter

import (
	"testing"

	"github.com/ericpollmann/botbus-proto/envelope"
)

func TestMatch(t *testing.T) {
	e := envelope.Envelope{From: "myth-boss", Subject: "CSS Modules bug", Body: "deploy to staging failed"}

	cases := []struct {
		name string
		rule Rule
		want bool
	}{
		{"sender hit", Rule{Match: "sender", Value: "myth-boss"}, true},
		{"sender miss", Rule{Match: "sender", Value: "other"}, false},
		{"topic in subject (case-insensitive)", Rule{Match: "topic", Value: "css modules"}, true},
		{"topic in body", Rule{Match: "topic", Value: "staging"}, true},
		{"topic miss", Rule{Match: "topic", Value: "compiler"}, false},
		{"regex hit", Rule{Match: "regex", Value: "(?i)deploy.*staging"}, true},
		{"regex miss", Rule{Match: "regex", Value: "^never$"}, false},
		{"bad regex never matches", Rule{Match: "regex", Value: "("}, false},
		{"unknown match kind", Rule{Match: "bogus", Value: "x"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Match(c.rule, e); got != c.want {
				t.Fatalf("Match=%v want %v", got, c.want)
			}
		})
	}
}

func TestDecideLadder(t *testing.T) {
	self := "myth-explore"
	base := envelope.Envelope{From: "myth-boss", Body: "general chatter"}

	t.Run("direct via To beats everything", func(t *testing.T) {
		e := base
		e.To = []string{self}
		rules := []Rule{{Action: "deny", Match: "sender", Value: "myth-boss"}}
		if got := Decide(e, self, rules); got != Deliver {
			t.Fatalf("got %v want Deliver", got)
		}
	})
	t.Run("dm kind delivers", func(t *testing.T) {
		e := base
		e.Kind = envelope.KindDM
		if got := Decide(e, self, nil); got != Deliver {
			t.Fatalf("got %v want Deliver", got)
		}
	})
	t.Run("@mention in body delivers", func(t *testing.T) {
		e := base
		e.Body = "hey @myth-explore look at this"
		if got := Decide(e, self, nil); got != Deliver {
			t.Fatalf("got %v want Deliver", got)
		}
	})
	t.Run("deny beats allow when both match", func(t *testing.T) {
		e := base
		rules := []Rule{
			{Action: "allow", Match: "sender", Value: "myth-boss"},
			{Action: "deny", Match: "topic", Value: "chatter"},
		}
		if got := Decide(e, self, rules); got != Drop {
			t.Fatalf("got %v want Drop", got)
		}
	})
	t.Run("allow delivers", func(t *testing.T) {
		e := base
		rules := []Rule{{Action: "allow", Match: "sender", Value: "myth-boss"}}
		if got := Decide(e, self, rules); got != Deliver {
			t.Fatalf("got %v want Deliver", got)
		}
	})
	t.Run("no rule match falls to classify", func(t *testing.T) {
		e := base
		if got := Decide(e, self, nil); got != Classify {
			t.Fatalf("got %v want Classify", got)
		}
	})
	t.Run("DM cannot be muted by a matching deny", func(t *testing.T) {
		e := base
		e.Kind = envelope.KindDM
		rules := []Rule{{Action: "deny", Match: "sender", Value: "myth-boss"}}
		if got := Decide(e, self, rules); got != Deliver {
			t.Fatalf("got %v want Deliver — a DM must not be mutable", got)
		}
	})
	t.Run("@mention cannot be muted by a matching deny", func(t *testing.T) {
		e := base
		e.Body = "hey @myth-explore look at this"
		rules := []Rule{{Action: "deny", Match: "sender", Value: "myth-boss"}}
		if got := Decide(e, self, rules); got != Deliver {
			t.Fatalf("got %v want Deliver — an @mention must not be mutable", got)
		}
	})
}
