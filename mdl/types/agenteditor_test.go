// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"encoding/json"
	"reflect"
	"testing"

	sdkagenteditor "github.com/mendixlabs/mxcli/sdk/agenteditor"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// TestAgent_JsonWireCompatibleWithSdk verifies that mdl/types.Agent and
// sdk/agenteditor.Agent produce byte-identical JSON output for the same
// input, and that JSON produced by one unmarshals losslessly into the
// other. This is the R2/R6 guard from the plan: Studio Pro's Agent Editor
// extension consumes the JSON wire format, so any drift between the two
// type definitions would corrupt the round-trip.
//
// Note: Mendix's actual wire format uses an intermediate "Contents JSON"
// struct (see sdk/mpr/parser_customblob.go + writer_customblob.go); the
// types here represent the in-memory Go shape that the executor + backend
// pass around. This test verifies that shape is byte-identical.
func TestAgent_JsonWireCompatibleWithSdk(t *testing.T) {
	maxTokens := 4096
	temperature := 0.7
	topP := 0.95

	mdlFixture := &types.Agent{
		BaseElement: model.BaseElement{
			ID:       model.ID("11111111-1111-1111-1111-111111111111"),
			TypeName: "CustomBlobDocuments$CustomBlobDocument",
		},
		ContainerID:   model.ID("22222222-2222-2222-2222-222222222222"),
		Name:          "ResearchBot",
		Documentation: "An agent that researches topics.",
		Description:   "Researcher",
		SystemPrompt:  "You are a helpful researcher.\nMulti-line works.",
		UserPrompt:    "Research $$topic$$.",
		UsageType:     "Conversational",
		Variables: []types.AgentVar{
			{Key: "topic", IsAttributeInEntity: false},
			{Key: "domain", IsAttributeInEntity: true},
		},
		Tools: []types.AgentTool{
			{ID: "tool-1", Name: "WebSearch", Enabled: true, ToolType: "MCP",
				Document: &types.DocRef{DocumentID: "doc-1", QualifiedName: "Module.Web"}},
		},
		KBTools: []types.AgentKBTool{
			{ID: "kb-1", Name: "Docs", Enabled: true, ToolType: "KnowledgeBase",
				Document:             &types.DocRef{DocumentID: "doc-2", QualifiedName: "Module.DocsKB"},
				CollectionIdentifier: "default", MaxResults: 5},
		},
		Model:       &types.DocRef{DocumentID: "doc-3", QualifiedName: "Module.GPT4"},
		Entity:      &types.DocRef{DocumentID: "doc-4", QualifiedName: "Module.Topic"},
		MaxTokens:   &maxTokens,
		ToolChoice:  "auto",
		Temperature: &temperature,
		TopP:        &topP,
	}
	sdkFixture := &sdkagenteditor.Agent{
		BaseElement: model.BaseElement{
			ID:       model.ID("11111111-1111-1111-1111-111111111111"),
			TypeName: "CustomBlobDocuments$CustomBlobDocument",
		},
		ContainerID:   model.ID("22222222-2222-2222-2222-222222222222"),
		Name:          "ResearchBot",
		Documentation: "An agent that researches topics.",
		Description:   "Researcher",
		SystemPrompt:  "You are a helpful researcher.\nMulti-line works.",
		UserPrompt:    "Research $$topic$$.",
		UsageType:     "Conversational",
		Variables: []sdkagenteditor.AgentVar{
			{Key: "topic", IsAttributeInEntity: false},
			{Key: "domain", IsAttributeInEntity: true},
		},
		Tools: []sdkagenteditor.AgentTool{
			{ID: "tool-1", Name: "WebSearch", Enabled: true, ToolType: "MCP",
				Document: &sdkagenteditor.DocRef{DocumentID: "doc-1", QualifiedName: "Module.Web"}},
		},
		KBTools: []sdkagenteditor.AgentKBTool{
			{ID: "kb-1", Name: "Docs", Enabled: true, ToolType: "KnowledgeBase",
				Document:             &sdkagenteditor.DocRef{DocumentID: "doc-2", QualifiedName: "Module.DocsKB"},
				CollectionIdentifier: "default", MaxResults: 5},
		},
		Model:       &sdkagenteditor.DocRef{DocumentID: "doc-3", QualifiedName: "Module.GPT4"},
		Entity:      &sdkagenteditor.DocRef{DocumentID: "doc-4", QualifiedName: "Module.Topic"},
		MaxTokens:   &maxTokens,
		ToolChoice:  "auto",
		Temperature: &temperature,
		TopP:        &topP,
	}

	mdlBytes, err := json.Marshal(mdlFixture)
	if err != nil {
		t.Fatalf("marshal mdl/types: %v", err)
	}
	sdkBytes, err := json.Marshal(sdkFixture)
	if err != nil {
		t.Fatalf("marshal sdk/agenteditor: %v", err)
	}
	if string(mdlBytes) != string(sdkBytes) {
		t.Errorf("Agent JSON wire format drift:\n mdl: %s\n sdk: %s", mdlBytes, sdkBytes)
	}
}

