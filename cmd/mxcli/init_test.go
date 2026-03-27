// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runInit is a test helper that sets initTools to the given list, invokes the
// cobra Run closure against dir, then restores the original value.  The
// function panics if initListTools or initAllTools are left non-default from a
// previous test.
func runInit(t *testing.T, tools []string, dir string) {
	t.Helper()
	prev := initTools
	t.Cleanup(func() { initTools = prev })
	initTools = tools
	initCmd.Run(initCmd, []string{dir})
}

// ── Unit tests: extractSkillDescription ──────────────────────────────────────

func TestExtractSkillDescription_FirstHeading(t *testing.T) {
	content := []byte("# My Skill\n\nSome content here.\n")
	got := extractSkillDescription(content)
	if got != "My Skill" {
		t.Errorf("got %q, want %q", got, "My Skill")
	}
}

func TestExtractSkillDescription_SkillPrefix(t *testing.T) {
	content := []byte("# Skill: Write Microflows\n\nContent.\n")
	got := extractSkillDescription(content)
	if got != "Write Microflows" {
		t.Errorf("got %q, want %q", got, "Write Microflows")
	}
}

func TestExtractSkillDescription_Fallback(t *testing.T) {
	content := []byte("No heading here.\nJust plain text.\n")
	got := extractSkillDescription(content)
	if got != "MDL skill" {
		t.Errorf("got %q, want %q", got, "MDL skill")
	}
}

func TestExtractSkillDescription_IgnoresSubheadings(t *testing.T) {
	content := []byte("## Section\n\n# Top Level\n")
	got := extractSkillDescription(content)
	if got != "Top Level" {
		t.Errorf("got %q, want %q", got, "Top Level")
	}
}

func TestExtractSkillDescription_EmptyContent(t *testing.T) {
	got := extractSkillDescription([]byte{})
	if got != "MDL skill" {
		t.Errorf("got %q, want %q", got, "MDL skill")
	}
}

// ── Unit tests: wrapSkillContent ─────────────────────────────────────────────

func TestWrapSkillContent_FrontmatterPresent(t *testing.T) {
	content := []byte("# My Skill\n\nbody text\n")
	wrapped := wrapSkillContent("my-skill", content)
	text := string(wrapped)

	if !strings.HasPrefix(text, "---\n") {
		t.Errorf("wrapped content should start with '---\\n', got: %q", text[:min(40, len(text))])
	}
	if !strings.Contains(text, "name: my-skill\n") {
		t.Error("frontmatter should contain 'name: my-skill'")
	}
	if !strings.Contains(text, "description: My Skill\n") {
		t.Error("frontmatter should contain 'description: My Skill'")
	}
	if !strings.Contains(text, "compatibility: opencode\n") {
		t.Error("frontmatter should contain 'compatibility: opencode'")
	}
}

func TestWrapSkillContent_OriginalContentPreserved(t *testing.T) {
	body := "# My Skill\n\nThis is the skill body.\n"
	wrapped := wrapSkillContent("my-skill", []byte(body))
	text := string(wrapped)

	if !strings.Contains(text, body) {
		t.Errorf("wrapped content should contain original body; got:\n%s", text)
	}
}

func TestWrapSkillContent_FrontmatterClosedBeforeBody(t *testing.T) {
	content := []byte("# Title\n\nbody\n")
	wrapped := string(wrapSkillContent("s", content))

	closingIdx := strings.Index(wrapped, "\n---\n")
	bodyIdx := strings.Index(wrapped, "# Title")

	if closingIdx == -1 {
		t.Fatal("closing '---' delimiter not found")
	}
	if bodyIdx < closingIdx {
		t.Errorf("body appears before closing delimiter: closing=%d body=%d", closingIdx, bodyIdx)
	}
}

// ── Integration tests ─────────────────────────────────────────────────────────

// fileExists is a small test helper.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// countFilesInDir returns the number of regular files directly inside dir.
func countFilesInDir(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return n
}

// countSubDirs returns the number of subdirectories directly inside dir.
func countSubDirs(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	return n
}

func TestInitOpenCode_CreatesRequiredStructure(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"opencode"}, dir)

	// opencode.json config
	if !fileExists(filepath.Join(dir, "opencode.json")) {
		t.Error("opencode.json should be created")
	}

	// .opencode/commands/ with at least one command
	commandsDir := filepath.Join(dir, ".opencode", "commands")
	if n := countFilesInDir(commandsDir); n == 0 {
		t.Error(".opencode/commands/ should contain command files")
	}

	// .opencode/skills/ with at least one skill subdirectory
	skillsDir := filepath.Join(dir, ".opencode", "skills")
	if n := countSubDirs(skillsDir); n == 0 {
		t.Error(".opencode/skills/ should contain skill subdirectories")
	}

	// Lint rules stay in .claude/lint-rules/ for mxcli lint compatibility
	lintDir := filepath.Join(dir, ".claude", "lint-rules")
	if n := countFilesInDir(lintDir); n == 0 {
		t.Error(".claude/lint-rules/ should contain lint rule files")
	}

	// Universal files
	if !fileExists(filepath.Join(dir, "AGENTS.md")) {
		t.Error("AGENTS.md should be created")
	}
	if n := countFilesInDir(filepath.Join(dir, ".ai-context", "skills")); n == 0 {
		t.Error(".ai-context/skills/ should contain skill files")
	}

	// Claude-specific files should NOT be present
	if fileExists(filepath.Join(dir, ".claude", "settings.json")) {
		t.Error(".claude/settings.json should NOT be created for opencode-only init")
	}
	if fileExists(filepath.Join(dir, "CLAUDE.md")) {
		t.Error("CLAUDE.md should NOT be created for opencode-only init")
	}
}

