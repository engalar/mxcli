package scaffold

import "fmt"

type EditorPreviewRenderer struct{}

func (EditorPreviewRenderer) Render(spec Spec) []File {
	name := spec.Name

	propName := "placeholder"
	for _, p := range spec.Properties {
		propName = p.Key
		break
	}

	content := fmt.Sprintf(`import { %[1]sSample } from "./components/%[1]sSample";

export function preview() {
    return <%[1]sSample %[2]s={"Open"} />;
}

export function getPreviewCss() {
    return require("./ui/%[1]s.css");
}
`, name, propName)
	return []File{{
		Path:    "src/" + name + ".editorPreview.jsx",
		Content: []byte(content),
	}}
}
