// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateEntity creates a new entity in a domain model.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) CreateEntity(domainModelID model.ID, entity *domainmodel.Entity) error {
	// Load the domain model by its ID
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	// Assign ID if not set
	if entity.ID == "" {
		entity.ID = model.ID(generateUUID())
	}
	entity.TypeName = "DomainModels$Entity"
	entity.ContainerID = domainModelID

	// Assign IDs to attributes if not set
	for _, attr := range entity.Attributes {
		if attr.ID == "" {
			attr.ID = model.ID(generateUUID())
		}
		attr.TypeName = "DomainModels$Attribute"
		attr.ContainerID = entity.ID
	}

	// Add entity to domain model
	dm.Entities = append(dm.Entities, entity)

	// Serialize and update
	return w.updateDomainModel(dm)
}

// UpdateEntity updates an existing entity.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) UpdateEntity(domainModelID model.ID, entity *domainmodel.Entity) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	// Find and replace the entity
	for i, e := range dm.Entities {
		if e.ID == entity.ID {
			dm.Entities[i] = entity
			return w.updateDomainModel(dm)
		}
	}

	return fmt.Errorf("entity not found: %s", entity.ID)
}

// DeleteEntity deletes an entity from a domain model.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) DeleteEntity(domainModelID model.ID, entityID model.ID) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	// Find and remove the entity
	for i, e := range dm.Entities {
		if e.ID == entityID {
			dm.Entities = append(dm.Entities[:i], dm.Entities[i+1:]...)
			return w.updateDomainModel(dm)
		}
	}

	return fmt.Errorf("entity not found: %s", entityID)
}

