// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/model"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/meta"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// domainModelBackend implements the read-only gen-typed DomainModel surface
// by wrapping the modelsdk-native DomainModelRepository. Write methods
// (UpdateDomainModelGen, delete helpers) remain on MprBackend.
type domainModelBackend struct {
	writer *mmpr.Writer
}

func newDomainModelBackend(writer *mmpr.Writer) *domainModelBackend {
	return &domainModelBackend{writer: writer}
}

func (b *domainModelBackend) ListDomainModelsGen() ([]*genDm.DomainModel, error) {
	dms, err := mprrepos.NewDomainModelRepository(b.writer).List("")
	if err != nil {
		return nil, err
	}
	return append(dms, builtinSystemDomainModel()), nil
}

func (b *domainModelBackend) GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error) {
	if string(moduleID) == meta.SystemModuleID {
		return builtinSystemDomainModel(), nil
	}
	dms, err := mprrepos.NewDomainModelRepository(b.writer).List(moduleID)
	if err != nil {
		return nil, err
	}
	if len(dms) == 0 {
		return nil, nil
	}
	return dms[0], nil
}

func (b *domainModelBackend) GetDomainModelByIDGen(id model.ID) (*genDm.DomainModel, error) {
	return mprrepos.NewDomainModelRepository(b.writer).Get(id)
}
