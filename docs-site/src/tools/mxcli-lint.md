# mxcli lint

The `mxcli lint` command runs all lint rules (built-in Go rules and Starlark rules) against a Mendix project and reports findings.

## Basic Usage

```bash
# Lint a project
mxcli lint -p app.mpr

# With colored terminal output
mxcli lint -p app.mpr --color
```

## Output Formats

### Text (Default)

Human-readable output with rule IDs, severity, and messages:

```bash
mxcli lint -p app.mpr
```

### JSON

Machine-readable JSON output for CI integration:

```bash
mxcli lint -p app.mpr --format json
```

### SARIF

SARIF (Static Analysis Results Interchange Format) output for GitHub Code Scanning and other SARIF-compatible tools:

```bash
mxcli lint -p app.mpr --format sarif > results.sarif
```

This file can be uploaded to GitHub as a code scanning result.

## Options

### List Available Rules

```bash
mxcli lint -p app.mpr --list-rules
```

Shows all available rules with their IDs, names, severity, and source (built-in or Starlark).

### Exclude Modules

System and marketplace modules often trigger false positives. Exclude them:

```bash
mxcli lint -p app.mpr --exclude System --exclude Administration
```

### Colored Output

```bash
mxcli lint -p app.mpr --color
```

## CI Integration

### GitHub Actions

```yaml
- name: Lint Mendix project
  run: |
    mxcli lint -p app.mpr --format sarif > results.sarif

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

### JSON Processing

```bash
# Count findings by severity
mxcli lint -p app.mpr --format json | jq 'group_by(.severity) | map({severity: .[0].severity, count: length})'

# Filter for errors only
mxcli lint -p app.mpr --format json | jq '[.[] | select(.severity == "error")]'
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No findings |
| 1 | Findings reported |
| 2 | Error running linter |
