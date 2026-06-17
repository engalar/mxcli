// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mendixlabs/mxcli/internal/widget/build"
	"github.com/spf13/cobra"
)

func runWidgetBuild(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		dir = "."
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}

	registry, _ := cmd.Flags().GetString("registry")
	httpsProxy, _ := cmd.Flags().GetString("https-proxy")

	result, err := build.Build(context.Background(), build.Config{
		ProjectDir: dir,
		Registry:   registry,
		HTTPSProxy: httpsProxy,
	})
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("Built %s (%d KB)\n", filepath.Base(result.MPKPath), result.SizeKB)

	install, _ := cmd.Flags().GetBool("install")
	if install {
		projectPath, _ := cmd.Flags().GetString("project")
		if projectPath == "" {
			return fmt.Errorf("--install requires -p <project.mpr>")
		}
		if err := build.InstallMPK(result.MPKPath, projectPath); err != nil {
			return fmt.Errorf("install: %w", err)
		}
	}
	return nil
}
