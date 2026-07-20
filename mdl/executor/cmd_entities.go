// SPDX-License-Identifier: Apache-2.0

// Package executor - entity command entrypoints.
package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
)

// ExecCreateEntity now routes directly through the gen-native path.
func ExecCreateEntity(ctx *ExecContext, s *ast.CreateEntityStmt) error {
	return execCreateEntityGen(ctx, s)
}

// ExecCreateViewEntity routes through the gen-native path. The source document
// is still managed via backend helpers, while the entity write itself now uses
// the gen bridge.
func ExecCreateViewEntity(ctx *ExecContext, s *ast.CreateViewEntityStmt) error {
	return execCreateViewEntityGen(ctx, s)
}

// execAlterEntity now routes directly through the gen-native path.
func execAlterEntity(ctx *ExecContext, s *ast.AlterEntityStmt) error {
	return ExecAlterEntityGenFn(ctx, s, ctx.Deps)
}

// ExecDropEntity now routes directly through the gen-native drop path. The
// backend delete API already operates by IDs, so direct call sites can use the
// same implementation as the registry.
func ExecDropEntity(ctx *ExecContext, s *ast.DropEntityStmt) error {
	return execDropEntityGen(ctx, s)
}
