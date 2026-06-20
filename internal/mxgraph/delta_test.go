package mxgraph

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// ── DeltaLog ──────────────────────────────────────────────

func TestDeltaLog_AppendAndReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "delta.log")

	dl, err := OpenDeltaLog(path)
	if err != nil {
		t.Fatalf("OpenDeltaLog: %v", err)
	}

	if err := dl.Emit([]Event{
		{Type: NodeCreated, Node: &Node{ID: "n1", Label: "Entity", Props: map[string]any{"Name": "A"}}},
		{Type: NodeCreated, Node: &Node{ID: "n2", Label: "Entity", Props: map[string]any{"Name": "B"}}},
		{Type: EdgeCreated, Edge: &Edge{ID: "e1", From: "n1", To: "n2", Type: "LINK"}},
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	if sz := dl.Size(); sz <= 0 {
		t.Fatalf("expected non-zero delta log size, got %d", sz)
	}

	g := New()
	sink := NewIndexManagerFromGraph(g)
	if err := dl.Replay(sink); err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if g.GetNode("n1") == nil {
		t.Error("n1 not found after replay")
	}
	if g.GetNode("n2") == nil {
		t.Error("n2 not found after replay")
	}
	if len(g.Edges("n1", Outbound)) != 1 {
		t.Errorf("expected 1 outbound edge from n1, got %d", len(g.Edges("n1", Outbound)))
	}
	dl.Close()
}

func TestDeltaLog_ResetClears(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "delta.log")

	dl, err := OpenDeltaLog(path)
	if err != nil {
		t.Fatalf("OpenDeltaLog: %v", err)
	}

	if err := dl.Emit([]Event{{Type: NodeCreated, Node: &Node{ID: "n1", Label: "Entity"}}}); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	if err := dl.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if sz := dl.Size(); sz != 0 {
		t.Errorf("expected size 0 after reset, got %d", sz)
	}

	g := New()
	sink := NewIndexManagerFromGraph(g)
	if err := dl.Replay(sink); err != nil {
		t.Fatalf("Replay after reset: %v", err)
	}
	if len(g.AllNodes()) != 0 {
		t.Error("expected empty graph after replaying reset delta")
	}
	dl.Close()
}

func TestDeltaLog_AppendAfterReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "delta.log")

	dl, err := OpenDeltaLog(path)
	if err != nil {
		t.Fatalf("OpenDeltaLog: %v", err)
	}

	// Write first batch
	if err := dl.Emit([]Event{{Type: NodeCreated, Node: &Node{ID: "n1", Label: "Entity"}}}); err != nil {
		t.Fatalf("first Emit: %v", err)
	}

	// Replay — should not interfere with subsequent writes
	g := New()
	sink := NewIndexManagerFromGraph(g)
	if err := dl.Replay(sink); err != nil {
		t.Fatalf("Replay: %v", err)
	}

	// Write second batch
	if err := dl.Emit([]Event{{Type: NodeCreated, Node: &Node{ID: "n2", Label: "Entity"}}}); err != nil {
		t.Fatalf("second Emit: %v", err)
	}

	// Replay again on a fresh graph — should see both batches
	g2 := New()
	sink2 := NewIndexManagerFromGraph(g2)
	if err := dl.Replay(sink2); err != nil {
		t.Fatalf("second Replay: %v", err)
	}
	if g2.GetNode("n1") == nil {
		t.Error("n1 missing after second replay")
	}
	if g2.GetNode("n2") == nil {
		t.Error("n2 missing after second replay")
	}
	dl.Close()
}

