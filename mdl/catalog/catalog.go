// SPDX-License-Identifier: Apache-2.0

// Package catalog provides SQL querying over Mendix project metadata.
package catalog

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Catalog provides SQL querying over Mendix project metadata.
type Catalog struct {
	db            CatalogDB
	projectID     string
	projectName   string
	mendixVersion string
	snapshots     map[string]*Snapshot
	activeSnap    string
	built         bool
}

// Snapshot represents a point-in-time view of project data.
type Snapshot struct {
	ID          string
	Name        string
	Date        time.Time
	Source      SnapshotSource
	SourceID    string
	Branch      string
	Revision    string
	ObjectCount int
	IsActive    bool
}

// SnapshotSource indicates where snapshot data came from.
type SnapshotSource string

const (
	SnapshotSourceLive    SnapshotSource = "LIVE"
	SnapshotSourceGit     SnapshotSource = "GIT"
	SnapshotSourceImport  SnapshotSource = "IMPORT"
	SnapshotSourceUnknown SnapshotSource = "UNKNOWN"
)

// QueryResult holds query results.
type QueryResult struct {
	Columns []string
	Rows    [][]any
	Count   int
}

// ProgressFunc is called during catalog building to report progress.
type ProgressFunc func(table string, count int)

// New creates a new catalog with an in-memory SQLite database.
func New() (*Catalog, error) {
	db, err := NewSqliteCatalogDB()
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}
	return NewFromDB(db)
}

// NewFromDB creates a catalog from an existing CatalogDB.
func NewFromDB(db CatalogDB) (*Catalog, error) {
	c := &Catalog{
		db:        db,
		projectID: "default",
		snapshots: make(map[string]*Snapshot),
	}

	// Create tables
	if err := c.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return c, nil
}

// SetProject sets the project metadata.
func (c *Catalog) SetProject(id, name, mendixVersion string) {
	c.projectID = id
	c.projectName = name
	c.mendixVersion = mendixVersion
}

// IsBuilt returns true if the catalog has been built.
func (c *Catalog) IsBuilt() bool {
	return c.built
}

// SetBuilt marks the catalog as built.
func (c *Catalog) SetBuilt(built bool) {
	c.built = built
}

// Tables returns the list of available catalog tables.
func (c *Catalog) Tables() []string {
	return []string{
		"CATALOG.MODULES",
		"CATALOG.ENTITIES",
		"CATALOG.ASSOCIATIONS",
		"CATALOG.ATTRIBUTES",
		"CATALOG.MICROFLOWS",
		"CATALOG.NANOFLOWS",
		"CATALOG.PAGES",
		"CATALOG.SNIPPETS",
		"CATALOG.LAYOUTS",
		"CATALOG.ENUMERATIONS",
		"CATALOG.JAVA_ACTIONS",
		"CATALOG.ACTIVITIES",
		"CATALOG.WIDGETS",
		"CATALOG.XPATH_EXPRESSIONS",
		"CATALOG.REFS",
		"CATALOG.ROLE_MAPPINGS",
		"CATALOG.PERMISSIONS",
		"CATALOG.PROJECTS",
		"CATALOG.SNAPSHOTS",
		"CATALOG.OBJECTS",
		"CATALOG.WORKFLOWS",
		"CATALOG.ODATA_CLIENTS",
		"CATALOG.ODATA_SERVICES",
		"CATALOG.BUSINESS_EVENT_SERVICES",
		"CATALOG.DATABASE_CONNECTIONS",
		"CATALOG.JAR_DEPENDENCIES",
		"CATALOG.CONSTANTS",
		"CATALOG.CONSTANT_VALUES",
		"CATALOG.REST_CLIENTS",
		"CATALOG.REST_OPERATIONS",
		"CATALOG.PUBLISHED_REST_SERVICES",
		"CATALOG.PUBLISHED_REST_OPERATIONS",
		"CATALOG.EXTERNAL_ENTITIES",
		"CATALOG.EXTERNAL_ACTIONS",
		"CATALOG.BUSINESS_EVENTS",
		"CATALOG.CONTRACT_ENTITIES",
		"CATALOG.CONTRACT_ACTIONS",
		"CATALOG.CONTRACT_MESSAGES",
		"CATALOG.JSON_STRUCTURES",
		"CATALOG.IMPORT_MAPPINGS",
		"CATALOG.EXPORT_MAPPINGS",
		"CATALOG.STRINGS",
		"CATALOG.SOURCE",
	}
}

