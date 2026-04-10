// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CheckOptions configures the mx check command.
type CheckOptions struct {
	// ProjectPath is the path to the .mpr file.
	ProjectPath string

	// MxBuildPath is an explicit path to the mxbuild executable (used to find mx).
	MxBuildPath string

	// SkipUpdateWidgets skips the 'mx update-widgets' step before checking.
	// By default, update-widgets runs first to normalize pluggable widget
	// definitions and prevent false CE0463 errors.
	SkipUpdateWidgets bool

	// Stdout for output messages.
	Stdout io.Writer

	// Stderr for error output.
	Stderr io.Writer
}

// Check runs 'mx check' on the project to validate it before building.
func Check(opts CheckOptions) error {
	w := opts.Stdout
	if w == nil {
		w = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	// Resolve mx binary
	mxPath, err := ResolveMx(opts.MxBuildPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Using mx: %s\n", mxPath)

	// Run mx update-widgets to normalize pluggable widget definitions.
	// This prevents false CE0463 ("widget definition changed") errors caused
	// by mismatch between widget Object properties and Type PropertyTypes.
	if !opts.SkipUpdateWidgets {
		fmt.Fprintf(w, "Updating widget definitions in %s...\n", opts.ProjectPath)
		uwCmd := exec.Command(mxPath, "update-widgets", opts.ProjectPath)
		uwCmd.Stdout = w
		uwCmd.Stderr = stderr
		if err := uwCmd.Run(); err != nil {
			// Non-fatal: warn and continue with check
			fmt.Fprintf(w, "Warning: update-widgets failed (continuing with check): %v\n", err)
		} else {
			fmt.Fprintln(w, "Widget definitions updated.")
		}
	}

	// Run mx check
	fmt.Fprintf(w, "Checking project %s...\n", opts.ProjectPath)
	cmd := exec.Command(mxPath, "check", opts.ProjectPath)
	cmd.Stdout = w
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("project check failed: %w", err)
	}

	fmt.Fprintln(w, "Project check passed.")
	return nil
}

// mxBinaryName returns the platform-specific mx binary name.
func mxBinaryName() string {
	if runtime.GOOS == "windows" {
		return "mx.exe"
	}
	return "mx"
}

// ResolveMx finds the mx executable.
// Priority: derive from mxbuild path > PATH lookup.
func ResolveMx(mxbuildPath string) (string, error) {
	if mxbuildPath != "" {
		// Resolve mxbuild first to handle directory paths
		resolvedMxBuild, err := resolveMxBuild(mxbuildPath)
		if err == nil {
			// Look for mx in the same directory as mxbuild
			mxDir := filepath.Dir(resolvedMxBuild)
			candidate := filepath.Join(mxDir, mxBinaryName())
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}

			// Try deriving mx name from mxbuild name (e.g. mxbuild11.6.3 -> mx11.6.3)
			mxbuildBase := filepath.Base(resolvedMxBuild)
			suffix := strings.TrimPrefix(mxbuildBase, "mxbuild")
			if runtime.GOOS == "windows" {
				suffix = strings.TrimPrefix(mxbuildBase, "mxbuild")
				suffix = strings.TrimSuffix(suffix, ".exe")
				candidate = filepath.Join(mxDir, "mx"+suffix+".exe")
			} else {
				candidate = filepath.Join(mxDir, "mx"+suffix)
			}
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}

	// Try PATH
	if p, err := exec.LookPath("mx"); err == nil {
		return p, nil
	}

	// Try cached mxbuild installations (~/.mxcli/mxbuild/*/modeler/mx).
	// NOTE: lexicographic sort is imperfect for versions (e.g. "9.x" > "10.x"),
	// but this is a fallback-of-last-resort — in practice users typically have
	// only one mxbuild version installed.
	if home, err := os.UserHomeDir(); err == nil {
		matches, _ := filepath.Glob(filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", mxBinaryName()))
		if len(matches) > 0 {
			return matches[len(matches)-1], nil
		}
	}

	return "", fmt.Errorf("mx not found; specify --mxbuild-path pointing to Mendix installation directory")
}
