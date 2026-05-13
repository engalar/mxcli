// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// Agents domain has 0 fixture units (Mendix 11.9+ AgentEditorCommons
// not present in the Stage 2 fixture). These tests exercise the
// wiring (empty-list, get-not-found, create-error-paths). Round-trip
// against a real Agent unit will land with the fixture upgrade.

func TestAgentsRepo_ListByType_Empty(t *testing.T) {
	w := openTestWriter(t)
	repo := NewAgentRepository(w)
	for _, typeName := range []string{
		"Agents$Agent", "Agents$KnowledgeBase", "Agents$Model",
		"Agents$ConsumedMCPService",
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

func TestAgentsRepo_ListByType_EmptyName_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewAgentRepository(w)
	if _, err := repo.ListByType(""); err == nil {
		t.Error("ListByType(\"\"): want error, got nil")
	}
}

func TestAgentsRepo_Get_NotFound(t *testing.T) {
	w := openTestWriter(t)
	repo := NewAgentRepository(w)
	if _, err := repo.Get("nonexistent-agent-id"); err == nil {
		t.Error("Get(nonexistent): want error, got nil")
	}
}

func TestAgentsRepo_Create_NilElement_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewAgentRepository(w)
	if err := repo.Create("p", "c", nil); err == nil {
		t.Error("Create(nil): want error, got nil")
	}
}

// TestAgentsRepo_Create_EmptyID_Errors verifies the
// "no SetID on Element interface" precondition surfaces a clear error
// instead of silently inserting a unit with empty ID. Uses a fresh
// *element.Base (the embeddable that backs every gen type) with no
// ID set — element.Element is satisfied via the embedded Base.
func TestAgentsRepo_Create_EmptyID_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewAgentRepository(w)
	stub := &agentStubElement{}
	if err := repo.Create("p", "c", stub); err == nil {
		t.Error("Create(empty-ID): want error, got nil")
	}
}

// agentStubElement embeds element.Base so it satisfies element.Element
// with all methods returning zero values. ID and TypeName remain
// empty, exercising the precondition checks.
type agentStubElement struct{ element.Base }

func TestAgentsRepo_Update_NilElement_Errors(t *testing.T) {
	w := openTestWriter(t)
	repo := NewAgentRepository(w)
	if err := repo.Update(nil); err == nil {
		t.Error("Update(nil): want error, got nil")
	}
}
