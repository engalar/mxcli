// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.6.3a: extracted from cmd_microflow_elk.go (deleted) so the
// shared ELK output schema lives in a file that is not tied to the
// legacy sdk/microflows-typed implementation. These types are consumed
// by the gen-typed ELK builders (cmd_microflow_elk_gen.go,
// cmd_nanoflow_elk_gen.go) and by the page wireframe / domain-model
// ELK files. No sdk/microflows imports — pure schema.

package executor

// elkSourceRange maps a diagram node to a line range in the MDL source.
type elkSourceRange struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

// microflowELKData is the JSON output schema for the microflow ELK diagram.
type microflowELKData struct {
	Format     string                    `json:"format"`
	Type       string                    `json:"type"`
	Name       string                    `json:"name"`
	Parameters []microflowELKParam       `json:"parameters"`
	ReturnType string                    `json:"returnType"`
	Nodes      []microflowELKNode        `json:"nodes"`
	Edges      []microflowELKEdge        `json:"edges"`
	MdlSource  string                    `json:"mdlSource,omitempty"`
	SourceMap  map[string]elkSourceRange `json:"sourceMap,omitempty"`
}

type microflowELKParam struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type microflowELKNode struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Category string   `json:"category"`
	Label    string   `json:"label"`
	Details  []string `json:"details,omitempty"`
	Width    float64  `json:"width"`
	Height   float64  `json:"height"`
	// Compound node fields (for loop bodies)
	Children []microflowELKNode `json:"children,omitempty"`
	Edges    []microflowELKEdge `json:"edges,omitempty"`
}

type microflowELKEdge struct {
	ID             string `json:"id"`
	SourceID       string `json:"sourceId"`
	TargetID       string `json:"targetId"`
	Label          string `json:"label,omitempty"`
	IsErrorHandler bool   `json:"isErrorHandler,omitempty"`
}
