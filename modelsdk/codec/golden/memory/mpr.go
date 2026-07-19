// Package memory provides an in-memory MPR fixture for integration tests.
package memory

import (
	"database/sql"
	"fmt"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"go.mongodb.org/mongo-driver/v2/bson"
	_ "modernc.org/sqlite"
)

type MPR struct {
	DB     *sql.DB
	Reader *mmpr.Reader
	Writer *mmpr.Writer
	Path   string
}

func New(projectVersion string) (*MPR, error) {
	return openDB(":memory:", projectVersion)
}

func NewFile(path, projectVersion string) (*MPR, error) {
	return openDB(path, projectVersion)
}

func openDB(dsn, projectVersion string) (*MPR, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	if err := insertMeta(db, projectVersion); err != nil {
		db.Close()
		return nil, fmt.Errorf("insert metadata: %w", err)
	}
	reader, err := mmpr.OpenWithDB(db, dsn, "")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open reader: %w", err)
	}
	writer := mmpr.NewWriterWithReader(reader)
	if err := seedProjectRoot(writer); err != nil {
		db.Close()
		return nil, fmt.Errorf("seed project root: %w", err)
	}
	return &MPR{DB: db, Reader: reader, Writer: writer, Path: dsn}, nil
}

func (m *MPR) Close() error { return m.DB.Close() }

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS _MetaData (
			_FormatVersion INTEGER,
			_ProductVersion TEXT,
			_BuildVersion TEXT,
			_SchemaHash TEXT
		);
		CREATE TABLE IF NOT EXISTS Unit (
			UnitID          BLOB    PRIMARY KEY NOT NULL,
			ContainerID     BLOB,
			ContainmentName TEXT,
			TreeConflict    INTEGER,
			ContentsHash    TEXT,
			ContentsConflicts TEXT,
			Contents        BLOB
		)
	`)
	return err
}

func insertMeta(db *sql.DB, projectVersion string) error {
	_, err := db.Exec(`
		INSERT INTO _MetaData (_FormatVersion, _ProductVersion, _BuildVersion, _SchemaHash)
		VALUES (1, ?, ?, ?)
	`, projectVersion, projectVersion, "test-schema-hash")
	return err
}

func seedProjectRoot(writer *mmpr.Writer) error {
	doc := bson.D{
		{Key: "$Type", Value: "Projects$Project"},
		{Key: "Name", Value: "TestProject"},
	}
	data, err := bson.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal bson: %w", err)
	}
	unitID := mmpr.GenerateID()
	return writer.InsertUnit(unitID, unitID, "", "Projects$Project", data)
}
