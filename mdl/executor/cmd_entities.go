// SPDX-License-Identifier: Apache-2.0

// Package executor - entity command entrypoints.
package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
)

// execAlterEntity now routes directly through the gen-native path.
func execAlterEntity(ctx *ExecContext, s *ast.AlterEntityStmt) error {
	return ExecAlterEntityGenFn(ctx, s, ctx.Deps)
}
