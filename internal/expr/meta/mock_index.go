// SPDX-License-Identifier: Apache-2.0

package meta

import "github.com/mendixlabs/mxcli/mdl/exprcheck"

// MockIndex 是供测试使用的轻量 Index 实现。
// 不读取 MPR；调用方通过 Add* 方法填充测试数据。
type MockIndex struct {
	enumValues     map[string][]string
	constants      map[string]bool
	entityAttrs    map[string]map[string]exprcheck.TypeKind
	assocEndpoints map[string]AssocMeta
	microflowVars  map[string]map[string]string // unitPath → (varName → entityQN)
}

// NewMockIndex 构造空的 MockIndex；可选地传入预填充的枚举映射。
func NewMockIndex(enumValues map[string][]string) *MockIndex {
	if enumValues == nil {
		enumValues = map[string][]string{}
	}
	return &MockIndex{
		enumValues:     enumValues,
		constants:      map[string]bool{},
		entityAttrs:    map[string]map[string]exprcheck.TypeKind{},
		assocEndpoints: map[string]AssocMeta{},
		microflowVars:  map[string]map[string]string{},
	}
}

// AddConstant 注册常量引用（key 形如 "@Module.Name"）。
func (m *MockIndex) AddConstant(ref string) { m.constants[ref] = true }

// AddEntityAttr 在某实体下注册属性及其类型。
func (m *MockIndex) AddEntityAttr(entityQN, attrName string, kind exprcheck.TypeKind) {
	attrs, ok := m.entityAttrs[entityQN]
	if !ok {
		attrs = map[string]exprcheck.TypeKind{}
		m.entityAttrs[entityQN] = attrs
	}
	attrs[attrName] = kind
}

// AddAssoc 注册关联及其两端实体。
func (m *MockIndex) AddAssoc(assocQN, parent, child string) {
	m.assocEndpoints[assocQN] = AssocMeta{Parent: parent, Child: child}
}

// AddMicroflowVar 注册微流变量的实体类型（用于测试）。
func (m *MockIndex) AddMicroflowVar(unitPath, varName, entityQN string) {
	if _, ok := m.microflowVars[unitPath]; !ok {
		m.microflowVars[unitPath] = map[string]string{}
	}
	m.microflowVars[unitPath][varName] = entityQN
}

func (m *MockIndex) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	attrs, ok := m.entityAttrs[entityQN]
	if !ok {
		return exprcheck.KindUnknown, false
	}
	k, ok := attrs[attrName]
	return k, ok
}

func (m *MockIndex) AttributeEnumQN(_, _ string) (string, bool) { return "", false }

func (m *MockIndex) EnumCases(enumQN string) ([]string, bool) {
	v, ok := m.enumValues[enumQN]
	return v, ok
}

func (m *MockIndex) MicroflowReturn(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

func (m *MockIndex) MicroflowParam(_, _ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

func (m *MockIndex) VarTypeKind(_, _ string) exprcheck.TypeKind { return exprcheck.KindUnknown }

func (m *MockIndex) MicroflowParamKind(_, _ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

func (m *MockIndex) MicroflowReturnKind(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

func (m *MockIndex) HasConstant(ref string) bool { return m.constants[ref] }

func (m *MockIndex) HasEntity(qn string) bool {
	_, ok := m.entityAttrs[qn]
	return ok
}

func (m *MockIndex) HasAssociation(qn string) bool {
	_, ok := m.assocEndpoints[qn]
	return ok
}

func (m *MockIndex) IsEntityComplete(_ string) bool { return true }

func (m *MockIndex) VarEntityQN(unitPath, varName string) string {
	if vars, ok := m.microflowVars[unitPath]; ok {
		return vars[varName]
	}
	return ""
}

func (m *MockIndex) AssocEndpoints(assocQN string) (parent, child string, ok bool) {
	meta, ok := m.assocEndpoints[assocQN]
	return meta.Parent, meta.Child, ok
}

func (m *MockIndex) EntityAttrs(entityQN string) []string {
	attrs, ok := m.entityAttrs[entityQN]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(attrs))
	for k := range attrs {
		out = append(out, k)
	}
	return out
}

func (m *MockIndex) EntityCount() int { return len(m.entityAttrs) }
func (m *MockIndex) EnumCount() int   { return len(m.enumValues) }

// 编译期检查：MockIndex 必须满足 exprcheck.CatalogReader。
var _ exprcheck.CatalogReader = (*MockIndex)(nil)
