// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"sort"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func Compare(aPath, bPath string, opts Options) ([]UnitDiff, error) {
	if aPath == bPath {
		return nil, nil
	}

	aReader, err := mmpr.OpenWithOptions(aPath, mmpr.OpenOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: open A (%s): %w", aPath, err)
	}
	defer aReader.Close()

	bReader, err := mmpr.OpenWithOptions(bPath, mmpr.OpenOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: open B (%s): %w", bPath, err)
	}
	defer bReader.Close()

	idMap, err := BuildIDMapFromReader(bReader)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: IDMap B: %w", err)
	}
	aIDMap, err := BuildIDMapFromReader(aReader)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: IDMap A: %w", err)
	}
	MergeInto(idMap, aIDMap)

	aUnits, err := ReadAllUnits(aPath)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: read A: %w", err)
	}
	bUnits, err := ReadAllUnits(bPath)
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: read B: %w", err)
	}

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

	dec := codec.NewDecoder(codec.DefaultRegistry)
	var result []UnitDiff

	for _, name := range names {
		au, aok := aIndex[name]
		bu, bok := bIndex[name]
		switch {
		case aok && !bok:
			result = append(result, UnitDiff{QualifiedName: name, UnitType: au.UnitType, Kind: DiffRemoved})
		case !aok && bok:
			actual, err := dec.DecodeBytes(bu.Raw)
			if err != nil {
				continue
			}
			result = append(result, UnitDiff{
				QualifiedName: name,
				UnitType:      bu.UnitType,
				Kind:          DiffAdded,
				ActualDoc:     actual,
			})
		default:
			if au.ContentHash == bu.ContentHash {
				continue
			}
			actual, aErr := dec.DecodeBytes(bu.Raw)
			golden, gErr := dec.DecodeBytes(au.Raw)
			if aErr != nil || gErr != nil {
				continue
			}
			fields := compareElements("", golden, actual, idMap, opts)
			if len(fields) > 0 {
				result = append(result, UnitDiff{
					QualifiedName: name,
					UnitType:      au.UnitType,
					Kind:          DiffChanged,
					Fields:        fields,
					ActualDoc:     actual,
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
