# Modelsdk-Native Stage 2: Repository Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the new `mdl/repos/` interface layer and its `mdl/backend/mpr/repos/` MPR implementations alongside the legacy `MprBackend`, per spec Section 10 Stage 2 (additive, non-breaking). Old `MprBackend`, `*ViaModelsdk*` functions, `MockBackend`, and every executor handler MUST stay untouched. Stage 3 (separate plan) handles cutover.

**Architecture:**
```
executor (unchanged)  →  backend.FullBackend (unchanged)  →  MprBackend (unchanged)
                                                                            │
                                                                            ▼
                                                                       OLD path (Stage 3 deletes)
            ┌────────────────────── new layers (this plan) ──────────────────────┐
            │  mdl/repos/             interfaces only                             │
            │  mdl/backend/mpr/repos/ MPR-backed implementations                  │
            │  mdl/repos/testing/     Recording mocks (microflows + pages only)   │
            │  mdl/backend/mpr/factory.go  NewExecutorContext(*mmpr.Writer)       │
            └─────────────────────────────────────────────────────────────────────┘
```

**Tech Stack:** Go 1.26, `modelsdk/codec`, `modelsdk/gen/*`, `modelsdk/mpr` (`mmpr` alias), `model` (for `model.ID`)

**Spec references:** All section numbers refer to `docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md` (Section 5 Interface Design, Section 6 ExecutorContext, Section 7 Construction Entry Point, Section 10 Stage 2). PoC findings live in `docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design-addendum.md`.

**Test Layout:**

| File | Responsibility |
|------|----------------|
| `mdl/repos/doc.go` | Package overview + dependency rules |
| `mdl/repos/microflows.go`, `pages.go`, … (16 files) | Reader/Writer/Repository interfaces per domain |
| `mdl/repos/page_mutator.go`, `workflow_mutator.go` | Large-unit incremental mutator interfaces (Section 5 amendment) |
| `mdl/repos/{id,resolver,cache,uow}.go` | Auxiliary interfaces (IDGenerator, QualifiedNameResolver, ReaderCache, TransactionFactory + UnitOfWork) |
| `mdl/backend/mpr/repos/doc.go` | Package overview |
| `mdl/backend/mpr/repos/microflows.go` | Full MicroflowRepository implementation |
| `mdl/backend/mpr/repos/pages.go` | Full PageRepository implementation incl. PageMutator |
| `mdl/backend/mpr/repos/{id,resolver,cache,uow}.go` | Auxiliary implementations wrapping `*mmpr.Writer` |
| `mdl/backend/mpr/factory.go` | `NewExecutorContext(*mmpr.Writer) *repos.ExecutorContext` |
| `mdl/repos/testing/doc.go`, `microflows.go`, `pages.go` | Recording mocks (only the two implemented domains) |
| Tests colocated with each implementation, fixture: `testdata/expr-checker/minimal.mpr` |

---

## Task 1: Package skeletons + dependency intent

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/doc.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/doc.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/testing/doc.go`

The three `doc.go` files anchor the new package tree, encode the dependency rules from spec Section 4, and let `go vet` confirm the import graph compiles before any interface lands.

- [ ] **Step 1: Verify the parent directories do not yet exist**
```bash
test ! -d /mnt/data_sdd/gh/mxcli-wt-02/mdl/repos && echo "ok: mdl/repos absent"
test ! -d /mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos && echo "ok: mdl/backend/mpr/repos absent"
```

- [ ] **Step 2: Create `mdl/repos/doc.go`**
```go
// SPDX-License-Identifier: Apache-2.0

// Package repos defines per-domain Repository / Service / Auxiliary
// interfaces used by the modelsdk-native executor.
//
// Layering (spec section 4):
//
//   executor → repos (this package, interfaces only)
//                ↑
//   mdl/backend/mpr/repos (MPR implementations, separate package)
//
// This package MUST NOT import any *implementation* package. It depends
// only on:
//   - github.com/mendixlabs/mxcli/model        (model.ID)
//   - github.com/mendixlabs/mxcli/modelsdk/gen/*  (gen types — interface signatures)
//   - github.com/mendixlabs/mxcli/modelsdk/element (element.Element)
//
// Stage 2 (this plan) defines all 16 domain interfaces. Microflows + Pages
// receive full implementations in mdl/backend/mpr/repos. The remainder are
// signature-only stubs marked `// TODO Stage 3 cutover` so Stage 3 handlers
// can compile.
package repos
```

- [ ] **Step 3: Create `mdl/backend/mpr/repos/doc.go`**
```go
// SPDX-License-Identifier: Apache-2.0

// Package mprrepos provides MPR-backed implementations of the
// mdl/repos interfaces. Every constructor accepts the shared
// *modelsdk/mpr.Writer (and derived dependencies) so a single Writer
// drives the entire repo set.
//
// Package name is "mprrepos" (not "repos") to avoid collision with the
// imported github.com/mendixlabs/mxcli/mdl/repos package.
package mprrepos
```

- [ ] **Step 4: Create `mdl/repos/testing/doc.go`**
```go
// SPDX-License-Identifier: Apache-2.0

// Package repostesting contains Recording mocks for the mdl/repos
// interfaces. Stage 2 only ships mocks for the two interfaces that
// have full implementations (Microflows + Pages); the remaining 14
// domains will gain mocks in the Stage 3 cutover plan.
package repostesting
```

- [ ] **Step 5: Sanity-check the empty packages**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go vet ./mdl/repos/... ./mdl/backend/mpr/repos/...
```
Expected: no output (success).

- [ ] **Step 6: Commit**
```bash
git add mdl/repos/doc.go mdl/backend/mpr/repos/doc.go mdl/repos/testing/doc.go
git commit -m "feat(repos): scaffold mdl/repos, mdl/backend/mpr/repos, mdl/repos/testing packages"
```

---

## Task 2: All 16 domain interface files

