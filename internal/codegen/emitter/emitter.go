// Package emitter generates Go source files from parsed TS SDK metadata.
package emitter

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/mendixlabs/mxcli/internal/codegen/dtsparser"
)

// ────────────────────────────────────────────────────────
// Template data structs
// ────────────────────────────────────────────────────────

// TypeData holds template data for one concrete struct.
type TypeData struct {
	Name              string
	StructureTypeName string
	StorageAlias      string // BSON $Type if different from StructureTypeName
	IsAbstract        bool   // abstract classes have structs but no factory createIn methods
	IsVersionRename   bool   // true when this type is the NEW name in a type_renames entry
	ClassIntroduced   string // from cls.VersionInfo.Introduced
	ClassDeleted      string // from cls.VersionInfo.Deleted
	Fields            []FieldData
	Refs              []RefData // reference properties for refs.go generation
}

// TypeRenameData is an alias for dtsparser.TypeRenameData so callers in main.go
// can reference it without importing dtsparser directly if needed.
type TypeRenameData = dtsparser.TypeRenameData

// RefData holds template data for a single reference property in refs.go.
type RefData struct {
	Prop   string // BSON key
	Kind   string // "RefByName", "RefByNameList", or "RefById"
	Target string // TargetType (empty for ByIdRef)
}

// FieldData holds template data for a single property field.
type FieldData struct {
	PropName       string // original JS property name
	BSONKey        string // PascalCase BSON key (e.g. "Name", "Attributes")
	FieldName      string // Go unexported struct field name (e.g. "name")
	FieldType      string // Go type, e.g. "*property.Primitive[string]"
	FieldIndex     int    // zero-based index of this field among all properties (for Bind bit)
	GetterName     string // Go exported getter method name (e.g. "Name")
	GetterCall     string // method to call on the property (Get, Items, QualifiedName, etc.)
	GetterReturn   string // Go return type
	SetterName     string
	SetterArg      string
	SetterCall     string
	HasSetter      bool
	AdderName      string
	AdderArg       string
	HasAdder       bool
	RemoverName    string
	NeedsInit      bool   // true for Primitive properties (have Init(bson.Raw))
	IsPartChild    bool   // true for Part/PartList — needs recursive child decode
	IsList         bool   // true for PartList — decoded via DecodeChildren
	IsRef          bool   // true for ByNameRef — string from BSON
	IsRefList      bool   // true for ByNameRefList — string array from BSON
	IsIdRef        bool   // true for ByIdRef — binary UUID from BSON
	TargetType     string // ByNameRef target type (e.g. "DomainModels$Entity")
	IsEnum         bool   // true for Enum — string from BSON
	IsEnumList     bool   // true for EnumList — string array from BSON
	Constructor    string // e.g. "property.NewPrimitive[string](\"name\", property.DecodeString)"
	DescriptorKind string // PropKind constant for descriptors.go (empty for abstract types)
}

// EnumData holds template data for one enum type.
type EnumData struct {
	Name   string
	Values []EnumValueData
}

// EnumValueData holds a single enum value.
type EnumValueData struct {
	Name      string // original value name
	GoConst   string // e.g. "ExportLevelHidden"
	TypeAlias string // e.g. "ExportLevel"
}

// VersionData holds template data for one class's version info.
type VersionData struct {
	StructureTypeName string
	GoName            string // Go struct name (e.g. "CallMicroflowTask"); empty for alias entries
	IsAlias           bool   // true for storage-alias duplicate entries — no method generated
	ClassIntroduced   string // Mendix version when this class was introduced
	ClassDeleted      string // Mendix version when this class was deleted
	Props             []VersionPropData
}

// HasVersionGatedProps reports whether any property has Introduced or Deleted set.
// Only those properties need a case in the PropertyVersionInfo switch.
func (vd VersionData) HasVersionGatedProps() bool {
	for _, p := range vd.Props {
		if p.Introduced != "" || p.Deleted != "" {
			return true
		}
	}
	return false
}

