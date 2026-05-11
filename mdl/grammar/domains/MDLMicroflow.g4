/**
 * MDL Microflow Grammar — microflows, nanoflows, Java actions, all microflow
 * body statements, REST call statements, list operations.
 */
parser grammar MDLMicroflow;

options { tokenVocab = MDLLexer; }

// =============================================================================
// MICROFLOW CREATION
// =============================================================================

/**
 * Creates a new microflow with parameters, return type, and activity body.
 */
createMicroflowStatement
    : MICROFLOW qualifiedName
      LPAREN microflowParameterList? RPAREN
      microflowReturnType?
      microflowOptions?
      BEGIN microflowBody END SEMICOLON? SLASH?
    ;

/**
 * Nanoflow creation — mirrors microflow syntax but targets client-side execution.
 */
createNanoflowStatement
    : NANOFLOW qualifiedName
      LPAREN microflowParameterList? RPAREN
      microflowReturnType?
      microflowOptions?
      BEGIN microflowBody END SEMICOLON? SLASH?
    ;

/**
 * Java Action creation with inline Java source code.
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

/**
 * Body shared by both microflow and nanoflow creation.
 * CALL NANOFLOW is valid in both contexts (microflows can call nanoflows).
 * Nanoflow-specific action restrictions are enforced at the executor level,
 * not at the grammar level.
 */
microflowStatement
    : annotation* declareStatement SEMICOLON?
    | annotation* caseStatement SEMICOLON?
    | annotation* inheritanceSplitStatement SEMICOLON?
    | annotation* castObjectStatement SEMICOLON?
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
    | annotation* callNanoflowStatement SEMICOLON?
    | annotation* callJavaActionStatement SEMICOLON?
    | annotation* callJavaScriptActionStatement SEMICOLON?
    | annotation* callWebServiceStatement SEMICOLON?
    | annotation* executeDatabaseQueryStatement SEMICOLON?
    | annotation* callExternalActionStatement SEMICOLON?
    | annotation* showPageStatement SEMICOLON?
    | annotation* closePageStatement SEMICOLON?
    | annotation* showHomePageStatement SEMICOLON?
    | annotation* showMessageStatement SEMICOLON?
    | annotation* downloadFileStatement SEMICOLON?
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
    | annotation* transformJsonStatement SEMICOLON?
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

caseStatement
    : CASE enumSplitSource
      (WHEN enumSplitCaseValue (COMMA enumSplitCaseValue)* THEN microflowBody)+
      (ELSE microflowBody)?
      END CASE
    ;

enumSplitSource
    : attributePath
    | VARIABLE
    ;

enumSplitCaseValue
    : identifierOrKeyword
    | LPAREN EMPTY RPAREN
    ;

inheritanceSplitStatement
    : SPLIT TYPE VARIABLE
      (inheritanceSplitCase+ (ELSE microflowBody)? END SPLIT)?
    ;

inheritanceSplitCase
    : CASE qualifiedName microflowBody
    ;

castObjectStatement
    : CAST VARIABLE
    | VARIABLE EQUALS CAST VARIABLE
    ;

setStatement
    : SET (VARIABLE | attributePath) EQUALS expression
    ;

// $NewProduct = CREATE MfTest.Product (Name = $Name, Code = $Code);
createObjectStatement
    : (VARIABLE EQUALS)? CREATE nonListDataType (LPAREN memberAssignmentList? RPAREN)? onErrorClause?
    ;

// CHANGE $Product (Name = $NewName, ModifiedDate = [%CurrentDateTime%]);
changeObjectStatement
    : CHANGE VARIABLE (LPAREN memberAssignmentList? RPAREN)? REFRESH?
    ;

// Shared by SET, LOOP, aggregate expressions, and validation feedback targets.
attributePath
    : VARIABLE ((SLASH | DOT) qualifiedName)+
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
    : LOG logLevel? (NODE expression)? expression logTemplateParams?
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

callNanoflowStatement
    : (VARIABLE EQUALS)? CALL NANOFLOW qualifiedName LPAREN callArgumentList? RPAREN onErrorClause?
    ;

// $Result = CALL JAVA ACTION CustomActivities.ExecuteOQL(OqlStatement = '...');
callJavaActionStatement
    : (VARIABLE EQUALS)? CALL JAVA ACTION qualifiedName LPAREN callArgumentList? RPAREN onErrorClause?
    ;

// $Result = CALL JAVASCRIPT ACTION Module.JSAction(Param = 'value');
callJavaScriptActionStatement
    : (VARIABLE EQUALS)? CALL JAVASCRIPT ACTION qualifiedName LPAREN callArgumentList? RPAREN onErrorClause?
    ;

