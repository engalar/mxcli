// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// MappingReader / MappingWriter / MappingRepository — signatures
// intentionally minimal until Stage 3 cutover.
//
// Mappings is an umbrella covering Import (modelsdk/gen/importmappings),
// Export (modelsdk/gen/exportmappings), and ML mappings
// (modelsdk/gen/mlmappings). Stage 2 keeps it as a single umbrella keyed
// by element.Element; Stage 3 will split into per-kind repositories.
//
// TODO Stage 3 cutover: split umbrella into typed sub-repositories and
// produce MPR implementations.
type MappingReader interface {
	Get(id model.ID) (element.Element, error)
	ListByType(typeName string) ([]element.Element, error)
}

type MappingWriter interface {
	Create(parentUUID string, containmentName string, mapping element.Element) error
	Update(mapping element.Element) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type MappingRepository interface {
	MappingReader
	MappingWriter
}
