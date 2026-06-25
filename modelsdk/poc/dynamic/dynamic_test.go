//go:build poc
package dynamic_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/poc/dynamic"
	"go.mongodb.org/mongo-driver/v2/bson"

	_ "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
)

// findTestMPR searches well-known locations for a test .mpr file.
func findTestMPR(t testing.TB) string {
	t.Helper()
	patterns := []string{
		"testdata/corpus-a/app.mpr",
		"testdata/*/app.mpr",
	}
	root := filepath.Join("..", "..", "..")
	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(root, p))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// firstEntity loads the first populated DomainModel and returns its first entity child.
func firstEntity(t testing.TB, m *modelsdk.Model) element.Element {
	t.Helper()
	dms := m.AllOfType("DomainModels$DomainModel")
	if len(dms) == 0 {
		t.Skip("no DomainModel units found")
	}
	for _, dm := range dms {
		children := dm.Properties()
		for _, prop := range children {
			if prop.Name() == "Entities" {
				if cl, ok := prop.(element.ChildListProperty); ok {
					for _, child := range cl.ChildElements() {
						if child != nil && (child.TypeName() == "DomainModels$Entity" || child.TypeName() == "DomainModels$EntityImpl") {
							return child
						}
					}
				}
			}
		}
	}
	// Debug: show what's in Entities for the first domain model
	if len(dms) > 0 {
		t.Logf("First DomainModel children:")
		for _, prop := range dms[0].Properties() {
			if prop.Name() == "Entities" {
				if cl, ok := prop.(element.ChildListProperty); ok {
					for _, child := range cl.ChildElements() {
						if child != nil {
							t.Logf("  child: TypeName=%q ID=%q", child.TypeName(), child.ID())
						} else {
							t.Logf("  child: nil")
						}
					}
				} else {
					t.Logf("  Entities property type: %T", prop)
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
	t.Logf("Element: %s (%s)", e.Element().TypeName(), e.Element().ID())
	t.Logf("Properties:")
	for _, p := range e.Properties() {
		t.Logf("  %s: kind=%s", p.Name(), p.Kind())
	}

	// Verify known property kinds on Entity.
	type entityProps map[string]dynamic.Kind
	want := entityProps{
		"Name":                dynamic.KindString,
		"DataStorageGuid":     dynamic.KindString,
		"Documentation":       dynamic.KindString,
		"IsRemote":            dynamic.KindBool,
		"Attributes":          dynamic.KindPartList,
		"ValidationRules":     dynamic.KindPartList,
		"MaybeGeneralization": dynamic.KindPart,
		"Capabilities":        dynamic.KindPart,
		"ExportLevel":         dynamic.KindString,
	}
	for name, expectedKind := range want {
		p := e.Property(name)
		if p == nil {
			t.Errorf("property %q not found", name)
			continue
		}
		if p.Kind() != expectedKind {
			t.Errorf("property %q: got kind=%v, want %v", name, p.Kind(), expectedKind)
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
	srcContents := filepath.Join(filepath.Dir(mprPath), "mprcontents")
	if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
		dstContents := filepath.Join(tmpDir, "mprcontents")
		copyDir(t, srcContents, dstContents)
	}

	m, err := modelsdk.OpenForWriting(tmpMPR)
	if err != nil {
		t.Fatalf("OpenForWriting: %v", err)
	}
	defer m.Close()

	entity := firstEntity(t, m)

	e := dynamic.WrapElement(entity)

	// Read via typed API.
	typed := entity.(*domainmodels.Entity)
	origName := typed.Name()
	t.Logf("Original name: %q", origName)

	// Read via dynamic API.
	dynName, ok := e.GetString("Name")
	if !ok {
		t.Fatal("GetString(Name) failed")
	}
	if dynName != origName {
		t.Errorf("dynamic name=%q, typed name=%q", dynName, origName)
	}

	// Write via dynamic API.
	newName := "DynamicRenameTest_" + origName
	if !e.SetString("Name", newName) {
		t.Fatal("SetString(Name) failed")
	}

	// Verify via typed API.
	if typed.Name() != newName {
		t.Errorf("after dynamic SetString, typed Name=%q, want %q", typed.Name(), newName)
	}

	// Verify dirty tracking.
	if !typed.IsDirty() {
		t.Error("element should be dirty after SetString")
	}

	// Restore.
	typed.SetName(origName)
}

func TestDynamicAllOfType(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	// Collect type summary using dynamic API.
	typeCounts := map[string]int{}
	for _, unit := range m.Units() {
		elem, err := m.LoadUnit(unit.ID)
		if err != nil {
			continue
		}
		typeCounts[elem.TypeName()]++
	}
	t.Logf("Type summary (%d types):", len(typeCounts))
	for typeName, count := range typeCounts {
		t.Logf("  %s: %d", typeName, count)
	}
}

func TestRawBSONAccess(t *testing.T) {
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
	// Level C: read directly from BSON.
	name, ok := dynamic.RawString(entity, "Name")
	if !ok {
		t.Fatal("RawString(Name) failed")
	}
	t.Logf("Raw BSON Name: %q", name)

	// Verify it matches typed access.
	typed := entity.(*domainmodels.Entity)
	if name != typed.Name() {
		t.Errorf("raw BSON name=%q, typed name=%q", name, typed.Name())
	}
}

func TestDescribeElement(t *testing.T) {
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

	desc := dynamic.DescribeElement(entity)
	t.Logf("Element description:")
	for k, v := range desc {
		t.Logf("  %s: %s", k, v)
	}
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
	entity := entityForBench(b)
	_ = entity.Name() // warm up lazy decode
	b.ResetTimer()
	var s string
	for range b.N {
		s = entity.Name()
	}
	_ = s
}

func BenchmarkReadStringDynamic(b *testing.B) {
	entity := entityForBench(b)
	e := dynamic.WrapElement(entity)
	_, _ = e.GetString("Name") // warm up
	b.ResetTimer()
	var s string
	var ok bool
	for range b.N {
		s, ok = e.GetString("Name")
	}
	_, _ = s, ok
}

func BenchmarkReadStringRaw(b *testing.B) {
	entity := entityForBench(b)
	b.ResetTimer()
	var s string
	var ok bool
	for range b.N {
		s, ok = dynamic.RawString(entity, "Name")
	}
	_, _ = s, ok
}

func BenchmarkWriteStringTyped(b *testing.B) {
	entity := entityForBench(b)
	orig := entity.Name()
	b.ResetTimer()
	for range b.N {
		entity.SetName(fmt.Sprintf("test%d", b.N))
	}
	entity.SetName(orig)
}

func BenchmarkWriteStringDynamic(b *testing.B) {
	entity := entityForBench(b)
	e := dynamic.WrapElement(entity)
	orig, _ := e.GetString("Name")
	b.ResetTimer()
	for range b.N {
		e.SetString("Name", fmt.Sprintf("test%d", b.N))
	}
	e.SetString("Name", orig)
}

func BenchmarkPropertyIterTyped(b *testing.B) {
	entity := entityForBench(b)
	b.ResetTimer()
	for range b.N {
		for _, p := range entity.Properties() {
			_ = p.Name()
		}
	}
}

func BenchmarkPropertyIterDynamic(b *testing.B) {
	entity := entityForBench(b)
	e := dynamic.WrapElement(entity)
	b.ResetTimer()
	for range b.N {
		for _, p := range e.Properties() {
			_ = p.Name()
			_ = p.Kind()
		}
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

func copyDir(t testing.TB, src, dst string) {
	t.Helper()
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

// TestAPIDemo shows what the dynamic API looks like in practice.
func TestAPIDemo(t *testing.T) {
	mprPath := findTestMPR(t)
	if mprPath == "" {
		t.Skip("no test MPR found")
	}
	m, err := modelsdk.Open(mprPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer m.Close()

	fmt.Println("=== Dynamic API Demo ===")
	fmt.Printf("MPR: %s\n", mprPath)
	fmt.Printf("Total units: %d\n\n", len(m.Units()))

	// Show all unique type names in the MPR (for debugging).
	typeNames := map[string]int{}
	for _, u := range m.Units() {
		typeNames[u.Type]++
	}
	fmt.Println("--- Unit types in MPR ---")
	for name, count := range typeNames {
		fmt.Printf("  %s: %d\n", name, count)
	}
	fmt.Println()

	// Level A: read/write properties by name.
	fmt.Println("--- Level A: Property Access by Name ---")
	if entity := firstEntity(t, m); entity != nil {
		e := dynamic.WrapElement(entity)
		name, _ := e.GetString("Name")
		doc, _ := e.GetString("Documentation")
		isRemote, _ := e.GetBool("IsRemote")
		fmt.Printf("Entity: name=%q doc=%q isRemote=%v\n", name, doc, isRemote)
		fmt.Printf("Properties (%d):\n", len(e.Properties()))
		for _, p := range e.Properties() {
			fmt.Printf("  %s: [%s]", p.Name(), p.Kind())
			if p.Kind() == dynamic.KindString || p.Kind() == dynamic.KindBool {
				fmt.Printf(" = %v", p.Value())
			}
			fmt.Println()
		}
	}
	fmt.Println()

	// Level B: type introspection.
	fmt.Println("--- Level B: Type Introspection ---")
	for range m.Units() {
		// already cached
	}
	for _, u := range m.Units() {
		elem, err := m.LoadUnit(u.ID)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(elem.TypeName(), "DomainModels$") {
			continue
		}
		desc := dynamic.DescribeElement(elem)
		fmt.Printf("Type: %s\n", desc["$Type"])
		for k, v := range desc {
			if k != "$Type" && k != "$ID" {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
		break // just one type
	}
	fmt.Println()

	// Level C: BSON-level access.
	fmt.Println("--- Level C: BSON-level Access ---")
	if entity := firstEntity(t, m); entity != nil {
		name, ok := dynamic.RawString(entity, "Name")
		if ok {
			fmt.Printf("Raw BSON Name: %q\n", name)
		}
		raw := entity.Raw()
		elems, _ := bson.Raw(raw).Elements()
		fmt.Println("BSON fields:")
		for _, f := range elems {
			fmt.Printf("  %s: type=%v\n", f.Key(), f.Value().Type)
		}
	}
}
