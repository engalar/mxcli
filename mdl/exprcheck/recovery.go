// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "strings"

// consumeUntilSafe advances the stream past unrecognised tokens until
// it encounters a "safe boundary" — a token that an outer parser rule
// can plausibly resume on. The consumed text is concatenated for the
// RecoveredExpr.SourceFragment.
func consumeUntilSafe(s *Stream) string {
	var b strings.Builder
	for {
		t := s.Peek()
		if isSafeBoundary(t) {
			return b.String()
		}
		b.WriteString(t.Text)
		s.Consume()
	}
}

func isSafeBoundary(t Token) bool {
	switch t.Kind {
	case TokEOF, TokRParen, TokComma, TokPlus, TokMinus, TokStar,
		TokEq, TokNeq, TokLt, TokLe, TokGt, TokGe:
		return true
	case TokIdent:
		switch strings.ToLower(t.Text) {
		case "then", "else", "end", "and", "or", "not":
			return true
		}
	}
	return false
}
