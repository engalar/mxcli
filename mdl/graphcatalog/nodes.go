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

// NavigationProfileNode 对应 label="NavigationProfile"。
type NavigationProfileNode struct {
	ID            string
	Name          string
	Kind          string   // "Responsive" | "Native" | "Hybrid"
	IsNative      bool
	QualifiedName string
	HomePage      string   // page QN or microflow QN
	HasRoleBased  bool     // has role-based home pages
	MenuItemCount int
	LoginPage     string
	NotFoundPage  string
}

// NavigationMenuItemNode 对应 label="NavigationMenuItem"。
type NavigationMenuItemNode struct {
	ID        string
	Caption   string
	Page      string   // target page QN or empty
	Microflow string   // target microflow QN or empty
	Depth     int
	ParentID  string
	ProfileID string
}

// NavigationTree 表示完整导航树。
type NavigationTree struct {
	Profile   NavigationProfileNode
	HomePage  *NavigationHomePageNode
	MenuItems []NavigationMenuItemTree
}

// NavigationHomePageNode 表示首页配置。
type NavigationHomePageNode struct {
	Profile   string
	Page      string
	Microflow string
	Role      string // empty for default
}

// NavigationMenuItemTree 递归菜单项。
type NavigationMenuItemTree struct {
	Item     NavigationMenuItemNode
	Children []NavigationMenuItemTree
}

// DataContainerNode 对应 label="DataContainer"。
type DataContainerNode struct {
	ID                  string
	PageQN              string
	WidgetType          string // "DataView" | "DataGrid" | "ListView" | etc.
	WidgetName          string
	DataSourceType      string // "database" | "microflow" | "nanoflow" | "parameter" | "association" | "selection" | "none"
	EntityPath          string
	TargetEntity        string
	DataSourceMicroflow string
	ParameterName       string
	ListenTargetWidget  string
	HasSelection        bool
	SelectionName       string
	Depth               int
	ChildWidgets        []ChildWidgetSummary
	ContextVariables    []ContextVariable
}

// ChildWidgetSummary 是数据容器内子 widget 的外观摘要。
type ChildWidgetSummary struct {
	Name                  string `json:"name"`
	WidgetType            string `json:"widgetType"`
	Class                 string `json:"class,omitempty"`
	Style                 string `json:"style,omitempty"`
	Caption               string `json:"caption,omitempty"`
	Attribute             string `json:"attribute,omitempty"`
	ConditionalVisibility string `json:"condVis,omitempty"`
	ConditionalEditability string `json:"condEdit,omitempty"`
}

// ContextVariable 描述数据容器层级的一个语境变量。
type ContextVariable struct {
	Name       string `json:"name"`       // "$currentObject" | "$parent" | "$<name>"
	EntityType string `json:"entityType"` // entity QN
	Source     string `json:"source"`     // "datasource" | "parameter" | "parent" | "selection"
}

// VariableScope 描述数据容器层级的所有可用变量。
type VariableScope struct {
	ContainerID string
	Depth       int
	Variables   []ContextVariable
}

// EntityPageRef 描述实体被页面引用的情况。
type EntityPageRef struct {
	Entity        string
	Page          string
	DataSourceType string // "database" | "association" | "microflow" | "parameter" | "selection"
	ContainerName string
}

// EntityDataFlow 描述实体的完整数据流。
type EntityDataFlow struct {
	Entity   string
	Pages    []PageDataFlowSummary
	Creators  []string // microflows that create this entity
	Retrievers []string // microflows that retrieve this entity
}

// PageDataFlowSummary 描述页面对实体的访问。
type PageDataFlowSummary struct {
	Page            string
	DataSourceType  string
	AllowedRoles    []string
	NavigationRefs  []string // profiles referencing this page
}
