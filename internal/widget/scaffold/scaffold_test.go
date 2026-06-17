package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePropertySpec(t *testing.T) {
	tests := []struct {
		input   string
		wantKey string
		wantXML string
		wantSub string
		wantErr bool
	}{
		{"value:attribute:Decimal", "value", "attribute", "Decimal", false},
		{"label:string", "label", "string", "", false},
		{"onChange:action", "onChange", "action", "", false},
		{"data:datasource", "data", "datasource", "", false},
		{"slot:widgets", "slot", "widgets", "", false},
		{"count:integer", "count", "integer", "", false},
		{"visible:boolean", "visible", "boolean", "", false},
		{"expr:expression", "expr", "expression", "", false},
		{"value:attribute:String", "value", "attribute", "String", false},
		{"bad", "", "", "", true},
		{"key:unknown", "", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			spec, err := ParsePropertySpec(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParsePropertySpec(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePropertySpec(%q) unexpected error: %v", tc.input, err)
			}
			if spec.Key != tc.wantKey || spec.XMLType != tc.wantXML || spec.Subtype != tc.wantSub {
				t.Errorf("got {%q %q %q}, want {%q %q %q}",
					spec.Key, spec.XMLType, spec.Subtype,
					tc.wantKey, tc.wantXML, tc.wantSub)
			}
		})
	}
}

func TestValidateWidgetIDFormat(t *testing.T) {
	t.Run("TooFewSegments", func(t *testing.T) {
		if err := ValidateWidgetIDFormat("com.acme.MyName"); err == nil {
			t.Fatal("expected error for ID with fewer than 4 segments, got nil")
		}
	})
	t.Run("ValidFourSegments", func(t *testing.T) {
		if err := ValidateWidgetIDFormat("com.acme.widget.MyName"); err != nil {
			t.Fatalf("expected no error for 4-segment ID, got %v", err)
		}
	})
	t.Run("ValidFiveSegments", func(t *testing.T) {
		if err := ValidateWidgetIDFormat("com.mendix.widget.custom.MyName"); err != nil {
			t.Fatalf("expected no error for 5-segment ID, got %v", err)
		}
	})
}

