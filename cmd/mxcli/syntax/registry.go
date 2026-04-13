// SPDX-License-Identifier: Apache-2.0

package syntax

import (
	"sort"
	"strings"
)

// SyntaxFeature describes a discoverable MDL syntax feature.
type SyntaxFeature struct {
	Path       string   `json:"path"`
	Summary    string   `json:"summary"`
	Keywords   []string `json:"keywords"`
	Syntax     string   `json:"syntax"`
	Example    string   `json:"example"`
	MinVersion string   `json:"min_version,omitempty"`
	SeeAlso    []string `json:"see_also,omitempty"`
}

var registry []SyntaxFeature

// Register adds a syntax feature to the global registry.
func Register(f SyntaxFeature) {
	registry = append(registry, f)
}

// All returns every registered feature, sorted by path.
func All() []SyntaxFeature {
	out := make([]SyntaxFeature, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// ByPrefix returns features whose path equals or starts with prefix+".".
func ByPrefix(prefix string) []SyntaxFeature {
	var out []SyntaxFeature
	for _, f := range registry {
		if f.Path == prefix || strings.HasPrefix(f.Path, prefix+".") {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// ByPath returns the feature with the exact path, or nil.
func ByPath(path string) *SyntaxFeature {
	for i := range registry {
		if registry[i].Path == path {
			return &registry[i]
		}
	}
	return nil
}

// HasPrefix reports whether any registered feature matches the prefix.
func HasPrefix(prefix string) bool {
	for _, f := range registry {
		if f.Path == prefix || strings.HasPrefix(f.Path, prefix+".") {
			return true
		}
	}
	return false
}
