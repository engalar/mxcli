// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ExitCreateJavaActionStatement handles CREATE JAVA ACTION statements.
func (b *Builder) ExitCreateJavaActionStatement(ctx *parser.CreateJavaActionStatementContext) {
	stmt := &ast.CreateJavaActionStmt{}

	// Get qualified name
	if qn := ctx.QualifiedName(); qn != nil {
		stmt.Name = buildQualifiedName(qn)
	}

	// Get parameters
	if paramList := ctx.JavaActionParameterList(); paramList != nil {
		for _, paramCtx := range paramList.AllJavaActionParameter() {
			param := ast.JavaActionParam{}
			if pn := paramCtx.ParameterName(); pn != nil {
				param.Name = parameterNameText(pn)
			}
			if dt := paramCtx.DataType(); dt != nil {
				param.Type = buildDataType(dt)
				// ENTITY <> anonymous type param: use the parameter name as type param name
				if param.Type.Kind == ast.TypeEntityTypeParam && param.Type.TypeParamName == "" {
					param.Type.TypeParamName = param.Name
				}
			}
			// Check for NOT NULL constraint
			if paramCtx.NOT_NULL() != nil {
				param.IsRequired = true
			}
			stmt.Parameters = append(stmt.Parameters, param)
		}
	}

	// Extract type parameters from ENTITY <pEntity> parameter declarations
	for _, param := range stmt.Parameters {
		if param.Type.Kind == ast.TypeEntityTypeParam && param.Type.TypeParamName != "" {
			found := false
			for _, existing := range stmt.TypeParameters {
				if existing == param.Type.TypeParamName {
					found = true
					break
				}
			}
			if !found {
				stmt.TypeParameters = append(stmt.TypeParameters, param.Type.TypeParamName)
			}
		}
	}

	// Get return type
	if retType := ctx.JavaActionReturnType(); retType != nil {
		if dt := retType.DataType(); dt != nil {
			stmt.ReturnType = buildJavaActionReturnType(dt)
		}
	}

	// Get exposed clause (EXPOSED AS 'caption' IN 'category')
	if exposed := ctx.JavaActionExposedClause(); exposed != nil {
		allStrings := exposed.AllSTRING_LITERAL()
		if len(allStrings) >= 2 {
			stmt.ExposedCaption = unquoteString(allStrings[0].GetText())
			stmt.ExposedCategory = unquoteString(allStrings[1].GetText())
		}
	}

	// Parse source blocks: imports / code / extra (or legacy AS $$ ... $$)
	if sc := ctx.JavaActionSourceClause(); sc != nil {
		for _, block := range sc.AllJavaActionSourceBlock() {
			raw := block.DOLLAR_STRING().GetText()
			// Strip $$ delimiters and trim
			content := strings.TrimSpace(raw[2 : len(raw)-2])

			if block.AS() != nil {
				// Backward compat: AS $$ ... $$ → code block with import extraction
				stmt.JavaCode, stmt.Imports = extractJavaImports(content)
				continue
			}
			switch strings.ToLower(block.IDENTIFIER().GetText()) {
			case "imports":
				for _, line := range strings.Split(content, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						stmt.Imports = append(stmt.Imports, line)
					}
				}
			case "code":
				stmt.JavaCode = content
			case "extra":
				stmt.ExtraCode = content
			}
		}
	}

	// Check for documentation comment and OR MODIFY/REPLACE from parent createStatement
	if parent, ok := ctx.GetParent().(*parser.CreateStatementContext); ok {
		if docComment := parent.DocComment(); docComment != nil {
			stmt.Documentation = extractDocComment(docComment.GetText())
		}
		if parent.OR() != nil && (parent.MODIFY() != nil || parent.REPLACE() != nil) {
			stmt.CreateOrModify = true
		}
	}

	// Also check for doc comment at statement level (grammar allows it at both levels)
	if stmt.Documentation == "" {
		if stmtCtx := findParentStatement(ctx); stmtCtx != nil {
			if docCtx := stmtCtx.DocComment(); docCtx != nil {
				stmt.Documentation = extractDocComment(docCtx.GetText())
			}
		}
	}

	b.statements = append(b.statements, stmt)
}

