package graphcatalog

import (
	"strings"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// ProjectGraph 实现 LintReader 和 TraversalReader，封装 mxgraph.IndexManager。
//
// 边的方向约定（来自 mpr 适配器）：containment 边（HAS_ENTITY/HAS_ATTRIBUTE 等）
// 的 To 是子节点的 NodeID；引用边（CALLS/CREATES/RETRIEVES/SHOWS_PAGE）的 To 是
// 目标的 QualifiedName 字符串，而非目标节点的 NodeID。因此遍历引用边时，按 QN
// 直接作为 NodeID 键查 inbound 索引即可命中（无需先解析目标节点）。
type ProjectGraph struct {
	mgr *mxgraph.IndexManager
}

// 编译期接口检查
var _ LintReader = (*ProjectGraph)(nil)
var _ TraversalReader = (*ProjectGraph)(nil)
var _ ThemeReader = (*ProjectGraph)(nil)
var _ StylingReader = (*ProjectGraph)(nil)

// NewProjectGraph 创建 ProjectGraph，接管已构建的 IndexManager。
func NewProjectGraph(mgr *mxgraph.IndexManager) *ProjectGraph {
	return &ProjectGraph{mgr: mgr}
}

// g 是内部便捷访问器。
func (pg *ProjectGraph) g() *mxgraph.Graph {
	if pg == nil || pg.mgr == nil {
		return nil
	}
	return pg.mgr.Query()
}

// MxGraph returns the underlying mxgraph.Graph for direct node queries.
func (pg *ProjectGraph) MxGraph() *mxgraph.Graph {
	if pg == nil || pg.mgr == nil {
		return nil
	}
	return pg.mgr.Query()
}

// nodeToQN 从节点 Props 提取 QualifiedName，回退到 Name，再回退到节点 ID。
func nodeToQN(n *mxgraph.Node) string {
	if qn, ok := n.Props["QualifiedName"].(string); ok && qn != "" {
		return qn
	}
	if name, ok := n.Props["Name"].(string); ok && name != "" {
		return name
	}
	return string(n.ID)
}

// ── DomainReader ──────────────────────────────────────────────

func (pg *ProjectGraph) Entities(module string) []EntityNode {
	nodes := pg.g().FindNodes("Entity", moduleFilter(module))
	result := make([]EntityNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, entityFromNode(n))
	}
	return result
}

func (pg *ProjectGraph) Entity(qualifiedName string) *EntityNode {
	nodes := pg.g().FindNodes("Entity", map[string]any{"QualifiedName": qualifiedName})
	if len(nodes) == 0 {
		return nil
	}
	e := entityFromNode(nodes[0])
	return &e
}

func (pg *ProjectGraph) Attributes(entityQN string) []AttributeNode {
	entityNodes := pg.g().FindNodes("Entity", map[string]any{"QualifiedName": entityQN})
	if len(entityNodes) == 0 {
		return nil
	}
	attrNodes := pg.g().Neighbors(entityNodes[0].ID, "HAS_ATTRIBUTE")
	result := make([]AttributeNode, 0, len(attrNodes))
	for _, n := range attrNodes {
		result = append(result, AttributeNode{
			ID:       string(n.ID),
			Name:     strProp(n, "Name"),
			DataType: strProp(n, "DataType"),
			Entity:   entityQN,
		})
	}
	return result
}

func (pg *ProjectGraph) Associations(module string) []AssociationNode {
	nodes := pg.g().FindNodes("Association", moduleFilter(module))
	result := make([]AssociationNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, AssociationNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
			Owner:         strProp(n, "Owner"),
		})
	}
	return result
}

// ── BehaviorReader ────────────────────────────────────────────

func (pg *ProjectGraph) Microflows(module string) []MicroflowNode {
	result := pg.microflowsByLabel("Microflow", module)
	result = append(result, pg.microflowsByLabel("Nanoflow", module)...)
	return result
}

func (pg *ProjectGraph) microflowsByLabel(label mxgraph.Label, module string) []MicroflowNode {
	nodes := pg.g().FindNodes(label, moduleFilter(module))
	result := make([]MicroflowNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, MicroflowNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
			IsNanoflow:    label == "Nanoflow",
			ReturnType:    strProp(n, "ReturnType"),
		})
	}
	return result
}

func (pg *ProjectGraph) Microflow(qn string) *MicroflowNode {
	for _, label := range []mxgraph.Label{"Microflow", "Nanoflow"} {
		nodes := pg.g().FindNodes(label, map[string]any{"QualifiedName": qn})
		if len(nodes) > 0 {
			mf := MicroflowNode{
				ID:            string(nodes[0].ID),
				Name:          strProp(nodes[0], "Name"),
				QualifiedName: nodeToQN(nodes[0]),
				Module:        strProp(nodes[0], "Module"),
				IsNanoflow:    label == "Nanoflow",
				ReturnType:    strProp(nodes[0], "ReturnType"),
			}
			return &mf
		}
	}
	return nil
}

