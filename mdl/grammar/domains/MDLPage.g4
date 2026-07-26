/**
 * MDL Page Grammar — pages, snippets, shared page/snippet rules, xpath expressions,
 * page V3 syntax, notebooks.
 */
parser grammar MDLPage;

options { tokenVocab = MDLLexer; }

// =============================================================================
// PAGE CREATION
// =============================================================================

/**
 * Creates a new page with layout, parameters, and widget content.
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
// LAYOUT CREATION / MODIFICATION
// =============================================================================

createLayoutStatement
    : LAYOUT qualifiedName
      (LPAREN layoutHeaderProperty (COMMA layoutHeaderProperty)* RPAREN)?
      (LBRACE layoutWidget* RBRACE)?
    ;

layoutHeaderProperty
    : TYPE COLON STRING_LITERAL
    | TYPE COLON identifierOrKeyword
    | FOLDER COLON STRING_LITERAL
    ;

layoutWidget
    : SCROLLCONTAINER identifierOrKeyword
      (LPAREN layoutScrollContainerProp (COMMA layoutScrollContainerProp)* RPAREN)?
      LBRACE layoutRegion* RBRACE
    | PLACEHOLDER identifierOrKeyword
    ;

layoutScrollContainerProp
    : identifierOrKeyword COLON identifierOrKeyword
    ;

layoutRegion
    : layoutRegionName LBRACE layoutRegionContent* RBRACE
    ;

layoutRegionName
    : CENTER | TOP | BOTTOM | LEFT | RIGHT
    ;

layoutRegionContent
    : PLACEHOLDER identifierOrKeyword
    | widgetV3
    ;

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

/** Qualified name in XPath context: accepts any keyword as identifier part. */
xpathQualifiedName
    : xpathWord (DOT xpathWord)*
    ;

/** Any single-word token that can appear as part of a name in XPath. */
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
    | FILEINPUT
    | IMAGEINPUT
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
    | SELECTION COLON selectionModeV3                 // Selection: Single | Multiple
    | SNIPPET COLON qualifiedName                     // Snippet: Module.SnippetName
    | PARAMS COLON snippetCallParamListV3             // Params: {$Asset: $var} — snippet call parameter mappings
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
    | IDENTIFIER COLON actionExprV3                   // Named action: onChange: nanoflow Module.NF
    | IDENTIFIER COLON propertyValueV3                // Generic: any other property
    | keyword COLON actionExprV3                      // Named action (keyword key): onCommit: nanoflow ...
    | keyword COLON propertyValueV3                  // Generic: keyword as property name (for pluggable widgets)
    ;

// Filter type values - handle keywords like CONTAINS that are also filter types
filterTypeValue
    : CONTAINS      // contains
    | EMPTY         // empty
    | IDENTIFIER    // startsWith, endsWith, greater, greaterEqual, equal, notEqual, smaller, smallerEqual, notEmpty
    ;

// Snippet call parameter mappings: {$Asset: $var, $Other: $other}
snippetCallParamListV3
    : LBRACE snippetCallParamMappingV3 (COMMA snippetCallParamMappingV3)* RBRACE
    ;

snippetCallParamMappingV3
    : (identifierOrKeyword | VARIABLE) COLON VARIABLE
    ;

// V3 Attribute list for filter widgets
attributeListV3
    : LBRACKET qualifiedName (COMMA qualifiedName)* RBRACKET
    ;

// V3 DataSource expressions
dataSourceExprV3
    : VARIABLE SLASH associationPathV3                // $currentObject/Module.Assoc (ByAssociation — sugar for ASSOCIATION)
    | VARIABLE                                        // $ParamName
    | DATABASE FROM? qualifiedName                    // DATABASE [FROM] Entity [WHERE ...] [SORT BY ...]
      (WHERE (xpathConstraint (andOrXpath? xpathConstraint)* | expression))?
      (SORT_BY sortColumn (COMMA sortColumn)*)?
    | MICROFLOW qualifiedName microflowArgsV3?        // MICROFLOW Module.Flow
    | NANOFLOW qualifiedName microflowArgsV3?         // NANOFLOW Module.Flow
    | ASSOCIATION associationPathV3                   // ASSOCIATION Module.Assoc (explicit form)
    | SELECTION IDENTIFIER                            // SELECTION widgetName
    ;

// Association path: Module.Assoc or Module.Assoc/Module.Entity or multi-step
associationPathV3
    : qualifiedName (SLASH qualifiedName)*
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
    | MICROFLOW qualifiedName microflowArgsV3? (CLOSE_PAGE)?   // MICROFLOW Module.Flow [close_page]
    | NANOFLOW qualifiedName microflowArgsV3? (CLOSE_PAGE)?    // NANOFLOW Module.Flow [close_page]
    | OPEN_LINK STRING_LITERAL                        // OPEN_LINK 'https://...'
    | SIGN_OUT                                        // SIGN_OUT
    | COMPLETE_TASK STRING_LITERAL                    // COMPLETE_TASK 'OutcomeName'
    ;

// V3 Microflow arguments: (Param: value, ...)
microflowArgsV3
    : LPAREN microflowArgV3 (COMMA microflowArgV3)* RPAREN
    ;

microflowArgV3
    : (IDENTIFIER | keyword) COLON expression        // Param: $value (canonical, keyword names allowed)
    | VARIABLE EQUALS expression                     // $Param = $value (microflow-style, also accepted)
    ;

// V3 Attribute path: Name, Product/Category, "Order" (quoted to escape reserved words)
attributePathV3
    : (IDENTIFIER | QUOTED_IDENTIFIER | keyword) (SLASH (IDENTIFIER | QUOTED_IDENTIFIER | keyword))*
    ;

// V3 String expression (may include template placeholders or attribute binding)
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
      LBRACE notebookPage* RBRACE
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
