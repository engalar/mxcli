// SPDX-License-Identifier: Apache-2.0

package exprcheck

import (
	"strings"
	"unicode"
)

type TokKind int

const (
	TokEOF TokKind = iota
	TokError
	TokIdent
	TokDollarIdent
	TokString
	TokNumber
	TokToken
	TokDot
	TokSlash
	TokAt
	TokLParen
	TokRParen
	TokComma
	TokPlus
	TokMinus
	TokStar
	TokEq
	TokNeq
	TokLt
	TokLe
	TokGt
	TokGe
)

type Token struct {
	Kind TokKind
	Text string
	Pos  Position
}

func Lex(src string) []Token {
	var (
		toks       []Token
		i, ln, col = 0, 1, 1
	)
	advance := func(n int) {
		for k := 0; k < n && i < len(src); k++ {
			if src[i] == '\n' {
				ln++
				col = 1
			} else {
				col++
			}
			i++
		}
	}
	push := func(k TokKind, text string, p Position) {
		toks = append(toks, Token{Kind: k, Text: text, Pos: p})
	}
	for i < len(src) {
		c := src[i]
		p := Position{Offset: i, Line: ln, Column: col}
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			advance(1)
		case c == '\'':
			j := i + 1
			for j < len(src) && src[j] != '\'' {
				j++
			}
			if j < len(src) {
				push(TokString, src[i:j+1], p)
				advance(j - i + 1)
			} else {
				push(TokError, src[i:], p)
				i = len(src)
			}
		case c == '$':
			j := i + 1
			for j < len(src) && isIdentChar(rune(src[j])) {
				j++
			}
			push(TokDollarIdent, src[i:j], p)
			advance(j - i)
		case c == '[' && i+1 < len(src) && src[i+1] == '%':
			j := strings.Index(src[i:], "%]")
			if j < 0 {
				push(TokError, src[i:], p)
				i = len(src)
			} else {
				end := i + j + 2
				push(TokToken, src[i:end], p)
				advance(end - i)
			}
		case c == '@':
			if i+1 < len(src) && (isIdentChar(rune(src[i+1])) || src[i+1] == '.') {
				push(TokAt, "@", p)
				advance(1)
			} else {
				push(TokError, "@", p)
				advance(1)
			}
		case c == '.':
			push(TokDot, ".", p)
			advance(1)
		case c == '/':
			push(TokSlash, "/", p)
			advance(1)
		case c == '(':
			push(TokLParen, "(", p)
			advance(1)
		case c == ')':
			push(TokRParen, ")", p)
			advance(1)
		case c == ',':
			push(TokComma, ",", p)
			advance(1)
		case c == '+':
			push(TokPlus, "+", p)
			advance(1)
		case c == '-':
			push(TokMinus, "-", p)
			advance(1)
		case c == '*':
			push(TokStar, "*", p)
			advance(1)
		case c == '=':
			push(TokEq, "=", p)
			advance(1)
		case c == '!' && i+1 < len(src) && src[i+1] == '=':
			push(TokNeq, "!=", p)
			advance(2)
		case c == '<' && i+1 < len(src) && src[i+1] == '>':
			push(TokNeq, "<>", p)
			advance(2)
		case c == '<' && i+1 < len(src) && src[i+1] == '=':
			push(TokLe, "<=", p)
			advance(2)
		case c == '<':
			push(TokLt, "<", p)
			advance(1)
		case c == '>' && i+1 < len(src) && src[i+1] == '=':
			push(TokGe, ">=", p)
			advance(2)
		case c == '>':
			push(TokGt, ">", p)
			advance(1)
		case unicode.IsDigit(rune(c)):
			j := i
			for j < len(src) && (unicode.IsDigit(rune(src[j])) || src[j] == '.') {
				j++
			}
			push(TokNumber, src[i:j], p)
			advance(j - i)
		case unicode.IsLetter(rune(c)) || c == '_':
			j := i
			for j < len(src) && isIdentChar(rune(src[j])) {
				j++
			}
			push(TokIdent, src[i:j], p)
			advance(j - i)
		default:
			push(TokError, string(c), p)
			advance(1)
		}
	}
	toks = append(toks, Token{Kind: TokEOF, Pos: Position{Offset: i, Line: ln, Column: col}})
	return toks
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
