// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"go.mongodb.org/mongo-driver/bson"
)

// CreateDatabaseConnection creates a new DatabaseConnector$DatabaseConnection document.
func (w *Writer) CreateDatabaseConnection(conn *model.DatabaseConnection) error {
	if conn.ID == "" {
		conn.ID = model.ID(generateUUID())
	}
	conn.TypeName = "DatabaseConnector$DatabaseConnection"

	contents, err := w.serializeDatabaseConnection(conn)
	if err != nil {
		return fmt.Errorf("failed to serialize database connection: %w", err)
	}

	return w.insertUnit(string(conn.ID), string(conn.ContainerID),
		"Documents", "DatabaseConnector$DatabaseConnection", contents)
}

// UpdateDatabaseConnection updates an existing database connection.
func (w *Writer) UpdateDatabaseConnection(conn *model.DatabaseConnection) error {
	contents, err := w.serializeDatabaseConnection(conn)
	if err != nil {
		return fmt.Errorf("failed to serialize database connection: %w", err)
	}

	return w.updateUnit(string(conn.ID), contents)
}

// MoveDatabaseConnection moves a database connection to a new container (module or folder).
func (w *Writer) MoveDatabaseConnection(conn *model.DatabaseConnection) error {
	return w.moveUnitByID(string(conn.ID), string(conn.ContainerID))
}

// DeleteDatabaseConnection deletes a database connection by ID.
func (w *Writer) DeleteDatabaseConnection(id model.ID) error {
	return w.deleteUnit(string(id))
}

func (w *Writer) serializeDatabaseConnection(conn *model.DatabaseConnection) ([]byte, error) {
	// Build ConnectionInput — stores actual JDBC URL for Studio Pro development
	connInput := bson.M{
		"$Type": "DatabaseConnector$ConnectionString",
		"$ID":   idToBsonBinary(generateUUID()),
		"Value": conn.ConnectionInputValue,
	}

	doc := bson.M{
		"$ID":              idToBsonBinary(string(conn.ID)),
		"$Type":            "DatabaseConnector$DatabaseConnection",
		"Name":             conn.Name,
		"DatabaseType":     conn.DatabaseType,
		"ConnectionString": conn.ConnectionString,
		"UserName":         conn.UserName,
		"Password":         conn.Password,
		"Documentation":    conn.Documentation,
		"Excluded":         conn.Excluded,
		"ExportLevel":      "Hidden",
		"ConnectionInput":  connInput,
	}

	// Serialize Queries
	queries := bson.A{int32(2)} // versioned array prefix
	for _, q := range conn.Queries {
		queries = append(queries, serializeDBQuery(q))
	}
	doc["Queries"] = queries

	// AdditionalProperties (empty array)
	doc["AdditionalProperties"] = bson.A{int32(2)}

	// LastSelectedQuery (empty ref)
	doc["LastSelectedQuery"] = ""

	return bson.Marshal(doc)
}

func serializeDBQuery(q *model.DatabaseQuery) bson.M {
	qDoc := bson.M{
		"$Type":     "DatabaseConnector$DatabaseQuery",
		"Name":      q.Name,
		"Query":     q.SQL,
		"QueryType": int64(q.QueryType),
	}
	if q.ID != "" {
		qDoc["$ID"] = idToBsonBinary(string(q.ID))
	} else {
		qDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// TableMappings
	mappings := bson.A{int32(2)}
	for _, m := range q.TableMappings {
		mappings = append(mappings, serializeDBTableMapping(m))
	}
	qDoc["TableMappings"] = mappings

	// Parameters
	params := bson.A{int32(2)}
	for _, p := range q.Parameters {
		params = append(params, serializeDBQueryParameter(p))
	}
	qDoc["Parameters"] = params

	return qDoc
}

func serializeDBQueryParameter(p *model.DatabaseQueryParameter) bson.M {
	pDoc := bson.M{
		"$Type":                 "DatabaseConnector$QueryParameter",
		"ParameterName":         p.ParameterName,
		"DatabaseParameterName": "",
		"DefaultValue":          p.DefaultValue,
		"EmptyValueBecomesNull": p.EmptyValueBecomesNull,
		"Mode":                  "Unknown",
		"TableMapping":          nil,
	}
	if p.ID != "" {
		pDoc["$ID"] = idToBsonBinary(string(p.ID))
	} else {
		pDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// DataType
	dataType := p.DataType
	if dataType == "" {
		dataType = "DataTypes$StringType"
	}
	pDoc["DataType"] = bson.M{
		"$Type": dataType,
		"$ID":   idToBsonBinary(generateUUID()),
	}

	// SqlDataType
	pDoc["SqlDataType"] = bson.M{
		"$Type":        "DatabaseConnector$SimpleSqlDataType",
		"DataTypeName": "",
		"$ID":          idToBsonBinary(generateUUID()),
	}

	return pDoc
}

func serializeDBTableMapping(m *model.DatabaseTableMapping) bson.M {
	mDoc := bson.M{
		"$Type":     "DatabaseConnector$TableMapping",
		"Entity":    m.Entity,
		"TableName": m.TableName,
	}
	if m.ID != "" {
		mDoc["$ID"] = idToBsonBinary(string(m.ID))
	} else {
		mDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// Columns
	columns := bson.A{int32(2)}
	for _, c := range m.Columns {
		columns = append(columns, serializeDBColumnMapping(c))
	}
	mDoc["Columns"] = columns

	return mDoc
}

func serializeDBColumnMapping(c *model.DatabaseColumnMapping) bson.M {
	cDoc := bson.M{
		"$Type":      "DatabaseConnector$ColumnMapping",
		"Attribute":  c.Attribute,
		"ColumnName": c.ColumnName,
	}
	if c.ID != "" {
		cDoc["$ID"] = idToBsonBinary(string(c.ID))
	} else {
		cDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// SqlDataType — use SimpleSqlDataType as default
	cDoc["SqlDataType"] = bson.M{
		"$Type": "DatabaseConnector$SimpleSqlDataType",
		"$ID":   idToBsonBinary(generateUUID()),
	}

	return cDoc
}
