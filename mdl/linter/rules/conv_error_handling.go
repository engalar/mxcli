// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// errorHandled reports whether the given gen action's error handling is
// already "Custom" or "Custom without rollback" — the two settings the
// CONV013 rule treats as adequate. Returns false for any other value
// or for actions that don't expose ErrorHandlingType().
//
// Gen renames the legacy ErrorHandlingTypeCustomWithoutRollback (lower-case
// 'b' in 'Rollback') to ErrorHandlingTypeCustomWithoutRollBack (upper-case
// 'B'); the underlying string value is unchanged ("CustomWithoutRollBack").
func errorHandled(action element.Element) bool {
	type ehGetter interface {
		ErrorHandlingType() string
	}
	if g, ok := action.(ehGetter); ok {
		eh := g.ErrorHandlingType()
		return eh == genMf.ErrorHandlingTypeCustom || eh == genMf.ErrorHandlingTypeCustomWithoutRollBack
	}
	return false
}

// errorHandlingType returns the action's ErrorHandlingType setting,
// or "" if the element is not an action with that property. Used by
// CONV014 (Continue swallows errors) which needs to compare against
// the "Continue" enum value across all action types and LoopedActivity.
func errorHandlingType(e element.Element) string {
	type ehGetter interface {
		ErrorHandlingType() string
	}
	if g, ok := e.(ehGetter); ok {
		return g.ErrorHandlingType()
	}
	return ""
}

// --- CONV013: ErrorHandlingOnCalls ---

// ErrorHandlingOnCallsRule flags external call actions (REST, web service, Java) without custom error handling.
type ErrorHandlingOnCallsRule struct{}

func NewErrorHandlingOnCallsRule() *ErrorHandlingOnCallsRule {
	return &ErrorHandlingOnCallsRule{}
}

func (r *ErrorHandlingOnCallsRule) ID() string                       { return "CONV013" }
func (r *ErrorHandlingOnCallsRule) Name() string                     { return "ErrorHandlingOnCalls" }
func (r *ErrorHandlingOnCallsRule) Category() string                 { return "quality" }
func (r *ErrorHandlingOnCallsRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }

func (r *ErrorHandlingOnCallsRule) Description() string {
	return "External service calls (REST, web service, Java action) should have custom error handling"
}

func (r *ErrorHandlingOnCallsRule) Check(ctx *linter.LintContext) []linter.Violation {
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

		findUnhandledCalls(oc.ObjectsItems(), mf, r, &violations)
	}

	return violations
}

func findUnhandledCalls(objects []element.Element, mf graphcatalog.MicroflowNode, r *ErrorHandlingOnCallsRule, violations *[]linter.Violation) {
	for _, obj := range objects {
		switch act := obj.(type) {
		case *genMf.ActionActivity:
			inner := act.Action()
			if inner == nil {
				continue
			}
			actionName := ""
			switch inner.(type) {
			case *genMf.RestCallAction:
				actionName = "REST call"
			case *genMf.WebServiceCallAction:
				actionName = "Web service call"
			case *genMf.JavaActionCallAction:
				actionName = "Java action call"
			default:
				continue
			}

			if errorHandled(inner) {
				continue
			}
			*violations = append(*violations, linter.Violation{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message: fmt.Sprintf("%s in '%s.%s' uses '%s' error handling instead of Custom.",
					actionName, mf.Module, mf.Name, errorHandlingType(inner)),
				Location: linter.Location{
					Module:       mf.Module,
					DocumentType: "microflow",
					DocumentName: mf.Name,
					DocumentID:   mf.ID,
				},
				Suggestion: "Set error handling to 'Custom with rollback' and add an error handler flow",
			})
		case *genMf.LoopedActivity:
			if body, ok := act.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && body != nil {
				findUnhandledCalls(body.ObjectsItems(), mf, r, violations)
			}
		}
	}
}

// --- CONV014: NoContinueErrorHandling ---

// NoContinueErrorHandlingRule flags activities with "Continue" error handling, which silently swallows errors.
type NoContinueErrorHandlingRule struct{}

func NewNoContinueErrorHandlingRule() *NoContinueErrorHandlingRule {
	return &NoContinueErrorHandlingRule{}
}

func (r *NoContinueErrorHandlingRule) ID() string       { return "CONV014" }
func (r *NoContinueErrorHandlingRule) Name() string     { return "NoContinueErrorHandling" }
func (r *NoContinueErrorHandlingRule) Category() string { return "quality" }
func (r *NoContinueErrorHandlingRule) DefaultSeverity() linter.Severity {
	return linter.SeverityWarning
}

func (r *NoContinueErrorHandlingRule) Description() string {
	return "Activities should not use 'Continue' error handling which silently swallows errors"
}

func (r *NoContinueErrorHandlingRule) Check(ctx *linter.LintContext) []linter.Violation {
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

		findContinueErrorHandling(oc.ObjectsItems(), mf, r, &violations)
	}

	return violations
}

func findContinueErrorHandling(objects []element.Element, mf graphcatalog.MicroflowNode, r *NoContinueErrorHandlingRule, violations *[]linter.Violation) {
	for _, obj := range objects {
		switch act := obj.(type) {
		case *genMf.ActionActivity:
			// In gen, ErrorHandlingType lives on the inner action (not the activity).
			if errorHandlingType(act.Action()) == genMf.ErrorHandlingTypeContinue {
				caption := act.Caption()
				if caption == "" {
					caption = "(unnamed activity)"
				}
				*violations = append(*violations, linter.Violation{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					Message: fmt.Sprintf("Activity '%s' in '%s.%s' uses 'Continue' error handling, which silently swallows errors.",
						caption, mf.Module, mf.Name),
					Location: linter.Location{
						Module:       mf.Module,
						DocumentType: "microflow",
						DocumentName: mf.Name,
						DocumentID:   mf.ID,
					},
					Suggestion: "Change error handling to 'Custom with rollback' or 'Abort' to properly handle errors",
				})
			}
		case *genMf.LoopedActivity:
			if act.ErrorHandlingType() == genMf.ErrorHandlingTypeContinue {
				// Gen LoopedActivity has no Caption property (the
				// legacy sdk's BaseActivity.Caption isn't carried
				// across); fall back to Documentation, then a
				// placeholder, so the message stays readable.
				caption := act.Documentation()
				if caption == "" {
					caption = "(unnamed loop)"
				}
				*violations = append(*violations, linter.Violation{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					Message: fmt.Sprintf("Loop '%s' in '%s.%s' uses 'Continue' error handling, which silently swallows errors.",
						caption, mf.Module, mf.Name),
					Location: linter.Location{
						Module:       mf.Module,
						DocumentType: "microflow",
						DocumentName: mf.Name,
						DocumentID:   mf.ID,
					},
					Suggestion: "Change error handling to 'Custom with rollback' or 'Abort' to properly handle errors",
				})
			}
			if body, ok := act.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && body != nil {
				findContinueErrorHandling(body.ObjectsItems(), mf, r, violations)
			}
		}
	}
}
