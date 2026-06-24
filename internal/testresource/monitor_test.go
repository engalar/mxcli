package testresource

import (
	"runtime"
	"testing"
)

type mockReader struct {
	cpuTicks   int64
	readBytes  int64
	writeBytes int64
}

func (m *mockReader) Read() (ResourceSnapshot, error) {
	return ResourceSnapshot{
		CPUTicks:   m.cpuTicks,
		ReadBytes:  m.readBytes,
		WriteBytes: m.writeBytes,
	}, nil
}

func TestMonitor_Done_ReturnsProfileWithName(t *testing.T) {
	m := NewMonitor(t, WithResourceReader(&mockReader{}))
	p := m.Done()
	if p.Name != t.Name() {
		t.Errorf("Name = %q, want %q", p.Name, t.Name())
	}
}

func TestMonitor_Done_CapturesHeapDelta(t *testing.T) {
	m := NewMonitor(t, WithResourceReader(&mockReader{}))
	buf := make([]byte, 1024*1024) // allocate 1MB to create a measurable delta
	p := m.Done()
	runtime.KeepAlive(buf)
	if p.HeapDelta <= 0 {
		t.Errorf("HeapDelta = %d, want > 0", p.HeapDelta)
	}
}

func TestMonitor_Done_CapturesCPUTime(t *testing.T) {
	reader := &mockReader{cpuTicks: 150, readBytes: 0, writeBytes: 0}
	m := NewMonitor(t, WithResourceReader(reader))
	reader.cpuTicks = 650 // simulate 500 ticks consumed
	p := m.Done()
	if p.CPUTimeMs <= 0 {
		t.Errorf("CPUTimeMs = %f, want > 0", p.CPUTimeMs)
	}
}
