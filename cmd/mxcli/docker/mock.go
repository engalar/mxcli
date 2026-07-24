// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type MockOptions struct {
	SpecPath string
	Port     int
}

func StartMockServer(opts MockOptions) (int, error) {
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		return 0, fmt.Errorf("npx not found: %w", err)
	}

	specPath := opts.SpecPath
	if !filepath.IsAbs(specPath) {
		abs, err := filepath.Abs(specPath)
		if err != nil {
			return 0, fmt.Errorf("resolving spec path: %w", err)
		}
		specPath = abs
	}

	port := opts.Port
	if port == 0 {
		port = 4000
	}

	cmd := exec.Command(npxPath, "@stoplight/prism-cli", "mock", specPath, "-p", fmt.Sprintf("%d", port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	CmdWithPdeathsig(cmd)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting prism: %w", err)
	}

	return cmd.Process.Pid, nil
}

type MockLock struct {
	PID       int       `json:"pid"`
	Port      int       `json:"port"`
	SpecPath  string    `json:"specPath"`
	StartedAt time.Time `json:"startedAt"`
}

func mockLockPath(projectDir string) string {
	return filepath.Join(projectDir, ".mxcli", "mock.lock")
}

func WriteMockLock(projectDir string, l *MockLock) error {
	if existing, err := ReadMockLock(projectDir); err == nil {
		if proc, err := os.FindProcess(existing.PID); err == nil && proc.Signal(syscall.Signal(0)) == nil {
			return fmt.Errorf("mock server already running (PID %d on port %d)", existing.PID, existing.Port)
		}
		_ = RemoveMockLock(projectDir)
	}
	dir := filepath.Dir(mockLockPath(projectDir))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mockLockPath(projectDir), data, 0644)
}

func ReadMockLock(projectDir string) (*MockLock, error) {
	data, err := os.ReadFile(mockLockPath(projectDir))
	if err != nil {
		return nil, err
	}
	var l MockLock
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func RemoveMockLock(projectDir string) error {
	path := mockLockPath(projectDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func KillMockServer(projectDir string) error {
	l, err := ReadMockLock(projectDir)
	if err != nil {
		return fmt.Errorf("no mock server PID file found")
	}
	proc, err := os.FindProcess(l.PID)
	if err != nil {
		_ = RemoveMockLock(projectDir)
		return fmt.Errorf("mock server process not found")
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		_ = RemoveMockLock(projectDir)
		return fmt.Errorf("mock server not running (stale PID)")
	}
	_ = syscall.Kill(-l.PID, syscall.SIGTERM)
	_ = RemoveMockLock(projectDir)
	return nil
}

func MockServerStatus(projectDir string) (*MockLock, error) {
	l, err := ReadMockLock(projectDir)
	if err != nil {
		return nil, fmt.Errorf("mock server is not running")
	}
	proc, err := os.FindProcess(l.PID)
	if err != nil || proc.Signal(syscall.Signal(0)) != nil {
		_ = RemoveMockLock(projectDir)
		return nil, fmt.Errorf("mock server is not running (stale PID)")
	}
	return l, nil
}
