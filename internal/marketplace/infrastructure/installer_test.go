// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstaller_RunsMxModuleImport(t *testing.T) {
	binDir := t.TempDir()
	mxPath := filepath.Join(binDir, "mx")
	script := "#!/bin/sh\necho 'import ok'"
	if err := os.WriteFile(mxPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	inst := NewInstaller(mxPath)
	mpkPath := filepath.Join(t.TempDir(), "test.mpk")
	if err := os.WriteFile(mpkPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	err := inst.InstallModule(context.Background(), mpkPath, "/tmp/test.mpr")
	if err != nil {
		t.Fatal(err)
	}
}

func TestInstaller_NotFoundReturnsError(t *testing.T) {
	inst := NewInstaller("/nonexistent/mx")
	err := inst.InstallModule(context.Background(), "test.mpk", "/tmp/test.mpr")
	if err == nil {
		t.Fatal("expected error for missing mx binary")
	}
}
