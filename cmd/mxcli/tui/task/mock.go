package task

import (
	"fmt"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

type MockOptions struct {
	SpecPath string
	Port     int
}

type MockTask struct {
	opts      MockOptions
	state     State
	events    chan Event
	done      bool
	cancelled bool
	prismPID  int
	stopCh    chan struct{}
}

func NewMockTask(opts MockOptions) *MockTask {
	return &MockTask{
		opts:   opts,
		state:  StatePending,
		events: make(chan Event, 100),
		stopCh: make(chan struct{}),
	}
}

func (t *MockTask) State() State   { return t.state }
func (t *MockTask) Done() bool     { return t.done }
func (t *MockTask) Opts() MockOptions { return t.opts }

func (t *MockTask) emit(ev Event) {
	t.state = ev.State
	select {
	case t.events <- ev:
	default:
	}
}

func (t *MockTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}

func (t *MockTask) Cancel() {
	t.cancelled = true
	t.state = StateCancelled
	if t.prismPID > 0 {
		_ = syscall.Kill(-t.prismPID, syscall.SIGTERM)
		GlobalProcTracker.Remove(t.prismPID)
	}
	close(t.stopCh)
}

func (t *MockTask) Start() tea.Cmd {
	go t.run()
	return StreamCmd(t.events)
}

func (t *MockTask) run() {
	defer close(t.events)
	defer func() { t.done = true }()
	defer func() {
		if t.prismPID > 0 {
			GlobalProcTracker.Remove(t.prismPID)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			taskDebug("MockTask.run: PANIC: %v", r)
			t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "panic",
				Message: fmt.Sprintf("Panic: %v", r)})
		}
	}()

	if t.cancelled {
		t.emit(Event{Type: EventPhaseChange, State: StateCancelled, Phase: "cancelled", Message: "Mock cancelled"})
		return
	}

	t.emit(Event{Type: EventPhaseChange, State: StateRunning, Phase: "startup", Message: "Starting Prism mock server...", Pct: 10})

	pid, err := docker.StartMockServer(docker.MockOptions{
		SpecPath: t.opts.SpecPath,
		Port:     t.opts.Port,
	})

	if err != nil {
		t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "error",
			Message: fmt.Sprintf("Mock server failed: %v", err), Err: err})
		return
	}

	t.prismPID = pid
	GlobalProcTracker.Add(pid, "prism-mock", nil)

	port := t.opts.Port
	if port == 0 {
		port = 4000
	}

	t.emit(Event{Type: EventPhaseChange, State: StateRunning, Phase: "running",
		Message: fmt.Sprintf("Prism mock server running on http://localhost:%d (PID %d)", port, pid), Pct: 100})

	<-t.stopCh

	t.emit(Event{Type: EventPhaseChange, State: StateCompleted, Phase: "done",
		Message: "Prism mock server stopped", Pct: 100})
}
