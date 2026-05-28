// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"encoding/hex"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type IDMap map[string]string

func BuildIDMap(units []UnitDoc) IDMap {
	m := make(IDMap, len(units)*40)
	for _, u := range units {
		collectIDs(u.Doc, u.QualifiedName, m, 0)
	}
	return m
}

func collectIDs(doc bson.D, ctx string, m IDMap, depth int) {
	if depth > 8 {
		return
	}
	var selfID []byte
	var name, typ string
	for _, e := range doc {
		switch e.Key {
		case "$ID":
			if b, ok := e.Value.(bson.Binary); ok && len(b.Data) == 16 {
				selfID = b.Data
			}
		case "Name":
			name, _ = e.Value.(string)
		case "$Type":
			typ, _ = e.Value.(string)
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
	for _, e := range doc {
		switch v := e.Value.(type) {
		case bson.D:
			collectIDs(v, ctx+"."+e.Key, m, depth+1)
		case bson.A:
			for _, item := range v {
				if sub, ok := item.(bson.D); ok {
					collectIDs(sub, ctx, m, depth+1)
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
