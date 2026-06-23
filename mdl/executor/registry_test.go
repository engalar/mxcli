package executor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/executor"
)

// knownStatementTypes lists every top-level MDL statement type name.
// When a new statement type is added, add its TypeName() string here
// alongside the handler registration.
var knownStatementTypes = []string{
	"Connect",
	"Disconnect",
	"Status",
	"Help",
	"Exit",
	"Set",
	"Show",
	"ShowFeatures",
	"Select",
	"Describe",
	"DescribeCatalogTable",
	"Update",
	"Refresh",
	"RefreshCatalog",
	"Search",
	"ExecuteScript",

	"CreateModule",
	"DropModule",
	"DropFolder",
	"MoveFolder",

	"CreateEnumeration",
	"AlterEnumeration",
	"DropEnumeration",

	"CreateConstant",
	"DropConstant",

	"CreateEntity",
	"AlterEntity",
	"DropEntity",
	"CreateViewEntity",

	"CreateAssociation",
	"AlterAssociation",
	"DropAssociation",

	"CreateMicroflow",
	"DropMicroflow",

	"CreateNanoflow",
	"DropNanoflow",

	"CreatePageStmtV3",
	"DropPage",

	"CreateSnippetStmtV3",
	"DropSnippet",

	"CreateLayout",

	"CreateWorkflow",
	"DropWorkflow",
	"AlterWorkflow",

	"CreateDatabaseConnection",

	"CreateDataTransformer",
	"DropDataTransformer",

	"CreateImageCollection",
	"DropImageCollection",
	"AlterImageCollection",

	"CreateJsonStructure",
	"DropJsonStructure",

	"CreateImportMapping",
	"DropImportMapping",

	"CreateExportMapping",
	"DropExportMapping",

	"CreateODataService",
	"AlterODataService",
	"DropODataService",

	"CreateODataClient",
	"AlterODataClient",
	"DropODataClient",
	"CreateExternalEntity",
	"CreateExternalEntities",

	"CreatePublishedRestService",
	"DropPublishedRestService",
	"AlterPublishedRestService",

	"CreateRestClient",
	"DropRestClient",

	"CreateJavaAction",
	"DropJavaAction",

	"CreateJavaScriptAction",

	"CreateBusinessEventService",
	"DropBusinessEventService",

	"CreateModuleRole",
	"DropModuleRole",
	"CreateUserRole",
	"AlterUserRole",
	"DropUserRole",
	"GrantEntityAccess",
	"RevokeEntityAccess",
	"GrantMicroflowAccess",
	"RevokeMicroflowAccess",
	"GrantNanoflowAccess",
	"RevokeNanoflowAccess",
	"GrantPageAccess",
	"RevokePageAccess",
	"GrantWorkflowAccess",
	"RevokeWorkflowAccess",
	"GrantODataServiceAccess",
	"RevokeODataServiceAccess",
	"GrantPublishedRestServiceAccess",
	"RevokePublishedRestServiceAccess",
	"AlterProjectSecurity",
	"CreateDemoUser",
	"DropDemoUser",
	"UpdateSecurity",

	"AlterNavigation",

	"AlterSettings",
	"CreateConfiguration",
	"DropConfiguration",

	"AlterModuleJarDep",

	"DefineFragment",
	"DescribeFragmentFrom",

	"Lint",

	"SQLConnect",
	"SQLDisconnect",
	"SQLConnections",
	"SQLQuery",
	"SQLShowTables",
	"SQLDescribeTable",
	"SQLShowViews",
	"SQLShowFunctions",
	"SQLGenerateConnector",

	"Import",

	"Move",
	"Rename",

	"AlterPage",

	"ShowDesignProperties",
	"DescribeStyling",
	"AlterStyling",

	"ShowThemeVariables",

	"AlterLanguage",
	"Translate",
	"TranslateMicroflow",
	"DescribeTranslations",

	"ShowWidgets",
	"ShowInstalledWidgets",
	"UpdateWidgets",

	"AlterImageCollection",

	"DescribeContractFromOpenAPI",

	"CreateModel",
	"DropModel",
	"CreateAgent",
	"DropAgent",
	"CreateKnowledgeBase",
	"DropKnowledgeBase",
	"CreateConsumedMCPService",
	"DropConsumedMCPService",

	// CreateScheduledEvent — no handler yet
	// DropScheduledEvent — no handler yet

	// AlterDatabaseConnection — no handler yet
}

func TestRegistry_NoAllStatementTypesCovered(t *testing.T) {
	// NewRegistry returns an empty registry. All handlers are populated via
	// registerFutureOverlays when SetBackend is called on an Executor.
	r := executor.NewRegistry()
	registered := r.RegisteredTypes()
	if len(registered) != 0 {
		t.Errorf("expected empty registry, got %d types", len(registered))
	}
}

func TestRegistry_HandlerCountSnapshot(t *testing.T) {
	// NewRegistry creates an empty registry; handlers are populated by
	// registerFutureOverlays when an Executor's SetBackend is called.
	r := executor.NewRegistry()
	count := r.HandlerCount()
	if count != 0 {
		t.Errorf("expected empty registry, got %d handlers", count)
	}
}

func TestRegistry_RegisteredTypes_Deduped(t *testing.T) {
	r := executor.NewRegistry()
	types := r.RegisteredTypes()
	if len(types) != 0 {
		t.Errorf("expected empty registry, got %d types", len(types))
	}
}
