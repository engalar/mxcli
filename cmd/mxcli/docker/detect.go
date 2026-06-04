// SPDX-License-Identifier: Apache-2.0

// Package docker implements Docker build and deployment support for Mendix projects.
//
// # Platform differences for Mendix tool resolution
//
// On Windows, Studio Pro installs mxbuild.exe, mx.exe, and the runtime under
// a Program Files directory (e.g., D:\Program Files\Mendix\11.6.4\).
// CDN downloads (mxbuild tar.gz) contain Linux ELF binaries that cannot
// execute on Windows, so Studio Pro installations MUST be preferred.
//
// On macOS, Studio Pro installs to "/Applications/Mendix Studio Pro X.Y.Z.app/".
// CDN downloads are Linux ELF binaries and also cannot run on macOS.
// Studio Pro installations are therefore preferred on macOS as well.
//
// On Linux (CI, devcontainers), Studio Pro is not available.
// CDN downloads are the primary source for mxbuild and runtime.
//
// Resolution priority (all platforms):
//  1. Explicit path (--mxbuild-path)
//  2. PATH lookup
//  3. OS-specific known locations (Studio Pro on Windows and macOS)
//  4. Cached CDN downloads (~/.mxcli/mxbuild/)
//
// Path discovery on Windows must NOT hardcode drive letters. Use environment
// variables (PROGRAMFILES, PROGRAMW6432, SystemDrive) to locate install dirs.
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// macOSApplicationsDir is the root directory scanned for Studio Pro .app bundles
// on macOS. It is a package-level variable so tests can redirect it to a temp dir
// to avoid interference from real Studio Pro installations on the test host.
var macOSApplicationsDir = "/Applications"

// mxbuildBinaryName returns the platform-specific mxbuild binary name.
func mxbuildBinaryName() string {
	if runtime.GOOS == "windows" {
		return "mxbuild.exe"
	}
	return "mxbuild"
}

// mxbuildBinaryNames returns all candidate binary names for mxbuild.
// On Windows, the Linux binary ("mxbuild") may also be cached when downloaded
// for Docker, so both names must be checked.
func mxbuildBinaryNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"mxbuild.exe", "mxbuild"}
	}
	return []string{"mxbuild"}
}

// findMxBuildInDir looks for the mxbuild binary inside a directory.
// Checks: dir/mxbuild, dir/modeler/mxbuild (Mendix installation layout).
func findMxBuildInDir(dir string) string {
	for _, bin := range mxbuildBinaryNames() {
		candidates := []string{
			filepath.Join(dir, bin),
			filepath.Join(dir, "modeler", bin),
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && !info.IsDir() {
				return c
			}
		}
	}
	return ""
}

// resolveMxBuild finds the MxBuild executable.
// Priority: explicit path > PATH lookup > OS-specific known locations > cached downloads.
// On Windows, Studio Pro installations are checked before cached downloads because
// CDN downloads are Linux binaries that cannot run natively on Windows.
// The explicit path can be the binary itself or a directory containing it
// (e.g., a Mendix installation root with modeler/mxbuild inside).
func resolveMxBuild(explicitPath string, preferredVersion ...string) (string, error) {
	targetVersion := ""
	if len(preferredVersion) > 0 {
		targetVersion = preferredVersion[0]
	}

	if explicitPath != "" {
		info, err := os.Stat(explicitPath)
		if err != nil {
			return "", fmt.Errorf("mxbuild not found at %s: %w", explicitPath, err)
		}
		// If it's a directory, look for the binary inside it
		if info.IsDir() {
			if found := findMxBuildInDir(explicitPath); found != "" {
				return found, nil
			}
			return "", fmt.Errorf("mxbuild binary not found inside directory %s (looked for mxbuild and modeler/mxbuild)", explicitPath)
		}
		return explicitPath, nil
	}

	// Try PATH
	if p, err := exec.LookPath("mxbuild"); err == nil {
		return p, nil
	}

	// Try the exact Studio Pro installation for the project version first.
	// This keeps Docker builds on the same Mendix line as the project when
	// multiple installs are present locally.
	if targetVersion != "" {
		if studioProDir := ResolveStudioProDir(targetVersion); studioProDir != "" {
			if found := findMxBuildInDir(studioProDir); found != "" {
				return found, nil
			}
		}
	}

	// Try OS-specific known locations (Studio Pro on Windows) BEFORE cached downloads.
	// On Windows, CDN downloads are Linux binaries — Studio Pro's mxbuild.exe is preferred.
	if matches := globVersionedMatches(mxbuildSearchPaths()); len(matches) > 0 {
		if exact := exactVersionedPath(matches, targetVersion); exact != "" {
			return exact, nil
		}
		if newest := NewestVersionedPath(matches); newest != "" {
			return newest, nil
		}
	}

	if targetVersion != "" {
		if p := CachedMxBuildPath(targetVersion); p != "" {
			return p, nil
		}
	}

	// Try cached downloads (~/.mxcli/mxbuild/*/modeler/mxbuild)
	if p := AnyCachedMxBuildPath(); p != "" {
		return p, nil
	}

	return "", fmt.Errorf("mxbuild not found; install Mendix Studio Pro or specify --mxbuild-path")
}

