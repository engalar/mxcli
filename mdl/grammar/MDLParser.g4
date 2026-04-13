/**
 * MDL (Mendix Definition Language) Parser Grammar
 *
 * ANTLR4 parser for MDL syntax used by the Mendix REPL.
 * Converted from Chevrotain-based parser.
 */
parser grammar MDLParser;

options {
    tokenVocab = MDLLexer;
}

// =============================================================================
// TOP-LEVEL RULES
// =============================================================================

/** Entry point: a program is a sequence of statements */
program
    : statement* EOF
    ;

/** A statement can be DDL, DQL, or utility */
statement
    : docComment? (ddlStatement | dqlStatement | utilityStatement) SEMICOLON? SLASH?
    ;

// =============================================================================
// DDL STATEMENTS (Data Definition Language)
// =============================================================================

ddlStatement
    : createStatement
    | alterStatement
    | dropStatement
    | renameStatement
    | moveStatement
    | updateWidgetsStatement
    | securityStatement
    ;

/**
 * Bulk update widget properties across pages/snippets.
 *
 * @example Preview changes (dry run)
 * ```mdl
 * UPDATE WIDGETS
 *   SET 'showLabel' = false
 *   WHERE WidgetType LIKE '%combobox%'
 *   DRY RUN;
 * ```
 *
 * @example Apply changes to widgets in a module
 * ```mdl
 * UPDATE WIDGETS
 *   SET 'filterMode' = 'contains'
 *   WHERE WidgetType LIKE '%DataGrid%'
 *   IN MyModule;
 * ```
 *
 * @example Multiple property assignments
 * ```mdl
 * UPDATE WIDGETS
 *   SET 'showLabel' = false, 'labelWidth' = 4
 *   WHERE WidgetType LIKE '%textbox%';
 * ```
 */
updateWidgetsStatement
    : UPDATE WIDGETS
      SET widgetPropertyAssignment (COMMA widgetPropertyAssignment)*
      WHERE widgetCondition (AND widgetCondition)*
      (IN (qualifiedName | IDENTIFIER))?
      (DRY RUN)?
    ;

createStatement
    : docComment? annotation*
      CREATE (OR (MODIFY | REPLACE))?
      ( createEntityStatement
      | createAssociationStatement
      | createModuleStatement
      | createMicroflowStatement
      | createJavaActionStatement
      | createPageStatement
      | createSnippetStatement
      | createEnumerationStatement
      | createValidationRuleStatement
      | createNotebookStatement
      | createDatabaseConnectionStatement
      | createConstantStatement
      | createRestClientStatement
      | createIndexStatement
      | createODataClientStatement
      | createODataServiceStatement
      | createExternalEntityStatement
      | createExternalEntitiesStatement
      | createNavigationStatement
      | createBusinessEventServiceStatement
      | createWorkflowStatement
      | createUserRoleStatement
      | createDemoUserStatement
      | createImageCollectionStatement
      | createJsonStructureStatement
      | createImportMappingStatement
      | createExportMappingStatement
      | createConfigurationStatement
      | createPublishedRestServiceStatement
      )
    ;

alterStatement
    : ALTER ENTITY qualifiedName alterEntityAction+
    | ALTER ASSOCIATION qualifiedName alterAssociationAction+
    | ALTER ENUMERATION qualifiedName alterEnumerationAction+
    | ALTER NOTEBOOK qualifiedName alterNotebookAction+
    | ALTER ODATA CLIENT qualifiedName SET odataAlterAssignment (COMMA odataAlterAssignment)*
    | ALTER ODATA SERVICE qualifiedName SET odataAlterAssignment (COMMA odataAlterAssignment)*
    | ALTER STYLING ON (PAGE | SNIPPET) qualifiedName WIDGET IDENTIFIER alterStylingAction+
    | ALTER SETTINGS alterSettingsClause
    | ALTER PAGE qualifiedName LBRACE alterPageOperation+ RBRACE
    | ALTER SNIPPET qualifiedName LBRACE alterPageOperation+ RBRACE
    | ALTER WORKFLOW qualifiedName alterWorkflowAction+ SEMICOLON?
    | ALTER PUBLISHED REST SERVICE qualifiedName alterPublishedRestServiceAction (COMMA? alterPublishedRestServiceAction)*
    ;

alterPublishedRestServiceAction
    : SET publishedRestAlterAssignment (COMMA publishedRestAlterAssignment)*
    | ADD publishedRestResource
    | DROP RESOURCE STRING_LITERAL
    ;

publishedRestAlterAssignment
    : identifierOrKeyword EQUALS STRING_LITERAL
    ;

/**
 * Styling modification actions for ALTER STYLING.
 *
 * @example Set Class and Style
 * ```mdl
 * ALTER STYLING ON PAGE MyModule.Page WIDGET btnSave
 *   SET Class = 'btn-lg', Style = 'margin-top: 8px;';
 * ```
 *
 * @example Set design property
 * ```mdl
 * ALTER STYLING ON PAGE MyModule.Page WIDGET ctn1
 *   SET 'Spacing top' = 'Large', 'Full width' = ON;
 * ```
 *
 * @example Clear all design properties
 * ```mdl
 * ALTER STYLING ON PAGE MyModule.Page WIDGET ctn1
 *   CLEAR DESIGN PROPERTIES;
 * ```
 */
alterStylingAction
    : SET alterStylingAssignment (COMMA alterStylingAssignment)*
    | CLEAR DESIGN PROPERTIES
    ;

alterStylingAssignment
    : CLASS EQUALS STRING_LITERAL                  // Class = 'my-class'
    | STYLE EQUALS STRING_LITERAL                  // Style = 'color: red;'
    | STRING_LITERAL EQUALS STRING_LITERAL         // 'Spacing top' = 'Large'
    | STRING_LITERAL EQUALS ON                     // 'Full width' = ON
    | STRING_LITERAL EQUALS OFF                    // 'Full width' = OFF
    ;

/**
 * ALTER PAGE operations for modifying widget trees in-place.
 *
 * @example Set property on widget
 * ```mdl
 * ALTER PAGE Module.Page {
 *   SET Caption = 'Save' ON btnSave
 * }
 * ```
 *
 * @example Insert widget after another
 * ```mdl
 * ALTER PAGE Module.Page {
 *   INSERT AFTER txtName { TEXTBOX txtNew (Label: 'New', Binds: Attr) }
 * }
 * ```
 *
 * @example Drop widgets
 * ```mdl
 * ALTER PAGE Module.Page {
 *   DROP WIDGET txtOld, txtUnused
 * }
 * ```
 *
 * @example Replace widget subtree
 * ```mdl
 * ALTER PAGE Module.Page {
 *   REPLACE footer1 WITH { FOOTER f1 { ACTIONBUTTON btn1 (Caption: 'OK', Action: SAVE_CHANGES) } }
 * }
 * ```
 */
alterPageOperation
    : alterPageSet SEMICOLON?
    | alterPageInsert SEMICOLON?
    | alterPageDrop SEMICOLON?
    | alterPageReplace SEMICOLON?
    | alterPageAddVariable SEMICOLON?
    | alterPageDropVariable SEMICOLON?
    ;

alterPageSet
    : SET LAYOUT EQUALS qualifiedName (MAP LPAREN alterLayoutMapping (COMMA alterLayoutMapping)* RPAREN)?  // SET Layout = Atlas_Core.TopBar MAP (Main -> Main)
    | SET alterPageAssignment ON widgetRef                             // SET Caption = 'Save' ON btnSave  |  ON dgProducts.Name
    | SET LPAREN alterPageAssignment (COMMA alterPageAssignment)* RPAREN ON widgetRef  // SET (Caption = 'Save', ButtonStyle = Success) ON btnSave
    | SET alterPageAssignment                                                    // SET Title = 'Edit' (page-level)
    ;

alterLayoutMapping
    : identifierOrKeyword AS identifierOrKeyword                                // OldPlaceholder AS NewPlaceholder
    ;

alterPageAssignment
    : DATASOURCE EQUALS dataSourceExprV3               // DataSource = SELECTION widgetName
    | identifierOrKeyword EQUALS propertyValueV3       // Caption = 'Save'
    | STRING_LITERAL EQUALS propertyValueV3             // 'showLabel' = false
    ;

alterPageInsert
    : INSERT AFTER widgetRef LBRACE pageBodyV3 RBRACE
    | INSERT BEFORE widgetRef LBRACE pageBodyV3 RBRACE
    ;

alterPageDrop
    : DROP WIDGET widgetRef (COMMA widgetRef)*
    ;

alterPageReplace
    : REPLACE widgetRef WITH LBRACE pageBodyV3 RBRACE
    ;

// Widget reference: plain name (btnSave) or dotted path (dgProducts.Name)
widgetRef
    : identifierOrKeyword DOT identifierOrKeyword    // dgProducts.Name (column ref)
    | identifierOrKeyword                            // btnSave (widget ref)
    ;

alterPageAddVariable
    : ADD VARIABLES_KW variableDeclaration    // ADD Variables $show: Boolean = 'true'
    ;

alterPageDropVariable
    : DROP VARIABLES_KW VARIABLE              // DROP Variables $show
    ;

navigationClause
    : HOME (PAGE | MICROFLOW) qualifiedName (FOR qualifiedName)?
    | LOGIN PAGE qualifiedName
    | NOT FOUND PAGE qualifiedName
    | MENU_KW LPAREN navMenuItemDef* RPAREN
    ;

navMenuItemDef
    : MENU_KW ITEM STRING_LITERAL ((PAGE qualifiedName) | (MICROFLOW qualifiedName))? SEMICOLON?
    | MENU_KW STRING_LITERAL LPAREN navMenuItemDef* RPAREN SEMICOLON?
    ;

dropStatement
    : DROP ENTITY qualifiedName
    | DROP ASSOCIATION qualifiedName
    | DROP ENUMERATION qualifiedName
    | DROP CONSTANT qualifiedName
    | DROP MICROFLOW qualifiedName
    | DROP NANOFLOW qualifiedName
    | DROP PAGE qualifiedName
    | DROP SNIPPET qualifiedName
    | DROP MODULE qualifiedName
    | DROP NOTEBOOK qualifiedName
    | DROP JAVA ACTION qualifiedName
    | DROP INDEX qualifiedName ON qualifiedName
    | DROP ODATA CLIENT qualifiedName
    | DROP ODATA SERVICE qualifiedName
    | DROP BUSINESS EVENT SERVICE qualifiedName
    | DROP WORKFLOW qualifiedName
    | DROP IMAGE COLLECTION qualifiedName
    | DROP JSON STRUCTURE qualifiedName
    | DROP IMPORT MAPPING qualifiedName
    | DROP EXPORT MAPPING qualifiedName
    | DROP REST CLIENT qualifiedName
    | DROP PUBLISHED REST SERVICE qualifiedName
    | DROP CONFIGURATION STRING_LITERAL
    | DROP FOLDER STRING_LITERAL IN (qualifiedName | IDENTIFIER)
    ;

renameStatement
    : RENAME renameTarget qualifiedName TO identifierOrKeyword (DRY RUN)?
    | RENAME MODULE identifierOrKeyword TO identifierOrKeyword (DRY RUN)?
    ;

renameTarget
    : ENTITY | MICROFLOW | NANOFLOW | PAGE | ENUMERATION | ASSOCIATION | CONSTANT
    ;

/**
 * Moves a document to a different folder or module.
 *
 * @example Move page to folder in same module
 * ```mdl
 * MOVE PAGE MyModule.MyPage TO FOLDER 'Resources/Pages';
 * ```
 *
 * @example Move microflow to folder in different module
 * ```mdl
 * MOVE MICROFLOW MyModule.MyMicroflow TO FOLDER 'Utils' IN OtherModule;
 * ```
 *
 * @example Move snippet to module root (no folder)
 * ```mdl
 * MOVE SNIPPET MyModule.MySnippet TO OtherModule;
 * ```
 *
 * @example Move entity to different module (no folder support)
 * ```mdl
 * MOVE ENTITY MyModule.Customer TO OtherModule;
 * ```
 *
 * @example Move enumeration to different module
 * ```mdl
 * MOVE ENUMERATION MyModule.OrderStatus TO OtherModule;
 * ```
 */
moveStatement
    : MOVE (PAGE | MICROFLOW | SNIPPET | NANOFLOW | ENUMERATION | CONSTANT | DATABASE CONNECTION) qualifiedName TO FOLDER STRING_LITERAL (IN (qualifiedName | IDENTIFIER))?
    | MOVE (PAGE | MICROFLOW | SNIPPET | NANOFLOW | ENUMERATION | CONSTANT | DATABASE CONNECTION) qualifiedName TO (qualifiedName | IDENTIFIER)
    | MOVE ENTITY qualifiedName TO (qualifiedName | IDENTIFIER)
    | MOVE FOLDER qualifiedName TO FOLDER STRING_LITERAL (IN (qualifiedName | IDENTIFIER))?
    | MOVE FOLDER qualifiedName TO (qualifiedName | IDENTIFIER)
    ;

// =============================================================================
// SECURITY STATEMENTS
// =============================================================================

securityStatement
    : createModuleRoleStatement
    | dropModuleRoleStatement
    | alterUserRoleStatement
    | dropUserRoleStatement
    | grantEntityAccessStatement
    | revokeEntityAccessStatement
    | grantMicroflowAccessStatement
    | revokeMicroflowAccessStatement
    | grantPageAccessStatement
    | revokePageAccessStatement
    | grantWorkflowAccessStatement
    | revokeWorkflowAccessStatement
    | grantODataServiceAccessStatement
    | revokeODataServiceAccessStatement
    | grantPublishedRestServiceAccessStatement
    | revokePublishedRestServiceAccessStatement
    | alterProjectSecurityStatement
    | dropDemoUserStatement
    | updateSecurityStatement
    ;

createModuleRoleStatement
    : CREATE MODULE ROLE qualifiedName (DESCRIPTION STRING_LITERAL)?
    ;

dropModuleRoleStatement
    : DROP MODULE ROLE qualifiedName
    ;

createUserRoleStatement
    : USER ROLE identifierOrKeyword
      LPAREN moduleRoleList RPAREN
      (MANAGE ALL ROLES)?
    ;

alterUserRoleStatement
    : ALTER USER ROLE identifierOrKeyword ADD MODULE ROLES LPAREN moduleRoleList RPAREN
    | ALTER USER ROLE identifierOrKeyword REMOVE MODULE ROLES LPAREN moduleRoleList RPAREN
    ;

dropUserRoleStatement
    : DROP USER ROLE identifierOrKeyword
    ;

grantEntityAccessStatement
    : GRANT moduleRoleList ON qualifiedName
      LPAREN entityAccessRightList RPAREN
      (WHERE STRING_LITERAL)?
    ;

revokeEntityAccessStatement
    : REVOKE moduleRoleList ON qualifiedName
      (LPAREN entityAccessRightList RPAREN)?
    ;

grantMicroflowAccessStatement
    : GRANT EXECUTE ON MICROFLOW qualifiedName TO moduleRoleList
    ;

