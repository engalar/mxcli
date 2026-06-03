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
			dir := padDir
			if dir == "" {
				if projectPath == "" {
					return fmt.Errorf("either -p or --pad-dir is required")
				}
				dir = filepath.Join(filepath.Dir(projectPath), ".docker", "build")
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
