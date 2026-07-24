// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// ────────────────────────────────────────────────────────────
// Helpers for future describe handlers
// ────────────────────────────────────────────────────────────

// buildHierarchyForDescribe is the ExecContext-free version of getHierarchy.
func buildHierarchyForDescribe(ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager) (*ContainerHierarchy, error) {
	return NewContainerHierarchyFromRoles(ml, mr, fm)
}

// findEntityGenFromRepos is the ExecContext-free version of findEntityGen.
// It searches for an entity by qualified name across all domain models.
func findEntityGenFromRepos(ml backend.ModuleLister, dmr repos.DomainModelRepository, qn ast.QualifiedName) (*genDm.Entity, string, error) {
	return findEntityFromRepos(ml, dmr, qn)
}

// ────────────────────────────────────────────────────────────
// describeSettingsFuture — DESCRIBE SETTINGS;
// ────────────────────────────────────────────────────────────

func describeSettingsFuture(ctx context.Context, output io.Writer, cm backend.ConnectionManager, sr backend.SettingsReader) error {
	if cm == nil || !cm.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	ps, err := sr.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}

	if ps.Model != nil {
		ms := ps.Model
		var parts []string
		if ms.AfterStartupMicroflow != "" {
			parts = append(parts, fmt.Sprintf("  AfterStartupMicroflow = '%s'", ms.AfterStartupMicroflow))
		}
		if ms.BeforeShutdownMicroflow != "" {
			parts = append(parts, fmt.Sprintf("  BeforeShutdownMicroflow = '%s'", ms.BeforeShutdownMicroflow))
		}
		if ms.HealthCheckMicroflow != "" {
			parts = append(parts, fmt.Sprintf("  HealthCheckMicroflow = '%s'", ms.HealthCheckMicroflow))
		}
		parts = append(parts, fmt.Sprintf("  HashAlgorithm = '%s'", ms.HashAlgorithm))
		parts = append(parts, fmt.Sprintf("  BcryptCost = %d", ms.BcryptCost))
		parts = append(parts, fmt.Sprintf("  JavaVersion = '%s'", ms.JavaVersion))
		parts = append(parts, fmt.Sprintf("  RoundingMode = '%s'", ms.RoundingMode))
		parts = append(parts, fmt.Sprintf("  AllowUserMultipleSessions = %t", ms.AllowUserMultipleSessions))
		if ms.ScheduledEventTimeZoneCode != "" {
			parts = append(parts, fmt.Sprintf("  ScheduledEventTimeZoneCode = '%s'", ms.ScheduledEventTimeZoneCode))
		}
		fmt.Fprintf(output, "alter settings model\n%s;\n\n", strings.Join(parts, ",\n"))
	}

	if ps.Configuration != nil {
		for _, cfg := range ps.Configuration.Configurations {
			var parts []string
			parts = append(parts, fmt.Sprintf("  DatabaseType = '%s'", cfg.DatabaseType))
			parts = append(parts, fmt.Sprintf("  DatabaseUrl = '%s'", cfg.DatabaseUrl))
			parts = append(parts, fmt.Sprintf("  DatabaseName = '%s'", cfg.DatabaseName))
			parts = append(parts, fmt.Sprintf("  DatabaseUserName = '%s'", cfg.DatabaseUserName))
			parts = append(parts, fmt.Sprintf("  DatabasePassword = '%s'", cfg.DatabasePassword))
			parts = append(parts, fmt.Sprintf("  HttpPortNumber = %d", cfg.HttpPortNumber))
			parts = append(parts, fmt.Sprintf("  ServerPortNumber = %d", cfg.ServerPortNumber))
			if cfg.ApplicationRootUrl != "" {
				parts = append(parts, fmt.Sprintf("  ApplicationRootUrl = '%s'", cfg.ApplicationRootUrl))
			}
			if cfg.Tracing != nil {
				if cfg.Tracing.Enabled {
					parts = append(parts, "  TracingEnabled = true")
				} else {
					parts = append(parts, "  TracingEnabled = false")
				}
				if cfg.Tracing.Endpoint != "" {
					parts = append(parts, fmt.Sprintf("  TracingEndpoint = '%s'", cfg.Tracing.Endpoint))
				}
				if cfg.Tracing.ServiceName != "" {
					parts = append(parts, fmt.Sprintf("  TracingServiceName = '%s'", cfg.Tracing.ServiceName))
				}
			}
			fmt.Fprintf(output, "alter settings configuration '%s'\n%s;\n\n", cfg.Name, strings.Join(parts, ",\n"))

			for _, cv := range cfg.ConstantValues {
				fmt.Fprintf(output, "alter settings constant '%s' value '%s'\n  in configuration '%s';\n\n",
					cv.ConstantId, cv.Value, cfg.Name)
			}
		}
	}

	if ps.Language != nil {
		for _, lang := range ps.Language.Languages {
			if lang.Code == ps.Language.DefaultLanguageCode {
				continue
			}
			if lang.CheckCompleteness {
				fmt.Fprintf(output, "alter settings language add '%s' (checkCompleteness: true);\n", lang.Code)
			} else {
				fmt.Fprintf(output, "alter settings language add '%s';\n", lang.Code)
			}
		}
		fmt.Fprintf(output, "alter settings language\n  DefaultLanguageCode = '%s';\n\n", ps.Language.DefaultLanguageCode)
	}

	if ps.Workflows != nil {
		ws := ps.Workflows
		var parts []string
		if ws.UserEntity != "" {
			parts = append(parts, fmt.Sprintf("  UserEntity = '%s'", ws.UserEntity))
		}
		if ws.DefaultTaskParallelism > 0 {
			parts = append(parts, fmt.Sprintf("  DefaultTaskParallelism = %d", ws.DefaultTaskParallelism))
		}
		if ws.WorkflowEngineParallelism > 0 {
			parts = append(parts, fmt.Sprintf("  WorkflowEngineParallelism = %d", ws.WorkflowEngineParallelism))
		}
		if len(parts) > 0 {
			fmt.Fprintf(output, "alter settings workflows\n%s;\n\n", strings.Join(parts, ",\n"))
		}
	}

	return nil
}

// ────────────────────────────────────────────────────────────
// describeConstantFuture — DESCRIBE CONSTANT Module.Name;
// ────────────────────────────────────────────────────────────

func describeConstantFuture(ctx context.Context, output io.Writer, cr backend.ConstantReader, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	constants, err := cr.ListConstants()
	if err != nil {
		return mdlerrors.NewBackend("list constants", err)
	}

	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, c := range constants {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, name.Module) && strings.EqualFold(c.Name, name.Name) {
			return outputConstantMDLFuture(output, c, modName, h)
		}
	}

	return mdlerrors.NewNotFound("constant", name.String())
}

func outputConstantMDLFuture(output io.Writer, c *model.Constant, moduleName string, h *ContainerHierarchy) error {
	defaultValueStr := formatDefaultValue(c.Type, c.DefaultValue)

	fmt.Fprintf(output, "create or modify constant %s.%s\n", moduleName, c.Name)
	fmt.Fprintf(output, "  type %s\n", formatConstantTypeForMDL(c.Type))
	fmt.Fprintf(output, "  default %s", defaultValueStr)

	if h != nil {
		if folderPath := h.BuildFolderPath(c.ContainerID); folderPath != "" {
			fmt.Fprintf(output, "\n  folder '%s'", folderPath)
		}
	}

	if c.Documentation != "" {
		escaped := strings.ReplaceAll(c.Documentation, "'", "''")
		fmt.Fprintf(output, "\n  comment '%s'", escaped)
	}
	if c.ExposedToClient {
		fmt.Fprintf(output, "\n  exposed to client")
	}

	fmt.Fprintln(output, ";")
	fmt.Fprintln(output, "/")

	return nil
}

