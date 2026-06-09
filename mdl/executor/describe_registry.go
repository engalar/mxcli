// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
)

// DescribeHandler is the function signature for DESCRIBE statement handlers.
// The handler is called inside writeDescribeJSON; it writes JSON fields via ctx.Output.
type DescribeHandler func(ctx *ExecContext, s *ast.DescribeStmt) error

type describeEntry struct {
	handler DescribeHandler
	label   string
}

// describeHandlers maps each DescribeObjectType to its handler and human label.
var describeHandlers = map[ast.DescribeObjectType]describeEntry{
	ast.DescribeEnumeration: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeEnumeration(ctx, s.Name) },
		label:   "enumeration",
	},
	ast.DescribeEntity: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeEntityGen(ctx, s.Name) },
		label:   "entity",
	},
	ast.DescribeAssociation: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeAssociation(ctx, s.Name) },
		label:   "association",
	},
	ast.DescribeMicroflow: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeMicroflowGen(ctx, s.Name) },
		label:   "microflow",
	},
	ast.DescribeNanoflow: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeNanoflowGen(ctx, s.Name) },
		label:   "nanoflow",
	},
	ast.DescribeModule: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeModule(ctx, s.Name.Module, s.WithAll)
		},
		label: "module",
	},
	ast.DescribePage: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describePage(ctx, s.Name) },
		label:   "page",
	},
	ast.DescribeSnippet: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeSnippet(ctx, s.Name) },
		label:   "snippet",
	},
	ast.DescribeLayout: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeLayout(ctx, s.Name) },
		label:   "layout",
	},
	ast.DescribeConstant: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeConstant(ctx, s.Name) },
		label:   "constant",
	},
	ast.DescribeJavaAction: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeJavaActionGen(ctx, s.Name) },
		label:   "javaaction",
	},
	ast.DescribeJavaScriptAction: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeJavaScriptActionGen(ctx, s.Name) },
		label:   "javascriptaction",
	},
	ast.DescribeModuleRole: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeModuleRoleGen(ctx, s.Name) },
		label:   "modulerole",
	},
	ast.DescribeUserRole: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeUserRoleGen(ctx, s.Name) },
		label:   "userrole",
	},
	ast.DescribeDemoUser: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeDemoUserGen(ctx, s.Name.Name) },
		label:   "demouser",
	},
	ast.DescribeODataClient: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeODataClient(ctx, s.Name) },
		label:   "odataclient",
	},
	ast.DescribeODataService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeODataService(ctx, s.Name) },
		label:   "odataservice",
	},
	ast.DescribeExternalEntity: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeExternalEntity(ctx, s.Name) },
		label:   "externalentity",
	},
	ast.DescribeNavigation: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeNavigation(ctx, s.Name) },
		label:   "navigation",
	},
	ast.DescribeWorkflow: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeWorkflowGen(ctx, s.Name) },
		label:   "workflow",
	},
	ast.DescribeBusinessEventService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeBusinessEventService(ctx, s.Name) },
		label:   "businesseventservice",
	},
	ast.DescribeDatabaseConnection: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeDatabaseConnection(ctx, s.Name) },
		label:   "databaseconnection",
	},
	ast.DescribeSettings: {
		handler: func(ctx *ExecContext, _ *ast.DescribeStmt) error { return describeSettings(ctx) },
		label:   "settings",
	},
	ast.DescribeFragment: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeFragment(ctx, s.Name) },
		label:   "fragment",
	},
	ast.DescribeImageCollection: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeImageCollection(ctx, s.Name) },
		label:   "imagecollection",
	},
	ast.DescribeModel: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeAgentEditorModel(ctx, s.Name) },
		label:   "model",
	},
	ast.DescribeAgent: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeAgentEditorAgent(ctx, s.Name) },
		label:   "agent",
	},
	ast.DescribeKnowledgeBase: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeAgentEditorKnowledgeBase(ctx, s.Name)
		},
		label: "knowledgebase",
	},
	ast.DescribeConsumedMCPService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeAgentEditorConsumedMCPService(ctx, s.Name)
		},
		label: "consumedmcpservice",
	},
	ast.DescribeRestClient: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeRestClient(ctx, s.Name) },
		label:   "restclient",
	},
	ast.DescribePublishedRestService: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describePublishedRestService(ctx, s.Name) },
		label:   "publishedrestservice",
	},
	ast.DescribeDataTransformer: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeDataTransformer(ctx, s.Name) },
		label:   "datatransformer",
	},
	ast.DescribeContractEntity: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeContractEntity(ctx, s.Name, s.Format)
		},
		label: "contractentity",
	},
	ast.DescribeContractAction: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return describeContractAction(ctx, s.Name, s.Format)
		},
		label: "contractaction",
	},
	ast.DescribeContractMessage: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeContractMessage(ctx, s.Name) },
		label:   "contractmessage",
	},
	ast.DescribeJsonStructure: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeJsonStructure(ctx, s.Name) },
		label:   "jsonstructure",
	},
	ast.DescribeImportMapping: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeImportMapping(ctx, s.Name) },
		label:   "importmapping",
	},
	ast.DescribeExportMapping: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error { return describeExportMapping(ctx, s.Name) },
		label:   "exportmapping",
	},
	ast.DescribeJarDependency: {
		handler: func(ctx *ExecContext, s *ast.DescribeStmt) error {
			return execDescribeJarDependency(ctx, s.Name.String(), s.Qualifier)
		},
		label: "jardependency",
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
