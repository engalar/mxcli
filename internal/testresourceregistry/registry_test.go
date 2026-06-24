package testresourceregistry

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/testresource"
)

func TestRegistry_RecordAndBuildSchedule(t *testing.T) {
	dir := t.TempDir()
	r := New(dir, 2, 4, 500)

	r.Record(testresource.Profile{
		Name: "TestA", ReadBytes: 50_000_000, DurationMs: 1000, CPUTimeMs: 100,
	})
	r.Record(testresource.Profile{
		Name: "TestB", DurationMs: 1000, CPUTimeMs: 800,
	})

	schedule := r.BuildSchedule()
	if len(schedule.Lanes) == 0 {
		t.Fatal("expected at least 1 lane")
	}
}

func TestRegistry_CheckRegressions_ReturnsDiffs(t *testing.T) {
	dir := t.TempDir()
	r := New(dir, 2, 4, 500)

	r.Record(testresource.Profile{
		Name: "TestA", HeapDelta: 1000, DurationMs: 100, CPUTimeMs: 50,
	})

	r.Record(testresource.Profile{
		Name: "TestA", HeapDelta: 2000, DurationMs: 100, CPUTimeMs: 50,
	})

	diffs, err := r.CheckRegressions()
	if err != nil {
		t.Fatalf("CheckRegressions: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected at least 1 diff")
	}
	if diffs[0].HeapDeltaPct < 90 || diffs[0].HeapDeltaPct > 110 {
		t.Errorf("HeapDeltaPct = %.1f, want ~100", diffs[0].HeapDeltaPct)
	}
}
