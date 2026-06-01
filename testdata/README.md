# testdata

Test fixture projects used by the mxcli test suite.

## Directory layout

| Directory | Mendix version | MPR format | Purpose |
|-----------|---------------|-----------|---------|
| `corpus-a/` | 11.6.6 | v1 (SQLite, single file) | Lightweight v1 format corpus for basic read/write tests |
| `corpus-b/` | 11.6.4 | v2 (SQLite + `mprcontents/`) | v2 format corpus; one of the `mx check` validation targets |
| `expr-checker/` | 11.6.6 | v2 | Expression checker pipeline (`mxcli expr`); primary `mx check` validation target |
| `helpdesk-clean-11.6.6/` | 11.6.6 | v2 | Blank 11.6.6 base with Atlas Core + widgets — input **A** for helpdesk golden build |
| `helpdesk-golden-11.6.6/` | 11.6.6 | v2 | Helpdesk app fully applied via `helpdesk-app.mdl` — golden reference **B1** (0 mx errors) |
| `helpdesk-clean-11.10.0/` | 11.10.0 | v2 | Blank 11.10.0 base — input for 11.10 golden build and version-compat testing |
| `helpdesk-golden-11.10.0/` | 11.10.0 | v2 | Helpdesk app on 11.10.0 — tracks known pre-existing CE0463/CE0553/CE1571 errors |
| `roundtrip/` | 11.6.6 | v2 | BSON encode/decode roundtrip correctness tests |

## clean vs golden convention

- **`helpdesk-clean-*`** — blank project straight from Studio Pro (`mx create-project`). No user
  MDL has been applied. Used as the starting point (overlay input) for golden builds.
- **`helpdesk-golden-*`** — the result of executing `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`
  against the corresponding clean project. Committed as the BSON reference; never hand-edited.

## Rebuilding the 11.6.6 golden

```bash
make update-helpdesk-golden
git add testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl \
        mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit
```

The `make update-helpdesk-golden` target runs `TestHelpdeskGolden_Update` (Linux + FUSE required)
and regenerates `helpdesk-golden-11.6.6/` plus both `describe-snapshot.mdl` files.

## mx check baseline

`helpdesk-golden-11.6.6/.mx-check-baseline` stores the accepted `[error]` count from
`mx check`. The pre-commit hook (`05-mx-check-golden.sh`) blocks commits that introduce
new errors beyond this baseline. Update the baseline only when errors are intentional.

## Working with testdata projects

```bash
# Apply an MDL change and validate with mx check
./bin/mxcli -p testdata/expr-checker/minimal.mpr -c "create or modify microflow MyFirstModule.ACT_Test () returns Nothing begin return; end;"
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/expr-checker/minimal.mpr 2>&1 | grep -i "StorageLoadException\|Invalid"
git restore testdata/expr-checker/

# Run regression tests (Linux only)
make test-helpdesk-regression
```

**Never run `mxcli exec` directly against `helpdesk-golden-*` projects.** Always use
`make update-helpdesk-golden` to rebuild — it keeps the MPR, mprcontents, and
`describe-snapshot.mdl` consistent. The pre-commit hook enforces this.
