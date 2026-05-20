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

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a previously exported MDL tree into a Mendix project",
	Long: `Walk inputDir for .mdl files, sort them in dependency order, and
execute each against the target project. _marketplace.mdl is always
skipped (it is informational only).

Examples:
  mxcli import -p app.mpr --input ./export-dir
  mxcli import -p app.mpr --input ./export-dir --module MyFirstModule
  mxcli import -p app.mpr --input ./export-dir --dry-run
  mxcli import -p app.mpr --input ./export-dir --skip-errors
`,
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("project")
		inputDir, _ := cmd.Flags().GetString("input")
		moduleFilter, _ := cmd.Flags().GetString("module")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		skipErrors, _ := cmd.Flags().GetBool("skip-errors")

		if projectPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
			os.Exit(1)
		}
		if inputDir == "" {
			fmt.Fprintln(os.Stderr, "Error: --input is required")
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
		fmt.Fprintf(os.Stderr, "Importing %s -> %s\n", inputDir, projectPath)

		opts := executor.ImportOptions{
			Module:     moduleFilter,
			DryRun:     dryRun,
			SkipErrors: skipErrors,
			Progress: func(line string) {
				fmt.Fprintln(os.Stderr, line)
			},
		}

		if err := exec.ImportProject(inputDir, opts); err != nil {
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
	importCmd.Flags().String("input", "", "Input directory containing exported MDL files (required)")
	importCmd.Flags().String("module", "", "Import only files for this module (default: all)")
	importCmd.Flags().Bool("dry-run", false, "Parse and list files without executing them")
	importCmd.Flags().Bool("skip-errors", false, "Continue past parse and execution errors, summarising at the end")
}
