// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const maxFilterHistory = 50

func filterHistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mxcli", "tui-filter-history.json")
}

// LoadFilterHistory returns recent filter queries, most recent first.
func LoadFilterHistory() []string {
	data, err := os.ReadFile(filterHistoryPath())
	if err != nil {
		return nil
	}
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		Trace("filterhistory: unmarshal error: %v", err)
		return nil
	}
	return entries
}

// AddFilterHistoryEntry adds a query to the front of filter history (deduped, max 50).
func AddFilterHistoryEntry(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}
	entries := LoadFilterHistory()
	entries = slices.DeleteFunc(entries, func(e string) bool { return e == query })
	entries = append([]string{query}, entries...)
	if len(entries) > maxFilterHistory {
		entries = entries[:maxFilterHistory]
	}
	if err := os.MkdirAll(filepath.Dir(filterHistoryPath()), 0755); err != nil {
		Trace("filterhistory: mkdir error: %v", err)
		return
	}
	data, err := json.Marshal(entries)
	if err != nil {
		Trace("filterhistory: marshal error: %v", err)
		return
	}
	if err := os.WriteFile(filterHistoryPath(), data, 0644); err != nil {
		Trace("filterhistory: write error: %v", err)
	}
}
