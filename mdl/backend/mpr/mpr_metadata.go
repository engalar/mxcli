// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type metadataBackend struct {
	reader *modelsdkmpr.Reader
}

func newMetadataBackend(reader *modelsdkmpr.Reader) *metadataBackend {
	return &metadataBackend{reader: reader}
}

func (b *metadataBackend) ListAllUnitIDs() ([]string, error) {
	return b.reader.ListAllUnitIDs()
}

func (b *metadataBackend) ListUnits() ([]*types.UnitInfo, error) {
	units, err := b.reader.ListUnits()
	if err != nil {
		return nil, err
	}
	return msdkUnitInfoSliceToTypes(units), nil
}

func (b *metadataBackend) ListUnitHashes() (map[string]string, error) {
	return b.reader.ListUnitHashes()
}

func (b *metadataBackend) GetUnitTypes() (map[string]int, error) {
	return b.reader.GetUnitTypes()
}

func (b *metadataBackend) GetProjectRootID() (string, error) {
	return b.reader.GetProjectRootID()
}

func (b *metadataBackend) ContentsDir() string {
	return b.reader.ContentsDir()
}

func (b *metadataBackend) InvalidateCache() {
	b.reader.InvalidateCache()
}
