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

// TestPageMutator_StubErrors_OnInvalidInput guards the explicit-error
// surface for invalid arguments after Stage 2.5 wired up the four
// previously-stub methods. The intent is the same as the original
// "Stage 2.5 follow-up" sentinel: never fail silently.
func TestPageMutator_StubErrors_OnInvalidInput(t *testing.T) {
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

	// nil widget on Insert
	if err := mut.InsertWidget(probeID, "Widgets", nil); err == nil ||
		!strings.Contains(err.Error(), "nil") {
		t.Errorf("InsertWidget(nil): want nil-widget error, got %v", err)
	}
	// Insert into a non-existent parent
	if err := mut.InsertWidget(probeID, "Widgets", genPg.NewLayoutCallArgument()); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("InsertWidget(unknown parent): want parent-not-found error, got %v", err)
	}
	// Delete unknown widget
	if err := mut.DeleteWidget(probeID); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("DeleteWidget(unknown): want not-found error, got %v", err)
	}
	// Replace with nil
	if err := mut.ReplaceWidget(probeID, nil); err == nil ||
		!strings.Contains(err.Error(), "nil") {
		t.Errorf("ReplaceWidget(nil): want nil-replacement error, got %v", err)
	}
	// SetLayout to invalid QN
	if err := mut.SetLayout("BadFormatNoDot"); err == nil ||
		!strings.Contains(err.Error(), "invalid qualified name") {
		t.Errorf("SetLayout(bad qn): want invalid-qn error, got %v", err)
	}
	// SetLayout to non-existent layout
	if err := mut.SetLayout("Atlas_Core.NoSuchLayout"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("SetLayout(missing): want not-found error, got %v", err)
	}
}

// TestPageMutator_InsertWidget_AddsToParentSlot verifies that
// InsertWidget appends a child to the named PartList slot on a parent
// element identified by ID, and the change survives a Commit + reopen.
func TestPageMutator_InsertWidget_AddsToParentSlot(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newPageWithLayoutCallArgument(t, "InsertProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}
	lc := page.LayoutCall().(*genPg.LayoutCall)
	arg := lc.ArgumentsItems()[0].(*genPg.LayoutCallArgument)

	// Open mutator → insert a fresh DataView under the LayoutCallArgument's "Widgets" slot
	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	dv := genPg.NewDataView()
	dv.SetID(element.ID(mmpr.GenerateID()))
	if err := mut.InsertWidget(model.ID(arg.ID()), "Widgets", dv); err != nil {
		t.Fatalf("InsertWidget: %v", err)
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Reopen and verify the widget shows up under arg.Widgets
	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-Commit: %v", err)
	}
	gotArg := got.LayoutCall().(*genPg.LayoutCall).ArgumentsItems()[0].(*genPg.LayoutCallArgument)
	widgets := gotArg.WidgetsItems()
	if len(widgets) != 1 {
		t.Fatalf("post-Commit Widgets len = %d, want 1", len(widgets))
	}
	if widgets[0].ID() != dv.ID() {
		t.Errorf("post-Commit widget ID = %s, want %s", widgets[0].ID(), dv.ID())
	}
}

// TestPageMutator_DeleteWidget_RoundTrip verifies that a widget added
// via InsertWidget can be removed via DeleteWidget, and the deletion
// survives Commit + reopen.
func TestPageMutator_DeleteWidget_RoundTrip(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newPageWithLayoutCallArgument(t, "DeleteProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}
	lc := page.LayoutCall().(*genPg.LayoutCall)
	arg := lc.ArgumentsItems()[0].(*genPg.LayoutCallArgument)

	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	dv := genPg.NewDataView()
	dv.SetID(element.ID(mmpr.GenerateID()))
	if err := mut.InsertWidget(model.ID(arg.ID()), "Widgets", dv); err != nil {
		t.Fatalf("InsertWidget: %v", err)
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit (post-insert): %v", err)
	}

	// Reopen and delete
	mut2, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation 2: %v", err)
	}
	if err := mut2.DeleteWidget(model.ID(dv.ID())); err != nil {
		t.Fatalf("DeleteWidget: %v", err)
	}
	if err := mut2.Commit(); err != nil {
		t.Fatalf("Commit (post-delete): %v", err)
	}

	// Reopen and verify gone
	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-delete: %v", err)
	}
	gotArg := got.LayoutCall().(*genPg.LayoutCall).ArgumentsItems()[0].(*genPg.LayoutCallArgument)
	if n := len(gotArg.WidgetsItems()); n != 0 {
		t.Errorf("post-delete Widgets len = %d, want 0", n)
	}
}

// TestPageMutator_ReplaceWidget_PreservesIndex verifies that ReplaceWidget
// swaps the middle of a 3-widget list while keeping the order:
// [A, B, C] → ReplaceWidget(B, X) → [A, X, C].
func TestPageMutator_ReplaceWidget_PreservesIndex(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newPageWithLayoutCallArgument(t, "ReplaceProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}
	lc := page.LayoutCall().(*genPg.LayoutCall)
	arg := lc.ArgumentsItems()[0].(*genPg.LayoutCallArgument)

	a := freshDataView(t, "A")
	b := freshDataView(t, "B")
	c := freshDataView(t, "C")

	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	for _, dv := range []*genPg.DataView{a, b, c} {
		if err := mut.InsertWidget(model.ID(arg.ID()), "Widgets", dv); err != nil {
			t.Fatalf("InsertWidget(%s): %v", dv.ID(), err)
		}
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit (post-3-inserts): %v", err)
	}

	// Reopen and replace B → X
	mut2, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation 2: %v", err)
	}
	x := freshDataView(t, "X")
	if err := mut2.ReplaceWidget(model.ID(b.ID()), x); err != nil {
		t.Fatalf("ReplaceWidget: %v", err)
	}
	if err := mut2.Commit(); err != nil {
		t.Fatalf("Commit (post-replace): %v", err)
	}

	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-replace: %v", err)
	}
	gotArg := got.LayoutCall().(*genPg.LayoutCall).ArgumentsItems()[0].(*genPg.LayoutCallArgument)
	widgets := gotArg.WidgetsItems()
	if len(widgets) != 3 {
		t.Fatalf("post-replace Widgets len = %d, want 3", len(widgets))
	}
	gotIDs := []element.ID{widgets[0].ID(), widgets[1].ID(), widgets[2].ID()}
	wantIDs := []element.ID{a.ID(), x.ID(), c.ID()}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("post-replace position %d = %s, want %s", i, gotIDs[i], wantIDs[i])
		}
	}
}

