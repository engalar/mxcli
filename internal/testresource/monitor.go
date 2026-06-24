package testresource

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type ResourceReader interface {
	Read() (ResourceSnapshot, error)
}

type ResourceSnapshot struct {
	CPUTicks   int64
	ReadBytes  int64
	WriteBytes int64
}

type MonitorOption func(*Monitor)

func WithResourceReader(rr ResourceReader) MonitorOption {
	return func(m *Monitor) {
		m.reader = rr
	}
}

type Monitor struct {
	t       *testing.T
	start   resourceSnapshot
	reader  ResourceReader
	startAt time.Time
}

func NewMonitor(t *testing.T, opts ...MonitorOption) *Monitor {
	m := &Monitor{
		t:       t,
		reader:  &procfsReader{},
		startAt: time.Now(),
	}
	for _, opt := range opts {
		opt(m)
	}
	m.captureStart()
	return m
}

func (m *Monitor) captureStart() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.start.heapAlloc = ms.HeapAlloc
	if snap, err := m.reader.Read(); err == nil {
		m.start.cpuTicks = snap.CPUTicks
		m.start.ioReadBytes = snap.ReadBytes
		m.start.ioWriteBytes = snap.WriteBytes
	}
	m.start.numGoroutines = runtime.NumGoroutine()
}

func (m *Monitor) Done() Profile {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	p := Profile{
		Name:          m.t.Name(),
		DurationMs:    float64(time.Since(m.startAt).Microseconds()) / 1000.0,
		HeapDelta:     int64(ms.HeapAlloc - m.start.heapAlloc),
		NumGoroutines: runtime.NumGoroutine() - m.start.numGoroutines,
	}

	if snap, err := m.reader.Read(); err == nil {
		const ticksPerSec = 100.0 // CLK_TCK on Linux
		cpuTicks := snap.CPUTicks - m.start.cpuTicks
		if cpuTicks < 0 {
			cpuTicks = 0
		}
		p.CPUTimeMs = float64(cpuTicks) * 1000.0 / ticksPerSec
		p.ReadBytes = snap.ReadBytes - m.start.ioReadBytes
		if p.ReadBytes < 0 {
			p.ReadBytes = 0
		}
		p.WriteBytes = snap.WriteBytes - m.start.ioWriteBytes
		if p.WriteBytes < 0 {
			p.WriteBytes = 0
		}
	}

	return p
}

type procfsReader struct{}

func (r *procfsReader) Read() (ResourceSnapshot, error) {
	var snap ResourceSnapshot

	// CPU ticks from /proc/self/stat
	if data, err := os.ReadFile("/proc/self/stat"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 15 {
			utime, _ := strconv.ParseInt(fields[13], 10, 64) // field 14 in /proc/self/stat
			stime, _ := strconv.ParseInt(fields[14], 10, 64) // field 15
			snap.CPUTicks = utime + stime
		}
	} else {
		return snap, fmt.Errorf("read /proc/self/stat: %w", err)
	}

	// I/O bytes from /proc/self/io
	if data, err := os.ReadFile("/proc/self/io"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "read_bytes:") {
				fmt.Sscanf(line, "read_bytes: %d", &snap.ReadBytes)
			} else if strings.HasPrefix(line, "write_bytes:") {
				fmt.Sscanf(line, "write_bytes: %d", &snap.WriteBytes)
			}
		}
	}

	return snap, nil
}
