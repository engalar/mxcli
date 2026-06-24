package testscheduler

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/testresource"
)

type Scheduler struct {
	ioToken  chan struct{}
	cpuToken chan struct{}
	reader   testresource.ResourceReader
}

func New(maxIO, maxCPU int, reader testresource.ResourceReader) *Scheduler {
	s := &Scheduler{
		ioToken:  make(chan struct{}, maxIO),
		cpuToken: make(chan struct{}, maxCPU),
		reader:   reader,
	}
	for i := 0; i < maxIO; i++ {
		s.ioToken <- struct{}{}
	}
	for i := 0; i < maxCPU; i++ {
		s.cpuToken <- struct{}{}
	}
	return s
}

func (s *Scheduler) AcquireIO(ctx context.Context) error {
	select {
	case <-s.ioToken:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) AcquireCPU(ctx context.Context) error {
	select {
	case <-s.cpuToken:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) ReleaseIO() {
	select {
	case s.ioToken <- struct{}{}:
	default:
		// Bucket full — release would overflow. Safe to drop:
		// the token was already returned by another path.
	}
}

func (s *Scheduler) ReleaseCPU() {
	select {
	case s.cpuToken <- struct{}{}:
	default:
	}
}

func (s *Scheduler) Adjust(limits ResourceLimit) {
	newIO := make(chan struct{}, limits.MaxParallelIO)
	newCPU := make(chan struct{}, limits.MaxParallelCPU)
	for i := 0; i < limits.MaxParallelIO; i++ {
		newIO <- struct{}{}
	}
	for i := 0; i < limits.MaxParallelCPU; i++ {
		newCPU <- struct{}{}
	}
	s.ioToken = newIO
	s.cpuToken = newCPU
}
