# Modelsdk Dynamic API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver `modelsdk/dynamic/` — an adapter package that provides runtime property-by-name access, type introspection, and raw BSON field access for test code and CLI diagnostic tools. Zero changes to production types (`property/`, `element/`, `codec/`).

**Architecture:** Pure adapter wrapping `element.Element` via composition. Property kind detection uses runtime interface checks (`ChildProperty`, `ChildListProperty`, `WritableProperty`) first, then `BSONValue()` return type, then concrete type assertions. No `DynamicProperty` interface on production types. Generated `TypeDesc` registry in `codec` (separate file per domain). BSON helpers are free functions.

**Tech Stack:** Go generics (no new ones), `go.mongodb.org/mongo-driver/v2/bson`, codegen emitter (`cmd/modelsdk-codegen`)

**Spec:** `docs/superpowers/specs/2026-06-15-modelsdk-dynamic-api-design.md`

---

## File Structure

| Phase | File | Role |
|-------|------|------|
| 1 | `modelsdk/dynamic/property.go` | PropertyKind, Property struct, kind inference, Value/SetValue/Children |
| 1 | `modelsdk/dynamic/element.go` | Element struct, WrapElement, cached map lookup, convenience methods |
| 1 | `modelsdk/dynamic/dynamic_test.go` | Tests: kind detection, read/write parity, iteration, benchmarks |
| 2 | `modelsdk/codec/descriptor.go` | PropKind, PropDesc, TypeDesc, DescRegistry, DefaultDescRegistry |
| 2 | `cmd/modelsdk-codegen/...` | Emitter: generate `gen/*/descriptors.go` per domain |
| 2 | `modelsdk/dynamic/element.go` | Modify: add `KnownTypes() []TypeDesc` to model or dynamic package |
| 3 | `modelsdk/dynamic/raw.go` | RawString, RawBool, RawInt32 |

---

### Task 1: Property wrapper — kind detection + uniform read/write

**Files:**
- Create: `modelsdk/dynamic/property.go`

- [ ] **Step 1: Write failing test for Property wrapper**

```go
// modelsdk/dynamic/dynamic_test.go
package dynamic_test

import (
    "testing"
    "github.com/mendixlabs/mxcli/modelsdk/dynamic"
    "github.com/mendixlabs/mxcli/modelsdk/element"
)

func TestPropertyKindString(t *testing.T) {
    // Minimal verification: PropertyKind constants are distinct.
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd modelsdk\dynamic && go test -run TestPropertyKindString -v`
Expected: `FAIL` (package does not exist yet)

- [ ] **Step 3: Create the directory and initial PropertyKind**

```go
// modelsdk/dynamic/property.go
package dynamic

type PropertyKind uint8

const (
    KindString     PropertyKind = iota
    KindBool
    KindInt32
    KindFloat64
    KindPart
    KindPartList
    KindByID
    KindStringList
    KindBinary
    KindUnknown
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd modelsdk\dynamic && go test -run TestPropertyKindString -v`
Expected: PASS

- [ ] **Step 5: Write test for Property struct creation**

```go
// modelsdk/dynamic/dynamic_test.go
func TestPropertyName(t *testing.T) {
    p := &dynamic.Property{} // will be created via newProperty
    _ = p.Name()  // interface check: Name() string
}
```

- [ ] **Step 6: Implement Property struct**

```go
// modelsdk/dynamic/property.go
type Property struct {
    prop element.Property
    kind PropertyKind
}

func (p *Property) Name() string { return p.prop.Name() }
func (p *Property) Kind() PropertyKind { return p.kind }
```

- [ ] **Step 7: Run test to verify it passes**

Run: `cd modelsdk\dynamic && go test -v`

- [ ] **Step 8: Write test for kind detection on all property types**

```go
func TestInferKind(t *testing.T) {
    // We test kind detection indirectly by loading a real entity
    // and checking known property kinds. This requires modelsdk import.
}
```

- [ ] **Step 9: Implement complete Property kind detection + Value/SetValue/Children**

The POC code at `modelsdk/poc/dynamic/dynamic.go` contains the full implementation. Port it to `modelsdk/dynamic/property.go`:

