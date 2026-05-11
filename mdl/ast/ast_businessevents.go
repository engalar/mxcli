// SPDX-License-Identifier: Apache-2.0

package ast

// CreateBusinessEventServiceStmt represents CREATE BUSINESS EVENT SERVICE.
type CreateBusinessEventServiceStmt struct {
	Name            QualifiedName
	ServiceName     string
	EventNamePrefix string
	Messages        []*BusinessEventMessageDef
	CreateOrModify bool
	Folder          string
	Documentation   string
}

func (s *CreateBusinessEventServiceStmt) isStatement() {}

// BusinessEventMessageDef defines a message within a business event service.
type BusinessEventMessageDef struct {
	MessageName string
	Attributes  []*BusinessEventAttributeDef
	Operation   string // "PUBLISH" or "SUBSCRIBE"
	Entity      string // Qualified name of linked entity
	Microflow   string // Optional handler microflow
}

// BusinessEventAttributeDef defines an attribute within a message.
type BusinessEventAttributeDef struct {
	Name     string
	TypeName string // "Long", "String", "Integer", "Boolean", "DateTime", "Decimal"
}

// DropBusinessEventServiceStmt represents DROP BUSINESS EVENT SERVICE.
type DropBusinessEventServiceStmt struct {
	Name QualifiedName
}

func (s *DropBusinessEventServiceStmt) isStatement() {}
