# Examples

Complete fluent API examples for common tasks.

## Setup

All examples start with the same setup:

```go
package main

import (
    "log"
    "github.com/mendixlabs/mxcli/api"
    "github.com/mendixlabs/mxcli/sdk/mpr"
)

func main() {
    writer, err := mpr.OpenForWriting("/path/to/MyApp.mpr")
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    modelAPI := api.New(writer)
    module, _ := modelAPI.Modules.GetModule("Sales")
    modelAPI.SetModule(module)

    // ... examples below
}
```

## Create a Domain Model

```go
// Create entity with multiple attribute types
customer, _ := modelAPI.DomainModels.CreateEntity("Customer").
    Persistent().
    WithStringAttribute("Name", 200).
    WithStringAttribute("Email", 254).
    WithIntegerAttribute("Age").
    WithBooleanAttribute("IsActive").
    WithDateTimeAttribute("CreatedDate", true).
    Build()

// Create another entity
order, _ := modelAPI.DomainModels.CreateEntity("Order").
    Persistent().
    WithDecimalAttribute("TotalAmount").
    WithDateTimeAttribute("OrderDate", true).
    Build()

// Create an enumeration
_, _ = modelAPI.Enumerations.CreateEnumeration("OrderStatus").
    WithValue("Pending", "Pending").
    WithValue("Processing", "Processing").
    WithValue("Completed", "Completed").
    WithValue("Cancelled", "Cancelled").
    Build()

// Create association
_, _ = modelAPI.DomainModels.CreateAssociation("Customer_Orders").
    From("Customer").
    To("Order").
    OneToMany().
    Build()
```

## Create a Microflow

```go
_, _ = modelAPI.Microflows.CreateMicroflow("ACT_ProcessOrder").
    WithParameter("Order", "Sales.Order").
    WithStringParameter("Message").
    ReturnsBoolean().
    Build()
```

## Create a Page

```go
_, _ = modelAPI.Pages.CreatePage("CustomerOverview").
    WithTitle("Customer Overview").
    Build()
```

## Complete Module Setup

This example creates a full module with entities, enumerations, associations, and microflows:

```go
package main

import (
    "log"
    "github.com/mendixlabs/mxcli/api"
    "github.com/mendixlabs/mxcli/sdk/mpr"
)

func main() {
    writer, err := mpr.OpenForWriting("/path/to/MyApp.mpr")
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    modelAPI := api.New(writer)
    module, _ := modelAPI.Modules.GetModule("ProductCatalog")
    modelAPI.SetModule(module)

    // Enumeration
    _, _ = modelAPI.Enumerations.CreateEnumeration("ProductCategory").
        WithValue("Electronics", "Electronics").
        WithValue("Clothing", "Clothing").
        WithValue("HomeGarden", "Home & Garden").
        Build()

    // Entities
    _, _ = modelAPI.DomainModels.CreateEntity("Product").
        Persistent().
        WithStringAttribute("SKU", 50).
        WithStringAttribute("Name", 200).
        WithDecimalAttribute("Price").
        WithIntegerAttribute("StockQuantity").
        WithBooleanAttribute("IsActive").
        WithEnumerationAttribute("Category", "ProductCatalog.ProductCategory").
        Build()

    _, _ = modelAPI.DomainModels.CreateEntity("Supplier").
        Persistent().
        WithStringAttribute("CompanyName", 200).
        WithStringAttribute("ContactEmail", 254).
        Build()

    // Association
    _, _ = modelAPI.DomainModels.CreateAssociation("Product_Supplier").
        From("Product").
        To("Supplier").
        OneToMany().
        Build()

    // Microflow
    _, _ = modelAPI.Microflows.CreateMicroflow("ValidateProduct").
        WithParameter("Product", "ProductCatalog.Product").
        ReturnsBoolean().
        Build()
}
```

## Comparison: Fluent API vs Direct SDK

The fluent API reduces boilerplate. Here is the same entity creation using both approaches:

### Fluent API

```go
entity, _ := modelAPI.DomainModels.CreateEntity("Customer").
    Persistent().
    WithStringAttribute("Name", 200).
    WithIntegerAttribute("Age").
    Build()
```

### Direct SDK

```go
import (
    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/sdk/domainmodel"
    modelsdk "github.com/mendixlabs/mxcli"
)

entity := &domainmodel.Entity{
    BaseElement: model.BaseElement{
        ID: model.ID(modelsdk.GenerateID()),
    },
    Name:        "Customer",
    Persistable: true,
    Attributes: []*domainmodel.Attribute{
        {
            BaseElement: model.BaseElement{ID: model.ID(modelsdk.GenerateID())},
            Name:        "Name",
            Type:        &domainmodel.StringAttributeType{Length: 200},
        },
        {
            BaseElement: model.BaseElement{ID: model.ID(modelsdk.GenerateID())},
            Name:        "Age",
            Type:        &domainmodel.IntegerAttributeType{},
        },
    },
}

err := writer.CreateEntity(domainModelID, entity)
```

The fluent API handles ID generation, domain model lookup, and module context automatically.
