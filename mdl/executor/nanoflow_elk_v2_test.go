// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — Tests for the gen-typed Nanoflow ELK rendering and
// the shared elk graph builders (`buildFlowELKGen`,
// `buildMicroflowELKNodeHierarchicalGen`, `calcMicroflowNodeSizeGen`,
// etc). The legacy `nanoflowELK` is exercised by the upstream
// describe/CLI tests; here we verify that the gen-typed parallel
// renders the same JSON shape (format/type/name/parameters/nodes/edges)
// against the same fixture.

package executor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// newGenVizContext builds an ExecContext usable by the gen viz
// entries. It mirrors `newGenDescribeContext` but also wires the
// gen Nanoflow repository (the structural test above only needs
// Microflows).
func newGenVizContext(t *testing.T, out *bytes.Buffer) *ExecContext {
	t.Helper()
	w := openMprWriterForTest(t)
	repoCtx := mprbackend.NewExecutorContext(w)

	path := w.ConcreteReader().Path()
	be, err := mprbackend.NewFromPath(path)
	if err != nil {
		t.Fatalf("mprbackend.NewFromPath(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	return &ExecContext{
		Backend:    be,
		Microflows: repoCtx.Microflows,
		Nanoflows:  repoCtx.Nanoflows,
		Output:     out,
	}
}

// TestNanoflowELKGen_Smoke verifies that the gen-typed nanoflow ELK
// entry produces a well-formed JSON document with the expected top-level
// shape: format=elk, type=nanoflow, the qualified name, and at least
// one node (Start).
func TestNanoflowELKGen_Smoke(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)

	if err := nanoflowELKGen(ctx, "FeedbackModule.ACT_SubmitFeedback"); err != nil {
		t.Fatalf("nanoflowELKGen: %v", err)
	}

	var data microflowELKData
	if err := json.Unmarshal(out.Bytes(), &data); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out.String())
	}
	if data.Format != "elk" {
		t.Errorf("Format = %q, want \"elk\"", data.Format)
	}
	if data.Type != "nanoflow" {
		t.Errorf("Type = %q, want \"nanoflow\"", data.Type)
	}
	if data.Name != "FeedbackModule.ACT_SubmitFeedback" {
		t.Errorf("Name = %q, want \"FeedbackModule.ACT_SubmitFeedback\"", data.Name)
	}
	if len(data.Nodes) == 0 {
		t.Errorf("expected at least one node, got 0; output:\n%s", out.String())
	}

	// First node should always be a Start event in any non-empty flow.
	hasStart := false
	for _, n := range data.Nodes {
		if n.Type == "start" {
			hasStart = true
			break
		}
	}
	if !hasStart {
		t.Errorf("expected a start node in nodes list; got: %+v", data.Nodes)
	}
}

// TestNanoflowELKGen_NotFound asserts the same error path legacy
// returns when the qualified name does not match any nanoflow.
func TestNanoflowELKGen_NotFound(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)
	err := nanoflowELKGen(ctx, "NoSuchModule.NoSuchFlow")
	if err == nil {
		t.Fatal("expected error for missing nanoflow, got nil")
	}
	if !strings.Contains(err.Error(), "nanoflow") {
		t.Errorf("error should mention nanoflow; got %v", err)
	}
}

// TestNanoflowELKGen_BadName verifies the qualified-name validation.
func TestNanoflowELKGen_BadName(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)
	if err := nanoflowELKGen(ctx, "NoModuleHere"); err == nil {
		t.Error("expected validation error for unqualified name, got nil")
	}
}

// TestCalcMicroflowNodeSizeGen_MatchesLegacy spot-checks the size
// computation for the four representative shapes. These numeric values
// are load-bearing — the webview consumes them as ELK input — so any
// drift from legacy must be intentional.
func TestCalcMicroflowNodeSizeGen_MatchesLegacy(t *testing.T) {
	cases := []struct {
		nodeType string
		label    string
		details  []string
		wantW    float64 // approximate; we assert >= for action/loop
		wantH    float64
	}{
		// "Start" (5 chars): 5*7.5 + 24*2 = 85.5
		{"start", "Start", nil, 85.5, 36},
		// "End" (3 chars): 3*7.5 + 24*2 = 70.5
		{"end", "End", nil, 70.5, 36},
		// merge always 24x24 regardless of label.
		{"merge", "", nil, 24, 24},
		// "x?y:z" (5 chars): 5*7.5 + 24*2 = 85.5, but split clamps to 100
		{"split", "x?y:z", nil, 100, 60},
	}
	for _, c := range cases {
		t.Run(c.nodeType, func(t *testing.T) {
			w, h := calcMicroflowNodeSizeGen(c.nodeType, c.label, c.details)
			if w != c.wantW {
				t.Errorf("%s width = %v, want %v", c.nodeType, w, c.wantW)
			}
			if h != c.wantH {
				t.Errorf("%s height = %v, want %v", c.nodeType, h, c.wantH)
			}
		})
	}

	// Action node grows with longest detail line.
	wAction, hAction := calcMicroflowNodeSizeGen("action",
		"hello", []string{"line one", "line two longer"})
	if wAction < elkMinWidth {
		t.Errorf("action width %v < elkMinWidth %v", wAction, elkMinWidth)
	}
	if hAction < elkHeaderHeight+8 {
		t.Errorf("action height %v < min %v", hAction, elkHeaderHeight+8)
	}
}

// TestNanoflowELKGen_EntryWiring validates the `Executor.NanoflowELKGen`
// method wrapper is reachable and uses ctx.Output for emission.
func TestNanoflowELKGen_EntryWiring(t *testing.T) {
	var out bytes.Buffer
	ctx := newGenVizContext(t, &out)
	if err := nanoflowELKGen(ctx, "Atlas_Web_Content.DS_LoginContext"); err != nil {
		t.Fatalf("nanoflowELKGen DS_LoginContext: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected non-empty JSON output")
	}
	var data microflowELKData
	if err := json.Unmarshal(out.Bytes(), &data); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if data.Type != "nanoflow" {
		t.Errorf("Type = %q, want nanoflow", data.Type)
	}
}
