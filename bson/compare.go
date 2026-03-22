package bson

import (
	"fmt"
	"strings"
)

// DiffType represents the kind of difference between two BSON trees.
type DiffType int

const (
	// OnlyInLeft indicates a field present only in the left document.
	OnlyInLeft DiffType = iota
	// OnlyInRight indicates a field present only in the right document.
	OnlyInRight
	// ValueMismatch indicates the same key has different values.
	ValueMismatch
	// TypeMismatch indicates the same key has different Go types.
	TypeMismatch
)

func (d DiffType) String() string {
	switch d {
	case OnlyInLeft:
		return "only-in-left"
	case OnlyInRight:
		return "only-in-right"
	case ValueMismatch:
		return "value-mismatch"
	case TypeMismatch:
		return "type-mismatch"
	default:
		return "unknown"
	}
}

// Diff represents a single difference between two BSON trees.
type Diff struct {
	Path       string
	Type       DiffType
	LeftValue  string // formatted value (empty when OnlyInRight)
	RightValue string // formatted value (empty when OnlyInLeft)
}

// CompareOptions controls what to include in comparison.
type CompareOptions struct {
	IncludeAll bool // if true, include structural/layout fields
}

// defaultSkipFields are fields skipped by default (structural + layout).
var defaultSkipFields = map[string]bool{
	"$ID":                 true,
	"PersistentId":        true,
	"RelativeMiddlePoint": true,
	"Size":                true,
}

// Compare performs a recursive diff of two BSON document trees.
// Input maps come from bson.Unmarshal into bson.M (map[string]any).
func Compare(left, right map[string]any, opts CompareOptions) []Diff {
	return compareMaps(left, right, "", opts)
}

// compareMaps recursively compares two maps.
func compareMaps(left, right map[string]any, path string, opts CompareOptions) []Diff {
	var diffs []Diff

	for key, leftVal := range left {
		if !opts.IncludeAll && defaultSkipFields[key] {
			continue
		}

		fullPath := joinPath(path, key)

		rightVal, exists := right[key]
		if !exists {
			diffs = append(diffs, Diff{
				Path:      fullPath,
				Type:      OnlyInLeft,
				LeftValue: FormatValue(leftVal),
			})
			continue
		}

		diffs = append(diffs, compareValues(leftVal, rightVal, fullPath, opts)...)
	}

	for key, rightVal := range right {
		if !opts.IncludeAll && defaultSkipFields[key] {
			continue
		}

		if _, exists := left[key]; exists {
			continue
		}

		fullPath := joinPath(path, key)
		diffs = append(diffs, Diff{
			Path:       fullPath,
			Type:       OnlyInRight,
			RightValue: FormatValue(rightVal),
		})
	}

	return diffs
}

// compareValues compares two values at the same path.
func compareValues(leftVal, rightVal any, path string, opts CompareOptions) []Diff {
	if leftVal == nil && rightVal == nil {
		return nil
	}
	if leftVal == nil {
		return []Diff{{
			Path:       path,
			Type:       ValueMismatch,
			LeftValue:  "null",
			RightValue: FormatValue(rightVal),
		}}
	}
	if rightVal == nil {
		return []Diff{{
			Path:       path,
			Type:       ValueMismatch,
			LeftValue:  FormatValue(leftVal),
			RightValue: "null",
		}}
	}

	// Both are maps → recurse
	leftMap, leftIsMap := toMap(leftVal)
	rightMap, rightIsMap := toMap(rightVal)
	if leftIsMap && rightIsMap {
		return compareMaps(leftMap, rightMap, path, opts)
	}

	// Both are slices → smart array matching
	leftSlice, leftIsSlice := toSlice(leftVal)
	rightSlice, rightIsSlice := toSlice(rightVal)
	if leftIsSlice && rightIsSlice {
		return compareSlices(leftSlice, rightSlice, path, opts)
	}

	// Type mismatch (one is map, other is slice, etc.)
	if leftIsMap != rightIsMap || leftIsSlice != rightIsSlice {
		return []Diff{{
			Path:       path,
			Type:       TypeMismatch,
			LeftValue:  FormatValue(leftVal),
			RightValue: FormatValue(rightVal),
		}}
	}

	// Scalar comparison
	leftStr := fmt.Sprintf("%v", leftVal)
	rightStr := fmt.Sprintf("%v", rightVal)
	if leftStr != rightStr {
		return []Diff{{
			Path:       path,
			Type:       ValueMismatch,
			LeftValue:  FormatValue(leftVal),
			RightValue: FormatValue(rightVal),
		}}
	}

	return nil
}

