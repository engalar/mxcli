// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/model"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

// securityBackend implements the read-only gen-typed Security surface by
// wrapping the modelsdk-native SecurityRepository. Write methods (Set*,
// AddUserRole, etc.) remain on MprBackend.
type securityBackend struct {
	writer *mmpr.Writer
}

func newSecurityBackend(writer *mmpr.Writer) *securityBackend {
	return &securityBackend{writer: writer}
}

func (b *securityBackend) GetProjectSecurityGen() (*genSec.ProjectSecurity, error) {
	return mprrepos.NewSecurityRepository(b.writer).Get()
}

func (b *securityBackend) GetModuleSecurityGen(moduleID model.ID) (*genSec.ModuleSecurity, error) {
	return mprrepos.NewSecurityRepository(b.writer).GetModuleSecurity(moduleID)
}

func (b *securityBackend) ListModuleSecurityGen() ([]*genSec.ModuleSecurity, error) {
	units, err := mprread.ListModules(b.writer.ConcreteReader())
	if err != nil {
		return nil, err
	}
	mods := moduleUnitsToModel(units)
	mods = append(mods, buildSystemModuleForBackend())
	repo := mprrepos.NewSecurityRepository(b.writer)
	out := make([]*genSec.ModuleSecurity, 0, len(mods))
	for _, m := range mods {
		ms, err := repo.GetModuleSecurity(m.ID)
		if err != nil || ms == nil {
			continue
		}
		out = append(out, ms)
	}
	return out, nil
}
