// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// idToBsonBinary converts a UUID string to BSON Binary format.
// Mendix stores IDs as Binary with Subtype 0.
func idToBsonBinary(id string) bson.Binary {
	blob := uuidToBlob(id)
	if blob == nil || len(blob) != 16 {
		// Generate a new UUID if the provided one is invalid
		blob = uuidToBlob(generateUUID())
	}
	return bson.Binary{
		Subtype: 0x00,
		Data:    blob,
	}
}

// Writer provides methods to write Mendix project files.
type Writer struct {
	reader *Reader
	// sessionBuf, when non-nil, diverts updateUnit calls to a buffering
	// callback instead of going to disk. Used by import-style flows that
	// want to batch many unit updates into a single transaction.
	sessionBuf func(unitID string, contents []byte) error
}

// SetSessionBuf installs a callback that intercepts every updateUnit call.
// While set, updateUnit short-circuits to fn and skips all disk/SQLite work.
// Callers (typically batch-import flows) are responsible for flushing the
// buffered bytes back to disk before calling ClearSessionBuf.
//
// Pass nil to disable; ClearSessionBuf is the preferred way to do that.
func (w *Writer) SetSessionBuf(fn func(unitID string, contents []byte) error) {
	w.sessionBuf = fn
}

// ClearSessionBuf removes any installed sessionBuf callback so subsequent
// updateUnit calls resume normal disk-backed writes.
func (w *Writer) ClearSessionBuf() {
	w.sessionBuf = nil
}

// NewWriter creates a new writer from a reader opened in read-write mode.
func NewWriter(path string) (*Writer, error) {
	reader, err := OpenWithOptions(path, OpenOptions{ReadOnly: false})
	if err != nil {
		return nil, err
	}
	return &Writer{reader: reader}, nil
}

// NewWriterFromDB creates a Writer that reuses an existing *sql.DB connection.
// The caller owns the db lifecycle; this Writer's Close() does not close the db.
// contentsDir should be the mprcontents folder path (v2) or empty string (v1).
func NewWriterFromDB(db *sql.DB, path, contentsDir string) (*Writer, error) {
	reader, err := OpenWithDB(db, path, contentsDir)
	if err != nil {
		return nil, fmt.Errorf("open reader with shared db: %w", err)
	}
	return &Writer{reader: reader}, nil
}

// NewWriterWithReader creates a Writer that reuses an existing Reader instead of
// opening a second reader. This ensures cache invalidation (called by insertUnit
// via w.reader.InvalidateCache()) propagates to the same Reader object that callers
// hold for listing — so writes are immediately visible to reads on the same backend.
func NewWriterWithReader(r *Reader) *Writer {
	return &Writer{reader: r}
}

// Close closes the writer.
func (w *Writer) Close() error {
	return w.reader.Close()
}

// Reader returns the underlying reader as a UnitReader interface.
func (w *Writer) Reader() UnitReader {
	return w.reader
}

// ConcreteReader returns the underlying *Reader for callers that need
// concrete methods not exposed by UnitReader (e.g. codec.Store).
func (w *Writer) ConcreteReader() *Reader {
	return w.reader
}

// Transaction support

// Transaction represents a database transaction.
type Transaction struct {
	tx     *sql.Tx
	writer *Writer
}

// BeginTransaction starts a new transaction.
func (w *Writer) BeginTransaction() (*Transaction, error) {
	tx, err := w.reader.db.Begin()
	if err != nil {
		return nil, err
	}
	return &Transaction{tx: tx, writer: w}, nil
}

// Commit commits the transaction.
func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *Transaction) Rollback() error {
	return t.tx.Rollback()
}

// WriteTransaction provides atomic write operations for MPR v2 format.
// It coordinates database and file system changes to ensure consistency.
type WriteTransaction struct {
	tx           *sql.Tx
	writer       *Writer
	pendingFiles []pendingFile
	// pendingCache tracks unitID → contents for units written in this tx.
	// On Commit, these are pushed to the reader's contentCache.
	pendingCache []cacheEntry
	committed    bool
}

type cacheEntry struct {
	unitID   string
	contents []byte
}

