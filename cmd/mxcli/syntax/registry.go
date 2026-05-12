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
var registeredPaths = map[string]bool{}

// topicAliases maps legacy topic names and common variants to registry paths.
var topicAliases = map[string]string{
	// Connection aliases
	"connection":  "connect",
	"open":        "connect",
	"close":       "disconnect",
	"show-status": "status",
	"show_status": "status",
	// Domain model aliases
	"keywords":        "domain-model.keywords",
	"reserved":        "domain-model.keywords",
	"types":           "domain-model.types",
	"datatypes":       "domain-model.types",
	"data-types":      "domain-model.types",
	"delete":          "domain-model.association.delete-behavior",
	"delete_behavior": "domain-model.association.delete-behavior",
	"delete-behavior": "domain-model.association.delete-behavior",
	"entity":          "domain-model.entity",
	"entities":        "domain-model.entity",
	"enumeration":     "domain-model.enumeration",
	"enum":            "domain-model.enumeration",
	"enumerations":    "domain-model.enumeration",
	"constant":        "domain-model.constant",
	"constants":       "domain-model.constant",
	"association":     "domain-model.association",
	"associations":    "domain-model.association",
	// Plural aliases
	"microflows": "microflow",
	"pages":      "page",
	"snippets":   "snippet",
	"fragments":  "fragment",
	"workflows":  "workflow",
	// Variant aliases
	"nav":               "navigation",
	"project-settings":  "settings",
	"rest-client":       "rest",
	"rest-clients":      "rest",
	"integrations":      "integration",
	"services":          "integration",
	"contract":          "integration",
	"contracts":         "integration",
	"javaaction":        "java-action",
	"java_action":       "java-action",
	"java-actions":      "java-action",
	"javaactions":       "java-action",
	"businessevents":    "business-events",
	"business_events":   "business-events",
	"be":                "business-events",
	"xpath-constraints": "xpath",
	"external-sql":      "sql",
	"validation":        "errors",
	"testing":           "test",
	"tests":             "test",
	// Agents aliases
	"agent":          "agents",
	"agent-editor":   "agents",
	"agenteditor":    "agents",
	"model":          "agents.model",
	"models":         "agents.model",
	"knowledge-base": "agents.knowledge-base",
	"knowledgebase":  "agents.knowledge-base",
	"mcp":            "agents.mcp-service",
	"mcp-service":    "agents.mcp-service",
}

// Register adds a syntax feature to the global registry.
// Panics if a feature with the same path is already registered.
func Register(f SyntaxFeature) {
	if registeredPaths[f.Path] {
		panic("syntax: duplicate feature path: " + f.Path)
	}
	registeredPaths[f.Path] = true
	registry = append(registry, f)
}

// ResolveAlias returns the canonical registry path for a topic alias.
// If the input is not an alias, it is returned unchanged.
func ResolveAlias(path string) string {
	if alias, ok := topicAliases[path]; ok {
		return alias
	}
	return path
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
