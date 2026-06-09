// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.d — error-handler queue + rejoin emission tests.
//
// Verifies the queue/active-state primitives (push/snapshot/restore/
// rewrite) and the merge-splicing rejoin emitters that build
// ExclusiveMerge-mediated handoffs between normal and error flows.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestPendingErrorHandlerStateGenActiveIsEmpty(t *testing.T) {
	cases := []struct {
		name string
		s    pendingErrorHandlerStateGen
		want bool
	}{
		{"all zero", pendingErrorHandlerStateGen{}, true},
		{"emptyFrom set", pendingErrorHandlerStateGen{emptyFrom: "x"}, false},
		{"tailFrom set", pendingErrorHandlerStateGen{tailFrom: "x"}, false},
		{"source set", pendingErrorHandlerStateGen{source: "x"}, false},
		{"skipVar set", pendingErrorHandlerStateGen{skipVar: "x"}, false},
		// tailCase / tailAnchor / tailIsSource / returnValue alone don't make active.
		{"only tailCase", pendingErrorHandlerStateGen{tailCase: "x"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.activeIsEmpty(); got != tc.want {
				t.Fatalf("activeIsEmpty = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestQueueActivePendingErrorHandlerGenSkipsEmpty(t *testing.T) {
	fb := &flowBuilderGen{}
	fb.queueActivePendingErrorHandlerGen()
	if got := len(fb.pendingErrorHandlers); got != 0 {
		t.Fatalf("queue should stay empty for empty active state, got %d", got)
	}
}

func TestQueueActivePendingErrorHandlerGenPushesAndClears(t *testing.T) {
	fb := &flowBuilderGen{
		emptyErrorHandlerFrom: "act-1",
	}
	fb.queueActivePendingErrorHandlerGen()
	if len(fb.pendingErrorHandlers) != 1 {
		t.Fatalf("queue size = %d, want 1", len(fb.pendingErrorHandlers))
	}
	if fb.pendingErrorHandlers[0].emptyFrom != "act-1" {
		t.Fatalf("queued emptyFrom = %s, want act-1", fb.pendingErrorHandlers[0].emptyFrom)
	}
	if fb.emptyErrorHandlerFrom != "" {
		t.Fatalf("active state must clear after push: emptyFrom = %s", fb.emptyErrorHandlerFrom)
	}
}

func TestSetActivePendingErrorHandlerGenRoundTrip(t *testing.T) {
	fb := &flowBuilderGen{}
	state := pendingErrorHandlerStateGen{
		emptyFrom:    "ef",
		tailFrom:     "tf",
		source:       "src",
		skipVar:      "x",
		tailCase:     "true",
		tailIsSource: true,
		returnValue:  "$Result",
	}
	fb.setActivePendingErrorHandlerGen(state)
	got := fb.activePendingErrorHandlerGen()
	if got != state {
		t.Fatalf("round-trip mismatch:\n got: %+v\nwant: %+v", got, state)
	}
}

func TestRewritePendingErrorHandlersGenDropsEmptyResults(t *testing.T) {
	fb := &flowBuilderGen{}
	fb.pendingErrorHandlers = []pendingErrorHandlerStateGen{
		{emptyFrom: "a"},
		{emptyFrom: "b"},
		{emptyFrom: "c"},
	}
	fb.rewritePendingErrorHandlersGen(func(s pendingErrorHandlerStateGen) pendingErrorHandlerStateGen {
		// Drop the middle entry by emptying it.
		if s.emptyFrom == "b" {
			return pendingErrorHandlerStateGen{}
		}
		return s
	})
	if len(fb.pendingErrorHandlers) != 2 {
		t.Fatalf("queue size after rewrite = %d, want 2", len(fb.pendingErrorHandlers))
	}
	if fb.pendingErrorHandlers[0].emptyFrom != "a" || fb.pendingErrorHandlers[1].emptyFrom != "c" {
		t.Fatalf("queue order/content wrong: %+v", fb.pendingErrorHandlers)
	}
}

func TestRegisterEmptyCustomErrorHandlerWithSkipGenNoSkipVar(t *testing.T) {
	fb := &flowBuilderGen{}
	eh := &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom}
	fb.registerEmptyCustomErrorHandlerWithSkipGen("act-1", eh, "")
	if fb.emptyErrorHandlerFrom != "act-1" {
		t.Fatalf("emptyErrorHandlerFrom = %s, want act-1", fb.emptyErrorHandlerFrom)
	}
	if fb.errorHandlerSource != "" {
		t.Fatalf("errorHandlerSource should be empty when skipVar is blank, got %s", fb.errorHandlerSource)
	}
}

func TestRegisterEmptyCustomErrorHandlerWithSkipGenWithSkipVar(t *testing.T) {
	fb := &flowBuilderGen{}
	eh := &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom}
	fb.registerEmptyCustomErrorHandlerWithSkipGen("act-1", eh, "Result")
	if fb.errorHandlerSource != "act-1" {
		t.Fatalf("errorHandlerSource = %s, want act-1", fb.errorHandlerSource)
	}
	if fb.errorHandlerSkipVar != "Result" {
		t.Fatalf("errorHandlerSkipVar = %s, want Result", fb.errorHandlerSkipVar)
	}
	if !fb.errorHandlerTailIsSource {
		t.Fatal("errorHandlerTailIsSource should be true when skipVar set")
	}
}

func TestRegisterEmptyCustomErrorHandlerWithSkipGenIgnoresNonEmpty(t *testing.T) {
	fb := &flowBuilderGen{}
	eh := &ast.ErrorHandlingClause{
		Type: ast.ErrorHandlingCustom,
		Body: []ast.MicroflowStatement{&ast.ReturnStmt{}},
	}
	fb.registerEmptyCustomErrorHandlerWithSkipGen("act-1", eh, "")
	if fb.emptyErrorHandlerFrom != "" {
		t.Fatalf("non-empty handler must not register: emptyFrom = %s", fb.emptyErrorHandlerFrom)
	}
}

func TestRegisterEmptyCustomErrorHandlerWithSkipGenQueuesPriorActive(t *testing.T) {
	fb := &flowBuilderGen{
		emptyErrorHandlerFrom: "prior",
	}
	eh := &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom}
	fb.registerEmptyCustomErrorHandlerWithSkipGen("new", eh, "")
	if len(fb.pendingErrorHandlers) != 1 {
		t.Fatalf("expected prior state to be queued, got queue size %d", len(fb.pendingErrorHandlers))
	}
	if fb.pendingErrorHandlers[0].emptyFrom != "prior" {
		t.Fatalf("queued state wrong: %+v", fb.pendingErrorHandlers[0])
	}
	if fb.emptyErrorHandlerFrom != "new" {
		t.Fatalf("active emptyFrom = %s, want new", fb.emptyErrorHandlerFrom)
	}
}

func TestApplyDeferredFlowCaseGenAttachesEnumerationCase(t *testing.T) {
	flow := newHorizontalFlowGen("o", "d")
	applyDeferredFlowCaseGen(flow, "RED", nil)
	cv, ok := flow.CaseValue().(*genMf.EnumerationCase)
	if !ok {
		t.Fatalf("CaseValue type = %T, want *EnumerationCase", flow.CaseValue())
	}
	if cv.Value() != "RED" {
		t.Fatalf("value = %q, want RED", cv.Value())
	}
}

func TestApplyDeferredFlowCaseGenEmptyValueLeavesCaseUntouched(t *testing.T) {
	flow := newHorizontalFlowGen("o", "d")
	applyDeferredFlowCaseGen(flow, "", nil)
	if flow.CaseValue() != nil {
		t.Fatalf("empty caseValue should leave CaseValue nil, got %T", flow.CaseValue())
	}
}

func TestApplyDeferredFlowCaseGenAppliesAnchorsBothSides(t *testing.T) {
	flow := newHorizontalFlowGen("o", "d")
	anchor := &ast.FlowAnchors{
		From: ast.AnchorSide(AnchorTop),
		To:   ast.AnchorSide(AnchorBottom),
	}
	applyDeferredFlowCaseGen(flow, "", anchor)
	if flow.OriginConnectionIndex() != int32(AnchorTop) {
		t.Fatalf("origin anchor not applied: %d", flow.OriginConnectionIndex())
	}
	if flow.DestinationConnectionIndex() != int32(AnchorBottom) {
		t.Fatalf("dest anchor not applied: %d", flow.DestinationConnectionIndex())
	}
}

func TestApplyDeferredFlowCaseGenNilFlowSafe(t *testing.T) {
	applyDeferredFlowCaseGen(nil, "X", nil)
}

func TestIsExclusiveMergeGenDetectsByID(t *testing.T) {
	fb := &flowBuilderGen{}
	merge := genMf.NewExclusiveMerge()
	mergeID := assignFreshID(merge)
	other := genMf.NewActionActivity()
	otherID := assignFreshID(other)
	fb.objects = []element.Element{merge, other}

	if !fb.isExclusiveMergeGen(mergeID) {
		t.Fatal("merge ID should match")
	}
	if fb.isExclusiveMergeGen(otherID) {
		t.Fatal("non-merge ID must not match")
	}
	if fb.isExclusiveMergeGen("nonexistent") {
		t.Fatal("missing ID must not match")
	}
}

func TestFindExistingRejoinMergeGenLocatesPattern(t *testing.T) {
	fb := &flowBuilderGen{}
	merge := genMf.NewExclusiveMerge()
	mergeID := assignFreshID(merge)
	fb.objects = []element.Element{merge}

	// Pattern: o → merge → d (both edges normal).
	fb.flows = append(fb.flows, newHorizontalFlowGen("o", mergeID))
	fb.flows = append(fb.flows, newHorizontalFlowGen(mergeID, "d"))

	if got := fb.findExistingRejoinMergeGen("o", "d"); got != mergeID {
		t.Fatalf("findExistingRejoinMergeGen = %s, want %s", got, mergeID)
	}
	// Wrong destination — no match.
	if got := fb.findExistingRejoinMergeGen("o", "other"); got != "" {
		t.Fatalf("expected empty for unmatched dest, got %s", got)
	}
}

func TestFindExistingRejoinMergeGenSkipsErrorHandlerEdges(t *testing.T) {
	fb := &flowBuilderGen{}
	merge := genMf.NewExclusiveMerge()
	mergeID := assignFreshID(merge)
	fb.objects = []element.Element{merge}

	// Origin edge is flagged as error handler — must not be considered.
	fb.flows = append(fb.flows, newErrorHandlerFlowGen("o", mergeID))
	fb.flows = append(fb.flows, newHorizontalFlowGen(mergeID, "d"))

	if got := fb.findExistingRejoinMergeGen("o", "d"); got != "" {
		t.Fatalf("error-handler origin edge must not be picked up, got %s", got)
	}
}

func TestAddEmptyErrorHandlerRejoinFlowFromGenNoExistingDirectFlow(t *testing.T) {
	// No prior horizontal flow — rejoin should be a single error-handler edge.
	fb := &flowBuilderGen{posX: 200, baseY: 200}
	fb.addEmptyErrorHandlerRejoinFlowFromGen("normal-origin", "error-origin", "dest")

	if len(fb.flows) != 1 {
		t.Fatalf("flows = %d, want 1", len(fb.flows))
	}
	flow := fb.flows[0]
	if !flow.IsErrorHandler() {
		t.Fatal("rejoin must be flagged IsErrorHandler")
	}
	if flow.OriginRefID() != element.ID("error-origin") {
		t.Fatalf("origin = %s, want error-origin", flow.OriginRefID())
	}
	if flow.DestinationRefID() != element.ID("dest") {
		t.Fatalf("dest = %s, want dest", flow.DestinationRefID())
	}
	if len(fb.objects) != 0 {
		t.Fatalf("no merge should be created, got %d objects", len(fb.objects))
	}
}

func TestAddEmptyErrorHandlerRejoinFlowFromGenSplicesMergeWhenFlowExists(t *testing.T) {
	fb := &flowBuilderGen{posX: 400, baseY: 200, spacing: HorizontalSpacing}
	// Pre-existing normal flow that we'll splice through.
	existing := newHorizontalFlowGen("normal-origin", "dest")
	fb.flows = append(fb.flows, existing)

	fb.addEmptyErrorHandlerRejoinFlowFromGen("normal-origin", "error-origin", "dest")

	// Expect: the original flow is removed; merge object created;
	// 3 new flows added (normal → merge, error → merge, merge → dest).
	if len(fb.objects) != 1 {
		t.Fatalf("objects = %d, want 1 (the merge)", len(fb.objects))
	}
	if _, ok := fb.objects[0].(*genMf.ExclusiveMerge); !ok {
		t.Fatalf("inserted object should be *ExclusiveMerge, got %T", fb.objects[0])
	}
	if len(fb.flows) != 3 {
		t.Fatalf("flows = %d, want 3 (normal→merge, error→merge, merge→dest)", len(fb.flows))
	}
	mergeID := fb.objects[0].ID()
	// The remaining flows should be: 0=normal→merge, 1=error→merge, 2=merge→dest.
	// (The original existing flow has been removed.)
	if fb.flows[0].DestinationRefID() != mergeID || fb.flows[0].OriginRefID() != "normal-origin" {
		t.Fatalf("first new flow wrong: %s → %s", fb.flows[0].OriginRefID(), fb.flows[0].DestinationRefID())
	}
	if !fb.flows[1].IsErrorHandler() || fb.flows[1].OriginRefID() != "error-origin" || fb.flows[1].DestinationRefID() != mergeID {
		t.Fatalf("error edge wrong: handler=%t %s → %s", fb.flows[1].IsErrorHandler(), fb.flows[1].OriginRefID(), fb.flows[1].DestinationRefID())
	}
	if fb.flows[2].OriginRefID() != mergeID || fb.flows[2].DestinationRefID() != "dest" {
		t.Fatalf("merge→dest wrong: %s → %s", fb.flows[2].OriginRefID(), fb.flows[2].DestinationRefID())
	}
}

func TestAddPendingErrorHandlerFlowToGenIsNoOpForEmptyDestination(t *testing.T) {
	fb := &flowBuilderGen{}
	fb.pendingErrorHandlers = []pendingErrorHandlerStateGen{{emptyFrom: "x"}}
	fb.addPendingErrorHandlerFlowToGen("")
	if len(fb.pendingErrorHandlers) != 1 {
		t.Fatalf("queue should be untouched for empty dest, got size %d", len(fb.pendingErrorHandlers))
	}
}

func TestAddPendingErrorHandlerFlowToGenDrainsEmptyHandler(t *testing.T) {
	fb := &flowBuilderGen{}
	fb.pendingErrorHandlers = []pendingErrorHandlerStateGen{{emptyFrom: "errsource"}}

	fb.addPendingErrorHandlerFlowToGen("dest")

	if len(fb.pendingErrorHandlers) != 0 {
		t.Fatalf("queue should be drained, got size %d", len(fb.pendingErrorHandlers))
	}
	if len(fb.flows) != 1 {
		t.Fatalf("flows = %d, want 1", len(fb.flows))
	}
	if !fb.flows[0].IsErrorHandler() {
		t.Fatal("drained edge must be IsErrorHandler")
	}
}
