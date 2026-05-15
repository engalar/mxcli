# Widget Round-Trip Baseline

CE0463 regression suite for Stage 3.3.5 Phase 1 (R3 widget tree gen migration).

## Purpose

Every commit that touches widget builders, mutators, or engines in Phase 1 must
run this suite before merging. A failure means the change introduced a BSON
property-count mismatch that Studio Pro will report as CE0463 ("widget definition
changed").

## Files

| Script | Widget | Mutations tested |
|--------|--------|-----------------|
| `datagrid-roundtrip.test.mdl` | DataGrid2 | CREATE with 3 columns, SET Visible, SET Caption, DROP column |
| `combobox-roundtrip.test.mdl` | ComboBox  | CREATE (association mode), SET Label, SET Visible x2 |
| `gallery-roundtrip.test.mdl`  | Gallery   | CREATE with template, SET Class, SET Visible x2 |

## How to run

These scripts are write-path integration tests, not microflow tests. Run each
one against a fresh copy of the fixture:

```bash
for SCRIPT in mdl-examples/widget-roundtrip/*.test.mdl; do
  TMPDIR=$(mktemp -d)
  cp testdata/expr-checker/minimal.mpr "$TMPDIR/test.mpr"
  cp -r testdata/expr-checker/mprcontents "$TMPDIR/mprcontents"
  if ./bin/mxcli exec "$SCRIPT" -p "$TMPDIR/test.mpr" > /dev/null 2>&1; then
    echo "PASS: $(basename $SCRIPT)"
  else
    echo "FAIL: $(basename $SCRIPT)"
    ./bin/mxcli exec "$SCRIPT" -p "$TMPDIR/test.mpr" 2>&1
  fi
  rm -rf "$TMPDIR"
done
```

**Pass criteria:** all three scripts print `PASS` and the overall exit code is 0.

## When to re-run

Re-run after any commit that touches:

- `mdl/executor/cmd_alter_page.go`
- `mdl/executor/cmd_pages_*.go`
- `mdl/backend/mpr/` (page mutator backend)
- `sdk/pages/` or any widget builder / datagrid builder

## Fixture

`testdata/expr-checker/minimal.mpr` is a Mendix 11.6.6 v2 project with Atlas Core
layouts available. The `mprcontents/` folder must be present alongside the `.mpr`
file when using a v2 project.

## Known limitations

- `INSERT AFTER dgName.col { column ... }` (adding a DataGrid2 column via ALTER)
  silently inserts the column outside the datagrid column array; the inserted column
  does not appear in DESCRIBE. This is a known engine limitation tracked separately.
  The baseline uses SET + DROP (not INSERT) for datagrid column mutations.
- Gallery TEXTFILTER is excluded; it has a known CE0463 issue tracked separately.
