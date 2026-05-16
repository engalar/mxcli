// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// updateNavigationProfileViaModelsdk patches a navigation profile through the
// modelsdk write path. Reads current raw BSON via the modelsdk reader, applies
// the BSON-only patch, then writes back through WriteTransaction (avoids 1544).
func (b *MprBackend) updateNavigationProfileViaModelsdk(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(navDocID))
	if err != nil {
		return fmt.Errorf("read unit: %w", err)
	}
	patched, err := sdkPatchNavigationProfile(rawBytes, profileName, spec)
	if err != nil {
		return err
	}
	return b.writeUnitContents(navDocID, patched)
}
