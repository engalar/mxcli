package testscheduler

import (
	"github.com/mendixlabs/mxcli/internal/testresource"
	"time"
)

type ResourceLimit struct {
	MaxParallelIO  int
	MaxParallelCPU int
	MaxHeapMB      int
}

type PlanSlot struct {
	Name     string
	Duration time.Duration
	Profile  testresource.Profile
}

type Lane struct {
	Name  string
	Tests []PlanSlot
}

type Schedule struct {
	Lanes  []Lane
	Limits ResourceLimit
}
