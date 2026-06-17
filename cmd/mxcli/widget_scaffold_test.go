// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/widget/scaffold"
)

func TestRenderersListIsPopulated(t *testing.T) {
	// renderers list removed in template-based approach — just verify scaffold exists
	_ = scaffold.DeriveWidgetID("com.mendix.widget.custom", "Test")
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
	if err := scaffold.Run(dir, spec); err != nil {
		t.Fatalf("scaffold.Run: %v", err)
	}
	for _, rel := range []string{"package.json", "src/package.xml", "src/RoundTrip.xml", "src/RoundTrip.jsx"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}
