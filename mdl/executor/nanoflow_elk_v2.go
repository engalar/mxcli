// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — Nanoflow ELK visualization (gen-typed track).
//
// This file is the gen-typed counterpart to legacy `cmd_nanoflow_elk.go`.
// It also seeds the shared elk graph builders (`buildFlowELKGen`,
// `buildMicroflowELKNodeGen`, `buildMicroflowELKNodeHierarchicalGen`,
// `buildMicroflowELKEdgeGen`, `calcMicroflowNodeSizeGen`,
// `emitMicroflowELKGen`) so commit 3 (cmd_microflow_elk_gen.go) can
// add the Microflow entry on top without re-introducing them.
//
// Legacy is left untouched and continues to drive the active code path
// (cmd/mxcli/cmd_describe.go → Executor.NanoflowELK). Tests in this
// file exercise the parallel `*Gen` entry directly.

package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ────────────────────────────────────────────────────────
// Entry point — gen-typed Nanoflow ELK rendering
// ────────────────────────────────────────────────────────

// NanoflowELKGen is the gen-typed parallel of Executor.NanoflowELK.
// Resolves the nanoflow via the gen NanoflowRepository and renders
// the same JSON ELK graph schema legacy produces.
func (e *Executor) NanoflowELKGen(name string) error {
	return nanoflowELKGen(e.newExecContext(context.Background()), name)
}

// NanoflowELK is the public Executor wrapper consumed by
// cmd/mxcli/cmd_describe.go. Stage 3.2.6.3a moved this method out of
// the (deleted) cmd_nanoflow_elk.go; the body always routes through
// the gen path now that legacy `nanoflowELK` is gone.
func (e *Executor) NanoflowELK(name string) error {
	return nanoflowELKGen(e.newExecContext(context.Background()), name)
}

func nanoflowELKGen(ctx *ExecContext, name string) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if ctx.Nanoflows == nil {
		return mdlerrors.NewBackend("nanoflow repository", fmt.Errorf("ctx.Nanoflows is nil"))
	}

	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return mdlerrors.NewValidationf("expected qualified name Module.Nanoflow, got: %s", name)
	}
	qn := ast.QualifiedName{Module: parts[0], Name: parts[1]}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	entityNames, err := buildEntityNames(ctx, h)
	if err != nil {
		return err
	}

	// Locate by walking all nanoflows and matching module+name —
	// nanoflow repo currently has List(moduleID) but not a direct
	// FindByQualifiedName analogue. Walking is a few microseconds.
	allNanoflows, err := ctx.Nanoflows.List("")
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	var targetNf *genMf.Nanoflow
	for _, nf := range allNanoflows {
		modName := lookupGenContainerModule(ctx, h, nf.ID())
		if modName == qn.Module && nf.Name() == qn.Name {
			targetNf = nf
			break
		}
	}
	if targetNf == nil {
		return mdlerrors.NewNotFound("nanoflow", name)
	}

	// MDL source for source map — best-effort, diagram works without it.
	mdlSource, sourceMap, _ := describeNanoflowGenToString(ctx, qn)

	return buildFlowELKGen(ctx, flowELKInputGen{
		FlowType:         "nanoflow",
		QualifiedName:    name,
		ReturnType:       targetNf.ReturnType(),
		Parameters:       genFlowParametersFromCollection(targetNf.ObjectCollection()),
		ObjectCollection: targetNf.ObjectCollection(),
		TopLevelFlows:    targetNf.FlowsItems(),
		EntityNames:      entityNames,
		MdlSource:        mdlSource,
		SourceMap:        sourceMap,
	})
}

// ────────────────────────────────────────────────────────
// Shared gen ELK input + builders (used by Microflow entry too)
// ────────────────────────────────────────────────────────

