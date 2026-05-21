// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// BrokenMFParamRefRule (MPR015) detects MicroflowCallParameterMapping entries
// whose ParameterQualifiedName references a microflow parameter that no longer
// exists. Left unfixed, Studio Pro reports CE1613 "Selected element no longer exists".
type BrokenMFParamRefRule struct{}

// NewBrokenMFParamRefRule creates a new BrokenMFParamRefRule.
func NewBrokenMFParamRefRule() *BrokenMFParamRefRule { return &BrokenMFParamRefRule{} }

func (r *BrokenMFParamRefRule) ID() string                       { return "MPR015" }
func (r *BrokenMFParamRefRule) Name() string                     { return "BrokenMicroflowParameterRef" }
func (r *BrokenMFParamRefRule) Category() string                 { return "MPR" }
func (r *BrokenMFParamRefRule) DefaultSeverity() linter.Severity { return linter.SeverityError }
func (r *BrokenMFParamRefRule) Description() string {
	return "MicroflowCallParameterMapping references a parameter that no longer exists (CE1613 risk)"
}

// Check runs MPR015 across all project microflows.
//
// Two-pass algorithm:
//  1. Collect every microflow from the catalog, fetch its gen-typed body, and
//     build a map of qualifiedName → set-of-parameter-names.
//  2. Walk each microflow's activity tree and flag any MicroflowCallAction
//     whose ParameterMappings reference a name absent from the target's set.
func (r *BrokenMFParamRefRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}

	type mfEntry struct {
		meta   linter.Microflow
		genObj *genMf.Microflow
	}

	// Pass 1: load all microflows and index their parameter names.
	var allEntries []mfEntry
	paramSets := make(map[string]map[string]bool) // qualifiedName → param names

	for mf := range ctx.Microflows() {
		if ctx.IsExcluded(mf.ModuleName) {
			continue
		}
		full, err := reader.GetMicroflowGen(model.ID(mf.ID))
		if err != nil || full == nil {
			continue
		}
		allEntries = append(allEntries, mfEntry{mf, full})
		paramSets[mf.QualifiedName] = mpr015CollectParams(full)
	}

	// Pass 2: check every call site for broken parameter references.
	var violations []linter.Violation
	for _, entry := range allEntries {
		oc, ok := entry.genObj.ObjectCollection().(*genMf.MicroflowObjectCollection)
		if !ok || oc == nil {
			continue
		}
		mpr015CheckObjects(oc.ObjectsItems(), entry.meta, paramSets, r, &violations)
	}
	return violations
}

// mpr015CollectParams returns the set of parameter names declared by mf.
func mpr015CollectParams(mf *genMf.Microflow) map[string]bool {
	out := make(map[string]bool)
	oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return out
	}
	for _, obj := range oc.ObjectsItems() {
		if param, ok := obj.(*genMf.MicroflowParameter); ok {
			if n := param.Name(); n != "" {
				out[n] = true
			}
		}
	}
	return out
}

// mpr015CheckObjects walks the activity tree rooted at objects and appends
// violations for any MicroflowCallParameterMapping that references a missing param.
func mpr015CheckObjects(
	objects []element.Element,
	mf linter.Microflow,
	paramSets map[string]map[string]bool,
	r *BrokenMFParamRefRule,
	violations *[]linter.Violation,
) {
	for _, obj := range objects {
		switch act := obj.(type) {
		case *genMf.MicroflowCallAction:
			call, ok := act.MicroflowCall().(*genMf.MicroflowCall)
			if !ok || call == nil {
				continue
			}
			targetQN := call.MicroflowQualifiedName()
			targetParams, known := paramSets[targetQN]
			if !known {
				// Target not in project catalog (marketplace/runtime module) — skip.
				continue
			}
			prefix := targetQN + "."
			for _, pmElem := range call.ParameterMappingsItems() {
				pm, ok := pmElem.(*genMf.MicroflowCallParameterMapping)
				if !ok || pm == nil {
					continue
				}
				pqn := pm.ParameterQualifiedName()
				if !strings.HasPrefix(pqn, prefix) {
					// Malformed or unrelated mapping; skip gracefully.
					continue
				}
				paramName := pqn[len(prefix):]
				if !targetParams[paramName] {
					*violations = append(*violations, linter.Violation{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						Message: fmt.Sprintf(
							"microflow %s.%s calls %s with parameter mapping %q which no longer exists (CE1613 risk)",
							mf.ModuleName, mf.Name, targetQN, pqn,
						),
						Location: linter.Location{
							Module:       mf.ModuleName,
							DocumentType: "microflow",
							DocumentName: mf.Name,
							DocumentID:   mf.ID,
						},
						Suggestion: fmt.Sprintf(
							"Remove or update the mapping for %q in the call to %s, or restore the missing parameter.",
							paramName, targetQN,
						),
					})
				}
			}

		case *genMf.LoopedActivity:
			if body, ok := act.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && body != nil {
				mpr015CheckObjects(body.ObjectsItems(), mf, paramSets, r, violations)
			}
		}
	}
}
