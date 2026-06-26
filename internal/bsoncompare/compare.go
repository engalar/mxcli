// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// compareElements compares two element.Element trees.
func compareElements(path string, golden, actual element.Element, idMap IDMap, opts Options) []FieldDiff {
	if golden == nil && actual == nil {
		return nil
	}
	if golden == nil {
		return []FieldDiff{{Path: path, Kind: DiffAdded, Actual: describe(actual)}}
	}
	if actual == nil {
		return []FieldDiff{{Path: path, Kind: DiffRemoved, Golden: describe(golden)}}
	}
	if isBareBase(golden) || isBareBase(actual) {
		return compareBareBase(path, golden, actual)
	}
	if golden.TypeName() != actual.TypeName() {
		return []FieldDiff{{
			Path: path + ".$Type", Golden: golden.TypeName(),
			Actual: actual.TypeName(), Kind: DiffChanged,
		}}
	}
	return compareProperties(path, golden.Properties(), actual.Properties(), idMap, opts)
}

func isBareBase(elem element.Element) bool {
	_, ok := elem.(*element.Base)
	return ok
}

func compareBareBase(path string, golden, actual element.Element) []FieldDiff {
	var diffs []FieldDiff
	if golden.TypeName() != actual.TypeName() {
		diffs = append(diffs, FieldDiff{Path: path + ".$Type",
			Golden: golden.TypeName(), Actual: actual.TypeName(), Kind: DiffChanged})
	}
	gn := elementName(golden)
	an := elementName(actual)
	if gn != an {
		diffs = append(diffs, FieldDiff{Path: path + ".Name",
			Golden: gn, Actual: an, Kind: DiffChanged})
	}
	return diffs
}

