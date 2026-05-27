package dtsparser

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// PropertyKind classifies how a property relates to its parent.
type PropertyKind int

const (
	KindPrimitive PropertyKind = iota // string, bool, number, point, etc.
	KindEnum                          // enumeration value
	KindPart                          // single contained element (1:1)
	KindPartList                      // list of contained elements (1:N)
	KindByNameRef                     // cross-unit reference by qualified name
	KindByIdRef                       // same-unit reference by ID
	KindUnknown
)

func (k PropertyKind) String() string {
	switch k {
	case KindPrimitive:
		return "Primitive"
	case KindEnum:
		return "Enum"
	case KindPart:
		return "Part"
	case KindPartList:
		return "PartList"
	case KindByNameRef:
		return "ByNameRef"
	case KindByIdRef:
		return "ByIdRef"
	default:
		return "Unknown"
	}
}

// DtsClass represents a class extracted from a .d.ts file.
type DtsClass struct {
	Name       string
	Base       string // extends ...
	Implements string // implements ...
	Abstract   bool
	Properties []DtsProperty
	Containers []string // containerAs* types
	Factories  []DtsFactory
	Comments   string // JSDoc before class
}

// DtsProperty represents a property (getter/setter pair) in a .d.ts class.
type DtsProperty struct {
	Name       string
	TypeStr    string       // raw TS type string
	Kind       PropertyKind // inferred kind
	HasSetter  bool
	Introduced string // from JSDoc "In version X.Y.Z: introduced"
	Deleted    string // from JSDoc "In version X.Y.Z: deleted"
	Nullable   bool   // type includes "| null"
}

// DtsFactory represents a static factory method.
type DtsFactory struct {
	MethodName    string
	ContainerType string
	ReturnType    string
}

// DtsEnum represents an enum class.
type DtsEnum struct {
	Name   string
	Values []string
}

var (
	// Class definition: class Foo extends Bar implements IFoo {
	reClass = regexp.MustCompile(`^\s+(abstract )?class (\w+) extends (\S+?)(?:<[^>]+>)? (?:implements (\w+) )?{`)
	// Getter: get name(): Type;
	reGetter = regexp.MustCompile(`^\s+get (\w+)\(\): (.+);`)
	// Setter: set name(newValue: Type);
	reSetter = regexp.MustCompile(`^\s+set (\w+)\(`)
	// ContainerAs: get containerAsFoo(): Foo;
	reContainer = regexp.MustCompile(`^\s+get containerAs(\w+)\(\): `)
	// Factory: static createIn(container: Type): ReturnType;
	reFactory = regexp.MustCompile(`^\s+static (create\w*)\((?:container: (\w+)|model: \w+)\): (\w+);`)
	// Version comment: * In version X.Y.Z: introduced/deleted
	reVersion = regexp.MustCompile(`In version ([\d.]+): (\w+)`)
	// Enum class: class Foo extends internal.AbstractEnum {
	reEnum = regexp.MustCompile(`^\s+class (\w+) extends internal\.AbstractEnum`)
	// Enum value: static Foo: EnumType;
	reEnumValue = regexp.MustCompile(`^\s+static (\w+): (\w+);`)
	// structureTypeName assignment (from .js): Foo.structureTypeName = "DomainModels$Foo";
	reStructType = regexp.MustCompile(`(\w+)\.structureTypeName\s*=\s*"([^"]+)"`) // relaxed: no anchor
	// IList type
	reIList = regexp.MustCompile(`internal\.IList<(.+)>`)
	// QualifiedName companion
	reQualifiedName = regexp.MustCompile(`(\w+)QualifiedName`)
)

