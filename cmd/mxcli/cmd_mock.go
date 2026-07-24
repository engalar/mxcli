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

func mockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mock",
		Short: "Manage Prism mock server for OpenAPI specs",
		Long:  `Start, stop, or check the status of a Prism mock server for OpenAPI specifications.`,
	}

	cmd.AddCommand(mockStartCmd())
	cmd.AddCommand(mockStopCmd())
	cmd.AddCommand(mockStatusCmd())

	return cmd
}

func mockStartCmd() *cobra.Command {
	var (
		port     int
		specPath string
	)

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start Prism mock server",
		Long: `Start a Prism mock server for the given OpenAPI spec.
Runs in the foreground until stopped with Ctrl+C.

Example:
  mxcli mock start -p app.mpr --port 4000 --spec docs/openapi/c01-api.yaml`,
		Example: `  mxcli mock start -p app.mpr
  mxcli mock start -p app.mpr --port 4000 --spec docs/openapi/c01-api.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath, _ := cmd.Root().PersistentFlags().GetString("project")
			if projectPath == "" {
				return fmt.Errorf("-p is required\n\nSpecify the .mpr file path:\n\n  mxcli mock start -p /path/to/app.mpr")
			}

			absPath, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("resolving project path: %w", err)
			}
			projectDir := filepath.Dir(absPath)

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

			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				return fmt.Errorf("spec file not found: %s\n\nCreate the OpenAPI spec or specify an existing path with --spec", specPath)
			}

			if port == 0 {
				port = 4000
			}

			npxPath, err := exec.LookPath("npx")
			if err != nil {
				return fmt.Errorf("npx not found in PATH: %w\n\nInstall Node.js/npx to use the mock server", err)
			}

			c := exec.Command(npxPath, "@stoplight/prism-cli", "mock", specPath, "-p", strconv.Itoa(port))
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			docker.CmdWithPdeathsig(c)

			if err := c.Start(); err != nil {
				return fmt.Errorf("starting prism: %w", err)
			}

			pid := c.Process.Pid
			if err := docker.WriteMockLock(projectDir, &docker.MockLock{
				PID:       pid,
				Port:      port,
				SpecPath:  specPath,
				StartedAt: time.Now(),
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write mock lock file: %v\n", err)
			}

			fmt.Fprintf(os.Stderr, "Prism mock server started on http://localhost:%d (PID %d)\n", port, pid)
			fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n\n")

			// Forward signals to child process group
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigCh
				_ = syscall.Kill(-pid, syscall.SIGTERM)
			}()

			waitErr := c.Wait()
			_ = docker.RemoveMockLock(projectDir)
			return waitErr
		},
	}

	startCmd.Flags().IntVar(&port, "port", 4000, "Port for the mock server")
	startCmd.Flags().StringVar(&specPath, "spec", "", "Path to OpenAPI spec file (default: docs/openapi/c01-api.yaml)")

	return startCmd
}

func mockStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop Prism mock server",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath, _ := cmd.Root().PersistentFlags().GetString("project")
			if projectPath == "" {
				return fmt.Errorf("-p is required")
			}
			projectDir := filepath.Dir(projectPath)
			return docker.KillMockServer(projectDir)
		},
	}
}

func mockStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check Prism mock server status",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath, _ := cmd.Root().PersistentFlags().GetString("project")
			if projectPath == "" {
				return fmt.Errorf("-p is required")
			}
			projectDir := filepath.Dir(projectPath)
			l, err := docker.MockServerStatus(projectDir)
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Mock server is not running")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Mock server is running:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  URL:  http://localhost:%d\n", l.Port)
			fmt.Fprintf(cmd.OutOrStdout(), "  PID:  %d\n", l.PID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Spec: %s\n", l.SpecPath)
			fmt.Fprintf(cmd.OutOrStdout(), "  Since: %s\n", l.StartedAt.Format(time.RFC3339))
			return nil
		},
	}
}
