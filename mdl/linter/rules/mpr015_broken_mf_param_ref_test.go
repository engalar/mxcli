// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// mpr015Reader implements linter.LintReader for MPR015 tests.
// Embeds the interface so that unused methods panic rather than silently return zero values.
type mpr015Reader struct {
	linter.LintReader // nil-panic default for methods we don't configure
	mfs               map[model.ID]*genMf.Microflow
}

func (r *mpr015Reader) GetMicroflowGen(id model.ID) (*genMf.Microflow, error) {
	return r.mfs[id], nil
}

// makeMFWithParam builds a Microflow that has one parameter named paramName.
// If targetQN and paramRef are non-empty, it also adds a MicroflowCallAction
// that calls targetQN with a single ParameterMapping referencing paramRef.
func makeMFWithParam(mfName, paramName, targetQN, paramRef string) *genMf.Microflow {
	mf := genMf.NewMicroflow()
	mf.SetName(mfName)

	oc := genMf.NewMicroflowObjectCollection()

	// Add a parameter
	param := genMf.NewMicroflowParameter()
	param.SetName(paramName)
	oc.AddObjects(param)

	// Optionally add a call action with a parameter mapping
	if targetQN != "" {
		callAction := genMf.NewMicroflowCallAction()

		call := genMf.NewMicroflowCall()
		call.SetMicroflowQualifiedName(targetQN)

		if paramRef != "" {
			mapping := genMf.NewMicroflowCallParameterMapping()
			mapping.SetParameterQualifiedName(paramRef)
			mapping.SetArgument("$x")
			call.AddParameterMappings(mapping)
		}

		callAction.SetMicroflowCall(call)
		oc.AddObjects(callAction)
	}

	mf.SetObjectCollection(oc)
	return mf
}

// mpr015Graph builds a mock graph whose Microflows listing mirrors the given
// rows (id / qualified name / module), replacing the former SQLite catalog.
func mpr015Graph(rows []struct{ id, qn, mod string }) *mock.MockProjectGraph {
	var nodes []graphcatalog.MicroflowNode
	for _, r := range rows {
		name := r.qn
		if parts := strings.SplitN(r.qn, ".", 2); len(parts) == 2 {
			name = parts[1]
		}
		nodes = append(nodes, graphcatalog.MicroflowNode{
			ID: r.id, Name: name, QualifiedName: r.qn, Module: r.mod,
		})
	}
	return &mock.MockProjectGraph{
		MicroflowsFunc: func(module string) []graphcatalog.MicroflowNode {
			if module == "" {
				return nodes
			}
			var out []graphcatalog.MicroflowNode
			for _, n := range nodes {
				if n.Module == module {
					out = append(out, n)
				}
			}
			return out
		},
	}
}

// makeID creates a deterministic element.ID from a string (good enough for tests).
func makeID(s string) element.ID { return element.ID(s) }

// TestMPR015_FlagsBrokenParameterRef verifies that a caller referencing a
// non-existent parameter ("OldParam") of a target microflow is flagged.
func TestMPR015_FlagsBrokenParameterRef(t *testing.T) {
	const targetID = "target-001"
	const callerID = "caller-001"

	// Target has param "MessageId"; caller erroneously references "OldParam".
	targetMF := makeMFWithParam("GET_Message_ById", "MessageId", "", "")
	targetMF.SetID(makeID(targetID))

	callerMF := makeMFWithParam(
		"Caller_MF", "SomeParam",
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam", // broken — param doesn't exist
	)
	callerMF.SetID(makeID(callerID))

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(targetID): targetMF,
		model.ID(callerID): callerMF,
	}}
	ctx := linter.NewLintContext(mpr015Graph([]struct{ id, qn, mod string }{
		{targetID, "Common_Utils.GET_Message_ById", "Common_Utils"},
		{callerID, "ContractorReg.Caller_MF", "ContractorReg"},
	}), reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	v := violations[0]
	if v.RuleID != "MPR015" {
		t.Errorf("RuleID = %q, want MPR015", v.RuleID)
	}
	if !strings.Contains(v.Message, "OldParam") {
		t.Errorf("message should mention OldParam: %q", v.Message)
	}
	if v.Location.Module != "ContractorReg" {
		t.Errorf("Location.Module = %q, want ContractorReg", v.Location.Module)
	}
	if v.Location.DocumentName != "Caller_MF" {
		t.Errorf("Location.DocumentName = %q, want Caller_MF", v.Location.DocumentName)
	}
}

