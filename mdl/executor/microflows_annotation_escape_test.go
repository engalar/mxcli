// SPDX-License-Identifier: Apache-2.0

// Regression tests for @annotation / @caption escaping in the microflow
// describer. The free-annotation emission path used a home-grown escape that
// only handled apostrophe doubling — newlines, tabs, and backslashes inside
// the annotation text were emitted as raw control characters, which then
// tripped the parser when the output was fed back through `mxcli exec`.
//
// The fix switches both the free-annotation and the ExclusiveSplit/
// InheritanceSplit @caption emission to `mdlQuote`, which handles the full
// set of MDL-sensitive characters and matches the single quoted-source helper
// used everywhere else.
package executor

import (
	"strings"
	"testing"
)

func TestMdlQuote_EscapesNewlinesAndBackslashes(t *testing.T) {
	in := "SvdV (24/Mar/2021):\r\n\r\nThis microflow uses \\d in a regex."
	out := mdlQuote(in)

	if strings.ContainsRune(out, '\n') {
		t.Errorf("mdlQuote output must not contain a raw newline, got %q", out)
	}
	if strings.ContainsRune(out, '\r') {
		t.Errorf("mdlQuote output must not contain a raw carriage return, got %q", out)
	}
	for _, want := range []string{`\r`, `\n`, `\\d`} {
		if !strings.Contains(out, want) {
			t.Errorf("mdlQuote output missing escaped sequence %q; got %q", want, out)
		}
	}
	if !strings.HasPrefix(out, "'") || !strings.HasSuffix(out, "'") {
		t.Errorf("mdlQuote output should be wrapped in single quotes: %q", out)
	}
}

func TestMdlQuote_EscapesApostrophesByDoubling(t *testing.T) {
	in := "it's here"
	out := mdlQuote(in)
	if out != "'it''s here'" {
		t.Errorf("got %q, want %q", out, "'it''s here'")
	}
}
