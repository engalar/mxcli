// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
	"github.com/spf13/cobra"
)

var hintCmd = &cobra.Command{
	Use:   "hint <code>",
	Short: "Explain an expression hint code (E001-E010)",
	Long: `Print the full registry entry for an expression hint code:
trigger conditions, why it is wrong, how to fix, and worked examples.

Use this when an mxcli check / mxcli explain expression run
produces a hint code you don't recognise.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHelpHint(cmd.OutOrStdout(), args[0])
	},
}

func runHelpHint(out io.Writer, code string) error {
	e, ok := hints.Registry.Lookup(code)
	if !ok {
		return fmt.Errorf("hint %q not found (known: E001-E010)", code)
	}
	fmt.Fprintf(out, "%s [%s] %s\n\n", e.Code, e.Slug, strings.ToUpper(hints.SeverityString(e.Severity)))
	fmt.Fprintf(out, "WHEN THIS APPEARS:\n  %s\n\n", e.Trigger)
	fmt.Fprintf(out, "WHY IT'S WRONG:\n  %s\n\n", e.WhyWrong)
	fmt.Fprintf(out, "HOW TO FIX:\n  %s\n\n", e.HowToFix)
	fmt.Fprintln(out, "EXAMPLES:")
	for _, ex := range e.Examples {
		if ex.Note != "" {
			fmt.Fprintf(out, "  # %s\n", ex.Note)
		}
		fmt.Fprintf(out, "  Wrong: %s\n", ex.Wrong)
		fmt.Fprintf(out, "  Right: %s\n\n", ex.Right)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(hintCmd)
}
