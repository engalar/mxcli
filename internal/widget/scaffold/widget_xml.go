package scaffold

import "fmt"

type WidgetXMLRenderer struct{}

func (WidgetXMLRenderer) Render(spec Spec) []File {
	human := HumanizeWidgetName(spec.Name)
	offlineStr := "false"
	if spec.Offline {
		offlineStr = "true"
	}
	var b builder
	b.Line(`<?xml version="1.0" encoding="utf-8"?>`)
	b.Line(fmt.Sprintf(`<widget id=%q pluginWidget="true" needsEntityContext="true" offlineCapable=%q`, spec.WidgetID, offlineStr))
	b.Line(`        supportedPlatform="Web"`)
	b.Line(`        xmlns="http://www.mendix.com/widget/1.0/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`)
	b.Line(`        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../node_modules/mendix/custom_widget.xsd">`)
	b.Line(fmt.Sprintf(`  <name>%s</name>`, human))
	b.Line(fmt.Sprintf(`  <description>%s</description>`, xmlEscape(spec.Description)))
	b.Line(`  <properties>`)
	b.Line(`    <propertyGroup caption="General">`)
	for _, p := range spec.Properties {
		b.Write(renderPropertyXML(p))
	}
	b.Line(`    </propertyGroup>`)
	b.Line(`  </properties>`)
	b.Line(`</widget>`)
	return []File{{
		Path:    "src/" + spec.Name + ".xml",
		Content: []byte(b.String()),
	}}
}

type builder struct{ data string }

func (b *builder) Line(s string) { b.data += s + "\n" }
func (b *builder) Write(s string) { b.data += s }
func (b *builder) String() string { return b.data }

func renderPropertyXML(p PropertySpec) string {
	var b builder
	human := HumanizeWidgetName(p.Key)
	switch p.XMLType {
	case "attribute":
		attrType := p.Subtype
		if attrType == "" {
			attrType = "String"
		}
		b.Line(fmt.Sprintf(
			`      <property key=%q type="attribute" required="true">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(fmt.Sprintf(`        <attributeTypes><attributeType name=%q/></attributeTypes>`, attrType))
		b.Line(`      </property>`)
	case "string":
		b.Line(fmt.Sprintf(
			`      <property key=%q type="string" required="true" defaultValue="">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(`      </property>`)
	case "integer":
		b.Line(fmt.Sprintf(
			`      <property key=%q type="integer" required="true" defaultValue="0">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(`      </property>`)
	case "boolean":
		b.Line(fmt.Sprintf(
			`      <property key=%q type="boolean" required="true" defaultValue="false">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(`      </property>`)
	case "action":
		b.Line(fmt.Sprintf(
			`      <property key=%q type="action" required="true">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(`      </property>`)
	case "datasource":
		b.Line(fmt.Sprintf(
			`      <property key=%q type="datasource" required="true" isList="true">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(`      </property>`)
	case "expression":
		b.Line(fmt.Sprintf(
			`      <property key=%q type="expression" required="true" defaultValue="true">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(`        <returnType type="Boolean"/>`)
		b.Line(`      </property>`)
	case "widgets":
		b.Line(fmt.Sprintf(
			`      <property key=%q type="widgets" required="false">`, p.Key))
		b.Line(fmt.Sprintf(`        <caption>%s</caption>`, human))
		b.Line(`        <description/>`)
		b.Line(`      </property>`)
	}
	return b.String()
}