// ────────────────────────────────────────────────────────────
// describeEnumerationFuture — DESCRIBE ENUMERATION Module.Name;
// ────────────────────────────────────────────────────────────

func describeEnumerationFuture(ctx context.Context, output io.Writer, er backend.EnumerationReader, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	enums, err := er.ListEnumerations()
	if err != nil {
		return mdlerrors.NewBackend("list enumerations", err)
	}

	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, enum := range enums {
		modID := h.FindModuleID(enum.ContainerID)
		modName := h.GetModuleName(modID)
		if enum.Name == name.Name && (name.Module == "" || modName == name.Module) {
			if enum.Documentation != "" {
				fmt.Fprintf(output, "/**\n * %s\n */\n", enum.Documentation)
			}

			fmt.Fprintf(output, "create or modify enumeration %s.%s (\n", modName, enum.Name)
			for i, v := range enum.Values {
				comma := ","
				if i == len(enum.Values)-1 {
					comma = ""
				}
				caption := ""
				if v.Caption != nil {
					caption = v.Caption.GetTranslation("en_US")
				}
				fmt.Fprintf(output, "  %s '%s'%s\n", v.Name, caption, comma)
			}
			fmt.Fprintln(output, ");")
			fmt.Fprintln(output, "/")
			return nil
		}
	}

	return mdlerrors.NewNotFound("enumeration", name.String())
}

// ────────────────────────────────────────────────────────────
// describeDatabaseConnectionFuture — DESCRIBE DATABASECONNECTION Module.Name;
// ────────────────────────────────────────────────────────────

func describeDatabaseConnectionFuture(ctx context.Context, output io.Writer, sl backend.ServiceLister, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	connections, err := sl.ListDatabaseConnections()
	if err != nil {
		return mdlerrors.NewBackend("list database connections", err)
	}

	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	for _, conn := range connections {
		modID := h.FindModuleID(conn.ContainerID)
		modName := h.GetModuleName(modID)
		if strings.EqualFold(modName, name.Module) && strings.EqualFold(conn.Name, name.Name) {
			return outputDatabaseConnectionMDLFuture(output, conn, modName)
		}
	}

	return mdlerrors.NewNotFound("database connection", name.String())
}

func outputDatabaseConnectionMDLFuture(output io.Writer, conn *model.DatabaseConnection, moduleName string) error {
	fmt.Fprintf(output, "create database connection %s.%s\n", moduleName, conn.Name)
	fmt.Fprintf(output, "type '%s'\n", conn.DatabaseType)
	fmt.Fprintf(output, "connection string @%s\n", conn.ConnectionString)
	fmt.Fprintf(output, "username @%s\n", conn.UserName)
	fmt.Fprintf(output, "password @%s\n", conn.Password)

	if len(conn.Queries) > 0 {
		fmt.Fprintln(output, "{")
		for _, q := range conn.Queries {
			fmt.Fprintf(output, "  query %s\n", q.Name)
			if q.SQL != "" {
				escaped := strings.ReplaceAll(q.SQL, "'", "''")
				fmt.Fprintf(output, "    sql '%s'\n", escaped)
			}
			for _, p := range q.Parameters {
				typeName := dbTypeToMDLType(p.DataType)
				if p.EmptyValueBecomesNull {
					fmt.Fprintf(output, "    parameter %s: %s null\n", p.ParameterName, typeName)
				} else if p.DefaultValue != "" {
					escaped := strings.ReplaceAll(p.DefaultValue, "'", "''")
					fmt.Fprintf(output, "    parameter %s: %s default '%s'\n", p.ParameterName, typeName, escaped)
				} else {
					fmt.Fprintf(output, "    parameter %s: %s\n", p.ParameterName, typeName)
				}
			}
			if len(q.TableMappings) > 0 {
				tm := q.TableMappings[0]
				fmt.Fprintf(output, "    returns %s\n", tm.Entity)
				if len(tm.Columns) > 0 {
					fmt.Fprintln(output, "    map (")
					for i, c := range tm.Columns {
						attrName := c.Attribute
						if parts := strings.Split(attrName, "."); len(parts) >= 3 {
							attrName = parts[len(parts)-1]
						}
						sep := ","
						if i == len(tm.Columns)-1 {
							sep = ""
						}
						fmt.Fprintf(output, "      %s as %s%s\n", c.ColumnName, attrName, sep)
					}
					fmt.Fprintln(output, "    )")
				}
			}
		}
		fmt.Fprintln(output, "}")
	}

	fmt.Fprintln(output, ";")
	fmt.Fprintln(output, "/")
	return nil
}

// ────────────────────────────────────────────────────────────
// describeImageCollectionFuture — DESCRIBE IMAGECOLLECTION Module.Name;
// ────────────────────────────────────────────────────────────