// VersionGatedProps returns the subset of Props that have Introduced or Deleted set.
func (vd VersionData) VersionGatedProps() []VersionPropData {
	var out []VersionPropData
	for _, p := range vd.Props {
		if p.Introduced != "" || p.Deleted != "" {
			out = append(out, p)
		}
	}
	return out
}

// VersionPropData holds template data for one property's version info.
type VersionPropData struct {
	Name       string
	Introduced string
	Deleted    string
	Required   bool
	Public     bool
}

// typesFileData is the top-level template data for types.go.
type typesFileData struct {
	Package string
	Types   []TypeData
	Renames []TypeRenameData
}

// versionsFileData is the top-level template data for version.go.
type versionsFileData struct {
	Package  string
	Versions []VersionData
}

// ────────────────────────────────────────────────────────
// Generate
// ────────────────────────────────────────────────────────

// Generate renders Go source files from the parsed domain metadata.
// It writes types.go, enums.go, version.go, and refs.go into outDir.
// The caller (cmd/modelsdk-codegen) is responsible for populating
// meta.StorageAliases before calling Generate.
func Generate(meta *dtsparser.DomainMeta, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	pkg := meta.Namespace
	if pkg == "" {
		pkg = "generated"
	}

	// Build inheritance graph: map class name → *JsClass
	classMap := map[string]*dtsparser.JsClass{}
	for i := range meta.Classes {
		classMap[meta.Classes[i].Name] = &meta.Classes[i]
	}
	// Inject cross-domain base classes so the inheritance walker can
	// resolve types like "Document" from projects domain.
	if meta.CrossDomainProps != nil {
		for name, props := range meta.CrossDomainProps {
			if _, exists := classMap[name]; !exists {
				classMap[name] = &dtsparser.JsClass{
					Name:       name,
					Properties: props,
				}
			}
		}
	}

	// Build types data — generate ALL classes (both abstract and concrete).
	// Abstract classes need structs for their properties; they just won't
	// have factory createIn methods.
	var types []TypeData
	for i := range meta.Classes {
		cls := &meta.Classes[i]
		if cls.StructureTypeName == "" {
			continue
		}
		// Collect inherited properties by walking up the inheritance chain.
		allProps := collectInheritedProps(cls, classMap)
		td := buildTypeDataWithProps(cls, allProps)
		td.IsAbstract = cls.IsAbstract
		// Populate version-info fields.
		if cls.VersionInfo != nil {
			td.ClassIntroduced = cls.VersionInfo.Introduced
			td.ClassDeleted = cls.VersionInfo.Deleted
		}
		// Mark as version-rename if this type is the NEW name in a type_renames entry.
		for _, r := range meta.TypeRenames {
			if cls.StructureTypeName == r.NewTypeName {
				td.IsVersionRename = true
				break
			}
		}
		// Look up storage alias if available.
		if meta.StorageAliases != nil {
			if alias, ok := meta.StorageAliases[cls.StructureTypeName]; ok {
				td.StorageAlias = alias
			}
		}
		// Apply namespace mapping so generated code uses BSON storage names
		// directly (e.g., "Pages$Page" → "Forms$Page").
		td.StructureTypeName = mapToStorageNamespace(td.StructureTypeName)
		for fi := range td.Fields {
			td.Fields[fi].Constructor = mapConstructorTargetType(td.Fields[fi].Constructor)
			// Apply property-level BSON key overrides.
			if meta.PropertyKeyOverrides != nil {
				key := cls.Name + "." + td.Fields[fi].PropName
				override, ok := meta.PropertyKeyOverrides[key]
				if !ok {
					// Wildcard: "*.propName" matches any class
					override, ok = meta.PropertyKeyOverrides["*."+td.Fields[fi].PropName]
				}
				if ok {
					td.Fields[fi].BSONKey = override
					td.Fields[fi].Constructor = strings.Replace(
						td.Fields[fi].Constructor,
						"\""+exportName(td.Fields[fi].PropName)+"\"",
						"\""+override+"\"", 1)
				}
			}
			// Apply RefList version3 override: switch NewByNameRefList → NewByNameRefListV3
			// for fields that require BSON version marker int32(3) instead of int32(1).
			if td.Fields[fi].IsRefList && meta.RefListVersion3Fields != nil {
				v3key := cls.Name + "." + td.Fields[fi].PropName
				if meta.RefListVersion3Fields[v3key] {
					td.Fields[fi].Constructor = strings.Replace(
						td.Fields[fi].Constructor,
						"property.NewByNameRefList[",
						"property.NewByNameRefListV3[", 1)
				}
			}
			// Apply PartList version2 override: switch NewPartList → NewPartListV2
			// for fields that require BSON version marker int32(2) instead of int32(3).
			// Mendix uses version 2 for Parameters/TypeParameters on JavaAction.
			if td.Fields[fi].IsList && !td.Fields[fi].IsRefList && meta.PartListVersion2Fields != nil {
				v2key := cls.Name + "." + td.Fields[fi].PropName
				if meta.PartListVersion2Fields[v2key] {
					td.Fields[fi].Constructor = strings.Replace(
						td.Fields[fi].Constructor,
						"property.NewPartList[",
						"property.NewPartListV2[", 1)
				}
			}
			// Apply BinaryUUID override: switch Primitive[string]/DecodeString to
			// BinaryUUIDPrimitive for fields that Mendix stores as BSON Binary UUID
			// (e.g. PersistentId). Match "ClassName.propName" or "*.propName".
			if meta.BinaryUUIDProps != nil && td.Fields[fi].NeedsInit {
				propKey := cls.Name + "." + td.Fields[fi].PropName
				wildKey := "*." + td.Fields[fi].PropName
				if meta.BinaryUUIDProps[propKey] || meta.BinaryUUIDProps[wildKey] {
					bsonKey := td.Fields[fi].BSONKey
					td.Fields[fi].FieldType = "*property.BinaryUUIDPrimitive"
					td.Fields[fi].GetterReturn = "string"
					td.Fields[fi].SetterArg = "string"
					td.Fields[fi].Constructor = "property.NewBinaryUUIDPrimitive(\"" + bsonKey + "\")"
				}
			}

			// Apply Binary override: switch Primitive[string]/DecodeString to
			// BinaryPrimitive for fields that Mendix stores as raw BSON Binary
			// blobs (e.g. Image.imageData). Match "ClassName.propName" or "*.propName".
			if meta.BinaryProps != nil && td.Fields[fi].NeedsInit {
				propKey := cls.Name + "." + td.Fields[fi].PropName
				wildKey := "*." + td.Fields[fi].PropName
				if meta.BinaryProps[propKey] || meta.BinaryProps[wildKey] {
					bsonKey := td.Fields[fi].BSONKey
					td.Fields[fi].FieldType = "*property.BinaryPrimitive"
					td.Fields[fi].GetterReturn = "[]byte"
					td.Fields[fi].SetterArg = "[]byte"
					td.Fields[fi].Constructor = "property.NewBinaryPrimitive(\"" + bsonKey + "\")"
				}
			}
		}
		// Apply property order overrides — reorder fields to match Mendix's
		// BSON serialization order (which may differ from the SDK definition order).
		if meta.PropertyOrderOverrides != nil {
			if order, ok := meta.PropertyOrderOverrides[cls.StructureTypeName]; ok {
				reorderFields(&td, order)
			}
		}
		types = append(types, td)
	}

	// Build reference metadata for refs.go.
	for ti := range types {
		td := &types[ti]
		for _, f := range td.Fields {
			switch {
			case f.IsRef:
				td.Refs = append(td.Refs, RefData{Prop: f.BSONKey, Kind: "RefByName", Target: f.TargetType})
			case f.IsRefList:
				td.Refs = append(td.Refs, RefData{Prop: f.BSONKey, Kind: "RefByNameList", Target: f.TargetType})
			case f.IsIdRef:
				td.Refs = append(td.Refs, RefData{Prop: f.BSONKey, Kind: "RefById", Target: ""})
			}
		}
	}

	// Build descriptor kind metadata for descriptors.go.
	for ti := range types {
		td := &types[ti]
		for fi := range td.Fields {
			f := &td.Fields[fi]
			switch {
			case f.IsPartChild && f.IsList:
				f.DescriptorKind = "PropKindPartList"
			case f.IsPartChild:
				f.DescriptorKind = "PropKindPart"
			case f.IsRef:
				f.DescriptorKind = "PropKindByNameRef"
			case f.IsRefList:
				f.DescriptorKind = "PropKindByNameList"
			case f.IsIdRef:
				f.DescriptorKind = "PropKindByIdRef"
			case f.IsEnum:
				f.DescriptorKind = "PropKindEnum"
			case f.IsEnumList:
				f.DescriptorKind = "PropKindEnumList"
			case strings.Contains(f.FieldType, "StringListPrimitive"):
				f.DescriptorKind = "PropKindStringList"
			case strings.Contains(f.FieldType, "BinaryUUIDPrimitive"):
				f.DescriptorKind = "PropKindBinaryUUID"
			case strings.Contains(f.FieldType, "BinaryPrimitive"):
				f.DescriptorKind = "PropKindBinary"
			case strings.Contains(f.FieldType, "Primitive[string]"):
				f.DescriptorKind = "PropKindString"
			case strings.Contains(f.FieldType, "Primitive[bool]"):
				f.DescriptorKind = "PropKindBool"
			case strings.Contains(f.FieldType, "Primitive[int32]"):
				f.DescriptorKind = "PropKindInt32"
			case strings.Contains(f.FieldType, "Primitive[float64]"):
				f.DescriptorKind = "PropKindFloat64"
			}
		}
	}

	// Build enums data.
	var enums []EnumData
	for _, e := range meta.Enums {
		goName := exportName(e.Name)
		ed := EnumData{Name: goName}
		for _, v := range e.Values {
			ed.Values = append(ed.Values, EnumValueData{
				Name:      v.Name,
				GoConst:   goName + exportName(v.Name),
				TypeAlias: goName,
			})
		}
		enums = append(enums, ed)
	}

	// Build versions data.
	var versions []VersionData
	for i := range meta.Classes {
		cls := &meta.Classes[i]
		if cls.VersionInfo == nil || cls.StructureTypeName == "" {
			continue
		}
		hasProps := len(cls.VersionInfo.PropertyInfos) > 0
		hasClassVer := cls.VersionInfo.Introduced != "" || cls.VersionInfo.Deleted != ""
		if !hasProps && !hasClassVer {
			continue
		}
		mappedName := mapToStorageNamespace(cls.StructureTypeName)
		vd := VersionData{
			StructureTypeName: mappedName,
			GoName:            exportName(cls.Name),
		}
		vd.ClassIntroduced = cls.VersionInfo.Introduced
		vd.ClassDeleted = cls.VersionInfo.Deleted
		// Sort property names for stable output across runs.
		propNames := make([]string, 0, len(cls.VersionInfo.PropertyInfos))
		for name := range cls.VersionInfo.PropertyInfos {
			propNames = append(propNames, name)
		}
		sort.Strings(propNames)
		for _, name := range propNames {
			pvi := cls.VersionInfo.PropertyInfos[name]
			vd.Props = append(vd.Props, VersionPropData{
				Name:       name,
				Introduced: pvi.Introduced,
				Deleted:    pvi.Deleted,
				Required:   pvi.RequiredNow,
				Public:     pvi.PublicNow,
			})
		}
		versions = append(versions, vd)
		// Emit a duplicate version entry for the storage alias (VersionInfos map only).
		// IsAlias=true prevents a second PropertyVersionInfo method on the same struct.
		if meta.StorageAliases != nil {
			if alias, ok := meta.StorageAliases[cls.StructureTypeName]; ok {
				aliasVD := VersionData{
					StructureTypeName: alias,
					IsAlias:           true,
					ClassIntroduced:   vd.ClassIntroduced,
					ClassDeleted:      vd.ClassDeleted,
					Props:             vd.Props,
				}
				versions = append(versions, aliasVD)
			}
		}
	}

	// Render types.go
	if err := renderFile(filepath.Join(outDir, "types.go"), typesTemplate,
		typesFileData{Package: pkg, Types: types, Renames: meta.TypeRenames}); err != nil {
		return fmt.Errorf("types.go: %w", err)
	}

	// Render enums.go
	if err := renderFile(filepath.Join(outDir, "enums.go"), enumsTemplate,
		struct {
			Package string
			Enums   []EnumData
		}{Package: pkg, Enums: enums}); err != nil {
		return fmt.Errorf("enums.go: %w", err)
	}

	// Render version.go
	if err := renderFile(filepath.Join(outDir, "version.go"), versionTemplate,
		versionsFileData{Package: pkg, Versions: versions}); err != nil {
		return fmt.Errorf("version.go: %w", err)
	}

	// Render refs.go
	if err := renderFile(filepath.Join(outDir, "refs.go"), refsTemplate,
		typesFileData{Package: pkg, Types: types}); err != nil {
		return fmt.Errorf("refs.go: %w", err)
	}

	// Render descriptors.go
	if err := renderFile(filepath.Join(outDir, "descriptors.go"), descriptorsTemplate,
		typesFileData{Package: pkg, Types: types}); err != nil {
		return fmt.Errorf("descriptors.go: %w", err)
	}

	return nil
}

