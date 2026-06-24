package testresourceregistry

import (
	"github.com/mendixlabs/mxcli/internal/testresource"
	"github.com/mendixlabs/mxcli/internal/testscheduler"
)

type Registry struct {
	store     *testresource.Store
	planner   *testscheduler.Planner
	scheduler *testscheduler.Scheduler
	previous  map[string]testresource.Profile
}

func New(profileDir string, maxIO, maxCPU, maxHeapMB int) *Registry {
	store := testresource.NewStore(profileDir)
	limits := testscheduler.ResourceLimit{
		MaxParallelIO:  maxIO,
		MaxParallelCPU: maxCPU,
		MaxHeapMB:      maxHeapMB,
	}
	planner := testscheduler.NewPlanner(store, limits)
	reader := &testresource.ProcfsReader{}
	scheduler := testscheduler.New(maxIO, maxCPU, reader)
	return &Registry{
		store:     store,
		planner:   planner,
		scheduler: scheduler,
		previous:  make(map[string]testresource.Profile),
	}
}

func (r *Registry) Record(p testresource.Profile) error {
	if existing, ok := r.store.Load(p.Name); ok {
		r.previous[p.Name] = existing
	}
	return r.store.Save(p)
}

func (r *Registry) BuildSchedule() testscheduler.Schedule {
	profiles, err := r.store.List()
	if err != nil || len(profiles) == 0 {
		return testscheduler.Schedule{}
	}
	return r.planner.Plan(profiles)
}

func (r *Registry) CheckRegressions() ([]testresource.Diff, error) {
	profiles, err := r.store.List()
	if err != nil {
		return nil, err
	}
	var diffs []testresource.Diff
	for _, current := range profiles {
		baseline, ok := r.previous[current.Name]
		if !ok {
			continue
		}
		diff, _ := r.store.Compare(baseline, current)
		diffs = append(diffs, diff)
	}
	return diffs, nil
}

func (r *Registry) Scheduler() *testscheduler.Scheduler {
	return r.scheduler
}
