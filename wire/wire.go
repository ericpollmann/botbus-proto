// Package wire holds the routing-fabric control-plane request/response shapes
// shared by the private router (control server) and the public CLI/daemon
// (control client). It carries no behavior — just the JSON contract.
package wire

// AgentSpec is the JSON body of a register request (PUT /v1/agents/{id}). The
// agent id travels in the path, the capability key in the Authorization header;
// neither is part of this body.
type AgentSpec struct {
	Name         string `json:"name,omitempty"`
	Host         string `json:"host,omitempty"`
	InboxChannel string `json:"inbox_channel"`
	Focus        string `json:"focus,omitempty"`
	Interest     string `json:"interest,omitempty"`
	Parent       string `json:"parent,omitempty"`
	Mode         string `json:"mode,omitempty"`
	BatchMS      int    `json:"batch_ms,omitempty"`
	BatchN       int    `json:"batch_n,omitempty"`
	BatchBytes   int    `json:"batch_bytes,omitempty"`
	ModelTier    string `json:"model_tier,omitempty"`
}