revokeMicroflowAccessStatement
    : REVOKE EXECUTE ON MICROFLOW qualifiedName FROM moduleRoleList
    ;

grantPageAccessStatement
    : GRANT VIEW ON PAGE qualifiedName TO moduleRoleList
    ;

revokePageAccessStatement
    : REVOKE VIEW ON PAGE qualifiedName FROM moduleRoleList
    ;

grantWorkflowAccessStatement
    : GRANT EXECUTE ON WORKFLOW qualifiedName TO moduleRoleList
    ;

revokeWorkflowAccessStatement
    : REVOKE EXECUTE ON WORKFLOW qualifiedName FROM moduleRoleList
    ;

grantODataServiceAccessStatement
    : GRANT ACCESS ON ODATA SERVICE qualifiedName TO moduleRoleList
    ;

revokeODataServiceAccessStatement
    : REVOKE ACCESS ON ODATA SERVICE qualifiedName FROM moduleRoleList
    ;

grantPublishedRestServiceAccessStatement
    : GRANT ACCESS ON PUBLISHED REST SERVICE qualifiedName TO moduleRoleList
    ;

revokePublishedRestServiceAccessStatement
    : REVOKE ACCESS ON PUBLISHED REST SERVICE qualifiedName FROM moduleRoleList
    ;

alterProjectSecurityStatement
    : ALTER PROJECT SECURITY LEVEL (PRODUCTION | PROTOTYPE | OFF)
    | ALTER PROJECT SECURITY DEMO USERS (ON | OFF)
    ;

createDemoUserStatement
    : DEMO USER STRING_LITERAL PASSWORD STRING_LITERAL (ENTITY qualifiedName)?
      LPAREN identifierOrKeyword (COMMA identifierOrKeyword)* RPAREN
    ;

dropDemoUserStatement
    : DROP DEMO USER STRING_LITERAL
    ;

updateSecurityStatement
    : UPDATE SECURITY (IN qualifiedName)?
    ;

moduleRoleList
    : qualifiedName (COMMA qualifiedName)*
    ;

entityAccessRightList
    : entityAccessRight (COMMA entityAccessRight)*
    ;

entityAccessRight
    : CREATE
    | DELETE
    | READ STAR
    | READ LPAREN IDENTIFIER (COMMA IDENTIFIER)* RPAREN
    | WRITE STAR
    | WRITE LPAREN IDENTIFIER (COMMA IDENTIFIER)* RPAREN
    ;

// =============================================================================
// ENTITY / ASSOCIATION CREATION
// =============================================================================

/**
 * Creates a new entity in the domain model.
 *
 * Entities can be persistent (stored in database), non-persistent (in-memory only),
 * view (based on OQL query), or external (from external data source).
 *
 * @example Persistent entity with attributes
 * ```mdl
 * CREATE PERSISTENT ENTITY MyModule.Customer (
 *   Name: String(100) NOT NULL,
 *   Email: String(200) UNIQUE,
 *   Age: Integer,
 *   Active: Boolean DEFAULT true
 * );
 * ```
 *
 * @example Non-persistent entity for search filters
 * ```mdl
 * CREATE NON-PERSISTENT ENTITY MyModule.SearchFilter (
 *   Query: String,
 *   MaxResults: Integer DEFAULT 100,
 *   IncludeArchived: Boolean DEFAULT false
 * );
 * ```
 *
 * @example View entity with OQL query
 * ```mdl
 * CREATE VIEW ENTITY MyModule.ActiveCustomers (
 *   CustomerId: Integer,
 *   CustomerName: String(100)
 * ) AS
 *   SELECT c.Id AS CustomerId, c.Name AS CustomerName
 *   FROM MyModule.Customer AS c
 *   WHERE c.Active = true;
 * ```
 *
 * @example Entity with index
 * ```mdl
 * CREATE PERSISTENT ENTITY MyModule.Order (
 *   OrderNumber: String(50) NOT NULL,
 *   CustomerRef: MyModule.Customer
 * )
 * INDEX (OrderNumber);
 * ```
 *
 * @see attributeDefinition for attribute syntax
 * @see dataType for supported data types
 * @see oqlQuery for view entity queries
 */
createEntityStatement
    : PERSISTENT ENTITY qualifiedName generalizationClause? entityBody?
    | NON_PERSISTENT ENTITY qualifiedName generalizationClause? entityBody?
    | VIEW ENTITY qualifiedName entityBody? AS LPAREN? oqlQuery RPAREN?  // Parentheses optional
    | EXTERNAL ENTITY qualifiedName entityBody?
    | ENTITY qualifiedName generalizationClause? entityBody?  // Default to persistent
    ;

generalizationClause
    : EXTENDS qualifiedName
    | GENERALIZATION qualifiedName
    ;

entityBody
    : LPAREN attributeDefinitionList? RPAREN entityOptions?
    | entityOptions
    ;

entityOptions
    : entityOption (COMMA? entityOption)*  // Allow optional commas between options
    ;

entityOption
    : COMMENT STRING_LITERAL
    | INDEX indexDefinition
    | eventHandlerDefinition
    ;

// Entity event handler: ON BEFORE/AFTER CREATE/COMMIT/DELETE/ROLLBACK CALL Mod.Microflow($currentObject) [RAISE ERROR]
// Empty parens () = don't pass object, ($currentObject) = pass object, no parens = pass object (default)
eventHandlerDefinition
    : ON eventMoment eventType CALL qualifiedName (LPAREN VARIABLE? RPAREN)? (RAISE ERROR)?
    ;

eventMoment
    : BEFORE | AFTER
    ;

eventType
    : CREATE | COMMIT | DELETE | ROLLBACK
    ;

attributeDefinitionList
    : attributeDefinition (COMMA attributeDefinition)*
    ;

/**
 * Defines an attribute within an entity.
 *
 * Attributes have a name, data type, and optional constraints like NOT NULL, UNIQUE, or DEFAULT.
 * Documentation comments can be added above the attribute.
 *
 * @example Simple attributes
 * ```mdl
 * Name: String(100),
 * Age: Integer,
 * Active: Boolean
 * ```
 *
 * @example Attributes with constraints
 * ```mdl
 * Code: String(50) NOT NULL,
 * Email: String(200) UNIQUE,
 * Status: Enum MyModule.Status DEFAULT MyModule.Status.Active
 * ```
 *
 * @example Attribute with custom error messages
 * ```mdl
 * Name: String(100) NOT NULL ERROR 'Name is required',
 * Code: String(50) UNIQUE ERROR 'Code must be unique'
 * ```
 *
 * @example Documented attribute
 * ```mdl
 * -- The customer's primary email address
 * Email: String(200) NOT NULL UNIQUE
 * ```
 *
 * @see dataType for available types
 * @see attributeConstraint for constraint options
 */
attributeDefinition
    : docComment? annotation* attributeName COLON dataType attributeConstraint*
    ;

// Allow reserved keywords as attribute names
attributeName
    : IDENTIFIER
    | QUOTED_IDENTIFIER                     // Escape any reserved word ("Range", `Order`)
    | keyword
    ;

attributeConstraint
    : NOT_NULL (ERROR STRING_LITERAL)?
    | NOT NULL (ERROR STRING_LITERAL)?
    | UNIQUE (ERROR STRING_LITERAL)?
    | DEFAULT (literal | expression)
    | REQUIRED (ERROR STRING_LITERAL)?
    | CALCULATED (BY? qualifiedName)?
    ;

/**
 * Specifies the data type for an attribute.
 *
 * MDL supports all Mendix primitive types, enumerations, and entity references.
 *
 * @example Primitive types
 * ```mdl
 * Name: String(200),       -- String with max length 200
 * Age: Integer,            -- 32-bit integer
 * Total: Decimal,          -- Fixed-point decimal
 * Active: Boolean,         -- true/false
 * Created: DateTime,       -- Date and time
 * BirthDate: Date,         -- Date only
 * Counter: AutoNumber,     -- Auto-incrementing number
 * Data: Binary,            -- Binary data (files)
 * Password: HashedString   -- Securely hashed string
 * ```
 *
 * @example Enumeration types
 * ```mdl
 * Status: Enum MyModule.OrderStatus,
 * Priority: Enumeration(MyModule.Priority)
 * ```
 *
 * @example Entity references
 * ```mdl
 * Customer: MyModule.Customer,       -- Single reference
 * Items: List of MyModule.OrderItem  -- List of references
 * ```
 */
dataType
    : STRING_TYPE (LPAREN (NUMBER_LITERAL | IDENTIFIER) RPAREN)?
    | INTEGER_TYPE
    | LONG_TYPE
    | DECIMAL_TYPE
    | BOOLEAN_TYPE
    | DATETIME_TYPE
    | DATE_TYPE
    | AUTONUMBER_TYPE
    | AUTOOWNER_TYPE
    | AUTOCHANGEDBY_TYPE
    | AUTOCREATEDDATE_TYPE
    | AUTOCHANGEDDATE_TYPE
    | BINARY_TYPE
    | HASHEDSTRING_TYPE
    | CURRENCY_TYPE
    | FLOAT_TYPE
    | STRINGTEMPLATE_TYPE LPAREN templateContext RPAREN  // StringTemplate(Sql) etc.
    | ENTITY LESS_THAN IDENTIFIER GREATER_THAN         // ENTITY <pEntity> type parameter declaration
    | ENUM_TYPE qualifiedName
    | ENUMERATION LPAREN qualifiedName RPAREN  // Enumeration(Module.Enum) syntax
    | LIST_OF qualifiedName
    | qualifiedName  // Entity reference type
    ;

// Template context for StringTemplate types - only SQL or Text are valid
templateContext
    : SQL
    | TEXT
    ;

// Non-list data type - used for createObjectStatement to avoid matching "CREATE LIST OF"
nonListDataType
    : STRING_TYPE (LPAREN (NUMBER_LITERAL | IDENTIFIER) RPAREN)?
    | INTEGER_TYPE
    | LONG_TYPE
    | DECIMAL_TYPE
    | BOOLEAN_TYPE
    | DATETIME_TYPE
    | DATE_TYPE
    | AUTONUMBER_TYPE
    | AUTOOWNER_TYPE
    | AUTOCHANGEDBY_TYPE
    | AUTOCREATEDDATE_TYPE
    | AUTOCHANGEDDATE_TYPE
    | BINARY_TYPE
    | HASHEDSTRING_TYPE
    | CURRENCY_TYPE
    | FLOAT_TYPE
    | ENUM_TYPE qualifiedName
    | ENUMERATION LPAREN qualifiedName RPAREN
    | qualifiedName  // Entity reference type (NOT list)
    ;

indexDefinition
    : IDENTIFIER? LPAREN indexAttributeList RPAREN
    ;

indexAttributeList
    : indexAttribute (COMMA indexAttribute)*
    ;

indexAttribute
    : indexColumnName (ASC | DESC)?  // Column name with optional sort order
    ;

// Allow keywords as index column names (same as attributeName)
indexColumnName
    : IDENTIFIER
    | QUOTED_IDENTIFIER                     // Escape any reserved word
    | keyword
    ;

createAssociationStatement
    : ASSOCIATION qualifiedName
      FROM qualifiedName
      TO qualifiedName
      associationOptions?
    | ASSOCIATION qualifiedName LPAREN
      FROM qualifiedName TO qualifiedName
      (COMMA associationOption)*
      RPAREN
    ;

associationOptions
    : associationOption+
    ;

associationOption
    : TYPE COLON? (REFERENCE | REFERENCE_SET)
    | OWNER COLON? (DEFAULT | BOTH)
    | STORAGE COLON? (COLUMN | TABLE)
    | DELETE_BEHAVIOR deleteBehavior
    | COMMENT STRING_LITERAL
    ;

deleteBehavior
    : DELETE_AND_REFERENCES
    | DELETE_BUT_KEEP_REFERENCES
    | DELETE_IF_NO_REFERENCES
    | CASCADE
    | PREVENT
    ;

// =============================================================================
// ALTER ENTITY ACTIONS
// =============================================================================

alterEntityAction
    : ADD ATTRIBUTE attributeDefinition
    | ADD COLUMN attributeDefinition
    | RENAME ATTRIBUTE attributeName TO attributeName
    | RENAME COLUMN attributeName TO attributeName
    | MODIFY ATTRIBUTE attributeName COLON? dataType attributeConstraint*
    | MODIFY COLUMN attributeName COLON? dataType attributeConstraint*
    | DROP ATTRIBUTE attributeName
    | DROP COLUMN attributeName
    | SET DOCUMENTATION STRING_LITERAL
    | SET COMMENT STRING_LITERAL
    | SET POSITION LPAREN NUMBER_LITERAL COMMA NUMBER_LITERAL RPAREN
    | ADD INDEX indexDefinition
    | DROP INDEX IDENTIFIER
    | ADD EVENT HANDLER eventHandlerDefinition
    | DROP EVENT HANDLER ON eventMoment eventType
    ;

alterAssociationAction
    : SET DELETE_BEHAVIOR deleteBehavior
    | SET OWNER (DEFAULT | BOTH)
    | SET STORAGE (COLUMN | TABLE)
    | SET COMMENT STRING_LITERAL
    ;

alterEnumerationAction
    : ADD VALUE IDENTIFIER (CAPTION STRING_LITERAL)?
    | RENAME VALUE IDENTIFIER TO IDENTIFIER
    | DROP VALUE IDENTIFIER
    | SET COMMENT STRING_LITERAL
    ;

alterNotebookAction
    : ADD PAGE qualifiedName (POSITION NUMBER_LITERAL)?
    | DROP PAGE qualifiedName
    | SET COMMENT STRING_LITERAL
    ;

// =============================================================================
// MODULE CREATION
// =============================================================================

createModuleStatement
    : MODULE IDENTIFIER moduleOptions?
    ;

moduleOptions
    : moduleOption+
    ;

moduleOption
    : COMMENT STRING_LITERAL
    | FOLDER STRING_LITERAL
    ;

// =============================================================================
// ENUMERATION CREATION
// =============================================================================

createEnumerationStatement
    : ENUMERATION qualifiedName
      LPAREN enumerationValueList RPAREN
      enumerationOptions?
    ;

enumerationValueList
    : enumerationValue (COMMA enumerationValue)*
    ;

enumerationValue
    : docComment? enumValueName (CAPTION? STRING_LITERAL)?
    ;

// Allow reserved keywords as enumeration value names.
// Uses the full `keyword` rule so any lexer token can appear as an enum value name.
enumValueName
    : IDENTIFIER
    | QUOTED_IDENTIFIER                                      // Escape any reserved word
    | keyword
    ;

enumerationOptions
    : enumerationOption+
    ;

enumerationOption
    : COMMENT STRING_LITERAL
    ;

// =============================================================================
// IMAGE COLLECTION CREATION
// =============================================================================

createImageCollectionStatement
    : IMAGE COLLECTION qualifiedName imageCollectionOptions? imageCollectionBody?
    ;

imageCollectionOptions
    : imageCollectionOption+
    ;

