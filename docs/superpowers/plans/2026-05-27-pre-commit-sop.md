# Pre-Commit SOP Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `.githooks/sop/` directory with one Markdown SOP file per check script, and patch each check script to emit `SOP:` + `CONTEXT:` lines before every `exit 1` so Claude Code can auto-repair pre-commit blocks without human involvement.

**Architecture:** Each check script gets a 2-line addition before each `exit 1` — a `SOP:` path line and a `CONTEXT:` key=value line using variables already in scope. Eight new SOP files (one per check) follow a fixed three-section format: Trigger / Context variables / Steps.

**Tech Stack:** POSIX sh (hooks), Markdown (SOP files), no new dependencies.

---

## File map

**New files (`.githooks/sop/`):**
- `01-skill-structure.md`
- `02-unit-tests.md`
- `03-describe-datasource-arch.md`
- `03-gen-consistency.md`
- `03-helpdesk-mdl-golden-sync.md`
- `03-protect-golden-mpr.md`
- `04-no-raw-bson-in-executor.md`
- `05-mx-check-golden.md`

**Modified files (`.githooks/checks/`):**
- `01-skill-structure.sh` — 1 `exit 1`, add 2 lines
- `02-unit-tests.sh` — 1 `exit 1`, add 2 lines
- `03-describe-datasource-arch.sh` — 1 `exit 1` (via `failed` flag), restructure to emit before exit
- `03-gen-consistency.sh` — 1 `exit 1`, add 2 lines
- `03-helpdesk-mdl-golden-sync.sh` — 2 `exit 1` paths (Rule A + Rule B), add 2 lines each
- `03-protect-golden-mpr.sh` — 1 `exit 1`, add 2 lines
- `04-no-raw-bson-in-executor.sh` — 1 `exit 1`, add 2 lines
- `05-mx-check-golden.sh` — 1 `exit 1`, add 2 lines

---

### Task 1: Create SOP directory and write 01 + 02 SOP files

**Files:**
- Create: `.githooks/sop/01-skill-structure.md`
- Create: `.githooks/sop/02-unit-tests.md`

- [ ] **Step 1: Create the sop directory and write `01-skill-structure.md`**

```markdown
# SOP: 01-skill-structure

## Trigger
Pre-commit blocks with: "ERROR: Invalid skill file structure."

## Context variables
- `{INVALID_FILES}` — space-separated list of files with wrong names

## Steps
1. For each file in `{INVALID_FILES}`:
   - Determine the intended skill name from the filename (e.g. `my-skill.md` → `my-skill`)
   - Run: `mkdir -p .claude/skills/<skill-name>`
   - Run: `git mv <file> .claude/skills/<skill-name>/SKILL.md`
2. Run: `git status` — verify staged renames look correct
3. Re-attempt commit
```

- [ ] **Step 2: Write `02-unit-tests.md`**

```markdown
# SOP: 02-unit-tests

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: unit tests failed."

## Context variables
- `{LOG_FILE}` — path to the test failure log (always `.test-fail.log`)

## Steps
1. Run: `cat {LOG_FILE}` — read the full failure output
2. Identify the failing test package and test name from the `FAIL` lines
3. Run: `CGO_ENABLED=0 go test -timeout 120s -run TestXxx ./path/to/pkg -v` (substitute actual test name and package)
4. Fix the failing code (implementation or test, depending on root cause)
5. Run: `CGO_ENABLED=0 go test -timeout 120s ./...` — confirm all pass
6. `git add` the changed files and re-attempt commit
```

- [ ] **Step 3: Commit**

```bash
git add .githooks/sop/01-skill-structure.md .githooks/sop/02-unit-tests.md
git commit -m "chore(hooks): add SOP files for 01-skill-structure and 02-unit-tests"
```

---

### Task 2: Write 03-describe-datasource-arch and 03-gen-consistency SOP files

**Files:**
- Create: `.githooks/sop/03-describe-datasource-arch.md`
- Create: `.githooks/sop/03-gen-consistency.md`

