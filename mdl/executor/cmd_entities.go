// SPDX-License-Identifier: Apache-2.0

// Package executor - Entity commands (SHOW/DESCRIBE/CREATE/DROP ENTITY)
package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// execCreateEntity handles CREATE ENTITY statements.
func (e *Executor) execCreateEntity(s *ast.CreateEntityStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	// Find module
	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	// Get domain model
	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	// Check if entity already exists
	var existingEntity *domainmodel.Entity
	for _, entity := range dm.Entities {
		if entity.Name == s.Name.Name {
			existingEntity = entity
			break
		}
	}

	// If entity exists and not using CREATE OR MODIFY, return error
	if existingEntity != nil && !s.CreateOrModify {
		return fmt.Errorf("entity already exists: %s.%s (use CREATE OR MODIFY to update)", s.Name.Module, s.Name.Name)
	}

	// Calculate position
	var location model.Point
	if s.Position != nil {
		location = model.Point{X: s.Position.X, Y: s.Position.Y}
	} else if existingEntity != nil {
		location = existingEntity.Location
	} else {
		// Auto-position based on existing entities
		location = model.Point{X: 100 + len(dm.Entities)*150, Y: 100}
	}

	// Determine persistable based on entity kind
	persistable := s.Kind != ast.EntityNonPersistent

	// Auto-default Boolean attributes to false if no DEFAULT specified
	for i := range s.Attributes {
		if s.Attributes[i].Type.Kind == ast.TypeBoolean && !s.Attributes[i].HasDefault {
			s.Attributes[i].HasDefault = true
			s.Attributes[i].DefaultValue = false
		}
	}

	// Create attributes and build name-to-ID map for validation rules and indexes
	var attrs []*domainmodel.Attribute
	attrNameToID := make(map[string]model.ID)
	for _, a := range s.Attributes {
		// Use Documentation if available, fall back to Comment
		doc := a.Documentation
		if doc == "" {
			doc = a.Comment
		}

		// Generate ID for the attribute so we can reference it in validation rules/indexes
		attrID := model.ID(mpr.GenerateID())
		attrNameToID[a.Name] = attrID

		attr := &domainmodel.Attribute{
			Name:          a.Name,
			Documentation: doc,
			Type:          convertDataType(a.Type),
		}
		attr.ID = attrID

		// Default value
		if a.HasDefault {
			defaultStr := fmt.Sprintf("%v", a.DefaultValue)
			// For enum attributes, Mendix stores just the value name (e.g., "Open"),
			// not the fully qualified name. The EnumerationRef already provides context.
			// Strip the enum prefix if the default is fully qualified.
			if a.Type.Kind == ast.TypeEnumeration && a.Type.EnumRef != nil {
				enumPrefix := a.Type.EnumRef.String() + "."
				if strings.HasPrefix(defaultStr, enumPrefix) {
					defaultStr = strings.TrimPrefix(defaultStr, enumPrefix)
				}
			}
			attr.Value = &domainmodel.AttributeValue{
				DefaultValue: defaultStr,
			}
		}

		attrs = append(attrs, attr)
	}

	// Create validation rules for NOT NULL and UNIQUE constraints
	var validationRules []*domainmodel.ValidationRule
	for _, a := range s.Attributes {
		attrID := attrNameToID[a.Name]

		// NOT NULL -> Required validation rule
		if a.NotNull {
			vr := &domainmodel.ValidationRule{
				AttributeID: attrID,
				Type:        "Required",
			}
			vr.ID = model.ID(mpr.GenerateID())
			if a.NotNullError != "" {
				vr.ErrorMessage = &model.Text{
					Translations: map[string]string{"en_US": a.NotNullError},
				}
				vr.ErrorMessage.ID = model.ID(mpr.GenerateID())
			}
			validationRules = append(validationRules, vr)
		}

		// UNIQUE -> Unique validation rule
		if a.Unique {
			vr := &domainmodel.ValidationRule{
				AttributeID: attrID,
				Type:        "Unique",
			}
			vr.ID = model.ID(mpr.GenerateID())
			if a.UniqueError != "" {
				vr.ErrorMessage = &model.Text{
					Translations: map[string]string{"en_US": a.UniqueError},
				}
				vr.ErrorMessage.ID = model.ID(mpr.GenerateID())
			}
			validationRules = append(validationRules, vr)
		}
	}

	// Create indexes
	var indexes []*domainmodel.Index
	for _, idx := range s.Indexes {
		idxID := model.ID(mpr.GenerateID())
		var indexAttrs []*domainmodel.IndexAttribute
		for _, col := range idx.Columns {
			if attrID, ok := attrNameToID[col.Name]; ok {
				iaID := model.ID(mpr.GenerateID())
				ia := &domainmodel.IndexAttribute{
					AttributeID: attrID,
					Ascending:   !col.Descending,
				}
				ia.ID = iaID
				indexAttrs = append(indexAttrs, ia)
			}
		}
		if len(indexAttrs) > 0 {
			index := &domainmodel.Index{
				Attributes: indexAttrs,
			}
			index.ID = idxID
			indexes = append(indexes, index)
		}
	}

	// Create entity
	entity := &domainmodel.Entity{
		Name:            s.Name.Name,
		Documentation:   s.Documentation,
		Location:        location,
		Persistable:     persistable,
		Attributes:      attrs,
		ValidationRules: validationRules,
		Indexes:         indexes,
	}

	// Set generalization (inheritance) if specified
	if s.Generalization != nil {
		entity.GeneralizationRef = s.Generalization.String()
	}

	if s.CreateOrModify && existingEntity != nil {
		// Update existing entity
		entity.ID = existingEntity.ID
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to update entity: %w", err)
		}
		// Invalidate caches so updated entity is visible
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Modified entity: %s\n", s.Name)
	} else {
		// Create new entity
		if err := e.writer.CreateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to create entity: %w", err)
		}
		// Invalidate caches so new entity is visible
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Created entity: %s\n", s.Name)
	}

	e.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

