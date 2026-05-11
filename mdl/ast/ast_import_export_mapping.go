// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Import Mapping Statements
// ============================================================================

// CreateImportMappingStmt represents:
//
//	CREATE IMPORT MAPPING Module.Name
//	  WITH JSON STRUCTURE Module.JsonStructure
//	{
//	  CREATE Module.Entity {
//	    PetId = id KEY,
//	    Name = name
//	  }
//	};
type CreateImportMappingStmt struct {
	Name           QualifiedName
	SchemaKind     string        // "JSON_STRUCTURE" or "XML_SCHEMA" or ""
	SchemaRef      QualifiedName // qualified name of the schema source
	RootElement    *ImportMappingElementDef
	CreateOrModify bool          // true for CREATE OR MODIFY / CREATE OR REPLACE
}

func (s *CreateImportMappingStmt) isStatement() {}

// DropImportMappingStmt represents: DROP IMPORT MAPPING Module.Name
type DropImportMappingStmt struct {
	Name QualifiedName
}

func (s *DropImportMappingStmt) isStatement() {}

// ImportMappingElementDef represents one element in the mapping tree.
type ImportMappingElementDef struct {
	// Object mapping fields
	Entity         string // qualified entity name (e.g. "Module.Customer")
	ObjectHandling string // "Create", "Find", "FindOrCreate"
	Association    string // qualified association name (from Assoc/Entity path)
	Children       []*ImportMappingElementDef

	// Value mapping fields
	Attribute string // entity attribute name (LHS of =)
	IsKey     bool

	// Shared
	JsonName string // JSON field name (RHS of = for both values and objects)

	// Value transform via microflow
	Converter      string // microflow qualified name (e.g. "Module.ConvertStringToDate")
	ConverterParam string // json field passed to converter
}

// ============================================================================
// Export Mapping Statements
// ============================================================================

// CreateExportMappingStmt represents:
//
//	CREATE EXPORT MAPPING Module.Name
//	  WITH JSON STRUCTURE Module.JsonStructure
//	{
//	  Module.Entity {
//	    jsonField = Attr,
//	    Module.Assoc/Module.Child AS jsonKey { ... }
//	  }
//	};
type CreateExportMappingStmt struct {
	Name            QualifiedName
	SchemaKind      string        // "JSON_STRUCTURE" or "XML_SCHEMA" or ""
	SchemaRef       QualifiedName // qualified name of the schema source
	NullValueOption string        // "LeaveOutElement" or "SendAsNil" (default: "LeaveOutElement")
	RootElement     *ExportMappingElementDef
	CreateOrModify  bool          // true for CREATE OR MODIFY / CREATE OR REPLACE
}

func (s *CreateExportMappingStmt) isStatement() {}

// DropExportMappingStmt represents: DROP EXPORT MAPPING Module.Name
type DropExportMappingStmt struct {
	Name QualifiedName
}

func (s *DropExportMappingStmt) isStatement() {}

// ExportMappingElementDef represents one element in an export mapping tree.
type ExportMappingElementDef struct {
	// Object mapping fields
	Entity      string // qualified entity name (e.g. "Module.Customer")
	Association string // qualified association name (from Assoc/Entity path)
	Children    []*ExportMappingElementDef

	// Value mapping fields
	Attribute string // entity attribute name (RHS of =)

	// Shared
	JsonName string // JSON field name (LHS of = for values, RHS of AS for objects)
}