- [ ] **Step 1: Write `03-describe-datasource-arch.md`**

```markdown
# SOP: 03-describe-datasource-arch

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: ... adds raw BSON field access by string literal."

## Context variables
- `{AFFECTED_FILE}` — the executor Go file that added raw field access
- `{BAD_FIELDS}` — e.g. `["NanoflowSettings"] ["Nanoflow"]`

## Steps
1. Open `{AFFECTED_FILE}` and find the lines accessing `{BAD_FIELDS}` via `ds["Field"]` or `w["Field"]`
2. Identify the BSON type (e.g. `Forms$NanoflowSource`) from the surrounding `case` or `if` block
3. Find the corresponding gen type in `modelsdk/gen/pages/types.go` — search for the struct whose `SetTypeName` matches the BSON type
4. Replace the raw map access with gen-type decode:
   ```go
   raw, _ := bson.Marshal(ds)
   elem, _ := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(raw))
   typed, ok := elem.(*genPg.TheType)
   if !ok { return "" }
   return typed.TheQualifiedName()
   ```
5. Run: `CGO_ENABLED=0 go test ./mdl/executor/... -v` — confirm tests pass
6. `git add {AFFECTED_FILE}` and re-attempt commit
```

- [ ] **Step 2: Write `03-gen-consistency.md`**

```markdown
# SOP: 03-gen-consistency

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: modelsdk/gen/ changed but codegen source unchanged."

## Context variables
- `{GEN_STAGED}` — number of staged files in modelsdk/gen/

## Steps
1. Run: `git diff --cached --name-only | grep '^modelsdk/gen/'` — list the staged gen files
2. Determine what manual edits were made (gen files must not be hand-edited)
3. If the edit was intentional (e.g. fixing a bug in gen output):
   - Find the generator template in `internal/codegen/` that produces the affected type
   - Fix the template instead
   - Run: `go run ./cmd/modelsdk-codegen <args>` to regenerate
   - `git add internal/codegen/ modelsdk/gen/` and re-attempt commit
4. If the gen file change was accidental: `git restore modelsdk/gen/` and re-attempt commit
```

- [ ] **Step 3: Commit**

```bash
git add .githooks/sop/03-describe-datasource-arch.md .githooks/sop/03-gen-consistency.md
git commit -m "chore(hooks): add SOP files for 03-describe-datasource-arch and 03-gen-consistency"
```

---

### Task 3: Write 03-helpdesk-mdl-golden-sync, 03-protect-golden-mpr SOP files

**Files:**
- Create: `.githooks/sop/03-helpdesk-mdl-golden-sync.md`
- Create: `.githooks/sop/03-protect-golden-mpr.md`

- [ ] **Step 1: Write `03-helpdesk-mdl-golden-sync.md`**

```markdown
# SOP: 03-helpdesk-mdl-golden-sync

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: BSON-affecting file staged without rebuilding"
OR: "COMMIT BLOCKED: testdata/helpdesk-golden/ staged without mdl-examples/..."

## Context variables
- `{TRIGGER_FILE}` — the file that triggered the sync requirement

## Steps (Rule A — golden not staged after BSON change)
1. Confirm the change in `{TRIGGER_FILE}` is complete and correct
2. Add a comment to `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` documenting what changed:
   - Find the relevant section (e.g. `-- MARK: Pages`) and add a `-- Note:` line
3. Run: `make update-helpdesk-golden`
4. Run: `git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden/ testdata/helpdesk-golden-clean/describe-snapshot.mdl`
5. Re-attempt commit

## Steps (Rule B — golden staged without MDL)
1. Open `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
2. Add or update a comment that explains why the golden was rebuilt (e.g. fix description, new example)
3. Run: `make update-helpdesk-golden`
4. Run: `git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden/ testdata/helpdesk-golden-clean/describe-snapshot.mdl`
5. Re-attempt commit
```

- [ ] **Step 2: Write `03-protect-golden-mpr.md`**

```markdown
# SOP: 03-protect-golden-mpr

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: testdata/helpdesk-golden/ MPR staged without describe-snapshot.mdl."

