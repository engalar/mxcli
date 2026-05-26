// SPDX-License-Identifier: Apache-2.0

// JSON content encoders and BSON wrapper builder for agent-editor documents.
// Replaces the sdk/mpr.SerializeAgentEditor* path — no sdk/mpr import needed.

package mprbackend

import (
	"encoding/json"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCBD "github.com/mendixlabs/mxcli/modelsdk/gen/customblobdocuments"
)

// newAgentBlobDoc builds a ready-to-encode *genCBD.CustomBlobDocument.
func newAgentBlobDoc(
	id string,
	name, documentation string,
	excluded bool,
	exportLevel string,
	customDocType, readableTypeName string,
	contentsJSON string,
) *genCBD.CustomBlobDocument {
	if exportLevel == "" {
		exportLevel = "Hidden"
	}
	doc := genCBD.NewCustomBlobDocument()
	doc.SetID(element.ID(id))
	doc.SetName(name)
	doc.SetDocumentation(documentation)
	doc.SetExcluded(excluded)
	doc.SetExportLevel(exportLevel)
	doc.SetCustomDocumentType(customDocType)
	doc.SetContents(contentsJSON)

	meta := genCBD.NewCustomBlobDocumentMetadata()
	meta.SetCreatedByExtension(types.CreatedByExtensionID)
	meta.SetReadableTypeName(readableTypeName)
	doc.SetMetadata(meta)
	return doc
}

// ── JSON content encoders ──────────────────────────────────────────────────

func encodeModelContentsJSON(m *types.Model) (string, error) {
	type providerFields struct {
		Environment  string             `json:"environment"`
		DeepLinkURL  string             `json:"deepLinkURL"`
		KeyID        string             `json:"keyId"`
		KeyName      string             `json:"keyName"`
		ResourceName string             `json:"resourceName"`
		Key          *types.ConstantRef `json:"key,omitempty"`
	}
	type shape struct {
		Type           string         `json:"type"`
		Name           string         `json:"name"`
		DisplayName    string         `json:"displayName"`
		Provider       string         `json:"provider"`
		ProviderFields providerFields `json:"providerFields"`
	}
	return marshalAgentJSON(shape{
		Type: m.Type, Name: m.InnerName, DisplayName: m.DisplayName, Provider: m.Provider,
		ProviderFields: providerFields{
			Environment: m.Environment, DeepLinkURL: m.DeepLinkURL,
			KeyID: m.KeyID, KeyName: m.KeyName, ResourceName: m.ResourceName, Key: m.Key,
		},
	})
}

func encodeKnowledgeBaseContentsJSON(k *types.KnowledgeBase) (string, error) {
	type providerFields struct {
		Environment      string             `json:"environment"`
		DeepLinkURL      string             `json:"deepLinkURL"`
		KeyID            string             `json:"keyId"`
		KeyName          string             `json:"keyName"`
		ModelDisplayName string             `json:"modelDisplayName"`
		ModelName        string             `json:"modelName"`
		Key              *types.ConstantRef `json:"key,omitempty"`
	}
	type shape struct {
		Name           string         `json:"name"`
		Provider       string         `json:"provider"`
		ProviderFields providerFields `json:"providerFields"`
	}
	return marshalAgentJSON(shape{
		Name: "", Provider: k.Provider,
		ProviderFields: providerFields{
			Environment: k.Environment, DeepLinkURL: k.DeepLinkURL,
			KeyID: k.KeyID, KeyName: k.KeyName,
			ModelDisplayName: k.ModelDisplayName, ModelName: k.ModelName, Key: k.Key,
		},
	})
}

func encodeMCPServiceContentsJSON(c *types.ConsumedMCPService) (string, error) {
	type shape struct {
		ProtocolVersion          string `json:"protocolVersion"`
		Documentation            string `json:"documentation"`
		Version                  string `json:"version"`
		ConnectionTimeoutSeconds int    `json:"connectionTimeoutSeconds"`
	}
	return marshalAgentJSON(shape{
		ProtocolVersion:          c.ProtocolVersion,
		Documentation:            c.InnerDocumentation,
		Version:                  c.Version,
		ConnectionTimeoutSeconds: c.ConnectionTimeoutSeconds,
	})
}

func encodeAgentContentsJSON(a *types.Agent) (string, error) {
	type toolEntry struct {
		ID          string        `json:"id"`
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Enabled     bool          `json:"enabled"`
		ToolType    string        `json:"toolType"`
		Document    *types.DocRef `json:"document,omitempty"`
	}
	type kbToolEntry struct {
		ID                   string        `json:"id"`
		Name                 string        `json:"name"`
		Description          string        `json:"description"`
		Enabled              bool          `json:"enabled"`
		ToolType             string        `json:"toolType"`
		Document             *types.DocRef `json:"document,omitempty"`
		CollectionIdentifier string        `json:"collectionIdentifier,omitempty"`
		MaxResults           int           `json:"maxResults,omitempty"`
	}
	type shape struct {
		Description        string           `json:"description"`
		SystemPrompt       string           `json:"systemPrompt"`
		UserPrompt         string           `json:"userPrompt"`
		UsageType          string           `json:"usageType"`
		Variables          []types.AgentVar `json:"variables"`
		Tools              []toolEntry      `json:"tools"`
		KnowledgebaseTools []kbToolEntry    `json:"knowledgebaseTools"`
		Model              *types.DocRef    `json:"model,omitempty"`
		Entity             *types.DocRef    `json:"entity,omitempty"`
		MaxTokens          *int             `json:"maxTokens,omitempty"`
		ToolChoice         string           `json:"toolChoice,omitempty"`
		Temperature        *float64         `json:"temperature,omitempty"`
		TopP               *float64         `json:"topP,omitempty"`
	}
	tools := make([]toolEntry, 0, len(a.Tools))
	for _, t := range a.Tools {
		tools = append(tools, toolEntry{
			ID: t.ID, Name: t.Name, Description: t.Description,
			Enabled: t.Enabled, ToolType: t.ToolType, Document: t.Document,
		})
	}
	kbTools := make([]kbToolEntry, 0, len(a.KBTools))
	for _, kb := range a.KBTools {
		kbTools = append(kbTools, kbToolEntry{
			ID: kb.ID, Name: kb.Name, Description: kb.Description,
			Enabled: kb.Enabled, ToolType: kb.ToolType, Document: kb.Document,
			CollectionIdentifier: kb.CollectionIdentifier, MaxResults: kb.MaxResults,
		})
	}
	vars := a.Variables
	if vars == nil {
		vars = []types.AgentVar{}
	}
	return marshalAgentJSON(shape{
		Description: a.Description, SystemPrompt: a.SystemPrompt,
		UserPrompt: a.UserPrompt, UsageType: a.UsageType,
		Variables: vars, Tools: tools, KnowledgebaseTools: kbTools,
		Model: a.Model, Entity: a.Entity,
		MaxTokens: a.MaxTokens, ToolChoice: a.ToolChoice,
		Temperature: a.Temperature, TopP: a.TopP,
	})
}

// marshalAgentJSON produces JSON without HTML escaping, matching Studio Pro output.
func marshalAgentJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("agent editor JSON encode: %w", err)
	}
	return string(b), nil
}
