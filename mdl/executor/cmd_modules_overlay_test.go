package executor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/model"
)

// TestAlterModuleJarDep_OverlayRead_Bug reproduces the overlay-read bug:
// consecutive ALTER MODULE ADD JAR DEPENDENCY in a single session overwrite
// each other because GetModuleSettings reads via ListUnitsWithContainer
// which reads scriptInserts without checking scriptOverlay.
//
// When a unit is first created (AddInsert → scriptInserts) then updated
// (AddUpdate → SetScriptOverlay), subsequent ListUnitsByType returns the
// stale scriptInserts content instead of the fresh overlay content.
func TestAlterModuleJarDep_OverlayRead_Bug(t *testing.T) {
	srcDir := filepath.Join("..", "..", "testdata", "helpdesk-clean-11.12.1")
	tmpDir := t.TempDir()
	cpDir(t, srcDir, tmpDir)
	mprPath := filepath.Join(tmpDir, "minimal.mpr")

	// Open backend and create executor
	backend := mprbackend.New()
	if err := backend.Connect(mprPath); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { backend.Disconnect() })

	var buf bytes.Buffer
	exec := New(&buf)
	exec.SetBackend(backend)

	// Execute MDL: create module + 2 add jar deps (single session)
	mdl := `
create module JD;
alter module JD add jar dependency ( group='org.bouncycastle', artifact='bcprov-jdk18on', version='1.78.1', included=true );
alter module JD add jar dependency ( group='eu.agno3.jcifs', artifact='jcifs-ng', version='2.1.10', included=true );
`
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs[0])
	}
	if err := exec.ExecuteProgram(prog); err != nil {
		t.Fatalf("exec: %v", err)
	}

	// Verify directly through backend after script commit.
	// The JD module ID is unknown; iterate all modules to find it.
	modules, err := backend.ListModules()
	if err != nil {
		t.Fatalf("list modules: %v", err)
	}
	var jdModule *model.Module
	for _, m := range modules {
		if m.Name == "JD" {
			jdModule = m
			break
		}
	}
	if jdModule == nil {
		t.Fatal("JD module not found after exec")
	}

	ms, err := backend.GetModuleSettings(jdModule.ID)
	if err != nil {
		t.Fatalf("get module settings: %v", err)
	}
	if ms == nil {
		t.Fatal("module settings is nil")
	}

	if len(ms.JarDependencies) != 2 {
		t.Errorf("expected 2 jar dependencies, got %d (bug: overlay write not visible to ListUnitsWithContainer)", len(ms.JarDependencies))
		for _, d := range ms.JarDependencies {
			t.Logf("  dep: %s:%s", d.GroupID, d.ArtifactID)
		}
	}
	// Also verify by listing their coordinates
	var bcprov, jcifs bool
	for _, d := range ms.JarDependencies {
		if d.GroupID == "org.bouncycastle" && d.ArtifactID == "bcprov-jdk18on" {
			bcprov = true
		}
		if d.GroupID == "eu.agno3.jcifs" && d.ArtifactID == "jcifs-ng" {
			jcifs = true
		}
	}
	if !bcprov {
		t.Error("missing bcprov-jdk18on (first ADD was overwritten)")
	}
	if !jcifs {
		t.Error("missing jcifs-ng")
	}
}

func cpDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			cpDir(t, s, d)
		} else {
			data, err := os.ReadFile(s)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(d, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
}