// execCreateViewEntity handles CREATE VIEW ENTITY statements.
func (e *Executor) execCreateViewEntity(s *ast.CreateViewEntityStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	// Validate OQL syntax before creating the view entity
	// This prevents creating view entities that will crash Studio Pro
	if s.Query.RawQuery != "" {
		if oqlErrors := ValidateOQLSyntax(s.Query.RawQuery); len(oqlErrors) > 0 {
			return fmt.Errorf("invalid OQL in view entity '%s':\n  - %s",
				s.Name.String(), strings.Join(oqlErrors, "\n  - "))
		}
	}

	// Find module
	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	// Get domain model
	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	// Check if entity already exists
	var existingEntity *domainmodel.Entity
	for _, entity := range dm.Entities {
		if entity.Name == s.Name.Name {
			existingEntity = entity
			break
		}
	}

	// If entity exists: REPLACE drops and recreates, MODIFY updates in place
	if existingEntity != nil && !s.CreateOrModify && !s.CreateOrReplace {
		return fmt.Errorf("entity already exists: %s.%s (use CREATE OR MODIFY to update, or CREATE OR REPLACE to drop and recreate)", s.Name.Module, s.Name.Name)
	}

	// CREATE OR REPLACE: delete existing entity and source doc, then recreate
	if existingEntity != nil && s.CreateOrReplace {
		// Preserve location from old entity if no explicit position
		if s.Position == nil {
			s.Position = &ast.Position{X: existingEntity.Location.X, Y: existingEntity.Location.Y}
		}
		// Delete ViewEntitySourceDocument
		if err := e.writer.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
			return fmt.Errorf("failed to delete existing ViewEntitySourceDocument: %w", err)
		}
		// Delete the entity itself
		if err := e.writer.DeleteEntity(dm.ID, existingEntity.ID); err != nil {
			return fmt.Errorf("failed to delete existing entity for replace: %w", err)
		}
		existingEntity = nil
		// Re-fetch domain model after deletion so entity count is correct for positioning
		dm, err = e.reader.GetDomainModel(module.ID)
		if err != nil {
			return fmt.Errorf("failed to get domain model after delete: %w", err)
		}
	}

	// Calculate position
	var location model.Point
	if s.Position != nil {
		location = model.Point{X: s.Position.X, Y: s.Position.Y}
	} else if existingEntity != nil {
		location = existingEntity.Location
	} else {
		location = model.Point{X: 100 + len(dm.Entities)*150, Y: 100}
	}

	// Create or update ViewEntitySourceDocument (separate document for OQL query)
	sourceDocRef := s.Name.Module + "." + s.Name.Name
	// Always delete any existing ViewEntitySourceDocument before creating a new one.
	// This prevents duplicate OQL documents from accumulating (e.g., from re-running
	// scripts or after a previous DROP that didn't clean up properly).
	if err := e.writer.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
		return fmt.Errorf("failed to delete existing ViewEntitySourceDocument: %w", err)
	}
	_, err = e.writer.CreateViewEntitySourceDocument(
		module.ID,
		s.Name.Module,
		s.Name.Name,
		s.Query.RawQuery,
		s.Documentation,
	)
	if err != nil {
		return fmt.Errorf("failed to create ViewEntitySourceDocument: %w", err)
	}

	// Create view attributes with OqlViewValue references
	// Note: Studio Pro may show "out of sync" errors if the attribute types don't match
	// what it infers from the OQL. Users can sync the view entity in Studio Pro to fix this.
	var attrs []*domainmodel.Attribute
	for _, a := range s.Attributes {
		attr := &domainmodel.Attribute{
			Name: a.Name,
			Type: convertDataType(a.Type),
			Value: &domainmodel.AttributeValue{
				ViewReference: a.Name, // OQL column alias matches attribute name
			},
		}
		attrs = append(attrs, attr)
	}

	// Create view entity with source document reference.
	// View entities use Persistable=true because they are retrievable from database (via OQL).
	// Studio Pro treats view entities as persistable for database retrieval purposes.
	entity := &domainmodel.Entity{
		Name:              s.Name.Name,
		Documentation:     s.Documentation,
		Location:          location,
		Persistable:       true,
		Attributes:        attrs,
		Source:            "DomainModels$OqlViewEntitySource",
		SourceDocumentRef: sourceDocRef,
	}

	if s.CreateOrModify && existingEntity != nil {
		// Update existing entity
		entity.ID = existingEntity.ID
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to update view entity: %w", err)
		}
		// Invalidate caches so updated entity is visible
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Modified view entity: %s\n", s.Name)
	} else {
		// Create new entity
		if err := e.writer.CreateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to create view entity: %w", err)
		}
		// Invalidate caches so new entity is visible
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Created view entity: %s\n", s.Name)
	}

	return nil
}

