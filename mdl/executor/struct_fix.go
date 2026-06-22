package executor

import "sort"

// toInt converts a catalog result value to int (stub for catalog removal).
func toInt(v any) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

// sortStrings sorts a string slice (replaces deprecated sort.Strings wrapper).
func sortStrings(s []string) { sort.Strings(s) }
