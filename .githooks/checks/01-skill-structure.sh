#!/bin/sh
# Validate .claude/skills/ structure: skills must use folder/SKILL.md format.
invalid=""
while IFS= read -r f; do
    base=$(basename "$f")
    [ "$base" = "README.md" ] && continue
    if [ "$base" != "SKILL.md" ] && echo "$f" | grep -qE '\.md$'; then
        invalid="$invalid\n  $f"
    fi
done << EOF
$(git diff --cached --name-only --diff-filter=ACR | grep '^\.claude/skills/')
EOF

if [ -n "$invalid" ]; then
    echo "ERROR: Invalid skill file structure." >&2
    echo "Skills must use folder/SKILL.md format, not loose .md files." >&2
    printf "%b\n" "$invalid" >&2
    echo "Fix: mkdir .claude/skills/<name> && mv <file>.md .claude/skills/<name>/SKILL.md" >&2
    exit 1
fi
