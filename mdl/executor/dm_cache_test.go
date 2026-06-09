// SPDX-License-Identifier: Apache-2.0
package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

type mockDMBackend struct {
	mock.MockBackend
	callCount int
	dm        *genDm.DomainModel
}

func (m *mockDMBackend) GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error) {
	m.callCount++
	return m.dm, nil
}

func newDMCtx(mb *mockDMBackend) *ExecContext {
	return &ExecContext{
		Backend:     mb,
		ExecSession: ExecSession{Cache: &executorCache{}},
	}
}

func TestGetDomainModelGenCached_FirstCallHitsBackend(t *testing.T) {
	dm := &genDm.DomainModel{}
	mb := &mockDMBackend{dm: dm}
	ctx := newDMCtx(mb)

	result, err := getDomainModelGenCached(ctx, "mod-1")
	if err != nil {
		t.Fatal(err)
	}
	if result != dm {
		t.Error("expected backend's dm")
	}
	if mb.callCount != 1 {
		t.Errorf("expected 1 backend call, got %d", mb.callCount)
	}
}

func TestGetDomainModelGenCached_SecondCallHitsCache(t *testing.T) {
	dm := &genDm.DomainModel{}
	mb := &mockDMBackend{dm: dm}
	ctx := newDMCtx(mb)

	getDomainModelGenCached(ctx, "mod-1") //nolint
	result, err := getDomainModelGenCached(ctx, "mod-1")
	if err != nil {
		t.Fatal(err)
	}
	if result != dm {
		t.Error("expected cached dm")
	}
	if mb.callCount != 1 {
		t.Errorf("second call must use cache (1 backend call total), got %d", mb.callCount)
	}
}

func TestSetDomainModelGenCached_WriteThrough(t *testing.T) {
	dm1 := &genDm.DomainModel{}
	dm2 := &genDm.DomainModel{}
	mb := &mockDMBackend{dm: dm1}
	ctx := newDMCtx(mb)

	getDomainModelGenCached(ctx, "mod-1") //nolint

	setDomainModelGenCached(ctx, "mod-1", dm2)

	result, err := getDomainModelGenCached(ctx, "mod-1")
	if err != nil {
		t.Fatal(err)
	}
	if result != dm2 {
		t.Errorf("write-through failed: got %p, want %p", result, dm2)
	}
	if mb.callCount != 1 {
		t.Errorf("expected 1 backend call total after write-through, got %d", mb.callCount)
	}
}

func TestGetDomainModelGenCached_NilCacheInitializes(t *testing.T) {
	dm := &genDm.DomainModel{}
	mb := &mockDMBackend{dm: dm}
	ctx := &ExecContext{
		Backend:     mb,
		ExecSession: ExecSession{Cache: nil},
	}

	result, err := getDomainModelGenCached(ctx, "mod-1")
	if err != nil {
		t.Fatal(err)
	}
	if result != dm {
		t.Error("expected dm even with nil cache")
	}
	if ctx.Cache == nil {
		t.Error("getDomainModelGenCached must initialize cache if nil")
	}
}
