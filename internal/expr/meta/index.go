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
	assocQNs         map[string]bool
}

// BuildFromBackend 从已连接的 MPR 后端构建 Index。
func BuildFromBackend(b backend.FullBackend) (*Index, error) {
	idx := &Index{
		entityAttrs:      make(map[string]map[string]exprcheck.TypeKind),
		entityAttrEnumQN: make(map[string]string),
		enumValues:       make(map[string][]string),
		constants:        make(map[string]exprcheck.TypeKind),
		assocQNs:         make(map[string]bool),
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
	if err := idx.buildAssociations(b); err != nil {
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

// resolveModule 根据 unit 的 ContainerID 沿 folder 链回溯到 module 名。
// 如果 containerID 直接是 module，立即返回；否则跳跃 folderParent 直到命中
// module，或超过深度限制返回空串。
func resolveModule(containerID string, modByID map[string]string, folderParent map[string]string) string {
	current := containerID
	for range 20 {
		if name, ok := modByID[current]; ok {
			return name
		}
		parent, ok := folderParent[current]
		if !ok || parent == current {
			break
		}
		current = parent
	}
	return ""
}

func (idx *Index) buildEnumValues(b backend.FullBackend) error {
	enums, err := b.ListEnumerations()
	if err != nil {
		return err
	}
	modByID, folderParent, err := loadModuleMap(b)
	if err != nil {
		return err
	}

	for _, enum := range enums {
		moduleName := resolveModule(string(enum.ContainerID), modByID, folderParent)
		if moduleName == "" {
			continue
		}
		qn := moduleName + "." + enum.Name
		vals := make([]string, 0, len(enum.Values))
		for _, v := range enum.Values {
			vals = append(vals, v.Name)
		}
		idx.enumValues[qn] = vals
	}
	return nil
}

func (idx *Index) buildConstants(b backend.FullBackend) error {
	constants, err := b.ListConstants()
	if err != nil {
		return err
	}
	modByID, folderParent, err := loadModuleMap(b)
	if err != nil {
		return err
	}

	for _, c := range constants {
		moduleName := resolveModule(string(c.ContainerID), modByID, folderParent)
		if moduleName == "" {
			continue
		}
		key := "@" + moduleName + "." + c.Name
		idx.constants[key] = constantKindToExprKind(c.Type.Kind)
	}
	return nil
}

func loadModuleMap(b backend.FullBackend) (map[string]string, map[string]string, error) {
	modules, err := b.ListModules()
	if err != nil {
		return nil, nil, err
	}
	modByID := make(map[string]string, len(modules))
	for _, m := range modules {
		modByID[string(m.ID)] = m.Name
	}

	folders, err := b.ListFolders()
	if err != nil {
		return nil, nil, err
	}
	folderParent := make(map[string]string, len(folders))
	for _, f := range folders {
		folderParent[string(f.ID)] = string(f.ContainerID)
	}
	return modByID, folderParent, nil
}

func constantKindToExprKind(kind string) exprcheck.TypeKind {
	switch kind {
	case "String":
		return exprcheck.KindString
	case "Integer":
		return exprcheck.KindInteger
	case "Long":
		return exprcheck.KindLong
	case "Decimal":
		return exprcheck.KindDecimal
	case "Boolean":
		return exprcheck.KindBoolean
	case "DateTime":
		return exprcheck.KindDateTime
	default:
		return exprcheck.KindUnknown
	}
}

func (idx *Index) buildAssociations(b backend.FullBackend) error {
	modules, err := b.ListModules()
	if err != nil {
		return err
	}
	for _, m := range modules {
		dm, err := b.GetDomainModelGen(m.ID)
		if err != nil || dm == nil {
			continue
		}
		for _, elem := range dm.AssociationsItems() {
			assoc, ok := elem.(*genDm.Association)
			if !ok {
				continue
			}
			idx.assocQNs[m.Name+"."+assoc.Name()] = true
		}
		for _, elem := range dm.CrossAssociationsItems() {
			assoc, ok := elem.(*genDm.CrossAssociation)
			if !ok {
				continue
			}
			idx.assocQNs[m.Name+"."+assoc.Name()] = true
		}
	}
	return nil
}

// EnumCases 返回某枚举的所有 value 名；找不到返回 (nil, false)。
func (idx *Index) EnumCases(enumQN string) ([]string, bool) {
	vals, ok := idx.enumValues[enumQN]
	return vals, ok
}

// HasConstant 检查给定常量引用是否存在（key 形如 "@Module.Name"）。
func (idx *Index) HasConstant(ref string) bool {
	_, ok := idx.constants[ref]
	return ok
}

// EnumCount 返回已索引枚举数。
func (idx *Index) EnumCount() int { return len(idx.enumValues) }

// ConstantsCount 返回已索引常量数。
func (idx *Index) ConstantsCount() int { return len(idx.constants) }

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