// TestMPR015_NoViolationWhenParamExists verifies no violation when the
// referenced parameter actually exists.
func TestMPR015_NoViolationWhenParamExists(t *testing.T) {
	const targetID = "target-002"
	const callerID = "caller-002"

	targetMF := makeMFWithParam("GET_Message_ById", "MessageId", "", "")
	targetMF.SetID(makeID(targetID))

	callerMF := makeMFWithParam(
		"Caller_OK", "SomeParam",
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.MessageId", // valid
	)
	callerMF.SetID(makeID(callerID))

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(targetID): targetMF,
		model.ID(callerID): callerMF,
	}}
	ctx := linter.NewLintContext(mpr015Graph([]struct{ id, qn, mod string }{
		{targetID, "Common_Utils.GET_Message_ById", "Common_Utils"},
		{callerID, "Common_Utils.Caller_OK", "Common_Utils"},
	}), reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d: %v", len(violations), violations)
	}
}

// TestMPR015_UnknownTargetSkipped verifies that calls to microflows not in the
// catalog (e.g. marketplace modules) are silently skipped.
func TestMPR015_UnknownTargetSkipped(t *testing.T) {
	const callerID = "caller-003"

	callerMF := makeMFWithParam(
		"Caller_Ext", "SomeParam",
		"External.SomeMF",
		"External.SomeMF.Gone",
	)
	callerMF.SetID(makeID(callerID))

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(callerID): callerMF,
	}}
	// Only the caller is in the catalog; External.SomeMF is absent.
	ctx := linter.NewLintContext(mpr015Graph([]struct{ id, qn, mod string }{
		{callerID, "MyModule.Caller_Ext", "MyModule"},
	}), reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations for unknown target, got %d: %v", len(violations), violations)
	}
}

// TestMPR015_BrokenRefInsideLoop verifies detection inside a LoopedActivity body.
func TestMPR015_BrokenRefInsideLoop(t *testing.T) {
	const targetID = "target-004"
	const callerID = "caller-004"

	targetMF := makeMFWithParam("SUB_Process", "ItemId", "", "")
	targetMF.SetID(makeID(targetID))

	// Build caller with broken call inside a loop body
	callerMF := genMf.NewMicroflow()
	callerMF.SetName("Caller_Loop")
	callerMF.SetID(makeID(callerID))

	outerOC := genMf.NewMicroflowObjectCollection()
	loopBody := genMf.NewMicroflowObjectCollection()

	callAction := genMf.NewMicroflowCallAction()
	call := genMf.NewMicroflowCall()
	call.SetMicroflowQualifiedName("Common.SUB_Process")
	mapping := genMf.NewMicroflowCallParameterMapping()
	mapping.SetParameterQualifiedName("Common.SUB_Process.DeadParam") // broken
	mapping.SetArgument("$item")
	call.AddParameterMappings(mapping)
	callAction.SetMicroflowCall(call)
	loopBody.AddObjects(callAction)

	loop := genMf.NewLoopedActivity()
	loop.SetObjectCollection(loopBody)
	outerOC.AddObjects(loop)
	callerMF.SetObjectCollection(outerOC)

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(targetID): targetMF,
		model.ID(callerID): callerMF,
	}}
	ctx := linter.NewLintContext(mpr015Graph([]struct{ id, qn, mod string }{
		{targetID, "Common.SUB_Process", "Common"},
		{callerID, "MyModule.Caller_Loop", "MyModule"},
	}), reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation inside loop, got %d: %v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "DeadParam") {
		t.Errorf("message should mention DeadParam: %q", violations[0].Message)
	}
}

// TestMPR015_ReaderNil_ReturnsEmpty verifies graceful no-op when reader is absent.
func TestMPR015_ReaderNil_ReturnsEmpty(t *testing.T) {
	ctx := linter.NewLintContext(nil, nil)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations without reader, got %d", len(violations))
	}
}

// TestMPR015_Metadata verifies rule metadata fields.
func TestMPR015_Metadata(t *testing.T) {
	r := NewBrokenMFParamRefRule()
	if r.ID() != "MPR015" {
		t.Errorf("ID = %q, want MPR015", r.ID())
	}
	if r.Category() != "MPR" {
		t.Errorf("Category = %q, want MPR", r.Category())
	}
	if r.DefaultSeverity() != linter.SeverityError {
		t.Errorf("DefaultSeverity = %v, want Error", r.DefaultSeverity())
	}
}
