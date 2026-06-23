package domainmodel

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func RegisterHandlers(r *executor.Registry, deps *executor.HandlerDeps) {
	// Single shared ExecContext for wiring DomainModelDeps callbacks.
	bgCtx := context.Background()
	ectx := executor.NewExecContext(bgCtx, deps)

	d := DomainModelDeps{
		ConnectionManager:          deps.ConnectionManager,
		ModuleLister:               deps.ModuleLister,
		ModuleWriter:               deps.ModuleWriter,
		DomainModelReader:          deps.DomainModelReader,
		DomainModelWriter:          deps.DomainModelWriter,
		EnumerationReader:          deps.EnumerationReader,
		EnumerationWriter:          deps.EnumerationWriter,
		ConstantReader:             deps.ConstantReader,
		ConstantWriter:             deps.ConstantWriter,
		ServiceLister:              deps.ServiceLister,
		ServiceWriter:              deps.ServiceWriter,
		FolderManager:              deps.FolderManager,
		MetadataReader:             deps.MetadataReader,
		SecurityEntityAccessManager: deps.SecurityEntityAccessManager,
		DomainModels:               deps.DomainModels,
		Output:                     deps.Output,

		FindModule: func(name string) (*model.Module, error) {
			return executor.FindModuleWrap(ectx, name)
		},
		FindOrCreateModule: func(name string) (*model.Module, error) {
			return executor.FindOrCreateModuleWrap(ectx, name)
		},

		GetDomainModelGenCached: func(moduleID model.ID) (*genDm.DomainModel, error) {
			if deps.Cache != nil && deps.Cache.DomainModelByModule() != nil {
				if dm, ok := deps.Cache.DomainModelByModule()[moduleID]; ok {
					return dm, nil
				}
			}
			dm, err := deps.DomainModelReader.GetDomainModelGen(moduleID)
			if err != nil {
				return nil, err
			}
			if deps.Cache != nil {
				m := deps.Cache.DomainModelByModule()
				if m == nil {
					m = make(map[model.ID]*genDm.DomainModel)
					deps.Cache.SetDomainModelByModule(m)
				}
				m[moduleID] = dm
			}
			return dm, nil
		},
		SetDomainModelGenCached: func(moduleID model.ID, dm *genDm.DomainModel) {
			if deps.Cache != nil {
				m := deps.Cache.DomainModelByModule()
				if m == nil {
					m = make(map[model.ID]*genDm.DomainModel)
					deps.Cache.SetDomainModelByModule(m)
				}
				m[moduleID] = dm
			}
		},
		InvalidateDomainModelGenCache: func(moduleID model.ID) {
			if deps.Cache != nil && deps.Cache.DomainModelByModule() != nil {
				delete(deps.Cache.DomainModelByModule(), moduleID)
			}
		},
		InvalidateDomainModelsCache: func() {
			if deps.Cache != nil {
				deps.Cache.SetDomainModels(nil)
				deps.Cache.SetDomainModelsGen(nil)
				deps.Cache.SetDomainModelsWithContainer(nil)
			}
		},
		InvalidateHierarchy: func() {
			executor.InvalidateHierarchyWrap(ectx)
		},

		FindEntityGen: func(qn ast.QualifiedName) (*genDm.Entity, string, error) {
			e := executor.NewExecContext(bgCtx, deps)
			return executor.FindEntityGenWrap(e, qn)
		},
		FindEnumeration: func(moduleName, enumName string) *model.Enumeration {
			e := executor.NewExecContext(bgCtx, deps)
			return executor.FindEnumerationWrap(e, moduleName, enumName)
		},

		EntityPersistable: func(entity *genDm.Entity) bool {
			if entity == nil {
				return false
			}
			if g, ok := entity.Generalization().(*genDm.NoGeneralization); ok {
				return g.Persistable()
			}
			return true
		},
		CheckFeature: func(area, name, statement, hint string) error {
			return executor.CheckFeatureWrap(ectx, area, name, statement, hint)
		},
		WarnEntityReferences: func(entityQN string) {
			executor.WarnEntityReferencesWrap(ectx, entityQN)
		},
		WarnMicroflowEntityParamRefs: func(entityQN string) {
			executor.WarnMicroflowEntityParamRefsWrap(ectx, entityQN)
		},
		TrackModifiedDomainModel: func(moduleID model.ID, moduleName string) {
			executor.TrackModifiedDomainModelWrap(ectx, moduleID, moduleName)
		},

		ValidateModuleRole: func(role ast.QualifiedName) (bool, error) {
			return false, fmt.Errorf("ValidateModuleRole not wired")
		},
		PruneInvalidUserRoles: func(exclude *model.ID) error {
			return nil
		},
		CascadeRemoveRoleFromMicroflows: func(moduleID model.ID, qualifiedRole string) error {
			return nil
		},
		CascadeRemoveRoleFromNanoflows: func(moduleID model.ID, qualifiedRole string) error {
			return nil
		},

		FindModuleName: func(containerID model.ID) string {
			h, err := executor.GetHierarchyForMining(ectx)
			if err != nil {
				return ""
			}
			return h.GetModuleName(h.FindModuleID(containerID))
		},
		FindModuleID: func(containerID model.ID) model.ID {
			h, err := executor.GetHierarchyForMining(ectx)
			if err != nil {
				return ""
			}
			return h.FindModuleID(containerID)
		},
	}

	r.RegisterFuture("CreateEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateEntityFn(ctx, stmt.(*ast.CreateEntityStmt), d)
	})
	r.RegisterFuture("AlterEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterEntityFn(ctx, stmt.(*ast.AlterEntityStmt), d)
	})
	r.RegisterFuture("DropEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropEntityFn(ctx, stmt.(*ast.DropEntityStmt), d)
	})
	r.RegisterFuture("CreateViewEntity", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateViewEntityFn(ctx, stmt.(*ast.CreateViewEntityStmt), d)
	})
	r.RegisterFuture("CreateAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecCreateAssociationFn(ctx, stmt.(*ast.CreateAssociationStmt), d)
	})
	r.RegisterFuture("AlterAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecAlterAssociationFn(ctx, stmt.(*ast.AlterAssociationStmt), d)
	})
	r.RegisterFuture("DropAssociation", func(ctx context.Context, stmt ast.Statement) error {
		return ExecDropAssociationFn(ctx, stmt.(*ast.DropAssociationStmt), d)
	})
	r.RegisterFuture("CreateEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateEnumeration(ectx, stmt.(*ast.CreateEnumerationStmt))
	})
	r.RegisterFuture("AlterEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return fmt.Errorf("alter enumeration not yet implemented")
	})
	r.RegisterFuture("DropEnumeration", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropEnumeration(ectx, stmt.(*ast.DropEnumerationStmt))
	})
	r.RegisterFuture("CreateConstant", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateConstant(ectx, stmt.(*ast.CreateConstantStmt))
	})
	r.RegisterFuture("DropConstant", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecDropConstant(ectx, stmt.(*ast.DropConstantStmt))
	})
	r.RegisterFuture("CreateDatabaseConnection", func(ctx context.Context, stmt ast.Statement) error {
		return executor.ExecCreateDatabaseConnection(ectx, stmt.(*ast.CreateDatabaseConnectionStmt))
	})
}