// collectCrossModuleEnums scans all .d.ts files in the gen/ directory for enum definitions.
func collectCrossModuleEnums(genDir string) map[string]bool {
	enums := map[string]bool{}
	entries, err := os.ReadDir(genDir)
	if err != nil {
		return enums
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".d.ts") {
			continue
		}
		data, err := os.ReadFile(genDir + "/" + entry.Name())
		if err != nil {
			continue
		}
		// Extract namespace name from "export declare namespace <name> {"
		nsName := ""
		reNs := regexp.MustCompile(`export declare namespace (\w+)`)
		if m := reNs.FindStringSubmatch(string(data)); m != nil {
			nsName = m[1]
		}

		for _, line := range strings.Split(string(data), "\n") {
			if m := reEnum.FindStringSubmatch(line); m != nil {
				enums[m[1]] = true
				if nsName != "" {
					enums[nsName+"."+m[1]] = true
				}
			}
		}
	}
	return enums
}

// inferPropertyKind infers the property kind from the TS type string and context.
func inferPropertyKind(name, typeStr string, hasSetter bool, hasQualifiedNameCompanion bool, knownEnums map[string]bool) PropertyKind {
	// Skip derived/accessor properties
	if strings.HasPrefix(name, "containerAs") {
		return KindUnknown
	}
	if strings.HasSuffix(name, "QualifiedName") || strings.HasSuffix(name, "QualifiedNames") {
		return KindUnknown // companion property, not a real stored property
	}

	// IList<T> → PartList (or ByNameRefList if T is an interface from another domain)
	if m := reIList.FindStringSubmatch(typeStr); m != nil {
		innerType := m[1]
		// If inner type starts with I and is from another module (has dot), it's a ByNameRef list
		if strings.Contains(innerType, ".I") || (strings.HasPrefix(innerType, "I") && len(innerType) > 1 && innerType[1] >= 'A' && innerType[1] <= 'Z') {
			// Check if there's a companion QualifiedNames property
			if hasQualifiedNameCompanion {
				return KindByNameRef // actually ByNameRefList
			}
		}
		return KindPartList
	}

	// Has companion "fooQualifiedName" → ByNameRef
	if hasQualifiedNameCompanion {
		return KindByNameRef
	}

	// Enum types (check if type is a known enum)
	cleanType := strings.TrimSuffix(typeStr, " | null")
	cleanType = strings.TrimSpace(cleanType)
	if knownEnums[cleanType] {
		return KindEnum
	}
	// Enums from other modules: projects.ExportLevel etc.
	if dotIdx := strings.LastIndex(cleanType, "."); dotIdx >= 0 {
		shortName := cleanType[dotIdx+1:]
		if knownEnums[shortName] {
			return KindEnum
		}
	}

	// Primitive types
	switch cleanType {
	case "string", "boolean", "number":
		return KindPrimitive
	case "string[]", "boolean[]", "number[]":
		return KindPrimitive
	}
	if strings.HasSuffix(cleanType, ".IPoint") || cleanType == "common.IPoint" {
		return KindPrimitive // Point is stored as primitive JSON string
	}
	if strings.HasSuffix(cleanType, ".ISize") || cleanType == "common.ISize" {
		return KindPrimitive
	}

	// Remaining: if type is a class/interface (capitalized), it's a Part
	if len(cleanType) > 0 && cleanType[0] >= 'A' && cleanType[0] <= 'Z' {
		return KindPart
	}
	if strings.Contains(cleanType, ".") {
		return KindPart
	}

	return KindUnknown
}

// parseDtsFile parses a .d.ts file and extracts classes, enums, and their metadata.
func parseDtsFile(content string) (classes []DtsClass, enums []DtsEnum) {
	return parseDtsFileWithEnums(content, nil)
}

