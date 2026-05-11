// SPDX-License-Identifier: Apache-2.0

package exprcheck

type Stream struct {
	toks []Token
	pos  int
}

func NewStream(toks []Token) *Stream {
	return &Stream{toks: toks}
}

func (s *Stream) Peek() Token {
	if s.pos < len(s.toks) {
		return s.toks[s.pos]
	}
	if n := len(s.toks); n > 0 {
		return Token{Kind: TokEOF, Pos: s.toks[n-1].Pos}
	}
	return Token{Kind: TokEOF}
}

func (s *Stream) Consume() Token {
	t := s.Peek()
	if s.pos < len(s.toks) {
		s.pos++
	}
	return t
}

func (s *Stream) Mark() int      { return s.pos }
func (s *Stream) Reset(mark int) { s.pos = mark }
