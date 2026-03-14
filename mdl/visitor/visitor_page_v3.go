// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// parseQualifiedName converts a string like "Module.Name" to ast.QualifiedName.
func parseQualifiedName(text string) ast.QualifiedName {
	parts := strings.Split(text, ".")
	if len(parts) == 1 {
		return ast.QualifiedName{Name: parts[0]}
	}
	return ast.QualifiedName{
		Module: parts[0],
		Name:   parts[len(parts)-1],
	}
}

// ============================================================================
// Page V3 Visitor Functions
// ============================================================================
//
// These functions handle the V3 page syntax with explicit properties.
// Pattern: WIDGET name (Prop: Value) { children }
//

// buildPageV3 builds a V3 page statement from the parse context.
func (b *Builder) buildPageV3(ctx *parser.CreatePageStatementContext) *ast.CreatePageStmtV3 {
	stmt := &ast.CreatePageStmtV3{}

	// Get page name
	if qn := ctx.QualifiedName(); qn != nil {
		stmt.Name = buildQualifiedName(qn)
	}

	// Check for CREATE OR REPLACE/MODIFY
	createStmt := findParentCreateStatement(ctx)
	if createStmt != nil {
		if createStmt.OR() != nil {
			if createStmt.REPLACE() != nil {
				stmt.IsReplace = true
			}
			if createStmt.MODIFY() != nil {
				stmt.IsModify = true
			}
		}
		stmt.Documentation = findDocCommentText(ctx)
	}

	// Parse V3 header
	if headerCtx := ctx.PageHeaderV3(); headerCtx != nil {
		b.parsePageHeaderV3(headerCtx, stmt)
	}

	// Parse V3 body
	if bodyCtx := ctx.PageBodyV3(); bodyCtx != nil {
		stmt.Widgets = buildPageBodyV3(bodyCtx, b)
	}

	return stmt
}

// parsePageHeaderV3 extracts properties from the V3 page header.
func (b *Builder) parsePageHeaderV3(ctx parser.IPageHeaderV3Context, stmt *ast.CreatePageStmtV3) {
	if ctx == nil {
		return
	}
	headerCtx := ctx.(*parser.PageHeaderV3Context)

	for _, propCtx := range headerCtx.AllPageHeaderPropertyV3() {
		prop := propCtx.(*parser.PageHeaderPropertyV3Context)

		if prop.PARAMS() != nil {
			// Params: { $Order: Entity, ... }
			if paramList := prop.PageParameterList(); paramList != nil {
				stmt.Parameters = buildPageParameters(paramList)
			}
		} else if prop.VARIABLES_KW() != nil {
			// Variables: { $showStock: Boolean = 'true', ... }
			if varList := prop.VariableDeclarationList(); varList != nil {
				stmt.Variables = buildVariableDeclarations(varList)
			}
		} else if prop.TITLE() != nil {
			// Title: 'My Page'
			if str := prop.STRING_LITERAL(); str != nil {
				stmt.Title = unquoteString(str.GetText())
			}
		} else if prop.LAYOUT() != nil {
			// Layout: Atlas_Core.Atlas_Default or 'Layout Name'
			if qn := prop.QualifiedName(); qn != nil {
				stmt.Layout = qn.GetText()
			} else if str := prop.STRING_LITERAL(); str != nil {
				stmt.Layout = unquoteString(str.GetText())
			}
		} else if prop.URL() != nil {
			// Url: 'my-page'
			if str := prop.STRING_LITERAL(); str != nil {
				stmt.URL = unquoteString(str.GetText())
			}
		} else if prop.FOLDER() != nil {
			// Folder: 'Pages/Admin'
			if str := prop.STRING_LITERAL(); str != nil {
				stmt.Folder = unquoteString(str.GetText())
			}
		}
	}
}

