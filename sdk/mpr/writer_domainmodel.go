// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// MoveViewEntitySourceDocument moves a ViewEntitySourceDocument to a new module.
func (w *Writer) MoveViewEntitySourceDocument(sourceModuleName string, targetModuleID model.ID, docName string) error {
	docID, err := w.reader.FindViewEntitySourceDocumentID(sourceModuleName, docName)
	if err != nil {
		return err
	}
	if docID == "" {
		return nil // No document to move
	}

	// Update ContainerID in database
	return w.moveUnitByID(string(docID), string(targetModuleID))
}

// ScanOqlQueryUpdates scans all ViewEntitySourceDocuments and returns patches
// for those whose OQL contains oldQualifiedName replaced with newQualifiedName.
// The returned count equals len(patches). No writes are performed.
func (r *Reader) ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName string) ([]UnitPatch, int, error) {
	units, err := r.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return nil, 0, err
	}

	var patches []UnitPatch
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}
		oql, _ := raw["Oql"].(string)
		if oql == "" || !strings.Contains(oql, oldQualifiedName) {
			continue
		}
		raw["Oql"] = strings.ReplaceAll(oql, oldQualifiedName, newQualifiedName)
		contents, err := bson.Marshal(raw)
		if err != nil {
			continue
		}
		patches = append(patches, UnitPatch{ID: u.ID, Contents: contents})
	}
	return patches, len(patches), nil
}

// ScanOqlQueryUpdates is the Writer method form; delegates to the Reader.
func (w *Writer) ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName string) ([]UnitPatch, int, error) {
	return w.reader.ScanOqlQueryUpdates(oldQualifiedName, newQualifiedName)
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

// DeleteViewEntitySourceDocumentByName deletes ALL ViewEntitySourceDocuments matching the
// given module and document name. This handles cleanup of duplicate documents that may
// have accumulated from previous script runs or incomplete deletions.
// Returns nil if documents were deleted or none existed.
func (w *Writer) DeleteViewEntitySourceDocumentByName(moduleName, docName string) error {
	docIDs, err := w.reader.FindAllViewEntitySourceDocumentIDs(moduleName, docName)
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

func serializeText(text *model.Text) bson.D {
	// Translations as Items array with version prefix 3
	// Use bson.D for ordered documents to match Studio Pro format
	items := bson.A{int32(3)}
	// Sort language keys for deterministic output
	langs := make([]string, 0, len(text.Translations))
	for lang := range text.Translations {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	for _, lang := range langs {
		value := text.Translations[lang]
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
