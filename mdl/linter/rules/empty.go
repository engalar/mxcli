// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// EmptyMicroflowRule checks for microflows with no activities.
type EmptyMicroflowRule struct{}

// NewEmptyMicroflowRule creates a new empty microflow rule.
func NewEmptyMicroflowRule() *EmptyMicroflowRule {
	return &EmptyMicroflowRule{}
}

func (r *EmptyMicroflowRule) ID() string                       { return "MPR002" }
func (r *EmptyMicroflowRule) Name() string                     { return "EmptyMicroflow" }
func (r *EmptyMicroflowRule) Category() string                 { return "quality" }
func (r *EmptyMicroflowRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }

func (r *EmptyMicroflowRule) Description() string {
	return "Checks for microflows that have no activities"
}

// Check runs the empty microflow check.
//
// Activity count is not a graph node property; the microflow body is read via
// the deep reader. A microflow is "empty" when its object collection holds no
// objects (start/end events live outside the collection).
func (r *EmptyMicroflowRule) Check(ctx *linter.LintContext) []linter.Violation {
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
		if oc != nil && len(oc.ObjectsItems()) > 0 {
			continue
		}

		violations = append(violations, linter.Violation{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("Microflow '%s' has no activities", mf.Name),
			Location: linter.Location{
				Module:       mf.Module,
				DocumentType: "microflow",
				DocumentName: mf.Name,
				DocumentID:   mf.ID,
			},
			Suggestion: "Add activities or remove unused microflow",
		})
	}

	return violations
}
