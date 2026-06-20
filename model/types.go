// SPDX-License-Identifier: Apache-2.0

package model

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ID represents a unique identifier for model elements.
// In Mendix, these are typically UUIDs.
type ID string

// QualifiedName represents a fully qualified name in the format "Module.Element".
type QualifiedName string

// Point represents a position in 2D space.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Size represents dimensions in 2D space.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Element is the base interface for all model elements.
type Element interface {
	GetID() ID
	GetTypeName() string
}

// NamedElement is an element with a name.
type NamedElement interface {
	Element
	GetName() string
}

// ContainedElement is an element that belongs to a container.
type ContainedElement interface {
	Element
	GetContainerID() ID
}

// BaseElement provides common fields for all model elements.
type BaseElement struct {
	ID       ID     `json:"$ID"`
	TypeName string `json:"$Type"`
}

// GetID returns the element's unique identifier.
func (e *BaseElement) GetID() ID {
	return e.ID
}

// GetTypeName returns the element's type name.
func (e *BaseElement) GetTypeName() string {
	return e.TypeName
}

// Unit represents a document unit in the Mendix model.
// Units are top-level elements like DomainModel, Microflow, Page, etc.
type Unit struct {
	BaseElement
	ContainerID ID     `json:"containerId"`
	Name        string `json:"name,omitempty"`
}

// GetName returns the unit's name.
func (u *Unit) GetName() string {
	return u.Name
}

// GetContainerID returns the ID of the containing element.
func (u *Unit) GetContainerID() ID {
	return u.ContainerID
}

// Module represents a Mendix module.
// GetName returns the module's name.
// Project represents a Mendix project.
// GetName returns the project's name.
// Folder represents a folder within a module for organizing documents.
// GetName returns the folder's name.
// GetContainerID returns the ID of the containing element.
// Text represents localized text.
type Text struct {
	BaseElement
	Translations map[string]string `json:"translations,omitempty"`
}

// GetTranslation returns the translation for a given language code.
func (t *Text) GetTranslation(languageCode string) string {
	if t.Translations == nil {
		return ""
	}
	return t.Translations[languageCode]
}

// TranslationNode represents a single translatable text field in a document,
// with its per-language translations. A missing language key means that
// language has not been translated for this field.
type TranslationNode struct {
	Path     string            `json:"path"`     // e.g. "Button_Submit.caption"
	Property string            `json:"property"` // e.g. "caption"
	DocType  string            `json:"docType"`  // "PAGE", "SNIPPET", "ENUMERATION", "WORKFLOW", "MICROFLOW"
	Texts    map[string]string `json:"texts"`    // langCode -> text; missing key = not translated
}

