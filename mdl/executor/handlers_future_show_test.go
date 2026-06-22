package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// testFutureBackend builds a mock backend that implements ModuleLister,
// MetadataReader, and FolderManager with the given modules and empty
// unit/folder listings.
func testFutureBackend(mods ...*model.Module) *mock.MockBackend {
	return &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return mods, nil },
		ListUnitsFunc:   func() ([]*types.UnitInfo, error) { return nil, nil },
		ListFoldersFunc: func() ([]*types.FolderInfo, error) { return nil, nil },
	}
}

// makeMFRepo creates a RecordingMicroflowRepository with a single microflow.
func makeMFRepo(mf *genMf.Microflow, containerID model.ID) *repostesting.RecordingMicroflowRepository {
	return &repostesting.RecordingMicroflowRepository{
		ListAllFunc:          func() ([]*genMf.Microflow, error) { return []*genMf.Microflow{mf}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return containerID, nil },
	}
}

// makeNFRepo creates a RecordingNanoflowRepository with a single nanoflow.
func makeNFRepo(nf *genMf.Nanoflow, containerID model.ID) *repostesting.RecordingNanoflowRepository {
	return &repostesting.RecordingNanoflowRepository{
		ListAllFunc:          func() ([]*genMf.Nanoflow, error) { return []*genMf.Nanoflow{nf}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return containerID, nil },
	}
}

// makeMFRepoMulti creates a RecordingMicroflowRepository with multiple
// microflows and per-ID container resolution.
func makeMFRepoMulti(mfs []*genMf.Microflow, containerByID map[element.ID]model.ID) *repostesting.RecordingMicroflowRepository {
	return &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) { return mfs, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return containerByID[element.ID(id)], nil
		},
	}
}

// makeNFRepoMulti creates a RecordingNanoflowRepository with multiple
// nanoflows and per-ID container resolution.
func makeNFRepoMulti(nfs []*genMf.Nanoflow, containerByID map[element.ID]model.ID) *repostesting.RecordingNanoflowRepository {
	return &repostesting.RecordingNanoflowRepository{
		ListAllFunc: func() ([]*genMf.Nanoflow, error) { return nfs, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			return containerByID[element.ID(id)], nil
		},
	}
}

// ─────────────────────────────────────────────────────────────
// listMicroflowsFuture tests
// ─────────────────────────────────────────────────────────────

func TestListMicroflowsFuture_Output(t *testing.T) {
	mod := mkModule("MyModule")
	mf := mkMicroflowGen("MyFlow")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	var buf bytes.Buffer
	err := listMicroflowsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, makeMFRepo(mf, mod.ID), "")
	if err != nil {
		t.Fatalf("listMicroflowsFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MyModule.MyFlow") {
		t.Errorf("expected 'MyModule.MyFlow' in output, got: %q", out)
	}
	if !strings.Contains(out, "(1 microflows)") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

func TestListMicroflowsFuture_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	mf1 := mkMicroflowGen("ProcessOrder")
	mf2 := mkMicroflowGen("OnboardEmployee")
	h := mkHierarchy(mod1, mod2)
	withContainer(h, mod1.ID, mod1.ID)
	withContainer(h, mod2.ID, mod2.ID)

	mb := testFutureBackend(mod1, mod2)
	var buf bytes.Buffer
	err := listMicroflowsFuture(context.Background(), &buf, FormatTable, mb, mb, mb,
		makeMFRepoMulti([]*genMf.Microflow{mf1, mf2},
			map[element.ID]model.ID{mf1.ID(): mod1.ID, mf2.ID(): mod2.ID}),
		"HR")
	if err != nil {
		t.Fatalf("listMicroflowsFuture: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "Sales.ProcessOrder") {
		t.Errorf("expected no Sales.ProcessOrder for HR filter, got: %s", out)
	}
	if !strings.Contains(out, "HR.OnboardEmployee") {
		t.Errorf("expected HR.OnboardEmployee in output, got: %s", out)
	}
}

func TestListMicroflowsFuture_Empty(t *testing.T) {
	mod := mkModule("MyModule")
	mb := testFutureBackend(mod)
	var buf bytes.Buffer
	err := listMicroflowsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, makeMFRepo(nil, mod.ID), "")
	if err != nil {
		t.Fatalf("listMicroflowsFuture: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(0 microflows)") {
		t.Errorf("expected '(0 microflows)', got: %q", out)
	}
}

func TestListMicroflowsFuture_BackendError(t *testing.T) {
	mod := mkModule("MyModule")
	mb := testFutureBackend(mod)
	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) { return nil, errBackend },
	}
	var buf bytes.Buffer
	err := listMicroflowsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, mfRepo, "")
	if err == nil {
		t.Fatal("expected error from listMicroflowsFuture, got nil")
	}
}

// ─────────────────────────────────────────────────────────────
// listNanoflowsFuture tests
// ─────────────────────────────────────────────────────────────

