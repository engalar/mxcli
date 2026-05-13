// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// ServiceReader / ServiceWriter / ServiceRepository — signatures
// intentionally minimal until Stage 3 cutover.
//
// "Services" is an umbrella domain in the legacy ServiceBackend covering
// REST clients, OData, business events, JS actions, Java actions, app
// services, etc. Stage 2 exposes one umbrella repository keyed by
// element.Element so handlers can be Stage-3-cutover without prior
// decomposition. Stage 3 will split this into per-sub-domain
// repositories (RestClientRepository, BusinessEventRepository, ...).
//
// TODO Stage 3 cutover: split umbrella into typed sub-repositories and
// produce MPR implementations.
type ServiceReader interface {
	Get(id model.ID) (element.Element, error)
	ListByType(typeName string) ([]element.Element, error)
}

type ServiceWriter interface {
	Create(parentUUID string, containmentName string, svc element.Element) error
	Update(svc element.Element) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type ServiceRepository interface {
	ServiceReader
	ServiceWriter
}
