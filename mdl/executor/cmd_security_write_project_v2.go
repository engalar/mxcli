// SPDX-License-Identifier: Apache-2.0

// Package executor - gen-typed ALTER PROJECT SECURITY handler (Stage 3.3 D4).
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execAlterProjectSecurityGen handles ALTER PROJECT SECURITY using the
// gen-typed read path (getProjectSecurityGen). No sdk/security import is
// needed: security level constants are inlined as BSON string literals.
func execAlterProjectSecurityGen(ctx *ExecContext, s *ast.AlterProjectSecurityStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ps, err := getProjectSecurityGen(ctx)
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
		if err := ctx.SecurityProjectManager.SetProjectSecurityLevel(model.ID(ps.ID()), bsonLevel); err != nil {
			return mdlerrors.NewBackend("set security level", err)
		}
		invalidateProjectSecurityCache(ctx)
		fmt.Fprintf(ctx.Output, "Set project security level to %s\n", s.SecurityLevel)
	}

	if s.DemoUsersEnabled != nil {
		if err := ctx.SecurityProjectManager.SetProjectDemoUsersEnabled(model.ID(ps.ID()), *s.DemoUsersEnabled); err != nil {
			return mdlerrors.NewBackend("set demo users", err)
		}
		invalidateProjectSecurityCache(ctx)
		state := "disabled"
		if *s.DemoUsersEnabled {
			state = "enabled"
		}
		fmt.Fprintf(ctx.Output, "Demo users %s\n", state)
	}

	if s.PasswordPolicy != nil {
		return execAlterPasswordPolicy(ctx, s)
	}

	return nil
}
