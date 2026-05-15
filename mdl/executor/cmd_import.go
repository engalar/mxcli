// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	sqllib "github.com/mendixlabs/mxcli/sql"
)

// execImport handles IMPORT FROM <alias> QUERY '<sql>' INTO Module.Entity MAP (...) [LINK (...)] [BATCH n] [LIMIT n]
func execImport(ctx *ExecContext, s *ast.ImportStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	// Validate entity exists
	tableName, err := sqllib.EntityToTableName(s.TargetEntity)
	if err != nil {
		return err
	}

	// Get source connection (auto-connects from config if needed)
	sourceConn, err := getOrAutoConnect(ctx, s.SourceAlias)
	if err != nil {
		return fmt.Errorf("source connection: %w", err)
	}

	// Get or create Mendix DB connection
	targetConn, err := ensureMendixDBConnection(ctx)
	if err != nil {
		return err
	}

	// Build column mappings
	colMap := make([]sqllib.ColumnMapping, len(s.Mappings))
	for i, m := range s.Mappings {
		colMap[i] = sqllib.ColumnMapping{
			SourceName: m.SourceColumn,
			TargetName: sqllib.AttributeToColumnName(m.TargetAttr),
		}
	}

	// Resolve association LINK mappings
	goCtx, cancel := context.WithTimeout(ctx.Context, 10*time.Minute)
	defer cancel()

	assocs, err := resolveImportLinks(ctx, goCtx, targetConn, s)
	if err != nil {
		return err
	}

	cfg := &sqllib.ImportConfig{
		SourceConn:  sourceConn,
		TargetConn:  targetConn,
		SourceQuery: s.Query,
		TargetTable: tableName,
		EntityName:  s.TargetEntity,
		ColumnMap:   colMap,
		Assocs:      assocs,
		BatchSize:   s.BatchSize,
		Limit:       s.Limit,
	}

	start := time.Now()

	result, err := sqllib.ExecuteImport(goCtx, cfg, func(batch, rows int) {
		fmt.Fprintf(ctx.Output, "  batch %d: %d rows imported\n", batch, rows)
	})
	if err != nil {
		return mdlerrors.NewBackend("import", err)
	}

	elapsed := time.Since(start)
	fmt.Fprintf(ctx.Output, "Imported %d rows into %s (%d batches, %s)\n",
		result.TotalRows, s.TargetEntity, result.BatchesWritten, elapsed.Round(time.Millisecond))

	// Report association link stats
	for _, a := range assocs {
		linked := result.LinksCreated[a.AssociationName]
		missed := result.LinksMissed[a.AssociationName]
		if missed > 0 {
			fmt.Fprintf(ctx.Output, "  %s: linked %d/%d rows (%d null — lookup value not found)\n",
				a.AssociationName, linked, linked+missed, missed)
		} else if linked > 0 {
			fmt.Fprintf(ctx.Output, "  %s: linked %d rows\n", a.AssociationName, linked)
		}
	}

	return nil
}

// resolveImportLinks resolves LINK mappings from the AST into AssocInfo structs
// by looking up association metadata from the MPR and the Mendix system tables.
func resolveImportLinks(ctx *ExecContext, goCtx context.Context, mendixConn *sqllib.Connection, s *ast.ImportStmt) ([]*sqllib.AssocInfo, error) {
	if len(s.Links) == 0 {
		return nil, nil
	}

	fmt.Fprintf(ctx.Output, "Resolving associations...\n")

	// Parse target entity module
	targetParts := strings.SplitN(s.TargetEntity, ".", 2)
	if len(targetParts) != 2 {
		return nil, mdlerrors.NewValidationf("invalid target entity %q", s.TargetEntity)
	}
	targetModule := targetParts[0]

	// Load domain models to find associations
	dms, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return nil, mdlerrors.NewBackend("list domain models", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return nil, mdlerrors.NewBackend("get hierarchy", err)
	}

	// Build entity ID → qualified name map
	entityNames := make(map[string]string) // entity ID string → "Module.Entity"
	for _, pair := range dms {
		if pair.DM == nil {
			continue
		}
		modID := h.FindModuleID(pair.ContainerID)
		modName := h.GetModuleName(modID)
		for _, item := range pair.DM.EntitiesItems() {
			ent, ok := item.(*genDm.Entity)
			if !ok || ent == nil {
				continue
			}
			entityNames[string(ent.ID())] = modName + "." + ent.Name()
		}
	}

	// Resolve each LINK mapping
	var assocs []*sqllib.AssocInfo
	for _, link := range s.Links {
		info, err := resolveOneLink(ctx, goCtx, mendixConn, link, targetModule, dms, h, entityNames)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(ctx.Output, "  %s: %s storage", info.AssociationName, info.StorageFormat)
		if info.LookupAttr != "" {
			fmt.Fprintf(ctx.Output, ", lookup by %s.%s (%d values cached)",
				info.ChildEntity, info.LookupAttr, len(info.LookupCache))
		} else {
			fmt.Fprintf(ctx.Output, ", direct ID")
		}
		fmt.Fprintln(ctx.Output)
		assocs = append(assocs, info)
	}

	return assocs, nil
}

