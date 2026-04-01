# Proposal: Multi-Agent Merge for Mendix Projects

**Status:** Draft
**Date:** 2026-03-30
**Author:** AI-assisted design
**Related:** [PROPOSAL_concurrent_access.md](PROPOSAL_concurrent_access.md)

---

## Summary

When multiple AI agents work on a Mendix project in parallel (e.g., Claude Code subagents in independent git worktrees), each produces a modified MPR file. Since MPR files are binary (SQLite databases), git cannot merge them. This proposal describes the problem space, evaluates solution options, and recommends a phased approach using git's custom merge/diff driver infrastructure with MDL as the merge representation.

## Problem

### Binary Files Don't Merge

Git's merge algorithm operates on text lines. When two branches modify the same binary file, git cannot merge them — it reports a conflict and the user must manually choose one version.

Mendix project files are binary:

| Format | Storage | Merge-friendly? |
|--------|---------|-----------------|
| MPR v1 | Single SQLite database | No — opaque binary blob |
| MPR v2 | SQLite metadata + `mprcontents/*.mxunit` (BSON files) | Partially — individual documents are separate files, but still binary |

### The Multi-Agent Scenario

Modern AI coding workflows spawn multiple agents in parallel, each working in an isolated git worktree:

```
main branch (base.mpr)
  ├── worktree-A: Agent creates Shop.Product entity
  ├── worktree-B: Agent creates Shop.ACT_ProcessOrder microflow
  └── worktree-C: Agent adds security rules to Shop module
```

Each agent modifies `app.mpr` independently. When merging back:

```
$ git merge worktree-A    # ✓ fast-forward, works
$ git merge worktree-B    # ✗ conflict: Binary files differ
```

The second merge always fails, even when the changes are logically independent (different documents, different modules). All agent work after the first merge is lost unless manually re-applied.

### Impact

This isn't hypothetical. Claude Code already uses worktrees for parallel agent work. As multi-agent workflows become standard, the inability to merge Mendix project changes becomes a hard blocker for productivity gains.

### MPR v2: Better But Not Solved

MPR v2 decomposes the monolithic database into individual `.mxunit` files under `mprcontents/`. Two agents touching different documents (e.g., different entities) modify different files — git can merge these without conflict.

However, MPR v2 still has merge problems:

1. **Same-document conflicts** — two agents modifying the same entity, microflow, or page produce conflicting BSON files that git cannot merge
2. **SQLite metadata** — the `app.mpr` file still contains a `_units` table that both agents update
3. **BSON is binary** — even individual `.mxunit` files are opaque to git

## Prior Art: How Other Tools Solved This

| System | Binary Problem | Solution | Outcome |
|--------|---------------|----------|---------|
| **Unity** | `.unity` scene files (binary) | Switched to YAML serialization + `UnityYAMLMerge` custom merge driver | Industry standard; YAML scenes merge via git |
| **Unreal Engine** | `.uasset` binary assets | File locking via Perforce; no merge | Works but serializes all access |
| **Figma** | Design files | Server-based OT (no files at all) | Eliminated the problem by removing files |
| **Jupyter Notebooks** | `.ipynb` (JSON, merge-hostile) | `nbdime`: custom diff/merge driver for notebooks | `git diff` shows cell-level diffs; `git merge` handles cell-level conflicts |
| **Database systems** | Binary data files | Migration scripts as source of truth | Universal approach in web development |
| **Terraform** | State file (JSON, merge-hostile) | State is a derived build artifact; `.tf` files are source | State never committed; only text files merge |
| **Xcode Storyboards** | XML but merge-hostile | Apple recommends splitting into smaller storyboards | Reduces conflict frequency but doesn't eliminate it |
| **gettext (`.po`)** | Translation catalogs | `merge-po-files` custom merge driver | Domain-aware merge handles message reordering |

**Common pattern**: Every successful solution either (a) converts to a text representation for merging, or (b) eliminates file-based collaboration entirely.

## Git Infrastructure for Custom Merge/Diff

Git provides several extension points for handling non-text files. These are the building blocks for any solution.

