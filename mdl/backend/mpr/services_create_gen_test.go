// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
)

// TestCreateBusinessEventServiceGen_WritesDefinition verifies that
// createBusinessEventServiceGen persists the Definition (Part) and
// OperationImplementations (PartList) sub-documents.
//
// Before the fix these fields were never written: Studio Pro would load the
// unit, call TryGetIcon on an IBusinessEventService with a nil Definition,
// and crash with "System.InvalidOperationException: unsupported document type".
func TestCreateBusinessEventServiceGen_WritesDefinition(t *testing.T) {
	const containerID = "00000001-0000-0000-0000-000000000001"
	const svcID = "00000002-0000-0000-0000-000000000002"

	// Minimal MPR: one dummy unit that acts as the module container.
	contents := makeBSONUnit(t, containerID, "Projects$Module", bson.D{
		{Key: "Name", Value: "TestModule"},
	})
	mprPath, _ := makeServiceTestMPR(t, containerID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	svc := &model.BusinessEventService{
		ContainerID: model.ID(containerID),
		Name:        "TestEventService",
		ExportLevel: "Hidden",
		Definition: &model.BusinessEventDefinition{
			ServiceName:     "TestEventService",
			EventNamePrefix: "test",
		},
		OperationImplementations: []*model.ServiceOperation{
			{
				MessageName: "OrderPlaced",
				Operation:   "publish",
				Entity:      "MyModule.PBE_OrderPlaced",
			},
		},
	}
	svc.ID = model.ID(svcID)
	svc.Definition.TypeName = "BusinessEvents$BusinessEventDefinition"
	svc.OperationImplementations[0].TypeName = "BusinessEvents$ServiceOperation"

	if err := b.createBusinessEventServiceGen(svc); err != nil {
		t.Fatalf("createBusinessEventServiceGen: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(svcID)
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}

	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal BSON: %v", err)
	}

	var definitionVal any
	var opsVal any
	for _, e := range doc {
		switch e.Key {
		case "Definition":
			definitionVal = e.Value
		case "OperationImplementations":
			opsVal = e.Value
		}
	}

	if definitionVal == nil {
		t.Error("Definition field missing or nil in BSON — Studio Pro crashes with 'unsupported document type'")
	}
	if opsVal == nil {
		t.Error("OperationImplementations field missing or nil in BSON")
	}

	// Verify Definition is a non-empty document (not just a null value).
	if raw, ok := definitionVal.(bson.Raw); ok {
		if len(raw) == 0 {
			t.Error("Definition is present but empty — must be a non-empty sub-document")
		}
	}
}

// TestCreateBusinessEventServiceGen_AttributeTypeUsesDomainModels verifies that
// MessageAttribute.AttributeType is written as a DomainModels$*AttributeType,
// NOT a DataTypes$*Type.  Mendix refuses to load the MPR if the wrong type
// family is used ("cannot be converted to type AttributeTypeBase").
func TestCreateBusinessEventServiceGen_AttributeTypeUsesDomainModels(t *testing.T) {
	const containerID = "00000005-0000-0000-0000-000000000005"
	const svcID = "00000006-0000-0000-0000-000000000006"

	contents := makeBSONUnit(t, containerID, "Projects$Module", bson.D{
		{Key: "Name", Value: "AttrTypeModule"},
	})
	mprPath, _ := makeServiceTestMPR(t, containerID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	ch := &model.BusinessEventChannel{ChannelName: "test-channel"}
	ch.TypeName = "BusinessEvents$Channel"
	msg := &model.BusinessEventMessage{MessageName: "TestMsg", CanPublish: true}
	msg.TypeName = "BusinessEvents$Message"
	// One attribute per type so we cover all branches of beAttrTypeElem.
	for _, tc := range []struct {
		name    string
		mdlType string
		want    string // expected BSON $Type
	}{
		{"IntAttr", "Integer", "DomainModels$IntegerAttributeType"},
		{"LongAttr", "Long", "DomainModels$LongAttributeType"},
		{"DecAttr", "Decimal", "DomainModels$DecimalAttributeType"},
		{"BoolAttr", "Boolean", "DomainModels$BooleanAttributeType"},
		{"DTAttr", "DateTime", "DomainModels$DateTimeAttributeType"},
		{"StrAttr", "String", "DomainModels$StringAttributeType"},
	} {
		a := &model.BusinessEventAttribute{AttributeName: tc.name, AttributeType: tc.mdlType}
		a.TypeName = "BusinessEvents$MessageAttribute"
		msg.Attributes = append(msg.Attributes, a)
	}
	ch.Messages = append(ch.Messages, msg)

	svc := &model.BusinessEventService{
		ContainerID: model.ID(containerID),
		Name:        "AttrTypeService",
		ExportLevel: "Hidden",
		Definition: &model.BusinessEventDefinition{
			ServiceName:     "AttrTypeService",
			EventNamePrefix: "attr",
			Channels:        []*model.BusinessEventChannel{ch},
		},
	}
	svc.ID = model.ID(svcID)
	svc.Definition.TypeName = "BusinessEvents$BusinessEventDefinition"

	if err := b.createBusinessEventServiceGen(svc); err != nil {
		t.Fatalf("createBusinessEventServiceGen: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(svcID)
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}

	// Decode and dig into Definition.Channels[1].Messages[1].Attributes[1..6].
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Helper: walk bson.D for a key, return value.
	field := func(d bson.D, key string) any {
		for _, e := range d {
			if e.Key == key {
				return e.Value
			}
		}
		return nil
	}
	asD := func(v any) bson.D {
		if d, ok := v.(bson.D); ok {
			return d
		}
		return nil
	}
	asA := func(v any) bson.A {
		if a, ok := v.(bson.A); ok {
			return a
		}
		return nil
	}

	defD := asD(field(doc, "Definition"))
	if defD == nil {
		t.Fatal("Definition missing")
	}
	channels := asA(field(defD, "Channels"))
	if len(channels) < 2 {
		t.Fatalf("Channels too short: %d", len(channels))
	}
	chD := asD(channels[1])
	if chD == nil {
		t.Fatal("channel element not bson.D")
	}
	messages := asA(field(chD, "Messages"))
	if len(messages) < 2 {
		t.Fatalf("Messages too short: %d", len(messages))
	}
	msgD := asD(messages[1])
	if msgD == nil {
		t.Fatal("message element not bson.D")
	}
	attrs := asA(field(msgD, "Attributes"))
	// attrs[0] is int32 version prefix, attrs[1..6] are the 6 attributes.
	if len(attrs) < 7 {
		t.Fatalf("Attributes too short: %d (want 7)", len(attrs))
	}

	wantTypes := []string{
		"DomainModels$IntegerAttributeType",
		"DomainModels$LongAttributeType",
		"DomainModels$DecimalAttributeType",
		"DomainModels$BooleanAttributeType",
		"DomainModels$DateTimeAttributeType",
		"DomainModels$StringAttributeType",
	}
	for i, want := range wantTypes {
		attrD := asD(attrs[i+1])
		if attrD == nil {
			t.Errorf("attr[%d] not bson.D", i)
			continue
		}
		attrTypeD := asD(field(attrD, "AttributeType"))
		if attrTypeD == nil {
			t.Errorf("attr[%d].AttributeType missing", i)
			continue
		}
		got, _ := field(attrTypeD, "$Type").(string)
		if got != want {
			t.Errorf("attr[%d].AttributeType.$Type = %q, want %q", i, got, want)
		}
	}
}

// TestCreateBusinessEventServiceGen_OperationEntityIsWritten verifies that
// ServiceOperation.Entity is persisted as a qualified-name reference so that
// Studio Pro can display the linked entity in the service editor.
func TestCreateBusinessEventServiceGen_OperationEntityIsWritten(t *testing.T) {
	const containerID = "00000003-0000-0000-0000-000000000003"
	const svcID = "00000004-0000-0000-0000-000000000004"

	contents := makeBSONUnit(t, containerID, "Projects$Module", bson.D{
		{Key: "Name", Value: "TestModule2"},
	})
	mprPath, _ := makeServiceTestMPR(t, containerID, contents)

	b := New()
	if err := b.Connect(mprPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	const wantEntity = "Orders.PBE_OrderPlaced"
	svc := &model.BusinessEventService{
		ContainerID: model.ID(containerID),
		Name:        "OrderEvents",
		ExportLevel: "Hidden",
		Definition: &model.BusinessEventDefinition{
			ServiceName:     "OrderEvents",
			EventNamePrefix: "orders",
		},
		OperationImplementations: []*model.ServiceOperation{
			{
				MessageName: "OrderPlaced",
				Operation:   "publish",
				Entity:      wantEntity,
			},
		},
	}
	svc.ID = model.ID(svcID)
	svc.Definition.TypeName = "BusinessEvents$BusinessEventDefinition"
	svc.OperationImplementations[0].TypeName = "BusinessEvents$ServiceOperation"

	if err := b.createBusinessEventServiceGen(svc); err != nil {
		t.Fatalf("createBusinessEventServiceGen: %v", err)
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(svcID)
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}

	// Extract OperationImplementations array and check Entity field.
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, e := range doc {
		if e.Key != "OperationImplementations" {
			continue
		}
		arr, ok := e.Value.(bson.A)
		if !ok || len(arr) == 0 {
			t.Fatal("OperationImplementations is not an array or is empty")
		}
		// arr[0] is the versioned array prefix (int32), arr[1] is the first op.
		if len(arr) < 2 {
			t.Fatal("OperationImplementations has no operation element after version prefix")
		}
		// arr[1] may be bson.D or bson.Raw depending on the unmarshal path.
		var opDoc bson.D
		switch v := arr[1].(type) {
		case bson.D:
			opDoc = v
		case bson.Raw:
			if err := bson.Unmarshal(v, &opDoc); err != nil {
				t.Fatalf("unmarshal op from Raw: %v", err)
			}
		default:
			t.Fatalf("operation[0] unexpected type %T", arr[1])
		}
		for _, f := range opDoc {
			if f.Key == "Entity" {
				if got, _ := f.Value.(string); got != wantEntity {
					t.Errorf("Entity = %q, want %q", got, wantEntity)
				}
				return
			}
		}
		t.Error("Entity field not found in ServiceOperation BSON")
		return
	}
	t.Error("OperationImplementations key not found in BSON")
}
