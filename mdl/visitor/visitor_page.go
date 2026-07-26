// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ============================================================================
// Page Statements
// ============================================================================

// ExitCreatePageStatement is called when exiting the createPageStatement production.
func (b *Builder) ExitCreatePageStatement(ctx *parser.CreatePageStatementContext) {
	stmt := b.buildPageV3(ctx)
	b.statements = append(b.statements, stmt)
}

// ExitCreateLayoutStatement is called when exiting the createLayoutStatement production.
func (b *Builder) ExitCreateLayoutStatement(ctx *parser.CreateLayoutStatementContext) {
	s := &ast.CreateLayoutStmt{}

	if qn := ctx.QualifiedName(); qn != nil {
		s.Name = buildQualifiedName(qn)
	}

	if createStmt := findParentCreateStatement(ctx); createStmt != nil {
		if createStmt.OR() != nil {
			if createStmt.REPLACE() != nil {
				s.IsReplace = true
			}
			if createStmt.MODIFY() != nil {
				s.IsModify = true
			}
		}
		s.Documentation = findDocCommentText(ctx)
	}

	for _, prop := range ctx.AllLayoutHeaderProperty() {
		propCtx := prop.(*parser.LayoutHeaderPropertyContext)
		switch {
		case propCtx.TYPE() != nil:
			if str := propCtx.STRING_LITERAL(); str != nil {
				s.LayoutType = unquoteString(str.GetText())
			} else if iok := propCtx.IdentifierOrKeyword(); iok != nil {
				s.LayoutType = identifierOrKeywordText(iok)
			}
		case propCtx.FOLDER() != nil:
			if str := propCtx.STRING_LITERAL(); str != nil {
				s.Folder = unquoteString(str.GetText())
			}
		}
	}

	for _, wctx := range ctx.AllLayoutWidget() {
		widgetCtx := wctx.(*parser.LayoutWidgetContext)
		var w ast.LayoutWidgetV3
		if widgetCtx.SCROLLCONTAINER() != nil {
			w.Kind = ast.LayoutWidgetScrollContainer
			w.Name = identifierOrKeywordText(widgetCtx.IdentifierOrKeyword())
			for _, rctx := range widgetCtx.AllLayoutRegion() {
				regionCtx := rctx.(*parser.LayoutRegionContext)
				region := &ast.LayoutRegionV3{
					Name: strings.ToLower(regionCtx.LayoutRegionName().GetText()),
				}
				for _, pcctx := range regionCtx.AllLayoutRegionContent() {
					contentCtx := pcctx.(*parser.LayoutRegionContentContext)
					if contentCtx.PLACEHOLDER() != nil {
						region.Placeholders = append(region.Placeholders,
							&ast.LayoutPlaceholderV3{Name: identifierOrKeywordText(contentCtx.IdentifierOrKeyword())})
					} else if contentCtx.WidgetV3() != nil {
						widget := buildWidgetV3(contentCtx.WidgetV3(), b)
						region.Widgets = append(region.Widgets, widget)
					}
				}
				w.Regions = append(w.Regions, region)
			}
		} else if widgetCtx.PLACEHOLDER() != nil {
			w.Kind = ast.LayoutWidgetPlaceholder
			w.PlaceholderName = identifierOrKeywordText(widgetCtx.IdentifierOrKeyword())
		}
		s.Widgets = append(s.Widgets, &w)
	}

	b.statements = append(b.statements, s)
}

// ExitCreateSnippetStatement is called when exiting the createSnippetStatement production.
func (b *Builder) ExitCreateSnippetStatement(ctx *parser.CreateSnippetStatementContext) {
	stmt := b.buildSnippetV3(ctx)
	b.statements = append(b.statements, stmt)
}

// buildPageParameters converts page parameter list to []ast.PageParameter.
func buildPageParameters(ctx parser.IPageParameterListContext) []ast.PageParameter {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.PageParameterListContext)
	var params []ast.PageParameter

	for _, param := range listCtx.AllPageParameter() {
		paramCtx := param.(*parser.PageParameterContext)
		name := ""
		if id := paramCtx.IDENTIFIER(); id != nil {
			name = id.GetText()
		} else if v := paramCtx.VARIABLE(); v != nil {
			// VARIABLE token is $name, strip the $ prefix
			name = strings.TrimPrefix(v.GetText(), "$")
		}
		var entityType ast.QualifiedName
		var dataType ast.DataType
		if dt := paramCtx.DataType(); dt != nil {
			dataType = buildDataType(dt)
			// For backward compatibility, also populate EntityType for entity/enum refs
			dtCtx := dt.(*parser.DataTypeContext)
			if qn := dtCtx.QualifiedName(); qn != nil {
				entityType = buildQualifiedName(qn)
			}
		}
		params = append(params, ast.PageParameter{
			Name:       name,
			EntityType: entityType,
			Type:       dataType,
		})
	}
	return params
}

// buildSnippetParameters converts snippet parameter list to []ast.PageParameter.
func buildSnippetParameters(ctx parser.ISnippetParameterListContext) []ast.PageParameter {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.SnippetParameterListContext)
	var params []ast.PageParameter

	for _, param := range listCtx.AllSnippetParameter() {
		paramCtx := param.(*parser.SnippetParameterContext)
		name := ""
		if id := paramCtx.IDENTIFIER(); id != nil {
			name = strings.TrimPrefix(id.GetText(), "$")
		}
		if v := paramCtx.VARIABLE(); v != nil {
			name = strings.TrimPrefix(v.GetText(), "$")
		}
		var entityType ast.QualifiedName
		if dt := paramCtx.DataType(); dt != nil {
			dtCtx := dt.(*parser.DataTypeContext)
			if qn := dtCtx.QualifiedName(); qn != nil {
				entityType = buildQualifiedName(qn)
			}
		}
		params = append(params, ast.PageParameter{
			Name:       name,
			EntityType: entityType,
		})
	}
	return params
}
