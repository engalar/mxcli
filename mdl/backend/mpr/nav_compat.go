// SPDX-License-Identifier: Apache-2.0

// nav_compat.go — Navigation document BSON parsing.
//
// Hand-decoded against modelsdk/mpr.Reader's raw bytes. The gen/navigation
// schema covers the data faithfully, but the executor consumes a
// flattened mdl/types.NavigationDocument view that does not map 1:1 to the
// underlying typed tree, so we keep the parser here rather than write a
// large gen→model converter.
//
// Logic mirrors what previously lived in sdk/mpr/parser_misc.go; only the
// host package and the input source (msdkReader bytes) changed.

package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

const navigationDocumentBsonType = "Navigation$NavigationDocument"

func (b *MprBackend) listNavigationDocumentsFromRaw() ([]*types.NavigationDocument, error) {
	rawUnits, err := b.msdkReader.ListRawUnitsByType(navigationDocumentBsonType)
	if err != nil {
		return nil, err
	}
	out := make([]*types.NavigationDocument, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		nav, err := parseNavigationDocumentRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
		if err != nil {
			return nil, fmt.Errorf("parse navigation %s: %w", ru.ID, err)
		}
		out = append(out, nav)
	}
	return out, nil
}

func (b *MprBackend) getNavigationFromRaw() (*types.NavigationDocument, error) {
	docs, err := b.listNavigationDocumentsFromRaw()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no navigation document found")
	}
	return docs[0], nil
}

func parseNavigationDocumentRaw(unitID, containerID string, contents []byte) (*types.NavigationDocument, error) {
	if len(contents) < 4 {
		return nil, fmt.Errorf("contents too short (%d bytes) for unit %s", len(contents), unitID)
	}
	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}

	nav := &types.NavigationDocument{
		BaseElement: model.BaseElement{
			ID:       model.ID(unitID),
			TypeName: navigationDocumentBsonType,
		},
		ContainerID: model.ID(containerID),
	}
	if name, ok := raw["Name"].(string); ok {
		nav.Name = name
	}

	for _, item := range extractBsonArray(raw["Profiles"]) {
		profMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if profile := parseNavigationProfile(profMap); profile != nil {
			nav.Profiles = append(nav.Profiles, profile)
		}
	}
	return nav, nil
}

func parseNavigationProfile(raw map[string]any) *types.NavigationProfile {
	typeName := extractString(raw["$Type"])
	profile := &types.NavigationProfile{
		Name: extractString(raw["Name"]),
		Kind: extractString(raw["Kind"]),
	}

	if typeName == "Navigation$NativeNavigationProfile" {
		profile.IsNative = true
		if hp, ok := raw["NativeHomePage"].(map[string]any); ok {
			page := extractString(hp["HomePagePage"])
			nanoflow := extractString(hp["HomePageNanoflow"])
			if page != "" || nanoflow != "" {
				profile.HomePage = &types.NavHomePage{Page: page, Microflow: nanoflow}
			}
		}
		for _, item := range extractBsonArray(raw["RoleBasedNativeHomePages"]) {
			if rbMap, ok := item.(map[string]any); ok {
				rbh := &types.NavRoleBasedHome{
					UserRole:  extractString(rbMap["UserRole"]),
					Page:      extractString(rbMap["HomePagePage"]),
					Microflow: extractString(rbMap["HomePageNanoflow"]),
				}
				if rbh.UserRole != "" {
					profile.RoleBasedHomePages = append(profile.RoleBasedHomePages, rbh)
				}
			}
		}
		for _, item := range extractBsonArray(raw["BottomBarItems"]) {
			if barMap, ok := item.(map[string]any); ok {
				if mi := parseNavMenuItemFromBottomBar(barMap); mi != nil {
					profile.MenuItems = append(profile.MenuItems, mi)
				}
			}
		}
	} else {
		if hp, ok := raw["HomePage"].(map[string]any); ok {
			page := extractString(hp["Page"])
			mf := extractString(hp["Microflow"])
			if page != "" || mf != "" {
				profile.HomePage = &types.NavHomePage{Page: page, Microflow: mf}
			}
		}
		for _, item := range extractBsonArray(raw["HomeItems"]) {
			if rbMap, ok := item.(map[string]any); ok {
				rbh := &types.NavRoleBasedHome{
					UserRole:  extractString(rbMap["UserRole"]),
					Page:      extractString(rbMap["Page"]),
					Microflow: extractString(rbMap["Microflow"]),
				}
				if rbh.UserRole != "" {
					profile.RoleBasedHomePages = append(profile.RoleBasedHomePages, rbh)
				}
			}
		}
		if lps, ok := raw["LoginPageSettings"].(map[string]any); ok {
			profile.LoginPage = extractString(lps["Form"])
		}
		if nfp, ok := raw["NotFoundHomepage"].(map[string]any); ok {
			profile.NotFoundPage = extractString(nfp["Page"])
			if profile.NotFoundPage == "" {
				profile.NotFoundPage = extractString(nfp["Microflow"])
			}
		}
		if menu, ok := raw["Menu"].(map[string]any); ok {
			for _, item := range extractBsonArray(menu["Items"]) {
				if miMap, ok := item.(map[string]any); ok {
					if mi := parseNavMenuItem(miMap); mi != nil {
						profile.MenuItems = append(profile.MenuItems, mi)
					}
				}
			}
		}
	}

	for _, item := range extractBsonArray(raw["OfflineEntityConfigs"]) {
		if oeMap, ok := item.(map[string]any); ok {
			oe := &types.NavOfflineEntity{
				Entity:     extractString(oeMap["Entity"]),
				SyncMode:   extractString(oeMap["SyncMode"]),
				Constraint: extractString(oeMap["Constraint"]),
			}
			if oe.Entity != "" {
				profile.OfflineEntities = append(profile.OfflineEntities, oe)
			}
		}
	}
	return profile
}

