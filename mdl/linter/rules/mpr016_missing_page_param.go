// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MissingPageParamRule (MPR016) detects ActionButtons that open a page via
// FormAction/PageClientAction but omit one or more required page parameters.
// Left unfixed, Studio Pro reports CE1571 "No argument has been selected for
// parameter '…' and no default is available."
type MissingPageParamRule struct{}

// NewMissingPageParamRule creates a new MissingPageParamRule.
func NewMissingPageParamRule() *MissingPageParamRule {
	return &MissingPageParamRule{}
}

func (r *MissingPageParamRule) ID() string                       { return "MPR016" }
func (r *MissingPageParamRule) Name() string                     { return "MissingRequiredPageParameter" }
func (r *MissingPageParamRule) Category() string                 { return "correctness" }
func (r *MissingPageParamRule) DefaultSeverity() linter.Severity { return linter.SeverityError }
func (r *MissingPageParamRule) Description() string {
	return "ActionButton opens a page via FormAction but omits a required parameter mapping (CE1571 risk)"
}

// pageScanInfo holds page metadata needed for scanning and reporting.
type pageScanInfo struct {
	ID            string
	QualifiedName string
	Module        string
	Name          string
}

// MPR016Fixer is the interface for fixing MPR016 violations.
type MPR016Fixer interface {
	SetPageParameterRequired(pageID model.ID, paramName string, required bool) error
}

// Check scans all pages for ActionButtons whose FormAction/PageClientAction
// targets a page with required parameters that lack corresponding mappings.
func (r *MissingPageParamRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	graph := ctx.Graph()
	if reader == nil || graph == nil {
		return nil
	}

	var containers []pageScanInfo
	for _, p := range ctx.Pages() {
		containers = append(containers, pageScanInfo{
			ID: p.ID, QualifiedName: p.QualifiedName,
			Module: p.Module, Name: p.Name,
		})
	}

	if len(containers) == 0 {
		return nil
	}

	qnToTargetParams := make(map[string]map[string]bool)
	for _, c := range containers {
		rawData, err := reader.GetRawUnit(model.ID(c.ID))
		if err != nil || rawData == nil {
			continue
		}
		params := readPageParams(rawData)
		if len(params) > 0 {
			qnToTargetParams[c.QualifiedName] = params
		}
	}

	var violations []linter.Violation

	for _, c := range containers {
		if ctx.IsExcluded(c.Module) {
			continue
		}

		rawData, err := reader.GetRawUnit(model.ID(c.ID))
		if err != nil || rawData == nil {
			continue
		}

		buttons := findAllFormActionButtons(rawData)
		for _, btn := range buttons {
			btnName := extractStr(btn["Name"])
			if btnName == "" {
				btnName = "(unnamed)"
			}

			action, _ := btn["Action"].(map[string]any)
			if action == nil {
				continue
			}

			rawPS, _ := action["FormSettings"].(map[string]any)
			if rawPS == nil {
				continue
			}

			targetQN := extractStr(rawPS["Form"])
			if targetQN == "" {
				continue
			}

			targetParams, ok := qnToTargetParams[targetQN]
			if !ok {
				continue
			}

			mappedParams := readActionMappedParams(rawPS)

			for paramName, required := range targetParams {
				if !required {
					continue
				}
				fullQN := targetQN + "." + paramName
				if mappedParams[fullQN] {
					continue
				}

				// Find target page ID for fix references.
				targetID := ""
				for _, tc := range containers {
					if tc.QualifiedName == targetQN {
						targetID = tc.ID
						break
					}
				}

				violations = append(violations, linter.Violation{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					Message: fmt.Sprintf(
						"ActionButton '%s' in '%s' opens '%s' without mapping the required parameter '%s' (CE1571)",
						btnName, c.QualifiedName, targetQN, paramName,
					),
					Location: linter.Location{
						Module:       c.Module,
						DocumentType: "page",
						DocumentName: c.Name,
						DocumentID:   c.ID,
					},
					Suggestion: fmt.Sprintf(
						"Add a parameter mapping for '%s' or set IsRequired=false on the target page parameter",
						paramName,
					),
					// Store target info for potential auto-fix.
					Extra: map[string]any{
						"targetPageID":    targetID,
						"targetQualified": targetQN,
						"paramName":       paramName,
					},
				})
			}
		}
	}

	return violations
}

