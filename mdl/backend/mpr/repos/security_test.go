// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

func TestSecurityRepo_Get_Singleton(t *testing.T) {
	w := openTestWriter(t)
	repo := NewSecurityRepository(w)
	ps, err := repo.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ps == nil {
		t.Fatal("Get returned nil project security")
	}
	if ps.ID() == "" {
		t.Error("project security has empty ID")
	}
	if ps.TypeName() != projectSecurityTypeName {
		t.Errorf("TypeName = %q, want %q", ps.TypeName(), projectSecurityTypeName)
	}
}

func TestSecurityRepo_UpdateRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	repo := NewSecurityRepository(w)
	ps, err := repo.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := repo.Update(ps); err != nil {
		t.Fatalf("Update: %v", err)
	}
	ps2, err := repo.Get()
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if ps2.ID() != ps.ID() {
		t.Errorf("post-Update ID = %s, want %s", ps2.ID(), ps.ID())
	}
}

func TestSecurityRepo_GetModuleSecurity_PerModule(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, err := r.ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}

	repo := NewSecurityRepository(w)
	hits := 0
	for _, m := range mods {
		if m.Name == "System" {
			continue
		}
		ms, err := repo.GetModuleSecurity(model.ID(m.ID))
		if err != nil {
			// Some user modules may not have security; that's fine.
			continue
		}
		if ms.TypeName() != moduleSecurityTypeName {
			t.Errorf("module %s: TypeName = %q, want %q", m.Name, ms.TypeName(), moduleSecurityTypeName)
		}
		hits++
	}
	if hits == 0 {
		t.Error("no module security units across non-System modules")
	}
}

func TestSecurityRepo_UpdateModuleSecurityRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, _ := r.ListModules()
	var moduleID model.ID
	for _, m := range mods {
		if m.Name == "MyFirstModule" {
			moduleID = model.ID(m.ID)
			break
		}
	}
	if moduleID == "" {
		t.Skip("fixture missing MyFirstModule")
	}

	repo := NewSecurityRepository(w)
	ms, err := repo.GetModuleSecurity(moduleID)
	if err != nil {
		t.Fatalf("GetModuleSecurity: %v", err)
	}
	if err := repo.UpdateModuleSecurity(moduleID, ms); err != nil {
		t.Fatalf("UpdateModuleSecurity: %v", err)
	}
	ms2, err := repo.GetModuleSecurity(moduleID)
	if err != nil {
		t.Fatalf("GetModuleSecurity after Update: %v", err)
	}
	if ms2.ID() != ms.ID() {
		t.Errorf("post-Update ID = %s, want %s", ms2.ID(), ms.ID())
	}
}
