// SPDX-License-Identifier: Apache-2.0

// Package executor - gen-typed ALTER PROJECT SECURITY handler (Stage 3.3 D4).
package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execAlterProjectSecurityGenFn is the HandlerDeps version of execAlterProjectSecurityGen.
func execAlterProjectSecurityGenFn(ctx context.Context, s *ast.AlterProjectSecurityStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	ps, err := getProjectSecurityGen(ectx)
	if err != nil || ps == nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if s.SecurityLevel != "" {
		var bsonLevel string
		switch s.SecurityLevel {
		case "Production":
			bsonLevel = "CheckEverything"
		case "Prototype":
			bsonLevel = "CheckFormsAndMicroflows"
		case "Off":
			bsonLevel = "CheckNothing"
		default:
			return mdlerrors.NewUnsupported(fmt.Sprintf("unknown security level: %s", s.SecurityLevel))
		}
		if err := deps.SecurityProjectManager.SetProjectSecurityLevel(model.ID(ps.ID()), bsonLevel); err != nil {
			return mdlerrors.NewBackend("set security level", err)
		}
		invalidateProjectSecurityCache(ectx)
		fmt.Fprintf(deps.Output, "Set project security level to %s\n", s.SecurityLevel)
	}
	if s.DemoUsersEnabled != nil {
		if err := deps.SecurityProjectManager.SetProjectDemoUsersEnabled(model.ID(ps.ID()), *s.DemoUsersEnabled); err != nil {
			return mdlerrors.NewBackend("set demo users", err)
		}
		invalidateProjectSecurityCache(ectx)
		state := "disabled"
		if *s.DemoUsersEnabled {
			state = "enabled"
		}
		fmt.Fprintf(deps.Output, "Demo users %s\n", state)
	}
	if s.PasswordPolicy != nil {
		return execAlterPasswordPolicyFn(ctx, s, deps)
	}
	return nil
}

// execAlterProjectSecurityGen handles ALTER PROJECT SECURITY. Delegates to Fn version.
func execAlterProjectSecurityGen(ctx *ExecContext, s *ast.AlterProjectSecurityStmt) error {
	return execAlterProjectSecurityGenFn(ctx, s, execContextToDeps(ctx))
}
