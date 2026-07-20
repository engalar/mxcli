// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runInit is a test helper that invokes the cobra Run closure against dir.
// vsixData is set to nil to prevent installVSCodeExtension from invoking
// the external 'code' CLI or writing .vsix files on CI / dev machines.
func runInit(t *testing.T, dir string) {
	t.Helper()
	prevVsix := vsixData
	t.Cleanup(func() {
		vsixData = prevVsix
	})
	vsixData = nil
	_ = initCmd.RunE(initCmd, []string{dir})
}

// ── Unit tests: generateClaudeMD ─────────────────────────────────────────────

func TestGenerateClaudeMD_ContainsProjectName(t *testing.T) {
	got := generateClaudeMD("MyProject", "MyProject.mpr", true)
	if !strings.Contains(got, "MyProject") {
		t.Error("CLAUDE.md should contain the project name")
	}
}

func TestGenerateClaudeMD_ContainsMprPath(t *testing.T) {
	got := generateClaudeMD("MyProject", "MyProject.mpr", true)
	if !strings.Contains(got, "MyProject.mpr") {
		t.Error("CLAUDE.md should contain the .mpr file path")
	}
}

func TestGenerateClaudeMD_ContainsVersionCheck(t *testing.T) {
	got := generateClaudeMD("P", "p.mpr", true)
	if !strings.Contains(got, "SHOW FEATURES") {
		t.Error("CLAUDE.md should include SHOW FEATURES version check instruction")
	}
}

func TestGenerateClaudeMD_SkillsPointToClaudeSkillsDir(t *testing.T) {
	got := generateClaudeMD("P", "p.mpr", true)
	if strings.Contains(got, ".ai-context/skills/") {
		t.Error("CLAUDE.md should not reference .ai-context/skills/ paths; use .claude/skills/ instead")
	}
	if !strings.Contains(got, ".claude/skills/") {
		t.Error("CLAUDE.md should reference .claude/skills/ for skill paths")
	}
}

func TestGenerateClaudeMD_NoAiContextReferences(t *testing.T) {
	got := generateClaudeMD("P", "p.mpr", true)
	if strings.Contains(got, ".ai-context") {
		t.Errorf("CLAUDE.md should not reference .ai-context; found: %q", ".ai-context")
	}
}

func TestGenerateClaudeMD_Devcontainer_UsesLocalPrefix(t *testing.T) {
	got := generateClaudeMD("P", "p.mpr", true)
	if !strings.Contains(got, "./mxcli") {
		t.Error("devcontainer CLAUDE.md should use ./mxcli prefix")
	}
	if !strings.Contains(got, "root folder of this project") {
		t.Error("devcontainer CLAUDE.md should warn about local binary location")
	}
}

func TestGenerateClaudeMD_Global_UsesGlobalCommand(t *testing.T) {
	got := generateClaudeMD("P", "p.mpr", false)
	if strings.Contains(got, "./mxcli") {
		t.Error("global-install CLAUDE.md must not use ./mxcli prefix")
	}
	// mxcli (without ./) must still appear in command examples
	if !strings.Contains(got, "mxcli") {
		t.Error("global-install CLAUDE.md should still reference mxcli")
	}
}

// ── Unit tests: generateClaudeSettings ───────────────────────────────────────

func TestGenerateClaudeSettings_ContainsMxcliPermission(t *testing.T) {
	got := generateClaudeSettings("P", "p.mpr")
	if !strings.Contains(got, "./mxcli") {
		t.Error("settings.json should allow ./mxcli commands")
	}
}

func TestGenerateClaudeSettings_ContainsExplorePermissions(t *testing.T) {
	got := generateClaudeSettings("P", "p.mpr")
	for _, perm := range []string{"find", "grep", "git status", "ls"} {
		if !strings.Contains(got, perm) {
			t.Errorf("settings.json should allow %q command", perm)
		}
	}
}