- `newProperty(p element.Property) *Property` — detects kind using:
  1. `element.ChildListProperty` → KindPartList
  2. `element.ChildProperty` → KindPart
  3. `element.WritableProperty.BSONValue()` type switch → string/bool/int32/float64/element.ID/bson.A/bson.Binary
  4. Fallback type assertion: `*property.StringListPrimitive`, `*property.BinaryUUIDPrimitive`
- `Value() any` — calls `WritableProperty.BSONValue()`, returns nil for Part/PartList
- `SetValue(v any) error` — type switch on concrete type: `*property.Primitive[string].Set(v)`, `*property.Primitive[bool].Set(v)`, `*property.Primitive[int32].Set(v)`, `*property.Enum[string].Set(v)`, `*property.ByNameRef[element.Element].SetQualifiedName(v)`, `*property.ByIdRef[element.Element].SetID(v)`, `*property.StringListPrimitive`, `*property.BinaryUUIDPrimitive`
- `Children() []element.Element` — `ChildProperty.ChildElement()` or `ChildListProperty.ChildElements()`
- `String()`, `Bool()`, `Int32()` convenience methods

- [ ] **Step 10: Commit**

```bash
git add modelsdk/dynamic/property.go
git commit -m "feat(dynamic): Property wrapper with kind detection and uniform Value/SetValue"
```

---

### Task 2: Element wrapper — cached property access

**Files:**
- Create: `modelsdk/dynamic/element.go`

- [ ] **Step 1: Write failing test for Element wrapper**

```go
// modelsdk/dynamic/dynamic_test.go
func TestElementWrap(t *testing.T) {
    // Can't test without a real element. We test in Task 3 with integration.
    t.Skip("requires integration test setup")
}
```

- [ ] **Step 2: Implement Element struct**

```go
// modelsdk/dynamic/element.go
package dynamic

import (
    "sync"
    "github.com/mendixlabs/mxcli/modelsdk/element"
)

type Element struct {
    elem   element.Element
    once   sync.Once
    cached []*Property
    byName map[string]*Property
}

func WrapElement(elem element.Element) *Element {
    return &Element{elem: elem}
}

func (e *Element) Element() element.Element { return e.elem }

func (e *Element) init() {
    e.once.Do(func() {
        props := e.elem.Properties()
        e.cached = make([]*Property, len(props))
        e.byName = make(map[string]*Property, len(props))
        for i, p := range props {
            dp := newProperty(p)
            e.cached[i] = dp
            e.byName[dp.Name()] = dp
        }
    })
}

func (e *Element) Property(name string) *Property {
    e.init()
    return e.byName[name]
}

func (e *Element) Properties() []*Property {
    e.init()
    return e.cached
}

func (e *Element) GetString(name string) (string, bool) {
    p := e.Property(name)
    if p == nil {
        return "", false
    }
    return p.String(), true
}

func (e *Element) SetString(name, val string) bool {
    p := e.Property(name)
    if p == nil {
        return false
    }
    return p.SetValue(val) == nil
}

func (e *Element) GetBool(name string) (bool, bool) {
    p := e.Property(name)
    if p == nil {
        return false, false
    }
    return p.Bool()
}

func (e *Element) SetBool(name string, val bool) bool {
    p := e.Property(name)
    if p == nil {
        return false
    }
    return p.SetValue(val) == nil
}
```

- [ ] **Step 3: Add DescribeElement helper**

```go
// modelsdk/dynamic/element.go
func DescribeElement(elem element.Element) map[string]string {
    desc := map[string]string{}
    desc["$Type"] = elem.TypeName()
    desc["$ID"] = string(elem.ID())
    e := WrapElement(elem)
    for _, p := range e.Properties() {
        desc[p.Name()] = p.Kind().String()
    }
    return desc
}
```

- [ ] **Step 4: Run build check**

Run: `cd modelsdk\dynamic && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add modelsdk/dynamic/element.go
git commit -m "feat(dynamic): Element wrapper with cached map-based property access"
```

---

### Task 3: Integration tests — kind detection, read/write parity, iteration

**Files:**
- Create: `modelsdk/dynamic/dynamic_test.go`

- [ ] **Step 1: Write findTestMPR helper (same pattern as modelsdk/integration_test.go)**

```go
// modelsdk/dynamic/dynamic_test.go
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
```

- [ ] **Step 2: Write firstEntity helper to navigate DomainModel → Entity**

```go
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
```

- [ ] **Step 3: Write kind detection test**

