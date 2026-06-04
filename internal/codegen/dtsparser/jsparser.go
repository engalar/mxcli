// Package dtsparser extracts Mendix metamodel metadata from the official
// TypeScript Model SDK's generated .d.ts and .js files.
package dtsparser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ────────────────────────────────────────────────────────────────
// Core types
// ────────────────────────────────────────────────────────────────

// PropKind classifies how a property relates to its owning element.
type PropKind string

const (
	PKPrimitive           PropKind = "Primitive"
	PKPrimitiveList       PropKind = "PrimitiveList"
	PKEnum                PropKind = "Enum"
	PKEnumList            PropKind = "EnumList"
	PKPart                PropKind = "Part"
	PKPartList            PropKind = "PartList"
	PKByNameRef           PropKind = "ByNameRef"
	PKByNameRefList       PropKind = "ByNameRefList"
	PKByIdRef             PropKind = "ByIdRef"
	PKStructuralChild     PropKind = "StructuralChild"
	PKStructuralChildList PropKind = "StructuralChildList"
	PKUnknown             PropKind = "Unknown"
)

// PrimitiveType is the specific primitive subtype.
type PrimitiveType string

const (
	PTString  PrimitiveType = "String"
	PTBoolean PrimitiveType = "Boolean"
	PTInteger PrimitiveType = "Integer"
	PTDouble  PrimitiveType = "Double"
	PTGuid    PrimitiveType = "Guid"
	PTPoint   PrimitiveType = "Point"
	PTSize    PrimitiveType = "Size"
	PTColor   PrimitiveType = "Color"
	PTBlob    PrimitiveType = "Blob"
)

// StructureKind classifies the element's position in the model tree.
type StructureKind string

const (
	SKElement        StructureKind = "Element"
	SKModelUnit      StructureKind = "ModelUnit"
	SKStructuralUnit StructureKind = "StructuralUnit"
	SKUnknown        StructureKind = "Unknown"
)

// JsClass represents a fully parsed class from a .js file.
type JsClass struct {
	Name              string
	StructureTypeName string         // e.g. "DomainModels$Entity"
	BaseClass         string         // extends ...
	StructureKind     StructureKind  // Element, ModelUnit, StructuralUnit
	IsAbstract        bool           // inferred from create method absence
	Properties        []JsProp       // all declared properties
	Defaults          []JsDefault    // from _initializeDefaultProperties
	VersionInfo       *JsVersionInfo // parsed versionInfo
	Containers        []string       // containerAs* type names (from .d.ts)
	FactoryMethods    []JsFactory    // createIn* methods (from .d.ts)
}

// JsProp is a property extracted from the .js constructor.
type JsProp struct {
	Name          string        // serialization key (e.g. "name", "generalization")
	Kind          PropKind      // classified from constructor call
	PrimitiveType PrimitiveType // only for PKPrimitive
	DefaultValue  string        // raw default value string
	TargetType    string        // for ByNameRef: "DomainModels$Entity"; for Enum: "ExportLevel"
	IsRequired    bool          // for Part: 5th arg is true
	// Version info (property-level)
	Introduced string
	Deleted    string
	Public     bool
	Required   bool
}

// JsDefault represents one default-setting statement from _initializeDefaultProperties.
type JsDefault struct {
	Property       string // property name
	DefaultExpr    string // raw expression (e.g. "NoGeneralization.create(this.model)")
	IsVersionGated bool   // wrapped in if (this.__prop.isAvailable)
}

// JsVersionInfo holds parsed StructureVersionInfo for a class.
type JsVersionInfo struct {
	Introduced    string
	Deleted       string
	DeletionMsg   string
	StructureKind StructureKind // from second param
	PropertyInfos map[string]*JsPropVersionInfo
}

// JsPropVersionInfo holds per-property version metadata.
type JsPropVersionInfo struct {
	Introduced  string
	Deleted     string
	DeletionMsg string
	PublicNow   bool
	RequiredNow bool
}