// compareSlices compares two slices with smart matching.
// For arrays of objects with $Type+Name fields, match by identity rather than index.
func compareSlices(left, right []any, path string, opts CompareOptions) []Diff {
	// Check if elements can be identity-matched (maps with $Type or Name)
	if canIdentityMatch(left) || canIdentityMatch(right) {
		return compareSlicesByIdentity(left, right, path, opts)
	}

	// Fall back to index-based comparison
	var diffs []Diff

	minLen := len(left)
	if len(right) < minLen {
		minLen = len(right)
	}

	for i := range minLen {
		elemPath := fmt.Sprintf("%s[%d]", path, i)
		diffs = append(diffs, compareValues(left[i], right[i], elemPath, opts)...)
	}

	for i := minLen; i < len(left); i++ {
		diffs = append(diffs, Diff{
			Path:      fmt.Sprintf("%s[%d]", path, i),
			Type:      OnlyInLeft,
			LeftValue: FormatValue(left[i]),
		})
	}

	for i := minLen; i < len(right); i++ {
		diffs = append(diffs, Diff{
			Path:       fmt.Sprintf("%s[%d]", path, i),
			Type:       OnlyInRight,
			RightValue: FormatValue(right[i]),
		})
	}

	return diffs
}

// identityKey builds a matching key from a map element using $Type and/or Name.
func identityKey(m map[string]any) string {
	typeName, _ := m["$Type"].(string)
	name, _ := m["Name"].(string)
	if typeName != "" && name != "" {
		return typeName + ":" + name
	}
	if typeName != "" {
		return typeName
	}
	if name != "" {
		return ":" + name
	}
	return ""
}

// canIdentityMatch returns true if at least one element in the slice is a map
// with a "$Type" or "Name" key.
func canIdentityMatch(slice []any) bool {
	for _, elem := range slice {
		m, ok := toMap(elem)
		if ok && identityKey(m) != "" {
			return true
		}
	}
	return false
}

// compareSlicesByIdentity matches array elements by $Type+Name identity.
func compareSlicesByIdentity(left, right []any, path string, opts CompareOptions) []Diff {
	var diffs []Diff

	type indexedMap struct {
		index int
		m     map[string]any
	}

	// Build identity → element mapping for both sides.
	// Elements without identity keys fall back to index-based comparison.
	leftByKey := map[string]indexedMap{}
	var leftUnkeyed []indexedMap
	for i, elem := range left {
		m, ok := toMap(elem)
		if !ok {
			leftUnkeyed = append(leftUnkeyed, indexedMap{i, nil})
			continue
		}
		key := identityKey(m)
		if key == "" {
			leftUnkeyed = append(leftUnkeyed, indexedMap{i, m})
			continue
		}
		leftByKey[key] = indexedMap{i, m}
	}

	rightByKey := map[string]indexedMap{}
	var rightUnkeyed []indexedMap
	for i, elem := range right {
		m, ok := toMap(elem)
		if !ok {
			rightUnkeyed = append(rightUnkeyed, indexedMap{i, nil})
			continue
		}
		key := identityKey(m)
		if key == "" {
			rightUnkeyed = append(rightUnkeyed, indexedMap{i, m})
			continue
		}
		rightByKey[key] = indexedMap{i, m}
	}

	// Match by identity key
	matchedRight := map[string]bool{}
	for key, lEntry := range leftByKey {
		rEntry, exists := rightByKey[key]
		if !exists {
			elemPath := fmt.Sprintf("%s[%s]", path, key)
			diffs = append(diffs, Diff{
				Path:      elemPath,
				Type:      OnlyInLeft,
				LeftValue: FormatValue(lEntry.m),
			})
			continue
		}
		matchedRight[key] = true
		elemPath := fmt.Sprintf("%s[%s]", path, key)
		diffs = append(diffs, compareMaps(lEntry.m, rEntry.m, elemPath, opts)...)
	}

	for key, rEntry := range rightByKey {
		if matchedRight[key] {
			continue
		}
		elemPath := fmt.Sprintf("%s[%s]", path, key)
		diffs = append(diffs, Diff{
			Path:       elemPath,
			Type:       OnlyInRight,
			RightValue: FormatValue(rEntry.m),
		})
	}

	// Handle unkeyed elements by index
	unkeyedMin := len(leftUnkeyed)
	if len(rightUnkeyed) < unkeyedMin {
		unkeyedMin = len(rightUnkeyed)
	}
	for i := range unkeyedMin {
		elemPath := fmt.Sprintf("%s[%d]", path, leftUnkeyed[i].index)
		diffs = append(diffs, compareValues(left[leftUnkeyed[i].index], right[rightUnkeyed[i].index], elemPath, opts)...)
	}
	for i := unkeyedMin; i < len(leftUnkeyed); i++ {
		elemPath := fmt.Sprintf("%s[%d]", path, leftUnkeyed[i].index)
		diffs = append(diffs, Diff{
			Path:      elemPath,
			Type:      OnlyInLeft,
			LeftValue: FormatValue(left[leftUnkeyed[i].index]),
		})
	}
	for i := unkeyedMin; i < len(rightUnkeyed); i++ {
		elemPath := fmt.Sprintf("%s[%d]", path, rightUnkeyed[i].index)
		diffs = append(diffs, Diff{
			Path:       elemPath,
			Type:       OnlyInRight,
			RightValue: FormatValue(right[rightUnkeyed[i].index]),
		})
	}

	return diffs
}

