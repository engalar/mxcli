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

func paramEntityRef(t ast.DataType) *ast.QualifiedName {
	if t.Kind == ast.TypeEntity && t.EntityRef != nil {
		return t.EntityRef
	}
	if t.Kind == ast.TypeListOf && t.EntityRef != nil {
		return t.EntityRef
	}
	if t.Kind == ast.TypeEnumeration && t.EnumRef != nil {
		return t.EnumRef
	}
	return nil
}

func paramASTToShortType(t ast.DataType) string {
	switch t.Kind {
	case ast.TypeBoolean:
		return "Boolean"
	case ast.TypeInteger:
		return "Integer"
	case ast.TypeLong:
		return "Long"
	case ast.TypeDecimal:
		return "Decimal"
	case ast.TypeString:
		return "String"
	case ast.TypeDateTime:
		return "DateTime"
	case ast.TypeBinary:
		return "Binary"
	case ast.TypeVoid:
		return "Void"
	case ast.TypeEntity:
		if t.EntityRef != nil {
			return t.EntityRef.Module + "." + t.EntityRef.Name
		}
	case ast.TypeEnumeration:
		if t.EnumRef != nil {
			return t.EnumRef.Module + "." + t.EnumRef.Name
		}
	case ast.TypeListOf:
		if t.EntityRef != nil {
			return "List of " + t.EntityRef.Module + "." + t.EntityRef.Name
		}
	}
	return ""
}

func ExecCreateMicroflowGenFn(ctx context.Context, s *ast.CreateMicroflowStmt, d *MicroflowDeps) error {
	if d.ConnectionManager == nil || !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.MicroflowsRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}
	if strings.TrimSpace(s.Name.Name) == "" {
		return mdlerrors.NewValidation("microflow name must not be empty")
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return err
	}

	containerID := module.ID
	qualifiedName := s.Name.Module + "." + s.Name.Name

	mf := genMf.NewMicroflow()
	mf.SetName(s.Name.Name)
	mf.SetDocumentation(s.Documentation)
	mf.SetExcluded(s.Excluded)
	mf.SetAllowConcurrentExecution(true)
	mf.SetAllowedModuleRolesQualifiedNames(d.DefaultDocumentAccessRoles(module))

	if s.ReturnType != nil {
		if t := paramASTToShortType(s.ReturnType.Type); t != "" {
			mf.SetReturnType(t)
		}
		if d.ConvertASTToGenDataType != nil {
			if dt := d.ConvertASTToGenDataType(s.ReturnType.Type); dt != nil {
				mf.SetMicroflowReturnType(dt)
			}
		}
		if s.ReturnType.Variable != "" {
			mf.SetReturnVariableName(s.ReturnType.Variable)
		}
	} else {
		mf.SetReturnType("Nothing")
	}

	if d.BuildMicroflowFlowGraph != nil {
		oc, flows, annotationFlows, errs, err2 := d.BuildMicroflowFlowGraph(s.Body, s.ReturnType, s.Parameters, false)
		if err2 != nil {
			return err2
		}
		if len(errs) > 0 {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("microflow '%s' has validation errors:\n", qualifiedName))
			for _, e := range errs {
				sb.WriteString(fmt.Sprintf("  - %s\n", e))
			}
			return fmt.Errorf("%s", sb.String())
		}
		mf.SetObjectCollection(oc)
		for _, flow := range flows {
			mf.AddFlows(flow)
		}
		for _, af := range annotationFlows {
			mf.AddFlows(af)
		}
	}

	if err := d.MicroflowsRepo.Create(string(containerID), "Documents", mf); err != nil {
		return mdlerrors.NewBackend("create microflow", err)
	}
	fmt.Fprintf(d.Output, "Created microflow: %s\n", qualifiedName)

	returnEntityName := ""
	if s.ReturnType != nil {
		if ref := paramEntityRef(s.ReturnType.Type); ref != nil && ref.Module != "" {
			returnEntityName = ref.Module + "." + ref.Name
		}
	}
	if d.TrackCreatedMicroflow != nil {
		d.TrackCreatedMicroflow(s.Name.Module, s.Name.Name, model.ID(mf.ID()), containerID, returnEntityName)
	}
	d.InvalidateCache()
	return nil
}

func ExecDropMicroflowFn(ctx context.Context, s *ast.DropMicroflowStmt, d *MicroflowDeps) error {
	if d.ConnectionManager == nil || !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if d.MicroflowsRepo == nil {
		return mdlerrors.NewBackend("microflows repo unavailable", nil)
	}
	mfs, err := d.MicroflowsRepo.ListAll()
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		mfID := model.ID(mf.ID())
		var containerID model.ID
		if d.MicroflowsRepo != nil {
			if c, err2 := d.MicroflowsRepo.GetContainerUUID(mfID); err2 == nil {
				containerID = c
			}
		}
		_ = containerID
		if mf.Name() == s.Name.Name {
			if err := d.MicroflowsRepo.Delete(mfID); err != nil {
				return mdlerrors.NewBackend("delete microflow", err)
			}
			d.InvalidateCache()
			fmt.Fprintf(d.Output, "Dropped microflow: %s.%s\n", s.Name.Module, s.Name.Name)
			return nil
		}
	}
	return mdlerrors.NewNotFound("microflow", s.Name.Module+"."+s.Name.Name)
}
