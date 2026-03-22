// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

func historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mxcli", "tui_history.json")
}

// LoadHistory returns recent project paths, most recent first.
func LoadHistory() []string {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return nil
	}
	var paths []string
	json.Unmarshal(data, &paths)
	return paths
}

// SaveHistory adds mprPath to the front of the history list (max 10 entries).
func SaveHistory(mprPath string) {
	paths := LoadHistory()
	paths = slices.DeleteFunc(paths, func(p string) bool { return p == mprPath })
	paths = append([]string{mprPath}, paths...)
	if len(paths) > 10 {
		paths = paths[:10]
	}
	os.MkdirAll(filepath.Dir(historyPath()), 0755)
	data, _ := json.Marshal(paths)
	os.WriteFile(historyPath(), data, 0644)
}
