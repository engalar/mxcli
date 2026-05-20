// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a Mendix project to structured MDL files",
	Long: `Export all documents in a Mendix project to a directory of MDL files.

Each document is exported to its own .mdl file. Marketplace modules are
listed in _marketplace.mdl (informational only — not executable).

Examples:
  mxcli export -p app.mpr --output ./export-dir
  mxcli export -p app.mpr --output ./export-dir --module MyFirstModule
  mxcli export -p app.mpr --output ./export-dir --dry-run
`,
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("project")
		outputDir, _ := cmd.Flags().GetString("output")
		moduleFilter, _ := cmd.Flags().GetString("module")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if projectPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
			os.Exit(1)
		}
		if outputDir == "" {
			fmt.Fprintln(os.Stderr, "Error: --output is required")
			os.Exit(1)
		}

		be, err := mprbackend.NewFromPath(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to %s: %v\n", projectPath, err)
			os.Exit(1)
		}
		defer func() { _ = be.Disconnect() }()

		exec := executor.New(os.Stdout)
		exec.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
		exec.SetBackend(be)

		start := time.Now()
		fmt.Fprintf(os.Stderr, "Exporting %s -> %s\n", projectPath, outputDir)

		opts := executor.ExportOptions{
			Module: moduleFilter,
			DryRun: dryRun,
			Progress: func(line string) {
				fmt.Fprintln(os.Stderr, line)
			},
		}

		if err := exec.ExportProject(outputDir, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if dryRun {
			fmt.Fprintf(os.Stderr, "Done (dry run) in %.1fs\n", time.Since(start).Seconds())
		} else {
			fmt.Fprintf(os.Stderr, "Done in %.1fs\n", time.Since(start).Seconds())
		}
	},
}

func init() {
	exportCmd.Flags().String("output", "", "Output directory for MDL files (required)")
	exportCmd.Flags().String("module", "", "Export only this module (default: all modules)")
	exportCmd.Flags().Bool("dry-run", false, "List files that would be written without writing them")
}
