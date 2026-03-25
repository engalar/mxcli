# Proposal: Consumed REST Services (SHOW / DESCRIBE / CREATE)

## Overview

**Document type:** `Rest$ConsumedRestService`
**Prevalence:** 2 in Evora project (not found in Enquiries or Lato)
**Priority:** Medium — newer Mendix feature (10.1.0+), growing in adoption
**Reference:** `mdl-examples/doctype-tests/06-rest-client-examples.mdl` (21 examples)

Consumed REST Services define external REST API connections. Each service has a base URL, authentication scheme, and one or more operations with HTTP methods, paths, headers, parameters, and response handling.

This is different from "Consumed OData Services" (which already have full SHOW/DESCRIBE/CREATE support). Consumed REST Services are a Mendix 10+ feature for calling arbitrary REST APIs.

### Version Requirements

| Feature | Minimum Version |
|---------|----------------|
| ConsumedRestService | Mendix 10.1.0 |
| Method property | Mendix 10.4.0 |
| Query parameters | Mendix 11.0.0 |

## What Already Exists

| Layer | Status | Location |
|-------|--------|----------|
| **Generated metamodel** | Yes | `generated/metamodel/types.go` — full type hierarchy |
| **Grammar rules** | Partial | `MDLParser.g4:1997-2048` — basic structure, needs revision |
| **Go model types** | No | — |
| **MPR parser** | No | — |
| **MPR reader** | No | — |
| **MPR writer** | No | — |
| **AST nodes** | No | — |
| **Visitor** | No | — |
| **Executor** | No | — |

## BSON Type Hierarchy

From `generated/metamodel/types.go` and reflection data:

```
Rest$ConsumedRestService
  Name: string
  Documentation: string
  Excluded: bool
  ExportLevel: RestExportLevel ("API" | "Hidden")
  BaseUrl: Rest$ValueTemplate { Value: string }
  BaseUrlParameter: Rest$RestParameter? { Name, DataType, TestValue }
  AuthenticationScheme: polymorphic? (see below)
  OpenApiFile: Rest$OpenApiFile? { Content: string }
  Operations: []*Rest$RestOperation

Rest$RestOperation
  Name: string
  Method: polymorphic (see below)
  Path: Rest$ValueTemplate { Value: string }
  Headers: []*Rest$HeaderWithValueTemplate { Name, Value: Rest$ValueTemplate }
  QueryParameters: []*Rest$QueryParameter { Name, ParameterUsage, TestValue }
  Parameters: []*Rest$RestOperationParameter { Name, DataType, TestValue }
  ResponseHandling: polymorphic (see below)
  Tags: []string
  Timeout: int (seconds, 0 = default 300s)
```

### Polymorphic Types

**Method** (`Rest$RestOperationMethod`):
- `Rest$RestOperationMethodWithBody` — POST, PUT, PATCH (has `Body` field)
  - `Body`: `Rest$ImplicitMappingBody` | `Rest$JsonBody` | `Rest$StringBody`
- `Rest$RestOperationMethodWithoutBody` — GET, DELETE, HEAD, OPTIONS

**Authentication** (`Rest$AuthenticationScheme`):
- `Rest$BasicAuthenticationScheme` — `Username: Rest$Value`, `Password: Rest$Value`
  - Values are `Rest$ConstantValue` (references a constant) or `Rest$StringValue` (literal)
- `null` — no authentication

**Response Handling** (`Rest$RestOperationResponseHandling`):
- `Rest$ImplicitMappingResponseHandling` — `ContentType`, `RootMappingElement`, `StatusCode`
- `Rest$NoResponseHandling` — `ContentType`, `StatusCode`

**Body** (`Rest$Body`):
- `Rest$ImplicitMappingBody` — export mapping with `RootMappingElement`
- `Rest$JsonBody` — raw JSON string in `Value`
- `Rest$StringBody` — template string in `ValueTemplate`

**Query Parameter Usage** (`Rest$QueryParameterUsage`):
- `Rest$RequiredQueryParameterUsage`
- `Rest$OptionalQueryParameterUsage` — has `Included` bool flag

