# Project Tree

The VS Code extension provides a **Mendix Project Tree** view that mirrors the App Explorer in Mendix Studio Pro, allowing you to browse modules and documents without leaving VS Code.

## Accessing the Tree

The project tree appears in the **Explorer** sidebar under the "Mendix Project" section. If it does not appear, check that:

1. The extension is installed and enabled
2. A `.mpr` file is present in the workspace (or configured via `mdl.mprPath`)
3. The `mxcli` binary is accessible

## Tree Structure

The tree is organized by module, with documents grouped by type:

```
Mendix Project
├── MyFirstModule
│   ├── Domain Model
│   │   ├── Customer
│   │   ├── Order
│   │   └── Product
│   ├── Microflows
│   │   ├── ACT_Customer_Create
│   │   ├── ACT_Customer_Delete
│   │   └── ProcessOrder
│   ├── Pages
│   │   ├── Customer_Overview
│   │   ├── Customer_NewEdit
│   │   └── Order_Overview
│   ├── Snippets
│   │   └── NavigationMenu
│   ├── Enumerations
│   │   └── OrderStatus
│   └── Nanoflows
│       └── NF_ValidateInput
├── Administration
│   └── ...
└── System
    └── ...
```

## Interactions

- **Click** a document to open its MDL source as a virtual read-only document (equivalent to `DESCRIBE` in the REPL)
- **Expand** a module to see all document types
- **Expand** a document type to see individual documents

## Refreshing

The tree refreshes automatically when the project file changes. You can also trigger a manual refresh from the tree view toolbar.

## Relationship to SHOW Commands

The tree is backed by the same metadata that powers the `SHOW` commands:

| Tree Level | Equivalent MDL Command |
|-----------|----------------------|
| Module list | `SHOW MODULES` |
| Entities in a module | `SHOW ENTITIES IN MyModule` |
| Microflows in a module | `SHOW MICROFLOWS IN MyModule` |
| Pages in a module | `SHOW PAGES IN MyModule` |
| Full structure | `SHOW STRUCTURE IN MyModule DEPTH 2` |
