# Proposal: Project Brain — Persistent Knowledge and Session Scaffolding for Long-Term AI Collaboration

## Problem Statement

mxcli is developed with significant AI involvement, yet the project's knowledge infrastructure is built for static human reading rather than persistent, connected memory. Three related friction points compound over time:

1. **Documentation goes stale without detection** — Proposals describe features as future work after they have already shipped (e.g., the registry implementation predating its own proposal by weeks). There is no mechanism to detect or prevent this drift.

2. **No orientation layer for new contributors or agent sessions** — Skills files cover specific tasks well, but there is no single place that answers: *what is the current state of the project, what is actively being worked on, and what is the path for adding something new end-to-end?* A new human contributor or an AI agent dropped into a fresh session must reconstruct this from scattered files, CLAUDE.md, and git history.

3. **Knowledge is not connected** — The internals documentation in `docs-site/src/internals/` covers the pipeline and key subsystems, but the pages are isolated documents rather than a traversable network. There is no way to navigate from "executor" to "backend interface" to "MPR backend" as a linked path, and no graph view for humans to explore topics.

These are not separate problems. They share the same root: the project has grown to a point where implicit conventions and scattered documents work for a small team in a single session but break down for parallel development, onboarding, and long-running AI collaboration across many sessions.

## Current State

The project already has several knowledge layers that are good individually but unconnected:

| Layer | Location | Audience | Character |
|---|---|---|---|
| Kitchen-sink AI context | `CLAUDE.md` | AI agents | Procedural, comprehensive, monolithic |
| Maintainer task skills | `.claude/skills/*.md` | AI agents (mxcli-dev) | Procedural ("how to do X") |
| User task skills | `.claude/skills/mendix/*.md` | AI agents (shipped to users) | Procedural ("how to do X") |
| Internals docs | `docs-site/src/internals/` | Curious users + contributors | Explanatory, isolated pages |
| Proposals | `docs/11-proposals/` | Contributors | Forward-looking, no lifecycle |
| Architecture docs | `docs/01-project/`, `docs/03-development/` | Contributors | Structural, sparse |

The gaps are: links between topics (graph), decision lifecycle (why things are the way they are), extension guides (how to add new things end-to-end), a raw material inbox, and a live "what's next" layer connected to GitHub data.

## Proposed System: Four Connected Layers

### Layer 1: Raw Inbox (`docs/raw/`)

An immutable drop zone for unprocessed source material: GitHub issue exports, research notes, discussion transcripts, external references. Files land here and are never edited after landing. The `/brain-ingest` command processes them into the wiki and marks each file as processed via frontmatter. The raw material remains as provenance.

This separates input from synthesised knowledge cleanly. An agent or contributor drops a file here; the brain processes it on the next ingest run.

### Layer 2: The Wiki (`wiki/`)

A contributor-facing wiki at the repo root. The character is **explanatory** — how things work and why — distinct from skills (procedural) and proposals (decisions in flight). One concept per file, explicit links between topics, with a Mermaid index in `README.md` that renders as a graph.

Both humans and agents write to the wiki. The agent drafts and updates via commands; humans edit directly. The wiki is not agent-owned — it is collaboratively maintained, with the agent doing the bulk of routine updates.

The `docs-site/src/internals/` pages are good raw material; overlapping topics link to the wiki or migrate into it. The distinction is audience: docs-site explains internals to curious users, the wiki explains them to contributors who need to change things.

**Structure:**

```
wiki/
  README.md               # visual index: Mermaid graph of clusters + entry points
  log.md                  # chronological record of all wiki operations
  MAP.md                  # source path → wiki topic manifest (drives freshness hooks)

  pipeline/               # the journey from MDL text to MPR write
    overview.md           # full pipeline diagram, links to each step
    grammar.md            # ANTLR4, domain split, make grammar
    ast.md                # statement node hierarchy, how kinds map to types
    visitor.md            # ANTLR listener → AST construction
    executor.md           # registry, dispatch, ExecContext, register_stubs.go
    backend.md            # hexagonal boundary, interface design, why it exists
    mpr.md                # BSON, reader/writer, storage names, v1 vs v2

  design/                 # why things are the way they are (ADRs)
    mdl-syntax-principles.md      # design guidelines, anti-patterns, decision framework
    backend-boundary.md           # no sdk/mpr in executor, why enforced by checklist
    registry-explicit-wiring.md   # why init() was rejected, emptyRegistry() testing
    grammar-domain-split.md       # how the domain split reduces conflict surface

  subsystems/             # key components in isolation
    catalog.md            # SQLite catalog, what it indexes, query interface
    lsp.md                # LSP capabilities, how diagnostics flow, wiring
    widget-engine.md      # widget registry, .def.json, template loading
    version-awareness.md  # feature registry, checkFeature(), version gates
    repl.md               # REPL architecture, session state

  extending/              # end-to-end contributor guides
    new-command.md        # full path: grammar → AST → visitor → executor → backend → MPR
    new-document-type.md  # adding a new Mendix document type
    new-backend-method.md # interface → MPR implementation → mock stub
    new-lint-rule.md      # rule registration, Starlark vs Go rules
    new-widget.md         # .def.json, widget registry, template extraction
```

