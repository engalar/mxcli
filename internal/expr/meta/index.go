// SPDX-License-Identifier: Apache-2.0

// Package meta 从 Mendix MPR 文件构建轻量语义元数据索引，
// 并实现 exprcheck.CatalogReader 接口供语义验证使用。
package meta

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// AssocMeta holds the two entity QNs that an association connects.
// Parent is the owning side; Child is the other side.
type AssocMeta struct {
	Parent string // e.g. "Common_Utils.PortalDispSetting"
	Child  string // e.g. "Administration.Account"
}

// Index 持有从 MPR 提取的语义元数据，常驻内存。
type Index struct {
	entityAttrs      map[string]map[string]exprcheck.TypeKind
	entityAttrEnumQN map[string]string
	enumValues       map[string][]string
	constants        map[string]exprcheck.TypeKind
	assocEndpoints   map[string]AssocMeta    // assocQN → {Parent, Child}
	entityByID       map[string]string       // element.ID → entityQN (for assoc resolution)
	microflowVars    map[string]map[string]string // unitPath → (varName → entityQN)
	// NEW — for SEM-03 type checking
	mfVarKinds    map[string]map[string]exprcheck.TypeKind // unitPath → varName → TypeKind
	mfParamKinds  map[string]map[string]exprcheck.TypeKind // bare MF name → paramName → TypeKind
	mfReturnKinds map[string]exprcheck.TypeKind            // bare MF name → return TypeKind
	// unitToQN maps unitPath → "Module.MFName" for human-readable error locations.
	unitToQN      map[string]string
	// incompleteEntities 记录属性集合不完整的实体：
	//   1. 继承链包含不可解析父类（父实体不在索引中）
	//   2. 来自 AppStore 市场模块（FromAppStore=true），其父类属性存储在受保护部分
	// 对这些实体的属性验证会跳过，以避免因无法读取父类属性而产生误报。
	incompleteEntities map[string]bool
}