### 1. Custom Diff via `textconv`

Converts binary files to text for `git diff` display. Does not affect merge.

```gitattributes
*.mpr diff=mpr
```

```ini
# .gitconfig or .git/config
[diff "mpr"]
    textconv = mxcli export --format mdl
    cachetextconv = true
```

Effect: `git diff`, `git log -p`, and `git show` display MDL diffs instead of "Binary files differ". The `cachetextconv` option avoids redundant exports for unchanged files.

### 2. Custom Merge Driver

Git invokes a custom command when merging conflicting files. The driver receives three versions (base, ours, theirs) and must produce the merged result.

```gitattributes
*.mpr merge=mpr
```

```ini
[merge "mpr"]
    name = Mendix MPR merge
    driver = mxcli merge-driver %O %A %B %P
    recursive = binary
```

Parameters:
- `%O` — ancestor (common base version)
- `%A` — current branch ("ours") — driver writes merged result here
- `%B` — other branch ("theirs")
- `%P` — file path

Exit codes: 0 = clean merge, non-zero = conflict (user must resolve).

### 3. Clean/Smudge Filters

Transform files between the working tree and git's internal storage.

```gitattributes
*.mpr filter=mpr
```

```ini
[filter "mpr"]
    clean = mxcli mpr-to-mdl %f
    smudge = mxcli mdl-to-mpr %f
```

Effect: Git internally stores MDL (text) but the working tree contains the MPR (binary). Merges happen on the clean (text) representation automatically. This is the most transparent option but requires perfect round-trip fidelity.

### 4. Smudge/Clean + LFS

Combine Git LFS for efficient binary storage with clean/smudge for merge-time conversion:

```gitattributes
*.mpr filter=lfs diff=mpr merge=mpr
```

LFS handles storage efficiency; the merge driver handles semantic merging.

## Solution Options

### Option A: MDL Script Merge (Migration Pattern)

Each agent produces an MDL change script rather than a modified MPR. Scripts are text files that merge naturally in git, then are applied sequentially to a base MPR.

```
repo/
  app.mpr                    # base project (committed)
  changes/
    001-add-product.mdl       # Agent A's changes
    002-add-microflow.mdl     # Agent B's changes
    003-add-security.mdl      # Agent C's changes
```

```bash
# Merge workflow
git merge feature-branch       # merges .mdl text files
mxcli exec changes/*.mdl -p app.mpr  # applies all scripts
```

**Pros:**
- Text files merge naturally in git
- No custom git infrastructure needed
- MDL scripts are already human-readable and reviewable
- Change history is preserved as individual scripts

**Cons:**
- Requires workflow discipline (agents must produce scripts, not modify MPR directly)
- Semantic conflicts (e.g., two agents creating same entity name) aren't caught until apply time
- Script ordering matters; concurrent agents can't know the correct sequence
- Requires MDL coverage for all changes an agent might make

### Option B: Custom Merge Driver (Recommended)

Register a git merge driver that converts MPR files to MDL, performs a three-way text merge, and applies the result back to the MPR.

```
git merge feature-branch
  └── git detects conflict on app.mpr
      └── invokes: mxcli merge-driver %O %A %B %P
          ├── mxcli export %O → base.mdl
          ├── mxcli export %A → ours.mdl
          ├── mxcli export %B → theirs.mdl
          ├── git merge-file ours.mdl base.mdl theirs.mdl
          ├── mxcli apply merged.mdl → %A (result)
          └── exit 0 (clean) or 1 (conflict)
```

**Pros:**
- Transparent to the user — standard `git merge` just works
- Three-way merge catches true conflicts (same entity modified in both branches)
- Can fall back to conflict markers in MDL for manual resolution
- Agents work normally (modify MPR directly), no workflow changes

**Cons:**
- Requires `mxcli` to be installed and on PATH for merges
- MDL round-trip fidelity gaps cause data loss (currently 47/52 domains not implemented)
- Export + merge + apply is slower than text merge
- Complex error handling (what if export fails? what if apply fails?)

**For MPR v2**, the merge driver can be more granular:

