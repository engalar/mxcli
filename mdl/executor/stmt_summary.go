// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// stmtTypeName returns the short type name of a statement (without package prefix).
func stmtTypeName(stmt ast.Statement) string {
	t := fmt.Sprintf("%T", stmt)
	// Remove "*ast." prefix
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}

// stmtSummary returns a safe one-line summary of a statement for logging.
func stmtSummary(stmt ast.Statement) string {
	switch s := stmt.(type) {
	// Connection
	case *ast.ConnectStmt:
		return fmt.Sprintf("CONNECT LOCAL '%s'", s.Path)
	case *ast.DisconnectStmt:
		return "DISCONNECT"
	case *ast.StatusStmt:
		return "STATUS"

	// Module
	case *ast.CreateModuleStmt:
		return fmt.Sprintf("CREATE MODULE %s", s.Name)
	case *ast.DropModuleStmt:
		return fmt.Sprintf("DROP MODULE %s", s.Name)

	// Entity
	case *ast.CreateEntityStmt:
		return fmt.Sprintf("CREATE ENTITY %s", s.Name)
	case *ast.CreateViewEntityStmt:
		return fmt.Sprintf("CREATE VIEW ENTITY %s", s.Name)
	case *ast.DropEntityStmt:
		return fmt.Sprintf("DROP ENTITY %s", s.Name)

	// Association
	case *ast.CreateAssociationStmt:
		return fmt.Sprintf("CREATE ASSOCIATION %s", s.Name)
	case *ast.DropAssociationStmt:
		return fmt.Sprintf("DROP ASSOCIATION %s", s.Name)

	// Enumeration
	case *ast.CreateEnumerationStmt:
		return fmt.Sprintf("CREATE ENUMERATION %s", s.Name)
	case *ast.AlterEnumerationStmt:
		return fmt.Sprintf("ALTER ENUMERATION %s", s.Name)
	case *ast.DropEnumerationStmt:
		return fmt.Sprintf("DROP ENUMERATION %s", s.Name)

	// Microflow
	case *ast.CreateMicroflowStmt:
		return fmt.Sprintf("CREATE MICROFLOW %s", s.Name)
	case *ast.DropMicroflowStmt:
		return fmt.Sprintf("DROP MICROFLOW %s", s.Name)

	// Page
	case *ast.CreatePageStmtV3:
		return fmt.Sprintf("CREATE PAGE %s", s.Name)
	case *ast.DropPageStmt:
		return fmt.Sprintf("DROP PAGE %s", s.Name)
	case *ast.CreateSnippetStmtV3:
		return fmt.Sprintf("CREATE SNIPPET %s", s.Name)
	case *ast.DropSnippetStmt:
		return fmt.Sprintf("DROP SNIPPET %s", s.Name)

	// Java actions
	case *ast.CreateJavaActionStmt:
		return fmt.Sprintf("CREATE JAVA ACTION %s", s.Name)
	case *ast.DropJavaActionStmt:
		return fmt.Sprintf("DROP JAVA ACTION %s", s.Name)

	// Move
	case *ast.MoveStmt:
		return fmt.Sprintf("MOVE %s %s", s.DocumentType, s.Name)

	// Security
	case *ast.CreateModuleRoleStmt:
		return fmt.Sprintf("CREATE MODULE ROLE %s", s.Name)
	case *ast.DropModuleRoleStmt:
		return fmt.Sprintf("DROP MODULE ROLE %s", s.Name)
	case *ast.CreateUserRoleStmt:
		return fmt.Sprintf("CREATE USER ROLE %s", s.Name)
	case *ast.DropUserRoleStmt:
		return fmt.Sprintf("DROP USER ROLE %s", s.Name)
	case *ast.GrantMicroflowAccessStmt:
		return fmt.Sprintf("GRANT EXECUTE ON MICROFLOW %s", s.Microflow)
	case *ast.RevokeMicroflowAccessStmt:
		return fmt.Sprintf("REVOKE EXECUTE ON MICROFLOW %s", s.Microflow)
	case *ast.GrantPageAccessStmt:
		return fmt.Sprintf("GRANT VIEW ON PAGE %s", s.Page)
	case *ast.RevokePageAccessStmt:
		return fmt.Sprintf("REVOKE VIEW ON PAGE %s", s.Page)
	case *ast.GrantWorkflowAccessStmt:
		return fmt.Sprintf("GRANT EXECUTE ON WORKFLOW %s", s.Workflow)
	case *ast.RevokeWorkflowAccessStmt:
		return fmt.Sprintf("REVOKE EXECUTE ON WORKFLOW %s", s.Workflow)
	case *ast.GrantEntityAccessStmt:
		return fmt.Sprintf("GRANT ON ENTITY %s", s.Entity)
	case *ast.RevokeEntityAccessStmt:
		return fmt.Sprintf("REVOKE ON ENTITY %s", s.Entity)
	case *ast.AlterProjectSecurityStmt:
		return "ALTER PROJECT SECURITY"
	case *ast.CreateDemoUserStmt:
		return fmt.Sprintf("CREATE DEMO USER %s", s.UserName)
	case *ast.DropDemoUserStmt:
		return fmt.Sprintf("DROP DEMO USER %s", s.UserName)
	case *ast.CreateExternalEntityStmt:
		return fmt.Sprintf("CREATE EXTERNAL ENTITY %s", s.Name)
	case *ast.GrantODataServiceAccessStmt:
		return fmt.Sprintf("GRANT ACCESS ON ODATA SERVICE %s", s.Service)
	case *ast.RevokeODataServiceAccessStmt:
		return fmt.Sprintf("REVOKE ACCESS ON ODATA SERVICE %s", s.Service)

	// Database Connection
	case *ast.CreateDatabaseConnectionStmt:
		return fmt.Sprintf("CREATE DATABASE CONNECTION %s", s.Name)

	// Business Events
	case *ast.CreateBusinessEventServiceStmt:
		return fmt.Sprintf("CREATE BUSINESS EVENT SERVICE %s", s.Name)
	case *ast.DropBusinessEventServiceStmt:
		return fmt.Sprintf("DROP BUSINESS EVENT SERVICE %s", s.Name)

	// Settings
	case *ast.AlterSettingsStmt:
		return fmt.Sprintf("ALTER SETTINGS %s", s.Section)

	// Navigation
	case *ast.AlterNavigationStmt:
		return fmt.Sprintf("CREATE NAVIGATION %s", s.ProfileName)

	// Query
	case *ast.ShowStmt:
		summary := fmt.Sprintf("SHOW %s", s.ObjectType)
		if s.Name != nil {
			summary += " " + s.Name.String()
		}
		if s.InModule != "" {
			summary += " IN " + s.InModule
		}
		return summary
	case *ast.DescribeStmt:
		return fmt.Sprintf("DESCRIBE %v %s", s.ObjectType, s.Name)
	case *ast.SelectStmt:
		return "SELECT ..."
	case *ast.SearchStmt:
		return fmt.Sprintf("SEARCH '%s'", s.Query)
	case *ast.ShowWidgetsStmt:
		return "SHOW WIDGETS"
	case *ast.UpdateWidgetsStmt:
		return "UPDATE WIDGETS"
	case *ast.ShowDesignPropertiesStmt:
		if s.WidgetType != "" {
			return fmt.Sprintf("SHOW DESIGN PROPERTIES FOR %s", s.WidgetType)
		}
		return "SHOW DESIGN PROPERTIES"
	case *ast.DescribeStylingStmt:
		summary := fmt.Sprintf("DESCRIBE STYLING ON %s %s", s.ContainerType, s.ContainerName)
		if s.WidgetName != "" {
			summary += " WIDGET " + s.WidgetName
		}
		return summary
	case *ast.AlterStylingStmt:
		return fmt.Sprintf("ALTER STYLING ON %s %s WIDGET %s", s.ContainerType, s.ContainerName, s.WidgetName)

	// ALTER PAGE / ALTER SNIPPET
	case *ast.AlterPageStmt:
		ct := s.ContainerType
		if ct == "" {
			ct = "PAGE"
		}
		return fmt.Sprintf("ALTER %s %s", ct, s.PageName)

	// Fragments
	case *ast.DefineFragmentStmt:
		return fmt.Sprintf("DEFINE FRAGMENT %s", s.Name)
	case *ast.DescribeFragmentFromStmt:
		return fmt.Sprintf("DESCRIBE FRAGMENT FROM %s %s WIDGET %s", s.ContainerType, s.ContainerName, s.WidgetName)

	// SQL
	case *ast.SQLConnectStmt:
		return fmt.Sprintf("SQL CONNECT %s AS %s", s.Driver, s.Alias)
	case *ast.SQLDisconnectStmt:
		return fmt.Sprintf("SQL DISCONNECT %s", s.Alias)
	case *ast.SQLConnectionsStmt:
		return "SQL CONNECTIONS"
	case *ast.SQLQueryStmt:
		q := s.Query
		if len(q) > 40 {
			q = q[:40] + "..."
		}
		return fmt.Sprintf("SQL %s %s", s.Alias, q)
	case *ast.SQLShowTablesStmt:
		return fmt.Sprintf("SQL %s SHOW TABLES", s.Alias)
	case *ast.SQLShowViewsStmt:
		return fmt.Sprintf("SQL %s SHOW VIEWS", s.Alias)
	case *ast.SQLShowFunctionsStmt:
		return fmt.Sprintf("SQL %s SHOW FUNCTIONS", s.Alias)
	case *ast.SQLDescribeTableStmt:
		return fmt.Sprintf("SQL %s DESCRIBE %s", s.Alias, s.Table)
	case *ast.SQLGenerateConnectorStmt:
		return fmt.Sprintf("SQL %s GENERATE CONNECTOR INTO %s", s.Alias, s.Module)

	// Import
	case *ast.ImportStmt:
		summary := fmt.Sprintf("IMPORT FROM %s INTO %s (%d mappings", s.SourceAlias, s.TargetEntity, len(s.Mappings))
		if len(s.Links) > 0 {
			summary += fmt.Sprintf(", %d links", len(s.Links))
		}
		return summary + ")"

	// Repository
	case *ast.RefreshCatalogStmt:
		return "REFRESH CATALOG"
	case *ast.RefreshStmt:
		return "REFRESH"
	// Session
	case *ast.ExitStmt:
		return "EXIT"
	case *ast.HelpStmt:
		return "HELP"
	case *ast.ExecuteScriptStmt:
		return fmt.Sprintf("EXECUTE '%s'", s.Path)
	case *ast.LintStmt:
		return "LINT"

	default:
		return stmtTypeName(stmt)
	}
}