// BuildFromBackend 从已连接的 MPR 后端构建 Index。
func BuildFromBackend(b backend.FullBackend) (*Index, error) {
	idx := &Index{
		entityAttrs:        make(map[string]map[string]exprcheck.TypeKind),
		entityAttrEnumQN:   make(map[string]string),
		enumValues:         make(map[string][]string),
		constants:          make(map[string]exprcheck.TypeKind),
		assocEndpoints:     make(map[string]AssocMeta),
		entityByID:         make(map[string]string),
		microflowVars:      make(map[string]map[string]string),
		incompleteEntities: make(map[string]bool),
		mfVarKinds:         make(map[string]map[string]exprcheck.TypeKind),
		mfParamKinds:       make(map[string]map[string]exprcheck.TypeKind),
		mfReturnKinds:      make(map[string]exprcheck.TypeKind),
		unitToQN:           make(map[string]string),
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
	if err := idx.buildMicroflowVars(b); err != nil {
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

			// ID → QN mapping for association resolution
			idx.entityByID[string(entity.ID())] = qn


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

			// Access rules list ALL accessible attributes (including those inherited
			// from protected marketplace base classes whose Generalization is not
			// readable from BSON). Scanning them discovers attributes that are not
			// directly on the entity but accessible through hidden specialization.
			// Scan access rules to discover inherited attributes from protected
			// base classes. Access rule MemberAccess QNs use the form
			// "Module.Entity.AttrName" and include ALL accessible attributes,
			// even those whose Generalization link isn't readable from BSON.
			accessRulesItems := entity.AccessRulesItems()
			seenAR := map[string]bool{}
			for _, arElem := range accessRulesItems {
				ar, ok := arElem.(*genDm.AccessRule)
				if !ok {
					continue
				}
				for _, maElem := range ar.MemberAccessesItems() {
					ma, ok := maElem.(*genDm.MemberAccess)
					if !ok {
						continue
					}
					attrQN := ma.AttributeQualifiedName()
					if attrQN == "" {
						continue
					}
					// Format: "Module.Entity.AttrName" → take after last dot.
					attrName := attrQN
					if i := strings.LastIndex(attrQN, "."); i >= 0 {
						attrName = attrQN[i+1:]
					}
					if attrName == "" || seenAR[attrName] {
						continue
					}
					seenAR[attrName] = true
					if _, exists := info.attrs[attrName]; !exists {
						info.attrs[attrName] = exprcheck.KindUnknown
					}
				}
			}

			// Marketplace module entities with no access rules have a hidden
			// generalization in the protected base — no path to discover
			// inherited attributes. Mark incomplete to suppress false positives.
			if m.FromAppStore && len(accessRulesItems) == 0 {
				idx.incompleteEntities[qn] = true
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
				// Parent entity is not in our index (protected marketplace module).
				// Mark this entity as incomplete so attribute validation is skipped.
				idx.incompleteEntities[qn] = true
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

	// Propagate incompleteEntities through the inheritance chain.
	// An entity is incomplete if any ancestor in its chain is incomplete.
	// Multiple passes handle chains of arbitrary depth (converges when no new
	// entities are added to the incomplete set).
	changed := true
	for changed {
		changed = false
		for qn, info := range entities {
			if idx.incompleteEntities[qn] {
				continue // already marked
			}
			parentQN := info.generalizationQN
			if parentQN != "" && idx.incompleteEntities[parentQN] {
				idx.incompleteEntities[qn] = true
				changed = true
			}
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
			parentQN := idx.entityByID[string(assoc.ParentRefID())]
			childQN := idx.entityByID[string(assoc.ChildRefID())]
			if parentQN == "" || childQN == "" {
				continue
			}
			idx.assocEndpoints[m.Name+"."+assoc.Name()] = AssocMeta{Parent: parentQN, Child: childQN}
		}
		for _, elem := range dm.CrossAssociationsItems() {
			assoc, ok := elem.(*genDm.CrossAssociation)
			if !ok {
				continue
			}
			parentQN := idx.entityByID[string(assoc.ParentRefID())]
			childQN := assoc.ChildQualifiedName()
			if parentQN == "" || childQN == "" {
				continue
			}
			idx.assocEndpoints[m.Name+"."+assoc.Name()] = AssocMeta{Parent: parentQN, Child: childQN}
		}
	}
	return nil
}

// buildMicroflowVars 遍历项目中的所有微流，完成两件事：
//
//  1. 为每个微流建立 varName → entityQN 映射（供 $var/attr 验证）。
//  2. 扫描 ChangeObjectAction / CreateObjectAction 中的 MemberChange，
//     发现隐式继承属性（三段式 AttributeQN = "Module.Entity.Attr"），
//     将 Attr 补录到操作对象所属实体的属性集合中。
//     这解决受保护市场模块 Generalization 无法通过 BSON 读取的问题。
// varName → entityQN 映射，存储在 idx.microflowVars[unitPath] 中。
//
// 变量来源（按 Mendix 规则）：
//   - MicroflowParameter — 实体类型参数
//   - CreateObjectAction — Create Object 活动的输出变量
//   - RetrieveAction（DatabaseRetrieveSource）— Retrieve 活动的输出变量
//   - LoopedActivity — LOOP 迭代变量（类型继承自被迭代列表的实体类型）
//
// 变量名在微流内全局唯一；LOOP 是唯一 scope 边界，迭代变量加入同一张 flat map，
// 递归处理 LoopedActivity.ObjectCollection() 内的嵌套活动。
func (idx *Index) buildMicroflowVars(b backend.FullBackend) error {
	mfs, err := b.ListMicroflowsGen()
	if err != nil {
		return err
	}

	// Build unitID → moduleName map by walking the container chain.
	findModuleName := idx.buildUnitModuleMap(b)

	// Pass 1: build microflow return-type index (bare name → entityQN).
	// MicroflowCall.MicroflowQualifiedName() gives "Module.MFName"; we index
	// by bare "MFName" for simplicity. Name collisions across modules are rare
	// in practice and the worst case is a missed validation, not a false positive.
	mfReturnType := make(map[string]string) // bare MF name → returned entityQN
	for _, mf := range mfs {
		rt := mf.MicroflowReturnType()
		if rt == nil {
			continue
		}
		if qn := entityQNFromElement(rt); qn != "" {
			mfReturnType[mf.Name()] = qn
		}
		kind := elementToKind(rt)
		if kind != exprcheck.KindUnknown {
			idx.mfReturnKinds[mf.Name()] = kind
		}
	}

	// Pass 2: per-microflow variable type maps + unit QN for error locations.
	for _, mf := range mfs {
		unitPath := unitPathFromID(string(mf.ID()))
		if moduleName := findModuleName(model.ID(string(mf.ID()))); moduleName != "" {
			idx.unitToQN[unitPath] = moduleName + "." + mf.Name()
		}
		varMap := make(map[string]string)
		paramKinds := make(map[string]exprcheck.TypeKind)

		if oc, ok := mf.ObjectCollection().(*genMf.MicroflowObjectCollection); ok {
			// Collect param kinds for mfParamKinds
			for _, obj := range oc.ObjectsItems() {
				if param, ok := obj.(*genMf.MicroflowParameter); ok {
					kind := elementToKind(param.ParameterType())
					if kind != exprcheck.KindUnknown {
						paramKinds[param.Name()] = kind
					}
				}
			}
			walkOC(oc, varMap, mfReturnType, unitPath, idx)
			idx.applyImplicitAttrs(oc, varMap)
		}

		if len(paramKinds) > 0 {
			idx.mfParamKinds[mf.Name()] = paramKinds
		}
		if len(varMap) > 0 {
			idx.microflowVars[unitPath] = varMap
		}
	}
	return nil
}

// applyImplicitAttrs scans ChangeObjectAction/CreateObjectAction items for
// three-part attribute QNs (Module.Entity.Attr), indicating an inherited
// attribute from a protected marketplace module generalization that is not
// readable from BSON. The attribute is added to the OPERATING entity's
// attribute set (looked up via varMap), not to the parent entity.
func (idx *Index) applyImplicitAttrs(oc *genMf.MicroflowObjectCollection, varMap map[string]string) {
	for _, obj := range oc.ObjectsItems() {
		switch o := obj.(type) {
		case *genMf.ActionActivity:
			idx.applyImplicitFromAction(o.Action(), varMap)
		case *genMf.LoopedActivity:
			if inner, ok := o.ObjectCollection().(*genMf.MicroflowObjectCollection); ok {
				idx.applyImplicitAttrs(inner, varMap)
			}
		}
	}
}

func (idx *Index) applyImplicitFromAction(action element.Element, varMap map[string]string) {
	var changeVarName string
	var items []element.Element

	switch a := action.(type) {
	case *genMf.ChangeObjectAction:
		changeVarName = a.ChangeVariableName()
		items = a.ItemsItems()
	case *genMf.CreateObjectAction:
		changeVarName = a.OutputVariableName()
		items = a.ItemsItems()
	default:
		return
	}

	entityQN, ok := varMap[changeVarName]
	if !ok || entityQN == "" {
		return
	}

	for _, item := range items {
		mc, ok := item.(*genMf.MemberChange)
		if !ok {
			continue
		}
		attrQN := mc.AttributeQualifiedName()
		// "Module.Entity.AttrName" = attribute from a protected generalization.
		parts := strings.SplitN(attrQN, ".", 3)
		if len(parts) != 3 {
			continue
		}
		attrName := parts[2]
		attrs, exists := idx.entityAttrs[entityQN]
		if !exists {
			attrs = make(map[string]exprcheck.TypeKind)
			idx.entityAttrs[entityQN] = attrs
		}
		if _, has := attrs[attrName]; !has {
			attrs[attrName] = exprcheck.KindUnknown
		}
	}
}

// walkOC 递归遍历 MicroflowObjectCollection，将变量声明写入 varMap。
func walkOC(oc *genMf.MicroflowObjectCollection, varMap map[string]string, mfReturnType map[string]string, unitPath string, idx *Index) {
	for _, obj := range oc.ObjectsItems() {
		switch o := obj.(type) {
		case *genMf.MicroflowParameter:
			if qn := entityQNFromElement(o.ParameterType()); qn != "" {
				varMap[o.Name()] = qn
			}
			// NEW: TypeKind for SEM-03
			kind := elementToKind(o.ParameterType())
			if kind != exprcheck.KindUnknown {
				idx.setVarKind(unitPath, o.Name(), kind)
			}
		case *genMf.ActionActivity:
			addActionVar(o.Action(), varMap, mfReturnType, unitPath, idx)
		case *genMf.LoopedActivity:
			loopVar := o.LoopVariableName()
			listVar := o.IteratedListVariableName()
			if loopVar != "" && listVar != "" {
				if entityQN, ok := varMap[listVar]; ok {
					varMap[loopVar] = entityQN
					idx.setVarKind(unitPath, loopVar, exprcheck.KindObject)
				}
			}
			if innerOC, ok := o.ObjectCollection().(*genMf.MicroflowObjectCollection); ok {
				walkOC(innerOC, varMap, mfReturnType, unitPath, idx)
			}
		}
	}
}

// addActionVar 从 ActionActivity 的具体 Action 中提取输出变量及其实体类型。
func addActionVar(action element.Element, varMap map[string]string, mfReturnType map[string]string, unitPath string, idx *Index) {
	switch a := action.(type) {
	case *genMf.CreateObjectAction:
		n, q := a.OutputVariableName(), a.EntityQualifiedName()
		if n != "" && q != "" {
			varMap[n] = q
			idx.setVarKind(unitPath, n, exprcheck.KindObject)
		}
	case *genMf.RetrieveAction:
		n := a.OutputVariableName()
		if n == "" {
			return
		}
		if src, ok := a.RetrieveSource().(*genMf.DatabaseRetrieveSource); ok {
			if q := src.EntityQualifiedName(); q != "" {
				varMap[n] = q
				idx.setVarKind(unitPath, n, exprcheck.KindObject)
			}
		}
	case *genMf.MicroflowCallAction:
		n := a.OutputVariableName()
		if n == "" || !a.UseReturnVariable() {
			return
		}
		mc, ok := a.MicroflowCall().(*genMf.MicroflowCall)
		if !ok {
			return
		}
		callee := mc.MicroflowQualifiedName()
		// Extract bare name: "Module.MFName" → "MFName"
		if i := strings.LastIndex(callee, "."); i >= 0 {
			callee = callee[i+1:]
		}
		if entityQN, ok := mfReturnType[callee]; ok {
			varMap[n] = entityQN
			idx.setVarKind(unitPath, n, exprcheck.KindObject)
		} else if kind, ok := idx.mfReturnKinds[callee]; ok {
			idx.setVarKind(unitPath, n, kind)
		}
	case *genMf.CreateVariableAction:
		n := a.VariableName()
		if n == "" {
			return
		}
		kind := dataTypeStringToKind(a.VariableDataType())
		if kind != exprcheck.KindUnknown {
			idx.setVarKind(unitPath, n, kind)
		}
	}
}

// entityQNFromElement 从 DataTypes 元素中提取实体 QN（仅 ObjectType）。
func entityQNFromElement(e element.Element) string {
	if t, ok := e.(*genDt.ObjectType); ok {
		return t.EntityQualifiedName()
	}
	return ""
}

// elementToKind converts a DataTypes element to its TypeKind.
// Note: genDt does not have a separate LongType; Long return types are
// represented as IntegerType in the reflection data at this schema version.
func elementToKind(e element.Element) exprcheck.TypeKind {
	switch e.(type) {
	case *genDt.BooleanType:
		return exprcheck.KindBoolean
	case *genDt.StringType:
		return exprcheck.KindString
	case *genDt.IntegerType:
		return exprcheck.KindInteger
	case *genDt.DecimalType:
		return exprcheck.KindDecimal
	case *genDt.DateTimeType:
		return exprcheck.KindDateTime
	case *genDt.BinaryType:
		return exprcheck.KindBinary
	case *genDt.ObjectType:
		return exprcheck.KindObject
	case *genDt.ListType:
		return exprcheck.KindList
	case *genDt.EnumerationType:
		return exprcheck.KindEnumeration
	}
	return exprcheck.KindUnknown
}

// dataTypeStringToKind converts CreateVariableAction.VariableDataType() string → TypeKind.
func dataTypeStringToKind(s string) exprcheck.TypeKind {
	switch s {
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
	case "Binary":
		return exprcheck.KindBinary
	}
	return exprcheck.KindUnknown
}

func (idx *Index) setVarKind(unitPath, varName string, kind exprcheck.TypeKind) {
	if idx.mfVarKinds[unitPath] == nil {
		idx.mfVarKinds[unitPath] = make(map[string]exprcheck.TypeKind)
	}
	idx.mfVarKinds[unitPath][varName] = kind
}

// unitPathFromID 将微流的 element.ID（标准 UUID 字符串）转换为
// mprcontents/ 中的相对路径，格式：ab/cd/abcd1234-...-....mxunit
func unitPathFromID(id string) string {
	clean := strings.ReplaceAll(id, "-", "")
	if len(clean) < 4 {
		return id + ".mxunit"
	}
	return clean[0:2] + "/" + clean[2:4] + "/" + id + ".mxunit"
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

// buildUnitModuleMap builds a function that resolves a unit ID to its module name
// by walking the container chain (unit → folder → ... → module).
func (idx *Index) buildUnitModuleMap(b backend.FullBackend) func(model.ID) string {
	modules, err := b.ListModules()
	if err != nil {
		return func(model.ID) string { return "" }
	}
	moduleNameOf := make(map[model.ID]string, len(modules))
	for _, m := range modules {
		moduleNameOf[m.ID] = m.Name
	}
	units, err := b.ListUnits()
	if err != nil {
		return func(model.ID) string { return "" }
	}
	parentOf := make(map[model.ID]model.ID, len(units))
	for _, u := range units {
		parentOf[u.ID] = u.ContainerID
	}
	cache := make(map[model.ID]string)
	var find func(model.ID) string
	find = func(id model.ID) string {
		if n, ok := cache[id]; ok {
			return n
		}
		if n, ok := moduleNameOf[id]; ok {
			cache[id] = n
			return n
		}
		parent, ok := parentOf[id]
		if !ok || parent == "" || parent == id {
			cache[id] = ""
			return ""
		}
		n := find(parent)
		cache[id] = n
		return n
	}
	return find
}
