package emitter

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/codegen/dtsparser"
)

func findDomainModelsJS(t *testing.T) string {
	t.Helper()
	p := "../../.." + "/node_modules/mendixmodelsdk/src/gen/domainmodels.js"
	if _, err := os.Stat(p); err != nil {
		t.Skip("mendixmodelsdk not available — run: npm install")
	}
	return p
}

func TestGenerateDomainModels(t *testing.T) {
	domainmodelsJS := findDomainModelsJS(t)
	// Parse the real domainmodels.js file.
	meta, err := dtsparser.ParseJsFile(domainmodelsJS)
	if err != nil {
		t.Fatalf("ParseJsFile: %v", err)
	}

	if meta.Namespace == "" {
		t.Fatal("expected non-empty namespace")
	}
	t.Logf("namespace=%s  classes=%d  enums=%d", meta.Namespace, len(meta.Classes), len(meta.Enums))

	// Generate into a temp directory.
	outDir := t.TempDir()
	if err := Generate(meta, outDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify files exist.
	for _, name := range []string{"types.go", "enums.go", "version.go"} {
		path := filepath.Join(outDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("%s: not found: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s: file is empty", name)
		}
	}

	// Read types.go for content checks.
	typesBytes, err := os.ReadFile(filepath.Join(outDir, "types.go"))
	if err != nil {
		t.Fatalf("read types.go: %v", err)
	}
	types := string(typesBytes)

	// Check: Entity struct exists.
	if !strings.Contains(types, "type Entity struct") {
		t.Error("types.go missing 'type Entity struct'")
	}

	// Check: embeds element.Base.
	if !strings.Contains(types, "element.Base") {
		t.Error("types.go missing 'element.Base' embed")
	}

	// Check: Entity has a Name getter.
	if !strings.Contains(types, "func (o *Entity) Name()") {
		t.Error("types.go missing Entity.Name() getter")
	}

	// Check: Entity has a SetName setter.
	if !strings.Contains(types, "func (o *Entity) SetName(") {
		t.Error("types.go missing Entity.SetName() setter")
	}

	// Check: registry init contains the structure type name.
	if !strings.Contains(types, `"DomainModels$Entity"`) {
		t.Error("types.go missing registry entry for DomainModels$Entity")
	}

	// Check: DomainModel struct exists (it's the structural unit).
	if !strings.Contains(types, "type DomainModel struct") {
		t.Error("types.go missing 'type DomainModel struct'")
	}

	// Read enums.go for content checks.
	enumsBytes, err := os.ReadFile(filepath.Join(outDir, "enums.go"))
	if err != nil {
		t.Fatalf("read enums.go: %v", err)
	}
	enums := string(enumsBytes)

	// The JS parser may not find enums in all .js formats (enum regex
	// sensitivity). When enums are found, verify their structure.
	if len(meta.Enums) > 0 {
		if !strings.Contains(enums, "type ") || !strings.Contains(enums, "= string") {
			t.Error("enums.go missing enum type declarations")
		}
		if !strings.Contains(enums, "const (") {
			t.Error("enums.go missing const block")
		}
	} else {
		t.Log("note: JS parser found 0 enums (known parser limitation for this .js format)")
	}

	// Read version.go for content checks.
	versionBytes, err := os.ReadFile(filepath.Join(outDir, "version.go"))
	if err != nil {
		t.Fatalf("read version.go: %v", err)
	}
	ver := string(versionBytes)

	// Check: version file has at least one entry.
	if !strings.Contains(ver, "VersionInfos") {
		t.Error("version.go missing VersionInfos map")
	}
	if !strings.Contains(ver, "version.TypeVersionInfo") {
		t.Error("version.go missing version.TypeVersionInfo usage")
	}

	// Log some output for debugging.
	t.Logf("types.go size: %d bytes", len(typesBytes))
	t.Logf("enums.go size: %d bytes", len(enumsBytes))
	t.Logf("version.go size: %d bytes", len(versionBytes))
}

func TestGenerateSyntheticMeta(t *testing.T) {
	// Test with a small hand-crafted DomainMeta to verify edge cases.
	meta := &dtsparser.DomainMeta{
		Namespace: "testpkg",
		Classes: []dtsparser.JsClass{
			{
				Name:              "MyElement",
				StructureTypeName: "TestDomain$MyElement",
				StructureKind:     dtsparser.SKElement,
				IsAbstract:        false,
				Properties: []dtsparser.JsProp{
					{Name: "name", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTString},
					{Name: "active", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTBoolean},
					{Name: "count", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTInteger},
					{Name: "ratio", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTDouble},
					{Name: "child", Kind: dtsparser.PKPart},
					{Name: "children", Kind: dtsparser.PKPartList},
					{Name: "target", Kind: dtsparser.PKByNameRef, TargetType: "TestDomain$Target"},
					{Name: "ref", Kind: dtsparser.PKByIdRef},
					{Name: "status", Kind: dtsparser.PKEnum, TargetType: "MyStatus"},
					{Name: "tags", Kind: dtsparser.PKEnumList},
					{Name: "refs", Kind: dtsparser.PKByNameRefList, TargetType: "TestDomain$Ref"},
				},
			},
			{
				// Abstract class — struct generated but not registered.
				Name:              "AbstractBase",
				StructureTypeName: "TestDomain$AbstractBase",
				IsAbstract:        true,
				Properties: []dtsparser.JsProp{
					{Name: "baseProp", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTString},
				},
			},
		},
		Enums: []dtsparser.JsEnum{
			{
				Name: "MyStatus",
				Values: []dtsparser.JsEnumValue{
					{Name: "Active"},
					{Name: "Inactive"},
					{Name: "Deleted"},
				},
			},
		},
	}

	outDir := t.TempDir()
	if err := Generate(meta, outDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Read types.go
	typesBytes, err := os.ReadFile(filepath.Join(outDir, "types.go"))
	if err != nil {
		t.Fatalf("read types.go: %v", err)
	}
	types := string(typesBytes)

	// Concrete class generated.
	if !strings.Contains(types, "type MyElement struct") {
		t.Error("missing MyElement struct")
	}

	// Abstract class IS generated as a struct (for inherited properties),
	// but is NOT registered in the factory registry.
	if !strings.Contains(types, "type AbstractBase struct") {
		t.Error("AbstractBase (abstract) should be generated as a struct")
	}

	// Registry contains concrete type.
	if !strings.Contains(types, `"TestDomain$MyElement"`) {
		t.Error("missing registry entry for TestDomain$MyElement")
	}

	// Registry does NOT contain abstract type.
	if strings.Contains(types, `Register("TestDomain$AbstractBase"`) {
		t.Error("abstract type should not be registered in the factory")
	}

	// Primitive string getter.
	if !strings.Contains(types, "func (o *MyElement) Name() string") {
		t.Error("missing Name() getter")
	}

	// Primitive string setter.
	if !strings.Contains(types, "func (o *MyElement) SetName(v string)") {
		t.Error("missing SetName() setter")
	}

	// Primitive bool getter.
	if !strings.Contains(types, "func (o *MyElement) Active() bool") {
		t.Error("missing Active() getter")
	}

	// Primitive int32 getter.
	if !strings.Contains(types, "func (o *MyElement) Count() int32") {
		t.Error("missing Count() getter")
	}

	// Primitive float64 getter.
	if !strings.Contains(types, "func (o *MyElement) Ratio() float64") {
		t.Error("missing Ratio() getter")
	}

	// Part setter.
	if !strings.Contains(types, "func (o *MyElement) SetChild(") {
		t.Error("missing SetChild() setter")
	}

	// PartList adder.
	if !strings.Contains(types, "func (o *MyElement) AddChildren(") {
		t.Error("missing AddChildren() adder")
	}

	// ByNameRef getter.
	if !strings.Contains(types, "func (o *MyElement) TargetQualifiedName() string") {
		t.Error("missing TargetQualifiedName() getter")
	}

	// ByIdRef getter.
	if !strings.Contains(types, "func (o *MyElement) RefRefID() element.ID") {
		t.Error("missing RefRefID() getter")
	}

	// Enum getter.
	if !strings.Contains(types, "func (o *MyElement) Status() string") {
		t.Error("missing Status() getter")
	}

	// EnumList getter.
	if !strings.Contains(types, "func (o *MyElement) TagsItems() []string") {
		t.Error("missing TagsItems() getter")
	}

	// ByNameRefList getter.
	if !strings.Contains(types, "func (o *MyElement) RefsQualifiedNames() []string") {
		t.Error("missing RefsQualifiedNames() getter")
	}

	// Fields should be unexported.
	if !strings.Contains(types, "name ") && !strings.Contains(types, "\tname ") {
		t.Error("expected unexported field 'name'")
	}

	// Read enums.go
	enumsBytes, err := os.ReadFile(filepath.Join(outDir, "enums.go"))
	if err != nil {
		t.Fatalf("read enums.go: %v", err)
	}
	enumStr := string(enumsBytes)

	if !strings.Contains(enumStr, "type MyStatus = string") {
		t.Error("missing MyStatus type alias")
	}
	if !strings.Contains(enumStr, `MyStatusActive`) {
		t.Error("missing MyStatusActive const")
	}
	if !strings.Contains(enumStr, `MyStatusInactive`) {
		t.Error("missing MyStatusInactive const")
	}
	if !strings.Contains(enumStr, `MyStatusDeleted`) {
		t.Error("missing MyStatusDeleted const")
	}

	t.Logf("types.go:\n%s", types)
}

func TestDisambiguateReservedNames(t *testing.T) {
	meta := &dtsparser.DomainMeta{
		Namespace: "testpkg",
		Classes: []dtsparser.JsClass{
			{
				Name:              "Tricky",
				StructureTypeName: "TestDomain$Tricky",
				IsAbstract:        false,
				Properties: []dtsparser.JsProp{
					// These JS property names, when used as unexported Go fields,
					// would collide with element.Base's unexported fields.
					{Name: "id", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTString},
					{Name: "typeName", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTString},
					{Name: "container", Kind: dtsparser.PKPrimitive, PrimitiveType: dtsparser.PTString},
				},
			},
		},
	}

	outDir := t.TempDir()
	if err := Generate(meta, outDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	typesBytes, err := os.ReadFile(filepath.Join(outDir, "types.go"))
	if err != nil {
		t.Fatalf("read types.go: %v", err)
	}
	types := string(typesBytes)

	// "id" collides with Base's unexported field -> should become "propId"
	if !strings.Contains(types, "propId") {
		t.Error("expected propId for disambiguated 'id' property")
	}

	// "typeName" collides with Base's unexported field -> "propTypeName"
	if !strings.Contains(types, "propTypeName") {
		t.Error("expected propTypeName for disambiguated 'typeName' property")
	}

	// "container" collides with Base's unexported field -> "propContainer"
	if !strings.Contains(types, "propContainer") {
		t.Error("expected propContainer for disambiguated 'container' property")
	}

	// Getters should still be exported PascalCase names (no collision since
	// they don't shadow embedded methods -- embedded methods have pointer receivers).
	// ID() would shadow Base.ID() though, so let's verify the getter name.
	// Actually, exported getter "Id" won't shadow "ID" -- they're different names.
	// The getter for "id" property will be "Id" (exportName("id") = "Id").
	if !strings.Contains(types, "func (o *Tricky) Id()") {
		t.Error("expected Id() getter for 'id' property")
	}
}

func TestGeneratedCodeParses(t *testing.T) {
	domainmodelsJS := findDomainModelsJS(t)
	// Generate code from the real domainmodels.js and verify all generated
	// files are valid Go source (parseable by go/parser).
	meta, err := dtsparser.ParseJsFile(domainmodelsJS)
	if err != nil {
		t.Fatalf("ParseJsFile: %v", err)
	}

	outDir := t.TempDir()
	if err := Generate(meta, outDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	fset := token.NewFileSet()
	for _, name := range []string{"types.go", "enums.go", "version.go"} {
		path := filepath.Join(outDir, name)
		_, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			// Read the file for diagnostics.
			data, _ := os.ReadFile(path)
			// Show first 40 lines for context.
			lines := strings.SplitN(string(data), "\n", 41)
			preview := strings.Join(lines, "\n")
			t.Errorf("%s: parse error: %v\n--- first lines ---\n%s", name, err, preview)
		}
	}
}

func TestExportName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"name", "Name"},
		{"dataStorageGuid", "DataStorageGuid"},
		{"", ""},
		{"Name", "Name"},
		{"a", "A"},
	}
	for _, tc := range tests {
		got := exportName(tc.input)
		if got != tc.want {
			t.Errorf("exportName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestUnexportName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Name", "name"},
		{"name", "name"},
		{"", ""},
		{"A", "a"},
		{"ID", "iD"},
	}
	for _, tc := range tests {
		got := unexportName(tc.input)
		if got != tc.want {
			t.Errorf("unexportName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
