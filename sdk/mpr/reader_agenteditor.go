// SPDX-License-Identifier: Apache-2.0

// Package mpr - Reader methods for agent-editor CustomBlobDocuments.
//
// Covers the four document types created by the Studio Pro Agent Editor
// extension: Agent, Model, Knowledge Base, Consumed MCP Service. Each
// shares the outer CustomBlobDocument BSON wrapper and is discriminated
// by CustomDocumentType. This file currently implements Model only; the
// other three will follow the same pattern.
package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// ListAgentEditorModels returns all agent-editor Model documents in the
// project (CustomDocumentType == "types.model").
func (r *Reader) ListAgentEditorModels() ([]*types.Model, error) {
	units, err := r.listUnitsByType(customBlobDocType)
	if err != nil {
		return nil, err
	}

	var result []*types.Model
	for _, u := range units {
		wrap, err := parseCustomBlobWrapper(u.Contents)
		if err != nil {
			// Skip units we can't decode; log to error list if useful later.
			continue
		}
		if wrap.CustomDocumentType != types.CustomTypeModel {
			continue
		}
		m, err := r.parseAgentEditorModel(u.ID, u.ContainerID, u.Contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent-editor model %s: %w", u.ID, err)
		}
		result = append(result, m)
	}
	return result, nil
}

// ListAgentEditorKnowledgeBases returns all agent-editor Knowledge Base
// documents in the project (CustomDocumentType == "types.knowledgebase").
func (r *Reader) ListAgentEditorKnowledgeBases() ([]*types.KnowledgeBase, error) {
	units, err := r.listUnitsByType(customBlobDocType)
	if err != nil {
		return nil, err
	}

	var result []*types.KnowledgeBase
	for _, u := range units {
		wrap, err := parseCustomBlobWrapper(u.Contents)
		if err != nil {
			continue
		}
		if wrap.CustomDocumentType != types.CustomTypeKnowledgeBase {
			continue
		}
		kb, err := r.parseAgentEditorKnowledgeBase(u.ID, u.ContainerID, u.Contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent-editor knowledge base %s: %w", u.ID, err)
		}
		result = append(result, kb)
	}
	return result, nil
}

// ListAgentEditorConsumedMCPServices returns all agent-editor Consumed MCP
// Service documents in the project (CustomDocumentType ==
// "types.consumedMCPService").
func (r *Reader) ListAgentEditorConsumedMCPServices() ([]*types.ConsumedMCPService, error) {
	units, err := r.listUnitsByType(customBlobDocType)
	if err != nil {
		return nil, err
	}

	var result []*types.ConsumedMCPService
	for _, u := range units {
		wrap, err := parseCustomBlobWrapper(u.Contents)
		if err != nil {
			continue
		}
		if wrap.CustomDocumentType != types.CustomTypeConsumedMCPService {
			continue
		}
		c, err := r.parseAgentEditorConsumedMCPService(u.ID, u.ContainerID, u.Contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent-editor consumed MCP service %s: %w", u.ID, err)
		}
		result = append(result, c)
	}
	return result, nil
}

// ListAgentEditorAgents returns all agent-editor Agent documents in the
// project (CustomDocumentType == "types.agent").
func (r *Reader) ListAgentEditorAgents() ([]*types.Agent, error) {
	units, err := r.listUnitsByType(customBlobDocType)
	if err != nil {
		return nil, err
	}

	var result []*types.Agent
	for _, u := range units {
		wrap, err := parseCustomBlobWrapper(u.Contents)
		if err != nil {
			continue
		}
		if wrap.CustomDocumentType != types.CustomTypeAgent {
			continue
		}
		a, err := r.parseAgentEditorAgent(u.ID, u.ContainerID, u.Contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent-editor agent %s: %w", u.ID, err)
		}
		result = append(result, a)
	}
	return result, nil
}
