// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

// moduleBackend implements backend.ModuleLister (read-only) by wrapping
// the mpr Reader. Write methods remain on MprBackend for now.
type moduleBackend struct {
	reader *modelsdkmpr.Reader
}

func newModuleBackend(reader *modelsdkmpr.Reader) *moduleBackend {
	return &moduleBackend{reader: reader}
}

func (b *moduleBackend) ListModules() ([]*model.Module, error) {
	units, err := mprread.ListModules(b.reader)
	if err != nil {
		return nil, err
	}
	mods := moduleUnitsToModel(units)
	mods = append(mods, buildSystemModuleForBackend())
	return mods, nil
}

func (b *moduleBackend) GetModule(id model.ID) (*model.Module, error) {
	mods, err := b.ListModules()
	if err != nil {
		return nil, err
	}
	for _, m := range mods {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, fmt.Errorf("module not found: %s", id)
}

func (b *moduleBackend) GetModuleByName(name string) (*model.Module, error) {
	mods, err := b.ListModules()
	if err != nil {
		return nil, err
	}
	for _, m := range mods {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, fmt.Errorf("module not found: %s", name)
}
