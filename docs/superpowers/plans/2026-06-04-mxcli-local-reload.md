# mxcli local reload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `mxcli local reload [--css|--model-only]` subcommand, refactor M2EE transport behind an interface, and fix three Windows/Git Bash bugs plus the NonInterruptingTimer End Event crash.

**Architecture:** Introduce `M2EECaller` interface with `DirectM2EECaller` (local, plain HTTP) and `DockerExecM2EECaller` (container, docker exec). `ReloadOptions.Caller M2EECaller` replaces the leaky `Direct/Host/Port/Token` fields. `cmd/mxcli-local/cmd_reload.go` wires `DirectM2EECaller` into the shared `docker.Reload()`.

**Tech Stack:** Go 1.24, Cobra, `net/http`, `net/http/httptest` (tests), existing `docker` package.

**Spec:** `docs/superpowers/specs/2026-06-04-mxcli-local-reload-design.md`

---

## File Map

| File | Change |
|------|--------|
| `mdl/executor/cmd_workflows_write_gen2.go` | Fix NonInterruptingTimer: auto-inject End Event |
| `mdl/executor/cmd_alter_workflow.go` | Fix NonInterruptingTimer: same auto-inject |
| `mdl/executor/cmd_workflows_write_gen2_test.go` | Update `NoAutoEnd` test → assert End is injected |
| `cmd/mxcli/docker/m2ee.go` | Add `M2EECaller` interface, `DirectM2EECaller`, `DockerExecM2EECaller`, `NewDockerExecM2EECaller` |
| `cmd/mxcli/docker/m2ee_test.go` | Tests for both caller types |
| `cmd/mxcli/docker/reload.go` | `ReloadOptions.Caller M2EECaller`; remove `Host/Port/Token/Direct`; update `Reload()` and `checkPendingDDL()` |
| `cmd/mxcli/docker/reload_test.go` | Update all tests to use `DirectM2EECaller` |
| `cmd/mxcli/docker.go` | Construct `DockerExecM2EECaller` in reload command |
| `cmd/mxcli-local/cmd_reload.go` | New: `reloadCmd()` using `DirectM2EECaller` |
| `cmd/mxcli-local/main.go` | Register `reloadCmd()` |
| `cmd/mxcli-local/cmd_run.go` | `filepath.Abs(projectPath)` |
| `cmd/mxcli-local/cmd_build.go` | `filepath.Abs(projectPath)` |
| `cmd/mxcli/docker/local.go` | Inject `JAVA_HOME` + `PATH` in `buildLocalEnv()` |
| `cmd/mxcli/docker/detect.go` | Improve `resolveJDK21()` error message with Git Bash hint |

---

## Task 1: Fix NonInterruptingTimer End Event (workflow crash bug)

**Files:**
- Modify: `mdl/executor/cmd_workflows_write_gen2.go:217-227`
- Modify: `mdl/executor/cmd_alter_workflow.go:162-174`
- Modify: `mdl/executor/cmd_workflows_write_gen2_test.go:255-276`

- [ ] **Step 1: Update the failing test to assert End IS injected**

In `mdl/executor/cmd_workflows_write_gen2_test.go`, rename and update `TestBuildBoundaryEventGen_NonInterrupting_NoAutoEnd`:

```go
func TestBuildBoundaryEventGen_NonInterrupting_AutoInjectsEnd(t *testing.T) {
	be := buildBoundaryEventGen(&wfBuildCtx{}, ast.WorkflowBoundaryEventNode{
		EventType: "NonInterruptingTimer",
		Delay:     "${PT12H}",
		Activities: []ast.WorkflowActivityNode{
			&ast.WorkflowCallMicroflowNode{Microflow: ast.QualifiedName{Module: "HD", Name: "ACT_Remind"}},
		},
	})
	v, ok := be.(*genWf.NonInterruptingTimerBoundaryEvent)
	if !ok {
		t.Fatalf("wrong type: %T", be)
	}
	flow, ok := v.Flow().(*genWf.Flow)
	if !ok || flow == nil {
		t.Fatal("expected non-nil Flow")
	}
	acts := flow.ActivitiesItems()
	// CallMicroflow + auto-injected EndWorkflowActivity
	if len(acts) != 2 {
		t.Errorf("expected 2 activities (call + end), got %d", len(acts))
	}
	if acts[1].TypeName() != "Workflows$EndWorkflowActivity" {
		t.Errorf("last activity = %q, want Workflows$EndWorkflowActivity", acts[1].TypeName())
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/ -run TestBuildBoundaryEventGen_NonInterrupting_AutoInjectsEnd -v
```
Expected: `FAIL — expected 2 activities, got 1`

