// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// MxBuildCacheDir returns the cache directory for a specific MxBuild version.
// Layout: ~/.mxcli/mxbuild/{version}/
func MxBuildCacheDir(version string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".mxcli", "mxbuild", version), nil
}

// MxBuildCDNURL constructs the CDN download URL for MxBuild.
// arm64 -> https://cdn.mendix.com/runtime/arm64-mxbuild-{version}.tar.gz
// amd64 -> https://cdn.mendix.com/runtime/mxbuild-{version}.tar.gz
func MxBuildCDNURL(version, goarch string) string {
	switch goarch {
	case "arm64":
		return fmt.Sprintf("https://cdn.mendix.com/runtime/arm64-mxbuild-%s.tar.gz", version)
	default:
		return fmt.Sprintf("https://cdn.mendix.com/runtime/mxbuild-%s.tar.gz", version)
	}
}

// CachedMxBuildPath returns the path to a cached mxbuild binary for the given version,
// or empty string if not cached.
func CachedMxBuildPath(version string) string {
	cacheDir, err := MxBuildCacheDir(version)
	if err != nil {
		return ""
	}
	bin := filepath.Join(cacheDir, "modeler", mxbuildBinaryName())
	if info, err := os.Stat(bin); err == nil && !info.IsDir() {
		return bin
	}
	return ""
}

// AnyCachedMxBuildPath searches for any cached mxbuild version.
// Returns the path to the first mxbuild binary found, or empty string.
func AnyCachedMxBuildPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	pattern := filepath.Join(home, ".mxcli", "mxbuild", "*", "modeler", mxbuildBinaryName())
	matches, _ := filepath.Glob(pattern)
	if len(matches) > 0 {
		// Return the last match (likely newest version by lexicographic sort)
		return matches[len(matches)-1]
	}
	return ""
}

// DownloadMxBuild downloads and extracts MxBuild for the given version.
// Returns the path to the mxbuild binary.
// If already cached, skips the download.
func DownloadMxBuild(version string, w io.Writer) (string, error) {
	// Check cache first
	if cached := CachedMxBuildPath(version); cached != "" {
		fmt.Fprintf(w, "  MxBuild %s already cached at %s\n", version, cached)
		return cached, nil
	}

	cacheDir, err := MxBuildCacheDir(version)
	if err != nil {
		return "", err
	}

	url := MxBuildCDNURL(version, runtime.GOARCH)
	fmt.Fprintf(w, "  Downloading MxBuild %s for %s...\n", version, runtime.GOARCH)
	fmt.Fprintf(w, "  URL: %s\n", url)

	// Download
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading mxbuild: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading mxbuild: HTTP %d from %s", resp.StatusCode, url)
	}

	// Report download size if available
	if resp.ContentLength > 0 {
		fmt.Fprintf(w, "  Size: %.1f MB\n", float64(resp.ContentLength)/(1024*1024))
	}

	// Extract tar.gz directly from response body
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache directory: %w", err)
	}

	fmt.Fprintf(w, "  Extracting to %s...\n", cacheDir)
	if err := extractTarGz(resp.Body, cacheDir); err != nil {
		// Clean up on failure
		os.RemoveAll(cacheDir)
		return "", fmt.Errorf("extracting mxbuild: %w", err)
	}

	// Verify the binary exists
	bin := filepath.Join(cacheDir, "modeler", mxbuildBinaryName())
	if _, err := os.Stat(bin); err != nil {
		os.RemoveAll(cacheDir)
		return "", fmt.Errorf("mxbuild binary not found after extraction (expected %s)", bin)
	}

	fmt.Fprintf(w, "  MxBuild cached at %s\n", bin)
	return bin, nil
}

// RuntimeCDNURL returns the CDN download URL for the Mendix runtime.
// The runtime is pure Java — no architecture-specific variants needed.
func RuntimeCDNURL(version string) string {
	return fmt.Sprintf("https://cdn.mendix.com/runtime/mendix-%s.tar.gz", version)
}

// RuntimeCacheDir returns the cache directory for a specific runtime version.
// Layout: ~/.mxcli/runtime/{version}/
func RuntimeCacheDir(version string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".mxcli", "runtime", version), nil
}

// CachedRuntimePath returns the path to a cached runtime for the given version,
// or empty string if not cached. Checks for runtime/launcher/runtimelauncher.jar.
func CachedRuntimePath(version string) string {
	cacheDir, err := RuntimeCacheDir(version)
	if err != nil {
		return ""
	}
	jar := filepath.Join(cacheDir, "runtime", "launcher", "runtimelauncher.jar")
	if info, err := os.Stat(jar); err == nil && !info.IsDir() {
		return cacheDir
	}
	return ""
}

