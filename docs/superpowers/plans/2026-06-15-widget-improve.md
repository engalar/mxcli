# mxcli widget — Improvement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve `mxcli widget` with early `--id` validation, `--description` flag, `.gitignore` + `README.md` scaffold, a `widget install` command, `widget build --install -p` flag, and a corrected `widget list` message.

**Architecture:** Changes touch three files: `widget_scaffold.go` (generators + scaffold logic), `widget_build.go` (install helpers), `cmd_widget.go` (command registration + list fix). A new test file `widget_build_test.go` covers the install helpers. All new behaviour is covered by unit tests before implementation.

**Tech Stack:** Go, `github.com/spf13/cobra`, stdlib `os` / `archive/zip` / `filepath`

---

## File Map

| File | Role |
|------|------|
| `cmd/mxcli/widget_scaffold.go` | Tasks 1–3: ID validation, description, gitignore/README generators |
| `cmd/mxcli/widget_scaffold_test.go` | Tests for Tasks 1–3 |
| `cmd/mxcli/widget_build.go` | Task 4–5: `installMPK`, `findMPKInCwd`, build `--install` flag |
| `cmd/mxcli/widget_build_test.go` | Tests for Task 4 (new file) |
| `cmd/mxcli/cmd_widget.go` | Tasks 4–6: `widget install` registration, build flags, list message |

---

## Task 1: Extract `validateWidgetIDFormat` and call early in `widget new`

**Files:**
- Modify: `cmd/mxcli/widget_build.go` (extract helper from `validateWidgetInfo`)
- Modify: `cmd/mxcli/widget_scaffold.go` (call helper before writing files)
- Modify: `cmd/mxcli/widget_scaffold_test.go` (add test)

- [ ] **Step 1.1 — Write the failing test**

Add to `cmd/mxcli/widget_scaffold_test.go`:

```go
func TestValidateWidgetIDFormat_TooFewSegments(t *testing.T) {
	err := validateWidgetIDFormat("helpdesk.MyWidget")
	if err == nil {
		t.Fatal("expected error for 2-segment ID, got nil")
	}
	if !strings.Contains(err.Error(), "4 dot-separated") {
		t.Errorf("expected '4 dot-separated' in error, got: %v", err)
	}
}

func TestValidateWidgetIDFormat_ValidFourSegments(t *testing.T) {
	if err := validateWidgetIDFormat("com.acme.widget.MyWidget"); err != nil {
		t.Fatalf("unexpected error for valid 4-segment ID: %v", err)
	}
}

func TestValidateWidgetIDFormat_ValidFiveSegments(t *testing.T) {
	if err := validateWidgetIDFormat("com.mendix.widget.custom.MyWidget.MyWidget"); err != nil {
		t.Fatalf("unexpected error for valid 5-segment ID: %v", err)
	}
}
```

- [ ] **Step 1.2 — Run tests to confirm they fail**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
CGO_ENABLED=0 go test ./cmd/mxcli/ -run TestValidateWidgetIDFormat -v
```

Expected: `FAIL — validateWidgetIDFormat undefined`

- [ ] **Step 1.3 — Extract `validateWidgetIDFormat` in `widget_build.go`**

In `cmd/mxcli/widget_build.go`, add this function before `validateWidgetInfo`:

```go
// validateWidgetIDFormat checks that id has at least 4 dot-separated segments.
func validateWidgetIDFormat(id string) error {
	if len(strings.Split(id, ".")) < 4 {
		return fmt.Errorf("widget ID must have at least 4 dot-separated segments (e.g. com.acme.widget.MyName), got %q", id)
	}
	return nil
}
```

Update `validateWidgetInfo` to delegate the ID check:

```go
func validateWidgetInfo(info widgetInfo) error {
	if err := validateWidgetIDFormat(info.WidgetID); err != nil {
		return fmt.Errorf("widget %q: %w", info.Name, err)
	}
	if info.DisplayName == "" {
		return fmt.Errorf("widget %q: <name> element is empty in XML", info.Name)
	}
	return nil
}
```

- [ ] **Step 1.4 — Call `validateWidgetIDFormat` early in `runWidgetNew`**

In `cmd/mxcli/widget_scaffold.go`, in `runWidgetNew`, add the validation immediately after the `widgetID` is determined (before `scaffoldWidget`):

```go
	widgetID, _ := cmd.Flags().GetString("id")
	if widgetID == "" {
		widgetID = deriveWidgetID(name)
	} else {
		if err := validateWidgetIDFormat(widgetID); err != nil {
			return err
		}
	}
