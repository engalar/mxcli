# Modelsdk Dynamic API Design

**Date:** 2026-06-15
**Status:** PoC validated — implementation plan pending
**Scope:** Add a dynamic/flexible API adapter to modelsdk for test code and auxiliary tooling. The dynamic API is NOT used by production execution paths — those continue to use the strongly-typed API exclusively.

---

## 1. Goal

The current modelsdk is purely **compile-time typed**: every Mendix element type has a generated Go struct with typed getters/setters. Users must import domain packages and type-assert to concrete types:

```go
import _ "modelsdk/gen/domainmodels"
entity := m.AllOfType("DomainModels$Entity")[0].(*domainmodels.Entity)
name := entity.Name()
```

This is ideal for production code (type-safe, zero overhead). But for **test code and auxiliary tooling** the typed API creates friction:

- **Tests**: asserting model state requires knowing the exact type, importing domain packages, and boilerplate type assertions — especially painful for generic test helpers that must work across domains
- **CLI diagnostic commands**: `mxcli model get <type> <name> <property>` can't know the property name at compile time
- **Feature exploration**: prototyping a new command that touches multiple property types requires importing many gen packages and repeatedly revisiting type assertions

**End state:** modelsdk exposes a lightweight dynamic adapter in `modelsdk/dynamic/` with three access layers, importable only by test code and CLI diagnostic commands:

| Layer | Capability | Use Case |
|-------|-----------|----------|
| **A: Property by name** | `elem.GetString("Name")`, `elem.GetBool("IsRemote")` | Test assertions, diagnostic commands |
| **B: Type introspection** | `elem.Properties()` → `[{Name, Kind}]`, type enumeration | Model browsers, generic diff tools |
| **C: Raw BSON access** | `RawString(elem, "Name")` | Fast field reads in search/indexing |

**Production code** (`mdl/repos/`, `mdl/backend/`, `executor/`, `cli/` command handlers) continues to use typed API exclusively. The dynamic package is a pure adapter — zero changes to production types.

---

## 2. SOLID Design Principles

This design must satisfy SOLID. Each section below identifies how:

| Principle | How this design satisfies it |
|-----------|-----------------------------|
| **S**ingle Responsibility | `modelsdk/dynamic/` has one job: provide dynamic access. Each internal type (`Element`, `Property`) has one responsibility. |
| **O**pen/Closed | No production type is modified. `property/` types are not touched. Dynamic access is achieved through interface checks (`ChildProperty`, `ChildListProperty`, `WritableProperty`) and external type switches — open to new property types without modifying the adapter. |
| **L**iskov Substitution | `dynamic.Element` wraps `element.Element` by composition, not inheritance. It is NOT a subtype of `element.Element` — there is no substitution. |
| **I**nterface Segregation | The dynamic API is its own interface (`DynamicProperty` inside the `dynamic` package). No new methods are added to `element.Property` or `element.Element` — consumers of the typed API see zero change. |
| **D**ependency Inversion | `modelsdk/dynamic/` depends on `element.Element` (interface) and `element.Property` (interface) — abstractions, not concretions. The adapter never imports `gen/*/` packages. |

---

## 3. Current State

### Existing infrastructure (usable without modification)

- `element.Property` interface: `Name() string`
- `element.WritableProperty`: `Dirty() bool`, `BSONValue() any`
- `element.ChildProperty` / `element.ChildListProperty`: child element access
- `element.Base.Properties()`: returns `[]Property`
- `codec.DefaultRegistry`: type name → factory mapping
- `codec.DefaultRefRegistry`: reference metadata per type

### Infrastructure NOT used (preserving SOLID O/C)

We do NOT add interfaces or methods to `property/` package types. Production types remain untouched. The dynamic adapter uses type switches on the public interfaces (`WritableProperty`, `ChildProperty`, `ChildListProperty`) and concrete types (`*property.Primitive[string]`, etc.) to provide uniform access.

### POC Validation

A PoC at `modelsdk/poc/dynamic/` implemented all three layers as an external adapter:

| Operation | Typed | Dynamic | Raw BSON |
|-----------|-------|---------|----------|
| Read string (warm) | 0.87 ns | 25 ns | 165 ns |
| Write string | 68 ns | 100 ns | — |
| Property iteration (18 props) | 22 ns | 24 ns | — |

The 25 ns dynamic read overhead is irrelevant for test code and CLI diagnostics.

---

## 4. Design

### 4.1 DynamicProperty adapter (Level A)

Defined in the `dynamic` package only. NOT added to `property/` production types:

```go
package dynamic

// PropertyKind identifies the category of a property value.
type PropertyKind uint8

const (
    KindString     PropertyKind = iota  // Primitive[string], Enum, ByNameRef
    KindBool                            // Primitive[bool]
    KindInt32                           // Primitive[int32]
    KindFloat64                         // Primitive[float64]
    KindPart                            // single child element
    KindPartList                        // list of child elements
    KindByID                            // element ID reference
    KindStringList                      // StringListPrimitive, EnumList
    KindBinary                          // BinaryUUIDPrimitive
    KindUnknown
)

// Property wraps an element.Property with dynamic access.
type Property struct {
    prop element.Property
    kind PropertyKind
    // lazily initialized via once.Do
}

func newProperty(p element.Property) *Property { ... }
func (p *Property) Name() string
func (p *Property) Kind() PropertyKind
func (p *Property) Value() any                // uniform read; nil for Part/PartList
func (p *Property) SetValue(v any) error      // uniform write via type switch
func (p *Property) Children() []element.Element  // for Part/PartList
func (p *Property) String() (string, bool)    // convenience
func (p *Property) Bool() (bool, bool)        // convenience
func (p *Property) Int32() (int32, bool)      // convenience
```

Kind detection algorithm (in `newProperty`):

1. Check `element.ChildListProperty` → KindPartList
2. Check `element.ChildProperty` → KindPart
3. Fall back to `element.WritableProperty.BSONValue()` return type:
   - `string` → KindString
   - `bool` → KindBool
   - `int32` → KindInt32
   - `float64` → KindFloat64
   - `element.ID` → KindByID
   - `bson.A` → KindStringList
   - `bson.Binary` → KindBinary
4. Fall back to type-assert on concrete types (`*property.StringListPrimitive`, `*property.BinaryUUIDPrimitive`)

`SetValue` uses a type switch on the concrete property type (`*property.Primitive[string]`, `*property.Primitive[bool]`, `*property.Enum[string]`, `*property.ByNameRef[element.Element]`, etc.) to call the typed setter.

### 4.2 Element wrapper (Level A user API)

```go
package dynamic

type Element struct {
    elem   element.Element
    cached []*Property     // lazy-init via sync.Once
    byName map[string]*Property
}

func WrapElement(elem element.Element) *Element
func (e *Element) Element() element.Element         // unwrap to typed
func (e *Element) Property(name string) *Property
func (e *Element) Properties() []*Property
func (e *Element) GetString(name string) (string, bool)
func (e *Element) SetString(name, val string) bool
func (e *Element) GetBool(name string) (bool, bool)
func (e *Element) SetBool(name string, val bool) bool
```

`WrapElement` is the only entry point. The wrapper lazily initializes the property cache on first access (`sync.Once`). The `Element()` method returns the underlying `element.Element` for callers that need to switch back to typed access.

### 4.3 Type Descriptor Registry (Level B)

A generated registry of type → property metadata. Stored in `codec` package alongside the existing `RefRegistry`:

```go
package codec

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
    Name    string   // property name
    BSONKey string   // BSON storage key
    Kind    PropKind
    RefType string   // target type for references; empty for primitives
}

type TypeDesc struct {
    TypeName   string
    Properties []PropDesc
}
```

Generated `init()` in a separate file per domain package (e.g. `gen/domainmodels/descriptors.go`):

```go
func init() {
    codec.DefaultDescRegistry.Register("DomainModels$Entity", &codec.TypeDesc{
        TypeName: "DomainModels$Entity",
        Properties: []codec.PropDesc{
            {Name: "Name", BSONKey: "Name", Kind: codec.PropKindString},
            {Name: "MaybeGeneralization", BSONKey: "MaybeGeneralization", Kind: codec.PropKindPart},
            {Name: "Attributes", BSONKey: "Attributes", Kind: codec.PropKindPartList},
        },
    })
}
```

Separating into `descriptors.go` (not `types.go`) keeps the SRP clear — factory registration and descriptor registration are different responsibilities.

### 4.4 BSON Helper Functions (Level C)

```go
package dynamic

func RawString(elem element.Element, key string) (string, bool)
func RawBool(elem element.Element, key string) (bool, bool)
func RawInt32(elem element.Element, key string) (int32, bool)
```

These call `elem.Raw().LookupErr(key)` and convert. No caching. Useful when only a single field is needed and full property decode is undesirable.

---

## 5. Package Structure

```
modelsdk/
  dynamic/                        # NEW — pure adapter, no production imports
    element.go                    # Element wrapper, WrapElement, convenience methods
    property.go                   # Property wrapper, kind detection, Value/SetValue

  property/                       # UNCHANGED — no new interfaces or methods
    ...

  codec/
    registry.go                   # UNCHANGED (DefaultRegistry)
    descriptor.go                 # NEW — TypeDesc, PropDesc, PropKind, DefaultDescRegistry

  gen/*/
    types.go                      # UNCHANGED (factory registration only)
    descriptors.go                # NEW — init() registers TypeDesc for each type
    refs.go                       # UNCHANGED
```

