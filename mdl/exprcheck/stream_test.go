// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestStream_PeekAndConsume(t *testing.T) {
	s := NewStream(Lex("a + b"))
	if s.Peek().Kind != TokIdent {
		t.Fatalf("peek 1 kind = %v, want TokIdent", s.Peek().Kind)
	}
	s.Consume()
	if s.Peek().Kind != TokPlus {
		t.Fatalf("peek 2 kind = %v, want TokPlus", s.Peek().Kind)
	}
	s.Consume()
	if s.Peek().Kind != TokIdent {
		t.Fatalf("peek 3 kind = %v, want TokIdent", s.Peek().Kind)
	}
	s.Consume()
	if s.Peek().Kind != TokEOF {
		t.Fatalf("peek 4 kind = %v, want TokEOF", s.Peek().Kind)
	}
}
