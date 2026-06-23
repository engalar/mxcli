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
			return describeEnumeration(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "enumeration",
	},
	ast.DescribeEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeEntityGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "entity",
	},
	ast.DescribeAssociation: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAssociation(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "association",
	},
	ast.DescribeMicroflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeMicroflowGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "microflow",
	},
	ast.DescribeNanoflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeNanoflowGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "nanoflow",
	},
	ast.DescribeModule: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeModule(phase3d2bNewExecContext(ctx, deps), s.Name.Module, s.WithAll)
		},
		label: "module",
	},
	ast.DescribePage: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describePage(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "page",
	},
	ast.DescribeSnippet: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeSnippet(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "snippet",
	},
	ast.DescribeLayout: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeLayout(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "layout",
	},
	ast.DescribeConstant: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeConstant(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "constant",
	},
	ast.DescribeJavaAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJavaActionGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "javaaction",
	},
	ast.DescribeJavaScriptAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJavaScriptActionGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "javascriptaction",
	},
	ast.DescribeModuleRole: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeModuleRoleGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "modulerole",
	},
	ast.DescribeUserRole: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeUserRoleGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "userrole",
	},
	ast.DescribeDemoUser: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDemoUserGen(phase3d2bNewExecContext(ctx, deps), s.Name.Name)
		},
		label: "demouser",
	},
	ast.DescribeODataClient: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeODataClient(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "odataclient",
	},
	ast.DescribeODataService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeODataService(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "odataservice",
	},
	ast.DescribeExternalEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeExternalEntity(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "externalentity",
	},
	ast.DescribeNavigation: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeNavigation(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "navigation",
	},
	ast.DescribeWorkflow: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeWorkflowGen(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "workflow",
	},
	ast.DescribeBusinessEventService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeBusinessEventService(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "businesseventservice",
	},
	ast.DescribeDatabaseConnection: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDatabaseConnection(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "databaseconnection",
	},
	ast.DescribeSettings: {
		handler: func(ctx context.Context, _ *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeSettings(phase3d2bNewExecContext(ctx, deps))
		},
		label: "settings",
	},
	ast.DescribeFragment: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeFragment(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "fragment",
	},
	ast.DescribeImageCollection: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeImageCollection(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "imagecollection",
	},
	ast.DescribeModel: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorModel(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "model",
	},
	ast.DescribeAgent: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorAgent(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "agent",
	},
	ast.DescribeKnowledgeBase: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorKnowledgeBase(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "knowledgebase",
	},
	ast.DescribeConsumedMCPService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeAgentEditorConsumedMCPService(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "consumedmcpservice",
	},
	ast.DescribeRestClient: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeRestClient(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "restclient",
	},
	ast.DescribePublishedRestService: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describePublishedRestService(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "publishedrestservice",
	},
	ast.DescribeDataTransformer: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeDataTransformer(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "datatransformer",
	},
	ast.DescribeContractEntity: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractEntity(phase3d2bNewExecContext(ctx, deps), s.Name, s.Format)
		},
		label: "contractentity",
	},
	ast.DescribeContractAction: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractAction(phase3d2bNewExecContext(ctx, deps), s.Name, s.Format)
		},
		label: "contractaction",
	},
	ast.DescribeContractMessage: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeContractMessage(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "contractmessage",
	},
	ast.DescribeJsonStructure: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeJsonStructure(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "jsonstructure",
	},
	ast.DescribeImportMapping: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeImportMapping(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "importmapping",
	},
	ast.DescribeExportMapping: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeExportMapping(phase3d2bNewExecContext(ctx, deps), s.Name)
		},
		label: "exportmapping",
	},
	ast.DescribeJarDependency: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return execDescribeJarDependency(phase3d2bNewExecContext(ctx, deps), s.Name.String(), s.Qualifier)
		},
		label: "jardependency",
	},
	ast.DescribeWidget: {
		handler: func(ctx context.Context, s *ast.DescribeStmt, deps *HandlerDeps) error {
			return describeWidget(phase3d2bNewExecContext(ctx, deps), s.Name)
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