- [ ] **Step 3: Fix `buildBoundaryEventGen` — add auto-inject for NonInterruptingTimer**

In `mdl/executor/cmd_workflows_write_gen2.go`, update the `NonInterruptingTimer` case (line 217):

```go
case "NonInterruptingTimer":
	// CE6665: same rule as interrupting — flow must end with jump or end activity.
	if !endsWithTerminalWorkflowActivity(subActivities) {
		end := genWf.NewEndWorkflowActivity()
		end.SetID(element.ID(types.GenerateID()))
		end.SetCaption("End")
		end.SetName("End")
		subActivities = append(subActivities, end)
	}
	flow := newGenFlowWithActivities(subActivities)
	ev := genWf.NewNonInterruptingTimerBoundaryEvent()
	ev.SetID(id)
	ev.SetIsInterrupting(false)
	ev.SetEventType(be.EventType)
	ev.SetDelay(be.Delay)
	if flow != nil {
		ev.SetFlow(flow)
	}
	return ev
```

- [ ] **Step 4: Fix `cmd_alter_workflow.go` — same auto-inject**

In `mdl/executor/cmd_alter_workflow.go`, update the `InsertBoundaryEventOp` case (line 162):

```go
case *ast.InsertBoundaryEventOp:
	acts := buildAndBindActivitiesGen(ctx, o.Activities)
	// CE6665: both interrupting and non-interrupting timer flows must end
	// with a JumpToActivity or EndWorkflowActivity.
	if !endsWithTerminalWorkflowActivity(acts) {
		end := genWf.NewEndWorkflowActivity()
		end.SetID(element.ID(types.GenerateID()))
		end.SetCaption("End")
		end.SetName("End")
		acts = append(acts, end)
	}
	if err := mutator.InsertBoundaryEventGen(o.ActivityRef, o.AtPosition, o.EventType, o.Delay, acts); err != nil {
		return mdlerrors.NewBackend("insert boundary event", err)
	}
```

- [ ] **Step 5: Run all workflow tests**

```bash
go test ./mdl/executor/ -run TestBuildBoundaryEvent -v
```
Expected: all pass including `TestBuildBoundaryEventGen_NonInterrupting_AutoInjectsEnd`.

- [ ] **Step 6: Run full executor test suite**

```bash
go test ./mdl/executor/ -count=1
```
Expected: PASS, no regressions.

- [ ] **Step 7: Commit**

```bash
git add mdl/executor/cmd_workflows_write_gen2.go \
        mdl/executor/cmd_alter_workflow.go \
        mdl/executor/cmd_workflows_write_gen2_test.go
git commit -m "fix(workflow): auto-inject EndWorkflowActivity in NonInterruptingTimer boundary events

Both CREATE WORKFLOW and ALTER WORKFLOW … insert boundary event now
apply the same CE6665 rule as InterruptingTimer: if the boundary event
body does not end with EndWorkflowActivity or JumpToActivity, one is
appended automatically. Without this, the Mendix runtime throws
'Expected the flow to end with an end event' on startup."
```

---

## Task 2: Add M2EECaller interface and DirectM2EECaller

**Files:**
- Modify: `cmd/mxcli/docker/m2ee.go`
- Modify: `cmd/mxcli/docker/m2ee_test.go`

- [ ] **Step 1: Write failing test for DirectM2EECaller**

Add to `cmd/mxcli/docker/m2ee_test.go`:

```go
func TestDirectM2EECaller_Call(t *testing.T) {
	var gotAction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		gotAction = body["action"].(string)
		auth := r.Header.Get("X-M2EE-Authentication")
		if auth == "" {
			t.Error("missing X-M2EE-Authentication header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{}}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)
	caller := &DirectM2EECaller{Host: host, Port: port, Token: "secret"}
	resp, err := caller.Call("update_styling", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Result != 0 {
		t.Errorf("result = %d, want 0", resp.Result)
	}
	if gotAction != "update_styling" {
		t.Errorf("action = %q, want update_styling", gotAction)
	}
}

func TestDirectM2EECaller_Defaults(t *testing.T) {
	// Verify zero-value Host/Port resolve to localhost:8090
	c := &DirectM2EECaller{Token: "x"}
	if c.host() != "localhost" {
		t.Errorf("host() = %q, want localhost", c.host())
	}
	if c.port() != 8090 {
		t.Errorf("port() = %d, want 8090", c.port())
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./cmd/mxcli/docker/ -run TestDirectM2EECaller -v
```
Expected: `FAIL — DirectM2EECaller undefined`

