// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"sort"
	"strings"
)

func DiffArray(path string, golden, actual []any, out *[]FieldDiff) {
	switch {
	case allRefs(golden) && allRefs(actual):
		diffSetRefs(path, golden, actual, out)
	case hasMapsWithName(golden) || hasMapsWithName(actual):
		diffByName(path, golden, actual, out)
	default:
		diffByPosition(path, golden, actual, out)
	}
}

func allRefs(arr []any) bool {
	if len(arr) == 0 {
		return false
	}
	for _, v := range arr {
		s, ok := v.(string)
		if !ok || !strings.HasPrefix(s, "<ref:") {
			return false
		}
	}
	return true
}

func hasMapsWithName(arr []any) bool {
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			if _, has := m["Name"]; has {
				return true
			}
		}
	}
	return false
}

func diffSetRefs(path string, golden, actual []any, out *[]FieldDiff) {
	gs := make(map[string]bool, len(golden))
	for _, v := range golden {
		if s, ok := v.(string); ok {
			gs[s] = true
		}
	}
	as := make(map[string]bool, len(actual))
	for _, v := range actual {
		if s, ok := v.(string); ok {
			as[s] = true
		}
	}
	removed := sortedDiff(gs, as)
	added := sortedDiff(as, gs)
	for _, r := range removed {
		*out = append(*out, FieldDiff{Path: path, Golden: r, Actual: "", Kind: DiffRemoved})
	}
	for _, a := range added {
		*out = append(*out, FieldDiff{Path: path, Golden: "", Actual: a, Kind: DiffAdded})
	}
}

func sortedDiff(a, b map[string]bool) []string {
	var result []string
	for k := range a {
		if !b[k] {
			result = append(result, k)
		}
	}
	sort.Strings(result)
	return result
}

func diffByName(path string, golden, actual []any, out *[]FieldDiff) {
	gByName := indexByName(golden)
	aByName := indexByName(actual)

	seen := make(map[string]bool)
	var names []string
	for _, v := range golden {
		if m, ok := v.(map[string]any); ok {
			if n, _ := m["Name"].(string); n != "" && !seen[n] {
				names = append(names, n)
				seen[n] = true
			}
		}
	}
	for _, v := range actual {
		if m, ok := v.(map[string]any); ok {
			if n, _ := m["Name"].(string); n != "" && !seen[n] {
				names = append(names, n)
				seen[n] = true
			}
		}
	}

	for _, name := range names {
		gv, gok := gByName[name]
		av, aok := aByName[name]
		elemPath := fmt.Sprintf("%s[%s]", path, name)
		switch {
		case gok && !aok:
			*out = append(*out, FieldDiff{Path: elemPath, Golden: fmt.Sprintf("%v", gv), Kind: DiffRemoved})
		case !gok && aok:
			*out = append(*out, FieldDiff{Path: elemPath, Actual: fmt.Sprintf("%v", av), Kind: DiffAdded})
		default:
			diffMaps(elemPath, gv, av, true, out)
		}
	}
}

func indexByName(arr []any) map[string]map[string]any {
	m := make(map[string]map[string]any, len(arr))
	for _, v := range arr {
		if doc, ok := v.(map[string]any); ok {
			if name, _ := doc["Name"].(string); name != "" {
				m[name] = doc
			}
		}
	}
	return m
}

func diffByPosition(path string, golden, actual []any, out *[]FieldDiff) {
	if len(golden) != len(actual) {
		*out = append(*out, FieldDiff{
			Path:   path + ".length",
			Golden: fmt.Sprintf("%d", len(golden)),
			Actual: fmt.Sprintf("%d", len(actual)),
			Kind:   DiffChanged,
		})
	}
}

// diffMaps recursively compares two map[string]any values.
// skipName=true skips the "Name" key (used when elements are aligned by Name).
func diffMaps(path string, golden, actual map[string]any, skipName bool, out *[]FieldDiff) {
	allKeys := make(map[string]bool)
	for k := range golden {
		allKeys[k] = true
	}
	for k := range actual {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if skipName && k == "Name" {
			continue
		}
		gv, gok := golden[k]
		av, aok := actual[k]
		fp := path + "." + k
		switch {
		case gok && !aok:
			*out = append(*out, FieldDiff{Path: fp, Golden: fmt.Sprintf("%v", gv), Kind: DiffRemoved})
		case !gok && aok:
			*out = append(*out, FieldDiff{Path: fp, Actual: fmt.Sprintf("%v", av), Kind: DiffAdded})
		default:
			diffValues(fp, gv, av, out)
		}
	}
}

func diffValues(path string, g, a any, out *[]FieldDiff) {
	gm, gok := g.(map[string]any)
	am, aok := a.(map[string]any)
	if gok && aok {
		diffMaps(path, gm, am, false, out)
		return
	}
	ga, gaok := g.([]any)
	aa, aaok := a.([]any)
	if gaok && aaok {
		DiffArray(path, ga, aa, out)
		return
	}
	gs := fmt.Sprintf("%v", g)
	as := fmt.Sprintf("%v", a)
	if gs != as {
		*out = append(*out, FieldDiff{Path: path, Golden: gs, Actual: as, Kind: DiffChanged})
	}
}