## Context variables
- `{STAGED_MPR}` — the MPR or mprcontents/ file that was directly modified

## Steps
1. Run: `git restore testdata/helpdesk-golden/` — undo any direct edits to the golden MPR
2. Make your change via the MDL source instead:
   - Edit `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
   - Run: `make update-helpdesk-golden`
3. Run: `git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl testdata/helpdesk-golden/ testdata/helpdesk-golden-clean/describe-snapshot.mdl`
4. Re-attempt commit
```

- [ ] **Step 3: Commit**

```bash
git add .githooks/sop/03-helpdesk-mdl-golden-sync.md .githooks/sop/03-protect-golden-mpr.md
git commit -m "chore(hooks): add SOP files for 03-helpdesk-mdl-golden-sync and 03-protect-golden-mpr"
```

---

### Task 4: Write 04-no-raw-bson-in-executor and 05-mx-check-golden SOP files

**Files:**
- Create: `.githooks/sop/04-no-raw-bson-in-executor.md`
- Create: `.githooks/sop/05-mx-check-golden.md`

- [ ] **Step 1: Write `04-no-raw-bson-in-executor.md`**

```markdown
# SOP: 04-no-raw-bson-in-executor

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: executor files must use gen modelsdk, not raw bson:"

## Context variables
- `{VIOLATIONS}` — space-separated list of executor Go files that added raw bson import

## Steps
1. Open each file in `{VIOLATIONS}`
2. Find the `bson.D` / `bson.M` construction site
3. Identify the Mendix type being built (look for `$Type` key or surrounding comments)
4. Find the gen constructor in `modelsdk/gen/` — search for `New<TypeName>()` e.g. `NewMicroflowSource()`
5. Replace the raw bson.D construction with:
   ```go
   elem := genPg.NewTheType()
   elem.SetTheProperty(value)
   // serialise via codec if needed: codec.NewEncoder().Encode(elem)
   ```
6. Run: `CGO_ENABLED=0 go build ./...` — confirm no compile errors
7. Run: `CGO_ENABLED=0 go test ./mdl/executor/... -v`
8. `git add {VIOLATIONS}` and re-attempt commit
```

- [ ] **Step 2: Write `05-mx-check-golden.md`**

```markdown
# SOP: 05-mx-check-golden

## Trigger
Pre-commit blocks with: "COMMIT BLOCKED: mx check found N new error(s)"

## Context variables
- `{ERROR_COUNT}` — total errors found by mx check
- `{BASELINE}` — accepted baseline error count
- `{MX_VERSION}` — Mendix version of the golden MPR

## Steps
1. Run: `~/.mxcli/mxbuild/{MX_VERSION}/modeler/mx check testdata/helpdesk-golden/minimal.mpr 2>&1 | grep '^\[error\]'`
2. Compare output with baseline ({BASELINE} errors expected). The excess lines are the new errors.
3. For each new `[error]` line, identify the domain:
   - `CE0463` → widget definition mismatch (see `.claude/skills/debug-bson.md`)
   - `CE1613` → wrong attribute ref (check `AttributeRef` in BSON write path)
   - `CE0003` → missing version prefix on array field
   - Other → search error code in `docs/` and `.claude/skills/`
4. Fix the relevant executor/backend/gen code
5. Run: `make update-helpdesk-golden`
6. Run: `git add testdata/helpdesk-golden/ mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
7. Re-attempt commit (if still blocked, repeat from step 1)
```

- [ ] **Step 3: Commit**

```bash
git add .githooks/sop/04-no-raw-bson-in-executor.md .githooks/sop/05-mx-check-golden.md
git commit -m "chore(hooks): add SOP files for 04-no-raw-bson-in-executor and 05-mx-check-golden"
```

---

### Task 5: Patch 01-skill-structure.sh and 02-unit-tests.sh

**Files:**
- Modify: `.githooks/checks/01-skill-structure.sh`
- Modify: `.githooks/checks/02-unit-tests.sh`

- [ ] **Step 1: Patch `01-skill-structure.sh` — add SOP + CONTEXT before `exit 1`**

Find the block ending with `exit 1` (lines 14-19 in the current file). Replace:
```sh
    exit 1
