# TypeScript SDK Equivalence

Mapping between the official Mendix Model SDK (TypeScript) and MDL/modelsdk-go equivalents.

## Overview

The [Mendix Model SDK](https://docs.mendix.com/apidocs-mxsdk/mxsdk/) is Mendix's official TypeScript library for programmatic model manipulation. It works through the Mendix Platform API and requires cloud connectivity. MDL and modelsdk-go provide similar capabilities but operate directly on local `.mpr` files.

## Feature Comparison

| Feature | Mendix Model SDK (TypeScript) | MDL / modelsdk-go |
|---------|-------------------------------|-------------------|
| Language | TypeScript / JavaScript | Go (library), MDL (DSL) |
| Runtime | Node.js | Native binary |
| Cloud required | Yes (Platform API) | No (local files) |
| Authentication | API key + PAT | None |
| Real-time collaboration | Yes | No |
| Read operations | Yes | Yes |
| Write operations | Yes | Yes |
| Type safety | TypeScript types | Go types |
| CLI tool | No | Yes (mxcli) |
| SQL-like DSL | No | Yes (MDL) |
| AI assistant integration | Limited | Built-in (multi-tool) |
| Full-text search | No | Yes (FTS5) |
| Cross-reference analysis | No | Yes (callers, impact) |
| Linting | No | Yes (Go + Starlark rules) |

## Operation Mapping

### Reading

| TypeScript SDK | MDL Equivalent | Go Library |
|---------------|----------------|------------|
| `model.allDomainModels()` | `SHOW ENTITIES` | `reader.ListDomainModels()` |
| `domainModel.entities` | `SHOW ENTITIES IN Module` | `reader.GetDomainModel(id)` |
| `entity.attributes` | `DESCRIBE ENTITY Module.Name` | `dm.Entities[i].Attributes` |
| `model.allMicroflows()` | `SHOW MICROFLOWS` | `reader.ListMicroflows()` |
| `model.allPages()` | `SHOW PAGES` | `reader.ListPages()` |
| `model.allEnumerations()` | `SHOW ENUMERATIONS` | `reader.ListEnumerations()` |

### Writing

| TypeScript SDK | MDL Equivalent | Go Library |
|---------------|----------------|------------|
| `domainmodels.Entity.createIn(dm)` | `CREATE PERSISTENT ENTITY Module.Name (...)` | `writer.CreateEntity(dmID, entity)` |
| `entity.name = "Foo"` | Part of `CREATE ENTITY` | `entity.Name = "Foo"` |
| `domainmodels.Attribute.createIn(entity)` | Column in entity definition | `writer.AddAttribute(dmID, entityID, attr)` |
| `domainmodels.Association.createIn(dm)` | `CREATE ASSOCIATION` | `writer.CreateAssociation(dmID, assoc)` |
| `microflows.Microflow.createIn(folder)` | `CREATE MICROFLOW` | `writer.CreateMicroflow(mf)` |
| `pages.Page.createIn(folder)` | `CREATE PAGE` | `writer.CreatePage(page)` |
| `enumerations.Enumeration.createIn(module)` | `CREATE ENUMERATION` | `writer.CreateEnumeration(enum)` |

### Security

| TypeScript SDK | MDL Equivalent |
|---------------|----------------|
| `security.ModuleRole.createIn(module)` | `CREATE MODULE ROLE Module.RoleName` |
| Manual access rule construction | `GRANT role ON Entity (permissions)` |
| Manual user role creation | `CREATE USER ROLE Name (ModuleRoles)` |

## Workflow Comparison

### TypeScript SDK Workflow

```typescript
const client = new MendixPlatformClient();
const app = await client.getApp("app-id");
const workingCopy = await app.createTemporaryWorkingCopy("main");
const model = await workingCopy.openModel();

const dm = model.allDomainModels().filter(d => d.containerAsModule.name === "Sales")[0];
await dm.load();

const entity = domainmodels.Entity.createIn(dm);
entity.name = "Customer";

const attr = domainmodels.Attribute.createIn(entity);
attr.name = "Name";
attr.type = domainmodels.StringAttributeType.create(model);

await model.flushChanges();
await workingCopy.commitToRepository("main");
```

### MDL Equivalent

```sql
CREATE PERSISTENT ENTITY Sales.Customer (
    Name: String(200)
);
```

### Go Library Equivalent

```go
reader, _ := modelsdk.Open("app.mpr")
modules, _ := reader.ListModules()
dm, _ := reader.GetDomainModel(modules[0].ID)

writer, _ := modelsdk.OpenForWriting("app.mpr")
entity := modelsdk.NewEntity("Customer")
entity.Attributes = append(entity.Attributes,
    modelsdk.NewStringAttribute("Name", 200))
writer.CreateEntity(dm.ID, entity)
```

## Key Differences

1. **No cloud dependency** -- modelsdk-go works entirely offline with local `.mpr` files
2. **No working copy management** -- changes are written directly to the file (back up first)
3. **MDL is declarative** -- one statement replaces multiple imperative SDK calls
4. **Token efficiency** -- MDL uses 5-10x fewer tokens than equivalent JSON model representations, making it better suited for AI assistant context windows
5. **Built-in search and analysis** -- full-text search, cross-reference tracking, and impact analysis are not available in the TypeScript SDK
