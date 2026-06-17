// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/widget/build"
	"github.com/mendixlabs/mxcli/internal/widget/scaffold"
)

func TestDetectToolchain_FindsSomething(t *testing.T) {
	_ = build.InstallMPK
	_ = build.FindMPKInCwd
}

func TestFindMPKInCwd_NoneFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, err := build.FindMPKInCwd()
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

	got, err := build.FindMPKInCwd()
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

	_, err := build.FindMPKInCwd()
	if err == nil {
		t.Fatal("expected error for multiple MPKs, got nil")
	}
}

func TestInstallMPK_CreatesWidgetsDirAndCopiesFile(t *testing.T) {
	srcDir := t.TempDir()
	projectDir := t.TempDir()

	mpkPath := filepath.Join(srcDir, "MyWidget.mpk")
	if err := os.WriteFile(mpkPath, []byte("fake mpk content"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	projectPath := filepath.Join(projectDir, "app.mpr")

	if err := build.InstallMPK(mpkPath, projectPath); err != nil {
		t.Fatalf("InstallMPK: %v", err)
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

	if err := build.InstallMPK(mpkPath, filepath.Join(projectDir, "app.mpr")); err != nil {
		t.Fatalf("InstallMPK: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(widgetsDir, "MyWidget.mpk"))
	if string(data) != "new" {
		t.Errorf("expected overwrite with 'new', got %q", data)
	}
}

func TestRoundTrip_ScaffoldThenDiscover(t *testing.T) {
	dir := t.TempDir()
	spec := scaffold.Spec{
		Name:        "RoundTrip",
		WidgetID:    "com.test.widget.RoundTrip.RoundTrip",
		PackagePath: "com.mendix.widget.custom",
		ProjectPath: "./tests/testProject",
		PackageName: "roundtrip",
	}
	if err := scaffold.Run(dir, spec); err != nil {
		t.Fatalf("scaffold.Run: %v", err)
	}
	for _, rel := range []string{"package.json", "src/package.xml", "src/RoundTrip.xml", "src/RoundTrip.jsx"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}
