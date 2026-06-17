package scaffold

import "strings"

type WidgetCSSRenderer struct{}

func (WidgetCSSRenderer) Render(spec Spec) []File {
	className := ".widget-" + strings.ToLower(spec.Name)
	content := `/*
Place your custom CSS here
*/
` + className + ` {

}
`
	return []File{{
		Path:    "src/ui/" + spec.Name + ".css",
		Content: []byte(content),
	}}
}
