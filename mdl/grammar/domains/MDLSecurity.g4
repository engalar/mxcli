/**
 * MDL Security Grammar — security statements (module roles, user roles,
 * grants, revokes, project security, demo users).
 */
parser grammar MDLSecurity;

options { tokenVocab = MDLLexer; }

// =============================================================================
// SECURITY STATEMENTS
// =============================================================================

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

grantNanoflowAccessStatement
    : GRANT EXECUTE ON NANOFLOW qualifiedName TO moduleRoleList
    ;

revokeNanoflowAccessStatement
    : REVOKE EXECUTE ON NANOFLOW qualifiedName FROM moduleRoleList
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
    | ALTER PROJECT SECURITY PASSWORD POLICY LPAREN passwordPolicyOptionList RPAREN
    ;

passwordPolicyOptionList
    : passwordPolicyOption (COMMA passwordPolicyOption)*
    ;

// Options use identifierOrKeyword keys so no new reserved tokens are needed.
// Accepted keys: min_length, require_digit, require_mixed_case, require_symbol
passwordPolicyOption
    : identifierOrKeyword COLON (NUMBER_LITERAL | TRUE | FALSE)
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
    | READ LPAREN grantMember (COMMA grantMember)* RPAREN
    | WRITE STAR
    | WRITE LPAREN grantMember (COMMA grantMember)* RPAREN
    ;

// A member name in a GRANT attribute/association list.  Uses qualifiedName so that
// module-qualified association names (e.g. HD.Ticket_Customer) are accepted alongside
// plain attribute names (e.g. Status).  QUOTED_IDENTIFIER supported via identifierOrKeyword.
grantMember
    : qualifiedName
    ;
