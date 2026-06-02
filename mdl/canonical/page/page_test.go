// SPDX-License-Identifier: Apache-2.0

package page_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/canonical/page"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func minimalPageAST() *ast.CreatePageStmtV3 {
	return &ast.CreatePageStmtV3{
		Name:   ast.QualifiedName{Module: "M", Name: "TestPage"},
		Layout: "Atlas_Core.Atlas_Default",
	}
}

func TestLift_MinimalPage(t *testing.T) {
	doc, err := page.Lift(minimalPageAST(), "M")
	require.NoError(t, err)
	require.NotNil(t, doc)
	pm := doc.PageModel()
	require.NotNil(t, pm)
	assert.Equal(t, "M", pm.ModuleName)
	assert.Equal(t, "TestPage", pm.Name)
	assert.Equal(t, "Atlas_Core.Atlas_Default", pm.Layout)
}

func TestLift_TopLevelFields(t *testing.T) {
	s := &ast.CreatePageStmtV3{
		Name:   ast.QualifiedName{Module: "Sales", Name: "OrderForm"},
		Title:  "Order Form",
		Layout: "Atlas_Core.Atlas_Default",
		Folder: "Orders/Forms",
		Parameters: []ast.PageParameter{
			{Name: "Order", EntityType: ast.QualifiedName{Module: "Sales", Name: "Order"}},
		},
	}
	doc, err := page.Lift(s, "Sales")
	require.NoError(t, err)
	pm := doc.PageModel()
	assert.Equal(t, "Sales", pm.ModuleName)
	assert.Equal(t, "OrderForm", pm.Name)
	assert.Equal(t, "Order Form", pm.Title)
	assert.Equal(t, "Orders/Forms", pm.Folder)
	require.Len(t, pm.Params, 1)
	assert.Equal(t, "Order", pm.Params[0].Name)
	assert.Equal(t, "Sales.Order", pm.Params[0].EntityName)
}

func TestLift_FallsBackToModuleNameArg(t *testing.T) {
	// Statement name omits the module; the moduleName argument fills it.
	s := &ast.CreatePageStmtV3{
		Name: ast.QualifiedName{Name: "Bare"},
	}
	doc, err := page.Lift(s, "Fallback")
	require.NoError(t, err)
	assert.Equal(t, "Fallback", doc.PageModel().ModuleName)
}

func TestLift_NilStmt_ReturnsError(t *testing.T) {
	_, err := page.Lift(nil, "M")
	assert.Error(t, err)
}
