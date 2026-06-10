// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestDocumentMoverRegistry_Coverage asserts every movable document type has a
// registered mover. Adding a new DocumentType to the MOVE command must come with
// a registry entry; this test fails loudly if one is forgotten (OCP guard).
func TestDocumentMoverRegistry_Coverage(t *testing.T) {
	expected := []ast.DocumentType{
		ast.DocumentTypePage,
		ast.DocumentTypeMicroflow,
		ast.DocumentTypeSnippet,
		ast.DocumentTypeNanoflow,
		ast.DocumentTypeEnumeration,
		ast.DocumentTypeConstant,
		ast.DocumentTypeDatabaseConnection,
		ast.DocumentTypeJavaAction,
		ast.DocumentTypeJavaScriptAction,
		ast.DocumentTypeLayout,
		ast.DocumentTypeWorkflow,
	}
	for _, dt := range expected {
		if _, ok := documentMoverRegistry[dt]; !ok {
			t.Errorf("documentMoverRegistry missing entry for %q", dt)
		}
	}
	if len(documentMoverRegistry) != len(expected) {
		t.Errorf("documentMoverRegistry has %d entries, expected %d", len(documentMoverRegistry), len(expected))
	}
}