## Proposed MDL Syntax

### Design Principles

1. **Roundtrip**: DESCRIBE output must be valid CREATE input
2. **Consistency**: Follow existing CREATE REST CLIENT grammar structure (BEGIN...END blocks)
3. **Alignment**: Match the syntax in `06-rest-client-examples.mdl`
4. **Simplicity**: Use MDL-native types ($variables, data types) rather than exposing BSON internals

### SHOW REST CLIENTS

```sql
SHOW REST CLIENTS [IN Module]
```

| Module | Name | Base URL | Auth | Operations |
|--------|------|----------|------|------------|
| RestTest | RC001_SimpleAPI | https://reqbin.com | NONE | 1 |
| RestTest | RC018_PetStoreAPI | https://petstore.swagger.io/v2 | NONE | 6 |

### DESCRIBE REST CLIENT

```sql
DESCRIBE REST CLIENT Module.Name
```

Outputs a valid `CREATE REST CLIENT` statement:

```sql
/**
 * Swagger Pet Store API
 * A complete REST client for the classic Pet Store demo API.
 */
CREATE REST CLIENT RestTest.PetStoreAPI
BASE URL 'https://petstore.swagger.io/v2'
AUTHENTICATION NONE
BEGIN
  /** List all pets with optional filtering */
  OPERATION ListPets
    METHOD GET
    PATH '/pet/findByStatus'
    QUERY $status: String
    HEADER 'Accept' = 'application/json'
    TIMEOUT 30
    RESPONSE JSON AS $PetList;

  /** Get a single pet by ID */
  OPERATION GetPet
    METHOD GET
    PATH '/pet/{petId}'
    PARAMETER $petId: Integer
    HEADER 'Accept' = 'application/json'
    RESPONSE JSON AS $Pet;

  /** Create a new pet */
  OPERATION AddPet
    METHOD POST
    PATH '/pet'
    HEADER 'Content-Type' = 'application/json'
    HEADER 'Accept' = 'application/json'
    BODY JSON FROM $NewPet
    RESPONSE JSON AS $CreatedPet;

  /** Delete a pet */
  OPERATION RemovePet
    METHOD DELETE
    PATH '/pet/{petId}'
    PARAMETER $petId: Integer
    HEADER 'api_key' = $ApiKey
    RESPONSE NONE;
END;
```

### CREATE REST CLIENT

Full syntax reference:

```sql
CREATE REST CLIENT qualifiedName
BASE URL 'url'
AUTHENTICATION authScheme
BEGIN
  operationDef*
END;
```

**Authentication schemes:**

```sql
AUTHENTICATION NONE
AUTHENTICATION BASIC (USERNAME = 'literal', PASSWORD = 'literal')
AUTHENTICATION BASIC (USERNAME = $Variable, PASSWORD = $Variable)
```

**Operation definition:**

```sql
[docComment]
OPERATION name
  METHOD GET|POST|PUT|PATCH|DELETE
  PATH '/path/{param}'
  [PARAMETER $name: Type]*          -- path parameters (extracted from {param} in PATH)
  [QUERY $name: Type]*              -- query parameters
  [HEADER 'name' = headerValue]*    -- static or dynamic headers
  [BODY bodySpec]                   -- request body (POST/PUT/PATCH only)
  [TIMEOUT seconds]                 -- override default 300s
  RESPONSE responseSpec;            -- response handling
```

**Header values:**

```sql
HEADER 'Accept' = 'application/json'           -- static literal
HEADER 'X-Request-ID' = $RequestId             -- dynamic from parameter
HEADER 'Authorization' = 'Bearer ' + $Token    -- concatenation
```

**Body types:**

```sql
BODY JSON FROM $Variable       -- JSON body from variable (maps to ImplicitMappingBody)
BODY FILE FROM $FileDocument   -- binary file upload (maps to StringBody with file content)
```

**Response types:**

```sql
RESPONSE JSON AS $Variable     -- JSON response mapped to entity
RESPONSE STRING AS $Variable   -- raw string response
RESPONSE FILE AS $Variable     -- binary file download
RESPONSE STATUS AS $Variable   -- HTTP status code only
RESPONSE NONE                  -- no response expected
```

