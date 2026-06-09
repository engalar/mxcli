// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — gen-typed visualization track for SHOW STRUCTURE.
//
// This file is the parallel of legacy `cmd_structure.go`'s
// microflow/nanoflow rendering paths. The legacy entry point
// `execShowStructure` is left untouched; this gen entry
// `execShowStructureGen` reuses every non-microflow helper
// (`structureEntities`, `structurePages`, `structureSnippets`,
// `structureWorkflows`, `outputJavaActions`,
// `structureODataClients/Services/BusinessEventServices`,
// `formatConstantTypeBrief`, `pluralize`, `shortName`, etc.) and only
// overrides the parts that consume `sdk/microflows.Microflow` /
// `sdk/microflows.Nanoflow`.
//
// Migrated surface from the legacy file:
//   - `mfByModule` / `nfByModule` building loops -> backed by
//     `ctx.Microflows.ListAll()` / `ctx.Nanoflows.List("")` and gen
//     types.
//   - `formatMicroflowSignature(mf.Parameters, mf.ReturnType, …)` ->
//     `formatMicroflowSignatureGen(mf)` / `formatNanoflowSignatureGen(nf)`
//     reading the gen ObjectCollection for parameters and the gen
//     `MicroflowReturnType()` element for the return type.
//   - `formatDataTypeDisplay` for sdk DataType -> already shipped as
//     `formatDataTypeDisplayGen` in `cmd_microflows_viz_helpers_gen.go`.
//   - `sortMicroflows` / `sortNanoflows` -> `sortGenMicroflows`,
//     `sortGenNanoflows` (slices of `*genMf.Microflow` / `*genMf.Nanoflow`).

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
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// ────────────────────────────────────────────────────────
// Entry point
// ────────────────────────────────────────────────────────

// execShowStructureGen is the gen-typed parallel of `execShowStructure`.
// Fully delegates depth-1 (catalog/SQL only) to the legacy helpers
// because nothing on that path touches sdk/microflows; depth-2 and -3
// are reimplemented to read microflow/nanoflow data via the gen
// repositories.
func execShowStructureGen(ctx *ExecContext, s *ast.ShowStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	depth := min(max(s.Depth, 1), 3)

	if err := ensureCatalog(ctx, false); err != nil {
		return mdlerrors.NewBackend("build catalog", err)
	}

	modules, err := getStructureModules(ctx, s.InModule, s.All)
	if err != nil {
		return err
	}

	if len(modules) == 0 {
		if ctx.Format == FormatJSON {
			fmt.Fprintln(ctx.Output, "[]")
		} else {
			fmt.Fprintln(ctx.Output, "(no modules found)")
		}
		return nil
	}

	if ctx.Format == FormatJSON {
		// JSON mode counts via SQL/catalog — no sdk types involved,
		// reuse legacy unchanged.
		return structureDepth1JSON(ctx, modules)
	}

	switch depth {
	case 1:
		// Depth 1 is purely catalog-driven; legacy helper has no sdk
		// types in its critical path.
		return structureDepth1(ctx, modules)
	case 2:
		return structureDepth2Gen(ctx, modules)
	case 3:
		return structureDepth3Gen(ctx, modules)
	default:
		return structureDepth2Gen(ctx, modules)
	}
}

// ────────────────────────────────────────────────────────
// Depth 2 — gen-typed
// ────────────────────────────────────────────────────────

