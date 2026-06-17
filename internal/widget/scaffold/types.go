package scaffold

// Spec holds all parameters needed to scaffold a widget project.
type Spec struct {
	Name        string // PascalCase, e.g. "TextBox"
	PackageName string // lowercase, e.g. "textbox"
	WidgetID    string // e.g. "com.mendix.widget.custom.TextBox"
	PackagePath string // e.g. "com.mendix.widget.custom"
	ProjectPath string // e.g. "./tests/testProject"
	Properties  []PropertySpec
	Offline     bool
	Description string
}

// PropertySpec represents one widget property definition.
type PropertySpec struct {
	Key     string
	XMLType string
	Subtype string
}

// File represents a file to be written during scaffolding.
type File struct {
	Path    string // relative to project root
	Content []byte
	Binary  bool
}

// Renderer renders one or more files for a widget scaffold.
type Renderer interface {
	Render(spec Spec) []File
}
