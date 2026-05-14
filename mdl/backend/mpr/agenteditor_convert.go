// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	sdkagenteditor "github.com/mendixlabs/mxcli/sdk/agenteditor"
)

// Stage 3.3.6.C1 — conversion shims between sdk/agenteditor (Stage 4
// territory; reader returns these) and mdl/types (the relocated inner
// types; backend interface + executor consumers see these).
//
// Both type families are byte-identical struct layouts (verified by
// mdl/types/agenteditor_test.go::TestAgentEditor_StructLayoutMirrorsSdk).
// These helpers do explicit field-by-field copies so no unsafe pointer
// magic is required. Once Stage 4 rewrites sdk/mpr to consume
// mdl/types directly, these helpers can be deleted.

func toTypesModel(m *sdkagenteditor.Model) *types.Model {
	if m == nil {
		return nil
	}
	out := &types.Model{
		BaseElement:   m.BaseElement,
		ContainerID:   m.ContainerID,
		Name:          m.Name,
		Documentation: m.Documentation,
		Excluded:      m.Excluded,
		ExportLevel:   m.ExportLevel,
		Type:          m.Type,
		InnerName:     m.InnerName,
		DisplayName:   m.DisplayName,
		Provider:      m.Provider,
		Environment:   m.Environment,
		DeepLinkURL:   m.DeepLinkURL,
		KeyID:         m.KeyID,
		KeyName:       m.KeyName,
		ResourceName:  m.ResourceName,
	}
	if m.Key != nil {
		out.Key = &types.ConstantRef{DocumentID: m.Key.DocumentID, QualifiedName: m.Key.QualifiedName}
	}
	return out
}

func toTypesModels(in []*sdkagenteditor.Model) []*types.Model {
	out := make([]*types.Model, len(in))
	for i, m := range in {
		out[i] = toTypesModel(m)
	}
	return out
}

func toSdkModel(m *types.Model) *sdkagenteditor.Model {
	if m == nil {
		return nil
	}
	out := &sdkagenteditor.Model{
		BaseElement:   m.BaseElement,
		ContainerID:   m.ContainerID,
		Name:          m.Name,
		Documentation: m.Documentation,
		Excluded:      m.Excluded,
		ExportLevel:   m.ExportLevel,
		Type:          m.Type,
		InnerName:     m.InnerName,
		DisplayName:   m.DisplayName,
		Provider:      m.Provider,
		Environment:   m.Environment,
		DeepLinkURL:   m.DeepLinkURL,
		KeyID:         m.KeyID,
		KeyName:       m.KeyName,
		ResourceName:  m.ResourceName,
	}
	if m.Key != nil {
		out.Key = &sdkagenteditor.ConstantRef{DocumentID: m.Key.DocumentID, QualifiedName: m.Key.QualifiedName}
	}
	return out
}

func toTypesKnowledgeBase(k *sdkagenteditor.KnowledgeBase) *types.KnowledgeBase {
	if k == nil {
		return nil
	}
	out := &types.KnowledgeBase{
		BaseElement:      k.BaseElement,
		ContainerID:      k.ContainerID,
		Name:             k.Name,
		Documentation:    k.Documentation,
		Excluded:         k.Excluded,
		ExportLevel:      k.ExportLevel,
		Provider:         k.Provider,
		Environment:      k.Environment,
		DeepLinkURL:      k.DeepLinkURL,
		KeyID:            k.KeyID,
		KeyName:          k.KeyName,
		ModelDisplayName: k.ModelDisplayName,
		ModelName:        k.ModelName,
	}
	if k.Key != nil {
		out.Key = &types.ConstantRef{DocumentID: k.Key.DocumentID, QualifiedName: k.Key.QualifiedName}
	}
	return out
}

func toTypesKnowledgeBases(in []*sdkagenteditor.KnowledgeBase) []*types.KnowledgeBase {
	out := make([]*types.KnowledgeBase, len(in))
	for i, k := range in {
		out[i] = toTypesKnowledgeBase(k)
	}
	return out
}

