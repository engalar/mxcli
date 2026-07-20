// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup development tools",
	Long: `Download and configure tools required for Mendix development.

Subcommands:
  mxbuild    Download MxBuild for the project's Mendix version
  mxruntime  Download the Mendix runtime for the project's Mendix version
  mxcli      Download an mxcli binary from GitHub releases

Examples:
  mxcli setup mxbuild -p app.mpr
  mxcli setup mxbuild --version 11.6.3
  mxcli setup mxruntime -p app.mpr
  mxcli setup mxruntime --version 11.6.3
  mxcli setup mxcli --os linux --arch amd64
`,
}

var setupMxBuildCmd = &cobra.Command{
	Use:   "mxbuild",
	Short: "Download MxBuild from the Mendix CDN",
	Long: `Download and cache MxBuild for a specific Mendix version.

The version is detected from the project file (--project) or specified
explicitly (--version). The binary is cached at ~/.mxcli/mxbuild/{version}/
and automatically found by 'mxcli docker build' and 'mxcli docker check'.

Examples:
  mxcli setup mxbuild -p app.mpr
  mxcli setup mxbuild --version 11.6.3
  mxcli setup mxbuild -p app.mpr --dry-run
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath, _ := cmd.Flags().GetString("project")
		versionStr, _ := cmd.Flags().GetString("version")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if versionStr == "" && projectPath == "" {
			return fmt.Errorf("specify --project (-p) or --version")
		}

		if versionStr == "" {
			reader, err := mmpr.Open(projectPath)
			if err != nil {
				return fmt.Errorf("opening project: %w", err)
			}
			pv := reader.ProjectVersion()
			reader.Close()
			versionStr = pv.ProductVersion
			fmt.Fprintf(os.Stdout, "Detected Mendix version: %s\n", versionStr)
		}

		if dryRun {
			url := docker.MxBuildCDNURL(versionStr, runtime.GOARCH)
			cacheDir, _ := docker.MxBuildCacheDir(versionStr)
			fmt.Fprintf(os.Stdout, "Dry run:\n")
			fmt.Fprintf(os.Stdout, "  Version:      %s\n", versionStr)
			fmt.Fprintf(os.Stdout, "  Architecture: %s\n", runtime.GOARCH)
			fmt.Fprintf(os.Stdout, "  URL:          %s\n", url)
			fmt.Fprintf(os.Stdout, "  Cache dir:    %s\n", cacheDir)

			if cached := docker.CachedMxBuildPath(versionStr); cached != "" {
				fmt.Fprintf(os.Stdout, "  Status:       already cached at %s\n", cached)
			} else {
				fmt.Fprintf(os.Stdout, "  Status:       not cached, would download\n")
			}
			return nil
		}

		path, err := docker.DownloadMxBuild(versionStr, os.Stdout)
		if err != nil {
			return fmt.Errorf("downloading mxbuild: %w", err)
		}

		fmt.Fprintf(os.Stdout, "\nMxBuild ready: %s\n", path)
		return nil
	},
}

var setupMxRuntimeCmd = &cobra.Command{
	Use:   "mxruntime",
	Short: "Download the Mendix runtime from the Mendix CDN",
	Long: `Download and cache the Mendix runtime for a specific Mendix version.

The version is detected from the project file (--project) or specified
explicitly (--version). The runtime is cached at ~/.mxcli/runtime/{version}/
and automatically used by 'mxcli docker build' when the PAD output does not
include the runtime (MxBuild 11.6.3+).

Examples:
  mxcli setup mxruntime -p app.mpr
  mxcli setup mxruntime --version 11.6.3
  mxcli setup mxruntime -p app.mpr --dry-run
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath, _ := cmd.Flags().GetString("project")
		versionStr, _ := cmd.Flags().GetString("version")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if versionStr == "" && projectPath == "" {
			return fmt.Errorf("specify --project (-p) or --version")
		}

		if versionStr == "" {
			reader, err := mmpr.Open(projectPath)
			if err != nil {
				return fmt.Errorf("opening project: %w", err)
			}
			pv := reader.ProjectVersion()
			reader.Close()
			versionStr = pv.ProductVersion
			fmt.Fprintf(os.Stdout, "Detected Mendix version: %s\n", versionStr)
		}

		if dryRun {
			url := docker.RuntimeCDNURL(versionStr)
			cacheDir, _ := docker.RuntimeCacheDir(versionStr)
			fmt.Fprintf(os.Stdout, "Dry run:\n")
			fmt.Fprintf(os.Stdout, "  Version:   %s\n", versionStr)
			fmt.Fprintf(os.Stdout, "  URL:       %s\n", url)
			fmt.Fprintf(os.Stdout, "  Cache dir: %s\n", cacheDir)

			if cached := docker.CachedRuntimePath(versionStr); cached != "" {
				fmt.Fprintf(os.Stdout, "  Status:    already cached at %s\n", cached)
			} else {
				fmt.Fprintf(os.Stdout, "  Status:    not cached, would download\n")
			}
			return nil
		}

		path, err := docker.DownloadRuntime(versionStr, os.Stdout)
		if err != nil {
			return fmt.Errorf("downloading runtime: %w", err)
		}

		fmt.Fprintf(os.Stdout, "\nMendix runtime ready: %s\n", path)
		return nil
	},
}

