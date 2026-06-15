// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ExclusiveSplitCaptionRule flags exclusive splits without meaningful captions.
type ExclusiveSplitCaptionRule struct{}

func NewExclusiveSplitCaptionRule() *ExclusiveSplitCaptionRule {
	return &ExclusiveSplitCaptionRule{}
}

func (r *ExclusiveSplitCaptionRule) ID() string                       { return "CONV012" }
func (r *ExclusiveSplitCaptionRule) Name() string                     { return "ExclusiveSplitCaption" }
func (r *ExclusiveSplitCaptionRule) Category() string                 { return "quality" }
func (r *ExclusiveSplitCaptionRule) DefaultSeverity() linter.Severity { return linter.SeverityInfo }

func (r *ExclusiveSplitCaptionRule) Description() string {
	return "Exclusive splits should have a meaningful caption describing the decision"
}

func (r *ExclusiveSplitCaptionRule) Check(ctx *linter.LintContext) []linter.Violation {
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

		findEmptySplitCaptions(oc.ObjectsItems(), mf, r, &violations)
	}

	return violations
}

func findEmptySplitCaptions(objects []element.Element, mf graphcatalog.MicroflowNode, r *ExclusiveSplitCaptionRule, violations *[]linter.Violation) {
	for _, obj := range objects {
		switch act := obj.(type) {
		case *genMf.ExclusiveSplit:
			caption := strings.TrimSpace(act.Caption())
			if caption == "" {
				*violations = append(*violations, linter.Violation{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					Message: fmt.Sprintf("Exclusive split in '%s.%s' has no caption. "+
						"Add a question or description to clarify the decision.",
						mf.Module, mf.Name),
					Location: linter.Location{
						Module:       mf.Module,
						DocumentType: "microflow",
						DocumentName: mf.Name,
						DocumentID:   mf.ID,
					},
					Suggestion: "Set a caption that describes the decision, e.g., 'Is the order valid?'",
				})
			}
		case *genMf.LoopedActivity:
			if body, ok := act.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && body != nil {
				findEmptySplitCaptions(body.ObjectsItems(), mf, r, violations)
			}
		}
	}
}
