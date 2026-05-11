// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestLexer_BasicTokens(t *testing.T) {
	cases := []struct {
		in   string
		kind []TokKind
	}{
		{`'hello'`, []TokKind{TokString, TokEOF}},
		{`123`, []TokKind{TokNumber, TokEOF}},
		{`true`, []TokKind{TokIdent, TokEOF}},
		{`empty`, []TokKind{TokIdent, TokEOF}},
		{`null`, []TokKind{TokIdent, TokEOF}},
		{`$Var`, []TokKind{TokDollarIdent, TokEOF}},
		{`$x/Attr`, []TokKind{TokDollarIdent, TokSlash, TokIdent, TokEOF}},
		{`Module.Enum.Value`, []TokKind{TokIdent, TokDot, TokIdent, TokDot, TokIdent, TokEOF}},
		{`@Module.Const`, []TokKind{TokAt, TokIdent, TokDot, TokIdent, TokEOF}},
		{`[%CurrentDateTime%]`, []TokKind{TokToken, TokEOF}},
		{`length(x)`, []TokKind{TokIdent, TokLParen, TokIdent, TokRParen, TokEOF}},
		{`a + b`, []TokKind{TokIdent, TokPlus, TokIdent, TokEOF}},
		{`a = b`, []TokKind{TokIdent, TokEq, TokIdent, TokEOF}},
		{`a != b`, []TokKind{TokIdent, TokNeq, TokIdent, TokEOF}},
		{`a <> b`, []TokKind{TokIdent, TokNeq, TokIdent, TokEOF}},
	}
	for _, c := range cases {
		toks := Lex(c.in)
		if len(toks) != len(c.kind) {
			t.Errorf("%q: got %d tokens, want %d (%+v)", c.in, len(toks), len(c.kind), toks)
			continue
		}
		for i, k := range c.kind {
			if toks[i].Kind != k {
				t.Errorf("%q: token %d kind = %v, want %v", c.in, i, toks[i].Kind, k)
			}
		}
	}
}

func TestLexer_ErrorToken(t *testing.T) {
	toks := Lex(`length(@@@x)`)
	var sawErr bool
	for _, tok := range toks {
		if tok.Kind == TokError {
			sawErr = true
		}
	}
	if !sawErr {
		t.Fatalf("expected TokError in %+v", toks)
	}
}
