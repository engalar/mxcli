package graphcatalog

// EntityNode 对应 graph 中 label="Entity" 的节点。
type EntityNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
	IsExternal    bool
}

// AttributeNode 对应 label="Attribute"。
type AttributeNode struct {
	ID       string
	Name     string
	DataType string
	Entity   string // 所属实体 QualifiedName
}

// AssociationNode 对应 label="Association"。
type AssociationNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
	Owner         string // "Default" | "Both" | "Neither"
}

// MicroflowNode 对应 label="Microflow" 或 "Nanoflow"。
type MicroflowNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
	IsNanoflow    bool
	ReturnType    string
}

// PageNode 对应 label="Page"。
type PageNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
}

// SnippetNode 对应 label="Snippet"。
type SnippetNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
}

// EnumerationNode 对应 label="Enumeration"。
type EnumerationNode struct {
	ID            string
	Name          string
	QualifiedName string
	Module        string
}

// PermissionNode 对应 graph 中 label="ModuleRole" 的节点（计划中的 Permission 概念）。
type PermissionNode struct {
	ID           string
	EntityName   string
	ModuleRole   string
	AccessRights string
}

// RoleMappingNode 对应 graph 中 label="UserRole" 的节点（计划中的 RoleMapping 概念）。
type RoleMappingNode struct {
	ID         string
	UserRole   string
	ModuleRole string
}

// WidgetNode 对应 label="Widget"。
type WidgetNode struct {
	ID         string
	WidgetType string
	Name       string
	PageID     string
}

// DatabaseConnectionNode 对应 label="DatabaseConnection"。
type DatabaseConnectionNode struct {
	ID           string
	Name         string
	DatabaseType string
}

// CallEdge 表示一条调用关系（CALLS 边）。
type CallEdge struct {
	Caller string // QualifiedName
	Callee string // QualifiedName
	Depth  int    // transitive 遍历时的层级，直接调用为 1
}

// RefEdge 表示任意引用关系（CREATES / RETRIEVES / SHOWS_PAGE / HAS_ACTION 等）。
type RefEdge struct {
	Source  string // QualifiedName
	Target  string // QualifiedName
	RefKind string // "CALLS" | "CREATES" | "RETRIEVES" | "SHOWS_PAGE" | ...
}
