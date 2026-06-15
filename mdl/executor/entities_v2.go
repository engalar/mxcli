// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 A1+: gen-typed read functions for the domainmodel domain.
// Uses listDomainModelsWithContainerGen (helpers_domainmodels_gen.go)
// to walk every module's DomainModel without re-querying. The cache is
// session-scoped; write paths must call invalidateDomainModelsCache.
//
// Per CLAUDE.md "Map iteration is deterministic" rule — every collection
// rendered for output sorts keys / slice contents before iterating to
// keep diffs stable across runs.

package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// entityKindForGen classifies an Entity per its Source / Generalization
// / IsRemote properties, mirroring the legacy listEntities switch in
// cmd_entities_describe.go.
//
// Returns:
//   - "View" — OQL view source
//   - "External" — remote entity (REST / OData / database connector)
//   - "Non-Persistent" — no source, no generalization, persistable=false
//   - "Persistent" — anything else (the default for typed app entities)
func entityKindForGen(entity *genDm.Entity) string {
	if entity == nil {
		return ""
	}
	if src := entity.Source(); src != nil {
		switch src.TypeName() {
		case "DomainModels$OqlViewEntitySource", "DomainModels$ViewEntitySource":
			return "View"
		case "DomainModels$RemoteEntitySource",
			"DomainModels$MaterializedRemoteEntitySource",
			"DomainModels$QueryBasedRemoteEntitySource":
			return "External"
		}
	}
	if entity.IsRemote() ||
		entity.RemoteSource() != "" ||
		entity.RemoteSourceDocumentQualifiedName() != "" {
		return "External"
	}
	if g, ok := entity.Generalization().(*genDm.NoGeneralization); ok {
		if !g.Persistable() {
			return "Non-Persistent"
		}
		return "Persistent"
	}
	// Has a *Generalization — entity inherits from a parent. The legacy
	// path also reported these as "Persistent" since a generalized entity
	// participates in the same persistability chain as its root.
	return "Persistent"
}

// entityGeneralizationQNGen returns the qualified name of the parent
// entity (extends), or "" when the entity has no generalization. Mirrors
// sdk-side entity.GeneralizationRef.
func entityGeneralizationQNGen(entity *genDm.Entity) string {
	if entity == nil {
		return ""
	}
	g, ok := entity.Generalization().(*genDm.Generalization)
	if !ok {
		return ""
	}
	return g.GeneralizationQualifiedName()
}

// associationCountsGen builds a map from Entity ID to the number of
// associations referencing that entity (as either FROM or TO endpoint).
// Mirrors the assocCounts loop in legacy listEntities.
func associationCountsGen(pairs []DomainModelGenWithContainer) map[model.ID]int {
	counts := make(map[model.ID]int)
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			parent := model.ID(assoc.ParentRefID())
			child := model.ID(assoc.ChildRefID())
			if parent != "" {
				counts[parent]++
			}
			if child != "" {
				counts[child]++
			}
		}
	}
	return counts
}

// listEntitiesGen handles SHOW ENTITIES on the gen-typed read path
// (Stage 3.3.4 A1 — replaces listEntities in cmd_entities_describe.go).
func listEntitiesGen(ctx *ExecContext, moduleName string) error {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	// Build module ID -> name lookup (single query).
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}

	assocCounts := associationCountsGen(pairs)

	// Collect System entities referenced via generalizations across the
	// project. Studio Pro shows these in SHOW ENTITIES as "System" rows
	// with placeholder counts.
	systemEntitiesSet := make(map[string]bool)
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			gen := entityGeneralizationQNGen(entity)
			if gen != "" && strings.HasPrefix(gen, "System.") {
				systemEntitiesSet[gen] = true
			}
		}
	}

	type row struct {
		qualifiedName  string
		entityType     string
		generalization string
		attrs          int
		assocs         int
		validations    int
		indexes        int
		events         int
		accessRules    int
	}
	var rows []row

	if moduleName == "" || moduleName == "System" {
		// Sorted iteration over System entities (deterministic).
		systemNames := make([]string, 0, len(systemEntitiesSet))
		for n := range systemEntitiesSet {
			systemNames = append(systemNames, n)
		}
		sort.Strings(systemNames)
		for _, n := range systemNames {
			rows = append(rows, row{
				qualifiedName: n,
				entityType:    "System",
				attrs:         -1,
				assocs:        -1,
				validations:   -1,
				indexes:       -1,
				events:        -1,
				accessRules:   -1,
			})
		}
	}

	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			rows = append(rows, row{
				qualifiedName:  modName + "." + entity.Name(),
				entityType:     entityKindForGen(entity),
				generalization: entityGeneralizationQNGen(entity),
				attrs:          len(entity.AttributesItems()),
				assocs:         assocCounts[model.ID(entity.ID())],
				validations:    len(entity.ValidationRulesItems()),
				indexes:        len(entity.IndexesItems()),
				events:         len(entity.EventHandlersItems()),
				accessRules:    len(entity.AccessRulesItems()),
			})
		}
	}

	hasGeneralizations := false
	for _, r := range rows {
		if r.generalization != "" {
			hasGeneralizations = true
			break
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	columns := []string{"Entity", "Type"}
	if hasGeneralizations {
		columns = append(columns, "Extends")
	}
	columns = append(columns, "Attrs", "Assocs", "Validations", "Indexes", "Events", "AccessRules")

	result := &TableResult{
		Columns: columns,
		Summary: fmt.Sprintf("(%d entities)", len(rows)),
	}
	for _, r := range rows {
		var rowData []any
		rowData = append(rowData, r.qualifiedName, r.entityType)
		if hasGeneralizations {
			rowData = append(rowData, r.generalization)
		}
		if r.entityType == "System" {
			rowData = append(rowData, "-", "-", "-", "-", "-", "-")
		} else {
			rowData = append(rowData, r.attrs, r.assocs, r.validations, r.indexes, r.events, r.accessRules)
		}
		result.Rows = append(result.Rows, rowData)
	}
	return writeResult(ctx, result)
}

// findEntityGen locates an Entity by qualified name across all
// DomainModels. Returns (nil, "", nil) when not found; module name is
// returned alongside the entity for downstream rendering.
func findEntityGen(ctx *ExecContext, qn ast.QualifiedName) (*genDm.Entity, string, error) {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return nil, "", err
	}
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return nil, "", err
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if modName != qn.Module {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			if entity.Name() == qn.Name {
				return entity, modName, nil
			}
		}
	}
	return nil, "", nil
}