func TestDeltaLog_ConcurrentSafe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "delta.log")

	dl, err := OpenDeltaLog(path)
	if err != nil {
		t.Fatalf("OpenDeltaLog: %v", err)
	}
	defer dl.Close()

	// Basic concurrency: emit from multiple goroutines
	done := make(chan struct{})
	const count = 20
	for i := 0; i < count; i++ {
		go func(i int) {
			_ = dl.Emit([]Event{
				{Type: NodeCreated, Node: &Node{ID: NodeID(string(rune('A' + i))), Label: "Entity"}},
			})
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < count; i++ {
		<-done
	}

	g := New()
	sink := NewIndexManagerFromGraph(g)
	if err := dl.Replay(sink); err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(g.AllNodes()) != count {
		t.Errorf("expected %d nodes after concurrent emit, got %d", count, len(g.AllNodes()))
	}
}

// ── LoggingSink ───────────────────────────────────────────

func TestLoggingSink_DelegatesToBoth(t *testing.T) {
	var primaryCalls, secondaryCalls int

	primary := &recordingSink{fn: func(events []Event) error {
		primaryCalls++
		return nil
	}}
	secondary := &recordingSink{fn: func(events []Event) error {
		secondaryCalls++
		return nil
	}}

	sink := NewLoggingSink(primary, secondary)
	if err := sink.Emit([]Event{{Type: NodeCreated, Node: &Node{ID: "x", Label: "Test"}}}); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	if primaryCalls != 1 {
		t.Errorf("primary called %d times, want 1", primaryCalls)
	}
	if secondaryCalls != 1 {
		t.Errorf("secondary called %d times, want 1", secondaryCalls)
	}
}

func TestLoggingSink_SecondaryErrorStopsPrimary(t *testing.T) {
	secondaryErr := &recordingSink{fn: func(events []Event) error {
		return errSecondary
	}}
	primaryNever := &recordingSink{fn: func(events []Event) error {
		t.Error("primary should not be called when secondary fails")
		return nil
	}}
	sink := NewLoggingSink(primaryNever, secondaryErr)
	err := sink.Emit([]Event{{Type: NodeCreated, Node: &Node{ID: "x", Label: "Test"}}})
	if err != errSecondary {
		t.Errorf("expected errSecondary, got %v", err)
	}
}

type recordingSink struct {
	fn func([]Event) error
}

func (s *recordingSink) Emit(events []Event) error {
	return s.fn(events)
}

var errSecondary = &sinkError{}

type sinkError struct{}

func (e *sinkError) Error() string { return "secondary error" }

// ── LoggingSink integration with IndexManager ─────────────

func TestLoggingSink_BuildAllWritesDelta(t *testing.T) {
	dir := t.TempDir()
	deltaPath := filepath.Join(dir, "delta.log")
	deltaLog, err := OpenDeltaLog(deltaPath)
	if err != nil {
		t.Fatalf("OpenDeltaLog: %v", err)
	}
	defer deltaLog.Close()

	mgr := NewIndexManager()
	mgr.RegisterAdapter(&testAdapter{
		name:   "test",
		schema: &GraphSchema{NodeLabels: []Label{"Entity"}},
		events: []Event{
			{Type: NodeCreated, Node: &Node{ID: "e1", Label: "Entity"}},
			{Type: NodeCreated, Node: &Node{ID: "e2", Label: "Entity"}},
		},
	})

	sink := NewLoggingSink(mgr, deltaLog)

	if err := mgr.BuildAll(context.Background(), sink); err != nil {
		t.Fatalf("BuildAll: %v", err)
	}

	// Events should be applied to graph
	if mgr.Query().GetNode("e1") == nil {
		t.Error("e1 missing from graph after BuildAll")
	}

	// Events should be in delta log
	g2 := New()
	sink2 := NewIndexManagerFromGraph(g2)
	if err := deltaLog.Replay(sink2); err != nil {
		t.Fatalf("Replay delta: %v", err)
	}
	if g2.GetNode("e1") == nil {
		t.Error("e1 missing from delta replay")
	}
	if g2.GetNode("e2") == nil {
		t.Error("e2 missing from delta replay")
	}
}

// ── RestoreFromSnapshot ───────────────────────────────────

func TestRestoreFromSnapshot_WithoutDelta(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "graph.gob")
	deltaPath := filepath.Join(dir, "delta.log")

	g := buildTestGraph()
	data, err := MarshalSnapshot(g)
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	if err := os.WriteFile(snapPath, data, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	restored, err := RestoreFromSnapshot(snapPath, deltaPath)
	if err != nil {
		t.Fatalf("RestoreFromSnapshot: %v", err)
	}
	if restored == nil {
		t.Fatal("RestoreFromSnapshot returned nil")
	}
	if restored.GetNode("e1") == nil {
		t.Error("e1 missing from restored graph")
	}
}

func TestRestoreFromSnapshot_WithDelta(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "graph.gob")
	deltaPath := filepath.Join(dir, "delta.log")

	// Save snapshot with n1
	g := New()
	g.AddNode("n1", "Entity", nil)
	data, err := MarshalSnapshot(g)
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	if err := os.WriteFile(snapPath, data, 0600); err != nil {
		t.Fatalf("WriteFile snapshot: %v", err)
	}

	// Write delta with n2
	dl, err := OpenDeltaLog(deltaPath)
	if err != nil {
		t.Fatalf("OpenDeltaLog: %v", err)
	}
	if err := dl.Emit([]Event{{Type: NodeCreated, Node: &Node{ID: "n2", Label: "Attribute"}}}); err != nil {
		t.Fatalf("Emit delta: %v", err)
	}
	dl.Close()

	// Restore should return snapshot + delta
	restored, err := RestoreFromSnapshot(snapPath, deltaPath)
	if err != nil {
		t.Fatalf("RestoreFromSnapshot: %v", err)
	}
	if restored == nil {
		t.Fatal("RestoreFromSnapshot returned nil")
	}
	if restored.GetNode("n1") == nil {
		t.Error("n1 missing (from snapshot)")
	}
	if restored.GetNode("n2") == nil {
		t.Error("n2 missing (from delta)")
	}
}

