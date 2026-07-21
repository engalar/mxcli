package task

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

type BuildOptions struct {
	ProjectPath string
	SkipCheck   bool
	DryRun      bool
}

type BuildTask struct {
	opts      BuildOptions
	state     State
	events    chan Event
	done      bool
	cancelled bool
}

func NewBuildTask(opts BuildOptions) *BuildTask {
	return &BuildTask{
		opts:   opts,
		state:  StatePending,
		events: make(chan Event, 200),
	}
}

func (t *BuildTask) State() State { return t.state }
func (t *BuildTask) Done() bool   { return t.done }

func (t *BuildTask) Cancel() {
	t.cancelled = true
	t.state = StateCancelled
}

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

	if t.cancelled {
		t.emit(Event{Type: EventPhaseChange, State: StateCancelled, Phase: "cancelled", Message: "Build cancelled"})
		return
	}

	if t.opts.DryRun {
		t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "dry-run", Message: "Dry run — not executing build"})
		return
	}

	// LineWriter captures raw mxbuild output
	lw := NewLineWriter(os.Stdout)
	go func() {
		for line := range lw.Lines {
			t.emit(Event{Type: EventLogLine, Phase: "raw", Line: line})
		}
	}()

	err := docker.Build(docker.BuildOptions{
		ProjectPath: t.opts.ProjectPath,
		SkipCheck:   t.opts.SkipCheck,
		Stdout:      lw,
		OnPhase: func(name, status string, pct int, msg string) {
			st := StateRunning
			if status == "completed" {
				st = StateCompleted
			}
			t.emit(Event{
				Type:    EventPhaseChange,
				State:   st,
				Phase:   name,
				Message: msg,
				Pct:     float64(pct),
			})
		},
	})

	lw.Close()

	if t.cancelled {
		t.emit(Event{Type: EventPhaseChange, State: StateCancelled, Phase: "cancelled", Message: "Build cancelled"})
		return
	}

	if err != nil {
		t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "error", Message: fmt.Sprintf("Build failed: %v", err), Err: err})
		return
	}

	t.emit(Event{Type: EventPhaseChange, State: StateCompleted, Phase: "done", Message: "Build complete", Pct: 100})
}

func (t *BuildTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}
