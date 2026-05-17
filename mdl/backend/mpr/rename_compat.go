// SPDX-License-Identifier: Apache-2.0

// rename_compat.go — rename / OQL-rewrite helpers ported from sdk/mpr.
//
// FindRenameTarget locates a document by name within a module and returns its
// unit ID plus rewritten BSON for the new name. scanOqlQueryUpdates rewrites
// OQL strings inside ViewEntitySourceDocument units.

package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
)

const viewEntitySourceDocumentBsonType = "DomainModels$ViewEntitySourceDocument"

// findRenameTarget walks the module's container hierarchy (module + nested
// folders), locates the first document whose Name field equals oldName, and
// returns its unit ID together with rewritten BSON in which Name is newName.
// No writes are performed — callers persist the bytes via the modelsdk writer.
func (b *MprBackend) findRenameTarget(moduleName, oldName, newName string) (string, []byte, error) {
	modules, err := b.ListModules()
	if err != nil {
		return "", nil, fmt.Errorf("list modules: %w", err)
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

	hierarchy := b.buildContainerSetCompat(moduleID)

	refs, err := b.msdkReader.ListUnitsByType("")
	if err != nil {
		return "", nil, fmt.Errorf("list units: %w", err)
	}
	for _, ref := range refs {
		if !hierarchy[ref.ContainerID] {
			continue
		}
		contents := ref.Contents
		if len(contents) == 0 {
			continue
		}
		var raw bson.D
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		for i, elem := range raw {
			if elem.Key != "Name" {
				continue
			}
			s, ok := elem.Value.(string)
			if !ok || s != oldName {
				continue
			}
			raw[i].Value = newName
			newContents, err := bson.Marshal(raw)
			if err != nil {
				return "", nil, fmt.Errorf("marshal renamed unit: %w", err)
			}
			return ref.ID, newContents, nil
		}
	}
	return "", nil, fmt.Errorf("document '%s.%s' not found", moduleName, oldName)
}

// buildContainerSetCompat returns the set of container IDs that belong to the
// given module — the module ID itself plus every folder transitively nested
// under it. Mirrors sdk/mpr.buildContainerSet.
func (b *MprBackend) buildContainerSetCompat(moduleID string) map[string]bool {
	set := map[string]bool{moduleID: true}
	folders, err := b.ListFolders()
	if err != nil {
		return set
	}
	for changed := true; changed; {
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

// scanOqlQueryUpdatesCompat rewrites occurrences of oldQualifiedName inside the
// Oql field of every DomainModels$ViewEntitySourceDocument unit. The function
// returns patch records describing the units that changed; the caller persists
// them through the modelsdk writer.
func (b *MprBackend) scanOqlQueryUpdatesCompat(oldQualifiedName, newQualifiedName string) ([]types.UnitPatch, int, error) {
	refs, err := b.msdkReader.ListUnitsByType(viewEntitySourceDocumentBsonType)
	if err != nil {
		return nil, 0, err
	}
	var patches []types.UnitPatch
	for _, ref := range refs {
		var raw map[string]any
		if err := bson.Unmarshal(ref.Contents, &raw); err != nil {
			continue
		}
		oql, _ := raw["Oql"].(string)
		if oql == "" || !strings.Contains(oql, oldQualifiedName) {
			continue
		}
		raw["Oql"] = strings.ReplaceAll(oql, oldQualifiedName, newQualifiedName)
		contents, err := bson.Marshal(raw)
		if err != nil {
			continue
		}
		patches = append(patches, types.UnitPatch{ID: ref.ID, Contents: contents})
	}
	return patches, len(patches), nil
}
