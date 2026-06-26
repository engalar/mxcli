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

	// Fast path: same file → identical by definition
	if aPath == bPath {
		return nil, nil
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
			bu.Decode()
			result = append(result, UnitDiff{
				QualifiedName: name,
				UnitType:      bu.UnitType,
				Kind:          DiffAdded,
				ActualDoc:     bu.Doc,
			})
		default:
			if au.ContentHash == bu.ContentHash {
				continue
			}
			au.Decode()
			bu.Decode()
			aN := Normalize(au.Doc, idMap, opts)
			bN := Normalize(bu.Doc, idMap, opts)
			var fields []FieldDiff
			diffMaps("", aN, bN, false, &fields)
			if len(fields) > 0 {
				result = append(result, UnitDiff{
					QualifiedName: name,
					UnitType:      au.UnitType,
					Kind:          DiffChanged,
					Fields:        fields,
					ActualDoc:     bu.Doc,
				})
			}
		}
	}
	return result, nil
}

func indexUnits(units []UnitDoc) map[string]*UnitDoc {
	m := make(map[string]*UnitDoc, len(units))
	for i := range units {
		m[units[i].QualifiedName] = &units[i]
	}
	return m
}