// MoveEntity moves an entity from one domain model to another.
// Associations referencing the moved entity are converted to CrossAssociations
// (cross-module associations with BY_NAME references to the remote entity).
// Validation rule attribute references are updated to reflect the new module name.
// Returns the names of converted associations (for caller to inform about).
func (w *Writer) MoveEntity(entity *domainmodel.Entity, sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string) ([]string, error) {
	// Load source domain model and remove the entity
	sourceDM, err := w.reader.GetDomainModelByID(sourceDMID)
	if err != nil {
		return nil, fmt.Errorf("failed to load source domain model: %w", err)
	}

	found := false
	for i, e := range sourceDM.Entities {
		if e.ID == entity.ID {
			sourceDM.Entities = append(sourceDM.Entities[:i], sourceDM.Entities[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("entity not found in source domain model: %s", entity.ID)
	}

	// Load target domain model
	targetDM, err := w.reader.GetDomainModelByID(targetDMID)
	if err != nil {
		return nil, fmt.Errorf("failed to load target domain model: %w", err)
	}

	// Convert associations referencing the moved entity to CrossAssociations.
	// - If moved entity is the child: CrossAssoc stays in source DM (parent is local)
	// - If moved entity is the parent: CrossAssoc goes to target DM (parent moves with entity)
	var convertedAssocs []string
	var keptAssocs []*domainmodel.Association
	for _, a := range sourceDM.Associations {
		if a.ChildID == entity.ID {
			// Child is being moved → CrossAssoc stays in source DM
			// ParentPointer = parent entity (stays local), Child = remote qualified name
			ca := &domainmodel.CrossModuleAssociation{}
			ca.ID = a.ID
			ca.TypeName = "DomainModels$CrossAssociation"
			ca.ContainerID = sourceDMID
			ca.Name = a.Name
			ca.Documentation = a.Documentation
			ca.ParentID = a.ParentID
			ca.ChildRef = targetModuleName + "." + entity.Name
			ca.Type = a.Type
			ca.Owner = a.Owner
			ca.StorageFormat = a.StorageFormat
			ca.ParentDeleteBehavior = a.ParentDeleteBehavior
			ca.ChildDeleteBehavior = a.ChildDeleteBehavior
			sourceDM.CrossAssociations = append(sourceDM.CrossAssociations, ca)
			convertedAssocs = append(convertedAssocs, a.Name)
		} else if a.ParentID == entity.ID {
			// Parent is being moved → CrossAssoc goes to target DM
			// ParentPointer = moved entity (will be local in target), Child = remote entity in source
			var childEntityName string
			for _, e := range sourceDM.Entities {
				if e.ID == a.ChildID {
					childEntityName = e.Name
					break
				}
			}
			ca := &domainmodel.CrossModuleAssociation{}
			ca.ID = a.ID
			ca.TypeName = "DomainModels$CrossAssociation"
			ca.ContainerID = targetDMID
			ca.Name = a.Name
			ca.Documentation = a.Documentation
			ca.ParentID = a.ParentID // parent entity ID (same, just moving to target DM)
			ca.ChildRef = sourceModuleName + "." + childEntityName
			ca.Type = a.Type
			ca.Owner = a.Owner
			ca.StorageFormat = a.StorageFormat
			ca.ParentDeleteBehavior = a.ParentDeleteBehavior
			ca.ChildDeleteBehavior = a.ChildDeleteBehavior
			targetDM.CrossAssociations = append(targetDM.CrossAssociations, ca)
			convertedAssocs = append(convertedAssocs, a.Name)
		} else {
			keptAssocs = append(keptAssocs, a)
		}
	}
	sourceDM.Associations = keptAssocs

	// Update validation rule attribute references in the moved entity.
	// These are BY_NAME qualified names like "OldModule.Entity.Attribute" that need
	// to be updated to "NewModule.Entity.Attribute".
	oldPrefix := sourceModuleName + "."
	newPrefix := targetModuleName + "."
	for _, vr := range entity.ValidationRules {
		attrIDStr := string(vr.AttributeID)
		if strings.HasPrefix(attrIDStr, oldPrefix) {
			vr.AttributeID = model.ID(newPrefix + attrIDStr[len(oldPrefix):])
		}
	}

	// Update SourceDocumentRef for view entities
	if entity.Source == "DomainModels$OqlViewEntitySource" && entity.SourceDocumentRef != "" {
		if strings.HasPrefix(entity.SourceDocumentRef, oldPrefix) {
			entity.SourceDocumentRef = newPrefix + entity.SourceDocumentRef[len(oldPrefix):]
		}
	}

	// Save source domain model
	if err := w.updateDomainModel(sourceDM); err != nil {
		return nil, fmt.Errorf("failed to update source domain model: %w", err)
	}

	// Add entity to target domain model and save
	entity.ContainerID = targetDMID
	targetDM.Entities = append(targetDM.Entities, entity)

	if err := w.updateDomainModel(targetDM); err != nil {
		return nil, fmt.Errorf("failed to update target domain model: %w", err)
	}

	return convertedAssocs, nil
}

// UpdateEnumerationRefsInAllDomainModels updates enumeration references across all domain models.
// When an enumeration is moved to a different module, its qualified name changes and all
// EnumerationAttributeType references need to be updated.
func (w *Writer) UpdateEnumerationRefsInAllDomainModels(oldQualifiedName, newQualifiedName string) error {
	dms, err := w.reader.ListDomainModels()
	if err != nil {
		return fmt.Errorf("failed to list domain models: %w", err)
	}

	for _, dm := range dms {
		changed := false
		for _, entity := range dm.Entities {
			for _, attr := range entity.Attributes {
				if enumType, ok := attr.Type.(*domainmodel.EnumerationAttributeType); ok {
					if enumType.EnumerationRef == oldQualifiedName {
						enumType.EnumerationRef = newQualifiedName
						enumType.EnumerationID = model.ID(newQualifiedName)
						changed = true
					}
				}
			}
		}
		if changed {
			if err := w.updateDomainModel(dm); err != nil {
				return fmt.Errorf("failed to update domain model %s: %w", dm.ID, err)
			}
		}
	}
	return nil
}

// MoveViewEntitySourceDocument moves a ViewEntitySourceDocument to a new module.
func (w *Writer) MoveViewEntitySourceDocument(sourceModuleName string, targetModuleID model.ID, docName string) error {
	docID, err := w.FindViewEntitySourceDocumentID(sourceModuleName, docName)
	if err != nil {
		return err
	}
	if docID == "" {
		return nil // No document to move
	}

	// Update ContainerID in database
	return w.moveUnitByID(string(docID), string(targetModuleID))
}

// UpdateOqlQueriesForMovedEntity updates OQL queries in all ViewEntitySourceDocuments
// to reflect a moved entity's new qualified name. For example, when DmTest.Customer moves
// to DmTest2.Customer, all OQL references like "DmTest.Customer" are updated.
func (w *Writer) UpdateOqlQueriesForMovedEntity(oldQualifiedName, newQualifiedName string) (int, error) {
	units, err := w.reader.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		oql, _ := raw["Oql"].(string)
		if oql == "" || !strings.Contains(oql, oldQualifiedName) {
			continue
		}

		// Replace entity references in OQL
		newOql := strings.ReplaceAll(oql, oldQualifiedName, newQualifiedName)
		raw["Oql"] = newOql

		// Re-serialize and update
		contents, err := bson.Marshal(raw)
		if err != nil {
			continue
		}
		if err := w.updateUnit(u.ID, contents); err != nil {
			return updated, fmt.Errorf("failed to update ViewEntitySourceDocument %s: %w", u.ID, err)
		}
		updated++
	}
	return updated, nil
}

// moveUnitByID changes a unit's ContainerID without modifying its contents.
func (w *Writer) moveUnitByID(unitID string, newContainerID string) error {
	unitIDBlob := uuidToBlob(unitID)
	containerIDBlob := uuidToBlob(newContainerID)

	_, err := w.reader.db.Exec(`UPDATE Unit SET ContainerID = ? WHERE UnitID = ?`, containerIDBlob, unitIDBlob)
	if err == nil {
		w.reader.InvalidateCache()
	}
	return err
}

// AddAttribute adds an attribute to an entity.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) AddAttribute(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	// Find the entity
	for _, e := range dm.Entities {
		if e.ID == entityID {
			if attr.ID == "" {
				attr.ID = model.ID(generateUUID())
			}
			attr.TypeName = "DomainModels$Attribute"
			attr.ContainerID = entityID
			e.Attributes = append(e.Attributes, attr)
			return w.updateDomainModel(dm)
		}
	}

	return fmt.Errorf("entity not found: %s", entityID)
}

// UpdateAttribute updates an existing attribute in an entity.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) UpdateAttribute(domainModelID model.ID, entityID model.ID, attr *domainmodel.Attribute) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	// Find the entity
	for _, e := range dm.Entities {
		if e.ID == entityID {
			// Find and update the attribute
			for i, a := range e.Attributes {
				if a.ID == attr.ID {
					e.Attributes[i] = attr
					return w.updateDomainModel(dm)
				}
			}
			return fmt.Errorf("attribute not found: %s", attr.ID)
		}
	}

	return fmt.Errorf("entity not found: %s", entityID)
}

