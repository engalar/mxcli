# Test Layering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire a 3-layer test gate — pre-commit unit tests (L1), pre-push change-aware integration tests (L2), and CI full suite (L3) — so failures surface as early and cheaply as possible.

**Architecture:** L1 blocks the commit locally via `.githooks/pre-commit`; L2 blocks the push via `.githooks/pre-push` using static file-path analysis to decide which integration tests are relevant; L3 runs the full suite in GitHub Actions on every PR and release. Claude Code injects failure logs automatically via `PostToolUse` hook.

**Tech Stack:** POSIX sh (hooks), Go (`make test`, `make test-integration`), GitHub Actions, Claude Code hooks (`.claude/settings.json`)

---

## Current State

| Layer | Trigger | Status |
|-------|---------|--------|
| L1 unit tests | pre-commit | ✅ Done — `.githooks/pre-commit` |
| Claude log injection | PostToolUse Bash hook | ✅ Done — `.claude/settings.json` |
| L2 integration tests | pre-push (change-aware) | ❌ Missing |
| L3 full suite | CI on PR + tag | ⚠️ Partial — `push-test.yml` runs on every push but has no fast/slow split |

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `.githooks/pre-push` | Create | L2: change-aware integration test gate |
| `.github/workflows/push-test.yml` | Modify | Split into fast job (unit) + slow job (integration) so unit feedback is instant |
| `Makefile` | Modify | Add `test-changed` target that accepts a file list |
| `.claude/settings.json` | Already done | PostToolUse log injection |

---

## Task 1: Pre-push hook — change-aware L2 integration tests

**Files:**
- Create: `.githooks/pre-push`

The hook reads the list of files changed between the local branch and the remote, maps them to test tags, and only runs `make test-integration` if relevant paths changed. This avoids 30-minute integration runs for doc or skill-only changes.

**Path → integration test mapping:**
```
mdl/backend/      → integration tests (BSON mutation)
mdl/executor/     → integration tests (command execution)
mdl/visitor/      → integration tests (parser → AST)
modelsdk/         → integration tests (BSON roundtrip)
internal/expr/    → integration tests (expression checker)
cmd/mxcli/        → integration tests (CLI commands)
```

Everything else (`.claude/`, `docs/`, `*.md`, `.githooks/`, `Makefile`) → skip L2.

- [ ] **Step 1: Write the pre-push hook**

```sh
cat > .githooks/pre-push << 'EOF'
#!/bin/sh
# L2: change-aware integration tests — runs only when relevant paths changed.

REMOTE="$1"
REMOTE_URL="$2"

# Collect commits being pushed
while read local_ref local_sha remote_ref remote_sha; do
    if [ "$local_sha" = "0000000000000000000000000000000000000000" ]; then
        continue  # branch deletion, skip
    fi

    if [ "$remote_sha" = "0000000000000000000000000000000000000000" ]; then
        # New branch — compare against default branch
        BASE=$(git rev-parse --verify origin/main 2>/dev/null || git rev-parse --verify origin/master 2>/dev/null)
    else
        BASE="$remote_sha"
    fi

    CHANGED=$(git diff --name-only "$BASE" "$local_sha" 2>/dev/null)

    # Check if any changed file touches integration-relevant paths
    NEEDS_INTEGRATION=0
    for f in $CHANGED; do
        case "$f" in
            mdl/backend/*|mdl/executor/*|mdl/visitor/*|modelsdk/*|internal/expr/*|cmd/mxcli/*)
                NEEDS_INTEGRATION=1
                break
                ;;
        esac
    done

    if [ "$NEEDS_INTEGRATION" = "1" ]; then
        echo "pre-push: integration-relevant paths changed, running L2 tests..."
        echo "pre-push: (skip with: git push --no-verify)"
        echo ""
        if ! CGO_ENABLED=0 make test-integration; then
            echo "" >&2
            echo "PUSH BLOCKED: integration tests failed." >&2
            echo "Run: make test-integration  to see full output." >&2
            exit 1
        fi
        echo "pre-push: integration tests passed."
    else
        echo "pre-push: no integration-relevant paths changed, skipping L2."
    fi
done

exit 0
EOF
chmod +x .githooks/pre-push
```

- [ ] **Step 2: Verify hook syntax**

```bash
sh -n .githooks/pre-push
```
Expected: no output (syntax clean).

- [ ] **Step 3: Smoke-test hook with a no-op push simulation**

