// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — Mermaid visualization (gen-typed track).
//
// Parallel of legacy `cmd_mermaid.go`'s microflow / nanoflow rendering
// paths. The legacy `describeMermaid` dispatcher is left untouched
// (still wired to Executor.DescribeMermaid → cmd/mxcli/cmd_describe.go);
// this file only exposes gen-typed `microflowToMermaidGen`,
// `nanoflowToMermaidGen`, and `renderFlowMermaidGen`. The other two
// branches of legacy `describeMermaid` (entity / page) keep using
// sdk/domainmodel and sdk/pages — those are out of scope for Stage
// 3.2.4 and tracked under task #3 (Stage 3.2.5b workflow + odata).
//
// All label / details / case-label / classification helpers come
// from `cmd_microflows_viz_helpers_gen.go` (introduced in Stage 3.2.4
// commit 1 alongside `cmd_nanoflow_elk_gen.go`).

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

// ────────────────────────────────────────────────────────
// Entry points
// ────────────────────────────────────────────────────────

// microflowToMermaidGen renders a gen Microflow as a Mermaid flowchart
// to ctx.Output. Mirrors legacy `microflowToMermaid`.
func microflowToMermaidGen(ctx *ExecContext, name ast.QualifiedName) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if ctx.Microflows == nil {
		return mdlerrors.NewBackend("microflow repository", fmt.Errorf("ctx.Microflows is nil"))
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	entityNames, err := buildEntityNames(ctx, h)
	if err != nil {
		return err
	}

	mf, err := ctx.Microflows.FindByQualifiedName(name.String())
	if err != nil {
		return mdlerrors.NewBackend("find microflow", err)
	}
	if mf == nil {
		return mdlerrors.NewNotFound("microflow", name.String())
	}

	return renderFlowMermaidGen(ctx, mf.ObjectCollection(), mf.FlowsItems(), entityNames)
}

// nanoflowToMermaidGen renders a gen Nanoflow as a Mermaid flowchart.
// Mirrors legacy `nanoflowToMermaid`.
func nanoflowToMermaidGen(ctx *ExecContext, name ast.QualifiedName) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if ctx.Nanoflows == nil {
		return mdlerrors.NewBackend("nanoflow repository", fmt.Errorf("ctx.Nanoflows is nil"))
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	entityNames, err := buildEntityNames(ctx, h)
	if err != nil {
		return err
	}

	all, err := ctx.Nanoflows.List("")
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	for _, nf := range all {
		modName := lookupGenContainerModule(ctx, h, nf.ID())
		if modName == name.Module && nf.Name() == name.Name {
			return renderFlowMermaidGen(ctx, nf.ObjectCollection(), nf.FlowsItems(), entityNames)
		}
	}
	return mdlerrors.NewNotFound("nanoflow", name.String())
}

// ────────────────────────────────────────────────────────
// Renderer
// ────────────────────────────────────────────────────────

