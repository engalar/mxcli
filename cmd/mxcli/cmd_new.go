// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <app-name>",
	Short: "Create a new Mendix project",
	Long: `Create a new Mendix project with all tooling configured.

This command performs the following steps:
  1. Downloads MxBuild for the specified Mendix version
  2. Creates a blank Mendix project using mx create-project
  3. Initializes AI tooling and devcontainer configuration (mxcli init)
  4. Downloads the correct mxcli binary for the devcontainer (linux)

Examples:
  mxcli new MyApp
  mxcli new MyApp --version 11.8.0
  mxcli new MyApp --version 10.24.0 --output-dir ./projects/my-app
  mxcli new --list-versions
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listVersions, _ := cmd.Flags().GetBool("list-versions")
		if listVersions {
			listMendixVersions()
			return nil
		}

		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: mxcli new <app-name> [--version X.Y.Z]")
			return fmt.Errorf("app name is required")
		}
		appName := args[0]
		mendixVersion, _ := cmd.Flags().GetString("version")
		outputDir, _ := cmd.Flags().GetString("output-dir")
		skipInit, _ := cmd.Flags().GetBool("skip-init")
		force, _ := cmd.Flags().GetBool("force")

		if mendixVersion == "" {
			return fmt.Errorf("--version is required (e.g., --version 11.8.0)")
		}

		// Resolve output directory
		if outputDir == "" {
			outputDir = appName
		}
		absDir, err := filepath.Abs(outputDir)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		// Check if directory already exists and has content
		if entries, err := os.ReadDir(absDir); err == nil && len(entries) > 0 {
			if force {
				fmt.Printf("  Directory %s is not empty (--force), proceeding...\n", absDir)
			} else {
				return fmt.Errorf("directory %s already exists and is not empty\nUse --force to override, or choose a different --output-dir", absDir)
			}
		}

		// Step 1: Resolve mx binary.
		// On Windows and macOS, Studio Pro ships a native mx binary — prefer it.
		// CDN downloads contain Linux ELF binaries that cannot run on those platforms.
		// On Linux (CI, devcontainers), download mxbuild from CDN and derive mx.
		fmt.Printf("Step 1/4: Resolving MxBuild %s...\n", mendixVersion)
		mxPath, err := docker.ResolveMxForNewProject(mendixVersion, os.Stdout)
		if err != nil {
			if runtime.GOOS == "darwin" {
				return fmt.Errorf("could not find mx binary for version %s: %w\nOn macOS, install Mendix Studio Pro %s from the Mendix Marketplace", mendixVersion, err, mendixVersion)
			}
			return fmt.Errorf("could not find mx binary for version %s: %w", mendixVersion, err)
		}

		// Step 2: Create project
		fmt.Printf("\nStep 2/4: Creating Mendix project '%s'...\n", appName)
		if err := os.MkdirAll(absDir, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		mxCmd := exec.Command(mxPath, "create-project", "--app-name", appName)
		mxCmd.Dir = absDir
		mxCmd.Stdout = os.Stdout
		mxCmd.Stderr = os.Stderr
		if err := mxCmd.Run(); err != nil {
			return fmt.Errorf("creating project: %w", err)
		}

		// Clean up duplicate locale files that mx create-project generates.
		// MxBuild's AtlasPlugin.LoadTranslations crashes with "An item with the same
		// key has already been added" when duplicate translation.json files exist.
		if removed := cleanupDuplicateLocaleFiles(absDir); removed > 0 {
			fmt.Printf("  Cleaned %d duplicate locale file(s)\n", removed)
		}

		// Verify .mpr was created — mx create-project names the file after --app-name
		mprPath := filepath.Join(absDir, appName+".mpr")
		if _, err := os.Stat(mprPath); os.IsNotExist(err) {
			// Fallback: check for App.mpr (default when --app-name is not used)
			fallback := filepath.Join(absDir, "App.mpr")
			if _, err := os.Stat(fallback); err == nil {
				mprPath = fallback
			} else {
				// Last resort: find any .mpr file
				matches, _ := filepath.Glob(filepath.Join(absDir, "*.mpr"))
				if len(matches) > 0 {
					mprPath = matches[0]
				} else {
					return fmt.Errorf("mx create-project did not produce an .mpr file in %s", absDir)
				}
			}
		}
		fmt.Printf("  Created %s\n", mprPath)

		// Step 3: Initialize tooling
		if !skipInit {
			fmt.Printf("\nStep 3/4: Initializing AI tooling...\n")
			initCmd.Run(initCmd, []string{absDir})
		} else {
			fmt.Printf("\nStep 3/4: Skipped (--skip-init)\n")
		}

		// Step 4: Ensure correct mxcli binary for devcontainer (Linux only).
		// On Windows/macOS the devcontainer postCreateCommand downloads the Linux
		// binary automatically on first start — no pre-download needed here.
		fmt.Printf("\nStep 4/4: Setting up mxcli binary...\n")
		if runtime.GOOS == "linux" {
			mxcliBinPath := filepath.Join(absDir, "mxcli")
			self, err := os.Executable()
			if err == nil {
				selfBytes, err := os.ReadFile(self)
				if err == nil {
					if err := os.WriteFile(mxcliBinPath, selfBytes, 0755); err != nil {
						fmt.Fprintf(os.Stderr, "  Warning: could not copy mxcli binary: %v\n", err)
					} else {
						fmt.Printf("  Copied mxcli to %s\n", mxcliBinPath)
					}
				}
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: could not copy mxcli binary: %v\n", err)
			}
		} else {
			fmt.Println("  Skipped (devcontainer will download the Linux binary on first start).")
		}

		fmt.Printf("\n✓ Project '%s' created at %s\n", appName, absDir)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Open the project folder in VS Code")
		fmt.Println("  2. Reopen in Dev Container when prompted")
		fmt.Printf("  3. Run './mxcli -p %s' to start working\n", filepath.Base(mprPath))
		return nil
	},
}

// listMendixVersions lists available Mendix versions from all sources.
func listMendixVersions() {
	all := map[string]string{} // version → source label
	add := func(version, source string) {
		if version != "" {
			all[version] = source
		}
	}

	// 1. Cached downloads (~/.mxcli/mxbuild/<version>/)
	for _, v := range allCachedMxBuildVersions() {
		add(v, "cached download")
	}

	// 2. Windows: C:\Program Files\Mendix\<version>\modeler\mxbuild.exe
	if runtime.GOOS == "windows" {
		for _, env := range []string{"PROGRAMFILES", "PROGRAMW6432", "PROGRAMFILES(X86)"} {
			if d := os.Getenv(env); d != "" {
				entries, _ := os.ReadDir(filepath.Join(d, "Mendix"))
				for _, e := range entries {
					if e.IsDir() {
						if _, err := os.Stat(filepath.Join(d, "Mendix", e.Name(), "modeler", "mxbuild.exe")); err == nil {
							add(e.Name(), "Studio Pro")
						}
					}
				}
			}
		}
		if sd := os.Getenv("SystemDrive"); sd != "" {
			root := sd + string(os.PathSeparator)
			for _, dir := range []string{"Program Files", "Program Files (x86)"} {
				entries, _ := os.ReadDir(filepath.Join(root, dir, "Mendix"))
				for _, e := range entries {
					if e.IsDir() {
						if _, err := os.Stat(filepath.Join(root, dir, "Mendix", e.Name(), "modeler", "mxbuild.exe")); err == nil {
							add(e.Name(), "Studio Pro")
						}
					}
				}
			}
		}
	}

	// 3. macOS: /Applications/Mendix Studio Pro *.app
	if runtime.GOOS == "darwin" {
		matches, _ := filepath.Glob("/Applications/Mendix Studio Pro *.app")
		re := regexp.MustCompile(`^Mendix Studio Pro (\d+\.\d+\.\d+)`)
		for _, match := range matches {
			base := strings.TrimSuffix(filepath.Base(match), ".app")
			if m := re.FindStringSubmatch(base); m != nil {
				add(m[1], "Studio Pro")
			}
		}
	}

	if len(all) == 0 {
		fmt.Println("No Mendix versions found.")
		fmt.Println()
		fmt.Println("To find available Mendix versions, visit:")
		fmt.Println("  https://docs.mendix.com/releasenotes/studio-pro/")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  mxcli new MyApp --version X.Y.Z")
		fmt.Println()
		fmt.Println("mxcli automatically downloads the required version on first use.")
		return
	}

	var versions []string
	for v := range all {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		// Prefer newer versions first (simple semver comparison by string).
		// This works for X.Y.Z format when all are the same length.
		return versions[i] > versions[j]
	})

	fmt.Println("Available Mendix versions:")
	for _, v := range versions {
		fmt.Printf("  %-12s  (%s)\n", v, all[v])
	}
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mxcli new MyApp --version X.Y.Z")
	fmt.Println()
	fmt.Println("Cached downloads are stored in ~/.mxcli/mxbuild/<version>/.")
	fmt.Println("mxcli automatically downloads the required version on first use.")
}

// cleanupDuplicateLocaleFiles removes duplicate locale files that mx create-project
// generates in themesource/atlas_core/. MxBuild crashes when multiple translation.json
// files map to the same locale key (e.g., "en-US").
//
// Studio Pro-created projects have locale files only at:
//
//	themesource/atlas_core/locales/<locale>/translation.json
//
// mx create-project additionally creates duplicates in nested subdirectories
// (e.g., locales/en-US/atlas_core/locales/en-US/translation.json).
// We keep only the top-level files and remove any deeper duplicates.
func cleanupDuplicateLocaleFiles(projectDir string) int {
	localesDir := filepath.Join(projectDir, "themesource", "atlas_core", "locales")
	if _, err := os.Stat(localesDir); os.IsNotExist(err) {
		return 0
	}

	removed := 0
	// Walk locale directories (en-US, nl-NL, etc.)
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		localeDir := filepath.Join(localesDir, entry.Name())
		// Check for nested subdirectories that duplicate the locale
		subEntries, err := os.ReadDir(localeDir)
		if err != nil {
			continue
		}
		for _, sub := range subEntries {
			if sub.IsDir() {
				// Any subdirectory under a locale dir is a duplicate tree
				dupPath := filepath.Join(localeDir, sub.Name())
				if err := os.RemoveAll(dupPath); err == nil {
					removed++
				}
			}
		}
	}
	return removed
}

func init() {
	newCmd.Flags().Bool("list-versions", false, "List cached Mendix versions and show how to find available ones")
	newCmd.Flags().String("version", "", "Mendix version (e.g., 11.8.0) — required")
	newCmd.Flags().String("output-dir", "", "Output directory (default: ./<app-name>)")
	newCmd.Flags().Bool("skip-init", false, "Skip AI tooling initialization (mxcli init)")
	newCmd.Flags().Bool("force", false, "Allow creating project in a non-empty directory")
}
