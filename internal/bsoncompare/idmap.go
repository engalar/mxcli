// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"strings"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type IDMap map[string]string

// BuildIDMapFromReader builds an IDMap (hex UUID → label) from all units.
// Delegates to mpr.Reader.ListUnitIdentities to avoid full Element decode.
func BuildIDMapFromReader(r *mmpr.Reader) (IDMap, error) {
	idents, err := r.ListUnitIdentities()
	if err != nil {
		return nil, err
	}
	m := make(IDMap, len(idents))
	for _, id := range idents {
		if _, exists := m[id.ID]; !exists {
			m[id.ID] = makeLabel(id.Type, id.Name, "")
		}
	}
	return m, nil
}

func makeLabel(bsonType, name, ctx string) string {
	short := bsonType
	if i := strings.Index(bsonType, "$"); i >= 0 {
		short = bsonType[i+1:]
	}
	if name != "" {
		return fmt.Sprintf("%s:%s", short, name)
	}
	if ctx != "" {
		return fmt.Sprintf("%s(%s)", short, ctx)
	}
	return short
}

func (m IDMap) Lookup(data []byte) string {
	if len(data) != 16 {
		return "<binary>"
	}
	key := mmpr.BinaryToUUID(data)
	if label, ok := m[key]; ok {
		return "<ref:" + label + ">"
	}
	return "<ref:?>"
}

// LookupID resolves a UUID-format element.ID string to a human-readable label.
// Used when comparing ByIdRef properties via IDMap labels rather than raw UUIDs.
func (m IDMap) LookupID(id string) string {
	if label, ok := m[id]; ok {
		return "<ref:" + label + ">"
	}
	return "<ref:?>"
}

func MergeInto(dst, src IDMap) {
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			dst[k] = v
		}
	}
}
