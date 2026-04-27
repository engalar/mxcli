// SPDX-License-Identifier: Apache-2.0

package mpk

import (
	"os"
	"path/filepath"
	"testing"
)

// findTestMPK finds a ComboBox .mpk file in the test projects directory.
func findTestMPK(t *testing.T) string {
	t.Helper()
	// Try multiple known locations
	candidates := []string{
		filepath.Join("..", "..", "..", "mx-test-projects", "template-app-116", "widgets", "com.mendix.widget.web.Combobox.mpk"),
		filepath.Join("..", "..", "..", "mx-test-projects", "LatoProductInventory", "widgets", "com.mendix.widget.web.Combobox.mpk"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	t.Skip("No test .mpk file found")
	return ""
}

func findTestProjectDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "..", "mx-test-projects", "template-app-116"),
		filepath.Join("..", "..", "..", "mx-test-projects", "LatoProductInventory"),
	}
	for _, c := range candidates {
		widgetsDir := filepath.Join(c, "widgets")
		if _, err := os.Stat(widgetsDir); err == nil {
			return c
		}
	}
	t.Skip("No test project directory found")
	return ""
}

func TestParseMPK(t *testing.T) {
	mpkPath := findTestMPK(t)
	ClearCache()

	def, err := ParseMPK(mpkPath)
	if err != nil {
		t.Fatalf("ParseMPK failed: %v", err)
	}

	if def.ID != "com.mendix.widget.web.combobox.Combobox" {
		t.Errorf("unexpected widget ID: %s", def.ID)
	}

	if def.Name != "Combo box" {
		t.Errorf("unexpected widget name: %s", def.Name)
	}

	if def.Version == "" {
		t.Error("version should not be empty")
	}

	if len(def.Properties) == 0 {
		t.Error("expected at least one property")
	}

	// Check that we found some known properties
	keys := def.PropertyKeys()
	expectedKeys := []string{"source", "attributeEnumeration", "clearable", "filterType"}
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("expected property key %q not found", k)
		}
	}

	// Check system properties
	if len(def.SystemProps) == 0 {
		t.Error("expected at least one system property")
	}
	sysKeys := def.SystemPropertyKeys()
	if !sysKeys["Label"] {
		t.Error("expected system property 'Label'")
	}

	// Check property types
	sourceProp := def.FindProperty("source")
	if sourceProp == nil {
		t.Fatal("source property not found")
	}
	if sourceProp.Type != "enumeration" {
		t.Errorf("expected source type 'enumeration', got %q", sourceProp.Type)
	}
	if sourceProp.DefaultValue != "context" {
		t.Errorf("expected source default 'context', got %q", sourceProp.DefaultValue)
	}

	// Check category tracking
	if sourceProp.Category == "" {
		t.Error("expected source to have a category")
	}
}

func TestParseMPK_Cached(t *testing.T) {
	mpkPath := findTestMPK(t)
	ClearCache()

	def1, err := ParseMPK(mpkPath)
	if err != nil {
		t.Fatalf("first ParseMPK failed: %v", err)
	}

	def2, err := ParseMPK(mpkPath)
	if err != nil {
		t.Fatalf("second ParseMPK failed: %v", err)
	}

	// Should return same pointer from cache
	if def1 != def2 {
		t.Error("expected cached result to return same pointer")
	}
}

func TestFindMPK(t *testing.T) {
	projectDir := findTestProjectDir(t)
	ClearCache()

	// Find ComboBox
	mpkPath, err := FindMPK(projectDir, "com.mendix.widget.web.combobox.Combobox")
	if err != nil {
		t.Fatalf("FindMPK failed: %v", err)
	}
	if mpkPath == "" {
		t.Fatal("expected to find ComboBox .mpk")
	}
	if !filepath.IsAbs(mpkPath) && !fileExists(mpkPath) {
		t.Errorf("mpk path should exist: %s", mpkPath)
	}

	// Find non-existent widget
	mpkPath2, err := FindMPK(projectDir, "com.example.nonexistent.Widget")
	if err != nil {
		t.Fatalf("FindMPK for nonexistent should not error: %v", err)
	}
	if mpkPath2 != "" {
		t.Errorf("expected empty path for nonexistent widget, got: %s", mpkPath2)
	}
}

func TestFindMPK_Cached(t *testing.T) {
	projectDir := findTestProjectDir(t)
	ClearCache()

	// First call scans directory
	path1, _ := FindMPK(projectDir, "com.mendix.widget.web.combobox.Combobox")
	// Second call should use cache
	path2, _ := FindMPK(projectDir, "com.mendix.widget.web.combobox.Combobox")

	if path1 != path2 {
		t.Error("expected same path from cache")
	}
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"attribute", "attribute"},
		{"Attribute", "attribute"},
		{"ATTRIBUTE", "attribute"},
		{"textTemplate", "textTemplate"},
		{"TextTemplate", "textTemplate"},
		{"datasource", "datasource"},
		{"unknownType", "unknownType"},
	}

	for _, tt := range tests {
		result := NormalizeType(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeType(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
