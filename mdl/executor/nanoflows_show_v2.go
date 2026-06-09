// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5c — gen-typed DESCRIBE NANOFLOW skeleton.
//
// Parallel of cmd_microflows_show_gen.go for nanoflows. Reuses every
// gen-typed building block introduced in Stage 3.2.1 / 3.2.2 (the
// `traverseFlowGen` graph walk and the family of `formatActionGen`
// formatters) — only the surface that differs between the two flow
// kinds is reimplemented here:
//
//   - Header keyword: `create or modify nanoflow ...`.
//   - Parameter element type: `Microflows$NanoflowParameter` instead of
//     the microflow's `Microflows$MicroflowParameter`. Falls back to
//     the microflow tag too because some fixtures alias them.
//   - Module-name resolution: identical SQL-backed container chain via
//     ctx.Microflows.GetContainerUUID — nanoflows live in the same
//     `unit` table alongside microflows.
//
// Legacy `describeNanoflow` / `describeNanoflowToString` in
// cmd_microflows_show.go are unchanged; this file is the parallel
// surface that Stage 3.2.6 will route the dispatcher onto.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// describeNanoflowGen prints the gen-rendered MDL for a nanoflow to
// ctx.Output. Mirrors legacy `describeNanoflow` (cmd_microflows_show.go
// line 329) but consumes only ctx.Nanoflows / ctx.Microflows.
func describeNanoflowGen(ctx *ExecContext, name ast.QualifiedName) error {
	out, _, err := describeNanoflowGenToString(ctx, name)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Output, out)
	return nil
}

