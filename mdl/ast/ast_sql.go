// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// SQL Statements (external database connectivity)
// ============================================================================

// SQLConnectStmt represents: SQL CONNECT <driver> '<dsn>' AS <alias>
type SQLConnectStmt struct {
	Driver string
	DSN    string
	Alias  string
}

func (s *SQLConnectStmt) isStatement() {}
func (s *SQLConnectStmt) TypeName() string { return "SQLConnect" }

// SQLDisconnectStmt represents: SQL DISCONNECT <alias>
type SQLDisconnectStmt struct {
	Alias string
}

func (s *SQLDisconnectStmt) isStatement() {}
func (s *SQLDisconnectStmt) TypeName() string { return "SQLDisconnect" }

// SQLConnectionsStmt represents: SQL CONNECTIONS
type SQLConnectionsStmt struct{}

func (s *SQLConnectionsStmt) isStatement() {}
func (s *SQLConnectionsStmt) TypeName() string { return "SQLConnections" }

// SQLQueryStmt represents: SQL <alias> <raw-sql>
type SQLQueryStmt struct {
	Alias string
	Query string
}

func (s *SQLQueryStmt) isStatement() {}
func (s *SQLQueryStmt) TypeName() string { return "SQLQuery" }

// SQLShowTablesStmt represents: SQL <alias> SHOW TABLES
type SQLShowTablesStmt struct {
	Alias string
}

func (s *SQLShowTablesStmt) isStatement() {}
func (s *SQLShowTablesStmt) TypeName() string { return "SQLShowTables" }

// SQLDescribeTableStmt represents: SQL <alias> DESCRIBE <table>
type SQLDescribeTableStmt struct {
	Alias string
	Table string
}

func (s *SQLDescribeTableStmt) isStatement() {}
func (s *SQLDescribeTableStmt) TypeName() string { return "SQLDescribeTable" }

// SQLShowViewsStmt represents: SQL <alias> SHOW VIEWS
type SQLShowViewsStmt struct {
	Alias string
}

func (s *SQLShowViewsStmt) isStatement() {}
func (s *SQLShowViewsStmt) TypeName() string { return "SQLShowViews" }

// SQLShowFunctionsStmt represents: SQL <alias> SHOW FUNCTIONS
type SQLShowFunctionsStmt struct {
	Alias string
}

func (s *SQLShowFunctionsStmt) isStatement() {}
func (s *SQLShowFunctionsStmt) TypeName() string { return "SQLShowFunctions" }

// SQLGenerateConnectorStmt represents: SQL <alias> GENERATE CONNECTOR INTO <module> [TABLES (...)] [VIEWS (...)] [EXEC]
type SQLGenerateConnectorStmt struct {
	Alias  string
	Module string
	Tables []string // nil = all tables
	Views  []string // nil = no views
	Exec   bool     // execute generated MDL immediately
}

func (s *SQLGenerateConnectorStmt) isStatement() {}
func (s *SQLGenerateConnectorStmt) TypeName() string { return "SQLGenerateConnector" }

// ============================================================================
// Database Connection Statements
// ============================================================================

// DatabaseQueryDef defines a query within a CREATE DATABASE CONNECTION statement.
type DatabaseQueryDef struct {
	Name       string
	SQL        string
	Returns    QualifiedName
	Mappings   []DatabaseQueryMappingDef
	Parameters []DatabaseQueryParamDef
}

// DatabaseQueryMappingDef defines a column→attribute mapping in a query.
type DatabaseQueryMappingDef struct {
	ColumnName    string
	AttributeName string
}

// DatabaseQueryParamDef defines a parameter for a query.
type DatabaseQueryParamDef struct {
	Name         string
	DataType     DataType
	DefaultValue string // test value for Studio Pro (empty string if not set)
	TestWithNull bool   // true = EmptyValueBecomesNull in BSON
}

// CreateDatabaseConnectionStmt represents: CREATE DATABASE CONNECTION Module.Name ...
type CreateDatabaseConnectionStmt struct {
	Name                  QualifiedName
	DatabaseType          string // "PostgreSQL", "MSSQL", "Oracle"
	ConnectionString      string // constant ref or string literal for connection string
	ConnectionStringIsRef bool   // true if @Module.Const reference
	UserName              string
	UserNameIsRef         bool
	Password              string
	PasswordIsRef         bool
	Host                  string
	Port                  int
	Database              string
	Queries               []DatabaseQueryDef
	CreateOrModify        bool
}

func (s *CreateDatabaseConnectionStmt) isStatement() {}
func (s *CreateDatabaseConnectionStmt) TypeName() string { return "CreateDatabaseConnection" }

// ImportMapping maps a source column to a target attribute.
type ImportMapping struct {
	SourceColumn string
	TargetAttr   string
}

// LinkMapping maps a source column to an association on the target entity.
// If LookupAttr is empty, the source value is treated as a raw Mendix object ID.
// If LookupAttr is set, the source value is looked up against that attribute on the child entity.
type LinkMapping struct {
	SourceColumn    string
	AssociationName string // unqualified association name
	LookupAttr      string // attribute on child entity for lookup (empty = direct ID)
}

// ImportStmt represents: IMPORT FROM <alias> QUERY '<sql>' INTO Module.Entity MAP (...) [LINK (...)] [BATCH n] [LIMIT n]
type ImportStmt struct {
	SourceAlias  string
	Query        string
	TargetEntity string // "Module.Entity"
	Mappings     []ImportMapping
	Links        []LinkMapping
	BatchSize    int // 0 = default (1000)
	Limit        int // 0 = no limit
}

func (s *ImportStmt) isStatement() {}
func (s *ImportStmt) TypeName() string { return "Import" }
