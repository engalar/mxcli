// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

func TestProjectSettingsRepo_Get_Singleton(t *testing.T) {
	w := openTestWriter(t)
	repo := NewProjectSettingsRepository(w)
	ps, err := repo.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ps == nil {
		t.Fatal("Get returned nil project settings")
	}
	if ps.ID() == "" {
		t.Error("project settings has empty ID")
	}
	if ps.TypeName() != projectSettingsTypeName {
		t.Errorf("TypeName = %q, want %q", ps.TypeName(), projectSettingsTypeName)
	}
}

func TestProjectSettingsRepo_UpdateRoundTrip(t *testing.T) {
	w := openTestWriter(t)
	repo := NewProjectSettingsRepository(w)
	ps, err := repo.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := repo.Update(ps); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Re-fetch and confirm same ID — round-trip integrity.
	ps2, err := repo.Get()
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if ps2.ID() != ps.ID() {
		t.Errorf("post-Update ID = %s, want %s", ps2.ID(), ps.ID())
	}
}

func TestModuleSettingsRepo_Get_PerModule(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	mods, err := r.ListModules()
	if err != nil {
		t.Fatalf("ListModules: %v", err)
	}

	repo := NewModuleSettingsRepository(w)
	hits := 0
	for _, m := range mods {
		// Skip System module — its settings unit has different
		// semantics and the fixture probe shows no Projects$ModuleSettings
		// for System (only the 8 user/Atlas modules).
		if m.Name == "System" {
			continue
		}
		s, err := repo.Get(model.ID(m.ID))
		if err != nil {
			t.Errorf("Get(%s): %v", m.Name, err)
			continue
		}
		if s == nil {
			t.Errorf("Get(%s): nil element", m.Name)
			continue
		}
		if s.TypeName() != moduleSettingsTypeName {
			t.Errorf("module %s: TypeName = %q, want %q", m.Name, s.TypeName(), moduleSettingsTypeName)
		}
		hits++
	}
	if hits == 0 {
		t.Error("no module settings found across all non-System modules")
	}
}

func TestModuleSettingsRepo_UpdateRoundTrip(t *testing.T) {
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

	repo := NewModuleSettingsRepository(w)
	s, err := repo.Get(moduleID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := repo.Update(moduleID, s); err != nil {
		t.Fatalf("Update: %v", err)
	}
	s2, err := repo.Get(moduleID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if s2.ID() != s.ID() {
		t.Errorf("post-Update ID = %s, want %s", s2.ID(), s.ID())
	}
}