// buildSnippetV3 builds a V3 snippet statement from the parse context.
func (b *Builder) buildSnippetV3(ctx *parser.CreateSnippetStatementContext) *ast.CreateSnippetStmtV3 {
	stmt := &ast.CreateSnippetStmtV3{}

	// Get snippet name
	if qn := ctx.QualifiedName(); qn != nil {
		stmt.Name = buildQualifiedName(qn)
	}

	// Check for CREATE OR REPLACE/MODIFY
	createStmt := findParentCreateStatement(ctx)
	if createStmt != nil {
		if createStmt.OR() != nil {
			if createStmt.REPLACE() != nil {
				stmt.IsReplace = true
			}
			if createStmt.MODIFY() != nil {
				stmt.IsModify = true
			}
		}
		stmt.Documentation = findDocCommentText(ctx)
	}

	// Parse V3 header
	if headerCtx := ctx.SnippetHeaderV3(); headerCtx != nil {
		b.parseSnippetHeaderV3(headerCtx, stmt)
	}

	// Parse options (FOLDER)
	if opts := ctx.SnippetOptions(); opts != nil {
		optsCtx := opts.(*parser.SnippetOptionsContext)
		for _, opt := range optsCtx.AllSnippetOption() {
			optCtx := opt.(*parser.SnippetOptionContext)
			if optCtx.FOLDER() != nil && optCtx.STRING_LITERAL() != nil {
				stmt.Folder = unquoteString(optCtx.STRING_LITERAL().GetText())
			}
		}
	}

	// Parse V3 body
	if bodyCtx := ctx.PageBodyV3(); bodyCtx != nil {
		stmt.Widgets = buildPageBodyV3(bodyCtx, b)
	}

	return stmt
}

// parseSnippetHeaderV3 extracts properties from the V3 snippet header.
func (b *Builder) parseSnippetHeaderV3(ctx parser.ISnippetHeaderV3Context, stmt *ast.CreateSnippetStmtV3) {
	if ctx == nil {
		return
	}
	headerCtx := ctx.(*parser.SnippetHeaderV3Context)

	for _, propCtx := range headerCtx.AllSnippetHeaderPropertyV3() {
		prop := propCtx.(*parser.SnippetHeaderPropertyV3Context)

		if prop.PARAMS() != nil {
			// Params: { $Customer: Entity, ... }
			if paramList := prop.SnippetParameterList(); paramList != nil {
				stmt.Parameters = buildSnippetParameterListAsPage(paramList)
			}
		} else if prop.VARIABLES_KW() != nil {
			// Variables: { $showStock: Boolean = 'true', ... }
			if varList := prop.VariableDeclarationList(); varList != nil {
				stmt.Variables = buildVariableDeclarations(varList)
			}
		} else if prop.FOLDER() != nil {
			// Folder: 'Snippets/Common'
			if str := prop.STRING_LITERAL(); str != nil {
				stmt.Folder = unquoteString(str.GetText())
			}
		}
	}
}

// buildSnippetParameterListAsPage converts snippet parameters to page parameters.
func buildSnippetParameterListAsPage(ctx parser.ISnippetParameterListContext) []ast.PageParameter {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.SnippetParameterListContext)
	var params []ast.PageParameter

	for _, sp := range listCtx.AllSnippetParameter() {
		spCtx := sp.(*parser.SnippetParameterContext)
		param := ast.PageParameter{}

		if id := spCtx.IDENTIFIER(); id != nil {
			param.Name = id.GetText()
		} else if v := spCtx.VARIABLE(); v != nil {
			// VARIABLE token is $name, strip the $ prefix
			param.Name = strings.TrimPrefix(v.GetText(), "$")
		}

		if dt := spCtx.DataType(); dt != nil {
			param.EntityType = parseQualifiedName(dt.GetText())
		}

		params = append(params, param)
	}

	return params
}

// buildVariableDeclarations builds variable declarations from the parse context.
func buildVariableDeclarations(ctx parser.IVariableDeclarationListContext) []ast.PageVariable {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.VariableDeclarationListContext)
	var vars []ast.PageVariable

	for _, vd := range listCtx.AllVariableDeclaration() {
		vdCtx := vd.(*parser.VariableDeclarationContext)
		v := ast.PageVariable{}

		if varTok := vdCtx.VARIABLE(); varTok != nil {
			v.Name = strings.TrimPrefix(varTok.GetText(), "$")
		}

		if dt := vdCtx.DataType(); dt != nil {
			v.DataType = dt.GetText()
		}

		if str := vdCtx.STRING_LITERAL(); str != nil {
			v.DefaultValue = unquoteString(str.GetText())
		}

		vars = append(vars, v)
	}

	return vars
}

// buildPageBodyV3 extracts widgets from a V3 page body.
// Handles both widgetV3 and useFragmentRef children in parse-tree order.
func buildPageBodyV3(ctx parser.IPageBodyV3Context, b *Builder) []*ast.WidgetV3 {
	if ctx == nil {
		return nil
	}
	bodyCtx := ctx.(*parser.PageBodyV3Context)
	var widgets []*ast.WidgetV3

	// Process children in parse-tree order (widgets and fragment refs interleaved)
	for _, child := range bodyCtx.GetChildren() {
		switch c := child.(type) {
		case *parser.WidgetV3Context:
			if widget := buildWidgetV3(c, b); widget != nil {
				widgets = append(widgets, widget)
			}
		case *parser.UseFragmentRefContext:
			if ref := buildUseFragmentRef(c); ref != nil {
				widgets = append(widgets, ref)
			}
		}
	}

	return widgets
}

