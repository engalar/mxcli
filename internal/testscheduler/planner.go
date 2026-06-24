package testscheduler

import (
	"sort"
	"time"

	"github.com/mendixlabs/mxcli/internal/testresource"
)

type PlanningStrategy interface {
	Plan(profiles []testresource.Profile, limits ResourceLimit) Schedule
}

type Planner struct {
	store    testresource.ProfileLoader
	limits   ResourceLimit
	strategy PlanningStrategy
}

func NewPlanner(store testresource.ProfileLoader, limits ResourceLimit) *Planner {
	return &Planner{
		store:    store,
		limits:   limits,
		strategy: &DefaultStrategy{},
	}
}

func (p *Planner) Plan(profiles []testresource.Profile) Schedule {
	return p.strategy.Plan(profiles, p.limits)
}

type DefaultStrategy struct{}

func (s *DefaultStrategy) Plan(profiles []testresource.Profile, limits ResourceLimit) Schedule {
	var ioHeavy, cpuHeavy, mixed, uncat []testresource.Profile

	for _, p := range profiles {
		switch testresource.Classify(p) {
		case testresource.CategoryIOHeavy:
			ioHeavy = append(ioHeavy, p)
		case testresource.CategoryCPUHeavy:
			cpuHeavy = append(cpuHeavy, p)
		case testresource.CategoryMixed:
			mixed = append(mixed, p)
		default:
			uncat = append(uncat, p)
		}
	}

	sortByDurationDesc := func(profiles []testresource.Profile) {
		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].DurationMs > profiles[j].DurationMs
		})
	}
	sortByDurationDesc(ioHeavy)
	sortByDurationDesc(cpuHeavy)
	sortByDurationDesc(mixed)
	sortByDurationDesc(uncat)

	lanes := []Lane{}
	if len(ioHeavy) > 0 {
		lanes = append(lanes, laneFromProfiles("io", ioHeavy))
	}
	if len(cpuHeavy) > 0 {
		lanes = append(lanes, laneFromProfiles("cpu", cpuHeavy))
	}
	if len(mixed) > 0 {
		lanes = append(lanes, laneFromProfiles("mixed", mixed))
	}
	if len(uncat) > 0 {
		lanes = append(lanes, laneFromProfiles("uncategorized", uncat))
	}

	return Schedule{Lanes: lanes, Limits: limits}
}

func laneFromProfiles(name string, profiles []testresource.Profile) Lane {
	slots := make([]PlanSlot, len(profiles))
	for i, p := range profiles {
		slots[i] = PlanSlot{
			Name:     p.Name,
			Duration: time.Duration(p.DurationMs) * time.Millisecond,
			Profile:  p,
		}
	}
	return Lane{Name: name, Tests: slots}
}