```gitattributes
*.mxunit merge=mxunit
app.mpr merge=mpr-metadata
```

Individual `.mxunit` files can be merged at the document level, with only the SQLite metadata requiring special handling.

### Option C: Clean/Smudge Filter (Text-as-Source)

Store MDL text in git, reconstruct MPR in the working tree via smudge filter. Git merges happen entirely on text.

```
What git stores (internal):     What developer sees (worktree):
  app.mdl (text)         ←→      app.mpr (binary)
                    clean↑  ↓smudge
```

**Pros:**
- Most transparent solution — git diffs, merges, blame all work on text
- No custom merge driver needed — standard text merge
- Git history is readable without special tooling
- Smallest conceptual overhead once set up

**Cons:**
- Requires near-perfect MDL round-trip fidelity (current gap: 47/52 metamodel domains)
- Performance: every checkout/commit runs an export/import
- Smudge filter needs a base MPR to apply MDL to (chicken-and-egg problem for fresh clones)
- Most ambitious option; premature until MDL coverage is comprehensive

### Option D: SQLite Changeset Merge

Use SQLite's [session extension](https://www.sqlite.org/sessionintro.html) to record row-level changesets, then merge at the SQL level.

```
base.mpr → Agent A works → changeset-a.sqlite-patch
         → Agent B works → changeset-b.sqlite-patch

Merge: apply changeset-a then changeset-b to base.mpr
```

**Pros:**
- Works at the storage level — no MDL coverage gaps
- SQLite's changeset format handles insert/update/delete conflicts
- Handles all 52 metamodel domains automatically

**Cons:**
- Row-level merge can succeed at SQL level but produce semantically invalid Mendix models
- Still needs `mx check` validation after merge
- Changeset generation requires instrumenting all MPR writes
- Not integrated with git (would need a wrapper merge driver anyway)
- SQLite session extension may not be available in pure-Go SQLite (`modernc.org/sqlite`)

### Option E: Module-Level Locking

Don't merge — instead, assign non-overlapping ownership. Each agent works on a different module.

```
Agent A: owns Shop module        → modifies Shop/* documents
Agent B: owns Inventory module   → modifies Inventory/* documents
Agent C: reads only, no writes
```

**Pros:**
- Simple coordination protocol
- No merge conflicts possible if ownership is respected
- Works today with MPR v2 (different modules = different files)

**Cons:**
- Severely limits parallelism (one agent per module)
- Cross-module changes (e.g., adding an association between modules) require coordination
- Doesn't scale — module count limits agent count
- MPR v1 still has the single-file problem

## Comparison Matrix

| Criterion | A: Script Merge | B: Merge Driver | C: Clean/Smudge | D: SQLite Changeset | E: Locking |
|-----------|:-:|:-:|:-:|:-:|:-:|
| Transparent to user | Low | **High** | **High** | Medium | Medium |
| Works with standard git | Yes | **Yes** | **Yes** | No (needs wrapper) | Yes |
| Handles all metamodel domains | No (MDL gaps) | No (MDL gaps) | No (MDL gaps) | **Yes** | **Yes** |
| Implementation complexity | Low | **Medium** | High | High | Low |
| Agent workflow changes needed | Yes | **No** | **No** | No | Yes |
| MPR v1 support | Yes | **Yes** | Yes | Yes | No |
| MPR v2 optimized | No | **Yes (per-document)** | N/A | Yes | Yes |
| Merge quality | Text-level | **3-way with validation** | Text-level | SQL-level | N/A |
| Available today (proof of concept) | **Mostly** | Partially | No | No | Yes |

## Recommended Approach

A phased approach, starting with immediate value and building toward the full solution.

### Phase 1: MDL textconv for Readable Diffs (immediate)

**Effort:** Small (configuration + thin wrapper script)

Register `mxcli` as a textconv driver so `git diff` and `git log` show meaningful diffs for MPR files.

```gitattributes
# .gitattributes
*.mpr diff=mpr
```

```ini
# Installed by: mxcli init --git-config
[diff "mpr"]
    textconv = mxcli export --format mdl --file
    cachetextconv = true
```

