// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDockerDir_FindsDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	dockerDir := filepath.Join(dir, ".docker")
	os.MkdirAll(dockerDir, 0755)
	os.WriteFile(filepath.Join(dockerDir, "docker-compose.yml"), []byte("services:"), 0644)

	opts := RuntimeOptions{ProjectPath: mprPath}
	result, err := resolveDockerDir(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != dockerDir {
		t.Errorf("expected %s, got %s", dockerDir, result)
	}
}

func TestResolveDockerDir_ErrorsWhenNoCompose(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	opts := RuntimeOptions{ProjectPath: mprPath}
	_, err := resolveDockerDir(opts)
	if err == nil {
		t.Error("expected error when docker-compose.yml is missing")
	}
}

func TestResolveDockerDir_UsesExplicitDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	customDir := filepath.Join(dir, "custom")
	os.MkdirAll(customDir, 0755)
	os.WriteFile(filepath.Join(customDir, "docker-compose.yml"), []byte("services:"), 0644)

	opts := RuntimeOptions{
		ProjectPath: filepath.Join(dir, "app.mpr"),
		DockerDir:   customDir,
	}
	result, err := resolveDockerDir(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != customDir {
		t.Errorf("expected %s, got %s", customDir, result)
	}
}

func TestResolveDockerDir_ErrorsWhenExplicitDirMissesCompose(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	customDir := filepath.Join(dir, "empty")
	os.MkdirAll(customDir, 0755)

	opts := RuntimeOptions{
		ProjectPath: filepath.Join(dir, "app.mpr"),
		DockerDir:   customDir,
	}
	_, err := resolveDockerDir(opts)
	if err == nil {
		t.Error("expected error when docker-compose.yml is missing in explicit dir")
	}
}
