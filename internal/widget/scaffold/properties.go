package scaffold

import (
	"fmt"
	"strings"
	"unicode"
)

// ParsePropertySpec parses a --property flag value of the form key:type[:subtype].
func ParsePropertySpec(s string) (PropertySpec, error) {
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

// DeriveWidgetID returns the canonical widget ID for the given package path and name.
// The middle segment (package name) is lowercased to match the directory that
// pluggable-widgets-tools emits, e.g. packagePath="com.mendix.widget.custom",
// name="MySlider" → "com.mendix.widget.custom.myslider.MySlider".
func DeriveWidgetID(packagePath, name string) string {
	return fmt.Sprintf("%s.%s.%s", packagePath, strings.ToLower(name), name)
}

// ValidateWidgetIDFormat checks that a widget ID has at least 4 dot-separated segments.
func ValidateWidgetIDFormat(id string) error {
	parts := strings.Split(id, ".")
	if len(parts) < 4 {
		return fmt.Errorf("widget ID must have at least 4 dot-separated segments (e.g. com.acme.widget.MyName), got %q", id)
	}
	return nil
}

// HumanizeWidgetName inserts a space before each uppercase letter (after the first).
func HumanizeWidgetName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
