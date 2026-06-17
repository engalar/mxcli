package scaffold

// Spec holds all parameters needed to scaffold a widget project from template.
type Spec struct {
	Name        string // PascalCase, e.g. "TextBox"
	PackageName string // lowercase, e.g. "textbox"
	WidgetID    string // e.g. "com.mendix.widget.custom.TextBox"
	PackagePath string // e.g. "com.mendix.widget.custom"
	ProjectPath string // e.g. "./tests/testProject"
	Properties  []PropertySpec
	Offline     bool
	Description string
	Author      string
	Copyright   string
}

// PropertySpec represents one widget property definition.
type PropertySpec struct {
	Key     string
	XMLType string
	Subtype string
}
