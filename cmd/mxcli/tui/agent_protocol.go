package tui

import "fmt"

// AgentRequest is a JSON command from an external agent (e.g. Claude).
type AgentRequest struct {
	ID     int    `json:"id"`
	Action string `json:"action"`          // "exec", "check", "state", "navigate"
	MDL    string `json:"mdl,omitempty"`    // for "exec"
	Target string `json:"target,omitempty"` // for "navigate" (e.g. "entity:Module.Entity")
}

// Validate checks that the request has all required fields for its action.
func (r AgentRequest) Validate() error {
	if r.ID == 0 {
		return fmt.Errorf("missing request id")
	}
	switch r.Action {
	case "exec":
		if r.MDL == "" {
			return fmt.Errorf("exec action requires mdl field")
		}
	case "check", "state":
		// no extra fields needed
	case "navigate":
		if r.Target == "" {
			return fmt.Errorf("navigate action requires target field")
		}
	default:
		return fmt.Errorf("unknown action: %q", r.Action)
	}
	return nil
}

// AgentResponse is the JSON response sent back to the agent.
type AgentResponse struct {
	ID     int    `json:"id"`
	OK     bool   `json:"ok"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
	Mode   string `json:"mode,omitempty"` // e.g. "overlay:exec-result"
}