// collectInheritedProps walks up the inheritance chain within the same domain
// and collects all properties (inherited first, then own).
func collectInheritedProps(cls *dtsparser.JsClass, classMap map[string]*dtsparser.JsClass) []dtsparser.JsProp {
	var chain []*dtsparser.JsClass
	chain = append(chain, cls)

	current := cls
	visited := map[string]bool{cls.Name: true}
	for {
		baseName := resolveBaseClassName(current.BaseClass)
		if visited[baseName] {
			break // cycle detected
		}
		base, ok := classMap[baseName]
		if !ok {
			break // base is in a different domain or is internal.Element
		}
		visited[baseName] = true
		chain = append(chain, base)
		current = base
	}

	// Reverse to get root-first order.
	var allProps []dtsparser.JsProp
	seen := map[string]bool{}
	for i := len(chain) - 1; i >= 0; i-- {
		for _, p := range chain[i].Properties {
			if seen[p.Name] {
				continue // skip duplicates (shouldn't happen normally)
			}
			seen[p.Name] = true
			allProps = append(allProps, p)
		}
	}
	return allProps
}

// resolveBaseClassName strips module/import prefixes from a base class reference.
// E.g. "projects_1.projects.ModuleDocument" → "ModuleDocument"
//
//	"internal.Element" → ""  (SDK base types, stop walk)
//	"AssociationBase" → "AssociationBase"
func resolveBaseClassName(base string) string {
	if base == "" {
		return ""
	}
	// internal.Element, internal.ModelUnit, internal.StructuralUnit are SDK
	// base types — not domain classes. Stop inheritance walk here.
	if strings.HasPrefix(base, "internal.") {
		return ""
	}
	// Cross-domain references like "projects_1.projects.Document":
	// extract the class name (last segment) so the inheritance walker
	// can look it up in classMap (populated from CrossDomainProps).
	if idx := strings.LastIndex(base, "."); idx >= 0 {
		return base[idx+1:]
	}
	return base
}

