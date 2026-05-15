#!/usr/bin/env bash
set -euo pipefail

project_file="${1:-testdata/expr-checker/minimal.mpr}"
contents_dir="${2:-testdata/expr-checker/mprcontents}"
scripts_glob="${3:-mdl-examples/widget-roundtrip/*.test.mdl}"
log_file="${4:-/tmp/widget-roundtrip-baseline.txt}"

mkdir -p "$(dirname "$log_file")"
: >"$log_file"

for script in $scripts_glob; do
  tmpdir="$(mktemp -d)"
  cp "$project_file" "$tmpdir/test.mpr"
  cp -r "$contents_dir" "$tmpdir/mprcontents"

  echo "Running $(basename "$script")" | tee -a "$log_file"
  if ./bin/mxcli exec "$script" -p "$tmpdir/test.mpr" >>"$log_file" 2>&1; then
    echo "PASS $(basename "$script")" | tee -a "$log_file"
  else
    status=$?
    echo "FAIL $(basename "$script")" | tee -a "$log_file"
    cat "$log_file"
    rm -rf "$tmpdir"
    exit "$status"
  fi

  rm -rf "$tmpdir"
done

echo "All widget round-trip scripts passed" | tee -a "$log_file"
