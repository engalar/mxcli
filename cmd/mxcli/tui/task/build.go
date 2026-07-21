package task

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

var taskLog = func() *os.File {
	if os.Getenv("MXCLI_TUI_DEBUG") != "1" {
		return nil
	}
	home, _ := os.UserHomeDir()
	f, err := os.OpenFile(filepath.Join(home, ".mxcli", "task-debug.log"),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil
	}
	fmt.Fprintf(f, "=== task debug started ===\n")
	return f
}()

func taskDebug(format string, args ...interface{}) {
	if taskLog == nil {
		return
	}
	fmt.Fprintf(taskLog, format+"\n", args...)
}

type BuildOptions struct {
	ProjectPath     string
	SkipCheck       bool
	DryRun          bool
	UseDeployLayout bool
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
	defer func() {
		if r := recover(); r != nil {
			taskDebug("BuildTask.run: PANIC: %v", r)
		}
	}()
	taskDebug("BuildTask.run: started")

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
		defer func() {
			if r := recover(); r != nil {
				taskDebug("LineWriter: PANIC: %v", r)
			}
		}()
		n := 0
		for line := range lw.Lines {
			t.emit(Event{Type: EventLogLine, Phase: "raw", Line: line})
			n++
		}
		taskDebug("LineWriter: %d lines processed", n)
	}()
	taskDebug("BuildTask: calling docker.Build(project=%q)", t.opts.ProjectPath)

	taskDebug("BuildTask: starting docker.Build...")
	err := docker.Build(docker.BuildOptions{
		ProjectPath:     t.opts.ProjectPath,
		SkipCheck:       t.opts.SkipCheck,
		UseDeployLayout: t.opts.UseDeployLayout,
		Stdout:          lw,
		OnPhase: func(name, status string, pct int, msg string) {
			taskDebug("OnPhase: name=%s status=%s pct=%d msg=%q", name, status, pct, msg)
			t.emit(Event{
				Type:    EventPhaseChange,
				State:   StateRunning,
				Phase:   name,
				Message: msg,
				Pct:     float64(pct),
			})
		},
	})
	taskDebug("BuildTask: docker.Build returned err=%v", err)

	lw.Close()

	if t.cancelled {
		taskDebug("BuildTask: cancelled after build")
		t.emit(Event{Type: EventPhaseChange, State: StateCancelled, Phase: "cancelled", Message: "Build cancelled"})
		return
	}

	if err != nil {
		taskDebug("BuildTask: build failed: %v", err)
		t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "error", Message: fmt.Sprintf("Build failed: %v", err), Err: err})
		return
	}

	taskDebug("BuildTask: build complete")
	t.emit(Event{Type: EventPhaseChange, State: StateCompleted, Phase: "done", Message: "Build complete", Pct: 100})
}

func (t *BuildTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}