func compareProperties(path string, gProps, aProps []element.Property, idMap IDMap, opts Options) []FieldDiff {
	gByKey := make(map[string]element.Property, len(gProps))
	for _, p := range gProps {
		gByKey[p.Name()] = p
	}
	aByKey := make(map[string]element.Property, len(aProps))
	for _, p := range aProps {
		aByKey[p.Name()] = p
	}

	keys := make(map[string]bool)
	for k := range gByKey {
		keys[k] = true
	}
	for k := range aByKey {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	var result []FieldDiff
	for _, k := range sorted {
		if shouldIgnore(k, opts) {
			continue
		}
		gp, gok := gByKey[k]
		ap, aok := aByKey[k]
		fp := path + "." + k
		switch {
		case gok && !aok:
			result = append(result, FieldDiff{Path: fp, Kind: DiffRemoved})
		case !gok && aok:
			result = append(result, FieldDiff{Path: fp, Kind: DiffAdded})
		default:
			result = append(result, compareProperty(fp, gp, ap, idMap, opts)...)
		}
	}
	return result
}

func compareProperty(path string, gp, ap element.Property, idMap IDMap, opts Options) []FieldDiff {
	if gc, ok := gp.(element.ChildProperty); ok {
		ac := ap.(element.ChildProperty)
		return compareElements(path, gc.ChildElement(), ac.ChildElement(), idMap, opts)
	}

	if gl, ok := gp.(element.ChildListProperty); ok {
		al := ap.(element.ChildListProperty)
		return compareChildList(path, gl.ChildElements(), al.ChildElements(), idMap, opts)
	}

	gw := gp.(element.WritableProperty)
	aw := ap.(element.WritableProperty)
	return compareBSONValue(path, gw.BSONValue(), aw.BSONValue(), idMap, opts)
}

func compareBSONValue(path string, g, a any, idMap IDMap, opts Options) []FieldDiff {
	switch gv := g.(type) {
	case string:
		av, _ := a.(string)
		if gv != av {
			return []FieldDiff{{Path: path, Golden: gv, Actual: av, Kind: DiffChanged}}
		}
	case int32:
		av, _ := a.(int32)
		if gv != av {
			return []FieldDiff{{Path: path, Golden: fmt.Sprintf("%d", gv), Actual: fmt.Sprintf("%d", av), Kind: DiffChanged}}
		}
	case bool:
		av, _ := a.(bool)
		if gv != av {
			return []FieldDiff{{Path: path, Golden: fmt.Sprintf("%v", gv), Actual: fmt.Sprintf("%v", av), Kind: DiffChanged}}
		}
	case float64:
		av, _ := a.(float64)
		if gv != av {
			return []FieldDiff{{Path: path, Golden: fmt.Sprintf("%v", gv), Actual: fmt.Sprintf("%v", av), Kind: DiffChanged}}
		}
	case bson.Binary:
		return compareBinary(path, gv, a, idMap)
	case element.ID:
		av, _ := a.(element.ID)
		// Resolve via IDMap for human-readable comparison, matching the
		// old bson.Binary behavior. Raw UUIDs that point to the same
		// entity produce equal labels (no spurious diff on non-deterministic
		// UUID regeneration).
		gl := idMap.LookupID(string(gv))
		al := idMap.LookupID(string(av))
		if gl != al {
			return []FieldDiff{{Path: path, Golden: gl, Actual: al, Kind: DiffChanged}}
		}
	case []string:
		av, _ := a.([]string)
		return compareStringSlice(path, gv, av)
	case []any:
		av, _ := a.([]any)
		return compareAnySlice(path, gv, av, idMap, opts)
	}
	return nil
}

func compareBinary(path string, gb bson.Binary, a any, idMap IDMap) []FieldDiff {
	ab, ok := a.(bson.Binary)
	if !ok {
		return []FieldDiff{{Path: path, Kind: DiffChanged}}
	}
	if len(gb.Data) == 16 && len(ab.Data) == 16 {
		gl := idMap.Lookup(gb.Data)
		al := idMap.Lookup(ab.Data)
		if gl != al {
			return []FieldDiff{{Path: path, Golden: gl, Actual: al, Kind: DiffChanged}}
		}
		return nil
	}
	if len(gb.Data) != len(ab.Data) {
		return []FieldDiff{{
			Path: path, Golden: fmt.Sprintf("<binary:%d>", len(gb.Data)),
			Actual: fmt.Sprintf("<binary:%d>", len(ab.Data)), Kind: DiffChanged,
		}}
	}
	return nil
}

func compareStringSlice(path string, g, a []string) []FieldDiff {
	if len(g) == 0 && len(a) == 0 {
		return nil
	}
	gs := make(map[string]int, len(g))
	for _, s := range g {
		gs[s]++
	}
	as := make(map[string]int, len(a))
	for _, s := range a {
		as[s]++
	}
	var diffs []FieldDiff
	for s, c := range gs {
		if as[s] != c {
			diffs = append(diffs, FieldDiff{Path: path, Golden: s, Kind: DiffRemoved})
		}
	}
	for s, c := range as {
		if gs[s] != c {
			diffs = append(diffs, FieldDiff{Path: path, Actual: s, Kind: DiffAdded})
		}
	}
	return diffs
}

func compareAnySlice(path string, g, a []any, idMap IDMap, opts Options) []FieldDiff {
	gItems := stripVersion(g)
	aItems := stripVersion(a)
	gStrs := make([]string, len(gItems))
	for i, v := range gItems {
		gStrs[i] = fmt.Sprintf("%v", v)
	}
	aStrs := make([]string, len(aItems))
	for i, v := range aItems {
		aStrs[i] = fmt.Sprintf("%v", v)
	}
	return compareStringSlice(path, gStrs, aStrs)
}

func stripVersion(arr []any) []any {
	if len(arr) > 0 {
		if _, ok := arr[0].(int32); ok {
			return arr[1:]
		}
	}
	return arr
}

func compareChildList(path string, golden, actual []element.Element, idMap IDMap, opts Options) []FieldDiff {
	switch {
	case allHaveName(golden) || allHaveName(actual):
		return compareByName(path, golden, actual, idMap, opts)
	default:
		return compareByPosition(path, golden, actual, idMap, opts)
	}
}

func elementName(elem element.Element) string {
	type namer interface{ NameValue() string }
	if n, ok := elem.(namer); ok {
		return n.NameValue()
	}
	return ""
}

func allHaveName(elems []element.Element) bool {
	if len(elems) == 0 {
		return false
	}
	for _, e := range elems {
		if e == nil || elementName(e) == "" {
			return false
		}
	}
	return true
}

func compareByName(path string, golden, actual []element.Element, idMap IDMap, opts Options) []FieldDiff {
	gByName := make(map[string]element.Element, len(golden))
	for _, e := range golden {
		if n := elementName(e); n != "" {
			gByName[n] = e
		}
	}
	aByName := make(map[string]element.Element, len(actual))
	for _, e := range actual {
		if n := elementName(e); n != "" {
			aByName[n] = e
		}
	}

	names := make(map[string]bool)
	for n := range gByName {
		names[n] = true
	}
	for n := range aByName {
		names[n] = true
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	var diffs []FieldDiff
	for _, name := range sorted {
		ge, gok := gByName[name]
		ae, aok := aByName[name]
		elemPath := fmt.Sprintf("%s[%s]", path, name)
		switch {
		case gok && !aok:
			diffs = append(diffs, FieldDiff{Path: elemPath, Golden: name, Kind: DiffRemoved})
		case !gok && aok:
			diffs = append(diffs, FieldDiff{Path: elemPath, Actual: name, Kind: DiffAdded})
		default:
			diffs = append(diffs, compareElements(elemPath, ge, ae, idMap, opts)...)
		}
	}
	return diffs
}

func compareByPosition(path string, golden, actual []element.Element, idMap IDMap, opts Options) []FieldDiff {
	if len(golden) != len(actual) {
		return []FieldDiff{{
			Path: path + ".length",
			Golden: fmt.Sprintf("%d", len(golden)),
			Actual: fmt.Sprintf("%d", len(actual)),
			Kind:  DiffChanged,
		}}
	}
	return nil
}

func describe(elem element.Element) string {
	if elem == nil {
		return ""
	}
	n := elementName(elem)
	if n != "" {
		return n
	}
	return elem.TypeName()
}
