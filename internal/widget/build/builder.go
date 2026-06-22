package build

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
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

	mpkPath := matches[0]
	if err := verifyMPK(mpkPath); err != nil {
		return nil, fmt.Errorf("mpk verification failed: %w", err)
	}

	fi, _ := os.Stat(mpkPath)
	var size int64
	if fi != nil {
		size = fi.Size() / 1024
	}
	return &Result{MPKPath: mpkPath, SizeKB: size}, nil
}

// verifyMPK opens the generated .mpk and checks for the required files:
// package.xml and at least one widget XML file. Catches issues where the
// build toolchain produces a ZIP without the XML manifest.
func verifyMPK(mpkPath string) error {
	r, err := zip.OpenReader(mpkPath)
	if err != nil {
		return fmt.Errorf("open mpk: %w", err)
	}
	defer r.Close()

	// Find and parse package.xml
	var widgetFiles []string
	for _, f := range r.File {
		if f.Name != "package.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open package.xml: %w", err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read package.xml: %w", err)
		}
		var pkg struct {
			ClientModule struct {
				WidgetFiles []struct {
					Path string `xml:"path,attr"`
				} `xml:"widgetFiles>widgetFile"`
			} `xml:"clientModule"`
		}
		if err := xml.Unmarshal(data, &pkg); err != nil {
			return fmt.Errorf("parse package.xml: %w", err)
		}
		for _, wf := range pkg.ClientModule.WidgetFiles {
			if wf.Path != "" {
				widgetFiles = append(widgetFiles, wf.Path)
			}
		}
		break
	}

	if len(widgetFiles) == 0 {
		return fmt.Errorf("no package.xml or widget XML files found in mpk - build toolchain may have omitted them")
	}

	// Verify at least one widget XML exists
	widgetsByName := make(map[string]bool, len(r.File))
	for _, f := range r.File {
		widgetsByName[f.Name] = true
	}
	var found bool
	for _, wf := range widgetFiles {
		if widgetsByName[wf] {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("widget XML file(s) declared in package.xml not found in mpk: %v", widgetFiles)
	}
	return nil
}

// patchRollupCP patches pluggable-widgets-tools' rollup.config.mjs for Windows
// compatibility. Two issues:
//  1. `@(tile|icon)?(.dark).png` — bash extended glob not supported by fast-glob.
//  2. `src/**/*.xml` — fast-glob on Windows returns 0 matches for absolute
//     paths with drive letters and ** globstar. package.xml and widget XML
//     are silently omitted from the MPK.
func patchRollupCP(projectDir string) error {
	cfg := filepath.Join(projectDir, "node_modules",
		"@mendix", "pluggable-widgets-tools", "configs", "rollup.config.mjs")

	data, err := os.ReadFile(cfg)
	if err != nil {
		return fmt.Errorf("read rollup config: %w", err)
	}
	content := string(data)

	// Patch 1: bash @() glob for PNG files
	oldPng := "if (existsSync(`src/${widgetName}.icon.png`) || existsSync(`src/${widgetName}.tile.png`)) {\n                        cp(join(sourcePath, `src/${widgetName}.@(tile|icon)?(.dark).png`), outDir);\n                    }"
	newPng := "// [mxcli-patched-png]\n                    ['icon.png','icon.dark.png','tile.png','tile.dark.png'].forEach(function(f){var p=join(sourcePath,'src',widgetName+'.'+f);if(existsSync(p))cp(p,join(outDir,widgetName+'.'+f));})"
	if strings.Contains(content, oldPng) {
		content = strings.ReplaceAll(content, oldPng, newPng)
	}

	// Patch 2: src/**/*.xml glob — fast-glob on Windows can't resolve ** globstar
	// in absolute paths with drive letters. Replace with explicit file copies.
	oldXml := "cp(join(sourcePath, \"src/**/*.xml\"), outDir);"
	newXml := "// [mxcli-patched-xml]\n                    var f0=['package.xml',widgetName+'.xml'];f0.forEach(function(f){var p=join(sourcePath,'src',f);if(existsSync(p))cp(p,outDir);});"
	if strings.Contains(content, oldXml) {
		content = strings.ReplaceAll(content, oldXml, newXml)
	}

	// Write only if something changed
	if content == string(data) {
		return nil
	}
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		return fmt.Errorf("write patched rollup config: %w", err)
	}
	fmt.Println("[mxcli] patched rollup.config.mjs for Windows compatibility")
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