// FormatDiffs returns a human-readable diff report.
func FormatDiffs(diffs []Diff) string {
	if len(diffs) == 0 {
		return "No differences found."
	}

	var sb strings.Builder

	onlyLeft, onlyRight, valueMismatch, typeMismatch := 0, 0, 0, 0

	for _, d := range diffs {
		switch d.Type {
		case OnlyInLeft:
			onlyLeft++
			fmt.Fprintf(&sb, "  - %s: %s (only in left)\n", d.Path, d.LeftValue)
		case OnlyInRight:
			onlyRight++
			fmt.Fprintf(&sb, "  + %s: %s (only in right)\n", d.Path, d.RightValue)
		case ValueMismatch:
			valueMismatch++
			fmt.Fprintf(&sb, "  ≠ %s: %s vs %s\n", d.Path, d.LeftValue, d.RightValue)
		case TypeMismatch:
			typeMismatch++
			fmt.Fprintf(&sb, "  ≠ %s: %s vs %s (type mismatch)\n", d.Path, d.LeftValue, d.RightValue)
		}
	}

	total := onlyLeft + onlyRight + valueMismatch + typeMismatch
	fmt.Fprintf(&sb, "\nSummary: %d differences, %d only-in-left, %d only-in-right, %d value-mismatches",
		total, onlyLeft, onlyRight, valueMismatch+typeMismatch)

	return sb.String()
}

// FormatValue formats a value for display, truncating long output.
func FormatValue(val any) string {
	switch v := val.(type) {
	case nil:
		return "null"
	case string:
		if len(v) > 100 {
			return fmt.Sprintf("%q... (truncated)", v[:100])
		}
		return fmt.Sprintf("%q", v)
	case map[string]any:
		typeName, _ := v["$Type"].(string)
		if typeName != "" {
			return fmt.Sprintf("{%s ...}", typeName)
		}
		return fmt.Sprintf("{map with %d keys}", len(v))
	case []any:
		return fmt.Sprintf("[array with %d elements]", len(v))
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		s := fmt.Sprintf("%v", val)
		if len(s) > 100 {
			return s[:100] + "... (truncated)"
		}
		return s
	}
}

// toMap attempts to convert a value to map[string]any.
func toMap(val any) (map[string]any, bool) {
	m, ok := val.(map[string]any)
	return m, ok
}

// toSlice attempts to convert a value to []any.
func toSlice(val any) ([]any, bool) {
	// bson.M unmarshals arrays as primitive.A which is []interface{}
	s, ok := val.([]any)
	return s, ok
}

// joinPath builds a dotted path.
func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
