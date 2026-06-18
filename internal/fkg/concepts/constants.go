// internal/fkg/concepts/constants.go
package concepts

import "github.com/mendixlabs/mxcli/internal/mxgraph"

// Node labels used in the feature knowledge graph.
const (
	LabelConcept       mxgraph.Label = "Concept"
	LabelSyntaxFeature mxgraph.Label = "SyntaxFeature"
	LabelSkill         mxgraph.Label = "Skill"
	LabelDoc           mxgraph.Label = "Doc"
	LabelPattern       mxgraph.Label = "Pattern"
	LabelImplDetail    mxgraph.Label = "ImplDetail"
	LabelCodeExtension mxgraph.Label = "CodeExtension"
	LabelCurriculum    mxgraph.Label = "Curriculum"
)

// Edge relation types used in the feature knowledge graph.
const (
	Specializes mxgraph.RelType = "SPECIALIZES" // SubConcept → Concept
	Requires    mxgraph.RelType = "REQUIRES"    // Concept → Concept (hard dep)
	RelatedTo   mxgraph.RelType = "RELATED_TO"  // Concept ↔ Concept (soft)
	HasSyntax   mxgraph.RelType = "HAS_SYNTAX"  // Concept → SyntaxFeature
	HasSkill    mxgraph.RelType = "HAS_SKILL"   // Concept → Skill
	HasDoc      mxgraph.RelType = "HAS_DOC"     // Concept → Doc
	HasPattern  mxgraph.RelType = "HAS_PATTERN" // Concept → Pattern
	HasExt      mxgraph.RelType = "HAS_EXT"     // Concept → CodeExtension
	Teaches     mxgraph.RelType = "TEACHES"     // Curriculum → Concept|Skill|Pattern
	Depends     mxgraph.RelType = "DEPENDS"     // Curriculum → Curriculum (prerequisite)
)