// readPageParams reads the Parameters array from a raw page BSON map.
func readPageParams(rawData map[string]any) map[string]bool {
	params := make(map[string]bool)
	for _, rawElem := range getBsonArray(rawData["Parameters"]) {
		elem, ok := rawElem.(map[string]any)
		if !ok {
			continue
		}
		typ := extractStr(elem["$Type"])
		if typ != "Forms$PageParameter" {
			continue
		}
		name := extractStr(elem["Name"])
		if name == "" {
			continue
		}
		required := false
		if v, ok := elem["IsRequired"]; ok {
			if b, ok := v.(bool); ok {
				required = b
			}
		}
		params[name] = required
	}
	return params
}

// readActionMappedParams reads the ParameterMappings from a FormSettings map.
func readActionMappedParams(rawPS map[string]any) map[string]bool {
	mapped := make(map[string]bool)
	for _, rawPM := range getBsonArray(rawPS["ParameterMappings"]) {
		pm, ok := rawPM.(map[string]any)
		if !ok {
			continue
		}
		if paramQN := extractStr(pm["Parameter"]); paramQN != "" {
			mapped[paramQN] = true
		}
	}
	return mapped
}

// FixMPR016 applies the auto-fix for a single MPR016 violation:
// it sets IsRequired=false on the target page parameter.
func FixMPR016(violation linter.Violation, fixer MPR016Fixer) error {
	extra, ok := violation.Extra.(map[string]any)
	if !ok {
		return fmt.Errorf("no extra data for violation")
	}
	targetID, _ := extra["targetPageID"].(string)
	paramName, _ := extra["paramName"].(string)
	if targetID == "" || paramName == "" {
		return fmt.Errorf("missing target info in violation extra data")
	}
	return fixer.SetPageParameterRequired(model.ID(targetID), paramName, false)
}

// findAllFormActionButtons deep-searches a BSON document for ActionButtons
// whose Action is a FormAction or PageClientAction.
func findAllFormActionButtons(v any) []map[string]any {
	var result []map[string]any
	walkValue(v, func(m map[string]any) {
		typ := extractStr(m["$Type"])
		if typ != "Forms$ActionButton" {
			return
		}
		action, ok := m["Action"].(map[string]any)
		if !ok {
			return
		}
		atyp := extractStr(action["$Type"])
		if atyp != "Forms$FormAction" && atyp != "Forms$PageClientAction" {
			return
		}
		result = append(result, m)
	})
	return result
}

// walkValue recursively walks any BSON-decoded value, calling fn for each map
// or bson.D (treating both as BSON documents). Handles map[string]any, bson.D,
// []any, and bson.A — the four forms that mongo-driver v2 produces when
// decoding BSON into map[string]any.
//
// NOTE: bson.D must be converted to map[string]any before calling fn because
// fn accepts map[string]any. Without this conversion, pluggable widgets
// stored as bson.D (from bson.Unmarshal into map[string]any) would be skipped.
func walkValue(v any, fn func(map[string]any)) {
	switch val := v.(type) {
	case map[string]any:
		fn(val)
		for _, child := range val {
			walkValue(child, fn)
		}
	case bson.D:
		// Convert to map[string]any and call fn (treat bson.D like map).
		m := make(map[string]any, len(val))
		for _, elem := range val {
			m[elem.Key] = elem.Value
		}
		fn(m)
		for _, elem := range val {
			walkValue(elem.Value, fn)
		}
	case []any:
		for _, child := range val {
			walkValue(child, fn)
		}
	case bson.A:
		for _, child := range val {
			walkValue(child, fn)
		}
	}
}
