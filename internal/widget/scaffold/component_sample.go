package scaffold

import (
	"fmt"
	"strings"
)

type ComponentSampleRenderer struct{}

func (ComponentSampleRenderer) Render(spec Spec) []File {
	name := spec.Name + "Sample"
	className := "widget-" + strings.ToLower(spec.Name)

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

	displayExpr := fmt.Sprintf("%q", spec.Name)
	if len(spec.Properties) > 0 {
		p := spec.Properties[0]
		if p.XMLType == "string" || p.XMLType == "attribute" {
			displayExpr = p.Key + " ?? " + fmt.Sprintf("%q", spec.Name)
		} else {
			displayExpr = fmt.Sprintf("String(%s) ?? %q", p.Key, spec.Name)
		}
	}

	content := fmt.Sprintf(`import classNames from "classnames";

export function %[1]s(%[2]s) {
    return (
        <div className={classNames(%[3]q)}>
            <span>%[4]s</span>
        </div>
    );
}
`, name, propsStr, className, displayExpr)
	return []File{{
		Path:    "src/components/" + name + ".jsx",
		Content: []byte(content),
	}}
}