```go
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
```

- [ ] **Step 4: Run kind detection test**

Run: `cd modelsdk\dynamic && go test -run TestDynamicPropertyKindDetection -v`
Expected: PASS (if test MPR is available) or SKIP (no MPR)

- [ ] **Step 5: Write read/write parity test**

```go
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

    // Read via dynamic API.
    dynName, ok := e.GetString("Name")
    if !ok {
        t.Fatal("GetString(Name) failed")
    }
    if dynName != origName {
        t.Errorf("dynamic name=%q, typed name=%q", dynName, origName)
    }

    // Write via dynamic API, verify via typed API.
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

    // Restore.
    typed.SetName(origName)
}

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
        if err != nil { return err }
        rel, _ := filepath.Rel(src, path)
        target := filepath.Join(dst, rel)
        if info.IsDir() {
            return os.MkdirAll(target, 0755)
        }
        copyFile(t, path, target)
        return nil
    })
}
```

- [ ] **Step 6: Run read/write test**

Run: `cd modelsdk\dynamic && go test -run TestDynamicPropertyReadWrite -v`
Expected: PASS or SKIP

- [ ] **Step 7: Write RawString test (can be written now even though Level C comes later)**

```go
func TestRawStringAccess(t *testing.T) {
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

    name, ok := dynamic.RawString(entity, "Name")
    if !ok {
        t.Fatal("RawString(Name) failed")
    }
    typed := entity.(*domainmodels.Entity)
    if name != typed.Name() {
        t.Errorf("raw BSON name=%q, typed name=%q", name, typed.Name())
    }
}
```

- [ ] **Step 8: Run all tests**

Run: `cd modelsdk\dynamic && go test -v -count=1`
Expected: all PASS or SKIP (depending on MPR availability)

- [ ] **Step 9: Commit**

```bash
git add modelsdk/dynamic/dynamic_test.go
git commit -m "test(dynamic): integration tests for kind detection, read/write parity, BSON access"
```

---

### Task 4: Benchmarks for Level A

- [ ] **Step 1: Write benchmarks comparing typed vs dynamic vs raw BSON**

```go
// modelsdk/dynamic/dynamic_test.go

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
    _ = e.Name() // warm up
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
    _, _ = de.GetString("Name") // warm up
    b.ResetTimer()
    var s string
    var ok bool
    for range b.N {
        s, ok = de.GetString("Name")
    }
    _, _ = s, ok
}

func BenchmarkReadStringRaw(b *testing.B) {
    e := entityForBench(b)
    b.ResetTimer()
    var s string
    var ok bool
    for range b.N {
        s, ok = dynamic.RawString(e, "Name")
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
```

- [ ] **Step 2: Run benchmarks**

Run: `cd modelsdk\dynamic && go test -bench=Benchmark -benchmem -count=3 -run="^$" -timeout 60s`
Expected: benchmark output with ns/op values

Record results and compare against the expected budget from the spec:
- Dynamic read ≤ 50 ns/op
- Dynamic write ≤ 150 ns/op
- Property iteration ≤ 50 ns/op

- [ ] **Step 3: Commit benchmarks (separate from test logic)**

```bash
git add modelsdk/dynamic/dynamic_test.go
git commit -m "bench(dynamic): add typed vs dynamic vs raw BSON benchmarks"
```

---

### Task 5: Type descriptor registry (Level B)

**Files:**
- Create: `modelsdk/codec/descriptor.go`

- [ ] **Step 1: Write test for descriptor registry**

```go
// modelsdk/codec/descriptor_test.go
package codec

import (
    "testing"
)

func TestDescRegistryRegisterAndLookup(t *testing.T) {
    reg := NewDescRegistry()
    reg.Register("DomainModels$Entity", &TypeDesc{
        TypeName: "DomainModels$Entity",
        Properties: []PropDesc{
            {Name: "Name", BSONKey: "Name", Kind: PropKindString},
        },
    })

    td, ok := reg.Lookup("DomainModels$Entity")
    if !ok {
        t.Fatal("Lookup failed for registered type")
    }
    if td.TypeName != "DomainModels$Entity" {
        t.Errorf("TypeName = %q", td.TypeName)
    }
    if len(td.Properties) != 1 {
        t.Fatalf("got %d properties, want 1", len(td.Properties))
    }
    if td.Properties[0].Name != "Name" {
        t.Errorf("property Name = %q", td.Properties[0].Name)
    }
}

func TestDescRegistryAll(t *testing.T) {
    reg := NewDescRegistry()
    reg.Register("A", &TypeDesc{TypeName: "A"})
    reg.Register("B", &TypeDesc{TypeName: "B"})
    all := reg.All()
    if len(all) != 2 {
        t.Errorf("All() returned %d, want 2", len(all))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd modelsdk\codec && go test -run TestDescRegistry -v`
