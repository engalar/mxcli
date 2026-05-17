// Package parse provides batch parsing of Mendix expressions using the
// existing mdl/exprcheck hand-written recursive-descent parser.
// It wraps exprcheck.NewParser() to work on ExprRecord slices from the
// scan package, producing ParseResult values suitable for validation.
package parse

import (
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
)

// ParseResult is the outcome of parsing one ExprRecord.
type ParseResult struct {
	Record scan.ExprRecord
	OK     bool         // true if zero hints with SeverityError
	Hints  []hints.Hint // all hints emitted by the parser
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

var parser = exprcheck.NewParser()

// ParseExpression parses a single raw expression string and returns a ParseResult.
// The ExprRecord's SlotPath is used to provide slot context to the parser.
func ParseExpression(rec scan.ExprRecord) ParseResult {
	ctx := exprcheck.Context{
		SlotPath: rec.SlotPath,
		Slots:    exprcheck.DefaultSlotResolver(),
	}
	_, hs := parser.Parse(rec.Raw, ctx)
	ok := true
	for _, h := range hs {
		if h.Severity == hints.SeverityError {
			ok = false
			break
		}
	}
	return ParseResult{Record: rec, OK: ok, Hints: hs}
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
