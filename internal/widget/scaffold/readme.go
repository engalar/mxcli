package scaffold

import "fmt"

type ReadmeRenderer struct{}

func (ReadmeRenderer) Render(spec Spec) []File {
	var b builder
	b.Line("# " + spec.Name)
	b.Line("")
	if spec.Description != "" {
		b.Line(spec.Description)
		b.Line("")
	}
	b.Line("## Build")
	b.Line("")
	b.Line("```bash")
	b.Line("npm run build")
	b.Line("```")
	b.Line("")
	b.Line("## Install into a Mendix project")
	b.Line("")
	b.Line("```bash")
	b.Line("mxcli widget install --project /path/to/app.mpr")
	b.Line("```")
	b.Line("")
	if len(spec.Properties) > 0 {
		b.Line("## Properties")
		b.Line("")
		b.Line("| Property | Type | Required |")
		b.Line("|----------|------|----------|")
		for _, p := range spec.Properties {
			typeStr := p.XMLType
			if p.Subtype != "" {
				typeStr += " (" + p.Subtype + ")"
			}
			req := "Yes"
			if p.XMLType == "widgets" {
				req = "No"
			}
			b.Line(fmt.Sprintf("| %s | %s | %s |", p.Key, typeStr, req))
		}
		b.Line("")
	}
	return []File{{
		Path:    "README.md",
		Content: []byte(b.String()),
	}}
}