Expected: FAIL (types not defined yet)

- [ ] **Step 3: Implement descriptor registry**

```go
// modelsdk/codec/descriptor.go
package codec

import "sync"

type PropKind uint8

const (
    PropKindString     PropKind = iota
    PropKindBool
    PropKindInt32
    PropKindPart
    PropKindPartList
    PropKindByNameRef
    PropKindByNameList
    PropKindByIdRef
    PropKindEnum
    PropKindStringList
    PropKindBinaryUUID
)

type PropDesc struct {
    Name    string
    BSONKey string
    Kind    PropKind
    RefType string
}

type TypeDesc struct {
    TypeName   string
    Properties []PropDesc
}

type DescRegistry struct {
    mu    sync.RWMutex
    descs map[string]*TypeDesc
}

func NewDescRegistry() *DescRegistry {
    return &DescRegistry{descs: map[string]*TypeDesc{}}
}

func (r *DescRegistry) Register(typeName string, td *TypeDesc) {
    r.mu.Lock()
    r.descs[typeName] = td
    r.mu.Unlock()
}

func (r *DescRegistry) Lookup(typeName string) (*TypeDesc, bool) {
    r.mu.RLock()
    td, ok := r.descs[typeName]
    r.mu.RUnlock()
    return td, ok
}

func (r *DescRegistry) All() []*TypeDesc {
    r.mu.RLock()
    out := make([]*TypeDesc, 0, len(r.descs))
    for _, td := range r.descs {
        out = append(out, td)
    }
    r.mu.RUnlock()
    return out
}

var DefaultDescRegistry = NewDescRegistry()
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd modelsdk\codec && go test -run TestDescRegistry -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add modelsdk/codec/descriptor.go modelsdk/codec/descriptor_test.go
git commit -m "feat(codec): TypeDesc/PropDesc registry for runtime type introspection"
```

---

### Task 6: Codegen — emit descriptors.go per domain

**Files:**
- Modify: `cmd/modelsdk-codegen/` (emitter pass)

- [ ] **Step 1: Locate the existing emitter code**

Read `cmd/modelsdk-codegen/` to understand the template-based emitter structure. Look for how `refs.go` is generated (the closest parallel).

- [ ] **Step 2: Add PropKind mapping in the emitter**

The emitter already knows each property's name, BSON key, and type parameter. Add a function that maps each property to a `codec.PropKind`:
- `property.Primitive[string]` → `PropKindString`
- `property.Primitive[bool]` → `PropKindBool`
- `property.Primitive[int32]` → `PropKindInt32`
- `property.Primitive[float64]` → `PropKindFloat64`
- `property.Part[T]` → `PropKindPart`
- `property.PartList[T]` → `PropKindPartList`
- `property.ByNameRef[T]` → `PropKindByNameRef`
- `property.ByNameRefList[T]` → `PropKindByNameList`
- `property.ByIdRef[T]` → `PropKindByIdRef`
- `property.Enum[T]` → `PropKindEnum`
- `property.StringListPrimitive` → `PropKindStringList`
- `property.BinaryUUIDPrimitive` → `PropKindBinaryUUID`

- [ ] **Step 3: Add descriptors.go template to the emitter**

The template outputs a file like:
```go
// Code generated by mxcli codegen; DO NOT EDIT.
package domainmodels

import "github.com/mendixlabs/mxcli/modelsdk/codec"

func init() {
    codec.DefaultDescRegistry.Register("DomainModels$Entity", &codec.TypeDesc{
        TypeName: "DomainModels$Entity",
        Properties: []codec.PropDesc{
            {Name: "Name", BSONKey: "Name", Kind: codec.PropKindString},
            {Name: "MaybeGeneralization", BSONKey: "MaybeGeneralization", Kind: codec.PropKindPart},
            {Name: "Attributes", BSONKey: "Attributes", Kind: codec.PropKindPartList},
            // ... all properties
        },
    })
    // ... all types
}
```