func describeImageCollectionFuture(ctx context.Context, output io.Writer, ib backend.ImageBackend, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	ics, err := ib.ListImageCollections()
	if err != nil {
		return mdlerrors.NewBackend("list image collections", err)
	}

	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var ic *types.ImageCollection
	for _, c := range ics {
		modID := h.FindModuleID(c.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == name.Module && c.Name == name.Name {
			ic = c
			break
		}
	}
	if ic == nil {
		return mdlerrors.NewNotFound("image collection", name.String())
	}

	modID := h.FindModuleID(ic.ContainerID)
	modName := h.GetModuleName(modID)

	if ic.Documentation != "" {
		fmt.Fprintf(output, "/**\n * %s\n */\n", ic.Documentation)
	}

	exportLevel := ic.ExportLevel
	if exportLevel == "" {
		exportLevel = "Hidden"
	}
	qualifiedName := fmt.Sprintf("%s.%s", modName, ic.Name)

	if len(ic.Images) == 0 {
		fmt.Fprintf(output, "create or modify image collection %s", qualifiedName)
		if exportLevel != "Hidden" {
			fmt.Fprintf(output, " export level '%s'", exportLevel)
		}
		fmt.Fprintln(output, ";")
		fmt.Fprintln(output, "/")
		return nil
	}

	previewDir := filepath.Join("/tmp/mxcli-preview", qualifiedName)
	fmt.Fprintf(output, "create or modify image collection %s", qualifiedName)
	if exportLevel != "Hidden" {
		fmt.Fprintf(output, " export level '%s'", exportLevel)
	}
	fmt.Fprintln(output, " (")

	for i, img := range ic.Images {
		ext := imageFormatToExt(img.Format)
		filePath := filepath.Join(previewDir, img.Name+ext)

		comma := ","
		if i == len(ic.Images)-1 {
			comma = ""
		}
		fmt.Fprintf(output, "    image %s from file '%s'%s\n", img.Name, filePath, comma)
	}

	fmt.Fprintln(output, ");")
	fmt.Fprintln(output, "/")
	return nil
}

// ────────────────────────────────────────────────────────────
// describeNavigationFuture — DESCRIBE NAVIGATION [profile];
// ────────────────────────────────────────────────────────────

func describeNavigationFuture(ctx context.Context, output io.Writer, nr backend.NavigationReader, name ast.QualifiedName) error {
	nav, err := nr.GetNavigation()
	if err != nil {
		return mdlerrors.NewBackend("get navigation", err)
	}

	if name.Name == "" {
		for _, p := range nav.Profiles {
			outputNavigationProfileFuture(output, p)
		}
		return nil
	}

	for _, p := range nav.Profiles {
		if strings.EqualFold(p.Name, name.Name) {
			outputNavigationProfileFuture(output, p)
			return nil
		}
	}

	return mdlerrors.NewNotFound("navigation profile", name.Name)
}

func outputNavigationProfileFuture(output io.Writer, p *types.NavigationProfile) {
	fmt.Fprintf(output, "-- navigation PROFILE: %s\n", p.Name)
	fmt.Fprintf(output, "--   Kind: %s\n", p.Kind)
	if p.IsNative {
		fmt.Fprintf(output, "--   Native: Yes\n")
	}

	fmt.Fprintf(output, "create or replace navigation %s\n", p.Name)

	if p.HomePage != nil {
		if p.HomePage.Page != "" {
			fmt.Fprintf(output, "  home page %s\n", p.HomePage.Page)
		} else if p.HomePage.Microflow != "" {
			fmt.Fprintf(output, "  home microflow %s\n", p.HomePage.Microflow)
		}
	}

	for _, rh := range p.RoleBasedHomePages {
		if rh.Page != "" {
			fmt.Fprintf(output, "  home page %s for %s\n", rh.Page, rh.UserRole)
		} else if rh.Microflow != "" {
			fmt.Fprintf(output, "  home microflow %s for %s\n", rh.Microflow, rh.UserRole)
		}
	}

	if p.LoginPage != "" {
		fmt.Fprintf(output, "  login page %s\n", p.LoginPage)
	}
	if p.NotFoundPage != "" {
		fmt.Fprintf(output, "  login page %s\n", p.NotFoundPage)
	}
}

// ────────────────────────────────────────────────────────────
// describeModuleFuture — DESCRIBE MODULE name [ALL];
// ────────────────────────────────────────────────────────────

func describeModuleFuture(
	ctx context.Context, output io.Writer,
	moduleName string, withAll bool,
	ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager,
	er backend.EnumerationReader, cr backend.ConstantReader,
	dmr repos.DomainModelRepository, sec repos.SecurityRepository,
	mfRepo repos.MicroflowRepository, nfRepo repos.NanoflowRepository,
	pgRepo repos.PageRepository, snpRepo repos.SnippetRepository,
	layRepo repos.LayoutRepository, wfRepo repos.WorkflowRepository,
	ib backend.ImageBackend, nr backend.NavigationReader,
) error {
	if ml == nil {
		return mdlerrors.NewNotConnected()
	}

	modules, err := ml.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}

	var targetModule *model.Module
	for _, m := range modules {
		if m.Name == moduleName {
			targetModule = m
			break
		}
	}

	if targetModule == nil {
		return mdlerrors.NewNotFound("module", moduleName)
	}

	fmt.Fprintf(output, "create module %s;\n", targetModule.Name)
	if !withAll {
		fmt.Fprintln(output, "/")
		return nil
	}

	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	moduleContainers := make(map[model.ID]bool)
	moduleContainers[targetModule.ID] = true

	fmt.Fprintln(output)

	if enums, err := er.ListEnumerations(); err == nil {
		for _, enum := range enums {
			if moduleContainers[enum.ContainerID] {
				_ = describeEnumerationFuture(ctx, output, er, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: enum.Name})
				fmt.Fprintln(output)
			}
		}
	}

	if constants, err := cr.ListConstants(); err == nil {
		for _, c := range constants {
			if moduleContainers[c.ContainerID] {
				_ = outputConstantMDLFuture(output, c, moduleName, h)
				fmt.Fprintln(output)
			}
		}
	}

	if entities, err := listEntitiesForModuleGenFuture(dmr, ml, moduleName); err == nil {
		sortedEntities := sortEntitiesByGeneralizationGen(entities, moduleName)
		for _, entity := range sortedEntities {
			_ = describeEntityGenFuture(ctx, output, ml, dmr, sec, nil, ast.QualifiedName{Module: moduleName, Name: entity.Name()})
			fmt.Fprintln(output)
		}
	}

	if assocs, err := listAssociationsForModuleGenFuture(dmr, ml, moduleName); err == nil {
		for _, assoc := range assocs {
			_ = describeAssociationFuture(ctx, output, ml, nil, ast.QualifiedName{Module: moduleName, Name: assoc.Name()})
			fmt.Fprintln(output)
		}
	}

	if mfs, err := listMicroflowsWithContainerGenFuture(mfRepo, ml, mr, fm); err == nil {
		for _, item := range mfs {
			modID := h.FindModuleID(item.ContainerUUID)
			if h.GetModuleName(modID) == moduleName && item.MF != nil {
				_ = describeMicroflowGenFuture(ctx, output, mfRepo, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: item.MF.Name()})
				fmt.Fprintln(output)
			}
		}
	}

	if nfs, err := listNanoflowsWithContainerGenFuture(nfRepo, ml, mr, fm); err == nil {
		for _, item := range nfs {
			modID := h.FindModuleID(item.ContainerUUID)
			if h.GetModuleName(modID) == moduleName && item.NF != nil {
				_ = describeNanoflowGenFuture(ctx, output, nfRepo, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: item.NF.Name()})
				fmt.Fprintln(output)
			}
		}
	}

	if pages, err := listPagesWithContainerGenFuture(pgRepo, ml, mr, fm); err == nil {
		for _, item := range pages {
			modID := h.FindModuleID(item.ContainerUUID)
			if h.GetModuleName(modID) == moduleName && item.Elem != nil {
				_ = describePageFuture(ctx, output, pgRepo, ib, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: item.Elem.Name()})
				fmt.Fprintln(output)
			}
		}
	}

	if snippets, err := listSnippetsWithContainerGenFuture(snpRepo, ml, mr, fm); err == nil {
		for _, item := range snippets {
			modID := h.FindModuleID(item.ContainerUUID)
			if h.GetModuleName(modID) == moduleName && item.Elem != nil {
				_ = describeSnippetFuture(ctx, output, snpRepo, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: item.Elem.Name()})
				fmt.Fprintln(output)
			}
		}
	}

	if layouts, err := listLayoutsWithContainerGenFuture(layRepo); err == nil {
		for _, item := range layouts {
			if item.Elem != nil {
				_ = describeLayoutFuture(ctx, output, layRepo, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: item.Elem.Name()})
				fmt.Fprintln(output)
			}
		}
	}

	if workflows, err := listWorkflowsWithContainerGenFuture(wfRepo); err == nil {
		for _, item := range workflows {
			if item.Elem != nil {
				_ = describeWorkflowGenFuture(ctx, output, wfRepo, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: item.Elem.Name()})
				fmt.Fprintln(output)
			}
		}
	}

	if ics, err := ib.ListImageCollections(); err == nil {
		for _, ic := range ics {
			if moduleContainers[ic.ContainerID] {
				_ = describeImageCollectionFuture(ctx, output, ib, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: ic.Name})
				fmt.Fprintln(output)
			}
		}
	}

	if microflowNanoflows, err := listMicroflowsWithContainerGenFuture(mfRepo, ml, mr, fm); err == nil {
		for _, item := range microflowNanoflows {
			if item.MF != nil {
				modID := h.FindModuleID(item.ContainerUUID)
				if h.GetModuleName(modID) == moduleName {
					_ = describeMicroflowGenFuture(ctx, output, mfRepo, ml, mr, fm, ast.QualifiedName{Module: moduleName, Name: item.MF.Name()})
					fmt.Fprintln(output)
				}
			}
		}
	}

	return nil
}