// buildTypeDataWithProps builds TypeData using a provided list of properties
// (which may include inherited ones).
func buildTypeDataWithProps(cls *dtsparser.JsClass, props []dtsparser.JsProp) TypeData {
	td := TypeData{
		Name:              exportName(cls.Name),
		StructureTypeName: cls.StructureTypeName,
	}
	for i, p := range props {
		fd := buildFieldData(&p)
		fd.FieldIndex = i
		td.Fields = append(td.Fields, fd)
	}
	return td
}

// ────────────────────────────────────────────────────────
// Build helpers
// ────────────────────────────────────────────────────────

// goKeywords is the set of Go reserved keywords that cannot be used as
// identifiers. Property names matching these get a "prop" prefix.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true,
	"for": true, "func": true, "go": true, "goto": true, "if": true,
	"import": true, "interface": true, "map": true, "package": true,
	"range": true, "return": true, "select": true, "struct": true,
	"switch": true, "type": true, "var": true,
}

// reservedFieldNames is the set of field names that would collide with
// embedded element.Base unexported fields if used as-is.
var reservedFieldNames = map[string]bool{
	"id": true, "typeName": true, "container": true,
	"unit": true, "raw": true, "dirty": true, "props": true,
}

func buildFieldData(p *dtsparser.JsProp) FieldData {
	fieldName := unexportName(p.Name)
	if reservedFieldNames[fieldName] || goKeywords[fieldName] {
		fieldName = "prop" + exportName(p.Name)
	}
	getterBase := exportName(p.Name)

	// BSON keys in Mendix are PascalCase (e.g. "Name", "Documentation"),
	// but JS SDK property names are camelCase (e.g. "name", "documentation").
	// The property Name() must match the BSON key for lazy decode to work.
	bsonKey := exportName(p.Name)
	// Exception: $ID and $Type are special and stay as-is.
	if p.Name == "$ID" || p.Name == "$Type" {
		bsonKey = p.Name
	}

	fd := FieldData{
		PropName:  p.Name,
		BSONKey:   exportName(p.Name),
		FieldName: fieldName,
	}

	switch p.Kind {
	case dtsparser.PKPrimitive:
		goType, decodeFunc := primitiveMapping(p.PrimitiveType)
		fd.FieldType = "*property.Primitive[" + goType + "]"
		fd.GetterName = getterBase
		fd.GetterCall = "Get"
		fd.GetterReturn = goType
		fd.SetterName = "Set" + getterBase
		fd.SetterArg = goType
		fd.SetterCall = "Set"
		fd.HasSetter = true
		fd.NeedsInit = true
		fd.Constructor = "property.NewPrimitive[" + goType + "](\"" + bsonKey + "\", property." + decodeFunc + ")"

	case dtsparser.PKPart, dtsparser.PKStructuralChild:
		fd.FieldType = "*property.Part[element.Element]"
		fd.GetterName = getterBase
		fd.GetterCall = "Get"
		fd.GetterReturn = "element.Element"
		fd.SetterName = "Set" + getterBase
		fd.SetterArg = "element.Element"
		fd.SetterCall = "Set"
		fd.HasSetter = true
		fd.IsPartChild = true
		fd.Constructor = "property.NewPart[element.Element](\"" + bsonKey + "\")"

	case dtsparser.PKPartList, dtsparser.PKStructuralChildList:
		fd.FieldType = "*property.PartList[element.Element]"
		fd.GetterName = getterBase + "Items"
		fd.GetterCall = "Items"
		fd.GetterReturn = "[]element.Element"
		fd.HasAdder = true
		fd.AdderName = "Add" + getterBase
		fd.AdderArg = "element.Element"
		fd.RemoverName = "Remove" + getterBase
		fd.IsPartChild = true
		fd.IsList = true
		fd.Constructor = "property.NewPartList[element.Element](\"" + bsonKey + "\")"

	case dtsparser.PKByNameRef:
		fd.FieldType = "*property.ByNameRef[element.Element]"
		fd.GetterName = getterBase + "QualifiedName"
		fd.GetterCall = "QualifiedName"
		fd.GetterReturn = "string"
		fd.SetterName = "Set" + getterBase + "QualifiedName"
		fd.SetterArg = "string"
		fd.SetterCall = "SetQualifiedName"
		fd.HasSetter = true
		fd.IsRef = true
		targetType := p.TargetType
		if targetType == "" {
			targetType = "unknown"
		}
		fd.TargetType = mapToStorageNamespace(targetType)
		fd.Constructor = "property.NewByNameRef[element.Element](\"" + bsonKey + "\", \"" + targetType + "\")"

	case dtsparser.PKByNameRefList:
		fd.FieldType = "*property.ByNameRefList[element.Element]"
		fd.GetterName = getterBase + "QualifiedNames"
		fd.GetterCall = "QualifiedNames"
		fd.GetterReturn = "[]string"
		fd.IsRefList = true
		fd.AdderName = "Add" + getterBase
		fd.AdderArg = "string"
		fd.HasAdder = true
		fd.SetterName = "Set" + getterBase + "QualifiedNames"
		fd.SetterArg = "[]string"
		fd.SetterCall = "SetQualifiedNames"
		fd.HasSetter = true
		targetType := p.TargetType
		if targetType == "" {
			targetType = "unknown"
		}
		fd.TargetType = mapToStorageNamespace(targetType)
		fd.Constructor = "property.NewByNameRefList[element.Element](\"" + bsonKey + "\", \"" + targetType + "\")"

	case dtsparser.PKByIdRef:
		fd.FieldType = "*property.ByIdRef[element.Element]"
		fd.GetterName = getterBase + "RefID"
		fd.GetterCall = "RefID"
		fd.GetterReturn = "element.ID"
		fd.SetterName = "Set" + getterBase + "ID"
		fd.SetterArg = "element.ID"
		fd.SetterCall = "SetID"
		fd.HasSetter = true
		fd.IsIdRef = true
		fd.Constructor = "property.NewByIdRef[element.Element](\"" + bsonKey + "\")"

	case dtsparser.PKEnum:
		fd.FieldType = "*property.Enum[string]"
		fd.GetterName = getterBase
		fd.GetterCall = "Get"
		fd.GetterReturn = "string"
		fd.SetterName = "Set" + getterBase
		fd.SetterArg = "string"
		fd.SetterCall = "Set"
		fd.HasSetter = true
		fd.IsEnum = true
		fd.Constructor = "property.NewEnum[string](\"" + bsonKey + "\")"

	case dtsparser.PKEnumList:
		fd.FieldType = "*property.EnumList[string]"
		fd.GetterName = getterBase + "Items"
		fd.GetterCall = "Items"
		fd.GetterReturn = "[]string"
		fd.IsEnumList = true
		fd.AdderName = "Add" + getterBase
		fd.AdderArg = "string"
		fd.HasAdder = true
		fd.Constructor = "property.NewEnumList[string](\"" + bsonKey + "\")"

	case dtsparser.PKPrimitiveList:
		// Mendix serializes PrimitiveListProperty as a BSON array of strings.
		// StringListPrimitive serializes as bson.A; a plain Primitive[string]
		// would emit a scalar and trigger InvalidCastException in mx check.
		fd.FieldType = "*property.StringListPrimitive"
		fd.GetterName = getterBase
		fd.GetterCall = "Get"
		fd.GetterReturn = "string"
		fd.SetterName = "Set" + getterBase
		fd.SetterArg = "string"
		fd.SetterCall = "Set"
		fd.HasSetter = true
		fd.NeedsInit = true
		fd.Constructor = "property.NewStringListPrimitive(\"" + bsonKey + "\")"

	default:
		// PKUnknown -- fall back to string primitive
		fd.FieldType = "*property.Primitive[string]"
		fd.GetterName = getterBase
		fd.GetterCall = "Get"
		fd.GetterReturn = "string"
		fd.NeedsInit = true
		fd.Constructor = "property.NewPrimitive[string](\"" + bsonKey + "\", property.DecodeString)"
	}

	return fd
}