func TestRestoreFromSnapshot_MissingSnapshot(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "nonexistent.gob")
	deltaPath := filepath.Join(dir, "delta.log")

	restored, err := RestoreFromSnapshot(snapPath, deltaPath)
	if err != nil {
		t.Fatalf("RestoreFromSnapshot: %v", err)
	}
	if restored != nil {
		t.Fatal("expected nil when snapshot is missing")
	}
}

// ── Compact ───────────────────────────────────────────────

func TestCompact_MergesDeltaIntoSnapshot(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "graph.gob")
	deltaPath := filepath.Join(dir, "delta.log")

	// Snapshot: n1
	g := New()
	g.AddNode("n1", "Entity", nil)
	data, _ := MarshalSnapshot(g)
	os.WriteFile(snapPath, data, 0600)

	// Delta: add n2
	dl, _ := OpenDeltaLog(deltaPath)
	dl.Emit([]Event{{Type: NodeCreated, Node: &Node{ID: "n2", Label: "Attribute"}}})
	dl.Close()

	// Compact
	if err := Compact(snapPath, deltaPath); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	// Verify snapshot now contains n2
	restored, err := RestoreFromSnapshot(snapPath, deltaPath)
	if err != nil {
		t.Fatalf("RestoreFromSnapshot after compact: %v", err)
	}
	if restored.GetNode("n1") == nil {
		t.Error("n1 missing after compact")
	}
	if restored.GetNode("n2") == nil {
		t.Error("n2 missing after compact")
	}

	// Verify delta is empty
	dl2, _ := OpenDeltaLog(deltaPath)
	if sz := dl2.Size(); sz != 0 {
		t.Errorf("expected empty delta after compact, got size %d", sz)
	}
	dl2.Close()
}

func TestCompact_EmptyDeltaDoesNothing(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "graph.gob")
	deltaPath := filepath.Join(dir, "delta.log")

	g := New()
	g.AddNode("n1", "Entity", nil)
	data, _ := MarshalSnapshot(g)
	os.WriteFile(snapPath, data, 0600)

	// Create empty delta
	dl, _ := OpenDeltaLog(deltaPath)
	dl.Close()

	if err := Compact(snapPath, deltaPath); err != nil {
		t.Fatalf("Compact on empty delta: %v", err)
	}

	// Snapshot should be unchanged
	restored, _ := RestoreFromSnapshot(snapPath, deltaPath)
	if restored.GetNode("n1") == nil {
		t.Error("n1 missing after empty compact")
	}
}

// ── End-to-end: BuildAll → delta → restore ────────────────

func TestBuildAllThenRestoreFromDelta(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "graph.gob")
	deltaPath := filepath.Join(dir, "delta.log")

	// Build graph with delta logging
	deltaLog, err := OpenDeltaLog(deltaPath)
	if err != nil {
		t.Fatalf("OpenDeltaLog: %v", err)
	}
	defer deltaLog.Close()

	mgr := NewIndexManager()
	mgr.RegisterAdapter(&testAdapter{
		name:   "test",
		schema: &GraphSchema{NodeLabels: []Label{"Entity"}},
		events: []Event{
			{Type: NodeCreated, Node: &Node{ID: "e1", Label: "Entity"}},
			{Type: EdgeCreated, Edge: &Edge{ID: "edge1", From: "e1", To: "e2", Type: "SELF"}},
		},
	})

	sink := NewLoggingSink(mgr, deltaLog)
	if err := mgr.BuildAll(context.Background(), sink); err != nil {
		t.Fatalf("BuildAll: %v", err)
	}

	// Save snapshot and reset delta (simulating what buildGraph does)
	data, err := MarshalSnapshot(mgr.Query())
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	if err := os.WriteFile(snapPath, data, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := deltaLog.Reset(); err != nil {
		t.Fatalf("Reset delta: %v", err)
	}

	// Now simulate a second session: restore from snapshot + delta
	restored, err := RestoreFromSnapshot(snapPath, deltaPath)
	if err != nil {
		t.Fatalf("RestoreFromSnapshot: %v", err)
	}
	if restored == nil {
		t.Fatal("RestoreFromSnapshot returned nil")
	}
	if restored.GetNode("e1") == nil {
		t.Error("e1 missing after restore")
	}
}