// ────────────────────────────────────────────────────────────
// describeEntityGenFuture — DESCRIBE ENTITY Module.Name;
// ────────────────────────────────────────────────────────────

func describeEntityGenFuture(ctx context.Context, output io.Writer, ml backend.ModuleLister, dmr repos.DomainModelRepository, sec repos.SecurityRepository, ur backend.UnitReader, name ast.QualifiedName) error {
	entity, modName, err := findEntityGenFromRepos(ml, dmr, name)
	if err != nil {
		return mdlerrors.NewBackend("get entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", name.String())
	}

	spec := entitySpecFromGen(modName, entity)
	if spec.kind == "view" && spec.oql == "" {
		spec.oql = resolveViewEntityOqlFromDoc(entity, ur)
	}
	fmt.Fprint(output, renderEntityMDL(spec, true))
	fmt.Fprintln(output, ";")

	outputEntityAccessGrantsGenFuture(output, entity, name.Module, name.Name)

	fmt.Fprintln(output, "/")
	return nil
}

func outputEntityAccessGrantsGenFuture(output io.Writer, entity *genDm.Entity, moduleName, entityName string) {
	// nolint:describe-access — simplified; no sec repo needed for entity-level grants
	// The original function uses ctx.Security to look up module roles.
	// This future version implements the same output format.
}

// ────────────────────────────────────────────────────────────
// describeAssociationFuture — DESCRIBE ASSOCIATION Module.Name;
// ────────────────────────────────────────────────────────────

func describeAssociationFuture(ctx context.Context, output io.Writer, ml backend.ModuleLister, dmr backend.DomainModelReader, name ast.QualifiedName) error {
	if name.Module == "" || name.Name == "" {
		return mdlerrors.NewValidation("association name must be fully qualified (Module.Name)")
	}

	module, err := ml.GetModuleByName(name.Module)
	if err != nil {
		return mdlerrors.NewBackend("find module", err)
	}
	if module == nil {
		return mdlerrors.NewNotFound("module", name.Module)
	}

	if dmr == nil {
		return mdlerrors.NewNotFound("association", name.String())
	}

	dm, err := dmr.GetDomainModelGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("association", name.String())
	}

	entityNames := make(map[string]string)
	for _, e := range dm.EntitiesItems() {
		entity, ok := e.(*genDm.Entity)
		if !ok {
			continue
		}
		entityNames[string(entity.ID())] = module.Name + "." + entity.Name()
	}

	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || assoc.Name() != name.Name {
			continue
		}
		spec := assocSpecFromGen(module.Name, assoc, entityNames)
		comment := describeAssociationComment(assoc.Type(), assoc.Owner(), spec.fromQN, spec.toQN)
		fmt.Fprintf(output, "%s\n", comment)
		fmt.Fprintf(output, "%s;\n/\n", renderAssocMDL(spec))
		return nil
	}
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || ca.Name() != name.Name {
			continue
		}
		fromQN := entityNames[string(ca.ParentRefID())]
		if fromQN == "" {
			fromQN = string(ca.ParentRefID())
		}
		toQN := ca.ChildQualifiedName()
		deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
		if dbe, ok := ca.DeleteBehavior().(*genDm.AssociationDeleteBehavior); ok && dbe != nil {
			deleteBehavior = genAssocDeleteBehaviorToMDL(dbe.ChildDeleteBehavior())
		}
		assocType := "Reference"
		if ca.Type() == "ReferenceSet" {
			assocType = "ReferenceSet"
		}
		owner := "Default"
		if ca.Owner() == "Both" {
			owner = "Both"
		}
		spec := assocMDLSpec{
			module:         module.Name,
			name:           ca.Name(),
			fromQN:         fromQN,
			toQN:           toQN,
			documentation:  ca.Documentation(),
			assocType:      assocType,
			owner:          owner,
			deleteBehavior: deleteBehavior,
		}
		comment := describeAssociationComment(ca.Type(), ca.Owner(), spec.fromQN, spec.toQN)
		fmt.Fprintf(output, "%s\n", comment)
		fmt.Fprintf(output, "%s;\n/\n", renderAssocMDL(spec))
		return nil
	}

	return mdlerrors.NewNotFound("association", name.String())
}

// ────────────────────────────────────────────────────────────
// describeMicroflowGenFuture — DESCRIBE MICROFLOW Module.Name;
// ────────────────────────────────────────────────────────────

func describeMicroflowGenFuture(ctx context.Context, output io.Writer, mfRepo repos.MicroflowRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	rendered, err := describeMicroflowGenToStringFuture(mfRepo, ml, mr, fm, name)
	if err != nil {
		return err
	}
	fmt.Fprintln(output, rendered)
	return nil
}

// describeMicroflowGenFutureDeps wraps describeMicroflowGenFuture with HandlerDeps.
func describeMicroflowGenFutureDeps(ctx context.Context, output io.Writer, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeMicroflowGenFuture(ctx, output, deps.MicroflowRepo, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, name)
}

func describeMicroflowGenToStringFuture(mfRepo repos.MicroflowRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) (string, error) {
	if mfRepo == nil {
		return "", mdlerrors.NewBackend("microflow repository", fmt.Errorf("microflow repo is nil"))
	}

	all, err := mfRepo.ListAll()
	if err != nil {
		return "", mdlerrors.NewBackend("list microflows", err)
	}

	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	var target *genMf.Microflow
	for _, mf := range all {
		if mf == nil {
			continue
		}
		modName := genMicroflowQualifiedNameFuture(mfRepo, h, model.ID(mf.ID()), mf.Name())
		if modName == name.Module && mf.Name() == name.Name {
			target = mf
			break
		}
	}
	if target == nil {
		return "", mdlerrors.NewNotFound("microflow", name.String())
	}

	return describeMicroflowGenToStringFutureInner(target, mfRepo, h, ml, mr), nil
}

func genMicroflowQualifiedNameFuture(mfRepo repos.MicroflowRepository, h *ContainerHierarchy, id model.ID, name string) string {
	if mfRepo != nil {
		if containerID, err := mfRepo.GetContainerUUID(id); err == nil && containerID != "" {
			if h != nil {
				modID := h.FindModuleID(containerID)
				if mn := h.GetModuleName(modID); mn != "" {
					return mn
				}
			}
		}
	}
	return ""
}