// buildUseFragmentRef creates a WidgetV3 with sentinel type USE_FRAGMENT.
func buildUseFragmentRef(ctx *parser.UseFragmentRefContext) *ast.WidgetV3 {
	if ctx == nil {
		return nil
	}
	w := &ast.WidgetV3{
		Type:       "USE_FRAGMENT",
		Properties: make(map[string]interface{}),
	}
	ids := ctx.AllIdentifierOrKeyword()
	if len(ids) > 0 {
		w.Name = identifierOrKeywordText(ids[0]) // Fragment name
	}
	if len(ids) > 1 {
		w.Properties["Prefix"] = identifierOrKeywordText(ids[1]) // Optional prefix
	}
	return w
}

// buildWidgetV3 builds a V3 widget from a widgetV3 context.
func buildWidgetV3(ctx parser.IWidgetV3Context, b *Builder) *ast.WidgetV3 {
	if ctx == nil {
		return nil
	}
	wCtx := ctx.(*parser.WidgetV3Context)

	widget := &ast.WidgetV3{
		Properties: make(map[string]any),
		Children:   []*ast.WidgetV3{},
	}

	// Get widget type
	if typeCtx := wCtx.WidgetTypeV3(); typeCtx != nil {
		widget.Type = strings.ToUpper(typeCtx.GetText())
	}

	// Get required identifier
	if id := wCtx.IDENTIFIER(); id != nil {
		widget.Name = id.GetText()
	}

	// Parse properties
	if propsCtx := wCtx.WidgetPropertiesV3(); propsCtx != nil {
		parseWidgetPropertiesV3(propsCtx, widget, b)
	}

	// Parse children
	if bodyCtx := wCtx.WidgetBodyV3(); bodyCtx != nil {
		widget.Children = buildWidgetBodyV3(bodyCtx, b)
	}

	return widget
}

// parseWidgetPropertiesV3 extracts properties from the widget properties context.
func parseWidgetPropertiesV3(ctx parser.IWidgetPropertiesV3Context, widget *ast.WidgetV3, b *Builder) {
	if ctx == nil {
		return
	}
	propsCtx := ctx.(*parser.WidgetPropertiesV3Context)

	for _, propCtx := range propsCtx.AllWidgetPropertyV3() {
		parseWidgetPropertyV3(propCtx, widget, b)
	}
}

