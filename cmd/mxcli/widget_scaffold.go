// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

// PropertySpec represents one parsed --property flag value (key:type[:subtype]).
type PropertySpec struct {
	Key     string
	XMLType string
	Subtype string
}

var validXMLTypes = map[string]bool{
	"attribute":  true,
	"string":     true,
	"integer":    true,
	"boolean":    true,
	"action":     true,
	"datasource": true,
	"expression": true,
	"widgets":    true,
}

// parsePropertySpec parses a --property flag value of the form key:type[:subtype].
func parsePropertySpec(s string) (PropertySpec, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return PropertySpec{}, fmt.Errorf("invalid property spec %q: must be key:type or key:type:subtype", s)
	}
	key, xmlType := parts[0], parts[1]
	if !validXMLTypes[xmlType] {
		return PropertySpec{}, fmt.Errorf("invalid property type %q in %q: must be one of attribute, string, integer, boolean, action, datasource, expression, widgets", xmlType, s)
	}
	subtype := ""
	if len(parts) == 3 {
		subtype = parts[2]
	}
	return PropertySpec{Key: key, XMLType: xmlType, Subtype: subtype}, nil
}

// deriveWidgetID returns the default widget ID for a widget named name.
func deriveWidgetID(name string) string {
	return fmt.Sprintf("com.mendix.widget.custom.%s.%s", name, name)
}

// humanizeWidgetName inserts a space before each uppercase letter (after the first).
func humanizeWidgetName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// xmlEscape escapes the five XML special characters in s.
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

// generateWidgetXML renders the widget property-definition XML for src/<Name>.xml.
func generateWidgetXML(name, widgetID, description string, offline bool, props []PropertySpec) string {
	human := humanizeWidgetName(name)
	offlineStr := "false"
	if offline {
		offlineStr = "true"
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(fmt.Sprintf(
		`<widget id=%q pluginWidget="true" offlineCapable=%q`+"\n"+
			`        xmlns="http://www.mendix.com/widget/1.0/"`+"\n"+
			`        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`+"\n"+
			`        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../../../../node_modules/mendix/custom_widget.xsd">`+"\n",
		widgetID, offlineStr))
	b.WriteString(fmt.Sprintf("  <name>%s</name>\n", human))
	b.WriteString(fmt.Sprintf("  <description>%s</description>\n", xmlEscape(description)))
	b.WriteString("  <properties>\n")
	b.WriteString("    <propertyGroup caption=\"General\">\n")
	for _, p := range props {
		b.WriteString(renderPropertyXML(p))
	}
	b.WriteString("    </propertyGroup>\n")
	b.WriteString("  </properties>\n")
	b.WriteString("</widget>\n")
	return b.String()
}

// renderPropertyXML renders a single <property> element for the widget XML.
func renderPropertyXML(p PropertySpec) string {
	var b strings.Builder
	human := humanizeWidgetName(p.Key)
	switch p.XMLType {
	case "attribute":
		attrType := p.Subtype
		if attrType == "" {
			attrType = "String"
		}
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"attribute\" required=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"        <attributeTypes><attributeType name=%q/></attributeTypes>\n"+
				"      </property>\n",
			p.Key, human, attrType))
	case "string":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"string\" required=\"true\" defaultValue=\"\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "integer":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"integer\" required=\"true\" defaultValue=\"0\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "boolean":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"boolean\" required=\"true\" defaultValue=\"false\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "action":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"action\" required=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "datasource":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"datasource\" required=\"true\" isList=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	case "expression":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"expression\" required=\"true\" defaultValue=\"true\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"        <returnType type=\"Boolean\"/>\n"+
				"      </property>\n",
			p.Key, human))
	case "widgets":
		b.WriteString(fmt.Sprintf(
			"      <property key=%q type=\"widgets\" required=\"false\">\n"+
				"        <caption>%s</caption>\n"+
				"        <description/>\n"+
				"      </property>\n",
			p.Key, human))
	}
	return b.String()
}

// generateJSX renders the React stub for src/<Name>.jsx.
func generateJSX(name string, props []PropertySpec) string {
	var params []string
	for _, p := range props {
		params = append(params, p.Key)
	}
	propsStr := ""
	if len(params) > 0 {
		propsStr = "{ " + strings.Join(params, ", ") + " }"
	} else {
		propsStr = "_props"
	}
	labelExpr := fmt.Sprintf("'%s'", name)
	for _, p := range props {
		if p.Key == "label" {
			labelExpr = "label ?? '" + name + "'"
			break
		}
	}
	return fmt.Sprintf(
		`import { createElement } from 'react';

export function %s(%s) {
    return createElement('div', { className: '%s' },
        createElement('span', null, %s),
        // TODO: implement
    );
}

export default %s;
`, name, propsStr, strings.ToLower(name), labelExpr, name)
}

