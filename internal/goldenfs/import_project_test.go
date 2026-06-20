// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// buildSyntheticExport writes a minimal mxcli-export-style MDL tree under dir,
// covering: a module role, a persistent entity with a role-scoped grant, and
// a page that grants view access to the same role. The shape mirrors the
// outputs of `mxcli export` so the importer exercises the real document
// ordering (roles → entities → pages).
func buildSyntheticExport(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"MyFirstModule/_module_roles.mdl": "create module role MyFirstModule.ImportTestRole description 'test';\n/\n",
		"MyFirstModule/Domain/MyFirstModule.ImportTestEntity.mdl": `create or modify persistent entity MyFirstModule.ImportTestEntity (
  Code: String(50)
);

grant MyFirstModule.ImportTestRole on MyFirstModule.ImportTestEntity (create, read *, write *);
/
`,
		"MyFirstModule/Pages/MyFirstModule.ImportTestPage.mdl": `create or modify page MyFirstModule.ImportTestPage (
  Title: 'Import Test',
  Layout: Atlas_Core.Atlas_Default
) { }

grant view on page MyFirstModule.ImportTestPage to MyFirstModule.ImportTestRole;
/
`,
	}
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("buildSyntheticExport mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("buildSyntheticExport write %s: %v", rel, err)
		}
	}
}

// runImportViaFUSE drives ImportProject through the FUSE-mounted .mpr.
// The connect happens via an MDL script (matching the runMDL pattern in
// bsoncompare_integration_test.go); ImportProject is then called directly
// because it is not a parseable MDL statement.
//
// The executor is closed before this function returns so the SQLite handle
// is released — without this, downstream bsoncompare opens of the same file
// can race against the still-attached connection.
func runImportViaFUSE(t *testing.T, mountMpr, exportDir string) string {
	t.Helper()
	var buf bytes.Buffer
	e := executor.New(&buf)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close: %v", err)
		}
	}()

	prog, errs := visitor.Build("connect local '" + mountMpr + "';")
	if len(errs) > 0 {
		t.Fatalf("parse connect: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Fatalf("connect: %v\n%s", err, buf.String())
	}

	opts := executor.ImportOptions{SkipErrors: true}
	if err := e.ImportProject(exportDir, opts); err != nil {
		t.Fatalf("ImportProject: %v\n%s", err, buf.String())
	}
	return buf.String()
}

// assertEntityHasGrant returns a bson.D check that fails unless the named
// entity's AccessRules contain an AllowedModuleRoles entry for roleName.
//
// Regression target: msdkWrite used to call BeginWriteTransaction inline,
// flushing entity+grant to disk; importBuf.Flush then overwrote the same
// entity from the cached pre-grant snapshot, silently dropping the grant.
func assertEntityHasGrant(entityName, roleName string) func(bson.D) error {
	return func(doc bson.D) error {
		for _, entElem := range bsonLookupArray(doc, "Entities") {
			entDoc, ok := entElem.(bson.D)
			if !ok {
				continue
			}
			if bsonLookupStr(entDoc, "Name") != entityName {
				continue
			}
			for _, ruleElem := range bsonLookupArray(entDoc, "AccessRules") {
				ruleDoc, ok := ruleElem.(bson.D)
				if !ok {
					continue
				}
				for _, roleElem := range bsonLookupArray(ruleDoc, "AllowedModuleRoles") {
					if roleStr, ok := roleElem.(string); ok && roleStr == roleName {
						return nil
					}
				}
			}
			return fmt.Errorf("entity %q has no AccessRule for role %q", entityName, roleName)
		}
		return fmt.Errorf("entity %q not found in domain model BSON", entityName)
	}
}

