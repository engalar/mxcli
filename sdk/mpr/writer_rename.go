// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// RenameHit describes a document that contains references to a renamed element.
type RenameHit struct {
	UnitID   string // Document UUID
	UnitType string // e.g., "Microflows$Microflow"
	Name     string // Document name (if found)
	Count    int    // Number of string replacements in this document
}

// RenameReferences scans all documents in the project and replaces qualified name
// strings matching oldName with newName. Returns the list of affected documents.
//
// Matching rules:
//   - Exact match: "Module.OldName" → "Module.NewName"
//   - Prefix match: "Module.OldName.Attr" → "Module.NewName.Attr"
//
// If dryRun is true, no modifications are written — only the hit list is returned.
func (w *Writer) RenameReferences(oldName, newName string, dryRun bool) ([]RenameHit, error) {
	// List all units (empty type prefix = all)
	units, err := w.reader.listUnitsByType("")
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %w", err)
	}

	var hits []RenameHit

	for _, unit := range units {
		contents, err := w.reader.resolveContents(unit.ID, unit.Contents)
		if err != nil {
			continue
		}
		if len(contents) == 0 {
			continue
		}

		var raw bson.D
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}

		count := 0
		updated := replaceStringsInDoc(raw, oldName, newName, &count)

		if count > 0 {
			// Extract document name for reporting
			docName := ""
			for _, elem := range updated {
				if elem.Key == "Name" {
					if s, ok := elem.Value.(string); ok {
						docName = s
					}
				}
			}

			hits = append(hits, RenameHit{
				UnitID:   unit.ID,
				UnitType: unit.Type,
				Name:     docName,
				Count:    count,
			})

			if !dryRun {
				newContents, err := bson.Marshal(updated)
				if err != nil {
					return hits, fmt.Errorf("failed to marshal updated document %s: %w", unit.ID, err)
				}
				if err := w.updateUnit(unit.ID, newContents); err != nil {
					return hits, fmt.Errorf("failed to write updated document %s: %w", unit.ID, err)
				}
			}
		}
	}

	return hits, nil
}

// ScanRenameReferences scans every unit and returns the patches + hit list
// produced by replacing oldName with newName (exact or prefix). It performs
// no writes — callers persist patches via the modelsdk write transaction.
func (w *Writer) ScanRenameReferences(oldName, newName string) ([]UnitPatch, []RenameHit, error) {
	units, err := w.reader.listUnitsByType("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list units: %w", err)
	}

	var (
		patches []UnitPatch
		hits    []RenameHit
	)

	for _, unit := range units {
		contents, err := w.reader.resolveContents(unit.ID, unit.Contents)
		if err != nil {
			continue
		}
		if len(contents) == 0 {
			continue
		}

		var raw bson.D
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}

		count := 0
		updated := replaceStringsInDoc(raw, oldName, newName, &count)
		if count == 0 {
			continue
		}

		docName := ""
		for _, elem := range updated {
			if elem.Key == "Name" {
				if s, ok := elem.Value.(string); ok {
					docName = s
				}
			}
		}

		hits = append(hits, RenameHit{
			UnitID:   unit.ID,
			UnitType: unit.Type,
			Name:     docName,
			Count:    count,
		})

		newContents, err := bson.Marshal(updated)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal updated document %s: %w", unit.ID, err)
		}
		patches = append(patches, UnitPatch{ID: unit.ID, Contents: newContents})
	}

	return patches, hits, nil
}

// FindRenameTarget locates the document Name==oldName within moduleName and
// returns its unit ID together with patched BSON contents that have the Name
// field rewritten to newName. It performs no writes — callers are expected to
// persist the returned bytes (e.g. via the modelsdk write path).
//
// Returns an error if the module or document cannot be found.
func (w *Writer) FindRenameTarget(moduleName, oldName, newName string) (string, []byte, error) {
	modules, err := w.reader.ListModules()
	if err != nil {
		return "", nil, fmt.Errorf("failed to list modules: %w", err)
	}

	var moduleID string
	for _, m := range modules {
		if m.Name == moduleName {
			moduleID = string(m.ID)
			break
		}
	}
	if moduleID == "" {
		return "", nil, fmt.Errorf("module not found: %s", moduleName)
	}

	hierarchy := buildContainerSet(w.reader, moduleID)

	units, err := w.reader.listUnitsByType("")
	if err != nil {
		return "", nil, fmt.Errorf("failed to list units: %w", err)
	}

	for _, unit := range units {
		if !hierarchy[unit.ContainerID] {
			continue
		}

		contents, err := w.reader.resolveContents(unit.ID, unit.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}

		var raw bson.D
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}

		for i, elem := range raw {
			if elem.Key == "Name" {
				if s, ok := elem.Value.(string); ok && s == oldName {
					raw[i].Value = newName
					newContents, err := bson.Marshal(raw)
					if err != nil {
						return "", nil, fmt.Errorf("failed to marshal: %w", err)
					}
					return unit.ID, newContents, nil
				}
			}
		}
	}

	return "", nil, fmt.Errorf("document '%s.%s' not found", moduleName, oldName)
}