// TestPageMutator_SetLayout_ChangesReference verifies that SetLayout
// updates the page's LayoutCall.LayoutQualifiedName and the change
// survives Commit + reopen.
func TestPageMutator_SetLayout_ChangesReference(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newPageWithLayoutCallArgument(t, "SetLayoutProbe")
	// Pre-set a layout so we can verify the change.
	page.LayoutCall().(*genPg.LayoutCall).SetLayoutQualifiedName("Atlas_Core.Atlas_TopBar")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}
	if err := mut.SetLayout("Atlas_Core.Atlas_Default"); err != nil {
		t.Fatalf("SetLayout: %v", err)
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-Commit: %v", err)
	}
	gotLC, ok := got.LayoutCall().(*genPg.LayoutCall)
	if !ok || gotLC == nil {
		t.Fatalf("post-Commit LayoutCall = %v (%T), want *genPg.LayoutCall", got.LayoutCall(), got.LayoutCall())
	}
	if qn := gotLC.LayoutQualifiedName(); qn != "Atlas_Core.Atlas_Default" {
		t.Errorf("post-Commit LayoutQualifiedName = %q, want Atlas_Core.Atlas_Default", qn)
	}
}

// TestPageMutator_SetLayout_UsesResolver asserts SetLayout now goes
// through QualifiedNameResolver: a QN that resolves to a non-layout
// kind (e.g. a microflow) is rejected with a wrong-kind error rather
// than the prior "layout not found" lookup miss.
func TestPageMutator_SetLayout_UsesResolver(t *testing.T) {
	w := openTestWriter(t)
	parentID := lookupModuleUUID(t, w, "MyFirstModule")
	repo := NewPageRepository(w)

	page := newPageWithLayoutCallArgument(t, "SetLayoutResolverProbe")
	if err := repo.Create(parentID, "Documents", page); err != nil {
		t.Fatalf("Create: %v", err)
	}
	mut, err := repo.OpenForMutation(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("OpenForMutation: %v", err)
	}

	// Resolver-known QN that points at a microflow, NOT a layout.
	if err := mut.SetLayout("Administration.ChangeMyPassword"); err == nil {
		t.Error("SetLayout(microflow QN): expected wrong-kind error, got nil")
	} else if !strings.Contains(err.Error(), "is a microflow") {
		t.Errorf("SetLayout(microflow QN) error = %v, want substring \"is a microflow\"", err)
	}

	// Real layout still works end-to-end.
	if err := mut.SetLayout("Atlas_Core.Atlas_Default"); err != nil {
		t.Fatalf("SetLayout(real layout): %v", err)
	}
	if err := mut.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	got, err := repo.Get(model.ID(page.ID()))
	if err != nil {
		t.Fatalf("Get post-Commit: %v", err)
	}
	gotLC, ok := got.LayoutCall().(*genPg.LayoutCall)
	if !ok || gotLC == nil {
		t.Fatalf("post-Commit LayoutCall = %v (%T)", got.LayoutCall(), got.LayoutCall())
	}
	if qn := gotLC.LayoutQualifiedName(); qn != "Atlas_Core.Atlas_Default" {
		t.Errorf("post-Commit LayoutQualifiedName = %q, want Atlas_Core.Atlas_Default", qn)
	}
}

// TestPageRepo_GetContainerUUID_NonEmpty verifies the Stage 3.3.5.A0
// container linkage — listPagesWithContainerGen relies on it to pair
// each page with its parent module/folder UUID without an extra unit
// scan.
func TestPageRepo_GetContainerUUID_NonEmpty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewPageRepository(w)
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("fixture has no pages")
	}
	cid, err := repo.GetContainerUUID(model.ID(all[0].ID()))
	if err != nil {
		t.Fatalf("GetContainerUUID: %v", err)
	}
	if cid == "" {
		t.Error("GetContainerUUID returned empty container UUID")
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

// newPageWithLayoutCallArgument builds a fresh page with a populated
// LayoutCall containing a single LayoutCallArgument — the standard
// shape Mendix pages use to host root widgets.
func newPageWithLayoutCallArgument(t *testing.T, name string) *genPg.Page {
	t.Helper()
	page := newEmptyPage(t, name)
	lc := genPg.NewLayoutCall()
	lc.SetID(element.ID(mmpr.GenerateID()))
	arg := genPg.NewLayoutCallArgument()
	arg.SetID(element.ID(mmpr.GenerateID()))
	lc.AddArguments(arg)
	page.SetLayoutCall(lc)
	return page
}

func freshDataView(t *testing.T, name string) *genPg.DataView {
	t.Helper()
	dv := genPg.NewDataView()
	dv.SetID(element.ID(mmpr.GenerateID()))
	if setter, ok := any(dv).(interface{ SetName(string) }); ok {
		setter.SetName(name)
	}
	return dv
}