**Files (all under `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/`):**
- Create: `microflows.go`, `pages.go` (FULL signatures — these get implemented in Tasks 6–7)
- Create: `nanoflows.go`, `layouts.go`, `snippets.go`, `domainmodels.go`, `modules.go`, `enumerations.go`, `constants.go`, `workflows.go`, `services.go`, `mappings.go`, `settings.go`, `security.go`, `folders.go`, `images.go`, `agents.go` (signature stubs with `// TODO Stage 3 cutover`)

The exact list comes from spec Section 4 / Section 6. The implementer must verify against the actual gen tree before writing each file:
```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/modelsdk/gen/
```
The list above is taken from `modelsdk/gen/` (microflows, nanoflows, pages, navigation, …). If gen has been regenerated since this plan was written and a domain is missing or renamed, prefer the gen tree as the source of truth and note the discrepancy in the commit message.

- [ ] **Step 1: Write the FULL `microflows.go`**

Path: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/microflows.go`
```go
// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// MicroflowReader queries microflows. Implementations decode raw BSON via
// codec.Decoder; freshly-decoded gen objects are returned by value-pointer
// (the caller may mutate freely without affecting the cache).
type MicroflowReader interface {
	Get(id model.ID) (*genMf.Microflow, error)
	List(moduleID model.ID) ([]*genMf.Microflow, error)
	ListAll() ([]*genMf.Microflow, error)
	FindByQualifiedName(qn string) (*genMf.Microflow, error)
	IsRule(qn string) (bool, error)
}