// ResolveStudioProDir finds the Studio Pro installation directory for a specific
// Mendix version. Returns the installation root on Windows (e.g.,
// "D:\Program Files\Mendix\11.6.4") or the app Contents directory on macOS
// (e.g., "/Applications/Mendix Studio Pro 11.10.0.app/Contents").
// Returns empty string if not found or on Linux.
func ResolveStudioProDir(version string) string {
	switch runtime.GOOS {
	case "windows":
		for _, dir := range windowsProgramDirs() {
			candidate := filepath.Join(dir, "Mendix", version)
			if info, err := os.Stat(filepath.Join(candidate, "modeler", "mxbuild.exe")); err == nil && !info.IsDir() {
				return candidate
			}
		}
	case "darwin":
		return resolveStudioProDirMacOS(version)
	}
	return ""
}

// resolveStudioProDirMacOS finds an installed Studio Pro on macOS that matches
// the requested version. App bundles are named "Mendix Studio Pro X.Y.Z*.app"
// where X.Y.Z is the base version (e.g., "11.10.0-rc.7 Beta" for 11.10.0).
// Returns the bundle's Contents directory, or "" if not found.
func resolveStudioProDirMacOS(version string) string {
	matches, _ := filepath.Glob(filepath.Join(macOSApplicationsDir, "Mendix Studio Pro *.app"))
	re := regexp.MustCompile(`^Mendix Studio Pro (\d+\.\d+\.\d+)`)
	for _, match := range matches {
		base := strings.TrimSuffix(filepath.Base(match), ".app")
		m := re.FindStringSubmatch(base)
		if m == nil || m[1] != version {
			continue
		}
		candidate := filepath.Join(match, "Contents")
		if _, err := os.Stat(filepath.Join(candidate, "modeler", "mx")); err == nil {
			return candidate
		}
	}
	return ""
}

// windowsProgramDirs returns candidate Program Files directories on Windows,
// derived from environment variables and the system drive letter.
func windowsProgramDirs() []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(d string) {
		if d != "" && !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	for _, env := range []string{"PROGRAMFILES", "PROGRAMW6432", "PROGRAMFILES(X86)"} {
		add(os.Getenv(env))
	}
	// Fallback: derive from SystemDrive (e.g., "D:\Program Files").
	// SystemDrive returns "D:" without a trailing separator; filepath.Join
	// treats "D:" as a relative path, producing "D:Program Files" instead of
	// "D:\Program Files". Append the separator explicitly.
	if sysDrive := os.Getenv("SystemDrive"); sysDrive != "" {
		root := sysDrive + string(os.PathSeparator)
		add(filepath.Join(root, "Program Files"))
		add(filepath.Join(root, "Program Files (x86)"))
	}
	return dirs
}

// mendixSearchPaths returns OS-specific glob patterns for a Mendix binary
// (e.g., "mxbuild.exe", "mx.exe", "mxbuild", "mx") inside Studio Pro installations.
func mendixSearchPaths(binaryName string) []string {
	switch runtime.GOOS {
	case "windows":
		var paths []string
		for _, dir := range windowsProgramDirs() {
			paths = append(paths, filepath.Join(dir, "Mendix", "*", "modeler", binaryName))
		}
		return paths
	case "darwin":
		return []string{filepath.Join(macOSApplicationsDir, "Mendix Studio Pro *.app", "Contents", "modeler", binaryName)}
	default: // linux
		paths := []string{filepath.Join("/opt/mendix/*/modeler", binaryName)}
		if home, err := os.UserHomeDir(); err == nil {
			paths = append(paths, filepath.Join(home, ".mendix/*/modeler", binaryName))
		}
		return paths
	}
}

