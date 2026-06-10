# Architecture Doc Rewrite — Design Spec

**Date:** 2026-06-10  
**Target file:** `docs/01-project/ARCHITECTURE.md`  
**Goal:** Replace the existing component-catalogue architecture doc with a "philosophy-first" document that helps Go+Mendix engineers build the right mental model in 30 minutes.

---

## Background

The existing `ARCHITECTURE.md` is a comprehensive component catalogue with Mermaid diagrams per subsystem. It answers "what exists" but not "why it was designed this way" or "where do I start when I want to do X". New engineers can read it top-to-bottom and still not know which package to open first or why the executor is forbidden from importing `sdk/mpr`.

The five design principles are scattered across `CLAUDE.md`, individual skill files, and design specs. The entry map ("what do I touch to add a command") does not exist anywhere. The invariants ("what is never allowed") are enforced by tests but not explained in one place.

---

## Design

### Structure

The new `ARCHITECTURE.md` follows **Approach A (philosophy-first)**:

```
§1  One-line positioning
§2  Five design principles  (the WHY)
§3  Six-layer diagram       (the WHAT)
§4  Entry map               (the WHERE)
§5  Key invariants          (the NEVER)
§6  Deep-reading guide      (the NEXT)
```

Each section scales to its complexity. §2 and §3 are the load-bearing sections; the rest are reference.

---

### §1 — One-line positioning

> mxcli is a Go-native CLI tool that lets AI agents and developers read and write Mendix project files (`.mpr`) offline via a SQL-style DSL (MDL), without requiring Studio Pro to be running.

---

### §2 — Five design principles

Each principle is stated as a rule, followed by a counter-example (the failure mode it prevents) and how the current code expresses it.

| # | Principle | Counter-example prevented |
|---|-----------|--------------------------|
| 1 | **AI-Code Co-design** — Every error must be actionable; every path must have a discovery trail; every new feature must update its skill file. | Error routed to bare `os.Stderr` → AI agent in daemon mode never sees it → silent failure. |
| 2 | **Backend abstraction + dependency inversion** — Executor describes *what* to do; `mdl/backend/mpr` knows *how* to do it in BSON. Executor never imports `sdk/mpr` or `modelsdk/codec`. | Executor with inline BSON → unit tests require real `.mpr` files → test suite slows from seconds to minutes, failure injection becomes impossible. |
| 3 | **Thin executor, fat backend** — Handler is a three-line dispatcher (parse AST → call `ctx.Backend.*` → format output). All business complexity lives in the backend. | Historical `buildDataGridDataSourceBSON` duplicated in executor alongside `buildDatasourceV3` → two implementations of the same datasource type diverged over time. |
| 4 | **Evidence-first TDD** — Write the minimal failing test first, trace the root cause by reading code, implement the minimal fix, add a regression guard. | "Fixed A, broke B" regressions in releases where guessed fixes were applied without a failing test as anchor. |
| 5 | **Pure Go / no CGO** — `modernc.org/sqlite` replaces the CGO SQLite driver; no C compiler required for cross-compilation. | CGO SQLite on devcontainer + CI fails when C toolchain versions diverge; Mendix user environments are even less controlled than developer machines. |

---

### §3 — Six-layer diagram

```
┌─────────────────────────────────────────────────────────────┐
│  Interface Layer  cmd/mxcli  (Cobra CLI / REPL / LSP / daemon)│
│  Only entry point for users and AI agents.                   │
│  Routes commands; contains zero business logic.              │
└────────────────────────┬────────────────────────────────────┘
                         │ MDL text
┌────────────────────────▼────────────────────────────────────┐
│  Language Layer  mdl/grammar · mdl/ast · mdl/visitor         │
│  ANTLR4 lex/parse → AST.                                     │
│  MDL is the stable contract; backend implementations can swap.│
└────────────────────────┬────────────────────────────────────┘
                         │ AST node
┌────────────────────────▼────────────────────────────────────┐
│  Execution Layer  mdl/executor                               │
│  Thin dispatcher: AST → ctx.Backend.* → formatted output.   │
│  FORBIDDEN: import sdk/mpr · modelsdk/codec                  │
└──────────┬─────────────────────────────────────┬────────────┘
           │ FullBackend interface                │ injected in tests
┌──────────▼──────────┐              ┌────────────▼───────────┐
│  Storage Impl       │              │  Mock Impl             │
│  mdl/backend/mpr    │              │  mdl/backend/mock      │
│  All BSON read/write│              │  Func-field injection, │
│  mx-check compliant │              │  no .mpr file needed   │
└──────────┬──────────┘              └────────────────────────┘
           │ modelsdk reader/writer
┌──────────▼──────────────────────────────────────────────────┐
│  File Layer  modelsdk/mpr · modelsdk/widgets                 │
│  MPR v1 (SQLite) / v2 (mprcontents/) transparent read/write. │
│  BSON ↔ Go struct; widget template clone + augment.          │
└──────────┬──────────────────────────────────────────────────┘
           │
┌──────────▼──────────────────────────────────────────────────┐
│  Disk Layer  .mpr / mprcontents/ / external DBs              │
└─────────────────────────────────────────────────────────────┘
```

