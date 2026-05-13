// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type sqlResolver struct {
	r *mmpr.Reader
}

func NewQualifiedNameResolver(w *mmpr.Writer) repos.QualifiedNameResolver {
	return &sqlResolver{r: w.ConcreteReader()}
}

// ModuleNameByID returns the module's display name for the given module
// unit ID. Backed by mmpr.Reader.GetModule which scans
// "Projects$ModuleImpl" units.
func (s *sqlResolver) ModuleNameByID(id model.ID) (string, error) {
	m, err := s.r.GetModule(string(id))
	if err != nil {
		return "", err
	}
	return m.Name, nil
}

// ResolveQualifiedName parses a "Module.ElementName" qualified name and
// returns the unit ID + storage type prefix (e.g. "Microflows$Microflow").
//
// TODO Stage 2 finish: port the per-kind walk from sdk/mpr.Reader.
// GetRawUnitByName (sdk/mpr/reader_units.go:271) onto modelsdk/mpr's
// ListUnitsByType. Stage 2 only ships Microflows + Pages repos so the
// caller surface is currently empty; finish this when MicroflowRepo
// or the executor begins consuming Names.ResolveQualifiedName.
func (s *sqlResolver) ResolveQualifiedName(qn string) (model.ID, string, error) {
	return "", "", fmt.Errorf("ResolveQualifiedName: implementer port pending — qn=%s", qn)
}
