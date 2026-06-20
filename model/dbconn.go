// SPDX-License-Identifier: Apache-2.0

package model

type DatabaseConnection struct {
	BaseElement
	ContainerID          ID               `json:"containerId"`
	Name                 string           `json:"name"`
	DatabaseType         string           `json:"databaseType"`         // "PostgreSQL", "MSSQL", "Oracle"
	ConnectionString     string           `json:"connectionString"`     // BY_NAME ref to constant: "Module.ConstantName"
	ConnectionInputValue string           `json:"connectionInputValue"` // Actual JDBC URL for Studio Pro dev
	UserName             string           `json:"userName"`             // BY_NAME ref to constant
	Password             string           `json:"password"`             // BY_NAME ref to constant
	Documentation        string           `json:"documentation,omitempty"`
	Excluded             bool             `json:"excluded,omitempty"`
	ExportLevel          string           `json:"exportLevel,omitempty"`
	Queries              []*DatabaseQuery `json:"queries,omitempty"`
}

type DatabaseQuery struct {
	BaseElement
	Name          string                    `json:"name"`
	QueryType     int                       `json:"queryType"`     // 1 = custom SQL
	SQL           string                    `json:"sql,omitempty"` // extracted from TableMappings
	TableMappings []*DatabaseTableMapping   `json:"tableMappings,omitempty"`
	Parameters    []*DatabaseQueryParameter `json:"parameters,omitempty"`
}

type DatabaseQueryParameter struct {
	BaseElement
	ParameterName         string `json:"parameterName"`
	DataType              string `json:"dataType"`              // e.g. "DataTypes$IntegerType", "DataTypes$StringType"
	DefaultValue          string `json:"defaultValue"`          // test value for Studio Pro
	EmptyValueBecomesNull bool   `json:"emptyValueBecomesNull"` // true = test with NULL
}

type DatabaseTableMapping struct {
	BaseElement
	Entity    string                   `json:"entity"`    // BY_NAME entity ref: "Module.Entity"
	TableName string                   `json:"tableName"` // SQL table name
	Columns   []*DatabaseColumnMapping `json:"columns,omitempty"`
}

type DatabaseColumnMapping struct {
	BaseElement
	Attribute   string `json:"attribute"`   // BY_NAME attribute ref: "Module.Entity.Attr"
	ColumnName  string `json:"columnName"`  // SQL column name
	SqlDataType string `json:"sqlDataType"` // simplified type name for display
}

