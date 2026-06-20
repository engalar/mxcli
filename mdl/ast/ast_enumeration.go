// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Enumeration Statements
// ============================================================================

// EnumValue represents a single enumeration value with its caption.
type EnumValue struct {
	Name          string
	Caption       string
	Documentation string // JavaDoc for individual enum value
}

// CreateModuleStmt represents: CREATE MODULE ModuleName
type CreateModuleStmt struct {
	Name string
}

func (s *CreateModuleStmt) isStatement() {}
func (s *CreateModuleStmt) TypeName() string { return "CreateModule" }

// DropModuleStmt represents: DROP MODULE ModuleName
type DropModuleStmt struct {
	Name string
}

func (s *DropModuleStmt) isStatement() {}
func (s *DropModuleStmt) TypeName() string { return "DropModule" }

// DropFolderStmt represents: DROP FOLDER 'path' IN Module
type DropFolderStmt struct {
	FolderPath string // Folder path (e.g., "Resources/Images")
	Module     string // Module name
}

func (s *DropFolderStmt) isStatement() {}
func (s *DropFolderStmt) TypeName() string { return "DropFolder" }

// MoveFolderStmt represents: MOVE FOLDER Module.FolderName TO ...
type MoveFolderStmt struct {
	Name         QualifiedName // Source: Module.FolderName (Name may be "Parent/Child" for nested)
	TargetFolder string        // Target folder path (empty = module root)
	TargetModule string        // Target module name (empty = same module)
}

func (s *MoveFolderStmt) isStatement() {}
func (s *MoveFolderStmt) TypeName() string { return "MoveFolder" }

// CreateEnumerationStmt represents: CREATE ENUMERATION Module.Name (values) COMMENT '...'
type CreateEnumerationStmt struct {
	Name           QualifiedName
	Values         []EnumValue
	Documentation  string
	Comment        string
	CreateOrModify bool // True if CREATE OR MODIFY was used
}

func (s *CreateEnumerationStmt) isStatement() {}
func (s *CreateEnumerationStmt) TypeName() string { return "CreateEnumeration" }

// AlterEnumerationStmt represents: ALTER ENUMERATION Module.Name ADD/DROP/RENAME VALUE ...
type AlterEnumerationStmt struct {
	Name      QualifiedName
	Operation AlterEnumOp
	ValueName string
	NewName   string // For RENAME
	Caption   string // For ADD
}

func (s *AlterEnumerationStmt) isStatement() {}
func (s *AlterEnumerationStmt) TypeName() string { return "AlterEnumeration" }

// AlterEnumOp represents the type of enumeration alteration.
type AlterEnumOp int

const (
	AlterEnumAdd AlterEnumOp = iota
	AlterEnumDrop
	AlterEnumRename
)

// DropEnumerationStmt represents: DROP ENUMERATION Module.Name
type DropEnumerationStmt struct {
	Name QualifiedName
}

func (s *DropEnumerationStmt) isStatement() {}
func (s *DropEnumerationStmt) TypeName() string { return "DropEnumeration" }

// ============================================================================
// Constant Statements
// ============================================================================

// CreateConstantStmt represents: CREATE CONSTANT Module.Name TYPE type DEFAULT value [COMMENT '...']
type CreateConstantStmt struct {
	Name            QualifiedName
	DataType        DataType
	DefaultValue    any // The default value (can be string, number, boolean, etc.)
	Documentation   string
	Comment         string
	Folder          string // Folder path within module (e.g., "Resources/Constants")
	ExposedToClient bool
	CreateOrModify  bool // True if CREATE OR MODIFY was used
}

func (s *CreateConstantStmt) isStatement() {}
func (s *CreateConstantStmt) TypeName() string { return "CreateConstant" }

// DropConstantStmt represents: DROP CONSTANT Module.Name
type DropConstantStmt struct {
	Name QualifiedName
}

func (s *DropConstantStmt) isStatement() {}
func (s *DropConstantStmt) TypeName() string { return "DropConstant" }
