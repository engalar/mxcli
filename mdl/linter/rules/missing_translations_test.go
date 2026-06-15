// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/linter"
)

// MissingTranslations is disabled during the graphcatalog migration (no strings
// data source in the in-memory graph). The rule must be a safe no-op until a
// translation reader is wired in.
func TestMissingTranslations_DisabledReturnsNil(t *testing.T) {
	rule := NewMissingTranslationsRule()
	if v := rule.Check(linter.NewLintContext(nil, nil)); v != nil {
		t.Errorf("expected nil (rule disabled), got %d violations", len(v))
	}
}

func TestMissingTranslations_Metadata(t *testing.T) {
	rule := NewMissingTranslationsRule()
	if rule.ID() != "QUAL005" {
		t.Errorf("ID = %q, want QUAL005", rule.ID())
	}
	if rule.Name() != "MissingTranslations" {
		t.Errorf("Name = %q, want MissingTranslations", rule.Name())
	}
}
