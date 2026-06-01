#!/bin/sh
# Guard: warn when the installed daemon binary is older than the locally-built one.
#
# The mxcli daemon performs all MDL execution and widget BSON generation.
# If the installed daemon is stale (downloaded from a release) while local code
# has changed, test results will not reflect the actual code — rebuilds and
# golden comparisons become unreliable.
#
# This check is WARNING-only (non-blocking) to allow commits when mxbuild is
# unavailable (CI without daemon installed). Developers must run:
#   make install-daemon
# after any change to widget generation, executor, or backend code.

DAEMON_INSTALLED="$HOME/.mxcli/daemon/mxcli-daemon.exe"
# On Linux/macOS the binary has no .exe suffix.
if [ ! -x "$DAEMON_INSTALLED" ]; then
    DAEMON_INSTALLED="$HOME/.mxcli/daemon/mxcli-daemon"
fi
if [ ! -x "$DAEMON_INSTALLED" ]; then
    exit 0  # No installed daemon — skip check.
fi

# Find locally-built daemon.
LOCAL_DAEMON=""
for candidate in "./bin/mxcli-daemon.exe" "./bin/mxcli-daemon"; do
    if [ -x "$candidate" ]; then
        LOCAL_DAEMON="$candidate"
        break
    fi
done
if [ -z "$LOCAL_DAEMON" ]; then
    exit 0  # No local build — skip check (CI may not build the daemon).
fi

installed_ver=$("$DAEMON_INSTALLED" --version 2>/dev/null | head -1)
local_ver=$("$LOCAL_DAEMON" --version 2>/dev/null | head -1)

if [ "$installed_ver" = "$local_ver" ]; then
    exit 0  # Versions match — OK.
fi

# Extract build timestamps for comparison.
installed_ts=$(echo "$installed_ver" | grep -oP '\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z' | head -1)
local_ts=$(echo "$local_ver" | grep -oP '\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z' | head -1)

if [ -n "$installed_ts" ] && [ -n "$local_ts" ] && [ "$local_ts" \> "$installed_ts" ]; then
    # Local build is newer than installed — installed is stale.
    echo "" >&2
    echo "WARNING: installed daemon is older than the local build." >&2
    echo "  Installed: $installed_ver" >&2
    echo "  Local:     $local_ver" >&2
    echo "" >&2
    echo "  Widget generation, executor, and backend changes may not be reflected" >&2
    echo "  in test results until you run:" >&2
    echo "    make install-daemon" >&2
    echo "" >&2
    echo "  Staged files that require an updated daemon:" >&2
    git diff --cached --name-only | grep -E \
        "^modelsdk/widgets/|^mdl/executor/|^mdl/backend/mpr/|^modelsdk/gen/" \
        | head -10 >&2
    echo "" >&2
    # Non-blocking: allow the commit but warn loudly.
fi
