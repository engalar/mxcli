// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// AgentReader / AgentWriter / AgentRepository — signatures intentionally
// minimal until Stage 3 cutover.
//
// Covers Agent Editor documents (Agent, KnowledgeBase, ConsumedMCPService,
// Model). modelsdk/gen has no first-class Agent type yet; Stage 2 keys
// these by element.Element and discriminates by typeName at the
// repository call site. Stage 3 will introduce typed sub-repositories
// once the gen tree exposes Agent / KnowledgeBase root types (likely
// landing through Mendix 11.9+ schema regen).
//
// TODO Stage 3 cutover: split umbrella into typed sub-repositories.
type AgentReader interface {
	Get(id model.ID) (element.Element, error)
	ListByType(typeName string) ([]element.Element, error)
}

type AgentWriter interface {
	Create(parentUUID string, containmentName string, agent element.Element) error
	Update(agent element.Element) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type AgentRepository interface {
	AgentReader
	AgentWriter
}
