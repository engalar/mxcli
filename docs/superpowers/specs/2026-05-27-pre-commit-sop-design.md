# Pre-Commit SOP Integration Design

**Date:** 2026-05-27  
**Status:** Approved

## Problem

When a pre-commit check blocks a commit, the AI (Claude Code) reads the block
message and must figure out the repair steps on its own. Two failure modes:

1. **No SOP**: AI guesses the repair, sometimes incorrectly.
2. **Missing context**: AI knows the rule that was violated but not which specific
   file triggered it, so steps like "git add X" require extra investigation.

Root cause observed: `NanoflowSource` BSON format bug survived for months because
the DataGrid2 nanoflow datasource path was never covered by the describe snapshot —
and no check pointed at the SOP for extending the snapshot page list.

## Goal

Every pre-commit block is a **self-contained trigger event**: it contains the SOP
path (where to find the repair instructions) and the CONTEXT variables (what
specifically triggered it). The AI reads the block, reads the SOP, substitutes
variables, executes steps — zero human involvement needed.

## Architecture

### Directory layout

```
.githooks/
├── pre-commit                  # main entry (unchanged)
├── checks/                     # 8 check scripts (unchanged except 2-line addition)
│   ├── 01-skill-structure.sh
│   ├── 02-unit-tests.sh
│   ├── 03-describe-datasource-arch.sh
│   ├── 03-gen-consistency.sh
│   ├── 03-helpdesk-mdl-golden-sync.sh
│   ├── 03-protect-golden-mpr.sh
│   ├── 04-no-raw-bson-in-executor.sh
│   └── 05-mx-check-golden.sh
└── sop/                        # NEW: one SOP per check, same stem name
    ├── 01-skill-structure.md
    ├── 02-unit-tests.md
    ├── 03-describe-datasource-arch.md
    ├── 03-gen-consistency.md
    ├── 03-helpdesk-mdl-golden-sync.md
    ├── 03-protect-golden-mpr.md
    ├── 04-no-raw-bson-in-executor.md
    └── 05-mx-check-golden.md
```

Naming rule: `sop/<stem>.md` where `<stem>` = check filename without `.sh`.
AI can derive the SOP path from the check name without any extra configuration.

### Block message format

Every `exit 1` in a check script is preceded by exactly two new lines:

```sh
echo "SOP: .githooks/sop/<stem>.md" >&2
echo "CONTEXT: VAR1=value1 VAR2=value2" >&2
exit 1
```

The AI protocol:
1. Detect `SOP:` line in pre-commit output
2. `Read` the referenced file
3. Extract `CONTEXT:` key=value pairs
4. Substitute `{VAR}` placeholders in SOP steps
5. Execute steps in order

### CONTEXT variables per check

| Check | CONTEXT variables |
|-------|------------------|
| `01-skill-structure` | `INVALID_FILES=<space-separated list>` |
| `02-unit-tests` | `LOG_FILE=.test-fail.log` |
| `03-describe-datasource-arch` | `AFFECTED_FILE=<file> BAD_FIELDS=<fields>` |
| `03-gen-consistency` | `GEN_STAGED=<count>` |
| `03-helpdesk-mdl-golden-sync` | `TRIGGER_FILE=<file>` |
| `03-protect-golden-mpr` | `STAGED_MPR=<file>` |
| `04-no-raw-bson-in-executor` | `VIOLATIONS=<space-separated files>` |
| `05-mx-check-golden` | `ERROR_COUNT=<n> BASELINE=<b> MX_VERSION=<version>` |

All variables are already computed inside each script — zero new logic needed,
only printing them.

## SOP file format

```markdown
# SOP: <check-stem>

## Trigger
Pre-commit blocks with: "<first words of the COMMIT BLOCKED message>"

## Context variables
- `{VAR}` — description

## Steps
1. concrete command or action
2. ...
```

Steps use `{VAR}` placeholders that the AI substitutes from the `CONTEXT:` line.
Commands in steps are ready to copy-run; no decision trees, no conditionals.

## Check script changes

Minimal: insert two `echo` lines before each `exit 1`. No logic changes.

Example (`03-helpdesk-mdl-golden-sync.sh`, Rule A block):

```sh
# before
    echo "  3. git add ${MDL_PATH} testdata/helpdesk-golden/" >&2
    echo "" >&2
    exit 1

# after
    echo "  3. git add ${MDL_PATH} testdata/helpdesk-golden/" >&2
    echo "" >&2
    echo "SOP: .githooks/sop/03-helpdesk-mdl-golden-sync.md" >&2
    echo "CONTEXT: TRIGGER_FILE=${staged_trigger}" >&2
    exit 1
```

Scripts with multiple `exit 1` paths (e.g. `03-helpdesk-mdl-golden-sync.sh` has
Rule A and Rule B) each get their own `SOP:` + `CONTEXT:` pair with appropriate
variable values for that path.

## Scope

- **In scope**: all 8 existing check scripts + 8 new SOP files
- **Out of scope**: new checks, changes to SOP content after initial write,
  auto-executing SOPs (AI reads and acts, but does not invoke tools automatically
  without user confirmation in interactive mode)

## Success criteria

- Every `exit 1` in every check script is preceded by `SOP:` + `CONTEXT:` lines
- Every SOP file exists with Trigger / Context variables / Steps sections
- `make test` passes with the new hook files in place
- A simulated block (run check script manually with a bad staged file) shows the
  SOP path and CONTEXT in the output