```bash
# Simulate: push with only doc changes
echo "docs/README.md" | \
  CHANGED="docs/README.md" sh -c '
    case "docs/README.md" in
      mdl/backend/*|mdl/executor/*|mdl/visitor/*|modelsdk/*|internal/expr/*|cmd/mxcli/*)
        echo "NEEDS_INTEGRATION=1";;
      *)
        echo "NEEDS_INTEGRATION=0 (correct)";;
    esac
  '
```
Expected: `NEEDS_INTEGRATION=0 (correct)`

```bash
# Simulate: push with executor changes
sh -c '
  case "mdl/executor/cmd_create.go" in
    mdl/backend/*|mdl/executor/*|mdl/visitor/*|modelsdk/*|internal/expr/*|cmd/mxcli/*)
      echo "NEEDS_INTEGRATION=1 (correct)";;
    *)
      echo "NEEDS_INTEGRATION=0";;
  esac
'
```
Expected: `NEEDS_INTEGRATION=1 (correct)`

- [ ] **Step 4: Commit**

```bash
git add .githooks/pre-push
git commit -m "chore(dev): add pre-push L2 integration test gate (change-aware)"
```

---

## Task 2: Split CI workflow into fast + slow jobs

**Files:**
- Modify: `.github/workflows/push-test.yml`

Currently the CI runs unit tests, integration tests, and lint in one sequential job. Unit test failure means waiting for the integration setup (mxbuild download) before seeing the error. Split into two jobs: `unit` (fast, ~1 min) and `integration` (slow, ~10 min, only runs if `unit` passes).

- [ ] **Step 1: Read current workflow**

```bash
cat .github/workflows/push-test.yml
```

- [ ] **Step 2: Rewrite workflow with fast/slow split**

Replace contents of `.github/workflows/push-test.yml`:

```yaml
name: Build, Test & Lint

on: [push, pull_request]

permissions:
  contents: read

jobs:
  unit:
    name: Unit tests & lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'
      - name: Build
        run: make build
      - name: Unit tests
        run: make test
      - name: Check MDL example scripts
        run: |
          FAILED=0
          for f in mdl-examples/doctype-tests/*.mdl; do
            case "$f" in *.test.mdl) continue ;; esac
            NAME=$(basename "$f")
            echo "::group::$NAME"
            if ./bin/mxcli check "$f"; then
              echo "PASS: $NAME"
            else
              echo "::error file=$f::Syntax check failed: $NAME"
              FAILED=1
            fi
            echo "::endgroup::"
          done
          exit $FAILED
      - name: Lint Go
        run: make lint-go
      - name: Vulnerability scan
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  integration:
    name: Integration tests
    runs-on: ubuntu-latest
    needs: unit
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - name: Cache ANTLR4 JAR
        uses: actions/cache@v5
        with:
          path: ~/.m2/repository/org/antlr/antlr4
          key: antlr4-4.13.2
      - name: Install ANTLR4
        run: pip install 'antlr4-tools==0.2.2'
      - name: Generate parser
        run: make grammar
        env:
          ANTLR4_TOOLS_ANTLR_VERSION: '4.13.2'
      - name: Build
        run: make build
      - name: Setup mxbuild
        run: ./bin/mxcli setup mxbuild --version 11.9.0
      - name: Integration tests
        run: make test-integration
        timeout-minutes: 30
```

- [ ] **Step 3: Verify YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/push-test.yml'))" && echo "YAML valid"
```
Expected: `YAML valid`

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/push-test.yml
git commit -m "ci: split push-test into fast unit job + slow integration job"
```

---

## Summary: Full 3-Layer Gate

```
git commit
  └─ pre-commit (L1): go test ./...
      FAIL → block commit, write .test-fail.log, Claude sees it via PostToolUse
      PASS → commit proceeds

git push
  └─ pre-push (L2): change-aware
      doc/skill changes only → skip (instant)
      mdl/backend, executor, modelsdk, expr, cmd/mxcli changed → make test-integration
      FAIL → block push
      PASS → push proceeds

GitHub Actions (L3): every PR + push
  └─ unit job (~1 min): build + make test + MDL check + lint + vuln scan
      FAIL → immediate red, no waiting for integration
  └─ integration job (~10 min): needs unit; make test-integration
      FAIL → red, PR blocked
```

**Developer setup (once after clone):**
```bash
make setup   # git config core.hooksPath .githooks
```