**ADR format for `design/`:** Each decision file uses:
- **Context** — what problem prompted this decision
- **Decision** — what was chosen and what was explicitly rejected
- **Status** — `accepted` | `superseded` | `under review`
- **Consequences** — what the decision implies for contributors and agents

**Links to skills:** Each `extending/` guide links to the relevant maintainer skill for the step-by-step checklist. The wiki provides narrative and rationale; the skill provides the checklist and gotchas.

**`MAP.md` — the source-to-topic manifest:**
Maps source paths to wiki topics, enabling automated freshness checking:
```
mdl/executor/registry.go        → wiki/pipeline/executor.md
mdl/grammar/domains/            → wiki/pipeline/grammar.md
mdl/backend/                    → wiki/pipeline/backend.md
sdk/mpr/                        → wiki/pipeline/mpr.md
```

### Layer 3: Live Project State (Connected to GitHub)

The wiki provides static knowledge. Current project state — what is in progress, ready to pick up, blocked — lives in GitHub (issues, milestones, project board) and must not be duplicated in the wiki, where it would go stale immediately.

A scheduled Claude Code agent generates `wiki/CURRENT.md` periodically — not raw GitHub data, but a synthesised briefing: given open issues, active PRs, and recent decisions, what does the project need next and why. This gives contributor personal second-brain systems a pull-able summary, and gives AI agents dropping into a fresh session an orientation point without querying the GitHub API themselves.

### Layer 4: CDC Feed (`feed/brain.xml`)

An Atom feed that publishes insight events whenever the wiki gains new or significantly revised knowledge. This is the machine-readable layer that other brains — personal (meowary) or project — subscribe to and integrate.

**Feed schema** uses a `brain:` namespace for knowledge-specific metadata:

```xml
<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:brain="https://projectbrain.dev/ns/1.0">
  <title>mxcli Project Brain</title>
  <subtitle>CDC feed — insight events from the mxcli knowledge base</subtitle>
  <link href="[PAGES_URL]/feed/brain.xml" rel="self"/>
  <id>[PAGES_URL]/feed/brain.xml</id>
  <updated>[ISO8601]</updated>
  <author><name>mxcli Brain Agent</name></author>
  <!-- entries appended by /brain-ingest and wiki operations -->
</feed>
```

Each entry includes:
- `brain:event_type` — `concept-created` | `concept-revised` | `adr-added` | `adr-superseded` | `divergence-detected` | `synthesis-updated`
- `brain:confidence` — 0.0–1.0 (agent confidence in the insight)
- `brain:trigger` — `empirical` | `discussion` | `research` | `contradiction` | `synthesis`
- `brain:supersedes` — URI of the previous entry if this revises an earlier one

**Feed entries are append-only.** Never delete or modify existing entries. The feed is the immutable audit log of the project brain's evolution.

A `feed/.feedmeta` file holds feed configuration and the list of external brain feed URLs to sync from (e.g., meowary feeds from contributors).

**Emission rules** — a feed entry must be emitted when:
- A new page is created in `wiki/design/` (ADR) or `wiki/pipeline/`
- An existing ADR is superseded or deprecated
- A `wiki/extending/` guide is substantially revised
- A DIVERGENCE is detected during `/brain-sync`
- `wiki/CURRENT.md` is regenerated with a materially different project state

## Agent Commands

### `/brain-ingest`
Processes new files in `docs/raw/`:
1. Read each unprocessed file
2. Extract key concepts, decisions, insights
3. Create or update relevant wiki pages with cross-links to existing pages
4. Mark source file as processed (add `processed: true` to frontmatter)
5. Update `wiki/log.md`
6. Evaluate each changed wiki page against feed emission rules; append entries to `feed/brain.xml`

### `/brain-sync`
Consumes external brain feeds listed in `feed/.feedmeta`:
1. For each new feed item since last sync, determine if it extends, confirms, or contradicts current wiki knowledge
2. If extends: draft wiki update (propose, do not auto-commit)
3. If contradicts: create a DIVERGENCE entry in `wiki/synthesis/open-questions.md`
4. If confirms: note corroboration on the relevant wiki page
5. Emit a feed event summarising what was integrated

