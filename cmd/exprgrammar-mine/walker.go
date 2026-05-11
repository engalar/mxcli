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
	listener := &slotListener{miner: m, microflow: microflow}
	walker.Walk(listener, tree)
	return nil
}

type slotListener struct {
	*parser.BaseMDLParserListener
	miner     *Miner
	microflow string
}

func (l *slotListener) add(slotPath, source string) {
	l.miner.Records = append(l.miner.Records, SlotRecord{
		SlotPath:   slotPath,
		SourceText: source,
		Microflow:  l.microflow,
	})
}

func (l *slotListener) EnterIfStatement(ctx *parser.IfStatementContext) {
	for _, expr := range ctx.AllExpression() {
		l.add("IfStmt.Condition", expr.GetText())
	}
}
