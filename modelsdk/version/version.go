package version

import (
	"strconv"
	"strings"
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
	Properties map[string]PropertyVersionInfo
}
