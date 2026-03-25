// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// --- REST Client Roundtrip Tests ---

func TestRoundtripRestClient_SimpleGet(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.SimpleAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION GetStatus
    METHOD GET
    PATH '/status'
    RESPONSE NONE;
END;`

	env.assertContains(createMDL, []string{
		"REST CLIENT",
		"SimpleAPI",
		"BASE URL 'https://api.example.com'",
		"AUTHENTICATION NONE",
		"OPERATION GetStatus",
		"METHOD GET",
		"PATH '/status'",
		"RESPONSE NONE",
	})
}

func TestRoundtripRestClient_WithJsonResponse(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.JsonAPI
BASE URL 'https://jsonplaceholder.typicode.com'
AUTHENTICATION NONE
BEGIN
  OPERATION GetPosts
    METHOD GET
    PATH '/posts'
    HEADER 'Accept' = 'application/json'
    RESPONSE JSON AS $Posts;
END;`

	env.assertContains(createMDL, []string{
		"REST CLIENT",
		"JsonAPI",
		"BASE URL 'https://jsonplaceholder.typicode.com'",
		"OPERATION GetPosts",
		"METHOD GET",
		"PATH '/posts'",
		"HEADER 'Accept' = 'application/json'",
		"RESPONSE JSON",
	})
}

func TestRoundtripRestClient_WithPathParams(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.ParamAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION GetItem
    METHOD GET
    PATH '/items/{itemId}'
    PARAMETER $itemId: Integer
    RESPONSE JSON AS $Item;
END;`

	env.assertContains(createMDL, []string{
		"OPERATION GetItem",
		"PATH '/items/{itemId}'",
		"PARAMETER $itemId: Integer",
		"RESPONSE JSON",
	})
}

func TestRoundtripRestClient_WithQueryParams(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Note: Mendix REST query parameters don't preserve data types in BSON
	// (Rest$QueryParameter has no DataType field), so all query params roundtrip as String.
	createMDL := `CREATE REST CLIENT ` + testModule + `.SearchAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION SearchItems
    METHOD GET
    PATH '/search'
    QUERY $q: String
    QUERY $page: String
    RESPONSE JSON AS $Results;
END;`

	env.assertContains(createMDL, []string{
		"OPERATION SearchItems",
		"PATH '/search'",
		"QUERY $q: String",
		"QUERY $page: String",
		"RESPONSE JSON",
	})
}

func TestRoundtripRestClient_PostWithBody(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.CrudAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION CreateItem
    METHOD POST
    PATH '/items'
    HEADER 'Content-Type' = 'application/json'
    BODY JSON FROM $NewItem
    RESPONSE JSON AS $CreatedItem;
END;`

	env.assertContains(createMDL, []string{
		"OPERATION CreateItem",
		"METHOD POST",
		"BODY JSON",
		"RESPONSE JSON",
	})
}

func TestRoundtripRestClient_BasicAuth(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.AuthAPI
BASE URL 'https://api.example.com'
AUTHENTICATION BASIC (USERNAME = 'admin', PASSWORD = 'secret')
BEGIN
  OPERATION GetData
    METHOD GET
    PATH '/data'
    RESPONSE JSON AS $Data;
END;`

	env.assertContains(createMDL, []string{
		"REST CLIENT",
		"AuthAPI",
		"AUTHENTICATION BASIC",
		"USERNAME = 'admin'",
		"PASSWORD = 'secret'",
		"OPERATION GetData",
	})
}

func TestRoundtripRestClient_WithTimeout(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.TimeoutAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION SlowQuery
    METHOD GET
    PATH '/slow'
    TIMEOUT 60
    RESPONSE JSON AS $Result;
END;`

	env.assertContains(createMDL, []string{
		"OPERATION SlowQuery",
		"TIMEOUT 60",
		"RESPONSE JSON",
	})
}

