// SPDX-License-Identifier: Apache-2.0

package ast

// CreateImageCollectionStmt represents:
//
//	CREATE IMAGE COLLECTION Module.Name [EXPORT LEVEL 'Public'] [COMMENT '...']
type CreateImageCollectionStmt struct {
	Name        QualifiedName
	ExportLevel string // "Hidden" (default) or "Public"
	Comment     string
}

func (s *CreateImageCollectionStmt) isStatement() {}

// DropImageCollectionStmt represents: DROP IMAGE COLLECTION Module.Name
type DropImageCollectionStmt struct {
	Name QualifiedName
}

func (s *DropImageCollectionStmt) isStatement() {}
