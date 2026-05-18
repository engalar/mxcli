// SPDX-License-Identifier: Apache-2.0

package meta

import "github.com/mendixlabs/mxcli/mdl/exprcheck"

// AttributeEnumQN 返回某实体属性对应的枚举 qualified name（仅枚举属性有）。
func (idx *Index) AttributeEnumQN(entityQN, attrName string) (string, bool) {
	key := entityQN + "." + attrName
	qn, ok := idx.entityAttrEnumQN[key]
	return qn, ok
}

// MicroflowReturn 返回 microflow 的返回类型。当前未索引 microflow 元数据，
// 始终返回 KindUnknown/false；后续接入 microflow 索引时填充。
func (idx *Index) MicroflowReturn(_ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

// MicroflowParam 返回 microflow 的某参数类型。当前未索引 microflow，
// 始终返回 KindUnknown/false。
func (idx *Index) MicroflowParam(_, _ string) (exprcheck.TypeKind, bool) {
	return exprcheck.KindUnknown, false
}

// HasEntity 检查项目中是否存在某实体。
func (idx *Index) HasEntity(entityQN string) bool {
	_, ok := idx.entityAttrs[entityQN]
	return ok
}

// HasAssociation 检查项目中是否存在某关联（key 形如 "Module.AssocName"）。
func (idx *Index) HasAssociation(assocQN string) bool {
	return idx.assocQNs[assocQN]
}

// EntityCount 返回已索引实体数。
func (idx *Index) EntityCount() int { return len(idx.entityAttrs) }

// 编译期检查：Index 必须满足 exprcheck.CatalogReader。
var _ exprcheck.CatalogReader = (*Index)(nil)
