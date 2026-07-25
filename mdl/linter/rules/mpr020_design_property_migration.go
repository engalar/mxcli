package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
)

// DesignPropMapping records a single old->new name mapping for a design property.
type DesignPropMapping struct {
	OldName string
	NewName string
	OldOpts map[string]string // old option name -> new option name
}

// DesignPropertyMigrationRule (MPR020) detects widgets that use old design
// property names which have been renamed in the theme. This causes CE6087:
// "Design properties have been renamed in your theme and need to be updated."
//
// The caller must set Mappings before calling Check, or use one of the
// constructor helpers.
type DesignPropertyMigrationRule struct {
	Mappings []DesignPropMapping
}

// NewDesignPropertyMigrationRule creates a new MPR020 rule.
func NewDesignPropertyMigrationRule() *DesignPropertyMigrationRule {
	return &DesignPropertyMigrationRule{}
}

// NewDesignPropertyMigrationRuleWithMappings creates a rule with pre-built
// old->new name mappings from the theme's design-properties.json.
func NewDesignPropertyMigrationRuleWithMappings(mappings []DesignPropMapping) *DesignPropertyMigrationRule {
	return &DesignPropertyMigrationRule{Mappings: mappings}
}

func (r *DesignPropertyMigrationRule) ID() string                       { return "MPR020" }
func (r *DesignPropertyMigrationRule) Name() string                     { return "DesignPropertyMigration" }
func (r *DesignPropertyMigrationRule) Category() string                 { return "correctness" }
func (r *DesignPropertyMigrationRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }
func (r *DesignPropertyMigrationRule) Description() string {
	return "Detects widgets using old design property names from a previous theme version -- " +
		"run with --fix to update to current names and resolve CE6087"
}

// Check walks all units, finds DesignPropertyValue entries with old names,
// and reports violations. Requires Mappings to be set.
func (r *DesignPropertyMigrationRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil || len(r.Mappings) == 0 {
		return nil
	}

	oldNames := buildOldNameSet(r.Mappings)
	unitIDs, err := reader.ListAllUnitIDs()
	if err != nil || unitIDs == nil {
		return nil
	}

	var violations []linter.Violation
	for _, uid := range unitIDs {
		rawData, err := reader.GetRawUnit(model.ID(uid))
		if err != nil || rawData == nil {
			continue
		}
		docType, _ := rawData["$Type"].(string)
		if !isWidgetContainer(docType) {
			continue
		}
		docName, _ := rawData["Name"].(string)
		label := containerTypeLabel(docType)

		if findOldDesignProps(rawData, oldNames) {
			violations = append(violations, linter.Violation{
				RuleID:   "MPR020",
				Severity: linter.SeverityWarning,
				Message: fmt.Sprintf(
					"%s %s has design properties with old names -- run --fix to update (resolves CE6087)",
					label, docName,
				),
				Location: linter.Location{
					DocumentType: label,
					DocumentName: docName,
					DocumentID:   uid,
				},
				Suggestion: "Run mxcli lint --fix to update design property names",
				Extra: map[string]any{
					"unitID": uid,
				},
			})
		}
	}
	return violations
}

func buildOldNameSet(mappings []DesignPropMapping) map[string]bool {
	s := make(map[string]bool, len(mappings))
	for _, m := range mappings {
		s[m.OldName] = true
	}
	return s
}

// findOldDesignProps recursively walks raw BSON maps and checks if any
// DesignPropertyValue.Key matches an old name. Returns true on first match.
func findOldDesignProps(data map[string]any, oldNames map[string]bool) bool {
	if dps, ok := data["DesignProperties"]; ok {
		if arr, ok := dps.([]any); ok {
			for _, item := range arr {
				if dpDoc, ok := item.(map[string]any); ok {
					if key, ok := dpDoc["Key"].(string); ok {
						if oldNames[key] {
							return true
						}
					}
				}
			}
		}
	}
	for _, v := range data {
		switch val := v.(type) {
		case map[string]any:
			if findOldDesignProps(val, oldNames) {
				return true
			}
		case []any:
			for _, elem := range val {
				if sub, ok := elem.(map[string]any); ok {
					if findOldDesignProps(sub, oldNames) {
						return true
					}
				}
			}
		}
	}
	return false
}

// MPR020Fixer provides the backend methods needed to apply CE6087 fixes.
type MPR020Fixer interface {
	FixDesignPropertyNames(unitID model.ID, propRenames, optRenames map[string]string) error
}

// FixMPR020 applies the auto-fix for a single MPR020 violation by renaming
// all old design property names in the affected unit.
func FixMPR020(violation linter.Violation, fixer MPR020Fixer, propRenames, optRenames map[string]string) error {
	extra, ok := violation.Extra.(map[string]any)
	if !ok {
		return fmt.Errorf("MPR020: no extra data in violation")
	}
	unitID, _ := extra["unitID"].(string)
	if unitID == "" {
		return fmt.Errorf("MPR020: no unitID in violation")
	}
	return fixer.FixDesignPropertyNames(model.ID(unitID), propRenames, optRenames)
}