// parseWidgetPropertyV3 extracts a single property.
func parseWidgetPropertyV3(ctx parser.IWidgetPropertyV3Context, widget *ast.WidgetV3, b *Builder) {
	if ctx == nil {
		return
	}
	propCtx := ctx.(*parser.WidgetPropertyV3Context)

	// DataSource: ...
	if propCtx.DATASOURCE() != nil {
		if dsCtx := propCtx.DataSourceExprV3(); dsCtx != nil {
			widget.Properties["DataSource"] = buildDataSourceV3(dsCtx)
		}
		return
	}

	// Attribute: ... (unified property for attribute bindings)
	if propCtx.ATTRIBUTE() != nil {
		if pathCtx := propCtx.AttributePathV3(); pathCtx != nil {
			widget.Properties["Attribute"] = buildAttributePathV3(pathCtx)
		}
		return
	}

	// Binds: ... (deprecated — hard error)
	if propCtx.BINDS() != nil {
		tok := propCtx.BINDS().GetSymbol()
		b.addError(fmt.Errorf("line %d:%d: 'Binds:' is no longer supported, use 'Attribute:' instead", tok.GetLine(), tok.GetColumn()))
		return
	}

	// Action: ...
	if propCtx.ACTION() != nil {
		if actCtx := propCtx.ActionExprV3(); actCtx != nil {
			widget.Properties["Action"] = buildActionV3(actCtx)
		}
		return
	}

	// Caption: ...
	if propCtx.CAPTION() != nil {
		if strCtx := propCtx.StringExprV3(); strCtx != nil {
			widget.Properties["Caption"] = buildStringExprV3(strCtx)
		}
		return
	}

	// Label: ...
	if propCtx.LABEL() != nil {
		if str := propCtx.STRING_LITERAL(); str != nil {
			widget.Properties["Label"] = unquoteString(str.GetText())
		}
		return
	}

	// Attr: ... (deprecated — hard error)
	if propCtx.ATTR() != nil {
		tok := propCtx.ATTR().GetSymbol()
		b.addError(fmt.Errorf("line %d:%d: 'Attr:' is no longer supported, use 'Attribute:' instead", tok.GetLine(), tok.GetColumn()))
		return
	}

	// Content: ...
	if propCtx.CONTENT() != nil {
		if strCtx := propCtx.StringExprV3(); strCtx != nil {
			widget.Properties["Content"] = buildStringExprV3(strCtx)
		}
		return
	}

	// RenderMode: ...
	if propCtx.RENDERMODE() != nil {
		if rmCtx := propCtx.RenderModeV3(); rmCtx != nil {
			widget.Properties["RenderMode"] = rmCtx.GetText()
		}
		return
	}

	// ContentParams: [...]
	if propCtx.CONTENTPARAMS() != nil {
		if plCtx := propCtx.ParamListV3(); plCtx != nil {
			widget.Properties["ContentParams"] = buildParamListV3(plCtx)
		}
		return
	}

	// CaptionParams: [...]
	if propCtx.CAPTIONPARAMS() != nil {
		if plCtx := propCtx.ParamListV3(); plCtx != nil {
			widget.Properties["CaptionParams"] = buildParamListV3(plCtx)
		}
		return
	}

	// ButtonStyle: ...
	if propCtx.BUTTONSTYLE() != nil {
		if styleCtx := propCtx.ButtonStyleV3(); styleCtx != nil {
			widget.Properties["ButtonStyle"] = styleCtx.GetText()
		}
		return
	}

	// Class: ...
	if propCtx.CLASS() != nil {
		if str := propCtx.STRING_LITERAL(); str != nil {
			widget.Properties["Class"] = unquoteString(str.GetText())
		}
		return
	}

	// Style: ...
	if propCtx.STYLE() != nil {
		if str := propCtx.STRING_LITERAL(); str != nil {
			widget.Properties["Style"] = unquoteString(str.GetText())
		}
		return
	}

	// DesktopWidth: ...
	if propCtx.DESKTOPWIDTH() != nil {
		if dwCtx := propCtx.DesktopWidthV3(); dwCtx != nil {
			text := dwCtx.GetText()
			if strings.EqualFold(text, "AutoFill") {
				widget.Properties["DesktopWidth"] = "AutoFill"
			} else {
				if n, err := strconv.Atoi(text); err == nil {
					widget.Properties["DesktopWidth"] = n
				} else {
					widget.Properties["DesktopWidth"] = text
				}
			}
		}
		return
	}

	// Where: and OrderBy: removed — now handled inline in dataSourceExprV3

	// Selection: ...
	if propCtx.SELECTION() != nil {
		if smCtx := propCtx.SelectionModeV3(); smCtx != nil {
			widget.Properties["Selection"] = smCtx.GetText()
		}
		return
	}

	// Snippet: ...
	if propCtx.SNIPPET() != nil {
		if qn := propCtx.QualifiedName(); qn != nil {
			widget.Properties["Snippet"] = qn.GetText()
		}
		return
	}

	// Attributes: [...] (for filter widgets)
	if propCtx.ATTRIBUTES() != nil {
		if attrListCtx := propCtx.AttributeListV3(); attrListCtx != nil {
			widget.Properties["Attributes"] = buildAttributeListV3(attrListCtx)
		}
		return
	}

	// FilterType: ... (for filter widgets)
	if propCtx.FILTERTYPE() != nil {
		if ftCtx := propCtx.FilterTypeValue(); ftCtx != nil {
			widget.Properties["FilterType"] = ftCtx.GetText()
		}
		return
	}

	// Width: number
	if propCtx.WIDTH() != nil {
		if num := propCtx.NUMBER_LITERAL(); num != nil {
			if n, err := strconv.Atoi(num.GetText()); err == nil {
				widget.Properties["Width"] = n
			}
		}
		return
	}

	// Height: number
	if propCtx.HEIGHT() != nil {
		if num := propCtx.NUMBER_LITERAL(); num != nil {
			if n, err := strconv.Atoi(num.GetText()); err == nil {
				widget.Properties["Height"] = n
			}
		}
		return
	}

	// DesignProperties: [...]
	if propCtx.DESIGNPROPERTIES() != nil {
		if dpCtx := propCtx.DesignPropertyListV3(); dpCtx != nil {
			widget.Properties["DesignProperties"] = buildDesignPropertyListV3(dpCtx)
		}
		return
	}

	// Visible: expression (keyword-based property)
	if propCtx.VISIBLE() != nil {
		if valCtx := propCtx.PropertyValueV3(); valCtx != nil {
			widget.Properties["Visible"] = buildPropertyValueV3(valCtx)
		}
		return
	}

	// Tooltip: 'text' (keyword-based property)
	if propCtx.TOOLTIP() != nil {
		if valCtx := propCtx.PropertyValueV3(); valCtx != nil {
			widget.Properties["Tooltip"] = buildPropertyValueV3(valCtx)
		}
		return
	}

	// Generic property: Identifier: value
	if id := propCtx.IDENTIFIER(); id != nil {
		if valCtx := propCtx.PropertyValueV3(); valCtx != nil {
			widget.Properties[id.GetText()] = buildPropertyValueV3(valCtx)
		}
		return
	}
}

