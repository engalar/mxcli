// SPDX-License-Identifier: Apache-2.0

package types

import (
	"testing"
)

func TestParseEdmx_OData4(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="DefaultNamespace" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Customer">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int64" Nullable="false"/>
        <Property Name="Name" Type="Edm.String" MaxLength="200"/>
        <NavigationProperty Name="Orders" Type="Collection(DefaultNamespace.Order)" Partner="Customer"/>
      </EntityType>
      <EntityType Name="Order">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int64" Nullable="false"/>
        <Property Name="Amount" Type="Edm.Decimal" Scale="variable"/>
        <NavigationProperty Name="Customer" Type="DefaultNamespace.Customer" Partner="Orders"/>
      </EntityType>
      <EntityContainer Name="Container">
        <EntitySet Name="Customers" EntityType="DefaultNamespace.Customer"/>
        <EntitySet Name="Orders" EntityType="DefaultNamespace.Order"/>
      </EntityContainer>
      <Action Name="PlaceOrder" IsBound="true">
        <Parameter Name="customer" Type="DefaultNamespace.Customer"/>
        <Parameter Name="quantity" Type="Edm.Int32" Nullable="false"/>
        <ReturnType Type="DefaultNamespace.Order"/>
      </Action>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	doc, err := ParseEdmx(xml)
	if err != nil {
		t.Fatal(err)
	}

	if doc.Version != "4.0" {
		t.Errorf("expected version 4.0, got %q", doc.Version)
	}
	if len(doc.Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(doc.Schemas))
	}
	if doc.Schemas[0].Namespace != "DefaultNamespace" {
		t.Errorf("expected namespace DefaultNamespace, got %q", doc.Schemas[0].Namespace)
	}
	if len(doc.Schemas[0].EntityTypes) != 2 {
		t.Fatalf("expected 2 entity types, got %d", len(doc.Schemas[0].EntityTypes))
	}

	// Check Customer entity
	customer := doc.Schemas[0].EntityTypes[0]
	if customer.Name != "Customer" {
		t.Errorf("expected Customer, got %q", customer.Name)
	}
	if len(customer.KeyProperties) != 1 || customer.KeyProperties[0] != "ID" {
		t.Errorf("expected key [ID], got %v", customer.KeyProperties)
	}
	if len(customer.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(customer.Properties))
	}

	// Check ID property nullable
	idProp := customer.Properties[0]
	if idProp.Nullable == nil || *idProp.Nullable {
		t.Error("expected ID property to be non-nullable")
	}

	// Check Name property MaxLength
	nameProp := customer.Properties[1]
	if nameProp.MaxLength != "200" {
		t.Errorf("expected MaxLength 200, got %q", nameProp.MaxLength)
	}

	// Check navigation property
	if len(customer.NavigationProperties) != 1 {
		t.Fatalf("expected 1 nav prop, got %d", len(customer.NavigationProperties))
	}
	nav := customer.NavigationProperties[0]
	if nav.Name != "Orders" {
		t.Errorf("expected Orders, got %q", nav.Name)
	}
	if !nav.IsMany {
		t.Error("expected Orders to be Collection")
	}
	if nav.TargetType != "Order" {
		t.Errorf("expected target type Order, got %q", nav.TargetType)
	}

	// Check entity sets
	if len(doc.EntitySets) != 2 {
		t.Fatalf("expected 2 entity sets, got %d", len(doc.EntitySets))
	}

	// Check action
	if len(doc.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(doc.Actions))
	}
	action := doc.Actions[0]
	if action.Name != "PlaceOrder" {
		t.Errorf("expected PlaceOrder, got %q", action.Name)
	}
	if !action.IsBound {
		t.Error("expected bound action")
	}
	if len(action.Parameters) != 2 {
		t.Errorf("expected 2 params, got %d", len(action.Parameters))
	}
	if action.ReturnType != "DefaultNamespace.Order" {
		t.Errorf("expected return type, got %q", action.ReturnType)
	}
}

