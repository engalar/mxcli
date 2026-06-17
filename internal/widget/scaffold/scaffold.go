package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

//go:embed template/.eslintrc.js template/.gitattributes template/.gitignore template/.prettierignore template/LICENSE template/prettier.config.js template/package.json.tmpl template/README.md.tmpl template/src/package.xml.tmpl template/src/WIDGET_NAME.xml.tmpl template/src/WIDGET_NAME.jsx.tmpl template/src/WIDGET_NAME.editorConfig.js.tmpl template/src/WIDGET_NAME.editorPreview.jsx.tmpl template/src/components/WIDGET_NAMESample.jsx.tmpl template/src/ui/WIDGET_NAME.css.tmpl
var templateFS embed.FS

// File describes a generated file.
type File struct {
	Path    string
	Content []byte
	Binary  bool
}

// Run scaffolds a widget project from the embedded reference template.
func Run(dir string, spec Spec) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	tmplCtx := newTemplateContext(spec)
	files, err := collectFiles("template", tmplCtx)
	if err != nil {
		return fmt.Errorf("collect template files: %w", err)
	}

	// Add placeholder icon PNGs
	icons := generateIcons(spec.Name)
	files = append(files, icons...)

	for _, f := range files {
		dest := filepath.Join(dir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(dest, f.Content, 0644); err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}
	return nil
}

func collectFiles(root string, ctx *templateContext) ([]File, error) {
	entries, err := templateFS.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var files []File
	for _, e := range entries {
		embedPath := root + "/" + e.Name()
		if e.IsDir() {
			sub, err := collectFiles(embedPath, ctx)
			if err != nil {
				return nil, err
			}
			files = append(files, sub...)
			continue
		}
		data, err := templateFS.ReadFile(embedPath)
		if err != nil {
			return nil, err
		}
		rel := strings.TrimPrefix(embedPath, "template/")

		if strings.HasSuffix(e.Name(), ".tmpl") {
			// Token replacement in content
			content := ctx.replace(string(data))
			// Output name: strip .tmpl, replace WIDGET_NAME tokens
			outName := strings.TrimSuffix(rel, ".tmpl")
			outName = ctx.replace(outName)
			files = append(files, File{Path: outName, Content: []byte(content)})
		} else {
			// Static file: copy as-is
			files = append(files, File{Path: rel, Content: data})
		}
	}
	return files, nil
}

type templateContext struct {
	Tokens map[string]string // token → value
}

func newTemplateContext(spec Spec) *templateContext {
	propsXML := renderPropertiesXML(spec.Properties)
	propParams := renderPropParams(spec.Properties)
	propAttrs := renderPropAttrs(spec.Properties)
	firstKey := firstPropKey(spec.Properties)
	firstDisplay := firstPropDisplay(spec.Properties)
	cssClass := "widget-" + strings.ToLower(spec.Name)
	copyright := spec.Copyright
	if copyright == "" {
		copyright = "© Mendix Technology BV 2026. All rights reserved."
	}

	offlineStr := "false"
	if spec.Offline {
		offlineStr = "true"
	}

	return &templateContext{Tokens: map[string]string{
		"WIDGET_NAMESample":      spec.Name + "Sample",
		"WIDGET_NAME":            spec.Name,
		"widget_name":            strings.ToLower(spec.Name),
		"WIDGET_ID":              spec.WidgetID,
		"PACKAGE_PATH":           spec.PackagePath,
		"PACKAGE_PATH_SLASH":     strings.ReplaceAll(spec.PackagePath, ".", "/"),
		"DESCRIPTION":            spec.Description,
		"OFFLINE_CAPABLE":        offlineStr,
		"HUMAN_NAME":             HumanizeWidgetName(spec.Name),
		"PROPERTIES_XML":         propsXML,
		"PROPERTIES_PARAMS":      propParams,
		"PROPERTIES_ATTRS":       propAttrs,
		"FIRST_PROP_KEY":         firstKey,
		"FIRST_PROP_DISPLAY":     firstDisplay,
		"CSS_CLASS":              cssClass,
		"PROJECT_PATH":           spec.ProjectPath,
		"AUTHOR":                 spec.Author,
		"COPYRIGHT":              copyright,
		"PACKAGE_NAME":           spec.PackageName,
	}}
}

