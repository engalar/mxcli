// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// ValidationFeedbackRule checks for validation feedback actions with empty message templates.
type ValidationFeedbackRule struct{}

// NewValidationFeedbackRule creates a new validation feedback rule.
func NewValidationFeedbackRule() *ValidationFeedbackRule {
	return &ValidationFeedbackRule{}
}

func (r *ValidationFeedbackRule) ID() string                       { return "MPR004" }
func (r *ValidationFeedbackRule) Name() string                     { return "EmptyValidationFeedback" }
func (r *ValidationFeedbackRule) Category() string                 { return "correctness" }
func (r *ValidationFeedbackRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }

func (r *ValidationFeedbackRule) Description() string {
	return "Checks for validation feedback actions with empty message templates (CE0091)"
}

// Check loads each microflow and inspects validation feedback actions for empty templates.
func (r *ValidationFeedbackRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}

	var violations []linter.Violation

	for _, mf := range ctx.Microflows() {
		if ctx.IsExcluded(mf.Module) {
			continue
		}

		fullMF, err := reader.GetMicroflowGen(model.ID(mf.ID))
		if err != nil || fullMF == nil {
			continue
		}
		oc, _ := fullMF.ObjectCollection().(*genMf.MicroflowObjectCollection)
		if oc == nil {
			continue
		}

		walkObjects(oc.ObjectsItems(), mf, r, &violations)
	}

	return violations
}

// walkObjects recursively walks gen ObjectCollection items looking
// for empty validation feedback templates.
func walkObjects(objects []element.Element, mf graphcatalog.MicroflowNode, r *ValidationFeedbackRule, violations *[]linter.Violation) {
	for _, obj := range objects {
		switch act := obj.(type) {
		case *genMf.ActionActivity:
			inner := act.Action()
			if inner == nil {
				continue
			}
			if vf, ok := inner.(*genMf.ValidationFeedbackAction); ok {
				if isEmptyFeedbackTemplate(vf) {
					*violations = append(*violations, linter.Violation{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						Message: fmt.Sprintf("Validation feedback in '%s.%s' has empty message template. "+
							"Mendix requires a non-empty feedback message (CE0091).",
							mf.Module, mf.Name),
						Location: linter.Location{
							Module:       mf.Module,
							DocumentType: "microflow",
							DocumentName: mf.Name,
							DocumentID:   mf.ID,
						},
						Suggestion: "Add a message template to the validation feedback action",
					})
				}
			}
		case *genMf.LoopedActivity:
			if body, ok := act.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && body != nil {
				walkObjects(body.ObjectsItems(), mf, r, violations)
			}
		}
	}
}

// isEmptyFeedbackTemplate reports whether a gen ValidationFeedbackAction
// has no usable feedback text. The template is empty when it's missing,
// not a *texts.Text, has no translations, or all translation texts are
// empty strings.
func isEmptyFeedbackTemplate(vf *genMf.ValidationFeedbackAction) bool {
	tmpl, ok := vf.FeedbackTemplate().(*texts.Text)
	if !ok || tmpl == nil {
		return true
	}
	items := tmpl.TranslationsItems()
	if len(items) == 0 {
		return true
	}
	for _, child := range items {
		tr, ok := child.(*texts.Translation)
		if !ok || tr == nil {
			continue
		}
		if tr.Text() != "" {
			return false
		}
	}
	return true
}
