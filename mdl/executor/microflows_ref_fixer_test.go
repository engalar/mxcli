// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// newFixerCtx 创建一个带 mock Microflows repo 的 ExecContext。
func newFixerCtx(t *testing.T, mfs []*genMf.Microflow, updateCalled *bool) *ExecContext {
	t.Helper()
	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc:          func() ([]*genMf.Microflow, error) { return mfs, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return "", nil },
		UpdateFunc: func(mf *genMf.Microflow) error {
			if updateCalled != nil {
				*updateCalled = true
			}
			return nil
		},
	}
	ctx := &ExecContext{
		Microflows: repo,
	}
	return ctx
}

func TestMFCallerRefFixer_RemoveStaleMappings_RemovesBrokenEntry(t *testing.T) {
	action := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam",  // 要删除的
		"Common_Utils.GET_Message_ById.MessageId", // 要保留的
	)
	caller := makeMFWithActions("M.Caller", action)

	updateCalled := false
	ctx := newFixerCtx(t, []*genMf.Microflow{caller}, &updateCalled)
	fixer := NewMFCallerRefFixer(ctx)

	report, err := fixer.RemoveStaleMappings("Common_Utils.GET_Message_ById", []string{"OldParam"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.FixedCount() != 1 {
		t.Errorf("FixedCount = %d, want 1", report.FixedCount())
	}
	if !updateCalled {
		t.Error("ctx.Microflows.Update should have been called")
	}
	if report.Fixed[0].OldParam != "Common_Utils.GET_Message_ById.OldParam" {
		t.Errorf("OldParam = %q", report.Fixed[0].OldParam)
	}
	if report.Fixed[0].NewParam != "" {
		t.Errorf("NewParam should be empty for removal, got %q", report.Fixed[0].NewParam)
	}

	// 验证 gen 对象中 mapping 已被删除（只剩 MessageId）
	call := action.MicroflowCall().(*genMf.MicroflowCall)
	if len(call.ParameterMappingsItems()) != 1 {
		t.Errorf("remaining mappings = %d, want 1", len(call.ParameterMappingsItems()))
	}
	remain := call.ParameterMappingsItems()[0].(*genMf.MicroflowCallParameterMapping)
	if remain.ParameterQualifiedName() != "Common_Utils.GET_Message_ById.MessageId" {
		t.Errorf("remaining param = %q", remain.ParameterQualifiedName())
	}
}

func TestMFCallerRefFixer_RemapParam_UpdatesQualifiedName(t *testing.T) {
	action := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam",
	)
	caller := makeMFWithActions("M.Caller", action)

	ctx := newFixerCtx(t, []*genMf.Microflow{caller}, nil)
	fixer := NewMFCallerRefFixer(ctx)

	report, err := fixer.RemapParam("Common_Utils.GET_Message_ById", "OldParam", "MessageId")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.FixedCount() != 1 {
		t.Errorf("FixedCount = %d, want 1", report.FixedCount())
	}
	if report.Fixed[0].NewParam != "MessageId" {
		t.Errorf("NewParam = %q, want MessageId", report.Fixed[0].NewParam)
	}

	call := action.MicroflowCall().(*genMf.MicroflowCall)
	pm := call.ParameterMappingsItems()[0].(*genMf.MicroflowCallParameterMapping)
	if pm.ParameterQualifiedName() != "Common_Utils.GET_Message_ById.MessageId" {
		t.Errorf("after remap, QN = %q", pm.ParameterQualifiedName())
	}
}

func TestMFCallerRefFixer_RemoveStaleMappings_NoOp_WhenNoBroken(t *testing.T) {
	action := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.MessageId", // 有效，不是 stale
	)
	caller := makeMFWithActions("M.Caller", action)

	updateCalled := false
	ctx := newFixerCtx(t, []*genMf.Microflow{caller}, &updateCalled)
	fixer := NewMFCallerRefFixer(ctx)

	report, err := fixer.RemoveStaleMappings("Common_Utils.GET_Message_ById", []string{"OldParam"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.FixedCount() != 0 {
		t.Errorf("FixedCount = %d, want 0", report.FixedCount())
	}
	if updateCalled {
		t.Error("Update should NOT have been called when nothing to fix")
	}
}

func TestMFCallerRefFixer_RemoveStaleMappings_EmptyParams_ReturnsEmpty(t *testing.T) {
	caller := makeMFWithActions("M.Caller")
	ctx := newFixerCtx(t, []*genMf.Microflow{caller}, nil)
	fixer := NewMFCallerRefFixer(ctx)

	report, err := fixer.RemoveStaleMappings("Common_Utils.GET_Message_ById", nil)
	if err != nil || report.FixedCount() != 0 {
		t.Errorf("expected empty report, got err=%v fixed=%d", err, report.FixedCount())
	}
}

func TestMFCallerRefFixer_MultipleCaller_FixesAll(t *testing.T) {
	action1 := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam",
	)
	action2 := makeCallAction(
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam",
	)
	caller1 := makeMFWithActions("M.Caller1", action1)
	caller2 := makeMFWithActions("M.Caller2", action2)

	updateCount := 0
	repo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) {
			return []*genMf.Microflow{caller1, caller2}, nil
		},
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return "", nil },
		UpdateFunc:           func(_ *genMf.Microflow) error { updateCount++; return nil },
	}
	ctx := &ExecContext{
		Microflows: repo,
	}
	fixer := NewMFCallerRefFixer(ctx)

	report, err := fixer.RemoveStaleMappings("Common_Utils.GET_Message_ById", []string{"OldParam"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.FixedCount() != 2 {
		t.Errorf("FixedCount = %d, want 2", report.FixedCount())
	}
	if updateCount != 2 {
		t.Errorf("Update called %d times, want 2", updateCount)
	}
}
