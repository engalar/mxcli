// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// updateNavigationProfileViaModelsdk patches a navigation profile through the
// modelsdk write path. It reads the current raw BSON via the modelsdk reader,
// applies the patch via sdk/mpr.Writer.PatchNavigationProfile (BSON-only), and
// writes the result back through the WriteTransaction. This avoids the
// _Transaction 1544 bug triggered by the legacy sdk/mpr.Writer.updateUnit path.
func (b *MprBackend) updateNavigationProfileViaModelsdk(navDocID model.ID, profileName string, spec mpr.NavigationProfileSpec) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(navDocID))
	if err != nil {
		return fmt.Errorf("read unit: %w", err)
	}
	patched, err := b.writer.PatchNavigationProfile(rawBytes, profileName, spec)
	if err != nil {
		return err
	}
	return b.writeUnitContents(navDocID, patched)
}
