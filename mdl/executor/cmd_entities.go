// SPDX-License-Identifier: Apache-2.0

// Package executor - entity command entrypoints.
package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
)

// execCreateEntity routes CREATE ENTITY through the gen-native path when the
// statement shape is supported there; otherwise it falls back to the isolated
// legacy implementation.
func execCreateEntity(ctx *ExecContext, s *ast.CreateEntityStmt) error {
	if canExecCreateEntityGen(ctx, s) {
		return execCreateEntityGen(ctx, s)
	}
	return execCreateEntityLegacy(ctx, s)
}

// execCreateViewEntity routes through the gen-native path. The source document
// is still managed via backend helpers, while the entity write itself now uses
// the gen bridge.
func execCreateViewEntity(ctx *ExecContext, s *ast.CreateViewEntityStmt) error {
	return execCreateViewEntityGen(ctx, s)
}

// execAlterEntity remains partially legacy-heavy because several sub-operations
// still mutate nested entity state that the gen bridge does not yet own
// end-to-end.
func execAlterEntity(ctx *ExecContext, s *ast.AlterEntityStmt) error {
	if canExecAlterEntityGen(ctx, s) {
		return execAlterEntityGen(ctx, s)
	}
	return execAlterEntityLegacy(ctx, s)
}

// execDropEntity now routes directly through the gen-native drop path. The
// backend delete API already operates by IDs, so direct call sites can use the
// same implementation as the registry.
func execDropEntity(ctx *ExecContext, s *ast.DropEntityStmt) error {
	return execDropEntityGen(ctx, s)
}
