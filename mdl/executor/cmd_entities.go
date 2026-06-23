// SPDX-License-Identifier: Apache-2.0

// Package executor - entity command entrypoints.
package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
)

// execCreateEntity now routes directly through the gen-native path.
func execCreateEntity(ctx *ExecContext, s *ast.CreateEntityStmt) error {
	return execCreateEntityGen(ctx, s)
}

// execCreateViewEntity routes through the gen-native path. The source document
// is still managed via backend helpers, while the entity write itself now uses
// the gen bridge.
func execCreateViewEntity(ctx *ExecContext, s *ast.CreateViewEntityStmt) error {
	return execCreateViewEntityGen(ctx, s)
}

// execAlterEntity now routes directly through the gen-native path.
func execAlterEntity(ctx *ExecContext, s *ast.AlterEntityStmt) error {
	return execAlterEntityGenFn(ctx, s, execContextToDeps(ctx))
}

// execDropEntity now routes directly through the gen-native drop path. The
// backend delete API already operates by IDs, so direct call sites can use the
// same implementation as the registry.
func execDropEntity(ctx *ExecContext, s *ast.DropEntityStmt) error {
	return execDropEntityGen(ctx, s)
}
