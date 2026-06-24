package testscheduler

import (
	"github.com/mendixlabs/mxcli/internal/testresource"
	"testing"
)

type mockProfileLoader struct {
	profiles []testresource.Profile
}

func (m *mockProfileLoader) Load(name string) (testresource.Profile, bool) {
	for _, p := range m.profiles {
		if p.Name == name {
			return p, true
		}
	}
	return testresource.Profile{}, false
}

func TestPlanner_AssignsIOHeavyToIOLane(t *testing.T) {
	profiles := []testresource.Profile{
		{Name: "IO_Test", ReadBytes: 50_000_000, WriteBytes: 0, DurationMs: 1000, CPUTimeMs: 100},
	}
	limits := ResourceLimit{MaxParallelIO: 2, MaxParallelCPU: 4}
	p := NewPlanner(nil, limits)
	schedule := p.Plan(profiles)

	if len(schedule.Lanes) == 0 {
		t.Fatal("expected at least 1 lane")
	}

	var found bool
	for _, lane := range schedule.Lanes {
		for _, slot := range lane.Tests {
			if slot.Name == "IO_Test" {
				if lane.Name != "io" {
					t.Errorf("IO_Test in lane %q, want 'io'", lane.Name)
				}
				found = true
			}
		}
	}
	if !found {
		t.Error("IO_Test not assigned to any lane")
	}
}

func TestPlanner_AssignsCPUHeavyToCPULane(t *testing.T) {
	profiles := []testresource.Profile{
		{Name: "CPU_Test", DurationMs: 1000, CPUTimeMs: 800, ReadBytes: 0},
	}
	limits := ResourceLimit{MaxParallelIO: 2, MaxParallelCPU: 4}
	p := NewPlanner(nil, limits)
	schedule := p.Plan(profiles)

	var found bool
	for _, lane := range schedule.Lanes {
		for _, slot := range lane.Tests {
			if slot.Name == "CPU_Test" {
				if lane.Name != "cpu" {
					t.Errorf("CPU_Test in lane %q, want 'cpu'", lane.Name)
				}
				found = true
			}
		}
	}
	if !found {
		t.Error("CPU_Test not assigned to any lane")
	}
}

func TestPlanner_StaggersByDuration(t *testing.T) {
	profiles := []testresource.Profile{
		{Name: "Long", DurationMs: 10000, ReadBytes: 50_000_000},
		{Name: "Short", DurationMs: 1000, ReadBytes: 50_000_000},
	}
	limits := ResourceLimit{MaxParallelIO: 2, MaxParallelCPU: 4}
	p := NewPlanner(nil, limits)
	schedule := p.Plan(profiles)

	for _, lane := range schedule.Lanes {
		if lane.Name != "io" {
			continue
		}
		if len(lane.Tests) < 2 {
			continue
		}
		if lane.Tests[0].Duration < lane.Tests[1].Duration {
			t.Errorf("expected Long before Short by duration, got Long=%v Short=%v",
				lane.Tests[0].Duration, lane.Tests[1].Duration)
		}
	}
}
