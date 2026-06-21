// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/exprcheck"
	"github.com/spf13/cobra"
)

var showFunctionsCmd = &cobra.Command{
	Use:   "functions [name]",
	Short: "List built-in expression functions (or describe one)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		return runShowFunctions(cmd.OutOrStdout(), name)
	},
}

func runShowFunctions(out io.Writer, name string) error {
	table := exprcheck.PublicFuncTable()
	if name != "" {
		sig, ok := table[name]
		if !ok {
			return fmt.Errorf("function %q not in table", name)
		}
		fmt.Fprintf(out, "Function: %s\n", name)
		fmt.Fprintf(out, "  Signature: %s(%s) -> %s\n", name, strings.Join(sig.Args, ", "), sig.Returns)
		return nil
	}
	keys := make([]string, 0, len(table))
	for k := range table {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sig := table[k]
		fmt.Fprintf(out, "%-15s (%s) -> %s\n", k, strings.Join(sig.Args, ", "), sig.Returns)
	}
	return nil
}

func init() {
	describeCmd.AddCommand(showFunctionsCmd)
	clone := *showFunctionsCmd
	showCmd.AddCommand(&clone)
}
