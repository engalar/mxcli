// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestReadAndRenderMxunit(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Forms$Page"},
		{Key: "Name", Value: "P_Test"},
		{Key: "Title", Value: "Test Page"},
	}
	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "test.mxunit")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ndsl, err := readAndRenderMxunit(tmp)
	if err != nil {
		t.Fatalf("readAndRenderMxunit: %v", err)
	}

	if !strings.Contains(ndsl, "Forms$Page") {
		t.Errorf("expected NDSL to contain 'Forms$Page', got:\n%s", ndsl)
	}
	if !strings.Contains(ndsl, `Name = "P_Test"`) {
		t.Errorf("expected NDSL to contain Name field, got:\n%s", ndsl)
	}
}

func TestBsonDiffOutput(t *testing.T) {
	mkUnit := func(t *testing.T, name, title string) string {
		t.Helper()
		doc := bson.D{
			{Key: "$Type", Value: "Forms$Page"},
			{Key: "Name", Value: name},
			{Key: "Title", Value: title},
		}
		data, _ := bson.Marshal(doc)
		tmp := filepath.Join(t.TempDir(), "f.mxunit")
		os.WriteFile(tmp, data, 0o644)
		return tmp
	}

	file1 := mkUnit(t, "P_Test", "Old Title")
	file2 := mkUnit(t, "P_Test", "New Title")

	out, changed, err := computeBsonDiff(file1, file2, "a.mxunit", "b.mxunit")
	if err != nil {
		t.Fatalf("computeBsonDiff: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if !strings.Contains(out, `-`) || !strings.Contains(out, `+`) {
		t.Errorf("expected unified diff markers, got:\n%s", out)
	}
	if !strings.Contains(out, "Old Title") {
		t.Errorf("expected removed line with 'Old Title', got:\n%s", out)
	}
	if !strings.Contains(out, "New Title") {
		t.Errorf("expected added line with 'New Title', got:\n%s", out)
	}
}

func TestBsonDiffIdentical(t *testing.T) {
	doc := bson.D{
		{Key: "$Type", Value: "Forms$Page"},
		{Key: "Name", Value: "P_Same"},
	}
	data, _ := bson.Marshal(doc)
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.mxunit")
	f2 := filepath.Join(dir, "b.mxunit")
	os.WriteFile(f1, data, 0o644)
	os.WriteFile(f2, data, 0o644)

	_, changed, err := computeBsonDiff(f1, f2, "a.mxunit", "b.mxunit")
	if err != nil {
		t.Fatalf("computeBsonDiff: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false for identical files")
	}
}