imageCollectionOption
    : EXPORT LEVEL STRING_LITERAL   // e.g. EXPORT LEVEL 'Public'
    | COMMENT STRING_LITERAL
    ;

imageCollectionBody
    : LPAREN imageCollectionItem (COMMA imageCollectionItem)* RPAREN
    ;

imageCollectionItem
    : IMAGE imageName FROM FILE_KW path=STRING_LITERAL   // IMAGE MyIcon FROM FILE '/path/to/file.png'
    ;

imageName
    : IDENTIFIER
    | QUOTED_IDENTIFIER
    | keyword
    ;

// =============================================================================
// JSON STRUCTURE CREATION
// =============================================================================

createJsonStructureStatement
    : JSON STRUCTURE qualifiedName (FOLDER STRING_LITERAL)? (COMMENT STRING_LITERAL)? SNIPPET (STRING_LITERAL | DOLLAR_STRING)
      (CUSTOM_NAME_MAP LPAREN customNameMapping (COMMA customNameMapping)* RPAREN)?
    ;

customNameMapping
    : STRING_LITERAL AS STRING_LITERAL   // 'jsonKey' AS 'CustomName'
    ;

/**
 * CREATE IMPORT MAPPING Module.Name
 *   WITH JSON STRUCTURE Module.JsonStructure
 * {
 *   CREATE Module.Entity {
 *     PetId = id KEY,
 *     Name = name,
 *     CREATE Module.Assoc/Module.Child = jsonKey {
 *       Email = email
 *     }
 *   }
 * };
 */
createImportMappingStatement
    : IMPORT MAPPING qualifiedName
      importMappingWithClause?
      LBRACE importMappingRootElement RBRACE
    ;

importMappingWithClause
    : WITH JSON STRUCTURE qualifiedName
    | WITH XML SCHEMA qualifiedName
    ;

importMappingRootElement
    : importMappingObjectHandling qualifiedName
      LBRACE importMappingChild (COMMA importMappingChild)* RBRACE
    ;

importMappingChild
    : importMappingObjectHandling qualifiedName SLASH qualifiedName EQUALS identifierOrKeyword
      LBRACE importMappingChild (COMMA importMappingChild)* RBRACE       // nested object with children
    | importMappingObjectHandling qualifiedName SLASH qualifiedName EQUALS identifierOrKeyword  // leaf object
    | identifierOrKeyword EQUALS qualifiedName LPAREN identifierOrKeyword RPAREN  // value transform: Attr = Module.MF(jsonField)
    | identifierOrKeyword EQUALS identifierOrKeyword KEY?                         // value: Attr = jsonField [KEY]
    ;

importMappingObjectHandling
    : CREATE
    | FIND
    | FIND OR CREATE
    ;

/**
 * CREATE EXPORT MAPPING Module.Name
 *   WITH JSON STRUCTURE Module.JsonStructure
 *   [NULL VALUES LeaveOutElement]
 * {
 *   Module.Entity {
 *     jsonField = Attr,
 *     Module.Assoc/Module.Child AS jsonKey {
 *       email = Email
 *     }
 *   }
 * };
 */
createExportMappingStatement
    : EXPORT MAPPING qualifiedName
      exportMappingWithClause?
      exportMappingNullValuesClause?
      LBRACE exportMappingRootElement RBRACE
    ;

exportMappingWithClause
    : WITH JSON STRUCTURE qualifiedName
    | WITH XML SCHEMA qualifiedName
    ;

exportMappingNullValuesClause
    : NULL VALUES identifierOrKeyword
    ;

exportMappingRootElement
    : qualifiedName
      LBRACE exportMappingChild (COMMA exportMappingChild)* RBRACE
    ;

exportMappingChild
    : qualifiedName SLASH qualifiedName AS identifierOrKeyword
      LBRACE exportMappingChild (COMMA exportMappingChild)* RBRACE       // nested object with children
    | qualifiedName SLASH qualifiedName AS identifierOrKeyword            // leaf object
    | identifierOrKeyword EQUALS identifierOrKeyword                      // value: jsonField = Attr
    ;

// =============================================================================
// VALIDATION RULE CREATION
// =============================================================================

createValidationRuleStatement
    : VALIDATION RULE qualifiedName
      FOR qualifiedName
      validationRuleBody
    ;

validationRuleBody
    : EXPRESSION expression FEEDBACK STRING_LITERAL
    | REQUIRED attributeReference FEEDBACK STRING_LITERAL
    | UNIQUE attributeReferenceList FEEDBACK STRING_LITERAL
    | RANGE attributeReference rangeConstraint FEEDBACK STRING_LITERAL
    | REGEX attributeReference STRING_LITERAL FEEDBACK STRING_LITERAL
    ;

rangeConstraint
    : BETWEEN literal AND literal
    | LESS_THAN literal
    | LESS_THAN_OR_EQUAL literal
    | GREATER_THAN literal
    | GREATER_THAN_OR_EQUAL literal
    ;

attributeReference
    : IDENTIFIER (SLASH IDENTIFIER)*
    ;

attributeReferenceList
    : attributeReference (COMMA attributeReference)*
    ;

// =============================================================================
// MICROFLOW CREATION
// =============================================================================

/**
 * Creates a new microflow with parameters, return type, and activity body.
 *
 * Microflows are server-side logic that can include database operations,
 * integrations, and complex business rules.
 *
 * @example Simple microflow returning a string
 * ```mdl
 * CREATE MICROFLOW MyModule.GetGreeting ($Name: String) RETURNS String
 * BEGIN
 *   RETURN 'Hello, ' + $Name + '!';
 * END;
 * ```
 *
 * @example Microflow with entity parameter and database operations
 * ```mdl
 * CREATE MICROFLOW MyModule.DeactivateCustomer ($Customer: MyModule.Customer) RETURNS Boolean
 * BEGIN
 *   $Customer.Active = false;
 *   COMMIT $Customer;
 *   RETURN true;
 * END;
 * ```
 *
 * @example Microflow with RETRIEVE and iteration
 * ```mdl
 * CREATE MICROFLOW MyModule.CountActiveOrders () RETURNS Integer
 * BEGIN
 *   DECLARE $Orders List of MyModule.Order;
 *   $Orders = RETRIEVE MyModule.Order WHERE Active = true;
 *   RETURN length($Orders);
 * END;
 * ```
 *
 * @example Microflow calling another microflow
 * ```mdl
 * CREATE MICROFLOW MyModule.ProcessOrder ($Order: MyModule.Order) RETURNS Boolean
 * BEGIN
 *   $Result = CALL MICROFLOW MyModule.ValidateOrder (Order = $Order);
 *   IF $Result THEN
 *     COMMIT $Order;
 *     RETURN true;
 *   END IF;
 *   RETURN false;
 * END;
 * ```
 *
 * @see microflowBody for available activities
 * @see microflowParameter for parameter syntax
 */
createMicroflowStatement
    : MICROFLOW qualifiedName
      LPAREN microflowParameterList? RPAREN
      microflowReturnType?
      microflowOptions?
      BEGIN microflowBody END SEMICOLON? SLASH?
    ;

/**
 * Java Action creation with inline Java source code.
 *
 * @example Basic Java action
 * ```mdl
 * CREATE JAVA ACTION MyModule.ValidateEmail(EmailAddress: String NOT NULL)
 * RETURNS Boolean
 * AS $$
 * String pattern = "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$";
 * return java.util.regex.Pattern.matches(pattern, EmailAddress);
 * $$;
 * ```
 *
 * @example Java action with multiple parameters
 * ```mdl
 * CREATE JAVA ACTION MyModule.CalculateHash(Input: String NOT NULL, Algorithm: String)
 * RETURNS String
 * AS $$
 * java.security.MessageDigest md = java.security.MessageDigest.getInstance(
 *     Algorithm != null ? Algorithm : "SHA-256");
 * byte[] hash = md.digest(Input.getBytes("UTF-8"));
 * return java.util.Base64.getEncoder().encodeToString(hash);
 * $$;
 * ```
 */
createJavaActionStatement
    : JAVA ACTION qualifiedName
      LPAREN javaActionParameterList? RPAREN
      javaActionReturnType?
      javaActionExposedClause?
      AS DOLLAR_STRING SEMICOLON?
    ;

javaActionParameterList
    : javaActionParameter (COMMA javaActionParameter)*
    ;

javaActionParameter
    : parameterName COLON dataType NOT_NULL?
    ;

javaActionReturnType
    : RETURNS dataType
    ;

javaActionExposedClause
    : EXPOSED AS STRING_LITERAL IN STRING_LITERAL
    ;

microflowParameterList
    : microflowParameter (COMMA microflowParameter)*
    ;

microflowParameter
    : (parameterName | VARIABLE) COLON dataType
    ;

// Allow reserved keywords as parameter names (similar to attributeName)
parameterName
    : IDENTIFIER
    | QUOTED_IDENTIFIER                            // Escape any reserved word
    | keyword
    ;

microflowReturnType
    : RETURNS dataType (AS VARIABLE)?
    ;

microflowOptions
    : microflowOption+
    ;

microflowOption
    : FOLDER STRING_LITERAL
    | COMMENT STRING_LITERAL
    ;

microflowBody
    : microflowStatement*
    ;

microflowStatement
    : annotation* declareStatement SEMICOLON?
    | annotation* setStatement SEMICOLON?
    | annotation* createListStatement SEMICOLON?       // Must be before createObjectStatement to match "CREATE LIST OF"
    | annotation* createObjectStatement SEMICOLON?
    | annotation* changeObjectStatement SEMICOLON?
    | annotation* commitStatement SEMICOLON?
    | annotation* deleteObjectStatement SEMICOLON?
    | annotation* rollbackStatement SEMICOLON?
    | annotation* retrieveStatement SEMICOLON?
    | annotation* ifStatement SEMICOLON?
    | annotation* loopStatement SEMICOLON?
    | annotation* whileStatement SEMICOLON?
    | annotation* continueStatement SEMICOLON?
    | annotation* breakStatement SEMICOLON?
    | annotation* returnStatement SEMICOLON?
    | annotation* raiseErrorStatement SEMICOLON?
    | annotation* logStatement SEMICOLON?
    | annotation* callMicroflowStatement SEMICOLON?
    | annotation* callJavaActionStatement SEMICOLON?
    | annotation* executeDatabaseQueryStatement SEMICOLON?
    | annotation* callExternalActionStatement SEMICOLON?
    | annotation* showPageStatement SEMICOLON?
    | annotation* closePageStatement SEMICOLON?
    | annotation* showHomePageStatement SEMICOLON?
    | annotation* showMessageStatement SEMICOLON?
    | annotation* throwStatement SEMICOLON?
    | annotation* listOperationStatement SEMICOLON?
    | annotation* aggregateListStatement SEMICOLON?
    | annotation* addToListStatement SEMICOLON?
    | annotation* removeFromListStatement SEMICOLON?
    | annotation* validationFeedbackStatement SEMICOLON?
    | annotation* restCallStatement SEMICOLON?
    | annotation* sendRestRequestStatement SEMICOLON?
    | annotation* importFromMappingStatement SEMICOLON?
    | annotation* exportToMappingStatement SEMICOLON?
    | annotation* callWorkflowStatement SEMICOLON?
    | annotation* getWorkflowDataStatement SEMICOLON?
    | annotation* getWorkflowsStatement SEMICOLON?
    | annotation* getWorkflowActivityRecordsStatement SEMICOLON?
    | annotation* workflowOperationStatement SEMICOLON?
    | annotation* setTaskOutcomeStatement SEMICOLON?
    | annotation* openUserTaskStatement SEMICOLON?
    | annotation* notifyWorkflowStatement SEMICOLON?
    | annotation* openWorkflowStatement SEMICOLON?
    | annotation* lockWorkflowStatement SEMICOLON?
    | annotation* unlockWorkflowStatement SEMICOLON?
    ;

declareStatement
    : DECLARE VARIABLE dataType (EQUALS expression)?
    ;

setStatement
    : SET (VARIABLE | attributePath) EQUALS expression
    ;

// $NewProduct = CREATE MfTest.Product (Name = $Name, Code = $Code);
// Aligned with CALL MICROFLOW/JAVA ACTION syntax
// Uses nonListDataType to avoid matching "CREATE LIST OF Entity" which should be createListStatement
createObjectStatement
    : (VARIABLE EQUALS)? CREATE nonListDataType (LPAREN memberAssignmentList? RPAREN)? onErrorClause?
    ;

// CHANGE $Product (Name = $NewName, ModifiedDate = [%CurrentDateTime%]);
// Aligned with CREATE syntax
changeObjectStatement
    : CHANGE VARIABLE (LPAREN memberAssignmentList? RPAREN)?
    ;

attributePath
    : VARIABLE ((SLASH | DOT) (IDENTIFIER | qualifiedName))+
    ;

// COMMIT $Product; or COMMIT $Product WITH EVENTS; or COMMIT $Product REFRESH;
commitStatement
    : COMMIT VARIABLE (WITH EVENTS)? REFRESH? onErrorClause?
    ;

deleteObjectStatement
    : DELETE VARIABLE onErrorClause?
    ;

// ROLLBACK $Product; or ROLLBACK $Product REFRESH;
rollbackStatement
    : ROLLBACK VARIABLE REFRESH?
    ;

// RETRIEVE $ProductList FROM MfTest.Product WHERE Code = $SearchCode SORT BY Name ASC LIMIT 1;
retrieveStatement
    : RETRIEVE VARIABLE FROM retrieveSource
      (WHERE (xpathConstraint (andOrXpath? xpathConstraint)* | expression))?
      (SORT_BY sortColumn (COMMA sortColumn)*)?
      (LIMIT limitExpr=expression)?
      (OFFSET offsetExpr=expression)?
      onErrorClause?
    ;

retrieveSource
    : qualifiedName                          // Database retrieve: Module.Entity
    | VARIABLE SLASH qualifiedName           // Association retrieve: $Parent/Module.Assoc
    | LPAREN oqlQuery RPAREN                 // OQL retrieve
    | DATABASE STRING_LITERAL                // External DB
    ;

// ON ERROR clause for microflow error handling
onErrorClause
    : ON ERROR CONTINUE                                    // Ignore error, continue
    | ON ERROR ROLLBACK                                    // Rollback and abort (default)
    | ON ERROR LBRACE microflowBody RBRACE                 // Custom error handler with rollback
    | ON ERROR WITHOUT ROLLBACK LBRACE microflowBody RBRACE // Custom error handler without rollback
    ;

// IF ... THEN ... END IF;
ifStatement
    : IF expression THEN microflowBody
      (ELSIF expression THEN microflowBody)*
      (ELSE microflowBody)?
      END IF
    ;

// LOOP $Product IN $ProductList BEGIN ... END LOOP;
loopStatement
    : LOOP VARIABLE IN (VARIABLE | attributePath)
      BEGIN microflowBody END LOOP
    ;

whileStatement
    : WHILE expression
      BEGIN? microflowBody END WHILE?
    ;

continueStatement
    : CONTINUE
    ;

breakStatement
    : BREAK
    ;

returnStatement
    : RETURN expression?
    ;

