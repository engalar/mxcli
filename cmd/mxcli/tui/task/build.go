package task

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

type BuildOptions struct {
	ProjectPath string
	SkipCheck   bool
	DryRun      bool
}

type BuildTask struct {
	opts   BuildOptions
	state  State
	events chan Event
	done   bool
}

func NewBuildTask(opts BuildOptions) *BuildTask {
	return &BuildTask{
		opts:   opts,
		state:  StatePending,
		events: make(chan Event, 10),
	}
}

func (t *BuildTask) State() State { return t.state }
func (t *BuildTask) Done() bool   { return t.done }

func (t *BuildTask) emit(ev Event) {
	t.state = ev.State
	select {
	case t.events <- ev:
	default:
	}
}

func (t *BuildTask) Start() tea.Cmd {
	go t.run()
	return StreamCmd(t.events)
}

func (t *BuildTask) run() {
	defer close(t.events)
	defer func() { t.done = true }()

	if t.opts.DryRun {
		t.emit(Event{State: StateFailed, Phase: "dry-run", Message: "Dry run — not executing build"})
		return
	}

	t.emit(Event{State: StateRunning, Phase: "building", Message: "Building PAD package...", Pct: 50})

	err := docker.Build(docker.BuildOptions{
		ProjectPath: t.opts.ProjectPath,
		SkipCheck:   t.opts.SkipCheck,
	})

	if err != nil {
		t.emit(Event{State: StateFailed, Phase: "error", Message: fmt.Sprintf("Build failed: %v", err), Err: err})
		return
	}

	t.emit(Event{State: StateCompleted, Phase: "done", Message: "Build complete", Pct: 100})
}

func (t *BuildTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}
