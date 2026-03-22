// SPDX-License-Identifier: Apache-2.0

package main

import "github.com/spf13/cobra"

var bsonCmd = &cobra.Command{
	Use:   "bson",
	Short: "BSON inspection and analysis tools",
	Long: `Tools for analyzing, comparing, and discovering BSON field coverage
in Mendix project files.

Subcommands:
  dump      Dump raw BSON data as JSON or NDSL
  discover  Analyze field coverage per $Type
  compare   Diff two BSON objects`,
}

func init() {
	rootCmd.AddCommand(bsonCmd)

	// Register subcommands
	bsonCmd.AddCommand(bsonDumpCmd)
	bsonCmd.AddCommand(discoverCmd)
	bsonCmd.AddCommand(bsonCompareCmd)
}
