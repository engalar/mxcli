// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerBusinessEventHandlers(r *Registry) {
	r.Register(&ast.CreateBusinessEventServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createBusinessEventService(ctx, stmt.(*ast.CreateBusinessEventServiceStmt))
	})
	r.Register(&ast.DropBusinessEventServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return dropBusinessEventService(ctx, stmt.(*ast.DropBusinessEventServiceStmt))
	})
}

func registerODataHandlers(r *Registry) {
	r.Register(&ast.CreateODataClientStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createODataClient(ctx, stmt.(*ast.CreateODataClientStmt))
	})
	r.Register(&ast.AlterODataClientStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return alterODataClient(ctx, stmt.(*ast.AlterODataClientStmt))
	})
	r.Register(&ast.DropODataClientStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return dropODataClient(ctx, stmt.(*ast.DropODataClientStmt))
	})
	r.Register(&ast.CreateODataServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createODataService(ctx, stmt.(*ast.CreateODataServiceStmt))
	})
	r.Register(&ast.AlterODataServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return alterODataService(ctx, stmt.(*ast.AlterODataServiceStmt))
	})
	r.Register(&ast.DropODataServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return dropODataService(ctx, stmt.(*ast.DropODataServiceStmt))
	})
}

func registerJSONStructureHandlers(r *Registry) {
	r.Register(&ast.CreateJsonStructureStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateJsonStructure(ctx, stmt.(*ast.CreateJsonStructureStmt))
	})
	r.Register(&ast.DropJsonStructureStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropJsonStructure(ctx, stmt.(*ast.DropJsonStructureStmt))
	})
}

func registerMappingHandlers(r *Registry) {
	r.Register(&ast.CreateImportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateImportMapping(ctx, stmt.(*ast.CreateImportMappingStmt))
	})
	r.Register(&ast.DropImportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropImportMapping(ctx, stmt.(*ast.DropImportMappingStmt))
	})
	r.Register(&ast.CreateExportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateExportMapping(ctx, stmt.(*ast.CreateExportMappingStmt))
	})
	r.Register(&ast.DropExportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropExportMapping(ctx, stmt.(*ast.DropExportMappingStmt))
	})
}

func registerRESTHandlers(r *Registry) {
	r.Register(&ast.CreateRestClientStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createRestClient(ctx, stmt.(*ast.CreateRestClientStmt))
	})
	r.Register(&ast.DropRestClientStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return dropRestClient(ctx, stmt.(*ast.DropRestClientStmt))
	})
	r.Register(&ast.DescribeContractFromOpenAPIStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return describeContractFromOpenAPI(ctx, stmt.(*ast.DescribeContractFromOpenAPIStmt))
	})
	r.Register(&ast.CreatePublishedRestServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreatePublishedRestService(ctx, stmt.(*ast.CreatePublishedRestServiceStmt))
	})
	r.Register(&ast.DropPublishedRestServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropPublishedRestService(ctx, stmt.(*ast.DropPublishedRestServiceStmt))
	})
	r.Register(&ast.AlterPublishedRestServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterPublishedRestService(ctx, stmt.(*ast.AlterPublishedRestServiceStmt))
	})
	r.Register(&ast.CreateExternalEntityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateExternalEntity(ctx, stmt.(*ast.CreateExternalEntityStmt))
	})
	r.Register(&ast.CreateExternalEntitiesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createExternalEntities(ctx, stmt.(*ast.CreateExternalEntitiesStmt))
	})
	r.Register(&ast.GrantODataServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantODataServiceAccessGen(ctx, stmt.(*ast.GrantODataServiceAccessStmt))
	})
	r.Register(&ast.RevokeODataServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeODataServiceAccessGen(ctx, stmt.(*ast.RevokeODataServiceAccessStmt))
	})
	r.Register(&ast.GrantPublishedRestServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantPublishedRestServiceAccessGen(ctx, stmt.(*ast.GrantPublishedRestServiceAccessStmt))
	})
	r.Register(&ast.RevokePublishedRestServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokePublishedRestServiceAccessGen(ctx, stmt.(*ast.RevokePublishedRestServiceAccessStmt))
	})
}

func registerDataTransformerHandlers(r *Registry) {
	r.Register(&ast.CreateDataTransformerStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateDataTransformer(ctx, stmt.(*ast.CreateDataTransformerStmt))
	})
	r.Register(&ast.DropDataTransformerStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropDataTransformer(ctx, stmt.(*ast.DropDataTransformerStmt))
	})
}
