# Proposal: Microflow Inheritance Split And Cast Statements

Status: Draft

## Summary

Add round-trip MDL support for type-based microflow decisions and cast actions:

```mdl
split type $Input
case Sample.SpecializedInput
  cast $SpecificInput;
else
  return false;
end split;
```

## Motivation

Studio Pro represents specialization/type decisions as `InheritanceSplit` objects and stores downcasts as `CastAction` activities. Without first-class MDL statements, `describe` can only emit unsupported comments or incomplete split output, and `exec` cannot rebuild the same graph.

## Semantics

`split type $Var` evaluates the runtime specialization of an object variable. Each `case Module.Entity` branch corresponds to an outgoing sequence flow with an `InheritanceCase`. The optional `else` branch maps to the outgoing flow without an inheritance case.

`cast $Output` emits a `CastAction` that produces the downcast variable. `$Output = cast $Input` is accepted for source-preserving authoring, but current Mendix BSON stores the generated cast variable as the primary persisted field.

## Tests And Examples

`mdl-examples/doctype-tests/inheritance_split_statement.test.mdl` demonstrates the syntax. Go regression tests cover parser construction, builder output, describer output, validation recursion, and BSON writer support for inheritance case values and cast actions.

## Open Questions

- Should `exec` validate `case Module.Entity` against the project's specialization hierarchy when connected?
- Should the source-preserving `$Output = cast $Input` form round-trip both variable names once the underlying BSON fields are confirmed for all supported Mendix versions?
