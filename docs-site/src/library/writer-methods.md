# Writer Methods

Complete list of methods available on the writer returned by `modelsdk.OpenForWriting()`.

## Accessing the Reader

The writer embeds a reader for querying the project:

```go
writer, _ := modelsdk.OpenForWriting("/path/to/MyApp.mpr")
defer writer.Close()

reader := writer.Reader()
modules, _ := reader.ListModules()
```

## Modules

```go
writer.CreateModule(module)      // error -- create a new module
writer.UpdateModule(module)      // error -- update module metadata
writer.DeleteModule(id)          // error -- delete a module and its contents
```

## Entities

```go
writer.CreateEntity(domainModelID, entity)       // error -- add entity to domain model
writer.UpdateEntity(domainModelID, entity)       // error -- update an existing entity
writer.DeleteEntity(domainModelID, entityID)     // error -- remove entity from domain model
```

## Attributes

```go
writer.AddAttribute(domainModelID, entityID, attribute)  // error -- add attribute to entity
```

## Associations

```go
writer.CreateAssociation(domainModelID, association)     // error -- create association
writer.DeleteAssociation(domainModelID, associationID)   // error -- remove association
```

## Microflows and Nanoflows

```go
writer.CreateMicroflow(microflow)    // error -- create a new microflow
writer.UpdateMicroflow(microflow)    // error -- update an existing microflow
writer.DeleteMicroflow(id)           // error -- delete a microflow

writer.CreateNanoflow(nanoflow)      // error -- create a new nanoflow
writer.UpdateNanoflow(nanoflow)      // error -- update an existing nanoflow
writer.DeleteNanoflow(id)            // error -- delete a nanoflow
```

## Pages and Layouts

```go
writer.CreatePage(page)              // error -- create a new page
writer.UpdatePage(page)              // error -- update an existing page
writer.DeletePage(id)                // error -- delete a page

writer.CreateLayout(layout)          // error -- create a new layout
writer.UpdateLayout(layout)          // error -- update an existing layout
writer.DeleteLayout(id)              // error -- delete a layout
```

## Enumerations and Constants

```go
writer.CreateEnumeration(enumeration)    // error -- create a new enumeration
writer.CreateConstant(constant)          // error -- create a new constant
```

## Helper Functions

The `modelsdk` package provides helper functions for creating common types:

```go
// Entities
modelsdk.NewEntity(name)                          // *Entity -- persistent entity
modelsdk.NewNonPersistableEntity(name)            // *Entity -- non-persistent entity

// Attributes
modelsdk.NewStringAttribute(name, length)         // *Attribute
modelsdk.NewIntegerAttribute(name)                // *Attribute
modelsdk.NewDecimalAttribute(name)                // *Attribute
modelsdk.NewBooleanAttribute(name)                // *Attribute
modelsdk.NewDateTimeAttribute(name, localize)     // *Attribute
modelsdk.NewEnumerationAttribute(name, enumID)    // *Attribute
modelsdk.NewAutoNumberAttribute(name)             // *Attribute

// Associations
modelsdk.NewAssociation(name, parentID, childID)          // *Association -- Reference (1:N)
modelsdk.NewReferenceSetAssociation(name, parentID, childID)  // *Association -- ReferenceSet (M:N)

// Flows
modelsdk.NewMicroflow(name)     // *Microflow
modelsdk.NewNanoflow(name)      // *Nanoflow

// Pages
modelsdk.NewPage(name)          // *Page

// IDs
modelsdk.GenerateID()           // string -- new UUID v4
```

## Example: Create an Entity with Attributes

```go
writer, _ := modelsdk.OpenForWriting("/path/to/MyApp.mpr")
defer writer.Close()

reader := writer.Reader()
modules, _ := reader.ListModules()
dm, _ := reader.GetDomainModel(modules[0].ID)

// Create entity
customer := modelsdk.NewEntity("Customer")
customer.Documentation = "Customer master data"
customer.Location = model.Point{X: 100, Y: 200}
writer.CreateEntity(dm.ID, customer)

// Add attributes
writer.AddAttribute(dm.ID, customer.ID, modelsdk.NewStringAttribute("Name", 200))
writer.AddAttribute(dm.ID, customer.ID, modelsdk.NewStringAttribute("Email", 254))
writer.AddAttribute(dm.ID, customer.ID, modelsdk.NewBooleanAttribute("IsActive"))
writer.AddAttribute(dm.ID, customer.ID, modelsdk.NewDateTimeAttribute("CreatedDate", true))

// Create association
order := modelsdk.NewEntity("Order")
writer.CreateEntity(dm.ID, order)

assoc := modelsdk.NewAssociation("Customer_Order", customer.ID, order.ID)
writer.CreateAssociation(dm.ID, assoc)
```

> **Warning:** Always back up your `.mpr` file before modifying it. The writer operates directly on the SQLite database.