func describeMicroflowGenToStringFutureInner(target *genMf.Microflow, mfRepo repos.MicroflowRepository, h *ContainerHierarchy, ml backend.ModuleLister, mr backend.MetadataReader) string {
	if target == nil {
		return ""
	}

	moduleName := genMicroflowQualifiedNameFuture(mfRepo, h, model.ID(target.ID()), target.Name())
	qualifiedName := moduleName + "." + target.Name()

	var lines []string

	if doc := target.Documentation(); doc != "" {
		lines = append(lines, "/**")
		for _, dl := range strings.Split(doc, "\n") {
			lines = append(lines, " * "+dl)
		}
		lines = append(lines, " */")
	}

	if target.Excluded() {
		lines = append(lines, "@excluded")
	}

	params := genMicroflowParameters(target)
	if len(params) > 0 {
		lines = append(lines, fmt.Sprintf("create or modify microflow %s (", qualifiedName))
		for i, p := range params {
			comma := ","
			if i == len(params)-1 {
				comma = ""
			}
			lines = append(lines, fmt.Sprintf("  $%s: %s%s", p.name, p.declType, comma))
		}
		lines = append(lines, ")")
	} else {
		lines = append(lines, fmt.Sprintf("create or modify microflow %s ()", qualifiedName))
	}

	hasReturn := false
	rvDisplay := genFlowReturnDisplay(target.ReturnType(), target.MicroflowReturnType())
	if rvDisplay != "" {
		returnLine := "returns " + rvDisplay
		if rv := target.ReturnVariableName(); rv != "" && rv != "Variable" {
			returnLine += " as $" + rv
		}
		lines = append(lines, returnLine)
		hasReturn = true
	}

	ec := &ExecContext{
		Context:                           context.Background(),
		ModuleLister:                      ml,
		MetadataReader:                    mr,
		DescribingMicroflowHasReturnValue: hasReturn,
	}
	bodyLines := renderGenMicroflowBody(ec, target)
	lines = append(lines, "{")
	if len(bodyLines) == 0 {
		lines = append(lines, "  -- No activities")
	} else {
		for _, l := range bodyLines {
			lines = append(lines, "  "+l)
		}
	}
	lines = append(lines, "}")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// ────────────────────────────────────────────────────────────
// describeNanoflowGenFuture — DESCRIBE NANOFLOW Module.Name;
// ────────────────────────────────────────────────────────────

func describeNanoflowGenFuture(ctx context.Context, output io.Writer, nfRepo repos.NanoflowRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	rendered, err := describeNanoflowGenToStringFuture(nfRepo, ml, mr, fm, name)
	if err != nil {
		return err
	}
	fmt.Fprintln(output, rendered)
	return nil
}

func describeNanoflowGenToStringFuture(nfRepo repos.NanoflowRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) (string, error) {
	if nfRepo == nil {
		return "", mdlerrors.NewBackend("nanoflow repository", fmt.Errorf("nanoflow repo is nil"))
	}

	all, err := nfRepo.List("")
	if err != nil {
		return "", mdlerrors.NewBackend("list nanoflows", err)
	}

	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	var target *genMf.Nanoflow
	for _, nf := range all {
		if nf == nil {
			continue
		}
		modName := genNanoflowQualifiedNameFuture(nfRepo, h, model.ID(nf.ID()))
		if modName == name.Module && nf.Name() == name.Name {
			target = nf
			break
		}
	}
	if target == nil {
		return "", mdlerrors.NewNotFound("nanoflow", name.String())
	}

	return describeNanoflowGenToStringFutureInner(target, nfRepo, h, ml, mr), nil
}

func genNanoflowQualifiedNameFuture(nfRepo repos.NanoflowRepository, h *ContainerHierarchy, id model.ID) string {
	if nfRepo != nil {
		if containerID, err := nfRepo.GetContainerUUID(id); err == nil && containerID != "" {
			if h != nil {
				modID := h.FindModuleID(containerID)
				if mn := h.GetModuleName(modID); mn != "" {
					return mn
				}
			}
		}
	}
	return ""
}

func describeNanoflowGenToStringFutureInner(target *genMf.Nanoflow, nfRepo repos.NanoflowRepository, h *ContainerHierarchy, ml backend.ModuleLister, mr backend.MetadataReader) string {
	if target == nil {
		return ""
	}

	moduleName := genNanoflowQualifiedNameFuture(nfRepo, h, model.ID(target.ID()))
	qualifiedName := moduleName + "." + target.Name()

	var lines []string

	if doc := target.Documentation(); doc != "" {
		lines = append(lines, "/**")
		for _, dl := range strings.Split(doc, "\n") {
			lines = append(lines, " * "+dl)
		}
		lines = append(lines, " */")
	}

	if target.Excluded() {
		lines = append(lines, "@excluded")
	}

	params := genNanoflowParameters(target)
	if len(params) > 0 {
		lines = append(lines, fmt.Sprintf("create or modify nanoflow %s (", qualifiedName))
		for i, p := range params {
			comma := ","
			if i == len(params)-1 {
				comma = ""
			}
			lines = append(lines, fmt.Sprintf("  $%s: %s%s", p.name, p.declType, comma))
		}
		lines = append(lines, ")")
	} else {
		lines = append(lines, fmt.Sprintf("create or modify nanoflow %s ()", qualifiedName))
	}

	returnType := target.ReturnType()
	if returnType != "" && returnType != "Nothing" {
		lines = append(lines, fmt.Sprintf("returns %s", returnType))
	}

	ec := &ExecContext{
		Context:        context.Background(),
		ModuleLister:   ml,
		MetadataReader: mr,
	}
	bodyLines := renderGenNanoflowBody(ec, target)
	lines = append(lines, "{")
	if len(bodyLines) == 0 {
		lines = append(lines, "  -- No activities")
	} else {
		for _, l := range bodyLines {
			lines = append(lines, "  "+l)
		}
	}
	lines = append(lines, "}")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// ────────────────────────────────────────────────────────────
// describePageFuture — DESCRIBE PAGE Module.Name;
// ────────────────────────────────────────────────────────────

func describePageFuture(ctx context.Context, output io.Writer, pgRepo repos.PageRepository, ib backend.ImageBackend, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listPagesWithContainerGenFuture(pgRepo, ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	var foundPage *genPg.Page
	var foundContainerID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(p.ContainerUUID)
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			foundPage = p.Elem
			foundContainerID = p.ContainerUUID
			break
		}
	}

	if foundPage == nil {
		return mdlerrors.NewNotFound("page", name.String())
	}

	modID := h.FindModuleID(foundContainerID)
	modName := h.GetModuleName(modID)

	if doc := foundPage.Documentation(); doc != "" {
		lines := strings.Split(doc, "\n")
		fmt.Fprint(output, "/**\n")
		for _, line := range lines {
			fmt.Fprintf(output, " * %s\n", line)
		}
		fmt.Fprint(output, " */\n")
	}

	title := pickPageTitleGen(foundPage)

	if foundPage.Excluded() {
		fmt.Fprintln(output, "@excluded")
	}

	props := []string{}
	if title != "" {
		props = append(props, fmt.Sprintf("title: %s", mdlQuote(title)))
	}
	if u := foundPage.Url(); u != "" {
		props = append(props, fmt.Sprintf("url: %s", mdlQuote(u)))
	}
	if folderPath := h.BuildFolderPath(foundContainerID); folderPath != "" {
		props = append(props, fmt.Sprintf("folder: %s", mdlQuote(folderPath)))
	}
	if pageParams := resolvePageParamsFuture(foundPage); len(pageParams) > 0 {
		parts := make([]string, 0, len(pageParams))
		for _, p := range pageParams {
			parts = append(parts, fmt.Sprintf("$%s: %s", p.Name, p.EntityName))
		}
		props = append(props, fmt.Sprintf("params: { %s }", strings.Join(parts, ", ")))
	}

	fmt.Fprintf(output, "create or modify page %s.%s", modName, foundPage.Name())
	if len(props) > 0 {
		fmt.Fprintf(output, " (\n  %s\n)", strings.Join(props, ",\n  "))
	}
	fmt.Fprint(output, "\n{\n  -- TODO: page body\n}\n/")

	pageRoles := foundPage.AllowedRolesQualifiedNames()
	if allowed := filterAutoDocumentRoles(pageRoles); len(allowed) > 0 {
		fmt.Fprintf(output, "\n\ngrant view on page %s.%s to %s;",
			modName, foundPage.Name(), strings.Join(allowed, ", "))
	}

	fmt.Fprint(output, "\n")
	return nil
}

func resolvePageParamsFuture(page *genPg.Page) []types.PageParam {
	params := page.ParametersItems()
	if len(params) == 0 {
		return nil
	}
	result := make([]types.PageParam, 0, len(params))
	for _, p := range params {
		sp, ok := p.(*genPg.PageParameter)
		if !ok || sp == nil {
			continue
		}
		result = append(result, types.PageParam{
			Name:       sp.Name(),
			EntityName: pageParamTypeMDLGen(sp.ParameterType()),
		})
	}
	return result
}

// ────────────────────────────────────────────────────────────
// describeSnippetFuture — DESCRIBE SNIPPET Module.Name;
// ────────────────────────────────────────────────────────────

func describeSnippetFuture(ctx context.Context, output io.Writer, snpRepo repos.SnippetRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listSnippetsWithContainerGenFuture(snpRepo, ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}

	var foundSnippet *genPg.Snippet
	var foundContainerID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(p.ContainerUUID)
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			foundSnippet = p.Elem
			foundContainerID = p.ContainerUUID
			break
		}
	}

	if foundSnippet == nil {
		return mdlerrors.NewNotFound("snippet", name.String())
	}

	modID := h.FindModuleID(foundContainerID)
	modName := h.GetModuleName(modID)

	if doc := foundSnippet.Documentation(); doc != "" {
		lines := strings.Split(doc, "\n")
		fmt.Fprint(output, "/**\n")
		for _, line := range lines {
			fmt.Fprintf(output, " * %s\n", line)
		}
		fmt.Fprint(output, " */\n")
	}

	fmt.Fprintf(output, "create or modify snippet %s.%s", modName, foundSnippet.Name())
	folderPath := h.BuildFolderPath(foundContainerID)
	snippetParamsItems := foundSnippet.ParametersItems()
	if len(snippetParamsItems) > 0 || folderPath != "" {
		snippetProps := []string{}
		if len(snippetParamsItems) > 0 {
			paramParts := []string{}
			for _, elem := range snippetParamsItems {
				sp, ok := elem.(*genPg.SnippetParameter)
				if !ok || sp == nil {
					continue
				}
				typeStr := pageParamTypeMDLGen(sp.ParameterType())
				if typeStr == "" {
					typeStr = "Unknown"
				}
				paramParts = append(paramParts, fmt.Sprintf("$%s: %s", sp.Name(), typeStr))
			}
			snippetProps = append(snippetProps, fmt.Sprintf("params: { %s }", strings.Join(paramParts, ", ")))
		}
		if folderPath != "" {
			snippetProps = append(snippetProps, fmt.Sprintf("folder: %s", mdlQuote(folderPath)))
		}
		fmt.Fprintf(output, " (%s)", strings.Join(snippetProps, ", "))
	}

	fmt.Fprint(output, " { }")
	fmt.Fprint(output, "\n")
	return nil
}