### DROP REST CLIENT

```sql
DROP REST CLIENT Module.Name;
```

## BSON ↔ MDL Mapping

### Authentication

| MDL | BSON |
|-----|------|
| `AUTHENTICATION NONE` | `AuthenticationScheme: null` |
| `AUTHENTICATION BASIC (USERNAME = 'user', PASSWORD = 'pass')` | `AuthenticationScheme: Rest$BasicAuthenticationScheme { Username: Rest$StringValue, Password: Rest$StringValue }` |
| `AUTHENTICATION BASIC (USERNAME = $Var, PASSWORD = $Var)` | `AuthenticationScheme: Rest$BasicAuthenticationScheme { Username: Rest$ConstantValue, Password: Rest$ConstantValue }` |

### Operation Method

| MDL | BSON |
|-----|------|
| `METHOD GET` (no BODY) | `Rest$RestOperationMethodWithoutBody { HttpMethod: "Get" }` |
| `METHOD DELETE` (no BODY) | `Rest$RestOperationMethodWithoutBody { HttpMethod: "Delete" }` |
| `METHOD POST` + `BODY ...` | `Rest$RestOperationMethodWithBody { HttpMethod: "Post", Body: ... }` |
| `METHOD PUT` + `BODY ...` | `Rest$RestOperationMethodWithBody { HttpMethod: "Put", Body: ... }` |
| `METHOD PATCH` + `BODY ...` | `Rest$RestOperationMethodWithBody { HttpMethod: "Patch", Body: ... }` |

### Response Handling

| MDL | BSON |
|-----|------|
| `RESPONSE NONE` | `Rest$NoResponseHandling` |
| `RESPONSE STATUS AS $Var` | `Rest$NoResponseHandling { StatusCode: ... }` |
| `RESPONSE JSON AS $Var` | `Rest$ImplicitMappingResponseHandling { ContentType: "application/json" }` |
| `RESPONSE STRING AS $Var` | `Rest$NoResponseHandling` (with string result handling) |
| `RESPONSE FILE AS $Var` | `Rest$NoResponseHandling` (with file result handling) |

### Headers

| MDL | BSON |
|-----|------|
| `HEADER 'Accept' = 'application/json'` | `Rest$HeaderWithValueTemplate { Name: "Accept", Value: Rest$ValueTemplate { Value: "application/json" } }` |
| `HEADER 'Auth' = 'Bearer ' + $Token` | `Rest$HeaderWithValueTemplate { Name: "Auth", Value: Rest$ValueTemplate { Value: "Bearer {1}" } }` + parameter reference |

### Parameters

| MDL | BSON |
|-----|------|
| `PARAMETER $userId: Integer` | `Rest$RestOperationParameter { Name: "userId", DataType: Integer }` (path parameter, matches `{userId}` in PATH) |
| `QUERY $search: String` | `Rest$QueryParameter { Name: "search", ParameterUsage: Rest$RequiredQueryParameterUsage }` |

## Implementation Plan

### Phase 1: Read Support (SHOW / DESCRIBE)

#### 1.1 Add Model Types (`model/types.go`)

```go
// ConsumedRestService represents a consumed REST service document.
type ConsumedRestService struct {
    ContainerID    ID
    Name           string
    Documentation  string
    Excluded       bool
    BaseUrl        string
    Authentication *RestAuthentication // nil = NONE
    Operations     []*RestClientOperation
}

// RestAuthentication represents authentication configuration.
type RestAuthentication struct {
    Scheme   string // "Basic"
    Username string // literal value or constant reference
    Password string // literal value or constant reference
}

// RestClientOperation represents a single REST operation.
type RestClientOperation struct {
    Name            string
    Documentation   string
    HttpMethod      string // "GET", "POST", etc.
    Path            string
    Parameters      []*RestClientParameter // path parameters
    QueryParameters []*RestClientParameter // query parameters
    Headers         []*RestClientHeader
    BodyType        string // "JSON", "FILE", "" (none)
    BodyVariable    string // variable name for body
    ResponseType    string // "JSON", "STRING", "FILE", "STATUS", "NONE"
    ResponseVariable string // variable name for response
    Timeout         int    // 0 = default
}

// RestClientParameter represents a path or query parameter.
type RestClientParameter struct {
    Name     string
    DataType string // "String", "Integer", "Boolean", "Decimal"
}

// RestClientHeader represents an HTTP header.
type RestClientHeader struct {
    Name  string
    Value string // literal value or expression with parameters
}
```

