// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestShowCatalogTables(t *testing.T) {
	inputs := []string{
		"SHOW CATALOG TABLES;",
		"show catalog tables;",
		"Show Catalog Tables;",
	}
	for _, input := range inputs {
		prog, errs := Build(input)
		if len(errs) > 0 {
			t.Errorf("Parse error for %q: %v", input, errs[0])
			continue
		}
		if len(prog.Statements) != 1 {
			t.Errorf("Expected 1 statement for %q, got %d", input, len(prog.Statements))
			continue
		}
		stmt, ok := prog.Statements[0].(*ast.ShowStmt)
		if !ok {
			t.Errorf("Expected ShowStmt for %q, got %T", input, prog.Statements[0])
			continue
		}
		if stmt.ObjectType != ast.ShowCatalogTables {
			t.Errorf("Expected ShowCatalogTables for %q, got %v", input, stmt.ObjectType)
		}
	}
}

func TestShowCatalogStatus(t *testing.T) {
	prog, errs := Build("SHOW CATALOG STATUS;")
	if len(errs) > 0 {
		t.Fatalf("Parse error: %v", errs[0])
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
	}
	stmt, ok := prog.Statements[0].(*ast.ShowStmt)
	if !ok {
		t.Fatalf("Expected ShowStmt, got %T", prog.Statements[0])
	}
	if stmt.ObjectType != ast.ShowCatalogStatus {
		t.Errorf("Expected ShowCatalogStatus, got %v", stmt.ObjectType)
	}
}

func TestSelectFromCatalog(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple select", "SELECT * FROM CATALOG.ENTITIES;"},
		{"with where", "SELECT Name, ModuleName FROM CATALOG.MODULES WHERE Source = '';"},
		{"lowercase", "select * from catalog.microflows;"},
		{"with alias", "SELECT e.Name FROM CATALOG.ENTITIES e;"},
		{"with join", "SELECT e.Name, a.Name FROM CATALOG.ENTITIES e JOIN CATALOG.ATTRIBUTES a ON e.Id = a.EntityId;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, errs := Build(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse error: %v", errs[0])
			}
			if len(prog.Statements) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
			}
			stmt, ok := prog.Statements[0].(*ast.SelectStmt)
			if !ok {
				t.Fatalf("Expected SelectStmt, got %T", prog.Statements[0])
			}
			if stmt.Query == "" {
				t.Error("Expected non-empty query")
			}
			t.Logf("Parsed query: %s", stmt.Query)
		})
	}
}

func TestDescribeCatalogTable(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tableName string
	}{
		{"entities", "DESCRIBE CATALOG.ENTITIES;", "entities"},
		{"modules", "DESCRIBE CATALOG.MODULES;", "modules"},
		{"widgets", "DESCRIBE CATALOG.WIDGETS;", "widgets"},
		{"lowercase", "describe catalog.pages;", "pages"},
		{"mixed case", "Describe Catalog.Microflows;", "microflows"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, errs := Build(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse error: %v", errs[0])
			}
			if len(prog.Statements) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
			}
			stmt, ok := prog.Statements[0].(*ast.DescribeCatalogTableStmt)
			if !ok {
				t.Fatalf("Expected DescribeCatalogTableStmt, got %T", prog.Statements[0])
			}
			if stmt.TableName != tt.tableName {
				t.Errorf("Expected table name %q, got %q", tt.tableName, stmt.TableName)
			}
		})
	}
}

func TestRefreshCatalog(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		full       bool
		source     bool
		force      bool
		background bool
	}{
		{"basic", "REFRESH CATALOG;", false, false, false, false},
		{"full", "REFRESH CATALOG FULL;", true, false, false, false},
		{"full force", "REFRESH CATALOG FULL FORCE;", true, false, true, false},
		{"full source", "REFRESH CATALOG FULL SOURCE;", true, true, false, false},
		{"full source force", "REFRESH CATALOG FULL SOURCE FORCE;", true, true, true, false},
		{"background", "REFRESH CATALOG BACKGROUND;", false, false, false, true},
		{"full background", "REFRESH CATALOG FULL BACKGROUND;", true, false, false, true},
		{"force", "REFRESH CATALOG FORCE;", false, false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, errs := Build(tt.input)
			if len(errs) > 0 {
				t.Fatalf("Parse error: %v", errs[0])
			}
			if len(prog.Statements) != 1 {
				t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
			}
			stmt, ok := prog.Statements[0].(*ast.RefreshCatalogStmt)
			if !ok {
				t.Fatalf("Expected RefreshCatalogStmt, got %T", prog.Statements[0])
			}
			if stmt.Full != tt.full {
				t.Errorf("Full: expected %v, got %v", tt.full, stmt.Full)
			}
			if stmt.Source != tt.source {
				t.Errorf("Source: expected %v, got %v", tt.source, stmt.Source)
			}
			if stmt.Force != tt.force {
				t.Errorf("Force: expected %v, got %v", tt.force, stmt.Force)
			}
			if stmt.Background != tt.background {
				t.Errorf("Background: expected %v, got %v", tt.background, stmt.Background)
			}
		})
	}
}
