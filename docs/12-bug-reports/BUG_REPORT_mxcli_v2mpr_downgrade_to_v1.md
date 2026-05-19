# Bug Report: mxcli exec silently downgrades MPR v2 to v1

## Summary

When `mxcli exec` (or any write operation) is run against a Mendix 11 MPR v2 project, mxcli silently converts the entire project from v2 format (split into `mprcontents/*.mxunit` files) to v1 format (monolithic SQLite with inline `Unit.Contents` column). This is the root cause of the Point/Size format incompatibility and the Page Title type error reported in the two companion bug reports.

## Environment

- **mxcli version**: v0.9.0-707-gb247683e (2026-05-19) and earlier
- **Mendix version**: 11.6.6
- **MPR format affected**: v2 (mxunit-based)
- **Platform**: Linux (Ubuntu 22.04)

## MPR Format Background

Mendix 11 uses **MPR v2** as the canonical format:
- Main `.mpr` file: small SQLite (~131KB) containing only a unit index (`Unit` table with `UnitID`, `ContainerID`, `ContainmentName`, `ContentsHash`, `ContentsConflicts` — **no `Contents` column**)
- `mprcontents/<aa>/<bb>/<uuid>.mxunit` files: raw BSON blobs, one per model unit (microflow, page, entity, etc.)
- Studio Pro reads/writes `ContentsHash` in the main SQLite and stores actual BSON in the `.mxunit` files

**MPR v1** (legacy) stores everything inline:
- Main `.mpr` file: large SQLite (~21MB) with `Unit.Contents` column containing the BSON inline
- No `mprcontents/` directory

## Steps to Reproduce

1. Start with a clean MPR v2 project (git-tracked, small `.mpr` + `mprcontents/` directory):
   ```bash
   ls -la project.mpr          # ~131KB
   ls mprcontents/ | wc -l     # 250 subdirectories
   python3 -c "import sqlite3; c=sqlite3.connect('project.mpr'); print([r[0] for r in c.execute('SELECT name FROM sqlite_master WHERE type=\\'table\\'').fetchall()])"
   # ['_MetaData', 'Unit', '_Transaction']  — no Contents column
   ```

2. Run any `mxcli exec` command that writes to the project:
   ```bash
   ./mxcli exec script.mdl -p project.mpr
   ```

3. Inspect the result:
   ```bash
   ls -la project.mpr          # Now ~21MB
   python3 -c "import sqlite3; c=sqlite3.connect('project.mpr'); print([d[0] for d in c.execute('SELECT * FROM Unit LIMIT 1').description])"
   # ['UnitID', 'ContainerID', 'ContainmentName', 'TreeConflict', 'ContentsHash', 'ContentsConflicts', 'Contents']
   # ↑ 'Contents' column added — project is now v1
   ```

## Actual Result

After any `mxcli exec`:
- `project.mpr` grows from ~131KB to ~21MB
- `Unit` table gains an inline `Contents` column (v1 format)
- `mprcontents/*.mxunit` files are no longer authoritative (contents have been merged into the main SQLite)
- New objects written by mxcli use v1-style BSON serialization, which is incompatible with the Mendix 11 mx validator

## Downstream Effects

This format downgrade directly causes two other bugs (separate reports):

| Bug | Symptom | Root cause |
|-----|---------|------------|
| Point/Size format | `RelativeMiddlePoint = "200 200"` crashes mx check | mxcli v1 serializer uses space; v2 expects semicolon |
| Page Title type | `ClientTemplate` cannot convert to `Text` | mxcli v1 serializer uses wrong type for page Title |

Both downstream bugs disappear if the v2 format is preserved, because Studio Pro's `.mxunit` files use the correct format.

## Expected Result

`mxcli exec` should preserve the v2 format:
- Keep the main `.mpr` as a small index-only SQLite (no `Contents` column)
- For new objects: create new `.mxunit` files in `mprcontents/<first2>/<next2>/<uuid>.mxunit`
- For modified objects: update the corresponding `.mxunit` file in-place, then update `ContentsHash` in the main SQLite
- The `ContentsHash` column should store `base64(SHA-256(mxunit_file_content))`

## Priority

**Critical** — this is the root cause of all mxcli write-path incompatibilities with Mendix 11.6.6 MPR v2. Fixing this single bug will resolve the Point/Size format bug and the Page Title type bug as natural consequences, since the existing `.mxunit` files already use the correct BSON format.

## Suggested Fix

In the mxcli write/exec code path:

1. **Detect MPR format** before writing: check whether the `Unit` table has a `Contents` column (v1) or not (v2).
2. **v2 write path**: instead of adding content to `Unit.Contents`, serialize the BSON to a new or updated `.mxunit` file in `mprcontents/<aa>/<bb>/<uuid>.mxunit`, then update `ContentsHash` in the `Unit` table.
3. **Do not inline content** when the project is v2. The v1 write path (inline `Contents`) should only be used for v1 projects.
4. **BSON format source of truth**: for v2, use the same BSON format as Studio Pro writes (semicolons for Points, `Text` not `ClientTemplate` for page titles, etc.). The existing `.mxunit` files in a Studio Pro project are the reference for correct format.

## Related Bug Reports

- `BUG_REPORT_mxcli_v2mpr_bson_point_format.md` — downstream effect of this bug
- `BUG_REPORT_mxcli_v2mpr_microflow_parameter_type_and_sequenceflow.md` — partially downstream; SequenceFlow issue may be separate
