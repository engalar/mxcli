// SPDX-License-Identifier: Apache-2.0

package ast

// CreateDataTransformerStmt represents:
//
//	CREATE DATA TRANSFORMER Module.Name SOURCE JSON '...' { JSLT '...'; };
type CreateDataTransformerStmt struct {
	Name           QualifiedName
	SourceType     string // "JSON" or "XML"
	SourceJSON     string // the source content
	Steps          []DataTransformerStepDef
	CreateOrModify bool // true for CREATE OR MODIFY / CREATE OR REPLACE
}

func (s *CreateDataTransformerStmt) isStatement() {}

// DataTransformerStepDef represents a single step: JSLT '...' or XSLT '...'
type DataTransformerStepDef struct {
	Technology string // "JSLT", "XSLT"
	Expression string
}

// DropDataTransformerStmt represents: DROP DATA TRANSFORMER Module.Name
type DropDataTransformerStmt struct {
	Name QualifiedName
}

func (s *DropDataTransformerStmt) isStatement() {}
