# Design: mxcli local run — Custom Port Flags & Port Conflict Guidance

**Date:** 2026-06-10  
**Status:** Approved

## Problem

`mxcli local run` uses hardcoded ports 8080 (app HTTP) and 8090 (admin API). When these ports are already in use:

- Port 8090: current preflight check catches it, but the error message has no actionable guidance.
- Port 8080: **no check at all** — the runtime crashes with a confusing Java exception.

Users have no way to specify alternative ports.

## Goal

1. Check **both** ports (8080 and 8090) before starting.
2. When a port is taken, report which port, then scan for and suggest a **concretely available** port pair in the error message — ready to copy-paste.
3. Support `--port` and `--admin-port` flags so users can avoid conflicts without trial-and-error.
4. Both PAD layout and deploy layout support custom ports.

## CLI Interface

New flags added to `mxcli local run` (and `mxcli-local run`):

```
--port        int   App HTTP port (default 8080)
--admin-port  int   Admin API port (default 8090)
```

Example usage:

```bash
mxcli local run -p app.mpr --port 8081 --admin-port 8091
```

## Architecture

### 1. `LocalRunOptions` — new fields

```go
type LocalRunOptions struct {
    // existing fields ...
    AppPort   int // 0 = default 8080
    AdminPort int // 0 = default 8090
}

func (o *LocalRunOptions) appPort() int {
    if o.AppPort == 0 { return 8080 }
    return o.AppPort
}
func (o *LocalRunOptions) adminPort() int {
    if o.AdminPort == 0 { return 8090 }
    return o.AdminPort
}
```

Callers that don't set these fields get the existing default behaviour unchanged.

### 2. Port preflight check

`preflightLocal` signature gains port and hint parameters:

```go
// cmdHint is the fragment used in error messages, e.g.:
//   "-p /path/to/app.mpr"  when -p was supplied
//   "--pad-dir /path/to/pad"  when --pad-dir was supplied
func preflightLocal(dir string, stderr io.Writer, isDeployDir bool, appPort, adminPort int, cmdHint string) error
```

`cmdHint` is built by the caller before invoking `preflightLocal`:
- If `-p` was used: `"-p " + projectPath`
- If `--pad-dir` was used: `"--pad-dir " + padDir`

Check order: admin port first, then app port.

When a port is taken:

1. Call `findAvailablePorts(appPort, adminPort)` to locate the next simultaneously-available pair.
2. Use `cmdHint` to compose the ready-to-run suggested command.
3. Return a structured error message.

Error message format (port 8090 taken):

```
port 8090 (admin API) is already in use.

Stop the existing Mendix runtime, or use available ports:

  mxcli local run -p /home/user/projects/MyApp/app.mpr --port 8081 --admin-port 8091
```

If `projectPath` is empty (user passed `--pad-dir`):

```
  mxcli local run --pad-dir /path/to/pad --port 8081 --admin-port 8091
```

### 3. `findAvailablePorts` helper

```go
// findAvailablePorts scans upward from (startApp, startAdmin) and returns
// the first offset where both ports are simultaneously bindable.
func findAvailablePorts(startApp, startAdmin int) (int, int) {
    for offset := 1; offset < 100; offset++ {
        ap, adm := startApp+offset, startAdmin+offset
        if canBind(ap) && canBind(adm) {
            return ap, adm
        }
    }
    return startApp + 1, startAdmin + 1 // fallback for extreme cases
}

func canBind(port int) bool {
    ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
    if err != nil { return false }
    ln.Close()
    return true
}
```

### 4. Deploy layout port injection

`writeDeployHOCON` gains `appPort, adminPort int` parameters. Three sites updated:

- `ApplicationRootUrl` default: `http://localhost:<appPort>/`
- `admin { port = <adminPort> }`
- `runtime.http { port = <appPort> }`

### 5. PAD layout port injection

In `startFromPADLayout`, when either port is non-default, create a `os.MkdirTemp` directory (deferred `os.RemoveAll`), write a HOCON override file there, and append it to the start script's argument list:

```go
if opts.AppPort != 0 || opts.AdminPort != 0 {
    tmpDir, _ := os.MkdirTemp("", "mxcli-local-*")
    defer os.RemoveAll(tmpDir)
    override := fmt.Sprintf(
        "admin { port = %d }\nruntime.http { port = %d }\n",
        opts.adminPort(), opts.appPort(),
    )
    overridePath := filepath.Join(tmpDir, "port-override.conf")
    os.WriteFile(overridePath, []byte(override), 0600)
    cmdArgs = append(cmdArgs, overridePath)
}
```

The PAD `bin/start` script passes all positional arguments as additional HOCON config files to the launcher; later files override earlier ones. No patching of the shipped start script is required.

## Files Changed

| File | Change |
|------|--------|
| `cmd/mxcli-local/cmd_run.go` | Add `--port`, `--admin-port` flags; pass to `LocalRunOptions` |
| `cmd/mxcli/docker/local.go` | `LocalRunOptions` new fields; `appPort()`/`adminPort()` helpers; `preflightLocal` checks both ports; `findAvailablePorts`/`canBind`; deploy HOCON port injection; PAD override file injection |
| `cmd/mxcli/docker/local_test.go` | Tests for port conflict errors, `findAvailablePorts`, both layout port injection |

## Error Handling

- Port scan fallback (offset 1–100 exhausted): suggest `+1` without guaranteeing availability — annotate with a note in the message if needed. In practice, 100 ports scanned is more than sufficient.
- `--pad-dir` path used in suggested command when `-p` is absent.
- No change to HSQLDB lock-file cleanup logic.

## Testing Plan

- Unit: `findAvailablePorts` returns a bindable pair.
- Unit: `preflightLocal` returns correct error when admin port is taken (mock listener).
- Unit: `preflightLocal` returns correct error when app port is taken (mock listener).
- Unit: error messages contain the actual project path, not `app.mpr`.
- Unit: `writeDeployHOCON` writes correct port values when non-default ports are supplied.
- Unit: `startFromPADLayout` appends override config file when ports are non-default; does not append when defaults are used.
