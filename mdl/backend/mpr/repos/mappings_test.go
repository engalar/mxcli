// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// Mappings domain has 0 fixture units. Tests exercise the wiring
// (empty-list, get-not-found, create-error-paths). Round-trip
// against a real Mapping unit will land with a fixture upgrade.

func TestMappingsRepo_ListByType_Empty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMappingRepository(w)
	for _, typeName := range []string{
		"Mappings$ImportMapping",
		"Mappings$ExportMapping",
	} {
		got, err := repo.ListByType(typeName)
		if err != nil {
			t.Errorf("ListByType(%s): %v", typeName, err)
			continue
		}
		if len(got) != 0 {
			t.Errorf("ListByType(%s): want empty, got %d", typeName, len(got))
		}
	}
}

func TestMappingsRepo_ListByType_EmptyName_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMappingRepository(w)
	if _, err := repo.ListByType(""); err == nil {
		t.Error("ListByType(\"\"): want error, got nil")
	}
}

func TestMappingsRepo_Get_NotFound(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMappingRepository(w)
	if _, err := repo.Get("nonexistent-mapping-id"); err == nil {
		t.Error("Get(nonexistent): want error, got nil")
	}
}

func TestMappingsRepo_Create_NilElement_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMappingRepository(w)
	if err := repo.Create("p", "c", nil); err == nil {
		t.Error("Create(nil): want error, got nil")
	}
}

func TestMappingsRepo_Create_EmptyID_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMappingRepository(w)
	stub := &mappingStubElement{}
	if err := repo.Create("p", "c", stub); err == nil {
		t.Error("Create(empty-ID): want error, got nil")
	}
}

func TestMappingsRepo_Update_NilElement_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewMappingRepository(w)
	if err := repo.Update(nil); err == nil {
		t.Error("Update(nil): want error, got nil")
	}
}

// mappingStubElement embeds element.Base so it satisfies element.Element
// with all methods returning zero values.
type mappingStubElement struct{ element.Base }
