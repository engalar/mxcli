package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type RunLock struct {
	PID       int       `json:"pid"`
	AppPort   int       `json:"appPort"`
	AdminPort int       `json:"adminPort"`
	Password  string    `json:"password"`
	StartedAt time.Time `json:"startedAt"`
}

func lockPath(projectDir string) string {
	return filepath.Join(projectDir, ".mxcli", "run.lock")
}

func ReadLock(projectDir string) (*RunLock, error) {
	data, err := os.ReadFile(lockPath(projectDir))
	if err != nil {
		return nil, err
	}
	var l RunLock
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (l *RunLock) Alive() bool {
	process, err := os.FindProcess(l.PID)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func (l *RunLock) Kill() error {
	process, err := os.FindProcess(l.PID)
	if err != nil {
		return nil
	}
	_ = process.Signal(syscall.SIGTERM)

	// Wait up to 5 seconds for graceful shutdown
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if process.Signal(syscall.Signal(0)) != nil {
			return nil // process exited
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Force kill
	return process.Signal(syscall.SIGKILL)
}

func WriteLock(projectDir string, l *RunLock) error {
	dir := filepath.Dir(lockPath(projectDir))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(lockPath(projectDir), data, 0644)
}

func RemoveLock(projectDir string) error {
	path := lockPath(projectDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func KillByProject(projectDir string) error {
	l, err := ReadLock(projectDir)
	if err != nil {
		return nil // no lock file, nothing to kill
	}
	if !l.Alive() {
		RemoveLock(projectDir)
		return nil
	}
	if err := l.Kill(); err != nil {
		return fmt.Errorf("kill runtime: %w", err)
	}
	RemoveLock(projectDir)
	return nil
}

func ensureDirectory(path string) string {
	dir := filepath.Dir(path)
	if p := os.Getenv("MXCLI_PROJECT_DIR"); p != "" {
		dir = p
	}
	return filepath.Join(dir, ".mxcli", "run.lock")
}

func projectDirFromProjectPath(projectPath string) string {
	return filepath.Dir(projectPath)
}

// DefaultRunOptions returns RunOptions with sensible defaults for the TUI.
func DefaultRunOptions(projectPath string) RunOptions {
	projectDir := filepath.Dir(projectPath)
	return RunOptions{
		DeployDir: filepath.Join(projectDir, "deployment"),
		CmdHint:   "-p " + projectPath,
	}
}

func (l *RunLock) Summary() string {
	extra := ""
	if l.Password != "" {
		extra = fmt.Sprintf(" pwd=%s", l.Password)
	}
	return fmt.Sprintf("PID=%d port=%d/admin=%d uptime=%s%s",
		l.PID, l.AppPort, l.AdminPort,
		time.Since(l.StartedAt).Round(time.Second), extra)
}
