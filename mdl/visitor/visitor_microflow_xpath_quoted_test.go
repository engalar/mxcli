// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strings"
	"testing"
)

// TestBuild_XPathWithQuotedAssoc_ReturnsError verifies that writing a quoted
// association or entity reference like ["Mod.Assoc" = $x] inside an XPath
// constraint is rejected at the AST layer with a clear, actionable error
// message — rather than silently being parsed as a string literal that
// produces an unhelpful CE error downstream.
func TestBuild_XPathWithQuotedAssoc_ReturnsError(t *testing.T) {
	mdl := `CREATE MICROFLOW Mod.TestFlow ()
RETURNS Boolean AS $r
{
    DECLARE $r Boolean = false;
    DECLARE $Other Mod.OtherEntity = empty;
    RETRIEVE $Obj FROM Mod.Entity WHERE ["Mod.Assoc" = $Other];
    RETURN $r;
}
`
	_, errs := Build(mdl)
	if len(errs) == 0 {
		t.Fatal("expected validation error for quoted identifier in XPath, got none")
	}
	found := false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "unquoted") || strings.Contains(msg, "must not be quoted") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error mentioning 'unquoted' or 'must not be quoted', got: %v", errs)
	}
}

// TestBuild_XPathWithUnquotedAssoc_NoError verifies that the canonical,
// unquoted form ([Mod.Assoc = $x]) is accepted without any 'unquoted' error.
func TestBuild_XPathWithUnquotedAssoc_NoError(t *testing.T) {
	mdl := `CREATE MICROFLOW Mod.TestFlow ()
RETURNS Boolean AS $r
{
    DECLARE $r Boolean = false;
    DECLARE $Other Mod.OtherEntity = empty;
    RETRIEVE $Obj FROM Mod.Entity WHERE [Mod.Assoc = $Other];
    RETURN $r;
}
`
	_, errs := Build(mdl)
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "unquoted") || strings.Contains(msg, "must not be quoted") {
			t.Errorf("unexpected 'unquoted' error for correct syntax: %v", e)
		}
	}
}

// TestBuild_XPathStringLiteralValue_NoError verifies that a legitimate string
// literal value in an XPath comparison (e.g. [Name = 'John']) is NOT mistaken
// for the quoted-identifier mistake. Only dotted quoted tokens like
// "Module.Name" should trigger the validation.
func TestBuild_XPathStringLiteralValue_NoError(t *testing.T) {
	mdl := `CREATE MICROFLOW Mod.TestFlow ()
RETURNS Boolean AS $r
{
    DECLARE $r Boolean = false;
    RETRIEVE $Obj FROM Mod.Entity WHERE [Name = 'John'];
    RETURN $r;
}
`
	_, errs := Build(mdl)
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "unquoted") || strings.Contains(msg, "must not be quoted") {
			t.Errorf("unexpected 'unquoted' error for plain string literal: %v", e)
		}
	}
}