```

- [ ] **Step 1.5 — Run tests to confirm they pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -run TestValidateWidgetIDFormat -v
```

Expected: all three tests PASS

- [ ] **Step 1.6 — Commit**

```bash
git add cmd/mxcli/widget_build.go cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go
git commit -m "feat(widget): validate --id format at scaffold time, not just build time"
```

---

## Task 2: `--description` flag + XML update

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go` (`generateWidgetXML`, `scaffoldWidget`, `runWidgetNew`, `runWidgetAddWidget`)
- Modify: `cmd/mxcli/cmd_widget.go` (register flag on both commands)
- Modify: `cmd/mxcli/widget_scaffold_test.go` (add test)

- [ ] **Step 2.1 — Write the failing test**

Add to `cmd/mxcli/widget_scaffold_test.go`:

```go
func TestGenerateWidgetXML_WithDescription(t *testing.T) {
	xml := generateWidgetXML("MySlider", "com.acme.widget.MySlider.MySlider", "A great slider widget", false, nil)
	if !strings.Contains(xml, "<description>A great slider widget</description>") {
		t.Errorf("expected description in XML, got:\n%s", xml)
	}
}

func TestGenerateWidgetXML_EmptyDescription(t *testing.T) {
	xml := generateWidgetXML("MySlider", "com.acme.widget.MySlider.MySlider", "", false, nil)
	if !strings.Contains(xml, "<description></description>") {
		t.Errorf("expected empty description in XML, got:\n%s", xml)
	}
}
```

- [ ] **Step 2.2 — Run tests to confirm they fail**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -run TestGenerateWidgetXML_With -v
```

Expected: FAIL — `generateWidgetXML` takes 4 args, not 5

- [ ] **Step 2.3 — Update `generateWidgetXML` signature and body**

In `cmd/mxcli/widget_scaffold.go`, change the signature from:

```go
func generateWidgetXML(name, widgetID string, offline bool, props []PropertySpec) string {
```

to:

```go
func generateWidgetXML(name, widgetID, description string, offline bool, props []PropertySpec) string {
```

Replace the description line inside the function body:

```go
// old:
b.WriteString("  <description></description>\n")

// new:
b.WriteString(fmt.Sprintf("  <description>%s</description>\n", xmlEscape(description)))
```

Add the `xmlEscape` helper just above `generateWidgetXML`:

```go
// xmlEscape escapes the five XML special characters for use in element content.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
```

- [ ] **Step 2.4 — Update `scaffoldWidget` to accept and thread `description`**

Change signature:

```go
// old:
func scaffoldWidget(dir, name, widgetID string, offline bool, props []PropertySpec) error {

// new:
func scaffoldWidget(dir, name, widgetID, description string, offline bool, props []PropertySpec) error {
```

Update the call to `generateWidgetXML` inside `scaffoldWidget`:

```go
// old:
name + ".xml": []byte(generateWidgetXML(name, widgetID, offline, props)),

// new:
name + ".xml": []byte(generateWidgetXML(name, widgetID, description, offline, props)),
```

- [ ] **Step 2.5 — Update callers of `scaffoldWidget`**

In `runWidgetNew`, read the description flag and pass it:

```go
	description, _ := cmd.Flags().GetString("description")
	// ...
	if err := scaffoldWidget(outDir, name, widgetID, description, offline, props); err != nil {
```

In `runWidgetAddWidget`, read the description flag and pass it:

```go
	description, _ := cmd.Flags().GetString("description")
	// ...
	if err := scaffoldWidget(dir, name, widgetID, description, offline, props); err != nil {
```

- [ ] **Step 2.6 — Register `--description` flag on both commands in `cmd_widget.go`**

In the `init()` function, after the existing flag registrations for `widgetNewCmd` and `widgetAddWidgetCmd`:

```go
widgetNewCmd.Flags().String("description", "", "Widget description (written into XML and README)")
widgetAddWidgetCmd.Flags().String("description", "", "Widget description (written into XML)")
```

- [ ] **Step 2.7 — Fix the existing `TestGenerateWidgetXML` test** which calls `generateWidgetXML` with the old 4-arg signature

In `widget_scaffold_test.go`, update every call to `generateWidgetXML` to pass `""` as the new third argument:

```go
// In TestGenerateWidgetXML:
xml := generateWidgetXML("MySlider", "com.acme.widget.MySlider.MySlider", "", false, props)

// In TestGenerateWidgetXML_Offline:
xml := generateWidgetXML("Foo", "com.a.b.c.Foo.Foo", "", true, nil)
```

- [ ] **Step 2.8 — Run all widget_scaffold tests**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -run TestGenerateWidgetXML -v
```

Expected: all 4 tests PASS

- [ ] **Step 2.9 — Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/cmd_widget.go cmd/mxcli/widget_scaffold_test.go
git commit -m "feat(widget): add --description flag; write into XML <description> element"
```

---

## Task 3: `generateGitignore`, `generateReadme`, and `scaffoldRootFiles`

**Files:**
- Modify: `cmd/mxcli/widget_scaffold.go` (three new functions + call from `runWidgetNew`)
- Modify: `cmd/mxcli/widget_scaffold_test.go` (add tests)

- [ ] **Step 3.1 — Write the failing tests**

Add to `cmd/mxcli/widget_scaffold_test.go`:

```go
func TestGenerateGitignore(t *testing.T) {
	gi := generateGitignore()
	for _, want := range []string{"node_modules/", "dist/", "*.mpk"} {
		if !strings.Contains(gi, want) {
			t.Errorf("generateGitignore: missing %q\ngot:\n%s", want, gi)
		}
	}
}

func TestGenerateReadme_WithPropsAndDescription(t *testing.T) {
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
		{Key: "label", XMLType: "string"},
		{Key: "slot",  XMLType: "widgets"},
	}
	md := generateReadme("MySlider", "A slider for numbers", props)
	checks := []string{
		"# MySlider",
		"A slider for numbers",
		"mxcli widget build",
		"--install -p",
		"## Properties",
		"| value | attribute (Decimal) | Yes |",
		"| label | string | Yes |",
		"| slot  | widgets | No |",
	}
	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("generateReadme: missing %q\ngot:\n%s", want, md)
		}
	}
}

func TestGenerateReadme_NoPropsNoDescription(t *testing.T) {
	md := generateReadme("Foo", "", nil)
	if strings.Contains(md, "## Properties") {
		t.Errorf("generateReadme: Properties table must be absent when no props\ngot:\n%s", md)
	}
	if !strings.Contains(md, "# Foo") {
		t.Errorf("generateReadme: missing # Foo\ngot:\n%s", md)
	}
}

func TestScaffoldRootFiles_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	props := []PropertySpec{{Key: "val", XMLType: "attribute", Subtype: "String"}}
	if err := scaffoldRootFiles(dir, "MyWidget", "My description", props); err != nil {
		t.Fatalf("scaffoldRootFiles: %v", err)
	}
	for _, rel := range []string{".gitignore", "README.md"} {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("%s must not be empty", rel)
		}
	}
}
```