#### 1.2 Add MPR Parser (`sdk/mpr/parser_rest.go`)

Extend existing file with:

```go
func (r *Reader) parseConsumedRestService(doc bson.Raw) *model.ConsumedRestService
```

Handles the polymorphic types: `RestOperationMethodWithBody` vs `WithoutBody`, `BasicAuthenticationScheme` vs null, `ImplicitMappingResponseHandling` vs `NoResponseHandling`.

#### 1.3 Add Reader (`sdk/mpr/reader_documents.go`)

```go
func (r *Reader) ListConsumedRestServices() []*model.ConsumedRestService
```

Pattern: follow `ListPublishedRestServices()` — query documents by `$Type = "Rest$ConsumedRestService"`.

#### 1.4 Add AST Types (`mdl/ast/ast_query.go`)

Add to existing enums:

```go
// In ShowObjectType
ShowRestClients

// In DescribeObjectType
DescribeRestClient
```

#### 1.5 Add Visitor (`mdl/visitor/visitor_query.go`)

Add cases in `exitShowStatement()` and `exitDescribeStatement()`:

```go
// SHOW REST CLIENTS [IN module]
if ctx.REST() != nil && ctx.CLIENTS() != nil { ... }

// DESCRIBE REST CLIENT qualifiedName
if ctx.REST() != nil && ctx.CLIENT() != nil { ... }
```

#### 1.6 Add Executor (`mdl/executor/cmd_rest_clients.go`)

New file, following `cmd_odata.go` pattern:

```go
func (e *Executor) showRestClients(moduleName string) error
func (e *Executor) describeRestClient(name ast.QualifiedName) error
func (e *Executor) outputConsumedRestServiceMDL(svc *model.ConsumedRestService) string
```

The `outputConsumedRestServiceMDL()` function must produce valid CREATE REST CLIENT syntax that roundtrips.

#### 1.7 Add Autocomplete

```go
func (e *Executor) GetRestClientNames(moduleFilter string) []string
```

### Phase 2: Write Support (CREATE / DROP)

#### 2.1 Add AST Types (`mdl/ast/ast_rest.go`)

New file:

```go
type CreateRestClientStmt struct {
    Name           QualifiedName
    BaseUrl        string
    Authentication *RestAuthDef // nil = NONE
    Operations     []*RestOperationDef
    Documentation  string
    CreateOrModify bool
}

type RestAuthDef struct {
    Scheme   string // "BASIC"
    Username string // literal or $variable
    Password string // literal or $variable
}

type RestOperationDef struct {
    Name            string
    Documentation   string
    Method          string // "GET", "POST", etc.
    Path            string
    Parameters      []RestParamDef // path parameters
    QueryParameters []RestParamDef // query parameters
    Headers         []RestHeaderDef
    BodyType        string // "JSON", "FILE", ""
    BodyVariable    string
    ResponseType    string // "JSON", "STRING", "FILE", "STATUS", "NONE"
    ResponseVariable string
    Timeout         int
}

type RestParamDef struct {
    Name     string // includes $ prefix
    DataType string
}

type RestHeaderDef struct {
    Name  string
    Value string // can be literal, $variable, or 'prefix' + $variable
}

type DropRestClientStmt struct {
    Name QualifiedName
}
```

#### 2.2 Update Grammar (`MDLParser.g4`)

The existing grammar rules (lines 1997-2048) need significant revision to match the MDL syntax:

