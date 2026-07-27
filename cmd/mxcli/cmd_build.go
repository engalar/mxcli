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
		projectPath string
		skipCheck   bool
		noClean     bool
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a Mendix project to deployment/ using MxBuild",
		Example: `  mxcli build -p app.mpr
  mxcli build -p app.mpr --skip-check
  mxcli build -p app.mpr --no-clean`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectPath != "" {
				if abs, err := filepath.Abs(projectPath); err == nil {
					projectPath = abs
				}
			}
			return docker.Build(docker.BuildOptions{
				ProjectPath: projectPath,
				SkipCheck:   skipCheck,
				SkipClean:   noClean,
				Stdout:      os.Stdout,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (required)")
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "Skip mx check before building")
	cmd.Flags().BoolVar(&noClean, "no-clean", false, "Skip cleaning deployment/ and .mendix-cache/ before build")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}
