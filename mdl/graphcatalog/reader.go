package graphcatalog

// DomainReader 读取域模型对象。linter 域模型规则使用。
type DomainReader interface {
	Entities(module string) []EntityNode
	Entity(qualifiedName string) *EntityNode
	Attributes(entityQualifiedName string) []AttributeNode
	Associations(module string) []AssociationNode
}

// BehaviorReader 读取行为对象（微流、页面、代码片段）。linter 行为规则使用。
type BehaviorReader interface {
	Microflows(module string) []MicroflowNode
	Microflow(qualifiedName string) *MicroflowNode
	Pages(module string) []PageNode
	Snippets(module string) []SnippetNode
	Enumerations(module string) []EnumerationNode
}

// SecurityReader 读取安全规则。linter 安全规则使用。
//
// 注意：底层 graph 用 label "ModuleRole" / "UserRole" 存储安全角色节点。
// Permissions() 返回 ModuleRole 节点，RoleMappings() 返回 UserRole 节点。
type SecurityReader interface {
	Permissions() []PermissionNode
	RoleMappings() []RoleMappingNode
}

// ExtensionReader 读取扩展对象（小部件、数据库连接）。linter 扩展规则使用。
type ExtensionReader interface {
	Widgets(pageQualifiedName string) []WidgetNode
	DatabaseConnections() []DatabaseConnectionNode
}

// TraversalReader 执行图遍历。executor search 命令使用。
type TraversalReader interface {
	Callers(qualifiedName string, transitive bool) []CallEdge
	Callees(qualifiedName string, transitive bool) []CallEdge
	Impact(qualifiedName string) []RefEdge
	References(qualifiedName string) []RefEdge
}

// EntityAccessReader 读取实体访问规则。安全 lint 规则和 access gap 分析使用。
type EntityAccessReader interface {
	EntityAccessRules(entityQN string) []AccessRuleNode
	EntityAccessRulesForRole(moduleRoleQN string) []AccessRuleNode
	EntitiesWithMissingAccessRules(module string) []EntityNode
}

// PageRefReader 读取页面 widget 树中的实体/微流引用。
type PageRefReader interface {
	PageEntityRefs(pageQN string) []string
	PageMFRefs(pageQN string) []string
}

// DocumentGrantReader 读取页面/微流的授权信息。
type DocumentGrantReader interface {
	PageAllowedRoles(pageQN string) []string
	MFAllowedRoles(mfQN string) []string
	ApplyEntityAccess(mfQN string) bool
}

// LintReader 是 linter 所需的完整接口（聚合 4 个子接口）。
type LintReader interface {
	DomainReader
	BehaviorReader
	SecurityReader
	ExtensionReader
}

// ThemeReader 读取主题变量。SHOW THEME 命令和 AI 工具使用。
type ThemeReader interface {
	ThemeVariables(module string, filter ThemeVarFilter) []ThemeVariableNode
	ThemeVariable(name string) *ThemeVariableNode
	OverriddenVariables() []ThemeVariableNode
}

// StylingReader 读取 widget 样式信息。
type StylingReader interface {
	WidgetInstances(pageQN string) []WidgetInstanceNode
	DesignProperties(widgetType string) []DesignPropertyNode
	DesignProperty(widgetType, name string) *DesignPropertyNode
}

// NavigationReader 读取导航树结构。
type NavigationReader interface {
	NavigationProfiles() []NavigationProfileNode
	NavigationMenuTree(profileName string) *NavigationTree
	NavigationHomePage(profileName string) *NavigationHomePageNode
	PagesReferencedByNavigation() []string
	OrphanPages() []string
}

// DataContainerReader 读取页面数据容器层次。
type DataContainerReader interface {
	PageDataContainerTree(pageQN string) []DataContainerNode
	PageContextVariables(pageQN string) []VariableScope
	EntityPages(entityQN string) []EntityPageRef
}

// DataFlowReader 读取端到端数据流。
type DataFlowReader interface {
	EntityDataFlow(entityQN string) *EntityDataFlow
	PageDataFlow(pageQN string) *PageDataFlowSummary
	NavigationEntityFlow(entityQN string) []PageDataFlowSummary
}

// FlowStep 描述传递式数据流中的一步。
type FlowStep struct {
	NodeType  string // "NavProfile" | "MenuItem" | "Page" | "Microflow" | "Nanoflow" | "Workflow" | "DataContainer"
	NodeName  string // QN or caption
	EdgeType  string // what led here: "TARGETS_PAGE" | "TARGETS_MICROFLOW" | "SHOWS_PAGE" | "CALLS_MICROFLOW" | "CALLS" | "HAS_DATASOURCE_MICROFLOW" | "HAS_DATA_CONTAINER"
	Depth     int
}

// FlowChain 是一条完整的入口→...→页面 链。
type FlowChain struct {
	EntryPoint string // "navigation:Responsive" | "workflow:MyWF" | "event:btnClick"
	Steps      []FlowStep
	TerminalPage string // final reachable page QN
}

// EntryPoint 描述一个应用入口点。
type EntryPoint struct {
	Kind string // "navigation" | "workflow" | "event" | "microflow"
	Name string // profile name / workflow QN / microflow QN
	Description string
}

// TransitiveFlowReader 执行多边类型传递式数据流分析。
type TransitiveFlowReader interface {
	// AllEntryPoints 返回所有已知入口点。
	AllEntryPoints() []EntryPoint

	// ReachablePagesFromEntry 从入口点出发，传递式遍历所有可达页面。
	// 返回完整的 FlowChain（含中间步骤）。
	ReachablePagesFromEntry(entryKind, entryName string) []FlowChain

	// AllReachablePages 从所有入口点出发，计算所有可达页面的并集。
	AllReachablePages() map[string][]string // pageQN → entry sources

	// EntryPointReachability 从指定入口做 N 层传递式 BFS。
	// maxDepth=0 表示无限制。
	EntryPointReachability(entryKind, entryName string, maxDepth int) (*FlowGraph, error)
}

// FlowGraph 是全量可达性的中间表示（去重的节点+边）。
type FlowGraph struct {
	Nodes []FlowGraphNode
	Edges []FlowGraphEdge
}

// FlowGraphNode 是流图中的节点。
type FlowGraphNode struct {
	ID    string
	Label string // "Page" | "Microflow" | "NavProfile" | "MenuItem"
	QN    string
}

// FlowGraphEdge 是流图中的边。
type FlowGraphEdge struct {
	From string
	To   string
	Type string
}
