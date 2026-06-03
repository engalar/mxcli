// cmd/mxcli/docker/local.go
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ProcessStarter abstracts exec.Cmd execution for testing.
type ProcessStarter interface {
	Run(cmd *exec.Cmd) error
}

// RealStarter executes the command for real (used in production).
type RealStarter struct{}

func (r *RealStarter) Run(cmd *exec.Cmd) error { return cmd.Run() }

// LocalRunOptions configures StartLocal.
type LocalRunOptions struct {
	// PadDir is the PAD output directory (.docker/build/ by default).
	PadDir string
	// DB is an optional postgres:// URL. Empty = use PAD defaults (HSQLDB).
	DB string
	// Stdout for runtime log output (defaults to os.Stdout).
	Stdout io.Writer
	// Stderr for runtime error output (defaults to os.Stderr).
	Stderr io.Writer
	// Starter is the process runner. Nil = RealStarter (exec.Cmd.Run).
	Starter ProcessStarter
}

// StartLocal starts the Mendix runtime from a pre-built PAD directory without Docker.
// It execs bin/start (Linux/macOS) or cmd.exe /c bin\start.bat (Windows),
// blocking until the process exits (Ctrl+C stops it).
func StartLocal(opts LocalRunOptions) error {
	if !hasExtractedPADLayout(opts.PadDir) {
		return fmt.Errorf("no PAD found at %s — run 'mxcli local build -p app.mpr' first", opts.PadDir)
	}

	cmdArgs, err := resolveStartScript(opts.PadDir)
	if err != nil {
		return err
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = opts.PadDir
	cmd.Env = append(os.Environ(), buildLocalEnv(opts.DB)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	starter := opts.Starter
	if starter == nil {
		starter = &RealStarter{}
	}
	return starter.Run(cmd)
}

// resolveStartScript returns [binary, args...] for the platform start script.
func resolveStartScript(padDir string) ([]string, error) {
	switch runtime.GOOS {
	case "windows":
		bat := filepath.Join(padDir, "bin", "start.bat")
		if _, err := os.Stat(bat); err == nil {
			return []string{"cmd.exe", "/c", bat}, nil
		}
		ps1 := filepath.Join(padDir, "bin", "start.ps1")
		if _, err := os.Stat(ps1); err == nil {
			return []string{"powershell.exe", "-ExecutionPolicy", "Bypass", "-File", ps1}, nil
		}
		return nil, fmt.Errorf("no Windows start script (start.bat or start.ps1) found in %s/bin/", padDir)
	default:
		sh := filepath.Join(padDir, "bin", "start")
		if _, err := os.Stat(sh); err != nil {
			return nil, fmt.Errorf("start script not found at %s", sh)
		}
		return []string{sh}, nil
	}
}

// buildLocalEnv returns environment variables required by the Mendix runtime.
// ADMIN_ADMINPASSWORD: M2EE admin API auth (required by runtimelauncher).
// RUNTIME_ADMINUSER_PASSWORD: creates/updates MxAdmin login user on startup.
func buildLocalEnv(dbURL string) []string {
	env := []string{
		"ADMIN_ADMINPASSWORD=Admin123!",
		"RUNTIME_ADMINUSER_PASSWORD=Admin123!",
	}
	if dbURL != "" {
		if dbEnv, err := parseDBURL(dbURL); err == nil {
			env = append(env, dbEnv...)
		}
	}
	return env
}

// parseDBURL converts a postgres:// URL into RUNTIME_PARAMS_* env vars
// consumed by etc/variables.conf inside the PAD.
func parseDBURL(rawURL string) ([]string, error) {
	// Implemented in Task 3.
	return nil, fmt.Errorf("parseDBURL: not implemented")
}