// DeleteAttribute deletes an attribute from an entity.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) DeleteAttribute(domainModelID model.ID, entityID model.ID, attrID model.ID) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	// Find the entity
	for _, e := range dm.Entities {
		if e.ID == entityID {
			// Find and remove the attribute
			for i, a := range e.Attributes {
				if a.ID == attrID {
					e.Attributes = append(e.Attributes[:i], e.Attributes[i+1:]...)
					return w.updateDomainModel(dm)
				}
			}
			return fmt.Errorf("attribute not found: %s", attrID)
		}
	}

	return fmt.Errorf("entity not found: %s", entityID)
}

// CreateAssociation creates a new association between entities.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) CreateAssociation(domainModelID model.ID, assoc *domainmodel.Association) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	if assoc.ID == "" {
		assoc.ID = model.ID(generateUUID())
	}
	assoc.TypeName = "DomainModels$Association"
	assoc.ContainerID = domainModelID

	dm.Associations = append(dm.Associations, assoc)
	return w.updateDomainModel(dm)
}

// CreateCrossAssociation creates a cross-module association in a domain model.
// The parent entity must be local to this domain model; the child entity is
// referenced by qualified name (BY_NAME) since it lives in another module.
func (w *Writer) CreateCrossAssociation(domainModelID model.ID, ca *domainmodel.CrossModuleAssociation) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	if ca.ID == "" {
		ca.ID = model.ID(generateUUID())
	}
	ca.TypeName = "DomainModels$CrossAssociation"
	ca.ContainerID = domainModelID

	dm.CrossAssociations = append(dm.CrossAssociations, ca)
	return w.updateDomainModel(dm)
}

