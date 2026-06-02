// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/pageast"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// pageASTToModel converts a CreatePageStmtV3 AST to a types.PageModel.
// The conversion logic lives in the pageast package so that both this executor
// and the canonical/page layer can convert without importing each other. This
// thin wrapper preserves the existing call sites; ctx is accepted but unused
// (the conversion needs no execution context).
func pageASTToModel(_ *ExecContext, s *ast.CreatePageStmtV3) (*types.PageModel, error) {
	return pageast.PageASTToModel(s, s.Name.Module)
}

// astWidgetToNode converts a single AST widget to a WidgetNode. Thin wrapper
// over pageast.AstWidgetToNode for existing executor call sites.
func astWidgetToNode(_ *ExecContext, w *ast.WidgetV3, moduleName string) (*types.WidgetNode, error) {
	return pageast.AstWidgetToNode(w, moduleName)
}