// buildDataSourceV3 builds a DataSource from the parse context.
func buildDataSourceV3(ctx parser.IDataSourceExprV3Context) *ast.DataSourceV3 {
	if ctx == nil {
		return nil
	}
	dsCtx := ctx.(*parser.DataSourceExprV3Context)
	ds := &ast.DataSourceV3{}

	if v := dsCtx.VARIABLE(); v != nil {
		// $ParamName
		ds.Type = "parameter"
		ds.Reference = v.GetText()
	} else if dsCtx.DATABASE() != nil {
		// DATABASE [FROM] Entity [WHERE ...] [SORT BY ...]
		ds.Type = "database"
		if qn := dsCtx.QualifiedName(); qn != nil {
			ds.Reference = qn.GetText()
		}

		// Inline WHERE clause
		if dsCtx.WHERE() != nil {
			xpathConstraints := dsCtx.AllXpathConstraint()
			if len(xpathConstraints) > 0 {
				ds.Where = buildXPathString(xpathConstraints, dsCtx.AllAndOrXpath())
			} else if expr := dsCtx.Expression(); expr != nil {
				ds.Where = "[" + xpathExprToString(buildExpression(expr)) + "]"
			}
		}

		// Inline SORT BY clause
		if dsCtx.SORT_BY() != nil {
			for _, sc := range dsCtx.AllSortColumn() {
				ds.OrderBy = append(ds.OrderBy, buildSortColumnAsOrderBy(sc))
			}
		}
	} else if dsCtx.MICROFLOW() != nil {
		// MICROFLOW Module.Flow
		ds.Type = "microflow"
		if qn := dsCtx.QualifiedName(); qn != nil {
			ds.Reference = qn.GetText()
		}
		if argsCtx := dsCtx.MicroflowArgsV3(); argsCtx != nil {
			ds.Args = buildMicroflowArgsV3(argsCtx)
		}
	} else if dsCtx.NANOFLOW() != nil {
		// NANOFLOW Module.Flow
		ds.Type = "nanoflow"
		if qn := dsCtx.QualifiedName(); qn != nil {
			ds.Reference = qn.GetText()
		}
		if argsCtx := dsCtx.MicroflowArgsV3(); argsCtx != nil {
			ds.Args = buildMicroflowArgsV3(argsCtx)
		}
	} else if dsCtx.ASSOCIATION() != nil {
		// ASSOCIATION Path
		ds.Type = "association"
		if pathCtx := dsCtx.AttributePathV3(); pathCtx != nil {
			ds.Reference = buildAttributePathV3(pathCtx)
		}
	} else if dsCtx.SELECTION() != nil {
		// SELECTION widgetName
		ds.Type = "selection"
		if id := dsCtx.IDENTIFIER(); id != nil {
			ds.Reference = id.GetText()
		}
	}

	return ds
}

// buildActionV3 builds an Action from the parse context.
func buildActionV3(ctx parser.IActionExprV3Context) *ast.ActionV3 {
	if ctx == nil {
		return nil
	}
	actCtx := ctx.(*parser.ActionExprV3Context)
	action := &ast.ActionV3{}

	if actCtx.SAVE_CHANGES() != nil {
		action.Type = "save"
		action.ClosePage = actCtx.CLOSE_PAGE() != nil
	} else if actCtx.CANCEL_CHANGES() != nil {
		action.Type = "cancel"
		action.ClosePage = actCtx.CLOSE_PAGE() != nil
	} else if actCtx.CLOSE_PAGE() != nil && actCtx.SAVE_CHANGES() == nil && actCtx.CANCEL_CHANGES() == nil {
		action.Type = "close"
	} else if actCtx.DELETE_OBJECT() != nil {
		action.Type = "delete"
	} else if actCtx.DELETE() != nil {
		action.Type = "delete"
		action.ClosePage = actCtx.CLOSE_PAGE() != nil
	} else if actCtx.CREATE_OBJECT() != nil {
		action.Type = "create"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = qn.GetText()
		}
		// Check for THEN action
		if thenCtx := actCtx.ActionExprV3(); thenCtx != nil {
			action.ThenAction = buildActionV3(thenCtx)
		}
	} else if actCtx.SHOW_PAGE() != nil {
		action.Type = "showPage"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = qn.GetText()
		}
		if argsCtx := actCtx.MicroflowArgsV3(); argsCtx != nil {
			action.Args = buildMicroflowArgsV3(argsCtx)
		}
	} else if actCtx.MICROFLOW() != nil {
		action.Type = "microflow"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = qn.GetText()
		}
		if argsCtx := actCtx.MicroflowArgsV3(); argsCtx != nil {
			action.Args = buildMicroflowArgsV3(argsCtx)
		}
	} else if actCtx.NANOFLOW() != nil {
		action.Type = "nanoflow"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = qn.GetText()
		}
		if argsCtx := actCtx.MicroflowArgsV3(); argsCtx != nil {
			action.Args = buildMicroflowArgsV3(argsCtx)
		}
	} else if actCtx.OPEN_LINK() != nil {
		action.Type = "openLink"
		if str := actCtx.STRING_LITERAL(); str != nil {
			action.LinkURL = unquoteString(str.GetText())
		}
	} else if actCtx.SIGN_OUT() != nil {
		action.Type = "signOut"
	}

	return action
}

