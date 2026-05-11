/**
 * MDL Catalog Grammar — DQL statements (SHOW/LIST/DESCRIBE/SELECT catalog), OQL query.
 */
parser grammar MDLCatalog;

options { tokenVocab = MDLLexer; }

// =============================================================================
// DQL STATEMENTS (Data Query Language)
// =============================================================================

showOrList: SHOW | LIST_KW ;

showStatement
    : showOrList MODULES
    | showOrList CONTRACT ENTITIES FROM qualifiedName    // SHOW CONTRACT ENTITIES FROM Module.Service
    | showOrList CONTRACT ACTIONS FROM qualifiedName     // SHOW CONTRACT ACTIONS FROM Module.Service
    | showOrList CONTRACT CHANNELS FROM qualifiedName   // SHOW CONTRACT CHANNELS FROM Module.Service (AsyncAPI)
    | showOrList CONTRACT MESSAGES FROM qualifiedName   // SHOW CONTRACT MESSAGES FROM Module.Service (AsyncAPI)
    | showOrList ENTITIES (IN (qualifiedName | IDENTIFIER))?
    | showOrList ASSOCIATIONS (IN (qualifiedName | IDENTIFIER))?
    | showOrList MICROFLOWS (IN (qualifiedName | IDENTIFIER))?
    | showOrList NANOFLOWS (IN (qualifiedName | IDENTIFIER))?
    | showOrList WORKFLOWS (IN (qualifiedName | IDENTIFIER))?
    | showOrList PAGES (IN (qualifiedName | IDENTIFIER))?
    | showOrList SNIPPETS (IN (qualifiedName | IDENTIFIER))?
    | showOrList ENUMERATIONS (IN (qualifiedName | IDENTIFIER))?
    | showOrList CONSTANTS (IN (qualifiedName | IDENTIFIER))?
    | showOrList CONSTANT VALUES (IN (qualifiedName | IDENTIFIER))?
    | showOrList LAYOUTS (IN (qualifiedName | IDENTIFIER))?
    | showOrList NOTEBOOKS (IN (qualifiedName | IDENTIFIER))?
    | showOrList JAVA ACTIONS (IN (qualifiedName | IDENTIFIER))?
    | showOrList JAVASCRIPT ACTIONS (IN (qualifiedName | IDENTIFIER))?
    | showOrList IMAGE COLLECTION (IN (qualifiedName | IDENTIFIER))?
    | showOrList MODELS (IN (qualifiedName | IDENTIFIER))?
    | showOrList AGENTS (IN (qualifiedName | IDENTIFIER))?
    | showOrList KNOWLEDGE BASES (IN (qualifiedName | IDENTIFIER))?
    | showOrList CONSUMED MCP SERVICES (IN (qualifiedName | IDENTIFIER))?
    | showOrList JSON STRUCTURES (IN (qualifiedName | IDENTIFIER))?
    | showOrList IMPORT MAPPINGS (IN (qualifiedName | IDENTIFIER))?
    | showOrList EXPORT MAPPINGS (IN (qualifiedName | IDENTIFIER))?
    | showOrList ENTITY qualifiedName
    | showOrList ASSOCIATION qualifiedName
    | showOrList PAGE qualifiedName
    | showOrList CONNECTIONS
    | showOrList STATUS
    | showOrList VERSION
    | showOrList CATALOG STATUS
    | showOrList CATALOG TABLES
    | showOrList CALLERS OF qualifiedName TRANSITIVE?
    | showOrList CALLEES OF qualifiedName TRANSITIVE?
    | showOrList REFERENCES TO qualifiedName
    | showOrList IMPACT OF qualifiedName
    | showOrList CONTEXT OF qualifiedName (DEPTH NUMBER_LITERAL)?
    | showOrList WIDGETS showWidgetsFilter?
    | showOrList PROJECT SECURITY
    | showOrList MODULE ROLES (IN (qualifiedName | IDENTIFIER))?
    | showOrList USER ROLES
    | showOrList DEMO USERS
    | showOrList ACCESS ON qualifiedName
    | showOrList ACCESS ON MICROFLOW qualifiedName
    | showOrList ACCESS ON PAGE qualifiedName
    | showOrList ACCESS ON WORKFLOW qualifiedName
    | showOrList ACCESS ON NANOFLOW qualifiedName
    | showOrList SECURITY MATRIX (IN (qualifiedName | IDENTIFIER))?
    | showOrList ODATA CLIENTS (IN (qualifiedName | IDENTIFIER))?
    | showOrList ODATA SERVICES (IN (qualifiedName | IDENTIFIER))?
    | showOrList EXTERNAL ENTITIES (IN (qualifiedName | IDENTIFIER))?
    | showOrList EXTERNAL ACTIONS (IN (qualifiedName | IDENTIFIER))?
    | showOrList NAVIGATION
    | showOrList NAVIGATION MENU_KW (qualifiedName | IDENTIFIER)?
    | showOrList NAVIGATION HOMES
    | showOrList DESIGN PROPERTIES (FOR widgetTypeKeyword)?
    | showOrList STRUCTURE (DEPTH NUMBER_LITERAL)? (IN (qualifiedName | IDENTIFIER))? ALL?
    | showOrList BUSINESS EVENT SERVICES (IN (qualifiedName | IDENTIFIER))?
    | showOrList BUSINESS EVENT CLIENTS (IN (qualifiedName | IDENTIFIER))?
    | showOrList BUSINESS EVENTS (IN (qualifiedName | IDENTIFIER))?
    | showOrList SETTINGS
    | showOrList FRAGMENTS
    | showOrList DATABASE CONNECTIONS (IN (qualifiedName | IDENTIFIER))?
    | showOrList REST CLIENTS (IN (qualifiedName | IDENTIFIER))?
    | showOrList PUBLISHED REST SERVICES (IN (qualifiedName | IDENTIFIER))?
    | showOrList DATA TRANSFORMERS (IN (qualifiedName | IDENTIFIER))?
    | showOrList LANGUAGES
    | showOrList FEATURES (IN IDENTIFIER)?
    | showOrList FEATURES FOR VERSION NUMBER_LITERAL
    | showOrList FEATURES ADDED SINCE NUMBER_LITERAL
    | showOrList JAR DEPENDENCIES (IN (qualifiedName | IDENTIFIER))?
    ;