// DeleteAssociation deletes an association.
// domainModelID is the ID of the domain model itself (not the module ID).
func (w *Writer) DeleteAssociation(domainModelID model.ID, assocID model.ID) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	for i, a := range dm.Associations {
		if a.ID == assocID {
			dm.Associations = append(dm.Associations[:i], dm.Associations[i+1:]...)
			return w.updateDomainModel(dm)
		}
	}

	return fmt.Errorf("association not found: %s", assocID)
}

// DeleteCrossAssociation removes a cross-module association from a domain model.
func (w *Writer) DeleteCrossAssociation(domainModelID model.ID, assocID model.ID) error {
	dm, err := w.reader.GetDomainModelByID(domainModelID)
	if err != nil {
		return err
	}

	for i, ca := range dm.CrossAssociations {
		if ca.ID == assocID {
			dm.CrossAssociations = append(dm.CrossAssociations[:i], dm.CrossAssociations[i+1:]...)
			return w.updateDomainModel(dm)
		}
	}

	return fmt.Errorf("cross-module association not found: %s", assocID)
}

// CreateViewEntitySourceDocument creates a ViewEntitySourceDocument for a view entity.
// This is a separate document that contains the OQL query for the view entity.
func (w *Writer) CreateViewEntitySourceDocument(moduleID model.ID, moduleName, docName, oqlQuery, documentation string) (model.ID, error) {
	docID := model.ID(generateUUID())

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(docID))},
		{Key: "$Type", Value: "DomainModels$ViewEntitySourceDocument"},
		{Key: "Documentation", Value: documentation},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "Name", Value: docName},
		{Key: "Oql", Value: oqlQuery},
	}

	contents, err := bson.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to serialize ViewEntitySourceDocument: %w", err)
	}

	if err := w.insertUnit(string(docID), string(moduleID), "Documents", "DomainModels$ViewEntitySourceDocument", contents); err != nil {
		return "", fmt.Errorf("failed to insert ViewEntitySourceDocument: %w", err)
	}

	return docID, nil
}

// DeleteViewEntitySourceDocument deletes a ViewEntitySourceDocument.
func (w *Writer) DeleteViewEntitySourceDocument(id model.ID) error {
	return w.deleteUnit(string(id))
}

// FindViewEntitySourceDocumentID finds a ViewEntitySourceDocument by module and document name.
// Returns the document ID if found, empty string if not found.
func (w *Writer) FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error) {
	units, err := w.reader.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return "", err
	}

	// Build module ID -> name map
	modules, err := w.reader.ListModules()
	if err != nil {
		return "", err
	}
	moduleNames := make(map[string]string)
	for _, m := range modules {
		moduleNames[string(m.ID)] = m.Name
	}

	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		name, _ := raw["Name"].(string)
		modName := moduleNames[u.ContainerID]

		if modName == moduleName && name == docName {
			return model.ID(u.ID), nil
		}
	}

	return "", nil // Not found
}

// DeleteViewEntitySourceDocumentByName deletes ALL ViewEntitySourceDocuments matching the
// given module and document name. This handles cleanup of duplicate documents that may
// have accumulated from previous script runs or incomplete deletions.
// Returns nil if documents were deleted or none existed.
func (w *Writer) DeleteViewEntitySourceDocumentByName(moduleName, docName string) error {
	docIDs, err := w.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
	if err != nil {
		return err
	}
	for _, docID := range docIDs {
		if err := w.deleteUnit(string(docID)); err != nil {
			return err
		}
	}
	return nil
}

