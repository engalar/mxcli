package task

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

type RunOptions struct {
	DeployDir     string
	DB            string
	AdminPassword string
	AppPort       int
	AdminPort     int
	CmdHint       string
	DryRun        bool

	ProjectDir string
}

type RunTask struct {
	opts       RunOptions
	state      State
	events     chan Event
	done       bool
	cancelled  bool
	lock       *RunLock
	runtimePID int // PID of the Java runtime process, 0 if not started
}

func NewRunTask(opts RunOptions) *RunTask {
	return &RunTask{
		opts:   opts,
		state:  StatePending,
		events: make(chan Event, 100),
	}
}

func (t *RunTask) State() State { return t.state }
func (t *RunTask) Done() bool   { return t.done }
func (t *RunTask) Lock() *RunLock { return t.lock }

func (t *RunTask) emit(ev Event) {
	t.state = ev.State
	select {
	case t.events <- ev:
	default:
	}
}

func (t *RunTask) StreamCmd() tea.Cmd {
	return StreamCmd(t.events)
}

// Cancel sends SIGTERM to the Java runtime process and marks the task cancelled.
// Unlike the previous no-op, this actually terminates the child process.
func (t *RunTask) Cancel() {
	t.cancelled = true
	t.state = StateCancelled
	if t.runtimePID > 0 {
		// Kill the process group (negative PID) so all child Java processes die
		_ = syscall.Kill(-t.runtimePID, syscall.SIGTERM)
		GlobalProcTracker.Remove(t.runtimePID)
	}
}

func (t *RunTask) Start() tea.Cmd {
	go t.run()
	return StreamCmd(t.events)
}

func (t *RunTask) projectDir() string {
	if t.opts.ProjectDir != "" {
		return t.opts.ProjectDir
	}
	return filepath.Dir(t.opts.DeployDir)
}

func (t *RunTask) run() {
	defer close(t.events)
	defer func() { t.done = true }()
	defer t.removeLock()
	defer func() {
		if r := recover(); r != nil {
			taskDebug("RunTask.run: PANIC: %v", r)
			t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "panic",
				Message: fmt.Sprintf("Panic: %v", r)})
		}
	}()

	if t.cancelled {
		t.emit(Event{Type: EventPhaseChange, State: StateCancelled, Phase: "cancelled", Message: "Run cancelled"})
		return
	}

	if t.opts.DryRun {
		t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "dry-run", Message: "Dry run — not executing run"})
		return
	}

	t.lock = &RunLock{
		AppPort:   t.opts.AppPort,
		AdminPort: t.opts.AdminPort,
		Password:  t.opts.AdminPassword,
		StartedAt: time.Now(),
	}

	pDir := t.projectDir()
	appPort := t.opts.AppPort
	if appPort == 0 {
		appPort = 8080
	}
	adminPort := t.opts.AdminPort
	if adminPort == 0 {
		adminPort = 8090
	}

	t.emit(Event{Type: EventPhaseChange, State: StateRunning, Phase: "startup", Message: "Starting Mendix Runtime...", Pct: 10})

	// Use RealStarter so we can capture the child PID
	starter := &docker.RealStarter{}
	startTime := time.Now()
	err := docker.StartLocal(docker.LocalRunOptions{
		DeployDir:     t.opts.DeployDir,
		DB:            t.opts.DB,
		AdminPassword: t.opts.AdminPassword,
		AppPort:       t.opts.AppPort,
		AdminPort:     t.opts.AdminPort,
		CmdHint:       t.opts.CmdHint,
		Starter:       starter,
	})

	elapsed := time.Since(startTime).Round(time.Second)

	if err == nil {
		// Capture PID from starter (available after Start succeeds)
		t.runtimePID = starter.ChildPID
		if t.runtimePID > 0 {
			GlobalProcTracker.Add(t.runtimePID, "java-runtime", nil)
		}
		t.lock.AppPort = appPort
		t.lock.AdminPort = adminPort
		t.lock.PID = t.runtimePID
		_ = WriteLock(pDir, t.lock)
		if t.runtimePID > 0 {
			GlobalProcTracker.Remove(t.runtimePID)
		}
		t.emit(Event{Type: EventPhaseChange, State: StateCompleted, Phase: "done",
			Message: fmt.Sprintf("Runtime stopped after %s", elapsed), Pct: 100})
	} else {
		t.emit(Event{Type: EventPhaseChange, State: StateFailed, Phase: "error",
			Message: fmt.Sprintf("Runtime failed: %v", err), Err: err})
	}
}

func (t *RunTask) removeLock() {
	if t.lock == nil {
		return
	}
	_ = RemoveLock(t.projectDir())
	t.lock = nil
}
