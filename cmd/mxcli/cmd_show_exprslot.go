// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/generated/exprgrammar"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	"github.com/spf13/cobra"
)

var showExprSlotCmd = &cobra.Command{
	Use:   "expr-slot <SlotPath>",
	Short: "Show expected expression kind and mined samples for a slot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShowExprSlot(cmd.OutOrStdout(), args[0])
	},
}

func runShowExprSlot(out io.Writer, slotPath string) error {
	sc, ok := exprcheck.DefaultSlotResolver().Expect(slotPath)
	if !ok {
		return fmt.Errorf("slot %q not in resolver", slotPath)
	}
	fmt.Fprintf(out, "SlotPath:     %s\n", slotPath)
	fmt.Fprintf(out, "Context:      %s\n", exprcheck.SlotToContext(slotPath))
	fmt.Fprintf(out, "ExpectedKind: %s\n", exprcheck.KindName(sc.Kind))
	if mined, ok := exprgrammar.SlotExpectations[slotPath]; ok {
		fmt.Fprintf(out, "Frequency:    %d\n\n", mined.Frequency)
		fmt.Fprintln(out, "Sample expressions (highest frequency first):")
		for _, s := range mined.Samples {
			fmt.Fprintf(out, "  %s\n", s)
		}
	}
	return nil
}

func init() {
	showCmd.AddCommand(showExprSlotCmd)
}
