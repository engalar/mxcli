// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateBusinessEventService creates a new business event service document.
func (w *Writer) CreateBusinessEventService(svc *model.BusinessEventService) error {
	if svc.ID == "" {
		svc.ID = model.ID(generateUUID())
	}
	svc.TypeName = "BusinessEvents$BusinessEventService"

	contents, err := serializeBusinessEventService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize business event service: %w", err)
	}

	return w.insertUnit(string(svc.ID), string(svc.ContainerID), "Documents", "BusinessEvents$BusinessEventService", contents)
}

// UpdateBusinessEventService updates an existing business event service.
func (w *Writer) UpdateBusinessEventService(svc *model.BusinessEventService) error {
	contents, err := serializeBusinessEventService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize business event service: %w", err)
	}

	return w.updateUnit(string(svc.ID), contents)
}

// DeleteBusinessEventService deletes a business event service by ID.
func (w *Writer) DeleteBusinessEventService(id model.ID) error {
	return w.deleteUnit(string(id))
}

// serializeBusinessEventService converts a BusinessEventService to BSON bytes.
func serializeBusinessEventService(svc *model.BusinessEventService) ([]byte, error) {
	doc := bson.M{
		"$ID":           idToBsonBinary(string(svc.ID)),
		"$Type":         "BusinessEvents$BusinessEventService",
		"Name":          svc.Name,
		"Documentation": svc.Documentation,
		"Excluded":      svc.Excluded,
		"ExportLevel":   svc.ExportLevel,
	}

	// Serialize Definition
	if svc.Definition != nil {
		doc["Definition"] = serializeBusinessEventDefinition(svc.Definition)
	} else {
		doc["Definition"] = nil
	}

	// Serialize OperationImplementations
	opImpls := bson.A{int32(2)} // versioned array prefix
	for _, op := range svc.OperationImplementations {
		opImpls = append(opImpls, serializeServiceOperation(op))
	}
	doc["OperationImplementations"] = opImpls

	// SourceApi is null for service definitions
	doc["SourceApi"] = nil

	return bson.Marshal(doc)
}

func serializeBusinessEventDefinition(def *model.BusinessEventDefinition) bson.M {
	defDoc := bson.M{
		"$Type":           "BusinessEvents$BusinessEventDefinition",
		"ServiceName":     def.ServiceName,
		"EventNamePrefix": def.EventNamePrefix,
		"Description":     def.Description,
		"Summary":         def.Summary,
	}
	if def.ID != "" {
		defDoc["$ID"] = idToBsonBinary(string(def.ID))
	} else {
		defDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// Serialize Channels
	channels := bson.A{int32(2)} // versioned array prefix
	for _, ch := range def.Channels {
		channels = append(channels, serializeBusinessEventChannel(ch))
	}
	defDoc["Channels"] = channels

	return defDoc
}

func serializeBusinessEventChannel(ch *model.BusinessEventChannel) bson.M {
	chDoc := bson.M{
		"$Type":       "BusinessEvents$Channel",
		"ChannelName": ch.ChannelName,
		"Description": ch.Description,
	}
	if ch.ID != "" {
		chDoc["$ID"] = idToBsonBinary(string(ch.ID))
	} else {
		chDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// Serialize Messages
	messages := bson.A{int32(2)} // versioned array prefix
	for _, msg := range ch.Messages {
		messages = append(messages, serializeBusinessEventMessage(msg))
	}
	chDoc["Messages"] = messages

	return chDoc
}

func serializeBusinessEventMessage(msg *model.BusinessEventMessage) bson.M {
	msgDoc := bson.M{
		"$Type":        "BusinessEvents$Message",
		"MessageName":  msg.MessageName,
		"Description":  msg.Description,
		"CanPublish":   msg.CanPublish,
		"CanSubscribe": msg.CanSubscribe,
	}
	if msg.ID != "" {
		msgDoc["$ID"] = idToBsonBinary(string(msg.ID))
	} else {
		msgDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// Serialize Attributes
	attrs := bson.A{int32(2)} // versioned array prefix
	for _, attr := range msg.Attributes {
		attrs = append(attrs, serializeBusinessEventAttribute(attr))
	}
	msgDoc["Attributes"] = attrs

	return msgDoc
}

func serializeBusinessEventAttribute(attr *model.BusinessEventAttribute) bson.M {
	attrDoc := bson.M{
		"$Type":         "BusinessEvents$MessageAttribute",
		"AttributeName": attr.AttributeName,
		"Description":   attr.Description,
	}
	if attr.ID != "" {
		attrDoc["$ID"] = idToBsonBinary(string(attr.ID))
	} else {
		attrDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// Convert attribute type to BSON format: "Long" → {"$Type": "DomainModels$LongAttributeType", "$ID": ...}
	attrTypeDoc := bson.M{
		"$Type": attributeTypeToBsonType(attr.AttributeType),
		"$ID":   idToBsonBinary(generateUUID()),
	}
	// Date and DateTime both use DateTimeAttributeType; distinguish via LocalizeDate
	if attr.AttributeType == "DateTime" {
		attrTypeDoc["LocalizeDate"] = true
	} else if attr.AttributeType == "Date" {
		attrTypeDoc["LocalizeDate"] = false
	}
	attrDoc["AttributeType"] = attrTypeDoc

	return attrDoc
}

// attributeTypeToBsonType converts a simple type name to a BSON $Type string.
func attributeTypeToBsonType(typeName string) string {
	switch typeName {
	case "Long":
		return "DomainModels$LongAttributeType"
	case "String":
		return "DomainModels$StringAttributeType"
	case "Integer":
		return "DomainModels$IntegerAttributeType"
	case "Boolean":
		return "DomainModels$BooleanAttributeType"
	case "DateTime", "Date":
		return "DomainModels$DateTimeAttributeType"
	case "Decimal":
		return "DomainModels$DecimalAttributeType"
	case "AutoNumber":
		return "DomainModels$AutoNumberAttributeType"
	case "Binary":
		return "DomainModels$BinaryAttributeType"
	default:
		return "DomainModels$StringAttributeType"
	}
}

func serializeServiceOperation(op *model.ServiceOperation) bson.M {
	opDoc := bson.M{
		"$Type":       "BusinessEvents$ServiceOperation",
		"MessageName": op.MessageName,
		"Operation":   op.Operation,
		"Entity":      op.Entity,
		"Microflow":   op.Microflow,
	}
	if op.ID != "" {
		opDoc["$ID"] = idToBsonBinary(string(op.ID))
	} else {
		opDoc["$ID"] = idToBsonBinary(generateUUID())
	}
	return opDoc
}
