package scaffold

import (
	"fmt"
	"strings"
)

type EditorConfigRenderer struct{}

func (EditorConfigRenderer) Render(spec Spec) []File {
	caption := fmt.Sprintf("%s", spec.Name)
	for _, p := range spec.Properties {
		if p.Key == "label" {
			caption = fmt.Sprintf("props && props.label ? props.label : %q", spec.Name)
			break
		}
	}

	var b strings.Builder
	b.WriteString("/**\n")
	b.WriteString(" * @typedef Property\n")
	b.WriteString(" * @type {object}\n")
	b.WriteString(" * @property {string} key\n")
	b.WriteString(" * @property {string} caption\n")
	b.WriteString(" * @property {string} description\n")
	b.WriteString(" */\n\n")
	b.WriteString(fmt.Sprintf("export function getProperties(_values, defaultProperties, _target) {\n"))
	b.WriteString("    return defaultProperties;\n")
	b.WriteString("}\n")
	b.WriteString(fmt.Sprintf("\nexport function getCustomCaption(props) {\n    return %s;\n}\n", caption))
	b.WriteString(fmt.Sprintf("\nexport function getPreview(_props, isDarkMode) {\n"))
	b.WriteString("    return {\n")
	b.WriteString("        type: \"RowLayout\",\n")
	b.WriteString("        columnSize: \"grow\",\n")
	b.WriteString("        children: [{\n")
	b.WriteString("            type: \"Text\",\n")
	b.WriteString(fmt.Sprintf("            content: %q,\n", spec.Name))
	b.WriteString("            fontColor: isDarkMode ? \"#cba6f7\" : \"#89b4fa\",\n")
	b.WriteString("        }]\n")
	b.WriteString("    };\n")
	b.WriteString("}\n")

	return []File{{
		Path:    "src/" + spec.Name + ".editorConfig.js",
		Content: []byte(b.String()),
	}}
}