// flowELKInputGen mirrors legacy `flowELKInput` but accepts gen types.
//
// Flows live on the Microflow / Nanoflow itself in the gen surface
// (not on the ObjectCollection); callers extract them with
// `FlowsItems()` and pass them in via TopLevelFlows.
type flowELKInputGen struct {
	FlowType         string
	QualifiedName    string
	ReturnType       string
	Parameters       []genFlowParameter
	ObjectCollection element.Element
	TopLevelFlows    []element.Element
	EntityNames      map[model.ID]string
	MdlSource        string
	SourceMap        map[string]elkSourceRange
}

// genFlowParameter is a flat (name, type) pair extracted from gen
// MicroflowParameter elements. We avoid leaking the gen type itself
// into builder signatures.
type genFlowParameter struct {
	Name string
	Type string
}

// genFlowParametersFromCollection collects MicroflowParameter elements
// from a gen ObjectCollection. Mirrors legacy's `Parameters` field;
// gen models parameters as objects inside the same collection.
func genFlowParametersFromCollection(oc element.Element) []genFlowParameter {
	col, ok := oc.(*genMf.MicroflowObjectCollection)
	if !ok || col == nil {
		return nil
	}
	var out []genFlowParameter
	for _, obj := range col.ObjectsItems() {
		if obj == nil {
			continue
		}
		if obj.TypeName() != "Microflows$MicroflowParameter" {
			continue
		}
		var p genFlowParameter
		if nv, ok := obj.(interface{ Name() string }); ok {
			p.Name = nv.Name()
		}
		if pt, ok := obj.(interface{ Type() string }); ok {
			p.Type = pt.Type()
		}
		out = append(out, p)
	}
	return out
}

// buildFlowELKGen renders the ELK JSON graph for any gen flow type.
// 1:1 textual match with legacy `buildFlowELK` for fixture diff = 0.
func buildFlowELKGen(ctx *ExecContext, in flowELKInputGen) error {
	data := microflowELKData{
		Format:     "elk",
		Type:       in.FlowType,
		Name:       in.QualifiedName,
		ReturnType: in.ReturnType,
		MdlSource:  in.MdlSource,
		SourceMap:  in.SourceMap,
	}

	// Parameters
	for _, p := range in.Parameters {
		data.Parameters = append(data.Parameters, microflowELKParam{
			Name: p.Name,
			Type: p.Type,
		})
	}

	col, hasCol := in.ObjectCollection.(*genMf.MicroflowObjectCollection)
	if !hasCol || col == nil || len(genActivityObjects(col)) == 0 {
		data.Nodes = []microflowELKNode{
			{ID: "node-start", Type: "start", Category: "event", Label: "Start", Width: 80, Height: 36},
			{ID: "node-end", Type: "end", Category: "event", Label: "End", Width: 70, Height: 36},
		}
		data.Edges = []microflowELKEdge{
			{ID: "edge-0", SourceID: "node-start", TargetID: "node-end"},
		}
		return emitMicroflowELKGen(ctx, data)
	}

	// Build nodes — loops become compound nodes with children.
	for _, obj := range genActivityObjects(col) {
		node := buildMicroflowELKNodeHierarchicalGen(obj, in.EntityNames, 0)
		data.Nodes = append(data.Nodes, node)
	}

	// Build edges — top-level flows live on the Microflow/Nanoflow,
	// passed in via in.TopLevelFlows. Loop body flows are still
	// co-mingled with activities inside the loop's own ObjectCollection
	// (handled by buildMicroflowELKNodeHierarchicalGen).
	flows := genSequenceFlowsFromList(in.TopLevelFlows)
	sort.SliceStable(flows, func(i, j int) bool {
		return flows[i].OriginConnectionIndex() < flows[j].OriginConnectionIndex()
	})
	for i, f := range flows {
		edge := buildMicroflowELKEdgeGen(f, i, "edge")
		data.Edges = append(data.Edges, edge)
	}

	return emitMicroflowELKGen(ctx, data)
}

