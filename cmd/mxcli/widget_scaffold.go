// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"
	"unicode"
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

// generateWidgetXML renders the widget property-definition XML for src/<Name>.xml.
func generateWidgetXML(name, widgetID string, offline bool, props []PropertySpec) string {
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
	b.WriteString("  <description></description>\n")
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

// generatePackageXML renders package.xml — the MPK manifest listing all widget XML files.
func generatePackageXML(packageName string, widgetNames []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(`<package xmlns="http://www.mendix.com/package/1.0/">` + "\n")
	b.WriteString(fmt.Sprintf(
		`  <clientModule name=%q version="1.0.0" xmlns="http://www.mendix.com/clientmodule/1.0/">`+"\n",
		packageName))
	b.WriteString("    <widgetFiles>\n")
	for _, name := range widgetNames {
		b.WriteString(fmt.Sprintf("      <widgetFile path=%q/>\n", name+".xml"))
	}
	b.WriteString("    </widgetFiles>\n")
	b.WriteString("  </clientModule>\n")
	b.WriteString("</package>\n")
	return b.String()
}