// assertPageHasAllowedRole returns a bson.D check that fails unless the page
// unit's AllowedRoles list contains roleName.
//
// Regression target: msdkWritePage had the same sessionBuf bypass as
// msdkWrite — page+AllowedRoles flushed inline, then overwritten by the
// cached page without AllowedRoles.
func assertPageHasAllowedRole(roleName string) func(bson.D) error {
	return func(doc bson.D) error {
		for _, elem := range bsonLookupArray(doc, "AllowedRoles") {
			if s, ok := elem.(string); ok && s == roleName {
				return nil
			}
		}
		return fmt.Errorf("page AllowedRoles does not contain %q", roleName)
	}
}

// bsonLookupArray returns the bson.A value at key, or nil if missing or
// wrong type. Used in lieu of a Lookup helper to keep tests self-contained.
func bsonLookupArray(doc bson.D, key string) bson.A {
	for _, e := range doc {
		if e.Key == key {
			if arr, ok := e.Value.(bson.A); ok {
				return arr
			}
		}
	}
	return nil
}

// bsonLookupStr returns the string value at key, or "" if missing or wrong type.
func bsonLookupStr(doc bson.D, key string) string {
	for _, e := range doc {
		if e.Key == key {
			if s, ok := e.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}

// TestImportProject_EntityGrantSurvivesSessionBuf verifies that an entity
// grant added via ImportProject is present in the on-disk BSON after the
// import finishes (i.e. survives importBuf.Flush). Before the fix, the
// inline msdkWrite transaction would write entity+grant, then the buffered
// flush would replay a pre-grant entity snapshot on top and discard the grant.
func TestImportProject_EntityGrantSurvivesSessionBuf(t *testing.T) {
	snap, err := Open(exprCheckerDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")
	exportDir := t.TempDir()
	buildSyntheticExport(t, exportDir)

	output := runImportViaFUSE(t, mountMpr, exportDir)
	t.Logf("ImportProject output:\n%s", output)

	// Write-path audit: surfaces exactly which overlay files were touched.
	t.Log("=== Write-path audit (DirtyPaths) ===")
	for _, p := range snap.DirtyPaths() {
		t.Logf("  %s", p)
	}

	// bsoncompare.AssertEqual cannot find the DomainModel by name because
	// DomainModels$DomainModel and Security$ModuleSecurity share the same
	// QualifiedName ("MyFirstModule.") when the domain model's Name field is
	// empty — indexUnits keeps only the last entry for a given QN. Instead,
	// scan the dirty overlay directly for the first DomainModels$DomainModel
	// file and assert the entity grant is present.
	foundDM := false
	for _, p := range snap.DirtyPaths() {
		if !strings.HasSuffix(p, ".mxunit") {
			continue
		}
		data := snap.ReadDirtyFile(p)
		if data == nil {
			continue
		}
		var doc bson.D
		if err := bson.Unmarshal(data, &doc); err != nil {
			continue
		}
		if bsonLookupStr(doc, "$Type") != "DomainModels$DomainModel" {
			continue
		}
		foundDM = true
		if err := assertEntityHasGrant("ImportTestEntity", "MyFirstModule.ImportTestRole")(doc); err != nil {
			t.Errorf("entity grant check failed on %s: %v", p, err)
		}
		break
	}
	if !foundDM {
		t.Errorf("no DomainModels$DomainModel found in dirty overlay — entity write may have been dropped")
	}

	snap.Rollback()
}

// TestImportProject_PageGrantSurvivesSessionBuf is the page-side analogue of
// TestImportProject_EntityGrantSurvivesSessionBuf: confirms AllowedRoles on
// MyFirstModule.ImportTestPage survives importBuf.Flush.
func TestImportProject_PageGrantSurvivesSessionBuf(t *testing.T) {
	snap, err := Open(exprCheckerDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")
	exportDir := t.TempDir()
	buildSyntheticExport(t, exportDir)

	output := runImportViaFUSE(t, mountMpr, exportDir)
	t.Logf("ImportProject output:\n%s", output)

	t.Log("=== Write-path audit (DirtyPaths) ===")
	for _, p := range snap.DirtyPaths() {
		t.Logf("  %s", p)
	}

	bsoncompare.AssertEqual(t,
		filepath.Join(exprCheckerDir(t), "minimal.mpr"),
		mountMpr,
		bsoncompare.DefaultOptions(),
		bsoncompare.WithUnitCheck("MyFirstModule.ImportTestPage",
			assertPageHasAllowedRole("MyFirstModule.ImportTestRole"),
		),
	)

	snap.Rollback()
}

// TestImportProject_PluggableWidgetTypeNotPointer scans every page unit that
// the importer wrote into the dirty layer and fails if any
// CustomWidgets$CustomWidget has a Type field that is a binary $ID pointer
// rather than an inline CustomWidgets$CustomWidgetType document.
//
// Regression target: the page builder used to emit pluggable-widget Type
// fields as $ID binary references, which caused mx check to crash with
// NullReferenceException because the referenced widget-type unit didn't
// exist.
//
// With the current synthetic export (no widgets in the page body), this is
// effectively a "no CustomWidget → no error" canary. Once the page builder
// emits real pluggable widgets through ImportProject, extend
// buildSyntheticExport to include one (e.g. a ComboBox) and this test will
// catch any regression to the binary-pointer form.
func TestImportProject_PluggableWidgetTypeNotPointer(t *testing.T) {
	snap, err := Open(exprCheckerDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mountMpr := filepath.Join(snap.MountDir(), "minimal.mpr")
	exportDir := t.TempDir()
	buildSyntheticExport(t, exportDir)

	output := runImportViaFUSE(t, mountMpr, exportDir)
	t.Logf("ImportProject output:\n%s", output)

	for _, relPath := range snap.DirtyPaths() {
		if !strings.HasSuffix(relPath, ".mxunit") {
			continue
		}
		data := snap.ReadDirtyFile(relPath)
		if data == nil {
			continue
		}
		var doc bson.D
		if err := bson.Unmarshal(data, &doc); err != nil {
			continue
		}
		if bsonLookupStr(doc, "$Type") != "Forms$Page" {
			continue
		}
		pageName := bsonLookupStr(doc, "Name")
		if err := assertNoPointerWidgetTypes(doc); err != nil {
			t.Errorf("page %q: %v", pageName, err)
		}
	}

	snap.Rollback()
}

// assertNoPointerWidgetTypes walks a page document and returns an error if any
// CustomWidgets$CustomWidget has a Type field that is a binary pointer ($ID
// bytes) rather than an inline CustomWidgets$CustomWidgetType document.
func assertNoPointerWidgetTypes(doc bson.D) error {
	return walkBSON(doc, func(d bson.D) error {
		if bsonLookupStr(d, "$Type") != "CustomWidgets$CustomWidget" {
			return nil
		}
		for _, e := range d {
			if e.Key != "Type" {
				continue
			}
			switch e.Value.(type) {
			case bson.D:
				return nil // inline object — correct
			case bson.Binary:
				return fmt.Errorf(
					"CustomWidget %q: Type is a binary pointer, expected inline CustomWidgets$CustomWidgetType",
					bsonLookupStr(d, "Name"),
				)
			default:
				return fmt.Errorf(
					"CustomWidget %q: Type has unexpected type %T",
					bsonLookupStr(d, "Name"), e.Value,
				)
			}
		}
		return fmt.Errorf("CustomWidget %q: Type field missing", bsonLookupStr(d, "Name"))
	})
}

// walkBSON recursively walks a bson.D tree, invoking fn on every nested
// bson.D (including the root). Returns the first non-nil error from fn.
func walkBSON(doc bson.D, fn func(bson.D) error) error {
	if err := fn(doc); err != nil {
		return err
	}
	for _, e := range doc {
		switch v := e.Value.(type) {
		case bson.D:
			if err := walkBSON(v, fn); err != nil {
				return err
			}
		case bson.A:
			for _, item := range v {
				if d, ok := item.(bson.D); ok {
					if err := walkBSON(d, fn); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