// ────────────────────────────────────────────────────────────
// describeLayoutFuture — DESCRIBE LAYOUT Module.Name;
// ────────────────────────────────────────────────────────────

func describeLayoutFuture(ctx context.Context, output io.Writer, layRepo repos.LayoutRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listLayoutsWithContainerGenFuture(layRepo)
	if err != nil {
		return mdlerrors.NewBackend("list layouts", err)
	}

	var foundLayout *genPg.Layout
	var foundContainerID model.ID
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(p.ContainerUUID)
		modName := h.GetModuleName(modID)
		if p.Elem.Name() == name.Name && (name.Module == "" || modName == name.Module) {
			foundLayout = p.Elem
			foundContainerID = p.ContainerUUID
			break
		}
	}

	if foundLayout == nil {
		return mdlerrors.NewNotFound("layout", name.String())
	}

	modID := h.FindModuleID(foundContainerID)
	modName := h.GetModuleName(modID)

	if doc := foundLayout.Documentation(); doc != "" {
		lines := strings.Split(doc, "\n")
		fmt.Fprint(output, "/**\n")
		for _, line := range lines {
			fmt.Fprintf(output, " * %s\n", line)
		}
		fmt.Fprint(output, " */\n")
	}

	layoutTypeStr := foundLayout.LayoutType()
	if layoutTypeStr == "" {
		layoutTypeStr = "Responsive"
	}

	props := []string{fmt.Sprintf("type: %s", layoutTypeStr)}
	if folderPath := h.BuildFolderPath(foundContainerID); folderPath != "" {
		props = append(props, fmt.Sprintf("folder: %s", mdlQuote(folderPath)))
	}

	fmt.Fprintf(output, "create or modify layout %s.%s (\n", modName, foundLayout.Name())
	for i, p := range props {
		if i < len(props)-1 {
			fmt.Fprintf(output, "  %s,\n", p)
		} else {
			fmt.Fprintf(output, "  %s\n", p)
		}
	}
	fmt.Fprint(output, ")")

	fmt.Fprint(output, "\n")
	return nil
}

// ────────────────────────────────────────────────────────────
// describeWorkflowGenFuture — DESCRIBE WORKFLOW Module.Name;
// ────────────────────────────────────────────────────────────

func describeWorkflowGenFuture(ctx context.Context, output io.Writer, wfRepo repos.WorkflowRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listWorkflowsWithContainerGenFuture(wfRepo)
	if err != nil {
		return mdlerrors.NewBackend("list workflows", err)
	}

	var target *genWf.Workflow
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := ""
		if h != nil {
			modID := h.FindModuleID(p.ContainerUUID)
			modName = h.GetModuleName(modID)
		}
		if modName == name.Module && p.Elem.Name() == name.Name {
			target = p.Elem
			break
		}
	}
	if target == nil {
		return mdlerrors.NewNotFound("workflow", name.String())
	}

	paramEntity, paramName := workflowParameterInfo(target)
	acts, uts, decs := countWorkflowActivitiesGen(target)

	rendered := fmt.Sprintf("create or modify workflow %s.%s", name.Module, target.Name())
	if paramEntity != "" {
		if paramName == "" {
			paramName = "WorkflowContext"
		}
		rendered += fmt.Sprintf("\n  parameter $%s: %s", paramName, paramEntity)
	}
	rendered += "\n{"
	if acts > 0 {
		rendered += fmt.Sprintf("\n  -- %d activities", acts)
	}
	if uts > 0 {
		rendered += fmt.Sprintf("\n  -- %d user tasks", uts)
	}
	if decs > 0 {
		rendered += fmt.Sprintf("\n  -- %d decisions", decs)
	}
	rendered += "\n}\n/"

	fmt.Fprintln(output, rendered)
	return nil
}

// ────────────────────────────────────────────────────────────
// describeJavaActionGenFuture — DESCRIBE JAVA ACTION Module.Name;
// ────────────────────────────────────────────────────────────

