package version

import (
	"strconv"
	"strings"
	"sync"
)

type Version struct {
	Major, Minor, Patch int
}

func Parse(s string) Version {
	parts := strings.SplitN(s, ".", 4)
	var v Version
	if len(parts) > 0 {
		v.Major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		v.Minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		v.Patch, _ = strconv.Atoi(parts[2])
	}
	return v
}

func (a Version) Compare(b Version) int {
	if a.Major != b.Major {
		return cmp(a.Major, b.Major)
	}
	if a.Minor != b.Minor {
		return cmp(a.Minor, b.Minor)
	}
	return cmp(a.Patch, b.Patch)
}

func (v Version) IsZero() bool { return v.Major == 0 && v.Minor == 0 && v.Patch == 0 }

func (v Version) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}

func cmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

type PropertyVersionInfo struct {
	Introduced string
	Deleted    string
	Required   bool
	Public     bool
}

func (p PropertyVersionInfo) IsAvailableIn(v Version) bool {
	if p.Deleted != "" && v.Compare(Parse(p.Deleted)) >= 0 {
		return false
	}
	if p.Introduced != "" && v.Compare(Parse(p.Introduced)) < 0 {
		return false
	}
	return true
}

type TypeVersionInfo struct {
	Introduced string // Mendix version when this type was introduced (empty = baseline)
	Deleted    string // Mendix version when this type was deleted (empty = still present)
	Properties map[string]PropertyVersionInfo
}

// PropertyVersioner is implemented by generated element types that carry
// per-property version constraints. The encoder uses this interface for
// zero-allocation, mutex-free version gating without consulting any global registry.
//
// camelName is the camelCase property key (e.g. "boundaryEvents").
// Returns (PropertyVersionInfo{}, false) when the property has no constraint.
type PropertyVersioner interface {
	PropertyVersionInfo(camelName string) (PropertyVersionInfo, bool)
}

// DefaultVersionRegistry is the global registry of TypeVersionInfo, kept for
// diagnostic / tooling use. The encoder no longer consults it at runtime.
var DefaultVersionRegistry = &VersionRegistry{}

// VersionRegistry stores TypeVersionInfo by BSON type name.
type VersionRegistry struct {
	mu      sync.RWMutex
	entries map[string]TypeVersionInfo
}

// Register adds or replaces a TypeVersionInfo entry.
func (r *VersionRegistry) Register(typeName string, info TypeVersionInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entries == nil {
		r.entries = make(map[string]TypeVersionInfo)
	}
	r.entries[typeName] = info
}

// Lookup returns the TypeVersionInfo for a BSON type name, if registered.
func (r *VersionRegistry) Lookup(typeName string) (TypeVersionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.entries[typeName]
	return info, ok
}
