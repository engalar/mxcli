// SPDX-License-Identifier: Apache-2.0

// Package meta 从 Mendix MPR 文件构建轻量语义元数据索引，
// 并实现 exprcheck.CatalogReader 接口供语义验证使用。
package meta

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// Index 持有从 MPR 提取的语义元数据，常驻内存。
type Index struct {
	entityAttrs      map[string]map[string]exprcheck.TypeKind
	entityAttrEnumQN map[string]string
	enumValues       map[string][]string
	constants        map[string]exprcheck.TypeKind
}

// BuildFromBackend 从已连接的 MPR 后端构建 Index。
func BuildFromBackend(b backend.FullBackend) (*Index, error) {
	idx := &Index{
		entityAttrs:      make(map[string]map[string]exprcheck.TypeKind),
		entityAttrEnumQN: make(map[string]string),
		enumValues:       make(map[string][]string),
		constants:        make(map[string]exprcheck.TypeKind),
	}

	if err := idx.buildEntityAttrs(b); err != nil {
		return nil, err
	}
	if err := idx.buildEnumValues(b); err != nil {
		return nil, err
	}
	if err := idx.buildConstants(b); err != nil {
		return nil, err
	}
	return idx, nil
}

func (idx *Index) buildEntityAttrs(b backend.FullBackend) error {
	modules, err := b.ListModules()
	if err != nil {
		return err
	}

	type entityInfo struct {
		qn               string
		generalizationQN string
		attrs            map[string]exprcheck.TypeKind
		attrEnumQN       map[string]string
	}
	entities := make(map[string]*entityInfo)

	for _, m := range modules {
		dm, err := b.GetDomainModelGen(m.ID)
		if err != nil || dm == nil {
			continue
		}
		moduleName := m.Name

		for _, elem := range dm.EntitiesItems() {
			entity, ok := elem.(*genDm.Entity)
			if !ok {
				continue
			}
			qn := moduleName + "." + entity.Name()

			info := &entityInfo{
				qn:         qn,
				attrs:      make(map[string]exprcheck.TypeKind),
				attrEnumQN: make(map[string]string),
			}

			if g, ok := entity.Generalization().(*genDm.Generalization); ok {
				info.generalizationQN = g.GeneralizationQualifiedName()
			}

			for _, aElem := range entity.AttributesItems() {
				attr, ok := aElem.(*genDm.Attribute)
				if !ok {
					continue
				}
				attrName := attr.Name()
				kind := attrTypeToKind(attr.Type())
				info.attrs[attrName] = kind

				if eat, ok := attr.Type().(*genDm.EnumerationAttributeType); ok {
					if eqn := eat.EnumerationQualifiedName(); eqn != "" {
						info.attrEnumQN[attrName] = eqn
					}
				}
			}
			entities[qn] = info
		}
	}

	for qn, info := range entities {
		attrs := make(map[string]exprcheck.TypeKind, len(info.attrs))
		enumQNs := make(map[string]string, len(info.attrEnumQN))
		for k, v := range info.attrs {
			attrs[k] = v
		}
		for k, v := range info.attrEnumQN {
			enumQNs[k] = v
		}

		seen := map[string]bool{qn: true}
		parentQN := info.generalizationQN
		for depth := 0; depth < 10 && parentQN != ""; depth++ {
			if seen[parentQN] {
				break
			}
			seen[parentQN] = true
			parent, ok := entities[parentQN]
			if !ok {
				break
			}
			for k, v := range parent.attrs {
				if _, exists := attrs[k]; !exists {
					attrs[k] = v
				}
			}
			for k, v := range parent.attrEnumQN {
				if _, exists := enumQNs[k]; !exists {
					enumQNs[k] = v
				}
			}
			parentQN = parent.generalizationQN
		}

		idx.entityAttrs[qn] = attrs
		for attrName, enumQN := range enumQNs {
			idx.entityAttrEnumQN[qn+"."+attrName] = enumQN
		}
	}

	return nil
}

func attrTypeToKind(t interface{}) exprcheck.TypeKind {
	switch t.(type) {
	case *genDm.StringAttributeType:
		return exprcheck.KindString
	case *genDm.IntegerAttributeType:
		return exprcheck.KindInteger
	case *genDm.LongAttributeType:
		return exprcheck.KindLong
	case *genDm.DecimalAttributeType:
		return exprcheck.KindDecimal
	case *genDm.BooleanAttributeType:
		return exprcheck.KindBoolean
	case *genDm.DateTimeAttributeType:
		return exprcheck.KindDateTime
	case *genDm.BinaryAttributeType:
		return exprcheck.KindBinary
	case *genDm.EnumerationAttributeType:
		return exprcheck.KindEnumeration
	default:
		return exprcheck.KindUnknown
	}
}

// buildEnumValues 与 buildConstants 在后续 commit 中扩展，
// 目前留空使 Index 构建成功并支持实体测试。
func (idx *Index) buildEnumValues(b backend.FullBackend) error { return nil }
func (idx *Index) buildConstants(b backend.FullBackend) error  { return nil }

// AllEntityQNs 返回所有已索引实体的 qualified name 列表（诊断用）。
func (idx *Index) AllEntityQNs() []string {
	out := make([]string, 0, len(idx.entityAttrs))
	for qn := range idx.entityAttrs {
		out = append(out, qn)
	}
	return out
}

// EntityAttrNames 返回某实体下所有已索引属性名（含父类继承，诊断用）。
func (idx *Index) EntityAttrNames(entityQN string) []string {
	attrs, ok := idx.entityAttrs[entityQN]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(attrs))
	for k := range attrs {
		out = append(out, k)
	}
	return out
}

// AttributeKind 返回给定实体的某个属性的类型。
func (idx *Index) AttributeKind(entityQN, attrName string) (exprcheck.TypeKind, bool) {
	attrs, ok := idx.entityAttrs[entityQN]
	if !ok {
		return exprcheck.KindUnknown, false
	}
	kind, ok := attrs[attrName]
	return kind, ok
}
