// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestExtractMicroflowsRepo_NilBackend(t *testing.T) {
	if got := extractMicroflowsRepo(nil); got != nil {
		t.Errorf("extractMicroflowsRepo(nil) = %v, want nil", got)
	}
}

func TestExtractMicroflowsRepo_MockBackend_ReturnsNil(t *testing.T) {
	// MockBackend has no Microflows() method, so extract should return nil
	// and handlers must fall back to ctx.Backend.
	mb := &mock.MockBackend{}
	var b backend.FullBackend = mb
	if got := extractMicroflowsRepo(b); got != nil {
		t.Errorf("extractMicroflowsRepo(MockBackend) = %v, want nil (mock has no Microflows())", got)
	}
}

func TestExtractMicroflowsRepo_MprBackend_Disconnected_ReturnsNil(t *testing.T) {
	// Unconnected MprBackend has nil msdkWriter — Microflows() returns nil.
	mb := mprbackend.New()
	var b backend.FullBackend = mb
	if got := extractMicroflowsRepo(b); got != nil {
		t.Errorf("extractMicroflowsRepo(unconnected MprBackend) = %v, want nil", got)
	}
}

func TestExtractNanoflowsRepo_NilBackend(t *testing.T) {
	if got := extractNanoflowsRepo(nil); got != nil {
		t.Errorf("extractNanoflowsRepo(nil) = %v, want nil", got)
	}
}

func TestExtractNanoflowsRepo_MockBackend_ReturnsNil(t *testing.T) {
	mb := &mock.MockBackend{}
	var b backend.FullBackend = mb
	if got := extractNanoflowsRepo(b); got != nil {
		t.Errorf("extractNanoflowsRepo(MockBackend) = %v, want nil", got)
	}
}

func TestExtractNanoflowsRepo_MprBackend_Disconnected_ReturnsNil(t *testing.T) {
	mb := mprbackend.New()
	var b backend.FullBackend = mb
	if got := extractNanoflowsRepo(b); got != nil {
		t.Errorf("extractNanoflowsRepo(unconnected MprBackend) = %v, want nil", got)
	}
}

func TestDeleteNanoflowViaRepoOrBackend_FallbackToMockBackend(t *testing.T) {
	called := false
	wantID := model.ID("nf-pilot-789")
	mb := &mock.MockBackend{
		DeleteNanoflowFunc: func(id model.ID) error {
			called = true
			if id != wantID {
				t.Errorf("Backend.DeleteNanoflow called with %q, want %q", id, wantID)
			}
			return nil
		},
	}
	ctx := &ExecContext{
		Backend:   mb,
		ExecRepos: ExecRepos{Nanoflows: nil},
	}
	ctx.initRoles()
	if err := ctx.deleteNanoflowViaRepoOrBackend(wantID); err != nil {
		t.Fatalf("deleteNanoflowViaRepoOrBackend: %v", err)
	}
	if !called {
		t.Error("Backend.DeleteNanoflow was not invoked when Nanoflows = nil")
	}
}

// fakeMicroflowRepo lets us assert isRuleViaRepoOrBackend prefers the repo
// path when set, without spinning up a real fixture.
type fakeMicroflowRepo struct {
	isRuleResult bool
	isRuleErr    error
	isRuleCalls  []string
}

func (f *fakeMicroflowRepo) Get(id model.ID) (*genMf.Microflow, error)              { return nil, nil }
func (f *fakeMicroflowRepo) List(_ model.ID) ([]*genMf.Microflow, error)            { return nil, nil }
func (f *fakeMicroflowRepo) ListAll() ([]*genMf.Microflow, error)                   { return nil, nil }
func (f *fakeMicroflowRepo) FindByQualifiedName(_ string) (*genMf.Microflow, error) { return nil, nil }
func (f *fakeMicroflowRepo) IsRule(qn string) (bool, error) {
	f.isRuleCalls = append(f.isRuleCalls, qn)
	return f.isRuleResult, f.isRuleErr
}
func (f *fakeMicroflowRepo) GetContainerUUID(_ model.ID) (model.ID, error) { return "", nil }
func (f *fakeMicroflowRepo) Create(_, _ string, _ *genMf.Microflow) error  { return nil }
func (f *fakeMicroflowRepo) Update(_ *genMf.Microflow) error               { return nil }
func (f *fakeMicroflowRepo) Delete(_ model.ID) error                       { return nil }
func (f *fakeMicroflowRepo) Move(_ model.ID, _ string) error               { return nil }

func TestIsRuleViaRepoOrBackend_PrefersRepoWhenSet(t *testing.T) {
	repo := &fakeMicroflowRepo{isRuleResult: true}
	mb := &mock.MockBackend{
		IsRuleFunc: func(_ string) (bool, error) { return false, errors.New("backend should not be called") },
	}
	ok, err := isRuleViaRepoOrBackend(repo, mb, "Module.SomeRule")
	if err != nil {
		t.Fatalf("isRuleViaRepoOrBackend: %v", err)
	}
	if !ok {
		t.Error("ok = false, want true (repo override)")
	}
	if got, want := len(repo.isRuleCalls), 1; got != want {
		t.Errorf("repo.IsRule calls = %d, want %d", got, want)
	}
	if repo.isRuleCalls[0] != "Module.SomeRule" {
		t.Errorf("repo.IsRule got %q, want Module.SomeRule", repo.isRuleCalls[0])
	}
}

func TestIsRuleViaRepoOrBackend_FallsBackToBackendWhenRepoNil(t *testing.T) {
	called := false
	mb := &mock.MockBackend{
		IsRuleFunc: func(qn string) (bool, error) {
			called = true
			return true, nil
		},
	}
	ok, err := isRuleViaRepoOrBackend(nil, mb, "Module.X")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !called {
		t.Errorf("ok=%v called=%v; both want true", ok, called)
	}
}

func TestDeleteMicroflowViaRepoOrBackend_FallbackToMockBackend(t *testing.T) {
	// When ctx.Microflows is nil, the bridge must call Backend.DeleteMicroflow.
	called := false
	wantID := model.ID("mf-pilot-123")
	mb := &mock.MockBackend{
		DeleteMicroflowFunc: func(id model.ID) error {
			called = true
			if id != wantID {
				t.Errorf("Backend.DeleteMicroflow called with %q, want %q", id, wantID)
			}
			return nil
		},
	}
	ctx := &ExecContext{
		Backend: mb,
		// explicit: no repo, must fall back
		ExecRepos: ExecRepos{Microflows: nil},
	}
	ctx.initRoles()
	if err := ctx.deleteMicroflowViaRepoOrBackend(wantID); err != nil {
		t.Fatalf("deleteMicroflowViaRepoOrBackend: %v", err)
	}
	if !called {
		t.Error("Backend.DeleteMicroflow was not invoked when Microflows = nil")
	}
}
