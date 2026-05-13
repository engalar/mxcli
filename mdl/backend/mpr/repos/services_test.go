// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// Services domain has 0 fixture units. Tests exercise wiring only.

func TestServicesRepo_ListByType_Empty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewServiceRepository(w)
	for _, typeName := range []string{
		"Services$ConsumedODataService",
		"Services$ConsumedRESTService",
		"Services$ConsumedAppService",
		"Services$PublishedODataService",
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

func TestServicesRepo_ListByType_EmptyName_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewServiceRepository(w)
	if _, err := repo.ListByType(""); err == nil {
		t.Error("ListByType(\"\"): want error, got nil")
	}
}

func TestServicesRepo_Get_NotFound(t *testing.T) {
	w := openTestWriter(t)
	repo := NewServiceRepository(w)
	if _, err := repo.Get("nonexistent-service-id"); err == nil {
		t.Error("Get(nonexistent): want error, got nil")
	}
}

func TestServicesRepo_Create_NilElement_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewServiceRepository(w)
	if err := repo.Create("p", "c", nil); err == nil {
		t.Error("Create(nil): want error, got nil")
	}
}

func TestServicesRepo_Create_EmptyID_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewServiceRepository(w)
	stub := &serviceStubElement{}
	if err := repo.Create("p", "c", stub); err == nil {
		t.Error("Create(empty-ID): want error, got nil")
	}
}

func TestServicesRepo_Update_NilElement_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewServiceRepository(w)
	if err := repo.Update(nil); err == nil {
		t.Error("Update(nil): want error, got nil")
	}
}

type serviceStubElement struct{ element.Base }
