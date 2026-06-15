// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"github.com/mendixlabs/mxcli/mdl/linter"
)

// MissingTranslationsRule checks for elements that have translations in some
// languages but not all languages used in the project.
type MissingTranslationsRule struct{}

// NewMissingTranslationsRule creates a new missing translations rule.
func NewMissingTranslationsRule() *MissingTranslationsRule {
	return &MissingTranslationsRule{}
}

func (r *MissingTranslationsRule) ID() string                       { return "QUAL005" }
func (r *MissingTranslationsRule) Name() string                     { return "MissingTranslations" }
func (r *MissingTranslationsRule) Category() string                 { return "quality" }
func (r *MissingTranslationsRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }

func (r *MissingTranslationsRule) Description() string {
	return "Checks for translatable strings that are missing translations in one or more project languages"
}

// Check runs the missing translations check.
//
// TODO: disabled during the graphcatalog migration. This rule needs a project
// strings/translations data source (formerly the SQLite `strings` FTS table
// populated by REFRESH CATALOG FULL), which the in-memory graph catalog does
// not surface. Re-enable once a translation reader is added to the linter
// context.
func (r *MissingTranslationsRule) Check(_ *linter.LintContext) []linter.Violation {
	return nil
}
