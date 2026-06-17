// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
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

	// Use pluggable-widgets-tools builder
	builder := build.PluggableWidgetsToolsBuilder{}
	mpkPath, err := builder.Build(context.Background(), dir)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fi, _ := os.Stat(mpkPath)
	size := int64(0)
	if fi != nil {
		size = fi.Size() / 1024
	}
	fmt.Printf("Built %s (%d KB)\n", filepath.Base(mpkPath), size)

	install, _ := cmd.Flags().GetBool("install")
	if install {
		projectPath, _ := cmd.Flags().GetString("project")
		if projectPath == "" {
			return fmt.Errorf("--install requires -p <project.mpr>")
		}
		if err := build.InstallMPK(mpkPath, projectPath); err != nil {
			return fmt.Errorf("install: %w", err)
		}
	}
	return nil
}
