# Plan 6: Cobra Command Layer Standardization

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate three inconsistencies in `cmd/mxcli/`: (1) mixed `Run`+`os.Exit(1)` vs `RunE`+`return error`, (2) mixed centralized vs decentralized `AddCommand` registration, (3) package-level global variables.

**Architecture:** Standardize every command to `RunE`, move all `init()` registrations to `main.go`, wrap package globals into a `Config` struct.

**Tech Stack:** Go, `cmd/mxcli/`, Cobra CLI

## Global Constraints

- Every commit must compile: `go build ./cmd/mxcli/...`
- Every commit must pass: existing tests must pass
- Zero behavior changes — only code structure changes
- Commands that call `os.Exit(1)` on error must migrate to `return error`; the root `Execute()` handles exit

---

### Task 1: Migrate all `Run` handlers to `RunE`

**Files:**
- Modify: All `cmd_*.go` files in `cmd/mxcli/`

- [ ] **Step 1: Find all `Run:` handlers**

```bash
rg -n '^\s+Run:\s*func' cmd/mxcli/ --no-filename | head -30
```

Catalog each one and determine if it uses `os.Exit(1)`.

- [ ] **Step 2: Migrate commands without `os.Exit(1)` first**

Pattern:
```go
// Before:
var cmdFoo = &cobra.Command{
    Run: func(cmd *cobra.Command, args []string) {
        if err := doFoo(); err != nil {
            fmt.Fprintln(os.Stderr, err)
            return
        }
    },
}

// After:
var cmdFoo = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        return doFoo()
    },
}
```

- [ ] **Step 3: Migrate commands with `os.Exit(1)`**

Each `os.Exit(1)` in a `Run` closure becomes `return err`:

```go
// Before:
var checkCmd = &cobra.Command{
    Run: func(cmd *cobra.Command, args []string) {
        projectPath, _ := cmd.Flags().GetString("project")
        if projectPath == "" {
            fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
            os.Exit(1)
        }
        if err := doCheck(projectPath); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
    },
}

// After:
var checkCmd = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        projectPath, _ := cmd.Flags().GetString("project")
        if projectPath == "" {
            return fmt.Errorf("--project (-p) is required")
        }
        return doCheck(projectPath)
    },
}
```

List of commands to migrate (expected ~15):
- `checkCmd`, `lintCmd`, `reportCmd`, `diffCmd`, `dockerCmd` variants
- `gitCmd` variants
- `oqlCmd`, `sqlCmd`
- `widgetCmd` variants
- Any other `Run:` handlers

- [ ] **Step 4: Batch build after each group**

```bash
go build ./cmd/mxcli/...
```

- [ ] **Step 5: Commit each group**

```bash
git add -A
git commit -m "refactor: migrate <group> commands from Run+os.Exit to RunE"
```

---

### Task 2: Standardize command registration to central `init()`

**Files:**
- Modify: `cmd/mxcli/main.go` (add `AddCommand` calls)
- Modify: `cmd_sql.go`, `cmd_explain.go`, `cmd_eval.go` etc. (remove decentralized `init()`)

- [ ] **Step 1: Find all decentralized `init()` registrations**

```bash
rg -n 'func init\(\)' cmd/mxcli/ --no-filename | grep -v main.go
rg -n 'rootCmd\.AddCommand' cmd/mxcli/ --no-filename | grep -v main.go
```

Identify files that register themselves outside `main.go`'s `init()`.

- [ ] **Step 2: Move each registration into `main.go`**

For each file with a decentralized `init()` that calls `rootCmd.AddCommand(xxxCmd)`:

1. Copy the `AddCommand` call into `main.go`'s `init()` function
2. Remove the `init()` function from the source file

```go
// In main.go init():
rootCmd.AddCommand(sqlCmd)     // was in cmd_sql.go init()
rootCmd.AddCommand(explainCmd)  // was in cmd_explain.go init()
rootCmd.AddCommand(evalCmd)     // was in cmd_eval.go init()
```

- [ ] **Step 3: Build and test**

```bash
go build ./cmd/mxcli/...
go test ./cmd/mxcli/... -count=1 -timeout 120s
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor: consolidate command registration in main.go init()"
```

---

### Task 3: Wrap package-level globals into Config struct

**Files:**
- Modify: `cmd/mxcli/main.go`

- [ ] **Step 1: Identify all package-level globals**

```bash
rg -n '^\s+var\s+(global|version|Build)' cmd/mxcli/ --no-filename
```

Likely candidates:
- `version`, `Version`, `BuildTime`, `CommitSHA`
- `globalJSONFlag`
- `globalVerboseLevel`

- [ ] **Step 2: Create `Config` struct for runtime config**

```go
// At top of main.go (or new file cmd_config.go)
type Config struct {
    JSONOutput    bool
    VerboseLevel  int
    Version       string
    BuildTime     string
    CommitSHA     string
}

var config Config
```

Replace package-level vars with `config.XXX` accesses.

For build info (`version`, `BuildTime`, `CommitSHA`), keep them as `var` but group:

```go
var (
    BuildVersion = "dev"
    BuildTime    string
    CommitSHA    string
)
```

- [ ] **Step 3: Update `PersistentPreRunE` to set config instead of globals**

- [ ] **Step 4: Build and test**

```bash
go build ./cmd/mxcli/...
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: group package-level globals into Config struct"
```

---

### Task 4: Final verification

- [ ] **Step 1: Full build and test cycle**

```bash
go build ./cmd/mxcli/...
go test ./cmd/mxcli/... -count=1 -timeout 180s
make build 2>&1 | tail -5
```

- [ ] **Step 2: Verify no `os.Exit(1)` remains in command handlers**

```bash
rg 'os\.Exit\(1\)' cmd/mxcli/ --no-filename | grep -v 'main()' | grep -v 'init()'
```

Should return zero results.

- [ ] **Step 3: Verify all commands have `RunE`**

```bash
rg 'Run:' cmd/mxcli/ --no-filename | grep -v 'RunE'
```

Should return zero results (or only legitimate non-RunE patterns).

- [ ] **Step 4: Verify all registrations in `main.go`**

```bash
rg 'rootCmd\.AddCommand|rootCmd\.AddGroup|parentCmd\.AddCommand' cmd/mxcli/ --no-filename
```

Should only appear in `main.go`.