// primitiveMapping maps a PrimitiveType to (Go type, decode func name).
func primitiveMapping(pt dtsparser.PrimitiveType) (goType, decodeFunc string) {
	switch pt {
	case dtsparser.PTBoolean:
		return "bool", "DecodeBool"
	case dtsparser.PTInteger:
		return "int32", "DecodeInt32"
	case dtsparser.PTDouble:
		return "float64", "DecodeFloat64"
	default:
		// String, Guid, Blob, Point, Size, Color all map to string
		return "string", "DecodeString"
	}
}

// ────────────────────────────────────────────────────────
// Name conversion
// ────────────────────────────────────────────────────────

// exportName converts a camelCase JS name to a Go exported PascalCase name.
func exportName(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// unexportName converts a name to an unexported Go identifier by lowercasing
// the first rune. This is used for struct field names to avoid collisions
// with getter methods.
func unexportName(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// ────────────────────────────────────────────────────────
// Template rendering
// ────────────────────────────────────────────────────────

func renderFile(path, tmplStr string, data interface{}) error {
	tmpl, err := template.New(filepath.Base(path)).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Format the generated source. For large files (>200KB) use the external
	// gofmt tool instead of go/format to avoid OOM in constrained environments.
	src := buf.Bytes()
	if len(src) < 200_000 {
		if formatted, err := format.Source(src); err == nil {
			src = formatted
		}
	} else {
		// Large file: write first, then gofmt -w in-place.
		if err := os.WriteFile(path, src, 0o644); err != nil {
			return fmt.Errorf("write (pre-fmt) %s: %w", path, err)
		}
		if gofmtPath, lookErr := exec.LookPath("gofmt"); lookErr == nil {
			_ = exec.Command(gofmtPath, "-w", path).Run()
		}
		return nil
	}

	return os.WriteFile(path, src, 0o644)
}

// namespaceMap maps SDK namespace prefixes to their BSON storage namespace.
// The JS SDK uses "Pages" but BSON stores "Forms". By applying this at codegen
// time, generated code registers types under their BSON storage names directly,
// eliminating runtime alias lookups.
var namespaceMap = map[string]string{
	"Pages": "Forms",
}

// mapToStorageNamespace replaces the namespace prefix in a structure type name
// using namespaceMap. E.g., "Pages$ActionButton" → "Forms$ActionButton".
func mapToStorageNamespace(stn string) string {
	if idx := strings.IndexByte(stn, '$'); idx > 0 {
		if mapped, ok := namespaceMap[stn[:idx]]; ok {
			return mapped + stn[idx:]
		}
	}
	return stn
}

// mapConstructorTargetType replaces ByNameRef target types inside constructor
// strings. E.g., `..."Pages$Page")` → `..."Forms$Page")`.
func mapConstructorTargetType(ctor string) string {
	for from, to := range namespaceMap {
		ctor = strings.ReplaceAll(ctor, "\""+from+"$", "\""+to+"$")
	}
	return ctor
}

// reorderFields reorders td.Fields according to propNames (JS property name order).
// Fields not in propNames are appended after the ordered ones in their original order.
// Also re-assigns sequential FieldIndex values so Bind bit positions stay contiguous.
func reorderFields(td *TypeData, propNames []string) {
	idx := make(map[string]int, len(propNames))
	for i, n := range propNames {
		idx[n] = i
	}
	sort.SliceStable(td.Fields, func(i, j int) bool {
		pi, ok1 := idx[td.Fields[i].PropName]
		pj, ok2 := idx[td.Fields[j].PropName]
		if ok1 && ok2 {
			return pi < pj
		}
		return ok1 // ordered before unordered
	})
	for i := range td.Fields {
		td.Fields[i].FieldIndex = i
	}
}

// init validates that templates parse at startup so errors surface early.
func init() {
	for name, ts := range map[string]string{
		"types":       typesTemplate,
		"enums":       enumsTemplate,
		"version":     versionTemplate,
		"refs":        refsTemplate,
		"descriptors": descriptorsTemplate,
	} {
		if _, err := template.New(name).Parse(ts); err != nil {
			panic(fmt.Sprintf("emitter: template %q parse error: %v", name, err))
		}
	}
}
