// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// Stage 2 fixture (testdata/expr-checker/minimal.mpr) actually has
// 16 Forms$Page units (plus 46 Forms$PageTemplate, 22 Forms$Layout,
// 4 Forms$Snippet — all sharing the Forms$ prefix). The PageRepo
// must filter strictly to ref.Type == "Forms$Page" so PageTemplates
// don't pollute ListAll.

const fixturePageCount = 16

// TestPageRepo_ListAll_FixtureCount asserts the strict Forms$Page
// filtering — without it ListAll returns 16+46 because mmpr's
// ListUnitsByType is prefix-based.
func TestPageRepo_ListAll_FixtureCount(t *testing.T) {
	w := openTestWriter(t)
	repo := NewPageRepository(w)
	pages, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(pages) != fixturePageCount {
		t.Errorf("ListAll: got %d pages, want %d (PageTemplate filtering broken?)", len(pages), fixturePageCount)
	}
}

func TestPageRepo_CreateGetCycle_FreshPage(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")

	repo := NewPageRepository(w)
	page := newEmptyPage(t, "FreshPageProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name() != "FreshPageProbe" {
		t.Errorf("post-Create Name = %q, want FreshPageProbe", got.Name())
	}
	if got.TypeName() != "Forms$Page" {
		t.Errorf("post-Create TypeName = %q, want Forms$Page", got.TypeName())
	}

	// ListAll must now see fixturePageCount + 1.
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != fixturePageCount+1 {
		t.Errorf("ListAll: got %d pages, want %d (fixture %d + 1 fresh)", len(all), fixturePageCount+1, fixturePageCount)
	}
}

func TestPageRepo_UpdateRoundTrip_FreshPage(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newEmptyPage(t, "UpdateProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got.SetName("UpdateProbeRenamed")
	if err := repo.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got2, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-Update: %v", err)
	}
	if got2.Name() != "UpdateProbeRenamed" {
		t.Errorf("post-Update Name = %q, want UpdateProbeRenamed", got2.Name())
	}
}

func TestPageRepo_FindByQualifiedName_FreshPage(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newEmptyPage(t, "QNProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByQualifiedName("MyFirstModule.QNProbe")
	if err != nil {
		t.Fatalf("FindByQualifiedName: %v", err)
	}
	if got == nil {
		t.Fatal("FindByQualifiedName returned nil for known page")
	}
	if got.ID() != page.ID() {
		t.Errorf("FindByQualifiedName ID = %s, want %s", got.ID(), page.ID())
	}
}

// TestPageMutator_OpenAndCommit verifies the OpenForMutation +
// Commit round-trip: open a fresh page, leave it unchanged, Commit,
// and verify the bytes still decode as the same page.
func TestPageMutator_OpenAndCommit(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newEmptyPage(t, "MutatorOpenCommit")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-Commit: %v", err)
	}
	if got.Name() != "MutatorOpenCommit" {
		t.Errorf("post-Commit Name = %q, want MutatorOpenCommit", got.Name())
	}
}

// TestPageMutator_SetWidgetProperty_OnPageRoot exercises the reflection
// dispatch by mutating the Page itself (the Page is its own root and
// findElementByID returns it for that ID).
func TestPageMutator_SetWidgetProperty_OnPageRoot(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newEmptyPage(t, "SetPropProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	if err := mut.SetWidgetProperty(model.ID(page.ID()), "Name", "SetPropProbeRenamed"); err != nil {
		t.Fatalf("SetWidgetProperty: %v", err)
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-Commit: %v", err)
	}
	if got.Name() != "SetPropProbeRenamed" {
		t.Errorf("post-Commit Name = %q, want SetPropProbeRenamed", got.Name())
	}
}

// TestPageMutator_StubMethods_ReturnExplicitErrors guards Stage 2.5
// follow-ups: InsertWidget / DeleteWidget / ReplaceWidget / SetLayout
// must surface explicit errors rather than fail silently.
func TestPageMutator_StubMethods_ReturnExplicitErrors(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)
	page := newEmptyPage(t, "StubProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}
	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}

	probeID := model.ID("00000000-0000-0000-0000-000000000999")

	if err := mut.InsertWidget(probeID, "slot", nil); err == nil ||
		!strings.Contains(err.Error(), "Stage 2.5") {
		t.Errorf("InsertWidget: want Stage 2.5 stub error, got %v", err)
	}
	if err := mut.DeleteWidget(probeID); err == nil ||
		!strings.Contains(err.Error(), "Stage 2.5") {
		t.Errorf("DeleteWidget: want Stage 2.5 stub error, got %v", err)
	}
	if err := mut.ReplaceWidget(probeID, nil); err == nil ||
		!strings.Contains(err.Error(), "Stage 2.5") {
		t.Errorf("ReplaceWidget: want Stage 2.5 stub error, got %v", err)
	}
	if err := mut.SetLayout("Layouts.Foo"); err == nil ||
		!strings.Contains(err.Error(), "Stage 2.5") {
		t.Errorf("SetLayout: want Stage 2.5 stub error, got %v", err)
	}
}

// --- helpers ---

func lookupModuleUUID(t *testing.T, w *mmpr.Writer, name string) string {
	t.Helper()
	mods, err := w.ConcreteReader().ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}
	for _, m := range mods {
		if m.Name == name {
			return m.ID
		}
	}
	t.Fatalf("fixture missing module %q", name)
	return ""
}

func newEmptyPage(t *testing.T, name string) *genPg.Page {
	t.Helper()
	page := genPg.NewPage() // already SetTypeName("Forms$Page") in initPage
	page.SetID(element.ID(mmpr.GenerateID()))
	page.SetName(name)
	return page
}