fi
```
With:
```sh
    echo "SOP: .githooks/sop/01-skill-structure.md" >&2
    echo "CONTEXT: INVALID_FILES=$(printf '%s' "$invalid" | tr '\n' ' ')" >&2
    exit 1
fi
```

- [ ] **Step 2: Verify the patch looks correct**

```bash
cat .githooks/checks/01-skill-structure.sh
```
Expected: `SOP:` and `CONTEXT:` lines appear immediately before `exit 1`.

- [ ] **Step 3: Patch `02-unit-tests.sh` — add SOP + CONTEXT before `exit 1`**

Find the block ending with `exit 1`. Replace:
```sh
    exit 1
fi
```
With:
```sh
    echo "SOP: .githooks/sop/02-unit-tests.md" >&2
    echo "CONTEXT: LOG_FILE=${LOG_FILE}" >&2
    exit 1
fi
```

- [ ] **Step 4: Commit**

```bash
git add .githooks/checks/01-skill-structure.sh .githooks/checks/02-unit-tests.sh
git commit -m "chore(hooks): add SOP+CONTEXT output to 01-skill-structure and 02-unit-tests"
```

---

### Task 6: Patch 03-describe-datasource-arch.sh and 03-gen-consistency.sh

**Files:**
- Modify: `.githooks/checks/03-describe-datasource-arch.sh`
- Modify: `.githooks/checks/03-gen-consistency.sh`

- [ ] **Step 1: Patch `03-describe-datasource-arch.sh`**

This check uses a `failed` flag and `[ "$failed" -eq 1 ] && exit 1` at the end.
The variables `$f` (last processed file) and `$bad` (last bad fields) are loop-local.
Add a `last_affected` accumulator to capture them, then emit before the final exit.

Replace the inner `if [ -n "$bad" ]; then` block:
```sh
    if [ -n "$bad" ]; then
        echo "COMMIT BLOCKED: $f (mdl/executor/) adds raw BSON field access by string literal." >&2
        echo "  Fields: $bad" >&2
        echo "" >&2
        echo "  Raw map access (ds[\"Field\"], w[\"Field\"]) bypasses type safety and causes" >&2
        echo "  silent nil bugs. Use gen modelsdk types instead:" >&2
        echo "    import genPg \"github.com/mendixlabs/mxcli/modelsdk/gen/pages\"" >&2
        echo "    ms.(*genPg.MicroflowSource).MicroflowSettings()..." >&2
        echo "" >&2
        echo "  If you are migrating existing code (not adding new raw access), use:" >&2
        echo "    // nolint:describe-raw-bson" >&2
        echo "  on the line to opt out (requires justification in PR)." >&2
        failed=1
    fi
```
With:
```sh
    if [ -n "$bad" ]; then
        echo "COMMIT BLOCKED: $f (mdl/executor/) adds raw BSON field access by string literal." >&2
        echo "  Fields: $bad" >&2
        echo "" >&2
        echo "  Raw map access (ds[\"Field\"], w[\"Field\"]) bypasses type safety and causes" >&2
        echo "  silent nil bugs. Use gen modelsdk types instead:" >&2
        echo "    import genPg \"github.com/mendixlabs/mxcli/modelsdk/gen/pages\"" >&2
        echo "    ms.(*genPg.MicroflowSource).MicroflowSettings()..." >&2
        echo "" >&2
        echo "  If you are migrating existing code (not adding new raw access), use:" >&2
        echo "    // nolint:describe-raw-bson" >&2
        echo "  on the line to opt out (requires justification in PR)." >&2
        echo "SOP: .githooks/sop/03-describe-datasource-arch.md" >&2
        echo "CONTEXT: AFFECTED_FILE=${f} BAD_FIELDS=${bad}" >&2
        failed=1
    fi
