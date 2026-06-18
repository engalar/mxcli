package scss

import (
	"fmt"
	"regexp"
	"strings"
)

// ScssVarOpts 是 SetVar 的操作选项。
type ScssVarOpts struct {
	InsertBefore string // import 语句关键字（如 "@import"），在该行前插入
	Force        bool   // 跳过变量名验证
	Comment      string // 行尾注释（自动加 //）
}

// ScssVarDecl 是变量声明。
type ScssVarDecl struct {
	Name      string // "$brand-primary" 或 "--brand-primary"
	Value     string // "#264ae5"
	IsDefault bool
	IsCSSVar  bool
	IsActive  bool   // false = 被注释掉了
	IsInRoot  bool   // 在 :root {} 块内
	Comment   string // 行尾注释内容
	LineIdx   int    // 在 Lines 中的索引
}

// ScssLine 是文件中的一行，保留原始文本。
type ScssLine struct {
	Raw string
}

// ScssDocument 是 SCSS 文件的内存模型，保留所有行的原始格式。
type ScssDocument struct {
	FilePath string
	Lines    []ScssLine
	Vars     []ScssVarDecl
}

// 匹配 SASS 变量: $name: value [!default] [;//comment]
var sassVarRe = regexp.MustCompile(`^\s*(\$[\w-]+)\s*:\s*(.+?)\s*(!default)?\s*;?\s*(//\s*(.*))?\s*$`)

// 匹配 CSS 自定义属性: --name: value [;//comment]
var cssVarRe = regexp.MustCompile(`^\s*(--[\w-]+)\s*:\s*(.+?)\s*;?\s*(//\s*(.*))?\s*$`)

// 注释前缀
var commentPrefixRe = regexp.MustCompile(`^\s*//`)

// :root 块进入
var rootOpenRe = regexp.MustCompile(`^\s*:root\s*\{`)

// Parse 解析 SCSS 文本内容，返回文档模型。
// 保留所有行（变量、注释、空行、import），保证 Write() 无损。
func Parse(filePath, content string) (*ScssDocument, error) {
	doc := &ScssDocument{
		FilePath: filePath,
	}
	lines := strings.Split(content, "\n")
	inRoot := false

	for i, line := range lines {
		doc.Lines = append(doc.Lines, ScssLine{Raw: line})
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Track :root block state; do NOT continue so same-line vars are caught
		if rootOpenRe.MatchString(trimmed) {
			inRoot = true
		}
		if trimmed == "}" && inRoot {
			inRoot = false
			continue
		}

		// Detect comment prefix and strip it so we can still match vars
		isCommented := commentPrefixRe.MatchString(trimmed)
		searchText := trimmed
		if isCommented {
			// Strip leading comment markers for regex matching
			searchText = commentPrefixRe.ReplaceAllString(searchText, "")
			searchText = strings.TrimSpace(searchText)
		}

		var decl *ScssVarDecl

		// Try SASS variable (search in comment-stripped text)
		if m := sassVarRe.FindStringSubmatch(searchText); m != nil {
			commentText := ""
			if m[5] != "" {
				commentText = strings.TrimSpace(m[5])
			}
			decl = &ScssVarDecl{
				Name:      m[1],
				Value:     strings.TrimSpace(m[2]),
				IsDefault: strings.TrimSpace(m[3]) == "!default",
				IsCSSVar:  false,
				IsActive:  !isCommented,
				IsInRoot:  inRoot,
				Comment:   commentText,
				LineIdx:   i,
			}
		}

		// Try CSS custom property
		if m := cssVarRe.FindStringSubmatch(searchText); m != nil && decl == nil {
			commentText := ""
			if m[4] != "" {
				commentText = strings.TrimSpace(m[4])
			}
			decl = &ScssVarDecl{
				Name:     m[1],
				Value:    strings.TrimSpace(m[2]),
				IsCSSVar: true,
				IsActive: !isCommented,
				IsInRoot: inRoot,
				Comment:  commentText,
				LineIdx:  i,
			}
		}

		if decl != nil {
			doc.Vars = append(doc.Vars, *decl)
		}
	}

	return doc, nil
}

// findVar 返回第一个匹配 name 的变量声明。
func (d *ScssDocument) findVar(name string) *ScssVarDecl {
	for i := range d.Vars {
		if d.Vars[i].Name == name {
			return &d.Vars[i]
		}
	}
	return nil
}

// SetVar 设置（新增或更新）一个变量的值。
func (d *ScssDocument) SetVar(name, value string, opts ScssVarOpts) error {
	// 尝试更新已有
	for i, v := range d.Vars {
		if v.Name == name {
			d.Vars[i].Value = value
			d.Vars[i].IsActive = true
			oldLine := d.Lines[v.LineIdx].Raw
			d.Lines[v.LineIdx].Raw = replaceVarValue(oldLine, v.Name, value)
			return nil
		}
	}

	// 新增
	var indent string
	if strings.HasPrefix(name, "--") {
		indent = "  "
	}
	line := fmt.Sprintf("%s%s: %s;", indent, name, value)

	insertAt := len(d.Lines)
	for i, l := range d.Lines {
		if opts.InsertBefore != "" && strings.Contains(l.Raw, opts.InsertBefore) {
			insertAt = i
			break
		}
	}

	d.Lines = append(d.Lines[:insertAt], append([]ScssLine{{Raw: line}}, d.Lines[insertAt:]...)...)

	decl := ScssVarDecl{
		Name:     name,
		Value:    value,
		IsCSSVar: strings.HasPrefix(name, "--"),
		IsActive: true,
		IsInRoot: strings.HasPrefix(name, "--"),
		LineIdx:  insertAt,
	}
	d.Vars = append(d.Vars, decl)

	for i := range d.Vars {
		if d.Vars[i].LineIdx >= insertAt && d.Vars[i].Name != name {
			d.Vars[i].LineIdx++
		}
	}

	return nil
}

// Write 将文档序列化回文本。
func (d *ScssDocument) Write() string {
	lines := make([]string, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = l.Raw
	}
	return strings.Join(lines, "\n")
}

// replaceVarValue 替换 SCSS 行中的变量值部分。
func replaceVarValue(line, name, newValue string) string {
	trimmed := strings.TrimLeft(line, " \t")
	prefix := line[:len(line)-len(trimmed)]

	re := regexp.MustCompile(`(\s*:\s*).+?\s*((!default)\s*)?;`)
	result := re.ReplaceAllString(trimmed, "${1}"+newValue+";")
	return prefix + result
}
