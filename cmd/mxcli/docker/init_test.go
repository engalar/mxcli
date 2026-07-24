// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_CreatesDockerDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	var buf bytes.Buffer
	opts := InitOptions{
		ProjectPath: mprPath,
		Stdout:      &buf,
	}

	if err := Init(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dockerDir := filepath.Join(dir, ".docker")

	// docker-compose.yml should exist
	composePath := filepath.Join(dockerDir, "docker-compose.yml")
	if !fileExists(composePath) {
		t.Error("docker-compose.yml not created")
	}
	content, _ := os.ReadFile(composePath)
	if !strings.Contains(string(content), "postgres") {
		t.Error("docker-compose.yml should contain postgres service")
	}

	// .env.example should exist
	envExamplePath := filepath.Join(dockerDir, ".env.example")
	if !fileExists(envExamplePath) {
		t.Error(".env.example not created")
	}

	// .env should exist (copied from .env.example)
	envPath := filepath.Join(dockerDir, ".env")
	if !fileExists(envPath) {
		t.Error(".env not created")
	}
}

func TestInit_CustomOutputDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	customDir := filepath.Join(dir, "custom-docker")

	var buf bytes.Buffer
	opts := InitOptions{
		ProjectPath: mprPath,
		OutputDir:   customDir,
		Stdout:      &buf,
	}

	if err := Init(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fileExists(filepath.Join(customDir, "docker-compose.yml")) {
		t.Error("docker-compose.yml not created in custom dir")
	}
}

func TestInit_SkipsExistingEnv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	dockerDir := filepath.Join(dir, ".docker")
	os.MkdirAll(dockerDir, 0755)

	// Create existing .env with custom content
	envPath := filepath.Join(dockerDir, ".env")
	os.WriteFile(envPath, []byte("CUSTOM=true\n"), 0644)

	var buf bytes.Buffer
	opts := InitOptions{
		ProjectPath: mprPath,
		Stdout:      &buf,
	}

	if err := Init(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// .env should preserve custom content
	content, _ := os.ReadFile(envPath)
	if !strings.Contains(string(content), "CUSTOM=true") {
		t.Error(".env should preserve existing content without --force")
	}
}

func TestInit_ForceOverwritesExistingFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	dockerDir := filepath.Join(dir, ".docker")
	os.MkdirAll(dockerDir, 0755)

	// Create existing files with custom content
	os.WriteFile(filepath.Join(dockerDir, "docker-compose.yml"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(dockerDir, ".env"), []byte("CUSTOM=true\n"), 0644)

	var buf bytes.Buffer
	opts := InitOptions{
		ProjectPath: mprPath,
		Force:       true,
		Stdout:      &buf,
	}

	if err := Init(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// docker-compose.yml should be overwritten
	content, _ := os.ReadFile(filepath.Join(dockerDir, "docker-compose.yml"))
	if string(content) == "old" {
		t.Error("docker-compose.yml should be overwritten with --force")
	}

	// .env should be overwritten
	envContent, _ := os.ReadFile(filepath.Join(dockerDir, ".env"))
	if strings.Contains(string(envContent), "CUSTOM=true") {
		t.Error(".env should be overwritten with --force")
	}
}

func TestInit_SkipsExistingComposeWithoutForce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	dockerDir := filepath.Join(dir, ".docker")
	os.MkdirAll(dockerDir, 0755)
	os.WriteFile(filepath.Join(dockerDir, "docker-compose.yml"), []byte("custom"), 0644)

	var buf bytes.Buffer
	opts := InitOptions{
		ProjectPath: mprPath,
		Stdout:      &buf,
	}

	if err := Init(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should preserve existing compose file
	content, _ := os.ReadFile(filepath.Join(dockerDir, "docker-compose.yml"))
	if string(content) != "custom" {
		t.Error("docker-compose.yml should be preserved without --force")
	}

	if !strings.Contains(buf.String(), "Skipped") {
		t.Error("output should mention skipped files")
	}
}

func TestInit_PortOffset(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	var buf bytes.Buffer
	opts := InitOptions{
		ProjectPath: mprPath,
		PortOffset:  2,
		Stdout:      &buf,
	}

	if err := Init(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envPath := filepath.Join(dir, ".docker", ".env")
	content, _ := os.ReadFile(envPath)
	env := string(content)

	if !strings.Contains(env, "APP_PORT=8082") {
		t.Errorf("expected APP_PORT=8082, got: %s", env)
	}
	if !strings.Contains(env, "ADMIN_PORT=8092") {
		t.Errorf("expected ADMIN_PORT=8092, got: %s", env)
	}
	if !strings.Contains(env, "DB_PORT=5434") {
		t.Errorf("expected DB_PORT=5434, got: %s", env)
	}
	if !strings.Contains(env, "http://localhost:8082") {
		t.Errorf("expected TEST_BASE_URL with port 8082, got: %s", env)
	}
	if !strings.Contains(env, "localhost:5434") {
		t.Errorf("expected TEST_DB_URL with port 5434, got: %s", env)
	}
	if !strings.Contains(buf.String(), "Applied port offset 2") {
		t.Error("output should mention applied port offset")
	}
}

func TestInit_DetectsExistingBuildDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "app.mpr")
	os.WriteFile(mprPath, []byte(""), 0644)

	// Create build directory with Dockerfile
	buildDir := filepath.Join(dir, ".docker", "build")
	os.MkdirAll(buildDir, 0755)
	os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte("FROM test"), 0644)

	var buf bytes.Buffer
	opts := InitOptions{
		ProjectPath: mprPath,
		Stdout:      &buf,
	}

	if err := Init(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "Found existing build") {
		t.Error("should detect existing build directory")
	}
}
