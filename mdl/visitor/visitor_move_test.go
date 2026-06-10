// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestMovePage_ToFolder(t *testing.T) {
	input := `MOVE PAGE MyModule.OldPage TO FOLDER 'Admin/Pages';`
	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Parse error: %v", e)
		}
		return
	}
	stmt, ok := prog.Statements[0].(*ast.MoveStmt)
	if !ok {
		t.Fatalf("Expected MoveStmt, got %T", prog.Statements[0])
	}
	if stmt.Name.Name != "OldPage" {
		t.Errorf("Got name %s", stmt.Name.Name)
	}
	if stmt.Folder != "Admin/Pages" {
		t.Errorf("Got Folder %q", stmt.Folder)
	}
}

func TestMoveEntity_ToModule(t *testing.T) {
	input := `MOVE ENTITY MyModule.Customer TO OtherModule;`
	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Parse error: %v", e)
		}
		return
	}
	stmt, ok := prog.Statements[0].(*ast.MoveStmt)
	if !ok {
		t.Fatalf("Expected MoveStmt, got %T", prog.Statements[0])
	}
	if stmt.Name.Name != "Customer" {
		t.Errorf("Got name %s", stmt.Name.Name)
	}
	if stmt.TargetModule != "OtherModule" {
		t.Errorf("Got TargetModule %q", stmt.TargetModule)
	}
}

func TestParse_MoveJavaAction(t *testing.T) {
	prog, errs := Build("MOVE JAVA ACTION MyModule.MyAction TO FOLDER 'actions' IN MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := prog.Statements[0].(*ast.MoveStmt)
	if !ok || s.DocumentType != ast.DocumentTypeJavaAction {
		t.Fatalf("expected JAVA ACTION, got %T/%q", prog.Statements[0], s.DocumentType)
	}
	if s.Folder != "actions" {
		t.Errorf("Folder=%q", s.Folder)
	}
}

func TestParse_MoveJavaScriptAction(t *testing.T) {
	prog, errs := Build("MOVE JAVASCRIPT ACTION MyModule.MyJSAction TO MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := prog.Statements[0].(*ast.MoveStmt)
	if !ok || s.DocumentType != ast.DocumentTypeJavaScriptAction {
		t.Fatalf("expected JAVASCRIPT ACTION, got %T/%q", prog.Statements[0], s.DocumentType)
	}
}

func TestParse_MoveLayout(t *testing.T) {
	prog, errs := Build("MOVE LAYOUT MyModule.MyLayout TO FOLDER 'layouts' IN MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := prog.Statements[0].(*ast.MoveStmt)
	if !ok || s.DocumentType != ast.DocumentTypeLayout {
		t.Fatalf("expected LAYOUT, got %T/%q", prog.Statements[0], s.DocumentType)
	}
}

func TestParse_MoveWorkflow(t *testing.T) {
	prog, errs := Build("MOVE WORKFLOW MyModule.MyWorkflow TO MyModule;")
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	s, ok := prog.Statements[0].(*ast.MoveStmt)
	if !ok || s.DocumentType != ast.DocumentTypeWorkflow {
		t.Fatalf("expected WORKFLOW, got %T/%q", prog.Statements[0], s.DocumentType)
	}
}
