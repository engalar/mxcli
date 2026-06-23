// SPDX-License-Identifier: Apache-2.0

// Task 5 tests: execCreateJavaScriptAction handler (Phase 1: file + platform).

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// jsActionRepoStub is a minimal repos.JavaScriptActionRepository for tests.
// ListAll returns the pre-populated items; GetContainerUUID resolves each
// item's container via containerOf so the hierarchy can resolve module names.
type jsActionRepoStub struct {
	items       []*genJSA.JavaScriptAction
	containerOf map[model.ID]model.ID
}

func (r *jsActionRepoStub) Get(id model.ID) (*genJSA.JavaScriptAction, error) {
	for _, j := range r.items {
		if model.ID(j.ID()) == id {
			return j, nil
		}
	}
	return nil, nil
}
func (r *jsActionRepoStub) List(moduleID model.ID) ([]*genJSA.JavaScriptAction, error) {
	return r.items, nil
}
func (r *jsActionRepoStub) ListAll() ([]*genJSA.JavaScriptAction, error) { return r.items, nil }
func (r *jsActionRepoStub) FindByQualifiedName(qn string) (*genJSA.JavaScriptAction, error) {
	return nil, nil
}
func (r *jsActionRepoStub) GetContainerUUID(id model.ID) (model.ID, error) {
	if c, ok := r.containerOf[id]; ok {
		return c, nil
	}
	return "", nil
}
func (r *jsActionRepoStub) Create(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error {
	r.items = append(r.items, jsa)
	if r.containerOf != nil {
		r.containerOf[model.ID(jsa.ID())] = model.ID(parentUUID)
	}
	return nil
}
func (r *jsActionRepoStub) Update(jsa *genJSA.JavaScriptAction) error { return nil }

func TestExecCreateJavaScriptAction_CreatesFromScratch(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	mprDir := t.TempDir()
	mprPath := mprDir + "/test.mpr"
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		CreateJavaScriptActionGenFunc: func(parentUUID, containmentName string, jsa *genJSA.JavaScriptAction) error {
			return nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.MprPath = mprPath
	ctx.JavaScriptActions = &jsActionRepoStub{} // empty — will be created

	stmt := &ast.CreateJavaScriptActionStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "MyAction"},
		CreateOrModify: true,
		UserCode:       "return true;",
	}
	if err := execCreateJavaScriptActionFn(ctx, stmt, execContextToDeps(ctx)); err != nil {
		t.Fatalf("expected success (create from scratch), got: %v", err)
	}
}

func TestExecCreateJavaScriptAction_UpdatesPlatform(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	existing := genJSA.NewJavaScriptAction()
	existingID := element.ID(nextID("jsa"))
	existing.SetID(existingID)
	existing.SetName("MyAction")
	repo := &jsActionRepoStub{
		items:       []*genJSA.JavaScriptAction{existing},
		containerOf: map[model.ID]model.ID{model.ID(existingID): mod.ID},
	}

	updated := []*genJSA.JavaScriptAction{}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		UpdateJavaScriptActionGenFunc: func(jsa *genJSA.JavaScriptAction) error {
			updated = append(updated, jsa)
			return nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	ctx.JavaScriptActions = repo

	stmt := &ast.CreateJavaScriptActionStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "MyAction"},
		Platform:       "Web",
		CreateOrModify: true,
	}
	if err := execCreateJavaScriptActionFn(ctx, stmt, execContextToDeps(ctx)); err != nil {
		t.Fatalf("execCreateJavaScriptAction: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected 1 UpdateJavaScriptActionGen call, got %d", len(updated))
	}
	if updated[0].Platform() != "Web" {
		t.Errorf("Platform = %q, want Web", updated[0].Platform())
	}
}
