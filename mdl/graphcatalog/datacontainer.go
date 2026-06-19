package graphcatalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// compile-time interface checks
var _ DataContainerReader = (*ProjectGraph)(nil)
var _ DataFlowReader = (*ProjectGraph)(nil)

// PageDataContainerTree returns the hierarchical data container tree for a page.
func (pg *ProjectGraph) PageDataContainerTree(pageQN string) []DataContainerNode {
	g := pg.g()
	if g == nil {
		return nil
	}
	pageNodes := g.FindNodes("Page", map[string]any{"QualifiedName": pageQN})
	if len(pageNodes) == 0 {
		return nil
	}
	pageID := pageNodes[0].ID

	// Get top-level containers
	topContainers := g.Neighbors(pageID, "HAS_DATA_CONTAINER")
	return pg.buildContainerTree(topContainers, 0)
}

func (pg *ProjectGraph) buildContainerTree(containers []*mxgraph.Node, depth int) []DataContainerNode {
	var result []DataContainerNode
	for _, n := range containers {
		dc := dataContainerFromNode(n)
		if dc == nil {
			continue
		}
		// Find children
		children := pg.g().Neighbors(n.ID, "HAS_DATA_CONTAINER")
		dc.ChildWidgets = parseChildWidgets(strProp(n, "ChildWidgets"))
		dc.ContextVariables = parseContextVars(strProp(n, "ContextVariables"))
		if len(children) > 0 {
			_ = pg.buildContainerTree(children, depth+1)
		}
		result = append(result, *dc)
	}
	return result
}

// PageContextVariables returns context variables for all container levels in a page.
func (pg *ProjectGraph) PageContextVariables(pageQN string) []VariableScope {
	g := pg.g()
	if g == nil {
		return nil
	}
	pageNodes := g.FindNodes("Page", map[string]any{"QualifiedName": pageQN})
	if len(pageNodes) == 0 {
		return nil
	}

	var scopes []VariableScope
	collected := map[string]bool{}
	var walkContainerVars func(nodes []*mxgraph.Node)
	walkContainerVars = func(nodes []*mxgraph.Node) {
		for _, n := range nodes {
			if collected[string(n.ID)] {
				continue
			}
			collected[string(n.ID)] = true
			vars := parseContextVars(strProp(n, "ContextVariables"))
			depth, _ := n.Props["Depth"].(int)
			scopes = append(scopes, VariableScope{
				ContainerID: string(n.ID),
				Depth:       depth,
				Variables:   vars,
			})
			children := g.Neighbors(n.ID, "HAS_DATA_CONTAINER")
			walkContainerVars(children)
		}
	}

	topContainers := g.Neighbors(pageNodes[0].ID, "HAS_DATA_CONTAINER")
	walkContainerVars(topContainers)

	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i].Depth < scopes[j].Depth
	})
	return scopes
}

// EntityPages returns all pages that reference an entity through data containers.
func (pg *ProjectGraph) EntityPages(entityQN string) []EntityPageRef {
	g := pg.g()
	if g == nil {
		return nil
	}
	entityNodes := g.FindNodes("Entity", map[string]any{"QualifiedName": entityQN})
	if len(entityNodes) == 0 {
		return nil
	}

	// Find all DataContainers with HAS_DATASOURCE_ENTITY -> entity
	dcs := g.FindNodes("DataContainer", nil)
	var refs []EntityPageRef
	for _, dc := range dcs {
		edges := g.Edges(dc.ID, mxgraph.Outbound, "HAS_DATASOURCE_ENTITY")
		for _, e := range edges {
			// e.To is the entity QN
			if strings.Contains(string(e.To), entityQN) {
				refs = append(refs, EntityPageRef{
					Entity:         entityQN,
					Page:           strProp(dc, "PageQN"),
					DataSourceType: strProp(dc, "DataSourceType"),
					ContainerName:  strProp(dc, "WidgetName"),
				})
			}
		}
	}
	return refs
}

