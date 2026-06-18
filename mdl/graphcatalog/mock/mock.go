package mock

import "github.com/mendixlabs/mxcli/mdl/graphcatalog"

// MockProjectGraph 实现 graphcatalog 的所有读取接口。
// 每个方法对应一个 Func 字段；未配置时 panic 给出明确错误。
type MockProjectGraph struct {
	EntitiesFunc            func(module string) []graphcatalog.EntityNode
	EntityFunc              func(qualifiedName string) *graphcatalog.EntityNode
	AttributesFunc          func(entityQN string) []graphcatalog.AttributeNode
	AssociationsFunc        func(module string) []graphcatalog.AssociationNode
	MicroflowsFunc          func(module string) []graphcatalog.MicroflowNode
	MicroflowFunc           func(qualifiedName string) *graphcatalog.MicroflowNode
	PagesFunc               func(module string) []graphcatalog.PageNode
	SnippetsFunc            func(module string) []graphcatalog.SnippetNode
	EnumerationsFunc        func(module string) []graphcatalog.EnumerationNode
	PermissionsFunc         func() []graphcatalog.PermissionNode
	RoleMappingsFunc        func() []graphcatalog.RoleMappingNode
	WidgetsFunc             func(pageQN string) []graphcatalog.WidgetNode
	DatabaseConnectionsFunc func() []graphcatalog.DatabaseConnectionNode
	CallersFunc             func(qualifiedName string, transitive bool) []graphcatalog.CallEdge
	CalleesFunc             func(qualifiedName string, transitive bool) []graphcatalog.CallEdge
	ImpactFunc              func(qualifiedName string) []graphcatalog.RefEdge
	ReferencesFunc          func(qualifiedName string) []graphcatalog.RefEdge
	// ThemeReader
	ThemeVariablesFunc   func(module string, filter graphcatalog.ThemeVarFilter) []graphcatalog.ThemeVariableNode
	ThemeVariableFunc    func(name string) *graphcatalog.ThemeVariableNode
	OverriddenVariablesFunc func() []graphcatalog.ThemeVariableNode
	// StylingReader
	WidgetInstancesFunc  func(pageQN string) []graphcatalog.WidgetInstanceNode
	DesignPropertiesFunc func(widgetType string) []graphcatalog.DesignPropertyNode
	DesignPropertyFunc   func(widgetType, name string) *graphcatalog.DesignPropertyNode
	// EntityAccessReader
	EntityAccessRulesFunc              func(entityQN string) []graphcatalog.AccessRuleNode
	EntityAccessRulesForRoleFunc       func(moduleRoleQN string) []graphcatalog.AccessRuleNode
	EntitiesWithMissingAccessRulesFunc func(module string) []graphcatalog.EntityNode
	// DocumentGrantReader (stub)
	PageAllowedRolesFunc  func(pageQN string) []string
	MFAllowedRolesFunc    func(mfQN string) []string
	ApplyEntityAccessFunc func(mfQN string) bool
}

// 编译期接口检查
var _ graphcatalog.LintReader = (*MockProjectGraph)(nil)
var _ graphcatalog.TraversalReader = (*MockProjectGraph)(nil)
var _ graphcatalog.ThemeReader = (*MockProjectGraph)(nil)
var _ graphcatalog.StylingReader = (*MockProjectGraph)(nil)
var _ graphcatalog.EntityAccessReader = (*MockProjectGraph)(nil)
var _ graphcatalog.DocumentGrantReader = (*MockProjectGraph)(nil)

func (m *MockProjectGraph) Entities(module string) []graphcatalog.EntityNode {
	if m.EntitiesFunc != nil {
		return m.EntitiesFunc(module)
	}
	panic("MockProjectGraph.Entities not configured")
}

func (m *MockProjectGraph) Entity(qn string) *graphcatalog.EntityNode {
	if m.EntityFunc != nil {
		return m.EntityFunc(qn)
	}
	panic("MockProjectGraph.Entity not configured")
}

func (m *MockProjectGraph) Attributes(entityQN string) []graphcatalog.AttributeNode {
	if m.AttributesFunc != nil {
		return m.AttributesFunc(entityQN)
	}
	panic("MockProjectGraph.Attributes not configured")
}

func (m *MockProjectGraph) Associations(module string) []graphcatalog.AssociationNode {
	if m.AssociationsFunc != nil {
		return m.AssociationsFunc(module)
	}
	panic("MockProjectGraph.Associations not configured")
}

func (m *MockProjectGraph) Microflows(module string) []graphcatalog.MicroflowNode {
	if m.MicroflowsFunc != nil {
		return m.MicroflowsFunc(module)
	}
	panic("MockProjectGraph.Microflows not configured")
}

func (m *MockProjectGraph) Microflow(qn string) *graphcatalog.MicroflowNode {
	if m.MicroflowFunc != nil {
		return m.MicroflowFunc(qn)
	}
	panic("MockProjectGraph.Microflow not configured")
}

func (m *MockProjectGraph) Pages(module string) []graphcatalog.PageNode {
	if m.PagesFunc != nil {
		return m.PagesFunc(module)
	}
	panic("MockProjectGraph.Pages not configured")
}

func (m *MockProjectGraph) Snippets(module string) []graphcatalog.SnippetNode {
	if m.SnippetsFunc != nil {
		return m.SnippetsFunc(module)
	}
	panic("MockProjectGraph.Snippets not configured")
}

func (m *MockProjectGraph) Enumerations(module string) []graphcatalog.EnumerationNode {
	if m.EnumerationsFunc != nil {
		return m.EnumerationsFunc(module)
	}
	panic("MockProjectGraph.Enumerations not configured")
}