// parseDtsFileWithEnums parses a .d.ts file with externally-provided enum knowledge.
func parseDtsFileWithEnums(content string, externalEnums map[string]bool) (classes []DtsClass, enums []DtsEnum) {
	lines := strings.Split(content, "\n")

	// First pass: collect enum names (local + external)
	knownEnums := map[string]bool{}
	for k, v := range externalEnums {
		knownEnums[k] = v
	}
	for _, line := range lines {
		if m := reEnum.FindStringSubmatch(line); m != nil {
			knownEnums[m[1]] = true
		}
	}

	// Second pass: parse enums
	var currentEnum *DtsEnum
	for _, line := range lines {
		if m := reEnum.FindStringSubmatch(line); m != nil {
			if currentEnum != nil {
				enums = append(enums, *currentEnum)
			}
			currentEnum = &DtsEnum{Name: m[1]}
			continue
		}
		if currentEnum != nil {
			if strings.TrimSpace(line) == "}" {
				enums = append(enums, *currentEnum)
				currentEnum = nil
				continue
			}
			if m := reEnumValue.FindStringSubmatch(line); m != nil {
				if m[1] != "qualifiedTsTypeName" { // skip internal field
					currentEnum.Values = append(currentEnum.Values, m[1])
				}
			}
		}
	}

	// Third pass: parse classes
	var current *DtsClass
	var commentBuf strings.Builder
	// Track all property names to identify QualifiedName companions
	var propNames []string
	for _, line := range lines {
		// Accumulate comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/**") || strings.HasPrefix(trimmed, "*/") {
			commentBuf.WriteString(line)
			commentBuf.WriteString("\n")
			continue
		}

		if m := reClass.FindStringSubmatch(line); m != nil {
			// Save previous class
			if current != nil {
				// Now infer property kinds with QualifiedName companion awareness
				qnSet := map[string]bool{}
				for _, p := range current.Properties {
					if strings.HasSuffix(p.Name, "QualifiedName") {
						base := strings.TrimSuffix(p.Name, "QualifiedName")
						qnSet[base] = true
					}
					if strings.HasSuffix(p.Name, "QualifiedNames") {
						base := strings.TrimSuffix(p.Name, "QualifiedNames")
						qnSet[base] = true
					}
				}
				for i := range current.Properties {
					p := &current.Properties[i]
					p.Kind = inferPropertyKind(p.Name, p.TypeStr, p.HasSetter, qnSet[p.Name], knownEnums)
				}
				classes = append(classes, *current)
			}
			current = &DtsClass{
				Name:       m[2],
				Base:       m[3],
				Implements: m[4],
				Abstract:   m[1] != "",
				Comments:   commentBuf.String(),
			}
			commentBuf.Reset()
			propNames = nil
			continue
		}

		if current == nil {
			commentBuf.Reset()
			continue
		}

		// Parse container accessors
		if m := reContainer.FindStringSubmatch(line); m != nil {
			current.Containers = append(current.Containers, m[1])
			commentBuf.Reset()
			continue
		}

		// Parse factory methods
		if m := reFactory.FindStringSubmatch(line); m != nil {
			current.Factories = append(current.Factories, DtsFactory{
				MethodName:    m[1],
				ContainerType: m[2],
				ReturnType:    m[3],
			})
			commentBuf.Reset()
			continue
		}

		// Parse setters (mark existing properties as having setters)
		if m := reSetter.FindStringSubmatch(line); m != nil {
			for i := range current.Properties {
				if current.Properties[i].Name == m[1] {
					current.Properties[i].HasSetter = true
					break
				}
			}
			commentBuf.Reset()
			continue
		}

		// Parse getters
		if m := reGetter.FindStringSubmatch(line); m != nil {
			propName := m[1]
			propType := m[2]

			// Extract version info from accumulated comments
			comment := commentBuf.String()
			var introduced, deleted string
			for _, vm := range reVersion.FindAllStringSubmatch(comment, -1) {
				switch vm[2] {
				case "introduced":
					introduced = vm[1]
				case "deleted":
					deleted = vm[1]
				}
			}

			nullable := strings.Contains(propType, "| null")

			prop := DtsProperty{
				Name:       propName,
				TypeStr:    propType,
				Introduced: introduced,
				Deleted:    deleted,
				Nullable:   nullable,
			}
			current.Properties = append(current.Properties, prop)
			propNames = append(propNames, propName)
			commentBuf.Reset()
			continue
		}

		// Closing brace
		if trimmed == "}" && current != nil {
			// could be end of class — but nested braces make this tricky
			// We rely on next class/enum/interface definition to finalize
		}

		commentBuf.Reset()
	}

	// Finalize last class
	if current != nil {
		qnSet := map[string]bool{}
		for _, p := range current.Properties {
			if strings.HasSuffix(p.Name, "QualifiedName") {
				base := strings.TrimSuffix(p.Name, "QualifiedName")
				qnSet[base] = true
			}
			if strings.HasSuffix(p.Name, "QualifiedNames") {
				base := strings.TrimSuffix(p.Name, "QualifiedNames")
				qnSet[base] = true
			}
		}
		for i := range current.Properties {
			p := &current.Properties[i]
			p.Kind = inferPropertyKind(p.Name, p.TypeStr, p.HasSetter, qnSet[p.Name], knownEnums)
		}
		classes = append(classes, *current)
	}

	_ = propNames
	return
}

