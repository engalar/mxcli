# Contributing to mxcli

Thank you for your interest in contributing to mxcli! This document explains what we expect from contributors and how to make effective contributions.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Contribution Workflow](#contribution-workflow)
3. [Quality Requirements](#quality-requirements)
4. [Testing Your Changes](#testing-your-changes)
5. [Testing with Agentic Code (Claude Code)](#testing-with-agentic-code-claude-code)
6. [Git Workflow](#git-workflow)
7. [Code Standards](#code-standards)
8. [Documentation](#documentation)
9. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites

- Go 1.26+
- Git
- Make
- (Optional) Docker or Podman 4.7+ for dev container
- (Optional) Claude Code / Cursor for agentic development

### Local Setup

**Option 1: Dev Container (Recommended)**
```bash
git clone https://github.com/mendixlabs/mxcli
cd mxcli
# open in VS Code, then: Command Palette -> "Reopen in container"
# Podman users: select the "Mendix model SDK Go (Podman)" configuration
```

**Option 2: Local Machine**
```bash
git clone https://github.com/mendixlabs/mxcli
cd mxcli
make build
./bin/mxcli --help
```

### Common Tasks

```bash
make build        # build binary
make test         # run unit tests
make test-mdl     # run MDL integration tests
make lint         # run linter (fmt + vet)
make clean        # Clean build artifacts
make grammar      # Regenerate ANTLR parser (after modifying mdl/grammar/MDLParser.g4)
```

---

## Contribution Workflow

**We follow a strict workflow to ensure quality and prevent duplicate work.**

### Step 1: File an Issue First

**Always** file an issue before starting work. This prevents wasted effort and duplicate contributions.

1. Go to **Issues** -> **New Issue**
2. Choose template: **Bug** or **Feature**
3. Fill out **all required fields**
4. Provide clear context and expected behavior
5. **Submit issue** (don't start coding yet!)

**Examples:**
- Bug: "Creating entity with 200+ character name causes parser error"
- Feature: "Add `mxcli impact` command to analyze change impact"

### Step 2: Get Issue Approved

The maintainer will review and respond:

- **Approved**: "Let's do it!" -> Go to Step 3
- **Needs clarification**: Respond to questions
- **Not aligned**: "We're focusing on X instead" -> Discuss or close

**Don't start coding until the issue is approved.**

### Step 3: Assign Issue to Yourself

Once approved:

1. Click **Assignees** -> Select yourself
2. This signals you're working on it
3. Prevents others from duplicating your work
4. Gives the maintainer visibility on what's in progress

### Step 4: Implement in Feature Branch

Create a feature branch with a descriptive name:

```bash
git checkout -b feature/123-add-impact-command
git checkout -b fix/456-entity-parser-unicode
git checkout -b docs/789-add-mdl-examples
```

**Commit messages should reference the issue:**

```bash
git commit -m "feat: add impact command for change analysis (closes #123)"
git commit -m "fix: handle 200+ character entity names (closes #456)"
```

### Step 5: Validate Locally (Before Pushing)

**This is critical.** Validate that your code works before pushing.

#### 5a. Compile & Test

```bash
make build  # Must succeed
make test   # Must pass all tests
make lint   # Must have no issues
```

#### 5b. MDL Syntax Changes

If your change adds or modifies MDL syntax:

```bash
make grammar                          # Regenerate parser
./bin/mxcli check your-script.mdl     # Verify syntax parses
```

Add working examples in `mdl-examples/doctype-tests/` for any new syntax.

#### 5c. Mendix Studio Pro Validation

**Required for any change that affects Mendix project behavior** (new MDL statements, BSON serialization, entity/microflow modifications).

1. Apply your changes to a test `.mpr` project
2. Open the project in Mendix Studio Pro
3. Verify no errors (`mx check` or Studio Pro's error list)
4. Document the Mendix version you tested with

**Example validation:**
```
Tested: created entity with folder clause via MDL
Mendix version: 11.8.0
Result: mx check passes, entity appears in correct folder in Studio Pro
```

### Step 6: Create Pull Request

```bash
git push origin feature/123-add-impact-command
```

In your PR description:

1. **Link issue**: "Closes #123"
2. **What does it do?**: Brief description
3. **Testing**: Confirm `make test` and `make lint` pass
4. **Mendix validation**: Document what you tested and which version
5. **Agentic testing**: Confirm Claude Code can use the feature (see below)

### Step 7: CI Validates PR

GitHub Actions automatically verify:

- Code compiles
- Tests pass
- Integration tests pass (with `mx check` against real Mendix runtime)

**If any check fails**: Fix locally and push again.

### Step 8: Maintainer Review

The maintainer will:

1. Review code quality and architecture
2. Verify documentation completeness
3. Check Mendix Studio Pro validation claim
4. Verify the PR checklist in CLAUDE.md is satisfied

Once approved, the PR is merged.

---

## Quality Requirements

### PR Checklist

Every PR is checked against the review checklist in `CLAUDE.md`. Key items:

| Area | Requirements |
|------|-------------|
| **Full-stack MDL** | Grammar, parser regenerated, AST, visitor, executor, DESCRIBE roundtrip |
| **Test coverage** | MDL examples in `mdl-examples/doctype-tests/`, integration paths tested |
| **Version compat** | Version-gated features have registry entry and executor pre-check |
| **Security** | Restrictive socket permissions, no file I/O in hot paths |
| **Scope** | One concern per PR, refactors in separate commits |
| **Documentation** | Skills, CLI help text, syntax reference, site docs updated |

### Code Quality

```bash
make build    # Code compiles (CGO_ENABLED=0, no C compiler needed)
make test     # all tests pass
make lint     # go fmt + go vet pass
```

---

## Testing Your Changes

### Unit Tests

Test files are `*_test.go` in the same package. This project uses the standard `testing` package:

```go
func TestCreateEntityWithLongName(t *testing.T) {
    reader, err := modelsdk.Open("testdata/test.mpr")
    if err != nil {
        t.Fatalf("failed to open project: %v", err)
    }
    defer reader.Close()

    entities, err := reader.ListEntities()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(entities) == 0 {
        t.Fatal("expected at least one entity")
    }
}
```

### MDL Integration Tests

For new MDL commands, add test scripts in `mdl-examples/doctype-tests/`:

```sql
-- Create a module for testing
create module TestFeature;

-- Test the new feature
create json structure TestFeature.MyStructure
  folder 'Resources'
  snippet '{"id": 1, "name": "test"}';

-- Verify it exists
describe json structure TestFeature.MyStructure;

-- Clean up
drop json structure TestFeature.MyStructure;
```

Validate with:
```bash
./bin/mxcli check mdl-examples/doctype-tests/your-test.mdl
```

### Roundtrip Tests

The CI runs `TestMxCheck_DoctypeScripts` which executes each MDL script against a real Mendix project and validates with `mx check`. Your test script must produce zero errors.

---

## Testing with Agentic Code (Claude Code)

**mxcli is designed for AI agents to use.** Your contribution must work well with them.

### Why This Matters

mxcli's primary value is that AI agents (Claude Code, Cursor, etc.) can generate correct MDL scripts. If an agent can't understand or use your feature, the feature needs more work.

### How to Test

1. **Open the dev container** with Claude Code installed
2. **Ask Claude Code to use your feature** without giving it the exact syntax:
   ```
   I want to create a json structure for a customer api response
   with name, email, and an addresses array. Can you write the MDL?
   ```
3. **Evaluate the result:**
   - Did Claude generate correct MDL syntax?
   - Did it find the right skill/documentation?
   - Did the generated script pass `mxcli check`?

### What to Fix If Claude Struggles

| Problem | Fix |
|---------|-----|
| Claude doesn't know the syntax exists | Add/update skill in `.claude/skills/` |
| Claude generates wrong syntax | Improve examples in skill files |
| Claude uses outdated patterns | Update CLI help text (`Short`/`long`/`Example` in Cobra) |
| Error messages are unhelpful | Improve error text with hints |

### PR Section

Include in your PR:

```markdown
## Agentic Code Testing

- [ ] Tested with Claude Code in dev container
- [ ] Claude can generate correct MDL for this feature
- [ ] Skills updated (if applicable)
- [ ] error messages are helpful for debugging
```

---

## Git Workflow

### Branch Naming

```
feature/123-add-impact-command
fix/456-handle-unicode-names
docs/789-add-mdl-examples
refactor/101-simplify-catalog
```

### Commit Messages

Use conventional commits:

```
feat: add impact command for change analysis (closes #123)
fix: handle unicode characters in entity names (closes #456)
docs: add examples for create workflow (closes #789)
refactor: simplify executor dispatch logic
```

### Fork PR Flow

If you're contributing from a fork, this is the full cycle:

```
   jsmith/mxcli (fork)               mendixlabs/mxcli (origin)
   ─────────────────────             ─────────────────────────

                                      ┌─────────────────────┐
                                      │   origin/main       │
                                      └──────────┬──────────┘
                                                 │
                      git fetch origin           │
   ┌─────────────────────┐ ◀────────────────────┘
   │  local main         │
   │  git merge origin/  │
   │  main               │
   └──────────┬──────────┘
              │
   ┌──────────▼──────────┐
   │   feature branch    │
   └──────────┬──────────┘
              │
   ┌──────────▼──────────┐
   │   git push fork     │  ← push to jsmith/mxcli
   └──────────┬──────────┘
              │
   ┌──────────▼──────────┐           ┌─────────────────────┐
   │   open PR on GH     │ ────────▶ │  mendixlabs/main    │
   │   (fork → origin)   │           └──────────┬──────────┘
   └─────────────────────┘                      │
                                     ┌──────────▼──────────┐
                                     │  review + merge     │
                                     └──────────┬──────────┘
                                                │
                                   git fetch origin (repeat ↑)
```

### Scope

- Each commit does **one thing** (feature, bugfix, or refactor)
- Each PR is scoped to a **single feature or concern**
- Independent features go in separate PRs even if developed together
- Refactors that touch many files get their own commit

---

## Code Standards

### Go Conventions

- Follow `go fmt` and `go vet`
- Use descriptive names matching Mendix terminology
- Keep BSON/JSON tags consistent with Mendix serialization format
- Export types that should be part of the public API
- Handle all errors (no `_` for error returns)

### Project Structure

| Directory | Purpose |
|-----------|---------|
| `cmd/mxcli/` | CLI commands (Cobra) |
| `sdk/mpr/` | MPR file reading/writing, BSON parsing |
| `sdk/microflows/`, `sdk/pages/`, etc. | Domain types |
| `mdl/grammar/` | ANTLR4 grammar (`MDLParser.g4`, `MDLLexer.g4`) |
| `mdl/ast/` | AST node types |
| `mdl/visitor/` | ANTLR listener -> AST |
| `mdl/executor/` | AST execution against modelsdk |
| `mdl/catalog/` | SQLite catalog for project metadata queries |
| `api/` | High-level fluent API |
| `.claude/skills/` | AI agent skill documentation |
| `docs-site/src/` | Documentation site pages |

### BSON Storage Names

**Critical**: Mendix uses different storage names in BSON `$type` fields than the qualified names in SDK documentation. Always verify against `reference/mendixmodellib/reflection-data/` or existing MPR files. See the table in `CLAUDE.md` for common mismatches.

### BSON Tooling

When adding or debugging a new Mendix document type, see the [BSON Tooling Guide](docs/03-development/BSON_TOOLING_GUIDE.md) for which tool to use at each stage (`bson dump`, `bson compare`, `bson discover`, TUI, Python scripts, `mx check`).

---

## Documentation

### What Needs Documentation

| Change | Where to Document |
|--------|-------------------|
| New CLI command | Cobra `Short`/`long`/`Example` fields |
| New MDL statement | `docs/01-project/MDL_QUICK_REFERENCE.md`, skill in `.claude/skills/` |
| New SDK function | Godoc comments, `docs/GO_LIBRARY.md` |
| New site-facing feature | `docs-site/src/` pages |
| Changelog-worthy change | `CHANGELOG.md` (unreleased section) |

### Changelog Entry

Update `CHANGELOG.md`:

```markdown
## [Unreleased]

### added
- `describe nanoflow` with activities and control flows (#42)
- folder support for create json structure (#38)

### Fixed
- nanoflow parser now reads activities and return type from BSON (#43)
```

---

## Troubleshooting

### "Tests fail locally"

```bash
make build        # Ensure binary is up to date
make test         # run tests, read error output
make lint         # check formatting and vet
```

### "Parser changes don't take effect"

```bash
make grammar      # Regenerate ANTLR parser after .g4 changes
make build        # Rebuild with new parser
```

### "mx check fails in CI but not locally"

The CI uses a specific Mendix version (check `.github/workflows/`). Ensure your BSON serialization matches what that version expects. See `.claude/skills/debug-bson.md` for the debugging workflow.

### "How do I test with Claude Code?"

1. Open the project in VS Code -> Reopen in Container
2. Claude Code is auto-installed in the dev container
3. Ask it to use your feature and verify the output

### "GitHub Actions failed"

1. Go to **Actions** tab -> click the failing run
2. Expand the failing job and read the error
3. Fix locally with `make test` / `make lint` / `make build`
4. Push again

---

## Getting Help

- **Question about workflow?** Ask in the issue
- **Stuck on implementation?** Comment in the PR
- **Design discussion?** Create an issue with a proposal
- **Need feedback on approach?** Comment on the issue before coding

---

## Summary

| Stage | What's Expected |
|-------|----------------|
| **Before coding** | File issue, get approval, assign to yourself |
| **While coding** | Follow architecture, write tests, follow style |
| **Before pushing** | `make build` + `make test` + `make lint` pass |
| **In the PR** | Mendix Studio Pro validation, Claude Code testing documented |
| **Review** | Documentation complete, PR checklist satisfied |

Thank you for contributing!

---

## Code Quality Monitoring

The `source_tree` tool visualizes file sizes, dependency tiers, and quality metrics across the codebase:

```bash
go run ./cmd/source_tree           # file sizes + dependency depth
go run ./cmd/source_tree --fn      # function count + longest function per file
go run ./cmd/source_tree --dup     # intra-file duplication
go run ./cmd/source_tree --churn   # commit frequency per file
go run ./cmd/source_tree --all     # all metrics
go run ./cmd/source_tree --cover   # with test coverage (runs tests if no coverage.out)
```

Output is color-coded by severity (green/yellow/orange/red), making it easy to spot files that need attention.