// EntityDataFlow returns the complete data flow for an entity.
func (pg *ProjectGraph) EntityDataFlow(entityQN string) *EntityDataFlow {
	g := pg.g()
	if g == nil || entityQN == "" {
		return nil
	}
	flow := &EntityDataFlow{Entity: entityQN}

	// Pages that use this entity
	pages := pg.EntityPages(entityQN)
	for _, p := range pages {
		roles := pg.PageAllowedRoles(p.Page)
		navRefs := findPageNavRefs(pg, p.Page)
		flow.Pages = append(flow.Pages, PageDataFlowSummary{
			Page:           p.Page,
			DataSourceType: p.DataSourceType,
			AllowedRoles:   roles,
			NavigationRefs: navRefs,
		})
	}

	// Microflows that create this entity
	mfNodes := g.FindNodes("Microflow", nil)
	for _, mf := range mfNodes {
		qn := nodeToQN(mf)
		edges := g.Edges(mf.ID, mxgraph.Outbound, "CREATES")
		for _, e := range edges {
			if strings.Contains(string(e.To), entityQN) {
				flow.Creators = append(flow.Creators, qn)
			}
		}
		edges = g.Edges(mf.ID, mxgraph.Outbound, "RETRIEVES")
		for _, e := range edges {
			if strings.Contains(string(e.To), entityQN) {
				flow.Retrievers = append(flow.Retrievers, qn)
			}
		}
	}
	return flow
}

// PageDataFlow returns a summary of a page's data flow.
func (pg *ProjectGraph) PageDataFlow(pageQN string) *PageDataFlowSummary {
	g := pg.g()
	if g == nil {
		return nil
	}
	pageNodes := g.FindNodes("Page", map[string]any{"QualifiedName": pageQN})
	if len(pageNodes) == 0 {
		return nil
	}
	roles := pg.PageAllowedRoles(pageQN)
	navRefs := findPageNavRefs(pg, pageQN)

	return &PageDataFlowSummary{
		Page:           pageQN,
		AllowedRoles:   roles,
		NavigationRefs: navRefs,
	}
}

// NavigationEntityFlow returns all pages referencing an entity through navigation.
func (pg *ProjectGraph) NavigationEntityFlow(entityQN string) []PageDataFlowSummary {
	flow := pg.EntityDataFlow(entityQN)
	if flow == nil {
		return nil
	}
	var result []PageDataFlowSummary
	for _, p := range flow.Pages {
		if len(p.NavigationRefs) > 0 {
			result = append(result, p)
		}
	}
	return result
}

// dataContainerFromNode converts a graph node to a DataContainerNode.
func dataContainerFromNode(n *mxgraph.Node) *DataContainerNode {
	if n == nil {
		return nil
	}
	hasSel, _ := n.Props["HasSelection"].(bool)
	depth, _ := n.Props["Depth"].(int)
	return &DataContainerNode{
		ID:                  string(n.ID),
		PageQN:              strProp(n, "PageQN"),
		WidgetType:          strProp(n, "WidgetType"),
		WidgetName:          strProp(n, "WidgetName"),
		DataSourceType:      strProp(n, "DataSourceType"),
		EntityPath:          strProp(n, "EntityPath"),
		TargetEntity:        strProp(n, "TargetEntity"),
		DataSourceMicroflow: strProp(n, "DataSourceMicroflow"),
		ParameterName:       strProp(n, "ParameterName"),
		ListenTargetWidget:  strProp(n, "ListenTargetWidget"),
		HasSelection:        hasSel,
		SelectionName:       strProp(n, "SelectionName"),
		Depth:               depth,
	}
}

func parseChildWidgets(jsonStr string) []ChildWidgetSummary {
	if jsonStr == "" {
		return nil
	}
	var result []ChildWidgetSummary
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil
	}
	return result
}

func parseContextVars(jsonStr string) []ContextVariable {
	if jsonStr == "" {
		return nil
	}
	var result []ContextVariable
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil
	}
	return result
}