// RenameDocumentByName finds a document by module and name, then updates its Name field.
// This works for any document type (microflow, nanoflow, page, constant, enumeration, etc.)
// by doing a raw BSON scan of all units in the module.
func (w *Writer) RenameDocumentByName(moduleName, oldName, newName string) error {
	// Find all modules to get the module ID
	modules, err := w.reader.ListModules()
	if err != nil {
		return fmt.Errorf("failed to list modules: %w", err)
	}

	var moduleID string
	for _, m := range modules {
		if m.Name == moduleName {
			moduleID = string(m.ID)
			break
		}
	}
	if moduleID == "" {
		return fmt.Errorf("module not found: %s", moduleName)
	}

	// Build container hierarchy to find documents in this module (including folders)
	hierarchy := buildContainerSet(w.reader, moduleID)

	// Scan all units looking for the document with matching Name
	units, err := w.reader.listUnitsByType("")
	if err != nil {
		return fmt.Errorf("failed to list units: %w", err)
	}

	for _, unit := range units {
		// Check if this unit belongs to the target module (direct or via folder)
		if !hierarchy[unit.ContainerID] {
			continue
		}

		contents, err := w.reader.resolveContents(unit.ID, unit.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}

		var raw bson.D
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}

		// Check if this document has Name == oldName
		for i, elem := range raw {
			if elem.Key == "Name" {
				if s, ok := elem.Value.(string); ok && s == oldName {
					raw[i].Value = newName
					newContents, err := bson.Marshal(raw)
					if err != nil {
						return fmt.Errorf("failed to marshal: %w", err)
					}
					return w.updateUnit(unit.ID, newContents)
				}
			}
		}
	}

	return fmt.Errorf("document '%s.%s' not found", moduleName, oldName)
}

// buildContainerSet returns a set of container IDs that belong to a module
// (the module ID itself plus all folder IDs nested under it).
func buildContainerSet(r *Reader, moduleID string) map[string]bool {
	set := map[string]bool{moduleID: true}

	folders, err := r.ListFolders()
	if err != nil {
		return set
	}

	// Iteratively expand: if a folder's container is in the set, add the folder
	changed := true
	for changed {
		changed = false
		for _, f := range folders {
			if set[string(f.ContainerID)] && !set[string(f.ID)] {
				set[string(f.ID)] = true
				changed = true
			}
		}
	}

	return set
}

// replaceStringsInDoc recursively walks a bson.D document and replaces string
// values that match oldName exactly or start with oldName + ".".
func replaceStringsInDoc(doc bson.D, oldName, newName string, count *int) bson.D {
	result := make(bson.D, len(doc))
	for i, elem := range doc {
		result[i] = bson.E{
			Key:   elem.Key,
			Value: replaceStringsInValue(elem.Value, oldName, newName, count),
		}
	}
	return result
}

// replaceStringsInValue replaces qualified name strings in any BSON value type.
func replaceStringsInValue(val any, oldName, newName string, count *int) any {
	switch v := val.(type) {
	case string:
		if v == oldName {
			*count++
			return newName
		}
		if strings.HasPrefix(v, oldName+".") {
			*count++
			return newName + v[len(oldName):]
		}
		return v

	case bson.D:
		return replaceStringsInDoc(v, oldName, newName, count)

	case bson.A:
		result := make(bson.A, len(v))
		for i, item := range v {
			result[i] = replaceStringsInValue(item, oldName, newName, count)
		}
		return result

	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = replaceStringsInValue(item, oldName, newName, count)
		}
		return result

	default:
		return v
	}
}
