// Package filter evaluates a destination agent's inbound filter ladder against
// an envelope. Within a layer, rules compose as OR. Across layers, precedence
// is: direct address > deny > allow > classify.
package filter

import (
	"regexp"
	"strings"

	"github.com/ericpollmann/botbus-proto/envelope"
)

// Rule is one allow/deny filter rule.
type Rule struct {
	ID     string `json:"id"`
	Action string `json:"action"` // "allow" | "deny"
	Match  string `json:"match"`  // "sender" | "topic" | "regex"
	Value  string `json:"value"`
}

// Match reports whether a single rule matches an envelope. A malformed regex or
// unknown match kind never matches (fail closed, no panic).
func Match(r Rule, e envelope.Envelope) bool {
	switch r.Match {
	case "sender":
		return e.From == r.Value
	case "topic":
		hay := strings.ToLower(e.Subject + " " + e.Body)
		return strings.Contains(hay, strings.ToLower(r.Value))
	case "regex":
		re, err := regexp.Compile(r.Value)
		if err != nil {
			return false
		}
		return re.MatchString(e.Subject + " " + e.Body)
	default:
		return false
	}
}

// Outcome is the result of evaluating the filter ladder.
type Outcome int

const (
	Deliver  Outcome = iota // deliver to this agent's inbox
	Drop                    // explicitly muted
	Classify                // no deterministic decision; defer to classification
)

func (o Outcome) String() string {
	switch o {
	case Deliver:
		return "Deliver"
	case Drop:
		return "Drop"
	default:
		return "Classify"
	}
}

// Decide runs the precedence ladder for destination agent `self`.
func Decide(e envelope.Envelope, self string, rules []Rule) Outcome {
	// Layer 1: direct address always delivers (can't be muted).
	for _, t := range e.To {
		if t == self {
			return Deliver
		}
	}
	if e.Kind == envelope.KindDM {
		return Deliver
	}
	if strings.Contains(e.Body, "@"+self) {
		return Deliver
	}

	// Layer 2: deny rules (evaluated before allow — an explicit mute wins).
	for _, r := range rules {
		if r.Action == "deny" && Match(r, e) {
			return Drop
		}
	}
	// Layer 3: allow rules.
	for _, r := range rules {
		if r.Action == "allow" && Match(r, e) {
			return Deliver
		}
	}
	// Layer 4: defer to classification.
	return Classify
}