type pendingFile struct {
	tempPath  string
	finalPath string
}

// BatchWriteOp describes a single insert or update for BatchWrite.
// When Insert is false, only UnitID and Contents are used.
type BatchWriteOp struct {
	Insert          bool
	UnitID          string
	ContainerID     string
	ContainmentName string
	UnitType        string
	Contents        []byte
}

// BeginWriteTransaction starts a new write transaction.
// For v2 format, this coordinates both database and file writes.
func (w *Writer) BeginWriteTransaction() (*WriteTransaction, error) {
	tx, err := w.reader.db.Begin()
	if err != nil {
		return nil, err
	}
	return &WriteTransaction{
		tx:           tx,
		writer:       w,
		pendingFiles: make([]pendingFile, 0),
	}, nil
}

// WriteUnit writes a unit within the transaction.
// The actual file write is deferred until Commit.
func (wt *WriteTransaction) WriteUnit(unitID string, contents []byte) error {
	unitIDBlob := uuidToBlob(unitID)

	if wt.writer.reader.version == MPRVersionV2 {
		swappedUUID := blobToUUIDSwapped(unitIDBlob)

		// Create directory if needed
		dir := filepath.Join(wt.writer.reader.contentsDir, swappedUUID[0:2], swappedUUID[2:4])
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Write to temp file first
		finalPath := filepath.Join(dir, swappedUUID+".mxunit")
		tempPath := finalPath + ".tmp"

		if err := os.WriteFile(tempPath, contents, 0644); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}

		wt.pendingFiles = append(wt.pendingFiles, pendingFile{
			tempPath:  tempPath,
			finalPath: finalPath,
		})
		wt.pendingCache = append(wt.pendingCache, cacheEntry{unitID: unitID, contents: contents})

		// Update hash in DB
		hash := sha256.Sum256(contents)
		contentsHash := base64.StdEncoding.EncodeToString(hash[:])
		_, err := wt.tx.Exec(`
			UPDATE Unit SET ContentsHash = ? WHERE UnitID = ?
		`, contentsHash, unitIDBlob)
		return err
	}

	// V1: Update in database directly
	_, err := wt.tx.Exec(`
		UPDATE Unit SET Contents = ? WHERE UnitID = ?
	`, contents, unitIDBlob)
	return err
}

// Commit commits the transaction.
// For v2, this first commits the database, then finalizes file writes.
// TODO: adopt two-phase approach (rename files first, then commit DB) to
// eliminate the partial-failure window where DB is committed but files are stale.
func (wt *WriteTransaction) Commit() error {
	if wt.committed {
		return fmt.Errorf("transaction already committed")
	}

	// Commit database transaction first
	if err := wt.tx.Commit(); err != nil {
		// Clean up temp files
		wt.cleanupTempFiles()
		return err
	}

	// Finalize file writes by renaming temp files to final paths
	for _, pf := range wt.pendingFiles {
		if err := os.Rename(pf.tempPath, pf.finalPath); err != nil {
			// Log error but continue - DB is already committed
			// This could leave some files in inconsistent state
			log.Printf("mpr.finalize_failed: path=%s error=%s", pf.finalPath, err.Error())
		}
	}

	wt.committed = true
	// Invalidate reader cache so next read sees the updated data
	wt.writer.reader.InvalidateCache()
	// Push committed contents to reader's content cache so subsequent
	// reads within the same session avoid disk I/O.
	for _, e := range wt.pendingCache {
		wt.writer.reader.contentCache[e.unitID] = e.contents
	}
	wt.pendingCache = nil
	return nil
}

// Rollback rolls back the transaction and cleans up temp files.
func (wt *WriteTransaction) Rollback() error {
	if wt.committed {
		return fmt.Errorf("transaction already committed")
	}

	// Clean up temp files
	wt.cleanupTempFiles()

	// Rollback database
	return wt.tx.Rollback()
}

func (wt *WriteTransaction) cleanupTempFiles() {
	for _, pf := range wt.pendingFiles {
		os.Remove(pf.tempPath)
	}
}

// generateUUID generates a new UUID v4 for model elements.
// Returns format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10

	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15])
}

