// SPDX-License-Identifier: Apache-2.0

//go:build integration

// Integration tests simulating the PayerBilling2 CE1613 scenario:
// Common_Utils.GET_Message_ById was redefined without $Locale,
// leaving 28+ callers with broken MicroflowCallParameterMapping refs.

package executor

import (
	"bytes"
	"strings"
	"testing"

	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ---------------------------------------------------------------------------
// Scenario 1: 28-caller GET_Message_ById breakage
// ---------------------------------------------------------------------------

func TestScanBrokenCallerRefs_SimulatesPayerBilling2GetMessageBreakage(t *testing.T) {
	// PayerBilling2 real scenario:
	// GET_Message_ById was redefined from ($MessageId, $Locale) to ($MessageId).
	// 28 callers still reference "Common_Utils.GET_Message_ById.Locale".
	// We simulate 3 representative callers (stands in for 28).

	const targetQN = "Common_Utils.GET_Message_ById"
	const brokenParam = "Common_Utils.GET_Message_ById.Locale"

	callerNames := []string{
		"ContractorRegistration.SUB_SalesAreaDataDto_OptimizeInput",
		"EndCustomerRegistration.GET_EndCustomer_Get2",
		"ShipmentRegistration.ACT_ShipmentDestination_Save",
	}

	var callers []*genMf.Microflow
	names := make(map[*genMf.Microflow]string)
	for _, name := range callerNames {
		action := makeCallAction(targetQN, brokenParam)
		mf := makeMFWithActions(name, action)
		callers = append(callers, mf)
		names[mf] = name
	}

	refs := scanBrokenCallerRefs(callers, names, targetQN, []string{"Locale"})

	if len(refs) != len(callerNames) {
		t.Fatalf("expected %d broken refs (simulating 28 real callers), got %d",
			len(callerNames), len(refs))
	}
	for i, r := range refs {
		if r.TargetMF != targetQN {
			t.Errorf("[%d] TargetMF = %q, want %q", i, r.TargetMF, targetQN)
		}
		if r.BrokenParam != brokenParam {
			t.Errorf("[%d] BrokenParam = %q, want %q", i, r.BrokenParam, brokenParam)
		}
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: warnBrokenCallerRefs output format
// ---------------------------------------------------------------------------

func TestWarnBrokenCallerRefs_OutputFormat(t *testing.T) {
	const targetQN = "Common_Utils.GET_Message_ById"
	action := makeCallAction(targetQN, targetQN+".OldParam")
	mf := makeMFWithActions("M.Caller", action)

	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mf}, nil
		},
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) {
			return "", nil
		},
	}

	var buf bytes.Buffer
	ctx := &ExecContext{
		ExecRepos: ExecRepos{Microflows: repo},
		ExecIO:    ExecIO{Output: &buf},
	}

	warnBrokenCallerRefs(ctx, targetQN, []string{"OldParam"})

	output := buf.String()
	checks := []struct {
		desc    string
		keyword string
	}{
		{"CE1613 keyword", "CE1613"},
		{"caller name", "M.Caller"},
		{"broken param name", "OldParam"},
		{"fix hint", "Fix"},
	}
	for _, c := range checks {
		if !strings.Contains(output, c.keyword) {
			t.Errorf("output should contain %s (%q), got:\n%s", c.desc, c.keyword, output)
		}
	}
}

// ---------------------------------------------------------------------------
// Scenario 3: MFCallerRefFixer end-to-end removes broken mappings
// ---------------------------------------------------------------------------

func TestMFCallerRefFixer_EndToEnd_RemovesAndUpdates(t *testing.T) {
	const targetQN = "Common_Utils.GET_Message_ById"

	// Two callers, each with one broken mapping.
	action1 := makeCallAction(targetQN, targetQN+".OldParam")
	action2 := makeCallAction(targetQN, targetQN+".OldParam")
	mf1 := makeMFWithActions("M.Caller1", action1)
	mf2 := makeMFWithActions("M.Caller2", action2)

	updateCount := 0
	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{mf1, mf2}, nil
		},
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return "", nil },
		UpdateFunc:           func(_ *genMf.Microflow) error { updateCount++; return nil },
	}

	ctx := &ExecContext{
		ExecRepos: ExecRepos{Microflows: repo},
	}
	fixer := NewMFCallerRefFixer(ctx)

	report, err := fixer.RemoveStaleMappings(targetQN, []string{"OldParam"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.FixedCount() != 2 {
		t.Errorf("FixedCount = %d, want 2", report.FixedCount())
	}
	if updateCount != 2 {
		t.Errorf("Update called %d times, want 2", updateCount)
	}

	// Verify gen objects are patched: no remaining mappings for the broken param.
	for i, action := range []*genMf.MicroflowCallAction{action1, action2} {
		call := action.MicroflowCall().(*genMf.MicroflowCall)
		if len(call.ParameterMappingsItems()) != 0 {
			t.Errorf("caller %d: expected 0 remaining mappings, got %d", i+1,
				len(call.ParameterMappingsItems()))
		}
	}
}