func findPageNavRefs(pg *ProjectGraph, pageQN string) []string {
	g := pg.g()
	if g == nil {
		return nil
	}
	// Find all navigation profiles or menu items with TARGETS_PAGE -> this page
	var refs []string
	profiles := g.FindNodes("NavigationProfile", nil)
	for _, prof := range profiles {
		edges := g.Edges(prof.ID, mxgraph.Outbound, "TARGETS_PAGE")
		for _, e := range edges {
			if string(e.To) == pageQN || strings.HasSuffix(string(e.To), "."+pageQN) {
				refs = append(refs, strProp(prof, "Name"))
			}
		}
	}
	menuItems := g.FindNodes("NavigationMenuItem", nil)
	for _, mi := range menuItems {
		edges := g.Edges(mi.ID, mxgraph.Outbound, "TARGETS_PAGE")
		for _, e := range edges {
			if string(e.To) == pageQN || strings.HasSuffix(string(e.To), "."+pageQN) {
				parent := pg.findParentProfile(mi)
				if parent != "" && !contains(refs, parent) {
					refs = append(refs, parent)
				}
			}
		}
	}
	return refs
}

func (pg *ProjectGraph) findParentProfile(node *mxgraph.Node) string {
	if node == nil {
		return ""
	}
	edges := pg.g().Edges(node.ID, mxgraph.Inbound, "HAS_MENU_ITEM")
	for _, e := range edges {
		parent := pg.g().GetNode(e.From)
		if parent == nil {
			continue
		}
		if parent.Label == "NavigationProfile" {
			return strProp(parent, "Name")
		}
		return pg.findParentProfile(parent)
	}
	return ""
}

// ── TransitiveFlowReader ──────────────────────────────────────

// compile-time check
var _ TransitiveFlowReader = (*ProjectGraph)(nil)

// AllEntryPoints returns all known entry points for the application.
func (pg *ProjectGraph) AllEntryPoints() []EntryPoint {
	g := pg.g()
	if g == nil {
		return nil
	}
	var eps []EntryPoint

	// Navigation profiles
	for _, n := range g.FindNodes("NavigationProfile", nil) {
		name := strProp(n, "Name")
		if _, ok := n.Props["ParentProfile"]; ok {
			continue // skip role-based sub-profiles
		}
		eps = append(eps, EntryPoint{
			Kind:        "navigation",
			Name:        name,
			Description: fmt.Sprintf("profile %s (%s)", name, strProp(n, "Kind")),
		})
	}

	// Workflows (each workflow is an entry point via user tasks)
	for _, n := range g.FindNodes("Workflow", nil) {
		eps = append(eps, EntryPoint{
			Kind:        "workflow",
			Name:        nodeToQN(n),
			Description: fmt.Sprintf("workflow %s", nodeToQN(n)),
		})
	}

	return eps
}

// ReachablePagesFromEntry computes the transitive closure of reachable pages
// from a given entry point.
func (pg *ProjectGraph) ReachablePagesFromEntry(entryKind, entryName string) []FlowChain {
	g := pg.g()
	if g == nil {
		return nil
	}

	var startNodes []*mxgraph.Node
	switch entryKind {
	case "navigation":
		startNodes = g.FindNodes("NavigationProfile", map[string]any{"Name": entryName})
	case "workflow":
		startNodes = g.FindNodes("Workflow", map[string]any{"QualifiedName": entryName})
		if len(startNodes) == 0 {
			startNodes = g.FindNodes("Workflow", map[string]any{"Name": entryName})
		}
	case "microflow":
		startNodes = g.FindNodes("Microflow", map[string]any{"QualifiedName": entryName})
		if len(startNodes) == 0 {
			startNodes = g.FindNodes("Nanoflow", map[string]any{"QualifiedName": entryName})
		}
	}

	if len(startNodes) == 0 {
		return nil
	}

	return pg.bfsChains(startNodes[0], entryKind, entryName, 0)
}

