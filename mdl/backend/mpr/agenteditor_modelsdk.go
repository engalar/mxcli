// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// All Create / Update methods for the four agent-editor document types
// (Model, KnowledgeBase, ConsumedMCPService, Agent) build the canonical
// CustomBlobDocument BSON via sdk/mpr.SerializeAgentEditor* helpers, then
// route writes through msdkWriter.InsertUnit / writeUnitContents — bypassing
// sdk/mpr's updateTransactionID() that fails on hard-linked MPR files
// (SQLITE_READONLY_DBMOVED 1544). The unit type for all four is
// "CustomBlobDocuments$CustomBlobDocument", containment "Documents".
//
// Stage 3.3.6.C2: helpers now take *mdl/types.* inputs (the relocated
// inner JSON types). The toSdk* converters in agenteditor_convert.go
// bridge to the still-sdk-typed mpr.SerializeAgentEditor* writers,
// which live in sdk/mpr (Stage 4 territory; not touched here).

const (
	customBlobUnitType        = "CustomBlobDocuments$CustomBlobDocument"
	customBlobContainmentName = "Documents"
)

// ── Agent Editor Model ────────────────────────────────────────────────────

func (b *MprBackend) createAgentEditorModelViaModelsdk(m *types.Model) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if m == nil {
		return fmt.Errorf("model is nil")
	}
	if m.Name == "" {
		return fmt.Errorf("model name is required")
	}
	if m.ContainerID == "" {
		return fmt.Errorf("model container ID is required")
	}
	if m.ID == "" {
		m.ID = model.ID(modelsdkmpr.GenerateID())
	}
	contents, err := mpr.SerializeAgentEditorModel(toSdkModel(m))
	if err != nil {
		return fmt.Errorf("serialize agent editor model: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(m.ID),
		string(m.ContainerID),
		customBlobContainmentName,
		customBlobUnitType,
		contents,
	)
}

func (b *MprBackend) updateAgentEditorModelViaModelsdk(m *types.Model) error {
	if m == nil {
		return fmt.Errorf("model is nil")
	}
	contents, err := mpr.SerializeAgentEditorModel(toSdkModel(m))
	if err != nil {
		return fmt.Errorf("serialize agent editor model: %w", err)
	}
	return b.writeUnitContents(m.ID, contents)
}

// ── Agent Editor Knowledge Base ───────────────────────────────────────────

func (b *MprBackend) createAgentEditorKnowledgeBaseViaModelsdk(k *types.KnowledgeBase) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if k == nil {
		return fmt.Errorf("knowledge base is nil")
	}
	if k.Name == "" {
		return fmt.Errorf("knowledge base name is required")
	}
	if k.ContainerID == "" {
		return fmt.Errorf("knowledge base container ID is required")
	}
	if k.ID == "" {
		k.ID = model.ID(modelsdkmpr.GenerateID())
	}
	contents, err := mpr.SerializeAgentEditorKnowledgeBase(toSdkKnowledgeBase(k))
	if err != nil {
		return fmt.Errorf("serialize agent editor knowledge base: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(k.ID),
		string(k.ContainerID),
		customBlobContainmentName,
		customBlobUnitType,
		contents,
	)
}

func (b *MprBackend) updateAgentEditorKnowledgeBaseViaModelsdk(k *types.KnowledgeBase) error {
	if k == nil {
		return fmt.Errorf("knowledge base is nil")
	}
	contents, err := mpr.SerializeAgentEditorKnowledgeBase(toSdkKnowledgeBase(k))
	if err != nil {
		return fmt.Errorf("serialize agent editor knowledge base: %w", err)
	}
	return b.writeUnitContents(k.ID, contents)
}

// ── Agent Editor Consumed MCP Service ─────────────────────────────────────

func (b *MprBackend) createAgentEditorConsumedMCPServiceViaModelsdk(c *types.ConsumedMCPService) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
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
	contents, err := mpr.SerializeAgentEditorConsumedMCPService(toSdkConsumedMCPService(c))
	if err != nil {
		return fmt.Errorf("serialize agent editor consumed MCP service: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(c.ID),
		string(c.ContainerID),
		customBlobContainmentName,
		customBlobUnitType,
		contents,
	)
}

func (b *MprBackend) updateAgentEditorConsumedMCPServiceViaModelsdk(c *types.ConsumedMCPService) error {
	if c == nil {
		return fmt.Errorf("consumed MCP service is nil")
	}
	contents, err := mpr.SerializeAgentEditorConsumedMCPService(toSdkConsumedMCPService(c))
	if err != nil {
		return fmt.Errorf("serialize agent editor consumed MCP service: %w", err)
	}
	return b.writeUnitContents(c.ID, contents)
}

// ── Agent Editor Agent ────────────────────────────────────────────────────

func (b *MprBackend) createAgentEditorAgentViaModelsdk(a *types.Agent) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
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
	contents, err := mpr.SerializeAgentEditorAgent(toSdkAgent(a))
	if err != nil {
		return fmt.Errorf("serialize agent editor agent: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(a.ID),
		string(a.ContainerID),
		customBlobContainmentName,
		customBlobUnitType,
		contents,
	)
}

func (b *MprBackend) updateAgentEditorAgentViaModelsdk(a *types.Agent) error {
	if a == nil {
		return fmt.Errorf("agent is nil")
	}
	contents, err := mpr.SerializeAgentEditorAgent(toSdkAgent(a))
	if err != nil {
		return fmt.Errorf("serialize agent editor agent: %w", err)
	}
	return b.writeUnitContents(a.ID, contents)
}
