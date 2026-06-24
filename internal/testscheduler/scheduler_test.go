package testscheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/testresource"
)

type fakeResourceReader struct{}

func (f *fakeResourceReader) Read() (testresource.ResourceSnapshot, error) {
	return testresource.ResourceSnapshot{}, nil
}

func TestScheduler_AcquireIO_BlocksWhenFull(t *testing.T) {
	s := New(1, 2, &fakeResourceReader{})

	ctx := context.Background()
	if err := s.AcquireIO(ctx); err != nil {
		t.Fatalf("AcquireIO: %v", err)
	}

	done := make(chan bool, 1)
	go func() {
		err := s.AcquireIO(ctx)
		done <- (err == nil)
	}()

	select {
	case <-done:
		t.Fatal("AcquireIO should have blocked, but returned immediately")
	case <-time.After(50 * time.Millisecond):
	}

	s.ReleaseIO()

	select {
	case ok := <-done:
		if !ok {
			t.Fatal("AcquireIO failed after release")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("AcquireIO should have unblocked after release")
	}
}

func TestScheduler_ConcurrentIOAndCPU(t *testing.T) {
	s := New(2, 2, &fakeResourceReader{})
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				s.AcquireIO(ctx)
				defer s.ReleaseIO()
			} else {
				s.AcquireCPU(ctx)
				defer s.ReleaseCPU()
			}
			time.Sleep(10 * time.Millisecond)
		}(i)
	}
	wg.Wait()
}

func TestScheduler_Adjust_ChangesTokenCounts(t *testing.T) {
	s := New(4, 4, &fakeResourceReader{})

	s.Adjust(ResourceLimit{MaxParallelIO: 1, MaxParallelCPU: 1})

	ctx := context.Background()
	if err := s.AcquireIO(ctx); err != nil {
		t.Fatalf("AcquireIO: %v", err)
	}

	done := make(chan bool, 1)
	go func() {
		err := s.AcquireIO(ctx)
		done <- (err == nil)
	}()

	select {
	case <-done:
		t.Fatal("second AcquireIO should have blocked after adjust to 1")
	case <-time.After(50 * time.Millisecond):
	}
}