// DownloadRuntime downloads and extracts the Mendix runtime for the given version.
// Returns the cache directory path (containing runtime/launcher/runtimelauncher.jar).
// If already cached, skips the download.
// The tarball extracts to {version}/runtime/... so we strip the top-level directory.
func DownloadRuntime(version string, w io.Writer) (string, error) {
	// Check cache first
	if cached := CachedRuntimePath(version); cached != "" {
		fmt.Fprintf(w, "  Mendix runtime %s already cached at %s\n", version, cached)
		return cached, nil
	}

	cacheDir, err := RuntimeCacheDir(version)
	if err != nil {
		return "", err
	}

	url := RuntimeCDNURL(version)
	fmt.Fprintf(w, "  Downloading Mendix runtime %s...\n", version)
	fmt.Fprintf(w, "  URL: %s\n", url)

	// Download
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading runtime: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading runtime: HTTP %d from %s", resp.StatusCode, url)
	}

	// Report download size if available
	if resp.ContentLength > 0 {
		fmt.Fprintf(w, "  Size: %.1f MB\n", float64(resp.ContentLength)/(1024*1024))
	}

	// Extract tar.gz directly from response body, stripping the top-level directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache directory: %w", err)
	}

	fmt.Fprintf(w, "  Extracting to %s...\n", cacheDir)
	if err := extractTarGzStrip1(resp.Body, cacheDir); err != nil {
		// Clean up on failure
		os.RemoveAll(cacheDir)
		return "", fmt.Errorf("extracting runtime: %w", err)
	}

	// Verify the launcher jar exists
	jar := filepath.Join(cacheDir, "runtime", "launcher", "runtimelauncher.jar")
	if _, err := os.Stat(jar); err != nil {
		os.RemoveAll(cacheDir)
		return "", fmt.Errorf("runtime launcher not found after extraction (expected %s)", jar)
	}

	fmt.Fprintf(w, "  Runtime cached at %s\n", cacheDir)
	return cacheDir, nil
}

// extractTarGzStrip1 extracts a tar.gz stream to the target directory,
// stripping the first path component (equivalent to tar --strip-components=1).
func extractTarGzStrip1(r io.Reader, targetDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Strip the first path component
		name := header.Name
		if strings.Contains(name, "..") {
			continue
		}

		// Find the first / and strip everything before it
		idx := strings.IndexByte(name, '/')
		if idx < 0 {
			// Top-level entry (the directory itself), skip
			continue
		}
		name = name[idx+1:]
		if name == "" {
			continue
		}

		target := filepath.Join(targetDir, filepath.FromSlash(name))

		// Ensure the target is within targetDir
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(targetDir)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent directory for %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			f.Close()
		case tar.TypeSymlink:
			linkTarget := header.Linkname
			// Resolve effective symlink destination and verify it stays within targetDir
			var resolved string
			if filepath.IsAbs(linkTarget) {
				resolved = filepath.Clean(linkTarget)
			} else {
				resolved = filepath.Clean(filepath.Join(filepath.Dir(target), linkTarget))
			}
			allowedPrefix := filepath.Clean(targetDir) + string(os.PathSeparator)
			if !strings.HasPrefix(resolved, allowedPrefix) && resolved != filepath.Clean(targetDir) {
				continue
			}
			os.Remove(target)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent directory for symlink %s: %w", target, err)
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				return fmt.Errorf("creating symlink %s: %w", target, err)
			}
		}
	}

	return nil
}

// extractTarGz extracts a tar.gz stream to the target directory.
func extractTarGz(r io.Reader, targetDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Sanitize path to prevent directory traversal
		name := header.Name
		if strings.Contains(name, "..") {
			continue
		}

		target := filepath.Join(targetDir, filepath.FromSlash(name))

		// Ensure the target is within targetDir
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(targetDir)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent directory for %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			f.Close()
		case tar.TypeSymlink:
			linkTarget := header.Linkname
			// Resolve effective symlink destination and verify it stays within targetDir
			var resolved string
			if filepath.IsAbs(linkTarget) {
				resolved = filepath.Clean(linkTarget)
			} else {
				resolved = filepath.Clean(filepath.Join(filepath.Dir(target), linkTarget))
			}
			allowedPrefix := filepath.Clean(targetDir) + string(os.PathSeparator)
			if !strings.HasPrefix(resolved, allowedPrefix) && resolved != filepath.Clean(targetDir) {
				continue
			}
			os.Remove(target) // Remove existing if any
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent directory for symlink %s: %w", target, err)
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				return fmt.Errorf("creating symlink %s: %w", target, err)
			}
		}
	}

	return nil
}