// genSequenceFlowsFromList filters a slice of element.Element for
// SequenceFlow elements, returning the strongly-typed slice.
func genSequenceFlowsFromList(items []element.Element) []*genMf.SequenceFlow {
	var out []*genMf.SequenceFlow
	for _, obj := range items {
		if sf, ok := obj.(*genMf.SequenceFlow); ok && sf != nil {
			out = append(out, sf)
		}
	}
	return out
}

// genActivityObjects returns the non-flow, non-parameter activity
// elements of a gen MicroflowObjectCollection. Sequence flows and
// MicroflowParameter elements are filtered out so we render only the
// activity nodes legacy emitted (the `Objects` array on legacy's
// `MicroflowObjectCollection`).
func genActivityObjects(col *genMf.MicroflowObjectCollection) []element.Element {
	if col == nil {
		return nil
	}
	out := make([]element.Element, 0, len(col.ObjectsItems()))
	for _, obj := range col.ObjectsItems() {
		if obj == nil {
			continue
		}
		switch obj.(type) {
		case *genMf.SequenceFlow, *genMf.MicroflowParameter,
			*genMf.Annotation, *genMf.AnnotationFlow:
			continue
		}
		out = append(out, obj)
	}
	return out
}

// genSequenceFlows extracts SequenceFlow elements from a gen
// MicroflowObjectCollection. Legacy stored flows in the `Flows`
// PartList; gen co-mingles them in `Objects` alongside activities.
func genSequenceFlows(col *genMf.MicroflowObjectCollection) []*genMf.SequenceFlow {
	if col == nil {
		return nil
	}
	var out []*genMf.SequenceFlow
	for _, obj := range col.ObjectsItems() {
		if sf, ok := obj.(*genMf.SequenceFlow); ok && sf != nil {
			out = append(out, sf)
		}
	}
	return out
}

// buildMicroflowELKNodeGen builds a leaf ELK node from a gen
// activity. Loops are handled by `buildMicroflowELKNodeHierarchicalGen`.
func buildMicroflowELKNodeGen(obj element.Element, entityNames map[model.ID]string) microflowELKNode {
	id := "node-" + string(obj.ID())
	label := mermaidActivityLabelGen(obj, entityNames)
	label = strings.ReplaceAll(label, "#quot;", "\"")

	details := mermaidActivityDetailsGen(obj, entityNames)
	for i, d := range details {
		details[i] = strings.ReplaceAll(d, "#quot;", "\"")
	}

	nodeType, category := classifyMicroflowNodeGen(obj)
	width, height := calcMicroflowNodeSizeGen(nodeType, label, details)

	return microflowELKNode{
		ID:       id,
		Type:     nodeType,
		Category: category,
		Label:    label,
		Details:  details,
		Width:    width,
		Height:   height,
	}
}

// buildMicroflowELKNodeHierarchicalGen handles `*genMf.LoopedActivity`
// as a compound node with children (the loop body) and inner edges.
func buildMicroflowELKNodeHierarchicalGen(obj element.Element, entityNames map[model.ID]string, depth int) microflowELKNode {
	loop, isLoop := obj.(*genMf.LoopedActivity)
	if !isLoop {
		return buildMicroflowELKNodeGen(obj, entityNames)
	}
	loopCol, ok := loop.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || loopCol == nil || len(genActivityObjects(loopCol)) == 0 {
		return buildMicroflowELKNodeGen(obj, entityNames)
	}

	id := "node-" + string(loop.ID())
	label := mermaidActivityLabelGen(obj, entityNames)
	label = strings.ReplaceAll(label, "#quot;", "\"")
	details := mermaidActivityDetailsGen(obj, entityNames)
	for i, d := range details {
		details[i] = strings.ReplaceAll(d, "#quot;", "\"")
	}

	node := microflowELKNode{
		ID:       id,
		Type:     "loop",
		Category: "loop",
		Label:    label,
		Details:  details,
		// Width/Height: 0 — ELK computes from children + padding.
	}

	for _, childObj := range genActivityObjects(loopCol) {
		child := buildMicroflowELKNodeHierarchicalGen(childObj, entityNames, depth+1)
		node.Children = append(node.Children, child)
	}

	innerFlows := genSequenceFlows(loopCol)
	sort.SliceStable(innerFlows, func(i, j int) bool {
		return innerFlows[i].OriginConnectionIndex() < innerFlows[j].OriginConnectionIndex()
	})
	for i, f := range innerFlows {
		edge := buildMicroflowELKEdgeGen(f, i, id+"-edge")
		node.Edges = append(node.Edges, edge)
	}
	return node
}

