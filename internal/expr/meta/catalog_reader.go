// SPDX-License-Identifier: Apache-2.0

package meta

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// AttributeEnumQN 返回某实体属性对应的枚举 qualified name（仅枚举属性有）。
func (idx *Index) AttributeEnumQN(entityQN, attrName string) (string, bool) {
	key := entityQN + "." + attrName
	qn, ok := idx.entityAttrEnumQN[key]
	return qn, ok
}

// MicroflowReturn 返回 microflow 的返回类型。
func (idx *Index) MicroflowReturn(mfName string) (exprcheck.TypeKind, bool) {
	return idx.MicroflowReturnKind(mfName)
}

// MicroflowParam 返回 microflow 的某参数类型。
func (idx *Index) MicroflowParam(calleeQN, paramName string) (exprcheck.TypeKind, bool) {
	return idx.MicroflowParamKind(calleeQN, paramName)
}

// VarTypeKind returns the TypeKind of a microflow variable.
func (idx *Index) VarTypeKind(unitPath, varName string) exprcheck.TypeKind {
	if m, ok := idx.mfVarKinds[unitPath]; ok {
		if k, ok := m[varName]; ok {
			return k
		}
	}
	return exprcheck.KindUnknown
}

// MicroflowParamKind returns the TypeKind of a named parameter of a microflow.
func (idx *Index) MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool) {
	name := calleeQN
	if i := strings.LastIndex(calleeQN, "."); i >= 0 {
		name = calleeQN[i+1:]
	}
	params, ok := idx.mfParamKinds[name]
	if !ok {
		return exprcheck.KindUnknown, false
	}
	k, ok := params[paramName]
	return k, ok
}

// MicroflowReturnKind returns the TypeKind of a microflow's return value.
func (idx *Index) MicroflowReturnKind(mfName string) (exprcheck.TypeKind, bool) {
	name := mfName
	if i := strings.LastIndex(mfName, "."); i >= 0 {
		name = mfName[i+1:]
	}
	k, ok := idx.mfReturnKinds[name]
	return k, ok
}

// HasEntity 检查项目中是否存在某实体。
func (idx *Index) HasEntity(entityQN string) bool {
	_, ok := idx.entityAttrs[entityQN]
	return ok
}

// HasAssociation 检查项目中是否存在某关联（key 形如 "Module.AssocName"）。
func (idx *Index) HasAssociation(assocQN string) bool {
	_, ok := idx.assocEndpoints[assocQN]
	return ok
}

// VarEntityQN 返回微流变量的实体 QN。
// unitPath 是 scan.ExprRecord.UnitPath（如 "ae/c3/aec3b3b3-....mxunit"）。
// varName 是变量名（不含 $ 前缀）。找不到返回 ""。
func (idx *Index) VarEntityQN(unitPath, varName string) string {
	if m, ok := idx.microflowVars[unitPath]; ok {
		return m[varName]
	}
	return ""
}

// AssocEndpoints 返回关联的两端实体 QN。parent 是拥有侧，child 是另一侧。
func (idx *Index) AssocEndpoints(assocQN string) (parent, child string, ok bool) {
	m, ok := idx.assocEndpoints[assocQN]
	return m.Parent, m.Child, ok
}

// EntityAttrs 返回某实体的所有属性名（含继承，用于错误提示）。
func (idx *Index) EntityAttrs(entityQN string) []string {
	return idx.EntityAttrNames(entityQN)
}

// IsEntityComplete 返回实体的属性集合是否完整（继承链可完全解析）。
// 对继承链包含受保护市场模块父类的实体返回 false，此时应跳过属性验证。
func (idx *Index) IsEntityComplete(entityQN string) bool {
	return !idx.incompleteEntities[entityQN]
}

// EntityCount 返回已索引实体数。
func (idx *Index) EntityCount() int { return len(idx.entityAttrs) }

// AssocCount 返回已索引关联数。
func (idx *Index) AssocCount() int { return len(idx.assocEndpoints) }

// UnitQN returns the human-readable qualified name ("Module.MFName") for a
// unit identified by its unitPath (relative path from mprcontents/).
// Returns "" when the name is not known.
func (idx *Index) UnitQN(unitPath string) string {
	return idx.unitToQN[unitPath]
}

// 编译期检查：Index 必须满足 exprcheck.CatalogReader。
var _ exprcheck.CatalogReader = (*Index)(nil)