// execAlterEntity handles ALTER ENTITY statements.
func (e *Executor) execAlterEntity(s *ast.AlterEntityStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	// Find module
	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	// Get domain model
	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	// Find entity
	var entity *domainmodel.Entity
	for _, ent := range dm.Entities {
		if ent.Name == s.Name.Name {
			entity = ent
			break
		}
	}
	if entity == nil {
		return fmt.Errorf("entity not found: %s", s.Name)
	}

	switch s.Operation {
	case ast.AlterEntityAddAttribute:
		a := s.Attribute
		if a == nil {
			return fmt.Errorf("no attribute definition provided")
		}
		// Auto-default Boolean attributes to false if no DEFAULT specified
		if a.Type.Kind == ast.TypeBoolean && !a.HasDefault {
			a.HasDefault = true
			a.DefaultValue = false
		}
		// Check for duplicate attribute name
		for _, existing := range entity.Attributes {
			if existing.Name == a.Name {
				return fmt.Errorf("attribute '%s' already exists on entity %s", a.Name, s.Name)
			}
		}

		attrID := model.ID(mpr.GenerateID())
		attr := &domainmodel.Attribute{
			Name:          a.Name,
			Documentation: a.Documentation,
			Type:          convertDataType(a.Type),
		}
		attr.ID = attrID
		if a.HasDefault {
			defaultStr := fmt.Sprintf("%v", a.DefaultValue)
			if a.Type.Kind == ast.TypeEnumeration && a.Type.EnumRef != nil {
				enumPrefix := a.Type.EnumRef.String() + "."
				if strings.HasPrefix(defaultStr, enumPrefix) {
					defaultStr = strings.TrimPrefix(defaultStr, enumPrefix)
				}
			}
			attr.Value = &domainmodel.AttributeValue{
				DefaultValue: defaultStr,
			}
		}
		entity.Attributes = append(entity.Attributes, attr)

		// Add validation rules for NOT NULL and UNIQUE
		if a.NotNull {
			vr := &domainmodel.ValidationRule{
				AttributeID: attrID,
				Type:        "Required",
			}
			vr.ID = model.ID(mpr.GenerateID())
			if a.NotNullError != "" {
				vr.ErrorMessage = &model.Text{
					Translations: map[string]string{"en_US": a.NotNullError},
				}
				vr.ErrorMessage.ID = model.ID(mpr.GenerateID())
			}
			entity.ValidationRules = append(entity.ValidationRules, vr)
		}
		if a.Unique {
			vr := &domainmodel.ValidationRule{
				AttributeID: attrID,
				Type:        "Unique",
			}
			vr.ID = model.ID(mpr.GenerateID())
			if a.UniqueError != "" {
				vr.ErrorMessage = &model.Text{
					Translations: map[string]string{"en_US": a.UniqueError},
				}
				vr.ErrorMessage.ID = model.ID(mpr.GenerateID())
			}
			entity.ValidationRules = append(entity.ValidationRules, vr)
		}

		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to add attribute: %w", err)
		}
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Added attribute '%s' to entity %s\n", a.Name, s.Name)

	case ast.AlterEntityRenameAttribute:
		found := false
		for _, attr := range entity.Attributes {
			if attr.Name == s.AttributeName {
				attr.Name = s.NewName
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("attribute '%s' not found on entity %s", s.AttributeName, s.Name)
		}
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to rename attribute: %w", err)
		}
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Renamed attribute '%s' to '%s' on entity %s\n", s.AttributeName, s.NewName, s.Name)

	case ast.AlterEntityModifyAttribute:
		found := false
		for _, attr := range entity.Attributes {
			if attr.Name == s.AttributeName {
				attr.Type = convertDataType(s.DataType)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("attribute '%s' not found on entity %s", s.AttributeName, s.Name)
		}
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to modify attribute: %w", err)
		}
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Modified attribute '%s' on entity %s\n", s.AttributeName, s.Name)

	case ast.AlterEntityDropAttribute:
		idx := -1
		for i, attr := range entity.Attributes {
			if attr.Name == s.AttributeName {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("attribute '%s' not found on entity %s", s.AttributeName, s.Name)
		}
		// Clean up entity-level references to the dropped attribute
		droppedID := entity.Attributes[idx].ID

		// Track what gets cleaned up for reporting
		origValidationCount := len(entity.ValidationRules)
		origIndexCount := len(entity.Indexes)

		// Remove validation rules that reference this attribute
		var keepRules []*domainmodel.ValidationRule
		for _, vr := range entity.ValidationRules {
			if vr.AttributeID != droppedID {
				keepRules = append(keepRules, vr)
			}
		}
		entity.ValidationRules = keepRules

		// Remove MemberAccess entries from access rules that reference this attribute
		removedMemberAccess := 0
		for _, rule := range entity.AccessRules {
			var keepMembers []*domainmodel.MemberAccess
			for _, ma := range rule.MemberAccesses {
				if ma.AttributeID != droppedID {
					keepMembers = append(keepMembers, ma)
				} else {
					removedMemberAccess++
				}
			}
			rule.MemberAccesses = keepMembers
		}

		// Remove index attributes that reference this attribute, and drop empty indexes
		var keepIndexes []*domainmodel.Index
		for _, idx := range entity.Indexes {
			var keepAttrs []*domainmodel.IndexAttribute
			for _, ia := range idx.Attributes {
				if ia.AttributeID != droppedID {
					keepAttrs = append(keepAttrs, ia)
				}
			}
			idx.Attributes = keepAttrs

			var keepIDs []model.ID
			for _, id := range idx.AttributeIDs {
				if id != droppedID {
					keepIDs = append(keepIDs, id)
				}
			}
			idx.AttributeIDs = keepIDs

			if len(idx.Attributes) > 0 || len(idx.AttributeIDs) > 0 {
				keepIndexes = append(keepIndexes, idx)
			}
		}
		entity.Indexes = keepIndexes

		// Remove the attribute
		entity.Attributes = append(entity.Attributes[:idx], entity.Attributes[idx+1:]...)
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to drop attribute: %w", err)
		}
		e.invalidateHierarchy()
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Dropped attribute '%s' from entity %s\n", s.AttributeName, s.Name)

		// Report what was cleaned up on the entity itself
		if n := origValidationCount - len(keepRules); n > 0 {
			fmt.Fprintf(e.output, "  Removed %d validation rule(s)\n", n)
		}
		if removedMemberAccess > 0 {
			fmt.Fprintf(e.output, "  Removed %d access rule member reference(s)\n", removedMemberAccess)
		}
		if n := origIndexCount - len(keepIndexes); n > 0 {
			fmt.Fprintf(e.output, "  Removed %d index(es)\n", n)
		}

		// Warn about references in other documents that are NOT auto-cleaned
		entityQName := s.Name.String()
		fmt.Fprintf(e.output, "  Warning: pages, microflows, and other documents may still reference '%s'. Update them manually.\n", s.AttributeName)
		fmt.Fprintf(e.output, "  Use SHOW REFERENCES TO %s to find usages (requires REFRESH CATALOG FULL).\n", entityQName)

	case ast.AlterEntitySetDocumentation:
		entity.Documentation = s.Documentation
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to set documentation: %w", err)
		}
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Set documentation on entity %s\n", s.Name)

	case ast.AlterEntitySetComment:
		// Comments are stored as documentation in the Mendix model
		entity.Documentation = s.Comment
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to set comment: %w", err)
		}
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Set comment on entity %s\n", s.Name)

	case ast.AlterEntityAddIndex:
		if s.Index == nil {
			return fmt.Errorf("no index definition provided")
		}
		// Build name-to-ID map for attribute lookup
		attrNameToID := make(map[string]model.ID)
		for _, attr := range entity.Attributes {
			attrNameToID[attr.Name] = attr.ID
		}
		idxID := model.ID(mpr.GenerateID())
		var indexAttrs []*domainmodel.IndexAttribute
		for _, col := range s.Index.Columns {
			if attrID, ok := attrNameToID[col.Name]; ok {
				ia := &domainmodel.IndexAttribute{
					AttributeID: attrID,
					Ascending:   !col.Descending,
				}
				ia.ID = model.ID(mpr.GenerateID())
				indexAttrs = append(indexAttrs, ia)
			} else {
				return fmt.Errorf("attribute '%s' not found for index on entity %s", col.Name, s.Name)
			}
		}
		index := &domainmodel.Index{
			Attributes: indexAttrs,
		}
		index.ID = idxID
		entity.Indexes = append(entity.Indexes, index)
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to add index: %w", err)
		}
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Added index to entity %s\n", s.Name)

	case ast.AlterEntityDropIndex:
		// Find and remove the index by position (Mendix indexes don't have user-visible names)
		if len(entity.Indexes) == 0 {
			return fmt.Errorf("no indexes on entity %s", s.Name)
		}
		// For now, drop by ordinal name ("idx1", "idx2", etc.) or drop all
		idx := -1
		for i := range entity.Indexes {
			indexName := fmt.Sprintf("idx%d", i+1)
			if indexName == s.IndexName {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("index '%s' not found on entity %s", s.IndexName, s.Name)
		}
		entity.Indexes = append(entity.Indexes[:idx], entity.Indexes[idx+1:]...)
		if err := e.writer.UpdateEntity(dm.ID, entity); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}
		e.invalidateDomainModelsCache()
		fmt.Fprintf(e.output, "Dropped index '%s' from entity %s\n", s.IndexName, s.Name)

	default:
		return fmt.Errorf("unsupported ALTER ENTITY operation")
	}

	e.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