// FindAllViewEntitySourceDocumentIDs finds ALL ViewEntitySourceDocuments matching the
// given module and document name. Returns all matching IDs (not just the first).
func (w *Writer) FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error) {
	units, err := w.reader.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return nil, err
	}

	// Build module ID -> name map
	modules, err := w.reader.ListModules()
	if err != nil {
		return nil, err
	}
	moduleNames := make(map[string]string)
	for _, m := range modules {
		moduleNames[string(m.ID)] = m.Name
	}

	var ids []model.ID
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		name, _ := raw["Name"].(string)
		modName := moduleNames[u.ContainerID]

		if modName == moduleName && name == docName {
			ids = append(ids, model.ID(u.ID))
		}
	}

	return ids, nil
}
func (w *Writer) serializeDomainModel(dm *domainmodel.DomainModel) ([]byte, error) {
	// Look up module name for qualified names in validation rules
	moduleName := ""
	if dm.ContainerID != "" {
		module, err := w.reader.GetModule(dm.ContainerID)
		if err == nil && module != nil {
			moduleName = module.Name
		}
	}

	// Entities array with version prefix 3
	entities := bson.A{int32(3)}
	for _, e := range dm.Entities {
		entities = append(entities, serializeEntity(e, moduleName))
	}

	// Associations array with version prefix 3
	associations := bson.A{int32(3)}
	for _, a := range dm.Associations {
		associations = append(associations, serializeAssociation(a))
	}

	// CrossAssociations array with version prefix 3
	crossAssociations := bson.A{int32(3)}
	for _, ca := range dm.CrossAssociations {
		crossAssociations = append(crossAssociations, serializeCrossAssociation(ca))
	}

	doc := bson.M{
		"$ID":               idToBsonBinary(string(dm.ID)),
		"$Type":             "DomainModels$DomainModel",
		"Documentation":     "",
		"Annotations":       bson.A{int32(3)}, // Empty with version prefix
		"Entities":          entities,
		"Associations":      associations,
		"CrossAssociations": crossAssociations,
	}
	return bson.Marshal(doc)
}

func serializeEntity(e *domainmodel.Entity, moduleName string) bson.D {
	// Attributes array with version prefix 3
	attrs := bson.A{int32(3)}
	for _, a := range e.Attributes {
		attrs = append(attrs, serializeAttribute(a))
	}

	// Indexes array with version prefix 3
	indexes := bson.A{int32(3)}
	for _, idx := range e.Indexes {
		indexes = append(indexes, serializeIndex(idx))
	}

	// ValidationRules array with version prefix 3
	validationRules := bson.A{int32(3)}
	for _, vr := range e.ValidationRules {
		validationRules = append(validationRules, serializeValidationRule(vr, moduleName, e))
	}

	// Generate a GUID for the entity if not present (used for qualified name)
	entityGUID := idToBsonBinary(string(e.ID))

	// Location is stored as "x;y" string format
	location := fmt.Sprintf("%d;%d", e.Location.X, e.Location.Y)

	// Serialize generalization: either a parent entity reference or NoGeneralization
	var maybeGeneralization bson.D
	if e.GeneralizationRef != "" {
		maybeGeneralization = serializeGeneralization(e.GeneralizationRef)
	} else {
		maybeGeneralization = serializeNoGeneralization(e.Persistable)
	}

	// AccessRules array with version prefix 3
	accessRules := bson.A{int32(3)}
	for _, ar := range e.AccessRules {
		accessRules = append(accessRules, serializeAccessRule(ar))
	}

	// Use bson.D (ordered document) to match Studio Pro field order
	// CRITICAL: Attributes MUST come before ValidationRules for attribute lookup to work
	doc := bson.D{
		{Key: "Name", Value: e.Name},
		{Key: "Documentation", Value: e.Documentation},
		{Key: "MaybeGeneralization", Value: maybeGeneralization},
		{Key: "Attributes", Value: attrs}, // Must come before ValidationRules!
		{Key: "AccessRules", Value: accessRules},
		{Key: "ValidationRules", Value: validationRules}, // After Attributes
		{Key: "$ID", Value: idToBsonBinary(string(e.ID))},
		{Key: "$Type", Value: "DomainModels$EntityImpl"},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "GUID", Value: entityGUID},
		{Key: "Location", Value: location},
		{Key: "Indexes", Value: indexes},
		{Key: "EventHandlers", Value: bson.A{int32(3)}},
	}

	// Add Source for view entities (references a ViewEntitySourceDocument)
	if e.Source == "DomainModels$OqlViewEntitySource" && e.SourceDocumentRef != "" {
		doc = append(doc, bson.E{Key: "Source", Value: serializeOqlViewEntitySource(e.SourceDocumentRef)})
	}

	// Add Source for external entities (OData remote entity source)
	if e.Source == "Rest$ODataRemoteEntitySource" && e.RemoteServiceName != "" {
		doc = append(doc, bson.E{Key: "Source", Value: serializeODataRemoteEntitySource(
			e.RemoteServiceName, e.RemoteEntitySet, e.RemoteEntityName,
			e.Countable, e.Creatable, e.Deletable, e.Updatable,
		)})
	}

	return doc
}

