# mxcli — Architecture

> This document is **philosophy-first**. It is meant to be read top-to-bottom in
> about 30 minutes by a Go engineer who already understands Mendix at the
> "I've opened Studio Pro" level. When you finish, you should know *why* the
> codebase is shaped the way it is, *where* to start for any given task, and
> *what* you are never allowed to do.
>
> For exhaustive component catalogues, per-subsystem diagrams, and BSON field
> tables, follow the pointers in [§6 Deep-reading guide](#6--deep-reading-guide-the-next).
> Those details age quickly and live next to the code they describe; this file
> stays small on purpose.

---

## §1 — One-line positioning

> **mxcli is a Go-native CLI tool that lets AI agents and developers read and
> write Mendix project files (`.mpr`) offline via a SQL-style DSL (MDL), without
> requiring Studio Pro to be running.**

Three words in that sentence carry the whole design:

- **Offline** — we parse and rewrite the on-disk `.mpr` storage format directly.
  There is no Mendix runtime, no cloud call, no Studio Pro process. The
  correctness bar is "Studio Pro opens the file we wrote without a
  `TypeCacheUnknownTypeException` or `CExxxx` consistency error."
- **AI agents** — the primary caller is not a human at a terminal; it is an LLM
  agent driving the tool in a loop. This is not a nice-to-have framing — it is a
  hard design constraint that shapes error handling, discovery trails, and
  documentation (see [§2 Principle 1](#1--ai-code-co-design)).
- **SQL-style DSL (MDL)** — users never touch BSON. They write English-like
  statements (`create entity`, `alter page`, `show callers of`), and the tool
  translates them to and from Mendix's BSON document format.

---

## §2 — Five design principles (the WHY)

Everything below this section is a consequence of these five rules. Each
principle is stated as a rule, followed by the failure mode it prevents
(the counter-example) and how the current code expresses it. If you only read
one section of this document, read this one.

### 1 — AI-Code Co-design

> **Every error must be actionable; every path must have a discovery trail;
> every new feature must update its skill file.**

mxcli's success criterion is *dual*: the program must execute correctly **and**
the AI agent driving it must be able to operate it correctly. These are not two
separate goals — they are one co-design requirement. A human operator can read a
manual, guess from experience, and ask a colleague when stuck. An AI agent
cannot. It needs each failure point to carry enough context to self-correct, and
it needs the guidance chain to be unbroken: **initial bootstrap → discovery path
→ actionable error.**

| | |
|---|---|
| **Counter-example prevented** | An error routed to bare `os.Stderr` (or worse, `os.Exit`). In daemon mode the agent's command runs over a socket; anything not routed through `cmd.ErrOrStderr()` lands in `/dev/null`. The agent sees an empty result, assumes success or retries blindly, and the failure is silent. |
| **How the code expresses it** | Errors flow through `cmd.ErrOrStderr()` so they reach the socket (`cmd/mxcli/cmd_exec.go`, `cmd/mxcli/cmd_query.go`). Error messages state *why* it failed and *how* to fix it, not just "not found." Command output embeds the next command to run (e.g. `widget list` prints the MDL syntax to use the result). New features ship with a matching skill file under `.claude/skills/`. |

The three layers this principle demands — **initial bootstrap** (does the agent
have the right guidance before it starts?), **discovery path** (can it find what
it needs without a human?), and **actionable error** (does a failure explain
itself and point to the next step?) — are spelled out in full in the
"AI-Code Co-design Principle" section of `CLAUDE.md`. Treat that section as the
authoritative checklist when adding any feature.

### 2 — Backend abstraction + dependency inversion

> **The executor describes *what* to do; `mdl/backend/mpr` knows *how* to do it
> in BSON. The executor never imports `sdk/mpr` or `modelsdk/codec`.**

The executor depends on an *interface* (`backend.FullBackend`), never on a
concrete storage implementation. The concrete BSON logic depends on that same
interface from below. This is classic dependency inversion: both sides point at
the abstraction in the middle.

| | |
|---|---|
| **Counter-example prevented** | An executor handler that builds BSON inline. The moment it does, every unit test for that handler needs a real `.mpr` file on disk to exercise the code path. The test suite drifts from seconds to minutes, and — worse — you can no longer inject failures (disk errors, malformed documents, version mismatches) because there is no seam to inject them at. |
| **How the code expresses it** | The executor talks only to `ctx.Backend.*`. A compile-time guard (`var _ backend.FullBackend = (*impl)(nil)`) keeps the MPR implementation honest, and an import-graph test (`TestNoDirectBSONImportInExecutor` in `mdl/executor/import_guard_test.go`) fails the build if any executor file imports a BSON package. In tests, `mdl/backend/mock` is injected instead of the MPR backend — no `.mpr` file required. |

### 3 — Thin executor, fat backend

> **A handler is a three-line dispatcher: parse the AST node → call
> `ctx.Backend.*` → format the output. All business complexity lives in the
> backend.**

This is the operational reading of Principle 2. The interface seam only pays off
if the executor side of it stays trivial. When a handler grows logic — BSON
construction, version branching, ID remapping — that logic belongs on the other
side of the interface, in `mdl/backend/mpr`.

| | |
|---|---|
| **Counter-example prevented** | The historical `buildDataGridDataSourceBSON` lived in the executor alongside `buildDatasourceV3` from the Forms path. Two implementations of the same datasource concept sat in two places and drifted apart; a fix to one silently left the other wrong. Pulling the shared logic behind one backend helper (the `buildNanoflowSourceGen` pattern) collapsed them back into a single source of truth. |
| **How the code expresses it** | Handlers in `mdl/executor/cmd_*.go` are short. ALTER-style mutations go through a mutator the backend hands back (`ctx.Backend.OpenPageForMutation(id)`, `OpenWorkflowForMutation(id)`); the executor calls `SetWidgetProperty` / `InsertWidget` and then `Save()`, but never constructs the document. Pluggable widgets go through the `WidgetRegistry` + `.def.json` engine, not hand-written BSON builders in the handler. |

### 4 — Evidence-first TDD

> **Write the minimal failing test first. Trace the root cause by reading code
> and git history. Implement the minimal fix. Add a regression guard. Show
> evidence at each step before moving on.**

No claim about a code path is made until a test or a read file proves it. No fix
is written until a failing test pins the hypothesis.

| | |
|---|---|
| **Counter-example prevented** | "Fixed A, broke B" regressions — the classic outcome of a guessed fix applied without a failing test as an anchor. Without the test, you cannot tell whether you fixed the bug or merely moved it, and the next release re-opens it. |
| **How the code expresses it** | Bug fixes start with a failing test — a backend mutation test in `mdl/backend/mpr/` or an executor handler test in `mdl/executor/` using `MockBackend`. BSON-level correctness is pinned by golden snapshots in `internal/goldenfs/` (e.g. `helpdesk_regression_test.go`). The PR checklist in `CLAUDE.md` requires the failing test to exist *before* the implementation. |

### 5 — Pure Go / no CGO

> **`modernc.org/sqlite` replaces the CGO SQLite driver. No C compiler is
> required to build or cross-compile mxcli.**

| | |
|---|---|
| **Counter-example prevented** | A CGO SQLite dependency. It compiles on the developer's machine and breaks in the devcontainer or in CI when the C toolchain versions diverge — and Mendix end-user environments are far less controlled than either. CGO also makes cross-compilation (Windows / macOS / Linux binaries from one host) painful. |
| **How the code expresses it** | The project depends on `modernc.org/sqlite` (pure Go) everywhere SQLite is used: MPR v1 reading, the catalog index, the lint cache. `make build` cross-compiles all three release binaries (`mxcli`, `mxcli-daemon`, `mxcli-local`) without a C compiler. |

---

## §3 — Six-layer diagram (the WHAT)

Data flows top-to-bottom: MDL text enters at the interface layer and ends as
bytes on disk. The two horizontal seams that matter most are between the
**Execution Layer** and the **Storage Impl** (the `FullBackend` interface — the
subject of Principles 2 and 3), and between the **File Layer** and the
**Disk Layer** (the MPR v1/v2 format boundary).

```
┌─────────────────────────────────────────────────────────────┐
│  Interface Layer   cmd/mxcli  (Cobra CLI / REPL / LSP /       │
│                               daemon)                         │
│  The only entry point for users and AI agents.               │
│  Routes commands. Contains zero business logic.              │
└────────────────────────┬────────────────────────────────────┘
                         │  MDL text
┌────────────────────────▼────────────────────────────────────┐
│  Language Layer    mdl/grammar · mdl/ast · mdl/visitor        │
│  ANTLR4 lex / parse  →  AST.                                  │
│  MDL is the stable contract; the backend below can be         │
│  swapped without changing the language.                       │
└────────────────────────┬────────────────────────────────────┘
                         │  AST node
┌────────────────────────▼────────────────────────────────────┐
│  Execution Layer   mdl/executor                               │
│  Thin dispatcher:  AST → ctx.Backend.* → formatted output.   │
│  FORBIDDEN:  import sdk/mpr · import modelsdk/codec           │
│              raw BSON type strings ("Forms$...")             │
└──────────┬─────────────────────────────────────┬────────────┘
           │  backend.FullBackend interface        │  injected in tests
┌──────────▼──────────┐               ┌───────────▼────────────┐
│  Storage Impl       │               │  Mock Impl             │
│  mdl/backend/mpr    │               │  mdl/backend/mock      │
│  All BSON read +    │               │  Func-field injection. │
│  write. mx-check    │               │  No .mpr file needed.  │
│  compliant.         │               │                        │
└──────────┬──────────┘               └────────────────────────┘
           │  modelsdk reader / writer
┌──────────▼──────────────────────────────────────────────────┐
│  File Layer    modelsdk/mpr · modelsdk/widgets               │
│  MPR v1 (SQLite) / v2 (mprcontents/) transparent read+write. │
│  BSON ↔ Go struct.  Widget template clone + augment.         │
└──────────┬──────────────────────────────────────────────────┘
           │
┌──────────▼──────────────────────────────────────────────────┐
│  Disk Layer    .mpr  /  mprcontents/  /  external DBs        │
└─────────────────────────────────────────────────────────────┘
```

### Two cross-cutting concerns

These do not sit in the vertical stack — they serve every layer:

- **`mdl/catalog`** — a SQLite metadata index built from the project. It powers
  `show callers` / `show callees` / `show references`, full-text `search`, and
  the Starlark-based lint rules. It is a *derived read model*: rebuilt from the
  `.mpr` on `refresh catalog`, never the source of truth.
- **`mdl/types`** — domain types shared by both the executor and the backend
  (e.g. `NavigationDocument`, `JavaAction`, `JsonStructure`, EDMX/AsyncAPI
  parse results, ID utilities). Putting them here, with the backend re-exporting
  type aliases (`type Foo = types.Foo`), is what prevents a circular import
  between the execution layer and the storage layer. A second small helper,
  `mdl/bsonutil`, holds CGO-free BSON ID conversions used on both sides.

### Why MDL is the contract, not the backend

The Language Layer is deliberately decoupled from storage. MDL — the statements a
user or agent writes — is the stable public surface. The backend that satisfies
`FullBackend` is an implementation detail. Today there is one real backend
(`mdl/backend/mpr`) and one test backend (`mdl/backend/mock`), but the seam is
real: an in-memory or cloud-hosted backend could be added without touching the
grammar, the AST, or a single executor handler.

---

## §4 — Entry map (the WHERE)

Organized by "what do I want to do," not by package. Follow the touch order
left-to-right; each arrow is a file or step.

| Task | Touch order |
|------|-------------|
| **Add a new MDL command** (e.g. `alter image collection`) | `mdl/grammar/MDLParser.g4` → `make grammar` (commit the regenerated `mdl/grammar/parser/` files) → `mdl/ast/` (node type) → `mdl/visitor/` (build AST from parse tree) → `mdl/executor/cmd_*.go` (thin handler) → `mdl/backend/` (interface method) → `mdl/backend/mpr/` (BSON impl) → `mdl/backend/mock/` (Func-field stub) → tests |
| **Fix a BSON write bug** | `mdl/backend/mpr/` (locate the write path) → compare against a fixture in `internal/goldenfs/` → write the failing test first → implement the minimal fix → `mx check` to confirm Studio Pro acceptance → add a regression guard |
| **Add pluggable widget support** | `modelsdk/widgets/definitions/*.def.json` (or `mxcli widget extract` to scaffold one from a `.mpk`) → `mdl/executor/widget_engine.go` (operation registry) → `modelsdk/widgets/templates/` (template) → `modelsdk/widgets/augment.go` (reconcile against the project's `.mpk`) |
| **Change existing MDL syntax** | `mdl/grammar/*.g4` → `make grammar` (commit regenerated parser alongside the `.g4` change) → `mdl/visitor/` → `mdl/ast/` |
| **Write an executor unit test** | `mdl/backend/mock/` (set the relevant `*Func` field) → `mdl/executor/*_test.go` — no `.mpr` file involved |
| **Write a BSON correctness test** | `internal/goldenfs/` (golden snapshot) → helpdesk regression test (`helpdesk_regression_test.go`) |
| **Gate a feature by Mendix version** | `sdk/versions/mendix-{9,10,11}.yaml` (registry entry with `min_version`) → `mdl/executor` `checkFeature()` pre-check (actionable error + hint before any BSON write) |
| **Add an external-SQL capability** | `sql/` (driver / connection / query / import / generate) — this subsystem is reached from the executor but lives outside the backend interface; it talks to PostgreSQL / Oracle / SQL Server, not to `.mpr` |
| **Add an editor / IDE-assist feature** | `mdl/executor/autocomplete*.go` (completion logic) → `cmd/mxcli/serve.go` / `cmd/mxcli/cmd_expr_daemon.go` (the daemon that hosts editor requests over a socket) — all of it reuses the same parser, AST, and catalog |

> **Tip:** the most common mistake is putting logic in the wrong column. If a
> step in your change involves constructing or walking BSON, it belongs in
> `mdl/backend/mpr/`, never in `mdl/executor/`. See [§2 Principle 3](#3--thin-executor-fat-backend).

---

## §5 — Key invariants (the NEVER)

Hard rules enforced by tests or CI. There are no exceptions; a PR that violates
one of these does not merge. Where the enforcement is "code review" rather than a
test, treat it as no less binding — it simply has not been automated yet.

| Invariant | Why | How enforced |
|-----------|-----|--------------|
| The executor must not import `sdk/mpr` or `modelsdk/codec`. | Breaks dependency inversion; forces unit tests to require a real `.mpr` and removes the seam for failure injection. | `TestNoDirectBSONImportInExecutor` (`mdl/executor/import_guard_test.go`) |
| The executor must not contain raw BSON type strings (e.g. `"Forms$..."`, `"CustomWidgets$..."`). | Bypasses the type system; such literals silently break on a Mendix version upgrade where the storage name changes. | `TestNoRawBSONTypeStringsInExecutor` (`mdl/executor/import_guard_test.go`) |
| Every new backend method needs a `Func`-field stub in `mdl/backend/mock/`. | Without a stub the mock returns the zero value silently, and tests miss the error path entirely. The stub's default must be a descriptive `"MockBackend.X not configured"` error, never `nil, nil`. | Compile-time `var _ backend.X = (*impl)(nil)` on both real and mock impls |
| Any map iterated to produce serialized output must sort its keys first. | Non-deterministic iteration order → non-deterministic BSON → flaky golden diffs and unstable `.mpr` output. | Golden regression suite (`internal/goldenfs/`) |
| Errors must route through `cmd.ErrOrStderr()` to the socket. No `os.Exit`, no bare `fmt.Fprintf(os.Stderr, ...)`. | In daemon mode a bare stderr write is invisible to the AI agent driving the tool — the canonical silent failure (see [§2 Principle 1](#1--ai-code-co-design)). | Code review |
| New shared types belong in `mdl/types/`, not defined inside `mdl/backend/mpr/`. | Keeps a single source of truth and prevents a circular import between the execution and storage layers. | Compiler import graph + code review |
| In a Mendix BSON array, index 0 is an `int32` version prefix; real entries start at index 1. | A missing prefix produces `CE0003` and similar Studio Pro consistency errors when the file is opened. Applies to `AccessRules`, `MemberAccesses`, `Entities`, `AllowedModuleRoles`, and friends. | `mx check` + golden snapshots |

---

## §6 — Deep-reading guide (the NEXT)

This document intentionally stops at the mental-model level. When you need the
exhaustive detail, go here:

| Goal | Read here |
|------|-----------|
| Full MDL syntax (every statement, every clause) | `docs/01-project/MDL_QUICK_REFERENCE.md` |
| Feature completeness across all dimensions | `docs/01-project/MDL_FEATURE_MATRIX.md` |
| Page / widget BSON serialization details, type mappings, required defaults | `docs/03-development/PAGE_BSON_SERIALIZATION.md` |
| ANTLR4 parser architecture and the `make grammar` workflow | `docs/03-development/MDL_PARSER_ARCHITECTURE.md` |
| Test layering strategy (L1 / L2 / L3 / L6b) and the PR test checklist | `docs/03-development/TESTING_GUIDE.md` |
| BSON bug debugging workflow with the `mx` tool | `.claude/skills/debug-bson.md` |
| Executor handler conventions, per-domain gotchas, MDL idioms | `.claude/skills/` (domain skill files) |
| Strategic positioning (why mxcli exists, vs. alternatives) | `docs/01-project/MXCLI_STRATEGIC_POSITIONING.md` |
| Recent design-decision history | `docs/superpowers/specs/`, sorted by date |
| The AI-Code Co-design contract in full (three-layer checklist) | the "AI-Code Co-design Principle" section of `CLAUDE.md` |
| Repository layout, build/test commands, BSON storage-name gotchas | `CLAUDE.md` (top-level project guidance) |

---

### A note on what this document is not

This file used to be a component catalogue: a Mermaid diagram per subsystem, a
table per package, sequence diagrams for read and write flows. That content was
accurate but answered "what exists" rather than "why, and where do I start." The
per-component detail now lives next to the code it describes (package docs and
the deep-reading targets above), where it can be kept current by the people
touching that code.

If, after reading this, you cannot answer **"why is the executor forbidden from
importing `sdk/mpr`?"** in under ten seconds — re-read [§2 Principle 2](#2--backend-abstraction--dependency-inversion)
and [§5](#5--key-invariants-the-never). That question is the litmus test for
whether this document is doing its job.