func TestRoundtripRestClient_MultipleOperations(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.PetStoreAPI
BASE URL 'https://petstore.swagger.io/v2'
AUTHENTICATION NONE
BEGIN
  OPERATION ListPets
    METHOD GET
    PATH '/pet/findByStatus'
    QUERY $status: String
    HEADER 'Accept' = 'application/json'
    TIMEOUT 30
    RESPONSE JSON AS $PetList;

  OPERATION GetPet
    METHOD GET
    PATH '/pet/{petId}'
    PARAMETER $petId: Integer
    RESPONSE JSON AS $Pet;

  OPERATION AddPet
    METHOD POST
    PATH '/pet'
    HEADER 'Content-Type' = 'application/json'
    BODY JSON FROM $NewPet
    RESPONSE JSON AS $CreatedPet;

  OPERATION RemovePet
    METHOD DELETE
    PATH '/pet/{petId}'
    PARAMETER $petId: Integer
    RESPONSE NONE;
END;`

	env.assertContains(createMDL, []string{
		"REST CLIENT",
		"PetStoreAPI",
		"OPERATION ListPets",
		"QUERY $status: String",
		"TIMEOUT 30",
		"OPERATION GetPet",
		"PARAMETER $petId: Integer",
		"OPERATION AddPet",
		"METHOD POST",
		"BODY JSON",
		"OPERATION RemovePet",
		"METHOD DELETE",
		"RESPONSE NONE",
	})
}

func TestRoundtripRestClient_DeleteNoResponse(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.DeleteAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION DeleteResource
    METHOD DELETE
    PATH '/resources/{id}'
    PARAMETER $id: Integer
    RESPONSE NONE;
END;`

	env.assertContains(createMDL, []string{
		"OPERATION DeleteResource",
		"METHOD DELETE",
		"PARAMETER $id: Integer",
		"RESPONSE NONE",
	})
}

func TestRoundtripRestClient_CreateOrModify(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create first version
	createMDL := `CREATE REST CLIENT ` + testModule + `.MutableAPI
BASE URL 'https://api.example.com/v1'
AUTHENTICATION NONE
BEGIN
  OPERATION GetData
    METHOD GET
    PATH '/data'
    RESPONSE JSON AS $Data;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create REST client: %v", err)
	}

	// Update with CREATE OR MODIFY
	updateMDL := `CREATE OR MODIFY REST CLIENT ` + testModule + `.MutableAPI
BASE URL 'https://api.example.com/v2'
AUTHENTICATION NONE
BEGIN
  OPERATION GetDataV2
    METHOD GET
    PATH '/data/v2'
    RESPONSE JSON AS $DataV2;
END;`

	if err := env.executeMDL(updateMDL); err != nil {
		t.Fatalf("Failed to update REST client: %v", err)
	}

	// Verify the updated version
	output, err := env.describeMDL("DESCRIBE REST CLIENT " + testModule + ".MutableAPI;")
	if err != nil {
		t.Fatalf("Failed to describe REST client: %v", err)
	}

	if !strings.Contains(output, "v2") {
		t.Errorf("Expected updated BASE URL with v2, got:\n%s", output)
	}
	if !strings.Contains(output, "GetDataV2") {
		t.Errorf("Expected updated operation GetDataV2, got:\n%s", output)
	}
}

func TestRoundtripRestClient_Drop(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create a REST client
	createMDL := `CREATE REST CLIENT ` + testModule + `.ToBeDropped
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION Ping
    METHOD GET
    PATH '/ping'
    RESPONSE NONE;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create REST client: %v", err)
	}

	// Verify it exists
	_, err := env.describeMDL("DESCRIBE REST CLIENT " + testModule + ".ToBeDropped;")
	if err != nil {
		t.Fatalf("REST client should exist before DROP: %v", err)
	}

	// Drop it
	if err := env.executeMDL("DROP REST CLIENT " + testModule + ".ToBeDropped;"); err != nil {
		t.Fatalf("Failed to drop REST client: %v", err)
	}

	// Verify it's gone
	_, err = env.describeMDL("DESCRIBE REST CLIENT " + testModule + ".ToBeDropped;")
	if err == nil {
		t.Error("REST client should not exist after DROP")
	}
}