func serializeAccessRule(ar *domainmodel.AccessRule) bson.D {
	// AllowedModuleRoles: storageListType 1 (BY_NAME references)
	roles := bson.A{int32(1)}
	for _, name := range ar.ModuleRoleNames {
		roles = append(roles, name)
	}

	// MemberAccesses: storageListType 3
	memberAccesses := bson.A{int32(3)}
	for _, ma := range ar.MemberAccesses {
		memberAccesses = append(memberAccesses, serializeMemberAccess(ma))
	}

	ruleID := string(ar.ID)
	if ruleID == "" {
		ruleID = generateUUID()
	}

	defaultMemberAccess := string(ar.DefaultMemberAccessRights)
	if defaultMemberAccess == "" {
		defaultMemberAccess = "None"
	}

	return bson.D{
		{Key: "$Type", Value: "DomainModels$AccessRule"},
		{Key: "$ID", Value: idToBsonBinary(ruleID)},
		{Key: "AllowedModuleRoles", Value: roles},
		{Key: "AllowCreate", Value: ar.AllowCreate},
		{Key: "AllowDelete", Value: ar.AllowDelete},
		{Key: "DefaultMemberAccessRights", Value: defaultMemberAccess},
		{Key: "XPathConstraint", Value: ar.XPathConstraint},
		{Key: "XPathConstraintCaption", Value: ""},
		{Key: "Documentation", Value: ""},
		{Key: "MemberAccesses", Value: memberAccesses},
	}
}

func serializeMemberAccess(ma *domainmodel.MemberAccess) bson.D {
	maID := string(ma.ID)
	if maID == "" {
		maID = generateUUID()
	}

	doc := bson.D{
		{Key: "$Type", Value: "DomainModels$MemberAccess"},
		{Key: "$ID", Value: idToBsonBinary(maID)},
		{Key: "AccessRights", Value: string(ma.AccessRights)},
	}

	// Attribute reference (BY_NAME)
	if ma.AttributeName != "" {
		doc = append(doc, bson.E{Key: "Attribute", Value: ma.AttributeName})
	}

	// Association reference (BY_NAME)
	if ma.AssociationName != "" {
		doc = append(doc, bson.E{Key: "Association", Value: ma.AssociationName})
	}

	return doc
}

func serializeNoGeneralization(persistable bool) bson.D {
	// Use bson.D with Studio Pro field order: $ID, $Type, Persistable
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "DomainModels$NoGeneralization"},
		{Key: "Persistable", Value: persistable},
	}
}

func serializeGeneralization(parentRef string) bson.D {
	// Generalization stores the parent entity as a BY_NAME qualified name string
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "DomainModels$Generalization"},
		{Key: "Generalization", Value: parentRef},
	}
}

func serializeOqlViewEntitySource(sourceDocumentRef string) bson.D {
	// OQL view source with reference to ViewEntitySourceDocument by qualified name
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "DomainModels$OqlViewEntitySource"},
		{Key: "SourceDocument", Value: sourceDocumentRef},
	}
}

func serializeODataRemoteEntitySource(serviceName, entitySet, remoteName string, countable, creatable, deletable, updatable bool) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "Rest$ODataRemoteEntitySource"},
		{Key: "SourceDocument", Value: serviceName},
		{Key: "EntitySet", Value: entitySet},
		{Key: "RemoteName", Value: remoteName},
		{Key: "Countable", Value: countable},
		{Key: "Creatable", Value: creatable},
		{Key: "Deletable", Value: deletable},
		{Key: "Updatable", Value: updatable},
	}
}

