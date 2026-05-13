// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
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
		Backend:    mb,
		Microflows: nil, // explicit: no repo, must fall back
	}
	if err := ctx.deleteMicroflowViaRepoOrBackend(wantID); err != nil {
		t.Fatalf("deleteMicroflowViaRepoOrBackend: %v", err)
	}
	if !called {
		t.Error("Backend.DeleteMicroflow was not invoked when Microflows = nil")
	}
}