// resolveOneLink resolves a single LINK mapping to an AssocInfo.
func resolveOneLink(
	ctx *ExecContext,
	goCtx context.Context,
	mendixConn *sqllib.Connection,
	link ast.LinkMapping,
	targetModule string,
	dms []DomainModelGenWithContainer,
	h *ContainerHierarchy,
	entityNames map[string]string,
) (*sqllib.AssocInfo, error) {

	// Find the association in the domain models
	assocQualName := targetModule + "." + link.AssociationName
	var foundAssoc *genDm.Association
	var foundCross *genDm.CrossAssociation

	for _, pair := range dms {
		if pair.DM == nil {
			continue
		}
		modID := h.FindModuleID(pair.ContainerID)
		modName := h.GetModuleName(modID)
		if modName != targetModule {
			continue
		}
		for _, item := range pair.DM.AssociationsItems() {
			a, ok := item.(*genDm.Association)
			if ok && a.Name() == link.AssociationName {
				foundAssoc = a
				break
			}
		}
		if foundAssoc != nil {
			break
		}
		for _, item := range pair.DM.CrossAssociationsItems() {
			ca, ok := item.(*genDm.CrossAssociation)
			if ok && ca.Name() == link.AssociationName {
				foundCross = ca
				break
			}
		}
		if foundCross != nil {
			break
		}
	}

	if foundAssoc == nil && foundCross == nil {
		return nil, mdlerrors.NewNotFoundMsg("association", link.AssociationName, fmt.Sprintf("association %q not found in module %q", link.AssociationName, targetModule))
	}

	// Extract association info
	var storageFormat string
	var childEntity string
	var assocType string

	if foundAssoc != nil {
		storageFormat = foundAssoc.StorageFormat()
		if storageFormat == "" {
			storageFormat = "Table" // default
		}
		childEntity = entityNames[string(foundAssoc.ChildRefID())]
		assocType = foundAssoc.Type()
	} else {
		storageFormat = foundCross.StorageFormat()
		if storageFormat == "" {
			storageFormat = "Table"
		}
		childEntity = foundCross.ChildQualifiedName()
		assocType = foundCross.Type()
	}

	// Reject ReferenceSet associations (not supported in MVP)
	if assocType == "ReferenceSet" {
		return nil, mdlerrors.NewUnsupported(fmt.Sprintf("association %q is ReferenceSet — not supported in import link (use manual sql)", assocQualName))
	}

	if childEntity == "" {
		return nil, mdlerrors.NewValidationf("could not resolve child entity for association %q", assocQualName)
	}

	info := &sqllib.AssocInfo{
		SourceColumn:    link.SourceColumn,
		LookupAttr:      link.LookupAttr,
		AssociationName: assocQualName,
		ChildEntity:     childEntity,
		StorageFormat:   storageFormat,
	}

	// Try to get exact column/table names from mendixsystem$association
	sysInfo, err := sqllib.LookupAssociationInfo(goCtx, mendixConn, assocQualName)
	if err != nil {
		return nil, err
	}

	if sysInfo != nil {
		// Use system table info
		info.StorageFormat = sysInfo.StorageFormat
		if info.StorageFormat == "Column" {
			info.FKColumnName = sysInfo.ChildColumnName
			// Fall back to naming convention if system table has NULL child_column_name
			if info.FKColumnName == "" {
				info.FKColumnName = sqllib.AssocColumnNameFromConvention(assocQualName)
			}
		} else {
			info.JunctionTable = sysInfo.TableName
			// Junction column names from conventions
			parentParts := strings.SplitN(entityNames[string(getParentID(foundAssoc, foundCross))], ".", 2)
			childParts := strings.SplitN(childEntity, ".", 2)
			if len(parentParts) == 2 && len(childParts) == 2 {
				info.ParentColName = sqllib.JunctionColumnFromConvention(entityNames[string(getParentID(foundAssoc, foundCross))])
				info.ChildColName = sqllib.JunctionColumnFromConvention(childEntity)
			}
		}
	} else {
		// Fall back to naming conventions
		if info.StorageFormat == "Column" {
			info.FKColumnName = sqllib.AssocColumnNameFromConvention(assocQualName)
		} else {
			info.JunctionTable = sqllib.JunctionTableFromConvention(assocQualName)
			info.ParentColName = sqllib.JunctionColumnFromConvention(getParentEntityName(foundAssoc, foundCross, entityNames))
			info.ChildColName = sqllib.JunctionColumnFromConvention(childEntity)
		}
	}

	// Build lookup cache if ON clause specified
	if link.LookupAttr != "" {
		childTable, err := sqllib.EntityToTableName(childEntity)
		if err != nil {
			// Kept as fmt.Errorf: wraps a cause with entity-specific context, not a standard "failed to" pattern.
			return nil, fmt.Errorf("invalid child entity %q: %w", childEntity, err)
		}
		lookupCol := sqllib.AttributeToColumnName(link.LookupAttr)
		cache, err := sqllib.BuildLookupCache(goCtx, mendixConn, childTable, lookupCol)
		if err != nil {
			return nil, mdlerrors.NewBackend(fmt.Sprintf("build lookup cache for %s", assocQualName), err)
		}
		info.LookupCache = cache
		if len(cache) == 0 {
			fmt.Fprintf(ctx.Output, "  warning: child table %q is empty; all %s associations will be null\n",
				childTable, assocQualName)
		}
	}

	return info, nil
}

