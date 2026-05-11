// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "strings"

type parserImpl struct{}

func NewParser() Parser { return &parserImpl{} }

func (p *parserImpl) Parse(src string, ctx Context) (RobustExpr, []Hint) {
	s := NewStream(Lex(src))
	var hints []Hint
	expr, h := parseOr(s, ctx)
	hints = append(hints, h...)
	return expr, hints
}

func parseOr(s *Stream, ctx Context) (RobustExpr, []Hint) {
	left, hints := parseAnd(s, ctx)
	for matchKeyword(s, "or") {
		right, h := parseAnd(s, ctx)
		left = &BinExpr{Op: "OR", L: left, R: right}
		hints = append(hints, h...)
	}
	return left, hints
}

func parseAnd(s *Stream, ctx Context) (RobustExpr, []Hint) {
	left, hints := parseNot(s, ctx)
	for matchKeyword(s, "and") {
		right, h := parseNot(s, ctx)
		left = &BinExpr{Op: "AND", L: left, R: right}
		hints = append(hints, h...)
	}
	return left, hints
}

func parseNot(s *Stream, ctx Context) (RobustExpr, []Hint) {
	if matchKeyword(s, "not") {
		inner, h := parseCmp(s, ctx)
		return &UnaryExpr{Op: "NOT", Operand: inner}, h
	}
	return parseCmp(s, ctx)
}

func parseCmp(s *Stream, ctx Context) (RobustExpr, []Hint) {
	left, hints := parseAdd(s, ctx)
	op := ""
	switch s.Peek().Kind {
	case TokEq:
		op = "="
	case TokNeq:
		op = "!="
	case TokLt:
		op = "<"
	case TokLe:
		op = "<="
	case TokGt:
		op = ">"
	case TokGe:
		op = ">="
	}
	if op == "" {
		return left, hints
	}
	s.Consume()
	right, h := parseAdd(s, ctx)
	return &BinExpr{Op: op, L: left, R: right}, append(hints, h...)
}

func parseAdd(s *Stream, ctx Context) (RobustExpr, []Hint) {
	left, hints := parseMul(s, ctx)
	for s.Peek().Kind == TokPlus || s.Peek().Kind == TokMinus {
		op := s.Consume().Text
		right, h := parseMul(s, ctx)
		left = &BinExpr{Op: op, L: left, R: right}
		hints = append(hints, h...)
	}
	return left, hints
}

func parseMul(s *Stream, ctx Context) (RobustExpr, []Hint) {
	left, hints := parseUnary(s, ctx)
	for s.Peek().Kind == TokStar {
		op := s.Consume().Text
		right, h := parseUnary(s, ctx)
		left = &BinExpr{Op: op, L: left, R: right}
		hints = append(hints, h...)
	}
	return left, hints
}

func parseUnary(s *Stream, ctx Context) (RobustExpr, []Hint) {
	if s.Peek().Kind == TokMinus {
		s.Consume()
		inner, h := parsePrimary(s, ctx)
		return &UnaryExpr{Op: "-", Operand: inner}, h
	}
	return parsePrimary(s, ctx)
}

func parsePrimary(s *Stream, ctx Context) (RobustExpr, []Hint) {
	t := s.Peek()
	switch t.Kind {
	case TokString:
		s.Consume()
		v := t.Text
		if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
			v = v[1 : len(v)-1]
		}
		return &StringLit{baseNode: baseNode{P: t.Pos}, Value: v}, nil
	case TokNumber:
		s.Consume()
		kind := KindInteger
		if strings.Contains(t.Text, ".") {
			kind = KindDecimal
		}
		return &NumberLit{baseNode: baseNode{P: t.Pos}, Value: t.Text, Kind: kind}, nil
	case TokIdent:
		return parseIdentLed(s, ctx)
	case TokDollarIdent:
		return parseDollar(s, ctx)
	case TokAt:
		s.Consume()
		return parseConstantRef(s, ctx, t.Pos)
	case TokToken:
		s.Consume()
		return parseTokenLit(t), nil
	case TokLParen:
		s.Consume()
		inner, hints := parseOr(s, ctx)
		if s.Peek().Kind == TokRParen {
			s.Consume()
		}
		return &ParenExpr{baseNode: baseNode{P: t.Pos}, Inner: inner}, hints
	}
	return &RecoveredExpr{
		baseNode:       baseNode{P: t.Pos},
		SourceFragment: t.Text,
		Reason:         "unrecognised token at primary position",
	}, nil
}