func serializeAttribute(a *domainmodel.Attribute) bson.D {
	// Attribute type with its own ID - use bson.D for ordered fields
	typeName := "DomainModels$StringAttributeType"
	if a.Type != nil {
		typeName = "DomainModels$" + a.Type.GetTypeName() + "AttributeType"
	}

	attrType := bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: typeName},
	}
	// Add type-specific properties
	if a.Type != nil {
		switch t := a.Type.(type) {
		case *domainmodel.StringAttributeType:
			attrType = append(attrType, bson.E{Key: "Length", Value: t.Length})
		case *domainmodel.EnumerationAttributeType:
			// Enumeration uses BY_NAME_REFERENCE - store as qualified name string
			enumRef := t.EnumerationRef
			if enumRef == "" && t.EnumerationID != "" {
				// Fall back to ID if no ref (though this shouldn't happen for new entities)
				enumRef = string(t.EnumerationID)
			}
			attrType = append(attrType, bson.E{Key: "Enumeration", Value: enumRef})
		}
	}

	// Determine value type: OqlViewValue, CalculatedValue, or StoredValue
	var valueDoc bson.D
	if a.Value != nil && a.Value.ViewReference != "" {
		// View entity attribute - use OqlViewValue
		valueDoc = bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DomainModels$OqlViewValue"},
			{Key: "Reference", Value: a.Value.ViewReference},
		}
	} else if a.Value != nil && a.Value.Type == "CalculatedValue" {
		// Calculated attribute - use CalculatedValue (Microflow is ByNameReference → string)
		microflowRef := a.Value.MicroflowName
		valueDoc = bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DomainModels$CalculatedValue"},
			{Key: "Microflow", Value: microflowRef},
			{Key: "PassEntity", Value: microflowRef != ""},
		}
	} else {
		// Regular entity attribute - use StoredValue
		defaultValue := ""
		if a.Value != nil && a.Value.DefaultValue != "" {
			defaultValue = a.Value.DefaultValue
		}
		valueDoc = bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DomainModels$StoredValue"},
			{Key: "DefaultValue", Value: defaultValue},
		}
	}

	// Use bson.D with Studio Pro field order:
	// Name, Documentation, ExportLevel, GUID, NewType, Value, $ID, $Type
	return bson.D{
		{Key: "Name", Value: a.Name},
		{Key: "Documentation", Value: a.Documentation},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "GUID", Value: idToBsonBinary(string(a.ID))},
		{Key: "NewType", Value: attrType},
		{Key: "Value", Value: valueDoc},
		{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
		{Key: "$Type", Value: "DomainModels$Attribute"},
	}
}

func serializeAssociation(a *domainmodel.Association) bson.M {
	storageFormat := string(a.StorageFormat)
	if storageFormat == "" {
		storageFormat = "Column"
	}
	return bson.M{
		"$ID":              idToBsonBinary(string(a.ID)),
		"$Type":            "DomainModels$Association",
		"Name":             a.Name,
		"Documentation":    a.Documentation,
		"ExportLevel":      "Hidden",
		"GUID":             idToBsonBinary(string(a.ID)),
		"ParentPointer":    idToBsonBinary(string(a.ParentID)),
		"ChildPointer":     idToBsonBinary(string(a.ChildID)),
		"Type":             string(a.Type),
		"Owner":            string(a.Owner),
		"ParentConnection": "0;50",
		"ChildConnection":  "100;50",
		"StorageFormat":    storageFormat,
		"Source":           nil,
		"DeleteBehavior":   serializeDeleteBehavior(a.ParentDeleteBehavior, a.ChildDeleteBehavior),
	}
}

func serializeCrossAssociation(ca *domainmodel.CrossModuleAssociation) bson.M {
	storageFormat := string(ca.StorageFormat)
	if storageFormat == "" {
		storageFormat = "Column"
	}
	return bson.M{
		"$ID":              idToBsonBinary(string(ca.ID)),
		"$Type":            "DomainModels$CrossAssociation",
		"Name":             ca.Name,
		"Documentation":    ca.Documentation,
		"ExportLevel":      "Hidden",
		"GUID":             idToBsonBinary(string(ca.ID)),
		"ParentPointer":    idToBsonBinary(string(ca.ParentID)),
		"Child":            ca.ChildRef,
		"Type":             string(ca.Type),
		"Owner":            string(ca.Owner),
		"ParentConnection": "0;50",
		"ChildConnection":  "100;50",
		"StorageFormat":    storageFormat,
		"Source":           nil,
		"DeleteBehavior":   serializeDeleteBehavior(ca.ParentDeleteBehavior, ca.ChildDeleteBehavior),
	}
}