// execDropEntity handles DROP ENTITY statements.
func (e *Executor) execDropEntity(s *ast.DropEntityStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	// Find module and entity
	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	for _, entity := range dm.Entities {
		if entity.Name == s.Name.Name {
			// Warn about references before deleting (best-effort)
			e.warnEntityReferences(s.Name.String())

			// If this is a view entity, also delete the associated ViewEntitySourceDocument
			if entity.Source == "DomainModels$OqlViewEntitySource" {
				if err := e.writer.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
					return fmt.Errorf("failed to delete view entity source document: %w", err)
				}
			}
			if err := e.writer.DeleteEntity(dm.ID, entity.ID); err != nil {
				return fmt.Errorf("failed to delete entity: %w", err)
			}
			e.invalidateDomainModelsCache()
			fmt.Fprintf(e.output, "Dropped entity: %s\n", s.Name)
			return nil
		}
	}

	return fmt.Errorf("entity not found: %s", s.Name)
}

// warnEntityReferences prints a warning if the entity is referenced by other elements.
// Uses the catalog if available; silently skips if catalog is not built.
func (e *Executor) warnEntityReferences(entityName string) {
	if e.catalog == nil || !e.catalog.IsBuilt() {
		return
	}

	query := fmt.Sprintf(
		"SELECT SourceType, SourceName, RefKind FROM refs WHERE TargetName = '%s'",
		strings.ReplaceAll(entityName, "'", "''"),
	)
	result, err := e.catalog.Query(query)
	if err != nil || result.Count == 0 {
		return
	}

	fmt.Fprintf(e.output, "WARNING: %s is referenced by %d element(s):\n", entityName, result.Count)
	for _, row := range result.Rows {
		sourceType, _ := row[0].(string)
		sourceName, _ := row[1].(string)
		refKind, _ := row[2].(string)
		fmt.Fprintf(e.output, "  - %s %s (%s)\n", sourceType, sourceName, refKind)
	}
}

