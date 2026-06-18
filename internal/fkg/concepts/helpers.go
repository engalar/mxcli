// internal/fkg/concepts/helpers.go
package concepts

import (
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// conceptNode returns a NodeCreated event for a Concept node.
func conceptNode(id, name, summary string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID(id),
			Label: LabelConcept,
			Props: map[string]any{"Name": name, "Summary": summary},
		},
	}
}

// syntaxNode returns a NodeCreated event for a SyntaxFeature node.
// id is prefixed with "syntax:" to avoid collisions with concept IDs.
func syntaxNode(path, summary string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID("syntax:" + path),
			Label: LabelSyntaxFeature,
			Props: map[string]any{"Name": path, "Summary": summary},
		},
	}
}

// skillNode returns a NodeCreated event for a Skill node.
// id is prefixed with "skill:" to avoid collisions.
func skillNode(name, summary string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID("skill:" + name),
			Label: LabelSkill,
			Props: map[string]any{"Name": name, "Summary": summary},
		},
	}
}

// patternNode returns a NodeCreated event for a Pattern node.
func patternNode(id, name, summary string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID("pattern:" + id),
			Label: LabelPattern,
			Props: map[string]any{"Name": name, "Summary": summary},
		},
	}
}

// implDetailNode returns a NodeCreated event for an ImplDetail node.
func implDetailNode(id, name, summary string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID("detail:" + id),
			Label: LabelImplDetail,
			Props: map[string]any{"Name": name, "Summary": summary},
		},
	}
}

// extNode returns a NodeCreated event for a CodeExtension node.
func extNode(id, name, summary string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID("ext:" + id),
			Label: LabelCodeExtension,
			Props: map[string]any{"Name": name, "Summary": summary},
		},
	}
}

// stepNode returns a NodeCreated event for an implementation step (ImplDetail with StepOrder/StepAction).
// These are consumed by Guide() to build ordered GuidanceStep results.
func stepNode(id, name, summary string, order int, action, targetType, targetName, syntaxHint string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID("step:" + id),
			Label: LabelImplDetail,
			Props: map[string]any{
				"Name":       name,
				"Summary":    summary,
				"StepOrder":  order,
				"StepAction": action,
				"TargetType": targetType,
				"TargetName": targetName,
				"SyntaxHint": syntaxHint,
			},
		},
	}
}

// curriculumNode returns a NodeCreated event for a Curriculum node.
func curriculumNode(id, name, summary string) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.NodeCreated,
		Node: &mxgraph.Node{
			ID:    mxgraph.NodeID("curriculum:" + id),
			Label: LabelCurriculum,
			Props: map[string]any{"Name": name, "Summary": summary},
		},
	}
}

// edge returns an EdgeCreated event. from and to are raw node IDs
// (already prefixed if needed, e.g. "syntax:page.create").
func edge(from, to string, rel mxgraph.RelType) mxgraph.Event {
	return mxgraph.Event{
		Type: mxgraph.EdgeCreated,
		Edge: &mxgraph.Edge{
			ID:   mxgraph.NodeID(fmt.Sprintf("%s-[%s]->%s", from, rel, to)),
			From: mxgraph.NodeID(from),
			To:   mxgraph.NodeID(to),
			Type: rel,
		},
	}
}
