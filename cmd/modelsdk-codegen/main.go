package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/mendixlabs/mxcli/internal/codegen/dtsparser"
	"github.com/mendixlabs/mxcli/internal/codegen/emitter"
)

// All codegen configuration (storage aliases, property key overrides,
// force-concrete types, edge kind overrides, id-ref scope) lives in
// internal/codegen/supplements.json — the single source of truth.
// See loadSupplements() for parsing.

func main() {
	genDir := flag.String("gen-dir", "reference/mendixmodelsdk/src/gen", "TS SDK gen/ directory")
	outBase := flag.String("output", "modelsdk/gen", "output base directory")
	domains := flag.String("domains", "", "comma-separated domain names (empty = auto-discover all)")
	audit := flag.Bool("audit", false, "scan MPR files/dirs (positional args) for unregistered $Type values")
	auditKeys := flag.Bool("audit-keys", false, "scan MPR files for ByIdRef BSON key mismatches (Pointer suffix)")
	flag.Parse()

	// Audit mode: scan MPR files, compare against registry, report gaps.
	if *audit {
		args := flag.Args()
		if len(args) == 0 {
			log.Fatal("audit mode requires at least one MPR file or mprcontents directory as positional argument")
		}
		registered := collectRegisteredTypes(*outBase)
		auditAliases(args, registered)
		return
	}

	// Audit-keys mode: scan MPR BSON for ByIdRef key mismatches.
	if *auditKeys {
		args := flag.Args()
		if len(args) == 0 {
			log.Fatal("audit-keys mode requires at least one MPR file or mprcontents directory as positional argument")
		}
		byIdRefs := collectByIdRefKeys(*outBase)
		auditPropertyKeys(args, byIdRefs)
		return
	}

	domainList := discoverDomains(*genDir, *domains)

	// Phase 1: Parse all domains to build cross-domain property map.
	// Classes like projects.Document define properties (Name, Documentation,
	// ExportLevel, Excluded) that are inherited by types in other domains
	// (Workflow, Page, Microflow, etc.). Without this map, the inheritance
	// walker stops at cross-domain boundaries and those setters are missing.
	crossDomainProps := buildCrossDomainProps(*genDir, domainList)
	fmt.Printf("Cross-domain base classes: %d\n", len(crossDomainProps))

	// Load all codegen configuration from supplements.json.
	suppl := loadSupplements()
	fmt.Printf("Config: %d aliases, %d prop-key overrides, %d force-concrete, %d edge-kinds, %d id-ref-scope, %d extra props, %d extra type groups\n",
		len(suppl.StorageAliases), len(suppl.PropertyKeyOverrides),
		len(suppl.forceConcreteSet), len(suppl.EdgeKindOverrides),
		len(suppl.IdRefScope), len(suppl.parsedExtraProps), len(suppl.parsedExtraTypes))

	// Phase 2: Generate each domain with cross-domain inheritance resolved.
	for _, domain := range domainList {
		jsPath := filepath.Join(*genDir, domain+".js")
		if _, err := os.Stat(jsPath); err != nil {
			log.Fatalf("js file not found: %s", jsPath)
		}
		meta, err := dtsparser.ParseJsFile(jsPath)
		if err != nil {
			log.Fatalf("parse %s: %v", jsPath, err)
		}

		// Apply storage name aliases from supplements.json.
		aliases := map[string]string{}
		for jsName, bsonName := range suppl.StorageAliases {
			for _, cls := range meta.Classes {
				if cls.StructureTypeName == jsName {
					aliases[jsName] = bsonName
					fmt.Printf("  alias: %s → %s\n", jsName, bsonName)
					break
				}
			}
		}
		if len(aliases) > 0 {
			meta.StorageAliases = aliases
		}

		// Force concrete: some classes the parser marks as abstract (no static
		// createIn method) but which appear as concrete $Type values in BSON.
		for i := range meta.Classes {
			if suppl.forceConcreteSet[meta.Classes[i].StructureTypeName] {
				meta.Classes[i].IsAbstract = false
				fmt.Printf("  force-concrete: %s\n", meta.Classes[i].StructureTypeName)
			}
		}

		// Inject supplemental extra_properties into existing classes.
		for i := range meta.Classes {
			cls := &meta.Classes[i]
			if extras, ok := suppl.parsedExtraProps[cls.Name]; ok {
				for _, ep := range extras {
					cls.Properties = append(cls.Properties, supplementPropToJsProp(ep))
				}
				fmt.Printf("  +%d extra props on %s\n", len(extras), cls.Name)
			}
		}

		// Inject supplemental extra_types as new classes.
		if extraTypes, ok := suppl.parsedExtraTypes[domain]; ok {
			for _, et := range extraTypes {
				cls := supplementTypeToJsClass(et)
				meta.Classes = append(meta.Classes, cls)
				fmt.Printf("  +extra type: %s (%s)\n", cls.Name, cls.StructureTypeName)
			}
		}

		// Apply property-level BSON key overrides and other supplements.
		meta.PropertyKeyOverrides = suppl.PropertyKeyOverrides
		meta.RefListVersion3Fields = suppl.refListVersion3Fields
		meta.EdgeKindOverrides = suppl.EdgeKindOverrides
		meta.IdRefScope = suppl.IdRefScope
		meta.CrossDomainProps = crossDomainProps

		outDir := filepath.Join(*outBase, domain)
		if err := emitter.Generate(meta, outDir); err != nil {
			log.Fatalf("generate %s: %v", domain, err)
		}
		fmt.Printf("Generated %s: %d classes, %d enums\n", domain, len(meta.Classes), len(meta.Enums))
	}
}