// AllReachablePages computes the union of all reachable pages from all entry points.
func (pg *ProjectGraph) AllReachablePages() map[string][]string {
	g := pg.g()
	if g == nil {
		return nil
	}
	result := map[string][]string{}
	allPages := map[string]bool{}

	for _, ep := range pg.AllEntryPoints() {
		chains := pg.ReachablePagesFromEntry(ep.Kind, ep.Name)
		for _, chain := range chains {
			p := chain.TerminalPage
			if !allPages[p] {
				allPages[p] = true
			}
			result[p] = append(result[p], ep.Name)
		}
	}
	return result
}

// EntryPointReachability returns a deduplicated flow graph for an entry point.
func (pg *ProjectGraph) EntryPointReachability(entryKind, entryName string, maxDepth int) (*FlowGraph, error) {
	g := pg.g()
	if g == nil {
		return nil, nil
	}

	var startNodes []*mxgraph.Node
	switch entryKind {
	case "navigation":
		startNodes = g.FindNodes("NavigationProfile", map[string]any{"Name": entryName})
	case "workflow":
		startNodes = g.FindNodes("Workflow", map[string]any{"QualifiedName": entryName})
		if len(startNodes) == 0 {
			startNodes = g.FindNodes("Workflow", map[string]any{"Name": entryName})
		}
	case "microflow":
		startNodes = g.FindNodes("Microflow", map[string]any{"QualifiedName": entryName})
		if len(startNodes) == 0 {
			startNodes = g.FindNodes("Nanoflow", map[string]any{"QualifiedName": entryName})
		}
	}
	if len(startNodes) == 0 {
		return nil, fmt.Errorf("entry point not found: %s:%s", entryKind, entryName)
	}

	fg := &FlowGraph{}
	seenNodes := map[string]bool{}
	seenEdges := map[string]bool{}

	type bfsItem struct {
		node  *mxgraph.Node
		depth int
	}
	queue := []bfsItem{{startNodes[0], 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if maxDepth > 0 && cur.depth >= maxDepth {
			continue
		}
		if seenNodes[string(cur.node.ID)] {
			continue
		}
		seenNodes[string(cur.node.ID)] = true

		nodeID := string(cur.node.ID)
		nodeQN := nodeToQN(cur.node)

		fg.Nodes = append(fg.Nodes, FlowGraphNode{
			ID:    nodeID,
			Label: string(cur.node.Label),
			QN:    nodeQN,
		})

		// Which edge types to follow depends on the node label
		outRelTypes := pg.outboundRelTypes(cur.node.Label)
		for _, rel := range outRelTypes {
			edges := g.Edges(cur.node.ID, mxgraph.Outbound, rel)
			for _, e := range edges {
				targetQN := string(e.To)
				if targetQN == "" {
					continue
				}
				edgeKey := fmt.Sprintf("%s--%s-->%s", nodeID, rel, targetQN)
				if seenEdges[edgeKey] {
					continue
				}
				seenEdges[edgeKey] = true
				fg.Edges = append(fg.Edges, FlowGraphEdge{
					From: nodeQN,
					To:   targetQN,
					Type: string(rel),
				})

				// Find the target node
				targetLabel := pg.labelForRef(rel)
				targetNodes := g.FindNodes(targetLabel, map[string]any{"QualifiedName": targetQN})
				if len(targetNodes) == 0 {
					// Try with simple name
					targetNodes = g.FindNodes(targetLabel, map[string]any{"Name": targetQN})
				}
				// For menu items and profiles, try node ID
				if len(targetNodes) == 0 {
					if n := g.GetNode(mxgraph.NodeID(targetQN)); n != nil {
						targetNodes = append(targetNodes, n)
					}
				}
				for _, tn := range targetNodes {
					if !seenNodes[string(tn.ID)] {
						queue = append(queue, bfsItem{tn, cur.depth + 1})
					}
				}
			}
		}
	}
	return fg, nil
}

// bfsChains does BFS from a start node and returns all distinct page-reachable chains.
func (pg *ProjectGraph) bfsChains(start *mxgraph.Node, entryKind, entryName string, maxDepth int) []FlowChain {
	g := pg.g()
	if g == nil {
		return nil
	}

	type state struct {
		qn    string
		node  *mxgraph.Node
		depth int
		steps []FlowStep
	}
	var chains []FlowChain
	visited := map[string]bool{}

	queue := []state{{nodeToQN(start), start, 0, nil}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if maxDepth > 0 && cur.depth >= maxDepth {
			continue
		}
		if visited[cur.qn] {
			continue
		}
		visited[cur.qn] = true

		if cur.node.Label == "Page" {
			chains = append(chains, FlowChain{
				EntryPoint:    fmt.Sprintf("%s:%s", entryKind, entryName),
				Steps:         cur.steps,
				TerminalPage:  cur.qn,
			})
		}

		outRels := pg.outboundRelTypes(cur.node.Label)
		for _, rel := range outRels {
			edges := g.Edges(cur.node.ID, mxgraph.Outbound, rel)
			for _, e := range edges {
				targetQN := string(e.To)
				if targetQN == "" || visited[targetQN] {
					continue
				}
				targetLabel := pg.labelForRef(rel)
				step := FlowStep{
					NodeType: string(targetLabel),
					NodeName: targetQN,
					EdgeType: string(rel),
					Depth:    cur.depth + 1,
				}
				newSteps := append(append([]FlowStep{}, cur.steps...), step)

				targetNodes := g.FindNodes(targetLabel, map[string]any{"QualifiedName": targetQN})
				if len(targetNodes) == 0 {
					targetNodes = g.FindNodes(targetLabel, map[string]any{"Name": targetQN})
				}
				if len(targetNodes) == 0 {
					if n := g.GetNode(mxgraph.NodeID(targetQN)); n != nil {
						targetNodes = append(targetNodes, n)
					}
				}
				for _, tn := range targetNodes {
					queue = append(queue, state{targetQN, tn, cur.depth + 1, newSteps})
				}
			}
		}
	}
	return chains
}

// outboundRelTypes returns which edge types to follow from a given node label
// in a transitive data-flow traversal.
func (pg *ProjectGraph) outboundRelTypes(label mxgraph.Label) []mxgraph.RelType {
	switch label {
	case "NavigationProfile", "NavigationMenuItem":
		return []mxgraph.RelType{"TARGETS_PAGE", "TARGETS_MICROFLOW", "HAS_MENU_ITEM"}
	case "Microflow", "Nanoflow":
		return []mxgraph.RelType{"SHOWS_PAGE", "CALLS", "CREATES", "RETRIEVES"}
	case "Page", "Layout", "Snippet":
		return []mxgraph.RelType{"CALLS_MICROFLOW", "HAS_DATASOURCE_MICROFLOW", "HAS_DATA_CONTAINER"}
	case "DataContainer":
		return []mxgraph.RelType{"HAS_DATASOURCE_MICROFLOW", "HAS_DATASOURCE_ENTITY"}
	case "Workflow":
		return []mxgraph.RelType{"CALLS"}
	}
	return nil
}

// labelForRef maps an edge type to the probable target node label.
func (pg *ProjectGraph) labelForRef(rel mxgraph.RelType) mxgraph.Label {
	switch rel {
	case "TARGETS_PAGE", "SHOWS_PAGE":
		return "Page"
	case "TARGETS_MICROFLOW", "CALLS_MICROFLOW", "HAS_DATASOURCE_MICROFLOW", "CALLS":
		return "Microflow"
	case "CREATES", "RETRIEVES", "HAS_DATASOURCE_ENTITY":
		return "Entity"
	case "HAS_MENU_ITEM":
		return "NavigationMenuItem"
	}
	return ""
}
