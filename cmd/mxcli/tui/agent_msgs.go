package tui

import tea "github.com/charmbracelet/bubbletea"

// AgentExecMsg requests MDL execution from an external agent.
type AgentExecMsg struct {
	RequestID  int
	MDL        string
	ResponseCh chan<- AgentResponse
}

// AgentCheckMsg requests a syntax/reference check.
type AgentCheckMsg struct {
	RequestID  int
	ResponseCh chan<- AgentResponse
}

// AgentStateMsg requests current TUI state (active view, project path, etc.).
type AgentStateMsg struct {
	RequestID  int
	ResponseCh chan<- AgentResponse
}

// AgentNavigateMsg requests navigation to a specific element.
type AgentNavigateMsg struct {
	RequestID  int
	Target     string
	ResponseCh chan<- AgentResponse
}

// agentExecDoneMsg carries exec result back to App for agent response.
type agentExecDoneMsg struct {
	RequestID  int
	Output     string
	Success    bool
	ResponseCh chan<- AgentResponse
}

// agentConfirmedMsg is sent when user presses q to confirm agent result.
type agentConfirmedMsg struct {
	RequestID  int
	ResponseCh chan<- AgentResponse
	Output     string
	Success    bool
}

// Ensure messages satisfy tea.Msg.
var (
	_ tea.Msg = AgentExecMsg{}
	_ tea.Msg = AgentCheckMsg{}
	_ tea.Msg = AgentStateMsg{}
	_ tea.Msg = AgentNavigateMsg{}
	_ tea.Msg = agentExecDoneMsg{}
	_ tea.Msg = agentConfirmedMsg{}
)
