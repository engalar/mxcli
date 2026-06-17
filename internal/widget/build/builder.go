package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	ProjectDir string
	Registry   string
	HTTPSProxy string
}

type Result struct {
	MPKPath string
	SizeKB  int64
}

func Build(ctx context.Context, cfg Config) (*Result, error) {
	if err := installDeps(cfg); err != nil {
		return nil, fmt.Errorf("install deps: %w", err)
	}

	if runtime.GOOS == "windows" {
		if err := patchRollupCP(cfg.ProjectDir); err != nil {
			return nil, fmt.Errorf("patch rollup config: %w", err)
		}
	}

	if err := runScript(ctx, cfg.ProjectDir, "build"); err != nil {
		return nil, fmt.Errorf("npm run build failed: %w", err)
	}

	matches, err := filepath.Glob(filepath.Join(cfg.ProjectDir, "dist", "1.0.0", "*.mpk"))
	if err != nil {
		return nil, fmt.Errorf("glob mpk: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .mpk found in dist/1.0.0/")
	}

	fi, _ := os.Stat(matches[0])
	var size int64
	if fi != nil {
		size = fi.Size() / 1024
	}
	return &Result{MPKPath: matches[0], SizeKB: size}, nil
}

// patchRollupCP replaces a bash extended glob (@(tile|icon)?(.dark))
// in pluggable-widgets-tools' rollup.config.mjs with explicit file checks.
// The `glob` npm package does not support bash @()/?!() patterns on any OS.
// `src/**/*.xml` works fine on Windows — only the @() line needs fixing.
func patchRollupCP(projectDir string) error {
	cfg := filepath.Join(projectDir, "node_modules",
		"@mendix", "pluggable-widgets-tools", "configs", "rollup.config.mjs")

	data, err := os.ReadFile(cfg)
	if err != nil {
		return fmt.Errorf("read rollup config: %w", err)
	}
	content := string(data)

	sentinel := "[mxcli-patched]"
	if strings.Contains(content, sentinel) {
		return nil
	}

	oldPng := "if (existsSync(`src/${widgetName}.icon.png`) || existsSync(`src/${widgetName}.tile.png`)) {\n                        cp(join(sourcePath, `src/${widgetName}.@(tile|icon)?(.dark).png`), outDir);\n                    }"
	newPng := "// " + sentinel + "\n                    ['icon.png','icon.dark.png','tile.png','tile.dark.png'].forEach(function(f){var p=join(sourcePath,'src',widgetName+'.'+f);if(existsSync(p))cp(p,join(outDir,widgetName+'.'+f));})"

	if !strings.Contains(content, oldPng) {
		return fmt.Errorf("rollup config: @() glob pattern not found")
	}
	content = strings.ReplaceAll(content, oldPng, newPng)

	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		return fmt.Errorf("write patched rollup config: %w", err)
	}
	fmt.Println("[mxcli] patched bash @() glob in rollup.config.mjs — see: https://github.com/mendix/pluggable-widgets-tools/issues")
	return nil
}

func installDeps(cfg Config) error {
	// Check for the specific tool, not just any node_modules directory.
	// node_modules/ may exist with only unrelated packages (e.g. esbuild alone).
	toolsPath := filepath.Join(cfg.ProjectDir, "node_modules", "@mendix", "pluggable-widgets-tools")
	if _, err := os.Stat(toolsPath); err == nil {
		return nil
	}

	tool := detectToolchain()
	args := []string{"install", "--no-audit", "--no-fund"}
	if cfg.Registry != "" {
		args = append(args, "--registry", cfg.Registry)
	}
	fmt.Printf("[mxcli] %s %s\n", tool, strings.Join(args, " "))
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
			"http_proxy="+cfg.HTTPSProxy)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s install failed: %w", tool, err)
	}
	fmt.Printf("[mxcli] npm install complete\n")
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
