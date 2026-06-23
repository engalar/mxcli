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
		return execCreateJsonStructureFn(ctx, stmt.(*ast.CreateJsonStructureStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropJsonStructureStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropJsonStructureFn(ctx, stmt.(*ast.DropJsonStructureStmt), execContextToDeps(ctx))
	})
}

func registerMappingHandlers(r *Registry) {
	r.Register(&ast.CreateImportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateImportMappingFn(ctx, stmt.(*ast.CreateImportMappingStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropImportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropImportMappingFn(ctx, stmt.(*ast.DropImportMappingStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateExportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateExportMappingFn(ctx, stmt.(*ast.CreateExportMappingStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropExportMappingStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropExportMappingFn(ctx, stmt.(*ast.DropExportMappingStmt), execContextToDeps(ctx))
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
		return execCreatePublishedRestServiceFn(ctx, stmt.(*ast.CreatePublishedRestServiceStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropPublishedRestServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropPublishedRestServiceFn(ctx, stmt.(*ast.DropPublishedRestServiceStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.AlterPublishedRestServiceStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execAlterPublishedRestServiceFn(ctx, stmt.(*ast.AlterPublishedRestServiceStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.CreateExternalEntityStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateExternalEntity(ctx, stmt.(*ast.CreateExternalEntityStmt))
	})
	r.Register(&ast.CreateExternalEntitiesStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createExternalEntities(ctx, stmt.(*ast.CreateExternalEntitiesStmt))
	})
	r.Register(&ast.GrantODataServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantODataServiceAccessGenFn(ctx, stmt.(*ast.GrantODataServiceAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.RevokeODataServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokeODataServiceAccessGenFn(ctx, stmt.(*ast.RevokeODataServiceAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.GrantPublishedRestServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execGrantPublishedRestServiceAccessGenFn(ctx, stmt.(*ast.GrantPublishedRestServiceAccessStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.RevokePublishedRestServiceAccessStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execRevokePublishedRestServiceAccessGenFn(ctx, stmt.(*ast.RevokePublishedRestServiceAccessStmt), execContextToDeps(ctx))
	})
}

func registerDataTransformerHandlers(r *Registry) {
	r.Register(&ast.CreateDataTransformerStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execCreateDataTransformerFn(ctx, stmt.(*ast.CreateDataTransformerStmt), execContextToDeps(ctx))
	})
	r.Register(&ast.DropDataTransformerStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return execDropDataTransformerFn(ctx, stmt.(*ast.DropDataTransformerStmt), execContextToDeps(ctx))
	})
}
