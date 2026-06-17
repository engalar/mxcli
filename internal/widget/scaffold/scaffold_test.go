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
				t.Errorf("got {%q %q %q}, want {%q %q %q}", spec.Key, spec.XMLType, spec.Subtype, tc.wantKey, tc.wantXML, tc.wantSub)
			}
		})
	}
}

func TestDeriveWidgetID(t *testing.T) {
	cases := []struct {
		packagePath string
		name        string
		want        string
	}{
		{"com.mendix.widget.custom", "MySlider", "com.mendix.widget.custom.myslider.MySlider"},
		{"com.helpdesk.widget", "TicketStatusBadge", "com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge"},
	}
	for _, c := range cases {
		if got := DeriveWidgetID(c.packagePath, c.name); got != c.want {
			t.Errorf("DeriveWidgetID(%q, %q) = %q, want %q", c.packagePath, c.name, got, c.want)
		}
	}
}

func TestHumanizeWidgetName(t *testing.T) {
	if got := HumanizeWidgetName("MySlider"); got != "My Slider" {
		t.Errorf("HumanizeWidgetName = %q", got)
	}
}

func TestRunScaffold_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{
		Name:        "MySlider",
		WidgetID:    "com.acme.widget.MySlider.MySlider",
		PackagePath: "com.mendix.widget.custom",
		ProjectPath: "./tests/testProject",
		Properties: []PropertySpec{
			{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
			{Key: "label", XMLType: "string"},
		},
	}
	if err := Run(dir, spec); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantFiles := []string{
		"package.json",
		"src/package.xml",
		"src/MySlider.xml",
		"src/MySlider.jsx",
		"src/MySlider.editorConfig.js",
		"src/MySlider.editorPreview.jsx",
		"src/components/MySliderSample.jsx",
		"src/ui/MySlider.css",
		"src/MySlider.icon.png",
		"src/MySlider.icon.dark.png",
		"src/MySlider.tile.png",
		"src/MySlider.tile.dark.png",
		"README.md",
		".eslintrc.js",
		"prettier.config.js",
		".gitignore",
		".gitattributes",
		".prettierignore",
		"LICENSE",
	}
	for _, rel := range wantFiles {
		full := filepath.Join(dir, rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
		}
	}

	// Verify key content replacements
	xmlData, _ := os.ReadFile(filepath.Join(dir, "src/MySlider.xml"))
	xml := string(xmlData)
	if !strings.Contains(xml, `id="com.acme.widget.MySlider.MySlider"`) {
		t.Errorf("widget XML missing widget ID:\n%s", xml)
	}
	if !strings.Contains(xml, `<icon/>`) {
		t.Errorf("widget XML missing <icon/>:\n%s", xml)
	}

	pkgXML, _ := os.ReadFile(filepath.Join(dir, "src/package.xml"))
	if !strings.Contains(string(pkgXML), `file path="com/mendix/widget/custom/myslider"`) {
		t.Errorf("package.xml wrong file path:\n%s", pkgXML)
	}

	jsx, _ := os.ReadFile(filepath.Join(dir, "src/MySlider.jsx"))
	if !strings.Contains(string(jsx), "export function MySlider({ value, label })") {
		t.Errorf("JSX wrong signature:\n%s", jsx)
	}
}

func TestRunScaffold_NoProps(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{Name: "Empty", WidgetID: DeriveWidgetID("com.mendix.widget.custom", "Empty"), PackagePath: "com.mendix.widget.custom"}
	if err := Run(dir, spec); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, rel := range []string{"package.json", "src/package.xml", "src/Empty.xml", "src/Empty.jsx"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
	// No-props JSX should use _props
	jsx, _ := os.ReadFile(filepath.Join(dir, "src/Empty.jsx"))
	if !strings.Contains(string(jsx), "export function Empty(_props)") {
		t.Errorf("no-props JSX wrong:\n%s", jsx)
	}
}

func TestRenderPropertiesXML(t *testing.T) {
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
		{Key: "label", XMLType: "string"},
		{Key: "onChange", XMLType: "action"},
	}
	xml := renderPropertiesXML(props)
	checks := []string{
		`key="value" type="attribute"`,
		`<attributeType name="Decimal"/>`,
		`key="label" type="string"`,
		`key="onChange" type="action"`,
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("property XML: missing %q\ngot:\n%s", want, xml)
		}
	}
}

func TestRenderPropertiesXML_Empty(t *testing.T) {
	xml := renderPropertiesXML(nil)
	if !strings.Contains(xml, "<propertyGroup") {
		t.Errorf("empty props should still have propertyGroup:\n%s", xml)
	}
}
