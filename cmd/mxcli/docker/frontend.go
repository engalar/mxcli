// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FrontendBuildOptions configures a React client frontend build.
type FrontendBuildOptions struct {
	// DeployDir is the deployment/ root directory (contains web/ subdirectory).
	DeployDir string

	// MxBuildDir is the {version}/modeler/ directory where tools/node/ lives.
	MxBuildDir string

	// Stdout receives build output.
	Stdout io.Writer
}

// RollupConfigExists reports whether the deploy directory contains a React
// client frontend (deployment/web/rollup.config.mjs).
func RollupConfigExists(deployDir string) bool {
	_, err := os.Stat(filepath.Join(deployDir, "web", "rollup.config.mjs"))
	return err == nil
}

// resolveNodeExe returns the absolute path to the bundled node executable for
// the current platform.
func resolveNodeExe(mxbuildDir string) string {
	return resolveNodeExeForPlatform(mxbuildDir, runtime.GOOS, runtime.GOARCH)
}

// resolveNodeExeForPlatform resolves the node executable path for an explicit
// platform, enabling table-driven tests without build tags.
func resolveNodeExeForPlatform(mxbuildDir, goos, goarch string) string {
	var subdir, exe string
	switch {
	case goos == "windows" && goarch == "amd64":
		subdir, exe = "win-x64", "node.exe"
	case goos == "linux" && goarch == "amd64":
		subdir, exe = "linux-x64", "node"
	case goos == "darwin" && goarch == "amd64":
		subdir, exe = "darwin-x64", "node"
	case goos == "darwin" && goarch == "arm64":
		subdir, exe = "darwin-arm64", "node"
	default:
		return ""
	}
	return filepath.Join(mxbuildDir, "tools", "node", subdir, exe)
}

// BuildFrontend runs the React client rollup build using the node binary
// bundled with mxbuild. It sets NODE_ENV=production so rollup performs a
// one-shot production build instead of entering watch mode.
//
// The process is launched with exec.Command (not via a shell), which makes it
// safe on Git Bash on Windows where MSYS path mangling would otherwise corrupt
// the arguments.
func BuildFrontend(opts FrontendBuildOptions) error {
	w := opts.Stdout
	if w == nil {
		w = os.Stdout
	}

	nodeExe := resolveNodeExe(opts.MxBuildDir)
	if nodeExe == "" {
		return fmt.Errorf("unsupported platform for React client build: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if _, err := os.Stat(nodeExe); err != nil {
		return fmt.Errorf("bundled node not found at %s: %w", nodeExe, err)
	}

	runnerMjs := filepath.Join(opts.MxBuildDir, "tools", "node", "rollup-runner.mjs")

	fmt.Fprintln(w, "Building React client frontend...")

	cmd := exec.Command(nodeExe, runnerMjs)
	CmdWithPdeathsig(cmd)
	cmd.Dir = filepath.Join(opts.DeployDir, "web")
	cmd.Env = append(os.Environ(), "NODE_ENV=production")
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("React client frontend build failed: %w", err)
	}

	fmt.Fprintln(w, "React client frontend built successfully.")
	return nil
}
