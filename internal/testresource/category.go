package testresource

const (
	ioReadThreshold   = 10 * 1024 * 1024 // 10MB
	ioWriteThreshold  = 1 * 1024 * 1024  // 1MB
	cpuRatioThreshold = 0.5
)

func Classify(p Profile) Category {
	isIO := p.ReadBytes > ioReadThreshold || p.WriteBytes > ioWriteThreshold
	isCPU := p.DurationMs > 0 && p.CPUTimeMs/p.DurationMs > cpuRatioThreshold

	switch {
	case isIO && isCPU:
		return CategoryMixed
	case isIO:
		return CategoryIOHeavy
	case isCPU:
		return CategoryCPUHeavy
	default:
		return CategoryUncategorized
	}
}
