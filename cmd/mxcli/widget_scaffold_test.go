// SPDX-License-Identifier: Apache-2.0
package main

import (
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
			spec, err := parsePropertySpec(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parsePropertySpec(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePropertySpec(%q) unexpected error: %v", tc.input, err)
			}
			if spec.Key != tc.wantKey || spec.XMLType != tc.wantXML || spec.Subtype != tc.wantSub {
				t.Errorf("got {%q %q %q}, want {%q %q %q}",
					spec.Key, spec.XMLType, spec.Subtype,
					tc.wantKey, tc.wantXML, tc.wantSub)
			}
		})
	}
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
			got := deriveWidgetID(tc.name)
			if got != tc.want {
				t.Errorf("deriveWidgetID(%q) = %q, want %q", tc.name, got, tc.want)
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
			got := humanizeWidgetName(tc.input)
			if got != tc.want {
				t.Errorf("humanizeWidgetName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGenerateWidgetXML(t *testing.T) {
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
		{Key: "label", XMLType: "string"},
		{Key: "onChange", XMLType: "action"},
	}
	xml := generateWidgetXML("MySlider", "com.acme.widget.MySlider.MySlider", false, props)

	checks := []string{
		`id="com.acme.widget.MySlider.MySlider"`,
		`pluginWidget="true"`,
		`offlineCapable="false"`,
		`<name>My Slider</name>`,
		`key="value" type="attribute"`,
		`<attributeType name="Decimal"/>`,
		`key="label" type="string"`,
		`key="onChange" type="action"`,
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("generateWidgetXML: missing %q\ngot:\n%s", want, xml)
		}
	}
}

func TestGenerateWidgetXML_Offline(t *testing.T) {
	xml := generateWidgetXML("Foo", "com.a.b.c.Foo.Foo", true, nil)
	if !strings.Contains(xml, `offlineCapable="true"`) {
		t.Errorf("expected offlineCapable=true, got:\n%s", xml)
	}
}

func TestGeneratePackageXML(t *testing.T) {
	xml := generatePackageXML("MyPkg", []string{"WidgetA", "WidgetB"})
	checks := []string{
		`name="MyPkg"`,
		`version="1.0.0"`,
		`<widgetFile path="WidgetA.xml"/>`,
		`<widgetFile path="WidgetB.xml"/>`,
	}
	for _, want := range checks {
		if !strings.Contains(xml, want) {
			t.Errorf("generatePackageXML: missing %q\ngot:\n%s", want, xml)
		}
	}
}
