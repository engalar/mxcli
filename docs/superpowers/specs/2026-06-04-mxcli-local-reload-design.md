# mxcli local reload — Design Spec

**Date:** 2026-06-04  
**Status:** Approved  
**Scope:** Add `mxcli local reload` subcommand + fix three Windows/Git Bash bugs in `mxcli local`

---

## 1. Goal

Bring the three hot-reload modes of `mxcli docker reload` into `mxcli local`, so users running without Docker can iterate without restarting the runtime process:

| Command | What it does |
|---------|-------------|
| `mxcli local reload -p app.mpr` | `local build` + `reload_model` (full rebuild, no restart) |
| `mxcli local reload -p app.mpr --model-only` | `reload_model` only (PAD already up to date) |
| `mxcli local reload -p app.mpr --css` | `update_styling` only (instant, no build) |

Additionally, fix three bugs discovered during Windows/Git Bash testing of `mxcli local`.

---

## 2. Architecture

### 2.1 Core Problem with Existing Code

`docker.Reload()` calls `CallM2EE(M2EEOptions, ...)` directly. `M2EEOptions.Direct bool` is a leaky abstraction that exposes transport decisions to callers. Adding local support by setting `Direct: true` would violate DIP and couple local code to docker internals.

### 2.2 Solution: M2EECaller Interface (Dependency Inversion)

Define an interface in `m2ee.go` and inject it into `Reload()`:

```go
// M2EECaller abstracts the transport for Mendix admin API calls.
type M2EECaller interface {
    Call(action string, params map[string]any) (*M2EEResponse, error)
}
```

Two concrete implementations:

**`DirectM2EECaller`** — for `mxcli local reload`
```go
type DirectM2EECaller struct {
    Host    string        // default: "localhost"
    Port    int           // default: 8090
    Token   string        // default: "Admin123!" (matches buildLocalEnv default)
    Timeout time.Duration // default: 10s
}
```
Wraps the existing `callM2EEDirect()` function.

**`DockerExecM2EECaller`** — for `mxcli docker reload`
```go
type DockerExecM2EECaller struct {
    DockerDir string
    Token     string
    Port      int // container-internal port, always 8090
}
```
Wraps the existing `callM2EEViaDocker()` function.

The existing `CallM2EE(M2EEOptions, ...)` function is **retained unchanged** for backward compatibility with `oql.go` and other callers not yet migrated.

### 2.3 ReloadOptions Change

```go
type ReloadOptions struct {
    ProjectPath string
    MxBuildPath string
    SkipCheck   bool
    SkipBuild   bool
    CSSOnly     bool
    Caller      M2EECaller // replaces: Host, Port, Token, Direct
    Stdout      io.Writer
    Stderr      io.Writer
}
```

`Reload()` body replaces all `CallM2EE(m2eeOpts, action, params)` calls with `opts.Caller.Call(action, params)`. `checkPendingDDL()` receives `M2EECaller` as a parameter instead of `M2EEOptions`.

### 2.4 File Map

| File | Change |
|------|--------|
| `cmd/mxcli/docker/m2ee.go` | Add `M2EECaller` interface, `DirectM2EECaller`, `DockerExecM2EECaller` |
| `cmd/mxcli/docker/reload.go` | `ReloadOptions.Caller M2EECaller`; replace `CallM2EE` calls; update `checkPendingDDL` |
| `cmd/mxcli/docker.go` | Construct `DockerExecM2EECaller`, pass to `Reload()` |
| `cmd/mxcli-local/cmd_reload.go` | New: `reloadCmd()` with `DirectM2EECaller` |
| `cmd/mxcli-local/main.go` | Register `reloadCmd()` |

---

## 3. New Command: `mxcli local reload`

```
Usage:
  mxcli local reload [flags]

Flags:
  -p, --project        string   Path to .mpr file (required for full reload)
      --admin-password string   M2EE admin password (default: Admin123!)
      --skip-check              Skip mx check before build
      --model-only              Skip build, call reload_model only
      --css                     CSS hot reload only (update_styling)
```

**Flag semantics:**
- `-p` is required only for full reload (build step needs the MPR path). `--css` and `--model-only` do not require it.
- `--admin-password` defaults to `Admin123!`, matching the default set by `mxcli local run`.
- `--skip-check` applies only when the build step runs (ignored with `--model-only` and `--css`).

