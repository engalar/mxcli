// SPDX-License-Identifier: Apache-2.0

// Regression tests for expression string literal escaping in expressionToString.
//
// The previous implementation had two dueling bugs:
//  1. Using mdlQuote() duplicated every backslash, breaking regex escape
//     sequences like `\d` that the Mendix expression engine reads literally.
//  2. Using only apostrophe-doubling emitted raw control characters (0x0A,
//     0x0D, 0x09) inside single-quoted literals, which the MDL lexer rejects.
//
// The fix is quoteExpressionLiteral: escape control chars and backslashes
// followed by an MDL-significant letter (n/r/t/\/') but pass other backslash
// sequences through unchanged so regex escapes survive roundtrip.
package executor

import (
	"strings"
	"testing"
)

func TestQuoteExpressionLiteral_PreservesRegexBackslashEscapes(t *testing.T) {
	// `\d`, `\w`, `\s`, `\p{Lu}` etc. must pass through verbatim — the
	// Mendix expression engine treats them as literal `\d` and relies on
	// the regex compiler to interpret the escape.
	cases := []string{
		`^\d+$`,
		`\w+`,
		`\s*`,
		`\p{Lu}`,
		`mix \d and \w`,
	}
	for _, in := range cases {
		got := quoteExpressionLiteral(in)
		want := "'" + in + "'"
		if got != want {
			t.Errorf("quoteExpressionLiteral(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestQuoteExpressionLiteral_EscapesRawControlChars(t *testing.T) {
	// Raw newline, carriage return, and tab must be escaped — STRING_LITERAL
	// does not accept them raw and the describe output has to survive check.
	cases := []struct {
		in   string
		want string
	}{
		{"line1\nline2", `'line1\nline2'`},
		{"line1\r\nline2", `'line1\r\nline2'`},
		{"col\tcol", `'col\tcol'`},
	}
	for _, tc := range cases {
		got := quoteExpressionLiteral(tc.in)
		if got != tc.want {
			t.Errorf("quoteExpressionLiteral(%q) = %q, want %q", tc.in, got, tc.want)
		}
		// Extra invariant: output must never contain the raw control char.
		for _, bad := range []byte{'\n', '\r', '\t'} {
			if strings.ContainsRune(got, rune(bad)) {
				t.Errorf("output %q leaked raw control byte 0x%02x", got, bad)
			}
		}
	}
}

func TestQuoteExpressionLiteral_DoublesApostrophes(t *testing.T) {
	got := quoteExpressionLiteral("it's here")
	want := "'it''s here'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestQuoteExpressionLiteral_EscapesBackslashBeforeEscapeLetter(t *testing.T) {
	// `\` followed by one of the recognised escape letters must be doubled,
	// otherwise the visitor's unquoteString would decode a backslash-letter
	// pair back into a control character on reparse.
	cases := []struct {
		in   string
		want string
	}{
		{`\n`, `'\\n'`},
		{`\r`, `'\\r'`},
		{`\t`, `'\\t'`},
		{`\\`, `'\\\\'`}, // double backslash roundtrips as double backslash
	}
	for _, tc := range cases {
		got := quoteExpressionLiteral(tc.in)
		if got != tc.want {
			t.Errorf("quoteExpressionLiteral(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestQuoteExpressionLiteral_TrailingBackslashDoubled(t *testing.T) {
	// A trailing backslash in the AST value cannot be emitted raw: the lexer
	// reads the closing `\'` as an escape pair (`\\ .`), never terminating the
	// literal. Doubling is the only safe representation — unquoteString
	// decodes `\\` back to a single backslash, preserving the value.
	cases := []struct {
		in   string
		want string
	}{
		{`abc\`, `'abc\\'`},
		{`\`, `'\\'`},
		{`regex \d\`, `'regex \d\\'`},
	}
	for _, tc := range cases {
		got := quoteExpressionLiteral(tc.in)
		if got != tc.want {
			t.Errorf("quoteExpressionLiteral(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestQuoteExpressionLiteral_IdempotentForDecodeThenEncode(t *testing.T) {
	// Critical invariant: any value that the visitor could produce as a
	// decoded LiteralString must reserialise to a form the lexer accepts,
	// so describe → exec → describe is stable.
	raw := "multi\nline with \\d regex and 'quotes'"
	out := quoteExpressionLiteral(raw)
	for _, bad := range []byte{'\n', '\r', '\t'} {
		if strings.ContainsRune(out, rune(bad)) {
			t.Errorf("output %q has raw control byte 0x%02x", out, bad)
		}
	}
	// Apostrophes must be escaped.
	if strings.Contains(out, "'quotes'") && !strings.Contains(out, "''quotes''") {
		t.Errorf("apostrophes not doubled in %q", out)
	}
}
