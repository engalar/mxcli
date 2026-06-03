// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ExitTranslateStatement builds a TranslateStmt from a parsed
// TRANSLATE <docType> <qname> IN <lang> SET path = 'text', ... statement.
func (b *Builder) ExitTranslateStatement(ctx *parser.TranslateStatementContext) {
	stmt := &ast.TranslateStmt{}

	if dt := ctx.TranslateDocType(); dt != nil {
		stmt.DocType = strings.ToUpper(dt.GetText())
	}
	if qn := ctx.QualifiedName(); qn != nil {
		stmt.QName = buildQualifiedName(qn)
	}
	if iok := ctx.IdentifierOrKeyword(); iok != nil {
		stmt.Lang = iok.GetText()
	}

	for _, opCtx := range ctx.AllTranslateSetOp() {
		op, ok := opCtx.(*parser.TranslateSetOpContext)
		if !ok {
			continue
		}
		var path string
		if p, ok := op.TranslatePath().(*parser.TranslatePathContext); ok {
			parts := make([]string, 0, 2)
			for _, ik := range p.AllIdentifierOrKeyword() {
				parts = append(parts, ik.GetText())
			}
			path = strings.Join(parts, ".")
		}
		text := unquoteString(op.STRING_LITERAL().GetText())
		stmt.Ops = append(stmt.Ops, ast.TranslateSetOp{Path: path, Text: text})
	}

	b.statements = append(b.statements, stmt)
}

// ExitTranslateMicroflowStatement builds a TranslateMicroflowStmt from a parsed
// TRANSLATE MICROFLOW <qname> IN <lang> SET ActionType[index].property = 'text'.
func (b *Builder) ExitTranslateMicroflowStatement(ctx *parser.TranslateMicroflowStatementContext) {
	stmt := &ast.TranslateMicroflowStmt{}

	if qn := ctx.QualifiedName(); qn != nil {
		stmt.QName = buildQualifiedName(qn)
	}
	if iok := ctx.IdentifierOrKeyword(); iok != nil {
		stmt.Lang = iok.GetText()
	}

	for _, opCtx := range ctx.AllTranslateMicroflowSetOp() {
		op, ok := opCtx.(*parser.TranslateMicroflowSetOpContext)
		if !ok {
			continue
		}
		idents := op.AllIdentifierOrKeyword()
		if len(idents) < 2 {
			continue
		}
		index, _ := strconv.Atoi(op.NUMBER_LITERAL().GetText())
		stmt.Ops = append(stmt.Ops, ast.TranslateMicroflowSetOp{
			ActionType: idents[0].GetText(),
			Index:      index,
			Property:   idents[1].GetText(),
			Text:       unquoteString(op.STRING_LITERAL().GetText()),
		})
	}

	b.statements = append(b.statements, stmt)
}
