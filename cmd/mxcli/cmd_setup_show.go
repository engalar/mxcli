// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

// colorSupport returns true when stdout appears to be a colour-capable terminal.
// We use a simple heuristic: TERM is set and not "dumb", and NO_COLOR is not set.
// On Windows the same logic applies — modern Windows Terminal supports ANSI.
func colorSupport() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if term := os.Getenv("TERM"); term == "dumb" || term == "" {
		// Windows: check COLORTERM or WT_SESSION (Windows Terminal)
		if runtime.GOOS == "windows" {
			return os.Getenv("COLORTERM") != "" || os.Getenv("WT_SESSION") != ""
		}
		return false
	}
	return true
}

// Note: ansiRed, ansiGreen, ansiReset are declared in cmd_bson_diff.go.

func colorize(useColor bool, code, text string) string {
	if !useColor {
		return text
	}
	return code + text + ansiReset
}

// foundTag returns "[found]" (green) or "[not found]" (red).
func foundTag(exists bool, useColor bool) string {
	if exists {
		return colorize(useColor, ansiGreen, "[found]")
	}
	return colorize(useColor, ansiRed, "[not found]")
}

// setupPathExists returns true if the path exists and is a regular file.
func setupPathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// javaVersionOutput runs "java -version" and returns the first line of output.
// java -version writes to stderr, so we capture CombinedOutput.
func javaVersionOutput() (path string, versionLine string) {
	javaPath, err := exec.LookPath("java")
	if err != nil {
		// Not in PATH; try JAVA_HOME
		if jh := os.Getenv("JAVA_HOME"); jh != "" {
			javaBin := filepath.Join(jh, "bin", "java")
			if runtime.GOOS == "windows" {
				javaBin += ".exe"
			}
			if setupPathExists(javaBin) {
				javaPath = javaBin
			}
		}
	}
	if javaPath == "" {
		return "", ""
	}

	out, err := exec.Command(javaPath, "-version").CombinedOutput()
	if err != nil {
		return javaPath, ""
	}
	lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)
	return javaPath, strings.TrimSpace(lines[0])
}

// allCachedMxBuildVersions lists every version directory under ~/.mxcli/mxbuild/.
func allCachedMxBuildVersions() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	base := filepath.Join(home, ".mxcli", "mxbuild")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	sort.Strings(versions)
	return versions
}

// studioPro mx.exe paths on Windows: enumerate C:/Program Files/Mendix/*/modeler/mx.exe
func allStudioProMxPaths() []struct{ version, path string } {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		return nil
	}

	mxBin := "mx"
	if runtime.GOOS == "windows" {
		mxBin = "mx.exe"
	}

	var results []struct{ version, path string }
	seen := map[string]bool{}

	var patterns []string
	if runtime.GOOS == "windows" {
		for _, env := range []string{"PROGRAMFILES", "PROGRAMW6432"} {
			if d := os.Getenv(env); d != "" {
				patterns = append(patterns, filepath.Join(d, "Mendix", "*", "modeler", mxBin))
			}
		}
		if sd := os.Getenv("SystemDrive"); sd != "" {
			root := sd + string(os.PathSeparator)
			patterns = append(patterns,
				filepath.Join(root, "Program Files", "Mendix", "*", "modeler", mxBin),
				filepath.Join(root, "Program Files (x86)", "Mendix", "*", "modeler", mxBin),
			)
		}
	} else if runtime.GOOS == "darwin" {
		patterns = []string{
			"/Applications/Mendix Studio Pro *.app/Contents/modeler/" + mxBin,
		}
	}

	for _, pattern := range patterns {
		found, _ := filepath.Glob(pattern)
		for _, p := range found {
			if seen[p] {
				continue
			}
			seen[p] = true
			// Extract version from path
			// Windows: .../Mendix/11.6.6/modeler/mx.exe  → grandparent = 11.6.6
			// macOS:   .../Mendix Studio Pro 11.6.6.app/... → extract from app name
			ver := versionFromMxPath(p)
			results = append(results, struct{ version, path string }{ver, p})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].version < results[j].version })
	return results
}

// versionFromMxPath extracts the Mendix version from a known mx binary path.
func versionFromMxPath(p string) string {
	// macOS: Mendix Studio Pro X.Y.Z.app
	parts := strings.Split(filepath.ToSlash(p), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "Mendix Studio Pro ") {
			name := strings.TrimPrefix(part, "Mendix Studio Pro ")
			name = strings.TrimSuffix(name, ".app")
			// name may be "11.6.6" or "11.6.6 Beta" – take the first token
			return strings.Fields(name)[0]
		}
	}
	// Windows/Linux: grandparent directory = version
	dir := filepath.Dir(p)      // modeler
	parent := filepath.Dir(dir) // version dir
	return filepath.Base(parent)
}