// showEntities handles SHOW ENTITIES command.
func (e *Executor) showEntities(moduleName string) error {
	// Build module ID -> name map (single query)
	modules, err := e.reader.ListModules()
	if err != nil {
		return fmt.Errorf("failed to list modules: %w", err)
	}
	moduleNames := make(map[model.ID]string)
	for _, m := range modules {
		moduleNames[m.ID] = m.Name
	}

	// Get all domain models in a single query (avoids O(n²) behavior)
	domainModels, err := e.reader.ListDomainModels()
	if err != nil {
		return fmt.Errorf("failed to list domain models: %w", err)
	}

	// Build entity ID -> association count map
	assocCounts := make(map[model.ID]int)
	for _, dm := range domainModels {
		for _, assoc := range dm.Associations {
			assocCounts[assoc.ParentID]++
			assocCounts[assoc.ChildID]++
		}
	}

	// Collect System entities referenced via generalizations
	systemEntities := make(map[string]bool)
	for _, dm := range domainModels {
		for _, entity := range dm.Entities {
			if entity.GeneralizationRef != "" && strings.HasPrefix(entity.GeneralizationRef, "System.") {
				systemEntities[entity.GeneralizationRef] = true
			}
		}
	}

	// Collect rows and calculate column widths
	type row struct {
		qualifiedName  string
		entityType     string
		generalization string
		attrs          int
		assocs         int
		validations    int
		indexes        int
		events         int
		accessRules    int
	}
	var rows []row
	qnWidth := len("Entity")
	typeWidth := len("Type")
	genWidth := len("Extends")

	// Add System entities first (if showing all or System module)
	if moduleName == "" || moduleName == "System" {
		for sysEntity := range systemEntities {
			r := row{
				qualifiedName: sysEntity,
				entityType:    "System",
				attrs:         -1, // Unknown - from runtime
				assocs:        -1,
				validations:   -1,
				indexes:       -1,
				events:        -1,
				accessRules:   -1,
			}
			rows = append(rows, r)
			if len(sysEntity) > qnWidth {
				qnWidth = len(sysEntity)
			}
		}
	}

	for _, dm := range domainModels {
		modName := moduleNames[dm.ContainerID]
		// Filter by module name if specified
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, entity := range dm.Entities {
			// Determine entity type based on Source field and Persistable flag
			entityType := "Persistent"
			if strings.Contains(entity.Source, "OqlView") {
				entityType = "View"
			} else if strings.Contains(entity.Source, "OData") || entity.RemoteSource != "" || entity.RemoteSourceDocument != "" {
				entityType = "External"
			} else if !entity.Persistable {
				entityType = "Non-Persistent"
			}

			qualifiedName := modName + "." + entity.Name
			r := row{
				qualifiedName:  qualifiedName,
				entityType:     entityType,
				generalization: entity.GeneralizationRef,
				attrs:          len(entity.Attributes),
				assocs:         assocCounts[entity.ID],
				validations:    len(entity.ValidationRules),
				indexes:        len(entity.Indexes),
				events:         len(entity.EventHandlers),
				accessRules:    len(entity.AccessRules),
			}
			rows = append(rows, r)

			if len(qualifiedName) > qnWidth {
				qnWidth = len(qualifiedName)
			}
			if len(entityType) > typeWidth {
				typeWidth = len(entityType)
			}
			if len(entity.GeneralizationRef) > genWidth {
				genWidth = len(entity.GeneralizationRef)
			}
		}
	}

	// Check if any entity has a generalization — only show column if needed
	hasGeneralizations := false
	for _, r := range rows {
		if r.generalization != "" {
			hasGeneralizations = true
			break
		}
	}

	// Sort by qualified name
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	// Markdown table with aligned columns
	if hasGeneralizations {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %5s | %6s | %11s | %7s | %6s | %11s |\n",
			qnWidth, "Entity", typeWidth, "Type", genWidth, "Extends", "Attrs", "Assocs", "Validations", "Indexes", "Events", "AccessRules")
		fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-------|--------|-------------|---------|--------|-------------|\n",
			strings.Repeat("-", qnWidth), strings.Repeat("-", typeWidth), strings.Repeat("-", genWidth))
	} else {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %5s | %6s | %11s | %7s | %6s | %11s |\n",
			qnWidth, "Entity", typeWidth, "Type", "Attrs", "Assocs", "Validations", "Indexes", "Events", "AccessRules")
		fmt.Fprintf(e.output, "|-%s-|-%s-|-------|--------|-------------|---------|--------|-------------|\n",
			strings.Repeat("-", qnWidth), strings.Repeat("-", typeWidth))
	}
	for _, r := range rows {
		genStr := r.generalization
		// For System entities, show "-" instead of -1
		if r.entityType == "System" {
			if hasGeneralizations {
				fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %5s | %6s | %11s | %7s | %6s | %11s |\n",
					qnWidth, r.qualifiedName, typeWidth, r.entityType, genWidth, genStr, "-", "-", "-", "-", "-", "-")
			} else {
				fmt.Fprintf(e.output, "| %-*s | %-*s | %5s | %6s | %11s | %7s | %6s | %11s |\n",
					qnWidth, r.qualifiedName, typeWidth, r.entityType, "-", "-", "-", "-", "-", "-")
			}
		} else if hasGeneralizations {
			fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %5d | %6d | %11d | %7d | %6d | %11d |\n",
				qnWidth, r.qualifiedName, typeWidth, r.entityType, genWidth, genStr, r.attrs, r.assocs, r.validations, r.indexes, r.events, r.accessRules)
		} else {
			fmt.Fprintf(e.output, "| %-*s | %-*s | %5d | %6d | %11d | %7d | %6d | %11d |\n",
				qnWidth, r.qualifiedName, typeWidth, r.entityType, r.attrs, r.assocs, r.validations, r.indexes, r.events, r.accessRules)
		}
	}
	fmt.Fprintf(e.output, "\n(%d entities)\n", len(rows))
	return nil
}

