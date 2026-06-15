package mxgraph

type NodeID string
type RelType string
type Label string

type Node struct {
	ID    NodeID
	Label Label
	Props map[string]any
}

type Edge struct {
	ID    NodeID
	From  NodeID
	To    NodeID
	Type  RelType
	Props map[string]any
}

type EventType uint8

const (
	NodeCreated EventType = iota
	NodeUpdated
	NodeDeleted
	EdgeCreated
	EdgeDeleted
)

type Event struct {
	Type EventType
	Node *Node
	Edge *Edge
}

type PathStep struct {
	NodeLabel Label
	RelType   RelType
}

type PathSchema struct {
	Steps []PathStep
	Label string
}

type PathNode struct {
	Node   *Node
	RelOut RelType
}

type Direction uint8

const (
	Outbound Direction = iota
	Inbound
	Both
)