func structureDepth2Gen(ctx *ExecContext, modules []structureModule) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	dmByModule, enumsByModule, constByModule, eventsByModule, jaByModule, wfByModule, err := loadStructureSharedDataGen(ctx, h)
	if err != nil {
		return err
	}

	mfByModule, err := loadGenMicroflowsByModule(ctx, h)
	if err != nil {
		return err
	}
	nfByModule, err := loadGenNanoflowsByModule(ctx, h)
	if err != nil {
		return err
	}

	for i, m := range modules {
		if i > 0 {
			fmt.Fprintln(ctx.Output)
		}
		fmt.Fprintln(ctx.Output, m.Name)

		structureEntitiesGen(ctx, m.Name, dmByModule[m.Name], false)

		// Enumerations — unchanged.
		if enums, ok := enumsByModule[m.Name]; ok {
			sortEnumerations(enums)
			for _, enum := range enums {
				values := make([]string, len(enum.Values))
				for i, v := range enum.Values {
					values[i] = v.Name
				}
				fmt.Fprintf(ctx.Output, "  Enumeration %s.%s [%s]\n", m.Name, enum.Name, strings.Join(values, ", "))
			}
		}

		// Microflows — gen-typed.
		if mfs, ok := mfByModule[m.Name]; ok {
			sortGenMicroflows(mfs)
			for _, mf := range mfs {
				fmt.Fprintf(ctx.Output, "  Microflow %s.%s%s\n",
					m.Name, mf.Name(), formatMicroflowSignatureGen(mf, false))
			}
		}

		// Nanoflows — gen-typed.
		if nfs, ok := nfByModule[m.Name]; ok {
			sortGenNanoflows(nfs)
			for _, nf := range nfs {
				fmt.Fprintf(ctx.Output, "  Nanoflow %s.%s%s\n",
					m.Name, nf.Name(), formatNanoflowSignatureGen(nf, false))
			}
		}

		// Workflows — unchanged sdk/workflows helper (out of scope for
		// 3.2.4; tracked under task #3 Stage 3.2.5b).
		structureWorkflows(ctx, m.Name, wfByModule[m.Name], false)

		structurePages(ctx, m.Name)
		structureSnippets(ctx, m.Name)

		outputJavaActionsGen(ctx, m.Name, jaByModule[m.Name], false)

		if consts, ok := constByModule[m.Name]; ok {
			sortConstants(consts)
			for _, c := range consts {
				fmt.Fprintf(ctx.Output, "  Constant %s.%s: %s\n", m.Name, c.Name, formatConstantTypeBrief(c.Type))
			}
		}
		if events, ok := eventsByModule[m.Name]; ok {
			sortScheduledEvents(events)
			for _, ev := range events {
				fmt.Fprintf(ctx.Output, "  ScheduledEvent %s.%s\n", m.Name, ev.Name)
			}
		}

		structureODataClients(ctx, m.Name)
		structureODataServices(ctx, m.Name)
		structureBusinessEventServices(ctx, m.Name)
	}

	return nil
}

// ────────────────────────────────────────────────────────
// Depth 3 — gen-typed (adds withTypes / withDetails to deep helpers)
// ────────────────────────────────────────────────────────

func structureDepth3Gen(ctx *ExecContext, modules []structureModule) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	dmByModule, enumsByModule, constByModule, eventsByModule, jaByModule, wfByModule, err := loadStructureSharedDataGen(ctx, h)
	if err != nil {
		return err
	}

	mfByModule, err := loadGenMicroflowsByModule(ctx, h)
	if err != nil {
		return err
	}
	nfByModule, err := loadGenNanoflowsByModule(ctx, h)
	if err != nil {
		return err
	}

	for i, m := range modules {
		if i > 0 {
			fmt.Fprintln(ctx.Output)
		}
		fmt.Fprintln(ctx.Output, m.Name)

		structureEntitiesGen(ctx, m.Name, dmByModule[m.Name], true)

		if enums, ok := enumsByModule[m.Name]; ok {
			sortEnumerations(enums)
			for _, enum := range enums {
				values := make([]string, len(enum.Values))
				for i, v := range enum.Values {
					values[i] = v.Name
				}
				fmt.Fprintf(ctx.Output, "  Enumeration %s.%s [%s]\n", m.Name, enum.Name, strings.Join(values, ", "))
			}
		}

		// Microflows with parameter names.
		if mfs, ok := mfByModule[m.Name]; ok {
			sortGenMicroflows(mfs)
			for _, mf := range mfs {
				fmt.Fprintf(ctx.Output, "  Microflow %s.%s%s\n",
					m.Name, mf.Name(), formatMicroflowSignatureGen(mf, true))
			}
		}

		// Nanoflows with parameter names.
		if nfs, ok := nfByModule[m.Name]; ok {
			sortGenNanoflows(nfs)
			for _, nf := range nfs {
				fmt.Fprintf(ctx.Output, "  Nanoflow %s.%s%s\n",
					m.Name, nf.Name(), formatNanoflowSignatureGen(nf, true))
			}
		}

		structureWorkflows(ctx, m.Name, wfByModule[m.Name], true)
		structurePages(ctx, m.Name)
		structureSnippets(ctx, m.Name)
		outputJavaActionsGen(ctx, m.Name, jaByModule[m.Name], true)

		if consts, ok := constByModule[m.Name]; ok {
			sortConstants(consts)
			for _, c := range consts {
				s := fmt.Sprintf("  Constant %s.%s: %s", m.Name, c.Name, formatConstantTypeBrief(c.Type))
				if c.DefaultValue != "" {
					s += " = " + c.DefaultValue
				}
				fmt.Fprintln(ctx.Output, s)
			}
		}
		if events, ok := eventsByModule[m.Name]; ok {
			sortScheduledEvents(events)
			for _, ev := range events {
				fmt.Fprintf(ctx.Output, "  ScheduledEvent %s.%s\n", m.Name, ev.Name)
			}
		}

		structureODataClients(ctx, m.Name)
		structureODataServices(ctx, m.Name)
		structureBusinessEventServices(ctx, m.Name)
	}

	return nil
}

