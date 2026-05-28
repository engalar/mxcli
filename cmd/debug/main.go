// SPDX-License-Identifier: Apache-2.0

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.mongodb.org/mongo-driver/v2/bson"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: debug <mprPath> <docID>\n")
		os.Exit(1)
	}

	mprPath := os.Args[1]
	docID := os.Args[2]

	db, err := sql.Open("sqlite", mprPath+"?mode=ro")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Try unit table first (MPR v1), then documents (MPR v2)
	var content []byte
	err = db.QueryRow("SELECT Contents FROM Unit WHERE ContainmentID = ?", docID).Scan(&content)
	if err != nil {
		// Try MPR v2 - look for mprcontents folder with nested directory structure
		// Structure is: mprcontents/<first2chars>/<next2chars>/<guid>.mxunit
		dir1 := docID[0:2]
		dir2 := docID[2:4]
		contentsPath := filepath.Join(filepath.Dir(mprPath), "mprcontents", dir1, dir2, docID+".mxunit")
		content, err = os.ReadFile(contentsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to find document: %v\n", err)
			os.Exit(1)
		}
	}

	var doc bson.M
	if err := bson.Unmarshal(content, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal BSON: %v\n", err)
		os.Exit(1)
	}

	data, _ := json.MarshalIndent(doc, "", "  ")
	fmt.Println(string(data))
}
