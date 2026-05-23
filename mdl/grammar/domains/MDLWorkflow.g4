/**
 * MDL Workflow Grammar — CREATE WORKFLOW, ALTER WORKFLOW.
 */
parser grammar MDLWorkflow;

options { tokenVocab = MDLLexer; }

// =============================================================================
// CREATE WORKFLOW
// =============================================================================

/**
 * Create a workflow with activities.
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
    : USER TASK (IDENTIFIER | QUOTED_IDENTIFIER) STRING_LITERAL
      (PAGE qualifiedName)?
      (TARGETING (USERS | GROUPS)? MICROFLOW qualifiedName)?
      (TARGETING (USERS | GROUPS)? XPATH STRING_LITERAL)?
      (ENTITY qualifiedName)?
      (DUE DATE_TYPE STRING_LITERAL)?
      (DESCRIPTION STRING_LITERAL)?
      (OUTCOMES workflowUserTaskOutcome+)?
      (BOUNDARY EVENT workflowBoundaryEventClause+)?
    | MULTI USER TASK (IDENTIFIER | QUOTED_IDENTIFIER) STRING_LITERAL
      (PAGE qualifiedName)?
      (TARGETING (USERS | GROUPS)? MICROFLOW qualifiedName)?
      (TARGETING (USERS | GROUPS)? XPATH STRING_LITERAL)?
      (ENTITY qualifiedName)?
      (DUE DATE_TYPE STRING_LITERAL)?
      (DESCRIPTION STRING_LITERAL)?
      (COMPLETION METHOD workflowCompletionMethod)?
      (OUTCOMES workflowUserTaskOutcome+)?
      (BOUNDARY EVENT workflowBoundaryEventClause+)?
    ;

workflowCompletionMethod
    : MAJORITY
    | THRESHOLD NUMBER_LITERAL PERCENT?
    | CONSENSUS
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
    : JUMP TO (IDENTIFIER | QUOTED_IDENTIFIER) (COMMENT STRING_LITERAL)?
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
    : identifierOrKeyword (AT NUMBER_LITERAL)?
    | STRING_LITERAL (AT NUMBER_LITERAL)?
    ;