// ────────────────────────────────────────────────────────
// Shared data loading (non-microflow) — typed loose helpers
// ────────────────────────────────────────────────────────

// loadStructureSharedDataGen loads everything except microflows and
// nanoflows. Mirrors the loading block at the top of legacy
// structureDepth2/Depth3. The returned types match what the unchanged
// helpers (structureEntities / structureWorkflows / outputJavaActions)
// already consume, so callers don't have to type-assert.
func loadStructureSharedDataGen(ctx *ExecContext, h *ContainerHierarchy) (
	dmByModule structureDmMapGen,
	enumsByModule structureEnumMapGen,
	constByModule structureConstMapGen,
	eventsByModule structureEventMapGen,
	jaByModule structureJaMapGen,
	wfByModule structureWfMapGen,
	err error,
) {
	domainModels, _ := listDomainModelsWithContainerGen(ctx)
	dmByModule = make(structureDmMapGen)
	for _, pair := range domainModels {
		if pair.DM == nil {
			continue
		}
		modID := h.FindModuleID(pair.ContainerID)
		modName := h.GetModuleName(modID)
		dmByModule[modName] = pair.DM
	}

	allEnums, _ := ctx.Backend.ListEnumerations()
	enumsByModule = make(structureEnumMapGen)
	for _, enum := range allEnums {
		modID := h.FindModuleID(enum.ContainerID)
		modName := h.GetModuleName(modID)
		enumsByModule[modName] = append(enumsByModule[modName], enum)
	}

	allConstants, _ := ctx.Backend.ListConstants()
	constByModule = make(structureConstMapGen)
	for _, c := range allConstants {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		constByModule[modName] = append(constByModule[modName], c)
	}

	allEvents, _ := ctx.Backend.ListScheduledEvents()
	eventsByModule = make(structureEventMapGen)
	for _, ev := range allEvents {
		modID := h.FindModuleID(ev.ContainerID)
		modName := h.GetModuleName(modID)
		eventsByModule[modName] = append(eventsByModule[modName], ev)
	}

	// gen JavaAction loses container linkage during BSON roundtrip; use
	// the cache helper which joins through the MPR Unit table to recover it.
	jaPairs, _ := listJavaActionsWithContainerGen(ctx)
	jaByModule = make(structureJaMapGen)
	for _, p := range jaPairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		jaByModule[modName] = append(jaByModule[modName], p.Elem)
	}

	// gen Workflow loses container linkage during BSON roundtrip; use
	// the cache helper which joins through the MPR Unit table to recover it.
	wfPairs, _ := listWorkflowsWithContainerGen(ctx)
	wfByModule = make(structureWfMapGen)
	for _, p := range wfPairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		wfByModule[modName] = append(wfByModule[modName], p.Elem)
	}
	return
}

// loadGenMicroflowsByModule fetches every microflow via the gen repo
// and groups them by their owning module's name. Module resolution
// uses the SQL-backed container chain (gen objects don't carry
// container linkage).
func loadGenMicroflowsByModule(ctx *ExecContext, h *ContainerHierarchy) (map[string][]*genMf.Microflow, error) {
	if ctx.Microflows == nil {
		return nil, mdlerrors.NewBackend("microflow repository", fmt.Errorf("ctx.Microflows is nil"))
	}
	all, err := ctx.Microflows.ListAll()
	if err != nil {
		return nil, mdlerrors.NewBackend("list microflows", err)
	}
	out := make(map[string][]*genMf.Microflow)
	for _, mf := range all {
		modName := lookupGenContainerModule(ctx, h, mf.ID())
		out[modName] = append(out[modName], mf)
	}
	return out, nil
}

