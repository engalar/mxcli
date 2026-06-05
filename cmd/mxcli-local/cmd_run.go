// cmd/mxcli-local/cmd_run.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var (
		projectPath   string
		dbURL         string
		padDir        string
		adminPassword string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the Mendix runtime from a pre-built PAD (no Docker required)",
		Long: `Run the Mendix runtime using the PAD built by 'mxcli local build'.
Blocks in the foreground — press Ctrl+C to stop.

Default database: HSQLDB (embedded, no external database needed).
Override with --db for PostgreSQL.`,
		Example: `  mxcli-local run -p app.mpr
  mxcli-local run -p app.mpr --db postgres://user:pass@localhost/mendix`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectPath != "" {
				if abs, err := filepath.Abs(projectPath); err == nil {
					projectPath = abs
				}
			}
			if padDir != "" {
				if abs, err := filepath.Abs(padDir); err == nil {
					padDir = abs
				}
			}
			dir := padDir
			if dir == "" {
				if projectPath == "" {
					return fmt.Errorf("-p is required\n\nSpecify the .mpr file path so mxcli can find the build output:\n\n  mxcli local run -p /path/to/app.mpr [--admin-password pw]\n\nIf you haven't built yet, run first:\n\n  mxcli local build -p /path/to/app.mpr")
				}
				// Prefer deploy-layout (no ZIP, faster build) when Studio Pro is installed.
				// Fall back to PAD layout (.docker/build/) otherwise.
				deployDir := filepath.Join(filepath.Dir(projectPath), "deployment")
				if docker.IsDeployLayout(deployDir) {
					dir = deployDir
				} else {
					dir = filepath.Join(filepath.Dir(projectPath), ".docker", "build")
				}
			}
			return docker.StartLocal(docker.LocalRunOptions{
				PadDir:        dir,
				DB:            dbURL,
				AdminPassword: adminPassword,
				Stdout:        os.Stdout,
				Stderr:        os.Stderr,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (derives PAD dir as .docker/build/)")
	cmd.Flags().StringVar(&dbURL, "db", "", "Database URL (postgres://user:pass@host/db). Default: HSQLDB (embedded)")
	cmd.Flags().StringVar(&padDir, "pad-dir", "", "Explicit PAD directory (overrides -p)")
	cmd.Flags().StringVar(&adminPassword, "admin-password", "", "MxAdmin login password (default: Admin123!)")
	return cmd
}