// localBinaryPath returns the path to the installed mxcli-local binary.
func localBinaryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	name := "mxcli-local"
	if runtime.GOOS == "windows" {
		name = "mxcli-local.exe"
	}
	return filepath.Join(home, ".mxcli", "local", name)
}

// readVersionFile reads a one-line version file and returns its trimmed content.
func readVersionFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// runSetupShow prints a structured overview of all key mxcli environment paths.
func runSetupShow(cmd *cobra.Command, _ []string) {
	out := cmd.OutOrStdout()
	useColor := colorSupport()

	home, _ := os.UserHomeDir()

	// ── mxcli binaries ───────────────────────────────────────────────────────

	fmt.Fprintln(out, "=== mxcli Environment ===")
	fmt.Fprintln(out)

	// This binary
	selfPath, _ := os.Executable()
	fmt.Fprintf(out, "mxcli:  %s (%s)\n", selfPath, version)

	// Local runner
	localBin := localBinaryPath()
	localVer := readVersionFile(filepath.Join(home, ".mxcli", "local", "version"))
	if localVer == "" {
		localVer = "not installed"
	}
	localExists := setupPathExists(localBin)
	if localExists {
		fmt.Fprintf(out, "mxcli local:     %s (%s) %s\n", localBin, localVer, foundTag(true, useColor))
	} else {
		fmt.Fprintf(out, "mxcli local:     %s (%s)\n", localBin, localVer)
	}

	// ── mxbuild (CDN cache) ───────────────────────────────────────────────────

	fmt.Fprintln(out)
	fmt.Fprintln(out, "=== mxbuild Installations (CDN cache) ===")
	fmt.Fprintln(out)

	cachedVersions := allCachedMxBuildVersions()
	if len(cachedVersions) == 0 {
		fmt.Fprintf(out, "  (none found in ~/.mxcli/mxbuild/)\n")
		fmt.Fprintf(out, "  Hint: mxcli setup mxbuild --version <version>\n")
	} else {
		for _, ver := range cachedVersions {
			cacheDir, _ := docker.MxBuildCacheDir(ver)
			mxPath := docker.CachedMxPath(ver)
			if mxPath == "" {
				// mx may not be cached but mxbuild is
				mxbuildPath := docker.CachedMxBuildPath(ver)
				mxPath = filepath.Join(cacheDir, "modeler", "mx")
				if mxbuildPath != "" {
					// Check if mx exists alongside mxbuild
					mxCandidate := filepath.Join(filepath.Dir(mxbuildPath), "mx")
					if runtime.GOOS == "windows" {
						mxCandidate += ".exe"
					}
					if setupPathExists(mxCandidate) {
						mxPath = mxCandidate
					}
				}
			}
			exists := setupPathExists(mxPath)
			if !exists {
				// Fallback: the mxbuild binary at least
				mxPath = docker.CachedMxBuildPath(ver)
				exists = mxPath != ""
				if !exists {
					mxPath = filepath.Join(cacheDir, "modeler", "mx")
				}
			}
			fmt.Fprintf(out, "  %-10s  %s  %s\n", ver, mxPath, foundTag(exists, useColor))
		}
	}

	// ── Studio Pro (Windows/macOS native) ────────────────────────────────────

	studioEntries := allStudioProMxPaths()
	if len(studioEntries) > 0 {
		fmt.Fprintln(out)
		label := "Studio Pro (Windows)"
		if runtime.GOOS == "darwin" {
			label = "Studio Pro (macOS)"
		}
		fmt.Fprintf(out, "=== %s ===\n", label)
		fmt.Fprintln(out)
		for _, entry := range studioEntries {
			exists := setupPathExists(entry.path)
			fmt.Fprintf(out, "  %-10s  %s  %s\n", entry.version, entry.path, foundTag(exists, useColor))
		}
	}

	// ── Java ─────────────────────────────────────────────────────────────────

	fmt.Fprintln(out)
	fmt.Fprintln(out, "=== Java ===")
	fmt.Fprintln(out)

	javaPath, javaVersion := javaVersionOutput()
	if javaPath == "" {
		fmt.Fprintf(out, "  java:     %s\n", colorize(useColor, ansiRed, "(not found — install JDK 21 and add to PATH or set JAVA_HOME)"))
	} else if javaVersion == "" {
		fmt.Fprintf(out, "  java:     %s  %s\n", javaPath, foundTag(true, useColor))
	} else {
		fmt.Fprintf(out, "  java:     %s  %s, %s\n", javaPath, foundTag(true, useColor), javaVersion)
	}

}

var setupShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show mxcli environment paths and tool status",
	Long: `Print a structured overview of all key mxcli dependency paths and their status.

Reports:
  - mxcli binary path and version
  - Cached mxbuild/mx installations (~/.mxcli/mxbuild/)
  - Studio Pro installations (Windows/macOS only)
  - Java runtime path and version

Examples:
  mxcli setup show
`,
	Run: runSetupShow,
}

func init() {
	setupCmd.AddCommand(setupShowCmd)
}
