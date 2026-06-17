package build

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallMPK copies mpkPath into <projectDir>/widgets/.
func InstallMPK(mpkPath, projectPath string) error {
	widgetsDir := filepath.Join(filepath.Dir(projectPath), "widgets")
	if err := os.MkdirAll(widgetsDir, 0755); err != nil {
		return fmt.Errorf("create widgets/ dir: %w", err)
	}
	dst := filepath.Join(widgetsDir, filepath.Base(mpkPath))
	data, err := os.ReadFile(mpkPath)
	if err != nil {
		return fmt.Errorf("read mpk: %w", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write mpk: %w", err)
	}
	fmt.Printf("Installed → %s\n", dst)
	return nil
}

// FindMPKInCwd globs *.mpk in the current working directory.
func FindMPKInCwd() (string, error) {
	matches, err := filepath.Glob("*.mpk")
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no .mpk file found — run 'mxcli widget build' first")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple .mpk files found (%v) — specify one with --mpk", matches)
	}
}
