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
			return describeEnumeration(NewExecContext(ctx, deps), s.Name)
		},
		label: "enumeration",
	},
	ast.DescribeEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeEntityGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "entity",
	},
	ast.DescribeAssociation: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAssociation(NewExecContext(ctx, deps), s.Name)
		},
		label: "association",
	},
	ast.DescribeMicroflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeMicroflowGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "microflow",
	},
	ast.DescribeNanoflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeNanoflowGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "nanoflow",
	},
	ast.DescribeModule: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeModule(NewExecContext(ctx, deps), s.Name.Module, s.WithAll)
		},
		label: "module",
	},
	ast.DescribePage: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describePage(NewExecContext(ctx, deps), s.Name)
		},
		label: "page",
	},
	ast.DescribeSnippet: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeSnippet(NewExecContext(ctx, deps), s.Name)
		},
		label: "snippet",
	},
	ast.DescribeLayout: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeLayout(NewExecContext(ctx, deps), s.Name)
		},
		label: "layout",
	},
	ast.DescribeConstant: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeConstant(NewExecContext(ctx, deps), s.Name)
		},
		label: "constant",
	},
	ast.DescribeJavaAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJavaActionGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "javaaction",
	},
	ast.DescribeJavaScriptAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJavaScriptActionGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "javascriptaction",
	},
	ast.DescribeModuleRole: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeModuleRoleGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "modulerole",
	},
	ast.DescribeUserRole: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeUserRoleGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "userrole",
	},
	ast.DescribeDemoUser: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDemoUserGen(NewExecContext(ctx, deps), s.Name.Name)
		},
		label: "demouser",
	},
	ast.DescribeODataClient: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeODataClient(NewExecContext(ctx, deps), s.Name)
		},
		label: "odataclient",
	},
	ast.DescribeODataService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeODataService(NewExecContext(ctx, deps), s.Name)
		},
		label: "odataservice",
	},
	ast.DescribeExternalEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeExternalEntity(NewExecContext(ctx, deps), s.Name)
		},
		label: "externalentity",
	},
	ast.DescribeNavigation: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeNavigation(NewExecContext(ctx, deps), s.Name)
		},
		label: "navigation",
	},
	ast.DescribeWorkflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeWorkflowGen(NewExecContext(ctx, deps), s.Name)
		},
		label: "workflow",
	},
	ast.DescribeBusinessEventService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeBusinessEventService(NewExecContext(ctx, deps), s.Name)
		},
		label: "businesseventservice",
	},
	ast.DescribeDatabaseConnection: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDatabaseConnection(NewExecContext(ctx, deps), s.Name)
		},
		label: "databaseconnection",
	},
	ast.DescribeSettings: {
		handler: func(ctx context.Context, _ *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeSettings(NewExecContext(ctx, deps))
		},
		label: "settings",
	},
	ast.DescribeFragment: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeFragment(NewExecContext(ctx, deps), s.Name)
		},
		label: "fragment",
	},
	ast.DescribeImageCollection: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeImageCollection(NewExecContext(ctx, deps), s.Name)
		},
		label: "imagecollection",
	},
	ast.DescribeModel: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorModel(NewExecContext(ctx, deps), s.Name)
		},
		label: "model",
	},
	ast.DescribeAgent: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorAgent(NewExecContext(ctx, deps), s.Name)
		},
		label: "agent",
	},
	ast.DescribeKnowledgeBase: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorKnowledgeBase(NewExecContext(ctx, deps), s.Name)
		},
		label: "knowledgebase",
	},
	ast.DescribeConsumedMCPService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorConsumedMCPService(NewExecContext(ctx, deps), s.Name)
		},
		label: "consumedmcpservice",
	},
	ast.DescribeRestClient: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeRestClient(NewExecContext(ctx, deps), s.Name)
		},
		label: "restclient",
	},
	ast.DescribePublishedRestService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describePublishedRestService(NewExecContext(ctx, deps), s.Name)
		},
		label: "publishedrestservice",
	},
	ast.DescribeDataTransformer: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDataTransformer(NewExecContext(ctx, deps), s.Name)
		},
		label: "datatransformer",
	},
	ast.DescribeContractEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractEntity(NewExecContext(ctx, deps), s.Name, s.Format)
		},
		label: "contractentity",
	},
	ast.DescribeContractAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractAction(NewExecContext(ctx, deps), s.Name, s.Format)
		},
		label: "contractaction",
	},
	ast.DescribeContractMessage: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractMessage(NewExecContext(ctx, deps), s.Name)
		},
		label: "contractmessage",
	},
	ast.DescribeJsonStructure: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJsonStructure(NewExecContext(ctx, deps), s.Name)
		},
		label: "jsonstructure",
	},
	ast.DescribeImportMapping: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeImportMapping(NewExecContext(ctx, deps), s.Name)
		},
		label: "importmapping",
	},
	ast.DescribeExportMapping: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeExportMapping(NewExecContext(ctx, deps), s.Name)
		},
		label: "exportmapping",
	},
	ast.DescribeJarDependency: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return execDescribeJarDependency(NewExecContext(ctx, deps), s.Name.String(), s.Qualifier)
		},
		label: "jardependency",
	},
	ast.DescribeWidget: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeWidget(NewExecContext(ctx, deps), s.Name)
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