// DeleteChildUnits recursively deletes all units whose ContainerID matches parentID.
func (w *Writer) DeleteChildUnits(parentID string) error {
	return w.deleteChildUnitsRecursive(parentID)
}

func (w *Writer) deleteChildUnitsRecursive(parentID string) error {
	parentBlob := uuidToBlob(parentID)
	if parentBlob == nil {
		return fmt.Errorf("invalid parent ID: %s", parentID)
	}

	rows, err := w.reader.db.Query("SELECT UnitID FROM Unit WHERE ContainerID = ? AND UnitID != ContainerID", parentBlob)
	if err != nil {
		return err
	}
	defer rows.Close()

	var childIDs []string
	for rows.Next() {
		var childBlob []byte
		if err := rows.Scan(&childBlob); err != nil {
			return err
		}
		childIDs = append(childIDs, blobToUUID(childBlob))
	}

	for _, childID := range childIDs {
		if err := w.deleteChildUnitsRecursive(childID); err != nil {
			return err
		}
		if err := w.deleteUnit(childID); err != nil {
			return err
		}
	}

	return nil
}

// uuidToBlob converts a UUID string to a 16-byte blob in Microsoft GUID format.
// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
// Microsoft GUID format byte-swaps the first 3 groups (little-endian):
// - First 4 bytes: reversed
// - Next 2 bytes: reversed
// - Next 2 bytes: reversed
// - Last 8 bytes: unchanged
func uuidToBlob(uuid string) []byte {
	if uuid == "" {
		return nil
	}
	// Remove dashes
	var clean strings.Builder
	for _, c := range uuid {
		if c != '-' {
			clean.WriteString(string(c))
		}
	}
	// Decode hex to bytes
	decoded, err := hex.DecodeString(clean.String())
	if err != nil || len(decoded) != 16 {
		return nil
	}
	// Swap bytes to Microsoft GUID format
	blob := make([]byte, 16)
	// First 4 bytes: reversed
	blob[0] = decoded[3]
	blob[1] = decoded[2]
	blob[2] = decoded[1]
	blob[3] = decoded[0]
	// Next 2 bytes: reversed
	blob[4] = decoded[5]
	blob[5] = decoded[4]
	// Next 2 bytes: reversed
	blob[6] = decoded[7]
	blob[7] = decoded[6]
	// Last 8 bytes: unchanged
	copy(blob[8:], decoded[8:])
	return blob
}

// ---------------------------------------------------------------------------
// Unit CRUD operations (merged from writer_units.go)
// ---------------------------------------------------------------------------

// updateTransactionID updates the _Transaction table with a new UUID.
// Studio Pro uses this to detect external changes during F4 sync.
// Only applies to MPR v2 projects (Mendix >= 10.18).
func (w *Writer) updateTransactionID() {
	if w.reader.version != MPRVersionV2 {
		return
	}
	newID := generateUUID()
	_, _ = w.reader.db.Exec(`UPDATE _Transaction SET LastTransactionID = ?`, newID)
}