// JsFactory represents a factory method signature.
type JsFactory struct {
	Method        string // "createIn", "createInEntityUnderAccessRules", etc.
	ContainerType string // "Entity", "DomainModel", etc.
	PropertyName  string // "accessRules", "attributes", etc. (from Under* suffix)
	VersionMin    string // min version constraint (empty = none)
	VersionMax    string // max version constraint (empty = none)
}

// JsEnum represents an enum from .js.
type JsEnum struct {
	Name   string
	Values []JsEnumValue
}

// JsEnumValue represents a single enum literal.
type JsEnumValue struct {
	Name       string
	Introduced string
	Deleted    string
}

// DomainMeta holds the complete extracted metadata for one domain.
type DomainMeta struct {
	Namespace      string
	Classes        []JsClass
	Enums          []JsEnum
	Interfaces     []JsInterface     // from .d.ts
	StorageAliases map[string]string // JS $Type → BSON $Type (e.g. "DomainModels$Entity" → "DomainModels$EntityImpl")

	// PropertyKeyOverrides maps "TypeName.propertyName" → BSON key for cases
	// where the BSON storage key differs from PascalCase(JS property name).
	PropertyKeyOverrides map[string]string

	// PropertyOrderOverrides maps class StructureTypeName → ordered list of
	// property names (JS names). When set, the emitter reorders the struct's
	// fields to match Mendix's serialization order (which may differ from the
	// TypeScript SDK definition order).
	PropertyOrderOverrides map[string][]string

	// RefListVersion3Fields is a set of "ClassName.propertyName" pairs whose
	// ByNameRefList fields require BSON version marker int32(3) instead of the
	// default int32(1). Mendix uses version 3 for AllowedRoles on Forms$Page.
	RefListVersion3Fields map[string]bool

	// PartListVersion2Fields is a set of "ClassName.propertyName" pairs whose
	// PartList fields require BSON version marker int32(2) instead of the
	// default int32(3). Mendix uses version 2 for Parameters/TypeParameters on
	// JavaActions$JavaAction; using version 3 causes MprTool to crash.
	PartListVersion2Fields map[string]bool

	EdgeKindOverrides map[string]string // TargetType → edge kind hint
	IdRefScope        map[string]string // "ClassName.propName" → "cross-unit" or "intra-unit"

	// BinaryUUIDProps is a set of "ClassName.propertyName" (or "*.propertyName")
	// pairs whose Primitive[string] fields should use BinaryUUIDPrimitive instead.
	// Used for PersistentId-style fields that Studio Pro serializes as BSON Binary
	// but the TypeScript SDK types as Guid/string.
	BinaryUUIDProps map[string]bool

	// CrossDomainProps maps class names from OTHER domains (e.g. "Document",
	// "ModuleDocument") to their properties. Used by the emitter to resolve
	// cross-domain inheritance (e.g. Workflow extends projects.Document).
	CrossDomainProps map[string][]JsProp

	// TypeRenames holds validated type rename entries for this domain.
	// Populated by the codegen main() after loading supplements.json.
	// Each entry describes a BSON $Type rename introduced at a specific version.
	TypeRenames []TypeRenameData
}

// TypeRenameData describes a versioned BSON type rename.
// Populated from supplements.json type_renames section by the codegen main().
type TypeRenameData struct {
	OldTypeName  string // e.g. "Workflows$CallMicroflowTask"
	NewTypeName  string // e.g. "Workflows$CallMicroflowActivity"
	Since        string // e.g. "11.9.0"
	OldGoName    string // e.g. "CallMicroflowTask"
	NewGoName    string // e.g. "CallMicroflowActivity"
	FuncBaseName string // e.g. "CallMicroflow" — longest common prefix of OldGoName/NewGoName
}

