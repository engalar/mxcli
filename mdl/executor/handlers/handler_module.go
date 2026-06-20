// SPDX-License-Identifier: Apache-2.0

// Package handlers provides statement handler registration functions.
// Each handler captures its exact dependencies via closure — no ExecContext.
package handlers

import (
	"context"
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/executor"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

func RegisterModuleHandlers(reg *executor.Registry, lister backend.ModuleLister, writer backend.ModuleWriter, output io.Writer) {
	reg.RegisterFuture("CreateModule", func(ctx context.Context, stmt ast.Statement) error {
		return execCreateModule(ctx, stmt.(*ast.CreateModuleStmt), lister, writer, output)
	})
	reg.RegisterFuture("DropModule", func(ctx context.Context, stmt ast.Statement) error {
		return execDropModule(ctx, stmt.(*ast.DropModuleStmt), lister, writer, output)
	})
}

func execCreateModule(ctx context.Context, s *ast.CreateModuleStmt, lister backend.ModuleLister, writer backend.ModuleWriter, output io.Writer) error {
	if s == nil || s.Name == "" {
		return mdlerrors.NewValidation("CREATE MODULE requires a name")
	}
	modules, err := lister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	for _, m := range modules {
		if m.Name == s.Name {
			fmt.Fprintf(output, "Module '%s' already exists\n", s.Name)
			return nil
		}
	}
	module := &model.Module{Name: s.Name}
	if err := writer.CreateModule(module); err != nil {
		return mdlerrors.NewBackend("create module", err)
	}
	fmt.Fprintf(output, "Created module: %s\n", s.Name)
	return nil
}

func execDropModule(ctx context.Context, s *ast.DropModuleStmt, lister backend.ModuleLister, writer backend.ModuleWriter, output io.Writer) error {
	if s == nil || s.Name == "" {
		return mdlerrors.NewValidation("DROP MODULE requires a name")
	}
	modules, err := lister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	var target *model.Module
	for _, m := range modules {
		if m.Name == s.Name {
			target = m
			break
		}
	}
	if target == nil {
		return mdlerrors.NewNotFound("module", s.Name)
	}
	if err := writer.DeleteModuleWithCleanup(target.ID, target.Name); err != nil {
		return mdlerrors.NewBackend("delete module", err)
	}
	fmt.Fprintf(output, "Dropped module: %s\n", s.Name)
	return nil
}