// loadGenNanoflowsByModule fetches every nanoflow via the gen repo
// and groups them by their owning module's name.
func loadGenNanoflowsByModule(ctx *ExecContext, h *ContainerHierarchy) (map[string][]*genMf.Nanoflow, error) {
	if ctx.Nanoflows == nil {
		return nil, mdlerrors.NewBackend("nanoflow repository", fmt.Errorf("ctx.Nanoflows is nil"))
	}
	all, err := ctx.Nanoflows.List("")
	if err != nil {
		return nil, mdlerrors.NewBackend("list nanoflows", err)
	}
	out := make(map[string][]*genMf.Nanoflow)
	for _, nf := range all {
		modName := lookupGenContainerModule(ctx, h, nf.ID())
		out[modName] = append(out[modName], nf)
	}
	return out, nil
}

// ────────────────────────────────────────────────────────
// Microflow / Nanoflow signature formatting — gen-typed
// ────────────────────────────────────────────────────────

// formatMicroflowSignatureGen produces the same `(Type, Type) → Ret`
// shape legacy `formatMicroflowSignature` produces, but reads from
// the gen Microflow's ObjectCollection (parameters live alongside
// activities) and from `MicroflowReturnType()`.
func formatMicroflowSignatureGen(mf *genMf.Microflow, withNames bool) string {
	if mf == nil {
		return "()"
	}
	params := genFlowParameterElems(mf.ObjectCollection())
	ret := mf.MicroflowReturnType()
	return formatGenFlowSignature(params, ret, withNames)
}

// formatNanoflowSignatureGen — same as above for Nanoflow.
func formatNanoflowSignatureGen(nf *genMf.Nanoflow, withNames bool) string {
	if nf == nil {
		return "()"
	}
	params := genFlowParameterElems(nf.ObjectCollection())
	ret := nf.MicroflowReturnType()
	return formatGenFlowSignature(params, ret, withNames)
}

// formatGenFlowSignature is the shared signature renderer for
// microflows + nanoflows. Mirrors legacy `formatMicroflowSignature`.
func formatGenFlowSignature(params []*genMf.MicroflowParameter, returnType element.Element, withNames bool) string {
	var paramParts []string
	for _, p := range params {
		typeName := formatGenParameterTypeDisplay(p)
		if withNames && p.Name() != "" {
			paramParts = append(paramParts, fmt.Sprintf("%s: %s", p.Name(), typeName))
		} else {
			paramParts = append(paramParts, typeName)
		}
	}

	sig := "(" + strings.Join(paramParts, ", ") + ")"

	if returnType != nil {
		retName := formatDataTypeDisplayGen(returnType)
		if retName != "" && retName != "Void" && retName != "Nothing" {
			sig += " → " + retName
		}
	}
	return sig
}

// formatGenParameterTypeDisplay extracts a parameter's type display
// name. Prefers the rich `ParameterType` element (DataType subtype),
// falling back to the `Type()` short string when it is set.
func formatGenParameterTypeDisplay(p *genMf.MicroflowParameter) string {
	if p == nil {
		return ""
	}
	if pt := p.ParameterType(); pt != nil {
		if name := formatDataTypeDisplayGen(pt); name != "" {
			return name
		}
	}
	if t := strings.TrimSpace(p.Type()); t != "" {
		return shortName(t)
	}
	return ""
}

// genFlowParameterElems extracts the strongly-typed gen
// MicroflowParameter elements from an ObjectCollection. Stays
// stable in declaration order (the order the BSON Objects array
// stored them).
func genFlowParameterElems(oc element.Element) []*genMf.MicroflowParameter {
	col, ok := oc.(*genMf.MicroflowObjectCollection)
	if !ok || col == nil {
		return nil
	}
	var out []*genMf.MicroflowParameter
	for _, obj := range col.ObjectsItems() {
		if p, ok := obj.(*genMf.MicroflowParameter); ok && p != nil {
			out = append(out, p)
		}
	}
	return out
}

