// SPDX-License-Identifier: Apache-2.0

package microflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func ExecCreateNanoflowGenFn(ctx context.Context, s *ast.CreateNanoflowStmt, d *MicroflowDeps) error {
	if d.ConnectionManager == nil || !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.NanoflowsRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}
	if strings.TrimSpace(s.Name.Name) == "" {
		return mdlerrors.NewValidation("nanoflow name must not be empty")
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return err
	}

	containerID := module.ID
	qualifiedName := s.Name.Module + "." + s.Name.Name

	nf := genMf.NewNanoflow()
	nf.SetName(s.Name.Name)
	nf.SetDocumentation(s.Documentation)
	nf.SetExcluded(s.Excluded)
	nf.SetAllowedModuleRolesQualifiedNames(d.DefaultDocumentAccessRoles(module))

	if d.BuildMicroflowFlowGraph != nil {
		oc, flows, annotationFlows, errs, err2 := d.BuildMicroflowFlowGraph(s.Body, s.ReturnType, s.Parameters, true)
		if err2 != nil {
			return err2
		}
		if len(errs) > 0 {
			return mdlerrors.NewValidationf("nanoflow '%s' has validation errors:\n  - %s", qualifiedName, strings.Join(errs, "\n  - "))
		}
		nf.SetObjectCollection(oc)
		for _, flow := range flows {
			nf.AddFlows(flow)
		}
		for _, af := range annotationFlows {
			nf.AddFlows(af)
		}
	}

	if s.ReturnType != nil {
		if t := paramASTToShortType(s.ReturnType.Type); t != "" {
			nf.SetReturnType(t)
		}
		if d.ConvertASTToGenDataType != nil {
			if dt := d.ConvertASTToGenDataType(s.ReturnType.Type); dt != nil {
				nf.SetMicroflowReturnType(dt)
			}
		}
		if s.ReturnType.Variable != "" {
			nf.SetReturnVariableName(s.ReturnType.Variable)
		}
	}

	if err := d.NanoflowsRepo.Create(string(containerID), "Documents", nf); err != nil {
		return mdlerrors.NewBackend("create nanoflow", err)
	}
	fmt.Fprintf(d.Output, "Created nanoflow: %s\n", qualifiedName)

	returnEntityName := ""
	if s.ReturnType != nil {
		if ref := paramEntityRef(s.ReturnType.Type); ref != nil && ref.Module != "" {
			returnEntityName = ref.Module + "." + ref.Name
		}
	}
	if d.TrackCreatedNanoflow != nil {
		d.TrackCreatedNanoflow(s.Name.Module, s.Name.Name, model.ID(nf.ID()), containerID, returnEntityName)
	}
	d.InvalidateCache()
	return nil
}

func ExecDropNanoflowGenFn(ctx context.Context, s *ast.DropNanoflowStmt, d *MicroflowDeps) error {
	if d.ConnectionManager == nil || !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.NanoflowsRepo == nil {
		return mdlerrors.NewBackend("nanoflows repo unavailable", nil)
	}
	all, err := d.NanoflowsRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}
	for _, nf := range all {
		if nf == nil {
			continue
		}
		if nf.Name() != s.Name.Name {
			continue
		}
		if err := d.NanoflowsRepo.Delete(model.ID(nf.ID())); err != nil {
			return mdlerrors.NewBackend("delete nanoflow", err)
		}
		d.InvalidateCache()
		fmt.Fprintf(d.Output, "Dropped nanoflow: %s\n", s.Name.Module+"."+s.Name.Name)
		return nil
	}
	return mdlerrors.NewNotFound("nanoflow", s.Name.Module+"."+s.Name.Name)
}
