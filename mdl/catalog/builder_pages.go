// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (b *Builder) buildPages() error {
	// Get all pages (cached — reused by buildReferences and buildStrings)
	pageList, err := b.cachedPages()
	if err != nil {
		return err
	}

	pageStmt, err := b.tx.Prepare(`
		INSERT INTO pages (Id, Name, QualifiedName, ModuleName, Folder, Title, URL, LayoutRef,
			Description, ParameterCount, WidgetCount, Excluded,
			ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
			SourceId, SourceBranch, SourceRevision)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer pageStmt.Close()

	// Prepare widget statement only in full mode
	var widgetStmt *sql.Stmt
	if b.fullMode {
		widgetStmt, err = b.tx.Prepare(`
			INSERT INTO widgets (Id, Name, WidgetType, ContainerId, ContainerQualifiedName, ContainerType,
				ModuleName, Folder, EntityRef, AttributeRef, Description,
				ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
				SourceId, SourceBranch, SourceRevision)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer widgetStmt.Close()
	}

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, sourceID, sourceBranch, sourceRevision := b.snapshotMeta()

	pageCount := 0
	widgetCount := 0

	for _, pg := range pageList {
		// Get module name
		moduleID := b.hierarchy.findModuleID(pg.ContainerID)
		moduleName := b.hierarchy.getModuleName(moduleID)
		qualifiedName := moduleName + "." + pg.Name
		folder := b.hierarchy.buildFolderPath(pg.ContainerID)

		title := ""
		if pg.Title != nil && pg.Title.Translations != nil {
			for _, t := range pg.Title.Translations {
				title = t
				break
			}
		}

		// Get layout ref and widgets from raw BSON data
		layoutRef := ""
		widgetCnt := 0
		var rawWidgets []rawWidgetInfo

		if b.fullMode {
			rawData, _ := b.reader.GetRawUnit(pg.ID)
			if rawData != nil {
				layoutRef = extractLayoutRef(rawData)
				rawWidgets = extractPageWidgets(rawData, string(pg.ID))
				widgetCnt = len(rawWidgets)
			}
		}

		_, err = pageStmt.Exec(
			string(pg.ID),
			pg.Name,
			qualifiedName,
			moduleName,
			folder,
			title,
			pg.URL,
			layoutRef,
			pg.Documentation,
			len(pg.Parameters),
			widgetCnt,
			pg.Excluded,
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			sourceID, sourceBranch, sourceRevision,
		)
		if err != nil {
			return err
		}
		pageCount++

		// Insert widgets only in full mode
		if b.fullMode && len(rawWidgets) > 0 {
			for _, w := range rawWidgets {
				if _, err := widgetStmt.Exec(
					w.ID,
					w.Name,
					w.WidgetType,
					string(pg.ID),
					qualifiedName,
					"PAGE",
					moduleName,
					folder,
					w.EntityRef,
					w.AttributeRef,
					"",
					projectID, projectName, snapshotID, snapshotDate, snapshotSource,
					sourceID, sourceBranch, sourceRevision,
				); err != nil {
					return fmt.Errorf("insert widget %s for page %s: %w", w.Name, qualifiedName, err)
				}
				widgetCount++
			}
		}
	}

	b.report("Pages", pageCount)
	if b.fullMode {
		b.report("Widgets", widgetCount)
	}
	return nil
}