// buildMicroflowArgsV3 builds flow arguments from the parse context.
func buildMicroflowArgsV3(ctx parser.IMicroflowArgsV3Context) []ast.FlowArgV3 {
	if ctx == nil {
		return nil
	}
	argsCtx := ctx.(*parser.MicroflowArgsV3Context)
	var args []ast.FlowArgV3

	for _, argCtx := range argsCtx.AllMicroflowArgV3() {
		arg := buildMicroflowArgV3(argCtx)
		args = append(args, arg)
	}

	return args
}

// buildMicroflowArgV3 builds a single flow argument.
func buildMicroflowArgV3(ctx parser.IMicroflowArgV3Context) ast.FlowArgV3 {
	argCtx := ctx.(*parser.MicroflowArgV3Context)
	arg := ast.FlowArgV3{}

	if id := argCtx.IDENTIFIER(); id != nil {
		arg.Name = id.GetText()
	}
	if expr := argCtx.Expression(); expr != nil {
		arg.Value = expr.GetText()
	}

	return arg
}

// buildAttributeListV3 builds a list of attribute paths from the parse context.
func buildAttributeListV3(ctx parser.IAttributeListV3Context) []string {
	if ctx == nil {
		return nil
	}
	attrListCtx := ctx.(*parser.AttributeListV3Context)
	var attrs []string

	for _, qnCtx := range attrListCtx.AllQualifiedName() {
		attrs = append(attrs, qnCtx.GetText())
	}

	return attrs
}

// buildAttributePathV3 builds an attribute path string.
// Handles quoted identifiers (e.g., "Order") by stripping quotes.
func buildAttributePathV3(ctx parser.IAttributePathV3Context) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	// Strip double quotes or backticks from each path segment
	if strings.ContainsAny(text, "\"`") {
		parts := strings.Split(text, "/")
		for i, p := range parts {
			parts[i] = unquoteIdentifier(p)
		}
		return strings.Join(parts, "/")
	}
	return text
}

// buildStringExprV3 extracts string from stringExprV3.
// Can return either a quoted literal string or an unquoted attribute reference.
func buildStringExprV3(ctx parser.IStringExprV3Context) string {
	if ctx == nil {
		return ""
	}
	strCtx := ctx.(*parser.StringExprV3Context)

	// String literal: 'Hello {1}'
	if str := strCtx.STRING_LITERAL(); str != nil {
		return unquoteString(str.GetText())
	}

	// Attribute path: Name or Entity/Attr
	if attrPath := strCtx.AttributePathV3(); attrPath != nil {
		return attrPath.GetText()
	}

	// Variable reference: $var or $var.Attr
	if variable := strCtx.VARIABLE(); variable != nil {
		result := variable.GetText()
		// Check for .Attr suffix
		if dot := strCtx.DOT(); dot != nil {
			if id := strCtx.IDENTIFIER(); id != nil {
				result += "." + id.GetText()
			} else if kw := strCtx.Keyword(); kw != nil {
				result += "." + kw.GetText()
			}
		}
		return result
	}

	return ""
}

// buildParamListV3 builds parameter assignments from paramListV3.
func buildParamListV3(ctx parser.IParamListV3Context) []ast.ParamAssignmentV3 {
	if ctx == nil {
		return nil
	}
	plCtx := ctx.(*parser.ParamListV3Context)
	var params []ast.ParamAssignmentV3

	for _, paCtx := range plCtx.AllParamAssignmentV3() {
		params = append(params, buildParamAssignmentV3(paCtx))
	}

	return params
}

