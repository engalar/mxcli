// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/sdk/pages"

	"go.mongodb.org/mongo-driver/bson"
)

// serializeContainer serializes a Container widget.
func serializeContainer(c *pages.Container) bson.D {
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(c.ID))},
		{Key: "$Type", Value: "Forms$DivContainer"},
		{Key: "Appearance", Value: serializeAppearance(c.Class, c.Style, c.DesignProperties)},
		{Key: "ConditionalVisibilitySettings", Value: nil},
		{Key: "Name", Value: c.Name},
		{Key: "NativeAccessibilitySettings", Value: nil},
		{Key: "OnClickAction", Value: serializeClientAction(nil)},
		{Key: "RenderMode", Value: "Div"},
		{Key: "ScreenReaderHidden", Value: false},
		{Key: "TabIndex", Value: int64(0)},
		{Key: "Widgets", Value: serializeWidgetArray(c.Widgets)},
	}
	return doc
}

// serializeGroupBox serializes a GroupBox widget.
func serializeGroupBox(gb *pages.GroupBox) bson.D {
	collapsible := gb.Collapsible
	if collapsible == "" {
		collapsible = "No"
	}
	headerMode := gb.HeaderMode
	if headerMode == "" {
		headerMode = "Div"
	}

	// Serialize CaptionTemplate
	var captionTemplate bson.D
	if gb.Caption != nil {
		captionTemplate = serializeClientTemplate(gb.Caption, nil, "")
	} else {
		captionTemplate = serializeClientTemplate(nil, nil, "")
	}

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(gb.ID))},
		{Key: "$Type", Value: "Forms$GroupBox"},
		{Key: "Appearance", Value: serializeAppearance(gb.Class, gb.Style, gb.DesignProperties)},
		{Key: "CaptionTemplate", Value: captionTemplate},
		{Key: "Collapsible", Value: collapsible},
		{Key: "ConditionalVisibilitySettings", Value: nil},
		{Key: "HeaderMode", Value: headerMode},
		{Key: "Name", Value: gb.Name},
		{Key: "TabIndex", Value: int64(0)},
		{Key: "Widgets", Value: serializeWidgetArray(gb.Widgets)},
	}
	return doc
}

// serializeLayoutGrid serializes a LayoutGrid widget.
func serializeLayoutGrid(lg *pages.LayoutGrid) bson.D {
	// Mendix uses [3] for empty arrays, [2, item1, item2, ...] for non-empty arrays
	// Items go directly after the version marker, NOT nested in another array
	rows := bson.A{int32(3)} // Start with empty marker
	hasRows := false
	for _, row := range lg.Rows {
		if !hasRows {
			rows = bson.A{int64(2)} // First item: change to version 2
			hasRows = true
		}
		rows = append(rows, serializeLayoutGridRow(row))
	}

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(lg.ID))},
		{Key: "$Type", Value: "Forms$LayoutGrid"},
		{Key: "Appearance", Value: serializeAppearance(lg.Class, lg.Style, lg.DesignProperties)},
		{Key: "ConditionalVisibilitySettings", Value: nil},
		{Key: "Name", Value: lg.Name},
		{Key: "Rows", Value: rows},
		{Key: "TabIndex", Value: int64(0)},
		{Key: "Width", Value: "FullWidth"},
	}
	return doc
}

// serializeLayoutGridRow serializes a LayoutGridRow.
func serializeLayoutGridRow(row *pages.LayoutGridRow) bson.D {
	// Mendix uses [3] for empty arrays, [2, item1, item2, ...] for non-empty arrays
	// Items go directly after the version marker, NOT nested in another array
	cols := bson.A{int32(3)} // Start with empty marker
	hasCols := false
	for _, col := range row.Columns {
		if !hasCols {
			cols = bson.A{int64(2)} // First item: change to version 2
			hasCols = true
		}
		cols = append(cols, serializeLayoutGridColumn(col))
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(row.ID))},
		{Key: "$Type", Value: "Forms$LayoutGridRow"},
		{Key: "Appearance", Value: serializeAppearance("", "", nil)},
		{Key: "Columns", Value: cols},
		{Key: "ConditionalVisibilitySettings", Value: nil},
		{Key: "HorizontalAlignment", Value: "None"},
		{Key: "SpacingBetweenColumns", Value: true},
		{Key: "VerticalAlignment", Value: "None"},
	}
}

// columnWeight returns the column weight, defaulting to -1 (auto) if 0.
func columnWeight(w int) int {
	if w == 0 {
		return -1
	}
	return w
}

// serializeLayoutGridColumn serializes a LayoutGridColumn.
func serializeLayoutGridColumn(col *pages.LayoutGridColumn) bson.D {
	// Weight for column width: -1 means auto-fill, 1-12 are explicit widths
	weight := col.Weight
	if weight == 0 {
		weight = -1 // Default to auto-fill
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(col.ID))},
		{Key: "$Type", Value: "Forms$LayoutGridColumn"},
		{Key: "Appearance", Value: serializeAppearance("", "", nil)},
		{Key: "PhoneWeight", Value: int64(columnWeight(col.PhoneWeight))},
		{Key: "PreviewWidth", Value: int64(-1)}, // Default preview width
		{Key: "TabletWeight", Value: int64(columnWeight(col.TabletWeight))},
		{Key: "VerticalAlignment", Value: "None"},
		{Key: "Weight", Value: int64(weight)}, // Desktop weight
		{Key: "Widgets", Value: serializeWidgetArray(col.Widgets)},
	}
}
