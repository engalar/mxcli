// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	bgCtx := context.Background()
	ectx := executor.NewMinimalExecCtx(bgCtx, deps)

	d := SecurityDeps{
		ConnectionManager:          deps.ConnectionManager,
		ModuleLister:               deps.ModuleLister,
		ModuleWriter:               deps.ModuleWriter,
		FolderManager:              deps.FolderManager,
		DomainModelReader:          deps.DomainModelReader,
		MetadataReader:             deps.MetadataReader,
		SecurityProjectManager:     deps.SecurityProjectManager,
		SecurityModuleManager:      deps.SecurityModuleManager,
		SecurityEntityAccessManager: deps.SecurityEntityAccessManager,
		NavigationWriter:           deps.NavigationWriter,
		NavigationReader:           deps.NavigationReader,
		ImageBackend:               deps.ImageBackend,
		SettingsReader:             deps.SettingsReader,
		SettingsWriter:             deps.SettingsWriter,
		ServiceLister:              deps.ServiceLister,
		RenameManager:              deps.RenameManager,

		Security:      deps.Security,
		PagesRepo:     deps.PageRepo,
		SnippetsRepo:  deps.SnippetRepo,
		LayoutsRepo:   deps.LayoutRepo,
		MicroflowsRepo: deps.MicroflowRepo,
		NanoflowsRepo:  deps.NanoflowRepo,

		Output: deps.Output,
		Quiet:  deps.Quiet,

		FindModule: func(name string) (*model.Module, error) {
			return executor.FindModuleWrap(ectx, name)
		},
		FindOrCreateModule: func(name string) (*model.Module, error) {
			return executor.FindOrCreateModuleWrap(ectx, name)
		},
		FindModuleName: func(containerID model.ID) string {
			h, err := executor.GetHierarchyForMining(ectx)
			if err != nil {
				return ""
			}
			return h.GetModuleName(containerID)
		},
		FindModuleID: func(containerID model.ID) model.ID {
			h, err := executor.GetHierarchyForMining(ectx)
			if err != nil {
				return ""
			}
			return h.FindModuleID(containerID)
		},

		ValidateModuleRole: func(role ast.QualifiedName) (bool, error) {
			return executor.ValidateModuleRoleWrap(ectx, role)
		},
		CascadeRemoveRoleFromMicroflows: func(moduleID model.ID, qualifiedRole string) error {
			return executor.CascadeRemoveRoleFromMicroflowsWrap(ectx, moduleID, qualifiedRole)
		},
		CascadeRemoveRoleFromNanoflows: func(moduleID model.ID, qualifiedRole string) error {
			return executor.CascadeRemoveRoleFromNanoflowsWrap(ectx, moduleID, qualifiedRole)
		},
		PruneInvalidUserRoles: func(exclude *model.ID) error {
			return executor.PruneInvalidUserRolesWrap(ectx, exclude)
		},

		LookupCreatedPageID: func(qualifiedName string) (model.ID, error) {
			return executor.LookupCreatedPageIDWrap(ectx, qualifiedName)
		},
		FilterAutoDocumentRoles: func(roles []string) []string {
			return executor.FilterAutoDocumentRolesWrap(ectx, roles)
		},
		MergeAllowedRoles: executor.MergeAllowedRolesStatic,
		FilterAllowedRoles: executor.FilterAllowedRolesStatic,

		CheckFeature: func(area, name, statement, hint string) error {
			return executor.CheckFeatureWrap(ectx, area, name, statement, hint)
		},

		GetProjectSecurityGen: func() (*genSec.ProjectSecurity, error) {
			return executor.GetProjectSecurityGenWrap(ectx)
		},
		InvalidateProjectSecurityCache: func() {
			executor.InvalidateProjectSecurityCacheWrap(ectx)
		},
		InvalidateModuleSecurityCache: func() {
			executor.InvalidateModuleSecurityCacheWrap(ectx)
		},
		GetDomainModelGenCached: func(moduleID model.ID) (*genDm.DomainModel, error) {
			return executor.GetDomainModelGenCachedWrap(ectx, moduleID)
		},
		InvalidateDomainModelGenForModule: func(moduleID model.ID) {
			executor.InvalidateDomainModelGenForModuleWrap(ectx, moduleID)
		},
		InvalidateDomainModelsCache: func() {
			executor.InvalidateDomainModelsCacheWrap(ectx)
		},
		GetModulesFromCache: func() ([]*model.Module, error) {
			return executor.GetModulesFromCacheWrap(ectx)
		},
		FindEntityGen: func(qn ast.QualifiedName) (*genDm.Entity, string, error) {
			return executor.FindEntityGenWrap(ectx, qn)
		},
		FormatAccessRuleResult: func(moduleName, entityName string, roleNames []string) string {
			return executor.FormatAccessRuleResultWrap(ectx, moduleName, entityName, roleNames)
		},
		DetectUserEntity: func() (string, error) {
			return executor.DetectUserEntityGenWrap(ectx)
		},
		CachedDomainModels: func() ([]*genDm.DomainModel, error) {
			return executor.CachedDomainModelsGenWrap(ectx)
		},
		EntityGeneralizationQN: func(entity *genDm.Entity) string {
			return executor.EntityGeneralizationQNWrap(entity)
		},
		TrackModifiedDomainModel: func(moduleID model.ID, moduleName string) {
			executor.TrackModifiedDomainModelWrap(ectx, moduleID, moduleName)
		},
	}

	r.RegisterFuture("CreateModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateModuleRoleFn(ctx, stmt.(*ast.CreateModuleRoleStmt), d)
	})
	r.RegisterFuture("DropModuleRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropModuleRoleFn(ctx, stmt.(*ast.DropModuleRoleStmt), d)
	})
	r.RegisterFuture("CreateUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateUserRoleFn(ctx, stmt.(*ast.CreateUserRoleStmt), d)
	})
	r.RegisterFuture("AlterUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterUserRoleFn(ctx, stmt.(*ast.AlterUserRoleStmt), d)
	})
	r.RegisterFuture("DropUserRole", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropUserRoleFn(ctx, stmt.(*ast.DropUserRoleStmt), d)
	})
	r.RegisterFuture("GrantEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantEntityAccessFn(ctx, stmt.(*ast.GrantEntityAccessStmt), d)
	})
	r.RegisterFuture("RevokeEntityAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeEntityAccessFn(ctx, stmt.(*ast.RevokeEntityAccessStmt), d)
	})
	r.RegisterFuture("GrantPageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantPageAccessFn(ctx, stmt.(*ast.GrantPageAccessStmt), d)
	})
	r.RegisterFuture("RevokePageAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokePageAccessFn(ctx, stmt.(*ast.RevokePageAccessStmt), d)
	})
	r.RegisterFuture("GrantMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantMicroflowAccessFn(ctx, stmt.(*ast.GrantMicroflowAccessStmt), d)
	})
	r.RegisterFuture("RevokeMicroflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeMicroflowAccessFn(ctx, stmt.(*ast.RevokeMicroflowAccessStmt), d)
	})
	r.RegisterFuture("GrantNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantNanoflowAccessFn(ctx, stmt.(*ast.GrantNanoflowAccessStmt), d)
	})
	r.RegisterFuture("RevokeNanoflowAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeNanoflowAccessFn(ctx, stmt.(*ast.RevokeNanoflowAccessStmt), d)
	})
	r.RegisterFuture("GrantODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantODataServiceAccessFn(ctx, stmt.(*ast.GrantODataServiceAccessStmt), d)
	})
	r.RegisterFuture("RevokeODataServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokeODataServiceAccessFn(ctx, stmt.(*ast.RevokeODataServiceAccessStmt), d)
	})
	r.RegisterFuture("GrantPublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecGrantPublishedRestServiceAccessFn(ctx, stmt.(*ast.GrantPublishedRestServiceAccessStmt), d)
	})
	r.RegisterFuture("RevokePublishedRestServiceAccess", func(ctx context.Context, stmt ast.Statement) error {
		return ExecRevokePublishedRestServiceAccessFn(ctx, stmt.(*ast.RevokePublishedRestServiceAccessStmt), d)
	})
	r.RegisterFuture("AlterProjectSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterProjectSecurityFn(ctx, stmt.(*ast.AlterProjectSecurityStmt), d)
	})
	r.RegisterFuture("UpdateSecurity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecUpdateSecurityFn(ctx, stmt.(*ast.UpdateSecurityStmt), d)
	})
	r.RegisterFuture("CreateDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateDemoUserFn(ctx, stmt.(*ast.CreateDemoUserStmt), d)
	})
	r.RegisterFuture("DropDemoUser", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropDemoUserFn(ctx, stmt.(*ast.DropDemoUserStmt), d)
	})
}