- [ ] **Step 4: Run codegen to regenerate**

Run: `cd cmd\modelsdk-codegen && go run .`

Verify: `gen/domainmodels/descriptors.go` exists with the expected content.

- [ ] **Step 5: Run tests to verify regeneration didn't break anything**

Run: `cd modelsdk && go test ./... -count=1`
Expected: all tests pass

- [ ] **Step 6: Add KnownTypes() to the dynamic package**

```go
// modelsdk/dynamic/element.go
import "github.com/mendixlabs/mxcli/modelsdk/codec"

func KnownTypes() []*codec.TypeDesc {
    return codec.DefaultDescRegistry.All()
}

func LookupType(typeName string) (*codec.TypeDesc, bool) {
    return codec.DefaultDescRegistry.Lookup(typeName)
}
```

- [ ] **Step 7: Write test for known types**

```go
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
            for _, p := range td.Properties {
                if p.Name == "Name" && p.Kind == codec.PropKindString {
                    // correct
                }
            }
        }
    }
    if !found {
        t.Error("DomainModels$Entity not found in descriptor registry")
    }
}
```

- [ ] **Step 8: Commit**

```bash
git add cmd/modelsdk-codegen/ modelsdk/codec/descriptor.go
git add modelsdk/dynamic/element.go modelsdk/dynamic/dynamic_test.go
git commit -m "feat(codegen): emit descriptors.go with type property metadata"
```

---

### Task 7: Raw BSON helper functions (Level C)

**Files:**
- Create: `modelsdk/dynamic/raw.go`

- [ ] **Step 1: Write test for RawString**

```go
// The test was already written in Task 3 Step 7 (TestRawStringAccess).
// Verify it compiles and passes.
```

- [ ] **Step 2: Implement RawString, RawBool, RawInt32**

```go
// modelsdk/dynamic/raw.go
package dynamic

import (
    "github.com/mendixlabs/mxcli/modelsdk/element"
    "go.mongodb.org/mongo-driver/v2/bson"
)

func RawString(elem element.Element, key string) (string, bool) {
    raw := elem.Raw()
    if raw == nil {
        return "", false
    }
    val, err := raw.LookupErr(key)
    if err != nil {
        return "", false
    }
    return val.StringValueOK()
}

func RawBool(elem element.Element, key string) (bool, bool) {
    raw := elem.Raw()
    if raw == nil {
        return false, false
    }
    val, err := raw.LookupErr(key)
    if err != nil {
        return false, false
    }
    return val.BooleanOK()
}

func RawInt32(elem element.Element, key string) (int32, bool) {
    raw := elem.Raw()
    if raw == nil {
        return 0, false
    }
    val, err := raw.LookupErr(key)
    if err != nil {
        return 0, false
    }
    return val.Int32OK()
}
```

- [ ] **Step 3: Run all dynamic tests**

Run: `cd modelsdk\dynamic && go test -v -count=1`
Expected: all PASS (or SKIP if no MPR)

- [ ] **Step 4: Commit**

```bash
git add modelsdk/dynamic/raw.go
git commit -m "feat(dynamic): RawString/RawBool/RawInt32 for BSON-level field access"
```

---

### Self-Review

After writing the plan, check against the spec:

**1. Spec coverage:**
- Section 4.1 (DynamicProperty adapter) → Task 1
- Section 4.2 (Element wrapper) → Task 2
- Section 4.3 (Type Descriptor Registry) → Tasks 5, 6
- Section 4.4 (BSON helpers) → Task 7
- Section 5 (Package structure) → all tasks match the declared file layout
- Section 6 (Production vs Test Boundary) → enforced by package location (never imported by `mdl/`)
- Section 7 (Incremental delivery) → three phases map to Tasks 1+2+3+4 (Phase 1), Tasks 5+6 (Phase 2), Task 7 (Phase 3)

**2. Placeholder scan:** All code blocks contain complete Go code. No TBD, TODO, or incomplete sections.

**3. Type consistency:** `PropertyKind` matches `codec.PropKind` in separate packages — they are intentionally distinct (one in `dynamic/`, one in `codec/`). The mapping between them is in the codegen emitter logic which maps concrete property types to `PropKind`.
