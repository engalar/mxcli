/**
 * MDL (Mendix Definition Language) Parser Grammar
 *
 * ANTLR4 parser for MDL syntax used by the Mendix REPL.
 * Converted from Chevrotain-based parser.
 *
 * This master file contains only the top-level dispatch rules.
 * Domain-specific rules live in domains/ and are merged at compile time
 * via ANTLR4's `import` directive.
 */
parser grammar MDLParser;

options {
    tokenVocab = MDLLexer;
}

import
    MDLDomainModel,
    MDLMicroflow,
    MDLPage,
    MDLSecurity,
    MDLAgent,
    MDLWorkflow,
    MDLService,
    MDLCatalog,
    MDLSettings;

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
      | createDataTransformerStatement
      | createModelStatement
      | createConsumedMCPServiceStatement
      | createKnowledgeBaseStatement
      | createAgentStatement
      | createNanoflowStatement
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
    | alterModuleJarDepStatement
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
    | DROP DATA TRANSFORMER qualifiedName
    | DROP MODEL qualifiedName                               // DROP MODEL Module.Name (agent-editor)
    | DROP CONSUMED MCP SERVICE qualifiedName                // DROP CONSUMED MCP SERVICE Module.Name
    | DROP KNOWLEDGE BASE qualifiedName                      // DROP KNOWLEDGE BASE Module.Name
    | DROP AGENT qualifiedName                               // DROP AGENT Module.Name
    | DROP CONFIGURATION STRING_LITERAL
    | DROP FOLDER STRING_LITERAL IN (qualifiedName | IDENTIFIER)
    ;

renameStatement
    : RENAME renameTarget qualifiedName TO identifierOrKeyword (DRY RUN)?
    | RENAME MODULE identifierOrKeyword TO identifierOrKeyword (DRY RUN)?
    ;

renameTarget
    : ENTITY | MICROFLOW | NANOFLOW | PAGE | ENUMERATION | ASSOCIATION | CONSTANT | JAVA ACTION | WORKFLOW
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
// SECURITY STATEMENTS (dispatch list — rules in MDLSecurity.g4)
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
    | grantNanoflowAccessStatement
    | revokeNanoflowAccessStatement
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

// =============================================================================
// DQL STATEMENTS (Data Query Language) — dispatch
// =============================================================================

dqlStatement
    : showStatement
    | describeStatement
    | catalogSelectQuery
    | oqlQuery
    ;
