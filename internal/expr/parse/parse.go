// Package parse provides batch parsing of Mendix expressions.
//
// Two parsing strategies are used depending on the expression category:
//
//  1. XPath expressions (SlotPath ends in ".XPath"): parsed by
//     mdl/visitor.ParseXPathConstraint, which uses the MDL ANTLR4 grammar's
//     xpathConstraint rule. This handles bracket-wrapped WHERE clauses such as
//     "[Code = $Code and IsActive = true]" correctly.
//
//  2. All other expressions: parsed by mdl/exprcheck.NewParser(), a
//     hand-written recursive-descent parser that emits typed Hint values.
//
// Previously, all records were sent to exprcheck, which generated 167 false-positive
// E007 warnings for valid XPath expressions (because exprcheck does not understand
// the "[bracket]" XPath syntax). The routing by SlotPath eliminates those false positives.
package parse

import (
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// ParseResult is the outcome of parsing one ExprRecord.
type ParseResult struct {
	Record scan.ExprRecord
	OK     bool                 // true if zero hints with SeverityError
	Hints  []hints.Hint         // all hints emitted by the parser
	AST    exprcheck.RobustExpr // nil for XPath expressions; non-nil for all others
}

// HasErrors reports whether any hint is SeverityError.
func (r ParseResult) HasErrors() bool {
	for _, h := range r.Hints {
		if h.Severity == hints.SeverityError {
			return true
		}
	}
	return false
}

var exprParser = exprcheck.NewParser()

// isXPathExpression reports whether a raw expression string is an XPath constraint.
//
// XPath constraints stored in Mendix BSON always start with "[" (the bracket is part
// of the stored value, e.g. "[Code = $Code and IsActive = true]").
// Regular Mendix expressions that start with "[" are system tokens like [%CurrentDateTime%],
// which always follow the "[%" prefix pattern.
//
// Verified against 16,125 real expressions from corpus-a + corpus-b:
//   - 0 false positives (no non-XPath expression matches this rule)
//   - 0 false negatives (every XPath expression matches this rule)
//
// This content-based check replaces SlotPath-based routing, removing the need for
// the SlotPath field in ExprRecord and its maintenance burden in the scan package.
func isXPathExpression(raw string) bool {
	s := strings.TrimSpace(raw)
	return len(s) > 0 && s[0] == '[' && !strings.HasPrefix(s, "[%")
}

// ParseExpression parses a single ExprRecord and returns a ParseResult.
// XPath expressions are routed to visitor.ParseXPathConstraint;
// all others go to exprcheck.NewParser().
func ParseExpression(rec scan.ExprRecord) ParseResult {
	if isXPathExpression(rec.Raw) {
		return parseXPath(rec)
	}
	return parseExpr(rec)
}

// parseXPath handles bracket-wrapped XPath constraint expressions.
// Uses visitor.ParseXPathConstraint (MDL ANTLR4 grammar).
func parseXPath(rec scan.ExprRecord) ParseResult {
	_, ok := visitor.ParseXPathConstraint(rec.Raw)
	if ok {
		return ParseResult{Record: rec, OK: true}
	}
	// ParseXPathConstraint returns (nil, false) for malformed XPath.
	return ParseResult{
		Record: rec,
		OK:     false,
		Hints: []hints.Hint{{
			Code:     "XPATH-01",
			Severity: hints.SeverityError,
			Problem:  "XPath constraint could not be parsed by the MDL grammar",
		}},
	}
}

// parseExpr handles regular Mendix expressions using the exprcheck parser.
func parseExpr(rec scan.ExprRecord) ParseResult {
	ctx := exprcheck.Context{
		Slots: exprcheck.DefaultSlotResolver(),
	}
	ast, hs := exprParser.Parse(rec.Raw, ctx)
	ok := true
	for _, h := range hs {
		if h.Severity == hints.SeverityError {
			ok = false
			break
		}
	}
	return ParseResult{Record: rec, OK: ok, Hints: hs, AST: ast}
}

// BatchParse parses a slice of ExprRecords sequentially.
// exprcheck.Parser is not documented as goroutine-safe, so we parse sequentially.
func BatchParse(records []scan.ExprRecord) []ParseResult {
	results := make([]ParseResult, len(records))
	for i, rec := range records {
		results[i] = ParseExpression(rec)
	}
	return results
}

// parseExprWithCatalog parses a regular expression with a CatalogReader injected
// into the exprcheck.Context. This enables semantic hints that require entity /
// enumeration / microflow lookup. XPath expressions still bypass exprcheck.
func parseExprWithCatalog(rec scan.ExprRecord, cat exprcheck.CatalogReader) ParseResult {
	ctx := exprcheck.Context{
		Slots:   exprcheck.DefaultSlotResolver(),
		Catalog: cat,
	}
	ast, hs := exprParser.Parse(rec.Raw, ctx)
	ok := true
	for _, h := range hs {
		if h.Severity == hints.SeverityError {
			ok = false
			break
		}
	}
	return ParseResult{Record: rec, OK: ok, Hints: hs, AST: ast}
}

// ParseExpressionWithCatalog parses one ExprRecord, routing XPath records through
// the MDL grammar and regular expressions through exprcheck with the catalog wired in.
func ParseExpressionWithCatalog(rec scan.ExprRecord, cat exprcheck.CatalogReader) ParseResult {
	if isXPathExpression(rec.Raw) {
		return parseXPath(rec)
	}
	return parseExprWithCatalog(rec, cat)
}

// BatchParseWithCatalog parses a slice of ExprRecords sequentially with the
// supplied catalog injected. cat may be nil — in which case results are
// equivalent to BatchParse.
func BatchParseWithCatalog(records []scan.ExprRecord, cat exprcheck.CatalogReader) []ParseResult {
	results := make([]ParseResult, len(records))
	for i, rec := range records {
		results[i] = ParseExpressionWithCatalog(rec, cat)
	}
	return results
}
