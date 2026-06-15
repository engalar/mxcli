// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// widgetInfo holds the discovered metadata for one widget (from src/<Name>.xml).
type widgetInfo struct {
	Name        string // e.g. "MySlider"
	WidgetID    string // e.g. "com.acme.widget.MySlider.MySlider"
	DisplayName string // e.g. "My Slider" (from <name> element)
	XMLPath     string // absolute path to the XML file
}

// xmlWidgetRoot is a minimal struct for parsing just id and name from a widget XML.
type xmlWidgetRoot struct {
	XMLName     xml.Name `xml:"widget"`
	ID          string   `xml:"id,attr"`
	DisplayName string   `xml:"name"`
}

// discoverWidgets globs src/*.xml in projectDir and parses each to extract widgetID and name.
func discoverWidgets(projectDir string) ([]widgetInfo, error) {
	pattern := filepath.Join(projectDir, "src", "*.xml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var infos []widgetInfo
	for _, xmlPath := range matches {
		data, err := os.ReadFile(xmlPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", xmlPath, err)
		}
		var root xmlWidgetRoot
		if err := xml.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", xmlPath, err)
		}
		name := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
		infos = append(infos, widgetInfo{
			Name:        name,
			WidgetID:    root.ID,
			DisplayName: root.DisplayName,
			XMLPath:     xmlPath,
		})
	}
	return infos, nil
}

// validateWidgetIDFormat checks that a widget ID has at least 4 dot-separated segments.
func validateWidgetIDFormat(id string) error {
	parts := strings.Split(id, ".")
	if len(parts) < 4 {
		return fmt.Errorf("widget ID must have at least 4 dot-separated segments (e.g. com.acme.widget.MyName), got %q", id)
	}
	return nil
}

// validateWidgetInfo checks that a discovered widget has a valid ID format and non-empty name.
func validateWidgetInfo(info widgetInfo) error {
	if err := validateWidgetIDFormat(info.WidgetID); err != nil {
		return fmt.Errorf("widget %q: %w", info.Name, err)
	}
	if info.DisplayName == "" {
		return fmt.Errorf("widget %q: <name> element is empty in XML", info.Name)
	}
	return nil
}

// detectToolchain returns "bun" or "npm" depending on what is available in PATH.
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

// installDeps runs bun install or npm install if node_modules/ is absent.
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

// compileWidget invokes esbuild (via `bun x esbuild` or `npx esbuild`) to bundle one widget as CJS and ESM.
func compileWidget(projectDir, tool string, info widgetInfo) error {
	distJS := filepath.Join(projectDir, "dist", "com", "mendix", "widget", "custom", info.Name, info.Name+".js")
	distMJS := filepath.Join(projectDir, "dist", "com", "mendix", "widget", "custom", info.Name, info.Name+".mjs")
	if err := os.MkdirAll(filepath.Dir(distJS), 0755); err != nil {
		return err
	}

	src := filepath.Join(projectDir, "src", info.Name+".jsx")
	externals := []string{"--external:react", "--external:react-dom", "--external:big.js"}

	for _, out := range []struct{ format, outfile string }{
		{"cjs", distJS},
		{"esm", distMJS},
	} {
		esbuildArgs := append([]string{src, "--bundle",
			"--format=" + out.format, "--outfile=" + out.outfile},
			externals...)

		var cmd *exec.Cmd
		if tool == "bun" {
			cmd = exec.Command("bun", append([]string{"x", "esbuild"}, esbuildArgs...)...)
		} else {
			cmd = exec.Command("npx", append([]string{"esbuild"}, esbuildArgs...)...)
		}
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("esbuild %s (%s): %w", info.Name, out.format, err)
		}
	}
	return nil
}