func (pg *ProjectGraph) Pages(module string) []PageNode {
	nodes := pg.g().FindNodes("Page", moduleFilter(module))
	result := make([]PageNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, PageNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
		})
	}
	return result
}

func (pg *ProjectGraph) Snippets(module string) []SnippetNode {
	nodes := pg.g().FindNodes("Snippet", moduleFilter(module))
	result := make([]SnippetNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, SnippetNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
		})
	}
	return result
}

func (pg *ProjectGraph) Enumerations(module string) []EnumerationNode {
	nodes := pg.g().FindNodes("Enumeration", moduleFilter(module))
	result := make([]EnumerationNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, EnumerationNode{
			ID:            string(n.ID),
			Name:          strProp(n, "Name"),
			QualifiedName: nodeToQN(n),
			Module:        strProp(n, "Module"),
		})
	}
	return result
}

// ── SecurityReader ────────────────────────────────────────────
//
// 底层 graph 用 label "ModuleRole" / "UserRole" 存储安全角色节点（见 mpr
// SecurityAdapter）。Permissions() 映射 ModuleRole，RoleMappings() 映射 UserRole。

func (pg *ProjectGraph) Permissions() []PermissionNode {
	nodes := pg.g().FindNodes("ModuleRole", nil)
	result := make([]PermissionNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, PermissionNode{
			ID:           string(n.ID),
			EntityName:   strProp(n, "EntityName"),
			ModuleRole:   strProp(n, "Name"),
			AccessRights: strProp(n, "AccessRights"),
		})
	}
	return result
}

func (pg *ProjectGraph) RoleMappings() []RoleMappingNode {
	nodes := pg.g().FindNodes("UserRole", nil)
	result := make([]RoleMappingNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, RoleMappingNode{
			ID:         string(n.ID),
			UserRole:   strProp(n, "Name"),
			ModuleRole: strProp(n, "ModuleRole"),
		})
	}
	return result
}

// ── ExtensionReader ───────────────────────────────────────────

func (pg *ProjectGraph) Widgets(pageQN string) []WidgetNode {
	pageNodes := pg.g().FindNodes("Page", map[string]any{"QualifiedName": pageQN})
	if len(pageNodes) == 0 {
		return nil
	}
	widgetNodes := pg.g().Neighbors(pageNodes[0].ID, "HAS_WIDGET")
	result := make([]WidgetNode, 0, len(widgetNodes))
	for _, n := range widgetNodes {
		result = append(result, WidgetNode{
			ID:         string(n.ID),
			WidgetType: strProp(n, "WidgetType"),
			Name:       strProp(n, "Name"),
			PageID:     string(pageNodes[0].ID),
		})
	}
	return result
}

func (pg *ProjectGraph) DatabaseConnections() []DatabaseConnectionNode {
	nodes := pg.g().FindNodes("DatabaseConnection", nil)
	result := make([]DatabaseConnectionNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, DatabaseConnectionNode{
			ID:           string(n.ID),
			Name:         strProp(n, "Name"),
			DatabaseType: strProp(n, "DatabaseType"),
		})
	}
	return result
}

// ── TraversalReader ───────────────────────────────────────────
//
// 引用边的 To 存的是目标 QualifiedName 字符串。因此查 target 的入边时，直接把
// QN 当作 NodeID 键传给 inbound 索引即可命中；查 source 的出边时，从已解析的
// source 节点 ID 出发，边的 To 已经是目标 QN，无需再次解析。

func (pg *ProjectGraph) Callers(qualifiedName string, transitive bool) []CallEdge {
	if !transitive {
		edges := pg.g().Edges(mxgraph.NodeID(qualifiedName), mxgraph.Inbound, "CALLS")
		result := make([]CallEdge, 0, len(edges))
		for _, e := range edges {
			caller := pg.g().GetNode(e.From)
			if caller == nil {
				continue
			}
			result = append(result, CallEdge{
				Caller: nodeToQN(caller),
				Callee: qualifiedName,
				Depth:  1,
			})
		}
		return result
	}
	return pg.bfsCallers(qualifiedName)
}