// showEntity handles SHOW ENTITY command.
func (e *Executor) showEntity(name *ast.QualifiedName) error {
	if name == nil {
		return fmt.Errorf("entity name required")
	}

	module, err := e.findModule(name.Module)
	if err != nil {
		return err
	}

	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	for _, entity := range dm.Entities {
		if entity.Name == name.Name {
			fmt.Fprintf(e.output, "**Entity: %s.%s**\n\n", module.Name, entity.Name)
			fmt.Fprintf(e.output, "- Persistable: %v\n", entity.Persistable)
			if entity.GeneralizationRef != "" {
				fmt.Fprintf(e.output, "- Extends: %s\n", entity.GeneralizationRef)
			}
			fmt.Fprintf(e.output, "- Location: (%d, %d)\n\n", entity.Location.X, entity.Location.Y)

			if len(entity.Attributes) > 0 {
				// Calculate column widths
				nameWidth, typeWidth := len("Attribute"), len("Type")
				type attrRow struct {
					name, typeName string
				}
				var rows []attrRow
				for _, attr := range entity.Attributes {
					typeName := getAttributeTypeName(attr.Type)
					rows = append(rows, attrRow{attr.Name, typeName})
					if len(attr.Name) > nameWidth {
						nameWidth = len(attr.Name)
					}
					if len(typeName) > typeWidth {
						typeWidth = len(typeName)
					}
				}

				fmt.Fprintf(e.output, "| %-*s | %-*s |\n", nameWidth, "Attribute", typeWidth, "Type")
				fmt.Fprintf(e.output, "|-%s-|-%s-|\n", strings.Repeat("-", nameWidth), strings.Repeat("-", typeWidth))
				for _, r := range rows {
					fmt.Fprintf(e.output, "| %-*s | %-*s |\n", nameWidth, r.name, typeWidth, r.typeName)
				}
				fmt.Fprintf(e.output, "\n(%d attributes)\n", len(entity.Attributes))
			}
			return nil
		}
	}

	return fmt.Errorf("entity not found: %s", name)
}

