// SPDX-License-Identifier: Apache-2.0

package syntax

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// WriteJSON writes features as a JSON array.
func WriteJSON(w io.Writer, features []SyntaxFeature) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(features)
}

// WriteText writes features in a human-readable grouped format.
func WriteText(w io.Writer, features []SyntaxFeature) {
	if len(features) == 0 {
		return
	}

	// Single feature — show full detail
	if len(features) == 1 {
		writeFeatureDetail(w, features[0])
		return
	}

	// Multiple features — group by top-level domain
	groups := groupByDomain(features)
	for i, g := range groups {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s\n", g.title)
		fmt.Fprintf(w, "%s\n\n", strings.Repeat("─", len(g.title)))
		for _, f := range g.features {
			fmt.Fprintf(w, "  %-40s %s\n", f.Path, f.Summary)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Use 'mxcli syntax <path> --json' for machine-readable detail.\n")
	fmt.Fprintf(w, "Use 'mxcli syntax <path>' to drill down (e.g., 'mxcli syntax workflow user-task').\n")
}

func writeFeatureDetail(w io.Writer, f SyntaxFeature) {
	fmt.Fprintf(w, "%s\n", f.Path)
	fmt.Fprintf(w, "%s\n\n", strings.Repeat("═", len(f.Path)))
	fmt.Fprintf(w, "%s\n\n", f.Summary)

	if f.MinVersion != "" {
		fmt.Fprintf(w, "Requires: Mendix %s+\n\n", f.MinVersion)
	}

	if len(f.Keywords) > 0 {
		fmt.Fprintf(w, "Keywords: %s\n\n", strings.Join(f.Keywords, ", "))
	}

	if f.Syntax != "" {
		fmt.Fprintln(w, "Syntax:")
		for _, line := range strings.Split(f.Syntax, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}

	if f.Example != "" {
		fmt.Fprintln(w, "Example:")
		for _, line := range strings.Split(f.Example, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}

	if len(f.SeeAlso) > 0 {
		fmt.Fprintf(w, "See also: %s\n", strings.Join(f.SeeAlso, ", "))
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

type featureGroup struct {
	title    string
	features []SyntaxFeature
}

func groupByDomain(features []SyntaxFeature) []featureGroup {
	order := []string{}
	m := map[string][]SyntaxFeature{}
	for _, f := range features {
		domain := f.Path
		if idx := strings.IndexByte(f.Path, '.'); idx > 0 {
			domain = f.Path[:idx]
		}
		if _, ok := m[domain]; !ok {
			order = append(order, domain)
		}
		m[domain] = append(m[domain], f)
	}
	groups := make([]featureGroup, len(order))
	for i, d := range order {
		groups[i] = featureGroup{title: capitalize(d), features: m[d]}
	}
	return groups
}
