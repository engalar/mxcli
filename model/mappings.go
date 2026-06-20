// SPDX-License-Identifier: Apache-2.0

package model

type ImportMapping struct {
	BaseElement
	ContainerID   ID     `json:"containerId"`
	Name          string `json:"name"`
	Documentation string `json:"documentation,omitempty"`
	Excluded      bool   `json:"excluded,omitempty"`
	ExportLevel   string `json:"exportLevel,omitempty"`
	// Schema source (at most one is set)
	JsonStructure     string `json:"jsonStructure,omitempty"`     // qualified name
	XmlSchema         string `json:"xmlSchema,omitempty"`         // qualified name
	MessageDefinition string `json:"messageDefinition,omitempty"` // qualified name
	// Mapping tree (top-level elements, usually one root)
	Elements []*ImportMappingElement `json:"elements,omitempty"`
}

func (m *ImportMapping) GetName() string { return m.Name }

func (m *ImportMapping) GetContainerID() ID { return m.ContainerID }

type ImportMappingElement struct {
	BaseElement
	// "Object", "Value", or "Array"
	Kind string `json:"kind"`
	// Object mapping fields
	Entity         string `json:"entity,omitempty"`         // qualified entity name
	ObjectHandling string `json:"objectHandling,omitempty"` // "Create", "Find", "FindOrCreate", "Custom"
	Association    string `json:"association,omitempty"`    // qualified association name
	// Value mapping fields
	Attribute string `json:"attribute,omitempty"` // qualified attribute name (Module.Entity.Attr)
	DataType  string `json:"dataType,omitempty"`  // "String", "Integer", "Boolean", etc.
	IsKey     bool   `json:"isKey,omitempty"`
	// Schema fields (cloned from JSON structure element)
	ExposedName    string `json:"exposedName,omitempty"`
	JsonPath       string `json:"jsonPath,omitempty"`
	MinOccurs      int    `json:"minOccurs,omitempty"`
	MaxOccurs      int    `json:"maxOccurs,omitempty"` // 0 = default from JSON structure
	Nillable       bool   `json:"nillable,omitempty"`
	OriginalValue  string `json:"originalValue,omitempty"`
	FractionDigits int    `json:"fractionDigits,omitempty"` // -1 = unset
	TotalDigits    int    `json:"totalDigits,omitempty"`    // -1 = unset
	MaxLength      int    `json:"maxLength,omitempty"`      // -1 = unset for non-string
	// Children
	Children []*ImportMappingElement `json:"children,omitempty"`
}

type ExportMapping struct {
	BaseElement
	ContainerID   ID     `json:"containerId"`
	Name          string `json:"name"`
	Documentation string `json:"documentation,omitempty"`
	Excluded      bool   `json:"excluded,omitempty"`
	ExportLevel   string `json:"exportLevel,omitempty"`
	// Schema source (at most one is set)
	JsonStructure     string `json:"jsonStructure,omitempty"`     // qualified name
	XmlSchema         string `json:"xmlSchema,omitempty"`         // qualified name
	MessageDefinition string `json:"messageDefinition,omitempty"` // qualified name
	// NullValueOption controls how null values are serialized: "LeaveOutElement" or "SendAsNil"
	NullValueOption string                  `json:"nullValueOption,omitempty"`
	Elements        []*ExportMappingElement `json:"elements,omitempty"`
}

func (m *ExportMapping) GetName() string { return m.Name }

func (m *ExportMapping) GetContainerID() ID { return m.ContainerID }

type ExportMappingElement struct {
	BaseElement
	// "Object" or "Value"
	Kind string `json:"kind"`
	// Object mapping fields
	Entity         string `json:"entity,omitempty"`         // qualified entity name
	Association    string `json:"association,omitempty"`    // qualified association name
	ObjectHandling string `json:"objectHandling,omitempty"` // "Parameter" for root, "Find" for children
	MaxOccurs      int    `json:"maxOccurs,omitempty"`      // 1 for Object, -1 for Array; 0 = default (1)
	// Value mapping fields
	Attribute string `json:"attribute,omitempty"` // qualified attribute name (Module.Entity.Attr)
	DataType  string `json:"dataType,omitempty"`  // "String", "Integer", "Boolean", etc.
	// Shared fields
	ExposedName string                  `json:"exposedName,omitempty"`
	JsonPath    string                  `json:"jsonPath,omitempty"`
	Children    []*ExportMappingElement `json:"children,omitempty"`
}
