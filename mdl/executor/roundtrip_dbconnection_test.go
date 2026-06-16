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
create constant ` + testModule + `.TestDatabase_DBSource type String default 'jdbc:postgresql://localhost:5432/testdb';
create constant ` + testModule + `.TestDatabase_DBUsername type String default 'testuser';
create constant ` + testModule + `.TestDatabase_DBPassword type String default '';
`
	if err := env.executeMDL(constMDL); err != nil {
		t.Fatalf("Failed to create constants: %v", err)
	}

	// Create a non-persistent entity for the query to return
	entityMDL := `create or modify non-persistent entity ` + testModule + `.Employee (
		EmployeeId: Integer,
		Name: String(100),
		Email: String(200)
	);`
	if err := env.executeMDL(entityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create database connection with query
	createMDL := `create database connection ` + connName + `
type 'PostgreSQL'
connection string @` + testModule + `.TestDatabase_DBSource
username @` + testModule + `.TestDatabase_DBUsername
password @` + testModule + `.TestDatabase_DBPassword
{
  query GetAllEmployees
    sql 'select id, name, email from employees'
    returns ` + testModule + `.Employee
    map (
      id as EmployeeId,
      name as Name,
      email as Email
    );
}`

	env.assertContains(createMDL, []string{
		"database connection",
		"TestDatabase",
		"type 'PostgreSQL'",
		"connection string @" + testModule + ".TestDatabase_DBSource",
		"username @" + testModule + ".TestDatabase_DBUsername",
		"password @" + testModule + ".TestDatabase_DBPassword",
		"query GetAllEmployees",
		"returns " + testModule + ".Employee",
		"id as EmployeeId",
		"name as Name",
		"email as Email",
	})
}

func TestRoundtripDatabaseConnection_WithParameters(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	connName := testModule + ".ParamDB"

	// Create constants
	constMDL := `
create constant ` + testModule + `.ParamDB_DBSource type String default 'jdbc:sqlserver://localhost:1433;databaseName=F1';
create constant ` + testModule + `.ParamDB_DBUsername type String default 'sa';
create constant ` + testModule + `.ParamDB_DBPassword type String default '';
`
	if err := env.executeMDL(constMDL); err != nil {
		t.Fatalf("Failed to create constants: %v", err)
	}

	// Create non-persistent entity
	entityMDL := `create or modify non-persistent entity ` + testModule + `.Race (
		RaceId: Integer,
		RaceYear: Integer,
		Round: Integer,
		RaceName: String(200)
	);`
	if err := env.executeMDL(entityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create connection with parameterized query including DEFAULT and NULL
	createMDL := `create database connection ` + connName + `
type 'MSSQL'
connection string @` + testModule + `.ParamDB_DBSource
username @` + testModule + `.ParamDB_DBUsername
password @` + testModule + `.ParamDB_DBPassword
{
  query GetRacesBySeason
    sql 'select raceId, year, round, name from races where year between {startYear} and {endYear}'
    parameter startYear: Integer default '1900'
    parameter endYear: Integer null
    returns ` + testModule + `.Race
    map (
      raceId as RaceId,
      year as RaceYear,
      round as Round,
      name as RaceName
    );
}`

	env.assertContains(createMDL, []string{
		"database connection",
		"ParamDB",
		"query GetRacesBySeason",
		"parameter startYear: Integer default '1900'",
		"parameter endYear: Integer null",
		"returns " + testModule + ".Race",
		"raceId as RaceId",
	})
}

func TestRoundtripDatabaseConnection_NoQueries(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	connName := testModule + ".SimpleDB"

	// Create constants
	constMDL := `
create constant ` + testModule + `.SimpleDB_DBSource type String default 'jdbc:sqlserver://localhost:1433;databaseName=test';
create constant ` + testModule + `.SimpleDB_DBUsername type String default '';
create constant ` + testModule + `.SimpleDB_DBPassword type String default '';
`
	if err := env.executeMDL(constMDL); err != nil {
		t.Fatalf("Failed to create constants: %v", err)
	}

	// Create connection without queries
	createMDL := `create database connection ` + connName + `
type 'MSSQL'
connection string @` + testModule + `.SimpleDB_DBSource
username @` + testModule + `.SimpleDB_DBUsername
password @` + testModule + `.SimpleDB_DBPassword;`

	env.assertContains(createMDL, []string{
		"database connection",
		"SimpleDB",
		"type 'MSSQL'",
		"connection string @" + testModule + ".SimpleDB_DBSource",
		"username @" + testModule + ".SimpleDB_DBUsername",
		"password @" + testModule + ".SimpleDB_DBPassword",
	})
}