**What this enables:**
- `git diff` shows MDL diffs instead of "Binary files differ"
- `git log -p -- app.mpr` shows change history in readable form
- Pull request diffs are human-reviewable
- Zero risk — display only, doesn't affect merge or checkout

**Implementation:**
- Add `mxcli export --format mdl --file <path>` command that exports a standalone MPR to MDL on stdout
- Add `mxcli init --git-config` flag that writes the `.gitattributes` and `.git/config` entries
- Document in README

### Phase 2: Custom Merge Driver for MPR v2 (short-term)

**Effort:** Medium

For MPR v2 projects, register a per-document merge driver for `.mxunit` files and a metadata merge driver for the SQLite file.

```gitattributes
# .gitattributes (MPR v2 projects)
*.mxunit merge=mxunit-merge diff=mxunit
app.mpr merge=mpr-metadata-merge diff=mpr
```

The `.mxunit` merge driver:

```
mxcli merge-mxunit %O %A %B %P
  1. Parse BSON from all three versions
  2. Convert each to MDL (or structured AST)
  3. Three-way merge on the structured representation
  4. Serialize merged result back to BSON
  5. Write to %A, exit 0 (clean) or 1 (conflict)
```

The `app.mpr` metadata merge driver:

```
mxcli merge-mpr-metadata %O %A %B %P
  1. Open all three SQLite files
  2. Diff _units tables (rows added/removed/modified)
  3. Apply non-conflicting row changes
  4. Report conflicts on same-row modifications
```

**What this enables:**
- `git merge` succeeds automatically when agents touch different documents
- Same-document conflicts produce readable MDL conflict markers
- Agents work with standard `mxcli` commands — no workflow changes

**Scope limitation:** Only handles document types that mxcli can export/import. For unsupported types (47/52 domains), the merge driver falls back to binary conflict (same as today).

### Phase 3: MDL Script Workflow for Claude Code (short-term, parallel)

**Effort:** Small (workflow/documentation + minor tooling)

Independent of Phase 2, establish a convention where Claude Code agents produce MDL scripts instead of modifying the MPR directly. This works today with existing mxcli capabilities.

```
# .claude/skills/multi-agent-workflow.md

When working in a worktree:
1. Write your changes as an MDL script (e.g., changes.mdl)
2. Validate with: mxcli check changes.mdl -p app.mpr --references
3. Commit only the .mdl script, not the modified app.mpr
4. The coordinator agent applies all scripts sequentially
```

**What this enables:**
- Multi-agent collaboration today, without custom git drivers
- Text-based merge of MDL scripts
- Clear audit trail of each agent's changes
- Works with both MPR v1 and v2

**Tooling needed:**
- `mxcli apply --dry-run` — validate a script can apply cleanly without executing
- `mxcli apply --sequential changes/*.mdl -p app.mpr` — apply scripts in order with conflict detection
- Coordinator logic in Claude Code skill files

### Phase 4: Full MPR Merge Driver (long-term)

**Effort:** Large

Extend the merge driver to handle MPR v1 (single SQLite file) and all metamodel domains. This requires:

1. Closing MDL coverage gaps (ongoing effort)
2. Implementing structured three-way merge at the AST level (not just text merge of MDL)
3. Handling complex conflicts (e.g., entity renamed in one branch, attributes added in another)
4. Integration testing against real Mendix projects with `mx check` validation

**Target state:** `git merge` on any Mendix project merges cleanly when changes don't conflict, produces readable conflict markers when they do, and the result always passes `mx check`.

### Phase 5: Clean/Smudge Filter (aspirational)

**Effort:** Very large

Once MDL covers all metamodel domains with perfect round-trip fidelity, switch to clean/smudge filters. Git stores MDL text internally; the working tree has MPR files. Merges operate entirely on text.

This is the end-state where Mendix projects merge exactly like source code. It depends on MDL becoming a complete, lossless representation of the MPR — a significant long-term investment.

## Implementation Details

### `mxcli merge-driver` Command