// Query executes a SQL query and returns results.
func (c *Catalog) Query(sqlQuery string) (*QueryResult, error) {
	rows, err := c.db.Query(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    make([][]any, 0),
	}

	// Scan rows
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Convert []byte to string for readability
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		result.Rows = append(result.Rows, values)
		result.Count++
	}

	return result, rows.Err()
}

// CatalogDB returns the underlying database abstraction.
func (c *Catalog) CatalogDB() CatalogDB {
	return c.db
}

// ProjectID returns the current project ID.
func (c *Catalog) ProjectID() string {
	return c.projectID
}

// ProjectName returns the current project name.
func (c *Catalog) ProjectName() string {
	return c.projectName
}

// Close releases catalog resources.
func (c *Catalog) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// GetActiveSnapshot returns the active snapshot.
func (c *Catalog) GetActiveSnapshot() *Snapshot {
	if c.activeSnap == "" {
		return nil
	}
	return c.snapshots[c.activeSnap]
}

// CreateSnapshot creates a new snapshot with the given parameters.
func (c *Catalog) CreateSnapshot(name string, source SnapshotSource) *Snapshot {
	id := fmt.Sprintf("snapshot-%d", len(c.snapshots)+1)
	snap := &Snapshot{
		ID:       id,
		Name:     name,
		Date:     time.Now(),
		Source:   source,
		IsActive: true,
	}
	c.snapshots[id] = snap
	c.activeSnap = id
	return snap
}

// Metadata keys for cache validation
const (
	MetaMprPath       = "mpr_path"
	MetaMprModTime    = "mpr_mod_time"
	MetaMendixVersion = "mendix_version"
	MetaBuildMode     = "build_mode"
	MetaBuildTime     = "build_time"
	MetaBuildDuration = "build_duration"
)

// CacheInfo contains information about the cached catalog.
type CacheInfo struct {
	MprPath       string
	MprModTime    time.Time
	MendixVersion string
	BuildMode     string // "fast" or "full"
	BuildTime     time.Time
	BuildDuration time.Duration
	IsValid       bool
	InvalidReason string
}

// NewFromFile opens a catalog from a persisted SQLite file.
func NewFromFile(path string) (*Catalog, error) {
	db, err := NewSqliteCatalogDBFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open catalog file: %w", err)
	}

	c := &Catalog{
		db:        db,
		projectID: "default",
		snapshots: make(map[string]*Snapshot),
		built:     true, // Assume built if loading from file
	}

	return c, nil
}

// SaveToFile saves the catalog to a SQLite file.
// This copies the in-memory database to a file for persistence.
// Requires the underlying CatalogDB to be a *SqliteCatalogDB.
func (c *Catalog) SaveToFile(path string) error {
	sdb, ok := c.db.(*SqliteCatalogDB)
	if !ok {
		return fmt.Errorf("SaveToFile requires a SQLite-backed catalog")
	}
	rawDB := sdb.RawDB()

	// Use SQLite backup API via VACUUM INTO (SQLite 3.27+)
	// Fall back to manual copy if not available
	safePath := strings.ReplaceAll(path, "'", "''")
	_, err := rawDB.Exec(fmt.Sprintf("VACUUM INTO '%s'", safePath))
	if err != nil {
		// Fall back: export and import
		return c.saveToFileManual(path, rawDB)
	}

	return nil
}

