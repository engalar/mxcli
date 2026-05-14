// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

const (
	// fixtureJavaActionCount counts MPR-stored Java actions only. The legacy
	// `show java actions` command additionally synthesizes System.VerifyPassword
	// from sdk/mpr.BuildSystemJavaActions, so user-visible counts are off by
	// one — that synthesis is wired in the Phase A read formatter, not at
	// the repo layer.
	fixtureJavaActionCount       = 2
	fixtureJavaScriptActionCount = 16
)

func TestJavaActionRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != fixtureJavaActionCount {
		t.Errorf("ListAll: got %d java actions, want %d", len(got), fixtureJavaActionCount)
	}
}

func TestJavaActionRepo_GetRoundTrip_DecodesGenType(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType(javaActionTypeName)
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("fixture has no java actions")
	}

	repo := NewJavaActionRepository(w)
	first, err := repo.Get(model.ID(refs[0].ID))
	if err != nil {
		t.Fatalf("Get(%s): %v", refs[0].ID, err)
	}
	if first.ID() == "" {
		t.Error("decoded java action has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded java action has empty Name")
	}
	if first.TypeName() != javaActionTypeName {
		t.Errorf("decoded java action TypeName = %q, want %q", first.TypeName(), javaActionTypeName)
	}
}

func TestJavaActionRepo_FindByQualifiedName(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	ja, err := repo.FindByQualifiedName("FeedbackModule.ValidateEmail")
	if err != nil {
		t.Fatalf("FindByQualifiedName: %v", err)
	}
	if ja == nil {
		t.Fatal("FindByQualifiedName returned nil for known fixture entry")
	}
	if ja.Name() != "ValidateEmail" {
		t.Errorf("Name = %q, want ValidateEmail", ja.Name())
	}
}

func TestJavaActionRepo_GetContainerUUID_NonEmpty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("fixture has no java actions")
	}
	cid, err := repo.GetContainerUUID(model.ID(all[0].ID()))
	if err != nil {
		t.Fatalf("GetContainerUUID: %v", err)
	}
	if cid == "" {
		t.Error("GetContainerUUID returned empty container UUID")
	}
}

func TestJavaActionRepo_WriterStubs_ReturnNotImplemented(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaActionRepository(w)
	if err := repo.Create("", "", nil); err == nil {
		t.Error("Create: expected not-implemented error, got nil")
	}
	if err := repo.Update(nil); err == nil {
		t.Error("Update: expected not-implemented error, got nil")
	}
	if err := repo.Delete("dummy"); err == nil {
		t.Error("Delete: expected not-implemented error, got nil")
	}
}

func TestJavaScriptActionRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewJavaScriptActionRepository(w)
	got, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) < fixtureJavaScriptActionCount {
		t.Errorf("ListAll: got %d javascript actions, want >= %d", len(got), fixtureJavaScriptActionCount)
	}
}

func TestJavaScriptActionRepo_GetRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	refs, err := r.ListUnitsByType(javaScriptActionTypeName)
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("fixture has no javascript actions")
	}
	repo := NewJavaScriptActionRepository(w)
	first, err := repo.Get(model.ID(refs[0].ID))
	if err != nil {
		t.Fatalf("Get(%s): %v", refs[0].ID, err)
	}
	if first.ID() == "" {
		t.Error("decoded javascript action has empty ID")
	}
	if first.Name() == "" {
		t.Error("decoded javascript action has empty Name")
	}
}