// ────────────────────────────────────────────────────────
// Sort helpers — gen-typed parallels of sortMicroflows/Nanoflows
// ────────────────────────────────────────────────────────

func sortGenMicroflows(mfs []*genMf.Microflow) {
	sort.Slice(mfs, func(i, j int) bool {
		return strings.ToLower(mfs[i].Name()) < strings.ToLower(mfs[j].Name())
	})
}

func sortGenNanoflows(nfs []*genMf.Nanoflow) {
	sort.Slice(nfs, func(i, j int) bool {
		return strings.ToLower(nfs[i].Name()) < strings.ToLower(nfs[j].Name())
	})
}

// ────────────────────────────────────────────────────────
// Loose map type aliases (improve readability of the loader signature)
// ────────────────────────────────────────────────────────
//
// These point at the same concrete sdk-typed slices the unchanged
// legacy helpers (structureEntities / structureWorkflows /
// outputJavaActions) expect. Stage 3.2.4 migrates only the
// sdk/microflows surface; sdk/domainmodel, sdk/workflows,
// sdk/javaactions remain in scope for later stages (#3 / #4).
type (
	structureDmMapGen    = map[string]*genDm.DomainModel
	structureEnumMapGen  = map[string][]*model.Enumeration
	structureConstMapGen = map[string][]*model.Constant
	structureEventMapGen = map[string][]*model.ScheduledEvent
	structureJaMapGen    = map[string][]*genJA.JavaAction
	structureWfMapGen    = map[string][]*genWf.Workflow
)

func structureEntitiesGen(ctx *ExecContext, moduleName string, dm *genDm.DomainModel, withTypes bool) {
	if dm == nil {
		return
	}

	entityByID := make(map[model.ID]string)
	var entities []*genDm.Entity
	for _, item := range dm.EntitiesItems() {
		ent, ok := item.(*genDm.Entity)
		if !ok || ent == nil {
			continue
		}
		entityByID[model.ID(ent.ID())] = ent.Name()
		entities = append(entities, ent)
	}

	sort.Slice(entities, func(i, j int) bool {
		return strings.ToLower(entities[i].Name()) < strings.ToLower(entities[j].Name())
	})

	assocByParent := make(map[model.ID][]*genDm.Association)
	for _, item := range dm.AssociationsItems() {
		assoc, ok := item.(*genDm.Association)
		if !ok || assoc == nil {
			continue
		}
		assocByParent[model.ID(assoc.ParentRefID())] = append(assocByParent[model.ID(assoc.ParentRefID())], assoc)
	}

	for _, ent := range entities {
		var attrParts []string
		for _, item := range ent.AttributesItems() {
			attr, ok := item.(*genDm.Attribute)
			if !ok || attr == nil {
				continue
			}
			if withTypes {
				attrParts = append(attrParts, fmt.Sprintf("%s: %s", attr.Name(), formatAttributeTypeGen(attr.Type())))
			} else {
				attrParts = append(attrParts, attr.Name())
			}
		}
		qualName := moduleName + "." + ent.Name()
		if len(attrParts) > 0 {
			fmt.Fprintf(ctx.Output, "  Entity %s [%s]\n", qualName, strings.Join(attrParts, ", "))
		} else {
			fmt.Fprintf(ctx.Output, "  Entity %s\n", qualName)
		}

		if assocs, ok := assocByParent[model.ID(ent.ID())]; ok {
			var assocParts []string
			for _, assoc := range assocs {
				childName := entityByID[model.ID(assoc.ChildRefID())]
				if childName == "" {
					childName = "?"
				}
				cardinality := "(1)"
				if assoc.Type() == "ReferenceSet" {
					cardinality = "(*)"
				}
				part := fmt.Sprintf("→ %s %s", childName, cardinality)
				if withTypes {
					if dbe, ok := assoc.DeleteBehavior().(*genDm.AssociationDeleteBehavior); ok && dbe != nil {
						switch dbe.ChildDeleteBehavior() {
						case "DeleteMeAndReferences":
							part += " cascade"
						case "DeleteMeIfNoReferences":
							part += " RESTRICT"
						}
					}
				}
				assocParts = append(assocParts, part)
			}
			if len(assocParts) > 0 {
				fmt.Fprintf(ctx.Output, "    %s\n", strings.Join(assocParts, ", "))
			}
		}
	}
}
