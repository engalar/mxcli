package task

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

type RunOptions struct {
	PadDir        string
	DB            string
	AdminPassword string
	AppPort       int
	AdminPort     int
	CmdHint       string
	DryRun        bool
}

type RunTask struct {
	opts   RunOptions
	state  State
	events chan Event
	done   bool
}

func NewRunTask(opts RunOptions) *RunTask {
	return &RunTask{
		opts:   opts,
		state:  StatePending,
		events: make(chan Event, 10),
	}
}

func (t *RunTask) State() State { return t.state }
func (t *RunTask) Done() bool   { return t.done }

func (t *RunTask) emit(ev Event) {
	t.state = ev.State
	select {
	case t.events <- ev:
	default:
	}
}

func (t *RunTask) Start() tea.Cmd {
	go t.run()
	return StreamCmd(t.events)
}

func (t *RunTask) run() {
	defer close(t.events)
	defer func() { t.done = true }()

	if t.opts.DryRun {
		t.emit(Event{State: StateFailed, Phase: "dry-run", Message: "Dry run — not executing run"})
		return
	}

	t.emit(Event{State: StateRunning, Phase: "startup", Message: "Starting Mendix Runtime...", Pct: 10})

	err := docker.StartLocal(docker.LocalRunOptions{
		PadDir:        t.opts.PadDir,
		DB:            t.opts.DB,
		AdminPassword: t.opts.AdminPassword,
		AppPort:       t.opts.AppPort,
		AdminPort:     t.opts.AdminPort,
		CmdHint:       t.opts.CmdHint,
	})

	if err != nil {
		t.emit(Event{State: StateFailed, Phase: "error", Message: fmt.Sprintf("Runtime error: %v", err), Err: err})
		return
	}

	t.emit(Event{State: StateCompleted, Phase: "done", Message: "Runtime stopped", Pct: 100})
}

func (t *RunTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}
