// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
)

// renameDocumentByNameViaModelsdk renames a document Name field using the
// modelsdk write transaction. The lookup + BSON patch is delegated to the
// existing sdk/mpr.Writer.FindRenameTarget helper; only the persistence step
// is rerouted through writeUnitContents to avoid the SQLITE_READONLY_DBMOVED
// 1544 path triggered by Writer.updateUnit.
func (b *MprBackend) renameDocumentByNameViaModelsdk(moduleName, oldName, newName string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	unitID, newContents, err := b.findRenameTarget(moduleName, oldName, newName)
	if err != nil {
		return err
	}
	return b.writeUnitContents(model.ID(unitID), newContents)
}