func (b *Builder) buildSnippets() error {
	// Stage 3.3.5.C7e: walks gen-typed Snippet units. Container linkage
	// resolved via the unit hierarchy. ParameterCount uses gen
	// ParametersItems(); the legacy code hard-coded 0.
	snippetGenList, err := b.cachedSnippetsGen()
	if err != nil {
		return err
	}

	snippetStmt, err := b.tx.Prepare(`
		INSERT INTO snippets (Id, Name, QualifiedName, ModuleName, Folder, Description,
			ParameterCount, WidgetCount,
			ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
			SourceId, SourceBranch, SourceRevision)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer snippetStmt.Close()

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, sourceID, sourceBranch, sourceRevision := b.snapshotMeta()

	count := 0
	for _, sn := range snippetGenList {
		if sn == nil {
			continue
		}
		snID := model.ID(sn.ID())
		containerID := b.hierarchy.containerParent[snID]
		moduleID := b.hierarchy.findModuleID(containerID)
		moduleName := b.hierarchy.getModuleName(moduleID)
		qualifiedName := moduleName + "." + sn.Name()
		folder := b.hierarchy.buildFolderPath(containerID)

		_, err := snippetStmt.Exec(
			string(snID),
			sn.Name(),
			qualifiedName,
			moduleName,
			folder,
			sn.Documentation(),
			len(sn.ParametersItems()),
			0, // WidgetCount — gen widget walking deferred to D-phase
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			sourceID, sourceBranch, sourceRevision,
		)
		if err != nil {
			return err
		}
		count++
	}

	b.report("Snippets", count)
	return nil
}

// rawWidgetInfo represents a widget extracted from raw BSON data.
type rawWidgetInfo struct {
	ID           string
	Name         string
	WidgetType   string
	EntityRef    string
	AttributeRef string
}

// extractLayoutRef extracts the layout reference from raw page BSON.
func extractLayoutRef(rawData map[string]any) string {
	formCall, ok := rawData["FormCall"].(map[string]any)
	if !ok {
		return ""
	}

	// Try Form field first (string layout name)
	if formName, ok := formCall["Form"].(string); ok && formName != "" {
		return formName
	}

	// Try Layout field (binary GUID) - extract and format
	if layoutID := extractBinaryID(formCall["Layout"]); layoutID != "" {
		return layoutID // Will be a GUID string
	}

	return ""
}

// extractPageWidgets extracts all widgets from raw page BSON data.
func extractPageWidgets(rawData map[string]any, containerID string) []rawWidgetInfo {
	formCall, ok := rawData["FormCall"].(map[string]any)
	if !ok {
		return nil
	}

	// Get Arguments array
	args := getBsonArrayElements(formCall["Arguments"])
	if args == nil {
		return nil
	}

	var widgets []rawWidgetInfo
	for _, arg := range args {
		argMap, ok := arg.(map[string]any)
		if !ok {
			continue
		}
		// Each argument has a Widgets array
		argWidgets := getBsonArrayElements(argMap["Widgets"])
		for _, w := range argWidgets {
			if wMap, ok := w.(map[string]any); ok {
				widgets = append(widgets, extractWidgetsRecursive(wMap)...)
			}
		}
	}
	return widgets
}

// extractWidgetsRecursive recursively extracts widgets from a widget map.
func extractWidgetsRecursive(w map[string]any) []rawWidgetInfo {
	var result []rawWidgetInfo

	// Extract this widget's info
	widget := rawWidgetInfo{
		ID:         extractBsonID(w["$ID"]),
		Name:       extractString(w["Name"]),
		WidgetType: extractString(w["$Type"]),
	}

	// Handle CustomWidgets - get the actual widget type ID
	if widget.WidgetType == "CustomWidgets$CustomWidget" {
		if typeObj, ok := w["Type"].(map[string]any); ok {
			if widgetID := extractString(typeObj["WidgetId"]); widgetID != "" {
				widget.WidgetType = widgetID
			}
		}
	}

	// Extract attribute reference for input widgets
	if attrRef, ok := w["AttributeRef"].(map[string]any); ok {
		widget.AttributeRef = extractString(attrRef["Attribute"])
	}

	// Skip DivContainer wrapper types - only collect their children
	if widget.WidgetType != "Forms$DivContainer" && widget.WidgetType != "Pages$DivContainer" {
		result = append(result, widget)
	}

	// Recurse into child widgets
	childWidgets := getBsonArrayElements(w["Widgets"])
	for _, child := range childWidgets {
		if childMap, ok := child.(map[string]any); ok {
			result = append(result, extractWidgetsRecursive(childMap)...)
		}
	}

	// Handle LayoutGrid rows/columns
	rows := getBsonArrayElements(w["Rows"])
	for _, row := range rows {
		if rowMap, ok := row.(map[string]any); ok {
			cols := getBsonArrayElements(rowMap["Columns"])
			for _, col := range cols {
				if colMap, ok := col.(map[string]any); ok {
					colWidgets := getBsonArrayElements(colMap["Widgets"])
					for _, cw := range colWidgets {
						if cwMap, ok := cw.(map[string]any); ok {
							result = append(result, extractWidgetsRecursive(cwMap)...)
						}
					}
				}
			}
		}
	}

	// Handle DataView/ListView footer widgets
	footerWidgets := getBsonArrayElements(w["FooterWidgets"])
	for _, fw := range footerWidgets {
		if fwMap, ok := fw.(map[string]any); ok {
			result = append(result, extractWidgetsRecursive(fwMap)...)
		}
	}

	// Handle TabContainer tab pages
	tabPages := getBsonArrayElements(w["TabPages"])
	for _, tp := range tabPages {
		if tpMap, ok := tp.(map[string]any); ok {
			tpWidgets := getBsonArrayElements(tpMap["Widgets"])
			for _, tw := range tpWidgets {
				if twMap, ok := tw.(map[string]any); ok {
					result = append(result, extractWidgetsRecursive(twMap)...)
				}
			}
		}
	}

	// Handle CustomWidget nested widgets in properties
	if obj, ok := w["Object"].(map[string]any); ok {
		props := getBsonArrayElements(obj["Properties"])
		for _, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				if value, ok := propMap["Value"].(map[string]any); ok {
					propWidgets := getBsonArrayElements(value["Widgets"])
					for _, pw := range propWidgets {
						if pwMap, ok := pw.(map[string]any); ok {
							result = append(result, extractWidgetsRecursive(pwMap)...)
						}
					}
				}
			}
		}
	}

	// Handle NavigationList items
	items := getBsonArrayElements(w["Items"])
	for _, item := range items {
		if itemMap, ok := item.(map[string]any); ok {
			itemWidgets := getBsonArrayElements(itemMap["Widgets"])
			for _, iw := range itemWidgets {
				if iwMap, ok := iw.(map[string]any); ok {
					result = append(result, extractWidgetsRecursive(iwMap)...)
				}
			}
		}
	}

	return result
}

// extractSnippetWidgets extracts all widgets from raw snippet BSON data.
func extractSnippetWidgets(rawData map[string]any) []rawWidgetInfo {
	// Handle both snippet formats:
	// - Studio Pro uses "Widgets" (plural): a top-level array of widgets
	// - mxcli uses "Widget" (singular): a single container whose "Widgets" field holds children
	var widgetsArray []any
	if wa := getBsonArrayElements(rawData["Widgets"]); wa != nil {
		widgetsArray = wa
	} else if widgetContainer, ok := rawData["Widget"].(map[string]any); ok {
		widgetsArray = getBsonArrayElements(widgetContainer["Widgets"])
	}
	if widgetsArray == nil {
		return nil
	}

	var widgets []rawWidgetInfo
	for _, w := range widgetsArray {
		if wMap, ok := w.(map[string]any); ok {
			widgets = append(widgets, extractWidgetsRecursive(wMap)...)
		}
	}
	return widgets
}

// getBsonArrayElements extracts array elements from BSON array format.
// BSON arrays have format [typeIndicator, item1, item2, ...] where typeIndicator is a number.
func getBsonArrayElements(v any) []any {
	arr := toBsonArray(v)
	if len(arr) == 0 {
		return nil
	}
	// Check if first element is a type indicator (integer)
	if _, ok := arr[0].(int32); ok {
		return arr[1:]
	}
	if _, ok := arr[0].(int); ok {
		return arr[1:]
	}
	// No type indicator, return as-is
	return arr
}

// toBsonArray converts various BSON array types to []interface{}.
func toBsonArray(v any) []any {
	switch arr := v.(type) {
	case []any:
		return arr
	case primitive.A:
		return []any(arr)
	default:
		return nil
	}
}

// extractString extracts a string value from an interface.
func extractString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// extractBsonID extracts an ID from a BSON field (can be string or binary).
func extractBsonID(v any) string {
	if v == nil {
		return ""
	}

	// Try string first
	if s, ok := v.(string); ok {
		return s
	}

	// Try BSON binary format: {"Subtype": 0, "Data": "base64..."}
	if m, ok := v.(map[string]any); ok {
		if data, ok := m["Data"].(string); ok {
			// Data is base64-encoded GUID, decode and format
			return decodeBase64GUID(data)
		}
	}

	// Try primitive.Binary
	if bin, ok := v.(primitive.Binary); ok {
		return formatGUID(bin.Data)
	}

	return ""
}

// decodeBase64GUID decodes a base64-encoded GUID and formats it.
func decodeBase64GUID(data string) string {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil || len(decoded) != 16 {
		return data // Return as-is if can't decode
	}
	return formatGUID(decoded)
}

// extractBinaryID extracts a UUID string from a BSON binary or string value.
func extractBinaryID(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return formatGUID(val)
	case primitive.Binary:
		return formatGUID(val.Data)
	default:
		return ""
	}
}

// formatGUID converts a 16-byte GUID to its string representation with proper byte ordering.
func formatGUID(data []byte) string {
	if len(data) != 16 {
		return string(data)
	}
	// Reverse first 4 bytes (group 1)
	g1 := []byte{data[3], data[2], data[1], data[0]}
	// Reverse next 2 bytes (group 2)
	g2 := []byte{data[5], data[4]}
	// Reverse next 2 bytes (group 3)
	g3 := []byte{data[7], data[6]}
	// Last 8 bytes stay in order
	return strings.ToLower(strings.Join([]string{
		bytesToHex(g1),
		bytesToHex(g2),
		bytesToHex(g3),
		bytesToHex(data[8:10]),
		bytesToHex(data[10:16]),
	}, "-"))
}

// bytesToHex converts a byte slice to a hex string.
func bytesToHex(data []byte) string {
	const hex = "0123456789abcdef"
	result := make([]byte, len(data)*2)
	for i, b := range data {
		result[i*2] = hex[b>>4]
		result[i*2+1] = hex[b&0x0f]
	}
	return string(result)
}

func (b *Builder) buildLayouts() error {
	// Stage 3.3.5.C7e: walks gen-typed Layout units. Container linkage
	// resolved via the unit hierarchy.
	layoutGenList, err := b.cachedLayoutsGen()
	if err != nil {
		return err
	}

	layoutStmt, err := b.tx.Prepare(`
		INSERT INTO layouts (Id, Name, QualifiedName, ModuleName, Folder, LayoutType, Description,
			ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
			SourceId, SourceBranch, SourceRevision)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer layoutStmt.Close()

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, sourceID, sourceBranch, sourceRevision := b.snapshotMeta()

	count := 0
	for _, l := range layoutGenList {
		if l == nil {
			continue
		}
		lID := model.ID(l.ID())
		containerID := b.hierarchy.containerParent[lID]
		moduleID := b.hierarchy.findModuleID(containerID)
		moduleName := b.hierarchy.getModuleName(moduleID)
		qualifiedName := moduleName + "." + l.Name()
		folder := b.hierarchy.buildFolderPath(containerID)

		_, err := layoutStmt.Exec(
			string(lID),
			l.Name(),
			qualifiedName,
			moduleName,
			folder,
			l.LayoutType(),
			l.Documentation(),
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			sourceID, sourceBranch, sourceRevision,
		)
		if err != nil {
			return err
		}
		count++
	}

	b.report("Layouts", count)
	return nil
}
