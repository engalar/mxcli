// SPDX-License-Identifier: Apache-2.0

package page

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func ExecAlterPageFn(ctx context.Context, s *ast.AlterPageStmt, deps *executor.HandlerDeps) error {
	return executor.ExecAlterPageFn(ctx, s, deps)
}
