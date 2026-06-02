// SPDX-License-Identifier: Apache-2.0

// Package entity holds the canonical in-memory model for Mendix entities and
// the Lift/Hydrate/ToMDL conversions between the parser AST, the gen-typed
// BSON representation, and MDL text.
package entity

import "github.com/mendixlabs/mxcli/mdl/canonical"

// QualifiedName is a module-qualified identifier ("Module.Entity").
type QualifiedName struct {
	Module string
	Name   string
}

// String renders as "Module.Name" (or just "Name" when Module is empty).
func (q QualifiedName) String() string {
	if q.Module == "" {
		return q.Name
	}
	return q.Module + "." + q.Name
}

// EntityKind mirrors ast.EntityKind in the canonical canonical.
type EntityKind int

const (
	EntityPersistent EntityKind = iota
	EntityNonPersistent
	EntityView
	EntityExternal
)

// EntityModel is the canonical in-memory representation of a Mendix entity.
type EntityModel struct {
	Name          QualifiedName
	Kind          EntityKind
	Documentation string
	Position      *Position
	Extends       *QualifiedName
	Attributes    []AttributeModel
	Indexes       []IndexModel
	SystemMembers []string
}

// Position is the entity's location on the domain-model canvas.
type Position struct {
	X int
	Y int
}

// AttributeModel is the canonical representation of an entity attribute.
type AttributeModel struct {
	Name                string
	Type                canonical.DataType
	Documentation       string
	NotNull             bool
	NotNullError        string
	Unique              bool
	UniqueError         string
	HasDefault          bool
	DefaultValue        string
	Calculated          bool
	CalculatedMicroflow *QualifiedName
}

// IndexModel is the canonical representation of a multi-column index.
type IndexModel struct {
	Name    string
	Columns []IndexColumn
}

// IndexColumn is one column within an IndexModel.
type IndexColumn struct {
	Name      string
	Ascending bool
}
