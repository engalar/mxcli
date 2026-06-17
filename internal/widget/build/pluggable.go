package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PluggableWidgetsToolsBuilder builds using @mendix/pluggable-widgets-tools.
type PluggableWidgetsToolsBuilder struct{}

func (PluggableWidgetsToolsBuilder) Name() string { return "pluggable-widgets-tools" }

func (PluggableWidgetsToolsBuilder) Available() bool {
	// Check if @mendix/pluggable-widgets-tools is installed in node_modules
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	toolsPath := filepath.Join(cwd, "node_modules", "@mendix", "pluggable-widgets-tools")
	if _, err := os.Stat(toolsPath); err == nil {
		return true
	}
	// Also check the project dir if we can detect it
	return false
}

func (PluggableWidgetsToolsBuilder) Build(ctx context.Context, projectDir string) (string, error) {
	tool, err := detectToolchain()
	if err != nil {
		return "", err
	}

	if err := installDeps(projectDir, tool); err != nil {
		return "", fmt.Errorf("install deps: %w", err)
	}

	// Run npm run build (which calls pluggable-widgets-tools build:web)
	cmd := exec.CommandContext(ctx, tool, "run", "build")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build failed: %w", err)
	}

	// Find the generated MPK in dist/1.0.0/
	mpkPattern := filepath.Join(projectDir, "dist", "1.0.0", "*.mpk")
	matches, err := filepath.Glob(mpkPattern)
	if err != nil {
		return "", fmt.Errorf("glob mpk: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no .mpk found in dist/1.0.0/ — build may have failed")
	}
	return matches[0], nil
}

func detectToolchain() (string, error) {
	if _, err := exec.LookPath("bun"); err == nil {
		return "bun", nil
	}
	if _, err := exec.LookPath("npm"); err == nil {
		return "npm", nil
	}
	return "", fmt.Errorf("bun not found, npm not found\n" +
		"  install bun: curl -fsSL https://bun.sh/install | bash\n" +
		"  install npm: https://nodejs.org/")
}

func installDeps(projectDir, tool string) error {
	if _, err := os.Stat(filepath.Join(projectDir, "node_modules")); err == nil {
		return nil
	}
	fmt.Printf("Installing dependencies (%s install)...\n", tool)
	cmd := exec.Command(tool, "install")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
