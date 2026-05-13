// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.1: gen-typed DESCRIBE MICROFLOW skeleton.
//
// This file is the new implementation track for DESCRIBE MICROFLOW
// that consumes *modelsdk/gen/microflows.Microflow exclusively. It
// lives alongside the legacy `cmd_microflows_show.go` (which still
// drives sdk/microflows-typed callers) — both paths coexist until
// Stage 3.2.x has migrated every call site.
//
// Scope of 3.2.1:
//   - Headers: documentation, @excluded, create-or-modify line,
//     parameters list, returns clause.
//   - Body framing: traverses the SequenceFlow graph using the same
//     splitMergeMap algorithm as legacy traverseFlow, emitting
//     `if/else/end if`, `case/end case`, `loop/end loop`, and
//     `while true/end while` for the six structural node kinds
//     (StartEvent / EndEvent / ExclusiveSplit / InheritanceSplit /
//     LoopedActivity / ExclusiveMerge).
//   - Activity bodies (ActionActivity subtypes, etc.): emit a single
//     `// TODO Stage 3.2.2: format <Go-type-name>` line as a
//     placeholder. Filling these in is the work of subsequent stages.

package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// DescribeMicroflowGenToString renders a *genMf.Microflow as MDL source
// using only the modelsdk/gen API surface. It returns the rendered text
// (no error path is exercised today; the signature mirrors the legacy
// describeMicroflowToString return shape so callers can be migrated
// without churn). Activity bodies inside structural framing are emitted
// as `// TODO Stage 3.2.2: format <typename>` placeholders.
func DescribeMicroflowGenToString(ctx *ExecContext, mf *genMf.Microflow) (string, error) {
	if mf == nil {
		return "", fmt.Errorf("DescribeMicroflowGenToString: nil microflow")
	}

	moduleName, qualifiedName := genMicroflowQualifiedName(ctx, mf)

	var lines []string

	// Documentation block.
	if doc := mf.Documentation(); doc != "" {
		lines = append(lines, "/**")
		for _, dl := range strings.Split(doc, "\n") {
			lines = append(lines, " * "+dl)
		}
		lines = append(lines, " */")
	}

	// @excluded annotation.
	if mf.Excluded() {
		lines = append(lines, "@excluded")
	}

	// Header. Parameters live inside the ObjectCollection as
	// MicroflowParameter elements (Stage 3.2.2 will materialise them);
	// for now we emit `()` and a TODO note inside the begin block.
	params := genMicroflowParameters(mf)
	if len(params) > 0 {
		lines = append(lines, fmt.Sprintf("create or modify microflow %s (", qualifiedName))
		for i, p := range params {
			comma := ","
			if i == len(params)-1 {
				comma = ""
			}
			lines = append(lines, fmt.Sprintf("  $%s: %s%s", p.name, p.declType, comma))
		}
		lines = append(lines, ")")
	} else {
		lines = append(lines, fmt.Sprintf("create or modify microflow %s ()", qualifiedName))
	}

	// Returns clause (skeleton: surface only the primitive name; full
	// formatting — entity QN resolution, "List of X", enums — comes in
	// 3.2.2 alongside the activity formatters).
	if rt := strings.TrimSpace(mf.ReturnType()); rt != "" && !strings.EqualFold(rt, "Void") {
		returnLine := "returns " + rt
		if rv := mf.ReturnVariableName(); rv != "" && rv != "Variable" {
			returnLine += " as $" + rv
		}
		lines = append(lines, returnLine)
	}

	// Begin block.
	lines = append(lines, "begin")

	bodyLines := renderGenMicroflowBody(ctx, mf)
	if len(bodyLines) == 0 {
		lines = append(lines, "  -- No activities")
	} else {
		for _, l := range bodyLines {
			lines = append(lines, "  "+l)
		}
	}
	lines = append(lines, "end;")

	// Allowed module roles → grant execute footer.
	if roles := mf.AllowedModuleRolesQualifiedNames(); len(roles) > 0 {
		// Strip module-qualifier; the legacy emitter writes bare role names.
		bare := make([]string, 0, len(roles))
		for _, r := range roles {
			bare = append(bare, lastDotSegment(r))
		}
		simple := strings.SplitN(qualifiedName, ".", 2)
		if len(simple) == 2 {
			lines = append(lines,
				"",
				fmt.Sprintf("grant execute on microflow %s.%s to %s;", moduleName, simple[1], strings.Join(bare, ", ")),
			)
		}
	}
	lines = append(lines, "/")

	return strings.Join(lines, "\n"), nil
}