// mxcliBinaryURL returns the GitHub releases download URL for an mxcli binary.
// ver should be a release tag like "v0.4.0" or "nightly".
// targetOS is "linux", "darwin", or "windows". targetArch is "amd64" or "arm64".
func mxcliBinaryURL(repo, ver, targetOS, targetArch string) string {
	name := fmt.Sprintf("mxcli-%s-%s", targetOS, targetArch)
	if targetOS == "windows" {
		name += ".exe"
	}
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, ver, name)
}

// mxcliReleaseTag returns the release tag that matches the running binary's
// version string. Tagged releases use "vX.Y.Z"; nightly builds contain
// "nightly" and map to the "nightly" release tag.
func mxcliReleaseTag() string {
	v := version // package-level var set from ldflags
	if strings.Contains(v, "nightly") {
		return "nightly"
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	// Strip build metadata after first hyphen-with-commit (e.g. "v0.4.0-3-gabcdef" -> "v0.4.0")
	if idx := strings.IndexByte(v, '-'); idx > 0 {
		v = v[:idx]
	}
	return v
}

// downloadMxcliBinary downloads the mxcli binary for the given OS/arch from
// GitHub releases and writes it to outputPath with executable permissions.
func downloadMxcliBinary(repo, tag, targetOS, targetArch, outputPath string, w io.Writer) error {
	url := mxcliBinaryURL(repo, tag, targetOS, targetArch)
	fmt.Fprintf(w, "Downloading mxcli %s (%s/%s)...\n", tag, targetOS, targetArch)
	fmt.Fprintf(w, "  URL: %s\n", url)
	return downloadMxcliBinaryFromURL(url, outputPath, w)
}

// downloadMxcliBinaryFromURL fetches a binary from rawURL and writes it to
// outputPath with executable permissions (0755). Used by downloadMxcliBinary
// and directly by tests via an httptest.Server URL.
func downloadMxcliBinaryFromURL(rawURL, outputPath string, w io.Writer) error {
	resp, err := http.Get(rawURL)
	if err != nil {
		return fmt.Errorf("downloading mxcli: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading mxcli: HTTP %d from %s", resp.StatusCode, rawURL)
	}

	if resp.ContentLength > 0 {
		fmt.Fprintf(w, "  Size: %.1f MB\n", float64(resp.ContentLength)/(1024*1024))
	}

	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("writing binary: %w", err)
	}

	fmt.Fprintf(w, "  Saved to %s\n", outputPath)
	return nil
}