func toSdkKnowledgeBase(k *types.KnowledgeBase) *sdkagenteditor.KnowledgeBase {
	if k == nil {
		return nil
	}
	out := &sdkagenteditor.KnowledgeBase{
		BaseElement:      k.BaseElement,
		ContainerID:      k.ContainerID,
		Name:             k.Name,
		Documentation:    k.Documentation,
		Excluded:         k.Excluded,
		ExportLevel:      k.ExportLevel,
		Provider:         k.Provider,
		Environment:      k.Environment,
		DeepLinkURL:      k.DeepLinkURL,
		KeyID:            k.KeyID,
		KeyName:          k.KeyName,
		ModelDisplayName: k.ModelDisplayName,
		ModelName:        k.ModelName,
	}
	if k.Key != nil {
		out.Key = &sdkagenteditor.ConstantRef{DocumentID: k.Key.DocumentID, QualifiedName: k.Key.QualifiedName}
	}
	return out
}

func toTypesConsumedMCPService(c *sdkagenteditor.ConsumedMCPService) *types.ConsumedMCPService {
	if c == nil {
		return nil
	}
	return &types.ConsumedMCPService{
		BaseElement:              c.BaseElement,
		ContainerID:              c.ContainerID,
		Name:                     c.Name,
		Documentation:            c.Documentation,
		Excluded:                 c.Excluded,
		ExportLevel:              c.ExportLevel,
		ProtocolVersion:          c.ProtocolVersion,
		Version:                  c.Version,
		InnerDocumentation:       c.InnerDocumentation,
		ConnectionTimeoutSeconds: c.ConnectionTimeoutSeconds,
	}
}

func toTypesConsumedMCPServices(in []*sdkagenteditor.ConsumedMCPService) []*types.ConsumedMCPService {
	out := make([]*types.ConsumedMCPService, len(in))
	for i, c := range in {
		out[i] = toTypesConsumedMCPService(c)
	}
	return out
}

func toSdkConsumedMCPService(c *types.ConsumedMCPService) *sdkagenteditor.ConsumedMCPService {
	if c == nil {
		return nil
	}
	return &sdkagenteditor.ConsumedMCPService{
		BaseElement:              c.BaseElement,
		ContainerID:              c.ContainerID,
		Name:                     c.Name,
		Documentation:            c.Documentation,
		Excluded:                 c.Excluded,
		ExportLevel:              c.ExportLevel,
		ProtocolVersion:          c.ProtocolVersion,
		Version:                  c.Version,
		InnerDocumentation:       c.InnerDocumentation,
		ConnectionTimeoutSeconds: c.ConnectionTimeoutSeconds,
	}
}

func toTypesAgent(a *sdkagenteditor.Agent) *types.Agent {
	if a == nil {
		return nil
	}
	out := &types.Agent{
		BaseElement:   a.BaseElement,
		ContainerID:   a.ContainerID,
		Name:          a.Name,
		Documentation: a.Documentation,
		Excluded:      a.Excluded,
		ExportLevel:   a.ExportLevel,
		Description:   a.Description,
		SystemPrompt:  a.SystemPrompt,
		UserPrompt:    a.UserPrompt,
		UsageType:     a.UsageType,
		ToolChoice:    a.ToolChoice,
		MaxTokens:     a.MaxTokens,
		Temperature:   a.Temperature,
		TopP:          a.TopP,
	}
	if a.Variables != nil {
		out.Variables = make([]types.AgentVar, len(a.Variables))
		for i, v := range a.Variables {
			out.Variables[i] = types.AgentVar(v)
		}
	}
	if a.Tools != nil {
		out.Tools = make([]types.AgentTool, len(a.Tools))
		for i, t := range a.Tools {
			out.Tools[i] = types.AgentTool{
				ID: t.ID, Name: t.Name, Description: t.Description,
				Enabled: t.Enabled, ToolType: t.ToolType,
			}
			if t.Document != nil {
				out.Tools[i].Document = &types.DocRef{DocumentID: t.Document.DocumentID, QualifiedName: t.Document.QualifiedName}
			}
		}
	}
	if a.KBTools != nil {
		out.KBTools = make([]types.AgentKBTool, len(a.KBTools))
		for i, t := range a.KBTools {
			out.KBTools[i] = types.AgentKBTool{
				ID: t.ID, Name: t.Name, Description: t.Description,
				Enabled: t.Enabled, ToolType: t.ToolType,
				CollectionIdentifier: t.CollectionIdentifier, MaxResults: t.MaxResults,
			}
			if t.Document != nil {
				out.KBTools[i].Document = &types.DocRef{DocumentID: t.Document.DocumentID, QualifiedName: t.Document.QualifiedName}
			}
		}
	}
	if a.Model != nil {
		out.Model = &types.DocRef{DocumentID: a.Model.DocumentID, QualifiedName: a.Model.QualifiedName}
	}
	if a.Entity != nil {
		out.Entity = &types.DocRef{DocumentID: a.Entity.DocumentID, QualifiedName: a.Entity.QualifiedName}
	}
	return out
}