// describeNanoflowGenToString renders the gen-typed MDL source for a
// nanoflow and returns it as a string along with a source map mapping
// node IDs to line ranges (currently unused — sourceMap stays empty
// because the gen traverser does not attribute lines to node IDs yet).
//
// Resolution path mirrors `describeMicroflowToString`:
//  1. List all nanoflows via ctx.Nanoflows.List("").
//  2. Walk the slice matching module + name. Module name is derived
//     from the SQL-backed container chain.
//  3. Render via `DescribeNanoflowGenToString` for the actual text.
func describeNanoflowGenToString(ctx *ExecContext, name ast.QualifiedName) (string, map[string]elkSourceRange, error) {
	if ctx == nil || ctx.Nanoflows == nil {
		return "", nil, mdlerrors.NewBackend("nanoflow repository", fmt.Errorf("ctx.Nanoflows is nil"))
	}

	all, err := ctx.Nanoflows.List("")
	if err != nil {
		return "", nil, mdlerrors.NewBackend("list nanoflows", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return "", nil, mdlerrors.NewBackend("build hierarchy", err)
	}

	var target *genMf.Nanoflow
	for _, nf := range all {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if modName == name.Module && nf.Name() == name.Name {
			target = nf
			break
		}
	}
	if target == nil {
		return "", nil, mdlerrors.NewNotFound("nanoflow", name.String())
	}

	rendered, err := DescribeNanoflowGenToString(ctx, target)
	if err != nil {
		return "", nil, err
	}
	return rendered, map[string]elkSourceRange{}, nil
}

// DescribeNanoflowGenToString renders a *genMf.Nanoflow as MDL source
// using only the modelsdk/gen API surface. Public for symmetry with
// DescribeMicroflowGenToString — both share the same downstream
// formatters / traverser.
func DescribeNanoflowGenToString(ctx *ExecContext, nf *genMf.Nanoflow) (string, error) {
	if nf == nil {
		return "", fmt.Errorf("DescribeNanoflowGenToString: nil nanoflow")
	}

	moduleName, qualifiedName := genNanoflowQualifiedName(ctx, nf)

	var lines []string

	// Documentation block.
	if doc := nf.Documentation(); doc != "" {
		lines = append(lines, "/**")
		for _, dl := range strings.Split(doc, "\n") {
			lines = append(lines, " * "+dl)
		}
		lines = append(lines, " */")
	}

	// @excluded annotation.
	if nf.Excluded() {
		lines = append(lines, "@excluded")
	}

	// Header + parameters.
	params := genNanoflowParameters(nf)
	if len(params) > 0 {
		lines = append(lines, fmt.Sprintf("create or modify nanoflow %s (", qualifiedName))
		for i, p := range params {
			comma := ","
			if i == len(params)-1 {
				comma = ""
			}
			lines = append(lines, fmt.Sprintf("  $%s: %s%s", p.name, p.declType, comma))
		}
		lines = append(lines, ")")
	} else {
		lines = append(lines, fmt.Sprintf("create or modify nanoflow %s ()", qualifiedName))
	}

	// Returns clause. Prefer the rich `MicroflowReturnType()` element
	// (DataType subtype) so entity-returning nanoflows render as
	// `returns Module.Entity` rather than the bare primitive name (the
	// string accessor `ReturnType()` is empty for entity returns —
	// the qualified name lives inside the part element).
	if rendered := genFlowReturnDisplay(nf.ReturnType(), nf.MicroflowReturnType()); rendered != "" {
		returnLine := "returns " + rendered
		if rv := nf.ReturnVariableName(); rv != "" && rv != "Variable" {
			returnLine += " as $" + rv
		}
		lines = append(lines, returnLine)
	}

	lines = append(lines, "{")

	bodyLines := renderGenNanoflowBody(ctx, nf)
	if len(bodyLines) == 0 {
		lines = append(lines, "  -- No activities")
	} else {
		for _, l := range bodyLines {
			lines = append(lines, "  "+l)
		}
	}
	lines = append(lines, "}")

	// Allowed module roles → grant execute footer.
	// Keep fully-qualified names and filter out the auto-created "User" placeholder.
	if roles := filterAutoDocumentRoles(nf.AllowedModuleRolesQualifiedNames()); len(roles) > 0 {
		simple := strings.SplitN(qualifiedName, ".", 2)
		if len(simple) == 2 {
			lines = append(lines,
				"",
				fmt.Sprintf("grant execute on nanoflow %s.%s to %s;", moduleName, simple[1], strings.Join(roles, ", ")),
			)
		}
	}
	lines = append(lines, "/")

	return strings.Join(lines, "\n"), nil
}

// genNanoflowQualifiedName resolves the nanoflow's owning module name.
// Uses the same SQL-backed strategy as genMicroflowQualifiedName —
// nanoflows live in the same `unit` table alongside microflows so
// ctx.Microflows.GetContainerUUID returns the correct container UUID
// for either flow kind.
func genNanoflowQualifiedName(ctx *ExecContext, nf *genMf.Nanoflow) (string, string) {
	name := nf.Name()
	module := ""

	// Path 1: SQL-backed container UUID + ContainerHierarchy.
	if ctx != nil && ctx.Microflows != nil && ctx.Connected() {
		if containerID, err := ctx.Microflows.GetContainerUUID(model.ID(nf.ID())); err == nil && containerID != "" {
			if h, err := getHierarchy(ctx); err == nil && h != nil {
				modID := h.FindModuleID(containerID)
				if mn := h.GetModuleName(modID); mn != "" {
					module = mn
				}
			}
		}
	}

	// Path 2: in-memory Container() walk (no roundtrip).
	if module == "" {
		for c := nf.Container(); c != nil; c = c.Container() {
			if t := c.TypeName(); t == "Projects$Module" {
				if nv, ok := c.(interface{ NameValue() string }); ok {
					if mn := nv.NameValue(); mn != "" {
						module = mn
						break
					}
				}
			}
		}
	}

	if module == "" {
		module = "<unknown>"
	}
	return module, module + "." + name
}

// genNanoflowParameters extracts parameter summaries from the
// nanoflow's ObjectCollection. Nanoflows store parameters as
// `Microflows$NanoflowParameter` elements (different tag from
// microflows' `Microflows$MicroflowParameter`); we accept both tags
// so fixtures that alias them still resolve.
func genNanoflowParameters(nf *genMf.Nanoflow) []genParamSummary {
	oc, ok := nf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil
	}
	var out []genParamSummary
	for _, obj := range oc.ObjectsItems() {
		if obj == nil {
			continue
		}
		switch obj.TypeName() {
		case "Microflows$NanoflowParameter", "Microflows$MicroflowParameter":
		default:
			continue
		}
		s := genParamSummary{declType: "Object"}
		if nv, ok := obj.(interface{ NameValue() string }); ok {
			if n := nv.NameValue(); n != "" {
				s.name = n
			}
		}
		if s.name == "" {
			if nv, ok := obj.(interface{ Name() string }); ok {
				s.name = nv.Name()
			}
		}
		// Read VariableType child element for the concrete type, same as
		// genMicroflowParameters. The deprecated Type() string is never set.
		if pt, ok := obj.(interface{ ParameterType() element.Element }); ok {
			if vt := pt.ParameterType(); vt != nil {
				s.declType = genVariableTypeToDeclType(vt)
			}
		}
		out = append(out, s)
	}
	return out
}

