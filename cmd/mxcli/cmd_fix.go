package main

import "github.com/spf13/cobra"

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Fix common project issues",
	Long:  `Automated fixes for known project issues.`,
}

func init() {
	fixCmd.AddCommand(fixGridSourcesCmd)
}
