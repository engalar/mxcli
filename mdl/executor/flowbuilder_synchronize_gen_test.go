// SPDX-License-Identifier: Apache-2.0

// SYNCHRONIZE action adder tests (TDD).
//
// Covers both modes: SYNCHRONIZE $Var → SelectedObjects with VariableNames set,
// and SYNCHRONIZE → All with no variable names.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestAddSynchronizeActionGenSelectedObjects(t *testing.T) {
	fb := newActionTestFb()
	id := fb.addSynchronizeActionGen(&ast.SynchronizeStmt{Variable: "Order"})
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	act := actionFromObjects(t, fb).(*genMf.SynchronizeAction)
	if act.Type() != "SelectedObjects" {
		t.Fatalf("Type = %q, want SelectedObjects", act.Type())
	}
	if act.VariableNames() != "Order" {
		t.Fatalf("VariableNames = %q, want Order", act.VariableNames())
	}
	// Default ErrorHandlingType for microflow = "Rollback"
	if act.ErrorHandlingType() != "Rollback" {
		t.Fatalf("eh = %q, want Rollback", act.ErrorHandlingType())
	}
}

func TestAddSynchronizeActionGenAllObjects(t *testing.T) {
	fb := newActionTestFb()
	fb.addSynchronizeActionGen(&ast.SynchronizeStmt{})
	act := actionFromObjects(t, fb).(*genMf.SynchronizeAction)
	if act.Type() != "All" {
		t.Fatalf("Type = %q, want All", act.Type())
	}
	if act.VariableNames() != "" {
		t.Fatalf("VariableNames = %q, want empty", act.VariableNames())
	}
}