/**
 * Widget filtering for SHOW WIDGETS and UPDATE WIDGETS.
 */
showWidgetsFilter
    : WHERE widgetCondition (AND widgetCondition)* (IN (qualifiedName | IDENTIFIER))?
    | IN (qualifiedName | IDENTIFIER)
    ;

/**
 * Widget type keyword for SHOW DESIGN PROPERTIES FOR <type>.
 */
widgetTypeKeyword
    : CONTAINER | TEXTBOX | TEXTAREA | CHECKBOX | RADIOBUTTONS | DATEPICKER
    | COMBOBOX | DYNAMICTEXT | ACTIONBUTTON | LINKBUTTON | DATAVIEW
    | LISTVIEW | DATAGRID | GALLERY | LAYOUTGRID | IMAGE | STATICIMAGE
    | DYNAMICIMAGE | HEADER | FOOTER | SNIPPETCALL | NAVIGATIONLIST
    | CUSTOMCONTAINER | TABCONTAINER | TABPAGE | DROPDOWN | REFERENCESELECTOR | GROUPBOX
    | IDENTIFIER
    ;

widgetCondition
    : WIDGETTYPE (EQUALS | LIKE) STRING_LITERAL
    | IDENTIFIER (EQUALS | LIKE) STRING_LITERAL
    ;

widgetPropertyAssignment
    : STRING_LITERAL EQUALS widgetPropertyValue
    ;

widgetPropertyValue
    : STRING_LITERAL
    | NUMBER_LITERAL
    | booleanLiteral
    | NULL
    ;

