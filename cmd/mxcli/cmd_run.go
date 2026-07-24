// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var (
		projectPath   string
		dbURL         string
		adminPassword string
		appPort       int
		adminPort     int
		securityMode  string
		mock          bool
		mockOnly      bool
		mockPort      int
		mockSpec      string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the Mendix runtime from deployment/ (no Docker required)",
		Long: `Run the Mendix runtime using the deployment/ output from 'mxcli build'.
Blocks in the foreground — press Ctrl+C to stop.

Default database: HSQLDB (embedded, no external database needed).
Override with --db for PostgreSQL.`,
		Example: `  mxcli run -p app.mpr
  mxcli run -p app.mpr --db postgres://user:pass@localhost/mendix`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectPath == "" {
				return fmt.Errorf("-p is required\n\nSpecify the .mpr file path so mxcli can find the build output:\n\n  mxcli run -p /path/to/app.mpr [--admin-password pw]\n\nIf you haven't built yet, run first:\n\n  mxcli build -p /path/to/app.mpr")
			}
			if abs, err := filepath.Abs(projectPath); err == nil {
				projectPath = abs
			}

			dir := docker.ResolveRunDir(filepath.Dir(projectPath))
			cmdHint := "-p " + projectPath
			projectDir := filepath.Dir(projectPath)

			// Start mock server if requested
			if mock || mockOnly {
				specPath := mockSpec
				if specPath == "" {
					projectRoot := filepath.Dir(projectDir)
					if projectRoot == "." {
						projectRoot = projectDir
					}
					specPath = filepath.Join(projectRoot, "docs", "openapi", "c01-api.yaml")
					if _, err := os.Stat(specPath); os.IsNotExist(err) {
						specPath = filepath.Join(projectDir, "docs", "openapi", "c01-api.yaml")
					}
				} else if !filepath.IsAbs(specPath) {
					specPath = filepath.Join(projectDir, specPath)
				}
				if _, err := os.Stat(specPath); err != nil {
					return fmt.Errorf("mock spec file not found: %s", specPath)
				}
				npxPath, err := exec.LookPath("npx")
				if err != nil {
					return fmt.Errorf("npx not found: %w", err)
				}
				mockCmd := exec.Command(npxPath, "@stoplight/prism-cli", "mock", specPath, "-p", strconv.Itoa(mockPort))
				mockCmd.Stdout = os.Stdout
				mockCmd.Stderr = os.Stderr
				docker.CmdWithPdeathsig(mockCmd)
				if err := mockCmd.Start(); err != nil {
					return fmt.Errorf("starting mock server: %w", err)
				}
				mockPID := mockCmd.Process.Pid
				if err := docker.WriteMockLock(projectDir, &docker.MockLock{
					PID: mockPID, Port: mockPort, SpecPath: specPath, StartedAt: time.Now(),
				}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to write mock lock file: %v\n", err)
				}
				defer func() {
					_ = syscall.Kill(-mockPID, syscall.SIGTERM)
					_ = docker.RemoveMockLock(projectDir)
				}()
				fmt.Fprintf(os.Stderr, "Mock server running on http://localhost:%d (PID %d)\n", mockPort, mockPID)

				if mockOnly {
					fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n")
					sigCh := make(chan os.Signal, 1)
					signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
					<-sigCh
					return nil
				}
			}

			return docker.StartLocal(docker.LocalRunOptions{
				DeployDir:     dir,
				DB:            dbURL,
				AdminPassword: adminPassword,
				AppPort:       appPort,
				AdminPort:     adminPort,
				SecurityMode:  securityMode,
				CmdHint:       cmdHint,
				Stdout:        os.Stdout,
				Stderr:        os.Stderr,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (derives deployment/ directory)")
	cmd.Flags().StringVar(&dbURL, "db", "", "Database URL (postgres://user:pass@host/db). Default: HSQLDB (embedded)")
	cmd.Flags().StringVar(&adminPassword, "admin-password", "", "MxAdmin login password (default: Admin123!)")
	cmd.Flags().StringVar(&securityMode, "security", "off", "Authentication mode: off (no auth), demo (demo users), production (full auth)")
	cmd.Flags().IntVar(&appPort, "port", 0, "App HTTP port (default 8080)")
	cmd.Flags().IntVar(&adminPort, "admin-port", 0, "Admin API port (default 8090)")
	cmd.Flags().BoolVar(&mock, "mock", false, "Start Prism mock server before runtime")
	cmd.Flags().BoolVar(&mockOnly, "mock-only", false, "Start Prism mock server only (no runtime)")
	cmd.Flags().IntVar(&mockPort, "mock-port", 4000, "Port for the mock server")
	cmd.Flags().StringVar(&mockSpec, "mock-spec", "", "Path to OpenAPI spec file (default: <project dir>/docs/openapi/c01-api.yaml)")
	return cmd
}