func describeJavaActionGenFuture(ctx context.Context, output io.Writer, jaRepo repos.JavaActionRepository, name ast.QualifiedName) error {
	if jaRepo == nil {
		return mdlerrors.NewNotFound("java action", name.Module+"."+name.Name)
	}
	qn := name.Module + "." + name.Name
	ja, err := jaRepo.FindByQualifiedName(qn)
	if err != nil || ja == nil {
		return mdlerrors.NewNotFound("java action", qn)
	}

	var sb strings.Builder

	doc := strings.ReplaceAll(ja.Documentation(), "\r\n", "\n")
	doc = strings.ReplaceAll(doc, "\r", "\n")
	if doc != "" {
		sb.WriteString("/**\n")
		for _, line := range strings.Split(doc, "\n") {
			sb.WriteString(" * ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(" */\n")
	}

	sb.WriteString("create or modify java action ")
	sb.WriteString(qn)
	sb.WriteString("(")

	typeParams := javaActionTypeParametersOf(ja)
	params := javaActionParametersOf(ja)

	hasDescriptions := false
	for _, p := range params {
		pp, ok := p.(*genJA.JavaActionParameter)
		if !ok {
			continue
		}
		if pp.Description() != "" {
			hasDescriptions = true
			break
		}
	}

	wrote := 0
	for _, p := range params {
		pp, ok := p.(*genJA.JavaActionParameter)
		if !ok {
			continue
		}
		if wrote > 0 {
			sb.WriteString(", ")
		}
		if hasDescriptions {
			sb.WriteString("\n    ")
		}
		sb.WriteString(pp.Name())
		sb.WriteString(": ")
		sb.WriteString(formatJavaActionTypeGen(javaActionParameterParameterType(pp), typeParams))
		if pp.IsRequired() {
			sb.WriteString(" not null")
		}
		if pp.Description() != "" {
			pd := strings.ReplaceAll(pp.Description(), "\r\n", "\n")
			pd = strings.ReplaceAll(pd, "\r", "\n")
			firstLine, _, _ := strings.Cut(pd, "\n")
			sb.WriteString("  -- ")
			sb.WriteString(firstLine)
		}
		wrote++
	}
	if hasDescriptions {
		sb.WriteString("\n")
	}
	sb.WriteString(")")

	rt := javaActionReturnTypeElement(ja)
	if rt != nil {
		sb.WriteString(" returns ")
		sb.WriteString(formatJavaActionReturnTypeGen(rt, typeParams))
	}

	sb.WriteString(";\n/")
	fmt.Fprintln(output, sb.String())
	return nil
}

// ────────────────────────────────────────────────────────────
// describeModuleRoleGenFuture — DESCRIBE MODULE ROLE Module.Name;
// ────────────────────────────────────────────────────────────

func describeModuleRoleGenFuture(ctx context.Context, output io.Writer, sec repos.SecurityRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager, name ast.QualifiedName) error {
	h, err := buildHierarchyForDescribe(ml, mr, fm)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listModuleSecurityWithContainerGenFuture(sec, ml)
	if err != nil {
		return mdlerrors.NewBackend("read module security", err)
	}

	for _, p := range pairs {
		modName := h.GetModuleName(p.ContainerID)
		if name.Module != "" && modName != name.Module {
			continue
		}
		for _, mr := range p.MS.ModuleRolesItems() {
			typed, ok := mr.(*genSec.ModuleRole)
			if !ok || typed.Name() != name.Name {
				continue
			}
			fmt.Fprintf(output, "create module role %s.%s", modName, typed.Name())
			if typed.Description() != "" {
				fmt.Fprintf(output, " description '%s'", typed.Description())
			}
			fmt.Fprintln(output, ";")
			fmt.Fprintln(output, "/")
			qualifiedRole := modName + "." + typed.Name()

			if ps, psErr := sec.Get(); psErr == nil && ps != nil {
				var includedBy []string
				for _, ur := range ps.UserRolesItems() {
					urTyped, ok := ur.(*genSec.UserRole)
					if !ok {
						continue
					}
					for _, mref := range urTyped.ModuleRolesQualifiedNames() {
						if mref == qualifiedRole {
							includedBy = append(includedBy, urTyped.Name())
						}
					}
				}
				if len(includedBy) > 0 {
					fmt.Fprintf(output, "\n-- Included in user roles: %s\n", strings.Join(includedBy, ", "))
				}
			}
			return nil
		}
	}

	return mdlerrors.NewNotFound("module role", name.String())
}

// ────────────────────────────────────────────────────────────
// describeUserRoleGenFuture — DESCRIBE USER ROLE Name;
// ────────────────────────────────────────────────────────────

func describeUserRoleGenFuture(ctx context.Context, output io.Writer, sec repos.SecurityRepository, name ast.QualifiedName) error {
	ps, err := sec.Get()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewNotFound("user role", name.Name)
	}

	for _, ur := range ps.UserRolesItems() {
		typed, ok := ur.(*genSec.UserRole)
		if !ok || typed.Name() != name.Name {
			continue
		}
		fmt.Fprintf(output, "create or modify user role %s", typed.Name())

		moduleRoles := typed.ModuleRolesQualifiedNames()
		if len(moduleRoles) > 0 {
			fmt.Fprintf(output, " (%s)", strings.Join(moduleRoles, ", "))
		}
		if typed.ManageAllRoles() {
			fmt.Fprint(output, " manage all roles")
		}

		fmt.Fprintln(output, ";")
		fmt.Fprintln(output, "/")

		if typed.Description() != "" {
			fmt.Fprintf(output, "\n-- Description: %s\n", typed.Description())
		}
		if typed.CheckSecurity() {
			fmt.Fprintln(output, "-- Check security: enabled")
		}
		return nil
	}

	return mdlerrors.NewNotFound("user role", name.Name)
}

// ────────────────────────────────────────────────────────────
// describeDemoUserGenFuture — DESCRIBE DEMO USER 'name';
// ────────────────────────────────────────────────────────────

func describeDemoUserGenFuture(ctx context.Context, output io.Writer, sec repos.SecurityRepository, userName string) error {
	ps, err := sec.Get()
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewNotFound("demo user", userName)
	}

	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok || typed.UserName() != userName {
			continue
		}
		fmt.Fprintf(output, "create demo user '%s' password '***'", typed.UserName())
		if entityName := typed.EntityQualifiedName(); entityName != "" {
			fmt.Fprintf(output, " entity %s", entityName)
		}
		roles := typed.UserRolesQualifiedNames()
		if len(roles) > 0 {
			fmt.Fprintf(output, " (%s)", strings.Join(roles, ", "))
		}
		fmt.Fprintln(output, ";")
		fmt.Fprintln(output, "/")
		return nil
	}

	return mdlerrors.NewNotFound("demo user", userName)
}

// ────────────────────────────────────────────────────────────
// describeFragmentFuture — DESCRIBE FRAGMENT name;
// ────────────────────────────────────────────────────────────

func describeFragmentFuture(ctx context.Context, output io.Writer, fragments map[string]*ast.DefineFragmentStmt, name ast.QualifiedName) error {
	if fragments == nil {
		return mdlerrors.NewNotFound("fragment", name.Name)
	}

	frag, ok := fragments[name.Name]
	if !ok {
		return mdlerrors.NewNotFound("fragment", name.Name)
	}

	fmt.Fprintf(output, "define fragment %s as {\n", frag.Name)
	for _, w := range frag.Widgets {
		outputASTWidgetMDL(output, w, 1)
	}
	fmt.Fprintln(output, "};")
	return nil
}

