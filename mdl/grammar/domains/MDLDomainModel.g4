/**
 * MDL Domain Model Grammar — entities, associations, enumerations, modules,
 * constants, validation rules, JSON structures, mappings, image collections,
 * index creation, data transformer.
 */
parser grammar MDLDomainModel;

options { tokenVocab = MDLLexer; }

// =============================================================================
// ENTITY / ASSOCIATION CREATION
// =============================================================================

/**
 * Creates a new entity in the domain model.
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
// ALTER ENTITY / ASSOCIATION / ENUMERATION / NOTEBOOK ACTIONS
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
    | SET ALLOW_CREATE_CHANGE_LOCALLY EQUALS (TRUE | FALSE)
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
    : MODULE identifierOrKeyword moduleOptions?
    ;

// =============================================================================
// ALTER MODULE — JAR DEPENDENCY MANAGEMENT
// =============================================================================

alterModuleJarDepStatement
    : ALTER MODULE (qualifiedName | IDENTIFIER) alterModuleJarDepAction+
    ;

alterModuleJarDepAction
    : ADD JAR DEPENDENCY LPAREN jarDepProperty (COMMA jarDepProperty)* COMMA? RPAREN
    | SET JAR DEPENDENCY STRING_LITERAL VERSION STRING_LITERAL
    | SET JAR DEPENDENCY STRING_LITERAL INCLUDED booleanLiteral
    | SET JAR DEPENDENCY STRING_LITERAL ADD EXCLUSION STRING_LITERAL
    | SET JAR DEPENDENCY STRING_LITERAL DROP EXCLUSION STRING_LITERAL
    | DROP JAR DEPENDENCY STRING_LITERAL
    ;

jarDepProperty
    : identifierOrKeyword EQUALS (STRING_LITERAL | booleanLiteral)
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

// =============================================================================
// IMPORT / EXPORT MAPPING CREATION
// =============================================================================

/**
 * CREATE IMPORT MAPPING Module.Name
 *   WITH JSON STRUCTURE Module.JsonStructure
 * {
 *   CREATE Module.Entity {
 *     PetId = id KEY,
 *     Name = name,
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
 * {
 *   Module.Entity {
 *     jsonField = Attr,
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
// CONSTANT CREATION
// =============================================================================

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

// =============================================================================
// INDEX CREATION (standalone)
// =============================================================================

createIndexStatement
    : INDEX IDENTIFIER ON qualifiedName LPAREN indexAttributeList RPAREN
    ;

// =============================================================================
// DATA TRANSFORMER
// =============================================================================

/**
 * CREATE DATA TRANSFORMER Module.Name
 * SOURCE JSON '{"latitude": 51.916, ...}'
 * {
 *   JSLT '{ "lat": .latitude }';
 * };
 */
createDataTransformerStatement
    : DATA TRANSFORMER qualifiedName
      SOURCE_KW (JSON | XML) STRING_LITERAL
      LBRACE dataTransformerStep* RBRACE
    ;

dataTransformerStep
    : (JSLT | XSLT) (STRING_LITERAL | DOLLAR_STRING) SEMICOLON?
    ;
