// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/pageast"
)

// Lift converts a parsed CREATE PAGE AST statement to a PageDocument.
//
// The AST→model conversion (including the full widget tree) lives in the
// pageast package, so canonical/page does not import the executor package
// (which would create an import cycle: executor imports canonical/page to
// register its codec).
//
// moduleName fills the owning module when the statement's qualified name omits
// it.
func Lift(s *ast.CreatePageStmtV3, moduleName string) (*PageDocument, error) {
	if s == nil {
		return nil, fmt.Errorf("page.Lift: nil statement")
	}
	pm, err := pageast.PageASTToModel(s, moduleName)
	if err != nil {
		return nil, err
	}
	return &PageDocument{pm: pm}, nil
}
