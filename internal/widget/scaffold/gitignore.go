package scaffold

type GitignoreRenderer struct{}

func (GitignoreRenderer) Render(spec Spec) []File {
	return []File{{
		Path: ".gitignore",
		Content: []byte(`tests/testProject/
.DS_Store
.idea
.vscode
dist
node_modules
.env
*.log
*.bak
*.launch
mxproject
coverage

**/results
mendixProject
**/e2e/diffs
**/screenshot
**/screenshot-results
**/tests/testProject
**/artifacts
`),
	}}
}