// Image represents an image reference.
type Image struct {
	BaseElement
	Name      string `json:"name,omitempty"`
	ImageData []byte `json:"imageData,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
}

// ConstantDataType represents the data type of a constant.
// Constant represents a constant value.
// GetName returns the constant's name.
// GetContainerID returns the ID of the containing element.
// Enumeration represents an enumeration type.
// GetName returns the enumeration's name.
// GetContainerID returns the ID of the containing element.
// EnumerationValue represents a value in an enumeration.
// GetName returns the enumeration value's name.
// RegularExpression represents a regular expression constraint.
// GetName returns the regular expression's name.
// GetContainerID returns the ID of the containing element.
// ScheduledEvent represents a scheduled event.
// GetName returns the scheduled event's name.
// GetContainerID returns the ID of the containing element.
// MarshalJSON provides custom JSON marshaling.
func (e *BaseElement) MarshalJSON() ([]byte, error) {
	type Alias BaseElement
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	})
}

// DocumentType is intentionally not defined here.
// Use codec.DefaultRegistry.TypeNameOf(reflect.TypeOf(gen.Type{})) to obtain
// the canonical BSON $Type name for a given gen/* type at compile time.
// The former DocumentType constants had wrong values (e.g. "Pages$Page" instead
// of the actual "Forms$Page") and no external callers — deleted to prevent misuse.
// ConsumedODataService represents a consumed OData service (OData client).
// HttpConfiguration represents the HTTP transport configuration (Microflows$HttpConfiguration).
// HttpHeaderEntry represents a custom HTTP header (Microflows$HttpHeaderEntry).
// GetName returns the service's name.
// GetContainerID returns the ID of the containing module.
// PublishedODataService represents a published OData service.
// GetName returns the service's name.
// GetContainerID returns the ID of the containing module.
// PublishedEntityType represents an entity type published in an OData service.
// PublishedEntitySet represents an entity set published in an OData service.
// PublishedMember represents a member (attribute/association/id) published in an OData entity type.
// ============================================================================
// Database Connection (DatabaseConnector marketplace module)
// ============================================================================
// DatabaseConnection represents a DatabaseConnector$DatabaseConnection document.
// DatabaseQuery represents a DatabaseConnector$DatabaseQuery.
// DatabaseQueryParameter represents a DatabaseConnector$QueryParameter.
// DatabaseTableMapping represents a DatabaseConnector$TableMapping.
// DatabaseColumnMapping represents a DatabaseConnector$ColumnMapping.
// ============================================================================
// Business Events
// ============================================================================
// BusinessEventService represents a BusinessEvents$BusinessEventService document.
// GetName returns the service's name.
// GetContainerID returns the ID of the containing module.
// BusinessEventDefinition represents BusinessEvents$BusinessEventDefinition.
// BusinessEventChannel represents BusinessEvents$Channel.
// BusinessEventMessage represents BusinessEvents$Message.
// BusinessEventAttribute represents BusinessEvents$MessageAttribute.
// ServiceOperation represents BusinessEvents$ServiceOperation.
// ============================================================================
// Published REST Services
// ============================================================================
// PublishedRestService represents a Rest$PublishedRestService document.
// GetName returns the service's name.
// GetContainerID returns the ID of the containing element.
// PublishedRestResource represents a Rest$PublishedRestServiceResource.
// PublishedRestOperation represents a Rest$PublishedRestServiceOperation.
// ============================================================================
// Consumed REST Services
// ============================================================================
// ConsumedRestService represents a Rest$ConsumedRestService document.
// GetName returns the service's name.
// GetContainerID returns the ID of the containing element.
// RestAuthentication represents authentication configuration for a consumed REST service.
// RestClientOperation represents a single operation in a consumed REST service.
// RestResponseMapping represents one element in a response mapping tree.
// It is either a value mapping (Attribute set, Entity empty) or an object mapping
// (Entity set, with its own Children).
// RestClientParameter represents a path or query parameter.
// RestClientHeader represents an HTTP header in a REST client operation.
// ============================================================================
// Data Transformers (Mendix 11.9+)
// ============================================================================
// DataTransformer represents a DataTransformers$DataTransformer document.
// GetName returns the transformer's name.
// GetContainerID returns the ID of the containing element.
// DataTransformerStep represents a single transformation step.
// ============================================================================
// Project Settings
// ============================================================================
// ProjectSettings represents the single Settings$ProjectSettings document.
// WebUISettings represents Forms$WebUIProjectSettingsPart.
// IntegrationSettings represents Settings$IntegrationProjectSettingsPart.
// ConfigurationSettings represents Settings$ConfigurationSettings.
// ServerConfiguration represents Settings$ServerConfiguration.
// ConstantValue represents Settings$ConstantValue (constant override per configuration).
// ModelSettings represents Settings$ModelSettings.
// ConventionSettings represents Settings$ConventionSettings.
// LanguageSettings represents Settings$LanguageSettings.
// Language represents a Texts$Language entry in the project language settings.
// The Languages slice is populated by parseLanguageSettings and is available
// for use by settings describers and future language-aware commands.
// CertificateSettings represents Settings$CertificateSettings.
// WorkflowsSettings represents Settings$WorkflowsProjectSettingsPart.
// JarDeploymentSettings represents Settings$JarDeploymentSettings.
// DistributionSettings represents Settings$DistributionSettings.
// ============================================================================
// Import Mappings
// ============================================================================
// ImportMapping represents an ImportMappings$ImportMapping document.
// GetName returns the import mapping's name.
// GetContainerID returns the ID of the containing module.
// ImportMappingElement represents either an object or value mapping element.
// ============================================================================
// Export Mappings
// ============================================================================
// ExportMapping represents an ExportMappings$ExportMapping document.
// GetName returns the export mapping's name.
// GetContainerID returns the ID of the containing module.
// ExportMappingElement represents either an object or value mapping element in an export mapping.
// UnknownElement is a generic fallback for BSON elements with unrecognized $Type values.
// It preserves all raw BSON fields so developers can diagnose unimplemented types
// without silent data loss.
//
// FieldKinds maps each raw field name to its inferred Mendix property kind
// (e.g. "primitive", "part", "by-name-reference", "collection:part-primary").
// This guides implementors in writing a proper parser without inspecting the
// mendixmodelsdk JS source manually.
type UnknownElement struct {
	BaseElement
	Position   Point             `json:"position,omitempty"`
	Name       string            `json:"name,omitempty"`
	Caption    string            `json:"caption,omitempty"`
	RawDoc     bson.D            `json:"-"`
	FieldKinds map[string]string `json:"-"`
}

// GetPosition returns the element's position (satisfies microflows.MicroflowObject).
func (u *UnknownElement) GetPosition() Point { return u.Position }

// SetPosition sets the element's position (satisfies microflows.MicroflowObject).
func (u *UnknownElement) SetPosition(p Point) { u.Position = p }

// GetName returns the element's name (satisfies workflows.WorkflowActivity).
func (u *UnknownElement) GetName() string { return u.Name }

// GetCaption returns the element's caption (satisfies workflows.WorkflowActivity).
func (u *UnknownElement) GetCaption() string { return u.Caption }

// ActivityType returns the type name (satisfies workflows.WorkflowActivity).
func (u *UnknownElement) ActivityType() string { return u.TypeName }
