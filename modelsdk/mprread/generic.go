// SPDX-License-Identifier: Apache-2.0

package mprread

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// ListUnitsByType decodes every unit whose $Type exactly matches typeName and
// returns them as []T. It is the shared implementation behind per-type helpers
// like ListMicroflows; domain marathon tasks (Stage 3.3) use this directly so
// they don't repeat the codec + raw-bytes pattern.
func ListUnitsByType[T element.Element](r *mmpr.Reader, typeName string) ([]T, error) {
	refs, err := r.ListUnitsByType(typeName)
	if err != nil {
		return nil, err
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	out := make([]T, 0, len(refs))
	for _, ref := range refs {
		if ref.Type != typeName {
			continue
		}
		raw, err := r.GetRawUnitBytes(ref.ID)
		if err != nil {
			return nil, fmt.Errorf("read unit %s: %w", ref.ID, err)
		}
		elem, err := dec.Decode(bson.Raw(raw))
		if err != nil {
			return nil, fmt.Errorf("decode unit %s: %w", ref.ID, err)
		}
		typed, ok := elem.(T)
		if !ok {
			return nil, fmt.Errorf("unit %s decoded as %T, want %T", ref.ID, elem, *new(T))
		}
		out = append(out, typed)
	}
	return out, nil
}