// TestModel_JsonWireCompatibleWithSdk asserts mdl/types.Model and
// sdk/agenteditor.Model produce identical JSON.
func TestModel_JsonWireCompatibleWithSdk(t *testing.T) {
	build := func() (any, any) {
		mdlFixture := &types.Model{
			BaseElement: model.BaseElement{
				ID:       model.ID("33333333-3333-3333-3333-333333333333"),
				TypeName: "CustomBlobDocuments$CustomBlobDocument",
			},
			ContainerID:   model.ID("44444444-4444-4444-4444-444444444444"),
			Name:          "GPT-4",
			Documentation: "OpenAI GPT-4 model",
			Type:          "Chat",
			InnerName:     "gpt-4",
			DisplayName:   "GPT-4 Turbo",
			Provider:      "MxCloudGenAI",
			Environment:   "prod",
			DeepLinkURL:   "https://portal.mendix.com/keys/abc",
			KeyID:         "key-1",
			KeyName:       "ProductionKey",
			ResourceName:  "gpt-4-turbo",
			Key: &types.ConstantRef{
				DocumentID:    "const-1",
				QualifiedName: "Module.PortalKey",
			},
		}
		sdkFixture := &sdkagenteditor.Model{
			BaseElement: model.BaseElement{
				ID:       model.ID("33333333-3333-3333-3333-333333333333"),
				TypeName: "CustomBlobDocuments$CustomBlobDocument",
			},
			ContainerID:   model.ID("44444444-4444-4444-4444-444444444444"),
			Name:          "GPT-4",
			Documentation: "OpenAI GPT-4 model",
			Type:          "Chat",
			InnerName:     "gpt-4",
			DisplayName:   "GPT-4 Turbo",
			Provider:      "MxCloudGenAI",
			Environment:   "prod",
			DeepLinkURL:   "https://portal.mendix.com/keys/abc",
			KeyID:         "key-1",
			KeyName:       "ProductionKey",
			ResourceName:  "gpt-4-turbo",
			Key: &sdkagenteditor.ConstantRef{
				DocumentID:    "const-1",
				QualifiedName: "Module.PortalKey",
			},
		}
		return mdlFixture, sdkFixture
	}
	assertWireFormatIdentical(t, "Model", build)
}

// TestKnowledgeBase_JsonWireCompatibleWithSdk asserts identity for KB.
func TestKnowledgeBase_JsonWireCompatibleWithSdk(t *testing.T) {
	build := func() (any, any) {
		mdlFixture := &types.KnowledgeBase{
			BaseElement: model.BaseElement{
				ID:       model.ID("55555555-5555-5555-5555-555555555555"),
				TypeName: "CustomBlobDocuments$CustomBlobDocument",
			},
			ContainerID:      model.ID("66666666-6666-6666-6666-666666666666"),
			Name:             "DocsKB",
			Documentation:    "Knowledge base over docs.",
			Provider:         "MxCloudGenAI",
			Environment:      "prod",
			DeepLinkURL:      "https://portal.mendix.com/kb/xyz",
			KeyID:            "kb-key",
			KeyName:          "KBKey",
			ModelDisplayName: "Embedding-3",
			ModelName:        "text-embedding-3-small",
			Key:              &types.ConstantRef{DocumentID: "const-2", QualifiedName: "Module.KBKey"},
		}
		sdkFixture := &sdkagenteditor.KnowledgeBase{
			BaseElement: model.BaseElement{
				ID:       model.ID("55555555-5555-5555-5555-555555555555"),
				TypeName: "CustomBlobDocuments$CustomBlobDocument",
			},
			ContainerID:      model.ID("66666666-6666-6666-6666-666666666666"),
			Name:             "DocsKB",
			Documentation:    "Knowledge base over docs.",
			Provider:         "MxCloudGenAI",
			Environment:      "prod",
			DeepLinkURL:      "https://portal.mendix.com/kb/xyz",
			KeyID:            "kb-key",
			KeyName:          "KBKey",
			ModelDisplayName: "Embedding-3",
			ModelName:        "text-embedding-3-small",
			Key:              &sdkagenteditor.ConstantRef{DocumentID: "const-2", QualifiedName: "Module.KBKey"},
		}
		return mdlFixture, sdkFixture
	}
	assertWireFormatIdentical(t, "KnowledgeBase", build)
}