// renderFlowMermaidGen renders a flow's gen ObjectCollection as a
// Mermaid flowchart. Mirrors legacy `renderFlowMermaid` 1:1 in shape
// (header, node defs, edges, style block, metadata footer).
func renderFlowMermaidGen(ctx *ExecContext, oc element.Element, topLevelFlows []element.Element, entityNames map[model.ID]string) error {
	var sb strings.Builder
	sb.WriteString("flowchart LR\n")

	col, ok := oc.(*genMf.MicroflowObjectCollection)
	if !ok || col == nil || len(genActivityObjects(col)) == 0 {
		sb.WriteString("    start([Start]) --> stop([End])\n")
		fmt.Fprint(ctx.Output, sb.String())
		return nil
	}

	// Collect all objects + flows recursively (loop bodies inline).
	allObjects, allFlows := collectAllObjectsAndFlowsGen(col, topLevelFlows)

	// Build activity map and find start event.
	activityMap := make(map[element.ID]element.Element, len(allObjects))
	var startID element.ID
	for _, obj := range allObjects {
		activityMap[obj.ID()] = obj
		if _, ok := obj.(*genMf.StartEvent); ok && startID == "" {
			startID = obj.ID()
		}
	}

	// Index flows by origin and sort by OriginConnectionIndex.
	flowsByOrigin := make(map[element.ID][]*genMf.SequenceFlow)
	for _, f := range allFlows {
		flowsByOrigin[f.OriginRefID()] = append(flowsByOrigin[f.OriginRefID()], f)
	}
	for originID := range flowsByOrigin {
		flows := flowsByOrigin[originID]
		// Bubble sort to mirror legacy exactly (small N, stable).
		for i := 0; i < len(flows)-1; i++ {
			for j := i + 1; j < len(flows); j++ {
				if flows[i].OriginConnectionIndex() > flows[j].OriginConnectionIndex() {
					flows[i], flows[j] = flows[j], flows[i]
				}
			}
		}
	}

	// Per-node detail metadata for the webview's expand-on-click view.
	nodeInfo := make(map[string][]string)

	// Emit node definitions in the same order legacy did (objects in
	// collection order; loop body objects appended after each loop).
	for _, obj := range allObjects {
		id := mermaidShortID(model.ID(obj.ID()))
		label := mermaidActivityLabelGen(obj, entityNames)

		switch obj.(type) {
		case *genMf.StartEvent:
			sb.WriteString(fmt.Sprintf("    %s([Start])\n", id))
		case *genMf.EndEvent:
			sb.WriteString(fmt.Sprintf("    %s([%s])\n", id, label))
		case *genMf.ExclusiveSplit:
			sb.WriteString(fmt.Sprintf("    %s{%s}\n", id, label))
		case *genMf.InheritanceSplit:
			sb.WriteString(fmt.Sprintf("    %s{%s}\n", id, label))
		case *genMf.ExclusiveMerge:
			sb.WriteString(fmt.Sprintf("    %s(( ))\n", id))
		case *genMf.LoopedActivity:
			sb.WriteString(fmt.Sprintf("    %s[/%s/]\n", id, label))
		default:
			sb.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", id, label))
		}

		if details := mermaidActivityDetailsGen(obj, entityNames); len(details) > 0 {
			nodeInfo[id] = details
		}
	}

	// Emit edges (deduped on (from, to)).
	visited := make(map[string]bool)
	for originID, flows := range flowsByOrigin {
		fromID := mermaidShortID(model.ID(originID))
		for _, flow := range flows {
			toID := mermaidShortID(model.ID(flow.DestinationRefID()))
			edgeKey := fromID + "->" + toID
			if visited[edgeKey] {
				continue
			}
			visited[edgeKey] = true

			label := mermaidCaseLabelGen(flow.CaseValue())
			if label != "" {
				sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", fromID, label, toID))
			} else {
				sb.WriteString(fmt.Sprintf("    %s --> %s\n", fromID, toID))
			}
		}
	}

	// Style the start node.
	if startID != "" {
		sb.WriteString(fmt.Sprintf("    style %s fill:#4CAF50,color:#fff\n", mermaidShortID(model.ID(startID))))
	}

	// Metadata footer for the webview.
	sb.WriteString("\n%% @type flowchart\n")
	sb.WriteString("%% @direction LR\n")

	if len(nodeInfo) > 0 {
		sb.WriteString("%% @nodeinfo {")
		first := true
		for id, details := range nodeInfo {
			if !first {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf(`"%s":[`, id))
			for i, d := range details {
				if i > 0 {
					sb.WriteString(",")
				}
				escaped := strings.ReplaceAll(d, `\`, `\\`)
				escaped = strings.ReplaceAll(escaped, `"`, `\"`)
				escaped = strings.ReplaceAll(escaped, "\n", `\n`)
				escaped = strings.ReplaceAll(escaped, "\r", `\r`)
				escaped = strings.ReplaceAll(escaped, "\t", `\t`)
				sb.WriteString(fmt.Sprintf(`"%s"`, escaped))
			}
			sb.WriteString("]")
			first = false
		}
		sb.WriteString("}\n")
	}

	fmt.Fprint(ctx.Output, sb.String())
	return nil
}

// ────────────────────────────────────────────────────────
// Recursive collection helper — gen analogue of
// collectAllObjectsAndFlows (loop body inline traversal).
// ────────────────────────────────────────────────────────

// collectAllObjectsAndFlowsGen walks a gen ObjectCollection plus a
// list of top-level flows and returns the flat list of every activity
// node and every sequence flow, recursing into LoopedActivity
// ObjectCollection bodies. Loop body flows are co-mingled with
// activities inside the loop's own ObjectCollection — matching how
// legacy stored them and how Stage 3.2.1's traverseFlowGen reads
// them.
func collectAllObjectsAndFlowsGen(col *genMf.MicroflowObjectCollection, topLevelFlows []element.Element) ([]element.Element, []*genMf.SequenceFlow) {
	if col == nil {
		return nil, nil
	}

	var objects []element.Element
	objects = append(objects, genActivityObjects(col)...)

	flows := genSequenceFlowsFromList(topLevelFlows)

	// Recurse into nested LoopedActivity bodies — flows live there
	// alongside activities (legacy parity).
	for _, obj := range genActivityObjects(col) {
		if loop, ok := obj.(*genMf.LoopedActivity); ok && loop.ObjectCollection() != nil {
			if inner, ok := loop.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && inner != nil {
				nestedObjs, _ := collectAllObjectsAndFlowsGen(inner, nil)
				objects = append(objects, nestedObjs...)
				flows = append(flows, genSequenceFlows(inner)...)
			}
		}
	}

	return objects, flows
}
