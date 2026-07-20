// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// execShowContext handles SHOW CONTEXT OF <name> [DEPTH n] command.
// Catalog SQLite system has been removed. Use SHOW CALLERS / SHOW REFERENCES instead.
func execShowContext(ctx *ExecContext, s *ast.ShowStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	return mdlerrors.NewUnsupported(
		"SHOW CONTEXT OF has been replaced by MXGraph-based commands.\n" +
			"Use SHOW CALLERS OF / SHOW CALLEES OF / SHOW REFERENCES TO / SHOW IMPACT OF instead.")
}

func execShowContextDeps(ctx context.Context, deps *HandlerDeps, s *ast.ShowStmt) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	return mdlerrors.NewUnsupported(
		"SHOW CONTEXT OF has been replaced by MXGraph-based commands.\n" +
			"Use SHOW CALLERS OF / SHOW CALLEES OF / SHOW REFERENCES TO / SHOW IMPACT OF instead.")
}

// ── Security ──

