// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// RenameBackend provides cross-cutting rename and reference-update operations.
type RenameBackend interface {
	UpdateQualifiedNameInAllUnits(oldName, newName string) (int, error)
	RenameReferences(oldName, newName string, dryRun bool) ([]types.RenameHit, error)
	RenameDocumentByName(moduleName, oldName, newName string) error
}

// RawUnitBackend provides low-level unit access for operations that
// manipulate raw unit contents (e.g. widget patching, alter page/workflow).
type RawUnitBackend interface {
	GetRawUnit(id model.ID) (map[string]any, error)
	GetRawUnitBytes(id model.ID) ([]byte, error)
	ListRawUnitsByType(typePrefix string) ([]*types.RawUnit, error)
	ListRawUnits(objectType string) ([]*types.RawUnitInfo, error)
	GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error)
	GetRawMicroflowByName(qualifiedName string) ([]byte, error)
	// UpdateRawUnit replaces the contents of a unit by ID.
	// Takes string (not model.ID) to match the SDK writer layer convention.
	UpdateRawUnit(unitID string, contents []byte) error
}

// MetadataBackend provides project-level metadata and introspection.
type MetadataBackend interface {
	ListAllUnitIDs() ([]string, error)
	ListUnits() ([]*types.UnitInfo, error)
	// ListUnitHashes returns a map from unit UUID string to its ContentsHash.
	// Units with empty ContentsHash are omitted. Returns nil map (not error)
	// when the backend does not support hash queries.
	ListUnitHashes() (map[string]string, error)
	GetUnitTypes() (map[string]int, error)
	GetProjectRootID() (string, error)
	ContentsDir() string
	InvalidateCache()
}

// WidgetBackend provides widget introspection operations.
type WidgetBackend interface {
	FindCustomWidgetType(widgetID string) (*types.RawCustomWidgetType, error)
	FindAllCustomWidgetTypes(widgetID string) ([]*types.RawCustomWidgetType, error)
}

// AgentEditorBackend provides agent editor document operations.
// Delete methods take string IDs to match the SDK writer layer convention.
type AgentEditorBackend interface {
	ListAgentEditorModels() ([]*types.Model, error)
	ListAgentEditorKnowledgeBases() ([]*types.KnowledgeBase, error)
	ListAgentEditorConsumedMCPServices() ([]*types.ConsumedMCPService, error)
	ListAgentEditorAgents() ([]*types.Agent, error)
	CreateAgentEditorModel(m *types.Model) error
	UpdateAgentEditorModel(m *types.Model) error
	DeleteAgentEditorModel(id string) error
	CreateAgentEditorKnowledgeBase(k *types.KnowledgeBase) error
	UpdateAgentEditorKnowledgeBase(k *types.KnowledgeBase) error
	DeleteAgentEditorKnowledgeBase(id string) error
	CreateAgentEditorConsumedMCPService(c *types.ConsumedMCPService) error
	UpdateAgentEditorConsumedMCPService(c *types.ConsumedMCPService) error
	DeleteAgentEditorConsumedMCPService(id string) error
	CreateAgentEditorAgent(a *types.Agent) error
	UpdateAgentEditorAgent(a *types.Agent) error
	DeleteAgentEditorAgent(id string) error
}

// SettingsBackend provides project settings operations.
type SettingsBackend interface {
	GetProjectSettings() (*model.ProjectSettings, error)
	UpdateProjectSettings(ps *model.ProjectSettings) error
	// ListTranslationNodes returns the translatable text fields of a document
	// (identified by its qualified name and optional doc type) with their
	// per-language translations.
	ListTranslationNodes(docQN, docType string) ([]model.TranslationNode, error)
	// SetEnumerationTranslation sets the langCode translation on the caption of
	// the enumeration value named valueName, within the enumeration identified
	// by its qualified name.
	SetEnumerationTranslation(enumQN, valueName, langCode, text string) error
	// SetMicroflowActionTranslation sets the langCode translation on a
	// translatable property of an action inside a microflow. Microflow actions
	// are unnamed and addressed by their BSON action type (e.g.
	// "Microflows$ShowMessageAction") and the 0-based ordinal index among
	// same-typed actions.
	SetMicroflowActionTranslation(docQN, actionType string, index int, property, langCode, text string) error
	// SetNavigationCaptionTranslation sets the langCode translation on a
	// navigation menu item caption identified by its caption path (hierarchy of
	// en_US menu item captions, e.g. ["Ticket Management", "All Tickets"]).
	SetNavigationCaptionTranslation(profileName string, menuPath []string, langCode, text string) error
}

// ImageBackend provides image collection operations.
type ImageBackend interface {
	ListImageCollections() ([]*types.ImageCollection, error)
	CreateImageCollection(ic *types.ImageCollection) error
	UpdateImageCollection(ic *types.ImageCollection) error
	DeleteImageCollection(id string) error
	MoveImageCollection(ic *types.ImageCollection) error
}

// ScheduledEventBackend provides scheduled event operations.
type ScheduledEventBackend interface {
	ListScheduledEvents() ([]*model.ScheduledEvent, error)
	GetScheduledEvent(id model.ID) (*model.ScheduledEvent, error)
}
