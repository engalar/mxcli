// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5a — ImportFromMapping + ExportToMapping adder tests (TDD).
//
// Both wrap XML/JSON marshaling actions:
//
//   addImportFromMappingActionGen
//     → genMf.ImportXmlAction
//        ├── XmlDocumentVariableName = source string variable
//        └── ResultHandling
//             ├── OutputVariableName  = result variable
//             └── ImportMappingCall   = mapping reference + cardinality
//
//   addExportToMappingActionGen
//     → genMf.ExportXmlAction
//        ├── MappingQualifiedName        = mapping reference
//        └── MappingArgumentVariableName = source entity variable
//
// Backend-driven cardinality inference (legacy reads import mapping
// metadata to flip SingleObject false when JsonStructure root is
// Array or Element[0].MaxOccurs > 1) is deferred — the offline write
// path defaults to single-object, matching the legacy fallback when
// fb.backend is nil.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddImportFromMappingActionGenSetsAllFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ImportFromMappingStmt{
		OutputVariable: "Order",
		Mapping:        ast.QualifiedName{Module: "Sales", Name: "OrderImport"},
		SourceVariable: "JsonInput",
	}
	fb.addImportFromMappingActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ImportXmlAction)
	if act.XmlDocumentVariableName() != "JsonInput" {
		t.Fatalf("xml doc var = %q, want JsonInput", act.XmlDocumentVariableName())
	}
	rh, ok := act.ResultHandling().(*genMf.ResultHandling)
	if !ok {
		t.Fatalf("ResultHandling = %T, want *ResultHandling", act.ResultHandling())
	}
	if rh.OutputVariableName() != "Order" {
		t.Fatalf("output var = %q, want Order", rh.OutputVariableName())
	}
	call, ok := rh.ImportMappingCall().(*genMf.ImportMappingCall)
	if !ok {
		t.Fatalf("ImportMappingCall = %T, want *ImportMappingCall", rh.ImportMappingCall())
	}
	if call.MappingRefID() != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("mapping ID = %q", call.MappingRefID())
	}
}

func TestAddImportFromMappingActionGenDefaultsToSingleObject(t *testing.T) {
	// Without backend metadata, legacy defaults SingleObject=true.
	fb := newActionTestFb()
	stmt := &ast.ImportFromMappingStmt{
		Mapping:        ast.QualifiedName{Module: "M", Name: "Map"},
		SourceVariable: "Src",
	}
	fb.addImportFromMappingActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ImportXmlAction)
	rh := act.ResultHandling().(*genMf.ResultHandling)
	call := rh.ImportMappingCall().(*genMf.ImportMappingCall)
	if !call.ForceSingleOccurrence() {
		t.Fatal("offline default ForceSingleOccurrence should be true")
	}
}

func TestAddImportFromMappingActionGenContinueErrorHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ImportFromMappingStmt{
		Mapping:        ast.QualifiedName{Module: "M", Name: "Map"},
		SourceVariable: "Src",
		ErrorHandling:  &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue},
	}
	fb.addImportFromMappingActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ImportXmlAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

func TestAddExportToMappingActionGenSetsFields(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ExportToMappingStmt{
		OutputVariable: "JsonOutput",
		Mapping:        ast.QualifiedName{Module: "Sales", Name: "OrderExport"},
		SourceVariable: "Order",
	}
	fb.addExportToMappingActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ExportXmlAction)
	if act.MappingRefID() != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("mapping ID = %q", act.MappingRefID())
	}
	if act.MappingArgumentVariableName() != "Order" {
		t.Fatalf("mapping arg var = %q", act.MappingArgumentVariableName())
	}
}

func TestAddExportToMappingActionGenContinueErrorHandling(t *testing.T) {
	fb := newActionTestFb()
	stmt := &ast.ExportToMappingStmt{
		Mapping:        ast.QualifiedName{Module: "M", Name: "Map"},
		SourceVariable: "Src",
		ErrorHandling:  &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue},
	}
	fb.addExportToMappingActionGen(stmt)
	act := actionFromObjects(t, fb).(*genMf.ExportXmlAction)
	if act.ErrorHandlingType() != "Continue" {
		t.Fatalf("eh = %q, want Continue", act.ErrorHandlingType())
	}
}

func TestAddImportFromMappingActionGenAdvancesPos(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	fb.addImportFromMappingActionGen(&ast.ImportFromMappingStmt{
		Mapping: ast.QualifiedName{Module: "M", Name: "Map"},
	})
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}

func TestAddExportToMappingActionGenAdvancesPos(t *testing.T) {
	fb := newActionTestFb()
	startX := fb.posX
	fb.addExportToMappingActionGen(&ast.ExportToMappingStmt{
		Mapping: ast.QualifiedName{Module: "M", Name: "Map"},
	})
	if fb.posX != startX+HorizontalSpacing {
		t.Fatalf("posX = %d, want %d", fb.posX, startX+HorizontalSpacing)
	}
}
