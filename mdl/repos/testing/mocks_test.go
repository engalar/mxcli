// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

func TestRecordingMicroflowRepository_RecordsCreate(t *testing.T) {
	rec := &RecordingMicroflowRepository{}
	mf := genMf.NewMicroflow()
	if err := rec.Create("parent-uuid", "Documents", mf); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(rec.Created) != 1 {
		t.Fatalf("Created length = %d, want 1", len(rec.Created))
	}
	got := rec.Created[0]
	if got.ParentUUID != "parent-uuid" {
		t.Errorf("ParentUUID = %q, want parent-uuid", got.ParentUUID)
	}
	if got.ContainmentName != "Documents" {
		t.Errorf("ContainmentName = %q, want Documents", got.ContainmentName)
	}
	if got.Microflow != mf {
		t.Errorf("Microflow != input pointer")
	}
}

func TestRecordingMicroflowRepository_FuncOverride(t *testing.T) {
	wantErr := errors.New("inject")
	rec := &RecordingMicroflowRepository{
		CreateFunc: func(MicroflowCreateCall) error { return wantErr },
	}
	err := rec.Create("p", "c", genMf.NewMicroflow())
	if !errors.Is(err, wantErr) {
		t.Errorf("Create err = %v, want %v", err, wantErr)
	}
	if len(rec.Created) != 1 {
		t.Errorf("call still recorded? Created length = %d, want 1", len(rec.Created))
	}
}

func TestRecordingMicroflowRepository_RecordsAllReadCalls(t *testing.T) {
	rec := &RecordingMicroflowRepository{}
	_, _ = rec.Get("id-1")
	_, _ = rec.Get("id-2")
	_, _ = rec.List("mod-1")
	_, _ = rec.ListAll()
	_, _ = rec.ListAll()
	_, _ = rec.FindByQualifiedName("M.Foo")
	_, _ = rec.IsRule("M.Bar")

	if len(rec.GotIDs) != 2 {
		t.Errorf("GotIDs = %d, want 2", len(rec.GotIDs))
	}
	if len(rec.ListedModule) != 1 || rec.ListedModule[0] != model.ID("mod-1") {
		t.Errorf("ListedModule = %v", rec.ListedModule)
	}
	if rec.ListedAll != 2 {
		t.Errorf("ListedAll = %d, want 2", rec.ListedAll)
	}
	if len(rec.FoundQNs) != 1 || rec.FoundQNs[0] != "M.Foo" {
		t.Errorf("FoundQNs = %v", rec.FoundQNs)
	}
	if len(rec.IsRuleQNs) != 1 || rec.IsRuleQNs[0] != "M.Bar" {
		t.Errorf("IsRuleQNs = %v", rec.IsRuleQNs)
	}
}

func TestRecordingPageRepository_OpenForMutation_DefaultMutator(t *testing.T) {
	rec := &RecordingPageRepository{}
	mut, err := rec.OpenForMutation("page-1")
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	if mut == nil {
		t.Fatal("OpenForMutation returned nil mutator")
	}
	if _, ok := mut.(*RecordingPageMutator); !ok {
		t.Errorf("default mutator type = %T, want *RecordingPageMutator", mut)
	}
	if len(rec.OpenedIDs) != 1 || rec.OpenedIDs[0] != model.ID("page-1") {
		t.Errorf("OpenedIDs = %v", rec.OpenedIDs)
	}
}

func TestRecordingPageRepository_OpenForMutation_InjectedMutator(t *testing.T) {
	injected := &RecordingPageMutator{}
	rec := &RecordingPageRepository{
		OpenForMutationFunc: func(model.ID) (repos.PageMutator, error) {
			return injected, nil
		},
	}
	mut, err := rec.OpenForMutation("page-2")
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	if mut != injected {
		t.Errorf("OpenForMutation returned %T, want injected pointer", mut)
	}

	// Drive the injected mutator and verify it records.
	if err := mut.SetWidgetProperty("widget-1", "Caption", "hi"); err != nil {
		t.Fatalf("SetWidgetProperty: %v", err)
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(injected.SetWidgetPropertyCalls) != 1 {
		t.Fatalf("SetWidgetPropertyCalls = %d, want 1", len(injected.SetWidgetPropertyCalls))
	}
	got := injected.SetWidgetPropertyCalls[0]
	if got.WidgetID != "widget-1" || got.Prop != "Caption" || got.Value != "hi" {
		t.Errorf("captured call = %+v", got)
	}
	if injected.CommitCalls != 1 {
		t.Errorf("CommitCalls = %d, want 1", injected.CommitCalls)
	}
}

func TestRecordingPageRepository_RecordsCreateAndUpdate(t *testing.T) {
	rec := &RecordingPageRepository{}
	page := genPg.NewPage()
	if err := rec.Create("p-uuid", "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := rec.Update(page); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(rec.Created) != 1 || rec.Created[0].ParentUUID != "p-uuid" {
		t.Errorf("Created = %+v", rec.Created)
	}
	if len(rec.Updated) != 1 || rec.Updated[0] != page {
		t.Errorf("Updated = %v (want single pointer to input page)", rec.Updated)
	}
}

func TestRecordingPageMutator_AllMethodsRecord(t *testing.T) {
	m := &RecordingPageMutator{}
	_ = m.SetWidgetProperty("w", "P", 7)
	_ = m.InsertWidget("p", "slot", nil)
	_ = m.DeleteWidget("d")
	_ = m.ReplaceWidget("r", nil)
	_ = m.SetLayout("Layouts.X")
	_ = m.Commit()
	_ = m.Commit()

	if len(m.SetWidgetPropertyCalls) != 1 ||
		len(m.InsertWidgetCalls) != 1 ||
		len(m.DeleteWidgetCalls) != 1 ||
		len(m.ReplaceWidgetCalls) != 1 ||
		len(m.SetLayoutCalls) != 1 ||
		m.CommitCalls != 2 {
		t.Errorf("counts: setProp=%d insert=%d delete=%d replace=%d setLayout=%d commit=%d",
			len(m.SetWidgetPropertyCalls), len(m.InsertWidgetCalls),
			len(m.DeleteWidgetCalls), len(m.ReplaceWidgetCalls),
			len(m.SetLayoutCalls), m.CommitCalls)
	}
}
