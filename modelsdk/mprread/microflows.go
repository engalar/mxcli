// SPDX-License-Identifier: Apache-2.0

package mprread

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// BSON $Type names for microflow-family units. ListUnitsByType performs
// a prefix match, so we re-filter on the exact type to avoid pulling in
// look-alike kinds.
const (
	microflowTypeName = "Microflows$Microflow"
	nanoflowTypeName  = "Microflows$Nanoflow"
)

// ListMicroflows decodes every Microflows$Microflow unit in the project
// into the gen-typed *microflows.Microflow form.
//
// Microflows$Rule is a sibling type, not a Microflow alias — it gets
// its own lister (TODO: ListRules) once a caller needs it. Including
// rules here would fail the type assertion on decode.
func ListMicroflows(r *mmpr.Reader) ([]*genMf.Microflow, error) {
	refs, err := r.ListUnitsByType(microflowTypeName)
	if err != nil {
		return nil, err
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	result := make([]*genMf.Microflow, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != microflowTypeName {
			continue
		}
		raw, err := r.GetRawUnitBytes(ref.ID)
		if err != nil {
			return nil, fmt.Errorf("read microflow %s: %w", ref.ID, err)
		}
		elem, err := dec.Decode(bson.Raw(raw))
		if err != nil {
			return nil, fmt.Errorf("decode microflow %s: %w", ref.ID, err)
		}
		mf, ok := elem.(*genMf.Microflow)
		if !ok {
			return nil, fmt.Errorf("unit %s is not a Microflow (got %T, type=%q)", ref.ID, elem, elem.TypeName())
		}
		result = append(result, mf)
	}
	return result, nil
}

// ListNanoflows decodes every Microflows$Nanoflow unit in the project
// into the gen-typed *microflows.Nanoflow form.
func ListNanoflows(r *mmpr.Reader) ([]*genMf.Nanoflow, error) {
	refs, err := r.ListUnitsByType(nanoflowTypeName)
	if err != nil {
		return nil, err
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	result := make([]*genMf.Nanoflow, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != nanoflowTypeName {
			continue
		}
		raw, err := r.GetRawUnitBytes(ref.ID)
		if err != nil {
			return nil, fmt.Errorf("read nanoflow %s: %w", ref.ID, err)
		}
		elem, err := dec.Decode(bson.Raw(raw))
		if err != nil {
			return nil, fmt.Errorf("decode nanoflow %s: %w", ref.ID, err)
		}
		nf, ok := elem.(*genMf.Nanoflow)
		if !ok {
			return nil, fmt.Errorf("unit %s is not a Nanoflow (got %T, type=%q)", ref.ID, elem, elem.TypeName())
		}
		result = append(result, nf)
	}
	return result, nil
}
