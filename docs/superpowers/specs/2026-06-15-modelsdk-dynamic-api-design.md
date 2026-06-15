# Modelsdk Dynamic API Design

**Date:** 2026-06-15
**Status:** PoC validated — implementation plan pending
**Scope:** Add a dynamic/flexible API layer to modelsdk that enables runtime property access by name, type schema introspection, and BSON-level field access, coexisting with the existing typed API.

---

## 1. Goal

The current modelsdk is purely **compile-time typed**: every Mendix element type has a generated Go struct with typed getters/setters. Users must import domain packages and type-assert to concrete types:

```go
import _ "modelsdk/gen/domainmodels"
entity := m.AllOfType("DomainModels$Entity")[0].(*domainmodels.Entity)
name := entity.Name()
```

This forces callers to know the exact type at compile time, making it impossible to write generic model inspection tools, CLI commands that operate on arbitrary properties by name, or scripts that work with model types unknown at build time.

**End state:** modelsdk exposes three complementary access layers:

| Layer | Capability | Use Case |
|-------|-----------|----------|
| **A: DynamicProperty** | `elem.GetString("Name")`, `elem.SetString("Name", "x")` | Generic model manipulation without type assertions |
| **B: Type Introspection** | `elem.Properties()` → `[{Name, Kind, ...}]`, type enumeration | Schema-aware tooling, model browsers |
| **C: Raw BSON Access** | `RawString(elem, "Name")` | Fast field reads, search indexing |

---

## 2. Current State

### Existing infrastructure (usable)

- `element.Property` interface: `Name() string`
- `element.WritableProperty`: `Dirty() bool`, `BSONValue() any`
- `element.ChildProperty` / `element.ChildListProperty`: child element access
- `element.Base.Properties()`: returns `[]Property`
- `codec.DefaultRegistry`: `map[string]func() element.Element` type factories
- `codec.DefaultRefRegistry`: reference metadata per type

### Gaps

- No way to get a `Property` by name without linear scan
- No uniform read/write API: each property type has its own `Get() T` / `Set(T)`
- `BSONValue() any` exists but `Part[T]` returns nil (child path)
- No generated property metadata (kind, value type, BSON key) outside of Go struct definitions
- `Registry.TypeRegistry` doesn't expose the list of registered type names

### POC Validation

A PoC at `modelsdk/poc/dynamic/` implemented all three layers and measured:

| Operation | Typed | Dynamic | Raw BSON |
|-----------|-------|---------|----------|
| Read string (warm) | 0.87 ns | 25 ns | 165 ns |
| Write string | 68 ns | 100 ns | — |
| Property iteration (18 props) | 22 ns | 24 ns | — |

Dynamic read is 28x slower than typed but takes only **25 nanoseconds** — irrelevant in CLI context where MPR I/O dominates (milliseconds). Write overhead is 1.5x. Property iteration is essentially equal after caching.

---

## 3. Design

### 3.1 DynamicProperty interface (Level A)

The core abstraction. Every existing property type implements this:

```go
package property

type DynamicProperty interface {
    element.Property
    Kind() PropertyKind
    Value() any              // uniform read
    SetValue(v any) error    // uniform write
    Children() []element.Element  // for Part/PartList
}
```

Implemented on each concrete type via type switch:

- `*Primitive[string]` → `Value() string`, `SetValue(any)` validates string
- `*Primitive[bool]` → `Value() bool`, `SetValue(any)` validates bool
- `*Enum[string]` → `Value() string`, `SetValue(any)` validates string
- `*Part[T]` → `Children()` returns `[element.Element]`
- `*PartList[T]` → `Children()` returns `[]element.Element`
- `*ByNameRef[T]` → `Value() string` (qualified name)
- `*ByIdRef[T]` → `Value() element.ID`

`PropertyKind` enum covers all property types: `KindString`, `KindBool`, `KindInt32`, `KindFloat64`, `KindPart`, `KindPartList`, `KindByNameRef`, `KindByNameRefList`, `KindByIdRef`, `KindEnum`, `KindStringList`, `KindBinaryUUID`.

### 3.2 Dynamic Element Wrapper (Level A user API)

A cached wrapper around `element.Element`:

```go
package dynamic  // modelsdk/dynamic/

type Element struct {
    elem   element.Element
    cached []*Property      // lazy-init via sync.Once
    byName map[string]*Property
}
```

```go
func WrapElement(elem element.Element) *Element
func (e *Element) Property(name string) *Property
func (e *Element) Properties() []*Property
func (e *Element) GetString(name string) (string, bool)
func (e *Element) SetString(name, val string) bool
func (e *Element) GetBool(name string) (bool, bool)
func (e *Element) SetBool(name string, val bool) bool
```

Kind detection: inferred once from `BSONValue()` return type + interface checks (`ChildProperty`/`ChildListProperty`).

### 3.3 Type Descriptor Registry (Level B)

Generated alongside existing types, extending the pattern from `codec.RefRegistry`:

```go
package codec

type PropertyKind uint8

const (
    PropKindString     PropertyKind = ...
    PropKindBool       PropertyKind = ...
    PropKindPart       PropertyKind = ...
    PropKindPartList   PropertyKind = ...
    PropKindByNameRef  PropertyKind = ...
    PropKindByNameList PropertyKind = ...
    PropKindByIdRef    PropertyKind = ...
    PropKindEnum       PropertyKind = ...
    PropKindStringList PropertyKind = ...
    PropKindBinaryUUID PropertyKind = ...
)

type PropertyDescriptor struct {
    Name    string       // property name (as returned by .Name())
    BSONKey string       // BSON storage key (may differ from Name)
    Kind    PropertyKind
    RefType string       // target type name for references (empty for primitives)
}

type TypeDescriptor struct {
    TypeName   string
    Properties []PropertyDescriptor
}

var DefaultTypeRegistry *TypeRegistry  // (replacing the old DefaultRegistry name)
```