// genMicroflowQualifiedName resolves the microflow's owning module name
// and returns (moduleName, "Module.Name").
//
// Resolution strategy (in order):
//   1. ctx.Microflows.GetContainerUUID(mf.ID) + ctx.Cache.hierarchy
//      (FindModuleID + GetModuleName) — the canonical path that works
//      after a BSON roundtrip strips Container() linkage.
//   2. Walk Container() chain looking for `Projects$Module` (works for
//      freshly-built in-memory graphs that still carry container refs).
//   3. Fallback "<unknown>.Name" — only if no other source resolves it.
func genMicroflowQualifiedName(ctx *ExecContext, mf *genMf.Microflow) (string, string) {
	name := mf.Name()
	module := ""

	// Path 1: SQL-backed container UUID + ContainerHierarchy.
	if ctx != nil && ctx.Microflows != nil && ctx.Connected() {
		if containerID, err := ctx.Microflows.GetContainerUUID(model.ID(mf.ID())); err == nil && containerID != "" {
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
		for c := mf.Container(); c != nil; c = c.Container() {
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

type genParamSummary struct {
	name     string
	declType string
}

// genMicroflowParameters extracts parameter summaries from the
// ObjectCollection. Parameters are stored as `Microflows$MicroflowParameter`
// elements alongside activities; we read them via TypeName + NameValue
// without depending on a strongly-typed accessor (gen MicroflowParameter
// type does not expose a typed enumerator on Microflow).
func genMicroflowParameters(mf *genMf.Microflow) []genParamSummary {
	oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil
	}
	var out []genParamSummary
	for _, obj := range oc.ObjectsItems() {
		if obj == nil {
			continue
		}
		if obj.TypeName() != "Microflows$MicroflowParameter" {
			continue
		}
		s := genParamSummary{declType: "Object"}
		if nv, ok := obj.(interface{ NameValue() string }); ok {
			s.name = nv.NameValue()
		}
		if pt, ok := obj.(interface{ Type() string }); ok {
			if t := pt.Type(); t != "" {
				s.declType = t
			}
		}
		out = append(out, s)
	}
	return out
}

// lastDotSegment returns the substring after the last '.' (or s itself).
// Used to bare-down a "Module.Role" role qualified name.
func lastDotSegment(s string) string {
	i := strings.LastIndex(s, ".")
	if i < 0 {
		return s
	}
	return s[i+1:]
}

// ────────────────────────────────────────────────────────
// Body rendering — gen-typed traverseFlow port
// ────────────────────────────────────────────────────────

// renderGenMicroflowBody renders the body of a microflow using only
// gen types. It returns the body lines (without the surrounding
// `begin` / `end;` framing).
func renderGenMicroflowBody(ctx *ExecContext, mf *genMf.Microflow) []string {
	oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
	if !ok || oc == nil {
		return nil
	}

	objects := oc.ObjectsItems()
	flowsByOrigin, flowsByDest := buildGenFlowMaps(mf)
	activityMap, startID := buildGenActivityMap(objects)
	splitMergeMap := findSplitMergePointsGen(activityMap, flowsByOrigin)

	// Sort each origin's outgoing flows by OriginConnectionIndex so
	// rendering order is deterministic — mirrors legacy bubble sort
	// in formatMicroflowActivities.
	for k := range flowsByOrigin {
		flows := flowsByOrigin[k]
		sort.SliceStable(flows, func(i, j int) bool {
			return flows[i].OriginConnectionIndex() < flows[j].OriginConnectionIndex()
		})
	}

	visited := make(map[element.ID]bool)
	var lines []string
	traverseFlowGen(ctx, startID, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, &lines, 0)
	return lines
}

// buildGenFlowMaps constructs origin/dest indexes for the microflow's
// SequenceFlows. Annotation flows and other non-SequenceFlow elements
// are ignored.
func buildGenFlowMaps(mf *genMf.Microflow) (map[element.ID][]*genMf.SequenceFlow, map[element.ID][]*genMf.SequenceFlow) {
	byOrigin := make(map[element.ID][]*genMf.SequenceFlow)
	byDest := make(map[element.ID][]*genMf.SequenceFlow)
	for _, fe := range mf.FlowsItems() {
		sf, ok := fe.(*genMf.SequenceFlow)
		if !ok || sf == nil {
			continue
		}
		byOrigin[sf.OriginRefID()] = append(byOrigin[sf.OriginRefID()], sf)
		byDest[sf.DestinationRefID()] = append(byDest[sf.DestinationRefID()], sf)
	}
	return byOrigin, byDest
}

// buildGenActivityMap returns id→element and the StartEvent ID, if any.
func buildGenActivityMap(objects []element.Element) (map[element.ID]element.Element, element.ID) {
	m := make(map[element.ID]element.Element, len(objects))
	var startID element.ID
	for _, o := range objects {
		if o == nil {
			continue
		}
		m[o.ID()] = o
		if _, ok := o.(*genMf.StartEvent); ok {
			startID = o.ID()
		}
	}
	return m, startID
}

// findNormalFlowsGen filters out error-handler flows.
func findNormalFlowsGen(flows []*genMf.SequenceFlow) []*genMf.SequenceFlow {
	out := flows[:0:0]
	for _, f := range flows {
		if !f.IsErrorHandler() {
			out = append(out, f)
		}
	}
	return out
}

// findSplitMergePointsGen mirrors legacy findSplitMergePointsForGraph
// over the gen graph. For each ExclusiveSplit / InheritanceSplit it
// finds the nearest common downstream node where the branches re-join.
func findSplitMergePointsGen(
	activityMap map[element.ID]element.Element,
	flowsByOrigin map[element.ID][]*genMf.SequenceFlow,
) map[element.ID]element.ID {
	result := make(map[element.ID]element.ID)
	for id, obj := range activityMap {
		switch obj.(type) {
		case *genMf.ExclusiveSplit, *genMf.InheritanceSplit:
			if mergeID := findMergeForSplitGen(id, flowsByOrigin, activityMap); mergeID != "" {
				result[id] = mergeID
			}
		}
	}
	return result
}

func findMergeForSplitGen(
	splitID element.ID,
	flowsByOrigin map[element.ID][]*genMf.SequenceFlow,
	activityMap map[element.ID]element.Element,
) element.ID {
	flows := findNormalFlowsGen(flowsByOrigin[splitID])
	if len(flows) < 2 {
		return ""
	}
	branchDistances := make([]map[element.ID]int, 0, len(flows))
	for _, f := range flows {
		branchDistances = append(branchDistances, collectReachableDistancesGen(f.DestinationRefID(), flowsByOrigin))
	}
	return selectNearestCommonJoinGen(activityMap, branchDistances)
}

func collectReachableDistancesGen(
	startID element.ID,
	flowsByOrigin map[element.ID][]*genMf.SequenceFlow,
) map[element.ID]int {
	distances := map[element.ID]int{}
	type item struct {
		id   element.ID
		dist int
	}
	q := []item{{id: startID}}
	for len(q) > 0 {
		it := q[0]
		q = q[1:]
		if prev, ok := distances[it.id]; ok && prev <= it.dist {
			continue
		}
		distances[it.id] = it.dist
		for _, f := range findNormalFlowsGen(flowsByOrigin[it.id]) {
			q = append(q, item{id: f.DestinationRefID(), dist: it.dist + 1})
		}
	}
	return distances
}

func selectNearestCommonJoinGen(
	activityMap map[element.ID]element.Element,
	branchDistances []map[element.ID]int,
) element.ID {
	if len(branchDistances) < 2 {
		return ""
	}
	type cand struct {
		id          element.ID
		maxDistance int
		sumDistance int
	}
	var candidates []cand
	for nodeID, firstDist := range branchDistances[0] {
		obj := activityMap[nodeID]
		if !isSplitJoinCandidateGen(obj) {
			continue
		}
		max := firstDist
		sum := firstDist
		common := true
		for _, d := range branchDistances[1:] {
			dd, ok := d[nodeID]
			if !ok {
				common = false
				break
			}
			if dd > max {
				max = dd
			}
			sum += dd
		}
		if common {
			candidates = append(candidates, cand{id: nodeID, maxDistance: max, sumDistance: sum})
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].maxDistance != candidates[j].maxDistance {
			return candidates[i].maxDistance < candidates[j].maxDistance
		}
		if candidates[i].sumDistance != candidates[j].sumDistance {
			return candidates[i].sumDistance < candidates[j].sumDistance
		}
		return string(candidates[i].id) < string(candidates[j].id)
	})
	return candidates[0].id
}

func isSplitJoinCandidateGen(obj element.Element) bool {
	switch obj.(type) {
	case nil, *genMf.StartEvent, *genMf.EndEvent:
		return false
	default:
		return true
	}
}

// ────────────────────────────────────────────────────────
// traverseFlowGen — port of legacy traverseFlow, gen types
// ────────────────────────────────────────────────────────

func traverseFlowGen(
	ctx *ExecContext,
	currentID element.ID,
	activityMap map[element.ID]element.Element,
	flowsByOrigin map[element.ID][]*genMf.SequenceFlow,
	flowsByDest map[element.ID][]*genMf.SequenceFlow,
	splitMergeMap map[element.ID]element.ID,
	visited map[element.ID]bool,
	lines *[]string,
	indent int,
) {
	if currentID == "" || visited[currentID] {
		return
	}
	obj := activityMap[currentID]
	if obj == nil {
		return
	}

	// Merge passthrough — paired merges are processed by their split's
	// recursion and would otherwise emit a duplicate `end if;`.
	if _, isMerge := obj.(*genMf.ExclusiveMerge); isMerge {
		if isMergePairedWithSplitGen(currentID, splitMergeMap) {
			return
		}
		visited[currentID] = true
		for _, f := range flowsByOrigin[currentID] {
			traverseFlowGen(ctx, f.DestinationRefID(), activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
		}
		return
	}

	visited[currentID] = true
	indentStr := strings.Repeat("  ", indent)

	// StartEvent: silent (no MDL emission); follow its outgoing flows.
	if _, isStart := obj.(*genMf.StartEvent); isStart {
		for _, f := range findNormalFlowsGen(flowsByOrigin[currentID]) {
			traverseFlowGen(ctx, f.DestinationRefID(), activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
		}
		return
	}

	// EndEvent: emit `return …;` framing. Body is a placeholder.
	if end, isEnd := obj.(*genMf.EndEvent); isEnd {
		emitEndEventGen(ctx, end, lines, indentStr)
		return
	}

	// InheritanceSplit: case-on-inheritance framing.
	if split, isInh := obj.(*genMf.InheritanceSplit); isInh && len(findNormalFlowsGen(flowsByOrigin[currentID])) > 1 {
		mergeID := splitMergeMap[currentID]
		emitInheritanceSplitGen(ctx, currentID, mergeID, split, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
		// Continue past the merge.
		if mergeID != "" {
			visited[mergeID] = true
			for _, f := range flowsByOrigin[mergeID] {
				traverseFlowGen(ctx, f.DestinationRefID(), activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
			}
		}
		return
	}

	// ExclusiveSplit: if/else or case-on-enum framing.
	if split, isXSplit := obj.(*genMf.ExclusiveSplit); isXSplit {
		mergeID := splitMergeMap[currentID]
		emitExclusiveSplitGen(ctx, currentID, mergeID, split, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
		// Continue past the merge.
		if mergeID != "" {
			visited[mergeID] = true
			for _, f := range flowsByOrigin[mergeID] {
				traverseFlowGen(ctx, f.DestinationRefID(), activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
			}
		}
		return
	}

	// LoopedActivity: foreach / while framing with nested body.
	if loop, isLoop := obj.(*genMf.LoopedActivity); isLoop {
		emitLoopedActivityGen(ctx, loop, lines, indentStr, indent)
		// Continue past the loop.
		for _, f := range flowsByOrigin[currentID] {
			traverseFlowGen(ctx, f.DestinationRefID(), activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
		}
		return
	}

	// Any other node: try the gen-typed activity formatter first; fall
	// back to a TODO placeholder for unsupported kinds.
	if rendered := formatActivityGen(ctx, obj); rendered != "" {
		*lines = append(*lines, indentStr+rendered)
	} else {
		*lines = append(*lines, indentStr+placeholderForGen(obj))
	}
	for _, f := range findNormalFlowsGen(flowsByOrigin[currentID]) {
		traverseFlowGen(ctx, f.DestinationRefID(), activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
	}
}

// isMergePairedWithSplitGen reports whether mergeID appears as a value in
// splitMergeMap (i.e. it is the structural pair of some split).
func isMergePairedWithSplitGen(mergeID element.ID, splitMergeMap map[element.ID]element.ID) bool {
	for _, v := range splitMergeMap {
		if v == mergeID {
			return true
		}
	}
	return false
}

// traverseFlowGenUntilMerge recursively walks until reaching mergeID,
// without crossing it. Used to emit a branch body framed by an
// `if`/`case`/`when` statement.
func traverseFlowGenUntilMerge(
	ctx *ExecContext,
	currentID element.ID,
	mergeID element.ID,
	activityMap map[element.ID]element.Element,
	flowsByOrigin map[element.ID][]*genMf.SequenceFlow,
	flowsByDest map[element.ID][]*genMf.SequenceFlow,
	splitMergeMap map[element.ID]element.ID,
	visited map[element.ID]bool,
	lines *[]string,
	indent int,
) {
	if currentID == "" || currentID == mergeID || visited[currentID] {
		return
	}
	traverseFlowGen(ctx, currentID, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent)
}

// ────────────────────────────────────────────────────────
// Emit helpers (six framing kinds)
// ────────────────────────────────────────────────────────

func emitEndEventGen(ctx *ExecContext, end *genMf.EndEvent, lines *[]string, indentStr string) {
	rv := strings.TrimSpace(end.ReturnValue())
	if ctx != nil && ctx.DescribingMicroflowHasReturnValue && rv == "" {
		// Defensive: a value-returning microflow with a bare `return`
		// is invalid MDL — emit a placeholder rather than mis-render.
		*lines = append(*lines, indentStr+"// TODO Stage 3.2.2: format Microflows$EndEvent (missing return value)")
		return
	}
	if rv == "" {
		*lines = append(*lines, indentStr+"return;")
		return
	}
	*lines = append(*lines, indentStr+"return "+rv+";")
}

func emitExclusiveSplitGen(
	ctx *ExecContext,
	currentID element.ID,
	mergeID element.ID,
	split *genMf.ExclusiveSplit,
	activityMap map[element.ID]element.Element,
	flowsByOrigin map[element.ID][]*genMf.SequenceFlow,
	flowsByDest map[element.ID][]*genMf.SequenceFlow,
	splitMergeMap map[element.ID]element.ID,
	visited map[element.ID]bool,
	lines *[]string,
	indent int,
) {
	indentStr := strings.Repeat("  ", indent)
	flows := findNormalFlowsGen(flowsByOrigin[currentID])

	// Case-on-enum split: ExpressionSplitCondition with > 1 branch and
	// flows carrying CaseValue elements is rendered as `case $expr`.
	// Skeleton output: `case <expr>` + one `when …` per flow + `end case;`.
	if isEnumSplitGen(split, flows) {
		expr := exclusiveSplitExpressionGen(split)
		*lines = append(*lines, indentStr+"case "+expr)
		for _, f := range flows {
			label := caseValueLabelGen(f)
			*lines = append(*lines, indentStr+"  when "+label+" then")
			traverseFlowGenUntilMerge(ctx, f.DestinationRefID(), mergeID, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent+2)
		}
		*lines = append(*lines, indentStr+"end case;")
		return
	}

	// Boolean split: emit `if <expr> then` with optional `else` branch.
	expr := exclusiveSplitExpressionGen(split)
	*lines = append(*lines, indentStr+"if "+expr+" then")

	trueFlow, falseFlow := pickBooleanBranchesGen(flows)
	if trueFlow != nil {
		traverseFlowGenUntilMerge(ctx, trueFlow.DestinationRefID(), mergeID, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent+1)
	}
	if falseFlow != nil && falseFlow.DestinationRefID() != mergeID {
		*lines = append(*lines, indentStr+"else")
		traverseFlowGenUntilMerge(ctx, falseFlow.DestinationRefID(), mergeID, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent+1)
	}
	*lines = append(*lines, indentStr+"end if;")
}

func emitInheritanceSplitGen(
	ctx *ExecContext,
	currentID element.ID,
	mergeID element.ID,
	split *genMf.InheritanceSplit,
	activityMap map[element.ID]element.Element,
	flowsByOrigin map[element.ID][]*genMf.SequenceFlow,
	flowsByDest map[element.ID][]*genMf.SequenceFlow,
	splitMergeMap map[element.ID]element.ID,
	visited map[element.ID]bool,
	lines *[]string,
	indent int,
) {
	indentStr := strings.Repeat("  ", indent)
	varName := strings.TrimSpace(split.SplitVariableName())
	if varName == "" {
		varName = "Variable"
	}
	*lines = append(*lines, indentStr+"case $"+varName+" inheritance")
	for _, f := range findNormalFlowsGen(flowsByOrigin[currentID]) {
		// CaseValueLabel is filled in 3.2.2 — for now the branch tag
		// is the placeholder so the structural framing is testable.
		*lines = append(*lines, indentStr+"  when // TODO Stage 3.2.2: format inheritance case label then")
		traverseFlowGenUntilMerge(ctx, f.DestinationRefID(), mergeID, activityMap, flowsByOrigin, flowsByDest, splitMergeMap, visited, lines, indent+2)
	}
	*lines = append(*lines, indentStr+"end case;")
}

func emitLoopedActivityGen(
	ctx *ExecContext,
	loop *genMf.LoopedActivity,
	lines *[]string,
	indentStr string,
	indent int,
) {
	// Header line: `loop` / `loop $var in $list` / `while <cond>`. The
	// detailed source-rendering of the LoopSource expression is 3.2.2's
	// job — emit a placeholder header and let the structural keywords
	// be exercised by tests.
	header := "loop"
	endKw := "end loop;"
	if _, isWhile := loop.LoopSource().(*genMf.WhileLoopCondition); isWhile {
		header = "while true"
		endKw = "end while;"
	}
	*lines = append(*lines, indentStr+header)
	*lines = append(*lines, indentStr+"begin")

	// Nested body: traverse the loop's own ObjectCollection. Reuses
	// the same gen traverser so structural framing nests correctly.
	if oc, ok := loop.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && oc != nil {
		// Build inner graph from the loop body's own SequenceFlows.
		// LoopedActivity.ObjectCollection holds both the body objects
		// and the body's flows (they live alongside as PartList items
		// of the same collection).
		inner := oc.ObjectsItems()
		innerByOrigin := make(map[element.ID][]*genMf.SequenceFlow)
		innerByDest := make(map[element.ID][]*genMf.SequenceFlow)
		innerActivities, _ := buildGenActivityMap(inner)
		for _, e := range inner {
			if sf, ok := e.(*genMf.SequenceFlow); ok && sf != nil {
				innerByOrigin[sf.OriginRefID()] = append(innerByOrigin[sf.OriginRefID()], sf)
				innerByDest[sf.DestinationRefID()] = append(innerByDest[sf.DestinationRefID()], sf)
			}
		}
		for k := range innerByOrigin {
			flows := innerByOrigin[k]
			sort.SliceStable(flows, func(i, j int) bool {
				return flows[i].OriginConnectionIndex() < flows[j].OriginConnectionIndex()
			})
		}
		innerSplitMerge := findSplitMergePointsGen(innerActivities, innerByOrigin)
		var startID element.ID
		// Loop bodies often do not have an explicit StartEvent — pick
		// a node with no incoming sequence flow as the entry, falling
		// back to the first object.
		for id := range innerActivities {
			if len(innerByDest[id]) == 0 {
				startID = id
				break
			}
		}
		if startID == "" && len(inner) > 0 {
			startID = inner[0].ID()
		}
		visited := make(map[element.ID]bool)
		var bodyLines []string
		traverseFlowGen(ctx, startID, innerActivities, innerByOrigin, innerByDest, innerSplitMerge, visited, &bodyLines, indent+1)
		*lines = append(*lines, bodyLines...)
	}
	*lines = append(*lines, indentStr+endKw)
}

// ────────────────────────────────────────────────────────
// Small utilities
// ────────────────────────────────────────────────────────

// placeholderForGen emits the canonical TODO marker used in 3.2.1 for
// any non-structural activity node. The Go type name lets 3.2.2
// formatters be added one-by-one against a stable test surface.
func placeholderForGen(obj element.Element) string {
	return fmt.Sprintf("// TODO Stage 3.2.2: format %T", obj)
}

// exclusiveSplitExpressionGen pulls the expression text from the split's
// SplitCondition. Skeleton: returns the raw expression string verbatim
// (or a TODO marker if no expression is present). 3.2.2 will route this
// through the gen-typed expression formatter.
func exclusiveSplitExpressionGen(split *genMf.ExclusiveSplit) string {
	if cond, ok := split.SplitCondition().(*genMf.ExpressionSplitCondition); ok && cond != nil {
		if e := strings.TrimSpace(cond.Expression()); e != "" {
			return e
		}
	}
	return "// TODO Stage 3.2.2: format split condition"
}

// isEnumSplitGen detects the case-on-enum shape: an ExpressionSplitCondition
// where at least one outgoing flow carries an EnumerationCase whose Value
// is not "true"/"false". Mendix stores boolean branches as EnumerationCase
// with literal "true"/"false" too — so the type alone is not enough; we
// also need a non-boolean literal to call this an enum split. (Mirrors
// legacy `hasEnumCaseFlows` in cmd_microflows_show_helpers.go.)
func isEnumSplitGen(split *genMf.ExclusiveSplit, flows []*genMf.SequenceFlow) bool {
	if _, ok := split.SplitCondition().(*genMf.ExpressionSplitCondition); !ok {
		return false
	}
	for _, f := range flows {
		if v, ok := enumCaseValueGen(f); ok && v != "true" && v != "false" {
			return true
		}
	}
	return false
}

// enumCaseValueGen extracts the case-value literal from a flow if it
// carries an EnumerationCase (either as the singular CaseValue or in
// the CaseValues PartList).
func enumCaseValueGen(f *genMf.SequenceFlow) (string, bool) {
	if f == nil {
		return "", false
	}
	if ec, ok := f.CaseValue().(*genMf.EnumerationCase); ok && ec != nil {
		return ec.Value(), true
	}
	for _, cv := range f.CaseValuesItems() {
		if ec, ok := cv.(*genMf.EnumerationCase); ok && ec != nil {
			return ec.Value(), true
		}
	}
	return "", false
}

// caseValueLabelGen returns a placeholder label for a CaseValue branch.
// 3.2.2 will resolve enum/literal values; the skeleton emits a stable
// TODO marker so the framing is structurally complete.
func caseValueLabelGen(_ *genMf.SequenceFlow) string {
	return "// TODO Stage 3.2.2: format case value"
}

// pickBooleanBranchesGen splits the outgoing flows of a boolean
// ExclusiveSplit into (true, false). Mendix stores the branch tag as
// an EnumerationCase with Value "true" / "false"; if a flow has no
// EnumerationCase we fall back to OriginConnectionIndex (0 = true).
func pickBooleanBranchesGen(flows []*genMf.SequenceFlow) (*genMf.SequenceFlow, *genMf.SequenceFlow) {
	var t, f *genMf.SequenceFlow
	for _, sf := range flows {
		if v, ok := enumCaseValueGen(sf); ok {
			switch v {
			case "true":
				if t == nil {
					t = sf
				}
				continue
			case "false":
				if f == nil {
					f = sf
				}
				continue
			}
		}
		// Index fallback.
		if sf.OriginConnectionIndex() == 0 && t == nil {
			t = sf
			continue
		}
		if f == nil {
			f = sf
		}
	}
	return t, f
}
