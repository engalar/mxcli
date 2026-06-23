package domainmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
)

func ValidateEnumeration(s *ast.CreateEnumerationStmt) []linter.Violation {
	if s == nil {
		return nil
	}
	var violations []linter.Violation
	for _, v := range s.Values {
		if strings.EqualFold(v.Name, "empty") {
			violations = append(violations, linter.Violation{
				Message:  fmt.Sprintf("enumeration value '%s' is a reserved word", v.Name),
				Severity: linter.SeverityError,
			})
		}
	}
	return violations
}

func ExecCreateEnumerationFn(ctx context.Context, s *ast.CreateEnumerationStmt, d *DomainModelDeps) error {
	if d.ConnectionManager == nil || !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	if violations := ValidateEnumeration(s); len(violations) > 0 {
		var msgs []string
		for _, v := range violations {
			msgs = append(msgs, v.Message)
		}
		return mdlerrors.NewValidationf("invalid enumeration '%s':\n  - %s",
			s.Name.String(), strings.Join(msgs, "\n  - "))
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return err
	}

	existingEnum := d.FindEnumeration(s.Name.Module, s.Name.Name)
	if existingEnum != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExistsMsg("enumeration", s.Name.Module+"."+s.Name.Name,
			fmt.Sprintf("enumeration already exists: %s.%s (use create or modify to update)", s.Name.Module, s.Name.Name))
	}

	var values []model.EnumerationValue
	for _, v := range s.Values {
		values = append(values, model.EnumerationValue{
			Name: v.Name,
			Caption: &model.Text{
				Translations: map[string]string{"en_US": v.Caption},
			},
		})
	}

	enum := &model.Enumeration{
		ContainerID:   module.ID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
		Values:        values,
	}

	if existingEnum != nil && s.CreateOrModify {
		enum.ID = existingEnum.ID
		if err := d.EnumerationWriter.UpdateEnumeration(enum); err != nil {
			return mdlerrors.NewBackend("update enumeration", err)
		}
		d.InvalidateHierarchy()
		fmt.Fprintf(d.Output, "Modified enumeration: %s\n", s.Name)
		return nil
	}

	if err := d.EnumerationWriter.CreateEnumeration(enum); err != nil {
		return mdlerrors.NewBackend("create enumeration", err)
	}

	d.InvalidateHierarchy()
	fmt.Fprintf(d.Output, "Created enumeration: %s\n", s.Name)
	return nil
}

func ExecDropEnumerationFn(ctx context.Context, s *ast.DropEnumerationStmt, d *DomainModelDeps) error {
	if d.ConnectionManager == nil || !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	enums, err := d.EnumerationReader.ListEnumerations()
	if err != nil {
		return mdlerrors.NewBackend("list enumerations", err)
	}

	var candidates []*model.Enumeration
	for _, enum := range enums {
		if enum.Name != s.Name.Name {
			continue
		}
		if s.Name.Module != "" {
			modName := d.FindModuleName(enum.ContainerID)
			if modName == s.Name.Module {
				candidates = append(candidates, enum)
			}
		} else {
			candidates = append(candidates, enum)
		}
	}

	if len(candidates) == 0 {
		return mdlerrors.NewNotFound("enumeration", s.Name.String())
	}
	if len(candidates) > 1 {
		return mdlerrors.NewValidationf("multiple enumerations named '%s' — specify the module: %s.<module_name>.%s",
			s.Name.Name, "", s.Name.Name)
	}

	enum := candidates[0]
	if err := d.EnumerationWriter.DeleteEnumeration(enum.ID); err != nil {
		return mdlerrors.NewBackend("drop enumeration", err)
	}

	d.InvalidateHierarchy()
	fmt.Fprintf(d.Output, "Dropped enumeration: %s\n", s.Name)
	return nil
}
