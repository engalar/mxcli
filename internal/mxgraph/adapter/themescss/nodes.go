package themescss

import (
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/internal/mxgraph/scss"
)

// varNode 从 ScssVarDecl 构造 mxgraph 节点。
func varNode(decl scss.ScssVarDecl, source, module, filePath string) *mxgraph.Node {
	category := inferCategory(decl.Name)
	varType := "sass"
	if decl.IsCSSVar {
		varType = "css-custom-property"
	}
	props := map[string]any{
		"Name":         decl.Name,
		"Value":        decl.Value,
		"VariableType": varType,
		"IsDefault":    decl.IsDefault,
		"IsActive":     decl.IsActive,
		"IsCSSVar":     decl.IsCSSVar,
		"Source":       source,
		"FilePath":     filePath,
		"LineNumber":   decl.LineIdx + 1,
		"Category":     category,
		"$Type":        "ThemeVariable",
	}
	if module != "" {
		props["Module"] = module
		props["QualifiedName"] = module + "." + decl.Name
	} else {
		props["QualifiedName"] = decl.Name
	}
	return &mxgraph.Node{
		ID:    mxgraph.NodeID(source + ":" + decl.Name),
		Label: "ThemeVariable",
		Props: props,
	}
}

// inferCategory 根据变量名前缀推断分类。
func inferCategory(name string) string {
	switch {
	case matchPrefix(name, "$brand", "--brand"):
		return "brand"
	case matchPrefix(name, "$font", "--font"):
		return "font"
	case matchPrefix(name, "$spacing", "--spacing"):
		return "spacing"
	case matchPrefix(name, "$nav", "--nav", "$navsidebar", "--navsidebar", "$navtopbar", "--navtopbar", "--navigation"):
		return "navigation"
	case matchPrefix(name, "$btn", "--btn"):
		return "button"
	case matchPrefix(name, "$form", "--form"):
		return "form"
	case matchPrefix(name, "$border", "--border"):
		return "border"
	case matchPrefix(name, "$bg", "--bg"):
		return "background"
	case matchPrefix(name, "$grid", "--grid"):
		return "grid"
	case matchPrefix(name, "$tab", "--tab"):
		return "tabs"
	case matchPrefix(name, "$modal", "--modal"):
		return "modal"
	case matchPrefix(name, "$card", "--card"):
		return "card"
	case matchPrefix(name, "$alert", "--alert"):
		return "alert"
	case matchPrefix(name, "$label", "--label"):
		return "label"
	case matchPrefix(name, "$shadow", "--shadow"):
		return "shadow"
	case matchPrefix(name, "$groupbox", "--groupbox"):
		return "groupbox"
	case matchPrefix(name, "$callout", "--callout"):
		return "callout"
	case matchPrefix(name, "$header", "--header"):
		return "header"
	case matchPrefix(name, "$link", "--link"):
		return "link"
	default:
		return "other"
	}
}

func matchPrefix(name string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}