// ExitCreateJavaScriptActionStatement handles CREATE [OR MODIFY] JAVASCRIPT ACTION statements.
func (b *Builder) ExitCreateJavaScriptActionStatement(ctx *parser.CreateJavaScriptActionStatementContext) {
	stmt := &ast.CreateJavaScriptActionStmt{}

	if qn := ctx.QualifiedName(); qn != nil {
		stmt.Name = buildQualifiedName(qn)
	}

	if paramList := ctx.JavaActionParameterList(); paramList != nil {
		for _, paramCtx := range paramList.AllJavaActionParameter() {
			param := ast.JavaActionParam{}
			if pn := paramCtx.ParameterName(); pn != nil {
				param.Name = parameterNameText(pn)
			}
			if dt := paramCtx.DataType(); dt != nil {
				param.Type = buildDataType(dt)
			}
			if paramCtx.NOT_NULL() != nil {
				param.IsRequired = true
			}
			stmt.Parameters = append(stmt.Parameters, param)
		}
	}

	if retType := ctx.JavaActionReturnType(); retType != nil {
		if dt := retType.DataType(); dt != nil {
			stmt.ReturnType = buildJavaActionReturnType(dt)
		}
	}

	// PLATFORM 'value' then FOLDER 'value' — order matches grammar rule
	allStrings := ctx.AllSTRING_LITERAL()
	strIdx := 0
	if ctx.PLATFORM() != nil && strIdx < len(allStrings) {
		stmt.Platform = unquoteString(allStrings[strIdx].GetText())
		strIdx++
	}
	if ctx.FOLDER() != nil && strIdx < len(allStrings) {
		stmt.Folder = unquoteString(allStrings[strIdx].GetText())
		strIdx++
	}

	for _, block := range ctx.AllJsActionBodyBlock() {
		raw := block.DOLLAR_STRING().GetText()
		content := strings.TrimSpace(raw[2 : len(raw)-2])
		switch strings.ToLower(block.IDENTIFIER().GetText()) {
		case "imports":
			stmt.Imports = content
		case "extra":
			stmt.ExtraCode = content
		case "code":
			stmt.UserCode = content
		}
	}

	if parent, ok := ctx.GetParent().(*parser.CreateStatementContext); ok {
		if docComment := parent.DocComment(); docComment != nil {
			stmt.Documentation = extractDocComment(docComment.GetText())
		}
		if parent.OR() != nil && (parent.MODIFY() != nil || parent.REPLACE() != nil) {
			stmt.CreateOrModify = true
		}
	}
	if stmt.Documentation == "" {
		if stmtCtx := findParentStatement(ctx); stmtCtx != nil {
			if docCtx := stmtCtx.DocComment(); docCtx != nil {
				stmt.Documentation = extractDocComment(docCtx.GetText())
			}
		}
	}

	b.statements = append(b.statements, stmt)
}

func buildJavaActionReturnType(ctx parser.IDataTypeContext) ast.DataType {
	dt := buildDataType(ctx)
	if isVoidReturnType(dt) {
		return ast.DataType{Kind: ast.TypeVoid}
	}
	return dt
}

func isVoidReturnType(dt ast.DataType) bool {
	var name ast.QualifiedName
	switch dt.Kind {
	case ast.TypeVoid:
		return true
	case ast.TypeEntity:
		if dt.EntityRef == nil {
			return false
		}
		name = *dt.EntityRef
	case ast.TypeEnumeration:
		if dt.EnumRef == nil {
			return false
		}
		name = *dt.EnumRef
	default:
		return false
	}
	return name.Module == "" && strings.EqualFold(name.Name, "void")
}

// extractJavaImports separates `import ...;` lines from Java code.
// Lines matching the Java import statement pattern are returned as imports;
// the remaining lines form the method body. This handles the common case
// where AI agents prepend import statements inside the $$ block, which
// would otherwise end up as illegal Java inside executeAction().
func extractJavaImports(code string) (body string, imports []string) {
	var bodyLines []string
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") && strings.HasSuffix(trimmed, ";") {
			imports = append(imports, trimmed)
		} else {
			bodyLines = append(bodyLines, line)
		}
	}
	return strings.TrimSpace(strings.Join(bodyLines, "\n")), imports
}
