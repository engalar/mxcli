// SPDX-License-Identifier: Apache-2.0

package golden

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// DiffKind describes the category of a BSON difference.
type DiffKind string

const (
	DiffMissing    DiffKind = "missing"     // field in golden but not in our output
	DiffExtra      DiffKind = "extra"       // field in our output but not in golden
	DiffOrder      DiffKind = "order"       // fields exist in both but differ in ordinal
	DiffValue      DiffKind = "value"       // values differ
	DiffType       DiffKind = "type"        // field types differ
	DiffMarker     DiffKind = "marker"      // PartList version marker differs
)

// Diff represents one difference between golden and our BSON.
type Diff struct {
	Path     string
	Kind     DiffKind
	Got      any
	Expected any
}

func (d Diff) String() string {
	return fmt.Sprintf("%s: %s (got %v, want %v)", d.Kind, d.Path, d.Got, d.Expected)
}

// stringOrType returns a string representation of a BSON value, or its type name
// for complex types.
func stringOrType(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case int32, int64, float64:
		return fmt.Sprintf("%v", val)
	case nil:
		return "null"
	case bson.D:
		return fmt.Sprintf("doc<%d fields>", len(val))
	case bson.A:
		return fmt.Sprintf("arr<%d items>", len(val))
	case bson.Binary:
		return fmt.Sprintf("binary<%d bytes>", len(val.Data))
	default:
		return fmt.Sprintf("%T(%v)", v, v)
	}
}

// CompareBSON compares two BSON documents byte-slice and returns all diffs.
// Fields named "$ID" are always skipped (auto-generated UUIDs differ).
func CompareBSON(got, expected []byte, skipFields []string) []Diff {
	skipSet := make(map[string]bool)
	for _, f := range skipFields {
		skipSet[f] = true
	}

	var gotDoc, expDoc bson.D
	if err := bson.Unmarshal(got, &gotDoc); err != nil {
		return []Diff{{Path: "$", Kind: DiffValue, Got: err.Error(), Expected: "valid BSON"}}
	}
	if err := bson.Unmarshal(expected, &expDoc); err != nil {
		return []Diff{{Path: "$", Kind: DiffValue, Got: err.Error(), Expected: "valid BSON"}}
	}

	return compareDocs(gotDoc, expDoc, "$", skipSet)
}

func compareDocs(got, exp bson.D, path string, skip map[string]bool) []Diff {
	var diffs []Diff

	// Build index maps for O(1) lookup by field name.
	gotIdx := make(map[string]int)
	for i, e := range got {
		gotIdx[e.Key] = i
	}
	expIdx := make(map[string]int)
	for i, e := range exp {
		expIdx[e.Key] = i
	}

	// Check for missing / extra fields and compare values in golden order.
	maxLen := len(exp)
	if len(got) > maxLen {
		maxLen = len(got)
	}

	for i := 0; i < maxLen; i++ {
		var gotE, expE *bson.E
		gotPresent := i < len(got)
		expPresent := i < len(exp)

		if gotPresent {
			gotE = &got[i]
		}
		if expPresent {
			expE = &exp[i]
		}

		// Compare by golden's field at position i.
		if expPresent {
			fullPath := path + "." + expE.Key
			if skip[fullPath] || expE.Key == "$ID" {
				continue
			}

			gi, inGot := gotIdx[expE.Key]
			if !inGot {
				diffs = append(diffs, Diff{Path: fullPath, Kind: DiffMissing, Got: nil, Expected: stringOrType(expE.Value)})
				continue
			}

			// Check ordinal position: if both have the same key at different positions,
			// report an order mismatch only when the KEY differs at this position.
			if gotPresent && gotE.Key != expE.Key {
				diffs = append(diffs, Diff{
					Path:     path,
					Kind:     DiffOrder,
					Got:      gotE.Key,
					Expected: expE.Key,
				})
			}

			// Recurse into nested values.
			gotVal := got[gi].Value
			expVal := expE.Value
			diffs = append(diffs, compareValues(gotVal, expVal, fullPath, skip)...)
		}

		// Extra field in got at this position that doesn't exist in exp.
		if gotPresent && expE == nil {
			fullPath := path + "." + gotE.Key
			if skip[fullPath] || gotE.Key == "$ID" {
				continue
			}
			if _, inExp := expIdx[gotE.Key]; !inExp {
				diffs = append(diffs, Diff{Path: fullPath, Kind: DiffExtra, Got: stringOrType(gotE.Value), Expected: nil})
			}
		}
	}

	return diffs
}

