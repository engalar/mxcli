# CREATE MODEL

## Synopsis

```sql
CREATE [ OR MODIFY ] MODEL module.Name (
    Provider: MxCloudGenAI,
    key: module.KeyConstant
    [, DisplayName: 'display name' ]
    [, KeyName: 'portal key name' ]
    [, Environment: 'environment' ]
);

DROP MODEL module.Name
```

Requires Mendix 11.9+.

## Description

Creates an agent-editor Model document. A model represents an LLM configuration in the Mendix Agent Editor. It references a String constant that holds the Mendix Cloud GenAI Portal resource key.

At runtime, the `ASU_AgentEditor` after-startup microflow reads the constant value and registers the corresponding `GenAICommons.DeployedModel`, making the model available to agents.

`DisplayName`, `KeyName`, and `Environment` are normally populated by Studio Pro when the user clicks "Test Key" against the Portal. They can be specified in MDL for round-trip preservation (when `DESCRIBE` output is re-executed), but setting them manually has no functional effect at runtime.

If `OR MODIFY` is specified and the model already exists, its properties are updated in place. The document UUID is preserved.

## Parameters

`module.Name`
:   The qualified name of the model document.

`Provider: MxCloudGenAI`
:   The provider. `MxCloudGenAI` is the only supported value in current Mendix versions.

`key: module.KeyConstant`
:   A qualified reference to a String constant that holds the Portal resource key. The constant must exist before the model is created.

`DisplayName: 'display name'`
:   Optional. The display name shown in Studio Pro. Populated by Studio Pro after key validation.

`KeyName: 'portal key name'`
:   Optional. The Portal key name. Populated by Studio Pro after key validation.

`Environment: 'environment'`
:   Optional. The deployment environment. Populated by Studio Pro after key validation.

## Examples

### Minimal model

```sql
CREATE CONSTANT MyModule."ModelKey"
    TYPE String
    DEFAULT '';
/

CREATE MODEL MyModule."GPT4Model" (
    Provider: MxCloudGenAI,
    key: MyModule.ModelKey
);
/
```

### Model with Portal metadata (for round-trip scripts)

```sql
CREATE MODEL MyModule."ConfiguredModel" (
    Provider: MxCloudGenAI,
    key: MyModule.ModelKey,
    DisplayName: 'GPT-4 Turbo (128K)',
    KeyName: 'prod-gpt4-turbo',
    Environment: 'production'
);
/
```

### Idempotent upsert

```sql
CREATE OR MODIFY MODEL MyModule."GPT4Model" (
    Provider: MxCloudGenAI,
    key: MyModule.ModelKey,
    DisplayName: 'GPT-4 Turbo (Updated)'
);
/
```

### Cleanup

```sql
DROP MODEL MyModule.GPT4Model;
/
```

## See Also

[CREATE AGENT](create-agent.md), [CREATE KNOWLEDGE BASE](create-knowledge-base.md)