// copyAssets copies XML descriptors, editor scripts, and icon/tile PNGs to dist/.
func copyAssets(projectDir string, infos []widgetInfo) error {
	srcDir := filepath.Join(projectDir, "src")
	distDir := filepath.Join(projectDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return err
	}

	if err := copyFile(filepath.Join(projectDir, "package.xml"), filepath.Join(distDir, "package.xml")); err != nil {
		return err
	}

	for _, info := range infos {
		suffixes := []string{
			".xml",
			".editorConfig.js",
			".editorPreview.js",
			".icon.png",
			".icon.dark.png",
			".tile.png",
			".tile.dark.png",
		}
		for _, suf := range suffixes {
			src := filepath.Join(srcDir, info.Name+suf)
			dst := filepath.Join(distDir, info.Name+suf)
			if _, err := os.Stat(src); err == nil {
				if err := copyFile(src, dst); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// copyFile copies src to dst, creating dst's parent directory if needed.
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

// packageMPK zips dist/ into <packageName>.mpk in projectDir and returns the MPK path.
func packageMPK(projectDir, packageName string) (string, error) {
	distDir := filepath.Join(projectDir, "dist")
	mpkPath := filepath.Join(projectDir, packageName+".mpk")

	f, err := os.Create(mpkPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()

	err = filepath.Walk(distDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(distDir, path)
		rel = filepath.ToSlash(rel)
		entry, err := w.Create(rel)
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
	return mpkPath, err
}

// verifyMPK opens the MPK ZIP and confirms each widget has a .js bundle inside.
func verifyMPK(mpkPath string, infos []widgetInfo) error {
	r, err := zip.OpenReader(mpkPath)
	if err != nil {
		return fmt.Errorf("opening MPK: %w", err)
	}
	defer r.Close()

	found := make(map[string]bool)
	for _, f := range r.File {
		found[f.Name] = true
	}
	for _, info := range infos {
		jsPath := fmt.Sprintf("com/mendix/widget/custom/%s/%s.js", info.Name, info.Name)
		if !found[jsPath] {
			return fmt.Errorf("MPK missing expected JS bundle: %s", jsPath)
		}
	}
	return nil
}

// findMPKInCwd globs *.mpk in the current working directory.
// Returns an error if 0 or 2+ files are found.
func findMPKInCwd() (string, error) {
	matches, err := filepath.Glob("*.mpk")
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no .mpk file found — run 'mxcli widget build' first")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple .mpk files found (%s) — specify one with --mpk", strings.Join(matches, ", "))
	}
}

// installMPK copies mpkPath into <projectDir>/widgets/, creating the directory if needed.
func installMPK(mpkPath, projectPath string) error {
	widgetsDir := filepath.Join(filepath.Dir(projectPath), "widgets")
	if err := os.MkdirAll(widgetsDir, 0755); err != nil {
		return fmt.Errorf("creating widgets/: %w", err)
	}
	dst := filepath.Join(widgetsDir, filepath.Base(mpkPath))
	if err := copyFile(mpkPath, dst); err != nil {
		return fmt.Errorf("copying MPK: %w", err)
	}
	fmt.Printf("Installed → %s\n", dst)
	return nil
}

// readPackageName extracts the <clientModule name="..."> attribute from package.xml.
func readPackageName(projectDir string) (string, error) {
	type xmlClientModule struct {
		Name string `xml:"name,attr"`
	}
	type xmlPackage struct {
		ClientModule xmlClientModule `xml:"clientModule"`
	}
	data, err := os.ReadFile(filepath.Join(projectDir, "package.xml"))
	if err != nil {
		return "", err
	}
	var pkg xmlPackage
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return "", err
	}
	if pkg.ClientModule.Name == "" {
		return "", fmt.Errorf("package.xml: missing clientModule name attribute")
	}
	return pkg.ClientModule.Name, nil
}

// runWidgetBuild implements `mxcli widget build [--dir <path>]`.
func runWidgetBuild(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		dir = "."
	}
	// Convert to absolute path: cmd.Dir + relative src would double the path in esbuild.
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}

	infos, err := discoverWidgets(dir)
	if err != nil {
		return fmt.Errorf("discovering widgets: %w", err)
	}
	if len(infos) == 0 {
		return fmt.Errorf("no widget XML files found in %s/src/", dir)
	}

	for _, info := range infos {
		if err := validateWidgetInfo(info); err != nil {
			return err
		}
	}
	fmt.Printf("Found %d widget(s): ", len(infos))
	for i, info := range infos {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(info.Name)
	}
	fmt.Println()

	tool, err := detectToolchain()
	if err != nil {
		return err
	}

	if err := installDeps(dir, tool); err != nil {
		return fmt.Errorf("installing dependencies: %w", err)
	}

	distDir := filepath.Join(dir, "dist")
	_ = os.RemoveAll(distDir)

	for _, info := range infos {
		fmt.Printf("  Compiling %s...\n", info.Name)
		if err := compileWidget(dir, tool, info); err != nil {
			return err
		}
	}

	if err := copyAssets(dir, infos); err != nil {
		return fmt.Errorf("copying assets: %w", err)
	}

	packageName, err := readPackageName(dir)
	if err != nil {
		return err
	}
	mpkPath, err := packageMPK(dir, packageName)
	if err != nil {
		return fmt.Errorf("packaging MPK: %w", err)
	}

	if err := verifyMPK(mpkPath, infos); err != nil {
		return err
	}

	fi, _ := os.Stat(mpkPath)
	size := int64(0)
	if fi != nil {
		size = fi.Size() / 1024
	}
	fmt.Printf("Built %s (%d widget(s), %d KB)\n", filepath.Base(mpkPath), len(infos), size)
	return nil
}