Zero imports of `modelsdk/dynamic/` from any production package (`mdl/`, `cmd/`, `executor/`, etc.).

---

## 6. Production vs Test Boundary

```
Production code path:
  executor/handler.go
    → mdl/repos/EntityRepo
      → mdl/backend/mpr/repos/entity.go
        → modelsdk (typed API)
          → codec.Encoder/Decoder
            → Store (MPR I/O)

Test/auxiliary code path:
  _test.go or cmd/mxcli/describe.go
    → modelsdk/dynamic.WrapElement(elem)
      → elem.GetString("Name")
```

`modelsdk/dynamic/` is imported only:
- In `_test.go` files
- In CLI diagnostic commands (`mxcli describe`, `mxcli inspect`)
- Never in `mdl/`, `executor/`, `modelsdk/codec/`, or command handler implementations

---

## 7. Incremental Delivery

### Phase 1 — Property adapter (Level A, ~250 lines)

1. Create `modelsdk/dynamic/property.go`: `PropertyKind`, `Property` struct, kind detection, `Value()/SetValue()/Children()`
2. Create `modelsdk/dynamic/element.go`: `Element` wrapper, `WrapElement`, `Property(name)`, `Properties()`, convenience methods
3. Tests: verify read/write parity with typed API for string/bool/part/partlist across 3+ element types
4. Add benchmark to ensure performance stays within budget

### Phase 2 — Type Descriptor generation (Level B, ~150 lines + codegen)

1. Define `codec.TypeDesc`, `codec.PropDesc`, `codec.PropKind`, `codec.DefaultDescRegistry`
2. Modify codegen emitter to output `gen/*/descriptors.go` alongside `gen/*/refs.go`
3. Add `Model.KnownTypes() []TypeDesc` or `dynamic.KnownTypes() []TypeDesc`
4. Tests: verify descriptor contents match actual generated struct fields

### Phase 3 — BSON helpers (Level C, ~40 lines)

1. Add `dynamic.RawString`, `RawBool`, `RawInt32`
2. Tests: verify raw access matches typed access

---

## 8. Performance Budget

From PoC benchmarks (Entity.Name on corpus-a MPR, avg of 5 runs):

| Layer | Latency | vs Typed | B/op | Allocs |
|-------|---------|----------|------|--------|
| Typed `entity.Name()` | 0.87 ns | 1x | 0 | 0 |
| Dynamic `e.GetString("Name")` | 25 ns | 28x | 16 | 1 |
| Raw BSON `RawString(e, "Name")` | 165 ns | 190x | 5 | 1 |
| Write typed `entity.SetName("x")` | 68 ns | 1x | 24 | 2 |
| Write dynamic `e.SetString("Name", "x")` | 100 ns | 1.5x | 40 | 3 |
| Property iter typed | 22 ns | 1x | 0 | 0 |
| Property iter dynamic (cached) | 24 ns | 1.1x | 0 | 0 |

Dynamic read overhead is 25 ns — irrelevant for test code and CLI diagnostics. The 28x ratio is a misleading comparison since the baseline (0.87 ns) is the cost of returning a cached Go string. In absolute terms, 25 ns per property access is negligible.

---

## 9. Key Decisions

### 9.1 DynamicProperty as an external adapter, not a production interface

The `Property` wrapper in `modelsdk/dynamic/` is a pure adapter. Kind detection uses runtime interface checks and type switches, not a production `DynamicProperty` interface. This avoids any modification to `property/` types and satisfies the Open/Closed principle.

### 9.2 TypeDescriptor generation is optional for test code

The PoC shows kind detection works without descriptors — the `dynamic.Property` wrapper infers kind from `BSONValue()` return type. TypeDescriptors are a complementary feature for tooling that needs schema queries without loading data. They can be deferred to Phase 2.

### 9.3 Dirty tracking works through existing typed setters

`Property.SetValue()` calls the concrete type's `Set()` / `SetQualifiedName()` / `SetID()`, which internally call `markDirty()`. The dirty bitmap + container chain propagation work unchanged. No additional dirty tracking logic needed in the dynamic adapter.

### 9.4 Separating descriptors from factory registration

Descriptors go in a separate `gen/*/descriptors.go` file (not appended to `types.go`) to maintain SRP. The `init()` in `descriptors.go` only registers descriptors; the `init()` in `types.go` only registers factories.
