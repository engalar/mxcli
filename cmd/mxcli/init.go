// SPDX-License-Identifier: Apache-2.0

// init.go - Initialize Mendix project for Claude Code integration
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	initContainerRuntime string

	// vsixData holds the VS Code extension binary. It was previously populated
	// by a go:embed directive; it remains as a var so tests can nil it out to
	// disable extension installation without import-cycle issues.
	vsixData []byte
)

const mendixGitignore = `# Mendix project
/**/node_modules/
!/javascriptsource/**/node_modules/
/*.launch
/.classpath
/.mendix-cache/
/.project
/deployment/
/javasource/*/proxies/
/javasource/system/
/modeler-merge-marker
/nativemobile/builds/
/packages/
/project-settings.user.json
/releases/
*.mpr.lock
*.mpr.bak
/vendorlib/temp/
/.svn/

# MPR v2 journal files
/mprcontents/mprjournal*

# OS
.DS_Store

# mxcli
.claude/settings.local.json
mxcli
mxcli.exe
.mxcli/
`

var initCmd = &cobra.Command{
	Use:   "init [project-directory]",
	Short: "Initialize a Mendix project for Claude Code",
	Long: `Initialize a Mendix project for Claude Code AI-assisted development.

Creates .claude/ configuration with skills, commands, lint rules, and
CLAUDE.md project guide. Also sets up .devcontainer/ for containerized
development with the Mendix runtime.

Examples:
  # Initialize current directory
  mxcli init

  # Initialize a specific project directory
  mxcli init /path/to/my-mendix-project

  # Use Podman instead of Docker for the devcontainer
  mxcli init --container-runtime podman
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := "."
		if len(args) > 0 {
			projectDir = args[0]
		}

		absDir, err := filepath.Abs(projectDir)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		info, err := os.Stat(absDir)
		if err != nil {
			return fmt.Errorf("directory does not exist: %s", absDir)
		}
		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", absDir)
		}

		mprFile := findMprFile(absDir)
		if mprFile == "" {
			mprFile = "project.mpr"
		}
		projectName := filepath.Base(absDir)

		fmt.Printf("Initializing Claude Code for: %s\n", absDir)

		// Create .claude directory structure
		claudeDir := filepath.Join(absDir, ".claude")
		commandsDir := filepath.Join(claudeDir, "commands")
		lintRulesDir := filepath.Join(claudeDir, "lint-rules")
		claudeSkillsDir := filepath.Join(claudeDir, "skills")

		for _, dir := range []string{commandsDir, lintRulesDir, claudeSkillsDir} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", dir, err)
				return fmt.Errorf("creating directory: %s", dir)
			}
		}

		// Write settings.json and CLAUDE.md
		settingsPath := filepath.Join(claudeDir, "settings.json")
		if err := os.WriteFile(settingsPath, []byte(generateClaudeSettings(projectName, mprFile)), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing settings.json: %v\n", err)
			return fmt.Errorf("writing settings.json: %w", err)
		}
		fmt.Println("  Created .claude/settings.json")

		claudeMDPath := filepath.Join(absDir, "CLAUDE.md")
		if err := os.WriteFile(claudeMDPath, []byte(generateClaudeMD(projectName, mprFile, isDevcontainer())), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CLAUDE.md: %v\n", err)
			return fmt.Errorf("writing CLAUDE.md: %w", err)
		}
		fmt.Println("  Created CLAUDE.md")

		// Write commands
		cmdCount := 0
		err = fs.WalkDir(commandsFS, "commands", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			content, err := commandsFS.ReadFile(path)
			if err != nil {
				return err
			}
			targetPath := filepath.Join(commandsDir, d.Name())
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return err
			}
			cmdCount++
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing commands: %v\n", err)
		} else {
			fmt.Printf("  Created %d command files in .claude/commands/\n", cmdCount)
		}

		// Write lint rules
		lintRuleCount := 0
		err = fs.WalkDir(lintRulesFS, "lint-rules", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			content, err := lintRulesFS.ReadFile(path)
			if err != nil {
				return err
			}
			targetPath := filepath.Join(lintRulesDir, d.Name())
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return err
			}
			lintRuleCount++
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing lint rules: %v\n", err)
		} else {
			fmt.Printf("  Created %d lint rule files in .claude/lint-rules/\n", lintRuleCount)
		}

		// Write skills to .claude/skills/<name>/SKILL.md
		skillCount := 0
		err = fs.WalkDir(skillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Name() == "README.md" {
				return nil
			}
			content, err := skillsFS.ReadFile(path)
			if err != nil {
				return err
			}
			skillName := strings.TrimSuffix(d.Name(), ".md")
			skillDir := filepath.Join(claudeSkillsDir, skillName)
			if err := os.MkdirAll(skillDir, 0755); err != nil {
				return err
			}
			targetPath := filepath.Join(skillDir, "SKILL.md")
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return err
			}
			skillCount++
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing skills: %v\n", err)
		} else {
			fmt.Printf("  Created %d skill files in .claude/skills/\n", skillCount)
		}

		// Write example MDL files to mdl-examples/
		examplesDir := filepath.Join(absDir, "mdl-examples")
		if err := os.MkdirAll(examplesDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating mdl-examples directory: %v\n", err)
		} else {
			exampleCount := 0
			if err := fs.WalkDir(examplesFS, "examples", func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				content, err := examplesFS.ReadFile(path)
				if err != nil {
					return err
				}
				targetPath := filepath.Join(examplesDir, d.Name())
				// Don't overwrite existing user examples
				if _, statErr := os.Stat(targetPath); statErr == nil {
					return nil
				}
				if err := os.WriteFile(targetPath, content, 0644); err != nil {
					return err
				}
				exampleCount++
				return nil
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing examples: %v\n", err)
			} else if exampleCount > 0 {
				fmt.Printf("  Created %d example MDL files in mdl-examples/\n", exampleCount)
			}
		}

		// Create .devcontainer/ configuration
		devcontainerDir := filepath.Join(absDir, ".devcontainer")
		devcontainerJSON := filepath.Join(devcontainerDir, "devcontainer.json")
		dcExisted := false
		if _, err := os.Stat(devcontainerJSON); err == nil {
			dcExisted = true
		}
		if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating .devcontainer directory: %v\n", err)
		} else {
			dcJSON := generateDevcontainerJSON(projectName, mprFile, initContainerRuntime)
			if err := os.WriteFile(devcontainerJSON, []byte(dcJSON), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing devcontainer.json: %v\n", err)
			}
			dockerfile := filepath.Join(devcontainerDir, "Dockerfile")
			dcDockerfile := generateDockerfile(projectName, mprFile)
			if err := os.WriteFile(dockerfile, []byte(dcDockerfile), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing Dockerfile: %v\n", err)
			}
			if dcExisted {
				fmt.Println("\nUpdated .devcontainer/ configuration")
			} else {
				fmt.Println("\nCreated .devcontainer/ configuration")
			}
			if runtime.GOOS == "windows" {
				fmt.Println("\n⚠  You are running on Windows. The devcontainer is Linux-based,")
				fmt.Println("   so the Windows mxcli.exe will not work inside it.")
				fmt.Println("   The devcontainer will auto-download the correct Linux binary on first start.")
				fmt.Println("   Or run: mxcli setup mxcli --os linux --output ./mxcli")
			}
		}

		// Create .gitignore if it doesn't exist
		gitignorePath := filepath.Join(absDir, ".gitignore")
		if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
			if err := os.WriteFile(gitignorePath, []byte(mendixGitignore), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing .gitignore: %v\n", err)
			} else {
				fmt.Println("\nCreated .gitignore")
			}
		}

		// Create .playwright/cli.config.json
		playwrightDir := filepath.Join(absDir, ".playwright")
		playwrightConfig := filepath.Join(playwrightDir, "cli.config.json")
		if _, err := os.Stat(playwrightConfig); os.IsNotExist(err) {
			if err := os.MkdirAll(playwrightDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating .playwright directory: %v\n", err)
			} else {
				configContent := generatePlaywrightConfig()
				if err := os.WriteFile(playwrightConfig, []byte(configContent), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing playwright config: %v\n", err)
				} else {
					fmt.Println("\nCreated .playwright/cli.config.json")
				}
			}
		}

		fmt.Println("\n✓ Initialization complete!")
		fmt.Println("\nWhat was created:")
		fmt.Println("  • CLAUDE.md — project guide for Claude Code")
		fmt.Println("  • .claude/settings.json — permissions and environment")
		fmt.Println("  • .claude/commands/ — slash commands")
		fmt.Println("  • .claude/lint-rules/ — Starlark lint rules")
		fmt.Println("  • .claude/skills/ — MDL pattern guides")
		fmt.Println("  • mdl-examples/ — example MDL scripts (helpdesk-app.mdl)")
		fmt.Println("  • .devcontainer/ — dev container configuration")
		fmt.Println("  • .gitignore — Mendix ignore patterns")

		fmt.Println("\nNext steps:")
		fmt.Println("  1. Open this project in Claude Code")
		fmt.Println("  2. Ask Claude: 'explore this project'")
		fmt.Println("  3. See mdl-examples/helpdesk-app.mdl for a comprehensive MDL reference")
		fmt.Println("  4. Use './mxcli -p " + mprFile + "' to work with the project")
		return nil
	},
}

func findMprFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".mpr") {
			return e.Name()
		}
	}
	return ""
}

// isDevcontainer reports whether the current process is running inside a
// development container (Docker or Podman). Used to tailor CLAUDE.md:
// devcontainers use the local ./mxcli binary; host machines use the
// globally-installed mxcli on the PATH.
func isDevcontainer() bool {
	// Docker daemon creates /.dockerenv in every container it starts.
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// Podman writes /run/.containerenv inside its containers.
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	// GitHub Codespaces (also a devcontainer variant).
	if os.Getenv("CODESPACES") == "true" {
		return true
	}
	return false
}

func init() {
	initCmd.Flags().StringVar(&initContainerRuntime, "container-runtime", "docker", "Container runtime for devcontainer (docker or podman)")
}