// renderGenNanoflowBody renders the nanoflow body using the same gen
// traverser as microflows — the structural framing (if/case/loop) is
// identical between the two flow kinds. Only the parameter element
// type and header keyword differ; both are handled in the caller.
//
// Flows are stored on the Nanoflow itself (FlowsItems) — same as
// Microflow — and activities live in the ObjectCollection.
func renderGenNanoflowBody(ctx *ExecContext, nf *genMf.Nanoflow) []string {
	oc, ok := nf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil
	}

	objects := oc.ObjectsItems()
	flowsByOrigin, flowsByDest := buildGenFlowMapsFromList(nf.FlowsItems())
	activityMap, startID := buildGenActivityMap(objects)
	splitMergeMap := findSplitMergePointsGen(activityMap, flowsByOrigin)

	visited := make(map[element.ID]bool)
	var lines []string
	traverseFlowGen(ctx, startID, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, flowsByOrigin, visited, &lines, 0)
	return lines
}

// genFlowReturnDisplay picks the best return-type display for a flow.
// Mirrors the resolution order legacy formatMicroflowDataType uses:
//
//  1. The rich `MicroflowReturnType` part element when present — this
//     is a DataType subtype that knows the entity qualified name and
//     "List of X" wrapping.
//  2. Fall back to the short string accessor when the part is missing
//     (some BSON shapes only carry the primitive tag).
//
// Returns "" for void / empty so the caller can omit the clause.
func genFlowReturnDisplay(short string, rt element.Element) string {
	if rt != nil {
		if name := strings.TrimSpace(formatDataTypeDisplayGen(rt)); name != "" && !strings.EqualFold(name, "Void") && !strings.EqualFold(name, "Nothing") {
			return name
		}
	}
	if s := strings.TrimSpace(short); s != "" && !strings.EqualFold(s, "Void") {
		return s
	}
	return ""
}

// buildGenFlowMapsFromList builds origin/dest indexes from a list of
// element.Element entries (Nanoflow.FlowsItems returns []element.Element
// rather than the strongly-typed slice that buildGenFlowMaps expects).
// Non-SequenceFlow elements are skipped.
func buildGenFlowMapsFromList(items []element.Element) (map[element.ID][]*genMf.SequenceFlow, map[element.ID][]*genMf.SequenceFlow) {
	byOrigin := make(map[element.ID][]*genMf.SequenceFlow)
	byDest := make(map[element.ID][]*genMf.SequenceFlow)
	for _, item := range items {
		sf, ok := item.(*genMf.SequenceFlow)
		if !ok || sf == nil {
			continue
		}
		byOrigin[sf.OriginRefID()] = append(byOrigin[sf.OriginRefID()], sf)
		byDest[sf.DestinationRefID()] = append(byDest[sf.DestinationRefID()], sf)
	}
	return byOrigin, byDest
}