raiseErrorStatement
    : RAISE ERROR
    ;

// LOG INFO NODE 'TEST' 'Message'; or LOG INFO 'Message'; or LOG WARNING 'Message' WITH ({1} = $var);
logStatement
    : LOG logLevel? (NODE STRING_LITERAL)? expression logTemplateParams?
    ;

logLevel
    : INFO
    | WARNING
    | ERROR
    | DEBUG
    | TRACE
    | CRITICAL
    ;

// Template parameters: WITH ({1} = expr, {2} = expr) or PARAMETERS [expr, expr]
// Used by LOG statements (microflows) and CONTENT/captions (pages)
templateParams
    : WITH LPAREN templateParam (COMMA templateParam)* RPAREN    // WITH ({1} = $var)
    | PARAMETERS arrayLiteral                                     // PARAMETERS ['val'] (deprecated)
    ;

templateParam
    : LBRACE NUMBER_LITERAL RBRACE EQUALS expression
    ;

// Backward compatibility aliases
logTemplateParams: templateParams;
logTemplateParam: templateParam;

// $Result = CALL MICROFLOW MfTest.M001_HelloWorld(); or CALL MICROFLOW MfTest.M001_HelloWorld();
callMicroflowStatement
    : (VARIABLE EQUALS)? CALL MICROFLOW qualifiedName LPAREN callArgumentList? RPAREN onErrorClause?
    ;

// $Result = CALL JAVA ACTION CustomActivities.ExecuteOQL(OqlStatement = '...');
callJavaActionStatement
    : (VARIABLE EQUALS)? CALL JAVA ACTION qualifiedName LPAREN callArgumentList? RPAREN onErrorClause?
    ;

// $Result = EXECUTE DATABASE QUERY Module.Connection.QueryName (param = 'value');
// $Result = EXECUTE DATABASE QUERY Module.Connection.QueryName DYNAMIC 'SELECT ...';
// $Result = EXECUTE DATABASE QUERY Module.Connection.QueryName CONNECTION (DBSource = $Url, DBUsername = $User, DBPassword = $Pass);
executeDatabaseQueryStatement
    : (VARIABLE EQUALS)? EXECUTE DATABASE QUERY qualifiedName
      (DYNAMIC (STRING_LITERAL | DOLLAR_STRING | expression))?
      (LPAREN callArgumentList? RPAREN)?
      (CONNECTION LPAREN callArgumentList? RPAREN)?
      onErrorClause?
    ;

// $Result = CALL EXTERNAL ACTION Module.ODataClient.ActionName(Param = $value);
// qualifiedName matches Module.ServiceName.ActionName; visitor splits off the last segment as action name
callExternalActionStatement
    : (VARIABLE EQUALS)? CALL EXTERNAL ACTION qualifiedName LPAREN callArgumentList? RPAREN onErrorClause?
    ;

// ============================================================================
// Workflow microflow actions
// ============================================================================

// $Wf = CALL WORKFLOW Module.WF_Name ($ContextObj);
callWorkflowStatement
    : (VARIABLE EQUALS)? CALL WORKFLOW qualifiedName LPAREN callArgumentList? RPAREN onErrorClause?
    ;

// $Data = GET WORKFLOW DATA $WorkflowVar AS Module.WorkflowName;
getWorkflowDataStatement
    : (VARIABLE EQUALS)? GET WORKFLOW DATA VARIABLE AS qualifiedName onErrorClause?
    ;

// $Wfs = GET WORKFLOWS FOR $ContextObj;
getWorkflowsStatement
    : (VARIABLE EQUALS)? GET WORKFLOWS FOR VARIABLE onErrorClause?
    ;

// $Records = GET WORKFLOW ACTIVITY RECORDS $WorkflowVar;
getWorkflowActivityRecordsStatement
    : (VARIABLE EQUALS)? GET WORKFLOW ACTIVITY RECORDS VARIABLE onErrorClause?
    ;

// WORKFLOW OPERATION ABORT $Wf REASON 'text';
// WORKFLOW OPERATION PAUSE $Wf;
workflowOperationStatement
    : WORKFLOW OPERATION workflowOperationType onErrorClause?
    ;

workflowOperationType
    : ABORT VARIABLE (REASON expression)?
    | CONTINUE VARIABLE
    | PAUSE VARIABLE
    | RESTART VARIABLE
    | RETRY VARIABLE
    | UNPAUSE VARIABLE
    ;

// SET TASK OUTCOME $UserTask 'OutcomeName';
setTaskOutcomeStatement
    : SET TASK OUTCOME VARIABLE STRING_LITERAL onErrorClause?
    ;

// OPEN USER TASK $UserTask;
openUserTaskStatement
    : OPEN USER TASK VARIABLE onErrorClause?
    ;

// NOTIFY WORKFLOW $Wf;
notifyWorkflowStatement
    : (VARIABLE EQUALS)? NOTIFY WORKFLOW VARIABLE onErrorClause?
    ;

// OPEN WORKFLOW $Wf;
openWorkflowStatement
    : OPEN WORKFLOW VARIABLE onErrorClause?
    ;

// LOCK WORKFLOW $Wf; or LOCK WORKFLOW ALL;
lockWorkflowStatement
    : LOCK WORKFLOW (VARIABLE | ALL) onErrorClause?
    ;

// UNLOCK WORKFLOW $Wf; or UNLOCK WORKFLOW ALL;
unlockWorkflowStatement
    : UNLOCK WORKFLOW (VARIABLE | ALL) onErrorClause?
    ;

callArgumentList
    : callArgument (COMMA callArgument)*
    ;

// Named arguments: $FirstName = 'Hello' or Level = 'INFO' or OqlStatement = '...'
callArgument
    : (VARIABLE | parameterName) EQUALS expression
    ;

showPageStatement
    : SHOW PAGE qualifiedName (LPAREN showPageArgList? RPAREN)? (FOR VARIABLE)? (WITH memberAssignmentList)?
    ;

showPageArgList
    : showPageArg (COMMA showPageArg)*
    ;

showPageArg
    : VARIABLE EQUALS (VARIABLE | expression)       // $Param = $value (canonical)
    | identifierOrKeyword COLON expression           // Param: $value (widget-style, also accepted)
    ;

closePageStatement
    : CLOSE PAGE
    ;

showHomePageStatement
    : SHOW HOME PAGE
    ;

// SHOW MESSAGE 'Hello {1}' TYPE Information OBJECTS [$Name];
showMessageStatement
    : SHOW MESSAGE expression (TYPE identifierOrKeyword)? (OBJECTS LBRACKET expressionList RBRACKET)?
    ;

throwStatement
    : THROW expression
    ;

// VALIDATION FEEDBACK $Product/Code MESSAGE 'Product code cannot be empty';
// VALIDATION FEEDBACK $Product/Code MESSAGE '{1}' OBJECTS [$Var1, $Var2];
validationFeedbackStatement
    : VALIDATION FEEDBACK attributePath MESSAGE expression (OBJECTS LBRACKET expressionList RBRACKET)?
    ;

// =============================================================================
// REST CALL STATEMENTS
// =============================================================================

/**
 * REST call statement for making HTTP requests to external APIs.
 *
 * @example Simple GET request returning string
 * ```mdl
 * $Response = REST CALL GET 'https://api.example.com/data'
 *     HEADER Accept = 'application/json'
 *     TIMEOUT 30
 *     RETURNS String;
 * ```
 *
 * @example POST with JSON body
 * ```mdl
 * $Response = REST CALL POST 'https://api.example.com/items'
 *     HEADER Content-Type = 'application/json'
 *     BODY '{"name": "{1}"}' WITH ({1} = $Name)
 *     TIMEOUT 30
 *     RETURNS String
 *     ON ERROR ROLLBACK;
 * ```
 *
 * @example GET with URL parameters
 * ```mdl
 * $Data = REST CALL GET 'https://api.example.com/search?q={1}' WITH ({1} = $Query)
 *     RETURNS MAPPING MyModule.ImportData AS MyModule.DataEntity;
 * ```
 *
 * @example POST with export mapping
 * ```mdl
 * $Result = REST CALL POST 'https://api.example.com/submit'
 *     BODY MAPPING MyModule.ExportMapping FROM $Entity
 *     RETURNS String;
 * ```
 */
restCallStatement
    : (VARIABLE EQUALS)? REST CALL httpMethod restCallUrl restCallUrlParams?
      restCallHeaderClause*
      restCallAuthClause?
      restCallBodyClause?
      restCallTimeoutClause?
      restCallReturnsClause
      onErrorClause?
    ;

httpMethod
    : GET
    | POST
    | PUT
    | PATCH
    | DELETE
    ;

// URL can be a string literal or expression
restCallUrl
    : STRING_LITERAL
    | expression
    ;

// URL template parameters: WITH ({1} = expr, {2} = expr)
restCallUrlParams
    : templateParams
    ;

// HEADER name = 'value' or HEADER 'Content-Type' = 'value'
restCallHeaderClause
    : HEADER (IDENTIFIER | STRING_LITERAL) EQUALS expression
    ;

// AUTH BASIC $user PASSWORD $pass
restCallAuthClause
    : AUTH BASIC expression PASSWORD expression
    ;

// BODY 'template' [WITH params] or BODY MAPPING Name FROM $var
restCallBodyClause
    : BODY STRING_LITERAL templateParams?                    // Custom body template
    | BODY expression templateParams?                        // Expression body
    | BODY MAPPING qualifiedName FROM VARIABLE               // Export mapping
    ;

// TIMEOUT expression (in seconds)
restCallTimeoutClause
    : TIMEOUT expression
    ;

// RETURNS clause specifies how to handle the response
restCallReturnsClause
    : RETURNS STRING_TYPE                                    // Return as string
    | RETURNS RESPONSE                                       // Return HttpResponse object
    | RETURNS MAPPING qualifiedName AS qualifiedName         // Import mapping with result entity
    | RETURNS NONE                                           // Ignore response
    | RETURNS NOTHING                                        // Ignore response (alias)
    ;

/**
 * SEND REST REQUEST — calls a consumed REST service operation defined via CREATE REST CLIENT.
 *
 * @example Simple call (no response)
 * ```mdl
 * SEND REST REQUEST RestTest.RC001_SimpleAPI.GetStatus;
 * ```
 *
 * @example Call with output variable
 * ```mdl
 * $RootResult = SEND REST REQUEST RestTest.RC008_HeaderAPI.GetData;
 * ```
 *
 * @example Call with body and error handling
 * ```mdl
 * $Result = SEND REST REQUEST Module.Service.CreateItem
 *     BODY $NewItem
 *     ON ERROR CONTINUE;
 * ```
 */
sendRestRequestStatement
    : (VARIABLE EQUALS)? SEND REST REQUEST qualifiedName
      sendRestRequestBodyClause?
      onErrorClause?
    ;

sendRestRequestBodyClause
    : BODY VARIABLE
    ;

/**
 * Import from mapping: [$Var =] IMPORT FROM MAPPING Module.IMM($SourceVar);
 */
importFromMappingStatement
    : (VARIABLE EQUALS)? IMPORT FROM MAPPING qualifiedName LPAREN VARIABLE RPAREN
      onErrorClause?
    ;

/**
 * Export to mapping: $Var = EXPORT TO MAPPING Module.EMM($SourceVar);
 */
exportToMappingStatement
    : (VARIABLE EQUALS)? EXPORT TO MAPPING qualifiedName LPAREN VARIABLE RPAREN
      onErrorClause?
    ;

// =============================================================================
// LIST OPERATIONS
// =============================================================================

/**
 * List operations that return a single item or a modified list.
 *
 * @example Get first item
 * ```mdl
 * $FirstProduct = HEAD($ProductList);
 * ```
 *
 * @example Get all but first item
 * ```mdl
 * $RestOfProducts = TAIL($ProductList);
 * ```
 *
 * @example Find first matching item
 * ```mdl
 * $ExpensiveProduct = FIND($Products, Price > 100);
 * ```
 *
 * @example Filter list
 * ```mdl
 * $ActiveProducts = FILTER($Products, Active = true);
 * ```
 *
 * @example Sort list
 * ```mdl
 * $SortedProducts = SORT($Products, Name ASC);
 * ```
 *
 * @example Set operations
 * ```mdl
 * $Combined = UNION($List1, $List2);
 * $Common = INTERSECT($List1, $List2);
 * $Difference = SUBTRACT($List1, $List2);
 * ```
 *
 * @example Boolean list operations
 * ```mdl
 * $HasItem = CONTAINS($List, $Item);
 * $AreEqual = EQUALS($List1, $List2);
 * ```
 */
listOperationStatement
    : VARIABLE EQUALS listOperation
    ;

listOperation
    : HEAD LPAREN VARIABLE RPAREN                                      // $var = HEAD($list)
    | TAIL LPAREN VARIABLE RPAREN                                      // $var = TAIL($list)
    | FIND LPAREN VARIABLE COMMA expression RPAREN                     // $var = FIND($list, condition)
    | FILTER LPAREN VARIABLE COMMA expression RPAREN                   // $var = FILTER($list, condition)
    | SORT LPAREN VARIABLE COMMA sortSpecList RPAREN                   // $var = SORT($list, attr ASC)
    | UNION LPAREN VARIABLE COMMA VARIABLE RPAREN                      // $var = UNION($list1, $list2)
    | INTERSECT LPAREN VARIABLE COMMA VARIABLE RPAREN                  // $var = INTERSECT($list1, $list2)
    | SUBTRACT LPAREN VARIABLE COMMA VARIABLE RPAREN                   // $var = SUBTRACT($list1, $list2)
    | CONTAINS LPAREN VARIABLE COMMA VARIABLE RPAREN                   // $bool = CONTAINS($list, $item)
    | EQUALS_OP LPAREN VARIABLE COMMA VARIABLE RPAREN                  // $bool = EQUALS($list1, $list2)
    ;

sortSpecList
    : sortSpec (COMMA sortSpec)*
    ;

sortSpec
    : IDENTIFIER (ASC | DESC)?
    ;

/**
 * Aggregate operations on lists.
 *
 * @example Count items
 * ```mdl
 * $TotalProducts = COUNT($ProductList);
 * ```
 *
 * @example Sum attribute values
 * ```mdl
 * $TotalPrice = SUM($Products.Price);
 * ```
 *
 * @example Average attribute values
 * ```mdl
 * $AvgPrice = AVERAGE($Products.Price);
 * ```
 *
 * @example Min/Max attribute values
 * ```mdl
 * $MinPrice = MINIMUM($Products.Price);
 * $MaxPrice = MAXIMUM($Products.Price);
 * ```
 */
aggregateListStatement
    : VARIABLE EQUALS listAggregateOperation
    ;

listAggregateOperation
    : COUNT LPAREN VARIABLE RPAREN                                     // $count = COUNT($list)
    | SUM LPAREN attributePath RPAREN                                  // $sum = SUM($list.attr)
    | AVERAGE LPAREN attributePath RPAREN                              // $avg = AVERAGE($list.attr)
    | MINIMUM LPAREN attributePath RPAREN                              // $min = MINIMUM($list.attr)
    | MAXIMUM LPAREN attributePath RPAREN                              // $max = MAXIMUM($list.attr)
    ;

