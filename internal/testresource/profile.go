package testresource

type Profile struct {
	Name          string  `json:"name"`
	DurationMs    float64 `json:"duration_ms"`
	HeapDelta     int64   `json:"heap_delta_bytes"`
	CPUTimeMs     float64 `json:"cpu_time_ms"`
	ReadBytes     int64   `json:"read_bytes"`
	WriteBytes    int64   `json:"write_bytes"`
	NumGoroutines int     `json:"num_goroutines"`
}

type resourceSnapshot struct {
	heapAlloc     uint64
	cpuTicks      int64
	ioReadBytes   int64
	ioWriteBytes  int64
	numGoroutines int
}

type Category int

const (
	CategoryUncategorized Category = iota
	CategoryIOHeavy
	CategoryCPUHeavy
	CategoryMixed
)
