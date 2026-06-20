// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"github.com/mendixlabs/mxcli/mdl/linter"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// entityFact captures the per-entity properties that the in-memory graph
// catalog does not surface as node properties (persistability and access-rule
// count). These are read from the gen-typed domain models via the deep reader.
type entityFact struct {
	Persistent      bool
	AccessRuleCount int
}

// entityFactsByID walks every domain model through the deep reader and returns a
// map keyed by entity element ID. Rules join this against graphcatalog entity
// nodes (which carry ID/Name/Module/QualifiedName) to recover the richer fields.
// Returns nil if no reader is available.
func entityFactsByID(ctx *linter.LintContext) map[string]entityFact {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}
	dms, err := reader.ListDomainModelsGen()
	if err != nil {
		return nil
	}
	facts := make(map[string]entityFact)
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		for _, item := range dm.EntitiesItems() {
			ent, ok := item.(*genDm.Entity)
			if !ok || ent == nil {
				continue
			}
			facts[string(ent.ID())] = entityFact{
				Persistent:      isPersistentEntity(ent),
				AccessRuleCount: len(ent.AccessRulesItems()),
			}
		}
	}
	return facts
}

// isPersistentEntity reports whether an entity stores data. A base entity is
// persistent when its NoGeneralization carries Persistable=true; a specialized
// entity (with a Generalization parent) inherits persistability from its root,
// which the linter treats as persistent for access-rule purposes.
func isPersistentEntity(ent *genDm.Entity) bool {
	switch g := ent.Generalization().(type) {
	case *genDm.NoGeneralization:
		return g != nil && g.Persistable()
	case *genDm.Generalization:
		// Specialization of another entity — persistable follows the parent.
		return true
	default:
		return false
	}
}