func (ctx *templateContext) replace(s string) string {
	// Multi-pass: replace longest tokens first to avoid partial matches
	keys := make([]string, 0, len(ctx.Tokens))
	for k := range ctx.Tokens {
		keys = append(keys, k)
	}
	// Sort by length descending so WIDGET_NAMESample matches before WIDGET_NAME
	// (simple bubble sort is fine for small N)
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if len(keys[j]) > len(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	result := s
	for _, k := range keys {
		result = strings.ReplaceAll(result, k, ctx.Tokens[k])
	}
	return result
}

var (
	validXMLTypes = map[string]bool{
		"attribute":  true,
		"string":     true,
		"integer":    true,
		"boolean":    true,
		"action":     true,
		"datasource": true,
		"expression": true,
		"widgets":    true,
	}
)

func renderPropertiesXML(props []PropertySpec) string {
	if len(props) == 0 {
		return `        <propertyGroup caption="General">
        </propertyGroup>`
	}
	var b strings.Builder
	b.WriteString("        <propertyGroup caption=\"General\">\n")
	for _, p := range props {
		human := HumanizeWidgetName(p.Key)
		switch p.XMLType {
		case "attribute":
			attrType := p.Subtype
			if attrType == "" {
				attrType = "String"
			}
			fmt.Fprintf(&b, `            <property key=%q type="attribute" required="true">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			fmt.Fprintf(&b, `                <attributeTypes><attributeType name=%q/></attributeTypes>`+"\n", attrType)
			b.WriteString("            </property>\n")
		case "string":
			fmt.Fprintf(&b, `            <property key=%q type="string" required="false" defaultValue="">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			b.WriteString("            </property>\n")
		case "integer":
			fmt.Fprintf(&b, `            <property key=%q type="integer" required="true" defaultValue="0">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			b.WriteString("            </property>\n")
		case "boolean":
			fmt.Fprintf(&b, `            <property key=%q type="boolean" required="true" defaultValue="false">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			b.WriteString("            </property>\n")
		case "action":
			fmt.Fprintf(&b, `            <property key=%q type="action" required="true">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			b.WriteString("            </property>\n")
		case "datasource":
			fmt.Fprintf(&b, `            <property key=%q type="datasource" required="true" isList="true">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			b.WriteString("            </property>\n")
		case "expression":
			fmt.Fprintf(&b, `            <property key=%q type="expression" required="true" defaultValue="true">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			b.WriteString("                <returnType type=\"Boolean\"/>\n")
			b.WriteString("            </property>\n")
		case "widgets":
			fmt.Fprintf(&b, `            <property key=%q type="widgets" required="false">`+"\n", p.Key)
			fmt.Fprintf(&b, `                <caption>%s</caption>`+"\n", human)
			b.WriteString("                <description/>\n")
			b.WriteString("            </property>\n")
		}
	}
	b.WriteString("        </propertyGroup>\n")
	return b.String()
}

func renderPropParams(props []PropertySpec) string {
	if len(props) == 0 {
		return "_props"
	}
	keys := make([]string, len(props))
	for i, p := range props {
		keys[i] = p.Key
	}
	return "{ " + strings.Join(keys, ", ") + " }"
}

func renderPropAttrs(props []PropertySpec) string {
	if len(props) == 0 {
		return ""
	}
	parts := make([]string, len(props))
	for i, p := range props {
		parts[i] = p.Key + "={" + p.Key + "}"
	}
	return strings.Join(parts, "\n        ")
}

func firstPropKey(props []PropertySpec) string {
	if len(props) == 0 {
		return "placeholder"
	}
	return props[0].Key
}

func firstPropDisplay(props []PropertySpec) string {
	if len(props) == 0 {
		return "\"WIDGET_NAME\""
	}
	p := props[0]
	if p.XMLType == "string" || p.XMLType == "attribute" {
		return p.Key + ` ?? "WIDGET_NAME"`
	}
	return "String(" + p.Key + `) ?? "WIDGET_NAME"`
}

func generateIcons(name string) []File {
	raw := minimalPNG()
	suffixes := []string{
		name + ".icon.png",
		name + ".icon.dark.png",
		name + ".tile.png",
		name + ".tile.dark.png",
	}
	var files []File
	for _, suf := range suffixes {
		files = append(files, File{
			Path:    "src/" + suf,
			Content: raw,
			Binary:  true,
		})
	}
	return files
}

func minimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Transparent)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