// describeEntity handles DESCRIBE ENTITY command.
func (e *Executor) describeEntity(name ast.QualifiedName) error {
	module, err := e.findModule(name.Module)
	if err != nil {
		return err
	}

	dm, err := e.reader.GetDomainModel(module.ID)
	if err != nil {
		return fmt.Errorf("failed to get domain model: %w", err)
	}

	for _, entity := range dm.Entities {
		if entity.Name == name.Name {
			// Output JavaDoc documentation if present
			if entity.Documentation != "" {
				fmt.Fprintf(e.output, "/**\n * %s\n */\n", entity.Documentation)
			}

			// Output position annotation
			fmt.Fprintf(e.output, "@Position(%d, %d)\n", entity.Location.X, entity.Location.Y)

			// Determine entity type based on Source field and Persistable flag
			entityType := "PERSISTENT"
			if strings.Contains(entity.Source, "OqlView") {
				entityType = "VIEW"
			} else if strings.Contains(entity.Source, "OData") || entity.RemoteSource != "" || entity.RemoteSourceDocument != "" {
				entityType = "EXTERNAL"
			} else if !entity.Persistable {
				entityType = "NON-PERSISTENT"
			}

			if entity.GeneralizationRef != "" {
				fmt.Fprintf(e.output, "CREATE %s ENTITY %s.%s EXTENDS %s (\n", entityType, module.Name, entity.Name, entity.GeneralizationRef)
			} else {
				fmt.Fprintf(e.output, "CREATE %s ENTITY %s.%s (\n", entityType, module.Name, entity.Name)
			}

			// Build validation rules map by attribute ID and name
			// The AttributeID can be a UUID or a qualified name like "DmTest.Cars.CarId"
			validationsByAttr := make(map[model.ID][]*domainmodel.ValidationRule)
			validationsByName := make(map[string][]*domainmodel.ValidationRule)
			for _, vr := range entity.ValidationRules {
				validationsByAttr[vr.AttributeID] = append(validationsByAttr[vr.AttributeID], vr)
				// Also index by attribute name extracted from qualified name
				attrName := extractAttrNameFromQualified(string(vr.AttributeID))
				if attrName != "" {
					validationsByName[attrName] = append(validationsByName[attrName], vr)
				}
			}

			// Output attributes
			for i, attr := range entity.Attributes {
				// Attribute documentation
				if attr.Documentation != "" {
					fmt.Fprintf(e.output, "  /** %s */\n", attr.Documentation)
				}

				typeStr := formatAttributeType(attr.Type)
				var constraints strings.Builder

				// Check for validation rules - try by ID first, then by name
				attrValidations := validationsByAttr[attr.ID]
				if len(attrValidations) == 0 {
					attrValidations = validationsByName[attr.Name]
				}
				for _, vr := range attrValidations {
					if vr.Type == "Required" {
						constraints.WriteString(" NOT NULL")
						if vr.ErrorMessage != nil {
							errMsg := vr.ErrorMessage.GetTranslation("en_US")
							if errMsg != "" {
								constraints.WriteString(fmt.Sprintf(" ERROR '%s'", errMsg))
							}
						}
					}
					if vr.Type == "Unique" {
						constraints.WriteString(" UNIQUE")
						if vr.ErrorMessage != nil {
							errMsg := vr.ErrorMessage.GetTranslation("en_US")
							if errMsg != "" {
								constraints.WriteString(fmt.Sprintf(" ERROR '%s'", errMsg))
							}
						}
					}
				}

				// Default value
				if attr.Value != nil && attr.Value.DefaultValue != "" {
					defaultVal := attr.Value.DefaultValue
					// Quote string defaults
					if _, ok := attr.Type.(*domainmodel.StringAttributeType); ok {
						defaultVal = fmt.Sprintf("'%s'", defaultVal)
					}
					// Re-qualify enum defaults for MDL syntax (BSON stores just the value name)
					if enumType, ok := attr.Type.(*domainmodel.EnumerationAttributeType); ok {
						if enumType.EnumerationRef != "" && !strings.Contains(defaultVal, ".") {
							defaultVal = enumType.EnumerationRef + "." + defaultVal
						}
					}
					constraints.WriteString(fmt.Sprintf(" DEFAULT %s", defaultVal))
				}

				comma := ","
				if i == len(entity.Attributes)-1 {
					comma = ""
				}
				fmt.Fprintf(e.output, "  %s: %s%s%s\n", attr.Name, typeStr, constraints.String(), comma)
			}
			fmt.Fprint(e.output, ")")

			// For VIEW entities, output the OQL query
			if entityType == "VIEW" && entity.OqlQuery != "" {
				fmt.Fprint(e.output, " AS (\n")
				// Indent OQL query lines
				oqlLines := strings.SplitSeq(entity.OqlQuery, "\n")
				for line := range oqlLines {
					fmt.Fprintf(e.output, "  %s\n", line)
				}
				fmt.Fprint(e.output, ")")
			}

			// Build attribute name map
			attrNames := make(map[model.ID]string)
			for _, attr := range entity.Attributes {
				attrNames[attr.ID] = attr.Name
			}

			// Output indexes
			for _, idx := range entity.Indexes {
				var cols []string
				for _, ia := range idx.Attributes {
					colName := attrNames[ia.AttributeID]
					if !ia.Ascending {
						colName += " DESC"
					}
					cols = append(cols, colName)
				}
				if len(cols) > 0 {
					fmt.Fprintf(e.output, "\nINDEX (%s)", strings.Join(cols, ", "))
				}
			}

			fmt.Fprintln(e.output, ";")

			// Output access rule GRANT statements
			e.outputEntityAccessGrants(entity, name.Module, name.Name)

			fmt.Fprintln(e.output, "/")
			return nil
		}
	}

	return fmt.Errorf("entity not found: %s", name)
}