```antlr
createRestClientStatement
    : REST CLIENT qualifiedName
      restClientBaseUrl
      restClientAuthentication
      BEGIN restOperationDef* END
    ;

restClientBaseUrl
    : BASE URL STRING_LITERAL
    ;

restClientAuthentication
    : AUTHENTICATION NONE
    | AUTHENTICATION BASIC LPAREN
        USERNAME ASSIGN restAuthValue COMMA
        PASSWORD ASSIGN restAuthValue
      RPAREN
    ;

restAuthValue
    : STRING_LITERAL          // literal: 'api_user'
    | VARIABLE               // parameter: $ApiUsername
    ;

restOperationDef
    : documentationComment?
      OPERATION (IDENTIFIER | STRING_LITERAL)
        METHOD restHttpMethod
        PATH STRING_LITERAL
        restOperationClause*
        RESPONSE restResponseSpec SEMICOLON
    ;

restHttpMethod
    : GET | POST | PUT | PATCH | DELETE
    ;

restOperationClause
    : PARAMETER VARIABLE COLON dataType                         // path param
    | QUERY VARIABLE COLON dataType                             // query param
    | HEADER STRING_LITERAL ASSIGN restHeaderValue              // header
    | BODY (JSON | FILE) FROM VARIABLE                          // request body
    | TIMEOUT NUMBER_LITERAL                                    // timeout override
    ;

restHeaderValue
    : STRING_LITERAL                                            // 'application/json'
    | VARIABLE                                                  // $RequestId
    | STRING_LITERAL PLUS VARIABLE                              // 'Bearer ' + $Token
    ;

restResponseSpec
    : JSON AS VARIABLE         // JSON response
    | STRING AS VARIABLE       // string response
    | FILE AS VARIABLE         // file download
    | STATUS AS VARIABLE       // status code only
    | NONE                     // no response
    ;
```

**New tokens needed:** `FILE` (if not already defined), `STRING` (keyword, not `STRING_LITERAL`).

#### 2.3 Add Visitor (`mdl/visitor/visitor_rest.go`)

New file to convert grammar parse tree → AST:

```go
func (b *ASTBuilder) exitCreateRestClientStatement(ctx *parser.CreateRestClientStatementContext)
```

#### 2.4 Add Writer (`sdk/mpr/writer_rest.go`)

New file for BSON serialization:

```go
func (w *Writer) WriteConsumedRestService(svc *model.ConsumedRestService) error
```

Must correctly produce:
- Polymorphic method types (`RestOperationMethodWithBody` vs `WithoutBody`)
- Authentication scheme or null
- ValueTemplate structures for URLs, paths, headers
- Response handling types
- Query parameter usage types

#### 2.5 Add Executor Create/Drop Handlers

In `mdl/executor/cmd_rest_clients.go`:

```go
func (e *Executor) createRestClient(s *ast.CreateRestClientStmt) error
func (e *Executor) dropRestClient(s *ast.DropRestClientStmt) error
```

### Phase 3: Test Enablement

Remove the `exit;` guard from `06-rest-client-examples.mdl` (line 34) and verify all 21 examples parse and execute correctly.

## Complexity

**Medium-High** — Multiple polymorphic BSON types (method, body, response handling, authentication, parameter values), value templates with parameter interpolation, and header expression parsing. The OData implementation provides a proven template but REST has more polymorphic variance.

## Testing

- Parse all 21 examples in `06-rest-client-examples.mdl` with `mxcli check`
- Roundtrip test: CREATE → DESCRIBE → re-parse must produce identical AST
- Verify BSON output against Evora project's existing consumed REST services
- **Important**: Before writing BSON, create a reference REST client in Studio Pro and compare the generated BSON structure field-by-field

## Related

- `docs/03-development/proposals/show-describe-published-rest-services.md` — Published REST services (opposite direction: exposing endpoints)
- `mdl/executor/cmd_odata.go` — OData client implementation (template for this work)
- `mdl/ast/ast_odata.go` — OData AST types (template for REST AST)
- `mdl/executor/cmd_microflows_builder_calls.go:576` — Existing REST CALL microflow action (different feature: inline REST calls in microflows)
