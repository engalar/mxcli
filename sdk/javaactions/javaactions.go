// SPDX-License-Identifier: Apache-2.0

// Package javaactions provides types for Mendix Java actions.
// Deprecated: All shared types are now defined in mdl/types; this package re-exports them as type
// aliases for backward compatibility and retains JavaAction (used by sdk/mpr parser layer).
package javaactions

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// Type aliases — canonical definitions are in mdl/types.
type (
	CodeActionReturnType        = types.CodeActionReturnType
	CodeActionParameterType     = types.CodeActionParameterType
	JavaActionParameter         = types.JavaActionParameter
	TypeParameterDef            = types.TypeParameterDef
	MicroflowActionInfo         = types.MicroflowActionInfo
	TypeParameter               = types.TypeParameter
	EntityTypeParameterType     = types.EntityTypeParameterType
	VoidType                    = types.VoidType
	BooleanType                 = types.BooleanType
	IntegerType                 = types.IntegerType
	LongType                    = types.LongType
	DecimalType                 = types.DecimalType
	StringType                  = types.StringType
	DateTimeType                = types.DateTimeType
	EntityType                  = types.EntityType
	ListType                    = types.ListType
	StringTemplateParameterType = types.StringTemplateParameterType
	FileDocumentType            = types.FileDocumentType
	EnumerationType             = types.EnumerationType
	MicroflowType               = types.MicroflowType
	NanoflowType                = types.NanoflowType
)

// JavaAction represents a Mendix Java action.
// The full definition is kept here because sdk/mpr's parser layer (parser_javaactions.go)
// still produces and returns *javaactions.JavaAction.
type JavaAction struct {
	model.BaseElement
	ContainerID             model.ID               `json:"containerId"`
	Name                    string                 `json:"name"`
	Documentation           string                 `json:"documentation,omitempty"`
	Excluded                bool                   `json:"excluded"`
	ExportLevel             string                 `json:"exportLevel,omitempty"`
	ActionDefaultReturnName string                 `json:"actionDefaultReturnName,omitempty"`
	ReturnType              CodeActionReturnType   `json:"returnType,omitempty"`
	Parameters              []*JavaActionParameter `json:"parameters,omitempty"`
	TypeParameters          []*TypeParameterDef    `json:"typeParameters,omitempty"`
	MicroflowActionInfo     *MicroflowActionInfo   `json:"microflowActionInfo,omitempty"`
}

// TypeParameterNames returns the type parameter names as a string slice (convenience).
func (ja *JavaAction) TypeParameterNames() []string {
	names := make([]string, len(ja.TypeParameters))
	for i, tp := range ja.TypeParameters {
		names[i] = tp.Name
	}
	return names
}

// FindTypeParameterName looks up a type parameter name by its ID.
func (ja *JavaAction) FindTypeParameterName(id model.ID) string {
	for _, tp := range ja.TypeParameters {
		if tp.ID == id {
			return tp.Name
		}
	}
	return ""
}

// GetName returns the Java action's name.
func (ja *JavaAction) GetName() string {
	return ja.Name
}

// GetContainerID returns the container ID.
func (ja *JavaAction) GetContainerID() model.ID {
	return ja.ContainerID
}
