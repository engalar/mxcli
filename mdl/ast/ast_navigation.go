// SPDX-License-Identifier: Apache-2.0

package ast

// AlterNavigationStmt represents: CREATE [OR REPLACE] NAVIGATION <profile> [clauses...]
// This is a full-replacement command: omitted clauses clear that section.
type AlterNavigationStmt struct {
	ProfileName    string           // e.g. "Responsive"
	HomePages      []NavHomePageDef // HOME PAGE/MICROFLOW ... [FOR role]
	LoginPage      *QualifiedName   // LOGIN PAGE ...
	NotFoundPage   *QualifiedName   // NOT FOUND PAGE ...
	MenuItems      []NavMenuItemDef // MENU (...) block
	HasMenuBlock   bool             // true if MENU (...) was present (even if empty → clears menu)
	CreateOrModify bool             // true if CREATE OR REPLACE/MODIFY was used
}

func (s *AlterNavigationStmt) isStatement()     {}
func (s *AlterNavigationStmt) TypeName() string { return "AlterNavigation" }

// NavHomePageDef represents a HOME PAGE or HOME MICROFLOW clause.
type NavHomePageDef struct {
	IsPage  bool           // true = PAGE, false = MICROFLOW
	Target  QualifiedName  // the page or microflow qualified name
	ForRole *QualifiedName // nil for default home, set for role-based
}

// NavMenuItemDef represents a MENU ITEM or MENU sub-menu definition.
type NavMenuItemDef struct {
	Caption   string           // from STRING_LITERAL
	Page      *QualifiedName   // PAGE target
	Microflow *QualifiedName   // MICROFLOW target
	Items     []NavMenuItemDef // Sub-items (for MENU 'caption' (...))
}
