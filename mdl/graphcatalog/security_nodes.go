package graphcatalog

// AccessRuleNode 对应 label="AccessRule" 的节点。
type AccessRuleNode struct {
	ID              string
	EntityQN        string
	ModuleRoleQN    string
	CanRead         bool
	CanWrite        bool
	CanCreate       bool
	CanDelete       bool
	XPathConstraint string
}

// PageGrantNode 对应页面允许角色信息。
type PageGrantNode struct {
	PageQN       string
	ModuleRoleQN string
}

// MFGrantNode 对应微流授权信息。
type MFGrantNode struct {
	MFQN              string
	ModuleRoleQN      string
	ApplyEntityAccess bool
}
