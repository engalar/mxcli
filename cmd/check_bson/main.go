// SPDX-License-Identifier: Apache-2.0

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/v2/bson"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: check_bson <mpr_path> <page_name_pattern>")
		os.Exit(1)
	}

	mprPath := os.Args[1]
	pagePattern := os.Args[2]

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer db.Close()

	// Find page
	row := db.QueryRow(`SELECT UnitID FROM Unit WHERE Name LIKE ?`, "%"+pagePattern+"%")
	var unitID string
	if err := row.Scan(&unitID); err != nil {
		fmt.Printf("Error finding page: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Found Unit ID: %s\n", unitID)

	// Read contents file - check for mprcontents folder (v2 format)
	contentsPath := fmt.Sprintf("%s/../mprcontents/%s", mprPath, unitID)
	data, err := os.ReadFile(contentsPath)
	if err != nil {
		// Try alternative path
		contentsPath = fmt.Sprintf("mx-test-projects/test2-go-app/mprcontents/%s", unitID)
		data, err = os.ReadFile(contentsPath)
		if err != nil {
			fmt.Printf("Error reading contents: %v\n", err)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Contents size: %d bytes\n", len(data))

	// First 4 bytes are length
	bsonData := data[4:]

	var doc bson.D
	if err := bson.Unmarshal(bsonData, &doc); err != nil {
		fmt.Printf("Error unmarshaling BSON: %v\n", err)
		return
	}

	// Convert to JSON for display - full output
	jsonBytes, _ := json.MarshalIndent(bsonToMap(doc), "", "  ")
	fmt.Println(string(jsonBytes))
}

func bsonToMap(d bson.D) map[string]any {
	result := make(map[string]any)
	for _, elem := range d {
		result[elem.Key] = bsonValueToInterface(elem.Value)
	}
	return result
}

func bsonValueToInterface(v any) any {
	switch val := v.(type) {
	case bson.D:
		return bsonToMap(val)
	case bson.A:
		arr := make([]any, len(val))
		for i, item := range val {
			arr[i] = bsonValueToInterface(item)
		}
		return arr
	case []byte:
		// Binary ID - show as hex
		return fmt.Sprintf("binary[%d]:%x", len(val), val)
	default:
		return val
	}
}