- [ ] **Step 3: Add interface and DirectM2EECaller to m2ee.go**

Insert after the `M2EEResponse` type in `cmd/mxcli/docker/m2ee.go`:

```go
// M2EECaller abstracts the transport for Mendix admin API calls.
type M2EECaller interface {
	Call(action string, params map[string]any) (*M2EEResponse, error)
}

// DirectM2EECaller sends M2EE requests directly via HTTP.
// Zero values: Host → "localhost", Port → 8090, Timeout → 10s.
type DirectM2EECaller struct {
	Host    string
	Port    int
	Token   string
	Timeout time.Duration
}

func (d *DirectM2EECaller) host() string {
	if d.Host == "" {
		return "localhost"
	}
	return d.Host
}

func (d *DirectM2EECaller) port() int {
	if d.Port == 0 {
		return 8090
	}
	return d.Port
}

func (d *DirectM2EECaller) Call(action string, params map[string]any) (*M2EEResponse, error) {
	timeout := d.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	opts := M2EEOptions{
		Host:    d.host(),
		Port:    d.port(),
		Token:   d.Token,
		Timeout: timeout,
		Direct:  true,
	}
	return callM2EEDirect(opts, action, params)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./cmd/mxcli/docker/ -run TestDirectM2EECaller -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/docker/m2ee.go cmd/mxcli/docker/m2ee_test.go
git commit -m "feat(m2ee): add M2EECaller interface and DirectM2EECaller"
```

---

## Task 3: Add DockerExecM2EECaller with token resolution

**Files:**
- Modify: `cmd/mxcli/docker/m2ee.go`
- Modify: `cmd/mxcli/docker/m2ee_test.go`

- [ ] **Step 1: Write failing test for NewDockerExecM2EECaller token resolution**

Add to `cmd/mxcli/docker/m2ee_test.go`:

```go
func TestNewDockerExecM2EECaller_ExplicitToken(t *testing.T) {
	caller, err := NewDockerExecM2EECaller(t.TempDir(), "mytoken")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.Token != "mytoken" {
		t.Errorf("Token = %q, want mytoken", caller.Token)
	}
}

func TestNewDockerExecM2EECaller_EnvToken(t *testing.T) {
	t.Setenv("M2EE_ADMIN_PASS", "envtoken")
	caller, err := NewDockerExecM2EECaller(t.TempDir(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.Token != "envtoken" {
		t.Errorf("Token = %q, want envtoken", caller.Token)
	}
}

func TestNewDockerExecM2EECaller_NoToken_Error(t *testing.T) {
	t.Setenv("M2EE_ADMIN_PASS", "")
	_, err := NewDockerExecM2EECaller(t.TempDir(), "")
	if err == nil {
		t.Fatal("expected error when no token available")
	}
}

func TestNewDockerExecM2EECaller_DotEnvToken(t *testing.T) {
	t.Setenv("M2EE_ADMIN_PASS", "")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("M2EE_ADMIN_PASS=filetoken\n"), 0600)
	caller, err := NewDockerExecM2EECaller(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.Token != "filetoken" {
		t.Errorf("Token = %q, want filetoken", caller.Token)
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./cmd/mxcli/docker/ -run TestNewDockerExecM2EECaller -v
```
Expected: `FAIL — NewDockerExecM2EECaller undefined`

- [ ] **Step 3: Add DockerExecM2EECaller and constructor to m2ee.go**

Append to `cmd/mxcli/docker/m2ee.go`:

