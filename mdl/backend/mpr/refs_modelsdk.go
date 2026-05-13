// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// updateQualifiedNameInAllUnitsViaModelsdk computes patches via sdk/mpr.Writer
// and persists each one through the modelsdk WriteTransaction, avoiding the
// SQLITE_READONLY_DBMOVED 1544 path triggered by Writer.updateUnit.
func (b *MprBackend) updateQualifiedNameInAllUnitsViaModelsdk(oldName, newName string) (int, error) {
	if b.msdkWriter == nil {
		return 0, fmt.Errorf("modelsdk writer not initialized")
	}
	patches, err := b.writer.ScanQualifiedNameUpdates(oldName, newName)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, p := range patches {
		if err := b.msdkWriteRaw(model.ID(p.ID), p.Contents); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

// renameReferencesViaModelsdk computes patches + hits via sdk/mpr.Writer and,
// when dryRun is false, persists each patch through the modelsdk write path.
func (b *MprBackend) renameReferencesViaModelsdk(oldName, newName string, dryRun bool) ([]sdkmpr.RenameHit, error) {
	if b.msdkWriter == nil {
		return nil, fmt.Errorf("modelsdk writer not initialized")
	}
	patches, hits, err := b.writer.ScanRenameReferences(oldName, newName)
	if err != nil {
		return nil, err
	}
	if !dryRun {
		for _, p := range patches {
			if err := b.msdkWriteRaw(model.ID(p.ID), p.Contents); err != nil {
				return hits, err
			}
		}
	}
	return hits, nil
}