/**
 * Create an empty list of a specific entity type.
 *
 * @example Create empty product list
 * ```mdl
 * $EmptyList = CREATE LIST OF MfTest.Product;
 * ```
 */
createListStatement
    : VARIABLE EQUALS CREATE LIST_OF qualifiedName
    ;

/**
 * Add an item to a list.
 *
 * @example Add product to list
 * ```mdl
 * ADD $Product TO $ProductList;
 * ```
 */
addToListStatement
    : ADD VARIABLE TO VARIABLE
    ;

/**
 * Remove an item from a list.
 *
 * @example Remove product from list
 * ```mdl
 * REMOVE $Product FROM $ProductList;
 * ```
 */
removeFromListStatement
    : REMOVE VARIABLE FROM VARIABLE
    ;

// Member assignments for CREATE and CHANGE: Name = $Name, Code = $Code
memberAssignmentList
    : memberAssignment (COMMA memberAssignment)*
    ;

memberAssignment
    : memberAttributeName EQUALS expression
    ;

// Allow keywords and qualified names as member attribute names
// Qualified names are needed for association references (e.g., Module.Order_Customer)
memberAttributeName
    : qualifiedName
    | IDENTIFIER
    | QUOTED_IDENTIFIER                     // Escape any reserved word
    | keyword
    ;

// Legacy changeList for backwards compatibility
changeList
    : changeItem (COMMA changeItem)*
    ;

changeItem
    : IDENTIFIER EQUALS expression
    ;

// =============================================================================
// PAGE CREATION
// =============================================================================

/**
 * Creates a new page with layout, parameters, and widget content.
 *
 * Pages define the user interface with widgets arranged in a layout structure.
 *
 * @example Simple page with text
 * ```mdl
 * CREATE PAGE MyModule.HomePage ()
 *   TITLE 'Welcome'
 *   LAYOUT Atlas_Core.Atlas_Default
 * BEGIN
 *   LAYOUTGRID BEGIN
 *     ROW BEGIN
 *       COLUMN 12 BEGIN
 *         DYNAMICTEXT (CONTENT 'Hello, World!', RENDERMODE 'H1')
 *       END
 *     END
 *   END
 * END;
 * ```
 *
 * @example Page with parameter and data view
 * ```mdl
 * CREATE PAGE MyModule.CustomerDetails ($Customer: MyModule.Customer)
 *   TITLE 'Customer Details'
 *   LAYOUT Atlas_Core.Atlas_Default
 * BEGIN
 *   DATAVIEW dvCustomer DATASOURCE $Customer BEGIN
 *     TEXTBOX (ATTRIBUTE Name, LABEL 'Name'),
 *     TEXTBOX (ATTRIBUTE Email, LABEL 'Email')
 *   END
 * END;
 * ```
 *
 * @example Page with action button
 * ```mdl
 * CREATE PAGE MyModule.OrderForm ($Order: MyModule.Order)
 *   TITLE 'New Order'
 *   LAYOUT Atlas_Core.Atlas_Default
 * BEGIN
 *   ACTIONBUTTON btnSave 'Save Order'
 *     ACTION CALL MICROFLOW MyModule.SaveOrder
 *     STYLE Primary
 * END;
 * ```
 *
 * @see pageBody for widget definitions
 * @see widgetDefinition for available widgets
 */
createPageStatement
    : PAGE qualifiedName
      pageHeaderV3
      LBRACE pageBodyV3 RBRACE
    ;

// =============================================================================
// SNIPPET CREATION
// =============================================================================

createSnippetStatement
    : SNIPPET qualifiedName
      snippetHeaderV3?
      snippetOptions?
      LBRACE pageBodyV3 RBRACE
    ;

snippetOptions: snippetOption+ ;
snippetOption: FOLDER STRING_LITERAL ;

// =============================================================================
// SHARED PAGE/SNIPPET RULES
// =============================================================================

pageParameterList
    : pageParameter (COMMA pageParameter)*
    ;

pageParameter
    : (IDENTIFIER | VARIABLE) COLON dataType
    ;

snippetParameterList
    : snippetParameter (COMMA snippetParameter)*
    ;

snippetParameter
    : (IDENTIFIER | VARIABLE) COLON dataType
    ;

variableDeclarationList
    : variableDeclaration (COMMA variableDeclaration)*
    ;

variableDeclaration
    : VARIABLE COLON dataType EQUALS STRING_LITERAL     // $varName: Boolean = 'expression'
    ;

sortColumn
    : (qualifiedName | IDENTIFIER) (ASC | DESC)?
    ;

xpathConstraint
    : LBRACKET xpathExpr RBRACKET
    ;

andOrXpath
    : AND
    | OR
    ;

// =============================================================================
// XPATH EXPRESSION RULES
// =============================================================================
//
// Dedicated grammar for XPath expressions inside [...] constraints.
// Separate from the general expression rules because XPath has different semantics:
// - '/' is always path traversal (not division)
// - '[...]' inside paths are nested predicates
// - Bare identifiers/paths are existence checks
// - Functions like not(), contains(), starts-with() are XPath-native
//

xpathExpr
    : xpathAndExpr (OR xpathAndExpr)*
    ;

xpathAndExpr
    : xpathNotExpr (AND xpathNotExpr)*
    ;

xpathNotExpr
    : NOT xpathNotExpr
    | xpathComparisonExpr
    ;

xpathComparisonExpr
    : xpathValueExpr (comparisonOperator xpathValueExpr)?
    ;

xpathValueExpr
    : xpathFunctionCall
    | xpathPath
    | LPAREN xpathExpr RPAREN
    ;

xpathPath
    : xpathStep (SLASH xpathStep)*
    ;

xpathStep
    : xpathStepValue (LBRACKET xpathExpr RBRACKET)?
    ;

xpathStepValue
    : xpathQualifiedName
    | VARIABLE
    | STRING_LITERAL
    | NUMBER_LITERAL
    | MENDIX_TOKEN
    ;

/** Qualified name in XPath context: accepts any keyword as identifier part.
 *  Unlike the general qualifiedName rule, this handles ALL lexer keywords
 *  so that attribute names like 'Attr', 'Entity', etc. work in XPath paths.
 */
xpathQualifiedName
    : xpathWord (DOT xpathWord)*
    ;

/** Any single-word token that can appear as part of a name in XPath.
 *  Uses exclusion: matches any token except operators, delimiters, and literals.
 */
xpathWord
    : ~( DOT | SLASH | LBRACKET | RBRACKET | LPAREN | RPAREN | COMMA
       | EQUALS | NOT_EQUALS | LESS_THAN | LESS_THAN_OR_EQUAL
       | GREATER_THAN | GREATER_THAN_OR_EQUAL
       | AND | OR | NOT
       | SEMICOLON
       | STRING_LITERAL | NUMBER_LITERAL | VARIABLE | MENDIX_TOKEN | DOLLAR_STRING
       )
    ;

xpathFunctionCall
    : xpathFunctionName LPAREN (xpathExpr (COMMA xpathExpr)*)? RPAREN
    ;

xpathFunctionName
    : IDENTIFIER
    | HYPHENATED_ID
    | NOT
    | TRUE
    | FALSE
    | CONTAINS
    ;

// =============================================================================
// PAGE V3 SYNTAX (Agent-Friendly: all properties in parentheses)
// =============================================================================
//
// V3 follows the pattern: WIDGET name (Prop: Value) { children }
//
// Key differences from V2:
// - Page header uses single () block with Params:, Title:, Layout:, Url:
// - DataSource: replaces -> for containers
// - Binds: replaces -> for input widgets
// - Action: replaces -> for buttons
// - Caption: replaces bare string for buttons
// - Label: replaces bare string for input widgets
//
// Examples:
//   CREATE PAGE Module.Page (Title: 'My Page', Layout: Atlas_Core.Atlas_Default) { ... }
//   TEXTBOX txtName (Label: 'Name', Binds: Name)
//   DATAVIEW dvProduct (DataSource: $Product) { ... }
//   ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, Style: Primary)
//

// V3 Page Header: all metadata in single () block
pageHeaderV3
    : LPAREN pageHeaderPropertyV3 (COMMA pageHeaderPropertyV3)* RPAREN
    ;

pageHeaderPropertyV3
    : PARAMS COLON LBRACE pageParameterList RBRACE                   // Params: { $Order: Entity }
    | VARIABLES_KW COLON LBRACE variableDeclarationList RBRACE       // Variables: { $show: Boolean = 'true' }
    | TITLE COLON STRING_LITERAL                                     // Title: 'My Page'
    | LAYOUT COLON (qualifiedName | STRING_LITERAL)                  // Layout: Atlas_Core.Atlas_Default
    | URL COLON STRING_LITERAL                                       // Url: 'my-page'
    | FOLDER COLON STRING_LITERAL                                    // Folder: 'Pages/Admin'
    ;

// V3 Snippet Header
snippetHeaderV3
    : LPAREN snippetHeaderPropertyV3 (COMMA snippetHeaderPropertyV3)* RPAREN
    ;

snippetHeaderPropertyV3
    : PARAMS COLON LBRACE snippetParameterList RBRACE              // Params: { $Customer: Entity }
    | VARIABLES_KW COLON LBRACE variableDeclarationList RBRACE     // Variables: { $show: Boolean = 'true' }
    | FOLDER COLON STRING_LITERAL                                  // Folder: 'Snippets/Common'
    ;

// V3 Page body
pageBodyV3
    : (widgetV3 | useFragmentRef)*
    ;

// USE FRAGMENT Name [AS prefix_]
useFragmentRef
    : USE FRAGMENT identifierOrKeyword (AS identifierOrKeyword)?
    ;

// V3 Widget: WIDGET name (Props) { children }
widgetV3
    : widgetTypeV3 IDENTIFIER widgetPropertiesV3? widgetBodyV3?
    | PLUGGABLEWIDGET STRING_LITERAL IDENTIFIER widgetPropertiesV3? widgetBodyV3?  // PLUGGABLEWIDGET 'widget.id' name
    | CUSTOMWIDGET STRING_LITERAL IDENTIFIER widgetPropertiesV3? widgetBodyV3?     // CUSTOMWIDGET 'widget.id' name (legacy)
    ;

// V3 Widget types (same as V2)
widgetTypeV3
    : LAYOUTGRID
    | ROW
    | COLUMN
    | DATAGRID
    | DATAVIEW
    | LISTVIEW
    | GALLERY
    | CONTAINER
    | NAVIGATIONLIST
    | ITEM
    | TEXTBOX
    | TEXTAREA
    | DATEPICKER
    | DROPDOWN
    | COMBOBOX
    | CHECKBOX
    | RADIOBUTTONS
    | REFERENCESELECTOR
    | ACTIONBUTTON
    | LINKBUTTON
    | TITLE
    | DYNAMICTEXT
    | STATICTEXT
    | SNIPPETCALL
    | CUSTOMWIDGET
    | TEXTFILTER
    | NUMBERFILTER
    | DROPDOWNFILTER
    | DATEFILTER
    | DROPDOWNSORT
    | FOOTER
    | HEADER
    | CONTROLBAR
    | FILTER
    | TEMPLATE
    | IMAGE
    | STATICIMAGE
    | DYNAMICIMAGE
    | CUSTOMCONTAINER
    | TABCONTAINER
    | TABPAGE
    | GROUPBOX
    ;

// V3 Widget properties: (Prop: Value, Prop: Value)
widgetPropertiesV3
    : LPAREN widgetPropertyV3 (COMMA widgetPropertyV3)* RPAREN
    ;

widgetPropertyV3
    : DATASOURCE COLON dataSourceExprV3               // DataSource: $var | DATABASE Entity | MICROFLOW ...
    | ATTRIBUTE COLON attributePathV3                 // Attribute: Name | Product/Category
    | BINDS COLON attributePathV3                     // Binds: (deprecated, use Attribute:)
    | ACTION COLON actionExprV3                       // Action: SAVE_CHANGES | SHOW_PAGE ...
    | CAPTION COLON stringExprV3                      // Caption: 'Save'
    | LABEL COLON STRING_LITERAL                      // Label: 'Name'
    | ATTR COLON attributePathV3                      // Attr: (deprecated, use Attribute:)
    | CONTENT COLON stringExprV3                      // Content: 'Hello {1}'
    | RENDERMODE COLON renderModeV3                   // RenderMode: H3
    | CONTENTPARAMS COLON paramListV3                 // ContentParams: [{1} = $var.Name]
    | CAPTIONPARAMS COLON paramListV3                 // CaptionParams: [{1} = 'hello']
    | BUTTONSTYLE COLON buttonStyleV3                  // ButtonStyle: Primary
    | CLASS COLON STRING_LITERAL                       // Class: 'my-class'
    | STYLE COLON STRING_LITERAL                       // Style: 'color: red'
    | DESKTOPWIDTH COLON desktopWidthV3               // DesktopWidth: 6 | AutoFill
    | TABLETWIDTH COLON desktopWidthV3                // TabletWidth: 6 | AutoFill
    | PHONEWIDTH COLON desktopWidthV3                 // PhoneWidth: 12 | AutoFill
    // Where: and OrderBy: removed — use inline WHERE/SORT BY in DataSource: expression
    | SELECTION COLON selectionModeV3                 // Selection: Single | Multiple
    | SNIPPET COLON qualifiedName                     // Snippet: Module.SnippetName
    | ATTRIBUTES COLON attributeListV3                // Attributes: [Entity.Attr1, Entity.Attr2]
    | FILTERTYPE COLON filterTypeValue                // FilterType: startsWith | contains | equal
    | DESIGNPROPERTIES COLON designPropertyListV3       // DesignProperties: [...]
    | WIDTH COLON NUMBER_LITERAL                        // Width: 200
    | HEIGHT COLON NUMBER_LITERAL                      // Height: 100
    | VISIBLE COLON xpathConstraint                    // Visible: [IsActive = true]
    | VISIBLE COLON propertyValueV3                   // Visible: false
    | EDITABLE COLON xpathConstraint                  // Editable: [Status != 'Closed']
    | EDITABLE COLON propertyValueV3                  // Editable: Never | Always
    | TOOLTIP COLON propertyValueV3                   // Tooltip: 'text'
    | IDENTIFIER COLON propertyValueV3                // Generic: any other property
    | keyword COLON propertyValueV3                  // Generic: keyword as property name (for pluggable widgets)
    ;

// Filter type values - handle keywords like CONTAINS that are also filter types
filterTypeValue
    : CONTAINS      // contains
    | EMPTY         // empty
    | IDENTIFIER    // startsWith, endsWith, greater, greaterEqual, equal, notEqual, smaller, smallerEqual, notEmpty
    ;

// V3 Attribute list for filter widgets
attributeListV3
    : LBRACKET qualifiedName (COMMA qualifiedName)* RBRACKET
    ;

