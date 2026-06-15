/**
 * MDL Service Grammar — database connections, REST clients, published REST services,
 * OData clients/services, business event services, navigation, configuration.
 */
parser grammar MDLService;

options { tokenVocab = MDLLexer; }

// =============================================================================
// DATABASE / REST CLIENT
// =============================================================================

createDatabaseConnectionStatement
    : DATABASE CONNECTION qualifiedName
      databaseConnectionOption+
      (LBRACE databaseQuery* RBRACE)?
    ;

databaseConnectionOption
    : TYPE STRING_LITERAL
    | CONNECTION STRING_TYPE (STRING_LITERAL | AT qualifiedName)
    | HOST STRING_LITERAL
    | PORT NUMBER_LITERAL
    | DATABASE STRING_LITERAL
    | USERNAME (STRING_LITERAL | AT qualifiedName)
    | PASSWORD (STRING_LITERAL | AT qualifiedName)
    ;

databaseQuery
    : QUERY identifierOrKeyword
      SQL (STRING_LITERAL | DOLLAR_STRING)
      (PARAMETER identifierOrKeyword COLON dataType (DEFAULT STRING_LITERAL | NULL)?)*
      (RETURNS qualifiedName
        (MAP LPAREN databaseQueryMapping (COMMA databaseQueryMapping)* RPAREN)?
      )?
      SEMICOLON
    ;

databaseQueryMapping
    : identifierOrKeyword AS identifierOrKeyword
    ;

createConfigurationStatement
    : CONFIGURATION STRING_LITERAL
      (settingsAssignment (COMMA settingsAssignment)*)?
    ;

/**
 * CREATE REST CLIENT — property-based syntax with { } blocks.
 */
createRestClientStatement
    : REST CLIENT qualifiedName
      LPAREN restClientProperty (COMMA restClientProperty)* RPAREN
      (LBRACE restClientOperation* RBRACE)?
    ;

restClientProperty
    : identifierOrKeyword COLON STRING_LITERAL                       // BaseUrl: '...', Username: '...'
    | identifierOrKeyword COLON VARIABLE                             // Username: $Constant (legacy, stored as Rest$ConstantValue)
    | identifierOrKeyword COLON AT qualifiedName                     // Username: @Module.Constant (preferred Mendix convention)
    | identifierOrKeyword COLON NONE                                 // Authentication: NONE
    | identifierOrKeyword COLON BASIC LPAREN restClientProperty (COMMA restClientProperty)* RPAREN
    ;

restClientOperation
    : docComment?
      OPERATION (identifierOrKeyword | STRING_LITERAL)
      LBRACE restClientOpProp (COMMA restClientOpProp)* RBRACE
    ;

restClientOpProp
    : identifierOrKeyword COLON restHttpMethod                       // Method: GET
    | identifierOrKeyword COLON STRING_LITERAL                       // Path: '/items'
    | identifierOrKeyword COLON NUMBER_LITERAL                       // Timeout: 30
    | identifierOrKeyword COLON NONE                                 // Response: NONE
    | identifierOrKeyword COLON LPAREN restClientParamItem (COMMA restClientParamItem)* RPAREN  // Parameters/Query
    | identifierOrKeyword COLON LPAREN restClientHeaderItem (COMMA restClientHeaderItem)* RPAREN  // Headers
    | identifierOrKeyword COLON (JSON | FILE_KW | STRING_TYPE | STATUS) (FROM | AS) VARIABLE  // Body: JSON FROM $v, Response: JSON AS $v
    | identifierOrKeyword COLON TEMPLATE STRING_LITERAL              // Body: TEMPLATE '...'
    | identifierOrKeyword COLON MAPPING qualifiedName (LBRACE restClientMappingEntry* RBRACE)?  // Body/Response: MAPPING Entity { ... }
    ;

restClientParamItem
    : VARIABLE COLON dataType
    ;

restClientHeaderItem
    : STRING_LITERAL EQUALS (STRING_LITERAL | VARIABLE | STRING_LITERAL PLUS VARIABLE)
    ;

restClientMappingEntry
    : identifierOrKeyword EQUALS identifierOrKeyword COMMA?                          // Attr = jsonField,
    | CREATE? qualifiedName SLASH qualifiedName EQUALS identifierOrKeyword
      (LBRACE restClientMappingEntry* RBRACE)? COMMA?                                // CREATE Assoc/Entity = jsonField { ... },
    ;

