// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// DescribeHandler is the function signature for DESCRIBE statement handlers.
type DescribeHandler func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error

type describeEntry struct {
	handler DescribeHandler
	label   string
}

// describeHandlers maps each DescribeObjectType to its handler and human label.
var describeHandlers = map[ast.DescribeObjectType]describeEntry{
	ast.DescribeEnumeration: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeEnumerationDeps(ctx, deps, s.Name)
		},
		label: "enumeration",
	},
	ast.DescribeEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeEntityGenDeps(ctx, deps, s.Name)
		},
		label: "entity",
	},
	ast.DescribeAssociation: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAssociationDeps(ctx, deps, s.Name)
		},
		label: "association",
	},
	ast.DescribeMicroflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeMicroflowGenDeps(ctx, deps, s.Name)
		},
		label: "microflow",
	},
	ast.DescribeNanoflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeNanoflowGenDeps(ctx, deps, s.Name)
		},
		label: "nanoflow",
	},
	ast.DescribeModule: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeModuleDeps(ctx, deps, s.Name.Module, s.WithAll)
		},
		label: "module",
	},
	ast.DescribePage: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describePageDeps(ctx, deps, s.Name)
		},
		label: "page",
	},
	ast.DescribeSnippet: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeSnippetDeps(ctx, deps, s.Name)
		},
		label: "snippet",
	},
	ast.DescribeLayout: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeLayoutDeps(ctx, deps, s.Name)
		},
		label: "layout",
	},
	ast.DescribeConstant: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeConstantDeps(ctx, deps, s.Name)
		},
		label: "constant",
	},
	ast.DescribeJavaAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJavaActionGenDeps(ctx, deps, s.Name)
		},
		label: "javaaction",
	},
	ast.DescribeJavaScriptAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJavaScriptActionGenDeps(ctx, deps, s.Name)
		},
		label: "javascriptaction",
	},
	ast.DescribeModuleRole: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeModuleRoleGenDeps(ctx, deps, s.Name)
		},
		label: "modulerole",
	},
	ast.DescribeUserRole: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeUserRoleGenDeps(ctx, deps, s.Name)
		},
		label: "userrole",
	},
	ast.DescribeDemoUser: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDemoUserGenDeps(ctx, deps, s.Name.Name)
		},
		label: "demouser",
	},
	ast.DescribeODataClient: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeODataClientDeps(ctx, deps, s.Name)
		},
		label: "odataclient",
	},
	ast.DescribeODataService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeODataServiceDeps(ctx, deps, s.Name)
		},
		label: "odataservice",
	},
	ast.DescribeExternalEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeExternalEntityDeps(ctx, deps, s.Name)
		},
		label: "externalentity",
	},
	ast.DescribeNavigation: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeNavigationDeps(ctx, deps, s.Name)
		},
		label: "navigation",
	},
	ast.DescribeWorkflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeWorkflowGenDeps(ctx, deps, s.Name)
		},
		label: "workflow",
	},
	ast.DescribeBusinessEventService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeBusinessEventServiceDeps(ctx, deps, s.Name)
		},
		label: "businesseventservice",
	},
	ast.DescribeDatabaseConnection: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDatabaseConnectionDeps(ctx, deps, s.Name)
		},
		label: "databaseconnection",
	},
	ast.DescribeSettings: {
		handler: func(ctx context.Context, _ *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeSettingsDeps(ctx, deps)
		},
		label: "settings",
	},
	ast.DescribeFragment: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeFragmentDeps(ctx, deps, s.Name)
		},
		label: "fragment",
	},
	ast.DescribeImageCollection: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeImageCollectionDeps(ctx, deps, s.Name)
		},
		label: "imagecollection",
	},
	ast.DescribeModel: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorModelDeps(ctx, deps, s.Name)
		},
		label: "model",
	},
	ast.DescribeAgent: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorAgentDeps(ctx, deps, s.Name)
		},
		label: "agent",
	},
	ast.DescribeKnowledgeBase: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorKnowledgeBaseDeps(ctx, deps, s.Name)
		},
		label: "knowledgebase",
	},
	ast.DescribeConsumedMCPService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorConsumedMCPServiceDeps(ctx, deps, s.Name)
		},
		label: "consumedmcpservice",
	},
	ast.DescribeRestClient: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeRestClientDeps(ctx, deps, s.Name)
		},
		label: "restclient",
	},
	ast.DescribePublishedRestService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describePublishedRestServiceDeps(ctx, deps, s.Name)
		},
		label: "publishedrestservice",
	},
	ast.DescribeDataTransformer: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDataTransformerDeps(ctx, deps, s.Name)
		},
		label: "datatransformer",
	},
	ast.DescribeContractEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractEntityDeps(ctx, deps, s.Name, s.Format)
		},
		label: "contractentity",
	},
	ast.DescribeContractAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractActionDeps(ctx, deps, s.Name, s.Format)
		},
		label: "contractaction",
	},
	ast.DescribeContractMessage: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractMessageDeps(ctx, deps, s.Name)
		},
		label: "contractmessage",
	},
	ast.DescribeJsonStructure: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJsonStructureDeps(ctx, deps, s.Name)
		},
		label: "jsonstructure",
	},
	ast.DescribeImportMapping: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeImportMappingDeps(ctx, deps, s.Name)
		},
		label: "importmapping",
	},
	ast.DescribeExportMapping: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeExportMappingDeps(ctx, deps, s.Name)
		},
		label: "exportmapping",
	},
	ast.DescribeJarDependency: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return execDescribeJarDependencyDeps(ctx, deps, s.Name.String(), s.Qualifier)
		},
		label: "jardependency",
	},
	ast.DescribeWidget: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeWidgetDeps(ctx, deps, s.Name)
		},
		label: "widget",
	},
}

// DescribeHandlers returns the describe handler map (exported for tests).
func DescribeHandlers() map[ast.DescribeObjectType]DescribeHandler {
	m := make(map[ast.DescribeObjectType]DescribeHandler, len(describeHandlers))
	for k, v := range describeHandlers {
		m[k] = v.handler
	}
	return m
}

// DescribeLabel returns the human-readable label for a describe object type.
func DescribeLabel(t ast.DescribeObjectType) string {
	if e, ok := describeHandlers[t]; ok {
		return e.label
	}
	return "unknown"
}
