// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — Tests for the gen-typed Mermaid renderer.
//
// These exercise both microflow and nanoflow paths against the
// expr-checker fixture: trivial Start→End flow, an if/else split
// (covering case labels + edges + style), a not-found path, and the
// metadata footer (`%% @type flowchart`, `%% @direction LR`,
// optional `%% @nodeinfo {…}`).

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestMicroflowToMermaidGen_TrivialFlow renders the empty
// MyFirstLogic microflow and verifies the synthetic Start/End
// flowchart shape is emitted.
func TestMicroflowToMermaidGen_TrivialFlow(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)

	qn := ast.QualifiedName{Module: "MyFirstModule", Name: "MyFirstLogic"}
	if err := microflowToMermaidGen(ctx, qn); err != nil {
		t.Fatalf("microflowToMermaidGen: %v", err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "flowchart LR") {
		t.Errorf("expected 'flowchart LR' header; got: %s", got)
	}
}

// TestMicroflowToMermaidGen_IfElseFlow renders a microflow with a
// boolean ExclusiveSplit and verifies that:
//   - flowchart LR header
//   - at least one Mermaid diamond ({…}) for the split
//   - at least one rounded rect or stadium node
//   - `--> ` edges (with optional case labels via |…|)
//   - the start-node style line
//   - the metadata footer (@type flowchart, @direction LR)
//   - and the @nodeinfo block when any node has details
func TestMicroflowToMermaidGen_IfElseFlow(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)

	qn := ast.QualifiedName{Module: "Administration", Name: "SaveNewAccount"}
	if err := microflowToMermaidGen(ctx, qn); err != nil {
		t.Fatalf("microflowToMermaidGen SaveNewAccount: %v", err)
	}
	got := out.String()
	mustContain(t, got,
		"flowchart LR\n",
		"{",     // ExclusiveSplit diamond
		" --> ", // at least one connector
		"\n%% @type flowchart\n",
		"%% @direction LR\n",
		"style ", // start-node styling
	)
	if !strings.Contains(got, "%% @nodeinfo {") {
		t.Errorf("expected @nodeinfo block in output; got:\n%s", got)
	}
	// Stadium node label for an EndEvent.
	if !strings.Contains(got, "([") {
		t.Errorf("expected stadium-node syntax for events; got:\n%s", got)
	}
}

// TestNanoflowToMermaidGen_FixtureRender renders one of the fixture
// nanoflows and verifies basic structural output. We pick
// `Atlas_Web_Content.DS_LoginContext` for parity with the elk smoke test.
func TestNanoflowToMermaidGen_FixtureRender(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)

	qn := ast.QualifiedName{Module: "Atlas_Web_Content", Name: "DS_LoginContext"}
	if err := nanoflowToMermaidGen(ctx, qn); err != nil {
		t.Fatalf("nanoflowToMermaidGen: %v", err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "flowchart LR\n") {
		t.Errorf("expected flowchart LR header; got: %s", got)
	}
	if !strings.Contains(got, "%% @type flowchart\n") {
		t.Errorf("expected @type flowchart metadata; got:\n%s", got)
	}
}

// TestMicroflowToMermaidGen_NotFound verifies the legacy error path.
func TestMicroflowToMermaidGen_NotFound(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)
	err := microflowToMermaidGen(ctx, ast.QualifiedName{Module: "X", Name: "Y"})
	if err == nil {
		t.Fatal("expected error for missing microflow, got nil")
	}
	if !strings.Contains(err.Error(), "microflow") {
		t.Errorf("error should mention microflow; got %v", err)
	}
}

// TestNanoflowToMermaidGen_NotFound — same for the nanoflow path.
func TestNanoflowToMermaidGen_NotFound(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)
	err := nanoflowToMermaidGen(ctx, ast.QualifiedName{Module: "X", Name: "Y"})
	if err == nil {
		t.Fatal("expected error for missing nanoflow, got nil")
	}
	if !strings.Contains(err.Error(), "nanoflow") {
		t.Errorf("error should mention nanoflow; got %v", err)
	}
}
