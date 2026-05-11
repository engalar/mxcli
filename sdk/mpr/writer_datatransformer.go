// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateDataTransformer creates a new DataTransformers$DataTransformer document.
func (w *Writer) CreateDataTransformer(dt *model.DataTransformer) error {
	if dt.ID == "" {
		dt.ID = model.ID(generateUUID())
	}
	dt.TypeName = "DataTransformers$DataTransformer"

	contents, err := serializeDataTransformer(dt)
	if err != nil {
		return fmt.Errorf("failed to serialize data transformer: %w", err)
	}

	return w.insertUnit(string(dt.ID), string(dt.ContainerID), "Documents", "DataTransformers$DataTransformer", contents)
}

// UpdateDataTransformer replaces an existing data transformer unit, preserving its UUID.
func (w *Writer) UpdateDataTransformer(dt *model.DataTransformer) error {
	dt.TypeName = "DataTransformers$DataTransformer"

	contents, err := serializeDataTransformer(dt)
	if err != nil {
		return fmt.Errorf("failed to serialize data transformer: %w", err)
	}

	return w.updateUnit(string(dt.ID), contents)
}

// DeleteDataTransformer deletes a data transformer by ID.
func (w *Writer) DeleteDataTransformer(id model.ID) error {
	return w.deleteUnit(string(id))
}

func serializeDataTransformer(dt *model.DataTransformer) ([]byte, error) {
	// Root element
	rootElemID := generateUUID()
	rootElement := bson.M{
		"$ID":        idToBsonBinary(rootElemID),
		"$Type":      "DataTransformers$StructureObject",
		"Attributes": bson.A{int32(2)},
	}

	// Source
	var source bson.M
	switch strings.ToUpper(dt.SourceType) {
	case "XML":
		source = bson.M{
			"$ID":     idToBsonBinary(generateUUID()),
			"$Type":   "DataTransformers$XmlSource",
			"Content": dt.SourceJSON,
		}
	default: // JSON
		source = bson.M{
			"$ID":     idToBsonBinary(generateUUID()),
			"$Type":   "DataTransformers$JsonSource",
			"Content": dt.SourceJSON,
		}
	}

	// Steps
	steps := bson.A{int32(2)}
	for _, step := range dt.Steps {
		var action bson.M
		switch strings.ToUpper(step.Technology) {
		case "JSLT":
			action = bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTransformers$JsltAction",
				"Jslt":  step.Expression,
			}
		case "XSLT":
			action = bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTransformers$XsltAction",
				"Xslt":  step.Expression,
			}
		default:
			action = bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTransformers$JsltAction",
				"Jslt":  step.Expression,
			}
		}

		steps = append(steps, bson.M{
			"$ID":                  idToBsonBinary(generateUUID()),
			"$Type":                "DataTransformers$Step",
			"Action":               action,
			"InputElementPointer":  idToBsonBinary(rootElemID),
			"OutputElementPointer": idToBsonBinary(rootElemID),
		})
	}

	doc := bson.M{
		"$ID":                idToBsonBinary(string(dt.ID)),
		"$Type":              "DataTransformers$DataTransformer",
		"Name":               dt.Name,
		"Documentation":      "",
		"Excluded":           dt.Excluded,
		"ExportLevel":        "Hidden",
		"Source":             source,
		"Elements":           bson.A{int32(2), rootElement},
		"RootElementPointer": idToBsonBinary(rootElemID),
		"Steps":              steps,
	}

	return bson.Marshal(doc)
}
