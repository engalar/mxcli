// SPDX-License-Identifier: Apache-2.0

// skills_content.go - Embedded skill and command content for mxcli init
//
// Skills are synced from .claude/skills/mendix/
// Commands are synced from .claude/commands/mendix/
// Both use go:embed directive to embed at compile time.
//
// To update skills/commands:
//
//	make sync-all   # Sync both
//	make build      # Build (auto-syncs)
package main

import (
	"embed"
)

// Embed all skill files from the synced directory
//
//go:embed skills/*.md
var skillsFS embed.FS

// Embed all command files from the synced directory
//
//go:embed commands/*.md
var commandsFS embed.FS

// Embed all lint rule files from the synced directory
//
//go:embed lint-rules/*.star
var lintRulesFS embed.FS

// Embed example MDL files bundled with the binary
//
//go:embed examples/*.mdl
var examplesFS embed.FS

// settingsJSON is the Claude Code settings for mxcli permissions
const settingsJSON = `{
  "permissions": {
    "allow": [
      "Bash(./mxcli:*)",
      "Bash(./mxcli *)",
      "Bash(find *)",
      "Bash(grep *)",
      "Bash(grep -r*)",
      "Bash(ls *)",
      "Bash(ls)",
      "Bash(git status)",
      "Bash(git log *)",
      "Bash(git diff *)",
      "Bash(git diff)",
      "Bash(playwright-cli:*)",
      "Bash(playwright-cli *)"
    ]
  },
  "env": {
    "MXCLI_QUIET": "1"
  }
}
`