// saveToFileManual saves by dumping and restoring (fallback method).
func (c *Catalog) saveToFileManual(path string, rawDB *sql.DB) error {
	// Open destination
	destDB, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer destDB.Close()

	// Get list of tables
	rows, err := rawDB.Query("SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return err
	}

	var tables []struct{ name, sql string }
	for rows.Next() {
		var t struct{ name, sql string }
		if err := rows.Scan(&t.name, &t.sql); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, t)
	}
	rows.Close()

	// Create tables and copy data
	for _, t := range tables {
		// Create table
		if _, err := destDB.Exec(t.sql); err != nil {
			return fmt.Errorf("failed to create table %s: %w", t.name, err)
		}

		// Copy data
		srcRows, err := rawDB.Query(fmt.Sprintf("SELECT * FROM %s", t.name))
		if err != nil {
			return err
		}

		cols, _ := srcRows.Columns()
		if len(cols) == 0 {
			srcRows.Close()
			continue
		}

		// Build insert statement
		placeholders := make([]string, len(cols))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		insertSQL := fmt.Sprintf("INSERT INTO %s VALUES (%s)", t.name,
			join(placeholders, ","))

		for srcRows.Next() {
			values := make([]any, len(cols))
			valuePtrs := make([]any, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := srcRows.Scan(valuePtrs...); err != nil {
				srcRows.Close()
				return err
			}
			if _, err := destDB.Exec(insertSQL, values...); err != nil {
				srcRows.Close()
				return err
			}
		}
		srcRows.Close()
	}

	// Copy views
	rows, err = rawDB.Query("SELECT sql FROM sqlite_master WHERE type='view'")
	if err != nil {
		return err
	}
	for rows.Next() {
		var viewSQL string
		if err := rows.Scan(&viewSQL); err != nil {
			rows.Close()
			return err
		}
		if viewSQL != "" {
			destDB.Exec(viewSQL) // Ignore errors for views
		}
	}
	rows.Close()

	// Copy indexes
	rows, err = rawDB.Query("SELECT sql FROM sqlite_master WHERE type='index' AND sql IS NOT NULL")
	if err != nil {
		return err
	}
	for rows.Next() {
		var indexSQL string
		if err := rows.Scan(&indexSQL); err != nil {
			rows.Close()
			return err
		}
		if indexSQL != "" {
			destDB.Exec(indexSQL) // Ignore errors for indexes
		}
	}
	rows.Close()

	return nil
}

// join is a simple string join helper.
func join(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString(s[0])
	for i := 1; i < len(s); i++ {
		result.WriteString(sep + s[i])
	}
	return result.String()
}

// SetMeta sets a metadata value in the catalog.
func (c *Catalog) SetMeta(key, value string) error {
	_, err := c.db.Exec(
		"INSERT OR REPLACE INTO catalog_meta (Key, Value) VALUES (?, ?)",
		key, value)
	return err
}

// GetMeta gets a metadata value from the catalog.
func (c *Catalog) GetMeta(key string) (string, error) {
	var value string
	err := c.db.QueryRow("SELECT Value FROM catalog_meta WHERE Key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetCacheInfo returns information about the cached catalog.
func (c *Catalog) GetCacheInfo() (*CacheInfo, error) {
	info := &CacheInfo{}

	info.MprPath, _ = c.GetMeta(MetaMprPath)
	info.MendixVersion, _ = c.GetMeta(MetaMendixVersion)
	info.BuildMode, _ = c.GetMeta(MetaBuildMode)

	if modTimeStr, _ := c.GetMeta(MetaMprModTime); modTimeStr != "" {
		info.MprModTime, _ = time.Parse(time.RFC3339, modTimeStr)
	}
	if buildTimeStr, _ := c.GetMeta(MetaBuildTime); buildTimeStr != "" {
		info.BuildTime, _ = time.Parse(time.RFC3339, buildTimeStr)
	}
	if durationStr, _ := c.GetMeta(MetaBuildDuration); durationStr != "" {
		info.BuildDuration, _ = time.ParseDuration(durationStr)
	}

	return info, nil
}

// SetCacheInfo stores cache metadata after building.
func (c *Catalog) SetCacheInfo(mprPath string, mprModTime time.Time, mendixVersion, buildMode string, buildDuration time.Duration) error {
	if err := c.SetMeta(MetaMprPath, mprPath); err != nil {
		return err
	}
	if err := c.SetMeta(MetaMprModTime, mprModTime.Format(time.RFC3339)); err != nil {
		return err
	}
	if err := c.SetMeta(MetaMendixVersion, mendixVersion); err != nil {
		return err
	}
	if err := c.SetMeta(MetaBuildMode, buildMode); err != nil {
		return err
	}
	if err := c.SetMeta(MetaBuildTime, time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	if err := c.SetMeta(MetaBuildDuration, buildDuration.String()); err != nil {
		return err
	}
	return nil
}