func TestParseEdmx_Empty(t *testing.T) {
	_, err := ParseEdmx("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseEdmx_InvalidXML(t *testing.T) {
	_, err := ParseEdmx("<not valid xml")
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestParseEdmx_EnumTypes(t *testing.T) {
	xml := `<?xml version="1.0"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="NS" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EnumType Name="Color">
        <Member Name="Red" Value="0"/>
        <Member Name="Green" Value="1"/>
        <Member Name="Blue" Value="2"/>
      </EnumType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	doc, err := ParseEdmx(xml)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Schemas[0].EnumTypes) != 1 {
		t.Fatalf("expected 1 enum type, got %d", len(doc.Schemas[0].EnumTypes))
	}
	enum := doc.Schemas[0].EnumTypes[0]
	if enum.Name != "Color" {
		t.Errorf("expected Color, got %q", enum.Name)
	}
	if len(enum.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(enum.Members))
	}
}

func TestParseEdmx_CapabilityAnnotations(t *testing.T) {
	xml := `<?xml version="1.0"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="NS" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="C">
        <EntitySet Name="ReadOnly" EntityType="NS.Item">
          <Annotation Term="Org.OData.Capabilities.V1.InsertRestrictions">
            <Record><PropertyValue Property="Insertable" Bool="false"/></Record>
          </Annotation>
          <Annotation Term="Org.OData.Capabilities.V1.DeleteRestrictions">
            <Record><PropertyValue Property="Deletable" Bool="false"/></Record>
          </Annotation>
        </EntitySet>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	doc, err := ParseEdmx(xml)
	if err != nil {
		t.Fatal(err)
	}
	es := doc.EntitySets[0]
	if es.Insertable == nil || *es.Insertable {
		t.Error("expected Insertable=false")
	}
	if es.Deletable == nil || *es.Deletable {
		t.Error("expected Deletable=false")
	}
	if es.Updatable != nil {
		t.Error("expected Updatable=nil (unspecified)")
	}
}

func TestFindEntityType(t *testing.T) {
	doc := &EdmxDocument{
		Schemas: []*EdmSchema{{
			Namespace:   "NS",
			EntityTypes: []*EdmEntityType{{Name: "Customer"}, {Name: "Order"}},
		}},
	}

	if got := doc.FindEntityType("Customer"); got == nil || got.Name != "Customer" {
		t.Error("expected to find Customer")
	}
	if got := doc.FindEntityType("NS.Customer"); got == nil || got.Name != "Customer" {
		t.Error("expected to find Customer with namespace prefix")
	}
	if got := doc.FindEntityType("Missing"); got != nil {
		t.Error("expected nil for missing type")
	}
}

func TestResolveNavType(t *testing.T) {
	tests := []struct {
		input    string
		typeName string
		isMany   bool
	}{
		{"Collection(NS.Order)", "Order", true},
		{"NS.Customer", "Customer", false},
		{"SimpleType", "SimpleType", false},
		{"Collection(SimpleType)", "SimpleType", true},
	}
	for _, tt := range tests {
		name, many := ResolveNavType(tt.input)
		if name != tt.typeName || many != tt.isMany {
			t.Errorf("ResolveNavType(%q) = (%q, %v), want (%q, %v)",
				tt.input, name, many, tt.typeName, tt.isMany)
		}
	}
}

func TestParseEdmx_AbstractAndOpenType(t *testing.T) {
	xml := `<?xml version="1.0"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="NS" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Base" Abstract="true" OpenType="true">
        <Property Name="ID" Type="Edm.Int64"/>
      </EntityType>
      <EntityType Name="Derived" BaseType="NS.Base">
        <Property Name="Extra" Type="Edm.String"/>
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	doc, err := ParseEdmx(xml)
	if err != nil {
		t.Fatal(err)
	}
	base := doc.Schemas[0].EntityTypes[0]
	if !base.IsAbstract {
		t.Error("expected IsAbstract=true")
	}
	if !base.IsOpen {
		t.Error("expected IsOpen=true")
	}
	derived := doc.Schemas[0].EntityTypes[1]
	if derived.BaseType != "NS.Base" {
		t.Errorf("expected BaseType NS.Base, got %q", derived.BaseType)
	}
}

// TestParseEdmx_ExternalAnnotations verifies that schema-level <Annotations Target="...">
// blocks (as used by Azure, SAP, and OData reference services like TripPin RW) are
// applied to the corresponding entity sets. CE6630 was caused by these annotations being
// silently ignored, leaving Insertable=nil and defaulting to false.
func TestParseEdmx_ExternalAnnotations(t *testing.T) {
	xmlStr := `<?xml version="1.0"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="NS" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Airline">
        <Key><PropertyRef Name="AirlineCode"/></Key>
        <Property Name="AirlineCode" Type="Edm.String"/>
        <Property Name="Name" Type="Edm.String"/>
      </EntityType>
      <EntityContainer Name="DefaultContainer">
        <EntitySet Name="Airlines" EntityType="NS.Airline"/>
      </EntityContainer>
      <Annotations Target="NS.DefaultContainer/Airlines">
        <Annotation Term="Org.OData.Capabilities.V1.InsertRestrictions">
          <Record>
            <PropertyValue Property="Insertable" Bool="true"/>
          </Record>
        </Annotation>
        <Annotation Term="Org.OData.Capabilities.V1.DeleteRestrictions">
          <Record>
            <PropertyValue Property="Deletable" Bool="false"/>
          </Record>
        </Annotation>
      </Annotations>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	doc, err := ParseEdmx(xmlStr)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.EntitySets) != 1 {
		t.Fatalf("expected 1 entity set, got %d", len(doc.EntitySets))
	}
	es := doc.EntitySets[0]
	if es.Insertable == nil || !*es.Insertable {
		t.Error("expected Insertable=true from external annotation")
	}
	if es.Deletable == nil || *es.Deletable {
		t.Error("expected Deletable=false from external annotation")
	}
	if es.Updatable != nil {
		t.Error("expected Updatable=nil (not specified)")
	}
}

// TestParseEdmx_ExternalAnnotations_WithoutSlash verifies that targets without a
// container prefix (e.g. just the entity set name) are also resolved correctly.
func TestParseEdmx_ExternalAnnotations_WithoutSlash(t *testing.T) {
	xmlStr := `<?xml version="1.0"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="NS" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Item">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int64"/>
      </EntityType>
      <EntityContainer Name="C">
        <EntitySet Name="Items" EntityType="NS.Item"/>
      </EntityContainer>
      <Annotations Target="Items">
        <Annotation Term="Org.OData.Capabilities.V1.UpdateRestrictions">
          <Record>
            <PropertyValue Property="Updatable" Bool="true"/>
          </Record>
        </Annotation>
      </Annotations>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	doc, err := ParseEdmx(xmlStr)
	if err != nil {
		t.Fatal(err)
	}
	es := doc.EntitySets[0]
	if es.Updatable == nil || !*es.Updatable {
		t.Error("expected Updatable=true from external annotation without slash prefix")
	}
}

// TestParseEdmx_ConcurrencyModeFixed verifies that a property with
// ConcurrencyMode="Fixed" (OData v3 ETag token) is treated as Computed=true
// so that mxcli does not mark it as Creatable in the generated external entity.
// Regression test for issue #525 (TripPin Airline.Concurrency).
func TestParseEdmx_ConcurrencyModeFixed(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="3.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="TripPin" xmlns="http://schemas.microsoft.com/ado/2009/11/edm">
      <EntityType Name="Airline">
        <Key><PropertyRef Name="AirlineCode" /></Key>
        <Property Name="AirlineCode" Type="Edm.String" Nullable="false" />
        <Property Name="Name"        Type="Edm.String" Nullable="false" />
        <Property Name="Concurrency" Type="Edm.Int64"  ConcurrencyMode="Fixed" />
      </EntityType>
      <EntityContainer Name="TripPinServiceRW" m:IsDefaultEntityContainer="true">
        <EntitySet Name="Airlines" EntityType="TripPin.Airline" />
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	doc, err := ParseEdmx(xml)
	if err != nil {
		t.Fatalf("ParseEdmx error: %v", err)
	}
	if len(doc.Schemas) == 0 || len(doc.Schemas[0].EntityTypes) == 0 {
		t.Fatal("expected at least one entity type")
	}
	et := doc.Schemas[0].EntityTypes[0]
	var concProp *EdmProperty
	for _, p := range et.Properties {
		if p.Name == "Concurrency" {
			concProp = p
			break
		}
	}
	if concProp == nil {
		t.Fatal("expected Concurrency property in parsed entity type")
	}
	if !concProp.Computed {
		t.Error("ConcurrencyMode='Fixed' must set Computed=true so the attribute is not marked Creatable (issue #525)")
	}
}
