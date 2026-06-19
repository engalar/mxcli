package graphcatalog

import (
	"sort"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// compile-time interface check
var _ NavigationReader = (*ProjectGraph)(nil)

// NavigationProfiles returns all navigation profile nodes.
func (pg *ProjectGraph) NavigationProfiles() []NavigationProfileNode {
	g := pg.g()
	if g == nil {
		return nil
	}
	nodes := g.FindNodes("NavigationProfile", nil)
	seen := map[string]bool{} // dedup by name (role-based profiles share the parent)
	var result []NavigationProfileNode
	for _, n := range nodes {
		profile := navProfileFromNode(n)
		if seen[profile.Name] {
			continue
		}
		seen[profile.Name] = true

		// Count menu items
		profile.MenuItemCount = len(g.Neighbors(n.ID, "HAS_MENU_ITEM"))

		// Home page target
		homePages := g.Neighbors(n.ID, "TARGETS_PAGE")
		if len(homePages) > 0 {
			profile.HomePage = nodeToQN(homePages[0])
		}
		if profile.HomePage == "" {
			homeMFs := g.Neighbors(n.ID, "TARGETS_MICROFLOW")
			if len(homeMFs) > 0 {
				profile.HomePage = "MF:" + nodeToQN(homeMFs[0])
			}
		}
		// Login page
		loginPages := g.Neighbors(n.ID, "HAS_LOGIN_PAGE")
		if len(loginPages) > 0 {
			profile.LoginPage = nodeToQN(loginPages[0])
		}
		// Not-found page
		nfPages := g.Neighbors(n.ID, "HAS_NOT_FOUND_PAGE")
		if len(nfPages) > 0 {
			profile.NotFoundPage = nodeToQN(nfPages[0])
		}

		result = append(result, profile)
	}
	return result
}

// NavigationMenuTree returns the menu tree for a profile.
func (pg *ProjectGraph) NavigationMenuTree(profileName string) *NavigationTree {
	g := pg.g()
	if g == nil {
		return nil
	}
	profileNodes := g.FindNodes("NavigationProfile", map[string]any{"Name": profileName})
	if len(profileNodes) == 0 {
		return nil
	}

	profile := navProfileFromNode(profileNodes[0])
	tree := &NavigationTree{Profile: profile}

	// Home page
	if homeNode := pg.NavigationHomePage(profileName); homeNode != nil {
		tree.HomePage = homeNode
	}

	// Menu items
	for _, mi := range g.Neighbors(profileNodes[0].ID, "HAS_MENU_ITEM") {
		if subTree := pg.buildMenuItemTree(mi); subTree != nil {
			tree.MenuItems = append(tree.MenuItems, *subTree)
		}
	}

	return tree
}

func (pg *ProjectGraph) buildMenuItemTree(node *mxgraph.Node) *NavigationMenuItemTree {
	if node == nil {
		return nil
	}
	item := navMenuItemFromNode(node)
	tree := &NavigationMenuItemTree{Item: item}
	for _, child := range pg.g().Neighbors(node.ID, "HAS_MENU_ITEM") {
		if sub := pg.buildMenuItemTree(child); sub != nil {
			tree.Children = append(tree.Children, *sub)
		}
	}
	return tree
}

// NavigationHomePage returns the home page config for a profile.
func (pg *ProjectGraph) NavigationHomePage(profileName string) *NavigationHomePageNode {
	g := pg.g()
	if g == nil {
		return nil
	}
	profileNodes := g.FindNodes("NavigationProfile", map[string]any{"Name": profileName})
	if len(profileNodes) == 0 {
		return nil
	}
	n := profileNodes[0]
	result := &NavigationHomePageNode{Profile: profileName}

	pages := g.Neighbors(n.ID, "TARGETS_PAGE")
	if len(pages) > 0 {
		result.Page = nodeToQN(pages[0])
	}
	mfs := g.Neighbors(n.ID, "TARGETS_MICROFLOW")
	if len(mfs) > 0 {
		result.Microflow = nodeToQN(mfs[0])
	}
	return result
}

// PagesReferencedByNavigation returns all pages that have at least one inbound
// TARGETS_PAGE or SHOWS_PAGE or CALLS_MICROFLOW edge.
func (pg *ProjectGraph) PagesReferencedByNavigation() []string {
	g := pg.g()
	if g == nil {
		return nil
	}
	pages := g.FindNodes("Page", nil)
	edgeTypes := []mxgraph.RelType{"TARGETS_PAGE", "SHOWS_PAGE", "CALLS_MICROFLOW"}
	seen := map[string]bool{}

	// Check by element ID
	for _, p := range pages {
		edges := g.Edges(p.ID, mxgraph.Inbound, edgeTypes...)
		if len(edges) > 0 {
			seen[nodeToQN(p)] = true
		}
	}
	// Also check by QN (for ref edges that store QN as To)
	for _, p := range pages {
		qn := nodeToQN(p)
		if seen[qn] {
			continue
		}
		edges := g.Edges(mxgraph.NodeID(qn), mxgraph.Inbound, edgeTypes...)
		if len(edges) > 0 {
			seen[qn] = true
		}
	}

	refs := make([]string, 0, len(seen))
	for qn := range seen {
		refs = append(refs, qn)
	}
	sort.Strings(refs)
	return refs
}

// OrphanPages returns pages with no inbound navigation/microflow/page-action edges.
func (pg *ProjectGraph) OrphanPages() []string {
	g := pg.g()
	if g == nil {
		return nil
	}
	pages := g.FindNodes("Page", nil)
	edgeTypes := []mxgraph.RelType{"TARGETS_PAGE", "SHOWS_PAGE", "CALLS_MICROFLOW"}
	var orphans []string
	for _, p := range pages {
		qn := nodeToQN(p)
		targetEdges := g.Edges(p.ID, mxgraph.Inbound, edgeTypes...)
		qnEdges := g.Edges(mxgraph.NodeID(qn), mxgraph.Inbound, edgeTypes...)
		if len(targetEdges) == 0 && len(qnEdges) == 0 {
			orphans = append(orphans, qn)
		}
	}
	sort.Strings(orphans)
	return orphans
}

// navProfileFromNode converts a graph node to a NavigationProfileNode.
func navProfileFromNode(n *mxgraph.Node) NavigationProfileNode {
	isNative, _ := n.Props["IsNative"].(bool)
	return NavigationProfileNode{
		ID:            string(n.ID),
		Name:          strProp(n, "Name"),
		Kind:          strProp(n, "Kind"),
		IsNative:      isNative,
		QualifiedName: nodeToQN(n),
	}
}

// navMenuItemFromNode converts a graph node to a NavigationMenuItemNode.
func navMenuItemFromNode(n *mxgraph.Node) NavigationMenuItemNode {
	depth, _ := n.Props["Depth"].(int)
	return NavigationMenuItemNode{
		ID:        string(n.ID),
		Caption:   strProp(n, "Caption"),
		Page:      strProp(n, "Page"),
		Microflow: strProp(n, "Microflow"),
		Depth:     depth,
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