- [ ] **Step 3.2 — Run tests to confirm they fail**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -run "TestGenerateGitignore|TestGenerateReadme|TestScaffoldRootFiles" -v
```

Expected: FAIL — functions undefined

- [ ] **Step 3.3 — Implement `generateGitignore` in `widget_scaffold.go`**

Add after `generatePackageJSON`:

```go
// generateGitignore returns the content of the .gitignore written by `widget new`.
func generateGitignore() string {
	return "node_modules/\ndist/\n*.mpk\n"
}
```

- [ ] **Step 3.4 — Implement `generateReadme` in `widget_scaffold.go`**

Add after `generateGitignore`:

```go
// generateReadme renders README.md for a widget project.
// The Properties table is omitted when props is empty.
func generateReadme(name, description string, props []PropertySpec) string {
	var b strings.Builder
	b.WriteString("# " + name + "\n\n")
	if description != "" {
		b.WriteString(description + "\n\n")
	}
	b.WriteString("## Build\n\n```bash\nmxcli widget build\n```\n\n")
	b.WriteString("## Install into a Mendix project\n\n```bash\nmxcli widget build --install -p /path/to/app.mpr\n```\n\n")
	if len(props) == 0 {
		return b.String()
	}
	b.WriteString("## Properties\n\n")
	b.WriteString("| Property | Type | Required |\n")
	b.WriteString("|----------|------|----------|\n")
	for _, p := range props {
		typeStr := p.XMLType
		if p.Subtype != "" {
			typeStr += " (" + p.Subtype + ")"
		}
		req := "Yes"
		if p.XMLType == "widgets" {
			req = "No"
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", p.Key, typeStr, req))
	}
	b.WriteString("\n")
	return b.String()
}
```

- [ ] **Step 3.5 — Implement `scaffoldRootFiles` in `widget_scaffold.go`**

Add after `generateReadme`:

```go
// scaffoldRootFiles writes .gitignore and README.md at the widget project root.
// Called only by runWidgetNew (not add-widget, which operates on an existing project).
func scaffoldRootFiles(dir, name, description string, props []PropertySpec) error {
	files := map[string][]byte{
		".gitignore": []byte(generateGitignore()),
		"README.md":  []byte(generateReadme(name, description, props)),
	}
	for filename, content := range files {
		dest := filepath.Join(dir, filename)
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}
	return nil
}
```

- [ ] **Step 3.6 — Call `scaffoldRootFiles` from `runWidgetNew`**

In `runWidgetNew`, after the existing `scaffoldWidget` call and before writing `package.json`:

```go
	if err := scaffoldWidget(outDir, name, widgetID, description, offline, props); err != nil {
		return fmt.Errorf("scaffolding widget: %w", err)
	}
	if err := scaffoldRootFiles(outDir, name, description, props); err != nil {
		return fmt.Errorf("scaffolding root files: %w", err)
	}
```

- [ ] **Step 3.7 — Update `TestScaffoldWidget_CreatesExpectedFiles` to not expect `.gitignore`/`README.md` from `scaffoldWidget`**

The test calls `scaffoldWidget` directly (not `runWidgetNew`), so no changes are needed — `.gitignore` and `README.md` are only written by `scaffoldRootFiles`. Verify the test still lists only `src/` files.

- [ ] **Step 3.8 — Run all new and existing scaffold tests**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -run "TestGenerateGitignore|TestGenerateReadme|TestScaffoldRootFiles|TestScaffoldWidget|TestScaffoldSingleWidget" -v
```

Expected: all PASS

- [ ] **Step 3.9 — Commit**

```bash
git add cmd/mxcli/widget_scaffold.go cmd/mxcli/widget_scaffold_test.go
git commit -m "feat(widget): scaffold .gitignore and README.md in widget new"
```

---

## Task 4: `installMPK`, `findMPKInCwd`, and `widget install` command

**Files:**
- Modify: `cmd/mxcli/widget_build.go` (two new functions)
- Create: `cmd/mxcli/widget_build_test.go`
- Modify: `cmd/mxcli/cmd_widget.go` (`widget install` command + registration)

- [ ] **Step 4.1 — Create `cmd/mxcli/widget_build_test.go` with failing tests**

```go
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindMPKInCwd_NoneFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, err := findMPKInCwd()
	if err == nil {
		t.Fatal("expected error when no MPK found, got nil")
	}
	if !strings.Contains(err.Error(), "widget build") {
		t.Errorf("error should mention 'widget build', got: %v", err)
	}
}

func TestFindMPKInCwd_OneFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	mpkPath := filepath.Join(dir, "MyWidget.mpk")
	os.WriteFile(mpkPath, []byte("fake"), 0644)

	got, err := findMPKInCwd()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "MyWidget.mpk" {
		t.Errorf("got %q, want %q", got, "MyWidget.mpk")
	}
}

func TestFindMPKInCwd_MultipleFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.WriteFile(filepath.Join(dir, "A.mpk"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(dir, "B.mpk"), []byte("fake"), 0644)

	_, err := findMPKInCwd()
	if err == nil {
		t.Fatal("expected error for multiple MPKs, got nil")
	}
	if !strings.Contains(err.Error(), "--mpk") {
		t.Errorf("error should mention '--mpk', got: %v", err)
	}
}

func TestInstallMPK_CreatesWidgetsDirAndCopiesFile(t *testing.T) {
	srcDir := t.TempDir()
	projectDir := t.TempDir()

	// Create a fake MPK
	mpkPath := filepath.Join(srcDir, "MyWidget.mpk")
	if err := os.WriteFile(mpkPath, []byte("fake mpk content"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Fake project path (widgets/ dir does not exist yet)
	projectPath := filepath.Join(projectDir, "app.mpr")

	if err := installMPK(mpkPath, projectPath); err != nil {
		t.Fatalf("installMPK: %v", err)
	}

	dst := filepath.Join(projectDir, "widgets", "MyWidget.mpk")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("expected installed file at %s: %v", dst, err)
	}
	if string(data) != "fake mpk content" {
		t.Errorf("installed file content mismatch: %q", data)
	}
}

func TestInstallMPK_OverwritesExistingFile(t *testing.T) {
	srcDir := t.TempDir()
	projectDir := t.TempDir()

	widgetsDir := filepath.Join(projectDir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	os.WriteFile(filepath.Join(widgetsDir, "MyWidget.mpk"), []byte("old"), 0644)

	mpkPath := filepath.Join(srcDir, "MyWidget.mpk")
	os.WriteFile(mpkPath, []byte("new"), 0644)

	if err := installMPK(mpkPath, filepath.Join(projectDir, "app.mpr")); err != nil {
		t.Fatalf("installMPK: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(widgetsDir, "MyWidget.mpk"))
	if string(data) != "new" {
		t.Errorf("expected overwrite with 'new', got %q", data)
	}
}
```

- [ ] **Step 4.2 — Run tests to confirm they fail**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -run "TestFindMPKInCwd|TestInstallMPK" -v
```

Expected: FAIL — `findMPKInCwd` and `installMPK` undefined

- [ ] **Step 4.3 — Implement `findMPKInCwd` and `installMPK` in `widget_build.go`**

Add after `verifyMPK`:

```go
// findMPKInCwd globs *.mpk in the current working directory.
// Returns an error if 0 or 2+ files are found.
func findMPKInCwd() (string, error) {
	matches, err := filepath.Glob("*.mpk")
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no .mpk file found — run 'mxcli widget build' first")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple .mpk files found (%s) — specify one with --mpk", strings.Join(matches, ", "))
	}
}