// placeholderBinaryPrefix is the GUID-swapped byte pattern for placeholder IDs generated
// by sdk/widgets/augment.go placeholderID(). These are "aa000000000000000000000000XXXXXX"
// hex strings which, after hex decode + GUID byte-swap, produce 16-byte blobs whose first
// 13 bytes are \x00\x00\x00\xaa followed by 9 zero bytes.
var placeholderBinaryPrefix = []byte{0x00, 0x00, 0x00, 0xaa, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

// placeholderStringPrefix is the ASCII prefix of a placeholder ID that leaked as a string.
var placeholderStringBytes = []byte("aa000000000000000000000000")

// validateNoPlaceholderIDs scans raw BSON bytes for leaked placeholder IDs.
// Returns an error if any placeholder pattern is found.
func validateNoPlaceholderIDs(unitID string, contents []byte) error {
	if bytes.Contains(contents, placeholderBinaryPrefix) {
		return fmt.Errorf("placeholder ID leak detected in unit %s: binary aa000000-prefix ID found in BSON contents", unitID)
	}
	if bytes.Contains(contents, placeholderStringBytes) {
		return fmt.Errorf("placeholder ID leak detected in unit %s: string aa000000-prefix ID found in BSON contents", unitID)
	}
	return nil
}

func (w *Writer) insertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error {
	if err := validateNoPlaceholderIDs(unitID, contents); err != nil {
		return err
	}

	// Convert UUID strings to 16-byte blobs for database
	unitIDBlob := uuidToBlob(unitID)
	containerIDBlob := uuidToBlob(containerID)

	if unitIDBlob == nil {
		return fmt.Errorf("invalid unit ID (not a valid UUID): %q", unitID)
	}
	if containerIDBlob == nil {
		return fmt.Errorf("invalid container ID (not a valid UUID): %q", containerID)
	}

	if w.reader.version == MPRVersionV2 {
		// Get swapped UUID for file path
		swappedUUID := blobToUUIDSwapped(unitIDBlob)

		// Create directory structure: mprcontents/XX/YY/
		dir := filepath.Join(w.reader.contentsDir, swappedUUID[0:2], swappedUUID[2:4])
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write content file
		filePath := filepath.Join(dir, swappedUUID+".mxunit")
		if err := os.WriteFile(filePath, contents, 0644); err != nil {
			return fmt.Errorf("failed to write unit file: %w", err)
		}

		// Compute content hash (base64-encoded SHA256)
		hash := sha256.Sum256(contents)
		contentsHash := base64.StdEncoding.EncodeToString(hash[:])

		// Insert reference to database
		_, err := w.reader.db.Exec(`
			INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts)
			VALUES (?, ?, ?, 0, ?, '')
		`, unitIDBlob, containerIDBlob, containmentName, contentsHash)
		if err != nil {
			// Clean up the file we just wrote — otherwise it becomes an orphan
			os.Remove(filePath)
			return err
		}
		w.reader.InvalidateCache()
		// Push written content to reader's in-memory cache so subsequent
		// reads within the same session avoid disk I/O.
		w.reader.contentCache[unitID] = contents
		w.updateTransactionID()
		return nil
	}

	// MPR v1: Store directly in database
	// Try new schema first (without Type column - Mendix 11.6.2+)
	_, err := w.reader.db.Exec(`
		INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
		VALUES (?, ?, ?, 0, '', '', ?)
	`, unitIDBlob, containerIDBlob, containmentName, contents)
	if err != nil {
		// Try old schema with Type column
		_, err = w.reader.db.Exec(`
			INSERT INTO Unit (UnitID, ContainerID, ContainmentName, Type, Contents)
			VALUES (?, ?, ?, ?, ?)
		`, unitIDBlob, containerIDBlob, containmentName, unitType, contents)
	}
	if err == nil {
		w.reader.InvalidateCache()
	}
	return err
}

func (w *Writer) updateUnit(unitID string, contents []byte) error {
	if err := validateNoPlaceholderIDs(unitID, contents); err != nil {
		return err
	}

	// Session mode: divert to in-memory buffer and skip all disk/SQLite work.
	// The caller (e.g. ImportProject) is responsible for flushing later.
	if w.sessionBuf != nil {
		return w.sessionBuf(unitID, contents)
	}

	// Convert UUID string to 16-byte blob
	unitIDBlob := uuidToBlob(unitID)

	if w.reader.version == MPRVersionV2 {
		// Get swapped UUID for file path
		swappedUUID := blobToUUIDSwapped(unitIDBlob)

		// Build file path: mprcontents/XX/YY/UUID.mxunit
		filePath := filepath.Join(
			w.reader.contentsDir,
			swappedUUID[0:2],
			swappedUUID[2:4],
			swappedUUID+".mxunit",
		)

		// Write to a temp file then atomically rename into place.
		// Direct WriteFile would overwrite the shared inode when the target
		// is a hard link (e.g. in test fixtures), corrupting the source file.
		// Rename replaces the directory entry, leaving the linked inode intact.
		tmpPath := filePath + ".tmp"
		if err := os.WriteFile(tmpPath, contents, 0644); err != nil {
			return fmt.Errorf("failed to write unit temp file: %w", err)
		}
		if err := os.Rename(tmpPath, filePath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to rename unit file: %w", err)
		}

		// Update ContentsHash in database
		hash := sha256.Sum256(contents)
		contentsHash := base64.StdEncoding.EncodeToString(hash[:])
		_, err := w.reader.db.Exec(`
			UPDATE Unit SET ContentsHash = ? WHERE UnitID = ?
		`, contentsHash, unitIDBlob)
		if err == nil {
			w.reader.InvalidateCache()
			// Push updated content to reader's cache so subsequent reads
			// within the same session avoid disk I/O.
			w.reader.contentCache[unitID] = contents
			w.updateTransactionID()
		}
		return err
	}

	// MPR v1: Update in database
	_, err := w.reader.db.Exec(`
		UPDATE Unit SET Contents = ? WHERE UnitID = ?
	`, contents, unitIDBlob)
	return err
}

// UpdateRawUnit saves raw BSON bytes for a unit, bypassing deserialization.
// Used by ALTER PAGE to modify the BSON widget tree directly.
func (w *Writer) UpdateRawUnit(unitID string, contents []byte) error {
	return w.updateUnit(unitID, contents)
}

// InsertUnit creates a new unit in the project database.
// This is the exported version of insertUnit for use by TreeWriter and other packages.
func (w *Writer) InsertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error {
	return w.insertUnit(unitID, containerID, containmentName, unitType, contents)
}

// DeleteUnit removes a unit from the project database.
// This is the exported version of deleteUnit for use by TreeWriter and other packages.
func (w *Writer) DeleteUnit(unitID string) error {
	return w.deleteUnit(unitID)
}

func (w *Writer) deleteUnit(unitID string) error {
	// Convert UUID string to 16-byte blob
	unitIDBlob := uuidToBlob(unitID)
	if unitIDBlob == nil {
		return fmt.Errorf("invalid unit ID: %s", unitID)
	}

	if w.reader.version == MPRVersionV2 {
		// Get swapped UUID for file path
		swappedUUID := blobToUUIDSwapped(unitIDBlob)

		// Delete external file
		subDir1 := swappedUUID[0:2]
		subDir2 := swappedUUID[2:4]
		filePath := filepath.Join(w.reader.contentsDir, subDir1, subDir2, swappedUUID+".mxunit")
		os.Remove(filePath) // Ignore error if file doesn't exist

		// Clean up empty parent directories (YY/, then XX/)
		dir2 := filepath.Join(w.reader.contentsDir, subDir1, subDir2)
		os.Remove(dir2) // Only succeeds if empty
		dir1 := filepath.Join(w.reader.contentsDir, subDir1)
		os.Remove(dir1) // Only succeeds if empty
	}

	result, err := w.reader.db.Exec(`DELETE FROM Unit WHERE UnitID = ?`, unitIDBlob)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("unit not found in database: %s", unitID)
	}

	w.reader.InvalidateCache()
	// Remove deleted unit from content cache.
	delete(w.reader.contentCache, unitID)
	w.updateTransactionID()
	return nil
}

