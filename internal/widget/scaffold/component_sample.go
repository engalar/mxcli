package scaffold

import (
	"fmt"
	"strings"
)

type ComponentSampleRenderer struct{}

func (ComponentSampleRenderer) Render(spec Spec) []File {
	var params []string
	for _, p := range spec.Properties {
		if p.Key == "label" {
			params = append(params, "label")
		} else {
			params = append(params, p.Key+": "+p.Key)
		}
	}
	propsStr := "{ " + strings.Join(params, ", ") + " }"
	if len(params) == 0 {
		propsStr = "_props"
	}
	name := spec.Name + "Sample"
	className := strings.ToLower(spec.Name)
	labelExpr := fmt.Sprintf("'%s'", spec.Name)
	for _, p := range spec.Properties {
		if p.Key == "label" {
			labelExpr = "label ?? '" + spec.Name + "'"
			break
		}
	}
	content := fmt.Sprintf(`import { createElement } from 'react';

export function %[1]s(%[2]s) {
    return createElement('div', { className: %[4]q },
        createElement('span', null, %[3]s),
    );
}
`, name, propsStr, labelExpr, "widget-"+className)
	return []File{{
		Path:    "src/components/" + name + ".jsx",
		Content: []byte(content),
	}}
}
