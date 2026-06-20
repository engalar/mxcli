// SPDX-License-Identifier: Apache-2.0

package ast

// RenameStmt represents:
//
//	RENAME ENTITY Module.OldName TO NewName [DRY RUN];
//	RENAME MODULE OldName TO NewName [DRY RUN];
//	RENAME JAVA ACTION Module.OldName TO NewName [DRY RUN];
//	... and microflow, nanoflow, page, enumeration, association, constant, workflow
type RenameStmt struct {
	ObjectType string        // lowercase type key, e.g. "entity", "module", "javaaction"
	Name       QualifiedName // Current name (Module.Entity or Module for modules)
	NewName    string        // New simple name (just the name, not qualified)
	DryRun     bool          // If true, report references without modifying
}

func (s *RenameStmt) isStatement()     {}
func (s *RenameStmt) TypeName() string { return "Rename" }
