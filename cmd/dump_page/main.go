// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: dump_page <mpr_path> <search_pattern>")
		os.Exit(1)
	}

	mprPath := os.Args[1]
	pattern := strings.ToLower(os.Args[2])

	mprContentsDir := filepath.Join(filepath.Dir(mprPath), "mprcontents")

	err := filepath.Walk(mprContentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".mxunit") {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var doc bson.D
		if err = bson.Unmarshal(contents, &doc); err != nil {
			return nil
		}

		var docType, docName string
		for _, elem := range doc {
			if elem.Key == "$Type" {
				docType, _ = elem.Value.(string)
			}
			if elem.Key == "Name" {
				docName, _ = elem.Value.(string)
			}
		}

		if strings.Contains(strings.ToLower(docName), pattern) {
			fmt.Printf("=== %s (%s) ===\n", docName, docType)
			jsonBytes, _ := json.MarshalIndent(doc, "", "  ")
			fmt.Println(string(jsonBytes))
			os.Exit(0) // Stop after first match
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
}