// describeEntityToString generates MDL source for an entity and returns it as a string.
func (e *Executor) describeEntityToString(name ast.QualifiedName) (string, error) {
	var buf strings.Builder
	origOutput := e.output
	e.output = &buf
	err := e.describeEntity(name)
	e.output = origOutput
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// outputEntityAccessGrants outputs GRANT statements for entity access rules.
func (e *Executor) outputEntityAccessGrants(entity *domainmodel.Entity, moduleName, entityName string) {
	if len(entity.AccessRules) == 0 {
		return
	}

	// Build attribute name map for resolving member accesses
	attrNames := make(map[string]string)
	for _, attr := range entity.Attributes {
		attrNames[string(attr.ID)] = attr.Name
	}

	for _, rule := range entity.AccessRules {
		// Get role names
		var roleStrs []string
		for _, rn := range rule.ModuleRoleNames {
			roleStrs = append(roleStrs, rn)
		}
		if len(roleStrs) == 0 {
			for _, rid := range rule.ModuleRoles {
				roleStrs = append(roleStrs, string(rid))
			}
		}
		if len(roleStrs) == 0 {
			continue
		}

		// Build rights list
		var rights []string
		if rule.AllowCreate {
			rights = append(rights, "CREATE")
		}
		if rule.AllowDelete {
			rights = append(rights, "DELETE")
		}

		// Determine READ/WRITE access.
		// Mendix has no AllowRead/AllowWrite on AccessRule — infer from
		// DefaultMemberAccessRights and individual MemberAccesses entries.
		hasRead := rule.DefaultMemberAccessRights == domainmodel.MemberAccessRightsReadOnly ||
			rule.DefaultMemberAccessRights == domainmodel.MemberAccessRightsReadWrite
		hasWrite := rule.DefaultMemberAccessRights == domainmodel.MemberAccessRightsReadWrite
		if !hasRead || !hasWrite {
			for _, ma := range rule.MemberAccesses {
				if ma.AccessRights == domainmodel.MemberAccessRightsReadOnly ||
					ma.AccessRights == domainmodel.MemberAccessRightsReadWrite {
					hasRead = true
				}
				if ma.AccessRights == domainmodel.MemberAccessRightsReadWrite {
					hasWrite = true
				}
			}
		}

		readMembers, writeMembers := e.resolveEntityMemberAccess(rule, attrNames)

		if hasRead {
			if readMembers == nil {
				rights = append(rights, "READ *")
			} else {
				rights = append(rights, fmt.Sprintf("READ (%s)", strings.Join(readMembers, ", ")))
			}
		}
		if hasWrite {
			if writeMembers == nil {
				rights = append(rights, "WRITE *")
			} else if len(writeMembers) > 0 {
				rights = append(rights, fmt.Sprintf("WRITE (%s)", strings.Join(writeMembers, ", ")))
			}
		}

		if len(rights) == 0 {
			continue
		}

		grantLine := fmt.Sprintf("\nGRANT %s ON %s.%s (%s)",
			strings.Join(roleStrs, ", "), moduleName, entityName, strings.Join(rights, ", "))

		if rule.XPathConstraint != "" {
			grantLine += fmt.Sprintf(" WHERE '%s'", rule.XPathConstraint)
		}
		grantLine += ";"

		fmt.Fprintln(e.output, grantLine)
	}
}

// resolveEntityMemberAccess determines per-member READ/WRITE access.
// Returns nil slices for "all members" (*), or specific member name lists.
func (e *Executor) resolveEntityMemberAccess(rule *domainmodel.AccessRule, attrNames map[string]string) (readMembers []string, writeMembers []string) {
	if len(rule.MemberAccesses) == 0 {
		// No per-member overrides: use default
		return nil, nil
	}

	// Check if all member accesses match the default — if so, treat as "*"
	allMatchDefault := true
	for _, ma := range rule.MemberAccesses {
		if ma.AccessRights != rule.DefaultMemberAccessRights {
			allMatchDefault = false
			break
		}
	}
	if allMatchDefault {
		return nil, nil
	}

	// Collect members by access level
	var readOnly, readWrite []string
	for _, ma := range rule.MemberAccesses {
		memberName := ma.AttributeName
		if memberName == "" {
			memberName = ma.AssociationName
		}
		if memberName == "" {
			if an, ok := attrNames[string(ma.AttributeID)]; ok {
				memberName = an
			} else {
				memberName = string(ma.AttributeID)
			}
		}

		switch ma.AccessRights {
		case domainmodel.MemberAccessRightsReadWrite:
			readWrite = append(readWrite, memberName)
		case domainmodel.MemberAccessRightsReadOnly:
			readOnly = append(readOnly, memberName)
		}
	}

	// If there are overrides, list specific members for READ and WRITE
	// READ includes both ReadOnly and ReadWrite members
	allReadable := append(readOnly, readWrite...)
	if len(allReadable) == 0 {
		readMembers = nil // all via default
	} else {
		readMembers = allReadable
	}

	if len(readWrite) == 0 {
		writeMembers = []string{} // no write members
	} else {
		writeMembers = readWrite
	}

	return readMembers, writeMembers
}

// extractAttrNameFromQualified extracts the attribute name from a qualified name.
// e.g., "DmTest.Cars.CarId" -> "CarId"
func extractAttrNameFromQualified(qualifiedName string) string {
	// Split by "." and return the last part
	parts := strings.Split(qualifiedName, ".")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}
