package scaffold

import (
	"fmt"
	"strings"
)

type WidgetJSXRenderer struct{}

func (WidgetJSXRenderer) Render(spec Spec) []File {
	var params []string
	for _, p := range spec.Properties {
		params = append(params, p.Key)
	}
	propsStr := ""
	if len(params) > 0 {
		propsStr = "{ " + strings.Join(params, ", ") + " }"
	} else {
		propsStr = "_props"
	}
	content := fmt.Sprintf(`import { createElement } from 'react';
import { %[1]sSample } from "./components/%[1]sSample";
import "./ui/%[1]s.css";

export function %[1]s(%[2]s) {
    return createElement(%[1]sSample, { %[3]s });
}
`, spec.Name, propsStr, jsxSpreadAttrs(spec.Properties))
	return []File{{
		Path:    "src/" + spec.Name + ".jsx",
		Content: []byte(content),
	}}
}

func jsxSpreadAttrs(props []PropertySpec) string {
	if len(props) == 0 {
		return ""
	}
	var parts []string
	for _, p := range props {
		parts = append(parts, p.Key+": "+p.Key)
	}
	return strings.Join(parts, ", ")
}