// Legacy SOAP call.
callWebServiceStatement
    : (VARIABLE EQUALS)? CALL WEB SERVICE
      (RAW STRING_LITERAL
      | webServiceReference
        (OPERATION webServiceReference)?
        (SEND MAPPING webServiceReference)?
        (RECEIVE MAPPING webServiceReference)?
        (TIMEOUT expression)?)
      onErrorClause?
    ;

webServiceReference
    : qualifiedName
    | STRING_LITERAL
    ;

// $Result = EXECUTE DATABASE QUERY Module.Connection.QueryName (param = 'value');
executeDatabaseQueryStatement
    : (VARIABLE EQUALS)? EXECUTE DATABASE QUERY qualifiedName
      (DYNAMIC (STRING_LITERAL | DOLLAR_STRING | expression))?
      (LPAREN callArgumentList? RPAREN)?
      (CONNECTION LPAREN callArgumentList? RPAREN)?
      onErrorClause?
    ;

// $Result = CALL EXTERNAL ACTION Module.ODataClient.ActionName(Param = $value);
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

downloadFileStatement
    : DOWNLOAD FILE_KW VARIABLE (SHOW IN BROWSER)? onErrorClause?
    ;

throwStatement
    : THROW expression
    ;

// VALIDATION FEEDBACK $Product/Code MESSAGE 'Product code cannot be empty';
validationFeedbackStatement
    : VALIDATION FEEDBACK (attributePath | VARIABLE) MESSAGE expression (OBJECTS LBRACKET expressionList RBRACKET)?
    ;

// =============================================================================
// REST CALL STATEMENTS
// =============================================================================

/**
 * REST call statement for making HTTP requests to external APIs.
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
    : RETURNS STRING_TYPE                                       // Return as string
    | RETURNS RESPONSE                                          // Return HttpResponse object
    | RETURNS MAPPING qualifiedName AS LIST_OF qualifiedName    // Import mapping → list result
    | RETURNS MAPPING qualifiedName AS qualifiedName            // Import mapping → single object
    | RETURNS NONE                                              // Ignore response
    | RETURNS NOTHING                                           // Ignore response (alias)
    ;

/**
 * SEND REST REQUEST — calls a consumed REST service operation defined via CREATE REST CLIENT.
 */
sendRestRequestStatement
    : (VARIABLE EQUALS)? SEND REST REQUEST qualifiedName
      sendRestRequestWithClause?
      sendRestRequestBodyClause?
      onErrorClause?
    ;

sendRestRequestWithClause
    : WITH LPAREN sendRestRequestParam (COMMA sendRestRequestParam)* RPAREN
    ;

sendRestRequestParam
    : VARIABLE EQUALS expression
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

/**
 * Transform JSON: $Result = TRANSFORM $Input WITH Module.Transformer;
 */
transformJsonStatement
    : (VARIABLE EQUALS)? TRANSFORM VARIABLE WITH qualifiedName
      onErrorClause?
    ;

// =============================================================================
// LIST OPERATIONS
// =============================================================================

/**
 * List operations that return a single item or a modified list.
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
    | RANGE LPAREN VARIABLE (COMMA expression (COMMA expression)?)? RPAREN // $var = RANGE($list, offset, limit)
    ;

sortSpecList
    : sortSpec (COMMA sortSpec)*
    ;

sortSpec
    : IDENTIFIER (ASC | DESC)?
    ;

/**
 * Aggregate operations on lists.
 */
aggregateListStatement
    : VARIABLE EQUALS listAggregateOperation
    ;

listAggregateOperation
    : COUNT LPAREN VARIABLE RPAREN                                                    // $count = COUNT($list)
    | SUM LPAREN VARIABLE COMMA expression RPAREN                                     // $sum = SUM($list, expr)
    | SUM LPAREN attributePath RPAREN                                                 // $sum = SUM($list.attr)
    | AVERAGE LPAREN VARIABLE COMMA expression RPAREN                                 // $avg = AVERAGE($list, expr)
    | AVERAGE LPAREN attributePath RPAREN                                             // $avg = AVERAGE($list.attr)
    | MINIMUM LPAREN VARIABLE COMMA expression RPAREN                                 // $min = MINIMUM($list, expr)
    | MINIMUM LPAREN attributePath RPAREN                                             // $min = MINIMUM($list.attr)
    | MAXIMUM LPAREN VARIABLE COMMA expression RPAREN                                 // $max = MAXIMUM($list, expr)
    | MAXIMUM LPAREN attributePath RPAREN                                             // $max = MAXIMUM($list.attr)
    ;

/**
 * Create an empty list of a specific entity type.
 */
createListStatement
    : VARIABLE EQUALS CREATE LIST_OF qualifiedName
    ;

/**
 * Add an item to a list.
 */
addToListStatement
    : ADD expression TO VARIABLE
    ;

/**
 * Remove an item from a list.
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
