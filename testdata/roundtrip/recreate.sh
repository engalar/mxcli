#!/usr/bin/env bash
# Recreate testdata/roundtrip/roundtrip.mpr from scratch.
# Requires: mxcli binary built (make build), Mendix 11.6.6 mxbuild available.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MXCLI="${REPO_ROOT}/mxcli"
SEED="${SCRIPT_DIR}/seed.mdl"
OUT_MPR="${SCRIPT_DIR}/roundtrip.mpr"

# Fall back to bin/mxcli if root mxcli not found
if [ ! -f "${MXCLI}" ]; then
  MXCLI="${REPO_ROOT}/bin/mxcli"
fi

# Build mxcli if needed
if [ ! -f "${MXCLI}" ]; then
  echo "Building mxcli..."
  (cd "${REPO_ROOT}" && make build)
  MXCLI="${REPO_ROOT}/bin/mxcli"
fi

# Find mx binary
MX_BIN="$(find "${HOME}/.mxcli/mxbuild/11.6.6" -name "mx" -type f 2>/dev/null | head -1 || true)"
if [ -z "${MX_BIN}" ]; then
  echo "Downloading mxbuild 11.6.6..."
  "${MXCLI}" setup mxbuild -p "${REPO_ROOT}/testdata/expr-checker/minimal.mpr"
  MX_BIN="$(find "${HOME}/.mxcli/mxbuild/11.6.6" -name "mx" -type f | head -1)"
fi

echo "Using mx: ${MX_BIN}"

# Create blank project in temp dir
TMPDIR_WORK="$(mktemp -d /tmp/roundtrip-XXXXXX)"
trap 'rm -rf "${TMPDIR_WORK}"' EXIT

echo "Creating blank Mendix 11.6.6 project..."
"${MX_BIN}" create-project --app-name roundtrip --output-dir "${TMPDIR_WORK}/project"

# Find the created MPR
CREATED_MPR="$(find "${TMPDIR_WORK}" -name "*.mpr" -not -name "*.snap" | head -1)"
if [ -z "${CREATED_MPR}" ]; then
  echo "ERROR: no MPR found in ${TMPDIR_WORK}"
  ls -la "${TMPDIR_WORK}/"
  exit 1
fi
echo "Created MPR: ${CREATED_MPR}"

# Helper: run an MDL command against the project (each call reconnects, so the
# reader cache is always fresh — required for enum-typed entity attributes).
mx() {
  "${MXCLI}" -p "${CREATED_MPR}" -c "$1"
}

# Create RoundtripModule
mx "create module RoundtripModule;"

# Module roles (before entities so GRANTs can reference them)
mx "create module role RoundtripModule.Viewer;"
mx "create module role RoundtripModule.Editor;"

# Enumeration (must be committed before entity references it)
mx "create or modify enumeration RoundtripModule.Status (
    Active caption 'Active',
    Inactive caption 'Inactive',
    Pending caption 'Pending'
);"

# Entities (each in its own invocation so the enumeration is visible)
mx "create or modify persistent entity RoundtripModule.Category (
    Label: String(100) not null
);"

mx "create or modify persistent entity RoundtripModule.Item (
    Name: String(200) not null,
    Price: Decimal,
    Status: enumeration(RoundtripModule.Status)
);"

# Entity access rules
mx "grant RoundtripModule.Viewer on RoundtripModule.Category (read *);"
mx "grant RoundtripModule.Editor on RoundtripModule.Category (create, read *, write *);"
mx "grant RoundtripModule.Viewer on RoundtripModule.Item (read *);"
mx "grant RoundtripModule.Editor on RoundtripModule.Item (create, read *, write *);"

# Association
mx "create or modify association RoundtripModule.Item_Category
    from RoundtripModule.Item to RoundtripModule.Category;"

# Constants
mx "create or modify constant RoundtripModule.ApiBaseUrl
type string
default 'https://example.com';"

mx "create or modify constant RoundtripModule.MaxRetries
type integer
default 3;"

# User roles (after module roles)
mx "create user role BasicUser (RoundtripModule.Viewer);"

mx "create user role PowerUser (RoundtripModule.Editor);"

# Java Action (stub — body is ignored by Studio Pro at load time)
mx "create java action RoundtripModule.ExternalCall (InputText: String) returns String
as \$\$
// stub
\$\$;"

# Copy result to testdata/roundtrip/
cp "${CREATED_MPR}" "${OUT_MPR}"
MPR_DIR="$(dirname "${CREATED_MPR}")"
if [ -d "${MPR_DIR}/mprcontents" ]; then
  rm -rf "${SCRIPT_DIR}/mprcontents"
  cp -r "${MPR_DIR}/mprcontents" "${SCRIPT_DIR}/mprcontents"
fi

echo "Done: ${OUT_MPR}"