// generateEditorConfig renders the Studio Pro design-time preview script.
func generateEditorConfig(name string, props []PropertySpec) string {
	hasLabel := false
	for _, p := range props {
		if p.Key == "label" {
			hasLabel = true
			break
		}
	}
	captionBody := fmt.Sprintf(`return %q;`, name)
	if hasLabel {
		captionBody = fmt.Sprintf(`return props && props.label ? props.label : %q;`, name)
	}
	return fmt.Sprintf(
		`"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getCustomCaption = function (props) {
    %s
};
exports.getPreview = function (props, isDarkMode) {
    return {
        type: "RowLayout",
        columnSize: "grow",
        children: [{
            type: "Text",
            content: %q,
            fontColor: isDarkMode ? "#cba6f7" : "#89b4fa",
        }]
    };
};
`, captionBody, name)
}

// generateEditorPreview renders the minimal browser-preview stub.
func generateEditorPreview() string {
	return `"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.preview = function () { return null; };
`
}

// generatePackageJSON renders the package.json with esbuild as the only dev dependency.
func generatePackageJSON(packageName string) string {
	return fmt.Sprintf(`{
  "name": %q,
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "devDependencies": {
    "esbuild": "^0.20.0"
  }
}
`, packageName)
}

// generateGitignore returns the contents of a widget project's .gitignore.
func generateGitignore() string {
	return "node_modules/\ndist/\n*.mpk\n"
}

// generateReadme renders README.md documenting the widget's build steps and properties.
func generateReadme(name, description string, props []PropertySpec) string {
	var b strings.Builder
	b.WriteString("# " + name + "\n\n")
	if description != "" {
		b.WriteString(description + "\n\n")
	}
	b.WriteString("## Build\n\n```bash\nmxcli widget build\n```\n\n")
	b.WriteString("## Install into a Mendix project\n\n```bash\nmxcli widget build --install -p /path/to/app.mpr\n```\n\n")
	if len(props) == 0 {
		return b.String()
	}
	b.WriteString("## Properties\n\n")
	b.WriteString("| Property | Type | Required |\n")
	b.WriteString("|----------|------|----------|\n")
	for _, p := range props {
		typeStr := p.XMLType
		if p.Subtype != "" {
			typeStr += " (" + p.Subtype + ")"
		}
		req := "Yes"
		if p.XMLType == "widgets" {
			req = "No"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", p.Key, typeStr, req))
	}
	b.WriteString("\n")
	return b.String()
}

// scaffoldRootFiles writes the project-root .gitignore and README.md into dir.
func scaffoldRootFiles(dir, name, description string, props []PropertySpec) error {
	files := map[string][]byte{
		".gitignore": []byte(generateGitignore()),
		"README.md":  []byte(generateReadme(name, description, props)),
	}
	for filename, content := range files {
		dest := filepath.Join(dir, filename)
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}
	return nil
}

// scaffoldWidget writes all source files for one widget into dir/src/.
func scaffoldWidget(dir, name, widgetID, description string, offline bool, props []PropertySpec) error {
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}
	files := map[string][]byte{
		name + ".xml":              []byte(generateWidgetXML(name, widgetID, description, offline, props)),
		name + ".jsx":              []byte(generateJSX(name, props)),
		name + ".editorConfig.js":  []byte(generateEditorConfig(name, props)),
		name + ".editorPreview.js": []byte(generateEditorPreview()),
		name + ".icon.png":         minimalPNG(),
		name + ".icon.dark.png":    minimalPNG(),
		name + ".tile.png":         minimalPNG(),
		name + ".tile.dark.png":    minimalPNG(),
	}
	for filename, content := range files {
		dest := filepath.Join(srcDir, filename)
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}
	return nil
}

// minimalPNG returns a minimal valid 1x1 transparent PNG image as bytes.
func minimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.Transparent)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// generatePackageXML renders package.xml — the MPK manifest listing all widget XML files.
func generatePackageXML(packageName string, widgetNames []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(`<package xmlns="http://www.mendix.com/package/1.0/">` + "\n")
	b.WriteString(fmt.Sprintf(
		`  <clientModule name=%q version="1.0.0" xmlns="http://www.mendix.com/clientModule/1.0/">`+"\n",
		packageName))
	if len(widgetNames) > 0 {
		b.WriteString("    <widgetFiles>\n")
		for _, name := range widgetNames {
			b.WriteString(fmt.Sprintf("      <widgetFile path=%q/>\n", name+".xml"))
		}
		b.WriteString("    </widgetFiles>\n")
	}
	b.WriteString("  </clientModule>\n")
	b.WriteString("</package>\n")
	return b.String()
}

// scaffoldPackage creates an empty multi-widget package project skeleton.
func scaffoldPackage(dir, name string) error {
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		return err
	}
	pkgName := strings.ToLower(name)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(generatePackageJSON(pkgName)), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "package.xml"), []byte(generatePackageXML(name, nil)), 0644)
}

