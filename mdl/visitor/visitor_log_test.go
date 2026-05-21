// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"testing"
)

// TestLogStatement_EmptyWithClause is a regression test for the nil pointer
// panic in buildTemplateParams when the `with ()` clause contains no params.
//
// `with ()` is a grammar error (templateParam requires {N} = expr), but
// ANTLR error recovery may insert a synthetic TemplateParamContext without a
// NUMBER_LITERAL token, which caused `NUMBER_LITERAL().GetText()` to panic.
// The fix guards against nil NUMBER_LITERAL with an explicit nil check.
func TestLogStatement_EmptyWithClause(t *testing.T) {
	src := `create microflow MfTest.M () begin
  log info node 'App' 'msg' with ();
end;`
	// Build must not panic — parse errors are acceptable, but a nil-pointer
	// dereference is not. We recover from any panic and report it as a failure.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Build panicked with 'with ()' clause: %v", r)
		}
	}()
	Build(src) // may return parse errors — that's fine, panic is not
}