// buildParamAssignmentV3 builds a single parameter assignment.
func buildParamAssignmentV3(ctx parser.IParamAssignmentV3Context) ast.ParamAssignmentV3 {
	paCtx := ctx.(*parser.ParamAssignmentV3Context)
	param := ast.ParamAssignmentV3{}

	if num := paCtx.NUMBER_LITERAL(); num != nil {
		if n, err := strconv.Atoi(num.GetText()); err == nil {
			param.Index = n
		}
	}
	if expr := paCtx.Expression(); expr != nil {
		param.Value = expr.GetText()
	}

	return param
}

// buildXPathString builds a WHERE string from xpath constraints and and/or operators.
func buildXPathString(xpathConstraints []parser.IXpathConstraintContext, andOrOps []parser.IAndOrXpathContext) string {
	if len(xpathConstraints) == 0 {
		return ""
	}

	// Build AST expressions from each xpath constraint
	var exprs []ast.Expression
	for _, xc := range xpathConstraints {
		xcCtx := xc.(*parser.XpathConstraintContext)
		if expr := xcCtx.Expression(); expr != nil {
			exprs = append(exprs, buildExpression(expr))
		}
	}

	if len(exprs) == 0 {
		return ""
	}

	if len(exprs) == 1 {
		return "[" + xpathExprToString(exprs[0]) + "]"
	}

	// Check if any operator is OR
	hasOr := false
	for _, op := range andOrOps {
		opCtx := op.(*parser.AndOrXpathContext)
		if opCtx.OR() != nil {
			hasOr = true
			break
		}
	}

	if hasOr {
		// If any OR operator, combine into single bracket: [(expr1) op (expr2) ...]
		var parts []string
		for i, expr := range exprs {
			parts = append(parts, "("+xpathExprToString(expr)+")")
			if i < len(andOrOps) {
				opCtx := andOrOps[i].(*parser.AndOrXpathContext)
				if opCtx.OR() != nil {
					parts = append(parts, "or")
				} else {
					parts = append(parts, "and")
				}
			}
		}
		return "[" + strings.Join(parts, " ") + "]"
	}

	// All AND: keep as separate brackets [expr1][expr2]
	var sb strings.Builder
	for _, expr := range exprs {
		sb.WriteString("[" + xpathExprToString(expr) + "]")
	}
	return sb.String()
}

// buildSortColumnAsOrderBy converts a sortColumn context to an OrderByItemV3.
func buildSortColumnAsOrderBy(ctx parser.ISortColumnContext) ast.OrderByItemV3 {
	scCtx := ctx.(*parser.SortColumnContext)
	item := ast.OrderByItemV3{Direction: "ASC"}

	if qn := scCtx.QualifiedName(); qn != nil {
		item.Attribute = qn.GetText()
	} else if id := scCtx.IDENTIFIER(); id != nil {
		item.Attribute = id.GetText()
	}

	if scCtx.DESC() != nil {
		item.Direction = "DESC"
	}

	return item
}

// buildPropertyValueV3 builds a generic property value.
func buildPropertyValueV3(ctx parser.IPropertyValueV3Context) any {
	if ctx == nil {
		return nil
	}
	pvCtx := ctx.(*parser.PropertyValueV3Context)

	if str := pvCtx.STRING_LITERAL(); str != nil {
		return unquoteString(str.GetText())
	}
	if num := pvCtx.NUMBER_LITERAL(); num != nil {
		text := num.GetText()
		if strings.Contains(text, ".") {
			if f, err := strconv.ParseFloat(text, 64); err == nil {
				return f
			}
		}
		if n, err := strconv.Atoi(text); err == nil {
			return n
		}
		return text
	}
	if bl := pvCtx.BooleanLiteral(); bl != nil {
		return strings.EqualFold(bl.GetText(), "true")
	}
	if qn := pvCtx.QualifiedName(); qn != nil {
		return qn.GetText()
	}
	if id := pvCtx.IDENTIFIER(); id != nil {
		return id.GetText()
	}
	// Handle H1-H6 tokens (used for HeaderMode)
	for _, hFn := range []func() antlr.TerminalNode{pvCtx.H1, pvCtx.H2, pvCtx.H3, pvCtx.H4, pvCtx.H5, pvCtx.H6} {
		if h := hFn(); h != nil {
			return h.GetText()
		}
	}

	// Handle array values: [expr1, expr2, ...]
	if pvCtx.LBRACKET() != nil {
		var items []string
		for _, expr := range pvCtx.AllExpression() {
			items = append(items, expr.GetText())
		}
		return items
	}

	return pvCtx.GetText()
}