func TestGenerateClaudeSettings_ValidJSON(t *testing.T) {
	got := generateClaudeSettings("P", "p.mpr")
	// Basic JSON structure checks
	if !strings.Contains(got, `"permissions"`) {
		t.Error("settings.json should have 'permissions' key")
	}
	if !strings.Contains(got, `"allow"`) {
		t.Error("settings.json should have 'allow' key")
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

func TestInit_CreatesClaudeFiles(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	if !fileExists(filepath.Join(dir, "CLAUDE.md")) {
		t.Error("CLAUDE.md should be created")
	}
	if !fileExists(filepath.Join(dir, ".claude", "settings.json")) {
		t.Error(".claude/settings.json should be created")
	}
	if n := countFilesInDir(filepath.Join(dir, ".claude", "commands")); n == 0 {
		t.Error(".claude/commands/ should contain command files")
	}
	if n := countFilesInDir(filepath.Join(dir, ".claude", "lint-rules")); n == 0 {
		t.Error(".claude/lint-rules/ should contain lint rule files")
	}
}

func TestInit_CreatesSkillsInClaudeDir(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	skillsDir := filepath.Join(dir, ".claude", "skills")
	if n := countSubDirs(skillsDir); n == 0 {
		t.Error(".claude/skills/ should contain skill subdirectories")
	}
}

func TestInit_EachSkillHasSKILLMd(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	skillsDir := filepath.Join(dir, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("could not read .claude/skills/: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no skill directories found")
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		if !fileExists(skillMD) {
			t.Errorf("skill %q: SKILL.md missing", e.Name())
		}
	}
}

func TestInit_SkillFilesHaveContent(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	skillsDir := filepath.Join(dir, ".claude", "skills")
	entries, _ := os.ReadDir(skillsDir)
	checked := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(skillMD)
		if err != nil {
			t.Errorf("skill %q: SKILL.md missing or unreadable: %v", e.Name(), err)
			continue
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			t.Errorf("skill %q: SKILL.md is empty", e.Name())
		}
		checked++
	}
	if checked == 0 {
		t.Error("no skills were checked — init may have failed")
	}
}

func TestInit_READMENotWrittenAsSkill(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	readmeSkillDir := filepath.Join(dir, ".claude", "skills", "README")
	if fileExists(readmeSkillDir) {
		t.Error("README should not be written as a skill directory")
	}
}

func TestInit_CreatesDevcontainer(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	if !fileExists(filepath.Join(dir, ".devcontainer", "devcontainer.json")) {
		t.Error(".devcontainer/devcontainer.json should be created")
	}
	if !fileExists(filepath.Join(dir, ".devcontainer", "Dockerfile")) {
		t.Error(".devcontainer/Dockerfile should be created")
	}
}

func TestInit_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	if !fileExists(filepath.Join(dir, ".gitignore")) {
		t.Error(".gitignore should be created")
	}
}

func TestInit_NoOtherToolDirsCreated(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	for _, unexpected := range []string{
		".opencode", "opencode.json",
		".cursorrules", ".windsurfrules",
		".continue", ".aider.conf.yml",
		".vibe", ".github/copilot-instructions.md",
		"AGENTS.md",
		".ai-context",
	} {
		if fileExists(filepath.Join(dir, unexpected)) {
			t.Errorf("%s should NOT be created (other tool support removed)", unexpected)
		}
	}
}

func TestInit_CommandsAreMarkdown(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	commandsDir := filepath.Join(dir, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("could not read .claude/commands/: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			t.Errorf(".claude/commands/%s: expected .md extension", e.Name())
		}
	}
}

func TestInit_LintRulesAreStarlark(t *testing.T) {
	dir := t.TempDir()
	runInit(t, dir)

	lintDir := filepath.Join(dir, ".claude", "lint-rules")
	entries, err := os.ReadDir(lintDir)
	if err != nil {
		t.Fatalf("could not read .claude/lint-rules/: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".star") {
			t.Errorf(".claude/lint-rules/%s: expected .star extension", e.Name())
		}
	}
}
