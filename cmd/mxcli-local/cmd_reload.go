// cmd/mxcli-local/cmd_reload.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

func reloadCmd() *cobra.Command {
	var (
		projectPath   string
		adminPassword string
		skipCheck     bool
		modelOnly     bool
		cssOnly       bool
	)

	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Hot reload the running local Mendix app (no restart required)",
		Long: `Hot reload the Mendix application running via 'mxcli local run'.

Modes:
  (default)      Build the PAD package, then call reload_model
  --model-only   Skip build, just call reload_model (PAD already up to date)
  --css          Update styling only (instant, no build or model reload)

The admin password must match what was passed to 'mxcli local run --admin-password'.
Default password: Admin123!`,
		Example: `  mxcli local reload -p app.mpr
  mxcli local reload -p app.mpr --model-only
  mxcli local reload -p app.mpr --css
  mxcli local reload -p app.mpr --skip-check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --css and --model-only don't need the project path (no build step).
			if !cssOnly && !modelOnly && projectPath == "" {
				return fmt.Errorf("--project (-p) is required for full reload; use --model-only or --css to skip the build step")
			}

			if projectPath != "" {
				if abs, err := filepath.Abs(projectPath); err == nil {
					projectPath = abs
				}
			}

			token := adminPassword
			if token == "" {
				token = "Admin123!"
			}

			caller := &docker.DirectM2EECaller{
				Host:  "localhost",
				Port:  8090,
				Token: token,
			}

			return docker.Reload(docker.ReloadOptions{
				ProjectPath: projectPath,
				SkipCheck:   skipCheck,
				SkipBuild:   modelOnly,
				CSSOnly:     cssOnly,
				Caller:      caller,
				Stdout:      os.Stdout,
				Stderr:      os.Stderr,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (required for full reload)")
	cmd.Flags().StringVar(&adminPassword, "admin-password", "", "M2EE admin password (default: Admin123!)")
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "Skip mx check before build")
	cmd.Flags().BoolVar(&modelOnly, "model-only", false, "Skip build, just call reload_model")
	cmd.Flags().BoolVar(&cssOnly, "css", false, "CSS hot reload only (update_styling, no build)")
	return cmd
}
