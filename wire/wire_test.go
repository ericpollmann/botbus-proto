package wire

import (
	"encoding/json"
	"testing"
)

func TestAgentNodeJSONRoundTrip(t *testing.T) {
	n := AgentNode{
		ID: "ct9av...", Name: "myth-compiler", Focus: "compile",
		Parent: "root-id", InboxChannel: "inbox-1", Mode: "session", Live: true,
	}
	b, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AgentNode
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != n {
		t.Fatalf("round-trip mismatch: %+v != %+v", got, n)
	}
}
