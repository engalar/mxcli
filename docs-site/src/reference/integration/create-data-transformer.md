# CREATE DATA TRANSFORMER

## Synopsis

```sql
CREATE [ OR MODIFY ] DATA TRANSFORMER module.Name
    SOURCE { JSON | XML } 'sample'
{
    { JSLT | XSLT } 'transformation';
    [ { JSLT | XSLT } 'next_step'; ... ]
};
```

## Description

Creates a data transformer that reshapes JSON or XML data through one or more transformation steps before it enters an import mapping or is returned to a caller.

A data transformer applies a pipeline of JSLT (for JSON) or XSLT (for XML) transformations to its input. Steps execute in order; each step's output becomes the next step's input.

If `OR MODIFY` is specified and the transformer already exists, it is updated in place. The document UUID is preserved.

### JSLT

JSLT is a JSON transformation language designed for structural reshaping. Common operations:

- Field selection: `.field`
- Rename: `"newName": .oldName`
- Arithmetic: `.price * .quantity`
- Conditional: `if (.status == "active") "yes" else "no"`

### Source Sample

The `SOURCE` clause provides a representative input document. Mendix uses it to validate the transformation at design time. The actual runtime input may differ in values but must match the structure.

## Parameters

`module.Name`
:   The qualified name of the data transformer.

`SOURCE JSON 'sample'`
:   A representative JSON input document. Multi-line samples can use `$$...$$` quoting.

`SOURCE XML 'sample'`
:   A representative XML input document.

`JSLT 'transformation'`
:   A JSLT transformation expression applied to the current input.

`XSLT 'transformation'`
:   An XSLT stylesheet applied to the current input.

## Examples

### Single-step JSON flattening

```sql
CREATE DATA TRANSFORMER MyModule.FlattenNested
    SOURCE JSON '{"wrapper": {"value": 42, "label": "hello"}}'
{
    JSLT '{"value": .wrapper.value, "label": .wrapper.label}';
};
```

### Multi-line JSLT with $$ quoting

```sql
CREATE DATA TRANSFORMER MyModule.WeatherTransform
    SOURCE JSON '{"latitude": 51.9, "longitude": 4.5, "current": {"temperature_2m": 12.8, "wind_speed_10m": 18.3}}'
{
    JSLT $$
{
    "lat":        .latitude,
    "lon":        .longitude,
    "temp":       .current.temperature_2m,
    "wind_speed": .current.wind_speed_10m
}
    $$;
};
```

### Idempotent update

```sql
CREATE OR MODIFY DATA TRANSFORMER MyModule.WeatherTransform
    SOURCE JSON '{"latitude": 51.9, "longitude": 4.5, "timezone": "UTC", "current": {"temperature_2m": 12.8}}'
{
    JSLT $$
{
    "lat":      .latitude,
    "lon":      .longitude,
    "timezone": .timezone,
    "temp":     .current.temperature_2m
}
    $$;
};
```

## See Also

[CREATE JSON STRUCTURE](create-json-structure.md), [CREATE IMPORT MAPPING](create-import-mapping.md)
