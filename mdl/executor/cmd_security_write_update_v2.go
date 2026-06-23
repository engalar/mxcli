// SPDX-License-Identifier: Apache-2.0

// Stage 3.3 D9 — trivial passthrough rename of execUpdateSecurity.
package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// execUpdateSecurityGenFn is the HandlerDeps version of execUpdateSecurityGen.
func execUpdateSecurityGenFn(ctx context.Context, s *ast.UpdateSecurityStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	modules, err := getModulesFromCache(ectx)
	if err != nil {
		return err
	}
	totalModified := 0
	for _, mod := range modules {
		if s.Module != "" && mod.Name != s.Module {
			continue
		}
		dm, err := getDomainModelGenCached(ectx, mod.ID)
		if err != nil || dm == nil {
			continue
		}
		msgs, err := deps.SecurityEntityAccessManager.ReconcileMemberAccesses(model.ID(dm.ID()), mod.Name)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("reconcile security for module %s", mod.Name), err)
		}
		if len(msgs) > 0 {
			invalidateDomainModelGenForModule(ectx, mod.ID)
		}
		for _, msg := range msgs {
			fmt.Fprintf(deps.Output, "  [%s] %s\n", mod.Name, msg)
			totalModified++
		}
	}
	if totalModified == 0 {
		fmt.Fprintf(deps.Output, "All entity access rules are up to date\n")
	}
	return nil
}

// execUpdateSecurityGen handles UPDATE SECURITY. Delegates to Fn version.