// JsInterface from .d.ts — shows which properties are part of the public API.
type JsInterface struct {
	Name       string
	Extends    []string // interface inheritance chain
	Properties []string // readonly property names
}

// ────────────────────────────────────────────────────────────────
// Regex patterns for .js parsing
// ────────────────────────────────────────────────────────────────

var (
	// Class definition: class Foo extends Bar {
	rjClass = regexp.MustCompile(`class (\w+) extends (\S+) \{`)

	// Property initialization in constructor: this.__xxx = new internal.PropertyType(...)
	// Note: args can contain nested braces like { x: 0, y: 0 }, so we match lazily
	// and extract args manually.
	rjPropStart = regexp.MustCompile(`this\.__(\w+)\s*=\s*new internal\.(\w+)\(`)

	// structureTypeName: Foo.structureTypeName = "DomainModels$Foo";
	rjStructType = regexp.MustCompile(`(\w+)\.structureTypeName\s*=\s*"([^"]+)"`)

	// versionInfo assignment start: Foo.versionInfo = new exports.StructureVersionInfo({
	rjVersionInfo = regexp.MustCompile(`(\w+)\.versionInfo\s*=\s*new exports\.StructureVersionInfo\(\{`)

	// StructureType parameter: }, internal.StructureType.Element);
	rjStructureType = regexp.MustCompile(`internal\.StructureType\.(\w+)\)`)

	// _initializeDefaultProperties method
	rjInitDefaults = regexp.MustCompile(`_initializeDefaultProperties\(\)`)

	// Version-gated default: if (this.__propName.isAvailable) {
	rjVersionGate = regexp.MustCompile(`if \(this\.__(\w+)\.isAvailable\)`)

	// Simple default assignment: this.propName = value;
	rjDefaultAssign = regexp.MustCompile(`^\s+this\.(\w+)\s*=\s*(.+);`)

	// PrimitiveTypeEnum value
	rjPrimitiveEnum = regexp.MustCompile(`internal\.PrimitiveTypeEnum\.(\w+)`)

	// Enum default: EnumType.Value
	rjEnumDefault = regexp.MustCompile(`(\w+(?:\.\w+)*)\.(\w+)$`)

	// versionInfo property intro/del
	rjViIntro  = regexp.MustCompile(`introduced:\s*"([^"]+)"`)
	rjViDel    = regexp.MustCompile(`deleted:\s*"([^"]+)"`)
	rjViDelMsg = regexp.MustCompile(`deletionMessage:\s*"([^"]+)"`)

	// versionInfo property public/required currentValue
	rjViPublic   = regexp.MustCompile(`public:\s*\{\s*currentValue:\s*(true|false)`)
	rjViRequired = regexp.MustCompile(`required:\s*\{\s*currentValue:\s*(true|false)`)

	// Namespace extraction: exports.domainmodels = ...
	rjNamespace = regexp.MustCompile(`exports\.(\w+)\s*=`)

	// Enum class pattern: class Foo extends internal.AbstractEnum {
	// Matches both "internal.AbstractEnum" and "internal_1.internal.AbstractEnum"
	rjEnumClass = regexp.MustCompile(`class (\w+) extends (?:internal_\d+\.)?internal\.AbstractEnum`)

	// Enum value: Foo.Bar = new Foo("Bar", { ... });
	rjEnumValue = regexp.MustCompile(`(\w+)\.(\w+)\s*=\s*new \w+\("(\w+)"`)

	// Enum value lifecycle: new internal.LifeCycle("x.y.z", null)
	rjLifeCycle = regexp.MustCompile(`new internal(?:_\d+)?\.LifeCycle\("([^"]*)",\s*"?([^")]*)"?\)`)
)

// ────────────────────────────────────────────────────────────────
// .js Parser
// ────────────────────────────────────────────────────────────────

