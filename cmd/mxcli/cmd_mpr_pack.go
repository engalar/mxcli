// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"

	"github.com/mendixlabs/mxcli/modelsdk/mpr"
)

var mprPackCmd = &cobra.Command{
	Use:   "mpr-pack",
	Short: "Convert MPR files between v1 (single file) and v2 (mprcontents folder) formats",
	Long: `mpr-pack converts Mendix .mpr files between format versions.

MPR v1: single SQLite file with Contents BLOB column in the Unit table.
MPR v2: SQLite metadata file + mprcontents/ folder with individual .mxunit files.

Subcommands:
  to-v1  <input.mpr> <output.mpr>  Convert a v2 MPR to v1 (inline all .mxunit into the SQLite file)
  to-v2  <input.mpr> <output.mpr>  Convert a v1 MPR to v2 (extract inline BLOB into .mxunit files)
`,
}

var mprPackToV1Cmd = &cobra.Command{
	Use:   "to-v1 <input.mpr> <output.mpr>",
	Short: "Convert v2 MPR (mprcontents folder) to v1 (single SQLite file)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return convertV2ToV1(args[0], args[1])
	},
}

var mprPackToV2Cmd = &cobra.Command{
	Use:   "to-v2 <input.mpr> <output.mpr>",
	Short: "Convert v1 MPR (single SQLite file) to v2 (mprcontents folder)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return convertV1ToV2(args[0], args[1])
	},
}

func init() {
	mprPackCmd.AddCommand(mprPackToV1Cmd)
	mprPackCmd.AddCommand(mprPackToV2Cmd)
}

// convertV2ToV1 reads a v2 MPR (SQLite + mprcontents/) and writes a v1 MPR (single SQLite file).
func convertV2ToV1(inputMPR, outputMPR string) error {
	inputMPR, err := filepath.Abs(inputMPR)
	if err != nil {
		return fmt.Errorf("resolving input path: %w", err)
	}
	outputMPR, err = filepath.Abs(outputMPR)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	contentsDir := filepath.Join(filepath.Dir(inputMPR), "mprcontents")
	if _, err := os.Stat(contentsDir); err != nil {
		return fmt.Errorf("mprcontents directory not found at %s (is this really a v2 MPR?): %w", contentsDir, err)
	}

	// Open input (read-only).
	srcDB, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", inputMPR))
	if err != nil {
		return fmt.Errorf("opening input MPR: %w", err)
	}
	defer srcDB.Close()

	// Remove existing output if present, then create fresh.
	_ = os.Remove(outputMPR)
	dstDB, err := sql.Open("sqlite", outputMPR)
	if err != nil {
		return fmt.Errorf("creating output MPR: %w", err)
	}
	defer dstDB.Close()

	// Create v1 schema.
	v1Schema := []string{
		`CREATE TABLE _MetaData (
			_ProductVersion TEXT,
			_BuildVersion TEXT,
			_SchemaHash TEXT,
			_DisableAutoMprV2Upgrade INTEGER
		)`,
		`CREATE TABLE Unit (
			UnitID BLOB PRIMARY KEY NOT NULL,
			ContainerID BLOB,
			ContainmentName TEXT,
			TreeConflict LONG,
			ContentsHash TEXT,
			ContentsConflicts TEXT,
			Contents BLOB
		)`,
	}
	for _, stmt := range v1Schema {
		if _, err := dstDB.Exec(stmt); err != nil {
			return fmt.Errorf("creating v1 schema (%s...): %w", stmt[:30], err)
		}
	}

	// Copy _MetaData from v2 → v1.
	var productVersion, buildVersion, schemaHash string
	err = srcDB.QueryRow("SELECT _ProductVersion, _BuildVersion, _SchemaHash FROM _MetaData").
		Scan(&productVersion, &buildVersion, &schemaHash)
	if err != nil {
		return fmt.Errorf("reading _MetaData from source: %w", err)
	}
	if _, err := dstDB.Exec(
		"INSERT INTO _MetaData (_ProductVersion, _BuildVersion, _SchemaHash, _DisableAutoMprV2Upgrade) VALUES (?, ?, ?, 0)",
		productVersion, buildVersion, schemaHash,
	); err != nil {
		return fmt.Errorf("writing _MetaData: %w", err)
	}

	// Iterate over Unit rows and inline mxunit contents.
	rows, err := srcDB.Query(
		"SELECT UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts FROM Unit",
	)
	if err != nil {
		return fmt.Errorf("querying Unit table: %w", err)
	}
	defer rows.Close()

	tx, err := dstDB.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	ins, err := tx.Prepare(
		"INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents) VALUES (?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("preparing insert: %w", err)
	}
	defer ins.Close()

	count := 0
	for rows.Next() {
		var unitIDBlob, containerIDBlob []byte
		var containmentName string
		var treeConflict int64
		var contentsHash, contentsConflicts *string

		if err := rows.Scan(&unitIDBlob, &containerIDBlob, &containmentName, &treeConflict, &contentsHash, &contentsConflicts); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("scanning Unit row: %w", err)
		}

		unitUUID := mpr.BlobToUUID(unitIDBlob)
		mxunitPath := filepath.Join(contentsDir, unitUUID[0:2], unitUUID[2:4], unitUUID+".mxunit")
		contents, err := os.ReadFile(mxunitPath)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("reading mxunit file %s: %w", mxunitPath, err)
		}

		if _, err := ins.Exec(unitIDBlob, containerIDBlob, containmentName, treeConflict, contentsHash, contentsConflicts, contents); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("inserting unit %s: %w", unitUUID, err)
		}

		count++
		if count%100 == 0 {
			fmt.Printf("  converted %d units...\n", count)
		}
	}
	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("iterating Unit rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	fmt.Printf("Done. Converted %d units from v2 → v1: %s\n", count, outputMPR)
	return nil
}

