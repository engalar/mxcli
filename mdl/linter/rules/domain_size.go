// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/linter"
)

// DefaultMaxPersistentEntities is the default maximum number of persistent entities per module.
const DefaultMaxPersistentEntities = 15

// DomainModelSizeRule checks that modules don't have too many persistent entities.
type DomainModelSizeRule struct {
	MaxPersistentEntities int
}

// NewDomainModelSizeRule creates a new domain model size rule with default thresholds.
func NewDomainModelSizeRule() *DomainModelSizeRule {
	return &DomainModelSizeRule{
		MaxPersistentEntities: DefaultMaxPersistentEntities,
	}
}

func (r *DomainModelSizeRule) ID() string                       { return "MPR003" }
func (r *DomainModelSizeRule) Name() string                     { return "DomainModelSize" }
func (r *DomainModelSizeRule) Category() string                 { return "design" }
func (r *DomainModelSizeRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }

func (r *DomainModelSizeRule) Description() string {
	return fmt.Sprintf("Checks that modules have no more than %d persistent entities", r.MaxPersistentEntities)
}

// Check counts persistent entities per module and flags those exceeding the limit.
func (r *DomainModelSizeRule) Check(ctx *linter.LintContext) []linter.Violation {
	// Count persistent entities per module. Persistability is not a graph node
	// property; join graph entity nodes (for module grouping) against gen-typed
	// facts from the deep reader.
	facts := entityFactsByID(ctx)
	counts := make(map[string]int)
	for _, entity := range ctx.Entities() {
		if facts[entity.ID].Persistent {
			counts[entity.Module]++
		}
	}

	var violations []linter.Violation
	for moduleName, count := range counts {
		if count > r.MaxPersistentEntities {
			violations = append(violations, linter.Violation{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				Message: fmt.Sprintf("Module '%s' has %d persistent entities (max %d). Consider splitting into smaller modules.",
					moduleName, count, r.MaxPersistentEntities),
				Location: linter.Location{
					Module:       moduleName,
					DocumentType: "domainmodel",
					DocumentName: "DomainModel",
				},
				Suggestion: fmt.Sprintf("Split module '%s' into smaller modules with focused domain models", moduleName),
			})
		}
	}

	return violations
}
