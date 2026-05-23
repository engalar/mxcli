//go:build !integration

// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

func TestExpressionToString_DatetimeToken(t *testing.T) {
	// '[%CurrentDateTime%]' as parsed by MDL parser = LiteralString
	expr := &ast.LiteralExpr{Kind: ast.LiteralString, Value: "[%CurrentDateTime%]"}
	got := expressionToString(expr)
	// Expected: bare token without quotes
	if got != "[%CurrentDateTime%]" {
		t.Errorf("datetime token: got %q, want %q", got, "[%CurrentDateTime%]")
	}
}

func TestExpressionToString_DatetimeToken_BeginOfDay(t *testing.T) {
	expr := &ast.LiteralExpr{Kind: ast.LiteralString, Value: "[%BeginOfCurrentDay%]"}
	got := expressionToString(expr)
	if got != "[%BeginOfCurrentDay%]" {
		t.Errorf("got %q, want %q", got, "[%BeginOfCurrentDay%]")
	}
}

func TestExpressionToString_RegularString(t *testing.T) {
	// Regular string literals must keep their quotes
	expr := &ast.LiteralExpr{Kind: ast.LiteralString, Value: "hello world"}
	got := expressionToString(expr)
	if got != "'hello world'" {
		t.Errorf("regular string: got %q, want %q", got, "'hello world'")
	}
}

func TestExpressionToString_NonDatetimeToken(t *testing.T) {
	// Token that looks like [%X%] but isn't a known pattern - still strip quotes
	// All [%...%] should be treated as tokens
	expr := &ast.LiteralExpr{Kind: ast.LiteralString, Value: "[%CurrentUser%]"}
	got := expressionToString(expr)
	if got != "[%CurrentUser%]" {
		t.Errorf("session token: got %q, want %q", got, "[%CurrentUser%]")
	}
}
