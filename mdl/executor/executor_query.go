// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

func execShow(ctx *ExecContext, s *ast.ShowStmt) error {
	if !ctx.Connected() && s.ObjectType != ast.ShowModules && s.ObjectType != ast.ShowFragments {
		return mdlerrors.NewNotConnected()
	}
	handler, ok := showHandlers[s.ObjectType]
	if !ok {
		return mdlerrors.NewUnsupported("unknown show object type")
	}
	return handler(ctx, s)
}

func execDescribe(ctx *ExecContext, s *ast.DescribeStmt) error {
	if !ctx.Connected() && s.ObjectType != ast.DescribeFragment {
		return mdlerrors.NewNotConnected()
	}
	entry, ok := describeHandlers[s.ObjectType]
	if !ok {
		return mdlerrors.NewUnsupported("unknown describe object type")
	}
	name := s.Name.String()
	return writeDescribeJSON(ctx, name, entry.label, func() error {
		return entry.handler(ctx, s)
	})
}