// ParseJsFile extracts all metadata from a single .js domain file.
func ParseJsFile(path string) (*DomainMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	meta := &DomainMeta{}

	// Extract namespace
	if m := rjNamespace.FindStringSubmatch(content); m != nil {
		meta.Namespace = m[1]
	}

	// Pass 1: structureTypeNames
	stnMap := map[string]string{}
	for _, m := range rjStructType.FindAllStringSubmatch(content, -1) {
		stnMap[m[1]] = m[2]
	}

	// Pass 2: Parse classes with constructors and properties
	// Use index into slice instead of pointer (slice may grow and reallocate)
	classIdxMap := map[string]int{} // name → index in meta.Classes
	currentClassIdx := -1
	inConstructor := false
	braceDepth := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Detect class start
		if m := rjClass.FindStringSubmatch(line); m != nil {
			cls := JsClass{
				Name:      m[1],
				BaseClass: m[2],
			}
			if stn, ok := stnMap[m[1]]; ok {
				cls.StructureTypeName = stn
			}
			meta.Classes = append(meta.Classes, cls)
			currentClassIdx = len(meta.Classes) - 1
			classIdxMap[m[1]] = currentClassIdx
			continue
		}

		// Detect constructor start
		if currentClassIdx >= 0 && strings.Contains(line, "constructor(") {
			inConstructor = true
			// Count braces on the constructor line itself (includes the opening {)
			braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}

		// Parse constructor body
		if inConstructor && currentClassIdx >= 0 {
			// Track brace depth
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth <= 0 {
				inConstructor = false
				continue
			}

			if m := rjPropStart.FindStringSubmatchIndex(line); m != nil {
				propName := line[m[2]:m[3]]
				propType := line[m[4]:m[5]]
				// Extract balanced args from after the opening paren
				argsStart := m[1] // end of match = right after "("
				args := extractBalancedArgs(line[argsStart:])
				prop := parsePropertyInit(propName, propType, args)
				meta.Classes[currentClassIdx].Properties = append(meta.Classes[currentClassIdx].Properties, prop)
			}
		}
	}

	// Rebuild stable classMap after pass 2 (slice is done growing)
	classMap := map[string]*JsClass{}
	for i := range meta.Classes {
		classMap[meta.Classes[i].Name] = &meta.Classes[i]
	}

	// Pass 3: Parse _initializeDefaultProperties
	// Track which class scope we're in using a simple stack approach
	currentClassName := ""
	for i := 0; i < len(lines); i++ {
		// Track class scope
		if m := rjClass.FindStringSubmatch(lines[i]); m != nil {
			currentClassName = m[1]
		}

		if !rjInitDefaults.MatchString(lines[i]) {
			continue
		}

		cls := classMap[currentClassName]
		if cls == nil {
			continue
		}

		// Parse the method body — starts at opening brace
		depth := strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
		versionGatedProp := ""
		for j := i + 1; j < len(lines); j++ {
			l := lines[j]
			depth += strings.Count(l, "{") - strings.Count(l, "}")

			if depth <= 0 {
				break
			}

			if m := rjVersionGate.FindStringSubmatch(l); m != nil {
				versionGatedProp = m[1]
				continue
			}

			if m := rjDefaultAssign.FindStringSubmatch(l); m != nil {
				propName := m[1]
				expr := strings.TrimSpace(m[2])
				// Skip "super._initializeDefaultProperties()"
				if propName == "super" {
					continue
				}
				gated := versionGatedProp != ""
				cls.Defaults = append(cls.Defaults, JsDefault{
					Property:       propName,
					DefaultExpr:    expr,
					IsVersionGated: gated,
				})
				if gated {
					versionGatedProp = ""
				}
			}
		}
	}

	// Pass 4: Parse versionInfo
	// Pattern: "    Entity.versionInfo = new exports.StructureVersionInfo({"
	rjVI2 := regexp.MustCompile(`(\w+)\.versionInfo\s*=\s*new exports\.StructureVersionInfo\(`)
	for i := 0; i < len(lines); i++ {
		m := rjVI2.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		className := m[1]
		cls := classMap[className]
		if cls == nil {
			continue
		}

		// Collect the full versionInfo block (find matching close)
		block := collectBracedBlock(lines, i)
		vi := parseVersionInfoBlock(block)

		// Find structure type from the block itself
		if sm := rjStructureType.FindStringSubmatch(block); sm != nil {
			switch sm[1] {
			case "Element":
				vi.StructureKind = SKElement
			case "ModelUnit":
				vi.StructureKind = SKModelUnit
			case "StructuralUnit":
				vi.StructureKind = SKStructuralUnit
			}
		}

		cls.VersionInfo = vi
		cls.StructureKind = vi.StructureKind
	}

	// Pass 5: Determine abstract status
	// Concrete classes have factory methods: "static create(model)" or "static createIn(container)".
	// Abstract classes have neither.
	reStaticFactory := regexp.MustCompile(`static create(In)?\(`)
	for i := range meta.Classes {
		cls := &meta.Classes[i]
		if cls.StructureTypeName == "" {
			continue // no $Type = helper class, not abstract/concrete distinction
		}
		// Find the class body extent and check for factory methods within it
		classPattern := "class " + cls.Name + " extends "
		classStart := strings.Index(content, classPattern)
		if classStart < 0 {
			cls.IsAbstract = true
			continue
		}
		// Find end of class (next class def at same indent, or next ClassName.structureTypeName after the class)
		stnPattern := "\n    " + cls.Name + ".structureTypeName"
		stnIdx := strings.Index(content[classStart:], stnPattern)
		var classBody string
		if stnIdx > 0 {
			classBody = content[classStart : classStart+stnIdx]
		} else {
			nextClass := strings.Index(content[classStart+len(classPattern):], "\n    class ")
			if nextClass > 0 {
				classBody = content[classStart : classStart+len(classPattern)+nextClass]
			} else {
				classBody = content[classStart:]
			}
		}
		cls.IsAbstract = !reStaticFactory.MatchString(classBody)
	}

	// Pass 6: Parse enum classes and their values
	{
		var currentEnum *JsEnum
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			if m := rjEnumClass.FindStringSubmatch(line); m != nil {
				meta.Enums = append(meta.Enums, JsEnum{Name: m[1]})
				currentEnum = &meta.Enums[len(meta.Enums)-1]
				continue
			}
			// End of enum class scope: next class definition
			if currentEnum != nil && rjClass.MatchString(line) {
				currentEnum = nil
			}
			// Enum value: ClassName.Value = new ClassName("Value", ...)
			if currentEnum != nil {
				if m := rjEnumValue.FindStringSubmatch(line); m != nil {
					className := m[1]
					valueName := m[3]
					if className == currentEnum.Name {
						ev := JsEnumValue{Name: valueName}
						// Try to parse lifecycle info from the same line
						if lc := rjLifeCycle.FindStringSubmatch(line); lc != nil {
							ev.Introduced = lc[1]
							if lc[2] != "null" && lc[2] != "" {
								ev.Deleted = lc[2]
							}
						}
						currentEnum.Values = append(currentEnum.Values, ev)
					}
				}
			}
		}
	}

	return meta, nil
}

