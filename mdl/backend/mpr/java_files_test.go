// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeJavaFilesBackend creates a minimal MprBackend with a real project directory
// (no SQLite needed since Java file ops are pure filesystem).
func makeJavaFilesBackend(t *testing.T) (*MprBackend, string) {
	t.Helper()
	dir := t.TempDir()
	// Create a fake .mpr file so filepath.Dir(b.path) resolves to dir.
	mprPath := filepath.Join(dir, "test.mpr")
	if err := os.WriteFile(mprPath, []byte{}, 0644); err != nil {
		t.Fatalf("write fake mpr: %v", err)
	}
	b := &MprBackend{path: mprPath}
	return b, dir
}

func TestWriteJavaSourceFileViaPathGen(t *testing.T) {
	b, projectRoot := makeJavaFilesBackend(t)

	// nil params + nil return type exercises the empty-params + Boolean
	// default return type path (matches the legacy
	// TestWriteJavaSourceFileViaPath behaviour).
	if err := b.writeJavaSourceFileViaPathGen("MyModule", "MyAction", "return true;", nil, nil, nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(projectRoot, "javasource", "mymodule", "actions", "MyAction.java")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("expected Java source file to exist at %s", expected)
	}
}

func TestDeleteJavaSourceFileViaPath_NotExist(t *testing.T) {
	b, _ := makeJavaFilesBackend(t)

	// Deleting non-existent file should not error
	if err := b.deleteJavaSourceFileViaPath("MyModule", "NonExistent"); err != nil {
		t.Fatalf("unexpected error on missing file: %v", err)
	}
}

func TestDeleteJavaSourceFileViaPath_Exists(t *testing.T) {
	b, projectRoot := makeJavaFilesBackend(t)

	// Create the file first
	dir := filepath.Join(projectRoot, "javasource", "mymodule", "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(dir, "MyAction.java")
	if err := os.WriteFile(target, []byte("// test"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := b.deleteJavaSourceFileViaPath("MyModule", "MyAction"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
}

func TestRenameJavaSourceFileViaPath(t *testing.T) {
	b, projectRoot := makeJavaFilesBackend(t)

	dir := filepath.Join(projectRoot, "javasource", "mymodule", "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	oldPath := filepath.Join(dir, "OldAction.java")
	if err := os.WriteFile(oldPath, []byte("// old"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := b.renameJavaSourceFileViaPath("MyModule", "OldAction", "NewAction"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newPath := filepath.Join(dir, "NewAction.java")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Fatal("expected renamed file to exist")
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("expected old file to be gone")
	}
}

func TestReadJavaSourceFileViaPath(t *testing.T) {
	b, projectRoot := makeJavaFilesBackend(t)

	dir := filepath.Join(projectRoot, "javasource", "mymodule", "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "// java source content"
	if err := os.WriteFile(filepath.Join(dir, "MyAction.java"), []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := b.readJavaSourceFileViaPath("MyModule", "MyAction")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Fatalf("content = %q, want %q", got, content)
	}
}

// =============================================================================
// UpdateRawUnit — msdkWriter == nil must return an error (no b.writer fallback)
// =============================================================================

func TestUpdateRawUnit_ReturnsErrorWhenMsdkWriterNil(t *testing.T) {
	b := &MprBackend{} // msdkWriter is nil, writer is nil
	err := b.UpdateRawUnit("some-unit-id", []byte("data"))
	if err == nil {
		t.Fatal("expected error when msdkWriter is nil, got nil")
	}
}

func TestUpdateJavaSourceSections_CodeOnly(t *testing.T) {
	b, projectRoot := makeJavaFilesBackend(t)

	dir := filepath.Join(projectRoot, "javasource", "mymodule", "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	existing := "package mymodule.actions;\n\nimport com.mendix.systemwideinterfaces.core.IContext;\nimport java.util.List;\n\npublic class MyAction extends UserAction<java.lang.String> {\n\tpublic java.lang.String executeAction() throws Exception {\n\t\t// BEGIN USER CODE\n\t\treturn \"old\";\n\t\t// END USER CODE\n\t}\n\t// BEGIN EXTRA CODE\n\t// END EXTRA CODE\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "MyAction.java"), []byte(existing), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := b.updateJavaSourceSections("MyModule", "MyAction", "return \"new\";", nil, ""); err != nil {
		t.Fatalf("updateJavaSourceSections: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "MyAction.java"))
	content := string(got)
	if !strings.Contains(content, `return "new";`) {
		t.Error("expected new code in user code section")
	}
	if strings.Contains(content, `return "old";`) {
		t.Error("old code should be replaced")
	}
	if !strings.Contains(content, "import java.util.List;") {
		t.Error("existing import must be preserved")
	}
}

func TestUpdateJavaSourceSections_ImportsMerge(t *testing.T) {
	b, projectRoot := makeJavaFilesBackend(t)

	dir := filepath.Join(projectRoot, "javasource", "mymodule", "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	existing := "package mymodule.actions;\n\nimport com.mendix.systemwideinterfaces.core.IContext;\nimport java.util.List;\n\npublic class MyAction extends UserAction<java.lang.String> {\n\tpublic java.lang.String executeAction() throws Exception {\n\t\t// BEGIN USER CODE\n\t\t// END USER CODE\n\t}\n\t// BEGIN EXTRA CODE\n\t// END EXTRA CODE\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "MyAction.java"), []byte(existing), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	newImports := []string{
		"import java.util.List;",
		"import javax.crypto.Cipher;",
	}
	if err := b.updateJavaSourceSections("MyModule", "MyAction", "", newImports, ""); err != nil {
		t.Fatalf("updateJavaSourceSections: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "MyAction.java"))
	content := string(got)

	count := strings.Count(content, "import java.util.List;")
	if count != 1 {
		t.Errorf("import java.util.List; appears %d times, want 1", count)
	}
	if !strings.Contains(content, "import javax.crypto.Cipher;") {
		t.Error("new import must be added")
	}
}

func TestUpdateJavaSourceSections_ExtraReplace(t *testing.T) {
	b, projectRoot := makeJavaFilesBackend(t)

	dir := filepath.Join(projectRoot, "javasource", "mymodule", "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	existing := "package mymodule.actions;\nimport com.mendix.systemwideinterfaces.core.IContext;\npublic class MyAction extends UserAction<java.lang.String> {\n\tpublic java.lang.String executeAction() throws Exception {\n\t\t// BEGIN USER CODE\n\t\t// END USER CODE\n\t}\n\t// BEGIN EXTRA CODE\n\tprivate String OLD = \"old\";\n\t// END EXTRA CODE\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "MyAction.java"), []byte(existing), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := b.updateJavaSourceSections("MyModule", "MyAction", "", nil, "private String NEW = \"new\";"); err != nil {
		t.Fatalf("updateJavaSourceSections: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "MyAction.java"))
	content := string(got)
	if strings.Contains(content, "OLD") {
		t.Error("old extra code should be replaced")
	}
	if !strings.Contains(content, "NEW") {
		t.Error("new extra code must appear")
	}
}
