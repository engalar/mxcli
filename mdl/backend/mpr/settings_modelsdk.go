// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
)

// updateProjectSettingsViaModelsdk reuses the canonical sdk/mpr serializer
// to produce BSON bytes, then writes them via msdkWriteRaw to avoid the
// SQLITE_READONLY_DBMOVED 1544 path triggered by Writer.updateUnit.
func (b *MprBackend) updateProjectSettingsViaModelsdk(ps *model.ProjectSettings) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeProjectSettings(ps)
	if err != nil {
		return fmt.Errorf("serialize project settings: %w", err)
	}
	return b.msdkWriteRaw(ps.ID, contents)
}