Generated `init()` in each `gen/*/types.go` registers descriptors:

```go
func init() {
    codec.DefaultTypeRegistry.Register("DomainModels$Entity", &codec.TypeDescriptor{
        TypeName: "DomainModels$Entity",
        Properties: []codec.PropertyDescriptor{
            {Name: "Name", BSONKey: "Name", Kind: codec.PropKindString},
            {Name: "MaybeGeneralization", BSONKey: "MaybeGeneralization", Kind: codec.PropKindPart},
            {Name: "Attributes", BSONKey: "Attributes", Kind: codec.PropKindPartList},
            // ...
        },
    })
}
```

The generator (`cmd/modelsdk-codegen`) already has per-type property knowledge — this is a new output from the existing emitter pass.

### 3.4 BSON Helper Functions (Level C)

Stateless functions for direct raw BSON access, already proven in PoC:

```go
package dynamic  // or codec

func RawString(elem element.Element, key string) (string, bool)
func RawBool(elem element.Element, key string) (bool, bool)
func RawInt32(elem element.Element, key string) (int32, bool)
```

These call `elem.Raw().LookupErr(key)` and convert. No caching. Useful when only a single field is needed and the full property decode overhead is undesirable.

---

## 4. Package Structure

```
modelsdk/
  dynamic/                        # NEW — user-facing dynamic API
    element.go                    # Element wrapper, WrapElement, convenience methods
    property.go                   # Property wrapper, Kind inference, Value/SetValue (if needed externally)
    raw.go                        # RawString, RawBool, RawInt32 (Level C)

  property/
    base.go                       # + DynamicProperty interface
    primitive.go                  # + Value()/SetValue() on Primitive[T]
    part.go                       # + Children() on Part[T]/PartList[T]
    reference.go                  # + Value()/SetValue() on ByNameRef/ByIdRef
    enum.go                       # + Value()/SetValue() on Enum[T]
                                    
  codec/
    registry.go                   # TypeRegistry → rename or extend for TypeDescriptor
    descriptor.go                 # NEW — TypeDescriptor, PropertyDescriptor, PropertyKind

  gen/*/
    types.go                      # + init() registers TypeDescriptor for each type
```

---

## 5. Incremental Delivery

### Phase 1 — DynamicProperty (Level A, ~300 lines)

1. Add `DynamicProperty` interface + `PropertyKind` to `property/`
2. Implement `Value() any` / `SetValue(any)` on all property types
3. Add `Element` wrapper to `modelsdk/dynamic/`
4. Add `Model.Types() []TypeDescriptor` to `modelsdk/model.go` (returns registered descriptors)
5. Codegen: emit `RegisterTypeDescriptors` calls in each `init()`
6. Tests: verify read/write parity with typed API for string/bool/part/partlist

### Phase 2 — Type Introspection (Level B, ~100 lines + codegen)

1. Implement `TypeDescriptor` generation in codegen emitter
2. Export `codec.TypeRegistry.AllTypes()` → `[]TypeDescriptor`
3. Add `Model.KnownTypes()` to `modelsdk/model.go`
4. Tests: verify descriptor contents match actual generated struct fields

### Phase 3 — BSON Helpers (Level C, ~50 lines)

1. Add `RawString`, `RawBool`, `RawInt32` to `modelsdk/dynamic/`
2. Tests: verify raw access matches typed access
3. Documentation: show when to use each level

---

## 6. Performance Budget

From PoC benchmarks (Entity.Name on corpus-a MPR, avg of 5 runs):

| Layer | Latency | vs Typed | B/op | Allocs |
|-------|---------|----------|------|--------|
| Typed | 0.87 ns | 1x | 0 | 0 |
| Dynamic (warm, cached) | 25 ns | 28x | 16 | 1 |
| Raw BSON | 165 ns | 190x | 5 | 1 |
| Write typed | 68 ns | 1x | 24 | 2 |
| Write dynamic | 100 ns | 1.5x | 40 | 3 |
| Property iter typed | 22 ns | 1x | 0 | 0 |
| Property iter dynamic | 24 ns | 1.1x | 0 | 0 |

Worst-case: dynamic read at 25 ns × 1,000 properties = 25 µs — invisible next to MPR I/O.

---

## 7. Key Decisions

### 7.1 Why not a standalone dynamic package?

Could put `Element` wrapper in a separate module. But `property.Value() any` must be on the concrete types in `modelsdk/property/`, so there's no avoiding modifying the core property types. Keeping everything in one module avoids split-package seams.

### 7.2 Why generated TypeDescriptors instead of reflect-based?

Reflection on generic types in Go is fragile — it cannot distinguish `Primitive[string]` from `Primitive[bool]` at the interface level without type-asserting every instantiation. A generated registry is deterministic, verifiable, and the codegen already has the required information.

### 7.3 Dirty tracking for dynamic writes

`DynamicProperty.SetValue()` calls the existing `Set()` / `SetQualifiedName()` / `SetID()` which internally call `markDirty()`. The dirty bitmap + container propagation work unchanged.

### 7.4 Kind inference without TypeDescriptors

The PoC shows BSONValue() return type is sufficient for kind detection (string → KindString, bool → KindBool, etc.). This works for all current property types. The TypeDescriptor registry is an optimization for schema queries, not required for runtime access.

---

## 8. Open Questions

None identified — POC resolved the design questions. See Section 7 for rationale on each.