// _ keeps the element import live during incremental landing — A2/A3
// will use element.Element pervasively for polymorphic dispatch.
var _ element.Element = (*genDm.Entity)(nil)

// formatAttributeTypeGen mirrors helpers.go::getAttributeTypeName on the
// gen-typed read path (Stage 3.3.4 A2). Polymorphic dispatch over the
// 14 concrete attribute-type subtypes; falls back to TypeName() for any
// unrecognised variant so future schema additions degrade gracefully.
//
// Per CLAUDE.md "Map iteration is deterministic": no map iteration here,
// but downstream callers must sort their attribute slices before
// rendering.
//
// Schema gap (per memory project_gen_schema_gaps + plan R7):
// gen has no separate DateAttributeType — Date and DateTime share
// DateTimeAttributeType. We render both as "DateTime" until the gen
// schema gains a Date discriminator. Mendix Studio Pro distinguishes
// purely date columns at the UI layer, not in the BSON shape exposed
// here.
func formatAttributeTypeGen(at element.Element) string {
	if at == nil {
		return "Unknown"
	}
	switch t := at.(type) {
	case *genDm.StringAttributeType:
		if l := t.Length(); l > 0 {
			return fmt.Sprintf("String(%d)", l)
		}
		return "String(unlimited)"
	case *genDm.IntegerAttributeType:
		return "Integer"
	case *genDm.LongAttributeType:
		return "Long"
	case *genDm.DecimalAttributeType:
		return "Decimal"
	case *genDm.FloatAttributeType:
		return "Float"
	case *genDm.CurrencyAttributeType:
		return "Currency"
	case *genDm.BooleanAttributeType:
		return "Boolean"
	case *genDm.DateTimeAttributeType:
		return "DateTime"
	case *genDm.AutoNumberAttributeType:
		return "AutoNumber"
	case *genDm.BinaryAttributeType:
		return "Binary"
	case *genDm.HashedStringAttributeType:
		return "HashedString"
	case *genDm.MultiLanguageAttributeType:
		return "MultiLanguage"
	case *genDm.EnumerationAttributeType:
		if qn := t.EnumerationQualifiedName(); qn != "" {
			return fmt.Sprintf("Enumeration(%s)", qn)
		}
		return "Enumeration"
	}
	// Fallback: use the BSON $Type tag minus the "DomainModels$" prefix.
	tn := at.TypeName()
	if tn == "" {
		return "Unknown"
	}
	if strings.HasPrefix(tn, "DomainModels$") {
		return strings.TrimPrefix(tn, "DomainModels$")
	}
	return tn
}

// describeEntityGen renders an entity in MDL "create or modify entity ..."
// form on the gen-typed read path (Stage 3.3.4 A3 — replaces describeEntity
// in cmd_entities_describe.go). Output is not byte-identical to the legacy
// path because entity.Location was a struct in sdk and a string in gen;
// the formatter still emits a Position annotation for backward-compatible
// re-execution.
//
// Access-rule output is delegated to outputEntityAccessGrants (legacy
// sdk-typed) and will become outputEntityAccessGrantsGen in Phase C3.
func describeEntityGen(ctx *ExecContext, name ast.QualifiedName) error {
	entity, modName, err := findEntityGen(ctx, name)
	if err != nil {
		return mdlerrors.NewBackend("get entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	// gen → spec → MDL pipeline: entitySpecFromGen extracts doc + position +
	// kind + extends + attributes (incl. validation/default/calculated) +
	// indexes; renderEntityMDL injects the `create or modify` prefix at the
	// statement line so DESCRIBE output is idempotent on re-execution.
	spec := entitySpecFromGen(modName, entity)
	fmt.Fprint(ctx.Output, renderEntityMDL(spec, true))
	fmt.Fprintln(ctx.Output, ";")

	outputEntityAccessGrantsGen(ctx, entity, name.Module, name.Name)

	fmt.Fprintln(ctx.Output, "/")
	return nil
}