// MicroflowWriter creates/updates/deletes/moves microflows. Container
// lineage is supplied positionally (parentUUID, containmentName) — gen
// objects do not store container identity (addendum Blocker 2).
type MicroflowWriter interface {
	Create(parentUUID string, containmentName string, mf *genMf.Microflow) error
	Update(mf *genMf.Microflow) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type MicroflowRepository interface {
	MicroflowReader
	MicroflowWriter
}
```

- [ ] **Step 2: Write the FULL `pages.go`**

Path: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/pages.go`
```go
// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

type PageReader interface {
	Get(id model.ID) (*genPg.Page, error)
	List(moduleID model.ID) ([]*genPg.Page, error)
	ListAll() ([]*genPg.Page, error)
	FindByQualifiedName(qn string) (*genPg.Page, error)
}

type PageWriter interface {
	Create(parentUUID string, containmentName string, page *genPg.Page) error
	Update(page *genPg.Page) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

// PageRepository combines reader + writer + the mutator factory for
// large-unit incremental editing (spec section 5 Mutator interfaces).
type PageRepository interface {
	PageReader
	PageWriter
	OpenForMutation(pageID model.ID) (PageMutator, error)
}
```

- [ ] **Step 3: Stub the remaining 14 domains**

For each of `nanoflows`, `layouts`, `snippets`, `domainmodels`, `modules`, `enumerations`, `constants`, `workflows`, `services`, `mappings`, `settings`, `security`, `folders`, `images`, `agents`, write a file in `mdl/repos/` matching this template (use the right gen import alias, e.g. `genDm` for `modelsdk/gen/domainmodels`, `genWf` for `modelsdk/gen/workflows`, etc.):

```go
// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genX "github.com/mendixlabs/mxcli/modelsdk/gen/<DOMAIN>"
)

// XReader / XWriter / XRepository — signatures intentionally minimal until
// Stage 3 cutover. Mirror the legacy <Domain>Backend interface for shape;
// see mdl/backend/<domain>.go.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type XReader interface {
	Get(id model.ID) (*genX.<RootType>, error)
	List(moduleID model.ID) ([]*genX.<RootType>, error)
}

type XWriter interface {
	Create(parentUUID string, containmentName string, x *genX.<RootType>) error
	Update(x *genX.<RootType>) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type XRepository interface {
	XReader
	XWriter
}
```

Special cases the implementer must respect:
- **`workflows.go`** — analogous to pages: include `OpenForMutation(workflowID model.ID) (WorkflowMutator, error)` on `WorkflowRepository`.
- **`services.go`** — composite domain (REST, OData, business events, JS actions, Java actions are all here in the legacy `ServiceBackend`). For Stage 2, expose **one** umbrella `ServiceRepository` with `Get(id) (element.Element, error)` + `Create/Update/Delete` taking `element.Element`. Decomposition into sub-types is a Stage 3 concern; mark TODO.
- **`security.go`** — root type is project-scoped (`genSec.ProjectSecurity`); `Reader.Get()` takes no ID.
- **`settings.go`** — covers project + module settings; mirror `SettingsBackend` and `ModuleSettingsBackend` — produce two interface pairs (`ProjectSettingsReader/Writer`, `ModuleSettingsReader/Writer`) under one file.
- **`folders.go`** — also project-scoped; the legacy `FolderBackend` returns folder trees by module ID.
- **`agents.go`** — covers Agent Editor (Agent, KnowledgeBase, ConsumedMCPService) — group as one repo with method-name discrimination.

If any gen package above does not export a `<RootType>`, replace with `element.Element` and add a TODO comment naming the gap.

- [ ] **Step 4: Verify all 16 files exist and compile**
```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/*.go | grep -v doc.go | wc -l
# expect: 16 (the 14 stubs + microflows.go + pages.go) — plus the four aux files added in Tasks 3–5 will bring the directory total higher
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go vet ./mdl/repos/...
```

- [ ] **Step 5: Commit**
```bash
git add mdl/repos/*.go
git commit -m "feat(repos): define 16 per-domain Reader/Writer/Repository interfaces (microflows + pages full; rest stub)"
```

---

## Task 3: Mutator interfaces (PageMutator, WorkflowMutator)

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/page_mutator.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/workflow_mutator.go`

These exist because addendum Blocker 3 proved that whole-element re-encode costs ~0.4 ms / 168 KB / 3 591 allocs even for a one-property change on a 25 KB unit. Pages and workflows are routinely an order of magnitude larger; ALTER throughput requires per-edit incremental application. The mutator interface mirrors the legacy `mdl/backend/mutation.go` shape so Stage 3 cutover can route ALTER handlers to it without API churn.

- [ ] **Step 1: Write `page_mutator.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// PageMutator performs localized edits on a page/snippet/layout unit
// without re-encoding the whole element on every call. Obtain via
// PageRepository.OpenForMutation; persist with Commit.
//
// Lifecycle: OpenForMutation → N edits → Commit (single InsertUnit /
// WriteTransaction.WriteUnit at the end). Mutators are not safe for
// concurrent use.
type PageMutator interface {
	SetWidgetProperty(widgetID model.ID, prop string, value any) error
	InsertWidget(parentID model.ID, slot string, widget element.Element) error
	DeleteWidget(widgetID model.ID) error
	ReplaceWidget(widgetID model.ID, replacement element.Element) error
	SetLayout(layoutQN string) error
	Commit() error
}
```

- [ ] **Step 2: Write `workflow_mutator.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// WorkflowMutator is the workflow-domain analogue of PageMutator.
// Activities, outcomes, branches, and paths are all addressed by their
// stable model.ID. Commit persists with one Writer call.
type WorkflowMutator interface {
	SetActivityProperty(activityID model.ID, prop string, value any) error
	InsertActivity(parentID model.ID, slot string, activity element.Element) error
	DeleteActivity(activityID model.ID) error
	ReplaceActivity(activityID model.ID, replacement element.Element) error
	Commit() error
}
```

The signature surface deliberately omits the wider legacy mutator API
(SetPluggableProperty, InsertPath, etc.); Stage 2 ships the canonical
shape, Stage 3 expands to cover every ALTER form the executor emits.

- [ ] **Step 3: Verify the new symbols are reachable from `pages.go` and `workflows.go`**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./mdl/repos/...
```

- [ ] **Step 4: Commit**
```bash
git add mdl/repos/page_mutator.go mdl/repos/workflow_mutator.go
git commit -m "feat(repos): add PageMutator and WorkflowMutator interfaces (addendum Blocker 3)"
```

---

## Task 4: Auxiliary service interfaces + minimal implementations

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/id.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/resolver.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/cache.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/id.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/resolver.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/cache.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/aux_test.go`

The three auxiliaries break circular dependencies and isolate infrastructure
concerns from domain repos (spec Section 5 Auxiliary Services).

- [ ] **Step 1: Interface — `mdl/repos/id.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repos

import "github.com/mendixlabs/mxcli/model"

// IDGenerator mints fresh model.IDs. Implementations must produce IDs
// in the same UUID-string shape mmpr.GenerateID produces (addendum
// Blocker 2: element.ID(mmpr.GenerateID()) is the canonical cast).
type IDGenerator interface {
	NewID() model.ID
}
```

- [ ] **Step 2: Interface — `mdl/repos/resolver.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repos

import "github.com/mendixlabs/mxcli/model"

// QualifiedNameResolver answers "what kind of element is this name?"
// without recursing through domain repositories — it queries the
// underlying SQLite catalog directly via *mmpr.Reader.
type QualifiedNameResolver interface {
	ModuleNameByID(id model.ID) (string, error)
	ResolveQualifiedName(qn string) (id model.ID, kind string, err error)
}
```

- [ ] **Step 3: Interface — `mdl/repos/cache.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repos

import "github.com/mendixlabs/mxcli/model"

// ReaderCache exposes explicit cache invalidation hooks. Per addendum
// Blocker 4, *mmpr.Writer auto-invalidates on InsertUnit and on
// WriteTransaction.Commit, so day-to-day repo code does NOT need to
// call this. The interface is retained for cross-process invalidation
// (another tool wrote to the .mpr / mprcontents directory) and for
// tests that bypass the writer.
type ReaderCache interface {
	Invalidate()
	InvalidateUnit(id model.ID)
}
```

- [ ] **Step 4: Implementation — `mdl/backend/mpr/repos/id.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type idGen struct{}

func NewIDGenerator() repos.IDGenerator { return idGen{} }

func (idGen) NewID() model.ID { return model.ID(mmpr.GenerateID()) }
```

- [ ] **Step 5: Implementation — `mdl/backend/mpr/repos/resolver.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"database/sql"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type sqlResolver struct {
	r  *mmpr.Reader
	db *sql.DB
}

func NewQualifiedNameResolver(w *mmpr.Writer) repos.QualifiedNameResolver {
	r := w.ConcreteReader()
	return &sqlResolver{r: r, db: r.DB()}
}

func (s *sqlResolver) ModuleNameByID(id model.ID) (string, error) {
	// Modules$Module units carry their Name in the BSON Name field.
	// Resolve via Unit table → bytes → bson.Lookup. Implementer should
	// reuse whatever the legacy MprBackend uses for the same lookup
	// (mpr.Reader exposes ListUnitsByType + GetRawUnitBytes).
	// TODO: copy the implementation pattern from
	//       sdk/mpr.Reader.ModuleByID (or equivalent).
	return "", fmt.Errorf("ModuleNameByID: implementer should port from sdk/mpr — id=%s", id)
}

func (s *sqlResolver) ResolveQualifiedName(qn string) (model.ID, string, error) {
	// Same: legacy MprBackend already does this. Port the catalog
	// query unchanged; do not invent a new lookup.
	return "", "", fmt.Errorf("ResolveQualifiedName: implementer should port from legacy backend — qn=%s", qn)
}
```

The two resolver methods are stubbed because the **legacy** implementation is the source of truth — the task list keeps Stage 2 truly additive by NOT touching `MprBackend`, but the implementer MUST locate the equivalent code in `mdl/backend/mpr/` (search for `ModuleByID` / `ResolveQualifiedName`) and translate it into this file. Mark with `TODO Stage 2 finish` in the source if time-boxed.

- [ ] **Step 6: Implementation — `mdl/backend/mpr/repos/cache.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type readerCache struct{ r *mmpr.Reader }

func NewReaderCache(w *mmpr.Writer) repos.ReaderCache {
	return &readerCache{r: w.ConcreteReader()}
}

func (c *readerCache) Invalidate()                       { c.r.InvalidateCache() }
func (c *readerCache) InvalidateUnit(id model.ID)        { c.r.InvalidateCache() } // mmpr.Reader has no per-unit cache; full invalidation is correct
```

- [ ] **Step 7: Tests — `mdl/backend/mpr/repos/aux_test.go`**

Cover:
- `NewIDGenerator().NewID()` returns 36-char UUIDs that round-trip through `mmpr.IDToBsonBinary` / `mmpr.BsonBinaryToID`.
- `NewReaderCache(w).Invalidate()` does not panic and forces the next `ListUnitsByType` to re-read (use `testdata/expr-checker/minimal.mpr` copied into `t.TempDir()`).
- `NewQualifiedNameResolver(w).ResolveQualifiedName("System.User")` returns a non-empty kind once the implementer ports the body. **If the body is still stubbed, mark the test `t.Skip("resolver port pending")` and open a follow-up TODO referenced in the commit message.**

- [ ] **Step 8: Verify and commit**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/repos/... ./mdl/backend/mpr/repos/... -count=1 -timeout 120s
git add mdl/repos/{id,resolver,cache}.go mdl/backend/mpr/repos/{id,resolver,cache,aux_test}.go
git commit -m "feat(repos): IDGenerator, QualifiedNameResolver, ReaderCache (interfaces + MPR impls)"
```

---

## Task 5: TransactionFactory + UnitOfWork

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/uow.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/uow.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/uow_test.go`

This is the trickiest piece in Stage 2. The challenge: a `UnitOfWork` must expose per-domain `Writer` accessors whose `InsertUnit`/`Update` calls are rerouted through the active `*mmpr.WriteTransaction`. We address this by giving each domain Writer implementation a private *write sink* abstraction (`type writeSink interface { ... }`) that has two implementations: the direct-mode `*mmpr.Writer` sink and the txn-mode `*mmpr.WriteTransaction` sink.

- [ ] **Step 1: Interface — `mdl/repos/uow.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repos

// UnitOfWork groups multi-domain writes into a single atomic commit.
// Per addendum Blocker 4, the underlying *mmpr.WriteTransaction commits
// the SQL row changes and renames the temp BSON files atomically; cache
// invalidation is automatic on Commit.
//
// Stage 2 only wires Microflows() and Pages() — the remaining accessors
// are reserved for Stage 3.
type UnitOfWork interface {
	Microflows() MicroflowWriter
	Pages()      PageWriter
	// ... Stage 3: NanoflowWriter, DomainModelWriter, ModuleWriter, etc.
	Commit() error
	Rollback() error
}

type TransactionFactory interface {
	Begin() (UnitOfWork, error)
}
```

- [ ] **Step 2: Implementation — `mdl/backend/mpr/repos/uow.go`**

Sketch:
```go
// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// writeSink hides whether writes go straight to the Writer (direct mode)
// or through an active WriteTransaction (txn mode). Repository
// constructors take a sinkFactory; in direct mode the factory returns a
// fresh writerSink each call, in txn mode it returns the shared
// txnSink for the duration of the UoW.
type writeSink interface {
	InsertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error
	UpdateRawUnit(unitID string, contents []byte) error
	DeleteUnit(unitID string) error
	UpdateUnitContainer(unitID, newContainerID string) error
}

type writerSink struct{ w *mmpr.Writer }

func (s writerSink) InsertUnit(u, c, cn, t string, b []byte) error { return s.w.InsertUnit(u, c, cn, t, b) }
func (s writerSink) UpdateRawUnit(u string, b []byte) error        { return s.w.UpdateRawUnit(u, b) }
func (s writerSink) DeleteUnit(u string) error                     { return s.w.DeleteUnit(u) }
func (s writerSink) UpdateUnitContainer(u, nc string) error        { return s.w.UpdateUnitContainer(u, nc) }

// txnSink defers all writes through WriteTransaction.WriteUnit. Inserts
// degrade to write-then-register-row, but mmpr.WriteTransaction only
// exposes WriteUnit; for Stage 2 we restrict UoW.Microflows()/Pages()
// to **Update** operations and document the limitation. Inserts inside
// a UoW remain a Stage 3 enhancement (it requires a new mmpr API or a
// staged-insert log on the txnSink).
type txnSink struct{ tx *mmpr.WriteTransaction }

func (s txnSink) InsertUnit(string, string, string, string, []byte) error {
	return fmt.Errorf("UnitOfWork: InsertUnit not supported inside a transaction (Stage 2 limitation)")
}
func (s txnSink) UpdateRawUnit(u string, b []byte) error      { return s.tx.WriteUnit(u, b) }
func (s txnSink) DeleteUnit(string) error                     { return fmt.Errorf("UnitOfWork: DeleteUnit not supported inside a transaction (Stage 2)") }
func (s txnSink) UpdateUnitContainer(string, string) error    { return fmt.Errorf("UnitOfWork: UpdateUnitContainer not supported inside a transaction (Stage 2)") }

// txFactory is the concrete TransactionFactory.
type txFactory struct {
	w   *mmpr.Writer
	dec *decoder
	enc *encoder
}

func NewTransactionFactory(w *mmpr.Writer) repos.TransactionFactory {
	return &txFactory{w: w, dec: newDecoder(), enc: newEncoder()}
}

func (f *txFactory) Begin() (repos.UnitOfWork, error) {
	wt, err := f.w.BeginWriteTransaction()
	if err != nil {
		return nil, err
	}
	return &uow{
		mfWriter:   newMicroflowWriterWithSink(f.w, txnSink{tx: wt}, f.enc, f.dec),
		pageWriter: newPageWriterWithSink(f.w, txnSink{tx: wt}, f.enc, f.dec),
		tx:         wt,
	}, nil
}

type uow struct {
	mfWriter   repos.MicroflowWriter
	pageWriter repos.PageWriter
	tx         *mmpr.WriteTransaction
}

func (u *uow) Microflows() repos.MicroflowWriter { return u.mfWriter }
func (u *uow) Pages()      repos.PageWriter      { return u.pageWriter }
func (u *uow) Commit() error                     { return u.tx.Commit() }
func (u *uow) Rollback() error                   { return u.tx.Rollback() }
```

The `newMicroflowWriterWithSink` / `newPageWriterWithSink` constructors live in the implementation files added by Tasks 6 and 7; they accept the sink in addition to the Writer and use the sink for all Insert/Update/Delete plumbing.

`encoder` / `decoder` private types are thin wrappers introduced in Task 6 around `&codec.Encoder{}` and `codec.NewDecoder(codec.DefaultRegistry)` for reuse.

- [ ] **Step 3: Tests — `mdl/backend/mpr/repos/uow_test.go`**

Cover at least:
- `Begin().Commit()` round-trips with **no writes** — proves the factory wires correctly.
- `Begin().Microflows().Update(mf)` followed by `Commit()` — write a microflow, verify the bytes change in the same `mmpr.Reader` after commit (mirror addendum Blocker 4 transactional test).
- `Begin().Microflows().Create(...)` returns the documented "not supported in transaction" error — proves the Stage 2 limitation is enforced explicitly, not silently broken.
- `Rollback()` after an Update leaves the file unchanged.

Use `testdata/expr-checker/minimal.mpr` copied into `t.TempDir()`.

- [ ] **Step 4: Verify and commit**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/repos/... -run TestUoW -v -count=1
git add mdl/repos/uow.go mdl/backend/mpr/repos/uow.go mdl/backend/mpr/repos/uow_test.go
git commit -m "feat(repos): UnitOfWork + TransactionFactory wrapping mmpr.WriteTransaction (Stage 2 update-only)"
```

---

## Task 6: MicroflowRepository implementation

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/microflows.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/microflows_test.go`
- Create (if not already present from Task 5): `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/codec.go` — private `encoder` / `decoder` helpers

- [ ] **Step 1: Codec helpers — `mdl/backend/mpr/repos/codec.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

type encoder struct{ e *codec.Encoder }
type decoder struct{ d *codec.Decoder }

func newEncoder() *encoder { return &encoder{e: &codec.Encoder{}} }
func newDecoder() *decoder { return &decoder{d: codec.NewDecoder(codec.DefaultRegistry)} }

func (e *encoder) Encode(elem element.Element) ([]byte, error) { return e.e.Encode(elem) }

func (d *decoder) Decode(raw []byte) (element.Element, error) {
	return d.d.Decode(bson.Raw(raw))
}
```

- [ ] **Step 2: Implementation — `mdl/backend/mpr/repos/microflows.go`**

Provide `microflowRepo` implementing `repos.MicroflowRepository`. Key methods:
- `NewMicroflowRepository(w *mmpr.Writer) repos.MicroflowRepository` — direct mode, writeSink wraps `w`.
- `newMicroflowWriterWithSink(w, sink, enc, dec) repos.MicroflowWriter` — used by UoW (Task 5).
- `Get(id)`: `bytes, _ := w.ConcreteReader().GetRawUnitBytes(string(id))` → `dec.Decode(bytes)` → cast to `*genMf.Microflow`.
- `List(moduleID)` / `ListAll()`: `refs, _ := w.ConcreteReader().ListUnitsByType("Microflows$Microflow")`; for `List(moduleID)` filter `ref.ContainerID == string(moduleID)` (or whatever the legacy code uses to scope to module — verify against `mdl/backend/mpr/mf_page_modelsdk.go`).
- `FindByQualifiedName(qn)`: split on `.`, resolve module via the catalog, then linear scan.
- `IsRule(qn)`: same lookup but returns true iff the unit's `$Type` is `Microflows$Rule`.
- `Create(parentUUID, containmentName, mf)`:
```go
if mf.ID() == "" {
    mf.SetID(element.ID(mmpr.GenerateID()))
}
if mf.TypeName() == "" {
    mf.SetTypeName("Microflows$Microflow")
}
contents, err := r.enc.Encode(mf)
if err != nil { return err }
return r.sink.InsertUnit(string(mf.ID()), parentUUID, containmentName, mf.TypeName(), contents)
```
- `Update(mf)`: encode, then `sink.UpdateRawUnit(string(mf.ID()), contents)`.
- `Delete(id)`: `sink.DeleteUnit(string(id))`.
- `Move(id, newParentUUID)`: `sink.UpdateUnitContainer(string(id), newParentUUID)`.

Document at the top of the file:
```
// All write paths rely on mmpr.Writer's automatic cache invalidation
// (addendum Blocker 4). Repos do NOT call ReaderCache.Invalidate()
// after each write.
```

- [ ] **Step 3: Tests — `mdl/backend/mpr/repos/microflows_test.go`**

Use the canonical fixture `testdata/expr-checker/minimal.mpr` (16 microflows, per addendum). Test cases:
- `TestMicroflowRepo_ListAll_FixtureCount` — assert `len(ListAll()) == 16` against the known fixture.
- `TestMicroflowRepo_GetRoundTrip` — `Get` first listed microflow, encode, decode, verify the `Name()` matches.
- `TestMicroflowRepo_CreateUpdateDeleteCycle` — create a fresh microflow under the System module's container UUID (look up via ListUnitsByType("Modules$Module") and pick "System"); verify Get sees it; Update its name; Delete it; verify ListAll is back to 16.
- `TestMicroflowRepo_IsRule_NegativeCase` — fixture probably has no rules, so `IsRule("System.SomeMicroflowName")` returns `false, nil`. If the fixture happens to contain one, flip the assertion.
- `TestMicroflowRepo_AutoCacheInvalidation` — Create + immediate List shows count + 1 (regression guard against accidentally re-introducing manual invalidation discipline).

Helper utility (top of test file):
```go
func openTestWriter(t *testing.T) *mmpr.Writer {
    t.Helper()
    src := filepath.Join("..", "..", "..", "..", "testdata", "expr-checker", "minimal.mpr")
    dst := filepath.Join(t.TempDir(), "minimal.mpr")
    data, err := os.ReadFile(src)
    if err != nil { t.Fatalf("read fixture: %v", err) }
    if err := os.WriteFile(dst, data, 0o644); err != nil { t.Fatalf("write tempfile: %v", err) }
    w, err := mmpr.NewWriter(dst)
    if err != nil { t.Fatalf("NewWriter: %v", err) }
    t.Cleanup(func() { _ = w.Close() })
    return w
}
```

- [ ] **Step 4: Run + commit**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/repos/... -run TestMicroflowRepo -v -count=1
git add mdl/backend/mpr/repos/codec.go mdl/backend/mpr/repos/microflows.go mdl/backend/mpr/repos/microflows_test.go
git commit -m "feat(mprrepos): full MicroflowRepository implementation against modelsdk/codec + mmpr.Writer"
```

---

## Task 7: PageRepository implementation + PageMutator (Option A)

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/pages.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/page_mutator.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/repos/pages_test.go`

**Mutator strategy decision: Option A.** Per the planning brief's two-option choice, Stage 2 implements `PageMutator` as a *decode-edit-encode* pipeline backed by the standard `&codec.Encoder{}` — same path as `Update`, just with finer-grained API. Rationale:
1. Closes Stage 2 with one self-contained code path; no new dependency on raw-BSON manipulation primitives.
2. Addendum Blocker 3 measured 1.06× full-vs-incremental, 0.4 ms / 168 KB / 3591 allocs on a 25 KB unit. For pages routinely 100 KB+ this scales linearly — an interactive ALTER is roughly 2–4 ms / page. Acceptable for Stage 2 dogfooding.
3. **Documented follow-up:** Stage 2.5 may introduce *Option B* — direct raw-BSON sub-tree edits via `bson.RawValue` lookups + `codec.bsonbuilder` patches — once the executor's ALTER cadence is measured against Option A. Track in `docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md` Section 11 as `Open Decision (Stage 2.5): page mutator raw-BSON path`.

- [ ] **Step 1: Repository — `mdl/backend/mpr/repos/pages.go`**

Mirror the `microflows.go` shape:
- `NewPageRepository(w *mmpr.Writer) repos.PageRepository`
- `newPageWriterWithSink(w, sink, enc, dec)` for UoW
- `Get`, `List(moduleID)`, `ListAll`, `FindByQualifiedName`
- `Create(parentUUID, containmentName, page)` — set `TypeName` to `"Pages$Page"` if empty; set fresh ID; encode; `sink.InsertUnit`
- `Update(page)` — encode; `sink.UpdateRawUnit`
- `Delete(id)`, `Move(id, newParentUUID)` — same as microflows
- `OpenForMutation(pageID)` — calls `pageMutatorOpen(...)` returning `*pageMutator` (declared in Step 2)

- [ ] **Step 2: Mutator — `mdl/backend/mpr/repos/page_mutator.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// pageMutator is the Stage 2 decode-edit-encode mutator (Option A —
// addendum Blocker 3 marginal-encode trade-off documented in the plan).
//
// Stage 2.5 may swap the body for direct raw-BSON edits if interactive
// ALTER throughput on pages > 100 KB proves insufficient.
type pageMutator struct {
	repo *pageRepo
	page *genPg.Page
}

func pageMutatorOpen(repo *pageRepo, pageID model.ID) (*pageMutator, error) {
	page, err := repo.Get(pageID)
	if err != nil {
		return nil, fmt.Errorf("OpenForMutation(%s): %w", pageID, err)
	}
	return &pageMutator{repo: repo, page: page}, nil
}

func (m *pageMutator) SetWidgetProperty(widgetID model.ID, prop string, value any) error {
	w, err := findWidgetByID(m.page, widgetID)
	if err != nil { return err }
	return setElementScalar(w, prop, value)   // helper: route to the codegen Set<Prop>(...)
}

func (m *pageMutator) InsertWidget(parentID model.ID, slot string, widget element.Element) error {
	parent, err := findWidgetByID(m.page, parentID)
	if err != nil { return err }
	return appendChildToSlot(parent, slot, widget)
}

func (m *pageMutator) DeleteWidget(widgetID model.ID) error  { return removeWidgetByID(m.page, widgetID) }
func (m *pageMutator) ReplaceWidget(widgetID model.ID, replacement element.Element) error {
	return replaceWidgetByID(m.page, widgetID, replacement)
}
func (m *pageMutator) SetLayout(layoutQN string) error {
	m.page.SetLayoutCall(/* element built from layoutQN */ nil)
	return fmt.Errorf("pageMutator.SetLayout: implementer must build a LayoutCall element from %q", layoutQN)
}

func (m *pageMutator) Commit() error { return m.repo.Update(m.page) }
```

The helpers (`findWidgetByID`, `appendChildToSlot`, `removeWidgetByID`, `replaceWidgetByID`, `setElementScalar`) are domain-aware traversals over `genPg.Page`'s widget tree. The implementer should:
1. Locate the page's `LayoutCall.Widgets` (or equivalent) field on `*genPg.Page` via `grep -n "Widgets\(\)\|Widget" modelsdk/gen/pages/types.go`.
2. Recursively visit every `element.Element` checking `e.ID() == widgetID`.
3. For `setElementScalar`, prefer reflection-free dispatch on `prop` ("Caption" → `w.SetCaption(value.(string))`, etc.) for the Stage 2 minimum surface; document unsupported props as TODO.

`SetLayout` is intentionally stubbed because the layout-call construction is non-trivial; mark Skip in tests if not finished by end of Stage 2 and capture as Stage 2.5 follow-up.

- [ ] **Step 3: Tests — `mdl/backend/mpr/repos/pages_test.go`**

Fixture caveat: the canonical `testdata/expr-checker/minimal.mpr` was hand-crafted for microflow testing and **may not contain any pages**. The implementer MUST run at task time:
```bash
ls /mnt/data_sdd/gh/mxcli-wt-02/testdata/
find /mnt/data_sdd/gh/mxcli-wt-02 -name "*.mpr" -not -path "*/node_modules/*" 2>/dev/null
# at time of plan writing this returns:
#   testdata/expr-checker/minimal.mpr
#   sdk/widgets/testdata/crushertestproject/testproject.mpr
```

Strategy:
1. **Try `testdata/expr-checker/minimal.mpr` first.** If `ListUnitsByType("Pages$Page")` returns ≥ 1, use it.
2. **Fall back to `sdk/widgets/testdata/crushertestproject/testproject.mpr`** if it has pages.
3. If **neither** has pages, document the gap, write the tests with `t.Skip("no page fixture available; track as Stage 2 follow-up: add pages to expr-checker fixture or vendor a minimal pages fixture")`, and proceed.

Test cases (run only when a usable fixture exists):
- `TestPageRepo_GetRoundTrip`
- `TestPageRepo_CreateUpdateDeleteCycle` — even if no fixture exists for read, `Create` of a synthetic `genPg.NewPage()` against an empty mpr (or microflow-only fixture inserting under a known module) is testable.
- `TestPageMutator_SetWidgetProperty_RoundTrip` — only runs if a fixture page has at least one widget.
- `TestPageMutator_DeleteWidget_PersistsViaCommit` — same caveat.

- [ ] **Step 4: Run + commit**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/repos/... -run TestPage -v -count=1
git add mdl/backend/mpr/repos/pages.go mdl/backend/mpr/repos/page_mutator.go mdl/backend/mpr/repos/pages_test.go
git commit -m "feat(mprrepos): full PageRepository + PageMutator (Option A: decode-edit-encode)"
```

---

## Task 8: Recording mocks for the two implemented domains

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/testing/microflows.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/testing/pages.go`
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/testing/mocks_test.go`

Pattern from spec Section 8: each method records its arguments in a typed slice and either returns canned data or invokes a `Func` field. Tests inspect the recorded slices.

- [ ] **Step 1: `mdl/repos/testing/microflows.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repostesting

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

type MicroflowCreateCall struct {
	ParentUUID, ContainmentName string
	Microflow                   *genMf.Microflow
}

type MicroflowMoveCall struct {
	ID            model.ID
	NewParentUUID string
}

// RecordingMicroflowRepository records all calls. Reads return zero
// values unless the matching Func is set; writes always succeed unless
// the matching Func returns an error.
type RecordingMicroflowRepository struct {
	GotIDs       []model.ID
	ListedModule []model.ID
	ListedAll    int
	Created      []MicroflowCreateCall
	Updated      []*genMf.Microflow
	Deleted      []model.ID
	Moved        []MicroflowMoveCall

	GetFunc                 func(model.ID) (*genMf.Microflow, error)
	ListFunc                func(model.ID) ([]*genMf.Microflow, error)
	ListAllFunc             func() ([]*genMf.Microflow, error)
	FindByQualifiedNameFunc func(string) (*genMf.Microflow, error)
	IsRuleFunc              func(string) (bool, error)
	CreateFunc              func(MicroflowCreateCall) error
	UpdateFunc              func(*genMf.Microflow) error
	DeleteFunc              func(model.ID) error
	MoveFunc                func(MicroflowMoveCall) error
}

var _ repos.MicroflowRepository = (*RecordingMicroflowRepository)(nil)

// ... method implementations following the pattern. Pseudocode for one:
func (m *RecordingMicroflowRepository) Get(id model.ID) (*genMf.Microflow, error) {
	m.GotIDs = append(m.GotIDs, id)
	if m.GetFunc != nil { return m.GetFunc(id) }
	return nil, nil
}
```

Implement all 9 methods (5 reader + 4 writer).

- [ ] **Step 2: `mdl/repos/testing/pages.go`**

Same pattern, plus:
- `RecordingPageMutator` implementing `repos.PageMutator` — records every SetWidgetProperty/Insert/Delete/Replace/SetLayout/Commit call.
- `RecordingPageRepository` with `OpenForMutationFunc func(model.ID) (repos.PageMutator, error)` so tests can inject a `RecordingPageMutator` of their own.

- [ ] **Step 3: Tests — `mdl/repos/testing/mocks_test.go`**

Trivial smoke tests:
- Instantiate `RecordingMicroflowRepository{}`, call `Create`, verify `len(m.Created) == 1` and the captured args match input.
- Instantiate `RecordingPageRepository{ OpenForMutationFunc: func(...) (repos.PageMutator, error) { return &RecordingPageMutator{}, nil } }`, exercise the mutator, verify both repo and mutator recorded the right calls.
- Compile-time interface satisfaction (the `var _ ... = (*Recording...)(nil)` lines in Step 1 already cover this).

- [ ] **Step 4: Run + commit**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/repos/testing/... -v -count=1
git add mdl/repos/testing/microflows.go mdl/repos/testing/pages.go mdl/repos/testing/mocks_test.go
git commit -m "feat(repostesting): RecordingMicroflowRepository + RecordingPageRepository + mutator"
```

---

## Task 9: ExecutorContext factory

**Files:**
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/repos/context.go` — defines the `ExecutorContext` struct (interfaces only).
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/factory.go` — `NewExecutorContext(*mmpr.Writer)` constructor.
- Create: `/mnt/data_sdd/gh/mxcli-wt-02/mdl/backend/mpr/factory_test.go`

Per spec Section 6 the `ExecutorContext` belongs in `mdl/executor/context.go`. **However**, putting it under `mdl/executor` in Stage 2 would force `mdl/executor` to depend on `mdl/repos` *while* the existing `executor.ExecContext` (Stage 1 codepath) remains untouched. To satisfy the "executor handlers untouched" constraint without an import cycle, Stage 2 ships the new struct under `mdl/repos` (`repos.ExecutorContext`). Stage 3 will move/rename it to `mdl/executor` as part of the cutover. The plan therefore lives at `mdl/repos/context.go` for now.

- [ ] **Step 1: `mdl/repos/context.go`**
```go
// SPDX-License-Identifier: Apache-2.0

package repos

// ExecutorContext aggregates every interface a Stage 3 executor handler
// needs. Each field is an interface; tests inject mocks from
// mdl/repos/testing.
//
// Stage 2 only populates the wired fields (Microflows, Pages, IDs, Tx,
// Names, Cache); the remaining Repository fields will be wired in
// Stage 3 as their implementations come online. Until then they are
// declared so handler code can compile against the final shape.
type ExecutorContext struct {
	Microflows MicroflowRepository
	Pages      PageRepository

	// Stage 3 wiring:
	// Nanoflows    NanoflowRepository
	// Layouts      LayoutRepository
	// Snippets     SnippetRepository
	// DomainModels DomainModelRepository
	// Modules      ModuleRepository
	// Enumerations EnumerationRepository
	// Constants    ConstantRepository
	// Workflows    WorkflowRepository
	// Services     ServiceRepository
	// Mappings     MappingRepository
	// Settings     SettingsRepository
	// Security     SecurityRepository
	// Folders      FolderRepository
	// Images       ImageRepository
	// Agents       AgentRepository

	IDs   IDGenerator
	Tx    TransactionFactory
	Names QualifiedNameResolver
	Cache ReaderCache
}
```

- [ ] **Step 2: `mdl/backend/mpr/factory.go`**
```go
// SPDX-License-Identifier: Apache-2.0

// NOTE: Lives in package mprbackend (same package as MprBackend) so it
// can be reached from existing call sites without a new import. It
// imports the new mprrepos sub-package by its full path.
package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// NewExecutorContext wires a single *mmpr.Writer through every Stage 2
// repository. The returned context is owned by the caller; closing the
// underlying Writer is the caller's responsibility.
//
// Stage 3 will replace this with a path-taking constructor that opens
// the Writer internally, mirroring spec section 7.
func NewExecutorContext(w *mmpr.Writer) *repos.ExecutorContext {
	return &repos.ExecutorContext{
		Microflows: mprrepos.NewMicroflowRepository(w),
		Pages:      mprrepos.NewPageRepository(w),
		IDs:        mprrepos.NewIDGenerator(),
		Tx:         mprrepos.NewTransactionFactory(w),
		Names:      mprrepos.NewQualifiedNameResolver(w),
		Cache:      mprrepos.NewReaderCache(w),
	}
}
```

- [ ] **Step 3: Tests — `mdl/backend/mpr/factory_test.go`**

Cover:
- `NewExecutorContext` returns a non-nil context with every wired field non-nil.
- `ctx.Microflows.ListAll()` works against the fixture (smoke test that wiring did not regress repo behaviour).
- `ctx.Tx.Begin().Rollback()` succeeds.
- Crucially: `grep`-equivalent assertion that **no** existing executor handler imports the new packages — see Task 10 verification.

- [ ] **Step 4: Run + commit**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./mdl/backend/mpr/... -run TestNewExecutorContext -v -count=1
git add mdl/repos/context.go mdl/backend/mpr/factory.go mdl/backend/mpr/factory_test.go
git commit -m "feat(mprbackend): NewExecutorContext factory wiring Stage 2 repos + UoW + auxiliaries"
```

---

## Task 10: Final verification + commit checklist

**Files:** none — verification only.

- [ ] **Step 1: Full build**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go build ./...
```
Expected: clean exit. Stage 2 is additive; build must not break.

- [ ] **Step 2: Full test suite**
```bash
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ~/go1.26/bin/go test ./... -count=1 -timeout 600s
```
Expected: zero new failures vs. main. Pre-existing failures (if any) must be documented in commit message but not fixed in Stage 2.

- [ ] **Step 3: Confirm no executor handler imports the new packages**
```bash
grep -rn "mdl/repos\|mdl/backend/mpr/repos" /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/ 2>/dev/null
```
Expected: NO output. If any handler imports `mdl/repos` or `mprrepos`, Stage 3 contract is violated — revert that import.

- [ ] **Step 4: Confirm legacy code untouched**
```bash
git diff --stat origin/main -- mdl/backend/mpr/backend.go mdl/backend/mock/ mdl/executor/
```
Expected: NO files listed (Stage 2 must be additive). `mdl/backend/microflow.go`, `mdl/backend/page.go`, `mdl/backend/mutation.go` likewise unchanged.

- [ ] **Step 5: Self-review checklist**

- [ ] All 16 domains have an interface file (`ls mdl/repos/*.go | grep -v doc.go | grep -v context.go | grep -v page_mutator.go | grep -v workflow_mutator.go | grep -v id.go | grep -v resolver.go | grep -v cache.go | grep -v uow.go` returns 16 lines: microflows, nanoflows, pages, layouts, snippets, domainmodels, modules, enumerations, constants, workflows, services, mappings, settings, security, folders, images, agents — and verify the count is 16 against the actual `modelsdk/gen/` list)
- [ ] `MicroflowRepository` + `PageRepository` have full implementations with passing tests against `testdata/expr-checker/minimal.mpr`
- [ ] `PageMutator` defines at least 5 methods (SetWidgetProperty, InsertWidget, DeleteWidget, ReplaceWidget, SetLayout, Commit — that's 6) and has a working Stage 2 implementation (Option A) with tests (or documented Skip)
- [ ] Recording mocks exist for `MicroflowRepository` and `PageRepository` (and `PageMutator`)
- [ ] `NewExecutorContext` factory wires `IDs`, `Tx`, `Names`, `Cache` plus `Microflows` and `Pages`
- [ ] No task in this plan instructed the implementer to modify `mdl/backend/microflow.go`, `mdl/backend/page.go`, `mdl/backend/mutation.go`, `mdl/backend/mpr/backend.go`, the existing `*ViaModelsdk*` files, `mdl/backend/mock/`, or any file under `mdl/executor/`
- [ ] Every task ends with a commit message
- [ ] Verification task ran the full test suite with zero new failures expected
- [ ] Stage 2.5 follow-up (Option B raw-BSON page mutator) captured in the commit message of Task 7 and worth filing as an issue / spec entry

- [ ] **Step 6: Optional smoke commit**
If anything in Step 5 surfaced a fix, commit it now. Otherwise no commit; the per-task commits already cover the work.

---

## Out of scope (explicit)

- Stage 3 cutover (replace `Backend` field, update every handler, remove `*ViaModelsdk*`, replace `MockBackend` callsites) — separate plan.
- Stage 4 deletion of `sdk/microflows`, `sdk/pages`, `sdk/mpr.SerializeXxx` etc. — separate plan.
- Filling out the 14 stub repositories with full implementations and Recording mocks — Stage 3.
- Wiring UoW.Insert/Delete inside transactions (mmpr API gap; documented limitation in Task 5).
- Direct raw-BSON page mutator (Option B) — Stage 2.5 follow-up.
- Move of `ExecutorContext` from `mdl/repos` to `mdl/executor` — Stage 3.