**Error messages:**
- Missing `-p` for full reload: `--project (-p) is required for full reload; use --model-only or --css to skip the build step`
- Cannot connect: `cannot connect to Mendix admin API at localhost:8090 — is the app running? Start with 'mxcli local run -p app.mpr'`

**Post-start hint** (added to `local run` output):
```
✅ Runtime started. To hot-reload: mxcli local reload -p <path>
```

---

## 4. Bug Fixes (Windows / Git Bash)

### Bug 1: Java Not Found in Git Bash PATH

**Symptom:** `java: command not found`, `mxcli local build` exits with status 9009.  
**Root cause:** Git Bash does not inherit Windows registry PATH entries; JDK installed via system installer is invisible.  
**Fix:** In `resolveJDK21()`, after the standard `JAVA_HOME` and PATH lookups fail, additionally glob common Windows installation paths:
```
C:/Program Files/Eclipse Adoptium/jdk-21*/bin/java.exe
C:/Program Files/Microsoft/jdk-21*/bin/java.exe
C:/Program Files/Java/jdk-21*/bin/java.exe
```
If still not found, error message includes the Git Bash fix:
```
JDK 21 not found. In Git Bash, run:
  export PATH="/c/Program Files/Eclipse Adoptium/jdk-21.x.y.z-hotspot/bin:$PATH"
```

### Bug 2: --admin-password Not Effective for reload on Windows

**Symptom:** `M2EE_ADMIN_PASS is not set` when running `mxcli local reload`.  
**Root cause:** `buildLocalEnv()` injects `ADMIN_ADMINPASSWORD` into the runtime child process. `resolveM2EEDefaults()` reads `M2EE_ADMIN_PASS` from the caller's shell — a different variable name. The runtime password and the caller token are never automatically linked.  
**Fix:** `DirectM2EECaller` defaults `Token` to `"Admin123!"` — the same value `buildLocalEnv()` uses when `--admin-password` is omitted. Users who change the default must pass `--admin-password` to both `local run` and `local reload` (documented in help text).

### Bug 3: Relative Paths Fail in Git Bash on Windows

**Symptom:** `mxcli local run -p minimal.mpr` → "系统找不到指定的路径".  
**Root cause:** Git Bash passes POSIX-style relative paths; the launcher proxies them to `mxcli-local` which passes them to `exec.Command` on Windows without normalization.  
**Fix:** In `cmd_run.go`, `cmd_build.go`, and the new `cmd_reload.go`, resolve `projectPath` to absolute at the top of `RunE`:
```go
if projectPath != "" {
    if abs, err := filepath.Abs(projectPath); err == nil {
        projectPath = abs
    }
}
```

---

## 5. Out of Scope (Separate Bugs)

| Issue | Description | Action |
|-------|-------------|--------|
| CE0126 | Timer boundary event expression stored in BSON but not read back by `mx check` | File separate bug ticket; investigate DESCRIBE vs mx check discrepancy in boundary event BSON serialization |
| CE0495 | Workflow activity name uniqueness not enforced by mxcli | File separate bug ticket; consider adding uniqueness check in MDL executor |
| Workflow End Event missing in non-interrupting boundary branches | MDL executor doesn't generate End Event nodes in boundary event sub-paths | File separate bug ticket |

---

## 6. Testing

### Unit Tests

**`M2EECaller` mock** (shared test helper):
```go
type mockCaller struct {
    calls []string
    resp  *docker.M2EEResponse
    err   error
}
func (m *mockCaller) Call(action string, _ map[string]any) (*docker.M2EEResponse, error) {
    m.calls = append(m.calls, action)
    return m.resp, m.err
}
```

Test cases for `Reload()`:
- `--css`: mock called once with `"update_styling"`, build not invoked
- `--model-only`: mock called with `"reload_model"` then `"get_ddl_commands"`, build not invoked
- Full reload: build runs, then mock called with `"reload_model"` + `"get_ddl_commands"`
- DDL warning path: `get_ddl_commands` returns non-empty DDL, warning printed

**`DirectM2EECaller`**: tested with `httptest.NewServer` serving a mock M2EE JSON response.  
**`DockerExecM2EECaller`**: tested with a fake `docker` binary on PATH.

### Integration
`mxcli local reload --css` in CI: start a real local runtime, call `--css`, verify exit 0. Requires JDK 21 on the CI agent.

---

## 7. Rollout

1. Implement and release with the next `local-v*` tag.
2. Launcher auto-downloads the new `mxcli-local` on first `mxcli local reload` invocation.
3. No changes to daemon or main launcher binary required.
