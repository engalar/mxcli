package build

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// pluggable-widgets-tools handles JS bundling via rollup
	if err := runScript(ctx, cfg.ProjectDir, "build"); err != nil {
		return nil, fmt.Errorf("npm run build failed: %w", err)
	}

	// Windows workaround: shelljs cp() in rollup.config.mjs uses bash-only globs
	// (src/**/*.xml, @(tile|icon)?(.dark).png) that fail on Windows.
	// We copy missing assets ourselves after build.
	if err := postProcessAssets(cfg.ProjectDir); err != nil {
		return nil, fmt.Errorf("post-process assets: %w", err)
	}

	// pluggable-widgets-tools outputs to dist/1.0.0/<PackageName>.mpk,
	// but that MPK may be empty on Windows (assets missing above).
	// Re-package from dist/tmp/widgets/ which now has all files.
	mpkPath, err := repackageMPK(cfg.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("repackage mpk: %w", err)
	}

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

// postProcessAssets copies XML, PNG, and package.xml into dist/tmp/widgets/
// after pluggable-widgets-tools finishes. This works around shelljs cp() glob
// limitations on Windows (src/**/*.xml, @(tile|icon)?(.dark).png).
func postProcessAssets(projectDir string) error {
	srcDir := filepath.Join(projectDir, "src")
	outDir := filepath.Join(projectDir, "dist", "tmp", "widgets")

	// Copy widget XML files (but not package.xml)
	matches, err := filepath.Glob(filepath.Join(srcDir, "*.xml"))
	if err != nil {
		return err
	}
	for _, src := range matches {
		if strings.HasSuffix(src, "package.xml") {
			continue
		}
		dst := filepath.Join(outDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy xml %s: %w", filepath.Base(src), err)
		}
		fmt.Printf("[mxcli] copy %s\n", filepath.Base(src))
	}

	// Copy package.xml from src/ to dist/tmp/widgets/
	pkgXML := filepath.Join(srcDir, "package.xml")
	if _, err := os.Stat(pkgXML); err == nil {
		dst := filepath.Join(outDir, "package.xml")
		if err := copyFile(pkgXML, dst); err != nil {
			return fmt.Errorf("copy package.xml: %w", err)
		}
		fmt.Println("[mxcli] copy package.xml")
	}

	// Copy PNG icon/tile files (Windows-friendly glob, no bash extensions)
	pngFiles, _ := filepath.Glob(filepath.Join(srcDir, "*.png"))
	for _, src := range pngFiles {
		dst := filepath.Join(outDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy png %s: %w", filepath.Base(src), err)
		}
		fmt.Printf("[mxcli] copy %s\n", filepath.Base(src))
	}

	return nil
}

// repackageMPK re-creates the MPK from dist/tmp/widgets/ contents,
// overwriting the empty/incomplete MPK that pluggable-widgets-tools produced.
func repackageMPK(projectDir string) (string, error) {
	// Find the existing MPK from dist/1.0.0/
	existing, err := filepath.Glob(filepath.Join(projectDir, "dist", "1.0.0", "*.mpk"))
	if err != nil {
		return "", err
	}
	if len(existing) == 0 {
		return "", fmt.Errorf("no existing MPK in dist/1.0.0/")
	}
	mpkPath := existing[0]

	widgetsDir := filepath.Join(projectDir, "dist", "tmp", "widgets")

	// Check if there are any files to package
	hasFiles := false
	filepath.Walk(widgetsDir, func(path string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() {
			hasFiles = true
		}
		return nil
	})
	if !hasFiles {
		return "", fmt.Errorf("dist/tmp/widgets/ is empty — nothing to package")
	}

	// Create new MPK from dist/tmp/widgets/
	tmpPath := mpkPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	w := zip.NewWriter(f)

	err = filepath.Walk(widgetsDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(widgetsDir, path)
		entry, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = entry.Write(data)
		return err
	})

	closeErr := w.Close()
	f.Close()

	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return "", closeErr
	}

	// Atomic replace
	if err := os.Rename(tmpPath, mpkPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return mpkPath, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
