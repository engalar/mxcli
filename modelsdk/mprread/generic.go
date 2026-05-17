// SPDX-License-Identifier: Apache-2.0

package mprread

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// Unit pairs a decoded gen-typed element with its SQLite ContainerID.
// ContainerID is a DB-level column that is not embedded in the BSON content,
// so callers that need it (e.g. building module-level trees) can obtain it
// here without a second query.
type Unit[T element.Element] struct {
	Element     T
	ContainerID model.ID
}

// ListUnitsByType decodes every unit whose $Type exactly matches typeName and
// returns them as []T. It is the shared implementation behind per-type helpers
// like ListMicroflows; domain marathon tasks (Stage 3.3) use this directly so
// they don't repeat the codec + raw-bytes pattern.
//
// ContainerID is not included in the result; use ListUnitsWithContainer if you
// need it.
func ListUnitsByType[T element.Element](r *mmpr.Reader, typeName string) ([]T, error) {
	units, err := ListUnitsWithContainer[T](r, typeName)
	if err != nil {
		return nil, err
	}
	out := make([]T, len(units))
	for i, u := range units {
		out[i] = u.Element
	}
	return out, nil
}

// ListUnitsWithContainer decodes every unit whose $Type exactly matches
// typeName and returns each element paired with its ContainerID.
// It uses ref.Contents from the unit index (already loaded by ListUnitsByType)
// instead of issuing a second GetRawUnitBytes query per unit.
func ListUnitsWithContainer[T element.Element](r *mmpr.Reader, typeName string) ([]Unit[T], error) {
	refs, err := r.ListUnitsByType(typeName)
	if err != nil {
		return nil, err
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	out := make([]Unit[T], 0, len(refs))
	for _, ref := range refs {
		if ref.Type != typeName {
			continue
		}
		elem, err := dec.Decode(bson.Raw(ref.Contents))
		if err != nil {
			return nil, fmt.Errorf("decode unit %s: %w", ref.ID, err)
		}
		typed, ok := elem.(T)
		if !ok {
			return nil, fmt.Errorf("unit %s decoded as %T, want %T", ref.ID, elem, *new(T))
		}
		out = append(out, Unit[T]{Element: typed, ContainerID: model.ID(ref.ContainerID)})
	}
	return out, nil
}