func (m *MockProjectGraph) Permissions() []graphcatalog.PermissionNode {
	if m.PermissionsFunc != nil {
		return m.PermissionsFunc()
	}
	panic("MockProjectGraph.Permissions not configured")
}

func (m *MockProjectGraph) RoleMappings() []graphcatalog.RoleMappingNode {
	if m.RoleMappingsFunc != nil {
		return m.RoleMappingsFunc()
	}
	panic("MockProjectGraph.RoleMappings not configured")
}

func (m *MockProjectGraph) Widgets(pageQN string) []graphcatalog.WidgetNode {
	if m.WidgetsFunc != nil {
		return m.WidgetsFunc(pageQN)
	}
	panic("MockProjectGraph.Widgets not configured")
}

func (m *MockProjectGraph) DatabaseConnections() []graphcatalog.DatabaseConnectionNode {
	if m.DatabaseConnectionsFunc != nil {
		return m.DatabaseConnectionsFunc()
	}
	panic("MockProjectGraph.DatabaseConnections not configured")
}

func (m *MockProjectGraph) Callers(qn string, transitive bool) []graphcatalog.CallEdge {
	if m.CallersFunc != nil {
		return m.CallersFunc(qn, transitive)
	}
	panic("MockProjectGraph.Callers not configured")
}

func (m *MockProjectGraph) Callees(qn string, transitive bool) []graphcatalog.CallEdge {
	if m.CalleesFunc != nil {
		return m.CalleesFunc(qn, transitive)
	}
	panic("MockProjectGraph.Callees not configured")
}

func (m *MockProjectGraph) Impact(qn string) []graphcatalog.RefEdge {
	if m.ImpactFunc != nil {
		return m.ImpactFunc(qn)
	}
	panic("MockProjectGraph.Impact not configured")
}

func (m *MockProjectGraph) References(qn string) []graphcatalog.RefEdge {
	if m.ReferencesFunc != nil {
		return m.ReferencesFunc(qn)
	}
	panic("MockProjectGraph.References not configured")
}

// ── ThemeReader ──────────────────────────────────────────────

func (m *MockProjectGraph) ThemeVariables(module string, filter graphcatalog.ThemeVarFilter) []graphcatalog.ThemeVariableNode {
	if m.ThemeVariablesFunc != nil {
		return m.ThemeVariablesFunc(module, filter)
	}
	panic("MockProjectGraph.ThemeVariables not configured")
}

func (m *MockProjectGraph) ThemeVariable(name string) *graphcatalog.ThemeVariableNode {
	if m.ThemeVariableFunc != nil {
		return m.ThemeVariableFunc(name)
	}
	panic("MockProjectGraph.ThemeVariable not configured")
}

func (m *MockProjectGraph) OverriddenVariables() []graphcatalog.ThemeVariableNode {
	if m.OverriddenVariablesFunc != nil {
		return m.OverriddenVariablesFunc()
	}
	panic("MockProjectGraph.OverriddenVariables not configured")
}

// ── StylingReader ───────────────────────────────────────────

func (m *MockProjectGraph) WidgetInstances(pageQN string) []graphcatalog.WidgetInstanceNode {
	if m.WidgetInstancesFunc != nil {
		return m.WidgetInstancesFunc(pageQN)
	}
	panic("MockProjectGraph.WidgetInstances not configured")
}

func (m *MockProjectGraph) DesignProperties(widgetType string) []graphcatalog.DesignPropertyNode {
	if m.DesignPropertiesFunc != nil {
		return m.DesignPropertiesFunc(widgetType)
	}
	panic("MockProjectGraph.DesignProperties not configured")
}

func (m *MockProjectGraph) DesignProperty(widgetType, name string) *graphcatalog.DesignPropertyNode {
	if m.DesignPropertyFunc != nil {
		return m.DesignPropertyFunc(widgetType, name)
	}
	panic("MockProjectGraph.DesignProperty not configured")
}

// ── EntityAccessReader ──────────────────────────────────────

func (m *MockProjectGraph) EntityAccessRules(entityQN string) []graphcatalog.AccessRuleNode {
	if m.EntityAccessRulesFunc != nil {
		return m.EntityAccessRulesFunc(entityQN)
	}
	panic("MockProjectGraph.EntityAccessRules not configured")
}

func (m *MockProjectGraph) EntityAccessRulesForRole(moduleRoleQN string) []graphcatalog.AccessRuleNode {
	if m.EntityAccessRulesForRoleFunc != nil {
		return m.EntityAccessRulesForRoleFunc(moduleRoleQN)
	}
	panic("MockProjectGraph.EntityAccessRulesForRole not configured")
}

func (m *MockProjectGraph) EntitiesWithMissingAccessRules(module string) []graphcatalog.EntityNode {
	if m.EntitiesWithMissingAccessRulesFunc != nil {
		return m.EntitiesWithMissingAccessRulesFunc(module)
	}
	panic("MockProjectGraph.EntitiesWithMissingAccessRules not configured")
}

// ── DocumentGrantReader (stub) ──────────────────────────────

func (m *MockProjectGraph) PageAllowedRoles(pageQN string) []string {
	if m.PageAllowedRolesFunc != nil {
		return m.PageAllowedRolesFunc(pageQN)
	}
	panic("MockProjectGraph.PageAllowedRoles not configured")
}

func (m *MockProjectGraph) MFAllowedRoles(mfQN string) []string {
	if m.MFAllowedRolesFunc != nil {
		return m.MFAllowedRolesFunc(mfQN)
	}
	panic("MockProjectGraph.MFAllowedRoles not configured")
}

func (m *MockProjectGraph) ApplyEntityAccess(mfQN string) bool {
	if m.ApplyEntityAccessFunc != nil {
		return m.ApplyEntityAccessFunc(mfQN)
	}
	panic("MockProjectGraph.ApplyEntityAccess not configured")
}
