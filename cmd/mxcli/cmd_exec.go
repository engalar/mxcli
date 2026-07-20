// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/spf13/cobra"
)

// findMxBinaryForProject locates the mx binary for the given project path.
// Checks PATH first, then falls back to ~/.mxcli/mxbuild/*/modeler/mx.
func findMxBinaryForProject(projectPath string) (string, error) {
	if p, err := exec.LookPath("mx"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		matches, _ := filepath.Glob(filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", "mx"))
		if len(matches) > 0 {
			return matches[len(matches)-1], nil
		}
	}
	return "", fmt.Errorf("mx not found in PATH or ~/.mxcli/mxbuild/")
}

var execCmd = &cobra.Command{
	Use:   "exec <file>",
	Short: "Execute an MDL script file",
	Long: `Execute an MDL script file containing MDL commands.

Example:
  mxcli exec setup.mdl
  mxcli exec -p app.mpr script.mdl
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		projectPath, _ := cmd.Flags().GetString("project")

		out := cmd.OutOrStdout()
		errOut := cmd.ErrOrStderr()

		// Read the file
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		exe, logger := buildExec("exec", out)
		defer logger.Close()
		defer exe.Close()

		// Suppress status messages when stdout is a pipe so that output
		// can be used programmatically (e.g. > describe-snapshot.mdl).
		if fi, statErr := out.(interface{ Fd() uintptr }); statErr {
			_ = fi // pipe detection not available for socket writers; always emit
		} else if fi2, err := os.Stdout.Stat(); err == nil && (fi2.Mode()&os.ModeCharDevice) == 0 {
			exe.SetQuiet(true)
		}

		// Auto-connect if project specified
		if projectPath != "" {
			connectCmd := fmt.Sprintf("CONNECT LOCAL '%s';", projectPath)
			prog, _ := visitor.Build(connectCmd)
			for _, stmt := range prog.Statements {
				if err := exe.Execute(stmt); err != nil {
					fmt.Fprintf(errOut, "Error: %v\n", err)
					return fmt.Errorf("connecting to project: %w", err)
				}
			}
		}

		// Parse and execute the file
		prog, errs := visitor.Build(string(content))
		if len(errs) > 0 {
			for _, parseErr := range errs {
				fmt.Fprintf(errOut, "Parse error: %v\n", parseErr)
			}
			return fmt.Errorf("parse failed with %d error(s)", len(errs))
		}

		progStart := time.Now()
		if err := exe.ExecuteProgram(prog); err != nil {
			if errors.Is(err, executor.ErrExit) {
				return nil
			}
			fmt.Fprintf(errOut, "Error: %v\n", err)
			return err
		}
		// Normalize pluggable widget definitions after script execution.
		// mx update-widgets reconciles widget Objects with .mpk Type definitions,
		// preventing CE0463 ("widget definition changed") on newly created widgets.
		if projectPath != "" {
			if mxPath, err := findMxBinaryForProject(projectPath); err == nil {
				uwCmd := exec.Command(mxPath, "update-widgets", projectPath)
				if output, err := uwCmd.CombinedOutput(); err != nil {
					fmt.Fprintf(errOut, "Warning: mx update-widgets failed: %v\n%s\n", err, output)
				}
			}
		}
		// Print performance report to stderr.
		exe.PerfReport(errOut)
		elapsed := time.Since(progStart)
		fmt.Fprintf(errOut, "  Script time: %s\n", executor.FormatDuration(elapsed))
		return nil
	},
}