// installMPK copies mpkPath into <projectDir>/widgets/, creating the directory if needed.
func installMPK(mpkPath, projectPath string) error {
	widgetsDir := filepath.Join(filepath.Dir(projectPath), "widgets")
	if err := os.MkdirAll(widgetsDir, 0755); err != nil {
		return fmt.Errorf("creating widgets/: %w", err)
	}
	dst := filepath.Join(widgetsDir, filepath.Base(mpkPath))
	if err := copyFile(mpkPath, dst); err != nil {
		return fmt.Errorf("copying MPK: %w", err)
	}
	fmt.Printf("Installed → %s\n", dst)
	return nil
}
```

- [ ] **Step 4.4 — Run tests to confirm they pass**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -run "TestFindMPKInCwd|TestInstallMPK" -v
```

Expected: all 5 tests PASS

- [ ] **Step 4.5 — Add `widget install` command to `cmd_widget.go`**

Add the command variable after `widgetBuildCmd`:

```go
var widgetInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a built .mpk into a Mendix project's widgets/ folder",
	Long: `Copy a widget .mpk file into <project>/widgets/, creating the directory if needed.

Without --mpk, auto-detects a single *.mpk in the current directory (the output of 'mxcli widget build').

Examples:
  mxcli widget install -p /path/to/app.mpr
  mxcli widget install --mpk TicketStatusBadge.mpk -p /path/to/app.mpr`,
	RunE: runWidgetInstall,
}
```