// BatchWrite executes all ops in a single atomic SQL transaction.
//
// For v2 MPR:
//   - Insert ops: content file written to final path BEFORE opening the SQL tx.
//   - Update ops: content written to a .tmp file BEFORE the tx; renamed to final AFTER Commit.
//
// For v1 MPR all content lives in SQLite; no file I/O is needed.
func (w *Writer) BatchWrite(ops []BatchWriteOp) error {
	type pendingRename struct{ tmp, final string }
	var renames []pendingRename

	// ── Phase 1: file writes (v2 only, before opening SQL tx) ──────────────
	if w.reader.version == MPRVersionV2 {
		for _, op := range ops {
			unitIDBlob := uuidToBlob(op.UnitID)
			if unitIDBlob == nil {
				return fmt.Errorf("BatchWrite: invalid unit ID %q", op.UnitID)
			}
			swappedUUID := blobToUUIDSwapped(unitIDBlob)
			dir := filepath.Join(w.reader.contentsDir, swappedUUID[0:2], swappedUUID[2:4])
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("BatchWrite: mkdir %s: %w", dir, err)
			}
			finalPath := filepath.Join(dir, swappedUUID+".mxunit")

			if op.Insert {
				if err := os.WriteFile(finalPath, op.Contents, 0644); err != nil {
					return fmt.Errorf("BatchWrite: write insert file: %w", err)
				}
			} else {
				tmpPath := finalPath + ".tmp"
				if err := os.WriteFile(tmpPath, op.Contents, 0644); err != nil {
					return fmt.Errorf("BatchWrite: write update tmp: %w", err)
				}
				renames = append(renames, pendingRename{tmp: tmpPath, final: finalPath})
			}
		}
	}

	// ── Phase 2: single SQL transaction ────────────────────────────────────
	sqlTx, err := w.reader.db.Begin()
	if err != nil {
		return fmt.Errorf("BatchWrite: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = sqlTx.Rollback()
			for _, r := range renames {
				_ = os.Remove(r.tmp)
			}
		}
	}()

	for _, op := range ops {
		unitIDBlob := uuidToBlob(op.UnitID)
		if unitIDBlob == nil {
			return fmt.Errorf("BatchWrite: invalid unit ID %q", op.UnitID)
		}

		if op.Insert {
			if w.reader.version == MPRVersionV2 {
				hash := sha256.Sum256(op.Contents)
				contentsHash := base64.StdEncoding.EncodeToString(hash[:])
				_, err = sqlTx.Exec(`
					INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts)
					VALUES (?, ?, ?, 0, ?, '')
				`, unitIDBlob, uuidToBlob(op.ContainerID), op.ContainmentName, contentsHash)
			} else {
				containerIDBlob := uuidToBlob(op.ContainerID)
				_, err = sqlTx.Exec(`
					INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
					VALUES (?, ?, ?, 0, '', '', ?)
				`, unitIDBlob, containerIDBlob, op.ContainmentName, op.Contents)
				if err != nil {
					_, err = sqlTx.Exec(`
						INSERT INTO Unit (UnitID, ContainerID, ContainmentName, Type, Contents)
						VALUES (?, ?, ?, ?, ?)
					`, unitIDBlob, containerIDBlob, op.ContainmentName, op.UnitType, op.Contents)
				}
			}
		} else {
			if w.reader.version == MPRVersionV2 {
				hash := sha256.Sum256(op.Contents)
				contentsHash := base64.StdEncoding.EncodeToString(hash[:])
				_, err = sqlTx.Exec(`UPDATE Unit SET ContentsHash = ? WHERE UnitID = ?`, contentsHash, unitIDBlob)
			} else {
				_, err = sqlTx.Exec(`UPDATE Unit SET Contents = ? WHERE UnitID = ?`, op.Contents, unitIDBlob)
			}
		}
		if err != nil {
			return fmt.Errorf("BatchWrite: sql op for unit %s: %w", op.UnitID, err)
		}
	}

	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("BatchWrite: commit: %w", err)
	}
	committed = true

	// ── Phase 3: rename .tmp → final (v2 updates) ──────────────────────────
	for _, r := range renames {
		if err := os.Rename(r.tmp, r.final); err != nil {
			log.Printf("mpr.batchwrite_rename_failed: tmp=%s final=%s err=%s", r.tmp, r.final, err)
		}
	}

	w.updateTransactionID()
	w.reader.InvalidateCache()
	// Push all written contents to reader's cache.
	for _, op := range ops {
		w.reader.contentCache[op.UnitID] = op.Contents
	}
	return nil
}

// UpdateUnitContainer changes the ContainerID of a unit, moving it to a new parent.
func (w *Writer) UpdateUnitContainer(unitID, newContainerID string) error {
	unitIDBlob := uuidToBlob(unitID)
	if unitIDBlob == nil {
		return fmt.Errorf("invalid unit ID: %s", unitID)
	}
	containerIDBlob := uuidToBlob(newContainerID)
	if containerIDBlob == nil {
		return fmt.Errorf("invalid container ID: %s", newContainerID)
	}

	result, err := w.reader.db.Exec(`UPDATE Unit SET ContainerID = ? WHERE UnitID = ?`, containerIDBlob, unitIDBlob)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("unit not found: %s", unitID)
	}

	w.reader.InvalidateCache()
	w.updateTransactionID()
	return nil
}