// ────────────────────────────────────────────────────────────────
// Supplements: types/properties/aliases not in TypeScript SDK
// ────────────────────────────────────────────────────────────────

type supplements struct {
	StorageAliases       map[string]string          `json:"storage_aliases"`
	PropertyKeyOverrides map[string]string          `json:"property_key_overrides"`
	RefListVersion3List  []string                   `json:"ref_list_version3_fields"`
	ForceConcreteTypes   []string                   `json:"force_concrete_types"`
	EdgeKindOverrides    map[string]string          `json:"edge_kind_overrides"`
	IdRefScope           map[string]string          `json:"id_ref_scope"`
	ExtraProperties      map[string]json.RawMessage `json:"extra_properties"`
	ExtraTypes           map[string]json.RawMessage `json:"extra_types"`

	// Derived after loading.
	forceConcreteSet      map[string]bool // built from ForceConcreteTypes slice
	refListVersion3Fields map[string]bool // built from RefListVersion3List
	parsedExtraProps      map[string][]supplementProp
	parsedExtraTypes      map[string][]supplementTypeDef
}

type supplementProp struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`           // Primitive, ByNameRef, Part, PartList
	PrimitiveType string `json:"primitive_type"` // String, Boolean, Integer, etc.
	Target        string `json:"target"`         // for ByNameRef
}

type supplementTypeDef struct {
	Name       string           `json:"name"`
	BSONType   string           `json:"bson_type"`
	Properties []supplementProp `json:"properties"`
}

func loadSupplements() supplements {
	data, err := os.ReadFile("internal/codegen/supplements.json")
	if err != nil {
		fmt.Printf("No supplements.json found, skipping\n")
		return supplements{
			parsedExtraProps: map[string][]supplementProp{},
			parsedExtraTypes: map[string][]supplementTypeDef{},
		}
	}
	var s supplements
	if err := json.Unmarshal(data, &s); err != nil {
		log.Fatalf("parse supplements.json: %v", err)
	}
	delete(s.StorageAliases, "_doc")
	delete(s.PropertyKeyOverrides, "_doc")
	delete(s.EdgeKindOverrides, "_doc")
	delete(s.IdRefScope, "_doc")

	// Build force-concrete lookup set.
	s.forceConcreteSet = map[string]bool{}
	for _, t := range s.ForceConcreteTypes {
		s.forceConcreteSet[t] = true
	}

	// Build ref_list_version3 lookup set from slice.
	s.refListVersion3Fields = map[string]bool{}
	for _, f := range s.RefListVersion3List {
		s.refListVersion3Fields[f] = true
	}

	// Parse extra_properties, skipping _doc string entries.
	s.parsedExtraProps = map[string][]supplementProp{}
	for key, raw := range s.ExtraProperties {
		if key == "_doc" {
			continue
		}
		var props []supplementProp
		if err := json.Unmarshal(raw, &props); err != nil {
			log.Fatalf("parse extra_properties[%s]: %v", key, err)
		}
		s.parsedExtraProps[key] = props
	}

	// Parse extra_types, skipping _doc string entries.
	s.parsedExtraTypes = map[string][]supplementTypeDef{}
	for key, raw := range s.ExtraTypes {
		if key == "_doc" {
			continue
		}
		var types []supplementTypeDef
		if err := json.Unmarshal(raw, &types); err != nil {
			log.Fatalf("parse extra_types[%s]: %v", key, err)
		}
		s.parsedExtraTypes[key] = types
	}

	return s
}

func supplementPropToJsProp(sp supplementProp) dtsparser.JsProp {
	p := dtsparser.JsProp{Name: sp.Name}
	switch sp.Kind {
	case "Primitive":
		p.Kind = dtsparser.PKPrimitive
		p.PrimitiveType = dtsparser.PrimitiveType(sp.PrimitiveType)
	case "ByNameRef":
		p.Kind = dtsparser.PKByNameRef
		p.TargetType = sp.Target
	case "Part":
		p.Kind = dtsparser.PKPart
	case "PartList":
		p.Kind = dtsparser.PKPartList
	case "Enum":
		p.Kind = dtsparser.PKEnum
		p.TargetType = sp.Target
	default:
		p.Kind = dtsparser.PKPrimitive
		p.PrimitiveType = dtsparser.PTString
	}
	return p
}

func supplementTypeToJsClass(et supplementTypeDef) dtsparser.JsClass {
	cls := dtsparser.JsClass{
		Name:              et.Name,
		StructureTypeName: et.BSONType,
	}
	for _, sp := range et.Properties {
		cls.Properties = append(cls.Properties, supplementPropToJsProp(sp))
	}
	return cls
}

// pascalCase converts a camelCase string to PascalCase.
func pascalCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// buildCrossDomainProps parses all domains and collects class properties
// (including inherited) that might be needed across domain boundaries.
// For example, projects.Document defines Name — MxSchema extends Document,
// and JsonStructure extends MxSchema. JsonStructure needs Name even though
// it crosses two domain boundaries.
func buildCrossDomainProps(genDir string, domainList []string) map[string][]dtsparser.JsProp {
	// Step 1: Collect all classes with their own properties and base class.
	type classInfo struct {
		baseName string
		ownProps []dtsparser.JsProp
	}
	allClasses := map[string]*classInfo{}
	for _, domain := range domainList {
		jsPath := filepath.Join(genDir, domain+".js")
		meta, err := dtsparser.ParseJsFile(jsPath)
		if err != nil {
			continue
		}
		for _, cls := range meta.Classes {
			base := cls.BaseClass
			// Strip cross-domain prefix: "projects_1.projects.Document" → "Document"
			if idx := strings.LastIndex(base, "."); idx >= 0 {
				base = base[idx+1:]
			}
			// Stop at SDK base types
			if strings.HasPrefix(cls.BaseClass, "internal.") {
				base = ""
			}
			allClasses[cls.Name] = &classInfo{
				baseName: base,
				ownProps: cls.Properties,
			}
		}
	}

	// Step 2: For each class, walk up the inheritance chain collecting ALL
	// properties (own + inherited). Cache results for efficiency.
	resolved := map[string][]dtsparser.JsProp{}
	var resolve func(name string) []dtsparser.JsProp
	resolve = func(name string) []dtsparser.JsProp {
		if cached, ok := resolved[name]; ok {
			return cached
		}
		ci, ok := allClasses[name]
		if !ok {
			return nil
		}
		// Collect inherited props first (root-first order).
		var all []dtsparser.JsProp
		seen := map[string]bool{}
		if ci.baseName != "" && ci.baseName != name {
			for _, p := range resolve(ci.baseName) {
				if !seen[p.Name] {
					seen[p.Name] = true
					all = append(all, p)
				}
			}
		}
		for _, p := range ci.ownProps {
			if !seen[p.Name] {
				seen[p.Name] = true
				all = append(all, p)
			}
		}
		resolved[name] = all
		return all
	}

	result := map[string][]dtsparser.JsProp{}
	for name := range allClasses {
		props := resolve(name)
		if len(props) > 0 {
			result[name] = props
		}
	}
	return result
}

// collectRegisteredTypes scans generated types.go files for all registered $Type names.
func collectRegisteredTypes(genBase string) map[string]bool {
	types := map[string]bool{}
	entries, err := os.ReadDir(genBase)
	if err != nil {
		log.Fatalf("read %s: %v", genBase, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		typesFile := filepath.Join(genBase, e.Name(), "types.go")
		data, err := os.ReadFile(typesFile)
		if err != nil {
			continue
		}
		// Extract: codec.DefaultRegistry.Register("Foo$Bar", ...)
		content := string(data)
		for {
			idx := strings.Index(content, `codec.DefaultRegistry.Register("`)
			if idx < 0 {
				break
			}
			content = content[idx+len(`codec.DefaultRegistry.Register("`):]
			end := strings.IndexByte(content, '"')
			if end < 0 {
				break
			}
			types[content[:end]] = true
			content = content[end:]
		}
	}
	return types
}

// discoverDomains returns the list of domains to generate.
// If explicit is non-empty, splits on comma. Otherwise auto-discovers
// from .js files in the gen directory.
func discoverDomains(genDir, explicit string) []string {
	if explicit != "" {
		var result []string
		for _, d := range strings.Split(explicit, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				result = append(result, d)
			}
		}
		return result
	}

	entries, err := os.ReadDir(genDir)
	if err != nil {
		log.Fatalf("auto-discover: read %s: %v", genDir, err)
	}
	var result []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".js") && !strings.HasSuffix(name, ".d.ts") {
			domain := strings.TrimSuffix(name, ".js")
			// Skip internal/base modules
			if domain == "all-model-classes" || domain == "base-model" {
				continue
			}
			result = append(result, domain)
		}
	}
	if len(result) == 0 {
		log.Fatalf("auto-discover: no .js files found in %s", genDir)
	}
	return result
}
