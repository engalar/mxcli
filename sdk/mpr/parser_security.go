// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	gensecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// securityDecoder is a package-level codec.Decoder for security gen types.
// Initialized lazily on first use.
var securityDecoder = codec.NewDecoder(codec.DefaultRegistry)

// parseProjectSecurity parses a Security$ProjectSecurity BSON document into a gen type.
func (r *Reader) parseProjectSecurity(unitID, _ string, contents []byte) (*gensecurity.ProjectSecurity, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}

	elem, err := securityDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode project security %s: %w", unitID, err)
	}
	ps, ok := elem.(*gensecurity.ProjectSecurity)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a ProjectSecurity (got %T)", unitID, elem)
	}
	ps.SetID(element.ID(unitID))
	return ps, nil
}

// parseModuleSecurity parses a Security$ModuleSecurity BSON document into a gen type.
func (r *Reader) parseModuleSecurity(unitID, _ string, contents []byte) (*gensecurity.ModuleSecurity, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}

	elem, err := securityDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode module security %s: %w", unitID, err)
	}
	ms, ok := elem.(*gensecurity.ModuleSecurity)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a ModuleSecurity (got %T)", unitID, elem)
	}
	ms.SetID(element.ID(unitID))
	return ms, nil
}