```go
// cmd/mxcli/cmd_merge.go

var mergeDriverCmd = &cobra.Command{
    Use:   "merge-driver <base> <ours> <theirs> [path]",
    Short: "Git merge driver for MPR/mxunit files",
    Args:  cobra.RangeArgs(3, 4),
    RunE: func(cmd *cobra.Command, args []string) error {
        base, ours, theirs := args[0], args[1], args[2]

        // 1. Export all three to MDL
        baseMDL, err := export(base)
        oursMDL, err := export(ours)
        theirsMDL, err := export(theirs)

        // 2. Three-way text merge
        merged, conflicts, err := threewayMerge(baseMDL, oursMDL, theirsMDL)

        // 3. Apply merged MDL to base MPR
        if err := applyMDL(merged, base, ours); err != nil {
            return err // writes conflict markers to ours
        }

        if conflicts > 0 {
            return fmt.Errorf("%d conflicts", conflicts)
        }
        return nil // clean merge
    },
}
```

### `mxcli init --git-config`

Extend the existing `mxcli init` command to set up git integration:

```go
func setupGitConfig(projectDir string) error {
    // Write .gitattributes entries
    appendGitattributes(projectDir, []string{
        "*.mpr diff=mpr merge=mpr-merge",
        "*.mxunit diff=mxunit merge=mxunit-merge",
    })

    // Configure local git config
    gitConfig(projectDir, "diff.mpr.textconv", "mxcli export --format mdl --file")
    gitConfig(projectDir, "diff.mpr.cachetextconv", "true")
    gitConfig(projectDir, "merge.mpr-merge.driver", "mxcli merge-driver %O %A %B %P")
    gitConfig(projectDir, "merge.mpr-merge.name", "Mendix MPR merge driver")

    return nil
}
```

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| MDL coverage gaps cause data loss during merge | High | Phase 2 merge driver detects unsupported types and falls back to binary conflict instead of silently dropping data |
| Merge produces MPR that fails `mx check` | Medium | Always run `mx check` after merge; merge driver can run validation automatically |
| Performance: export+merge+apply too slow | Low | Cache exports (`cachetextconv`); most merges involve small diffs |
| Agent produces MDL that conflicts semantically | Medium | `mxcli check --references` catches duplicate names, missing references |
| `mxcli` not installed on machine performing merge | Medium | Merge driver degrades gracefully to binary conflict; `mxcli init` checks prerequisites |

## Verification Plan

```bash
# Test 1: textconv shows readable diff
git diff HEAD~1 -- app.mpr
# Expected: MDL output, not "Binary files differ"

# Test 2: Non-conflicting merge (different documents)
git checkout -b agent-a && # add entity → commit
git checkout -b agent-b main && # add microflow → commit
git checkout main && git merge agent-a && git merge agent-b
# Expected: clean merge, both entity and microflow present

# Test 3: Conflicting merge (same document)
git checkout -b agent-c && # add Name attribute to Customer → commit
git checkout -b agent-d main && # add Age attribute to Customer → commit
git checkout main && git merge agent-c && git merge agent-d
# Expected: merge driver produces merged entity with both attributes

# Test 4: True conflict
git checkout -b agent-e && # rename Customer to Client → commit
git checkout -b agent-f main && # rename Customer to Buyer → commit
git checkout main && git merge agent-e && git merge agent-f
# Expected: conflict reported with readable MDL conflict markers

# Test 5: Merged project validates
mx check app.mpr
# Expected: no errors after any successful merge
```

## Open Questions

1. **Structured vs. text merge**: Should the merge driver do a text-level merge of MDL strings, or a structured merge on the AST? Text merge is simpler but can't handle reordering. AST merge is more robust but significantly more complex.

2. **Conflict representation**: When a merge conflict occurs, should the driver write MDL conflict markers (like `<<<<<<< OURS`) to a `.mdl` file for manual resolution, or should it launch an interactive resolver?

3. **mxcli version compatibility**: The merge driver must match the MDL capabilities of the mxcli that created the project. Should the driver version be pinned in `.gitattributes` or resolved dynamically?

4. **CI/CD integration**: Should the merge driver be available as a GitHub Action or pre-built binary for CI environments where `mxcli` isn't installed?