// convertV1ToV2 reads a v1 MPR (single SQLite) and writes a v2 MPR (SQLite + mprcontents/).
func convertV1ToV2(inputMPR, outputMPR string) error {
	inputMPR, err := filepath.Abs(inputMPR)
	if err != nil {
		return fmt.Errorf("resolving input path: %w", err)
	}
	outputMPR, err = filepath.Abs(outputMPR)
	if err != nil {
		return fmt.Errorf("resolving output path: %w", err)
	}

	outputContentsDir := filepath.Join(filepath.Dir(outputMPR), "mprcontents")

	// Open input (read-only).
	srcDB, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", inputMPR))
	if err != nil {
		return fmt.Errorf("opening input MPR: %w", err)
	}
	defer srcDB.Close()

	// Remove existing output if present, then create fresh.
	_ = os.Remove(outputMPR)
	_ = os.RemoveAll(outputContentsDir)
	dstDB, err := sql.Open("sqlite", outputMPR)
	if err != nil {
		return fmt.Errorf("creating output MPR: %w", err)
	}
	defer dstDB.Close()

	// Create v2 schema.
	v2Schema := []string{
		`CREATE TABLE _MetaData (
			_FormatVersion INTEGER,
			_ProductVersion TEXT,
			_BuildVersion TEXT,
			_SchemaHash TEXT
		)`,
		`CREATE TABLE Unit (
			UnitID BLOB NOT NULL,
			ContainerID BLOB NOT NULL,
			ContainmentName TEXT NOT NULL,
			TreeConflict INTEGER NOT NULL DEFAULT 0,
			ContentsHash TEXT,
			ContentsConflicts TEXT
		)`,
		`CREATE TABLE _Transaction (
			LastTransactionID TEXT NOT NULL
		)`,
	}
	for _, stmt := range v2Schema {
		if _, err := dstDB.Exec(stmt); err != nil {
			return fmt.Errorf("creating v2 schema: %w", err)
		}
	}

	// Copy _MetaData from v1 → v2 (add _FormatVersion=2).
	var productVersion, buildVersion, schemaHash string
	err = srcDB.QueryRow("SELECT _ProductVersion, _BuildVersion, _SchemaHash FROM _MetaData").
		Scan(&productVersion, &buildVersion, &schemaHash)
	if err != nil {
		return fmt.Errorf("reading _MetaData from source: %w", err)
	}
	if _, err := dstDB.Exec(
		"INSERT INTO _MetaData (_FormatVersion, _ProductVersion, _BuildVersion, _SchemaHash) VALUES (2, ?, ?, ?)",
		productVersion, buildVersion, schemaHash,
	); err != nil {
		return fmt.Errorf("writing _MetaData: %w", err)
	}

	// Insert _Transaction row.
	txID := uuid.New().String()
	if _, err := dstDB.Exec("INSERT INTO _Transaction (LastTransactionID) VALUES (?)", txID); err != nil {
		return fmt.Errorf("writing _Transaction: %w", err)
	}

	// Iterate over Unit rows and write mxunit files.
	rows, err := srcDB.Query(
		"SELECT UnitID, ContainerID, ContainmentName, TreeConflict, ContentsConflicts, Contents FROM Unit",
	)
	if err != nil {
		return fmt.Errorf("querying Unit table: %w", err)
	}
	defer rows.Close()

	tx, err := dstDB.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	ins, err := tx.Prepare(
		"INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts) VALUES (?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return fmt.Errorf("preparing insert: %w", err)
	}
	defer ins.Close()

	count := 0
	for rows.Next() {
		var unitIDBlob, containerIDBlob []byte
		var containmentName string
		var treeConflict int64
		var contentsConflicts *string
		var contents []byte

		if err := rows.Scan(&unitIDBlob, &containerIDBlob, &containmentName, &treeConflict, &contentsConflicts, &contents); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("scanning Unit row: %w", err)
		}

		unitUUID := mpr.BlobToUUID(unitIDBlob)

		// Write mxunit file.
		dir := filepath.Join(outputContentsDir, unitUUID[0:2], unitUUID[2:4])
		if err := os.MkdirAll(dir, 0o755); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("creating mxunit directory %s: %w", dir, err)
		}
		mxunitPath := filepath.Join(dir, unitUUID+".mxunit")
		if err := os.WriteFile(mxunitPath, contents, 0o644); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("writing mxunit file %s: %w", mxunitPath, err)
		}

		// Compute SHA256 hash of contents.
		sum := sha256.Sum256(contents)
		contentsHash := base64.StdEncoding.EncodeToString(sum[:])

		if _, err := ins.Exec(unitIDBlob, containerIDBlob, containmentName, treeConflict, contentsHash, contentsConflicts); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("inserting unit %s: %w", unitUUID, err)
		}

		count++
		if count%100 == 0 {
			fmt.Printf("  converted %d units...\n", count)
		}
	}
	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("iterating Unit rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	fmt.Printf("Done. Converted %d units from v1 → v2: %s\n", count, outputMPR)
	return nil
}