// getParentID returns the parent entity ID from either a regular or cross-module association.
func getParentID(a *genDm.Association, ca *genDm.CrossAssociation) string {
	if a != nil {
		return string(a.ParentRefID())
	}
	if ca != nil {
		return string(ca.ParentRefID())
	}
	return ""
}

// getParentEntityName returns the parent entity qualified name.
func getParentEntityName(a *genDm.Association, ca *genDm.CrossAssociation, entityNames map[string]string) string {
	id := getParentID(a, ca)
	return entityNames[id]
}

// ensureMendixDBConnection reads the project settings and auto-connects to the Mendix app DB.
func ensureMendixDBConnection(ctx *ExecContext) (*sqllib.Connection, error) {
	mgr := ensureSQLManager(ctx)

	// Check if already connected
	if conn, err := mgr.Get(sqllib.MendixDBAlias); err == nil {
		return conn, nil
	}

	// Read project settings to get DB configuration
	ps, err := ctx.Backend.GetProjectSettings()
	if err != nil {
		return nil, mdlerrors.NewBackend("read project settings", err)
	}

	if ps.Configuration == nil || len(ps.Configuration.Configurations) == 0 {
		return nil, mdlerrors.NewValidation("no server configurations found in project settings")
	}

	// Use the first configuration (typically "default")
	cfg := ps.Configuration.Configurations[0]

	dsn, err := sqllib.BuildMendixDSN(cfg.DatabaseType, cfg.DatabaseUrl, cfg.DatabaseName,
		cfg.DatabaseUserName, cfg.DatabasePassword)
	if err != nil {
		return nil, fmt.Errorf("cannot build Mendix DB DSN: %w", err)
	}

	if err := mgr.Connect(sqllib.DriverPostgres, dsn, sqllib.MendixDBAlias); err != nil {
		return nil, mdlerrors.NewBackend("connect to Mendix app database", err)
	}

	fmt.Fprintf(ctx.Output, "Auto-connected to Mendix app database as '%s'\n", sqllib.MendixDBAlias)

	conn, err := mgr.Get(sqllib.MendixDBAlias)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
