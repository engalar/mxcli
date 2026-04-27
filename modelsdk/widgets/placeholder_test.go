// SPDX-License-Identifier: Apache-2.0

package widgets

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestContainsPlaceholderID_BinaryBlob(t *testing.T) {
	// Create a bson.D with a placeholder binary blob (post-GUID-swap)
	// Placeholder "aa000000000000000000000000000001" after hexToIDBlob:
	// bytes 0-3 reversed: \x00\x00\x00\xaa, bytes 4-12: zeros, bytes 13-15: \x00\x00\x01
	blob := hexToIDBlob("aa000000000000000000000000000001")
	doc := bson.D{{Key: "$ID", Value: blob}}

	if !containsPlaceholderID(doc) {
		t.Error("expected containsPlaceholderID to detect placeholder binary blob")
	}
}

func TestContainsPlaceholderID_StringValue(t *testing.T) {
	// A placeholder that leaked as a string (e.g., unmapped TypePointer)
	doc := bson.D{{Key: "TypePointer", Value: "aa000000000000000000000000000005"}}

	if !containsPlaceholderID(doc) {
		t.Error("expected containsPlaceholderID to detect placeholder string value")
	}
}

func TestContainsPlaceholderID_Nested(t *testing.T) {
	blob := hexToIDBlob("aa000000000000000000000000000002")
	doc := bson.D{
		{Key: "$ID", Value: hexToIDBlob("abcdef01234567890abcdef012345678")},
		{Key: "Children", Value: bson.A{
			bson.D{
				{Key: "$ID", Value: blob},
				{Key: "Name", Value: "test"},
			},
		}},
	}

	if !containsPlaceholderID(doc) {
		t.Error("expected containsPlaceholderID to detect nested placeholder blob")
	}
}

func TestContainsPlaceholderID_Clean(t *testing.T) {
	// A legitimate UUID should not trigger detection
	doc := bson.D{
		{Key: "$ID", Value: hexToIDBlob("abcdef01234567890abcdef012345678")},
		{Key: "Name", Value: "SomeWidget"},
		{Key: "Items", Value: bson.A{
			bson.D{{Key: "$ID", Value: hexToIDBlob("12345678abcdef0012345678abcdef00")}},
		}},
	}

	if containsPlaceholderID(doc) {
		t.Error("expected containsPlaceholderID to NOT trigger on legitimate UUIDs")
	}
}
