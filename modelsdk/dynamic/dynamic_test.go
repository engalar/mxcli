package dynamic_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/dynamic"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"

	_ "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
)

func TestPropertyKindConstantsAreDistinct(t *testing.T) {
	kinds := []dynamic.PropertyKind{
		dynamic.KindString,
		dynamic.KindBool,
		dynamic.KindInt32,
		dynamic.KindFloat64,
		dynamic.KindPart,
		dynamic.KindPartList,
		dynamic.KindByID,
		dynamic.KindStringList,
		dynamic.KindBinary,
		dynamic.KindUnknown,
	}
	seen := map[dynamic.PropertyKind]bool{}
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate kind value: %v", k)
		}
		seen[k] = true
	}
}

// ── Integration tests (require test MPR) ─────────────────────────

func findTestMPR(t testing.TB) string {
	t.Helper()
	patterns := []string{
		"testdata/corpus-a/app.mpr",
		"testdata/*/app.mpr",
	}
	root := filepath.Join("..", "..")
	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(root, p))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func firstEntity(t testing.TB, m *modelsdk.Model) element.Element {
	t.Helper()
	dms := m.AllOfType("DomainModels$DomainModel")
	for _, dm := range dms {
		for _, prop := range dm.Properties() {
			if prop.Name() == "Entities" {
				if cl, ok := prop.(element.ChildListProperty); ok {
					for _, child := range cl.ChildElements() {
						if child != nil && (child.TypeName() == "DomainModels$Entity" ||
							child.TypeName() == "DomainModels$EntityImpl") {
							return child
						}
					}
				}
			}
		}
	}
	t.Skip("no Entity child found in DomainModel")
	return nil
}

func TestDynamicPropertyKindDetection(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	entity := firstEntity(t, m)
	e := dynamic.WrapElement(entity)

	type check struct {
		name string
		kind dynamic.PropertyKind
	}
	checks := []check{
		{"Name", dynamic.KindString},
		{"Documentation", dynamic.KindString},
		{"IsRemote", dynamic.KindBool},
		{"Attributes", dynamic.KindPartList},
		{"MaybeGeneralization", dynamic.KindPart},
	}
	for _, c := range checks {
		p := e.Property(c.name)
		if p == nil {
			t.Errorf("property %q not found", c.name)
			continue
		}
		if p.Kind() != c.kind {
			t.Errorf("property %q: got kind=%v, want %v", c.name, p.Kind(), c.kind)
		}
	}
}

func TestDynamicPropertyReadWrite(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}

	tmpDir := t.TempDir()
	tmpMPR := filepath.Join(tmpDir, "test.mpr")
	copyFile(t, mprPath, tmpMPR)
	copyDirIfExists(t, filepath.Join(filepath.Dir(mprPath), "mprcontents"),
		filepath.Join(tmpDir, "mprcontents"))

	m, err := modelsdk.OpenForWriting(tmpMPR)
	if err != nil {
		t.Fatalf("OpenForWriting: %v", err)
	}
	defer m.Close()

	entity := firstEntity(t, m)
	typed := entity.(*domainmodels.Entity)
	origName := typed.Name()

	e := dynamic.WrapElement(entity)

	dynName, ok := e.GetString("Name")
	if !ok {
		t.Fatal("GetString(Name) failed")
	}
	if dynName != origName {
		t.Errorf("dynamic name=%q, typed name=%q", dynName, origName)
	}

	newName := "DynamicTest_" + origName
	if !e.SetString("Name", newName) {
		t.Fatal("SetString(Name) failed")
	}
	if typed.Name() != newName {
		t.Errorf("after dynamic write, typed Name=%q, want %q", typed.Name(), newName)
	}
	if !typed.IsDirty() {
		t.Error("element should be dirty after SetString")
	}

	typed.SetName(origName)
}

// ── Benchmarks ──────────────────────────────────────────────────

func entityForBench(b *testing.B) *domainmodels.Entity {
	b.Helper()
	mprPath := findTestMPR(b)
	if mprPath == "" {
		b.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	b.Cleanup(func() { m.Close() })

	dms := m.AllOfType("DomainModels$DomainModel")
	for _, dm := range dms {
		for _, prop := range dm.Properties() {
			if prop.Name() == "Entities" {
				if cl, ok := prop.(element.ChildListProperty); ok {
					for _, child := range cl.ChildElements() {
						if ent, ok := child.(*domainmodels.Entity); ok {
							return ent
						}
					}
				}
			}
		}
	}
	b.Skip("no Entity found")
	return nil
}

func BenchmarkReadStringTyped(b *testing.B) {
	e := entityForBench(b)
	_ = e.Name()
	b.ResetTimer()
	var s string
	for range b.N {
		s = e.Name()
	}
	_ = s
}

func BenchmarkReadStringDynamic(b *testing.B) {
	e := entityForBench(b)
	de := dynamic.WrapElement(e)
	_, _ = de.GetString("Name")
	b.ResetTimer()
	var s string
	var ok bool
	for range b.N {
		s, ok = de.GetString("Name")
	}
	_, _ = s, ok
}

func BenchmarkWriteStringTyped(b *testing.B) {
	e := entityForBench(b)
	orig := e.Name()
	b.ResetTimer()
	for range b.N {
		e.SetName("x")
	}
	e.SetName(orig)
}

func BenchmarkWriteStringDynamic(b *testing.B) {
	e := entityForBench(b)
	de := dynamic.WrapElement(e)
	orig, _ := de.GetString("Name")
	b.ResetTimer()
	for range b.N {
		de.SetString("Name", "x")
	}
	de.SetString("Name", orig)
}

func TestKnownTypes(t *testing.T) {
	descs := dynamic.KnownTypes()
	if len(descs) == 0 {
		t.Skip("no types registered — codegen may not have run")
	}
	t.Logf("registered %d types", len(descs))
	found := false
	for _, td := range descs {
		if td.TypeName == "DomainModels$Entity" {
			found = true
			break
		}
	}
	if !found {
		t.Error("DomainModels$Entity not found in descriptor registry")
	}
}

// ── Helpers ──────────────────────────────────────────────────────

func copyFile(t testing.TB, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func copyDirIfExists(t testing.TB, src, dst string) {
	t.Helper()
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return
	}
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		copyFile(t, path, target)
		return nil
	})
}