func TestListNanoflowsFuture_Output(t *testing.T) {
	mod := mkModule("MyModule")
	nf := mkNanoflowGen("MyNano")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	var buf bytes.Buffer
	err := listNanoflowsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, makeNFRepo(nf, mod.ID), "")
	if err != nil {
		t.Fatalf("listNanoflowsFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MyModule.MyNano") {
		t.Errorf("expected 'MyModule.MyNano' in output, got: %q", out)
	}
	if !strings.Contains(out, "(1 nanoflows)") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

func TestListNanoflowsFuture_FilterByModule(t *testing.T) {
	mod1 := mkModule("A")
	mod2 := mkModule("B")
	nf1 := mkNanoflowGen("One")
	nf2 := mkNanoflowGen("Two")
	h := mkHierarchy(mod1, mod2)
	withContainer(h, mod1.ID, mod1.ID)
	withContainer(h, mod2.ID, mod2.ID)

	mb := testFutureBackend(mod1, mod2)
	var buf bytes.Buffer
	err := listNanoflowsFuture(context.Background(), &buf, FormatTable, mb, mb, mb,
		makeNFRepoMulti([]*genMf.Nanoflow{nf1, nf2},
			map[element.ID]model.ID{nf1.ID(): mod1.ID, nf2.ID(): mod2.ID}),
		"B")
	if err != nil {
		t.Fatalf("listNanoflowsFuture: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "A.One") {
		t.Errorf("expected no A.One for B filter, got: %s", out)
	}
	if !strings.Contains(out, "B.Two") {
		t.Errorf("expected B.Two in output, got: %s", out)
	}
}

// ─────────────────────────────────────────────────────────────
// listPagesFuture tests
// ─────────────────────────────────────────────────────────────

func TestListPagesFuture_Output(t *testing.T) {
	mod := mkModule("MyModule")
	pg := mkPageGen(string(nextID("pg")), "Home")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	pageRepo := &repostesting.RecordingPageRepository{
		ListAllFunc:          func() ([]*genPg.Page, error) { return []*genPg.Page{pg}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}
	var buf bytes.Buffer
	err := listPagesFuture(context.Background(), &buf, FormatTable, mb, mb, mb, pageRepo, "")
	if err != nil {
		t.Fatalf("listPagesFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MyModule.Home") {
		t.Errorf("expected 'MyModule.Home' in output, got: %q", out)
	}
	if !strings.Contains(out, "(1 pages)") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

func TestListPagesFuture_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	pg1 := mkPageGen(string(nextID("pg")), "OrderList")
	pg2 := mkPageGen(string(nextID("pg")), "EmployeeList")
	h := mkHierarchy(mod1, mod2)
	withContainer(h, mod1.ID, mod1.ID)
	withContainer(h, mod2.ID, mod2.ID)

	mb := testFutureBackend(mod1, mod2)
	pageRepo := &repostesting.RecordingPageRepository{
		ListAllFunc: func() ([]*genPg.Page, error) { return []*genPg.Page{pg1, pg2}, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == model.ID(pg1.ID()) {
				return mod1.ID, nil
			}
			return mod2.ID, nil
		},
	}
	var buf bytes.Buffer
	err := listPagesFuture(context.Background(), &buf, FormatTable, mb, mb, mb, pageRepo, "HR")
	if err != nil {
		t.Fatalf("listPagesFuture: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "Sales.OrderList") {
		t.Errorf("expected no Sales.OrderList for HR filter, got: %s", out)
	}
	if !strings.Contains(out, "HR.EmployeeList") {
		t.Errorf("expected HR.EmployeeList in output, got: %s", out)
	}
}

// ─────────────────────────────────────────────────────────────
// listSnippetsFuture tests
// ─────────────────────────────────────────────────────────────

func TestListSnippetsFuture_Output(t *testing.T) {
	mod := mkModule("MyModule")
	snp := mkSnippetGen(string(nextID("snp")), "Header")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	snpRepo := &repostesting.RecordingSnippetRepository{
		ListAllFunc:          func() ([]*genPg.Snippet, error) { return []*genPg.Snippet{snp}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}
	var buf bytes.Buffer
	err := listSnippetsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, snpRepo, "")
	if err != nil {
		t.Fatalf("listSnippetsFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MyModule.Header") {
		t.Errorf("expected 'MyModule.Header' in output, got: %q", out)
	}
	if !strings.Contains(out, "(1 snippets)") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

// ─────────────────────────────────────────────────────────────
// listLayoutsFuture tests
// ─────────────────────────────────────────────────────────────

func TestListLayoutsFuture_Output(t *testing.T) {
	mod := mkModule("MyModule")
	lay := mkLayoutGen(string(nextID("lay")), "Default")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	layRepo := &repostesting.RecordingLayoutRepository{
		ListAllFunc:          func() ([]*genPg.Layout, error) { return []*genPg.Layout{lay}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}
	var buf bytes.Buffer
	err := listLayoutsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, layRepo, "")
	if err != nil {
		t.Fatalf("listLayoutsFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MyModule.Default") {
		t.Errorf("expected 'MyModule.Default' in output, got: %q", out)
	}
	if !strings.Contains(out, "(1 layouts)") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

func TestListLayoutsFuture_FilterByModule(t *testing.T) {
	mod1 := mkModule("X")
	mod2 := mkModule("Y")
	lay1 := mkLayoutGen(string(nextID("lay")), "LayoutA")
	lay2 := mkLayoutGen(string(nextID("lay")), "LayoutB")
	h := mkHierarchy(mod1, mod2)
	withContainer(h, mod1.ID, mod1.ID)
	withContainer(h, mod2.ID, mod2.ID)

	mb := testFutureBackend(mod1, mod2)
	layRepo := &repostesting.RecordingLayoutRepository{
		ListAllFunc: func() ([]*genPg.Layout, error) { return []*genPg.Layout{lay1, lay2}, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == model.ID(lay1.ID()) {
				return mod1.ID, nil
			}
			return mod2.ID, nil
		},
	}
	var buf bytes.Buffer
	err := listLayoutsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, layRepo, "Y")
	if err != nil {
		t.Fatalf("listLayoutsFuture: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "X.LayoutA") {
		t.Errorf("expected no X.LayoutA for Y filter, got: %s", out)
	}
	if !strings.Contains(out, "Y.LayoutB") {
		t.Errorf("expected Y.LayoutB in output, got: %s", out)
	}
}

// ─────────────────────────────────────────────────────────────
// listJavaActionsFuture tests
// ─────────────────────────────────────────────────────────────

func TestListJavaActionsFuture_Output(t *testing.T) {
	mod := mkModule("MyModule")
	ja := genJA.NewJavaAction()
	ja.SetID(element.ID(nextID("ja")))
	ja.SetName("DoSomething")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	jaRepo := &repostesting.RecordingJavaActionRepository{
		ListAllFunc:          func() ([]*genJA.JavaAction, error) { return []*genJA.JavaAction{ja}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}
	var buf bytes.Buffer
	err := listJavaActionsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, jaRepo, "")
	if err != nil {
		t.Fatalf("listJavaActionsFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MyModule.DoSomething") {
		t.Errorf("expected 'MyModule.DoSomething' in output, got: %q", out)
	}
	if !strings.Contains(out, "java actions") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

func TestListJavaActionsFuture_BackendError(t *testing.T) {
	mod := mkModule("MyModule")
	mb := testFutureBackend(mod)
	jaRepo := &repostesting.RecordingJavaActionRepository{
		ListAllFunc: func() ([]*genJA.JavaAction, error) { return nil, errBackend },
	}
	var buf bytes.Buffer
	err := listJavaActionsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, jaRepo, "")
	if err == nil {
		t.Fatal("expected error from listJavaActionsFuture, got nil")
	}
}

// ─────────────────────────────────────────────────────────────
// listJavaScriptActionsFuture tests
// ─────────────────────────────────────────────────────────────

func TestListJavaScriptActionsFuture_Output(t *testing.T) {
	mod := mkModule("MyModule")
	jsa := genJSA.NewJavaScriptAction()
	jsa.SetID(element.ID(nextID("jsa")))
	jsa.SetName("MyAction")
	jsa.SetPlatform("Web")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	jsaRepo := &repostesting.RecordingJavaScriptActionRepository{
		ListAllFunc:          func() ([]*genJSA.JavaScriptAction, error) { return []*genJSA.JavaScriptAction{jsa}, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return mod.ID, nil },
	}
	var buf bytes.Buffer
	err := listJavaScriptActionsFuture(context.Background(), &buf, FormatTable, mb, mb, mb, jsaRepo, "")
	if err != nil {
		t.Fatalf("listJavaScriptActionsFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "MyModule.MyAction") {
		t.Errorf("expected 'MyModule.MyAction' in output, got: %q", out)
	}
	if !strings.Contains(out, "javascript actions") {
		t.Errorf("expected count summary, got: %q", out)
	}
}

// ─────────────────────────────────────────────────────────────
// Registration dispatch test
// ─────────────────────────────────────────────────────────────

func TestListMicroflowsFuture_JSON(t *testing.T) {
	mod := mkModule("MyModule")
	mf := mkMicroflowGen("JSONFlow")
	h := mkHierarchy(mod)
	withContainer(h, mod.ID, mod.ID)

	mb := testFutureBackend(mod)
	var buf bytes.Buffer
	err := listMicroflowsFuture(context.Background(), &buf, FormatJSON, mb, mb, mb, makeMFRepo(mf, mod.ID), "")
	if err != nil {
		t.Fatalf("listMicroflowsFuture: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "JSONFlow") {
		t.Errorf("expected 'JSONFlow' in JSON output, got: %q", out)
	}
}
