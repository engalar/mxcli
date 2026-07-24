// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuild_FrontendStep_SkippedWhenNoRollupConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if RollupConfigExists(dir) {
		t.Error("should not detect React client when rollup.config.mjs absent")
	}
}

func TestBuild_FrontendStep_DetectedWhenRollupConfigPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	webDir := filepath.Join(dir, "web")
	os.MkdirAll(webDir, 0755)
	os.WriteFile(filepath.Join(webDir, "rollup.config.mjs"), []byte("export default {}"), 0644)

	if !RollupConfigExists(dir) {
		t.Error("should detect React client when rollup.config.mjs present")
	}
}

func TestWriteDeployConfigJSON_CreatesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "model")
	os.MkdirAll(modelDir, 0755)

	if err := writeDeployConfigJSON(dir); err != nil {
		t.Fatalf("writeDeployConfigJSON: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(modelDir, "config.json"))
	if err != nil {
		t.Fatalf("config.json not created: %v", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	if _, ok := cfg["Constants"]; !ok {
		t.Error("config.json missing 'Constants' key")
	}
	if _, ok := cfg["Configuration"]; !ok {
		t.Error("config.json missing 'Configuration' key")
	}
}

func TestWriteDeployConfigJSON_SkipsIfExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "model")
	os.MkdirAll(modelDir, 0755)

	existing := `{"Configuration":{},"Constants":{},"AdminPassword":""}`
	os.WriteFile(filepath.Join(modelDir, "config.json"), []byte(existing), 0644)

	if err := writeDeployConfigJSON(dir); err != nil {
		t.Fatalf("writeDeployConfigJSON: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(modelDir, "config.json"))
	if string(data) != existing {
		t.Error("writeDeployConfigJSON should not overwrite existing config.json")
	}
}