This is the primary mechanism for meowary (retran's personal second brain) to contribute knowledge back to the project brain, and for the project brain to receive updates from contributor systems.

### `/mxcli-dev:update-wiki`
End-of-feature wiki update command. Reads the current diff, identifies affected topics via `MAP.md`, and drafts updates to relevant wiki pages. Invoked intentionally at the end of a feature as part of the definition of done.

### `/brain-lint`
Health-checks the wiki:
- Verify all `[[wikilinks]]` resolve
- Check `wiki/README.md` lists all pages
- Flag concepts mentioned across multiple pages but without their own topic page
- Flag ADRs that may be invalidated by recent wiki changes
- Flag MAP.md entries whose source path no longer exists
- Report `wiki/design/` entries whose status has not been reviewed in 90 days

## Freshness Enforcement

**Hook — flag on edit:**
A `PostToolUse` hook on writes to mapped source paths looks up `MAP.md` and surfaces which wiki topics may be affected. It flags the connection; it does not generate content.

**Review integration:**
The existing `/mxcli-dev:review` command checks wiki freshness: for each source file changed in the PR, verify the corresponding wiki topic has been touched or explicitly noted as unaffected.

**CI check:**
A Makefile target compares modification dates of source files against their mapped wiki topics and warns when source is newer than documentation by more than a threshold. Staleness becomes visible at PR time.

## Skill Level Mapping

| | Maintainer (mxcli-dev) | User (mendix/) |
|---|---|---|
| **Procedural skills** | `.claude/skills/*.md` | `.claude/skills/mendix/*.md` |
| **Explanatory wiki** | `wiki/` (this proposal) | `docs-site/src/` (exists) |
| **Live state** | `wiki/CURRENT.md` (generated) | — |
| **CDC feed** | `feed/brain.xml` | — |

## Relationship to Meowary and Federated Brains

Retran's meowary system (https://github.com/retran/meowary) is a personal second brain with GitHub CLI integration, PARA structure, and session-planning scaffolding. The project brain is the *supply side* that feeds meowary and equivalent systems:

- **Project brain publishes:** `feed/brain.xml` (knowledge events), `wiki/CURRENT.md` (project state)
- **Meowary subscribes:** adds `feed/brain.xml` to `feed/.feedmeta` subscribed_feeds; `/brain-sync` pulls new entries and integrates them into meowary's knowledge base
- **Meowary contributes back:** decisions and insights from retran's sessions can be published to meowary's own feed, which the project brain pulls via `/brain-sync` and proposes as wiki updates

This is the federated model: each brain (project or personal) publishes a CDC feed; `/brain-sync` connects them bidirectionally. The `brain:` namespace makes feeds from different systems structurally compatible.

## What Is Not Changing

- **`CLAUDE.md`** — remains the primary AI context file as-is. This proposal does not replace or restructure it. The wiki complements CLAUDE.md; it does not supersede it.
- **`.claude/skills/`** — remain the procedural task references. The wiki's `extending/` guides are the narrative counterpart, not a replacement.
- **`docs-site/src/`** — remains the user-facing published documentation. Overlapping internals topics link between the two rather than merging.
- **`docs/11-proposals/`** — remains for in-flight feature proposals. Accepted proposals that establish lasting architectural decisions migrate to `wiki/design/` as ADRs.

## Summary and Priority

| Component | Problem solved | Effort |
|---|---|---|
| `wiki/pipeline/` + `wiki/extending/` | Orients contributors; raw material exists in docs-site | Low |
| `wiki/design/` (ADRs) | Captures why; prevents decision re-litigation | Low per decision |
| `MAP.md` + hook | Freshness enforcement without manual discipline | Low |
| `docs/raw/` + `/brain-ingest` | Structured intake of source material | Low |
| `feed/brain.xml` + emission rules | CDC feed for federated brain subscriptions | Low |
| `/brain-sync` | Bidirectional meowary ↔ project brain connection | Medium |
| `wiki/CURRENT.md` (scheduled agent) | Live session briefing from GitHub data | Medium |
| `/brain-lint` | Automated health checks | Medium |

**Recommended order:** `wiki/pipeline/` and `wiki/extending/` first — highest immediate value, raw material already exists. Add `MAP.md` and the hook immediately after. Then `docs/raw/`, `feed/brain.xml`, and `/brain-ingest` together as a batch — they form one coherent intake workflow. `/brain-sync`, CURRENT.md, and `/brain-lint` are independent and can follow in any order.
