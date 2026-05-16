// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// All Create / Update methods for the four agent-editor document types
// (Model, KnowledgeBase, ConsumedMCPService, Agent) use gen-native
// CustomBlobDocument + the agent repo — no sdk/mpr import needed.
// JSON content encoding is in agenteditor_serialize.go.

const (
	customBlobUnitType        = "CustomBlobDocuments$CustomBlobDocument"
	customBlobContainmentName = "Documents"
)

// ── Agent Editor Model ────────────────────────────────────────────────────

func (b *MprBackend) createAgentEditorModelViaModelsdk(m *types.Model) error {
	if m == nil {
		return fmt.Errorf("model is nil")
	}
	if m.Name == "" {
		return fmt.Errorf("model name is required")
	}
	if m.ContainerID == "" {
		return fmt.Errorf("model container ID is required")
	}
	if m.Provider == "" {
		m.Provider = "MxCloudGenAI"
	}
	if m.ID == "" {
		m.ID = model.ID(modelsdkmpr.GenerateID())
	}
	contentsJSON, err := encodeModelContentsJSON(m)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(m.ID), m.Name, m.Documentation, m.Excluded, m.ExportLevel,
		types.CustomTypeModel, types.ReadableModel, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Create(string(m.ContainerID), customBlobContainmentName, doc)
}

func (b *MprBackend) updateAgentEditorModelViaModelsdk(m *types.Model) error {
	if m == nil {
		return fmt.Errorf("model is nil")
	}
	contentsJSON, err := encodeModelContentsJSON(m)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(m.ID), m.Name, m.Documentation, m.Excluded, m.ExportLevel,
		types.CustomTypeModel, types.ReadableModel, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Update(doc)
}

// ── Agent Editor Knowledge Base ───────────────────────────────────────────

func (b *MprBackend) createAgentEditorKnowledgeBaseViaModelsdk(k *types.KnowledgeBase) error {
	if k == nil {
		return fmt.Errorf("knowledge base is nil")
	}
	if k.Name == "" {
		return fmt.Errorf("knowledge base name is required")
	}
	if k.ContainerID == "" {
		return fmt.Errorf("knowledge base container ID is required")
	}
	if k.Provider == "" {
		k.Provider = "MxCloudGenAI"
	}
	if k.ID == "" {
		k.ID = model.ID(modelsdkmpr.GenerateID())
	}
	contentsJSON, err := encodeKnowledgeBaseContentsJSON(k)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(k.ID), k.Name, k.Documentation, k.Excluded, k.ExportLevel,
		types.CustomTypeKnowledgeBase, types.ReadableKnowledgeBase, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Create(string(k.ContainerID), customBlobContainmentName, doc)
}

func (b *MprBackend) updateAgentEditorKnowledgeBaseViaModelsdk(k *types.KnowledgeBase) error {
	if k == nil {
		return fmt.Errorf("knowledge base is nil")
	}
	contentsJSON, err := encodeKnowledgeBaseContentsJSON(k)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(k.ID), k.Name, k.Documentation, k.Excluded, k.ExportLevel,
		types.CustomTypeKnowledgeBase, types.ReadableKnowledgeBase, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Update(doc)
}

// ── Agent Editor Consumed MCP Service ─────────────────────────────────────

func (b *MprBackend) createAgentEditorConsumedMCPServiceViaModelsdk(c *types.ConsumedMCPService) error {
	if c == nil {
		return fmt.Errorf("consumed MCP service is nil")
	}
	if c.Name == "" {
		return fmt.Errorf("consumed MCP service name is required")
	}
	if c.ContainerID == "" {
		return fmt.Errorf("consumed MCP service container ID is required")
	}
	if c.ID == "" {
		c.ID = model.ID(modelsdkmpr.GenerateID())
	}
	contentsJSON, err := encodeMCPServiceContentsJSON(c)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(c.ID), c.Name, c.Documentation, c.Excluded, c.ExportLevel,
		types.CustomTypeConsumedMCPService, types.ReadableConsumedMCPService, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Create(string(c.ContainerID), customBlobContainmentName, doc)
}

func (b *MprBackend) updateAgentEditorConsumedMCPServiceViaModelsdk(c *types.ConsumedMCPService) error {
	if c == nil {
		return fmt.Errorf("consumed MCP service is nil")
	}
	contentsJSON, err := encodeMCPServiceContentsJSON(c)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(c.ID), c.Name, c.Documentation, c.Excluded, c.ExportLevel,
		types.CustomTypeConsumedMCPService, types.ReadableConsumedMCPService, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Update(doc)
}

// ── Agent Editor Agent ────────────────────────────────────────────────────

func (b *MprBackend) createAgentEditorAgentViaModelsdk(a *types.Agent) error {
	if a == nil {
		return fmt.Errorf("agent is nil")
	}
	if a.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if a.ContainerID == "" {
		return fmt.Errorf("agent container ID is required")
	}
	if a.ID == "" {
		a.ID = model.ID(modelsdkmpr.GenerateID())
	}
	contentsJSON, err := encodeAgentContentsJSON(a)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(a.ID), a.Name, a.Documentation, a.Excluded, a.ExportLevel,
		types.CustomTypeAgent, types.ReadableAgent, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Create(string(a.ContainerID), customBlobContainmentName, doc)
}

func (b *MprBackend) updateAgentEditorAgentViaModelsdk(a *types.Agent) error {
	if a == nil {
		return fmt.Errorf("agent is nil")
	}
	contentsJSON, err := encodeAgentContentsJSON(a)
	if err != nil {
		return err
	}
	doc := newAgentBlobDoc(string(a.ID), a.Name, a.Documentation, a.Excluded, a.ExportLevel,
		types.CustomTypeAgent, types.ReadableAgent, contentsJSON)
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return mprrepos.NewAgentRepository(w).Update(doc)
}