// V3 DataSource expressions
dataSourceExprV3
    : VARIABLE                                        // $ParamName
    | DATABASE FROM? qualifiedName                    // DATABASE [FROM] Entity [WHERE ...] [SORT BY ...]
      (WHERE (xpathConstraint (andOrXpath? xpathConstraint)* | expression))?
      (SORT_BY sortColumn (COMMA sortColumn)*)?
    | MICROFLOW qualifiedName microflowArgsV3?        // MICROFLOW Module.Flow
    | NANOFLOW qualifiedName microflowArgsV3?         // NANOFLOW Module.Flow
    | ASSOCIATION attributePathV3                     // ASSOCIATION Path
    | SELECTION IDENTIFIER                            // SELECTION widgetName
    ;

// V3 Action expressions
actionExprV3
    : SAVE_CHANGES (CLOSE_PAGE)?                      // SAVE_CHANGES or SAVE_CHANGES CLOSE_PAGE
    | CANCEL_CHANGES (CLOSE_PAGE)?                    // CANCEL_CHANGES
    | CLOSE_PAGE                                      // CLOSE_PAGE
    | DELETE_OBJECT                                   // DELETE_OBJECT
    | DELETE (CLOSE_PAGE)?                            // DELETE (legacy)
    | CREATE_OBJECT qualifiedName (THEN actionExprV3)? // CREATE_OBJECT Entity THEN SHOW_PAGE ...
    | SHOW_PAGE qualifiedName microflowArgsV3?        // SHOW_PAGE Module.Page (Param: val)
    | MICROFLOW qualifiedName microflowArgsV3?        // MICROFLOW Module.Flow
    | NANOFLOW qualifiedName microflowArgsV3?         // NANOFLOW Module.Flow
    | OPEN_LINK STRING_LITERAL                        // OPEN_LINK 'https://...'
    | SIGN_OUT                                        // SIGN_OUT
    | COMPLETE_TASK STRING_LITERAL                    // COMPLETE_TASK 'OutcomeName'
    ;

// V3 Microflow arguments: (Param: value, ...)
microflowArgsV3
    : LPAREN microflowArgV3 (COMMA microflowArgV3)* RPAREN
    ;

microflowArgV3
    : IDENTIFIER COLON expression                    // Param: $value (canonical)
    | VARIABLE EQUALS expression                     // $Param = $value (microflow-style, also accepted)
    ;

// V3 Attribute path: Name, Product/Category, "Order" (quoted to escape reserved words)
attributePathV3
    : (IDENTIFIER | QUOTED_IDENTIFIER | keyword) (SLASH (IDENTIFIER | QUOTED_IDENTIFIER | keyword))*
    ;

// V3 String expression (may include template placeholders or attribute binding)
// STRING_LITERAL: 'Hello {1}' for template text
// attributePathV3: Name or $widget.Name for direct attribute binding
// VARIABLE: $var for variable references
stringExprV3
    : STRING_LITERAL
    | attributePathV3
    | VARIABLE (DOT (IDENTIFIER | keyword))?
    ;

// V3 Parameter list: [{1} = value, {2} = value]
paramListV3
    : LBRACKET paramAssignmentV3 (COMMA paramAssignmentV3)* RBRACKET
    ;

paramAssignmentV3
    : LBRACE NUMBER_LITERAL RBRACE EQUALS expression
    ;

// V3 Render modes
renderModeV3
    : H1 | H2 | H3 | H4 | H5 | H6 | PARAGRAPH | TEXT | IDENTIFIER
    ;

// V3 Button styles
buttonStyleV3
    : PRIMARY | DEFAULT | SUCCESS | DANGER | WARNING | WARNING_STYLE | INFO | INFO_STYLE | IDENTIFIER
    ;

// V3 Desktop width
desktopWidthV3
    : NUMBER_LITERAL | AUTOFILL
    ;

// V3 Selection mode
selectionModeV3
    : SINGLE | MULTIPLE | NONE
    ;

// V3 Generic property value
propertyValueV3
    : STRING_LITERAL
    | NUMBER_LITERAL
    | booleanLiteral
    | qualifiedName
    | IDENTIFIER
    | H1 | H2 | H3 | H4 | H5 | H6  // HeaderMode values
    | LBRACKET (expression (COMMA expression)*)? RBRACKET  // Array
    ;

// V3 Design property list: ['Key': 'Value', 'Key': ON]
designPropertyListV3
    : LBRACKET designPropertyEntryV3 (COMMA designPropertyEntryV3)* RBRACKET
    | LBRACKET RBRACKET
    ;

designPropertyEntryV3
    : STRING_LITERAL COLON STRING_LITERAL
    | STRING_LITERAL COLON ON
    | STRING_LITERAL COLON OFF
    ;

// V3 Widget body: { children }
widgetBodyV3
    : LBRACE pageBodyV3 RBRACE
    ;

// =============================================================================
// NOTEBOOK CREATION
// =============================================================================

createNotebookStatement
    : NOTEBOOK qualifiedName
      notebookOptions?
      BEGIN notebookPage* END
    ;

notebookOptions
    : notebookOption+
    ;

notebookOption
    : COMMENT STRING_LITERAL
    ;

notebookPage
    : PAGE qualifiedName (CAPTION STRING_LITERAL)?
    ;

// =============================================================================
// DATABASE / REST CLIENT
// =============================================================================

createDatabaseConnectionStatement
    : DATABASE CONNECTION qualifiedName
      databaseConnectionOption+
      (BEGIN databaseQuery* END)?
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

createConstantStatement
    : CONSTANT qualifiedName
      TYPE dataType
      DEFAULT literal
      constantOptions?
    ;

constantOptions
    : constantOption+
    ;

constantOption
    : COMMENT STRING_LITERAL
    | FOLDER STRING_LITERAL
    | EXPOSED TO CLIENT
    ;

createConfigurationStatement
    : CONFIGURATION STRING_LITERAL
      (settingsAssignment (COMMA settingsAssignment)*)?
    ;

/**
 * CREATE REST CLIENT — property-based syntax with { } blocks.
 *
 * @example Simple GET client
 * ```mdl
 * CREATE REST CLIENT Module.Api (
 *   BaseUrl: 'https://api.example.com/v1',
 *   Authentication: NONE
 * )
 * {
 *   OPERATION GetData {
 *     Method: GET,
 *     Path: '/data',
 *     Parameters: ($userId: Integer),
 *     Headers: ('Accept' = 'application/json'),
 *     Response: JSON AS $Result
 *   }
 * };
 * ```
 *
 * @example POST with body and response mapping
 * ```mdl
 * OPERATION PostOrder {
 *   Method: POST,
 *   Path: '/orders',
 *   Headers: ('Content-Type' = 'application/json'),
 *   Body: MAPPING Module.OrderRequest {
 *     orderId = OrderId,
 *   },
 *   Response: MAPPING Module.OrderResponse {
 *     OrderId = orderId,
 *     CREATE Module.Items_Order/Module.Item = items {
 *       Sku = sku,
 *     }
 *   }
 * }
 * ```
 */
createRestClientStatement
    : REST CLIENT qualifiedName
      LPAREN restClientProperty (COMMA restClientProperty)* RPAREN
      LBRACE restClientOperation* RBRACE
    ;

restClientProperty
    : identifierOrKeyword COLON STRING_LITERAL                       // BaseUrl: '...', Username: '...'
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

/**
 * CREATE PUBLISHED REST SERVICE Module.Name (
 *   Path: 'api/v1',
 *   Version: '1.0.0',
 *   ServiceName: 'My API'
 * )
 * {
 *   RESOURCE 'orders' {
 *     Get / MICROFLOW Module.GetOrders;
 *     Post / MICROFLOW Module.CreateOrder;
 *   }
 * };
 */
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
// INDEX CREATION (standalone)
// =============================================================================

createIndexStatement
    : INDEX IDENTIFIER ON qualifiedName LPAREN indexAttributeList RPAREN
    ;

// =============================================================================
// ODATA CLIENT / SERVICE
// =============================================================================

/**
 * CREATE ODATA CLIENT Module.Name (
 *   Version: '1.0',
 *   ODataVersion: OData4,
 *   MetadataUrl: 'https://...',
 *   Timeout: 300,
 *   ProxyType: DefaultProxy
 * );
 */
createODataClientStatement
    : ODATA CLIENT qualifiedName
      LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN
      odataHeadersClause?
    ;

/**
 * CREATE ODATA SERVICE Module.Name (
 *   Path: '/odata/v1',
 *   Version: '1.0.0',
 *   ODataVersion: OData4,
 *   Namespace: 'MyApp',
 *   ServiceName: 'My Service',
 *   Summary: 'Description of the service',
 *   PublishAssociations: Yes
 * )
 * AUTHENTICATION Basic, Session
 * {
 *   PUBLISH ENTITY Module.Entity AS 'EntitySet' (
 *     ReadMode: SOURCE,
 *     InsertMode: SOURCE,
 *     UpdateMode: NOT_SUPPORTED,
 *     DeleteMode: NOT_SUPPORTED,
 *     UsePaging: Yes,
 *     PageSize: 100
 *   )
 *   EXPOSE (
 *     Id AS 'customerId',
 *     Name (Filterable, Sortable),
 *     Email
 *   );
 * }
 */
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
    : IDENTIFIER (AS STRING_LITERAL)? exposeMemberOptions?
    ;

exposeMemberOptions
    : LPAREN IDENTIFIER (COMMA IDENTIFIER)* RPAREN
    ;

/**
 * CREATE [OR MODIFY] EXTERNAL ENTITY Module.Name
 * FROM ODATA CLIENT Module.ServiceName
 * (EntitySet: 'Accounts', RemoteName: 'Account', Countable: Yes, ...)
 * (Id: String(200), Name: String(255));
 */
createExternalEntityStatement
    : EXTERNAL ENTITY qualifiedName
      FROM ODATA CLIENT qualifiedName
      LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN
      (LPAREN attributeDefinitionList? RPAREN)?
    ;

/**
 * CREATE [OR MODIFY] EXTERNAL ENTITIES FROM Module.Service [INTO Module] [ENTITIES (Name1, Name2)]
 * Bulk-creates external entities from a consumed OData service's cached $metadata.
 */
createExternalEntitiesStatement
    : EXTERNAL ENTITIES FROM qualifiedName
      (INTO (qualifiedName | IDENTIFIER))?
      (ENTITIES LPAREN identifierOrKeyword (COMMA identifierOrKeyword)* RPAREN)?
    ;

/**
 * CREATE [OR REPLACE] NAVIGATION Responsive
 *   HOME PAGE Module.Page
 *   LOGIN PAGE Module.LoginPage
 *   MENU (
 *     MENU ITEM 'Home' PAGE Module.Page;
 *   );
 */
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

/**
 * CREATE BUSINESS EVENT SERVICE Module.Name (ServiceName: 'name', EventNamePrefix: '') {
 *   MESSAGE MsgName (AttrName: Type) PUBLISH ENTITY Module.Entity;
 * };
 */
createBusinessEventServiceStatement
    : BUSINESS EVENT SERVICE qualifiedName
      LPAREN odataPropertyAssignment (COMMA odataPropertyAssignment)* RPAREN
      LBRACE businessEventMessageDef+ RBRACE
    ;

businessEventMessageDef
    : MESSAGE IDENTIFIER
      LPAREN businessEventAttrDef (COMMA businessEventAttrDef)* RPAREN
      (PUBLISH | SUBSCRIBE)
      (ENTITY qualifiedName)?
      (MICROFLOW qualifiedName)?
      SEMICOLON
    ;

businessEventAttrDef
    : IDENTIFIER COLON dataType
    ;

// =============================================================================
// CREATE WORKFLOW
// =============================================================================

/**
 * Create a workflow with activities.
 *
 * @example Simple workflow with user task
 * ```mdl
 * CREATE WORKFLOW MyModule.ApprovalWorkflow
 *   PARAMETER $WorkflowContext: MyModule.Request
 * BEGIN
 *   USER TASK ReviewRequest 'Review the request'
 *     PAGE MyModule.ReviewPage
 *     OUTCOMES
 *       'Approve'
 *       'Reject'
 *
 *   END
 * END WORKFLOW;
 * ```
 */
createWorkflowStatement
    : WORKFLOW qualifiedName
      (PARAMETER VARIABLE COLON qualifiedName)?
      (DISPLAY STRING_LITERAL)?
      (DESCRIPTION STRING_LITERAL)?
      (EXPORT LEVEL (IDENTIFIER | API))?
      (OVERVIEW PAGE qualifiedName)?
      (DUE DATE_TYPE STRING_LITERAL)?
      BEGIN workflowBody END WORKFLOW SEMICOLON? SLASH?
    ;

workflowBody
    : workflowActivityStmt*
    ;

workflowActivityStmt
    : workflowUserTaskStmt SEMICOLON
    | workflowCallMicroflowStmt SEMICOLON
    | workflowCallWorkflowStmt SEMICOLON
    | workflowDecisionStmt SEMICOLON
    | workflowParallelSplitStmt SEMICOLON
    | workflowJumpToStmt SEMICOLON
    | workflowWaitForTimerStmt SEMICOLON
    | workflowWaitForNotificationStmt SEMICOLON
    | workflowAnnotationStmt SEMICOLON
    ;

workflowUserTaskStmt
    : USER TASK IDENTIFIER STRING_LITERAL
      (PAGE qualifiedName)?
      (TARGETING MICROFLOW qualifiedName)?
      (TARGETING XPATH STRING_LITERAL)?
      (ENTITY qualifiedName)?
      (DUE DATE_TYPE STRING_LITERAL)?
      (DESCRIPTION STRING_LITERAL)?
      (OUTCOMES workflowUserTaskOutcome+)?
      (BOUNDARY EVENT workflowBoundaryEventClause+)?
    | MULTI USER TASK IDENTIFIER STRING_LITERAL
      (PAGE qualifiedName)?
      (TARGETING MICROFLOW qualifiedName)?
      (TARGETING XPATH STRING_LITERAL)?
      (ENTITY qualifiedName)?
      (DUE DATE_TYPE STRING_LITERAL)?
      (DESCRIPTION STRING_LITERAL)?
      (OUTCOMES workflowUserTaskOutcome+)?
      (BOUNDARY EVENT workflowBoundaryEventClause+)?
    ;

workflowBoundaryEventClause
    : INTERRUPTING TIMER STRING_LITERAL? (LBRACE workflowBody RBRACE)?
    | NON INTERRUPTING TIMER STRING_LITERAL? (LBRACE workflowBody RBRACE)?
    | TIMER STRING_LITERAL? (LBRACE workflowBody RBRACE)?
    ;

workflowUserTaskOutcome
    : STRING_LITERAL LBRACE workflowBody RBRACE
    ;

workflowCallMicroflowStmt
    : CALL MICROFLOW qualifiedName (COMMENT STRING_LITERAL)?
      (WITH LPAREN workflowParameterMapping (COMMA workflowParameterMapping)* RPAREN)?
      (OUTCOMES workflowConditionOutcome+)?
      (BOUNDARY EVENT workflowBoundaryEventClause+)?
    ;

workflowParameterMapping
    : qualifiedName EQUALS STRING_LITERAL
    ;

