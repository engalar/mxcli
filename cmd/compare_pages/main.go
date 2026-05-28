// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: compare_pages <file1.mxunit> <file2.mxunit>\n")
		os.Exit(1)
	}

	file1, file2 := os.Args[1], os.Args[2]

	fmt.Printf("=== Comparing %s vs %s ===\n\n", file1, file2)

	// Read and decode file 1
	data1, err := os.ReadFile(file1)
	if err != nil {
		fmt.Printf("Error reading file 1: %v\n", err)
		return
	}

	var doc1 bson.D
	if err := bson.Unmarshal(data1, &doc1); err != nil {
		fmt.Printf("Error unmarshaling file 1: %v\n", err)
		return
	}

	// Read and decode file 2
	data2, err := os.ReadFile(file2)
	if err != nil {
		fmt.Printf("Error reading file 2: %v\n", err)
		return
	}

	var doc2 bson.D
	if err := bson.Unmarshal(data2, &doc2); err != nil {
		fmt.Printf("Error unmarshaling file 2: %v\n", err)
		return
	}

	// Find and compare DataGrid2 widgets
	dg1 := findDataGrid2(doc1)
	dg2 := findDataGrid2(doc2)

	if dg1 == nil {
		fmt.Println("WARNING: DataGrid2 not found in file 1, dumping full page")
		json1 := bsonDToOrderedJSON(doc1, 0)
		os.WriteFile("/tmp/page1_broken.json", []byte(json1), 0644)
	} else {
		json1 := bsonDToOrderedJSON(dg1, 0)
		os.WriteFile("/tmp/page1_broken.json", []byte(json1), 0644)
		fmt.Printf("Page 1 DataGrid2: /tmp/page1_broken.json (%d bytes)\n", len(json1))
	}

	if dg2 == nil {
		fmt.Println("WARNING: DataGrid2 not found in file 2, dumping full page")
		json2 := bsonDToOrderedJSON(doc2, 0)
		os.WriteFile("/tmp/page2_fixed.json", []byte(json2), 0644)
	} else {
		json2 := bsonDToOrderedJSON(dg2, 0)
		os.WriteFile("/tmp/page2_fixed.json", []byte(json2), 0644)
		fmt.Printf("Page 2 DataGrid2: /tmp/page2_fixed.json (%d bytes)\n", len(json2))
	}

	fmt.Println("\nRun: diff /tmp/page1_broken.json /tmp/page2_fixed.json")
}

// bsonDToOrderedJSON converts bson.D to JSON string preserving key order
func bsonDToOrderedJSON(d bson.D, indent int) string {
	prefix := strings.Repeat("  ", indent)
	inner := strings.Repeat("  ", indent+1)
	var sb strings.Builder
	sb.WriteString("{\n")
	for i, elem := range d {
		sb.WriteString(inner)
		keyJSON, _ := json.Marshal(elem.Key)
		sb.Write(keyJSON)
		sb.WriteString(": ")
		sb.WriteString(valueToOrderedJSON(elem.Value, indent+1))
		if i < len(d)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(prefix)
	sb.WriteString("}")
	return sb.String()
}

// valueToOrderedJSON converts a BSON value to ordered JSON string
func valueToOrderedJSON(v any, indent int) string {
	switch val := v.(type) {
	case bson.D:
		return bsonDToOrderedJSON(val, indent)
	case bson.A:
		if len(val) == 0 {
			return "[]"
		}
		prefix := strings.Repeat("  ", indent)
		inner := strings.Repeat("  ", indent+1)
		var sb strings.Builder
		sb.WriteString("[\n")
		for i, item := range val {
			sb.WriteString(inner)
			sb.WriteString(valueToOrderedJSON(item, indent+1))
			if i < len(val)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString(prefix)
		sb.WriteString("]")
		return sb.String()
	case []byte:
		s, _ := json.Marshal(fmt.Sprintf("0x%x", val))
		return string(s)
	default:
		s, _ := json.Marshal(val)
		return string(s)
	}
}

// findDataGrid2 searches for a DataGrid2 widget in the page structure
func findDataGrid2(doc bson.D) bson.D {
	// Try with known DataGrid2 widget IDs
	ids := []string{
		"com.mendix.widget.web.datagrid.Datagrid",
		"DataGrid2.DataGrid2",
	}
	for _, id := range ids {
		if result := findCustomWidget(doc, id); result != nil {
			return result
		}
	}
	return nil
}

// findCustomWidget finds a CustomWidgets$CustomWidget by WidgetId in nested Type
func findCustomWidget(doc bson.D, widgetId string) bson.D {
	docType := getFieldString(doc, "$Type")
	if docType == "CustomWidgets$CustomWidget" {
		// Check Type.WidgetId
		for _, elem := range doc {
			if elem.Key == "Type" {
				if typeDoc, ok := elem.Value.(bson.D); ok {
					wid := getFieldString(typeDoc, "WidgetId")
					if wid == widgetId {
						return doc
					}
				}
			}
		}
	}
	for _, elem := range doc {
		switch val := elem.Value.(type) {
		case bson.D:
			if result := findCustomWidget(val, widgetId); result != nil {
				return result
			}
		case bson.A:
			for _, item := range val {
				if d, ok := item.(bson.D); ok {
					if result := findCustomWidget(d, widgetId); result != nil {
						return result
					}
				}
			}
		}
	}
	return nil
}

// findWidgetByType recursively searches for a widget by type and widgetId
func findWidgetByType(doc bson.D, typeName string, widgetId string) bson.D {
	docType := getFieldString(doc, "$Type")

	// Check if this is the widget we're looking for
	if docType == typeName {
		wid := getFieldString(doc, "WidgetId")
		if widgetId == "" || wid == widgetId {
			return doc
		}
	}

	// Recursively search in all array and document fields
	for _, elem := range doc {
		switch val := elem.Value.(type) {
		case bson.D:
			if result := findWidgetByType(val, typeName, widgetId); result != nil {
				return result
			}
		case bson.A:
			for _, item := range val {
				if d, ok := item.(bson.D); ok {
					if result := findWidgetByType(d, typeName, widgetId); result != nil {
						return result
					}
				}
			}
		}
	}

	return nil
}

// getFieldString gets a string field from a bson.D
func getFieldString(doc bson.D, key string) string {
	for _, elem := range doc {
		if elem.Key == key {
			if s, ok := elem.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}
