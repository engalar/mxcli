// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"github.com/mendixlabs/mxcli/internal/marketplace/domain"
	"github.com/mendixlabs/mxcli/model"
)

type ModuleLister interface {
	ListModules() ([]*model.Module, error)
}

type ProjectReader struct {
	lister ModuleLister
}

func NewProjectReader(lister ModuleLister) *ProjectReader {
	return &ProjectReader{lister: lister}
}

func (pr *ProjectReader) ListInstalledModules(projectPath string) ([]domain.InstalledModule, error) {
	modules, err := pr.lister.ListModules()
	if err != nil {
		return nil, err
	}
	var out []domain.InstalledModule
	for _, m := range modules {
		if m.FromAppStore {
			out = append(out, domain.InstalledModule{
				Name:            m.Name,
				ModuleID:        string(m.ID),
				AppStoreGuid:    m.AppStoreGuid,
				AppStoreVersion: m.AppStoreVersion,
			})
		}
	}
	return out, nil
}
