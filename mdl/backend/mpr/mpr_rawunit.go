// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type rawUnitBackend struct {
	reader *modelsdkmpr.Reader
	writer modelsdkmpr.UnitWriter
}

func newRawUnitBackend(reader *modelsdkmpr.Reader, writer modelsdkmpr.UnitWriter) *rawUnitBackend {
	return &rawUnitBackend{reader: reader, writer: writer}
}

func (b *rawUnitBackend) GetRawUnit(id model.ID) (map[string]any, error) {
	return b.reader.GetRawUnit(id)
}

func (b *rawUnitBackend) GetRawUnitBytes(id model.ID) ([]byte, error) {
	return b.reader.GetRawUnitBytes(string(id))
}

func (b *rawUnitBackend) ListRawUnitsByType(typePrefix string) ([]*types.RawUnit, error) {
	return b.reader.ListRawUnitsByType(typePrefix)
}

func (b *rawUnitBackend) ListRawUnits(objectType string) ([]*types.RawUnitInfo, error) {
	return b.reader.ListRawUnits(objectType)
}

func (b *rawUnitBackend) GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error) {
	return b.reader.GetRawUnitByName(objectType, qualifiedName)
}

func (b *rawUnitBackend) GetRawMicroflowByName(qualifiedName string) ([]byte, error) {
	return b.reader.GetRawMicroflowByName(qualifiedName)
}

func (b *rawUnitBackend) UpdateRawUnit(unitID string, contents []byte) error {
	if b.writer == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.writer.UpdateRawUnit(unitID, contents)
}
