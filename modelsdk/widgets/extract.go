// SPDX-License-Identifier: Apache-2.0

package widgets

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// extractTemplateFromProject scans the project's mprcontents/ directory for an
// existing CustomWidgets$CustomWidget instance whose Type has the given widgetID.
// If found, the CustomWidgets$CustomWidgetType and CustomWidgets$WidgetObject BSONs
// are extracted and converted to a WidgetTemplate with placeholder IDs, ready for
// use by GetTemplateFullBSON.
//
// This is the preferred source over GenerateFromMPK because the Type+Object BSONs
// were originally created by Studio Pro (or a previous correct mxcli run), ensuring
// they exactly match Studio Pro's expectations and avoids CE0463.
func extractTemplateFromProject(widgetID, projectPath string) *WidgetTemplate {
	projectDir := filepath.Dir(projectPath)
	contentsDir := filepath.Join(projectDir, "mprcontents")
	if _, err := os.Stat(contentsDir); err != nil {
		return nil // v1 MPR or mprcontents missing — skip
	}

	type extractedWidget struct {
		typeDoc   bson.D
		objectDoc bson.D
	}
	var found *extractedWidget

	_ = filepath.WalkDir(contentsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".mxunit") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// Quick byte-level filter before full BSON decode.
		if !containsBytes(data, widgetID) || !containsBytes(data, "CustomWidgets$CustomWidget") {
			return nil
		}
		var doc bson.D
		if err := bson.Unmarshal(data, &doc); err != nil {
			return nil
		}
		if t, o := findCustomWidget(doc, widgetID); t != nil && o != nil {
			found = &extractedWidget{typeDoc: t, objectDoc: o}
		}
		return nil
	})

	if found == nil {
		return nil
	}

	tmpl, err := bsonWidgetToTemplate(found.typeDoc, found.objectDoc, widgetID)
	if err != nil {
		log.Printf("widgets: extractTemplateFromProject %s: %v", widgetID, err)
		return nil
	}
	return tmpl
}