func TestDeriveWidgetID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"MySlider", "com.mendix.widget.custom.MySlider.MySlider"},
		{"CrusherSlider", "com.mendix.widget.custom.CrusherSlider.CrusherSlider"},
		{"Foo", "com.mendix.widget.custom.Foo.Foo"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveWidgetID(tc.name)
			if got != tc.want {
				t.Errorf("DeriveWidgetID(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestHumanizeWidgetName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"MySlider", "My Slider"},
		{"CrusherSimCanvas", "Crusher Sim Canvas"},
		{"Foo", "Foo"},
		{"HeatmapViz", "Heatmap Viz"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := HumanizeWidgetName(tc.input)
			if got != tc.want {
				t.Errorf("HumanizeWidgetName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRenderWidgetXML(t *testing.T) {
	spec := Spec{
		Name:     "MySlider",
		WidgetID: "com.acme.widget.MySlider.MySlider",
		Properties: []PropertySpec{
			{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
			{Key: "label", XMLType: "string"},
			{Key: "onChange", XMLType: "action"},
		},
		Description: "",
		Offline:     false,
		PackagePath: "com.mendix.widget.custom",
	}
	files := WidgetXMLRenderer{}.Render(spec)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	xml := string(files[0].Content)
	checks := []string{
		`id="com.acme.widget.MySlider.MySlider"`,
		`pluginWidget="true"`,
		`needsEntityContext="true"`,
		`offlineCapable="false"`,
		`supportedPlatform="Web"`,
		`<name>My Slider</name>`,
		`key="value" type="attribute"`,
		`<attributeType name="Decimal"/>`,
		`key="label" type="string"`,
		`key="onChange" type="action"`,
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("widget XML: missing %q\ngot:\n%s", want, xml)
		}
	}
}

func TestRenderWidgetXML_Offline(t *testing.T) {
	spec := Spec{Name: "Foo", WidgetID: "com.a.b.c.Foo", Offline: true}
	files := WidgetXMLRenderer{}.Render(spec)
	if !strings.Contains(string(files[0].Content), `offlineCapable="true"`) {
		t.Errorf("expected offlineCapable=true, got:\n%s", files[0].Content)
	}
}

func TestRenderWidgetXML_WithDescription(t *testing.T) {
	spec := Spec{Name: "Foo", WidgetID: "com.a.b.c.Foo", Description: `A <special> & "quoted" widget`}
	files := WidgetXMLRenderer{}.Render(spec)
	want := `<description>A &lt;special&gt; &amp; &quot;quoted&quot; widget</description>`
	if !strings.Contains(string(files[0].Content), want) {
		t.Errorf("missing escaped description %q\ngot:\n%s", want, files[0].Content)
	}
}

func TestRenderWidgetXML_EmptyDescription(t *testing.T) {
	spec := Spec{Name: "Foo", WidgetID: "com.a.b.c.Foo"}
	files := WidgetXMLRenderer{}.Render(spec)
	if !strings.Contains(string(files[0].Content), "<description></description>") {
		t.Errorf("expected empty description element\ngot:\n%s", files[0].Content)
	}
}

func TestRenderPackageXML(t *testing.T) {
	spec := Spec{Name: "MyPkg", PackageName: "mypkg", PackagePath: "com.mendix.widget.custom"}
	files := PackageXMLRenderer{}.Render(spec)
	xml := string(files[0].Content)
	checks := []string{
		`name="MyPkg"`,
		`version="1.0.0"`,
		`<widgetFile path="MyPkg.xml"/>`,
		`<file path="com/mendix/widget/custom/MyPkg/"/>`,
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("package XML: missing %q\ngot:\n%s", want, xml)
		}
	}
}

func TestRenderJSX(t *testing.T) {
	spec := Spec{
		Name: "MySlider",
		Properties: []PropertySpec{
			{Key: "value", XMLType: "attribute"},
			{Key: "label", XMLType: "string"},
			{Key: "onChange", XMLType: "action"},
		},
	}
	files := WidgetJSXRenderer{}.Render(spec)
	jsx := string(files[0].Content)
	checks := []string{
		`export function MySlider`,
		`{ value, label, onChange }`,
		`./components/MySliderSample`,
		`./ui/MySlider.css`,
	}
	for _, want := range checks {
		if !strings.Contains(jsx, want) {
			t.Errorf("JSX: missing %q\ngot:\n%s", want, jsx)
		}
	}
}

func TestRenderJSX_NoProps(t *testing.T) {
	spec := Spec{Name: "Foo"}
	files := WidgetJSXRenderer{}.Render(spec)
	if !strings.Contains(string(files[0].Content), "export function Foo(") {
		t.Errorf("JSX (no props): missing function signature\ngot:\n%s", files[0].Content)
	}
}

func TestRenderEditorConfig(t *testing.T) {
	spec := Spec{Name: "MySlider", Properties: []PropertySpec{{Key: "label", XMLType: "string"}}}
	files := EditorConfigRenderer{}.Render(spec)
	js := string(files[0].Content)
	if !strings.Contains(js, "getCustomCaption") || !strings.Contains(js, "getPreview") {
		t.Errorf("editor config: missing functions\ngot:\n%s", js)
	}
	if !strings.Contains(js, "MySlider") {
		t.Errorf("editor config: missing widget name\ngot:\n%s", js)
	}
}

func TestRenderPackageJSON(t *testing.T) {
	spec := Spec{Name: "MySlider", PackageName: "my-slider", PackagePath: "com.mendix.widget.custom", ProjectPath: "./tests/testProject"}
	files := PackageJSONRenderer{}.Render(spec)
	json := string(files[0].Content)
	checks := []string{
		`"@mendix/pluggable-widgets-tools"`,
		`"name": "my-slider"`,
		`"widgetName": "MySlider"`,
		`"projectPath": "./tests/testProject"`,
		`"packagePath": "com.mendix.widget.custom"`,
		`"react": "^19.0.0"`,
	}
	for _, want := range checks {
		if !strings.Contains(json, want) {
			t.Errorf("package.json: missing %q\ngot:\n%s", want, json)
		}
	}
}

func TestRunScaffold_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{
		Name:        "MySlider",
		WidgetID:    "com.acme.widget.MySlider.MySlider",
		PackagePath: "com.mendix.widget.custom",
		Properties: []PropertySpec{
			{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
			{Key: "label", XMLType: "string"},
		},
	}
	allRenderers := []Renderer{
		PackageJSONRenderer{},
		PackageXMLRenderer{},
		WidgetXMLRenderer{},
		WidgetJSXRenderer{},
		ComponentSampleRenderer{},
		EditorConfigRenderer{},
		EditorPreviewRenderer{},
		WidgetCSSRenderer{},
		IconRenderer{},
		ReadmeRenderer{},
		GitignoreRenderer{},
		ConfigFilesRenderer{},
	}
	if err := Run(dir, spec, allRenderers); err != nil {
		t.Fatalf("Run: %v", err)
	}
	wantFiles := []string{
		"src/MySlider.xml",
		"src/MySlider.jsx",
		"src/MySlider.editorConfig.js",
		"src/MySlider.editorPreview.jsx",
		"src/MySlider.icon.png",
		"src/MySlider.icon.dark.png",
		"src/MySlider.tile.png",
		"src/MySlider.tile.dark.png",
		"src/components/MySliderSample.jsx",
		"src/ui/MySlider.css",
		"src/package.xml",
		"package.json",
		".gitignore",
		"README.md",
		".eslintrc.js",
		"prettier.config.js",
		".gitattributes",
		"LICENSE",
		".prettierignore",
	}
	for _, rel := range wantFiles {
		full := filepath.Join(dir, rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
		}
	}
	xmlBytes, _ := os.ReadFile(filepath.Join(dir, "src/MySlider.xml"))
	if !strings.Contains(string(xmlBytes), "com.acme.widget.MySlider.MySlider") {
		t.Errorf("MySlider.xml missing widget ID")
	}
}

func TestRunScaffold_TopLevelFiles(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{Name: "MySlider", WidgetID: DeriveWidgetID("MySlider"), PackagePath: "com.mendix.widget.custom"}
	if err := Run(dir, spec, []Renderer{
		PackageJSONRenderer{},
		PackageXMLRenderer{},
		WidgetXMLRenderer{},
		WidgetJSXRenderer{},
		ComponentSampleRenderer{},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, rel := range []string{"package.json", "src/package.xml", "src/MySlider.xml", "src/MySlider.jsx"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}

func TestGitignore(t *testing.T) {
	spec := Spec{Name: "Foo"}
	files := GitignoreRenderer{}.Render(spec)
	out := string(files[0].Content)
	for _, want := range []string{"node_modules", "dist", "tests/testProject/"} {
		if !strings.Contains(out, want) {
			t.Errorf("gitignore: missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestReadme_WithPropsAndDescription(t *testing.T) {
	spec := Spec{
		Name:        "MySlider",
		Description: "A nice slider",
		Properties: []PropertySpec{
			{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
			{Key: "slot", XMLType: "widgets"},
		},
	}
	files := ReadmeRenderer{}.Render(spec)
	out := string(files[0].Content)
	checks := []string{
		"# MySlider",
		"A nice slider",
		"npm run build",
		"## Properties",
		"| value | attribute (Decimal) | Yes |",
		"| slot | widgets | No |",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("readme: missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestReadme_NoPropsNoDescription(t *testing.T) {
	spec := Spec{Name: "Foo"}
	files := ReadmeRenderer{}.Render(spec)
	out := string(files[0].Content)
	if !strings.Contains(out, "# Foo") {
		t.Errorf("readme: missing title\ngot:\n%s", out)
	}
	if strings.Contains(out, "## Properties") {
		t.Errorf("readme: should not have Properties section when no props\ngot:\n%s", out)
	}
}
