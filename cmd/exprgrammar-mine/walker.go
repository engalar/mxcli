// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

func WalkMDL(m *Miner, microflow, source string) error {
	is := antlr.NewInputStream(source)
	lex := parser.NewMDLLexer(is)
	lex.RemoveErrorListeners()
	toks := antlr.NewCommonTokenStream(lex, antlr.TokenDefaultChannel)
	p := parser.NewMDLParser(toks)
	p.RemoveErrorListeners()
	tree := p.Program()
	if tree == nil {
		return fmt.Errorf("parse returned nil tree")
	}
	walker := antlr.NewParseTreeWalker()
	listener := &slotListener{miner: m, microflow: microflow, input: is}
	walker.Walk(listener, tree)
	return nil
}

type slotListener struct {
	*parser.BaseMDLParserListener
	miner     *Miner
	microflow string
	input     *antlr.InputStream
}

func (l *slotListener) sourceText(ctx antlr.ParserRuleContext) string {
	start, stop := ctx.GetStart(), ctx.GetStop()
	if start == nil || stop == nil {
		return ctx.GetText()
	}
	return l.input.GetTextFromInterval(antlr.NewInterval(start.GetStart(), stop.GetStop()))
}

func (l *slotListener) record(slotPath string, ctx antlr.ParserRuleContext) {
	l.miner.Records = append(l.miner.Records, SlotRecord{
		SlotPath:   slotPath,
		SourceText: l.sourceText(ctx),
		Microflow:  l.microflow,
	})
}

func (l *slotListener) recordExpr(slotPath string, expr parser.IExpressionContext) {
	if expr == nil {
		return
	}
	if prc, ok := expr.(antlr.ParserRuleContext); ok {
		l.record(slotPath, prc)
	}
}

func (l *slotListener) EnterIfStatement(ctx *parser.IfStatementContext) {
	for _, expr := range ctx.AllExpression() {
		l.recordExpr("IfStmt.Condition", expr)
	}
}

func (l *slotListener) EnterDeclareStatement(ctx *parser.DeclareStatementContext) {
	l.recordExpr("DeclareStmt.InitialValue", ctx.Expression())
}

func (l *slotListener) EnterSetStatement(ctx *parser.SetStatementContext) {
	l.recordExpr("MfSetStmt.Value", ctx.Expression())
}

func (l *slotListener) EnterWhileStatement(ctx *parser.WhileStatementContext) {
	l.recordExpr("WhileStmt.Condition", ctx.Expression())
}

func (l *slotListener) EnterRetrieveStatement(ctx *parser.RetrieveStatementContext) {
	l.recordExpr("RetrieveStmt.LimitExpr", ctx.GetLimitExpr())
	l.recordExpr("RetrieveStmt.OffsetExpr", ctx.GetOffsetExpr())
}

func (l *slotListener) EnterLogStatement(ctx *parser.LogStatementContext) {
	exprs := ctx.AllExpression()
	if len(exprs) == 0 {
		return
	}
	l.recordExpr("LogStmt.Message", exprs[len(exprs)-1])
}

func (l *slotListener) EnterReturnStatement(ctx *parser.ReturnStatementContext) {
	l.recordExpr("ReturnStmt.Value", ctx.Expression())
}
