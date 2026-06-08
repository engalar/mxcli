// cmd/mxcli-local/cmd_build.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

func buildCmd() *cobra.Command {
	var (
		projectPath       string
		skipCheck         bool
		skipUpdateWidgets bool
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a PAD package from an MPR file (no Docker required)",
		Example: `  mxcli-local build -p app.mpr
  mxcli-local build -p app.mpr --skip-check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectPath != "" {
				if abs, err := filepath.Abs(projectPath); err == nil {
					projectPath = abs
				}
			}
			return docker.Build(docker.BuildOptions{
				ProjectPath:       projectPath,
				SkipCheck:         skipCheck,
				SkipUpdateWidgets: skipUpdateWidgets,
				UseDeployLayout:   true,
				Stdout:            os.Stdout,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (required)")
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "Skip mx check before building")
	cmd.Flags().BoolVar(&skipUpdateWidgets, "skip-update-widgets", false, "Skip widget update step")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}
