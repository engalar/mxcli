// SPDX-License-Identifier: Apache-2.0

package model

type ConstantDataType struct {
	Kind      string `json:"kind"`                // "String", "Integer", "Long", "Decimal", "Boolean", "DateTime", "Enumeration", "Binary"
	EnumRef   string `json:"enumRef,omitempty"`   // For Enumeration type: qualified name of the enumeration
	EntityRef string `json:"entityRef,omitempty"` // For Object/List types: qualified name of the entity
}

type Constant struct {
	BaseElement
	ContainerID     ID               `json:"containerId"`
	Name            string           `json:"name"`
	Documentation   string           `json:"documentation,omitempty"`
	Type            ConstantDataType `json:"type"`
	DefaultValue    string           `json:"defaultValue,omitempty"`
	ExposedToClient bool             `json:"exposedToClient,omitempty"`
	Excluded        bool             `json:"excluded,omitempty"`
	ExportLevel     string           `json:"exportLevel,omitempty"` // "Hidden" or "API"
}

func (c *Constant) GetName() string {
	return c.Name
}

func (c *Constant) GetContainerID() ID {
	return c.ContainerID
}