// buildMicroflowELKEdgeGen builds an ELK edge from a gen sequence flow.
func buildMicroflowELKEdgeGen(flow *genMf.SequenceFlow, index int, prefix string) microflowELKEdge {
	edge := microflowELKEdge{
		ID:       fmt.Sprintf("%s-%d", prefix, index),
		SourceID: "node-" + string(flow.OriginRefID()),
		TargetID: "node-" + string(flow.DestinationRefID()),
	}

	label := mermaidCaseLabelGen(flow.CaseValue())
	if label != "" {
		label = strings.ReplaceAll(label, "#quot;", "\"")
	}
	edge.Label = label
	edge.IsErrorHandler = flow.IsErrorHandler()

	return edge
}

// calcMicroflowNodeSizeGen returns width/height for a gen ELK node.
// Identical math to legacy `calcMicroflowNodeSize`.
func calcMicroflowNodeSizeGen(nodeType, label string, details []string) (float64, float64) {
	switch nodeType {
	case "start", "end", "continue", "break", "error":
		w := float64(len(label))*elkCharWidth + elkHPadding*2
		if w < 70 {
			w = 70
		}
		return w, 36
	case "split":
		w := float64(len(label))*elkCharWidth + elkHPadding*2
		if w < 100 {
			w = 100
		}
		return w, 60
	case "merge":
		return 24, 24
	case "loop":
		maxLen := float64(len(label))
		for _, d := range details {
			if l := float64(len(d)); l > maxLen {
				maxLen = l
			}
		}
		w := maxLen*elkCharWidth + elkHPadding
		if w < elkMinWidth {
			w = elkMinWidth
		}
		h := elkHeaderHeight + float64(len(details))*16
		if len(details) == 0 {
			h = elkHeaderHeight + 16
		}
		return w, h
	default:
		maxLen := float64(len(label))
		for _, d := range details {
			if l := float64(len(d)); l > maxLen {
				maxLen = l
			}
		}
		w := maxLen*elkCharWidth + elkHPadding
		if w < elkMinWidth {
			w = elkMinWidth
		}
		h := elkHeaderHeight
		if len(details) > 0 {
			h += float64(len(details)) * 16
		}
		if h < elkHeaderHeight+8 {
			h = elkHeaderHeight + 8
		}
		return math.Ceil(w), math.Ceil(h)
	}
}

// emitMicroflowELKGen marshals and writes the data — same JSON shape
// as legacy `emitMicroflowELK`.
func emitMicroflowELKGen(ctx *ExecContext, data microflowELKData) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mdlerrors.NewBackend("marshal json", err)
	}
	fmt.Fprint(ctx.Output, string(out))
	return nil
}

// ────────────────────────────────────────────────────────
// Module-name resolution for gen flow IDs
// ────────────────────────────────────────────────────────

// lookupGenContainerModule resolves the module name for a gen flow's
// ID using the SQL-backed container chain — gen objects do not carry
// container linkage after a BSON decode roundtrip. Returns "" when
// the container chain doesn't resolve to a module.
func lookupGenContainerModule(ctx *ExecContext, h *ContainerHierarchy, id element.ID) string {
	if id == "" || ctx == nil || ctx.Microflows == nil {
		return ""
	}
	containerID, err := ctx.Microflows.GetContainerUUID(model.ID(id))
	if err != nil || containerID == "" {
		return ""
	}
	modID := h.FindModuleID(containerID)
	return h.GetModuleName(modID)
}
