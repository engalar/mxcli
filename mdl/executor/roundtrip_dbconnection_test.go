// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"
)

func TestRoundtripDatabaseConnection_Simple(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	connName := testModule + ".TestDatabase"

	// First create the constants that the connection references
	constMDL := `
CREATE CONSTANT ` + testModule + `.TestDatabase_DBSource TYPE String DEFAULT 'jdbc:postgresql://localhost:5432/testdb';
CREATE CONSTANT ` + testModule + `.TestDatabase_DBUsername TYPE String DEFAULT 'testuser';
CREATE CONSTANT ` + testModule + `.TestDatabase_DBPassword TYPE String DEFAULT '';
`
	if err := env.executeMDL(constMDL); err != nil {
		t.Fatalf("Failed to create constants: %v", err)
	}

	// Create a non-persistent entity for the query to return
	entityMDL := `CREATE OR MODIFY NON-PERSISTENT ENTITY ` + testModule + `.Employee (
		EmployeeId: Integer,
		Name: String(100),
		Email: String(200)
	);`
	if err := env.executeMDL(entityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create database connection with query
	createMDL := `CREATE DATABASE CONNECTION ` + connName + `
TYPE 'PostgreSQL'
CONNECTION STRING @` + testModule + `.TestDatabase_DBSource
USERNAME @` + testModule + `.TestDatabase_DBUsername
PASSWORD @` + testModule + `.TestDatabase_DBPassword
BEGIN
  QUERY GetAllEmployees
    SQL 'SELECT id, name, email FROM employees'
    RETURNS ` + testModule + `.Employee
    MAP (
      id AS EmployeeId,
      name AS Name,
      email AS Email
    );
END;`

	env.assertContains(createMDL, []string{
		"DATABASE CONNECTION",
		"TestDatabase",
		"TYPE 'PostgreSQL'",
		"CONNECTION STRING @" + testModule + ".TestDatabase_DBSource",
		"USERNAME @" + testModule + ".TestDatabase_DBUsername",
		"PASSWORD @" + testModule + ".TestDatabase_DBPassword",
		"QUERY GetAllEmployees",
		"RETURNS " + testModule + ".Employee",
		"id AS EmployeeId",
		"name AS Name",
		"email AS Email",
	})
}

func TestRoundtripDatabaseConnection_WithParameters(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	connName := testModule + ".ParamDB"

	// Create constants
	constMDL := `
CREATE CONSTANT ` + testModule + `.ParamDB_DBSource TYPE String DEFAULT 'jdbc:sqlserver://localhost:1433;databaseName=F1';
CREATE CONSTANT ` + testModule + `.ParamDB_DBUsername TYPE String DEFAULT 'sa';
CREATE CONSTANT ` + testModule + `.ParamDB_DBPassword TYPE String DEFAULT '';
`
	if err := env.executeMDL(constMDL); err != nil {
		t.Fatalf("Failed to create constants: %v", err)
	}

	// Create non-persistent entity
	entityMDL := `CREATE OR MODIFY NON-PERSISTENT ENTITY ` + testModule + `.Race (
		RaceId: Integer,
		RaceYear: Integer,
		Round: Integer,
		RaceName: String(200)
	);`
	if err := env.executeMDL(entityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create connection with parameterized query including DEFAULT and NULL
	createMDL := `CREATE DATABASE CONNECTION ` + connName + `
TYPE 'MSSQL'
CONNECTION STRING @` + testModule + `.ParamDB_DBSource
USERNAME @` + testModule + `.ParamDB_DBUsername
PASSWORD @` + testModule + `.ParamDB_DBPassword
BEGIN
  QUERY GetRacesBySeason
    SQL 'SELECT raceId, year, round, name FROM races WHERE year BETWEEN {startYear} AND {endYear}'
    PARAMETER startYear: Integer DEFAULT '1900'
    PARAMETER endYear: Integer NULL
    RETURNS ` + testModule + `.Race
    MAP (
      raceId AS RaceId,
      year AS RaceYear,
      round AS Round,
      name AS RaceName
    );
END;`

	env.assertContains(createMDL, []string{
		"DATABASE CONNECTION",
		"ParamDB",
		"QUERY GetRacesBySeason",
		"PARAMETER startYear: Integer DEFAULT '1900'",
		"PARAMETER endYear: Integer NULL",
		"RETURNS " + testModule + ".Race",
		"raceId AS RaceId",
	})
}

func TestRoundtripDatabaseConnection_NoQueries(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	connName := testModule + ".SimpleDB"

	// Create constants
	constMDL := `
CREATE CONSTANT ` + testModule + `.SimpleDB_DBSource TYPE String DEFAULT 'jdbc:sqlserver://localhost:1433;databaseName=test';
CREATE CONSTANT ` + testModule + `.SimpleDB_DBUsername TYPE String DEFAULT '';
CREATE CONSTANT ` + testModule + `.SimpleDB_DBPassword TYPE String DEFAULT '';
`
	if err := env.executeMDL(constMDL); err != nil {
		t.Fatalf("Failed to create constants: %v", err)
	}

	// Create connection without queries
	createMDL := `CREATE DATABASE CONNECTION ` + connName + `
TYPE 'MSSQL'
CONNECTION STRING @` + testModule + `.SimpleDB_DBSource
USERNAME @` + testModule + `.SimpleDB_DBUsername
PASSWORD @` + testModule + `.SimpleDB_DBPassword;`

	env.assertContains(createMDL, []string{
		"DATABASE CONNECTION",
		"SimpleDB",
		"TYPE 'MSSQL'",
		"CONNECTION STRING @" + testModule + ".SimpleDB_DBSource",
		"USERNAME @" + testModule + ".SimpleDB_DBUsername",
		"PASSWORD @" + testModule + ".SimpleDB_DBPassword",
	})
}
