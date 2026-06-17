package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Config holds parameters for a widget build.
type Config struct {
	ProjectDir string // Absolute path to the widget project root
	Registry   string // npm registry URL for dependency install (e.g. http://localhost:29758/)
	HTTPSProxy string // HTTPS proxy URL for npm install (e.g. http://127.0.0.1:29758)
}

// Result holds the outcome of a build.
type Result struct {
	MPKPath string
	SizeKB  int64
}

// Build installs dependencies via npm install (with optional registry proxy),
// then runs npm run build which delegates to @mendix/pluggable-widgets-tools.
func Build(ctx context.Context, cfg Config) (*Result, error) {
	if err := installDeps(cfg); err != nil {
		return nil, fmt.Errorf("install deps: %w", err)
	}

	if err := runScript(ctx, cfg.ProjectDir, "build"); err != nil {
		return nil, fmt.Errorf("npm run build failed: %w", err)
	}

	// pluggable-widgets-tools outputs to dist/1.0.0/<PackageName>.mpk
	mpkPattern := filepath.Join(cfg.ProjectDir, "dist", "1.0.0", "*.mpk")
	matches, err := filepath.Glob(mpkPattern)
	if err != nil {
		return nil, fmt.Errorf("glob mpk: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .mpk found in dist/1.0.0/ — build may have failed")
	}

	mpkPath := matches[0]
	fi, _ := os.Stat(mpkPath)
	var size int64
	if fi != nil {
		size = fi.Size() / 1024
	}
	return &Result{MPKPath: mpkPath, SizeKB: size}, nil
}

func installDeps(cfg Config) error {
	nodeModules := filepath.Join(cfg.ProjectDir, "node_modules")
	if _, err := os.Stat(nodeModules); err == nil {
		return nil
	}

	tool := detectToolchain()

	args := []string{"install", "--no-audit", "--no-fund"}
	if cfg.Registry != "" {
		args = append(args, "--registry", cfg.Registry)
	}

	fmt.Printf("[mxcli] %s %s\n", tool, args)
	if cfg.HTTPSProxy != "" {
		fmt.Printf("[mxcli] proxy: HTTPS_PROXY=%s\n", cfg.HTTPSProxy)
	}

	cmd := exec.Command(tool, args...)
	cmd.Dir = cfg.ProjectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if cfg.HTTPSProxy != "" {
		cmd.Env = append(cmd.Env,
			"HTTPS_PROXY="+cfg.HTTPSProxy,
			"https_proxy="+cfg.HTTPSProxy,
			"HTTP_PROXY="+cfg.HTTPSProxy,
			"http_proxy="+cfg.HTTPSProxy,
		)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s install failed: %w", tool, err)
	}

	toolsPath := filepath.Join(cfg.ProjectDir, "node_modules", "@mendix", "pluggable-widgets-tools")
	if _, err := os.Stat(toolsPath); err != nil {
		return fmt.Errorf("@mendix/pluggable-widgets-tools not found after install — check network or registry proxy")
	}
	fmt.Printf("[mxcli] npm install complete — %s installed\n", tool)
	return nil
}

func runScript(ctx context.Context, projectDir, script string) error {
	tool := detectToolchain()
	fmt.Printf("Running npm run %s...\n", script)
	cmd := exec.CommandContext(ctx, tool, "run", script)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func detectToolchain() string {
	if _, err := exec.LookPath("bun"); err == nil {
		return "bun"
	}
	return "npm"
}
