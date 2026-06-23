// SPDX-License-Identifier: Apache-2.0

// Package executor - CREATE/DROP handlers for Consumed MCP Service,
// Knowledge Base, and Agent agent-editor documents.
package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

func execCreateConsumedMCPServiceFn(ctx context.Context, s *ast.CreateConsumedMCPServiceStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	existing := findAgentEditorConsumedMCPService(ectx, s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("consumed mcp service", s.Name.String())
	}
	module, err := findOrCreateModule(ectx, s.Name.Module)
	if err != nil {
		return err
	}
	c := &types.ConsumedMCPService{
		ContainerID:              module.ID,
		Name:                     s.Name.Name,
		Documentation:            s.OuterDocumentation,
		ProtocolVersion:          s.ProtocolVersion,
		Version:                  s.Version,
		InnerDocumentation:       s.InnerDocumentation,
		ConnectionTimeoutSeconds: s.ConnectionTimeoutSeconds,
	}
	if existing != nil {
		c.ID = existing.ID
		if err := deps.AgentEditorOperator.UpdateAgentEditorConsumedMCPService(c); err != nil {
			return mdlerrors.NewBackend("update consumed mcp service", err)
		}
		invalidateHierarchy(ectx)
		fmt.Fprintf(deps.Output, "Modified consumed mcp service: %s\n", s.Name)
		return nil
	}
	if err := deps.AgentEditorOperator.CreateAgentEditorConsumedMCPService(c); err != nil {
		return mdlerrors.NewBackend("create consumed mcp service", err)
	}
	invalidateHierarchy(ectx)
	fmt.Fprintf(deps.Output, "Created consumed mcp service: %s\n", s.Name)
	return nil
}

func execDropConsumedMCPServiceFn(ctx context.Context, s *ast.DropConsumedMCPServiceStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	c := findAgentEditorConsumedMCPService(ectx, s.Name.Module, s.Name.Name)
	if c == nil {
		return mdlerrors.NewNotFound("consumed mcp service", s.Name.String())
	}
	if err := deps.AgentEditorOperator.DeleteAgentEditorConsumedMCPService(string(c.ID)); err != nil {
		return mdlerrors.NewBackend("delete consumed mcp service", err)
	}
	fmt.Fprintf(deps.Output, "Dropped consumed mcp service: %s\n", s.Name)
	return nil
}

func execCreateKnowledgeBaseFn(ctx context.Context, s *ast.CreateKnowledgeBaseStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	existing := findAgentEditorKnowledgeBase(ectx, s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("knowledge base", s.Name.String())
	}
	module, err := findOrCreateModule(ectx, s.Name.Module)
	if err != nil {
		return err
	}
	var keyRef *types.ConstantRef
	if s.Key != nil {
		keyRef, err = resolveConstantRef(ectx, *s.Key)
		if err != nil {
			return fmt.Errorf("create knowledge base %s: %w", s.Name, err)
		}
	}
	provider := s.Provider
	if provider == "" {
		provider = "MxCloudGenAI"
	}
	k := &types.KnowledgeBase{
		ContainerID:      module.ID,
		Name:             s.Name.Name,
		Documentation:    s.Documentation,
		Provider:         provider,
		Key:              keyRef,
		ModelDisplayName: s.ModelDisplayName,
		ModelName:        s.ModelName,
		KeyName:          s.KeyName,
		KeyID:            s.KeyID,
		Environment:      s.Environment,
		DeepLinkURL:      s.DeepLinkURL,
	}
	if existing != nil {
		k.ID = existing.ID
		if err := deps.AgentEditorOperator.UpdateAgentEditorKnowledgeBase(k); err != nil {
			return mdlerrors.NewBackend("update knowledge base", err)
		}
		invalidateHierarchy(ectx)
		fmt.Fprintf(deps.Output, "Modified knowledge base: %s\n", s.Name)
		return nil
	}
	if err := deps.AgentEditorOperator.CreateAgentEditorKnowledgeBase(k); err != nil {
		return mdlerrors.NewBackend("create knowledge base", err)
	}
	invalidateHierarchy(ectx)
	fmt.Fprintf(deps.Output, "Created knowledge base: %s\n", s.Name)
	return nil
}

func execDropKnowledgeBaseFn(ctx context.Context, s *ast.DropKnowledgeBaseStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	k := findAgentEditorKnowledgeBase(ectx, s.Name.Module, s.Name.Name)
	if k == nil {
		return mdlerrors.NewNotFound("knowledge base", s.Name.String())
	}
	if err := deps.AgentEditorOperator.DeleteAgentEditorKnowledgeBase(string(k.ID)); err != nil {
		return mdlerrors.NewBackend("delete knowledge base", err)
	}
	fmt.Fprintf(deps.Output, "Dropped knowledge base: %s\n", s.Name)
	return nil
}