```go
// DockerExecM2EECaller sends M2EE requests via docker compose exec.
// Zero value Port → containerAdminPort (8090).
type DockerExecM2EECaller struct {
	DockerDir string
	Token     string
	Port      int
}

func (d *DockerExecM2EECaller) Call(action string, params map[string]any) (*M2EEResponse, error) {
	port := d.Port
	if port == 0 {
		port = containerAdminPort
	}
	opts := M2EEOptions{Token: d.Token, Port: port}
	return callM2EEViaDocker(opts, d.DockerDir, action, params)
}

// NewDockerExecM2EECaller creates a DockerExecM2EECaller with token resolved from:
//  1. explicitToken (if non-empty)
//  2. M2EE_ADMIN_PASS environment variable
//  3. M2EE_ADMIN_PASS in dockerDir/.env file
//
// Returns an error if no token is found.
func NewDockerExecM2EECaller(dockerDir, explicitToken string) (*DockerExecM2EECaller, error) {
	token := explicitToken
	if token == "" {
		token = os.Getenv("M2EE_ADMIN_PASS")
	}
	if token == "" {
		envPath := filepath.Join(dockerDir, ".env")
		if parsed, err := parseEnvFile(envPath); err == nil {
			token = parsed["M2EE_ADMIN_PASS"]
		}
	}
	if token == "" {
		return nil, fmt.Errorf("admin password required: set --token, M2EE_ADMIN_PASS env var, or configure .docker/.env")
	}
	return &DockerExecM2EECaller{DockerDir: dockerDir, Token: token, Port: containerAdminPort}, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./cmd/mxcli/docker/ -run TestNewDockerExecM2EECaller -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/docker/m2ee.go cmd/mxcli/docker/m2ee_test.go
git commit -m "feat(m2ee): add DockerExecM2EECaller and NewDockerExecM2EECaller"
```

---

## Task 4: Refactor ReloadOptions to use M2EECaller

**Files:**
- Modify: `cmd/mxcli/docker/reload.go`
- Modify: `cmd/mxcli/docker/reload_test.go`

- [ ] **Step 1: Update ReloadOptions and Reload() in reload.go**

Replace the `ReloadOptions` struct and `Reload()` function body:

```go
// ReloadOptions configures the docker reload command.
type ReloadOptions struct {
	ProjectPath string
	MxBuildPath string
	SkipCheck   bool
	SkipBuild   bool
	CSSOnly     bool
	Caller      M2EECaller
	Stdout      io.Writer
	Stderr      io.Writer
}

// Reload performs a hot reload of the Mendix application.
func Reload(opts ReloadOptions) error {
	w := opts.Stdout
	if w == nil {
		w = os.Stdout
	}

	// CSS-only mode: just update styling
	if opts.CSSOnly {
		resp, err := opts.Caller.Call("update_styling", nil)
		if err != nil {
			return fmt.Errorf("update_styling failed: %w", err)
		}
		if errMsg := resp.M2EEError(); errMsg != "" {
			return fmt.Errorf("update_styling failed: %s", errMsg)
		}
		fmt.Fprintln(w, "Styling updated.")
		return nil
	}

	// Build step (unless --model-only)
	if !opts.SkipBuild {
		buildOpts := BuildOptions{
			ProjectPath: opts.ProjectPath,
			MxBuildPath: opts.MxBuildPath,
			SkipCheck:   opts.SkipCheck,
			Stdout:      w,
		}
		if err := Build(buildOpts); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
		fmt.Fprintln(w, "")
	}

	// Reload model
	fmt.Fprintln(w, "Reloading model...")
	resp, err := opts.Caller.Call("reload_model", nil)
	if err != nil {
		return fmt.Errorf("reload_model failed: %w", err)
	}
	if errMsg := resp.M2EEError(); errMsg != "" {
		return fmt.Errorf("reload failed: %s", errMsg)
	}

	if durationStr := extractReloadDuration(resp.Feedback()); durationStr != "" {
		fmt.Fprintf(w, "Model reloaded (%s).\n", durationStr)
	} else {
		fmt.Fprintln(w, "Model reloaded.")
	}

	if pending := checkPendingDDL(opts.Caller); pending != "" {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "WARNING: Database schema changes detected after reload.")
		fmt.Fprintln(w, "  The model was reloaded, but new entities or attributes require")
		fmt.Fprintln(w, "  a database schema update that hot-reload cannot perform.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  Pending DDL:")
		for _, line := range strings.Split(pending, "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Fprintf(w, "    %s\n", line)
			}
		}
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  Fix: run 'mxcli docker up --fresh' to restart with schema sync.")
	}

	return nil
}

// checkPendingDDL queries the runtime for pending DDL commands.
func checkPendingDDL(caller M2EECaller) string {
	resp, err := caller.Call("get_ddl_commands", nil)
	if err != nil {
		return ""
	}
	if resp.Result != 0 {
		return ""
	}
	feedback := resp.Feedback()
	if feedback == nil {
		return ""
	}
	ddl, ok := feedback["ddl_commands"]
	if !ok {
		return ""
	}
	ddlStr, ok := ddl.(string)
	if !ok || strings.TrimSpace(ddlStr) == "" {
		return ""
	}
	return ddlStr
}
```