func parseNavMenuItem(raw map[string]any) *types.NavMenuItem {
	mi := &types.NavMenuItem{}

	if caption, ok := raw["Caption"].(map[string]any); ok {
		mi.Caption = extractTextFromBson(caption)
	}

	if action, ok := raw["Action"].(map[string]any); ok {
		actionType := extractString(action["$Type"])
		switch {
		case strings.HasSuffix(actionType, "FormAction") || strings.HasSuffix(actionType, "PageClientAction"):
			mi.ActionType = "PageAction"
			if fs, ok := action["FormSettings"].(map[string]any); ok {
				mi.Page = extractString(fs["Form"])
			}
		case strings.HasSuffix(actionType, "MicroflowAction") || strings.HasSuffix(actionType, "MicroflowClientAction"):
			mi.ActionType = "MicroflowAction"
			if ms, ok := action["MicroflowSettings"].(map[string]any); ok {
				mi.Microflow = extractString(ms["Microflow"])
			}
		case strings.HasSuffix(actionType, "OpenLinkAction") || strings.HasSuffix(actionType, "OpenLinkClientAction"):
			mi.ActionType = "OpenLinkAction"
		case strings.HasSuffix(actionType, "NoAction") || strings.HasSuffix(actionType, "NoClientAction"):
			mi.ActionType = "NoAction"
		default:
			mi.ActionType = actionType
		}
	}

	for _, item := range extractBsonArray(raw["Items"]) {
		if subMap, ok := item.(map[string]any); ok {
			if sub := parseNavMenuItem(subMap); sub != nil {
				mi.Items = append(mi.Items, sub)
			}
		}
	}

	if mi.Caption == "" && mi.Page == "" && len(mi.Items) == 0 {
		return nil
	}
	return mi
}

func parseNavMenuItemFromBottomBar(raw map[string]any) *types.NavMenuItem {
	mi := &types.NavMenuItem{}
	if caption, ok := raw["Caption"].(map[string]any); ok {
		mi.Caption = extractTextFromBson(caption)
	}
	mi.Page = extractString(raw["Page"])
	if mi.Caption == "" && mi.Page == "" {
		return nil
	}
	return mi
}

// ---------------------------------------------------------------------------
// Shared BSON helpers (also used by other *_compat.go files in this package).
// Mirrors sdk/mpr/parser.go::extract* helpers.
// ---------------------------------------------------------------------------

func extractString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func extractBsonArray(v any) []any {
	if v == nil {
		return nil
	}
	arr, ok := v.(primitive.A)
	if !ok {
		if slice, ok := v.([]any); ok {
			if len(slice) > 0 {
				if marker, ok := slice[0].(int32); ok && (marker == 2 || marker == 3) {
					return slice[1:]
				}
			}
			return slice
		}
		return nil
	}
	slice := []any(arr)
	if len(slice) > 0 {
		if marker, ok := slice[0].(int32); ok && (marker == 2 || marker == 3) {
			return slice[1:]
		}
	}
	return slice
}

func extractTextFromBson(raw map[string]any) string {
	for _, item := range extractBsonArray(raw["Items"]) {
		if transMap, ok := item.(map[string]any); ok {
			text := extractString(transMap["Text"])
			if text != "" {
				return text
			}
		}
	}
	for _, item := range extractBsonArray(raw["Translations"]) {
		if transMap, ok := item.(map[string]any); ok {
			text := extractString(transMap["Text"])
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func extractBool(v any, defaultVal bool) bool {
	if v == nil {
		return defaultVal
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return defaultVal
}

func extractInt(v any) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	}
	return 0
}

func extractBsonID(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return blobToUUID(val)
	case primitive.Binary:
		return blobToUUID(val.Data)
	case map[string]any:
		if id, ok := val["$ID"].(string); ok {
			return id
		}
		if b, ok := val["$ID"].(primitive.Binary); ok {
			return blobToUUID(b.Data)
		}
		if b, ok := val["$ID"].([]byte); ok {
			return blobToUUID(b)
		}
	}
	return ""
}

func extractBsonMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		return val
	case primitive.D:
		return val.Map()
	case primitive.M:
		return map[string]any(val)
	}
	return nil
}

// blobToUUID converts a 16-byte BSON binary into the canonical UUID string.
func blobToUUID(b []byte) string {
	if len(b) != 16 {
		return ""
	}
	return fmt.Sprintf(
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	)
}
