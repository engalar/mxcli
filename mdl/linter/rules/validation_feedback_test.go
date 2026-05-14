// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// makeText builds a gen *texts.Text with the supplied (lang -> body)
// translations. Order doesn't matter for the empty-template check.
func makeText(translations map[string]string) *texts.Text {
	t := texts.NewText()
	for lang, body := range translations {
		tr := texts.NewTranslation()
		tr.SetLanguageCode(lang)
		tr.SetText(body)
		t.AddTranslations(tr)
	}
	return t
}

// vfWith builds a ValidationFeedbackAction whose FeedbackTemplate
// is the given gen text element (or nil to model the absent case).
func vfWith(template element.Element) *genMf.ValidationFeedbackAction {
	vf := genMf.NewValidationFeedbackAction()
	if template != nil {
		vf.SetFeedbackTemplate(template)
	}
	return vf
}

// activityWith wraps an inner action in a freshly-built ActionActivity.
func activityWith(inner element.Element) *genMf.ActionActivity {
	a := genMf.NewActionActivity()
	a.SetAction(inner)
	return a
}

func TestIsEmptyTemplate_Nil(t *testing.T) {
	if !isEmptyFeedbackTemplate(vfWith(nil)) {
		t.Error("expected true for nil template")
	}
}

func TestIsEmptyTemplate_EmptyTranslations(t *testing.T) {
	if !isEmptyFeedbackTemplate(vfWith(makeText(map[string]string{}))) {
		t.Error("expected true for empty translations")
	}
}

func TestIsEmptyTemplate_AllEmpty(t *testing.T) {
	if !isEmptyFeedbackTemplate(vfWith(makeText(map[string]string{"en_US": "", "nl_NL": ""}))) {
		t.Error("expected true when all translations empty")
	}
}

func TestIsEmptyTemplate_HasContent(t *testing.T) {
	if isEmptyFeedbackTemplate(vfWith(makeText(map[string]string{"en_US": "Please fill in this field"}))) {
		t.Error("expected false when translation has content")
	}
}

func TestWalkObjects_EmptyValidation(t *testing.T) {
	objects := []element.Element{activityWith(vfWith(nil))}

	var violations []linter.Violation
	r := NewValidationFeedbackRule()
	walkObjects(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "MPR004" {
		t.Errorf("expected MPR004, got %s", violations[0].RuleID)
	}
}

func TestWalkObjects_ValidFeedback(t *testing.T) {
	objects := []element.Element{
		activityWith(vfWith(makeText(map[string]string{"en_US": "Required"}))),
	}

	var violations []linter.Violation
	r := NewValidationFeedbackRule()
	walkObjects(objects, testMicroflow(), r, &violations)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestWalkObjects_InsideLoop(t *testing.T) {
	loopBody := genMf.NewMicroflowObjectCollection()
	loopBody.AddObjects(activityWith(vfWith(nil)))

	loop := genMf.NewLoopedActivity()
	loop.SetObjectCollection(loopBody)

	objects := []element.Element{loop}

	var violations []linter.Violation
	r := NewValidationFeedbackRule()
	walkObjects(objects, testMicroflow(), r, &violations)

	if len(violations) != 1 {
		t.Errorf("expected 1 violation inside loop, got %d", len(violations))
	}
}

func TestValidationFeedbackRule_Metadata(t *testing.T) {
	r := NewValidationFeedbackRule()
	if r.ID() != "MPR004" {
		t.Errorf("ID = %q, want MPR004", r.ID())
	}
	if r.Category() != "correctness" {
		t.Errorf("Category = %q, want correctness", r.Category())
	}
}
