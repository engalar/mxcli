// SPDX-License-Identifier: Apache-2.0

package ast

// LintFormat specifies the output format for lint results.
type LintFormat string

const (
	LintFormatText  LintFormat = "text"
	LintFormatJSON  LintFormat = "json"
	LintFormatSARIF LintFormat = "sarif"
)

// LintStmt represents a LINT statement.
type LintStmt struct {
	// Target specifies what to lint:
	// - nil: lint all
	// - QualifiedName with Name="*": lint module (e.g., "Sales.*")
	// - QualifiedName: lint specific element
	Target *QualifiedName

	// ModuleOnly is true when targeting a whole module (Module.*)
	ModuleOnly bool

	// Format specifies the output format (text, json, sarif)
	Format LintFormat

	// ShowRules is true for "SHOW LINT RULES" command
	ShowRules bool
}

func (s *LintStmt) isStatement()     {}
func (s *LintStmt) TypeName() string { return "Lint" }
