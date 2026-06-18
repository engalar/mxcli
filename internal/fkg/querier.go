// internal/fkg/querier.go
package fkg

// Querier is the only interface the CLI and tests depend on.
// It exposes three read-only queries over the feature knowledge graph.
type Querier interface {
	Explore(id string, depth int) (*ExploreResult, error)
	Path(from, to string) ([]PathSchema, error)
	Schema() *SchemaResult
}

// GuidanceQuerier provides implementation guidance for a concept.
type GuidanceQuerier interface {
	Guide(conceptID string) (*GuidanceResult, error)
}

// CurriculumQuerier provides learning module planning.
type CurriculumQuerier interface {
	Plan(moduleName string) (*CurriculumPlan, error)
}

// NodeSummary is a compact representation of any graph node.
type NodeSummary struct {
	ID      string
	Label   string // "Concept" | "SyntaxFeature" | "Skill" | "Doc" | "Pattern" | etc.
	Name    string
	Summary string
}

// EdgeSummary represents a single directed edge in the graph.
type EdgeSummary struct {
	RelType string
	From    string
	To      string
}

// ExploreResult contains the seed node plus all nodes and edges
// reachable within the requested depth.
type ExploreResult struct {
	Seed  NodeSummary
	Nodes []NodeSummary
	Edges []EdgeSummary
}

// PathStep is one hop in a path.
type PathStep struct {
	NodeLabel string // type label
	RelType   string
	NodeID    string // concrete node ID
	NodeName  string // concrete display name
}

// PathSchema describes a structural path pattern between two nodes.
type PathSchema struct {
	Steps []PathStep
	Label string
}

// NodeTypeInfo holds a node label and how many nodes carry it.
type NodeTypeInfo struct {
	Label string
	Count int
}

// EdgeTypeInfo holds an edge relation type and its frequency.
type EdgeTypeInfo struct {
	RelType string
	Count   int
}

// SchemaResult describes the full ontology skeleton.
type SchemaResult struct {
	NodeTypes []NodeTypeInfo
	EdgeTypes []EdgeTypeInfo
	Roots     []NodeSummary
}

// GuidanceStep is one actionable implementation instruction.
type GuidanceStep struct {
	Order       int
	Action      string // "create", "configure", "wire", "grant"
	TargetType  string // "Entity", "Microflow", "Page", "Security"
	TargetName  string
	Description string
	SyntaxHint  string
}

// GuidanceResult aggregates everything needed to implement a concept.
type GuidanceResult struct {
	Concept    NodeSummary
	Patterns   []NodeSummary
	Steps      []GuidanceStep
	SyntaxRefs []NodeSummary
	Skills     []NodeSummary
	Extensions []NodeSummary
}

// Orchestrator plans multi-concept implementation ordering.
type Orchestrator interface {
	Orchestrate(conceptIDs []string) (*OrchestrationPlan, error)
}

// OrchestrationStep is one step in a multi-concept implementation plan.
type OrchestrationStep struct {
	Concept   NodeSummary
	Order     int
	DependsOn []string
	Patterns  []NodeSummary
	Skills    []NodeSummary
}

// OrchestrationPlan is an ordered sequence of implementation steps.
type OrchestrationPlan struct {
	Steps []OrchestrationStep
}

// CurriculumPlan describes a module's concept/skill dependencies.
type CurriculumPlan struct {
	Module        NodeSummary
	Prerequisites []NodeSummary
	Concepts      []NodeSummary
	Skills        []NodeSummary
	Patterns      []NodeSummary
	Extensions    []NodeSummary
}
