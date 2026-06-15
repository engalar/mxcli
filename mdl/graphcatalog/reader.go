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

// LintReader 是 linter 所需的完整接口（聚合 4 个子接口）。
type LintReader interface {
	DomainReader
	BehaviorReader
	SecurityReader
	ExtensionReader
}
