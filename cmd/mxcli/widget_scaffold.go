// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/internal/widget/scaffold"
	"github.com/spf13/cobra"
)

func runWidgetNew(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("widget name required: mxcli widget new <name>")
	}
	name := args[0]

	outDir, _ := cmd.Flags().GetString("dir")
	if outDir == "" {
		outDir = name
	}
	if abs, err := filepath.Abs(outDir); err == nil {
		outDir = abs
	}

	if _, err := os.Stat(outDir); err == nil {
		return fmt.Errorf("directory %q already exists", outDir)
	}

	packagePath, _ := cmd.Flags().GetString("package-path")
	widgetID, _ := cmd.Flags().GetString("id")
	if widgetID == "" {
		widgetID = scaffold.DeriveWidgetID(packagePath, name)
	} else {
		if err := scaffold.ValidateWidgetIDFormat(widgetID); err != nil {
			return err
		}
	}

	projectPath, _ := cmd.Flags().GetString("project-path")
	offline, _ := cmd.Flags().GetBool("offline")
	description, _ := cmd.Flags().GetString("description")
	propStrs, _ := cmd.Flags().GetStringArray("property")

	var props []scaffold.PropertySpec
	for _, s := range propStrs {
		p, err := scaffold.ParsePropertySpec(s)
		if err != nil {
			return err
		}
		props = append(props, p)
	}

	spec := scaffold.Spec{
		Name:        name,
		PackageName: strings.ToLower(name),
		WidgetID:    widgetID,
		PackagePath: packagePath,
		ProjectPath: projectPath,
		Properties:  props,
		Offline:     offline,
		Description: description,
	}

	if err := scaffold.Run(outDir, spec); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	fmt.Printf("Created widget project: %s/\n", outDir)
	fmt.Printf("  Widget ID: %s\n", widgetID)
	fmt.Printf("  Edit:      %s/src/%s.jsx\n", outDir, name)
	fmt.Printf("  Build:     mxcli widget build --dir %s\n", outDir)
	return nil
}
