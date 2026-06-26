// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"encoding/hex"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// rawDocOK returns the Raw value if doc is valid, nil otherwise.
func rawDocOK(doc bson.Raw) bson.Raw {
	if len(doc) > 0 {
		return doc
	}
	return nil
}

type IDMap map[string]string

func BuildIDMap(units []UnitDoc) IDMap {
	m := make(IDMap, len(units)*40)
	for i := range units {
		collectIDsRaw(units[i].Raw, units[i].QualifiedName, m, 0)
	}
	return m
}

// collectIDsRaw walks raw BSON to extract $ID → label mappings without
// decoding into bson.D. This avoids the expensive bson.Unmarshal for the
// 10k+ entries typically found in a corpus-b project.
func collectIDsRaw(raw bson.Raw, ctx string, m IDMap, depth int) {
	if depth > 8 || len(raw) == 0 {
		return
	}
	elems, err := raw.Elements()
	if err != nil {
		return
	}

	var selfID []byte
	var name, typ string
	for _, elem := range elems {
		switch elem.Key() {
		case "$ID":
			val := elem.Value()
			if val.Type == bson.TypeBinary {
				_, data := val.Binary()
				if len(data) == 16 {
					selfID = data
				}
			}
		case "Name":
			val := elem.Value()
			if val.Type == bson.TypeString {
				name = val.StringValue()
			}
		case "$Type":
			val := elem.Value()
			if val.Type == bson.TypeString {
				typ = val.StringValue()
			}
		}
	}
	if len(selfID) == 16 {
		key := hex.EncodeToString(selfID)
		if _, exists := m[key]; !exists {
			m[key] = makeLabel(typ, name, ctx)
		}
		if name != "" {
			ctx = name
		}
	}

	for _, elem := range elems {
		val := elem.Value()
		switch val.Type {
		case bson.TypeEmbeddedDocument:
			sub := val.Document()
			collectIDsRaw(sub, ctx+"."+elem.Key(), m, depth+1)
		case bson.TypeArray:
			arr := val.Array()
			arrVals, err := arr.Values()
			if err != nil {
				continue
			}
			for _, item := range arrVals {
				if item.Type == bson.TypeEmbeddedDocument {
					sub := item.Document()
					collectIDsRaw(sub, ctx, m, depth+1)
				}
			}
		}
	}
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
	key := hex.EncodeToString(data)
	if label, ok := m[key]; ok {
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
