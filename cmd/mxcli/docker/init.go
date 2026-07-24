// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// InitOptions configures the docker init command.
type InitOptions struct {
	// ProjectPath is the path to the .mpr file.
	ProjectPath string

	// OutputDir overrides the output directory (default: .docker/ next to MPR).
	OutputDir string

	// Force overwrites existing files.
	Force bool

	// PortOffset shifts all default ports by N (APP_PORT=8080+N, ADMIN_PORT=8090+N, DB_PORT=5432+N).
	// Useful for running multiple Mendix projects simultaneously.
	PortOffset int

	// Stdout for output messages.
	Stdout io.Writer
}

// Init generates the Docker Compose stack files for a Mendix project.
func Init(opts InitOptions) error {
	w := opts.Stdout
	if w == nil {
		w = os.Stdout
	}

	// Determine output directory
	dockerDir := opts.OutputDir
	if dockerDir == "" {
		dockerDir = filepath.Join(filepath.Dir(opts.ProjectPath), ".docker")
	}

	// Create .docker/ directory
	if err := os.MkdirAll(dockerDir, 0755); err != nil {
		return fmt.Errorf("creating docker directory: %w", err)
	}

	// Write docker-compose.yml
	composePath := filepath.Join(dockerDir, "docker-compose.yml")
	if err := writeTemplate(composePath, "templates/docker-compose.yml", opts.Force, w); err != nil {
		return err
	}

	// Write .env.example
	envExamplePath := filepath.Join(dockerDir, ".env.example")
	if err := writeTemplate(envExamplePath, "templates/env.example", opts.Force, w); err != nil {
		return err
	}

	// Copy .env.example -> .env if .env doesn't exist (or force)
	envPath := filepath.Join(dockerDir, ".env")
	if opts.Force || !fileExists(envPath) {
		data, err := templatesFS.ReadFile("templates/env.example")
		if err != nil {
			return fmt.Errorf("reading env template: %w", err)
		}
		content := string(data)
		if opts.PortOffset != 0 {
			content = applyPortOffset(content, opts.PortOffset)
		}
		if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing .env: %w", err)
		}
		fmt.Fprintf(w, "  Created %s\n", envPath)
		if opts.PortOffset != 0 {
			fmt.Fprintf(w, "  Applied port offset %d (APP=%d, ADMIN=%d, DB=%d)\n",
				opts.PortOffset, 8080+opts.PortOffset, 8090+opts.PortOffset, 5432+opts.PortOffset)
		}
	} else {
		fmt.Fprintf(w, "  Skipped %s (already exists)\n", envPath)
	}

	// Check if build directory exists
	buildDir := filepath.Join(dockerDir, "build")
	if _, err := os.Stat(filepath.Join(buildDir, "Dockerfile")); err == nil {
		fmt.Fprintf(w, "  Found existing build in %s\n", buildDir)
	} else {
		fmt.Fprintf(w, "  Note: Run 'mxcli docker build' to create the build in %s\n", buildDir)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Docker init complete.")
	fmt.Fprintf(w, "  Directory: %s\n", dockerDir)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Next steps:")
	fmt.Fprintln(w, "  1. mxcli docker build -p <project.mpr>   # Build PAD package")
	fmt.Fprintln(w, "  2. mxcli docker up -p <project.mpr>      # Start containers")

	return nil
}

// writeTemplate writes an embedded template file to disk.
func writeTemplate(destPath, templateName string, force bool, w io.Writer) error {
	if !force && fileExists(destPath) {
		fmt.Fprintf(w, "  Skipped %s (already exists)\n", destPath)
		return nil
	}

	data, err := templatesFS.ReadFile(templateName)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", templateName, err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", destPath, err)
	}

	fmt.Fprintf(w, "  Created %s\n", destPath)
	return nil
}

// applyPortOffset replaces the default ports in the .env template content
// with offset ports. APP_PORT 8080->8080+N, ADMIN_PORT 8090->8090+N, DB_PORT 5432->5432+N.
func applyPortOffset(content string, offset int) string {
	appPort := fmt.Sprintf("%d", 8080+offset)
	adminPort := fmt.Sprintf("%d", 8090+offset)
	dbPort := fmt.Sprintf("%d", 5432+offset)

	content = strings.Replace(content, "APP_PORT=8080", "APP_PORT="+appPort, 1)
	content = strings.Replace(content, "ADMIN_PORT=8090", "ADMIN_PORT="+adminPort, 1)
	content = strings.Replace(content, "DB_PORT=5432", "DB_PORT="+dbPort, 1)
	content = strings.Replace(content, "http://localhost:8080", "http://localhost:"+appPort, 1)
	content = strings.Replace(content, "localhost:5432", "localhost:"+dbPort, 1)

	return content
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