- [ ] **Step 2: Update reload_test.go — replace Host/Port/Token/Direct with Caller**

Update every test that constructs `ReloadOptions`. Pattern for all tests:

```go
// Old:
opts := ReloadOptions{
    CSSOnly: true,
    Host:    host,
    Port:    port,
    Token:   "testpass",
    Direct:  true,
    Stdout:  &stdout,
}

// New:
opts := ReloadOptions{
    CSSOnly: true,
    Caller:  &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
    Stdout:  &stdout,
}
```

Apply this pattern to all 7 test functions: `TestReload_CSSOnly`, `TestReload_ModelOnly`, `TestReload_ModelOnly_WithDuration`, `TestReload_CSSOnly_Error`, `TestReload_ModelOnly_PendingDDL`, `TestReload_ModelOnly_ReloadError`, and any others that use `ReloadOptions`.

- [ ] **Step 3: Run reload tests**

```bash
go test ./cmd/mxcli/docker/ -run TestReload -v
```
Expected: PASS

- [ ] **Step 4: Run full docker package tests**

```bash
go test ./cmd/mxcli/docker/ -count=1
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/mxcli/docker/reload.go cmd/mxcli/docker/reload_test.go
git commit -m "refactor(reload): replace Host/Port/Token/Direct with M2EECaller interface

ReloadOptions.Caller M2EECaller replaces the four transport fields.
Reload() and checkPendingDDL() depend only on the abstraction.
Existing tests updated to use DirectM2EECaller with httptest.Server."
```

---

## Task 5: Update docker.go to use DockerExecM2EECaller

**Files:**
- Modify: `cmd/mxcli/docker.go:425-458`

- [ ] **Step 1: Replace M2EEOptions construction in dockerReloadCmd**

Replace lines 425–457 in `cmd/mxcli/docker.go`:

```go
Run: func(cmd *cobra.Command, args []string) {
    projectPath, _ := cmd.Flags().GetString("project")
    if projectPath == "" {
        fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
        os.Exit(1)
    }

    mxbuildPath, _ := cmd.Flags().GetString("mxbuild-path")
    skipCheck, _ := cmd.Flags().GetBool("skip-check")
    modelOnly, _ := cmd.Flags().GetBool("model-only")
    cssOnly, _ := cmd.Flags().GetBool("css")
    token, _ := cmd.Flags().GetString("token")

    dockerDir := filepath.Join(filepath.Dir(projectPath), ".docker")
    caller, err := docker.NewDockerExecM2EECaller(dockerDir, token)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    opts := docker.ReloadOptions{
        ProjectPath: projectPath,
        MxBuildPath: mxbuildPath,
        SkipCheck:   skipCheck,
        SkipBuild:   modelOnly,
        CSSOnly:     cssOnly,
        Caller:      caller,
        Stdout:      os.Stdout,
        Stderr:      os.Stderr,
    }

    if err := docker.Reload(opts); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
},
```

Also remove the now-unused `--host`, `--port`, `--direct` flag registrations (lines ~533–535) if they exist, or leave them as deprecated no-ops if other code references them.

- [ ] **Step 2: Verify build compiles**

```bash
go build ./cmd/mxcli/...
```
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/docker.go
git commit -m "feat(docker): use DockerExecM2EECaller in docker reload command"
```

---

## Task 6: Add cmd/mxcli-local/cmd_reload.go

**Files:**
- Create: `cmd/mxcli-local/cmd_reload.go`

- [ ] **Step 1: Create the file**

```go
// cmd/mxcli-local/cmd_reload.go
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
	"github.com/spf13/cobra"
)