func parsePropertyInit(name, propType, argsStr string) JsProp {
	p := JsProp{Name: name}

	switch propType {
	case "PrimitiveProperty":
		p.Kind = PKPrimitive
		if m := rjPrimitiveEnum.FindStringSubmatch(argsStr); m != nil {
			p.PrimitiveType = PrimitiveType(m[1])
		}
		p.DefaultValue = extractDefaultArg(argsStr, 3)

	case "PrimitiveListProperty":
		p.Kind = PKPrimitiveList
		p.DefaultValue = "[]"

	case "EnumProperty":
		p.Kind = PKEnum
		// args: Class, this, "name", EnumType.Default, EnumType
		p.DefaultValue = extractDefaultArg(argsStr, 3)
		p.TargetType = extractArgTrimmed(argsStr, 4)

	case "EnumListProperty":
		p.Kind = PKEnumList
		p.DefaultValue = "[]"

	case "PartProperty":
		p.Kind = PKPart
		p.DefaultValue = extractDefaultArg(argsStr, 3)
		lastArg := extractArgTrimmed(argsStr, 4)
		p.IsRequired = lastArg == "true"

	case "PartListProperty":
		p.Kind = PKPartList
		p.DefaultValue = "[]"

	case "ByNameReferenceProperty":
		p.Kind = PKByNameRef
		p.TargetType = extractQuotedArg(argsStr, 4)

	case "ByNameReferenceListProperty":
		p.Kind = PKByNameRefList
		p.TargetType = extractQuotedArg(argsStr, 4)

	case "ByIdReferenceProperty":
		p.Kind = PKByIdRef

	case "StructuralChildProperty":
		p.Kind = PKStructuralChild

	case "StructuralChildListProperty":
		p.Kind = PKStructuralChildList

	case "LocalByNameReferenceProperty":
		// Local by-name references (intra-unit) behave like ByNameRef
		// for serialization: stored as a string qualified name in BSON.
		p.Kind = PKByNameRef
		p.TargetType = extractQuotedArg(argsStr, 4)

	default:
		p.Kind = PKUnknown
	}

	return p
}