restHttpMethod
    : GET | POST | PUT | PATCH | DELETE
    ;

// =============================================================================
// PUBLISHED REST SERVICE CREATION
// =============================================================================

createPublishedRestServiceStatement
    : PUBLISHED REST SERVICE qualifiedName
      LPAREN publishedRestProperty (COMMA publishedRestProperty)* RPAREN
      LBRACE publishedRestResource* RBRACE
    ;

publishedRestProperty
    : identifierOrKeyword COLON STRING_LITERAL
    ;

publishedRestResource
    : RESOURCE STRING_LITERAL LBRACE publishedRestOperation* RBRACE
    ;

publishedRestOperation
    : restHttpMethod publishedRestOpPath?
      MICROFLOW qualifiedName
      (DEPRECATED)?
      (IMPORT MAPPING qualifiedName)?
      (EXPORT MAPPING qualifiedName)?
      (COMMIT identifierOrKeyword)?
      SEMICOLON?
    ;

publishedRestOpPath
    : STRING_LITERAL
    | SLASH
    ;

// =============================================================================
// ODATA CLIENT / SERVICE
// =============================================================================

createODataClientStatement
    : ODATA CLIENT qualifiedName
      LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN
      odataHeadersClause?
    ;

createODataServiceStatement
    : ODATA SERVICE qualifiedName
      LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN
      odataAuthenticationClause?
      (LBRACE publishEntityBlock* RBRACE)?
    ;

odataPropertyValue
    : STRING_LITERAL
    | NUMBER_LITERAL
    | TRUE
    | FALSE
    | MICROFLOW qualifiedName?
    | AT qualifiedName              // @Module.ConstantName (Mendix constant reference — required for ServiceUrl)
    | qualifiedName
    ;

odataPropertyAssignment
    : identifierOrKeyword COLON odataPropertyValue
    ;

odataAlterAssignment
    : identifierOrKeyword EQUALS odataPropertyValue
    ;

odataAuthenticationClause
    : AUTHENTICATION odataAuthType (COMMA odataAuthType)*
    ;

odataAuthType
    : BASIC
    | SESSION
    | GUEST
    | MICROFLOW qualifiedName?
    | IDENTIFIER  // For custom types like 'Custom'
    ;

publishEntityBlock
    : PUBLISH ENTITY qualifiedName (AS STRING_LITERAL)?
      (LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN)?
      exposeClause?
      SEMICOLON?
    ;

exposeClause
    : EXPOSE LPAREN (STAR | exposeMember (COMMA exposeMember)*) RPAREN
    ;

exposeMember
    : identifierOrKeyword (AS STRING_LITERAL)? exposeMemberOptions?
    ;

exposeMemberOptions
    : LPAREN identifierOrKeyword (COMMA identifierOrKeyword)* RPAREN
    ;

createExternalEntityStatement
    : EXTERNAL ENTITY qualifiedName
      FROM ODATA CLIENT qualifiedName
      LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN
      (LPAREN attributeDefinitionList? RPAREN)?
    ;

createExternalEntitiesStatement
    : EXTERNAL ENTITIES FROM qualifiedName
      (INTO (qualifiedName | IDENTIFIER))?
      (ENTITIES LPAREN identifierOrKeyword (COMMA identifierOrKeyword)* RPAREN)?
    ;

createNavigationStatement
    : NAVIGATION (qualifiedName | IDENTIFIER) navigationClause*
    ;

odataHeadersClause
    : HEADERS LPAREN odataHeaderEntry (COMMA odataHeaderEntry)* RPAREN
    ;

odataHeaderEntry
    : STRING_LITERAL COLON odataPropertyValue
    ;

// =============================================================================
// BUSINESS EVENT SERVICE
// =============================================================================

createBusinessEventServiceStatement
    : BUSINESS EVENT SERVICE qualifiedName
      LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN
      LBRACE businessEventMessageDef+ RBRACE
    ;

businessEventMessageDef
    : MESSAGE identifierOrKeyword
      LPAREN businessEventAttrDef (COMMA businessEventAttrDef)* RPAREN
      (PUBLISH | SUBSCRIBE)
      (ENTITY qualifiedName)?
      (MICROFLOW qualifiedName)?
      SEMICOLON
    ;

businessEventAttrDef
    : identifierOrKeyword COLON dataType
    ;
