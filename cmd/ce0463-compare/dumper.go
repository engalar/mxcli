package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

func dumpElement(elem element.Element) map[string]any {
	result := make(map[string]any)
	dumpRecursive(elem, "", result)
	return result
}

func dumpRecursive(elem element.Element, prefix string, out map[string]any) {
	for _, prop := range elem.Properties() {
		name := prop.Name()
		if name == "$ID" || name == "$Type" {
			continue
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		switch p := prop.(type) {
		case element.ChildProperty:
			if child := p.ChildElement(); child != nil {
				dumpRecursive(child, path, out)
			} else {
				out[path] = nil
			}

		case element.ChildListProperty:
			children := p.ChildElements()
			if len(children) > 0 {
				for i, child := range children {
					idxPath := fmt.Sprintf("%s[%d]", path, i)
					dumpRecursive(child, idxPath, out)
				}
			} else {
				out[path] = "[]"
			}

		default:
			if wp, ok := prop.(element.WritableProperty); ok {
				val := wp.BSONValue()
				if idStr, ok := val.(element.ID); ok {
					out[path] = "[ID:" + string(idStr) + "]"
				} else {
					out[path] = val
				}
			}
		}
	}
}

func compareDumps(a, b map[string]any) (onlyInA, onlyInB []string, different map[string][2]any) {
	different = make(map[string][2]any)

	allKeys := make(map[string]bool)
	for k := range a {
		allKeys[k] = true
	}
	for k := range b {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if strings.Contains(k, ".$ID") || strings.Contains(k, "TypePointer") ||
			strings.Contains(k, "$binary") || strings.Contains(k, "base64") ||
			strings.Contains(k, "subType") {
			continue
		}

		va, aOk := a[k]
		vb, bOk := b[k]

		if !aOk && bOk {
			onlyInB = append(onlyInB, k)
			continue
		}
		if aOk && !bOk {
			onlyInA = append(onlyInA, k)
			continue
		}

		sa := fmt.Sprintf("%v", va)
		sb := fmt.Sprintf("%v", vb)
		if sa != sb {
			different[k] = [2]any{va, vb}
		}
	}

	return onlyInA, onlyInB, different
}

func printDiff(onlyInA, onlyInB []string, different map[string][2]any) {
	if len(onlyInA) == 0 && len(onlyInB) == 0 && len(different) == 0 {
		fmt.Println("NO DIFFERENCES FOUND")
		return
	}

	if len(different) > 0 {
		fmt.Printf("\n=== FIELD VALUE DIFFERENCES (%d) ===\n", len(different))
		keys := make([]string, 0, len(different))
		for k := range different {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			vals := different[k]
			fmt.Printf("  %s:\n", k)
			fmt.Printf("    GOLDEN: %v (%T)\n", vals[0], vals[0])
			fmt.Printf("    BUILT:  %v (%T)\n", vals[1], vals[1])
		}
	}

	if len(onlyInA) > 0 {
		fmt.Printf("\n=== ONLY IN GOLDEN (%d) ===\n", len(onlyInA))
		for _, k := range onlyInA {
			fmt.Printf("  %s\n", k)
		}
	}

	if len(onlyInB) > 0 {
		fmt.Printf("\n=== ONLY IN BUILDER (%d) ===\n", len(onlyInB))
		for _, k := range onlyInB {
			fmt.Printf("  %s\n", k)
		}
	}
}

func filterDump(m map[string]any, keepID bool) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if !keepID && strings.HasPrefix(fmt.Sprintf("%v", v), "[ID:") {
			continue
		}
		result[k] = v
	}
	return result
}

func countDump(m map[string]any) (total, nonID int) {
	for _, v := range m {
		total++
		if !strings.HasPrefix(fmt.Sprintf("%v", v), "[ID:") {
			nonID++
		}
	}
	return
}