func toTypesAgents(in []*sdkagenteditor.Agent) []*types.Agent {
	out := make([]*types.Agent, len(in))
	for i, a := range in {
		out[i] = toTypesAgent(a)
	}
	return out
}

func toSdkAgent(a *types.Agent) *sdkagenteditor.Agent {
	if a == nil {
		return nil
	}
	out := &sdkagenteditor.Agent{
		BaseElement:   a.BaseElement,
		ContainerID:   a.ContainerID,
		Name:          a.Name,
		Documentation: a.Documentation,
		Excluded:      a.Excluded,
		ExportLevel:   a.ExportLevel,
		Description:   a.Description,
		SystemPrompt:  a.SystemPrompt,
		UserPrompt:    a.UserPrompt,
		UsageType:     a.UsageType,
		ToolChoice:    a.ToolChoice,
		MaxTokens:     a.MaxTokens,
		Temperature:   a.Temperature,
		TopP:          a.TopP,
	}
	if a.Variables != nil {
		out.Variables = make([]sdkagenteditor.AgentVar, len(a.Variables))
		for i, v := range a.Variables {
			out.Variables[i] = sdkagenteditor.AgentVar(v)
		}
	}
	if a.Tools != nil {
		out.Tools = make([]sdkagenteditor.AgentTool, len(a.Tools))
		for i, t := range a.Tools {
			out.Tools[i] = sdkagenteditor.AgentTool{
				ID: t.ID, Name: t.Name, Description: t.Description,
				Enabled: t.Enabled, ToolType: t.ToolType,
			}
			if t.Document != nil {
				out.Tools[i].Document = &sdkagenteditor.DocRef{DocumentID: t.Document.DocumentID, QualifiedName: t.Document.QualifiedName}
			}
		}
	}
	if a.KBTools != nil {
		out.KBTools = make([]sdkagenteditor.AgentKBTool, len(a.KBTools))
		for i, t := range a.KBTools {
			out.KBTools[i] = sdkagenteditor.AgentKBTool{
				ID: t.ID, Name: t.Name, Description: t.Description,
				Enabled: t.Enabled, ToolType: t.ToolType,
				CollectionIdentifier: t.CollectionIdentifier, MaxResults: t.MaxResults,
			}
			if t.Document != nil {
				out.KBTools[i].Document = &sdkagenteditor.DocRef{DocumentID: t.Document.DocumentID, QualifiedName: t.Document.QualifiedName}
			}
		}
	}
	if a.Model != nil {
		out.Model = &sdkagenteditor.DocRef{DocumentID: a.Model.DocumentID, QualifiedName: a.Model.QualifiedName}
	}
	if a.Entity != nil {
		out.Entity = &sdkagenteditor.DocRef{DocumentID: a.Entity.DocumentID, QualifiedName: a.Entity.QualifiedName}
	}
	return out
}
