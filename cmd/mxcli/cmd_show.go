// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/mendixlabs/mxcli/cmd/mxcli/completion"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:     "show",
	Aliases: []string{"list"},
	Short:   "Alias for describe",
	Long:    `Show is an alias for describe. Use "mxcli describe <type> <name>" to describe project elements.`,
	ValidArgsFunction: completion.ShowValidArgsFunction(comp),
	Args:    cobra.MinimumNArgs(2),
	Run:     describeCmd.Run,
}
