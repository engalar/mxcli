// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func TestDescribeRegistryCoverage(t *testing.T) {
	allTypes := []ast.DescribeObjectType{
		ast.DescribeEnumeration, ast.DescribeEntity, ast.DescribeAssociation,
		ast.DescribeMicroflow, ast.DescribeNanoflow, ast.DescribeModule,
		ast.DescribePage, ast.DescribeSnippet, ast.DescribeLayout,
		ast.DescribeConstant, ast.DescribeJavaAction, ast.DescribeJavaScriptAction,
		ast.DescribeModuleRole, ast.DescribeUserRole, ast.DescribeDemoUser,
		ast.DescribeODataClient, ast.DescribeODataService, ast.DescribeExternalEntity,
		ast.DescribeNavigation, ast.DescribeWorkflow, ast.DescribeBusinessEventService,
		ast.DescribeDatabaseConnection, ast.DescribeSettings, ast.DescribeFragment,
		ast.DescribeImageCollection, ast.DescribeModel, ast.DescribeAgent,
		ast.DescribeKnowledgeBase, ast.DescribeConsumedMCPService,
		ast.DescribeRestClient, ast.DescribePublishedRestService,
		ast.DescribeDataTransformer, ast.DescribeContractEntity,
		ast.DescribeContractAction, ast.DescribeContractMessage,
		ast.DescribeJsonStructure, ast.DescribeImportMapping, ast.DescribeExportMapping,
		ast.DescribeJarDependency,
	}
	for _, ot := range allTypes {
		if _, ok := executor.DescribeHandlers()[ot]; !ok {
			t.Errorf("DescribeObjectType %v has no handler — add it to describe_registry.go", ot)
		}
		if executor.DescribeLabel(ot) == "unknown" {
			t.Errorf("DescribeObjectType %v has no label — add it to describe_registry.go", ot)
		}
	}
}