```

- [ ] **Step 2: Patch `03-gen-consistency.sh` — add SOP + CONTEXT before `exit 1`**

Replace:
```sh
    exit 1
fi
```
With:
```sh
    echo "SOP: .githooks/sop/03-gen-consistency.md" >&2
    echo "CONTEXT: GEN_STAGED=${gen_staged}" >&2
    exit 1
fi
```

- [ ] **Step 3: Commit**

```bash
git add .githooks/checks/03-describe-datasource-arch.sh .githooks/checks/03-gen-consistency.sh
git commit -m "chore(hooks): add SOP+CONTEXT output to 03-describe-datasource-arch and 03-gen-consistency"
```

---

### Task 7: Patch 03-helpdesk-mdl-golden-sync.sh and 03-protect-golden-mpr.sh

**Files:**
- Modify: `.githooks/checks/03-helpdesk-mdl-golden-sync.sh`
- Modify: `.githooks/checks/03-protect-golden-mpr.sh`

- [ ] **Step 1: Patch `03-helpdesk-mdl-golden-sync.sh` — Rule A exit**

The file has two `exit 1` paths. Patch Rule A (first exit, ~line 45):

Replace:
```sh
    echo "" >&2
    exit 1
fi

# Rule B: golden staged but MDL not staged.
```
With:
```sh
    echo "" >&2
    echo "SOP: .githooks/sop/03-helpdesk-mdl-golden-sync.md" >&2
    echo "CONTEXT: TRIGGER_FILE=${staged_trigger}" >&2
    exit 1
fi

# Rule B: golden staged but MDL not staged.
```

- [ ] **Step 2: Patch `03-helpdesk-mdl-golden-sync.sh` — Rule B exit**

Replace the second `exit 1` block:
```sh
    echo "" >&2
    exit 1
fi
```
With:
```sh
    echo "" >&2
    echo "SOP: .githooks/sop/03-helpdesk-mdl-golden-sync.md" >&2
    echo "CONTEXT: TRIGGER_FILE=${staged_golden}" >&2
    exit 1
fi
```

- [ ] **Step 3: Patch `03-protect-golden-mpr.sh` — add SOP + CONTEXT before `exit 1`**

Replace:
```sh
        echo "" >&2
        exit 1
    fi
```
With:
```sh
        echo "" >&2
        echo "SOP: .githooks/sop/03-protect-golden-mpr.md" >&2
        echo "CONTEXT: STAGED_MPR=${staged_mpr}" >&2
        exit 1
    fi
