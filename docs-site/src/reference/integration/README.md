# Integration Statements

Statements for managing JSON structures, import mappings, export mappings, and data transformers.

These document types support moving data between Mendix entities and external data formats (JSON, XML). They are typically used together: a JSON structure defines the external shape, import and export mappings translate between that shape and Mendix entities, and data transformers reshape raw data before it enters a mapping.

## Statements

| Statement | Description |
|-----------|-------------|
| [CREATE JSON STRUCTURE](create-json-structure.md) | Define the shape of a JSON document from a sample snippet |
| [CREATE IMPORT MAPPING](create-import-mapping.md) | Map incoming JSON/XML to Mendix entities |
| [CREATE EXPORT MAPPING](create-export-mapping.md) | Map Mendix entities to outgoing JSON/XML |
| [CREATE DATA TRANSFORMER](create-data-transformer.md) | Transform raw JSON or XML with JSLT/XSLT steps |

## Related Show/Describe Statements

| Statement | Syntax |
|-----------|--------|
| List JSON structures | `LIST JSON STRUCTURES [IN module]` |
| List import mappings | `LIST IMPORT MAPPINGS [IN module]` or `SHOW IMPORT MAPPINGS [IN module]` |
| List export mappings | `LIST EXPORT MAPPINGS [IN module]` or `SHOW EXPORT MAPPINGS [IN module]` |
| List data transformers | `LIST DATA TRANSFORMERS [IN module]` |
| Describe JSON structure | `DESCRIBE JSON STRUCTURE module.Name` |
| Describe import mapping | `DESCRIBE IMPORT MAPPING module.Name` |
| Describe export mapping | `DESCRIBE EXPORT MAPPING module.Name` |
| Describe data transformer | `DESCRIBE DATA TRANSFORMER module.Name` |

## Drop Statements

```sql
DROP JSON STRUCTURE module.Name;
DROP IMPORT MAPPING module.Name;
DROP EXPORT MAPPING module.Name;
DROP DATA TRANSFORMER module.Name;
```
