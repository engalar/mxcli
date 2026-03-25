# Diff

mxcli provides two diff commands for comparing MDL scripts against project state and viewing local changes in MPR v2 projects.

## mxcli diff

Compares an MDL script against the current project state, showing what would change if the script were executed. This is a dry-run preview.

**Usage:**

```bash
mxcli diff -p app.mpr changes.mdl
```

This shows:
- Elements that would be created (new entities, microflows, pages)
- Elements that would be modified (changed attributes, altered properties)
- Elements that would be removed (DROP statements)

Use `mxcli diff` to review changes before applying them, especially when working with AI-generated scripts.

## mxcli diff-local

Compares local changes against a git reference for MPR v2 projects. MPR v2 (Mendix >= 10.18) stores documents as individual files in an `mprcontents/` folder, making git diff feasible.

**Usage:**

```bash
# Compare against HEAD (latest commit)
mxcli diff-local -p app.mpr --ref HEAD

# Compare against a specific commit
mxcli diff-local -p app.mpr --ref HEAD~1

# Compare against a branch
mxcli diff-local -p app.mpr --ref main
```

### MPR v2 Requirement

`diff-local` only works with MPR v2 format (Mendix >= 10.18), where documents are stored as individual files. MPR v1 projects store everything in a single SQLite database, making file-level git diff impractical.

## Workflow

### Review Before Applying

```bash
# 1. Generate MDL changes
# (AI assistant creates changes.mdl)

# 2. Review what would change
mxcli diff -p app.mpr changes.mdl

# 3. If satisfied, apply
mxcli exec changes.mdl -p app.mpr
```

### Track Changes Over Time

```bash
# After making changes, see what changed since last commit
mxcli diff-local -p app.mpr --ref HEAD

# See changes since two commits ago
mxcli diff-local -p app.mpr --ref HEAD~2
```