func execCreateAgentFn(ctx context.Context, s *ast.CreateAgentStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	existingAgent := findAgentEditorAgent(ectx, s.Name.Module, s.Name.Name)
	if existingAgent != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("agent", s.Name.String())
	}
	module, err := findOrCreateModule(ectx, s.Name.Module)
	if err != nil {
		return err
	}
	a := &types.Agent{
		ContainerID:   module.ID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
		Description:   s.Description,
		SystemPrompt:  s.SystemPrompt,
		UserPrompt:    s.UserPrompt,
		UsageType:     s.UsageType,
		MaxTokens:     s.MaxTokens,
		ToolChoice:    s.ToolChoice,
		Temperature:   s.Temperature,
		TopP:          s.TopP,
	}
	if s.Model != nil {
		m := findAgentEditorModel(ectx, s.Model.Module, s.Model.Name)
		if m == nil {
			return fmt.Errorf("create agent %s: model not found: %s", s.Name, s.Model)
		}
		a.Model = &types.DocRef{
			DocumentID:    string(m.ID),
			QualifiedName: s.Model.String(),
		}
	}
	if s.Entity != nil {
		a.Entity = &types.DocRef{
			QualifiedName: s.Entity.String(),
		}
	}
	for _, v := range s.Variables {
		a.Variables = append(a.Variables, types.AgentVar{
			Key:                 v.Key,
			IsAttributeInEntity: v.IsAttributeInEntity,
		})
	}
	for _, td := range s.Tools {
		at := types.AgentTool{
			Name:        td.Name,
			Description: td.Description,
			Enabled:     td.Enabled,
			ToolType:    td.ToolType,
		}
		if td.Document != nil && td.ToolType == "mcp" {
			svc := findAgentEditorConsumedMCPService(ectx, td.Document.Module, td.Document.Name)
			if svc == nil {
				return fmt.Errorf("create agent %s: consumed mcp service not found: %s", s.Name, td.Document)
			}
			at.Document = &types.DocRef{
				DocumentID:    string(svc.ID),
				QualifiedName: td.Document.String(),
			}
		}
		a.Tools = append(a.Tools, at)
	}
	for _, kbd := range s.KBTools {
		akt := types.AgentKBTool{
			Name:                 kbd.Name,
			Description:          kbd.Description,
			Enabled:              kbd.Enabled,
			CollectionIdentifier: kbd.Collection,
			MaxResults:           kbd.MaxResults,
		}
		if kbd.Source != nil {
			kb := findAgentEditorKnowledgeBase(ectx, kbd.Source.Module, kbd.Source.Name)
			if kb == nil {
				return fmt.Errorf("create agent %s: knowledge base not found: %s", s.Name, kbd.Source)
			}
			akt.Document = &types.DocRef{
				DocumentID:    string(kb.ID),
				QualifiedName: kbd.Source.String(),
			}
		}
		a.KBTools = append(a.KBTools, akt)
	}
	if existingAgent != nil {
		a.ID = existingAgent.ID
		if err := deps.AgentEditorOperator.UpdateAgentEditorAgent(a); err != nil {
			return mdlerrors.NewBackend("update agent", err)
		}
		invalidateHierarchy(ectx)
		fmt.Fprintf(deps.Output, "Modified agent: %s\n", s.Name)
		return nil
	}
	if err := deps.AgentEditorOperator.CreateAgentEditorAgent(a); err != nil {
		return mdlerrors.NewBackend("create agent", err)
	}
	invalidateHierarchy(ectx)
	fmt.Fprintf(deps.Output, "Created agent: %s\n", s.Name)
	return nil
}

func execDropAgentFn(ctx context.Context, s *ast.DropAgentStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	ectx := phase3d2bNewExecContext(ctx, deps)
	a := findAgentEditorAgent(ectx, s.Name.Module, s.Name.Name)
	if a == nil {
		return mdlerrors.NewNotFound("agent", s.Name.String())
	}
	if err := deps.AgentEditorOperator.DeleteAgentEditorAgent(string(a.ID)); err != nil {
		return mdlerrors.NewBackend("delete agent", err)
	}
	fmt.Fprintf(deps.Output, "Dropped agent: %s\n", s.Name)
	return nil
}

func execCreateConsumedMCPService(ctx *ExecContext, s *ast.CreateConsumedMCPServiceStmt) error {
	return execCreateConsumedMCPServiceFn(ctx, s, execContextToDeps(ctx))
}

func execDropConsumedMCPService(ctx *ExecContext, s *ast.DropConsumedMCPServiceStmt) error {
	return execDropConsumedMCPServiceFn(ctx, s, execContextToDeps(ctx))
}

func execCreateKnowledgeBase(ctx *ExecContext, s *ast.CreateKnowledgeBaseStmt) error {
	return execCreateKnowledgeBaseFn(ctx, s, execContextToDeps(ctx))
}

func execDropKnowledgeBase(ctx *ExecContext, s *ast.DropKnowledgeBaseStmt) error {
	return execDropKnowledgeBaseFn(ctx, s, execContextToDeps(ctx))
}

func execCreateAgent(ctx *ExecContext, s *ast.CreateAgentStmt) error {
	return execCreateAgentFn(ctx, s, execContextToDeps(ctx))
}

func execDropAgent(ctx *ExecContext, s *ast.DropAgentStmt) error {
	return execDropAgentFn(ctx, s, execContextToDeps(ctx))
}
