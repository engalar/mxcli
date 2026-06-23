// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execAlterPasswordPolicyFn is the HandlerDeps version of execAlterPasswordPolicy.
func execAlterPasswordPolicyFn(ctx context.Context, s *ast.AlterProjectSecurityStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	opts := s.PasswordPolicy
	if opts == nil {
		return nil
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	ps, err := getProjectSecurityGen(ectx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	if err := deps.SecurityProjectManager.SetPasswordPolicy(
		model.ID(ps.ID()),
		opts.MinLength,
		opts.RequireDigit,
		opts.RequireMixedCase,
		opts.RequireSymbol,
	); err != nil {
		return mdlerrors.NewBackend("set password policy", err)
	}
	invalidateProjectSecurityCache(ectx)
	fmt.Fprintf(deps.Output, "Updated password policy")
	if opts.MinLength != nil {
		fmt.Fprintf(deps.Output, ": min_length=%d", *opts.MinLength)
	}
	if opts.RequireDigit != nil {
		fmt.Fprintf(deps.Output, ", require_digit=%v", *opts.RequireDigit)
	}
	if opts.RequireMixedCase != nil {
		fmt.Fprintf(deps.Output, ", require_mixed_case=%v", *opts.RequireMixedCase)
	}
	if opts.RequireSymbol != nil {
		fmt.Fprintf(deps.Output, ", require_symbol=%v", *opts.RequireSymbol)
	}
	fmt.Fprintln(deps.Output)
	return nil
}

// execAlterPasswordPolicy handles:
//
//	ALTER PROJECT SECURITY PASSWORD POLICY (min_length: N, require_digit: BOOL, ...)
func execAlterPasswordPolicy(ctx *ExecContext, s *ast.AlterProjectSecurityStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	opts := s.PasswordPolicy
	if opts == nil {
		return nil
	}

	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}

	if err := ctx.SecurityProjectManager.SetPasswordPolicy(
		model.ID(ps.ID()),
		opts.MinLength,
		opts.RequireDigit,
		opts.RequireMixedCase,
		opts.RequireSymbol,
	); err != nil {
		return mdlerrors.NewBackend("set password policy", err)
	}
	invalidateProjectSecurityCache(ctx)

	fmt.Fprintf(ctx.Output, "Updated password policy")
	if opts.MinLength != nil {
		fmt.Fprintf(ctx.Output, ": min_length=%d", *opts.MinLength)
	}
	if opts.RequireDigit != nil {
		fmt.Fprintf(ctx.Output, ", require_digit=%v", *opts.RequireDigit)
	}
	if opts.RequireMixedCase != nil {
		fmt.Fprintf(ctx.Output, ", require_mixed_case=%v", *opts.RequireMixedCase)
	}
	if opts.RequireSymbol != nil {
		fmt.Fprintf(ctx.Output, ", require_symbol=%v", *opts.RequireSymbol)
	}
	fmt.Fprintln(ctx.Output)
	return nil
}