// bfsCallers 反向遍历 CALLS 边。队列中存 QualifiedName（因为入边以 QN 为键），
// 已访问集合也按 QN 去重。
func (pg *ProjectGraph) bfsCallers(targetQN string) []CallEdge {
	visited := map[string]bool{targetQN: true}
	var result []CallEdge
	type item struct {
		qn    string
		depth int
	}
	queue := []item{{targetQN, 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		edges := pg.g().Edges(mxgraph.NodeID(cur.qn), mxgraph.Inbound, "CALLS")
		for _, e := range edges {
			caller := pg.g().GetNode(e.From)
			if caller == nil {
				continue
			}
			callerQN := nodeToQN(caller)
			if visited[callerQN] {
				continue
			}
			visited[callerQN] = true
			result = append(result, CallEdge{
				Caller: callerQN,
				Callee: cur.qn,
				Depth:  cur.depth + 1,
			})
			queue = append(queue, item{callerQN, cur.depth + 1})
		}
	}
	return result
}

func (pg *ProjectGraph) Callees(qualifiedName string, transitive bool) []CallEdge {
	source := pg.findNodeByQN(qualifiedName)
	if source == nil {
		return nil
	}
	if !transitive {
		edges := pg.g().Edges(source.ID, mxgraph.Outbound, "CALLS")
		result := make([]CallEdge, 0, len(edges))
		for _, e := range edges {
			result = append(result, CallEdge{
				Caller: qualifiedName,
				Callee: string(e.To), // To 已是 callee 的 QualifiedName
				Depth:  1,
			})
		}
		return result
	}
	return pg.bfsCallees(qualifiedName)
}

// bfsCallees 正向遍历 CALLS 边。出边的 To 是 callee QN；下一层需先把 QN 解析回
// 节点以取得其出边。
func (pg *ProjectGraph) bfsCallees(sourceQN string) []CallEdge {
	visited := map[string]bool{sourceQN: true}
	var result []CallEdge
	type item struct {
		qn    string
		depth int
	}
	queue := []item{{sourceQN, 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		node := pg.findNodeByQN(cur.qn)
		if node == nil {
			continue
		}
		edges := pg.g().Edges(node.ID, mxgraph.Outbound, "CALLS")
		for _, e := range edges {
			calleeQN := string(e.To)
			if visited[calleeQN] {
				continue
			}
			visited[calleeQN] = true
			result = append(result, CallEdge{
				Caller: cur.qn,
				Callee: calleeQN,
				Depth:  cur.depth + 1,
			})
			queue = append(queue, item{calleeQN, cur.depth + 1})
		}
	}
	return result
}

func (pg *ProjectGraph) Impact(qualifiedName string) []RefEdge {
	// Impact = 所有指向 target 的引用边（任意 RelType）。入边以目标 QN 为键。
	edges := pg.g().Edges(mxgraph.NodeID(qualifiedName), mxgraph.Inbound)
	result := make([]RefEdge, 0, len(edges))
	for _, e := range edges {
		src := pg.g().GetNode(e.From)
		if src == nil {
			continue
		}
		result = append(result, RefEdge{
			Source:  nodeToQN(src),
			Target:  qualifiedName,
			RefKind: string(e.Type),
		})
	}
	return result
}

func (pg *ProjectGraph) References(qualifiedName string) []RefEdge {
	// References = source 发出的所有引用边。出边的 To 已是目标 QN。
	source := pg.findNodeByQN(qualifiedName)
	if source == nil {
		return nil
	}
	edges := pg.g().Edges(source.ID, mxgraph.Outbound)
	result := make([]RefEdge, 0, len(edges))
	for _, e := range edges {
		result = append(result, RefEdge{
			Source:  qualifiedName,
			Target:  string(e.To),
			RefKind: string(e.Type),
		})
	}
	return result
}

// ── 内部辅助 ─────────────────────────────────────────────────

// findNodeByQN 按 QualifiedName 属性查找节点（遍历常见 label）。
func (pg *ProjectGraph) findNodeByQN(qn string) *mxgraph.Node {
	filter := map[string]any{"QualifiedName": qn}
	for _, label := range []mxgraph.Label{"Microflow", "Nanoflow", "Entity", "Page", "Snippet", "Enumeration", "Association"} {
		nodes := pg.g().FindNodes(label, filter)
		if len(nodes) > 0 {
			return nodes[0]
		}
	}
	return nil
}

func moduleFilter(module string) map[string]any {
	if module == "" {
		return nil
	}
	return map[string]any{"Module": module}
}

func strProp(n *mxgraph.Node, key string) string {
	if v, ok := n.Props[key].(string); ok {
		return v
	}
	return ""
}

func entityFromNode(n *mxgraph.Node) EntityNode {
	isExt, _ := n.Props["IsExternal"].(bool)
	return EntityNode{
		ID:            string(n.ID),
		Name:          strProp(n, "Name"),
		QualifiedName: nodeToQN(n),
		Module:        strProp(n, "Module"),
		IsExternal:    isExt,
	}
}

// ── ThemeReader ──────────────────────────────────────────────

func (pg *ProjectGraph) ThemeVariables(module string, filter ThemeVarFilter) []ThemeVariableNode {
	var nodes []*mxgraph.Node
	if filter.Source != "" {
		nodes = pg.g().FindNodes("ThemeVariable", map[string]any{"Source": filter.Source})
	} else {
		nodes = pg.g().FindNodes("ThemeVariable", nil)
	}
	var result []ThemeVariableNode
	for _, n := range nodes {
		if filter.ActiveOnly {
			if active := boolProp(n, "IsActive"); !active {
				continue
			}
		}
		if filter.Like != "" {
			name := strProp(n, "Name")
			if !strings.Contains(name, filter.Like) {
				continue
			}
		}
		if module != "" {
			m := strProp(n, "Module")
			if m != module {
				continue
			}
		}
		result = append(result, toThemeVarNode(n))
	}
	return result
}

func (pg *ProjectGraph) ThemeVariable(name string) *ThemeVariableNode {
	nodes := pg.g().FindNodes("ThemeVariable", map[string]any{"Name": name})
	if len(nodes) == 0 {
		return nil
	}
	r := toThemeVarNode(nodes[0])
	return &r
}

func (pg *ProjectGraph) OverriddenVariables() []ThemeVariableNode {
	nodes := pg.g().FindNodes("ThemeVariable", nil)
	var result []ThemeVariableNode
	for _, n := range nodes {
		src := strProp(n, "Source")
		if src == "atlas-core-default" || src == "" {
			continue
		}
		if active := boolProp(n, "IsActive"); !active {
			continue
		}
		result = append(result, toThemeVarNode(n))
	}
	return result
}

func toThemeVarNode(n *mxgraph.Node) ThemeVariableNode {
	lineNum, _ := n.Props["LineNumber"].(int)
	return ThemeVariableNode{
		Name:         strProp(n, "Name"),
		Value:        strProp(n, "Value"),
		VariableType: strProp(n, "VariableType"),
		IsDefault:    boolProp(n, "IsDefault"),
		IsActive:     boolProp(n, "IsActive"),
		Source:       strProp(n, "Source"),
		Module:       strProp(n, "Module"),
		Category:     strProp(n, "Category"),
		FilePath:     strProp(n, "FilePath"),
		LineNumber:   lineNum,
	}
}

func boolProp(n *mxgraph.Node, key string) bool {
	if v, ok := n.Props[key].(bool); ok {
		return v
	}
	return false
}

// ── StylingReader ────────────────────────────────────────────

func (pg *ProjectGraph) WidgetInstances(pageQN string) []WidgetInstanceNode {
	nodes := pg.g().FindNodes("WidgetInstance", nil)
	var result []WidgetInstanceNode
	for _, n := range nodes {
		if pageQN != "" {
			qn := strProp(n, "QualifiedName")
			if !strings.HasPrefix(qn, pageQN) {
				continue
			}
		}
		result = append(result, toWidgetInstanceNode(n))
	}
	return result
}

func (pg *ProjectGraph) DesignProperties(widgetType string) []DesignPropertyNode {
	filter := map[string]any{}
	if widgetType != "" {
		filter["WidgetType"] = widgetType
	}
	nodes := pg.g().FindNodes("DesignProperty", filter)
	result := make([]DesignPropertyNode, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, toDesignPropertyNode(n))
	}
	return result
}

func (pg *ProjectGraph) DesignProperty(widgetType, name string) *DesignPropertyNode {
	nodes := pg.g().FindNodes("DesignProperty", map[string]any{
		"WidgetType": widgetType,
		"Name":       name,
	})
	if len(nodes) == 0 {
		return nil
	}
	r := toDesignPropertyNode(nodes[0])
	return &r
}

func toWidgetInstanceNode(n *mxgraph.Node) WidgetInstanceNode {
	dps, _ := n.Props["DesignProperties"].(map[string]string)
	if dps == nil {
		dps = make(map[string]string)
	}
	return WidgetInstanceNode{
		ID:               string(n.ID),
		Name:             strProp(n, "Name"),
		WidgetType:       strProp(n, "WidgetType"),
		Class:            strProp(n, "Class"),
		Style:            strProp(n, "Style"),
		DesignProperties: dps,
		Page:             strProp(n, "Page"),
	}
}

func toDesignPropertyNode(n *mxgraph.Node) DesignPropertyNode {
	refVars, _ := n.Props["ReferencedVars"].([]string)
	opts, _ := n.Props["Options"].([]string)
	if opts == nil {
		opts = []string{}
	}
	if refVars == nil {
		refVars = []string{}
	}
	return DesignPropertyNode{
		WidgetType:     strProp(n, "WidgetType"),
		Name:           strProp(n, "Name"),
		Type:           strProp(n, "Type"),
		Category:       strProp(n, "Category"),
		Description:    strProp(n, "Description"),
		Options:        opts,
		ReferencedVars: refVars,
		SourceModule:   strProp(n, "SourceModule"),
	}
}
