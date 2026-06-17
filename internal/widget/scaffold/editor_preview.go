package scaffold

import "fmt"

type EditorPreviewRenderer struct{}

func (EditorPreviewRenderer) Render(spec Spec) []File {
	name := spec.Name
	content := fmt.Sprintf(`import { createElement } from 'react';
import { %[1]sSample } from "./components/%[1]sSample";

export function preview({ placeholder }) {
    return createElement(%[1]sSample, { sampleText: placeholder });
}

export function getPreviewCss() {
    return require("./ui/%[1]s.css");
}
`, name)
	return []File{{
		Path:    "src/" + name + ".editorPreview.jsx",
		Content: []byte(content),
	}}
}