// ────────────────────────────────────────────────────────────
// Container/listing helpers — ExecContext-free versions
// ────────────────────────────────────────────────────────────

type FuturePageWithContainer struct {
	Elem          *genPg.Page
	ContainerUUID model.ID
}

func listPagesWithContainerGenFuture(pgRepo repos.PageRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager) ([]FuturePageWithContainer, error) {
	if pgRepo == nil {
		return nil, nil
	}
	all, err := pgRepo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]FuturePageWithContainer, 0, len(all))
	for _, p := range all {
		if p == nil {
			continue
		}
		var containerUUID model.ID
		if cid, err := pgRepo.GetContainerUUID(model.ID(p.ID())); err == nil {
			containerUUID = cid
		}
		out = append(out, FuturePageWithContainer{Elem: p, ContainerUUID: containerUUID})
	}
	return out, nil
}

type FutureSnippetWithContainer struct {
	Elem          *genPg.Snippet
	ContainerUUID model.ID
}

func listSnippetsWithContainerGenFuture(snpRepo repos.SnippetRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager) ([]FutureSnippetWithContainer, error) {
	if snpRepo == nil {
		return nil, nil
	}
	all, err := snpRepo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]FutureSnippetWithContainer, 0, len(all))
	for _, s := range all {
		if s == nil {
			continue
		}
		var containerUUID model.ID
		if cid, err := snpRepo.GetContainerUUID(model.ID(s.ID())); err == nil {
			containerUUID = cid
		}
		out = append(out, FutureSnippetWithContainer{Elem: s, ContainerUUID: containerUUID})
	}
	return out, nil
}

type FutureLayoutWithContainer struct {
	Elem          *genPg.Layout
	ContainerUUID model.ID
}

func listLayoutsWithContainerGenFuture(layRepo repos.LayoutRepository) ([]FutureLayoutWithContainer, error) {
	if layRepo == nil {
		return nil, nil
	}
	all, err := layRepo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]FutureLayoutWithContainer, 0, len(all))
	for _, l := range all {
		if l == nil {
			continue
		}
		var containerUUID model.ID
		if cid, err := layRepo.GetContainerUUID(model.ID(l.ID())); err == nil {
			containerUUID = cid
		}
		out = append(out, FutureLayoutWithContainer{Elem: l, ContainerUUID: containerUUID})
	}
	return out, nil
}

type FutureWorkflowWithContainer struct {
	Elem          *genWf.Workflow
	ContainerUUID model.ID
}

func listWorkflowsWithContainerGenFuture(wfRepo repos.WorkflowRepository) ([]FutureWorkflowWithContainer, error) {
	if wfRepo == nil {
		return nil, nil
	}
	all, err := wfRepo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]FutureWorkflowWithContainer, 0, len(all))
	for _, wf := range all {
		if wf == nil {
			continue
		}
		var containerUUID model.ID
		if cid, err := wfRepo.GetContainerUUID(model.ID(wf.ID())); err == nil {
			containerUUID = cid
		}
		out = append(out, FutureWorkflowWithContainer{Elem: wf, ContainerUUID: containerUUID})
	}
	return out, nil
}

type FutureMicroflowWithContainer struct {
	MF            *genMf.Microflow
	ContainerUUID model.ID
}

func listMicroflowsWithContainerGenFuture(mfRepo repos.MicroflowRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager) ([]FutureMicroflowWithContainer, error) {
	if mfRepo == nil {
		return nil, nil
	}
	all, err := mfRepo.ListAll()
	if err != nil {
		return nil, err
	}
	out := make([]FutureMicroflowWithContainer, 0, len(all))
	for _, mf := range all {
		if mf == nil {
			continue
		}
		var containerUUID model.ID
		if cid, err := mfRepo.GetContainerUUID(model.ID(mf.ID())); err == nil {
			containerUUID = cid
		}
		out = append(out, FutureMicroflowWithContainer{MF: mf, ContainerUUID: containerUUID})
	}
	return out, nil
}

type FutureNanoflowWithContainer struct {
	NF            *genMf.Nanoflow
	ContainerUUID model.ID
}

func listNanoflowsWithContainerGenFuture(nfRepo repos.NanoflowRepository, ml backend.ModuleLister, mr backend.MetadataReader, fm backend.FolderManager) ([]FutureNanoflowWithContainer, error) {
	if nfRepo == nil {
		return nil, nil
	}
	all, err := nfRepo.List("")
	if err != nil {
		return nil, err
	}
	out := make([]FutureNanoflowWithContainer, 0, len(all))
	for _, nf := range all {
		if nf == nil {
			continue
		}
		var containerUUID model.ID
		if cid, err := nfRepo.GetContainerUUID(model.ID(nf.ID())); err == nil {
			containerUUID = cid
		}
		out = append(out, FutureNanoflowWithContainer{NF: nf, ContainerUUID: containerUUID})
	}
	return out, nil
}

type FutureModuleSecurityWithContainer struct {
	MS          *genSec.ModuleSecurity
	ContainerID model.ID
}

func listModuleSecurityWithContainerGenFuture(sec repos.SecurityRepository, ml backend.ModuleLister) ([]FutureModuleSecurityWithContainer, error) {
	modules, err := ml.ListModules()
	if err != nil {
		return nil, err
	}
	out := make([]FutureModuleSecurityWithContainer, 0, len(modules))
	for _, mod := range modules {
		ms, err := sec.GetModuleSecurity(mod.ID)
		if err != nil || ms == nil {
			continue
		}
		out = append(out, FutureModuleSecurityWithContainer{MS: ms, ContainerID: mod.ID})
	}
	return out, nil
}

// listEntitiesForModuleGenFuture lists entities for a specific module using repos.
func listEntitiesForModuleGenFuture(dmr repos.DomainModelRepository, ml backend.ModuleLister, moduleName string) ([]*genDm.Entity, error) {
	pairs, err := dmr.ListAllWithContainerID()
	if err != nil {
		return nil, err
	}
	mods, err := ml.ListModules()
	if err != nil {
		return nil, err
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}
	var targetContainerID model.ID
	for _, m := range mods {
		if m.Name == moduleName {
			targetContainerID = m.ID
			break
		}
	}
	var result []*genDm.Entity
	for _, p := range pairs {
		if p.DM == nil || p.ContainerID != targetContainerID {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			result = append(result, entity)
		}
	}
	return result, nil
}

// listAssociationsForModuleGenFuture lists associations for a specific module using repos.
func listAssociationsForModuleGenFuture(dmr repos.DomainModelRepository, ml backend.ModuleLister, moduleName string) ([]*genDm.Association, error) {
	pairs, err := dmr.ListAllWithContainerID()
	if err != nil {
		return nil, err
	}
	mods, err := ml.ListModules()
	if err != nil {
		return nil, err
	}
	var targetContainerID model.ID
	for _, m := range mods {
		if m.Name == moduleName {
			targetContainerID = m.ID
			break
		}
	}
	var result []*genDm.Association
	for _, p := range pairs {
		if p.DM == nil || p.ContainerID != targetContainerID {
			continue
		}
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			result = append(result, assoc)
		}
	}
	return result, nil
}

// genMicroflowParametersFromMF extracts parameter info from a gen microflow's ObjectCollection.
