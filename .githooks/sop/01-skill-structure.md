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
