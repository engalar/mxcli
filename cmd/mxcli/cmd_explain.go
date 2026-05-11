// SPDX-License-Identifier: Apache-2.0

package main

import "github.com/spf13/cobra"

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain MDL constructs (expressions, hints, slots)",
}

func init() {
	rootCmd.AddCommand(explainCmd)
}
