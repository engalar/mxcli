// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

// getBSONField looks up a field in a bson.D document by key.
func getBSONField(doc bson.D, key string) any {
	for _, elem := range doc {
		if elem.Key == key {
			return elem.Value
		}
	}
	return nil
}

// assertArrayMarker checks that doc[key] is a bson.A whose first element equals marker.
func assertArrayMarker(t *testing.T, doc bson.D, key string, marker any) {
	t.Helper()
	v := getBSONField(doc, key)
	arr, ok := v.(bson.A)
	if !ok {
		t.Errorf("%s: expected bson.A, got %T", key, v)
		return
	}
	if len(arr) == 0 {
		t.Errorf("%s: array is empty, expected marker %v as first element", key, marker)
		return
	}
	if arr[0] != marker {
		t.Errorf("%s[0] = %v (%T), want %v (%T)", key, arr[0], arr[0], marker, marker)
	}
}
