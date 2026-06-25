// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSingleConnection_SharedDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mprPath := filepath.Join(dir, "test.mpr")

	db, err := sql.Open("sqlite", mprPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE _MetaData (_FormatVersion INTEGER, _ProductVersion TEXT,
		                        _BuildVersion TEXT, _SchemaHash TEXT);
		INSERT INTO _MetaData VALUES (1, '10.18.0', '10.18.0.0', '');
		CREATE TABLE _Transaction (LastTransactionID TEXT);
		INSERT INTO _Transaction VALUES ('00000000-0000-0000-0000-000000000000');
		CREATE TABLE Unit (UnitID BLOB PRIMARY KEY NOT NULL, ContainerID BLOB,
		                   ContainmentName TEXT, TreeConflict LONG,
		                   ContentsHash TEXT, ContentsConflicts TEXT, Contents BLOB);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	db.Close()

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	legacyDB := b.reader.DB()
	msdkDB := b.msdkWriter.Reader().DB()
	msdkReaderDB := b.msdkReader.DB()

	if legacyDB != msdkDB {
		t.Errorf("db pointers differ: legacy=%p msdkWriter=%p — two connections open", legacyDB, msdkDB)
	}
	if msdkReaderDB != msdkDB {
		t.Errorf("msdkReader.DB() %p differs from msdkWriter.Reader().DB() %p", msdkReaderDB, msdkDB)
	}
}