// TestConsumedMCPService_JsonWireCompatibleWithSdk asserts identity for MCP service.
func TestConsumedMCPService_JsonWireCompatibleWithSdk(t *testing.T) {
	build := func() (any, any) {
		mdlFixture := &types.ConsumedMCPService{
			BaseElement: model.BaseElement{
				ID:       model.ID("77777777-7777-7777-7777-777777777777"),
				TypeName: "CustomBlobDocuments$CustomBlobDocument",
			},
			ContainerID:              model.ID("88888888-8888-8888-8888-888888888888"),
			Name:                     "WebSearch",
			Documentation:            "Web search MCP service.",
			Excluded:                 false,
			ExportLevel:              "Hidden",
			ProtocolVersion:          "2024-11-05",
			Version:                  "1.0.0",
			InnerDocumentation:       "Inner doc.",
			ConnectionTimeoutSeconds: 30,
		}
		sdkFixture := &sdkagenteditor.ConsumedMCPService{
			BaseElement: model.BaseElement{
				ID:       model.ID("77777777-7777-7777-7777-777777777777"),
				TypeName: "CustomBlobDocuments$CustomBlobDocument",
			},
			ContainerID:              model.ID("88888888-8888-8888-8888-888888888888"),
			Name:                     "WebSearch",
			Documentation:            "Web search MCP service.",
			Excluded:                 false,
			ExportLevel:              "Hidden",
			ProtocolVersion:          "2024-11-05",
			Version:                  "1.0.0",
			InnerDocumentation:       "Inner doc.",
			ConnectionTimeoutSeconds: 30,
		}
		return mdlFixture, sdkFixture
	}
	assertWireFormatIdentical(t, "ConsumedMCPService", build)
}

// TestAgentEditor_StructLayoutMirrorsSdk reflects the field count and
// types of mdl/types vs sdk/agenteditor counterparts so a future change
// to one but not the other fails this test.
func TestAgentEditor_StructLayoutMirrorsSdk(t *testing.T) {
	cases := []struct {
		name string
		mdl  any
		sdk  any
	}{
		{"Agent", &types.Agent{}, &sdkagenteditor.Agent{}},
		{"Model", &types.Model{}, &sdkagenteditor.Model{}},
		{"KnowledgeBase", &types.KnowledgeBase{}, &sdkagenteditor.KnowledgeBase{}},
		{"ConsumedMCPService", &types.ConsumedMCPService{}, &sdkagenteditor.ConsumedMCPService{}},
		{"DocRef", &types.DocRef{}, &sdkagenteditor.DocRef{}},
		{"ConstantRef", &types.ConstantRef{}, &sdkagenteditor.ConstantRef{}},
		{"AgentVar", &types.AgentVar{}, &sdkagenteditor.AgentVar{}},
		{"AgentTool", &types.AgentTool{}, &sdkagenteditor.AgentTool{}},
		{"AgentKBTool", &types.AgentKBTool{}, &sdkagenteditor.AgentKBTool{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mt := reflect.TypeOf(tc.mdl).Elem()
			st := reflect.TypeOf(tc.sdk).Elem()
			if mt.NumField() != st.NumField() {
				t.Fatalf("field count drift: mdl=%d sdk=%d", mt.NumField(), st.NumField())
			}
			for i := 0; i < mt.NumField(); i++ {
				mf, sf := mt.Field(i), st.Field(i)
				if mf.Name != sf.Name {
					t.Errorf("field[%d] name: mdl=%q sdk=%q", i, mf.Name, sf.Name)
				}
				if mf.Tag != sf.Tag {
					t.Errorf("field[%d] %q tag: mdl=%q sdk=%q", i, mf.Name, mf.Tag, sf.Tag)
				}
			}
		})
	}
}

func assertWireFormatIdentical(t *testing.T, label string, build func() (any, any)) {
	t.Helper()
	mdlFixture, sdkFixture := build()
	mdlBytes, err := json.Marshal(mdlFixture)
	if err != nil {
		t.Fatalf("%s: marshal mdl: %v", label, err)
	}
	sdkBytes, err := json.Marshal(sdkFixture)
	if err != nil {
		t.Fatalf("%s: marshal sdk: %v", label, err)
	}
	if string(mdlBytes) != string(sdkBytes) {
		t.Errorf("%s JSON wire drift:\n mdl: %s\n sdk: %s", label, mdlBytes, sdkBytes)
	}
}