workflowCallWorkflowStmt
    : CALL WORKFLOW qualifiedName (COMMENT STRING_LITERAL)?
      (WITH LPAREN workflowParameterMapping (COMMA workflowParameterMapping)* RPAREN)?
    ;

workflowDecisionStmt
    : DECISION STRING_LITERAL? (COMMENT STRING_LITERAL)?
      (OUTCOMES workflowConditionOutcome+)?
    ;

workflowConditionOutcome
    : (TRUE | FALSE | STRING_LITERAL | DEFAULT) ARROW LBRACE workflowBody RBRACE
    ;

workflowParallelSplitStmt
    : PARALLEL SPLIT (COMMENT STRING_LITERAL)?
      workflowParallelPath+
    ;

workflowParallelPath
    : PATH NUMBER_LITERAL LBRACE workflowBody RBRACE
    ;

workflowJumpToStmt
    : JUMP TO IDENTIFIER (COMMENT STRING_LITERAL)?
    ;

workflowWaitForTimerStmt
    : WAIT FOR TIMER STRING_LITERAL? (COMMENT STRING_LITERAL)?
    ;

workflowWaitForNotificationStmt
    : WAIT FOR NOTIFICATION (COMMENT STRING_LITERAL)?
      (BOUNDARY EVENT workflowBoundaryEventClause+)?
    ;

workflowAnnotationStmt
    : ANNOTATION STRING_LITERAL
    ;

// workflowEndStmt removed - END activities are implicit and conflict with END WORKFLOW

// =============================================================================
// ALTER WORKFLOW
// =============================================================================

alterWorkflowAction
    : SET workflowSetProperty
    | SET ACTIVITY alterActivityRef activitySetProperty
    | INSERT AFTER alterActivityRef workflowActivityStmt
    | DROP ACTIVITY alterActivityRef
    | REPLACE ACTIVITY alterActivityRef WITH workflowActivityStmt
    | INSERT OUTCOME STRING_LITERAL ON alterActivityRef LBRACE workflowBody RBRACE
    | INSERT PATH ON alterActivityRef LBRACE workflowBody RBRACE
    | DROP OUTCOME STRING_LITERAL ON alterActivityRef
    | DROP PATH STRING_LITERAL ON alterActivityRef
    | INSERT BOUNDARY EVENT ON alterActivityRef workflowBoundaryEventClause
    | DROP BOUNDARY EVENT ON alterActivityRef
    | INSERT CONDITION STRING_LITERAL ON alterActivityRef LBRACE workflowBody RBRACE
    | DROP CONDITION STRING_LITERAL ON alterActivityRef
    ;

workflowSetProperty
    : DISPLAY STRING_LITERAL
    | DESCRIPTION STRING_LITERAL
    | EXPORT LEVEL (IDENTIFIER | API)
    | DUE DATE_TYPE STRING_LITERAL
    | OVERVIEW PAGE qualifiedName
    | PARAMETER VARIABLE COLON qualifiedName
    ;

activitySetProperty
    : PAGE qualifiedName
    | DESCRIPTION STRING_LITERAL
    | TARGETING MICROFLOW qualifiedName
    | TARGETING XPATH STRING_LITERAL
    | DUE DATE_TYPE STRING_LITERAL
    ;

alterActivityRef
    : IDENTIFIER (AT NUMBER_LITERAL)?
    | STRING_LITERAL (AT NUMBER_LITERAL)?
    ;

// =============================================================================
// ALTER SETTINGS
// =============================================================================

/**
 * ALTER SETTINGS MODEL Key = Value, ...;
 * ALTER SETTINGS CONFIGURATION 'name' Key = Value, ...;
 * ALTER SETTINGS CONSTANT 'name' VALUE 'value' [IN CONFIGURATION 'name'];
 * ALTER SETTINGS LANGUAGE Key = Value, ...;
 * ALTER SETTINGS WORKFLOWS Key = Value, ...;
 */
alterSettingsClause
    : settingsSection settingsAssignment (COMMA settingsAssignment)*
    | CONSTANT STRING_LITERAL (VALUE settingsValue | DROP) (IN CONFIGURATION STRING_LITERAL)?
    | DROP CONSTANT STRING_LITERAL (IN CONFIGURATION STRING_LITERAL)?
    | CONFIGURATION STRING_LITERAL settingsAssignment (COMMA settingsAssignment)*
    ;

settingsSection
    : IDENTIFIER   // MODEL, LANGUAGE
    | WORKFLOWS
    ;

settingsAssignment
    : IDENTIFIER EQUALS settingsValue
    ;

settingsValue
    : STRING_LITERAL
    | NUMBER_LITERAL
    | booleanLiteral
    | qualifiedName
    ;

// =============================================================================
// DQL STATEMENTS (Data Query Language)
// =============================================================================

dqlStatement
    : showStatement
    | describeStatement
    | catalogSelectQuery
    | oqlQuery
    ;

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
    | showOrList LANGUAGES
    | showOrList FEATURES (IN IDENTIFIER)?
    | showOrList FEATURES FOR VERSION NUMBER_LITERAL
    | showOrList FEATURES ADDED SINCE NUMBER_LITERAL
    ;

/**
 * Widget filtering for SHOW WIDGETS and UPDATE WIDGETS.
 *
 * @example Filter by widget type
 * ```mdl
 * SHOW WIDGETS WHERE WidgetType LIKE '%combobox%';
 * ```
 *
 * @example Filter within a module
 * ```mdl
 * SHOW WIDGETS IN MyModule;
 * ```
 *
 * @example Combined filter
 * ```mdl
 * SHOW WIDGETS WHERE WidgetType LIKE '%DataGrid%' IN MyModule;
 * ```
 */
showWidgetsFilter
    : WHERE widgetCondition (AND widgetCondition)* (IN (qualifiedName | IDENTIFIER))?
    | IN (qualifiedName | IDENTIFIER)
    ;

/**
 * Widget type keyword for SHOW DESIGN PROPERTIES FOR <type>.
 * Matches MDL widget type names and also allows identifiers for custom/pluggable widgets.
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
    | DESCRIBE MODULE IDENTIFIER (WITH ALL)?  // DESCRIBE MODULE Name [WITH ALL] - optionally include all objects
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
    | DESCRIBE JSON STRUCTURE qualifiedName              // DESCRIBE JSON STRUCTURE Module.Name
    | DESCRIBE IMPORT MAPPING qualifiedName             // DESCRIBE IMPORT MAPPING Module.Name
    | DESCRIBE EXPORT MAPPING qualifiedName             // DESCRIBE EXPORT MAPPING Module.Name
    | DESCRIBE REST CLIENT qualifiedName                // DESCRIBE REST CLIENT Module.Name
    | DESCRIBE PUBLISHED REST SERVICE qualifiedName    // DESCRIBE PUBLISHED REST SERVICE Module.Name
    | DESCRIBE FRAGMENT identifierOrKeyword            // DESCRIBE FRAGMENT Name
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
// Many table names are MDL keywords, so we need to list them explicitly
catalogTableName
    : MODULES
    | ENTITIES
    | MICROFLOWS
    | NANOFLOWS
    | PAGES
    | SNIPPETS
    | LAYOUTS
    | ENUMERATIONS
    | ATTRIBUTES
    | WIDGETS
    | WORKFLOWS
    | SOURCE_KW   // For CATALOG.SOURCE FTS table
    | ODATA       // For CATALOG.ODATA_CLIENTS and CATALOG.ODATA_SERVICES (via IDENTIFIER)
    | IDENTIFIER  // For tables like activities, xpath_expressions, objects, projects, snapshots, refs, strings, odata_clients, odata_services, java_actions
    ;

// =============================================================================
// OQL QUERY (Object Query Language)
// =============================================================================

/**
 * OQL (Object Query Language) query for retrieving data.
 *
 * OQL is similar to SQL but operates on Mendix entities and supports
 * associations, aggregations, and subqueries.
 *
 * @example Simple SELECT query
 * ```mdl
 * SELECT Name, Email FROM MyModule.Customer
 * ```
 *
 * @example Query with WHERE clause
 * ```mdl
 * SELECT c.Name, c.Email
 * FROM MyModule.Customer AS c
 * WHERE c.Active = true AND c.Age > 18
 * ```
 *
 * @example Query with JOIN via association
 * ```mdl
 * SELECT o.OrderNumber, c.Name AS CustomerName
 * FROM MyModule.Order AS o
 * INNER JOIN o/MyModule.Order_Customer/MyModule.Customer AS c
 * WHERE o.Status = 'Completed'
 * ```
 *
 * @example Aggregation query
 * ```mdl
 * SELECT c.Country, COUNT(*) AS CustomerCount, AVG(c.Age) AS AvgAge
 * FROM MyModule.Customer AS c
 * GROUP BY c.Country
 * HAVING COUNT(*) > 10
 * ORDER BY CustomerCount DESC
 * ```
 *
 * @example Subquery
 * ```mdl
 * SELECT p.Name, p.Price
 * FROM MyModule.Product AS p
 * WHERE p.Price > (SELECT AVG(p2.Price) FROM MyModule.Product AS p2)
 * ```
 *
 * @see createEntityStatement for using OQL in VIEW entities
 * @see retrieveStatement for using OQL in microflows
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

// =============================================================================
// UTILITY STATEMENTS
// =============================================================================

utilityStatement
    : connectStatement
    | disconnectStatement
    | updateStatement
    | checkStatement
    | buildStatement
    | executeScriptStatement
    | executeRuntimeStatement
    | lintStatement
    | searchStatement
    | useSessionStatement
    | introspectApiStatement
    | debugStatement
    | defineFragmentStatement
    | sqlStatement
    | importStatement
    | helpStatement
    ;

searchStatement
    : SEARCH STRING_LITERAL
    ;

connectStatement
    : CONNECT TO PROJECT STRING_LITERAL (BRANCH STRING_LITERAL)? TOKEN STRING_LITERAL
    | CONNECT LOCAL STRING_LITERAL
    | CONNECT RUNTIME HOST STRING_LITERAL PORT NUMBER_LITERAL (TOKEN STRING_LITERAL)?
    ;

disconnectStatement
    : DISCONNECT
    ;

updateStatement
    : UPDATE
    | REFRESH CATALOG FULL? SOURCE_KW? FORCE? BACKGROUND?
    | REFRESH
    ;

checkStatement
    : CHECK
    ;

buildStatement
    : BUILD
    ;

executeScriptStatement
    : EXECUTE SCRIPT STRING_LITERAL
    ;

executeRuntimeStatement
    : EXECUTE RUNTIME STRING_LITERAL
    ;

lintStatement
    : LINT lintTarget? (FORMAT lintFormat)?
    | SHOW LINT RULES
    ;

lintTarget
    : qualifiedName DOT STAR  // Module.* - lint all in module
    | qualifiedName           // Specific element
    | STAR                    // All
    ;

lintFormat
    : TEXT
    | JSON
    | SARIF
    ;

useSessionStatement
    : USE sessionIdList
    | USE ALL
    ;

sessionIdList
    : sessionId (COMMA sessionId)*
    ;

sessionId
    : IDENTIFIER
    | HYPHENATED_ID
    ;

introspectApiStatement
    : INTROSPECT API
    ;

debugStatement
    : DEBUG STRING_LITERAL
    ;

/**
 * SQL statements for external database connectivity.
 * SQL CONNECT <driver> '<dsn>' AS <alias>
 * SQL DISCONNECT <alias>
 * SQL CONNECTIONS
 * SQL <alias> SHOW TABLES
 * SQL <alias> DESCRIBE <table>
 * SQL <alias> <raw-sql-passthrough>
 */
sqlStatement
    : SQL CONNECT IDENTIFIER STRING_LITERAL AS IDENTIFIER          # sqlConnect
    | SQL DISCONNECT IDENTIFIER                                     # sqlDisconnect
    | SQL CONNECTIONS                                               # sqlConnections
    | SQL IDENTIFIER SHOW IDENTIFIER                                # sqlShowTables
    | SQL IDENTIFIER DESCRIBE IDENTIFIER                            # sqlDescribeTable
    | SQL IDENTIFIER GENERATE CONNECTOR INTO identifierOrKeyword
      (TABLES LPAREN identifierOrKeyword (COMMA identifierOrKeyword)* RPAREN)?
      (VIEWS LPAREN identifierOrKeyword (COMMA identifierOrKeyword)* RPAREN)?
      EXEC?                                                          # sqlGenerateConnector
    | SQL IDENTIFIER sqlPassthrough                                  # sqlQuery
    ;

sqlPassthrough
    : ~(SEMICOLON | SLASH | EOF)+
    ;

importStatement
    : IMPORT FROM identifierOrKeyword QUERY (STRING_LITERAL | DOLLAR_STRING)
      INTO qualifiedName
      MAP LPAREN importMapping (COMMA importMapping)* RPAREN
      (LINK LPAREN linkMapping (COMMA linkMapping)* RPAREN)?
      (BATCH NUMBER_LITERAL)?
      (LIMIT NUMBER_LITERAL)?                                    # importFromQuery
    ;

importMapping
    : identifierOrKeyword AS identifierOrKeyword
    ;

linkMapping
    : identifierOrKeyword TO identifierOrKeyword ON identifierOrKeyword   # linkLookup
    | identifierOrKeyword TO identifierOrKeyword                          # linkDirect
    ;

helpStatement
    : IDENTIFIER (identifierOrKeyword)*  // HELP [topic words...]
    ;

/**
 * DEFINE FRAGMENT Name AS { widgets }
 * Defines a reusable widget group for use in page/snippet bodies.
 */
defineFragmentStatement
    : DEFINE FRAGMENT identifierOrKeyword AS LBRACE pageBodyV3 RBRACE
    ;

// =============================================================================
// EXPRESSIONS (operator precedence from lowest to highest)
// =============================================================================

expression
    : orExpression
    ;

orExpression
    : andExpression (OR andExpression)*
    ;

andExpression
    : notExpression (AND notExpression)*
    ;

notExpression
    : NOT? comparisonExpression
    ;

comparisonExpression
    : additiveExpression
      ( comparisonOperator additiveExpression
      | IS_NULL
      | IS_NOT_NULL
      | IN LPAREN (oqlQuery | expressionList) RPAREN
      | NOT? BETWEEN additiveExpression AND additiveExpression
      | NOT? LIKE additiveExpression
      | MATCH additiveExpression
      )?
    ;

comparisonOperator
    : EQUALS
    | NOT_EQUALS
    | LESS_THAN
    | LESS_THAN_OR_EQUAL
    | GREATER_THAN
    | GREATER_THAN_OR_EQUAL
    ;

additiveExpression
    : multiplicativeExpression ((PLUS | MINUS) multiplicativeExpression)*
    ;

multiplicativeExpression
    : unaryExpression ((STAR | SLASH | COLON | PERCENT | MOD | DIV) unaryExpression)*  // COLON is OQL division
    ;

unaryExpression
    : (PLUS | MINUS)? primaryExpression
    ;

