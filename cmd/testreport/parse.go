// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

// TestEvent は `go test -json` が出力する1行のイベント。
type TestEvent struct {
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"`
	Output  string    `json:"Output"`
	Time    time.Time `json:"Time"`
}

// parseTestResults reads go test -json output and returns all events.
func parseTestResults(r io.Reader) ([]TestEvent, error) {
	var events []TestEvent
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20) // 1MB per line
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev TestEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue // skip malformed lines
		}
		events = append(events, ev)
	}
	return events, sc.Err()
}

// LayerSummary holds aggregated stats for one test layer.
type LayerSummary struct {
	ID       string
	Name     string
	Priority int
	Pass     int
	Fail     int
	Elapsed  float64 // seconds
	Failures []FailureDetail
}

// FailureDetail holds info about a single failing test.
type FailureDetail struct {
	Package string
	Test    string
	Output  string
}

// aggregateLayers groups test events into LayerSummary by layer classification.
// idx is a pre-built TestIndex mapping "pkg/TestName" → layerID; falls back to
// classifyTest(pkg, "") for any test not in the index.
func aggregateLayers(events []TestEvent, idx TestIndex) map[string]*LayerSummary {
	out := map[string]*LayerSummary{}
	// accumulate output per test for failure details
	outputBuf := map[string]string{}

	for _, ev := range events {
		key := ev.Package + "/" + ev.Test

		switch ev.Action {
		case "output":
			if ev.Test != "" {
				outputBuf[key] += ev.Output
			}
		case "pass", "fail":
			if ev.Test == "" {
				// package-level event: record elapsed for layer timing
				layerID, layerName := classifyTest(ev.Package, "")
				l := ensureLayer(out, layerID, layerName)
				l.Elapsed += ev.Elapsed
				continue
			}
			layerID, layerName := classify(ev.Package, ev.Test, idx)
			l := ensureLayer(out, layerID, layerName)
			if ev.Action == "pass" {
				l.Pass++
			} else {
				l.Fail++
				l.Failures = append(l.Failures, FailureDetail{
					Package: ev.Package,
					Test:    ev.Test,
					Output:  outputBuf[key],
				})
			}
		}
	}
	return out
}

// classify returns the layer for a specific test, using the index first and
// falling back to package-level heuristics.
func classify(pkg, testName string, idx TestIndex) (id, name string) {
	if layerID, ok := idx[pkg+"/"+testName]; ok {
		for _, ld := range layers {
			if ld.ID == layerID {
				return ld.ID, ld.Name
			}
		}
	}
	return classifyTest(pkg, "")
}

func ensureLayer(m map[string]*LayerSummary, id, name string) *LayerSummary {
	if l, ok := m[id]; ok {
		return l
	}
	priority := 99
	for _, ld := range layers {
		if ld.ID == id {
			priority = ld.Priority
			break
		}
	}
	l := &LayerSummary{ID: id, Name: name, Priority: priority}
	m[id] = l
	return l
}
