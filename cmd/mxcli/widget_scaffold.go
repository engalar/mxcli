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