```

- [ ] **Step 4: Commit**

```bash
git add .githooks/checks/03-helpdesk-mdl-golden-sync.sh .githooks/checks/03-protect-golden-mpr.sh
git commit -m "chore(hooks): add SOP+CONTEXT output to 03-helpdesk-mdl-golden-sync and 03-protect-golden-mpr"
```

---

### Task 8: Patch 04-no-raw-bson-in-executor.sh and 05-mx-check-golden.sh

**Files:**
- Modify: `.githooks/checks/04-no-raw-bson-in-executor.sh`
- Modify: `.githooks/checks/05-mx-check-golden.sh`

- [ ] **Step 1: Patch `04-no-raw-bson-in-executor.sh` — add SOP + CONTEXT before `exit 1`**

Replace:
```sh
if [ -n "$VIOLATIONS" ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: executor files must use gen modelsdk, not raw bson:" >&2
    for f in $VIOLATIONS; do
        echo "  $f" >&2
    done
    echo "" >&2
    echo "Replace bson.D/bson.M construction with gen types from modelsdk/gen/." >&2
    echo "Use setRawBSONField() only as a last resort with a comment explaining why." >&2
    exit 1
fi
```
With:
```sh
if [ -n "$VIOLATIONS" ]; then
    echo "" >&2
    echo "COMMIT BLOCKED: executor files must use gen modelsdk, not raw bson:" >&2
    for f in $VIOLATIONS; do
        echo "  $f" >&2
    done
    echo "" >&2
    echo "Replace bson.D/bson.M construction with gen types from modelsdk/gen/." >&2
    echo "Use setRawBSONField() only as a last resort with a comment explaining why." >&2
    echo "SOP: .githooks/sop/04-no-raw-bson-in-executor.md" >&2
    echo "CONTEXT: VIOLATIONS=$(echo $VIOLATIONS | tr ' ' ':')" >&2
    exit 1
fi
```

- [ ] **Step 2: Patch `05-mx-check-golden.sh` — add SOP + CONTEXT before `exit 1`**

Replace:
```sh
    echo "" >&2
    echo "  Fix the errors, then re-run: make update-helpdesk-golden" >&2
    echo "  If errors are intentional (baseline change), update: $BASELINE_FILE" >&2
    echo "" >&2
    exit 1
fi
```
With:
```sh
    echo "" >&2
    echo "  Fix the errors, then re-run: make update-helpdesk-golden" >&2
    echo "  If errors are intentional (baseline change), update: $BASELINE_FILE" >&2
    echo "" >&2
    echo "SOP: .githooks/sop/05-mx-check-golden.md" >&2
    echo "CONTEXT: ERROR_COUNT=${error_count} BASELINE=${baseline} MX_VERSION=${mx_version}" >&2
    exit 1
fi
```

- [ ] **Step 3: Commit**

```bash
git add .githooks/checks/04-no-raw-bson-in-executor.sh .githooks/checks/05-mx-check-golden.sh
git commit -m "chore(hooks): add SOP+CONTEXT output to 04-no-raw-bson-in-executor and 05-mx-check-golden"
```

---

### Task 9: Smoke test all patched checks

**Files:** no changes — validation only

- [ ] **Step 1: Verify `01-skill-structure.sh` emits SOP line**

```bash
cd /tmp && mkdir test-sop && cd test-sop && git init -q
mkdir -p .claude/skills
echo "bad" > .claude/skills/bad-skill.md
git add .claude/skills/bad-skill.md
sh /mnt/data_sdd/gh/mxcli-wt-02/.githooks/checks/01-skill-structure.sh 2>&1 | grep -E "^SOP:|^CONTEXT:"
```
Expected output:
```
SOP: .githooks/sop/01-skill-structure.md
CONTEXT: INVALID_FILES=.claude/skills/bad-skill.md
```

- [ ] **Step 2: Verify `03-helpdesk-mdl-golden-sync.sh` emits SOP line (Rule A)**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
# Stage a BSON-affecting file without staging golden
git stash
touch mdl/backend/mpr/__sop_test.go
git add mdl/backend/mpr/__sop_test.go
sh .githooks/checks/03-helpdesk-mdl-golden-sync.sh 2>&1 | grep -E "^SOP:|^CONTEXT:"
git restore --staged mdl/backend/mpr/__sop_test.go
rm mdl/backend/mpr/__sop_test.go
```
Expected output:
```
SOP: .githooks/sop/03-helpdesk-mdl-golden-sync.md
CONTEXT: TRIGGER_FILE=mdl/backend/mpr/__sop_test.go
```

- [ ] **Step 3: Confirm all 8 SOP files exist**

```bash
ls .githooks/sop/
```
Expected: 8 `.md` files, one per check script stem.

- [ ] **Step 4: Confirm unit tests still pass**

```bash
CGO_ENABLED=0 go test -timeout 120s ./... 2>&1 | tail -5
```
Expected: `ok` lines, no `FAIL`.

- [ ] **Step 5: Final commit (if any cleanup needed)**

```bash
git add .githooks/
git commit -m "chore(hooks): smoke-test verified — SOP integration complete"
```