// runWidgetNew implements `mxcli widget new <name>`.
func runWidgetNew(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("widget name required: mxcli widget new <name>")
	}
	name := args[0]
	isPackage, _ := cmd.Flags().GetBool("package")
	outDir := name

	if _, err := os.Stat(outDir); err == nil {
		return fmt.Errorf("directory %q already exists", outDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if isPackage {
		if err := scaffoldPackage(outDir, name); err != nil {
			return fmt.Errorf("scaffolding package: %w", err)
		}
		fmt.Printf("Created widget package project: %s/\n", outDir)
		fmt.Printf("  Add widgets: cd %s && mxcli widget add-widget <WidgetName>\n", outDir)
		fmt.Printf("  Build:       mxcli widget build\n")
		return nil
	}

	widgetID, _ := cmd.Flags().GetString("id")
	if widgetID == "" {
		widgetID = deriveWidgetID(name)
	} else {
		if err := validateWidgetIDFormat(widgetID); err != nil {
			return err
		}
	}
	offline, _ := cmd.Flags().GetBool("offline")
	description, _ := cmd.Flags().GetString("description")
	propStrs, _ := cmd.Flags().GetStringArray("property")
	var props []PropertySpec
	for _, s := range propStrs {
		p, err := parsePropertySpec(s)
		if err != nil {
			return err
		}
		props = append(props, p)
	}

	if err := scaffoldWidget(outDir, name, widgetID, description, offline, props); err != nil {
		return fmt.Errorf("scaffolding widget: %w", err)
	}
	if err := scaffoldRootFiles(outDir, name, description, props); err != nil {
		return fmt.Errorf("scaffolding root files: %w", err)
	}
	pkgName := strings.ToLower(name)
	if err := os.WriteFile(filepath.Join(outDir, "package.json"), []byte(generatePackageJSON(pkgName)), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "package.xml"), []byte(generatePackageXML(name, []string{name})), 0644); err != nil {
		return err
	}

	fmt.Printf("Created widget project: %s/\n", outDir)
	fmt.Printf("  Widget ID: %s\n", widgetID)
	fmt.Printf("  Edit:      %s/src/%s.jsx\n", outDir, name)
	fmt.Printf("  Build:     cd %s && mxcli widget build\n", outDir)
	return nil
}

// appendWidgetFileToPackageXML reads package.xml, adds a <widgetFile> entry for widgetName
// (if not already present), and writes it back. Inserts before </widgetFiles> if the
// container exists, otherwise creates a new <widgetFiles> block before </clientModule>.
func appendWidgetFileToPackageXML(pkgXMLPath, widgetName string) error {
	data, err := os.ReadFile(pkgXMLPath)
	if err != nil {
		return fmt.Errorf("reading package.xml: %w", err)
	}
	entry := fmt.Sprintf(`path="%s.xml"`, widgetName)
	if strings.Contains(string(data), entry) {
		return nil
	}
	content := string(data)
	if strings.Contains(content, "</widgetFiles>") {
		newEntry := fmt.Sprintf("      <widgetFile path=%q/>\n", widgetName+".xml")
		content = strings.Replace(content, "</widgetFiles>", newEntry+"    </widgetFiles>", 1)
	} else if strings.Contains(content, "</clientModule>") {
		block := fmt.Sprintf("    <widgetFiles>\n      <widgetFile path=%q/>\n    </widgetFiles>\n  ", widgetName+".xml")
		content = strings.Replace(content, "</clientModule>", block+"</clientModule>", 1)
	} else {
		return fmt.Errorf("could not find </widgetFiles> or </clientModule> in package.xml")
	}
	return os.WriteFile(pkgXMLPath, []byte(content), 0644)
}

// runWidgetAddWidget implements `mxcli widget add-widget <name>` (run inside package dir).
func runWidgetAddWidget(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("widget name required: mxcli widget add-widget <name>")
	}
	name := args[0]
	dir := "."

	pkgXMLPath := filepath.Join(dir, "package.xml")
	if _, err := os.Stat(pkgXMLPath); err != nil {
		return fmt.Errorf("package.xml not found in current directory — run this command inside a widget package project")
	}

	widgetID, _ := cmd.Flags().GetString("id")
	if widgetID == "" {
		widgetID = deriveWidgetID(name)
	}
	offline, _ := cmd.Flags().GetBool("offline")
	description, _ := cmd.Flags().GetString("description")
	propStrs, _ := cmd.Flags().GetStringArray("property")
	var props []PropertySpec
	for _, s := range propStrs {
		p, err := parsePropertySpec(s)
		if err != nil {
			return err
		}
		props = append(props, p)
	}

	if _, err := os.Stat(filepath.Join(dir, "src", name+".xml")); err == nil {
		return fmt.Errorf("widget %q already exists in src/", name)
	}

	if err := scaffoldWidget(dir, name, widgetID, description, offline, props); err != nil {
		return fmt.Errorf("scaffolding widget: %w", err)
	}
	if err := appendWidgetFileToPackageXML(pkgXMLPath, name); err != nil {
		return fmt.Errorf("updating package.xml: %w", err)
	}

	fmt.Printf("Added widget: %s\n", name)
	fmt.Printf("  Edit: src/%s.jsx\n", name)
	fmt.Printf("  Build: mxcli widget build\n")
	return nil
}
