package task

import (
	"testing"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{StatePending, "pending"},
		{StateRunning, "running"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StateCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestBuildTaskDryRun(t *testing.T) {
	task := NewBuildTask(BuildOptions{
		ProjectPath: "/fake/path.mpr",
		DryRun:      true,
	})
	if task.State() != StatePending {
		t.Fatalf("expected StatePending, got %v", task.State())
	}

	// Start launches a goroutine, returns StreamCmd for first event
	cmd := task.Start()
	msg := cmd()
	ev, ok := msg.(Event)
	if !ok {
		t.Fatalf("expected Event, got %T", msg)
	}
	if ev.State != StateFailed {
		t.Fatalf("expected StateFailed for dry run, got %v", ev.State)
	}
	if task.State() != StateFailed {
		t.Fatalf("expected StateFailed, got %v", task.State())
	}

	// Stream should be exhausted
	nextCmd := task.StreamCmd()
	if nextCmd != nil {
		msg2 := nextCmd()
		if msg2 != nil {
			t.Fatalf("expected nil after stream end, got %T: %v", msg2, msg2)
		}
	}
}

func TestBuildTaskCancel(t *testing.T) {
	task := NewBuildTask(BuildOptions{
		ProjectPath: "/fake/path.mpr",
		DryRun:      true,
	})
	task.Cancel()
	if task.State() != StateCancelled {
		t.Fatalf("expected StateCancelled after Cancel(), got %v", task.State())
	}

	cmd := task.Start()
	msg := cmd()
	ev, ok := msg.(Event)
	if !ok {
		t.Fatalf("expected Event, got %T", msg)
	}
	if ev.State != StateCancelled {
		t.Fatalf("expected StateCancelled after cancel+start, got %v", ev.State)
	}
}

func TestRunTaskDryRun(t *testing.T) {
	task := NewRunTask(RunOptions{
		CmdHint: "test-run",
		DryRun:  true,
	})
	if task.State() != StatePending {
		t.Fatalf("expected StatePending, got %v", task.State())
	}

	cmd := task.Start()
	msg := cmd()
	ev, ok := msg.(Event)
	if !ok {
		t.Fatalf("expected Event, got %T", msg)
	}
	if ev.State != StateFailed {
		t.Fatalf("expected StateFailed for dry run, got %v", ev.State)
	}

	nextCmd := task.StreamCmd()
	if nextCmd != nil {
		msg2 := nextCmd()
		if msg2 != nil {
			t.Fatalf("expected nil after stream end, got %T: %v", msg2, msg2)
		}
	}
}
