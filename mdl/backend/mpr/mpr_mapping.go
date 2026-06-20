// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genExpMap "github.com/mendixlabs/mxcli/modelsdk/gen/exportmappings"
	genImpMap "github.com/mendixlabs/mxcli/modelsdk/gen/importmappings"
	genJson "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

type mappingBackend struct {
	reader *modelsdkmpr.Reader
}

func newMappingBackend(reader *modelsdkmpr.Reader) *mappingBackend {
	return &mappingBackend{reader: reader}
}

func (b *mappingBackend) ListImportMappings() ([]*model.ImportMapping, error) {
	units, err := mprread.ListUnitsWithContainer[*genImpMap.ImportMapping](b.reader)
	if err != nil {
		return nil, err
	}
	return importMappingUnitsToModel(units), nil
}

func (b *mappingBackend) GetImportMappingByQualifiedName(moduleName, name string) (*model.ImportMapping, error) {
	units, err := mprread.ListUnitsWithContainer[*genImpMap.ImportMapping](b.reader)
	if err != nil {
		return nil, err
	}
	for _, u := range units {
		if u.Element.Name() == name {
			return importMappingToModel(u), nil
		}
	}
	return nil, nil
}

func (b *mappingBackend) ListExportMappings() ([]*model.ExportMapping, error) {
	units, err := mprread.ListUnitsWithContainer[*genExpMap.ExportMapping](b.reader)
	if err != nil {
		return nil, err
	}
	return exportMappingUnitsToModel(units), nil
}

func (b *mappingBackend) GetExportMappingByQualifiedName(moduleName, name string) (*model.ExportMapping, error) {
	units, err := mprread.ListUnitsWithContainer[*genExpMap.ExportMapping](b.reader)
	if err != nil {
		return nil, err
	}
	for _, u := range units {
		if u.Element.Name() == name {
			return exportMappingToModel(u), nil
		}
	}
	return nil, nil
}

func (b *mappingBackend) ListJsonStructures() ([]*types.JsonStructure, error) {
	units, err := mprread.ListUnitsWithContainer[*genJson.JsonStructure](b.reader)
	if err != nil {
		return nil, err
	}
	return jsonStructureUnitsToTypes(units), nil
}