Add `runWidgetInstall` in `cmd_widget.go` (or `widget_build.go` — keep it in `cmd_widget.go` for command layer separation):

```go
func runWidgetInstall(cmd *cobra.Command, args []string) error {
	mpkFlag, _ := cmd.Flags().GetString("mpk")
	projectPath, _ := cmd.Flags().GetString("project")

	mpkPath := mpkFlag
	if mpkPath == "" {
		var err error
		mpkPath, err = findMPKInCwd()
		if err != nil {
			return err
		}
	}
	return installMPK(mpkPath, projectPath)
}
```

Register in `init()`:

```go
widgetInstallCmd.Flags().String("mpk", "", "Path to .mpk file (default: auto-detect *.mpk in current directory)")
widgetInstallCmd.Flags().StringP("project", "p", "", "Path to Mendix project (.mpr file)")
widgetInstallCmd.MarkFlagRequired("project")

widgetCmd.AddCommand(widgetInstallCmd)
```

- [ ] **Step 4.6 — Build and smoke-test**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build -o bin/mxcli ./cmd/mxcli
./bin/mxcli widget install --help
```

Expected: help text shows `--mpk` and `-p` flags

- [ ] **Step 4.7 — Commit**

```bash
git add cmd/mxcli/widget_build.go cmd/mxcli/widget_build_test.go cmd/mxcli/cmd_widget.go
git commit -m "feat(widget): add widget install command and installMPK/findMPKInCwd helpers"
```

---

## Task 5: `widget build --install -p` flag

**Files:**
- Modify: `cmd/mxcli/widget_build.go` (`runWidgetBuild` — call `installMPK` when `--install`)
- Modify: `cmd/mxcli/cmd_widget.go` (register flags)

- [ ] **Step 5.1 — Register `--install` and `-p` flags on `widgetBuildCmd` in `cmd_widget.go`**

In `init()`, after `widgetBuildCmd.Flags().String("dir", ...)`:

```go
widgetBuildCmd.Flags().Bool("install", false, "Install the built MPK into the Mendix project's widgets/ folder")
widgetBuildCmd.Flags().StringP("project", "p", "", "Path to Mendix project (.mpr file) — required with --install")
```

- [ ] **Step 5.2 — Call `installMPK` after a successful build in `runWidgetBuild`**

In `cmd/mxcli/widget_build.go`, at the end of `runWidgetBuild`, replace:

```go
	fmt.Printf("Built %s (%d widget(s), %d KB)\n", filepath.Base(mpkPath), len(infos), size)
	return nil
