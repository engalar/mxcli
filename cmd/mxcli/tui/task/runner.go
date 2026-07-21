package task

import tea "github.com/charmbracelet/bubbletea"

type State int

const (
	StatePending State = iota
	StateRunning
	StateCompleted
	StateFailed
	StateCancelled
)

func (s State) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateRunning:
		return "running"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

type Event struct {
	State   State
	Phase   string
	Message string
	Pct     float64
	Err     error
}

func StreamCmd(ch <-chan Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return ev
	}
}