primaryExpression
    : LPAREN expression RPAREN
    | LPAREN oqlQuery RPAREN          // Scalar subquery
    | EXISTS LPAREN oqlQuery RPAREN   // EXISTS / NOT EXISTS subquery
    | ifThenElseExpression            // Inline if...then...else (Mendix expression)
    | caseExpression
    | castExpression                  // CAST(expr AS type) for OQL type conversion
    | listAggregateOperation          // COUNT, SUM, etc. on lists as expressions (must be before aggregateFunction)
    | listOperation                   // HEAD, TAIL, FIND, etc. as expressions
    | aggregateFunction               // SQL aggregate functions (COUNT, SUM, AVG, etc.) for OQL
    | functionCall
    | atomicExpression
    ;

caseExpression
    : CASE
      (WHEN expression THEN expression)+
      (ELSE expression)?
      END
    ;

/** Inline if-then-else expression (Mendix expression syntax):
 *  if condition then trueExpr else falseExpr
 *  Distinguished from statement-level IF...THEN...END IF by:
 *  - appearing inside an expression context (RHS of SET, DECLARE, etc.)
 *  - requiring ELSE clause (no standalone if-then)
 *  - no END IF terminator
 */
ifThenElseExpression
    : IF condition=expression THEN thenExpr=expression ELSE elseExpr=expression
    ;

/** CAST expression for OQL type conversion: CAST(expr AS type) */
castExpression
    : CAST LPAREN expression AS castDataType RPAREN
    ;

/** Data types supported by CAST in OQL */
castDataType
    : BOOLEAN_TYPE
    | DATETIME_TYPE
    | DECIMAL_TYPE
    | INTEGER_TYPE
    | LONG_TYPE
    | STRING_TYPE
    ;

aggregateFunction
    : (COUNT | SUM | AVG | MIN | MAX) LPAREN (DISTINCT? expression | STAR) RPAREN
    ;

functionCall
    : functionName LPAREN argumentList? RPAREN
    ;

/** Function names - includes identifiers and keywords that are valid function names */
functionName
    : IDENTIFIER
    | HYPHENATED_ID
    | TRUE           // true() function
    | FALSE          // false() function
    | CONTAINS       // contains(string, substring)
    | LENGTH         // length(string)
    | TRIM           // trim(string)
    | FIND           // find(list, condition)
    | FILTER         // filter(list, condition)
    | EMPTY          // empty(value)
    | COUNT          // count(list)
    | SUM            // sum(list, attribute)
    | AVG            // avg(list, attribute)
    | MIN            // min(list, attribute)
    | MAX            // max(list, attribute)
    ;

argumentList
    : expression (COMMA expression)*
    ;

atomicExpression
    : literal
    | VARIABLE (DOT attributeName)*    // $Var or $Widget.Attribute (data source ref)
    | qualifiedName
    | IDENTIFIER
    | MENDIX_TOKEN
    ;

expressionList
    : expression (COMMA expression)*
    ;

// =============================================================================
// COMMON RULES
// =============================================================================

/** Qualified name: Module.Entity or Module.Entity.Attribute */
qualifiedName
    : identifierOrKeyword (DOT identifierOrKeyword)*
    ;

/** An identifier that may be a keyword or a quoted name like "ComboBox" */
identifierOrKeyword
    : IDENTIFIER
    | QUOTED_IDENTIFIER
    | keyword
    ;

/** Literal values */
literal
    : STRING_LITERAL
    | NUMBER_LITERAL
    | booleanLiteral
    | NULL
    | EMPTY
    ;

arrayLiteral
    : LBRACKET (literal (COMMA literal)*)? RBRACKET
    ;

booleanLiteral
    : TRUE
    | FALSE
    ;

/** Documentation comment */
docComment
    : DOC_COMMENT
    ;

/** Annotation: @Name or @Name(params) or @Name value */
annotation
    : AT annotationName (LPAREN annotationParams RPAREN | annotationValue)?
    ;

annotationName
    : IDENTIFIER
    | POSITION
    | COMMENT
    | ICON
    | FOLDER
    | REQUIRED
    | CAPTION
    | ANNOTATION
    ;

annotationParams
    : annotationParam (COMMA annotationParam)*
    ;

annotationParam
    : IDENTIFIER COLON annotationValue   // Named parameter
    | annotationValue                     // Positional parameter
    ;

annotationValue
    : literal
    | expression
    | qualifiedName
    ;

/** Keywords that can be used as identifiers in certain contexts (module/entity names via qualifiedName,
 *  attribute names, enum values, parameter names, etc.).
 *  Every word-type lexer token must appear here so that user-defined names (entity, attribute,
 *  enum value, module) that happen to match a keyword can still be parsed.
 *  Maintain alphabetical order within each group for easy auditing.
 */
keyword
    // DDL / DML
    : ADD | ALTER | BATCH | CHANGE | CLOSE | COMMIT | CREATE | DECLARE | DELETE | DESCRIBE
    | DROP | EXECUTE | EXPORT | GENERATE | IMPORT | INSERT | INTO | MODIFY | MOVE | REFRESH
    | REMOVE | RENAME | REPLACE | RETRIEVE | RETURN | ROLLBACK | SET | UPDATE

    // Entity / Domain model
    | ASSOCIATION | ASSOCIATIONS | CALCULATED | CONSTANT | CONSTANTS | ENTITY | ENTITIES
    | ENUMERATION | ENUMERATIONS | GENERALIZATION | EXTENDS | INDEX | PERSISTENT
    | NON_PERSISTENT | REFERENCE | REFERENCE_SET | STORAGE | TABLE | UNIQUE
    | CASCADE | PREVENT | DELETE_BEHAVIOR | DELETE_AND_REFERENCES
    | DELETE_BUT_KEEP_REFERENCES | DELETE_IF_NO_REFERENCES
    | CHANGED | CREATED

    // Types
    | AUTONUMBER_TYPE | AUTOOWNER_TYPE | AUTOCHANGEDBY_TYPE | AUTOCREATEDDATE_TYPE | AUTOCHANGEDDATE_TYPE
    | BINARY_TYPE | BOOLEAN_TYPE | CURRENCY_TYPE | DATE_TYPE
    | DATETIME_TYPE | DECIMAL_TYPE | ENUM_TYPE | FLOAT_TYPE | HASHEDSTRING_TYPE
    | INTEGER_TYPE | LONG_TYPE | STRING_TYPE | STRINGTEMPLATE_TYPE

    // Module / project structure
    | ACTIONS | COLLECTION | FOLDER | LAYOUT | LAYOUTS | LOCAL | MODULE | MODULES
    | NOTEBOOK | NOTEBOOKS | PAGE | PAGES | PROJECT | SNIPPET | SNIPPETS
    | STORE | STRUCTURE | STRUCTURES | VIEW

    // Microflow / Nanoflow
    | MICROFLOW | MICROFLOWS | NANOFLOW | NANOFLOWS
    | BEGIN | END | IF | ELSE | ELSIF | ELSEIF | THEN | WHILE | LOOP
    | BREAK | CONTINUE | THROW | RAISE | CASE | WHEN
    | CALL | LOG | TRACE | WITH | FOR | TO | OF | RETURNING | RETURNS
    | NOTHING | EXPRESSION | JAVASCRIPT

    // Query / SQL
    | SELECT | FROM | WHERE | JOIN | LEFT | RIGHT | INNER | OUTER | FULL | CROSS
    | ORDER_BY | GROUP_BY | SORT_BY | HAVING | LIMIT | OFFSET | AS | ON
    | AND | OR | NOT | NULL | IN | LIKE | BETWEEN | TRUE | FALSE
    | COUNT | SUM | AVG | MIN | MAX | DISTINCT | ALL
    | ASC | DESC | UNION | INTERSECT | SUBTRACT | EXISTS
    | CAST | COALESCE | TRIM | LENGTH | CONTAINS | MATCH
    | AVERAGE | MINIMUM | MAXIMUM
    | IS_NULL | IS_NOT_NULL | NOT_NULL
    | HEAD | TAIL | FIND | SORT | EMPTY
    | LIST_OF | LIST_KW | EQUALS_OP

    // Control flow / microflow statements
    | BEGIN | END | IF | ELSE | ELSIF | THEN | WHILE | LOOP
    | DECLARE | SET | CHANGE | RETRIEVE | DELETE | COMMIT | RETURN
    | CALL | LOG | WITH | FOR | TO | OF
    | DESCRIBE | CLOSE | REPLACE | UPDATE | REFRESH | EXECUTE
    | INSERT | IMPORT | EXPORT | INTO | BATCH | NOTHING

    // Database / connection
    | CONNECT | CONNECTION | CONNECTIONS | DATABASE | DISCONNECT | QUERY
    | HOST | PORT | TOKEN | RUNTIME | BRANCH | INTROSPECT
    | SCHEMA | KEY | VALUES | RECORDS

    // Entity / structure
    | INDEX | UNIQUE | REFERENCE | CASCADE | STORAGE | TABLE
    | LOCAL | PROJECT | STRUCTURE | COLLECTION | STRUCTURES
    | LAYOUT | EXPRESSION | CHANGED | CREATED | GENERATE | FOLDER

    // Widget types
    | ACTIONBUTTON | CHECKBOX | COMBOBOX | CONTAINER | CONTROLBAR
    | CUSTOMCONTAINER | CUSTOMWIDGET | DATAGRID | DATEPICKER | DATAVIEW
    | DATEFILTER | DROPDOWN | DROPDOWNFILTER | DROPDOWNSORT | DYNAMICTEXT
    | FILEINPUT | GALLERY | GROUPBOX | IMAGE | IMAGEINPUT
    | INPUTREFERENCESETSELECTOR | LAYOUTGRID | LINKBUTTON | LISTVIEW
    | NAVIGATIONLIST | NUMBERFILTER | PLACEHOLDER | PLUGGABLEWIDGET
    | RADIOBUTTONS | REFERENCESELECTOR | SEARCHBAR | SNIPPETCALL
    | STATICIMAGE | STATICTEXT | DYNAMICIMAGE | TEXTAREA | TEXTBOX | TEXTFILTER
    | TABCONTAINER | TABPAGE | WIDGET | WIDGETS

    // Widget properties
    | ATTR | ATTRIBUTES | ATTRIBUTE | AUTOFILL | BINDS | BUTTONSTYLE
    | CAPTION | CAPTIONPARAMS | CLASS | COLUMN | COLUMNS | CONTENT | CONTENTPARAMS
    | DATASOURCE | DEFAULT | DESIGNPROPERTIES | DESKTOPWIDTH | DISPLAY | DOCUMENTATION
    | EDITABLE | FILTER | FILTERTYPE | HEADER | FOOTER
    | ICON | LABEL | ONCLICK | ONCHANGE | PARAMS | PASSING
    | PHONEWIDTH | TABLETWIDTH | READONLY | RENDERMODE | REQUIRED
    | SELECTION | STYLE | STYLING | TABINDEX | TITLE | TOOLTIP
    | URL | POSITION | VISIBLE | WIDTH | HEIGHT | WIDGETTYPE
    | VARIABLES_KW

    // Button actions
    | CALL_MICROFLOW | CALL_NANOFLOW | CANCEL_CHANGES | CLOSE_PAGE
    | CREATE_OBJECT | DELETE_ACTION | DELETE_OBJECT | OPEN_LINK
    | SAVECHANGES | SAVE_CHANGES | SHOW_PAGE | SIGN_OUT

    // Button styles / headings
    | BUTTON | PRIMARY | DANGER | CANCEL | INFO_STYLE | WARNING_STYLE
    | H1 | H2 | H3 | H4 | H5 | H6 | PARAGRAPH | ROW

    // Security
    | ACCESS | APPLY | AUTH | AUTHENTICATION | BASIC | DEMO
    | DESCRIPTION | GRANT | GUEST | LEVEL | MANAGE | MATRIX
    | OFF | OWNER | PASSWORD | PRODUCTION | PROTOTYPE
    | REVOKE | ROLE | ROLES | SECURITY | SESSION | USER | USERNAME | USERS

    // Validation
    | CONSTRAINT | FEEDBACK | PATTERN | RANGE | REGEX | RULE | VALIDATION | WITHOUT

    // Navigation
    | FOUND | HOME | HOMES | LOGIN | MENU_KW | NAVIGATION

    // Log levels
    | CRITICAL | DEBUG | ERROR | INFO | SUCCESS | WARNING

    // OData / REST / API
    | API | BASE | BODY | CHANNELS | CLIENT | CLIENTS | CONTRACT
    | DEPRECATED | EXPOSE | EXPOSED | EXTERNAL | HEADERS | JSON
    | MAP | MAPPING | MAPPINGS | MESSAGES | METHOD | NAMESPACE_KW
    | NOT_SUPPORTED | ODATA | OAUTH | OPERATION | PAGING
    | PARAMETER | PARAMETERS | PATH | PUBLISH | PUBLISHED
    | REQUEST | RESOURCE | RESPONSE | REST | SEND | SERVICE | SERVICES
    | SOURCE_KW | TIMEOUT | VERSION | XML
    | FILE_KW | LINK | DYNAMIC

    // HTTP methods
    | GET | POST | PUT | PATCH

    // Workflow
    | ABORT | ACTIVITY | ANNOTATION | BOUNDARY | BY | COMPLETE_TASK
    | CONDITION | DATE | DECISION | DUE | INTERRUPTING | JUMP
    | LOCK | MULTI | NODE | NON | NOTIFICATION | NOTIFY
    | OPEN | OUTCOME | OUTCOMES | OVERVIEW | PARALLEL | PAUSE
    | REASON | RESTART | RETRY | SPLIT | TARGETING | TASK | TIMER
    | UNLOCK | UNPAUSE | WAIT | WORKFLOW | WORKFLOWS

    // Business events / settings
    | BUSINESS | CONFIGURATION | EVENT | EVENTS | HANDLER | SETTINGS | SUBSCRIBE

    // Code search / analysis
    | BACKGROUND | CALLERS | CALLEES | DEPTH | IMPACT | REFERENCES
    | SEARCH | TRANSITIVE

    // CLI commands
    | BUILD | CATALOG | CHECK | CLEAR | COMMENT | CUSTOM_NAME_MAP
    | DESIGN | DRY | EXEC | FEATURES | ADDED | SINCE | FORCE
    | LANGUAGES | LINT | PROPERTIES | READ | RULES | RUN | SARIF | SCRIPT
    | SHOW | USE | STATUS | WRITE | VIA | VIEWS | TABLES

    // Fragment / ALTER PAGE
    | AFTER | BEFORE | DEFINE | FRAGMENT | FRAGMENTS

    // General-purpose words (only tokens not already listed above)
    | ACTION | BOTH | CONTEXT | DATA | FORMAT | ITEM | LIST
    | MESSAGE | MOD | DIV | MULTIPLE | NONE | OBJECT | OBJECTS
    | SINGLE | SQL | TEMPLATE | TEXT | TYPE | VALUE

    // Import/Export mapping / SQL generate
    | ATTRIBUTE_NAME | CONNECTOR | MEMBERS | OVER | JAVA | XPATH
    ;
