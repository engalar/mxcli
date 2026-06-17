package scaffold

import "fmt"

type EditorConfigRenderer struct{}

func (EditorConfigRenderer) Render(spec Spec) []File {
	hasLabel := false
	for _, p := range spec.Properties {
		if p.Key == "label" {
			hasLabel = true
			break
		}
	}
	captionBody := fmt.Sprintf(`return %q;`, spec.Name)
	if hasLabel {
		captionBody = fmt.Sprintf(`return props && props.label ? props.label : %q;`, spec.Name)
	}
	content := fmt.Sprintf(`"use strict";
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
`, captionBody, spec.Name)
	return []File{{
		Path:    "src/" + spec.Name + ".editorConfig.js",
		Content: []byte(content),
	}}
}