func reloadCmd() *cobra.Command {
	var (
		projectPath   string
		adminPassword string
		skipCheck     bool
		modelOnly     bool
		cssOnly       bool
	)

	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Hot reload the running local Mendix app (no restart required)",
		Long: `Hot reload the Mendix application running via 'mxcli local run'.

Modes:
  (default)      Build the PAD package, then call reload_model
  --model-only   Skip build, just call reload_model (PAD already up to date)
  --css          Update styling only (instant, no build or model reload)

The admin password must match what was passed to 'mxcli local run --admin-password'.
Default password: Admin123!`,
		Example: `  mxcli local reload -p app.mpr
  mxcli local reload -p app.mpr --model-only
  mxcli local reload -p app.mpr --css
  mxcli local reload -p app.mpr --skip-check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --css and --model-only don't need the project path (no build step).
			if !cssOnly && !modelOnly && projectPath == "" {
				return fmt.Errorf("--project (-p) is required for full reload; use --model-only or --css to skip the build step")
			}

			if projectPath != "" {
				if abs, err := filepath.Abs(projectPath); err == nil {
					projectPath = abs
				}
			}

			token := adminPassword
			if token == "" {
				token = "Admin123!"
			}

			caller := &docker.DirectM2EECaller{
				Host:  "localhost",
				Port:  8090,
				Token: token,
			}

			return docker.Reload(docker.ReloadOptions{
				ProjectPath: projectPath,
				SkipCheck:   skipCheck,
				SkipBuild:   modelOnly,
				CSSOnly:     cssOnly,
				Caller:      caller,
				Stdout:      os.Stdout,
				Stderr:      os.Stderr,
			})
		},
	}

	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Path to .mpr file (required for full reload)")
	cmd.Flags().StringVar(&adminPassword, "admin-password", "", "M2EE admin password (default: Admin123!)")
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "Skip mx check before build")
	cmd.Flags().BoolVar(&modelOnly, "model-only", false, "Skip build, just call reload_model")
	cmd.Flags().BoolVar(&cssOnly, "css", false, "CSS hot reload only (update_styling, no build)")
	return cmd
}
```

- [ ] **Step 2: Verify compile**

```bash
go build ./cmd/mxcli-local/
```
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli-local/cmd_reload.go
git commit -m "feat(local): add cmd_reload.go with DirectM2EECaller"
```

---

## Task 7: Register reloadCmd in main.go

**Files:**
- Modify: `cmd/mxcli-local/main.go:22`

- [ ] **Step 1: Add reloadCmd() to root**

```go
root.AddCommand(buildCmd(), runCmd(), reloadCmd())
```

- [ ] **Step 2: Verify help output**

```bash
go run ./cmd/mxcli-local/ --help
```
Expected: output includes `reload` in the command list.

```bash
go run ./cmd/mxcli-local/ reload --help
```
Expected: shows all four flags (`--admin-password`, `--css`, `--model-only`, `--skip-check`).

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli-local/main.go
git commit -m "feat(local): register reload subcommand"
```

---

## Task 8: Fix filepath.Abs for Git Bash relative paths

**Files:**
- Modify: `cmd/mxcli-local/cmd_run.go`
- Modify: `cmd/mxcli-local/cmd_build.go`

- [ ] **Step 1: Add filepath.Abs to cmd_run.go**

In `cmd_run.go` inside `RunE`, add after the flag reads:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if projectPath != "" {
        if abs, err := filepath.Abs(projectPath); err == nil {
            projectPath = abs
        }
    }
    if padDir != "" {
        if abs, err := filepath.Abs(padDir); err == nil {
            padDir = abs
        }
    }
    // ... rest of existing code unchanged
```

- [ ] **Step 2: Add filepath.Abs to cmd_build.go**

In `cmd_build.go` inside `RunE`, add after the flag reads:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if projectPath != "" {
        if abs, err := filepath.Abs(projectPath); err == nil {
            projectPath = abs
        }
    }
    // ... rest of existing code unchanged
```

- [ ] **Step 3: Build and spot-check**

```bash
go build ./cmd/mxcli-local/
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli-local/cmd_run.go cmd/mxcli-local/cmd_build.go
git commit -m "fix(local): resolve project path to absolute before use

Git Bash passes POSIX-style relative paths that Windows APIs reject.
filepath.Abs() normalises the path early, before any exec or file ops."
```

---

## Task 9: Inject JAVA_HOME into local run environment

**Files:**
- Modify: `cmd/mxcli/docker/local.go`
- Modify: `cmd/mxcli/docker/detect.go`

- [ ] **Step 1: Update buildLocalEnv to inject JAVA_HOME and prepend to PATH**

In `cmd/mxcli/docker/local.go`, update `buildLocalEnv`:

```go
func buildLocalEnv(dbURL, adminPassword string) []string {
	if adminPassword == "" {
		adminPassword = "Admin123!"
	}
	env := []string{
		"ADMIN_ADMINPASSWORD=" + adminPassword,
		"RUNTIME_ADMINUSER_PASSWORD=" + adminPassword,
	}
	if dbURL != "" {
		if dbEnv, err := parseDBURL(dbURL); err == nil {
			env = append(env, dbEnv...)
		}
	}

	// Inject JAVA_HOME so bin/start can find java even when it's not in
	// the shell PATH (common in Git Bash on Windows).
	if javaHome, err := resolveJDK21(); err == nil {
		env = append(env, "JAVA_HOME="+javaHome)
		javaBin := filepath.Join(javaHome, "bin")
		if currentPath := os.Getenv("PATH"); currentPath != "" {
			env = append(env, "PATH="+javaBin+string(os.PathListSeparator)+currentPath)
		} else {
			env = append(env, "PATH="+javaBin)
		}
	}

	return env
}
```

Add `"path/filepath"` to the import if not already present (it is, for existing `padDir` usage).

- [ ] **Step 2: Improve error message in resolveJDK21()**

In `cmd/mxcli/docker/detect.go`, update the final return:

```go
return "", fmt.Errorf(
	"JDK 21 not found; install JDK 21 and set JAVA_HOME, or in Git Bash run:\n" +
		"  export PATH=\"/c/Program Files/Eclipse Adoptium/jdk-21.x.y.z-hotspot/bin:$PATH\"",
)
```

- [ ] **Step 3: Run detect tests**

```bash
go test ./cmd/mxcli/docker/ -run TestResolveJDK -v 2>/dev/null || echo "no JDK tests, checking build"
go build ./cmd/mxcli/docker/
```
Expected: builds cleanly.

- [ ] **Step 4: Commit**

```bash
git add cmd/mxcli/docker/local.go cmd/mxcli/docker/detect.go
git commit -m "fix(local): inject JAVA_HOME into runtime process environment

bin/start calls java directly; if Java is not in the shell PATH
(common in Git Bash), the runtime exits with status 9009.
buildLocalEnv() now resolves JAVA_HOME via resolveJDK21() and
prepends JAVA_HOME/bin to PATH for the child process.
Also improves the JDK-not-found error to include a Git Bash hint."
```

---

## Task 10: Final integration check

- [ ] **Step 1: Full build**

```bash
make build
```
Expected: exits 0.

- [ ] **Step 2: Full test suite**

```bash
make test
```
Expected: no new failures.

- [ ] **Step 3: Smoke-test reload help**

```bash
./bin/mxcli local reload --help
```
Expected output includes:
```
Hot reload the running local Mendix app (no restart required)
...
  --admin-password string   M2EE admin password (default: Admin123!)
  --css                     CSS hot reload only (update_styling, no build)
  --model-only              Skip build, just call reload_model
  --skip-check              Skip mx check before build
```

- [ ] **Step 4: Commit if any stray changes**

```bash
git status
# commit anything uncommitted
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Task |
|-----------------|------|
| `M2EECaller` interface | Task 2 |
| `DirectM2EECaller` | Task 2 |
| `DockerExecM2EECaller` + token resolution | Task 3 |
| `ReloadOptions.Caller` replaces Direct/Host/Port/Token | Task 4 |
| `docker.go` uses `DockerExecM2EECaller` | Task 5 |
| `mxcli local reload` subcommand with 4 flags | Tasks 6–7 |
| `--admin-password` default `Admin123!` | Task 6 |
| `--skip-check` for full reload | Task 6 |
| Missing `-p` error message | Task 6 |
| `filepath.Abs` for Git Bash paths | Task 8 |
| JAVA_HOME injection for local run | Task 9 |
| NonInterruptingTimer End Event fix | Task 1 |

All spec requirements covered. No placeholders. Types consistent across tasks (`DirectM2EECaller`, `DockerExecM2EECaller`, `NewDockerExecM2EECaller`, `ReloadOptions.Caller`).
