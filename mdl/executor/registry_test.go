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

func TestRegistry_AllStatementTypesCovered(t *testing.T) {
	r := executor.NewRegistry()
	registered := make(map[string]bool)
	for _, typ := range r.RegisteredTypes() {
		registered[typ] = true
	}
	var missing []string
	for _, typ := range knownStatementTypes {
		if !registered[typ] {
			missing = append(missing, typ)
		}
	}
	if len(missing) > 0 {
		t.Errorf("unregistered statement types (%d):", len(missing))
		for _, name := range missing {
			t.Logf("  %s", name)
		}
	}
}

func TestRegistry_HandlerCountSnapshot(t *testing.T) {
	r := executor.NewRegistry()
	count := r.HandlerCount()
	if count < 50 {
		t.Errorf("handler count seems too low: got %d, expected >= 50", count)
	}
	if count > 200 {
		t.Errorf("handler count seems too high: got %d, expected <= 200", count)
	}
}

func TestRegistry_RegisteredTypes_Deduped(t *testing.T) {
	r := executor.NewRegistry()
	types := r.RegisteredTypes()
	seen := make(map[string]bool, len(types))
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("duplicate registration for type %s", typ)
		}
		seen[typ] = true
	}
}