```

with:

```go
	fmt.Printf("Built %s (%d widget(s), %d KB)\n", filepath.Base(mpkPath), len(infos), size)

	install, _ := cmd.Flags().GetBool("install")
	if install {
		projectPath, _ := cmd.Flags().GetString("project")
		if projectPath == "" {
			return fmt.Errorf("--install requires -p <project.mpr>")
		}
		if err := installMPK(mpkPath, projectPath); err != nil {
			return fmt.Errorf("install: %w", err)
		}
	}
	return nil
```

- [ ] **Step 5.3 — Build and smoke-test**

```bash
go build -o bin/mxcli ./cmd/mxcli
./bin/mxcli widget build --help
```

Expected: `--install` and `-p` appear in the flags section

- [ ] **Step 5.4 — Run full test suite to check no regressions**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ -v 2>&1 | tail -20
```

Expected: all PASS

- [ ] **Step 5.5 — Commit**

```bash
git add cmd/mxcli/widget_build.go cmd/mxcli/cmd_widget.go
git commit -m "feat(widget): add --install -p flag to widget build for post-build auto-install"
```

---

## Task 6: Fix misleading `widget list` message

**Files:**
- Modify: `cmd/mxcli/cmd_widget.go` (two string changes in `runWidgetList`)

- [ ] **Step 6.1 — Apply the two string changes in `runWidgetList`**

In `runWidgetList`, find and replace:

```go
// old title:
fmt.Fprintf(out, "\n--- Discovered in widgets/*.mpk (not yet extracted) ---\n\n")

// new title:
fmt.Fprintf(out, "\n--- Auto-discovered from widgets/*.mpk ---\n\n")
```

```go
// old footer (single line):
fmt.Fprintf(out, "\nRun 'mxcli widget extract --mpk widgets/<file>.mpk' to generate .def.json with property names\n")

// new footer (two lines):
fmt.Fprintf(out, "\nMPK widgets are auto-discovered — no extraction needed.\n")
fmt.Fprintf(out, "To override property mappings: mxcli widget extract --mpk widgets/<file>.mpk\n")
```

- [ ] **Step 6.2 — Build and verify output**

```bash
go build -o bin/mxcli ./cmd/mxcli
./bin/mxcli widget list --help
```

Expected: command works (no compile errors)

- [ ] **Step 6.3 — Run full test suite**

```bash
CGO_ENABLED=0 go test ./cmd/mxcli/ 2>&1 | tail -5
```

Expected: `ok  	github.com/mendixlabs/mxcli/cmd/mxcli`

- [ ] **Step 6.4 — Commit**

```bash
git add cmd/mxcli/cmd_widget.go
git commit -m "fix(widget): fix widget list message — remove misleading extract suggestion"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Task |
|-----------------|------|
| Early `--id` validation | Task 1 |
| `--description` flag → XML + README | Tasks 2, 3 |
| `.gitignore` generation | Task 3 |
| `README.md` generation | Task 3 |
| `widget install` command | Task 4 |
| `widget build --install -p` | Task 5 |
| `widget list` message fix | Task 6 |

All requirements covered. ✓

**Placeholder scan:** No TBD/TODO/placeholder patterns. All code blocks are complete. ✓

**Type consistency:**
- `generateWidgetXML(name, widgetID, description string, offline bool, props []PropertySpec)` — used consistently in Tasks 2, 3
- `scaffoldWidget(dir, name, widgetID, description string, offline bool, props []PropertySpec)` — updated in Task 2, called with description in Tasks 2, 3
- `scaffoldRootFiles(dir, name, description string, props []PropertySpec)` — defined and called in Task 3
- `installMPK(mpkPath, projectPath string) error` — defined in Task 4, called in Tasks 4, 5
- `findMPKInCwd() (string, error)` — defined in Task 4, called in Tasks 4
- `validateWidgetIDFormat(id string) error` — defined in Task 1, used in Tasks 1 ✓
