// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestCreateImageCollection(t *testing.T) {
	input := `CREATE IMAGE COLLECTION MyModule.Icons EXPORT LEVEL 'Public' (
		IMAGE MyIcon FROM FILE '/images/icon.png'
	);`
	prog, errs := Build(input)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Parse error: %v", e)
		}
		return
	}
	stmt, ok := prog.Statements[0].(*ast.CreateImageCollectionStmt)
	if !ok {
		t.Fatalf("Expected CreateImageCollectionStmt, got %T", prog.Statements[0])
	}
	if stmt.Name.Name != "Icons" {
		t.Errorf("Got name %s", stmt.Name.Name)
	}
	if stmt.ExportLevel != "Public" {
		t.Errorf("Got ExportLevel %q", stmt.ExportLevel)
	}
	if len(stmt.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(stmt.Images))
	}
	if stmt.Images[0].Name != "MyIcon" {
		t.Errorf("Got image name %q", stmt.Images[0].Name)
	}
	if stmt.Images[0].FilePath != "/images/icon.png" {
		t.Errorf("Got FilePath %q", stmt.Images[0].FilePath)
	}
}

// parseICStmts parses src and returns the statements, failing on parse errors.
func parseICStmts(t *testing.T, src string) []ast.Statement {
	t.Helper()
	prog, errs := Build(src)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Parse error: %v", e)
		}
		t.Fatalf("unexpected parse errors")
	}
	return prog.Statements
}

func TestAlterImageCollection_Add(t *testing.T) {
	stmts := parseICStmts(t, `ALTER IMAGE COLLECTION MyModule.Icons ADD IMAGE logo FROM FILE './logo.png';`)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	s, ok := stmts[0].(*ast.AlterImageCollectionStmt)
	if !ok {
		t.Fatalf("expected *ast.AlterImageCollectionStmt, got %T", stmts[0])
	}
	if s.Name.Module != "MyModule" || s.Name.Name != "Icons" {
		t.Errorf("unexpected name: %v", s.Name)
	}
	if len(s.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(s.Actions))
	}
	add, ok := s.Actions[0].(*ast.AddImageAction)
	if !ok {
		t.Fatalf("expected *ast.AddImageAction, got %T", s.Actions[0])
	}
	if add.ImageName != "logo" {
		t.Errorf("ImageName = %q, want logo", add.ImageName)
	}
	if add.FilePath != "./logo.png" {
		t.Errorf("FilePath = %q, want ./logo.png", add.FilePath)
	}
}

func TestAlterImageCollection_MultipleActions(t *testing.T) {
	src := `ALTER IMAGE COLLECTION Mod.Icons
		ADD IMAGE logo FROM FILE './logo.png',
		DROP IMAGE banner,
		RENAME IMAGE old TO new_name,
		SET IMAGE logo FROM FILE './logo2.png',
		MOVE TO Other.Icons,
		EXPORT IMAGE logo TO FILE './out.png';`
	stmts := parseICStmts(t, src)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	s := stmts[0].(*ast.AlterImageCollectionStmt)
	if len(s.Actions) != 6 {
		t.Fatalf("expected 6 actions, got %d: %v", len(s.Actions), s.Actions)
	}
	if _, ok := s.Actions[0].(*ast.AddImageAction); !ok {
		t.Errorf("actions[0]: want AddImageAction, got %T", s.Actions[0])
	}
	if _, ok := s.Actions[1].(*ast.DropImageAction); !ok {
		t.Errorf("actions[1]: want DropImageAction, got %T", s.Actions[1])
	}
	if r, ok := s.Actions[2].(*ast.RenameImageAction); !ok || r.From != "old" || r.To != "new_name" {
		t.Errorf("actions[2]: want RenameImageAction{old,new_name}, got %T %v", s.Actions[2], s.Actions[2])
	}
	if _, ok := s.Actions[3].(*ast.SetImageAction); !ok {
		t.Errorf("actions[3]: want SetImageAction, got %T", s.Actions[3])
	}
	mv, ok := s.Actions[4].(*ast.MoveImageCollectionAction)
	if !ok || mv.Target.Module != "Other" || mv.Target.Name != "Icons" {
		t.Errorf("actions[4]: want MoveImageCollectionAction{Other.Icons}, got %T %v", s.Actions[4], s.Actions[4])
	}
	if _, ok := s.Actions[5].(*ast.ExportImageAction); !ok {
		t.Errorf("actions[5]: want ExportImageAction, got %T", s.Actions[5])
	}
}