describeStatement
    : DESCRIBE CONTRACT ENTITY qualifiedName (FORMAT IDENTIFIER)?   // DESCRIBE CONTRACT ENTITY Service.Entity [FORMAT mdl] (must precede DESCRIBE ENTITY)
    | DESCRIBE CONTRACT ACTION qualifiedName (FORMAT IDENTIFIER)?   // DESCRIBE CONTRACT ACTION Service.Action [FORMAT mdl]
    | DESCRIBE CONTRACT MESSAGE qualifiedName    // DESCRIBE CONTRACT MESSAGE Module.Service.MessageName
    | DESCRIBE ENTITY qualifiedName
    | DESCRIBE ASSOCIATION qualifiedName
    | DESCRIBE MICROFLOW qualifiedName
    | DESCRIBE NANOFLOW qualifiedName
    | DESCRIBE WORKFLOW qualifiedName
    | DESCRIBE PAGE qualifiedName
    | DESCRIBE SNIPPET qualifiedName
    | DESCRIBE LAYOUT qualifiedName
    | DESCRIBE ENUMERATION qualifiedName
    | DESCRIBE CONSTANT qualifiedName
    | DESCRIBE JAVA ACTION qualifiedName
    | DESCRIBE JAVASCRIPT ACTION qualifiedName
    | DESCRIBE MODULE identifierOrKeyword (WITH ALL)?  // DESCRIBE MODULE Name [WITH ALL] - optionally include all objects
    | DESCRIBE MODULE ROLE qualifiedName        // DESCRIBE MODULE ROLE Module.RoleName
    | DESCRIBE USER ROLE STRING_LITERAL          // DESCRIBE USER ROLE 'Administrator'
    | DESCRIBE DEMO USER STRING_LITERAL          // DESCRIBE DEMO USER 'demo_admin'
    | DESCRIBE ODATA CLIENT qualifiedName       // DESCRIBE ODATA CLIENT Module.ServiceName
    | DESCRIBE ODATA SERVICE qualifiedName      // DESCRIBE ODATA SERVICE Module.ServiceName
    | DESCRIBE EXTERNAL ENTITY qualifiedName    // DESCRIBE EXTERNAL ENTITY Module.EntityName
    | DESCRIBE NAVIGATION (qualifiedName | IDENTIFIER)?  // DESCRIBE NAVIGATION [profile]
    | DESCRIBE STYLING ON (PAGE | SNIPPET) qualifiedName (WIDGET IDENTIFIER)?  // DESCRIBE STYLING ON PAGE Module.Page [WIDGET name]
    | DESCRIBE CATALOG DOT (catalogTableName)  // DESCRIBE CATALOG.ENTITIES
    | DESCRIBE BUSINESS EVENT SERVICE qualifiedName  // DESCRIBE BUSINESS EVENT SERVICE Module.Name
    | DESCRIBE DATABASE CONNECTION qualifiedName       // DESCRIBE DATABASE CONNECTION Module.Name
    | DESCRIBE SETTINGS                               // DESCRIBE SETTINGS
    | DESCRIBE FRAGMENT FROM PAGE qualifiedName WIDGET identifierOrKeyword     // DESCRIBE FRAGMENT FROM PAGE Module.Page WIDGET name
    | DESCRIBE FRAGMENT FROM SNIPPET qualifiedName WIDGET identifierOrKeyword  // DESCRIBE FRAGMENT FROM SNIPPET Module.Snippet WIDGET name
    | DESCRIBE IMAGE COLLECTION qualifiedName           // DESCRIBE IMAGE COLLECTION Module.Name
    | DESCRIBE MODEL qualifiedName                      // DESCRIBE MODEL Module.Name (agent-editor)
    | DESCRIBE AGENT qualifiedName                      // DESCRIBE AGENT Module.Name (agent-editor)
    | DESCRIBE KNOWLEDGE BASE qualifiedName             // DESCRIBE KNOWLEDGE BASE Module.Name
    | DESCRIBE CONSUMED MCP SERVICE qualifiedName       // DESCRIBE CONSUMED MCP SERVICE Module.Name
    | DESCRIBE JSON STRUCTURE qualifiedName              // DESCRIBE JSON STRUCTURE Module.Name
    | DESCRIBE IMPORT MAPPING qualifiedName             // DESCRIBE IMPORT MAPPING Module.Name
    | DESCRIBE EXPORT MAPPING qualifiedName             // DESCRIBE EXPORT MAPPING Module.Name
    | DESCRIBE REST CLIENT qualifiedName                // DESCRIBE REST CLIENT Module.Name
    | DESCRIBE CONTRACT OPERATION FROM OPENAPI STRING_LITERAL   // DESCRIBE CONTRACT OPERATION FROM OPENAPI '/path/to/spec.json'
    | DESCRIBE PUBLISHED REST SERVICE qualifiedName    // DESCRIBE PUBLISHED REST SERVICE Module.Name
    | DESCRIBE DATA TRANSFORMER qualifiedName          // DESCRIBE DATA TRANSFORMER Module.Name
    | DESCRIBE FRAGMENT identifierOrKeyword            // DESCRIBE FRAGMENT Name
    | DESCRIBE JAR DEPENDENCY (qualifiedName | IDENTIFIER) STRING_LITERAL   // DESCRIBE JAR DEPENDENCY ModuleName 'group:artifact'
    ;

