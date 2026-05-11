// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
)

// Generate writes the markdown reference for every registered hint code
// to w, sorted by code. Used by both `go run ./cmd/expr-hints-md` and
// the package's tests.
func Generate(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "# Expression Checker Hint Reference"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Generated from `mdl/exprcheck/hints/registry.go`. Do not edit by hand."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	entries := hints.Registry.All()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Code < entries[j].Code })
	for _, e := range entries {
		fmt.Fprintf(w, "## %s — %s (%s)\n\n", e.Code, e.Slug, hints.SeverityString(e.Severity))
		fmt.Fprintf(w, "**When this appears:** %s\n\n", e.Trigger)
		fmt.Fprintf(w, "**Why it's wrong:** %s\n\n", e.WhyWrong)
		fmt.Fprintf(w, "**How to fix:** %s\n\n", e.HowToFix)
		if len(e.Examples) > 0 {
			fmt.Fprintln(w, "**Examples:**")
			fmt.Fprintln(w)
			for _, ex := range e.Examples {
				if ex.Note != "" {
					fmt.Fprintf(w, "*%s:*\n\n", ex.Note)
				}
				fmt.Fprintln(w, "```mdl")
				fmt.Fprintf(w, "%s   -- wrong\n", ex.Wrong)
				fmt.Fprintf(w, "%s   -- right\n", ex.Right)
				fmt.Fprintln(w, "```")
				fmt.Fprintln(w)
			}
		}
	}
	return nil
}
