// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerConnectionHandlers(r *Registry) {
	r.Register(&ast.ConnectStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execConnect(ctx, stmt.(*ast.ConnectStmt))
	})
	r.Register(&ast.DisconnectStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDisconnect(ctx)
	})
	r.Register(&ast.StatusStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execStatus(ctx)
	})
}