**Two cross-cutting concerns (serve all layers):**
- `mdl/catalog` — SQLite metadata index; powers `show callers`, full-text search, Starlark lint
- `mdl/types` — shared domain types used by both executor and backend; prevents circular imports

---

### §4 — Entry map

Organized by "what do I want to do", not by package.

| Task | Touch order |
|------|-------------|
| Add a new MDL command (e.g. `alter image collection`) | `MDLParser.g4` → `make grammar` → `mdl/ast/` → `mdl/visitor/` → `mdl/executor/cmd_*.go` → `mdl/backend/` interface → `mdl/backend/mpr/` impl → `mdl/backend/mock/` stub → tests |
| Fix a BSON write bug | `mdl/backend/mpr/` locate write path → compare against fixture BSON (`internal/goldenfs/`) → write failing test → implement fix → `mx check` validation |
| Add pluggable widget support | `modelsdk/widgets/definitions/*.def.json` → `mdl/executor/widget_engine.go` operation registry → `modelsdk/widgets/templates/` template → `modelsdk/widgets/augment.go` |
| Change MDL syntax | `.g4` file → `make grammar` (commit generated `mdl/grammar/parser/` files alongside `.g4`) → visitor → ast |
| Write executor unit test | `mdl/backend/mock/` injection → `mdl/executor/*_test.go` |
| Write BSON correctness test | `internal/goldenfs/` golden snapshot → helpdesk regression test |
| Gate a feature by Mendix version | `sdk/versions/mendix-*.yaml` entry → executor `checkFeature()` pre-check |

---

### §5 — Key invariants

Hard rules enforced by tests or CI. No exceptions.

| Invariant | Why | How enforced |
|-----------|-----|--------------|
| Executor must not import `sdk/mpr` or `modelsdk/codec` | Breaks dependency inversion; makes unit tests require real `.mpr` | `TestNoDirectBSONImportInExecutor` |
| Executor must not contain raw BSON type strings (e.g. `"Forms$..."`, `"CustomWidgets$..."`) | Bypasses type system; silently fails on version upgrades | `TestNoRawBSONTypeStringsInExecutor` |
| Every new backend method needs a Func-field stub in `mock/` | Without stub, mock returns `nil, nil` silently; tests miss error paths | Compile-time `var _ backend.X = (*impl)(nil)` |
| Map iteration for serialised output must sort keys first | Non-deterministic output → flaky BSON diffs → flaky golden tests | Golden regression suite |
| Errors must route through `cmd.ErrOrStderr()` to the socket | Bare `os.Stderr` is invisible to AI agents in daemon mode | Code review |
| New shared types belong in `mdl/types/`, not defined in `mdl/backend/mpr/` | Prevents circular imports; keeps types at a single source of truth | Compiler import graph |
| BSON arrays: index 0 is an `int32` version prefix; real entries start at index 1 | Missing prefix → CE0003 and similar Studio Pro errors | `mx check` + golden |

---

### §6 — Deep-reading guide

| Goal | Read here |
|------|-----------|
| Full MDL syntax | `docs/01-project/MDL_QUICK_REFERENCE.md` |
| Feature completeness across all dimensions | `docs/01-project/MDL_FEATURE_MATRIX.md` |
| Page/Widget BSON serialisation details | `docs/03-development/PAGE_BSON_SERIALIZATION.md` |
| ANTLR4 parser architecture | `docs/03-development/MDL_PARSER_ARCHITECTURE.md` |
| Test layering strategy (L1/L2/L3/L6b) | `docs/03-development/TESTING_GUIDE.md` |
| BSON bug debugging workflow | `.claude/skills/debug-bson.md` |
| Executor handler conventions | `.claude/skills/` domain skill files |
| Strategic positioning (vs PED) | `docs/01-project/MXCLI_STRATEGIC_POSITIONING.md` |
| Recent design decision history | `docs/superpowers/specs/` sorted by date |

---

## Implementation Notes

- The new `ARCHITECTURE.md` replaces the existing file entirely. The existing content (detailed component tables, per-subsystem Mermaid diagrams, sequence diagrams) is useful but belongs in a separate technical reference or inline in the subsystem's own package doc.
- Keep Mermaid only for the six-layer diagram (§3); prose tables for everything else — they age better.
- The principles in §2 should be written in English (the existing file is in English; consistency matters for a reference doc read by international contributors).
- Total target length: ~500–700 lines. Tighter than the current ~900-line file despite covering more ground, because we drop the per-component deep-dives.
- After writing, verify: does the doc answer "why is the executor forbidden from importing sdk/mpr?" in under 10 seconds of scanning? If not, §2 or §5 needs adjustment.