// containsBytes returns true if needle appears as a byte sequence in haystack.
func containsBytes(haystack []byte, needle string) bool {
	n := []byte(needle)
	if len(n) > len(haystack) {
		return false
	}
	for i := 0; i <= len(haystack)-len(n); i++ {
		match := true
		for j, b := range n {
			if haystack[i+j] != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// findCustomWidget recursively searches for a CustomWidgets$CustomWidget with the
// given widgetID and returns its (Type, Object) pair.
func findCustomWidget(doc bson.D, widgetID string) (bson.D, bson.D) {
	var docType string
	var typeDoc, objectDoc bson.D
	var foundWidgetID bool

	for _, e := range doc {
		switch e.Key {
		case "$Type":
			if s, ok := e.Value.(string); ok {
				docType = s
			}
		case "Type":
			if sub, ok := e.Value.(bson.D); ok {
				typeDoc = sub
			}
		case "Object":
			if sub, ok := e.Value.(bson.D); ok {
				objectDoc = sub
			}
		}
	}

	// Check if this document's Type has the matching WidgetId
	if docType == "CustomWidgets$CustomWidget" && typeDoc != nil {
		for _, e := range typeDoc {
			if e.Key == "WidgetId" {
				if s, ok := e.Value.(string); ok && s == widgetID {
					foundWidgetID = true
				}
			}
		}
		if foundWidgetID && objectDoc != nil {
			return typeDoc, objectDoc
		}
	}

	// Recurse into child elements
	for _, e := range doc {
		if sub, ok := e.Value.(bson.D); ok {
			if t, o := findCustomWidget(sub, widgetID); t != nil {
				return t, o
			}
		}
		if arr, ok := e.Value.(bson.A); ok {
			for _, item := range arr {
				if sub, ok := item.(bson.D); ok {
					if t, o := findCustomWidget(sub, widgetID); t != nil {
						return t, o
					}
				}
			}
		}
	}
	return nil, nil
}

// bsonWidgetToTemplate converts extracted Type and Object bson.D documents into a
// WidgetTemplate. Binary UUIDs are stored as UUID-order 32-char hex strings using
// a shared idMap (same binary → same hex across Type+Object). StableIds=true tells
// GetTemplateFullBSON to use collectIDsIdentity, preserving the original Studio Pro
// $IDs so Mendix doesn't trigger CE0463 for repeated instances of the same widget type.
func bsonWidgetToTemplate(typeDoc, objectDoc bson.D, widgetID string) (*WidgetTemplate, error) {
	// bsonDToMapPreserveIDs converts binary UUIDs to UUID-order 32-char hex via a shared
	// map so the same binary gets the same hex string across both documents.
	sharedIDMap := make(map[string]string) // raw-bytes key → uuid-order hex value
	typeMap := bsonDToMapPreserveIDs(typeDoc, sharedIDMap)
	objectMap := bsonDToMapPreserveIDs(objectDoc, sharedIDMap)

	typeMapTyped, ok := typeMap.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("type document is not a map")
	}
	objectMapTyped, ok := objectMap.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("object document is not a map")
	}

	return &WidgetTemplate{
		WidgetID:  widgetID,
		Generated: true, // skip augmentFromMPK — Studio Pro BSON is authoritative
		StableIds: true, // use collectIDsIdentity so original $IDs are reused
		Type:      typeMapTyped,
		Object:    objectMapTyped,
	}, nil
}

// bsonDToMapPreserveIDs converts a bson.D to map[string]any, encoding each binary
// UUID as a 32-char UUID-order hex string. A shared idMap ensures the same binary
// always maps to the same hex across both Type and Object documents (so TypePointers
// in Object correctly reference PropertyType $IDs from Type after remapping).
func bsonDToMapPreserveIDs(doc bson.D, idMap map[string]string) any {
	m := make(map[string]any, len(doc))
	for _, e := range doc {
		m[e.Key] = bsonValuePreserveIDs(e.Value, idMap)
	}
	return m
}

func bsonValuePreserveIDs(val any, idMap map[string]string) any {
	switch v := val.(type) {
	case bson.D:
		return bsonDToMapPreserveIDs(v, idMap)
	case bson.A:
		result := make([]any, 0, len(v))
		for _, item := range v {
			result = append(result, bsonValuePreserveIDs(item, idMap))
		}
		return result
	case bson.Binary:
		return binaryToUUIDOrderHex(v.Data, idMap)
	default:
		return v
	}
}

// binaryToUUIDOrderHex converts a Microsoft GUID binary to the UUID-byte-order 32-char hex
// string that round-trips through hexToIDBlob back to the original binary. A shared
// idMap ensures the same binary always produces the same hex string.
func binaryToUUIDOrderHex(data []byte, idMap map[string]string) string {
	key := string(data)
	if h, ok := idMap[key]; ok {
		return h
	}
	var h string
	if len(data) == 16 {
		// Convert from Microsoft GUID byte order (segments 1-3 are little-endian)
		// to UUID byte order by reversing those segments. hexToIDBlob then applies
		// the same swap in reverse to recover the original binary.
		uuid := make([]byte, 16)
		uuid[0], uuid[1], uuid[2], uuid[3] = data[3], data[2], data[1], data[0]
		uuid[4], uuid[5] = data[5], data[4]
		uuid[6], uuid[7] = data[7], data[6]
		copy(uuid[8:], data[8:])
		h = hex.EncodeToString(uuid)
	} else {
		h = hex.EncodeToString(data)
	}
	idMap[key] = h
	return h
}

// bsonDToMapWithIDMap converts a bson.D to map[string]any, replacing binary UUIDs
// with placeholder hex strings using a shared idMap for consistency across Type+Object.
func bsonDToMapWithIDMap(doc bson.D, idMap map[string]string) any {
	m := make(map[string]any, len(doc))
	for _, e := range doc {
		m[e.Key] = bsonValueToAnyWithIDMap(e.Value, idMap)
	}
	return m
}

func bsonValueToAnyWithIDMap(val any, idMap map[string]string) any {
	switch v := val.(type) {
	case bson.D:
		return bsonDToMapWithIDMap(v, idMap)
	case bson.A:
		result := make([]any, 0, len(v))
		for _, item := range v {
			result = append(result, bsonValueToAnyWithIDMap(item, idMap))
		}
		return result
	case []byte:
		return binaryToPlaceholder(v, idMap)
	case bson.Binary:
		return binaryToPlaceholder(v.Data, idMap)
	default:
		return v
	}
}

// binaryToPlaceholder converts a binary UUID to a placeholder hex string.
// The same binary UUID always maps to the same placeholder (via idMap).
func binaryToPlaceholder(data []byte, idMap map[string]string) string {
	key := string(data) // use raw bytes as map key
	if p, ok := idMap[key]; ok {
		return p
	}
	p := placeholderID()
	idMap[key] = p
	return p
}
