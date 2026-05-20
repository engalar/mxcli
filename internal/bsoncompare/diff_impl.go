// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"sort"
)

func Compare(aPath, bPath string, opts Options) ([]UnitDiff, error) {
	aUnits, err := ReadAllUnits(aPath)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: read A (%s): %w", aPath, err)
	}
	bUnits, err := ReadAllUnits(bPath)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: read B (%s): %w", bPath, err)
	}

	idMap := BuildIDMap(bUnits)
	MergeInto(idMap, BuildIDMap(aUnits))

	aIndex := indexUnits(aUnits)
	bIndex := indexUnits(bUnits)

	allNames := make(map[string]bool)
	for k := range aIndex {
		allNames[k] = true
	}
	for k := range bIndex {
		allNames[k] = true
	}
	names := make([]string, 0, len(allNames))
	for k := range allNames {
		names = append(names, k)
	}
	sort.Strings(names)

	var result []UnitDiff
	for _, name := range names {
		au, aok := aIndex[name]
		bu, bok := bIndex[name]
		switch {
		case aok && !bok:
			result = append(result, UnitDiff{QualifiedName: name, UnitType: au.UnitType, Kind: DiffRemoved})
		case !aok && bok:
			result = append(result, UnitDiff{QualifiedName: name, UnitType: bu.UnitType, Kind: DiffAdded})
		default:
			aN := Normalize(au.Doc, idMap, opts)
			bN := Normalize(bu.Doc, idMap, opts)
			var fields []FieldDiff
			diffDoc("", aN, bN, &fields)
			if len(fields) > 0 {
				result = append(result, UnitDiff{
					QualifiedName: name,
					UnitType:      au.UnitType,
					Kind:          DiffChanged,
					Fields:        fields,
				})
			}
		}
	}
	return result, nil
}

func indexUnits(units []UnitDoc) map[string]UnitDoc {
	m := make(map[string]UnitDoc, len(units))
	for _, u := range units {
		m[u.QualifiedName] = u
	}
	return m
}

func diffDoc(path string, golden, actual map[string]any, out *[]FieldDiff) {
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