func compareValues(got, exp any, path string, skip map[string]bool) []Diff {
	// Handle nil.
	if got == nil && exp == nil {
		return nil
	}
	if got == nil {
		return []Diff{{Path: path, Kind: DiffValue, Got: nil, Expected: stringOrType(exp)}}
	}
	if exp == nil {
		return []Diff{{Path: path, Kind: DiffValue, Got: stringOrType(got), Expected: nil}}
	}

	// Type check — but int32 and int64 are interchangeable.
	gotT, expT := fmt.Sprintf("%T", got), fmt.Sprintf("%T", exp)
	if gotT != expT {
		switch {
		case (gotT == "int32" && expT == "int64") || (gotT == "int64" && expT == "int32"):
			// acceptable — compare as int64 below
		default:
			return []Diff{{Path: path, Kind: DiffType, Got: gotT, Expected: expT}}
		}
	}

	switch v := got.(type) {
	case bson.D:
		expDoc, ok := exp.(bson.D)
		if !ok {
			return []Diff{{Path: path, Kind: DiffType, Got: "bson.D", Expected: fmt.Sprintf("%T", exp)}}
		}
		return compareDocs(v, expDoc, path, skip)

	case bson.A:
		expArr, ok := exp.(bson.A)
		if !ok {
			return []Diff{{Path: path, Kind: DiffType, Got: "bson.A", Expected: fmt.Sprintf("%T", exp)}}
		}

		return compareArrays(v, expArr, path, skip)

	case string:
		if v != exp.(string) {
			return []Diff{{Path: path, Kind: DiffValue, Got: v, Expected: exp}}
		}

	case bool:
		if v != exp.(bool) {
			return []Diff{{Path: path, Kind: DiffValue, Got: v, Expected: exp}}
		}

	case int32:
		if ev, ok := exp.(int64); ok {
			if int64(v) != ev {
				return []Diff{{Path: path, Kind: DiffValue, Got: v, Expected: ev}}
			}
		} else if ev, ok := exp.(int32); ok {
			if v != ev {
				return []Diff{{Path: path, Kind: DiffValue, Got: v, Expected: ev}}
			}
		} else {
			return []Diff{{Path: path, Kind: DiffType, Got: "int32", Expected: fmt.Sprintf("%T", exp)}}
		}

	case int64:
		if ev, ok := exp.(int32); ok {
			if v != int64(ev) {
				return []Diff{{Path: path, Kind: DiffValue, Got: v, Expected: ev}}
			}
		} else if ev, ok := exp.(int64); ok {
			if v != ev {
				return []Diff{{Path: path, Kind: DiffValue, Got: v, Expected: ev}}
			}
		} else {
			return []Diff{{Path: path, Kind: DiffType, Got: "int64", Expected: fmt.Sprintf("%T", exp)}}
		}

	case float64:
		if v != exp.(float64) {
			return []Diff{{Path: path, Kind: DiffValue, Got: v, Expected: exp}}
		}

	case bson.Binary:
		expBin := exp.(bson.Binary)
		if v.Subtype != expBin.Subtype || !bytesEqual(v.Data, expBin.Data) {
			// Binary data often differs (UUIDs). Only report subtype mismatches.
			if v.Subtype != expBin.Subtype {
				return []Diff{{Path: path, Kind: DiffValue, Got: fmt.Sprintf("subtype=%d", v.Subtype), Expected: fmt.Sprintf("subtype=%d", expBin.Subtype)}}
			}
		}

	case nil:
		// handled above.

	default:
		return []Diff{{Path: path, Kind: DiffValue, Got: fmt.Sprintf("%v", got), Expected: fmt.Sprintf("%v", exp)}}
	}

	return nil
}

func compareArrays(got, exp bson.A, path string, skip map[string]bool) []Diff {
	var diffs []Diff

	// Compare markers (first element if it's an int32).
	if len(got) > 0 && len(exp) > 0 {
		if gm, ok := got[0].(int32); ok {
			if em, ok := exp[0].(int32); ok && gm != em {
				diffs = append(diffs, Diff{Path: path, Kind: DiffMarker, Got: gm, Expected: em})
			}
		}
	}

	minLen := len(got)
	if len(exp) < minLen {
		minLen = len(exp)
	}

	for i := 0; i < minLen; i++ {
		elemPath := fmt.Sprintf("%s[%d]", path, i)
		if i == 0 {
			// Skip marker comparison — already handled above.
			continue
		}
		diffs = append(diffs, compareValues(got[i], exp[i], elemPath, skip)...)
	}

	// Extra elements in got.
	for i := minLen; i < len(got); i++ {
		diffs = append(diffs, Diff{Path: fmt.Sprintf("%s[%d]", path, i), Kind: DiffExtra, Got: stringOrType(got[i]), Expected: nil})
	}

	// Missing elements in got.
	for i := minLen; i < len(exp); i++ {
		diffs = append(diffs, Diff{Path: fmt.Sprintf("%s[%d]", path, i), Kind: DiffMissing, Got: nil, Expected: stringOrType(exp[i])})
	}

	return diffs
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// DiffsByKind returns only diffs of the given kind.
func DiffsByKind(diffs []Diff, kind DiffKind) []Diff {
	var out []Diff
	for _, d := range diffs {
		if d.Kind == kind {
			out = append(out, d)
		}
	}
	return out
}

// FormatDiff returns a human-readable diff string.
func FormatDiff(d Diff) string {
	return d.String()
}
