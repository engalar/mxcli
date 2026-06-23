// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

func execShow(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
	handler, ok := showHandlers[s.ObjectType]
	if !ok {
		return mdlerrors.NewUnsupported("unknown show object type")
	}
	return handler(ctx, s, deps)
}

func execDescribe(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
	entry, ok := describeHandlers[s.ObjectType]
	if !ok {
		return mdlerrors.NewUnsupported("unknown describe object type")
	}
	name := s.Name.String()
	return writeDescribeJSON(ctx, name, entry.label, deps, func() error {
		return entry.handler(ctx, s, deps)
	})
}
