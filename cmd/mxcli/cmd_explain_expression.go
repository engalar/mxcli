// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	"github.com/mendixlabs/mxcli/mdl/exprcheck/hints"
	"github.com/spf13/cobra"
)

var explainExprIn string

var explainExpressionCmd = &cobra.Command{
	Use:   "expression <text>",
	Short: "Parse a single expression and print any hints",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExplainExpression(cmd.OutOrStdout(), args[0], explainExprIn)
	},
}

func runExplainExpression(out io.Writer, src, slot string) error {
	p := exprcheck.NewParser()
	_, hs := p.Parse(src, exprcheck.Context{
		SlotPath: slot,
		Slots:    exprcheck.DefaultSlotResolver(),
	})
	if len(hs) == 0 {
		fmt.Fprintln(out, "no hints — expression is well-formed for this slot")
		return nil
	}
	for _, h := range hs {
		fmt.Fprintln(out, hints.FormatText(h))
	}
	return nil
}

func init() {
	explainExpressionCmd.Flags().StringVar(&explainExprIn, "in", "", "slot path context (e.g. IfStmt.Condition)")
	explainCmd.AddCommand(explainExpressionCmd)
}
