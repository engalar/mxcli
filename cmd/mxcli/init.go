// SPDX-License-Identifier: Apache-2.0

// init.go - Initialize Mendix project for Claude Code integration
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var (
	initTools     []string
	initAllTools  bool
	initListTools bool
)

var initCmd = &cobra.Command{
	Use:   "init [project-directory]",
	Short: "Initialize a Mendix project for AI-assisted development",
	Long: `Initialize a Mendix project for AI-assisted development using mxcli.

This command creates configuration for AI coding assistants and MDL development.

Default: Claude Code + universal documentation

Examples:
  # Initialize with Claude Code (default)
  mxcli init

  # Initialize for Cursor
  mxcli init --tool cursor

  # Initialize for multiple tools
  mxcli init --tool claude --tool cursor --tool continue

  # Initialize for all supported tools
  mxcli init --all-tools

  # List supported tools
  mxcli init --list-tools

Supported Tools:
  - claude      Claude Code with skills and commands
  - cursor      Cursor AI with MDL rules
  - continue    Continue.dev with custom commands
  - windsurf    Windsurf (Codeium) with MDL rules
  - aider       Aider with project configuration
  - opencode    OpenCode AI agent with MDL commands and skills

All tools receive universal documentation in AGENTS.md and .ai-context/
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// List tools if requested
		if initListTools {
			fmt.Println("Supported AI Tools:")
			fmt.Println()
			for key, tool := range SupportedTools {
				fmt.Printf("  %-12s %s\n", key, tool.Description)
			}
			return
		}
		projectDir := "."
		if len(args) > 0 {
			projectDir = args[0]
		}

		// Make path absolute
		absDir, err := filepath.Abs(projectDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			os.Exit(1)
		}

		// Check if directory exists
		info, err := os.Stat(absDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: directory does not exist: %s\n", absDir)
			os.Exit(1)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: not a directory: %s\n", absDir)
			os.Exit(1)
		}

		// Find .mpr file
		mprFile := findMprFile(absDir)
		if mprFile == "" {
			mprFile = "project.mpr" // Default if not found
		}
		projectName := filepath.Base(absDir)

		// Determine which tools to initialize
		tools := initTools
		if initAllTools {
			tools = []string{}
			for key := range SupportedTools {
				tools = append(tools, key)
			}
		}
		if len(tools) == 0 {
			// Default to Claude + universal
			tools = []string{"claude"}
		}

		fmt.Printf("Initializing AI-assisted development for: %s\n", absDir)
		fmt.Printf("Tools: %v\n", tools)

		// Create .ai-context directory for universal content
		aiContextDir := filepath.Join(absDir, ".ai-context")
		skillsDir := filepath.Join(aiContextDir, "skills")
		examplesDir := filepath.Join(aiContextDir, "examples")

		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating .ai-context/skills directory: %v\n", err)
			os.Exit(1)
		}
		if err := os.MkdirAll(examplesDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating .ai-context/examples directory: %v\n", err)
			os.Exit(1)
		}

		// Create .claude directory for Claude-specific content (if Claude is selected)
		var claudeDir, commandsDir, lintRulesDir string
		if slices.Contains(tools, "claude") {
			claudeDir = filepath.Join(absDir, ".claude")
			commandsDir = filepath.Join(claudeDir, "commands")
			lintRulesDir = filepath.Join(claudeDir, "lint-rules")

			if err := os.MkdirAll(commandsDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating .claude/commands directory: %v\n", err)
				os.Exit(1)
			}
			if err := os.MkdirAll(lintRulesDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating .claude/lint-rules directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Create .opencode directory for OpenCode-specific content (if OpenCode is selected)
		var opencodeCommandsDir, opencodeSkillsDir string
		if slices.Contains(tools, "opencode") {
			opencodeDir := filepath.Join(absDir, ".opencode")
			opencodeCommandsDir = filepath.Join(opencodeDir, "commands")
			opencodeSkillsDir = filepath.Join(opencodeDir, "skills")

			if err := os.MkdirAll(opencodeCommandsDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating .opencode/commands directory: %v\n", err)
				os.Exit(1)
			}
			if err := os.MkdirAll(opencodeSkillsDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating .opencode/skills directory: %v\n", err)
				os.Exit(1)
			}

			// Lint rules stay in .claude/lint-rules/ (read by mxcli lint).
			// Ensure that directory exists even when claude tool is not selected.
			if !slices.Contains(tools, "claude") {
				if lintRulesDir == "" {
					lintRulesDir = filepath.Join(absDir, ".claude", "lint-rules")
				}
				if err := os.MkdirAll(lintRulesDir, 0755); err != nil {
					fmt.Fprintf(os.Stderr, "Error creating .claude/lint-rules directory: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// Write universal skills to .ai-context/skills/
		skillCount := 0
		err = fs.WalkDir(skillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			// Read from embedded FS
			content, err := skillsFS.ReadFile(path)
			if err != nil {
				return err
			}
			// Write to target directory
			targetPath := filepath.Join(skillsDir, d.Name())
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return err
			}
			skillCount++
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing skills: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  Created %d skill files in .ai-context/skills/\n", skillCount)

		// Write tool-specific configurations
		for _, toolName := range tools {
			toolConfig, ok := SupportedTools[toolName]
			if !ok {
				fmt.Fprintf(os.Stderr, "Warning: unknown tool '%s', skipping\n", toolName)
				continue
			}

			fmt.Printf("\nConfiguring %s:\n", toolConfig.Name)

			for _, file := range toolConfig.Files {
				filePath := filepath.Join(absDir, file.Path)

				// Create parent directory if needed
				if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
					fmt.Fprintf(os.Stderr, "  Error creating directory for %s: %v\n", file.Path, err)
					continue
				}

				// Generate content
				content := file.Content(projectName, mprFile)

				// Write file
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "  Error writing %s: %v\n", file.Path, err)
					continue
				}

				fmt.Printf("  Created %s\n", file.Path)
			}

			// Claude-specific: write commands and lint rules
			if toolName == "claude" && commandsDir != "" {
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
					fmt.Fprintf(os.Stderr, "  Error writing commands: %v\n", err)
				} else {
					fmt.Printf("  Created %d command files in .claude/commands/\n", cmdCount)
				}

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
					fmt.Fprintf(os.Stderr, "  Error writing lint rules: %v\n", err)
				} else {
					fmt.Printf("  Created %d lint rule files in .claude/lint-rules/\n", lintRuleCount)
				}
			}

			// OpenCode-specific: write commands, lint rules, and skills
			if toolName == "opencode" && opencodeCommandsDir != "" {
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
					targetPath := filepath.Join(opencodeCommandsDir, d.Name())
					if err := os.WriteFile(targetPath, content, 0644); err != nil {
						return err
					}
					cmdCount++
					return nil
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Error writing OpenCode commands: %v\n", err)
				} else {
					fmt.Printf("  Created %d command files in .opencode/commands/\n", cmdCount)
				}

				lintRuleCount := 0
				// Only write lint rules from the OpenCode path when Claude is not also
				// being initialised — the Claude path already writes the same files to
				// .claude/lint-rules/ and we don't want duplicate log output or writes.
				if !slices.Contains(tools, "claude") {
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
						fmt.Fprintf(os.Stderr, "  Error writing lint rules: %v\n", err)
					} else {
						fmt.Printf("  Created %d lint rule files in .claude/lint-rules/\n", lintRuleCount)
					}
				}

				skillCount2 := 0
				err = fs.WalkDir(skillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						return nil
					}
					// Skip README
					if d.Name() == "README.md" {
						return nil
					}
					content, err := skillsFS.ReadFile(path)
					if err != nil {
						return err
					}
					// Derive skill name from filename (strip .md)
					skillName := strings.TrimSuffix(d.Name(), ".md")
					// Create per-skill subdirectory
					skillDir := filepath.Join(opencodeSkillsDir, skillName)
					if err := os.MkdirAll(skillDir, 0755); err != nil {
						return err
					}
					// Wrap content with OpenCode frontmatter
					wrapped := wrapSkillContent(skillName, content)
					targetPath := filepath.Join(skillDir, "SKILL.md")
					if err := os.WriteFile(targetPath, wrapped, 0644); err != nil {
						return err
					}
					skillCount2++
					return nil
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Error writing OpenCode skills: %v\n", err)
				} else {
					fmt.Printf("  Created %d skill directories in .opencode/skills/\n", skillCount2)
				}
			}
		}

		// Write universal AGENTS.md
		fmt.Println("\nCreating universal documentation:")
		for _, file := range UniversalFiles {
			filePath := filepath.Join(absDir, file.Path)
			content := file.Content(projectName, mprFile)

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "  Error writing %s: %v\n", file.Path, err)
				os.Exit(1)
			}

			fmt.Printf("  Created %s\n", file.Path)
		}

		// Create or update .devcontainer/ configuration
		devcontainerDir := filepath.Join(absDir, ".devcontainer")
		devcontainerJSON := filepath.Join(devcontainerDir, "devcontainer.json")
		dcExisted := false
		if _, err := os.Stat(devcontainerJSON); err == nil {
			dcExisted = true
		}
		if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating .devcontainer directory: %v\n", err)
		} else {
			dcJSON := generateDevcontainerJSON(projectName, mprFile)
			if err := os.WriteFile(devcontainerJSON, []byte(dcJSON), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "  Error writing devcontainer.json: %v\n", err)
			}
			dockerfile := filepath.Join(devcontainerDir, "Dockerfile")
			dcDockerfile := generateDockerfile(projectName, mprFile)
			if err := os.WriteFile(dockerfile, []byte(dcDockerfile), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "  Error writing Dockerfile: %v\n", err)
			}
			if dcExisted {
				fmt.Println("\nUpdated .devcontainer/ configuration")
			} else {
				fmt.Println("\nCreated .devcontainer/ configuration")
			}
		}

		// Create .playwright/cli.config.json for playwright-cli
		playwrightDir := filepath.Join(absDir, ".playwright")
		playwrightConfig := filepath.Join(playwrightDir, "cli.config.json")
		if _, err := os.Stat(playwrightConfig); os.IsNotExist(err) {
			if err := os.MkdirAll(playwrightDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating .playwright directory: %v\n", err)
			} else {
				configContent := generatePlaywrightConfig()
				if err := os.WriteFile(playwrightConfig, []byte(configContent), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "  Error writing playwright config: %v\n", err)
				} else {
					fmt.Println("\nCreated .playwright/cli.config.json")
				}
			}
		}

		// Install VS Code extension if Claude or OpenCode is selected
		if slices.Contains(tools, "claude") || slices.Contains(tools, "opencode") {
			installVSCodeExtension(absDir)
		}

		fmt.Println("\n✓ Initialization complete!")
		fmt.Println("\nWhat was created:")
		fmt.Println("  • AGENTS.md - Universal AI assistant guide")
		fmt.Println("  • .ai-context/skills/ - MDL pattern guides")
		fmt.Println("  • .devcontainer/ - Dev container configuration")
		for _, toolName := range tools {
			if config, ok := SupportedTools[toolName]; ok {
				fmt.Printf("  • %s configuration\n", config.Name)
			}
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  1. Open this project in your AI coding assistant")
		fmt.Println("  2. Read AGENTS.md for command reference")
		fmt.Println("  3. Use './mxcli -p " + mprFile + "' to work with the project")
		fmt.Println("\nAvailable commands:")
		fmt.Println("  ./mxcli check <script>         - Validate MDL")
		fmt.Println("  ./mxcli exec <script>          - Execute MDL")
		fmt.Println("  ./mxcli search \"pattern\"       - Search project")
		fmt.Println("  ./mxcli lint                   - Check for issues")
	},
}

// yamlSingleQuote wraps s in YAML single quotes and escapes any internal
// single quotes by doubling them, so the result is safe to embed in a YAML
// value without further quoting.
func yamlSingleQuote(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "'", "''")
	return "'" + s + "'"
}

// wrapSkillContent prepends OpenCode-compatible YAML frontmatter to a skill file.
// OpenCode requires each skill to live in its own subdirectory as SKILL.md and
// the file must start with YAML frontmatter containing name, description, and
// compatibility fields.
func wrapSkillContent(skillName string, content []byte) []byte {
	description := extractSkillDescription(content)
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\ncompatibility: opencode\n---\n\n", yamlSingleQuote(skillName), yamlSingleQuote(description))
	return append([]byte(frontmatter), content...)
}

// extractSkillDescription returns a one-line description for the skill by
// finding the first top-level markdown heading (# ...) and stripping a leading
// "Skill: " prefix if present.  Falls back to "MDL skill" if no heading is
// found.
func extractSkillDescription(content []byte) string {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			desc := strings.TrimPrefix(line, "# ")
			desc = strings.TrimPrefix(desc, "Skill: ")
			return strings.TrimSpace(desc)
		}
	}
	return "MDL skill"
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

func generateClaudeMD(projectName, mprFile string) string {
	mprPath := mprFile
	if mprPath == "" {
		mprPath = "<project>.mpr"
	}

	bt := "`"    // backtick helper
	bt3 := "```" // triple backtick helper

	var sb strings.Builder
	w := func(s string) { sb.WriteString(s) }

	// ── Header ──────────────────────────────────────────────────────
	w("# Mendix Project: " + projectName + "\n\n")
	w("This is a Mendix project configured for AI-assisted development using mxcli and MDL (Mendix Definition Language).\n\n")

	// ── Communication Style ────────────────────────────────────────
	w("## Communication Style\n\n")
	w("When discussing changes with the user:\n\n")
	w("- **Never show raw MDL scripts in chat.** Instead, describe changes in plain language as a numbered list.\n")
	w("- After the user approves, write the MDL to a script file, validate it, and execute it silently.\n")
	w("- Only show MDL code if the user explicitly asks to see the script.\n")
	w("- When reporting results, summarize what was created/modified in plain language.\n")
	w("- **Always quote identifiers** in MDL scripts with double quotes (" + bt + "\"Name\"" + bt + "). This prevents conflicts with MDL reserved keywords and is always safe — quotes are stripped automatically. Quote entity names, attribute names, parameter names, variable names, and association names.\n\n")
	w("**Example — instead of showing MDL code, write this:**\n\n")
	w("> Here's what I'll do:\n")
	w("> 1. Create a new **Customer** entity in MyModule with:\n")
	w(">    - **Name** (text, up to 100 characters)\n")
	w(">    - **Email** (text, up to 200 characters)\n")
	w(">    - **Age** (whole number)\n")
	w(">\n")
	w("> Shall I go ahead?\n\n")

	// ── mxcli Location ─────────────────────────────────────────────
	w("## Important: mxcli Location\n\n")
	w("The " + bt + "mxcli" + bt + " tool is located in the **root folder of this project**, not in the system PATH. Always use the local path:\n\n")
	w(bt3 + "bash\n./mxcli -p " + mprPath + "    # Correct - uses local binary\n" + bt3 + "\n\n")
	w("**Do NOT use** " + bt + "mxcli" + bt + " directly - it will fail with \"command not found\". Always prefix with " + bt + "./" + bt + " to run the local binary.\n\n")

	// ── Mendix Validation Tool ─────────────────────────────────────
	w("## Mendix Validation Tool (mx)\n\n")
	w("The " + bt + "mx" + bt + " command validates Mendix projects (same checks as Studio Pro). To set it up:\n\n")
	w(bt3 + "bash\n")
	w("./mxcli setup mxbuild -p " + mprPath + "    # Auto-download for project's Mendix version\n")
	w(bt3 + "\n\n")
	w("After setup, " + bt + "mx" + bt + " is at " + bt + "~/.mxcli/mxbuild/{version}/modeler/mx" + bt + ". Usage:\n\n")
	w(bt3 + "bash\n")
	w("~/.mxcli/mxbuild/*/modeler/mx check " + mprPath + "   # Validate project\n")
	w("./mxcli docker check -p " + mprPath + "               # Alternative (auto-downloads mxbuild)\n")
	w(bt3 + "\n\n")

	// ── Quick Start ─────────────────────────────────────────────────
	w("## Quick Start\n\n")
	w("### Execute a Single Command\n\n")
	w("Use the " + bt + "-c" + bt + " flag to run a single MDL command:\n\n")
	w(bt3 + "bash\n")
	w("./mxcli -p " + mprPath + " -c \"SHOW MODULES\"              # List all modules\n")
	w("./mxcli -p " + mprPath + " -c \"SHOW STRUCTURE\"             # Project overview\n")
	w("./mxcli -p " + mprPath + " -c \"SHOW ENTITIES IN MyModule\"  # Entities in a module\n")
	w("./mxcli -p " + mprPath + " -c \"DESCRIBE ENTITY MyModule.Customer\"  # Entity details\n")
	w(bt3 + "\n\n")
	w("### Execute an MDL Script File\n\n")
	w(bt3 + "bash\n./mxcli exec script.mdl -p " + mprPath + "\n" + bt3 + "\n\n")
	w("### Start Interactive REPL\n\n")
	w(bt3 + "bash\n./mxcli\n# Then: CONNECT LOCAL '" + mprPath + "';\n" + bt3 + "\n\n")

	// ── IMPORTANT: Before Writing MDL ───────────────────────────────
	w("## IMPORTANT: Before Writing MDL Scripts or Working with Data\n\n")
	w("**Read the relevant skill files FIRST before writing any MDL, seeding data, or doing database/import work:**\n\n")
	w("| Skill File | When to Read |\n")
	w("|------------|-------------|\n")
	w("| " + bt + ".ai-context/skills/write-microflows.md" + bt + " | **Before writing any microflow** - syntax, common mistakes, validation checklist |\n")
	w("| " + bt + ".ai-context/skills/create-page.md" + bt + " | **Before creating any page** - widget syntax reference |\n")
	w("| " + bt + ".ai-context/skills/alter-page.md" + bt + " | **Before modifying pages** - ALTER PAGE/SNIPPET SET, INSERT, DROP, REPLACE |\n")
	w("| " + bt + ".ai-context/skills/overview-pages.md" + bt + " | CRUD page patterns (overview + edit) |\n")
	w("| " + bt + ".ai-context/skills/master-detail-pages.md" + bt + " | Master-detail page patterns |\n")
	w("| " + bt + ".ai-context/skills/generate-domain-model.md" + bt + " | Entity, association, enumeration syntax |\n")
	w("| " + bt + ".ai-context/skills/organize-project.md" + bt + " | Folders, MOVE command, project structure |\n")
	w("| " + bt + ".ai-context/skills/manage-security.md" + bt + " | Security roles, GRANT/REVOKE, access control |\n")
	w("| " + bt + ".ai-context/skills/manage-navigation.md" + bt + " | Navigation profiles, menus, home/login pages |\n")
	w("| " + bt + ".ai-context/skills/check-syntax.md" + bt + " | **Pre-flight** validation checklist |\n")
	w("| " + bt + ".ai-context/skills/demo-data.md" + bt + " | **READ for any database/import work** - Mendix ID system, demo data |\n")
	w("| " + bt + ".ai-context/skills/test-microflows.md" + bt + " | **READ for testing** - test annotations, file formats, Docker setup |\n")
	w("\n")
	w("**Always validate before presenting to user:**\n\n")
	w(bt3 + "bash\n")
	w("./mxcli check script.mdl                              # Syntax check\n")
	w("./mxcli check script.mdl -p " + mprPath + " --references  # With reference validation\n")
	w(bt3 + "\n\n")

	// ── MDL Commands by Domain ──────────────────────────────────────
	w("## MDL Commands by Domain\n\n")

	// Exploration & Structure
	w("### Exploration & Structure\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW MODULES" + bt + " | List all modules |\n")
	w("| " + bt + "SHOW STRUCTURE [DEPTH 1|2|3] [IN Module] [ALL]" + bt + " | Compact project overview at different detail levels |\n")
	w("| " + bt + "SHOW CALLERS OF Module.Microflow" + bt + " | Find what calls a microflow |\n")
	w("| " + bt + "SHOW CALLEES OF Module.Microflow" + bt + " | Find what a microflow calls |\n")
	w("| " + bt + "SHOW REFERENCES OF Module.Entity" + bt + " | Find all references to an element |\n")
	w("| " + bt + "SHOW IMPACT OF Module.Entity" + bt + " | Impact analysis for changes |\n")
	w("| " + bt + "SHOW CONTEXT OF Module.Microflow" + bt + " | Show callers + callees + references |\n")
	w("| " + bt + "SEARCH 'keyword'" + bt + " | Full-text search across all strings and source |\n")
	w("| " + bt + "HELP [topic]" + bt + " | Show all commands or help on a topic |\n")
	w("\n")

	// Domain Model
	w("### Domain Model\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW ENTITIES [IN Module]" + bt + " | List entities |\n")
	w("| " + bt + "SHOW ASSOCIATIONS [IN Module]" + bt + " | List associations |\n")
	w("| " + bt + "SHOW ENUMERATIONS [IN Module]" + bt + " | List enumerations |\n")
	w("| " + bt + "SHOW CONSTANTS [IN Module]" + bt + " | List constants |\n")
	w("| " + bt + "DESCRIBE ENTITY Module.Entity" + bt + " | Show entity definition in MDL |\n")
	w("| " + bt + "DESCRIBE ASSOCIATION Module.Assoc" + bt + " | Show association definition |\n")
	w("| " + bt + "DESCRIBE ENUMERATION Module.Enum" + bt + " | Show enumeration definition |\n")
	w("| " + bt + "CREATE MODULE ModuleName" + bt + " | Create a new module |\n")
	w("| " + bt + "CREATE PERSISTENT ENTITY ..." + bt + " | Create a persistent entity with attributes |\n")
	w("| " + bt + "CREATE NON-PERSISTENT ENTITY ..." + bt + " | Create a non-persistent (transient) entity |\n")
	w("| " + bt + "CREATE ASSOCIATION ..." + bt + " | Create an association between entities |\n")
	w("| " + bt + "CREATE ENUMERATION ..." + bt + " | Create an enumeration |\n")
	w("| " + bt + "ALTER ENTITY Module.Entity ADD ..." + bt + " | Add/rename/modify/drop attributes, indexes, docs |\n")
	w("| " + bt + "DROP ENTITY Module.Entity" + bt + " | Delete an entity |\n")
	w("| " + bt + "DROP ASSOCIATION Module.Assoc" + bt + " | Delete an association |\n")
	w("| " + bt + "DROP ENUMERATION Module.Enum" + bt + " | Delete an enumeration |\n")
	w("\n")

	// Microflows & Nanoflows
	w("### Microflows & Nanoflows\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW MICROFLOWS [IN Module]" + bt + " | List microflows |\n")
	w("| " + bt + "SHOW NANOFLOWS [IN Module]" + bt + " | List nanoflows |\n")
	w("| " + bt + "DESCRIBE MICROFLOW Module.Flow" + bt + " | Show microflow definition in MDL |\n")
	w("| " + bt + "DESCRIBE NANOFLOW Module.Flow" + bt + " | Show nanoflow definition in MDL |\n")
	w("| " + bt + "CREATE MICROFLOW ... BEGIN ... END;" + bt + " | Create a microflow with activities |\n")
	w("| " + bt + "CREATE NANOFLOW ... BEGIN ... END;" + bt + " | Create a nanoflow with activities |\n")
	w("| " + bt + "DROP MICROFLOW Module.Flow" + bt + " | Delete a microflow |\n")
	w("| " + bt + "DROP NANOFLOW Module.Flow" + bt + " | Delete a nanoflow |\n")
	w("\n")

	// Pages & Snippets
	w("### Pages & Snippets\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW PAGES [IN Module]" + bt + " | List pages |\n")
	w("| " + bt + "SHOW SNIPPETS [IN Module]" + bt + " | List snippets |\n")
	w("| " + bt + "DESCRIBE PAGE Module.Page" + bt + " | Show page definition in MDL |\n")
	w("| " + bt + "DESCRIBE SNIPPET Module.Snippet" + bt + " | Show snippet definition |\n")
	w("| " + bt + "CREATE PAGE ... { widgets }" + bt + " | Create a page with widget syntax |\n")
	w("| " + bt + "CREATE SNIPPET ... { widgets }" + bt + " | Create a reusable snippet |\n")
	w("| " + bt + "ALTER PAGE Module.Page { ops }" + bt + " | Modify page in-place (SET, INSERT, DROP, REPLACE) |\n")
	w("| " + bt + "ALTER SNIPPET Module.Snippet { ops }" + bt + " | Modify snippet in-place |\n")
	w("| " + bt + "DROP PAGE Module.Page" + bt + " | Delete a page |\n")
	w("| " + bt + "DROP SNIPPET Module.Snippet" + bt + " | Delete a snippet |\n")
	w("\n")

	// Security
	w("### Security\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW PROJECT SECURITY" + bt + " | Security level, admin, demo users overview |\n")
	w("| " + bt + "SHOW MODULE ROLES [IN Module]" + bt + " | Module-level roles |\n")
	w("| " + bt + "SHOW USER ROLES" + bt + " | Project-level user roles |\n")
	w("| " + bt + "SHOW DEMO USERS" + bt + " | Configured demo users |\n")
	w("| " + bt + "SHOW ACCESS ON MICROFLOW|PAGE|ENTITY Mod.Name" + bt + " | Role access on element |\n")
	w("| " + bt + "SHOW SECURITY MATRIX [IN Module]" + bt + " | Full access overview |\n")
	w("| " + bt + "CREATE MODULE ROLE Mod.Role" + bt + " | Create a module role |\n")
	w("| " + bt + "CREATE USER ROLE Name (Mod.Role, ...)" + bt + " | Create a user role aggregating module roles |\n")
	w("| " + bt + "ALTER USER ROLE Name ADD|REMOVE MODULE ROLES (...)" + bt + " | Modify user role |\n")
	w("| " + bt + "GRANT EXECUTE ON MICROFLOW Mod.MF TO Mod.Role" + bt + " | Grant microflow access |\n")
	w("| " + bt + "GRANT VIEW ON PAGE Mod.Page TO Mod.Role" + bt + " | Grant page access |\n")
	w("| " + bt + "GRANT Mod.Role ON Mod.Entity (CREATE, DELETE, READ *, WRITE *)" + bt + " | Grant entity access |\n")
	w("| " + bt + "REVOKE EXECUTE|VIEW|role ON element FROM role" + bt + " | Revoke access |\n")
	w("| " + bt + "ALTER PROJECT SECURITY LEVEL OFF|PROTOTYPE|PRODUCTION" + bt + " | Set security level |\n")
	w("| " + bt + "ALTER PROJECT SECURITY DEMO USERS ON|OFF" + bt + " | Toggle demo users |\n")
	w("| " + bt + "CREATE DEMO USER 'name' PASSWORD 'pass' (UserRole, ...)" + bt + " | Create demo user |\n")
	w("| " + bt + "DROP MODULE ROLE|USER ROLE|DEMO USER ..." + bt + " | Delete roles/users |\n")
	w("\n")

	// Navigation
	w("### Navigation\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW NAVIGATION" + bt + " | Summary of all profiles |\n")
	w("| " + bt + "SHOW NAVIGATION MENU [Profile]" + bt + " | Menu tree for profile or all |\n")
	w("| " + bt + "SHOW NAVIGATION HOMES" + bt + " | Home page assignments across profiles |\n")
	w("| " + bt + "DESCRIBE NAVIGATION [Profile]" + bt + " | Full MDL output (round-trippable) |\n")
	w("| " + bt + "CREATE OR REPLACE NAVIGATION Profile ..." + bt + " | Full replacement of a navigation profile |\n")
	w("\n")

	// Project Settings
	w("### Project Settings\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW SETTINGS" + bt + " | Overview of all settings |\n")
	w("| " + bt + "DESCRIBE SETTINGS" + bt + " | Full MDL output (round-trippable) |\n")
	w("| " + bt + "ALTER SETTINGS MODEL Key = Value" + bt + " | AfterStartupMicroflow, HashAlgorithm, JavaVersion, etc. |\n")
	w("| " + bt + "ALTER SETTINGS CONFIGURATION 'Name' Key = Value" + bt + " | DatabaseType, DatabaseUrl, HttpPortNumber, etc. |\n")
	w("| " + bt + "ALTER SETTINGS CONSTANT 'Name' VALUE 'val' IN CONFIGURATION 'cfg'" + bt + " | Override constant per configuration |\n")
	w("| " + bt + "ALTER SETTINGS LANGUAGE Key = Value" + bt + " | DefaultLanguageCode |\n")
	w("| " + bt + "ALTER SETTINGS WORKFLOWS Key = Value" + bt + " | UserEntity, DefaultTaskParallelism |\n")
	w("\n")

	// Business Events & Java Actions
	w("### Business Events & Java Actions\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW DATABASE CONNECTIONS [IN Module]" + bt + " | List database connections |\n")
	w("| " + bt + "DESCRIBE DATABASE CONNECTION Mod.Name" + bt + " | Show connection definition in MDL |\n")
	w("| " + bt + "SHOW BUSINESS EVENTS [IN Module]" + bt + " | List business event services |\n")
	w("| " + bt + "DESCRIBE BUSINESS EVENT SERVICE Mod.Name" + bt + " | Full MDL output |\n")
	w("| " + bt + "CREATE BUSINESS EVENT SERVICE ..." + bt + " | Create a business event service |\n")
	w("| " + bt + "DROP BUSINESS EVENT SERVICE Mod.Name" + bt + " | Delete a service |\n")
	w("| " + bt + "SHOW JAVA ACTIONS [IN Module]" + bt + " | List Java actions |\n")
	w("| " + bt + "DESCRIBE JAVA ACTION Mod.Name" + bt + " | Full MDL output with signature |\n")
	w("| " + bt + "CREATE JAVA ACTION ... AS $$ ... $$" + bt + " | Create with inline Java code |\n")
	w("| " + bt + "DROP JAVA ACTION Mod.Name" + bt + " | Delete a Java action |\n")
	w("\n")

	// OData
	w("### OData\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SHOW ODATA CLIENTS [IN Module]" + bt + " | List consumed OData services |\n")
	w("| " + bt + "SHOW ODATA SERVICES [IN Module]" + bt + " | List published OData services |\n")
	w("| " + bt + "DESCRIBE ODATA CLIENT Mod.Name" + bt + " | Full consumed OData MDL output |\n")
	w("| " + bt + "DESCRIBE ODATA SERVICE Mod.Name" + bt + " | Full published OData MDL output |\n")
	w("| " + bt + "CREATE ODATA CLIENT ..." + bt + " | Create a consumed OData service |\n")
	w("| " + bt + "CREATE ODATA SERVICE ..." + bt + " | Create a published OData service |\n")
	w("| " + bt + "ALTER ODATA CLIENT|SERVICE ..." + bt + " | Modify an OData service |\n")
	w("| " + bt + "DROP ODATA CLIENT|SERVICE Mod.Name" + bt + " | Delete an OData service |\n")
	w("\n")

	// External SQL
	w("### External SQL\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "SQL CONNECT <driver> '<dsn>' AS <alias>" + bt + " | Connect to external database (postgres) |\n")
	w("| " + bt + "SQL DISCONNECT <alias>" + bt + " | Close connection |\n")
	w("| " + bt + "SQL CONNECTIONS" + bt + " | List active connections (alias + driver only) |\n")
	w("| " + bt + "SQL <alias> SHOW TABLES" + bt + " | List tables via information_schema |\n")
	w("| " + bt + "SQL <alias> DESCRIBE <table>" + bt + " | Show columns, types, nullability |\n")
	w("| " + bt + "SQL <alias> <any-sql>" + bt + " | Raw SQL passthrough to external DB |\n")
	w("\n")

	// Catalog Queries
	w("### Catalog Queries\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "REFRESH CATALOG" + bt + " | Build catalog (metadata only) |\n")
	w("| " + bt + "REFRESH CATALOG FULL" + bt + " | Full catalog with activities, widgets, cross-refs |\n")
	w("| " + bt + "SHOW CATALOG TABLES" + bt + " | List available catalog tables |\n")
	w("| " + bt + "SELECT ... FROM CATALOG.ENTITIES WHERE ..." + bt + " | SQL queries against project metadata |\n")
	w("\n")

	w("Available catalog tables: " + bt + "CATALOG.MODULES" + bt + ", " + bt + "CATALOG.ENTITIES" + bt + ", " + bt + "CATALOG.MICROFLOWS" + bt + ", " + bt + "CATALOG.PAGES" + bt + ", " + bt + "CATALOG.WORKFLOWS" + bt + ", " + bt + "CATALOG.ENUMERATIONS" + bt + ", " + bt + "CATALOG.ASSOCIATIONS" + bt + ", " + bt + "CATALOG.SNIPPETS" + bt + ", " + bt + "CATALOG.REFS" + bt + " (requires FULL mode).\n\n")

	// Project Organization
	w("### Project Organization\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "MOVE PAGE|MICROFLOW|SNIPPET|... Mod.Name TO FOLDER 'path'" + bt + " | Move element to folder |\n")
	w("| " + bt + "MOVE PAGE Mod.Name TO Module" + bt + " | Move to module root |\n")
	w("| " + bt + "MOVE ENTITY Old.Name TO NewModule" + bt + " | Move entity across modules |\n")
	w("| " + bt + "SHOW WORKFLOWS [IN Module]" + bt + " | List workflows |\n")
	w("| " + bt + "DESCRIBE WORKFLOW Module.Workflow" + bt + " | Show workflow definition |\n")
	w("| " + bt + "SHOW WIDGETS [IN Module]" + bt + " | Widget discovery (experimental) |\n")
	w("\n")

	// ── Script Validation ───────────────────────────────────────────
	w("## Script Validation (mxcli check)\n\n")
	w("Before executing MDL scripts, validate them for syntax errors:\n\n")
	w(bt3 + "bash\n./mxcli check script.mdl\n" + bt3 + "\n\n")
	w("### Check with Reference Validation\n\n")
	w("Validate that all referenced modules, entities, and associations exist:\n\n")
	w(bt3 + "bash\n./mxcli check script.mdl -p " + mprPath + " --references\n" + bt3 + "\n\n")
	w("The reference checker is smart - it automatically skips references to objects that are created within the same script.\n\n")

	// ── Linting ─────────────────────────────────────────────────────
	w("## Linting\n\n")
	w("Check your project for common issues:\n\n")
	w(bt3 + "bash\n")
	w("# Lint the project\n./mxcli lint -p " + mprPath + "\n\n")
	w("# With colored output\n./mxcli lint -p " + mprPath + " --color\n\n")
	w("# List available rules\n./mxcli lint -p " + mprPath + " --list-rules\n\n")
	w("# Output as SARIF\n./mxcli lint -p " + mprPath + " --format sarif > results.sarif\n")
	w(bt3 + "\n\n")
	w("### Built-in Rules\n\n")
	w("| Rule | Category | Description |\n")
	w("|------|----------|-------------|\n")
	w("| MPR001 | quality | PascalCase naming conventions (entities, microflows, pages, enumerations) |\n")
	w("| MPR002 | quality | Empty microflows (no activities) |\n")
	w("| MPR003 | design | Domain model size (>15 persistent entities per module) |\n")
	w("| MPR004 | correctness | Empty validation feedback message (CE0091) |\n")
	w("| MPR005 | correctness | Unconfigured image widget source |\n")
	w("| MPR006 | correctness | Empty containers (runtime crash) |\n")
	w("| MPR007 | security | Navigation page without allowed role (CE0557) |\n")
	w("| SEC001 | security | Persistent entity without access rules |\n")
	w("| SEC002 | security | Weak password policy (minimum length < 8) |\n")
	w("| SEC003 | security | Demo users active at non-development security level |\n")
	w("\n")
	w("### Bundled Starlark Rules\n\n")
	w("27 additional rules in " + bt + ".claude/lint-rules/*.star" + bt + ":\n\n")
	w("| Rule | Category | Description |\n")
	w("|------|----------|-------------|\n")
	w("| SEC004 | security | Guest access enabled - review anonymous entity access |\n")
	w("| SEC005 | security | Strict mode disabled - XPath constraint enforcement off |\n")
	w("| SEC006 | security | PII attributes exposed without access rules |\n")
	w("| SEC007 | security | Anonymous unconstrained READ (DIVD-2022-00019) |\n")
	w("| SEC008 | security | PII entities readable without row scoping |\n")
	w("| SEC009 | security | Large entities missing member-level access restrictions |\n")
	w("| ARCH001 | architecture | Cross-module data access in pages |\n")
	w("| ARCH002 | architecture | Data changes should go through microflows |\n")
	w("| ARCH003 | architecture | Persistent entities need a unique business key |\n")
	w("| QUAL001 | quality | McCabe cyclomatic complexity threshold |\n")
	w("| QUAL002 | quality | Missing documentation on entities/microflows |\n")
	w("| QUAL003 | quality | Long microflows (too many activities) |\n")
	w("| QUAL004 | quality | Orphaned/unreferenced elements |\n")
	w("| DESIGN001 | design | Entity with too many attributes |\n")
	w("| CONV001 | naming | Boolean attributes must start with Is/Has/Can/Should/Was/Will |\n")
	w("| CONV002 | quality | String/numeric attributes should not have default values |\n")
	w("| CONV003 | naming | Pages should follow Entity_NewEdit/View/Overview naming |\n")
	w("| CONV004 | naming | Enumerations should be prefixed with ENUM_ |\n")
	w("| CONV005 | naming | Snippets should be prefixed with SNIPPET_ |\n")
	w("| CONV006 | security | Entity access rules should not grant Create/Delete rights |\n")
	w("| CONV007 | security | All persistent entity access rules need XPath constraints |\n")
	w("| CONV008 | security | Each module role should map to exactly one user role |\n")
	w("| CONV009 | quality | Microflows should have at most 15 objects |\n")
	w("| CONV015 | quality | Entities should not have validation rules |\n")
	w("| CONV016 | performance | Entities should not have event handlers |\n")
	w("| CONV017 | performance | Attributes should not be calculated (virtual) |\n")
	w("\n")
	w("Custom Starlark rules in " + bt + ".claude/lint-rules/*.star" + bt + " are loaded automatically. See " + bt + "write-lint-rules" + bt + " skill for authoring guide.\n\n")

	// ── Report ──────────────────────────────────────────────────────
	w("## Best Practices Report\n\n")
	w("Generate a scored best practices report:\n\n")
	w(bt3 + "bash\n")
	w("# Markdown report (default)\n./mxcli report -p " + mprPath + "\n\n")
	w("# JSON report\n./mxcli report -p " + mprPath + " --format json\n\n")
	w("# HTML report\n./mxcli report -p " + mprPath + " --format html\n")
	w(bt3 + "\n\n")
	w("The report scores 6 categories (Naming, Security, Quality, Architecture, Performance, Design) on a 0-100 scale. See " + bt + "assess-quality" + bt + " skill for the full assessment guide.\n\n")

	// ── Slash Commands ──────────────────────────────────────────────
	w("## Slash Commands\n\n")
	w("Use these commands to quickly perform common tasks:\n\n")
	w("| Command | Description |\n")
	w("|---------|-------------|\n")
	w("| " + bt + "/create-entity" + bt + " | Create a new entity with attributes |\n")
	w("| " + bt + "/create-crud" + bt + " | Generate entity + overview + edit pages |\n")
	w("| " + bt + "/refresh-catalog" + bt + " | Rebuild catalog for queries |\n")
	w("| " + bt + "/explore" + bt + " | Explore project structure |\n")
	w("| " + bt + "/check-script" + bt + " | Validate MDL script syntax |\n")
	w("| " + bt + "/validate-project" + bt + " | Run mx check to validate project |\n")
	w("| " + bt + "/lint" + bt + " | Check project for common issues |\n")
	w("| " + bt + "/test" + bt + " | Run Playwright tests against the running app |\n")
	w("| " + bt + "/diff-local" + bt + " | Show git diff of local MPR v2 changes |\n")
	w("| " + bt + "/diff-script" + bt + " | Compare MDL script against project state |\n")
	w("\n")

	// ── Skills Reference ────────────────────────────────────────────
	w("## Skills Reference\n\n")
	w("Skills are in " + bt + ".ai-context/skills/" + bt + ". Read the relevant skill before starting work.\n\n")

	w("### Quick Reference\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| cheatsheet-variables | Variable declaration syntax quick lookup |\n")
	w("| cheatsheet-errors | Common MDL errors and fixes |\n")
	w("\n")

	w("### Syntax Reference\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| mdl-entities | Entity, attribute, association syntax |\n")
	w("| write-microflows | **Read first** - Microflow syntax, common mistakes |\n")
	w("| write-oql-queries | OQL query syntax for VIEW entities |\n")
	w("| create-page | Page and widget syntax |\n")
	w("| fragments | Reusable widget group syntax |\n")
	w("\n")

	w("### Patterns\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| patterns-crud | Create/Read/Update/Delete patterns |\n")
	w("| patterns-data-processing | Loops, aggregates, batch processing |\n")
	w("| validation-microflows | Validation feedback patterns |\n")
	w("\n")

	w("### Pages\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| overview-pages | List/grid overview page patterns |\n")
	w("| master-detail-pages | Master-detail layout patterns |\n")
	w("| alter-page | ALTER PAGE/SNIPPET in-place modifications |\n")
	w("| bulk-widget-updates | Bulk widget property updates across pages |\n")
	w("\n")

	w("### Integration\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| database-connections | External database connections (PostgreSQL, Oracle) |\n")
	w("| rest-client | REST API consumption |\n")
	w("| java-actions | Custom Java actions |\n")
	w("| odata-data-sharing | OData services and external entities |\n")
	w("\n")

	w("### Operations\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| manage-security | Security roles, GRANT/REVOKE, access control |\n")
	w("| manage-navigation | Navigation profiles, menus, home/login pages |\n")
	w("| organize-project | Folders, MOVE command, project structure |\n")
	w("| project-settings | Project configuration (model, runtime, language) |\n")
	w("| business-events | Business event services |\n")
	w("\n")

	w("### Infrastructure\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| docker-workflow | Docker build and deployment |\n")
	w("| run-app | Running the Mendix app locally |\n")
	w("| runtime-admin-api | M2EE admin API |\n")
	w("| system-module | System module entities reference |\n")
	w("| verify-with-oql | OQL verification queries |\n")
	w("| demo-data | **Read first for data work** - Demo data insertion |\n")
	w("\n")

	w("### Testing & Quality\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| test-app | Playwright UI tests + DB assertions |\n")
	w("| test-microflows | Microflow unit testing (.test.mdl files) |\n")
	w("| write-lint-rules | Custom Starlark lint rule authoring |\n")
	w("| assess-quality | **Full project quality assessment** against best practices |\n")
	w("\n")

	w("### Domain Model\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| generate-domain-model | Full domain model generation |\n")
	w("\n")

	w("### Migration\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| assess-migration | Migration assessment and planning |\n")
	w("| migrate-k2-nintex | K2/Nintex workflow migration |\n")
	w("| migrate-outsystems | OutSystems migration |\n")
	w("| migrate-oracle-forms | Oracle Forms migration |\n")
	w("| graph-studio-app | Reverse-engineer/graph Studio Pro app |\n")
	w("\n")

	w("### Debugging & Preflight\n\n")
	w("| Skill | Purpose |\n")
	w("|-------|--------|\n")
	w("| debug-bson | BSON serialization debugging |\n")
	w("| check-syntax | Pre-flight validation checklist |\n")
	w("\n")

	// ── MDL Syntax Quick Reference ──────────────────────────────────
	w("## MDL Syntax Quick Reference\n\n")

	w("### Entity Generalization (EXTENDS)\n\n")
	w("**CRITICAL: EXTENDS goes BEFORE the opening parenthesis, not after!**\n\n")
	w(bt3 + "sql\nCREATE PERSISTENT ENTITY Module.ProductPhoto EXTENDS System.Image (\n  PhotoCaption: String(200)\n);\n" + bt3 + "\n\n")

	w("### Microflows - Supported Statements\n\n")
	w("| Statement | Syntax |\n")
	w("|-----------|--------|\n")
	w("| Variable declaration | " + bt + "DECLARE $Var Type = value;" + bt + " |\n")
	w("| Entity declaration | " + bt + "DECLARE $Entity Module.Entity;" + bt + " |\n")
	w("| List declaration | " + bt + "DECLARE $List List of Module.Entity = empty;" + bt + " |\n")
	w("| Assignment | " + bt + "SET $Var = expression;" + bt + " |\n")
	w("| Create object | " + bt + "$Var = CREATE Module.Entity (Attr = value);" + bt + " |\n")
	w("| Change object | " + bt + "CHANGE $Entity (Attr = value);" + bt + " |\n")
	w("| Commit | " + bt + "COMMIT $Entity [WITH EVENTS] [REFRESH];" + bt + " |\n")
	w("| Delete | " + bt + "DELETE $Entity;" + bt + " |\n")
	w("| Rollback | " + bt + "ROLLBACK $Entity [REFRESH];" + bt + " |\n")
	w("| Retrieve | " + bt + "RETRIEVE $Var FROM Module.Entity [WHERE condition];" + bt + " |\n")
	w("| Call microflow | " + bt + "$Result = CALL MICROFLOW Module.Name (Param = $value);" + bt + " |\n")
	w("| Call nanoflow | " + bt + "$Result = CALL NANOFLOW Module.Name (Param = $value);" + bt + " |\n")
	w("| Call Java action | " + bt + "$Result = CALL JAVA ACTION Module.Name (Param = value);" + bt + " |\n")
	w("| Show page | " + bt + "SHOW PAGE Module.PageName ($Param = $value);" + bt + " |\n")
	w("| Close page | " + bt + "CLOSE PAGE;" + bt + " |\n")
	w("| Validation | " + bt + "VALIDATION FEEDBACK $Entity/Attribute MESSAGE 'message';" + bt + " |\n")
	w("| Log | " + bt + "LOG INFO|WARNING|ERROR [NODE 'name'] 'message';" + bt + " |\n")
	w("| Annotation | " + bt + "@annotation 'text'" + bt + " (before activity) |\n")
	w("| Position | " + bt + "@position(x, y)" + bt + " (before activity) |\n")
	w("| Error handling | " + bt + "... ON ERROR CONTINUE|ROLLBACK|{ handler };" + bt + " |\n")
	w("| IF | " + bt + "IF condition THEN ... [ELSE ...] END IF;" + bt + " |\n")
	w("| LOOP | " + bt + "LOOP $Item IN $List BEGIN ... END LOOP;" + bt + " |\n")
	w("| WHILE | " + bt + "WHILE condition BEGIN ... END WHILE;" + bt + " |\n")
	w("| Return | " + bt + "RETURN $value;" + bt + " |\n")
	w("\n")

	w("### Microflows - NOT Supported (Will Cause Parse Errors)\n\n")
	w("| Unsupported | Use Instead |\n")
	w("|-------------|-------------|\n")
	w("| " + bt + "CASE ... WHEN ... END CASE" + bt + " | Nested " + bt + "IF ... ELSE ... END IF" + bt + " |\n")
	w("| " + bt + "TRY ... CATCH" + bt + " | " + bt + "ON ERROR { ... }" + bt + " blocks |\n")
	w("\n")
	w("**Notes:**\n")
	w("- " + bt + "RETRIEVE ... LIMIT n" + bt + " IS supported. " + bt + "LIMIT 1" + bt + " returns a single entity.\n")
	w("- " + bt + "ROLLBACK $Entity [REFRESH];" + bt + " IS supported. Rolls back uncommitted changes.\n\n")

	w("### Pages Syntax Summary\n\n")
	w("| Element | Syntax | Example |\n")
	w("|---------|--------|--------|\n")
	w("| Page properties | " + bt + "(Key: value, ...)" + bt + " | " + bt + "(Title: 'Edit', Layout: Atlas_Core.Atlas_Default)" + bt + " |\n")
	w("| Widget name | Required after type | " + bt + "TEXTBOX txtName (...)" + bt + " |\n")
	w("| Attribute binding | " + bt + "Attribute: AttrName" + bt + " | " + bt + "TEXTBOX txt (Label: 'Name', Attribute: Name)" + bt + " |\n")
	w("| Microflow action | " + bt + "Action: MICROFLOW Name(Param: val)" + bt + " | " + bt + "Action: MICROFLOW Mod.ACT_Process(Order: $Order)" + bt + " |\n")
	w("| Database source | " + bt + "DataSource: DATABASE Entity" + bt + " | " + bt + "DATAGRID dg (DataSource: DATABASE Mod.Entity)" + bt + " |\n")
	w("| Selection source | " + bt + "DataSource: SELECTION widget" + bt + " | " + bt + "DATAVIEW dv (DataSource: SELECTION galleryList)" + bt + " |\n")
	w("\n")

	w("**Supported Widgets:** LAYOUTGRID, ROW, COLUMN, CONTAINER, TEXTBOX, TEXTAREA, CHECKBOX, RADIOBUTTONS, DATEPICKER, COMBOBOX, DYNAMICTEXT, DATAGRID, GALLERY, LISTVIEW, IMAGE, STATICIMAGE, DYNAMICIMAGE, ACTIONBUTTON, LINKBUTTON, DATAVIEW, HEADER, FOOTER, CONTROLBAR, SNIPPETCALL, NAVIGATIONLIST, CUSTOMCONTAINER.\n\n")

	// ALTER PAGE summary
	w("### ALTER PAGE / ALTER SNIPPET\n\n")
	w("Modify existing pages in-place without full " + bt + "CREATE OR REPLACE" + bt + ":\n\n")
	w("| Operation | Syntax |\n")
	w("|-----------|--------|\n")
	w("| Set property | " + bt + "SET Caption = 'New' ON widgetName" + bt + " |\n")
	w("| Set multiple | " + bt + "SET (Caption = 'Save', ButtonStyle = Success) ON btn" + bt + " |\n")
	w("| Page-level set | " + bt + "SET Title = 'New Title'" + bt + " (no ON clause) |\n")
	w("| Insert after | " + bt + "INSERT AFTER widgetName { widgets }" + bt + " |\n")
	w("| Insert before | " + bt + "INSERT BEFORE widgetName { widgets }" + bt + " |\n")
	w("| Drop widgets | " + bt + "DROP WIDGET name1, name2" + bt + " |\n")
	w("| Replace widget | " + bt + "REPLACE widgetName WITH { widgets }" + bt + " |\n")
	w("\n")

	// Reserved words
	w("### Quoted Identifiers\n\n")
	w("**Always quote all identifiers** (entity names, attribute names, parameter names) with double quotes. This eliminates all reserved keyword conflicts and is always safe — quotes are stripped automatically.\n\n")
	w(bt3 + "sql\nCREATE PERSISTENT ENTITY Module.\"Customer\" (\n  \"Name\": String(200),\n  \"Status\": String(50),\n  \"Create\": DateTime\n);\n" + bt3 + "\n\n")

	// ── MDL Script Files ────────────────────────────────────────────
	w("## MDL Script Files\n\n")
	w("Store MDL scripts in the " + bt + "mdlsource/" + bt + " directory:\n\n")
	w(bt3 + "\nmdlsource/\n")
	w("\u251c\u2500\u2500 domain-model.mdl      # Entity definitions\n")
	w("\u251c\u2500\u2500 microflows.mdl        # Business logic\n")
	w("\u2514\u2500\u2500 setup.mdl             # Initial setup script\n")
	w(bt3 + "\n\n")
	w("Execute a script:\n\n")
	w(bt3 + "sql\nEXECUTE SCRIPT 'mdlsource/domain-model.mdl';\n" + bt3 + "\n\n")

	// ── Example: Entity ─────────────────────────────────────────────
	w("## Example: Create an Entity\n\n")
	w(bt3 + "sql\n")
	w("/**\n * Customer entity\n *\n * Stores customer information.\n */\n")
	w("@Position(100, 100)\n")
	w("CREATE PERSISTENT ENTITY Sales.Customer (\n")
	w("  /** Customer name */\n")
	w("  Name: String(200) NOT NULL ERROR 'Name is required',\n")
	w("  /** Email address */\n")
	w("  Email: String(200) UNIQUE ERROR 'Email must be unique',\n")
	w("  /** Phone number */\n")
	w("  Phone: String(50),\n")
	w("  /** Active status */\n")
	w("  IsActive: Boolean DEFAULT true\n")
	w(");\n")
	w(bt3 + "\n\n")

	// ── Example: Microflow ──────────────────────────────────────────
	w("## Example: Create a Microflow\n\n")
	w(bt3 + "sql\n")
	w("/**\n * Validates a customer before saving\n *\n * @param $Customer The customer to validate\n * @returns Boolean indicating validity\n */\n")
	w("CREATE MICROFLOW Sales.VAL_Customer (\n")
	w("  $Customer: Sales.Customer\n)\n")
	w("RETURNS Boolean AS $IsValid\n")
	w("BEGIN\n")
	w("  DECLARE $IsValid Boolean = true;\n\n")
	w("  IF trim($Customer/Name) = '' THEN\n")
	w("    SET $IsValid = false;\n")
	w("    VALIDATION FEEDBACK $Customer/Name MESSAGE 'Name is required';\n")
	w("  END IF;\n\n")
	w("  RETURN $IsValid;\n")
	w("END;\n/\n")
	w(bt3 + "\n\n")

	w("## MDL Reference\n\n")
	w("For detailed MDL syntax, see the skill files in " + bt + ".ai-context/skills/" + bt + ".\n")

	return sb.String()
}

// installVSCodeExtension extracts the embedded .vsix and installs it into VS Code.
func installVSCodeExtension(projectDir string) {
	// Skip if no embedded vsix data
	if len(vsixData) == 0 {
		return
	}

	// Write .vsix to the project directory
	vsixPath := filepath.Join(projectDir, ".claude", "vscode-mdl.vsix")
	if err := os.WriteFile(vsixPath, vsixData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not write VS Code extension: %v\n", err)
		return
	}

	// Try to find the VS Code CLI
	codeCLI := findCodeCLI()
	if codeCLI == "" {
		fmt.Printf("  Extracted VS Code extension to .claude/vscode-mdl.vsix\n")
		fmt.Printf("  Install manually: code --install-extension %s\n", vsixPath)
		return
	}

	// Install the extension
	cmd := exec.Command(codeCLI, "--install-extension", vsixPath, "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: VS Code extension install failed: %v\n", err)
		if len(output) > 0 {
			fmt.Fprintf(os.Stderr, "  %s\n", strings.TrimSpace(string(output)))
		}
		fmt.Printf("  Install manually: %s --install-extension %s\n", codeCLI, vsixPath)
		return
	}
	fmt.Println("  Installed VS Code MDL extension")

	// Clean up the extracted .vsix
	os.Remove(vsixPath)
}

// findCodeCLI looks for the VS Code CLI executable.
func findCodeCLI() string {
	for _, name := range []string{"code", "code-insiders"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags for tool selection
	initCmd.Flags().StringSliceVar(&initTools, "tool", []string{}, "AI tool(s) to configure (claude, opencode, cursor, continue, windsurf, aider)")
	initCmd.Flags().BoolVar(&initAllTools, "all-tools", false, "Initialize for all supported AI tools")
	initCmd.Flags().BoolVar(&initListTools, "list-tools", false, "List supported AI tools and exit")
}