// extractDefaultArg extracts the nth (0-based) comma-separated argument.
func extractDefaultArg(args string, n int) string {
	return extractArgTrimmed(args, n)
}

// extractArgTrimmed splits args by comma (respecting nesting) and returns nth arg trimmed.
func extractArgTrimmed(args string, n int) string {
	parts := splitArgs(args)
	if n < len(parts) {
		return strings.TrimSpace(parts[n])
	}
	return ""
}

// extractQuotedArg extracts a quoted string from nth argument.
func extractQuotedArg(args string, n int) string {
	val := extractArgTrimmed(args, n)
	val = strings.Trim(val, "\"' ")
	return val
}

// extractBalancedArgs extracts everything up to the matching closing paren.
// Input starts right after the opening "(".
func extractBalancedArgs(s string) string {
	depth := 1
	for i, ch := range s {
		switch ch {
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
			if depth == 0 {
				return s[:i]
			}
		}
	}
	return s
}

// splitArgs splits comma-separated args respecting parentheses and brackets.
func splitArgs(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, ch := range s {
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// findEnclosingClass looks backward from line i to find the class name.
func findEnclosingClass(lines []string, i int) string {
	for j := i - 1; j >= 0 && j > i-200; j-- {
		if m := rjClass.FindStringSubmatch(lines[j]); m != nil {
			return m[1]
		}
	}
	return ""
}

// collectBracedBlock collects text from a starting line until braces balance.
func collectBracedBlock(lines []string, start int) string {
	var buf strings.Builder
	depth := 0
	for i := start; i < len(lines); i++ {
		buf.WriteString(lines[i])
		buf.WriteString("\n")
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
		if depth <= 0 && i > start {
			break
		}
	}
	return buf.String()
}

// parseVersionInfoBlock extracts version metadata from a StructureVersionInfo block.
// The block is a JS object literal like: ClassName.versionInfo = new exports.StructureVersionInfo({ properties: { name: { ... }, ... }, public: { ... } }, internal.StructureType.Element);
func parseVersionInfoBlock(block string) *JsVersionInfo {
	vi := &JsVersionInfo{
		PropertyInfos: map[string]*JsPropVersionInfo{},
	}

	// Extract top-level introduced/deleted from the block before the properties: section.
	// These appear directly inside the StructureVersionInfo({...}) object literal.
	topLevel := block
	if idx := strings.Index(block, "properties:"); idx >= 0 {
		topLevel = block[:idx]
	}
	if m := rjViIntro.FindStringSubmatch(topLevel); m != nil {
		vi.Introduced = m[1]
	}
	if m := rjViDel.FindStringSubmatch(topLevel); m != nil {
		vi.Deleted = m[1]
	}
	if m := rjViDelMsg.FindStringSubmatch(topLevel); m != nil {
		vi.DeletionMsg = m[1]
	}

	lines := strings.Split(block, "\n")

	// State machine: track when we're inside "properties: { ... }"
	// and parse each property section individually.
	type state int
	const (
		stOutside state = iota
		stInProperties
		stInPropertyBlock
	)

	st := stOutside
	propBraceDepth := 0
	currentPropName := ""
	var currentPropLines []string

	rePropKey := regexp.MustCompile(`^\s+(\w+):\s*\{`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch st {
		case stOutside:
			if strings.Contains(line, "properties:") && strings.Contains(line, "{") {
				st = stInProperties
				propBraceDepth = 1
			}

		case stInProperties:
			// Are we done with properties block?
			propBraceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if propBraceDepth <= 0 {
				st = stOutside
				continue
			}

			// Start of a new property?
			if m := rePropKey.FindStringSubmatch(line); m != nil {
				name := m[1]
				// Skip nested public/required/experimental blocks (they're inside property blocks)
				if name == "public" || name == "required" || name == "experimental" || name == "changedIn" {
					continue
				}
				if currentPropName != "" {
					// Finalize previous property
					vi.PropertyInfos[currentPropName] = parsePropertyVersionLines(currentPropLines)
				}
				currentPropName = name
				currentPropLines = []string{line}
				st = stInPropertyBlock
			}

		case stInPropertyBlock:
			currentPropLines = append(currentPropLines, line)
			// Check if this property block closed. We count braces relative to start.
			braces := 0
			for _, pl := range currentPropLines {
				braces += strings.Count(pl, "{") - strings.Count(pl, "}")
			}
			if braces <= 0 {
				vi.PropertyInfos[currentPropName] = parsePropertyVersionLines(currentPropLines)
				currentPropName = ""
				currentPropLines = nil
				st = stInProperties
			}
		}

		_ = trimmed
	}

	// Handle last property
	if currentPropName != "" {
		vi.PropertyInfos[currentPropName] = parsePropertyVersionLines(currentPropLines)
	}

	return vi
}

// parsePropertyVersionLines extracts version info from a property's lines.
func parsePropertyVersionLines(lines []string) *JsPropVersionInfo {
	pvi := &JsPropVersionInfo{}
	joined := strings.Join(lines, "\n")
	if m := rjViIntro.FindStringSubmatch(joined); m != nil {
		pvi.Introduced = m[1]
	}
	if m := rjViDel.FindStringSubmatch(joined); m != nil {
		pvi.Deleted = m[1]
	}
	if m := rjViDelMsg.FindStringSubmatch(joined); m != nil {
		pvi.DeletionMsg = m[1]
	}
	if m := rjViPublic.FindStringSubmatch(joined); m != nil {
		pvi.PublicNow = m[1] == "true"
	}
	if m := rjViRequired.FindStringSubmatch(joined); m != nil {
		pvi.RequiredNow = m[1] == "true"
	}
	return pvi
}

// ────────────────────────────────────────────────────────────────
// Multi-domain scanning
// ────────────────────────────────────────────────────────────────

// ParseAllDomains parses all .js files in the gen/ directory.
func ParseAllDomains(genDir string) ([]*DomainMeta, error) {
	entries, err := os.ReadDir(genDir)
	if err != nil {
		return nil, err
	}
	var results []*DomainMeta
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".js") || strings.HasSuffix(entry.Name(), ".js.map") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".js")
		if name == "base-model" || name == "all-model-classes" {
			continue
		}
		meta, err := ParseJsFile(filepath.Join(genDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		results = append(results, meta)
	}
	return results, nil
}