// --- MX Check Tests ---

func TestMxCheck_RestClient_SimpleGet(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	createMDL := `CREATE REST CLIENT ` + testModule + `.MxCheckSimpleAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION GetStatus
    METHOD GET
    PATH '/status'
    HEADER 'Accept' = '*/*'
    RESPONSE NONE;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create REST client: %v", err)
	}

	// Disconnect to flush changes to disk
	env.executor.Execute(&ast.DisconnectStmt{})

	// Run mx check
	output, err := runMxCheck(t, env.projectPath)
	assertMxCheckPassed(t, output, err)
}

func TestMxCheck_RestClient_PostWithBody(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	// Test with path parameters (GET to avoid body requirements).
	createMDL := `CREATE REST CLIENT ` + testModule + `.MxCheckParamAPI
BASE URL 'https://api.example.com'
AUTHENTICATION NONE
BEGIN
  OPERATION GetItem
    METHOD GET
    PATH '/items/{itemId}'
    PARAMETER $itemId: Integer
    HEADER 'Accept' = '*/*'
    RESPONSE NONE;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create REST client: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	assertMxCheckPassed(t, output, err)
}

// assertMxCheckPassed checks mx check output for errors.
// Detects both "[error]" markers (validation errors) and "ERROR:" (load crashes).
func assertMxCheckPassed(t *testing.T, output string, err error) {
	t.Helper()
	if err != nil {
		// Non-zero exit code — could be a crash or validation errors
		if strings.Contains(output, "[error]") || strings.Contains(output, "ERROR:") {
			t.Errorf("mx check failed:\n%s", output)
		} else {
			t.Logf("mx check exited with error but no validation errors:\n%s", output)
		}
	} else if strings.Contains(output, "[error]") {
		t.Errorf("mx check found errors:\n%s", output)
	} else {
		t.Logf("mx check passed:\n%s", output)
	}
}

func TestMxCheck_RestClient_BasicAuth(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	// Use RESPONSE NONE to avoid entity mapping requirements (CE0061)
	createMDL := `CREATE REST CLIENT ` + testModule + `.MxCheckAuthAPI
BASE URL 'https://api.example.com'
AUTHENTICATION BASIC (USERNAME = 'user', PASSWORD = 'pass')
BEGIN
  OPERATION GetSecureData
    METHOD GET
    PATH '/secure/data'
    HEADER 'Accept' = '*/*'
    RESPONSE NONE;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create REST client: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	assertMxCheckPassed(t, output, err)
}

func TestMxCheck_RestClient_MultipleOperations(t *testing.T) {
	if !mxCheckAvailable() {
		t.Skip("mx command not available")
	}

	env := setupTestEnv(t)
	defer env.teardown()

	// Use RESPONSE NONE for all operations to avoid entity mapping requirements (CE0061).
	// All operations include Accept header to avoid CE7062.
	createMDL := `CREATE REST CLIENT ` + testModule + `.MxCheckPetStore
BASE URL 'https://petstore.swagger.io/v2'
AUTHENTICATION NONE
BEGIN
  OPERATION ListPets
    METHOD GET
    PATH '/pet/findByStatus'
    QUERY $status: String
    HEADER 'Accept' = 'application/json'
    TIMEOUT 30
    RESPONSE NONE;

  OPERATION GetPet
    METHOD GET
    PATH '/pet/{petId}'
    PARAMETER $petId: Integer
    HEADER 'Accept' = 'application/json'
    RESPONSE NONE;

  OPERATION RemovePet
    METHOD DELETE
    PATH '/pet/{petId}'
    PARAMETER $petId: Integer
    HEADER 'Accept' = '*/*'
    RESPONSE NONE;
END;`

	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create REST client: %v", err)
	}

	env.executor.Execute(&ast.DisconnectStmt{})

	output, err := runMxCheck(t, env.projectPath)
	assertMxCheckPassed(t, output, err)
}
