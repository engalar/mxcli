// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/widget/scaffold"
)

func TestRenderersListIsPopulated(t *testing.T) {
	if len(renderers) == 0 {
		t.Fatal("renderers list is empty")
	}
}

func TestScaffoldThenDiscover(t *testing.T) {
	dir := t.TempDir()
	spec := scaffold.Spec{
		Name:        "RoundTrip",
		PackageName: "roundtrip",
		WidgetID:    "com.test.widget.RoundTrip.RoundTrip",
		PackagePath: "com.mendix.widget.custom",
		ProjectPath: "./tests/testProject",
	}
	if err := scaffold.Run(dir, spec, renderers); err != nil {
		t.Fatalf("scaffold.Run: %v", err)
	}
	for _, rel := range []string{"package.json", "src/package.xml", "src/RoundTrip.xml", "src/RoundTrip.jsx"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}