// buildDesignPropertyListV3 builds design properties from the parse context.
func buildDesignPropertyListV3(ctx parser.IDesignPropertyListV3Context) []ast.DesignPropertyEntryV3 {
	if ctx == nil {
		return nil
	}
	dpCtx := ctx.(*parser.DesignPropertyListV3Context)
	var props []ast.DesignPropertyEntryV3

	for _, entryCtx := range dpCtx.AllDesignPropertyEntryV3() {
		if entry := buildDesignPropertyEntryV3(entryCtx); entry != nil {
			props = append(props, *entry)
		}
	}

	return props
}

// buildDesignPropertyEntryV3 builds a single design property entry.
func buildDesignPropertyEntryV3(ctx parser.IDesignPropertyEntryV3Context) *ast.DesignPropertyEntryV3 {
	if ctx == nil {
		return nil
	}
	entryCtx := ctx.(*parser.DesignPropertyEntryV3Context)

	// Key is always the first STRING_LITERAL
	allStrings := entryCtx.AllSTRING_LITERAL()
	if len(allStrings) == 0 {
		return nil
	}

	key := unquoteString(allStrings[0].GetText())

	// Value: second STRING_LITERAL, ON, or OFF
	if entryCtx.ON() != nil {
		return &ast.DesignPropertyEntryV3{Key: key, Value: "ON"}
	}
	if entryCtx.OFF() != nil {
		return &ast.DesignPropertyEntryV3{Key: key, Value: "OFF"}
	}
	if len(allStrings) >= 2 {
		return &ast.DesignPropertyEntryV3{Key: key, Value: unquoteString(allStrings[1].GetText())}
	}

	return nil
}

// buildWidgetBodyV3 extracts children from a widget body.
func buildWidgetBodyV3(ctx parser.IWidgetBodyV3Context, b *Builder) []*ast.WidgetV3 {
	if ctx == nil {
		return nil
	}
	bodyCtx := ctx.(*parser.WidgetBodyV3Context)

	if pbCtx := bodyCtx.PageBodyV3(); pbCtx != nil {
		return buildPageBodyV3(pbCtx, b)
	}

	return nil
}

// ExitDefineFragmentStatement handles DEFINE FRAGMENT Name AS { widgets }.
func (b *Builder) ExitDefineFragmentStatement(ctx *parser.DefineFragmentStatementContext) {
	stmt := &ast.DefineFragmentStmt{}
	if iok := ctx.IdentifierOrKeyword(); iok != nil {
		stmt.Name = identifierOrKeywordText(iok)
	}
	if bodyCtx := ctx.PageBodyV3(); bodyCtx != nil {
		stmt.Widgets = buildPageBodyV3(bodyCtx, b)
	}
	b.statements = append(b.statements, stmt)
}

// xpathExprToString converts an AST Expression to a properly formatted XPath expression string.
// XPath uses lowercase boolean operators (and, or, not) and requires proper whitespace.
func xpathExprToString(expr ast.Expression) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		switch e.Kind {
		case ast.LiteralString:
			strVal := fmt.Sprintf("%v", e.Value)
			strVal = strings.ReplaceAll(strVal, `'`, `''`)
			return "'" + strVal + "'"
		case ast.LiteralBoolean:
			if e.Value.(bool) {
				return "true"
			}
			return "false"
		case ast.LiteralNull:
			return "empty"
		default:
			return fmt.Sprintf("%v", e.Value)
		}
	case *ast.VariableExpr:
		return "$" + e.Name
	case *ast.AttributePathExpr:
		return "$" + e.Variable + "/" + strings.Join(e.Path, "/")
	case *ast.BinaryExpr:
		left := xpathExprToString(e.Left)
		right := xpathExprToString(e.Right)
		op := strings.ToLower(e.Operator)
		return left + " " + op + " " + right
	case *ast.UnaryExpr:
		operand := xpathExprToString(e.Operand)
		op := strings.ToLower(e.Operator)
		return op + " " + operand
	case *ast.FunctionCallExpr:
		var args []string
		for _, arg := range e.Arguments {
			args = append(args, xpathExprToString(arg))
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")"
	case *ast.TokenExpr:
		return "[%" + e.Token + "%]"
	case *ast.ParenExpr:
		return "(" + xpathExprToString(e.Inner) + ")"
	case *ast.IdentifierExpr:
		return e.Name
	case *ast.QualifiedNameExpr:
		return e.QualifiedName.String()
	default:
		return ""
	}
}
