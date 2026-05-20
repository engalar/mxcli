// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/model"
)

func TestDocumentFilePath_NoFolder(t *testing.T) {
	got := documentFilePath("/out", "MyModule", "", "MyModule.Customer")
	want := filepath.Join("/out", "MyModule", "MyModule.Customer.mdl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDocumentFilePath_WithFolder(t *testing.T) {
	got := documentFilePath("/out", "MyModule", "Microflows/ACT", "MyModule.ACT_Foo")
	want := filepath.Join("/out", "MyModule", "Microflows", "ACT", "MyModule.ACT_Foo.mdl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestClassifyModules(t *testing.T) {
	mods := []*model.Module{
		{Name: "MyFirstModule", FromAppStore: false},
		{Name: "Administration", FromAppStore: true, AppStoreVersion: "3.4.0"},
		{Name: "AtlasCore", FromAppStore: true, AppStoreVersion: "4.0.0"},
	}
	regular, marketplace := classifyModules(mods)
	if len(regular) != 1 || regular[0].Name != "MyFirstModule" {
		t.Errorf("regular modules: got %v", regular)
	}
	if len(marketplace) != 2 {
		t.Errorf("marketplace modules: got %v", marketplace)
	}
}

func TestMarketplaceFileContent(t *testing.T) {
	mods := []*model.Module{
		{Name: "Administration", FromAppStore: true, AppStoreVersion: "3.4.0"},
		{Name: "AtlasCore", FromAppStore: true, AppStoreVersion: "4.0.0"},
	}
	got := marketplaceFileContent(mods)
	if !strings.Contains(got, "Administration") || !strings.Contains(got, "3.4.0") {
		t.Errorf("missing module in output: %q", got)
	}
	if !strings.Contains(got, "AtlasCore") {
		t.Errorf("missing AtlasCore: %q", got)
	}
}

func TestCaptureDescribe_WritesToBuffer(t *testing.T) {
	var buf bytes.Buffer
	ctx := &ExecContext{Output: &buf}

	result, err := captureDescribe(ctx, func(c *ExecContext) error {
		fmt.Fprintln(c.Output, "hello world")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world\n" {
		t.Errorf("got %q, want %q", result, "hello world\n")
	}
	if buf.String() != "" {
		t.Errorf("original output was written to: %q", buf.String())
	}
}

func TestExportProject_ProjectLevelFiles(t *testing.T) {
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be, err := mprbackend.NewFromPath(dst)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	outDir := t.TempDir()
	exec := New(os.Stderr)
	exec.backend = be
	exec.cache = &executorCache{}

	opts := ExportOptions{
		Progress: func(line string) { t.Log(line) },
	}
	if err := exec.ExportProject(outDir, opts); err != nil {
		t.Fatalf("ExportProject: %v", err)
	}

	marketFile := filepath.Join(outDir, "_marketplace.mdl")
	if _, err := os.Stat(marketFile); err != nil {
		t.Errorf("_marketplace.mdl missing: %v", err)
	}

	projDir := filepath.Join(outDir, "_project")
	if _, err := os.Stat(projDir); err != nil {
		t.Errorf("_project/ dir missing: %v", err)
	}
}

func TestExportProject_ModuleDocuments(t *testing.T) {
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be, err := mprbackend.NewFromPath(dst)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	outDir := t.TempDir()
	exec := New(os.Stderr)
	exec.backend = be
	exec.cache = &executorCache{}

	if err := exec.ExportProject(outDir, ExportOptions{Progress: func(l string) { t.Log(l) }}); err != nil {
		t.Fatalf("ExportProject: %v", err)
	}

	var mdlFiles []string
	_ = filepath.WalkDir(outDir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".mdl") {
			mdlFiles = append(mdlFiles, p)
		}
		return nil
	})
	if len(mdlFiles) == 0 {
		t.Errorf("no .mdl files exported under %s", outDir)
	}
	hasModuleFile := false
	hasPerDocumentFile := false
	for _, f := range mdlFiles {
		rel, _ := filepath.Rel(outDir, f)
		if !strings.HasPrefix(rel, "_") {
			hasModuleFile = true
		}
		base := filepath.Base(rel)
		if base != "_module.mdl" && base != "_associations.mdl" && base != "_module_roles.mdl" && !strings.HasPrefix(rel, "_") {
			hasPerDocumentFile = true
		}
	}
	if !hasModuleFile {
		t.Errorf("no module-level .mdl files found in %v", mdlFiles)
	}
	if !hasPerDocumentFile {
		t.Errorf("no per-document .mdl files (entities/microflows/pages) found in %v", mdlFiles)
	}
}

func TestSortMDLFiles_OrderedByType(t *testing.T) {
	files := []string{
		"MyModule/Pages/MyModule.Home.mdl",
		"MyModule/Domain/MyModule.Customer.mdl",
		"MyModule/_module.mdl",
		"MyModule/_associations.mdl",
		"MyModule/Microflows/ACT/MyModule.ACT_Create.mdl",
		"_project/settings.mdl",
		"_project/security.mdl",
		"_project/navigation.mdl",
		"_marketplace.mdl",
	}
	sorted := sortMDLFiles(files)

	moduleIdx := indexOfSuffix(sorted, "_module.mdl")
	domainIdx := indexOfSuffix(sorted, "MyModule.Customer.mdl")
	if moduleIdx < 0 || domainIdx < 0 {
		t.Fatal("missing expected files")
	}
	if moduleIdx > domainIdx {
		t.Errorf("_module.mdl (idx %d) should come before Domain/ (idx %d)", moduleIdx, domainIdx)
	}

	assocIdx := indexOfSuffix(sorted, "_associations.mdl")
	if assocIdx < domainIdx {
		t.Errorf("_associations.mdl (idx %d) should come after Domain/ (idx %d)", assocIdx, domainIdx)
	}

	mfIdx := indexOfSuffix(sorted, "ACT_Create.mdl")
	pgIdx := indexOfSuffix(sorted, "Home.mdl")
	if mfIdx > pgIdx {
		t.Errorf("microflow (idx %d) should come before page (idx %d)", mfIdx, pgIdx)
	}

	settingsIdx := indexOfSuffix(sorted, "settings.mdl")
	if settingsIdx < pgIdx {
		t.Errorf("settings.mdl (idx %d) should come after pages (idx %d)", settingsIdx, pgIdx)
	}
}

func indexOfSuffix(slice []string, suffix string) int {
	for i, s := range slice {
		if strings.HasSuffix(filepath.ToSlash(s), suffix) {
			return i
		}
	}
	return -1
}

func TestImportProject_ExecutesFiles(t *testing.T) {
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be, err := mprbackend.NewFromPath(dst)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	exportDir := t.TempDir()
	exportExec := New(os.Stderr)
	exportExec.backend = be
	exportExec.cache = &executorCache{}
	if err := exportExec.ExportProject(exportDir, ExportOptions{Progress: func(l string) { t.Log("export:", l) }}); err != nil {
		t.Fatalf("ExportProject: %v", err)
	}
	_ = be.Disconnect()

	dst2 := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be2, err := mprbackend.NewFromPath(dst2)
	if err != nil {
		t.Fatalf("NewFromPath dst2: %v", err)
	}
	t.Cleanup(func() { _ = be2.Disconnect() })

	importExec := New(os.Stderr)
	importExec.backend = be2
	importExec.cache = &executorCache{}
	err = importExec.ImportProject(exportDir, ImportOptions{
		SkipErrors: true,
		Progress:   func(l string) { t.Log("import:", l) },
	})
	if err != nil {
		t.Fatalf("ImportProject: %v", err)
	}
}