// mxbuildSearchPaths returns OS-specific glob patterns for MxBuild.
func mxbuildSearchPaths() []string {
	return mendixSearchPaths(mxbuildBinaryName())
}

// resolveJDK21 finds a JDK 21 installation.
// Priority: JAVA_HOME (verify version) > macOS java_home > java in PATH (verify version) > OS-specific known locations.
func resolveJDK21() (string, error) {
	// Try JAVA_HOME
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		if isJDK21(javaHome) {
			return javaHome, nil
		}
		// JAVA_HOME set but wrong version — continue searching
	}

	// On macOS, use /usr/libexec/java_home to find the real JDK path.
	// The /usr/bin/java shim is not a symlink, so EvalSymlinks won't resolve
	// it to the actual JDK, and MxBuild needs the real installation path.
	if runtime.GOOS == "darwin" {
		if javaHome, err := resolveMacOSJavaHome(); err == nil && isJDK21(javaHome) {
			return javaHome, nil
		}
	}

	// Try java in PATH
	if javaPath, err := exec.LookPath("java"); err == nil {
		// Resolve symlinks to find the actual installation
		resolved, err := filepath.EvalSymlinks(javaPath)
		if err == nil {
			javaPath = resolved
		}
		// java binary is typically at <jdk>/bin/java
		javaHome := filepath.Dir(filepath.Dir(javaPath))
		if isJDK21(javaHome) {
			return javaHome, nil
		}
	}

	// Try OS-specific known locations
	for _, pattern := range jdkSearchPaths() {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if isJDK21(m) {
				return m, nil
			}
		}
	}

	return "", fmt.Errorf(
		"JDK 21 not found; install JDK 21 and set JAVA_HOME, or in Git Bash run:\n" +
			"  export PATH=\"/c/Program Files/Eclipse Adoptium/jdk-21.x.y.z-hotspot/bin:$PATH\"",
	)
}

// resolveMacOSJavaHome uses /usr/libexec/java_home to find a JDK 21 on macOS.
func resolveMacOSJavaHome() (string, error) {
	out, err := exec.Command("/usr/libexec/java_home", "-v", "21").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// jdkSearchPaths returns OS-specific glob patterns for JDK installations.
func jdkSearchPaths() []string {
	switch runtime.GOOS {
	case "windows":
		var paths []string
		for _, dir := range windowsProgramDirs() {
			paths = append(paths,
				filepath.Join(dir, "Eclipse Adoptium", "jdk-21*"),
				filepath.Join(dir, "Java", "jdk-21*"),
				filepath.Join(dir, "Microsoft", "jdk-21*"),
			)
		}
		return paths
	case "darwin":
		return []string{
			"/Library/Java/JavaVirtualMachines/temurin-21*/Contents/Home",
			"/Library/Java/JavaVirtualMachines/jdk-21*/Contents/Home",
		}
	default: // linux
		return []string{
			"/usr/lib/jvm/java-21-*",
			"/usr/lib/jvm/temurin-21-*",
		}
	}
}

// isJDK21 checks if the given JAVA_HOME points to a JDK 21 installation.
func isJDK21(javaHome string) bool {
	javaBin := filepath.Join(javaHome, "bin", "java")
	if runtime.GOOS == "windows" {
		javaBin += ".exe"
	}
	if _, err := os.Stat(javaBin); err != nil {
		return false
	}

	out, err := exec.Command(javaBin, "-version").CombinedOutput()
	if err != nil {
		return false
	}

	return jdk21VersionRegex.Match(out)
}

var jdk21VersionRegex = regexp.MustCompile(`version "21[\.\s"]`)

// javaVersionString runs java -version and returns the output for diagnostics.
func javaVersionString(javaHome string) string {
	javaBin := filepath.Join(javaHome, "bin", "java")
	out, err := exec.Command(javaBin, "-version").CombinedOutput()
	if err != nil {
		return "(unknown)"
	}
	lines := strings.SplitN(string(out), "\n", 2)
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "(unknown)"
}