catalogSelectQuery
    : SELECT (DISTINCT | ALL)? selectList
      FROM CATALOG DOT catalogTableName (AS? IDENTIFIER)?
      (catalogJoinClause)*
      (WHERE whereExpr=expression)?
      (GROUP_BY groupByList (HAVING havingExpr=expression)?)?
      (ORDER_BY orderByList)?
      (LIMIT NUMBER_LITERAL)?
      (OFFSET NUMBER_LITERAL)?
    ;

catalogJoinClause
    : joinType? JOIN CATALOG DOT catalogTableName (AS? IDENTIFIER)? (ON expression)?
    ;

// Table names for catalog can be keywords or identifiers
catalogTableName
    : MODULES
    | ENTITIES
    | ASSOCIATIONS  // keyword token — must be listed explicitly
    | MICROFLOWS
    | NANOFLOWS
    | PAGES
    | SNIPPETS
    | LAYOUTS
    | ENUMERATIONS
    | ATTRIBUTES
    | WIDGETS
    | WORKFLOWS
    | CONSTANTS     // keyword token — must be listed explicitly
    | OBJECTS       // keyword token — must be listed explicitly
    | SOURCE_KW     // For CATALOG.SOURCE FTS table
    | ODATA         // For CATALOG.ODATA_CLIENTS and CATALOG.ODATA_SERVICES (via IDENTIFIER)
    | IDENTIFIER    // For tables like activities, xpath_expressions, projects, snapshots, refs, strings, odata_clients, odata_services, java_actions
    ;

// =============================================================================
// OQL QUERY (Object Query Language)
// =============================================================================

/**
 * OQL (Object Query Language) query for retrieving data.
 */
oqlQuery
    : oqlQueryTerm (UNION ALL? oqlQueryTerm)*
    ;

oqlQueryTerm
    : selectClause fromClause? whereClause? groupByClause? havingClause?
      orderByClause? limitOffsetClause?
    | fromClause whereClause? groupByClause? havingClause?
      selectClause orderByClause? limitOffsetClause?
    ;

selectClause
    : SELECT (DISTINCT | ALL)? selectList
    ;

selectList
    : STAR
    | selectItem (COMMA selectItem)*
    ;

selectItem
    : expression (AS selectAlias)?
    | aggregateFunction (AS selectAlias)?
    ;

// Allow keywords as aliases in SELECT
selectAlias
    : IDENTIFIER
    | keyword
    ;

fromClause
    : FROM tableReference (joinClause)*
    ;

tableReference
    : qualifiedName (AS? IDENTIFIER)?
    | LPAREN oqlQuery RPAREN (AS? IDENTIFIER)?
    ;

joinClause
    : joinType? JOIN tableReference (ON expression)?
    | joinType? JOIN associationPath (AS? IDENTIFIER)?
    ;

// OQL association path formats:
// - Association/Entity (e.g., Shop.BillingAddress_Customer/Shop.Customer)
// - alias/Association/Entity (e.g., c/Shop.DeliveryAddress_Customer/Shop.Address)
associationPath
    : IDENTIFIER SLASH qualifiedName SLASH qualifiedName  // alias/Association/Entity
    | qualifiedName SLASH qualifiedName                    // Association/Entity
    ;

joinType
    : LEFT OUTER?
    | RIGHT OUTER?
    | INNER
    | FULL OUTER?
    | CROSS
    ;

whereClause
    : WHERE expression
    ;

groupByClause
    : GROUP_BY expressionList
    ;

havingClause
    : HAVING expression
    ;

orderByClause
    : ORDER_BY orderByList
    ;

orderByList
    : orderByItem (COMMA orderByItem)*
    ;

orderByItem
    : expression (ASC | DESC)?
    ;

groupByList
    : expression (COMMA expression)*
    ;

limitOffsetClause
    : LIMIT NUMBER_LITERAL (OFFSET NUMBER_LITERAL)?
    | OFFSET NUMBER_LITERAL (LIMIT NUMBER_LITERAL)?
    ;
