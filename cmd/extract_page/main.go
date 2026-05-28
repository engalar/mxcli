// SPDX-License-Identifier: Apache-2.0

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: extract_page <mpr_file> <page_name> [output_file]")
		os.Exit(1)
	}

	mprFile := os.Args[1]
	pageName := os.Args[2]
	outputFile := ""
	if len(os.Args) > 3 {
		outputFile = os.Args[3]
	}

	// Open the MPR database
	db, err := sql.Open("sqlite", mprFile+"?mode=ro")
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Find the page by searching through all units
	rows, err := db.Query("SELECT UnitID, Contents FROM Unit WHERE Contents IS NOT NULL")
	if err != nil {
		fmt.Printf("Error querying units: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	count := 0
	pageCount := 0
	for rows.Next() {
		var unitID []byte
		var contents []byte
		if err := rows.Scan(&unitID, &contents); err != nil {
			continue
		}

		count++
		if len(contents) == 0 {
			continue
		}

		// Try to unmarshal and check if it's our page
		var doc bson.D
		if err := bson.Unmarshal(contents, &doc); err != nil {
			continue
		}

		// Check if this is the page we're looking for
		name := getFieldString(doc, "Name")
		docType := getFieldString(doc, "$Type")

		// Debug: show all pages
		if docType == "Forms$Page" {
			pageCount++
			fmt.Printf("Found page #%d: %s\n", pageCount, name)
		}

		if docType == "Forms$Page" && (name == pageName || strings.HasSuffix(name, "."+pageName)) {
			// Found it!
			jsonData, err := json.MarshalIndent(bsonDToMap(doc), "", "  ")
			if err != nil {
				fmt.Printf("Error converting to JSON: %v\n", err)
				os.Exit(1)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
					fmt.Printf("Error writing output: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Written %d bytes to %s\n", len(jsonData), outputFile)
			} else {
				fmt.Println(string(jsonData))
			}
			return
		}
	}

	fmt.Printf("Scanned %d units, found %d pages, but not %s\n", count, pageCount, pageName)
	os.Exit(1)
}

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

func bsonDToMap(d bson.D) map[string]any {
	result := make(map[string]any)
	for _, elem := range d {
		result[elem.Key] = convertValue(elem.Value)
	}
	return result
}

func convertValue(v any) any {
	switch val := v.(type) {
	case bson.D:
		return bsonDToMap(val)
	case bson.A:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = convertValue(item)
		}
		return result
	case []byte:
		return fmt.Sprintf("0x%x", val)
	default:
		return val
	}
}
