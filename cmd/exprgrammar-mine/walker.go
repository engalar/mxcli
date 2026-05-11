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

func (l *slotListener) EnterIfStatement(ctx *parser.IfStatementContext) {
	for _, expr := range ctx.AllExpression() {
		if prc, ok := expr.(antlr.ParserRuleContext); ok {
			l.record("IfStmt.Condition", prc)
		}
	}
}