func TestInitOpenCode_EachSkillHasValidFrontmatter(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"opencode"}, dir)

	skillsDir := filepath.Join(dir, ".opencode", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("could not read .opencode/skills/: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no skill directories found — init may have failed")
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(skillMD)
		if err != nil {
			t.Errorf("skill %q: SKILL.md missing: %v", e.Name(), err)
			continue
		}
		text := string(data)
		if !strings.HasPrefix(text, "---\n") {
			t.Errorf("skill %q: SKILL.md should start with YAML frontmatter '---'", e.Name())
		}
		if !strings.Contains(text, "name: "+e.Name()) {
			t.Errorf("skill %q: SKILL.md frontmatter should contain 'name: %s'", e.Name(), e.Name())
		}
		if !strings.Contains(text, "compatibility: opencode") {
			t.Errorf("skill %q: SKILL.md should contain 'compatibility: opencode'", e.Name())
		}
	}
}

func TestInitOpenCode_READMENotWrittenAsSkill(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"opencode"}, dir)

	readmeSkillDir := filepath.Join(dir, ".opencode", "skills", "README")
	if fileExists(readmeSkillDir) {
		t.Error("README should not be written as an OpenCode skill directory")
	}
}

func TestInitOpenCode_LintRulesCreatedWithoutClaudeTool(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"opencode"}, dir) // no "claude"

	lintDir := filepath.Join(dir, ".claude", "lint-rules")
	if !fileExists(lintDir) {
		t.Error(".claude/lint-rules/ should exist even when only opencode is selected")
	}
	if n := countFilesInDir(lintDir); n == 0 {
		t.Error(".claude/lint-rules/ should contain .star lint rule files")
	}

	// Verify they are .star files
	entries, _ := os.ReadDir(lintDir)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".star") {
			t.Errorf("unexpected file in .claude/lint-rules/: %s (expected .star)", e.Name())
		}
	}
}

func TestInitOpenCode_CommandsAreMarkdown(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"opencode"}, dir)

	commandsDir := filepath.Join(dir, ".opencode", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("could not read .opencode/commands/: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			t.Errorf(".opencode/commands/%s: expected .md extension", e.Name())
		}
	}
}

func TestInitClaude_CreatesClaudeSpecificFiles(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"claude"}, dir)

	// Claude-specific
	if !fileExists(filepath.Join(dir, "CLAUDE.md")) {
		t.Error("CLAUDE.md should be created for claude tool")
	}
	if !fileExists(filepath.Join(dir, ".claude", "settings.json")) {
		t.Error(".claude/settings.json should be created for claude tool")
	}
	if n := countFilesInDir(filepath.Join(dir, ".claude", "commands")); n == 0 {
		t.Error(".claude/commands/ should contain command files")
	}
	if n := countFilesInDir(filepath.Join(dir, ".claude", "lint-rules")); n == 0 {
		t.Error(".claude/lint-rules/ should contain lint rule files")
	}

	// Universal files
	if !fileExists(filepath.Join(dir, "AGENTS.md")) {
		t.Error("AGENTS.md should be created")
	}
}

func TestInitClaude_NoOpenCodeDirCreated(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"claude"}, dir)

	if fileExists(filepath.Join(dir, ".opencode")) {
		t.Error(".opencode/ should NOT be created for claude-only init")
	}
	if fileExists(filepath.Join(dir, "opencode.json")) {
		t.Error("opencode.json should NOT be created for claude-only init")
	}
}

func TestInitBothTools_CreatesAllFiles(t *testing.T) {
	dir := t.TempDir()
	runInit(t, []string{"claude", "opencode"}, dir)

	// Claude files
	if !fileExists(filepath.Join(dir, "CLAUDE.md")) {
		t.Error("CLAUDE.md should exist when claude tool is selected")
	}
	if !fileExists(filepath.Join(dir, ".claude", "settings.json")) {
		t.Error(".claude/settings.json should exist when claude tool is selected")
	}

	// OpenCode files
	if !fileExists(filepath.Join(dir, "opencode.json")) {
		t.Error("opencode.json should exist when opencode tool is selected")
	}
	if n := countSubDirs(filepath.Join(dir, ".opencode", "skills")); n == 0 {
		t.Error(".opencode/skills/ should contain skill subdirectories")
	}

	// Lint rules should be present exactly once
	if n := countFilesInDir(filepath.Join(dir, ".claude", "lint-rules")); n == 0 {
		t.Error(".claude/lint-rules/ should contain lint rule files")
	}
}
