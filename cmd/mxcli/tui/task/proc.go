package task

import (
	"os"
	"sync"
	"syscall"
	"time"
)

// trackedProc holds a child process registered with the global ProcTracker.
type trackedProc struct {
	PID    int
	Label  string
	cancel func() // optional, called before SIGTERM
}

// GlobalProcTracker is the singleton process tracker for the mxcli TUI.
// All child processes spawned by build/run tasks are registered here
// and killed when the TUI exits (clean or crash).
var GlobalProcTracker = &ProcTracker{}

// ProcTracker tracks child processes and provides a KillAll method
// for cleanup on TUI exit. It uses process-group killing (SIGTERM then SIGKILL)
// so each child's entire subprocess tree is cleaned up.
type ProcTracker struct {
	mu     sync.Mutex
	procs  []trackedProc
	cleaned bool
}

func (pt *ProcTracker) Add(pid int, label string, cancel func()) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.cleaned {
		return
	}
	pt.procs = append(pt.procs, trackedProc{PID: pid, Label: label, cancel: cancel})
}

func (pt *ProcTracker) Remove(pid int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	for i, p := range pt.procs {
		if p.PID == pid {
			pt.procs = append(pt.procs[:i], pt.procs[i+1:]...)
			return
		}
	}
}

// KillAll sends SIGTERM to every tracked process group, waits briefly for
// graceful shutdown, then escalates to SIGKILL for survivors.
// After this call the tracker is empty and cannot accept new processes.
func (pt *ProcTracker) KillAll() {
	pt.mu.Lock()
	procs := pt.procs
	pt.procs = nil
	pt.cleaned = true
	pt.mu.Unlock()

	if len(procs) == 0 {
		return
	}

	// Phase 1: cancel callbacks (e.g. context cancel for CommandContext)
	for _, p := range procs {
		if p.cancel != nil {
			p.cancel()
		}
	}

	// Phase 2: SIGTERM to each process group
	for _, p := range procs {
		proc, err := os.FindProcess(p.PID)
		if err != nil {
			continue
		}
		// Negative PID = process group signal (Linux)
		_ = syscall.Kill(-p.PID, syscall.SIGTERM)
		_ = proc.Release()
	}

	// Phase 3: wait then SIGKILL survivors
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		allDead := true
		for _, p := range procs {
			proc, err := os.FindProcess(p.PID)
			if err != nil {
				continue
			}
			// Signal(0) = test if alive
			if err := proc.Signal(syscall.Signal(0)); err == nil {
				allDead = false
			}
			proc.Release()
		}
		if allDead {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Phase 4: SIGKILL survivors
	for _, p := range procs {
		_ = syscall.Kill(-p.PID, syscall.SIGKILL)
	}
}
