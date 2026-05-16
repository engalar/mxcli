// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	gendomainmodels "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"

	"go.mongodb.org/mongo-driver/bson"
)

// domainModelDecoder is a package-level codec.Decoder for domainmodels gen types.
var domainModelDecoder = codec.NewDecoder(codec.DefaultRegistry)

// parseDomainModelGen parses a DomainModels$DomainModel BSON document into a gen type.
func (r *Reader) parseDomainModelGen(unitID, _ string, contents []byte) (*gendomainmodels.DomainModel, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}

	elem, err := domainModelDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode domain model %s: %w", unitID, err)
	}
	dm, ok := elem.(*gendomainmodels.DomainModel)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a DomainModel (got %T)", unitID, elem)
	}
	dm.SetID(element.ID(unitID))
	return dm, nil
}