// parseStructureTypeNames extracts structureTypeName assignments from a .js file.
func parseStructureTypeNames(jsContent string) map[string]string {
	result := map[string]string{}
	for _, m := range reStructType.FindAllStringSubmatch(jsContent, -1) {
		result[m[1]] = m[2]
	}
	return result
}

func TestParseDomainModelsDts(t *testing.T) {
	genDir := findMendixModelSDKGenDir(t)
	dtsPath := genDir + "/domainmodels.d.ts"
	jsPath := genDir + "/domainmodels.js"

	dtsData, err := os.ReadFile(dtsPath)
	if err != nil {
		t.Fatalf("cannot read .d.ts: %v", err)
	}
	jsData, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("cannot read .js: %v", err)
	}

	// Collect enums from all modules for cross-module enum detection
	allEnums := collectCrossModuleEnums(genDir)

	classes, enums := parseDtsFileWithEnums(string(dtsData), allEnums)
	structTypeNames := parseStructureTypeNames(string(jsData))

	// ==================== Enum Verification ====================
	t.Run("Enums", func(t *testing.T) {
		if len(enums) == 0 {
			t.Fatal("expected to find enums")
		}
		t.Logf("Found %d enums:", len(enums))
		for _, e := range enums {
			t.Logf("  %s: %v", e.Name, e.Values)
		}

		// Verify specific enum
		found := false
		for _, e := range enums {
			if e.Name == "AssociationType" {
				found = true
				if len(e.Values) != 2 {
					t.Errorf("AssociationType should have 2 values, got %d", len(e.Values))
				}
				for _, v := range e.Values {
					if v != "Reference" && v != "ReferenceSet" {
						t.Errorf("unexpected AssociationType value: %s", v)
					}
				}
			}
		}
		if !found {
			t.Error("AssociationType enum not found")
		}
	})

	// ==================== Class Verification ====================
	t.Run("Classes", func(t *testing.T) {
		if len(classes) == 0 {
			t.Fatal("expected to find classes")
		}
		t.Logf("Found %d classes", len(classes))

		// Build class index
		classIdx := map[string]*DtsClass{}
		for i := range classes {
			classIdx[classes[i].Name] = &classes[i]
		}

		// Verify Entity class
		entity := classIdx["Entity"]
		if entity == nil {
			t.Fatal("Entity class not found")
		}
		t.Logf("Entity class: extends=%s, implements=%s, abstract=%v", entity.Base, entity.Implements, entity.Abstract)
		t.Logf("  Containers: %v", entity.Containers)
		t.Logf("  Factories: %d", len(entity.Factories))
		for _, f := range entity.Factories {
			t.Logf("    %s(container: %s): %s", f.MethodName, f.ContainerType, f.ReturnType)
		}

		// Verify Entity properties
		t.Logf("  Properties (%d):", len(entity.Properties))
		for _, p := range entity.Properties {
			t.Logf("    %-30s %-40s Kind=%-10s Setter=%v Intro=%s Del=%s Null=%v",
				p.Name, p.TypeStr, p.Kind, p.HasSetter, p.Introduced, p.Deleted, p.Nullable)
		}

		// Check specific property kinds
		propIdx := map[string]*DtsProperty{}
		for i := range entity.Properties {
			propIdx[entity.Properties[i].Name] = &entity.Properties[i]
		}

		checks := []struct {
			name string
			kind PropertyKind
		}{
			{"name", KindPrimitive},
			{"documentation", KindPrimitive},
			{"location", KindPrimitive},       // common.IPoint → Primitive
			{"generalization", KindPart},      // GeneralizationBase → Part
			{"attributes", KindPartList},      // IList<Attribute> → PartList
			{"validationRules", KindPartList}, // IList<ValidationRule> → PartList
			{"indexes", KindPartList},         // IList<Index> → PartList
			{"accessRules", KindPartList},     // IList<AccessRule> → PartList
			{"image", KindByNameRef},          // IImage with imageQualifiedName → ByNameRef
			{"exportLevel", KindEnum},         // projects.ExportLevel → Enum (need cross-domain enum detection)
			{"source", KindPart},              // EntitySource → Part
			{"capabilities", KindPart},        // EntityCapabilities → Part
		}

		for _, c := range checks {
			p := propIdx[c.name]
			if p == nil {
				t.Errorf("property %q not found", c.name)
				continue
			}
			if p.Kind != c.kind {
				t.Errorf("property %q: got Kind=%s, want %s (type=%s)", c.name, p.Kind, c.kind, p.TypeStr)
			}
		}

		// Version info checks
		imageData := propIdx["imageData"]
		if imageData == nil {
			t.Error("imageData property not found")
		} else {
			if imageData.Introduced != "9.17.0" {
				t.Errorf("imageData.Introduced = %q, want 9.17.0", imageData.Introduced)
			}
		}

		isRemote := propIdx["isRemote"]
		if isRemote == nil {
			t.Error("isRemote property not found")
		} else {
			if isRemote.Introduced != "7.17.0" {
				t.Errorf("isRemote.Introduced = %q, want 7.17.0", isRemote.Introduced)
			}
			if isRemote.Deleted != "8.10.0" {
				t.Errorf("isRemote.Deleted = %q, want 8.10.0", isRemote.Deleted)
			}
		}

		// Verify abstract class
		assocBase := classIdx["AssociationBase"]
		if assocBase == nil {
			t.Fatal("AssociationBase not found")
		}
		if !assocBase.Abstract {
			t.Error("AssociationBase should be abstract")
		}
	})

	// ==================== StructureTypeName Verification ====================
	t.Run("StructureTypeNames", func(t *testing.T) {
		if len(structTypeNames) == 0 {
			t.Fatal("expected structureTypeName assignments")
		}
		t.Logf("Found %d structureTypeName assignments", len(structTypeNames))

		checks := map[string]string{
			"Entity":           "DomainModels$Entity",
			"Attribute":        "DomainModels$Attribute",
			"Association":      "DomainModels$Association",
			"DomainModel":      "DomainModels$DomainModel",
			"NoGeneralization": "DomainModels$NoGeneralization",
			"Generalization":   "DomainModels$Generalization",
		}
		for className, want := range checks {
			got := structTypeNames[className]
			if got != want {
				t.Errorf("%s.structureTypeName = %q, want %q", className, got, want)
			}
		}
	})

	// ==================== Coverage Summary ====================
	t.Run("CoverageSummary", func(t *testing.T) {
		totalProps := 0
		kindCounts := map[PropertyKind]int{}
		unknownProps := []string{}
		for _, c := range classes {
			for _, p := range c.Properties {
				totalProps++
				kindCounts[p.Kind]++
				if p.Kind == KindUnknown {
					unknownProps = append(unknownProps, fmt.Sprintf("%s.%s (%s)", c.Name, p.Name, p.TypeStr))
				}
			}
		}
		t.Logf("Total properties across %d classes: %d", len(classes), totalProps)
		for k, v := range kindCounts {
			t.Logf("  %-10s: %d (%.1f%%)", k, v, float64(v)/float64(totalProps)*100)
		}
		if len(unknownProps) > 0 {
			t.Logf("Unknown properties (%d):", len(unknownProps))
			for _, p := range unknownProps {
				t.Logf("  %s", p)
			}
		}

		// Classification rate should be high
		unknownPct := float64(kindCounts[KindUnknown]) / float64(totalProps) * 100
		t.Logf("Classification rate: %.1f%%", 100-unknownPct)
	})
}
