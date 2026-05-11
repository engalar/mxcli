// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestConsumeUntilSafe_StopsAtRParen(t *testing.T) {
	s := NewStream(Lex("@@@broken@@@) more"))
	salvage := consumeUntilSafe(s)
	if salvage != "@@@broken@@@" {
		t.Errorf("salvage = %q", salvage)
	}
	if s.Peek().Kind != TokRParen {
		t.Errorf("next = %v, want TokRParen", s.Peek().Kind)
	}
}

func TestConsumeUntilSafe_StopsAtComma(t *testing.T) {
	s := NewStream(Lex("@@@a@@@, b"))
	salvage := consumeUntilSafe(s)
	if salvage != "@@@a@@@" {
		t.Errorf("salvage = %q", salvage)
	}
	if s.Peek().Kind != TokComma {
		t.Errorf("next = %v, want TokComma", s.Peek().Kind)
	}
}

func TestConsumeUntilSafe_StopsAtKeyword(t *testing.T) {
	s := NewStream(Lex("@@@a@@@ then b"))
	salvage := consumeUntilSafe(s)
	if salvage != "@@@a@@@" {
		t.Errorf("salvage = %q", salvage)
	}
}

func TestConsumeUntilSafe_StopsAtEOF(t *testing.T) {
	s := NewStream(Lex("@@@trailing@@@"))
	salvage := consumeUntilSafe(s)
	if salvage != "@@@trailing@@@" {
		t.Errorf("salvage = %q", salvage)
	}
	if s.Peek().Kind != TokEOF {
		t.Errorf("not at EOF: %v", s.Peek())
	}
}