func parseIdentLed(s *Stream, ctx Context) (RobustExpr, []Hint) {
	t := s.Consume()
	name := t.Text
	switch strings.ToLower(name) {
	case "true":
		return &BoolLit{baseNode: baseNode{P: t.Pos}, Value: true}, nil
	case "false":
		return &BoolLit{baseNode: baseNode{P: t.Pos}, Value: false}, nil
	case "empty", "null":
		return &EmptyExpr{baseNode: baseNode{P: t.Pos}}, nil
	case "if":
		return parseIfThenElse(s, ctx, t.Pos)
	}
	if s.Peek().Kind == TokLParen {
		s.Consume()
		var args []RobustExpr
		var hints []Hint
		if s.Peek().Kind != TokRParen {
			for {
				a, h := parseOr(s, ctx)
				args = append(args, a)
				hints = append(hints, h...)
				if s.Peek().Kind == TokComma {
					s.Consume()
					continue
				}
				break
			}
		}
		if s.Peek().Kind == TokRParen {
			s.Consume()
		}
		return &CallExpr{baseNode: baseNode{P: t.Pos}, Name: name, Args: args}, hints
	}
	if s.Peek().Kind == TokDot {
		s.Consume()
		if s.Peek().Kind != TokIdent {
			return &QNameExpr{baseNode: baseNode{P: t.Pos}, Module: name}, nil
		}
		n2 := s.Consume().Text
		if s.Peek().Kind == TokDot {
			s.Consume()
			if s.Peek().Kind == TokIdent {
				n3 := s.Consume().Text
				return &QNameExpr{baseNode: baseNode{P: t.Pos}, Module: name, Name: n2, Sub: n3}, nil
			}
		}
		return &QNameExpr{baseNode: baseNode{P: t.Pos}, Module: name, Name: n2}, nil
	}
	return &VariableExpr{baseNode: baseNode{P: t.Pos}, Name: name}, nil
}

func parseDollar(s *Stream, ctx Context) (RobustExpr, []Hint) {
	t := s.Consume()
	name := strings.TrimPrefix(t.Text, "$")
	if s.Peek().Kind != TokSlash {
		return &VariableExpr{baseNode: baseNode{P: t.Pos}, Name: name}, nil
	}
	var path []string
	for s.Peek().Kind == TokSlash {
		s.Consume()
		if s.Peek().Kind == TokIdent {
			seg := s.Consume().Text
			for s.Peek().Kind == TokDot {
				s.Consume()
				if s.Peek().Kind != TokIdent {
					break
				}
				seg += "." + s.Consume().Text
			}
			path = append(path, seg)
		} else {
			break
		}
	}
	return &AttributePathExpr{baseNode: baseNode{P: t.Pos}, Variable: name, Path: path}, nil
}

func parseConstantRef(s *Stream, ctx Context, p Position) (RobustExpr, []Hint) {
	if s.Peek().Kind != TokIdent {
		return &RecoveredExpr{baseNode: baseNode{P: p}, SourceFragment: "@", Reason: "expected qualified name after '@'"}, nil
	}
	parts := []string{s.Consume().Text}
	for s.Peek().Kind == TokDot {
		s.Consume()
		if s.Peek().Kind != TokIdent {
			break
		}
		parts = append(parts, s.Consume().Text)
	}
	return &ConstantRef{baseNode: baseNode{P: p}, QName: strings.Join(parts, ".")}, nil
}

func parseTokenLit(t Token) *TokenExpr {
	inner := strings.TrimPrefix(t.Text, "[%")
	inner = strings.TrimSuffix(inner, "%]")
	arg := ""
	if i := strings.Index(inner, "'"); i >= 0 {
		arg = inner[i:]
		inner = inner[:i]
	}
	return &TokenExpr{baseNode: baseNode{P: t.Pos}, Token: inner, Arg: arg}
}

func parseIfThenElse(s *Stream, ctx Context, p Position) (RobustExpr, []Hint) {
	cond, h1 := parseOr(s, ctx)
	if !matchKeyword(s, "then") {
		return &IfThenElseExpr{baseNode: baseNode{P: p}, Cond: cond}, h1
	}
	thn, h2 := parseOr(s, ctx)
	var els RobustExpr
	var h3 []Hint
	if matchKeyword(s, "else") {
		els, h3 = parseOr(s, ctx)
	}
	return &IfThenElseExpr{baseNode: baseNode{P: p}, Cond: cond, Then: thn, Else: els}, append(append(h1, h2...), h3...)
}

func matchKeyword(s *Stream, kw string) bool {
	t := s.Peek()
	if t.Kind == TokIdent && strings.EqualFold(t.Text, kw) {
		s.Consume()
		return true
	}
	return false
}