var setupMxcliCmd = &cobra.Command{
	Use:   "mxcli",
	Short: "Download an mxcli binary from GitHub releases",
	Long: `Download an mxcli binary for a specific OS/architecture from GitHub releases.

By default, downloads the version matching the currently running binary for
linux/amd64 — the typical target for devcontainers.

Examples:
  mxcli setup mxcli                           # Linux amd64 binary to ./mxcli
  mxcli setup mxcli --output /usr/local/bin/mxcli
  mxcli setup mxcli --os darwin --arch arm64   # macOS Apple Silicon
  mxcli setup mxcli --tag v0.4.0               # Specific release
  mxcli setup mxcli --tag nightly              # Latest nightly build
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetOS, _ := cmd.Flags().GetString("os")
		targetArch, _ := cmd.Flags().GetString("arch")
		output, _ := cmd.Flags().GetString("output")
		tag, _ := cmd.Flags().GetString("tag")
		repo, _ := cmd.Flags().GetString("repo")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if tag == "" {
			tag = mxcliReleaseTag()
		}

		if dryRun {
			url := mxcliBinaryURL(repo, tag, targetOS, targetArch)
			fmt.Fprintf(os.Stdout, "Dry run:\n")
			fmt.Fprintf(os.Stdout, "  Tag:    %s\n", tag)
			fmt.Fprintf(os.Stdout, "  OS:     %s\n", targetOS)
			fmt.Fprintf(os.Stdout, "  Arch:   %s\n", targetArch)
			fmt.Fprintf(os.Stdout, "  URL:    %s\n", url)
			fmt.Fprintf(os.Stdout, "  Output: %s\n", output)
			return nil
		}

		if err := downloadMxcliBinary(repo, tag, targetOS, targetArch, output, os.Stdout); err != nil {
			return fmt.Errorf("downloading mxcli: %w", err)
		}

		fmt.Fprintf(os.Stdout, "\nmxcli ready: %s\n", output)
		return nil
	},
}

// setupCompletionsCmd installs shell completion scripts.
var setupCompletionsCmd = &cobra.Command{
	Use:       "completions [bash|zsh|fish|powershell]",
	Short:     "Install shell completion scripts for bash/zsh/fish/powershell",
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := detectShell()
		if len(args) > 0 {
			shell = args[0]
		}

		path := completionPath(shell)
		if path == "" {
			return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", shell)
		}

		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create completion directory %s: %w", dir, err)
		}

		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create completion file %s: %w", path, err)
		}
		defer f.Close()

		switch shell {
		case "bash":
			if err := rootCmd.GenBashCompletionV2(f, true); err != nil {
				return err
			}
		case "zsh":
			if err := rootCmd.GenZshCompletion(f); err != nil {
				return err
			}
		case "fish":
			if err := rootCmd.GenFishCompletion(f, true); err != nil {
				return err
			}
		case "powershell":
			if err := rootCmd.GenPowerShellCompletionWithDesc(f); err != nil {
				return err
			}
			if msg, err := installPowerShellCompletion(path); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: %v\n", err)
			} else if msg != "" {
				fmt.Fprint(os.Stdout, msg)
			}
		}
		fmt.Fprintf(os.Stdout, "Shell completions installed: %s\n", path)
		fmt.Fprintf(os.Stdout, "Restart your shell or run: source %s\n", path)
		if shell == "zsh" {
			fmt.Fprintf(os.Stdout, "  Or: compinit\n")
		}
		return nil
	},
}

func detectShell() string {
	path, _ := os.LookupEnv("SHELL")
	if path == "" {
		// Fallback: detect from parent process
		return ""
	}
	switch filepath.Base(path) {
	case "bash", "zsh", "fish":
		return filepath.Base(path)
	}
	return ""
}

func completionPath(shell string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	switch shell {
	case "bash":
		return filepath.Join(home, ".bash_completion.d", "mxcli")
	case "zsh":
		return filepath.Join(home, ".zsh", "completions", "_mxcli")
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions", "mxcli.fish")
	case "powershell":
		return filepath.Join(home, ".config", "powershell", "completions", "mxcli.ps1")
	}
	return ""
	}

// installPowerShellCompletion adds a dot-source line to the PowerShell profile
// to load the completion script. Idempotent: skips if the marker already exists.
func installPowerShellCompletion(completionPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home dir: %w", err)
	}

	// Determine profile path: prefer PowerShell 7, fall back to 5.1 on Windows.
	profilePaths := []string{
		filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
	}
	if runtime.GOOS == "windows" {
		profilePaths = []string{
			filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		}
	}

	dotSourceLine := ". \"" + completionPath + "\""
	marker := "# mxcli completion"

	for _, profilePath := range profilePaths {
		// Read existing profile.
		data, err := os.ReadFile(profilePath)
		if err != nil {
			if !os.IsNotExist(err) {
				continue // permission error, try next path
			}
			// Profile doesn't exist — create directory.
			if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
				continue
			}
		} else if strings.Contains(string(data), marker) {
			// Already installed — skip silently for all paths.
			return "", nil
		}

		// Append with marker block.
		block := "\n" + marker + "\n" + dotSourceLine + "\n# mxcli completion (end)\n"
		f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}
		if _, err := f.WriteString(block); err != nil {
			f.Close()
			continue
		}
		f.Close()
		return fmt.Sprintf("  Added to PowerShell profile: %s\n", profilePath), nil
	}
	return "", nil
}

func init() {
	setupMxBuildCmd.Flags().String("version", "", "Mendix version to download (e.g., 11.6.3)")
	setupMxBuildCmd.Flags().Bool("dry-run", false, "Show what would be downloaded without downloading")

	setupMxRuntimeCmd.Flags().String("version", "", "Mendix version to download (e.g., 11.6.3)")
	setupMxRuntimeCmd.Flags().Bool("dry-run", false, "Show what would be downloaded without downloading")

	setupMxcliCmd.Flags().String("os", "linux", "Target operating system (linux, darwin, windows)")
	setupMxcliCmd.Flags().String("arch", "amd64", "Target architecture (amd64, arm64)")
	// ./mxcli (with ./) is intentional: the VS Code extension (mdl.mxcliPath) and
	// devcontainer postCreateCommand both reference the binary as "./mxcli".
	setupMxcliCmd.Flags().String("output", "./mxcli", "Output file path")
	setupMxcliCmd.Flags().String("tag", "", "Release tag to download (default: match running version)")
	setupMxcliCmd.Flags().String("repo", "engalar/mxcli", "GitHub repository")
	setupMxcliCmd.Flags().Bool("dry-run", false, "Show what would be downloaded without downloading")

	setupCmd.AddCommand(setupMxBuildCmd)
	setupCmd.AddCommand(setupMxRuntimeCmd)
	setupCmd.AddCommand(setupMxcliCmd)
	setupCmd.AddCommand(setupCompletionsCmd)
}
