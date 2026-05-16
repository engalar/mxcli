// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UpdateQualifiedNameInAllUnits replaces all occurrences of oldName with newName
// in string values across all BSON documents in the project. Handles both exact
// matches and prefix matches (e.g., "Module.Name.Param" when renaming "Module.Name").
// Returns the number of documents that were updated.
func (w *Writer) UpdateQualifiedNameInAllUnits(oldName, newName string) (int, error) {
	units, err := w.reader.listUnitsByType("")
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		if replaceStringsInMap(raw, oldName, newName) {
			contents, err := bson.Marshal(raw)
			if err != nil {
				continue
			}
			if err := w.updateUnit(u.ID, contents); err != nil {
				return updated, err
			}
			updated++
		}
	}

	return updated, nil
}

// ScanQualifiedNameUpdates scans every unit in the project and returns the set
// of patches needed to replace oldName with newName (exact or "oldName." prefix
// match). It performs no writes — callers persist the returned patches via the
// modelsdk write transaction.
func (r *Reader) ScanQualifiedNameUpdates(oldName, newName string) ([]UnitPatch, error) {
	units, err := r.listUnitsByType("")
	if err != nil {
		return nil, err
	}

	var patches []UnitPatch
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		if replaceStringsInMap(raw, oldName, newName) {
			contents, err := bson.Marshal(raw)
			if err != nil {
				continue
			}
			patches = append(patches, UnitPatch{ID: u.ID, Contents: contents})
		}
	}

	return patches, nil
}

// ScanQualifiedNameUpdates is the Writer method form; delegates to the Reader.
func (w *Writer) ScanQualifiedNameUpdates(oldName, newName string) ([]UnitPatch, error) {
	return w.reader.ScanQualifiedNameUpdates(oldName, newName)
}

// replaceStringsInMap recursively walks a map and replaces string values that
// match oldName exactly or have oldName as a prefix (followed by ".").
// Returns true if any replacement was made.
func replaceStringsInMap(m map[string]any, oldName, newName string) bool {
	changed := false
	for k, v := range m {
		if replaced, ok := replaceInValue(v, oldName, newName); ok {
			m[k] = replaced
			changed = true
		}
	}
	return changed
}

// replaceInValue recursively processes a value and returns the replacement and
// whether any change was made.
func replaceInValue(v any, oldName, newName string) (any, bool) {
	switch val := v.(type) {
	case string:
		if newStr, ok := replaceQualifiedName(val, oldName, newName); ok {
			return newStr, true
		}
	case map[string]any:
		if replaceStringsInMap(val, oldName, newName) {
			return val, true
		}
	case primitive.M:
		m := map[string]any(val)
		if replaceStringsInMap(m, oldName, newName) {
			return val, true
		}
	case primitive.A:
		changed := false
		for i, elem := range val {
			if replaced, ok := replaceInValue(elem, oldName, newName); ok {
				val[i] = replaced
				changed = true
			}
		}
		if changed {
			return val, true
		}
	case []any:
		changed := false
		for i, elem := range val {
			if replaced, ok := replaceInValue(elem, oldName, newName); ok {
				val[i] = replaced
				changed = true
			}
		}
		if changed {
			return val, true
		}
	case primitive.D:
		changed := false
		for i, elem := range val {
			if replaced, ok := replaceInValue(elem.Value, oldName, newName); ok {
				val[i].Value = replaced
				changed = true
			}
		}
		if changed {
			return val, true
		}
	}
	return v, false
}

// replaceQualifiedName checks if s matches oldName exactly or as a prefix
// (e.g., "OldModule.Microflow.Param") and returns the replacement.
func replaceQualifiedName(s, oldName, newName string) (string, bool) {
	if s == oldName {
		return newName, true
	}
	// Prefix match: "OldModule.Microflow.Param" → "NewModule.Microflow.Param"
	if strings.HasPrefix(s, oldName+".") {
		return newName + s[len(oldName):], true
	}
	return "", false
}
