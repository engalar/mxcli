// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// updateQualifiedNameInAllUnitsViaModelsdk computes patches via the sdk/mpr
// Reader and persists each one through the modelsdk WriteTransaction, avoiding
// the SQLITE_READONLY_DBMOVED 1544 path triggered by Writer.updateUnit.
func (b *MprBackend) updateQualifiedNameInAllUnitsViaModelsdk(oldName, newName string) (int, error) {
	if b.msdkWriter == nil {
		return 0, fmt.Errorf("modelsdk writer not initialized")
	}
	patches, err := b.reader.ScanQualifiedNameUpdates(oldName, newName)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, p := range patches {
		if err := b.writeUnitContents(model.ID(p.ID), p.Contents); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

// renameReferencesViaModelsdk computes patches + hits via the sdk/mpr Reader
// and, when dryRun is false, persists each patch through the modelsdk write path.
func (b *MprBackend) renameReferencesViaModelsdk(oldName, newName string, dryRun bool) ([]types.RenameHit, error) {
	if b.msdkWriter == nil {
		return nil, fmt.Errorf("modelsdk writer not initialized")
	}
	patches, hits, err := b.reader.ScanRenameReferences(oldName, newName)
	if err != nil {
		return nil, err
	}
	if !dryRun {
		for _, p := range patches {
			if err := b.writeUnitContents(model.ID(p.ID), p.Contents); err != nil {
				return hits, err
			}
		}
	}
	return hits, nil
}