func serializeDeleteBehavior(parentBehavior, childBehavior *domainmodel.DeleteBehavior) bson.M {
	parentType := "DeleteMeButKeepReferences"
	childType := "DeleteMeButKeepReferences"

	if parentBehavior != nil && parentBehavior.Type != "" {
		parentType = string(parentBehavior.Type)
	}
	if childBehavior != nil && childBehavior.Type != "" {
		childType = string(childBehavior.Type)
	}

	return bson.M{
		"$ID":                  idToBsonBinary(generateUUID()),
		"$Type":                "DomainModels$DeleteBehavior",
		"ChildDeleteBehavior":  childType,
		"ChildErrorMessage":    nil,
		"ParentDeleteBehavior": parentType,
		"ParentErrorMessage":   nil,
	}
}

func serializeIndex(idx *domainmodel.Index) bson.M {
	// Index attributes array with version prefix 3
	attrs := bson.A{int32(3)}
	for _, ia := range idx.Attributes {
		attrs = append(attrs, serializeIndexAttribute(ia))
	}

	return bson.M{
		"$ID":        idToBsonBinary(string(idx.ID)),
		"$Type":      "DomainModels$EntityIndex",
		"Attributes": attrs,
	}
}

func serializeIndexAttribute(ia *domainmodel.IndexAttribute) bson.M {
	return bson.M{
		"$ID":              idToBsonBinary(string(ia.ID)),
		"$Type":            "DomainModels$IndexedAttribute",
		"AttributePointer": idToBsonBinary(string(ia.AttributeID)), // BSON Binary like $ID
		"SortOrder":        getSortOrder(ia.Ascending),
	}
}

func getSortOrder(ascending bool) string {
	if ascending {
		return "Ascending"
	}
	return "Descending"
}

func serializeValidationRule(vr *domainmodel.ValidationRule, moduleName string, entity *domainmodel.Entity) bson.D {
	// Look up attribute name from the entity's attributes using AttributeID
	// The Attribute field uses BY_NAME_REFERENCE, so it must be a qualified name STRING
	// Format: "ModuleName.EntityName.AttributeName"
	//
	// NOTE: AttributeID can be either:
	// 1. A UUID (when entity was just created) - compare with attr.ID
	// 2. A qualified name string (when entity was read from disk) - extract attr name and compare
	attributeQualifiedName := ""
	attrIDStr := string(vr.AttributeID)

	// Check if AttributeID is already a qualified name (contains dots)
	if strings.Contains(attrIDStr, ".") {
		// It's already a qualified name - use it directly
		attributeQualifiedName = attrIDStr
	} else {
		// It's a UUID - look up the attribute name
		for _, attr := range entity.Attributes {
			if attr.ID == vr.AttributeID {
				attributeQualifiedName = fmt.Sprintf("%s.%s.%s", moduleName, entity.Name, attr.Name)
				break
			}
		}
	}

	// Use bson.D (ordered document) to match Studio Pro's field order:
	// $ID, $Type, Attribute, Message, RuleInfo
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(vr.ID))},
		{Key: "$Type", Value: "DomainModels$ValidationRule"},
		{Key: "Attribute", Value: attributeQualifiedName}, // BY_NAME_REFERENCE: qualified name STRING
	}

	// Message comes before RuleInfo in Studio Pro's format
	if vr.ErrorMessage != nil && len(vr.ErrorMessage.Translations) > 0 {
		doc = append(doc, bson.E{Key: "Message", Value: serializeText(vr.ErrorMessage)})
	}

	// RuleInfo comes last
	doc = append(doc, bson.E{Key: "RuleInfo", Value: serializeRuleInfo(vr.Type)})

	return doc
}

func serializeRuleInfo(ruleType string) bson.D {
	// Use bson.D (ordered document) - Studio Pro uses $ID first, then $Type
	switch ruleType {
	case "Required":
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DomainModels$RequiredRuleInfo"},
		}
	case "Unique":
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DomainModels$UniqueRuleInfo"},
		}
	default:
		// Fallback to required
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DomainModels$RequiredRuleInfo"},
		}
	}
}

func serializeText(text *model.Text) bson.D {
	// Translations as Items array with version prefix 3
	// Use bson.D for ordered documents to match Studio Pro format
	items := bson.A{int32(3)}
	for lang, value := range text.Translations {
		items = append(items, bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Texts$Translation"},
			{Key: "LanguageCode", Value: lang},
			{Key: "Text", Value: value},
		})
	}

	// Studio Pro order: $ID, $Type, Items
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(text.ID))},
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: items},
	}
}
