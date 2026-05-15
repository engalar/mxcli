// SPDX-License-Identifier: Apache-2.0

// Stage 3.3 D9 — trivial passthrough rename of execUpdateSecurity.
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execUpdateSecurityGen handles UPDATE SECURITY [IN Module].
func execUpdateSecurityGen(ctx *ExecContext, s *ast.UpdateSecurityStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	modules, err := getModulesFromCache(ctx)
	if err != nil {
		return err
	}

	totalModified := 0
	for _, mod := range modules {
		if s.Module != "" && mod.Name != s.Module {
			continue
		}

		dm, err := ctx.Backend.GetDomainModelGen(mod.ID)
		if err != nil || dm == nil {
			continue // module may not have a domain model
		}

		count, err := ctx.Backend.ReconcileMemberAccesses(model.ID(dm.ID()), mod.Name)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("reconcile security for module %s", mod.Name), err)
		}
		if count > 0 {
			fmt.Fprintf(ctx.Output, "Reconciled %d access rule(s) in module %s\n", count, mod.Name)
			totalModified += count
		}
	}

	if totalModified == 0 {
		fmt.Fprintf(ctx.Output, "All entity access rules are up to date\n")
	}

	return nil
}
