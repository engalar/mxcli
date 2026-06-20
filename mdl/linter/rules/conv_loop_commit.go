// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// NoCommitInLoopRule flags commit actions inside loops, which cause N+1 database operations.
type NoCommitInLoopRule struct{}

func NewNoCommitInLoopRule() *NoCommitInLoopRule { return &NoCommitInLoopRule{} }

func (r *NoCommitInLoopRule) ID() string                       { return "CONV011" }
func (r *NoCommitInLoopRule) Name() string                     { return "NoCommitInLoop" }
func (r *NoCommitInLoopRule) Category() string                 { return "performance" }
func (r *NoCommitInLoopRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }

func (r *NoCommitInLoopRule) Description() string {
	return "Commit actions should not be inside loops (N+1 performance issue)"
}

func (r *NoCommitInLoopRule) Check(ctx *linter.LintContext) []linter.Violation {
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

		findCommitsInLoops(oc.ObjectsItems(), mf, r, &violations, false)
	}

	return violations
}

// findCommitsInLoops walks a gen ObjectCollection looking for
// CommitAction inner actions inside an ActionActivity that lives
// directly or transitively inside a LoopedActivity. Note: gen renames
// "CommitObjectsAction" to "CommitAction" but the BSON storage type
// stays "CommitAction" (see CLAUDE.md storage-name table).
func findCommitsInLoops(objects []element.Element, mf graphcatalog.MicroflowNode, r *NoCommitInLoopRule, violations *[]linter.Violation, insideLoop bool) {
	for _, obj := range objects {
		switch act := obj.(type) {
		case *genMf.ActionActivity:
			if !insideLoop {
				continue
			}
			inner := act.Action()
			if inner == nil {
				continue
			}
			if _, ok := inner.(*genMf.CommitAction); ok {
				*violations = append(*violations, linter.Violation{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					Message: fmt.Sprintf("Microflow '%s.%s' has a Commit action inside a loop. "+
						"This causes N+1 database operations.",
						mf.Module, mf.Name),
					Location: linter.Location{
						Module:       mf.Module,
						DocumentType: "microflow",
						DocumentName: mf.Name,
						DocumentID:   mf.ID,
					},
					Suggestion: "Move the commit outside the loop, or collect objects in a list and commit once after the loop",
				})
			}
		case *genMf.LoopedActivity:
			if body, ok := act.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && body != nil {
				findCommitsInLoops(body.ObjectsItems(), mf, r, violations, true)
			}
		}
	}
}
