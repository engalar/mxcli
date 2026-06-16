# JS Action BSON 修复调试记录

## 问题

mxbuild 11.6.6 对 `mxcli` 创建的 JavaScript Action 报 CE1613：
```
The selected JavaScript action parameter 'HD.JSA_CopyToClipboard.Text' no longer exists.
```

## 根因

gen codec 为 `JavaScriptAction` 生成的 BSON 使用**新版** `ActionParameters`/`ActionParameterType` key，但 mxbuild 11.6.6 **只读旧版** `Parameters`/`ParameterType` key。

### Studio Pro 11.6.6 生成的 BSON（可工作）

```json
{
  "Parameters": [2, {
    "$Type": "JavaScriptActions$JavaScriptActionParameter",
    "Name": "startDate",
    "ParameterType": {
      "$Type": "CodeActions$BasicParameterType",
      "Type": { "$Type": "CodeActions$DateTimeType" }
    }
  }]
}
```

### gen codec 生成的 BSON（mxbuild 报错）

```json
{
  "ActionParameters": [{
    "$Type": "JavaScriptActions$JavaScriptActionParameter",
    "Name": "startDate",
    "ActionParameterType": { "$Type": "CodeActions$DateTimeType" }
  }]
}
```

差异：
| 方面 | Studio Pro | gen codec |
|------|-----------|-----------|
| 参数列表 BSON key | `Parameters` | `ActionParameters` |
| 参数类型 BSON key | `ParameterType` | `ActionParameterType` |
| 类型包装 | `BasicParameterType` 包装 | 裸类型 |
| 列表版本标记 | `[2, ...]` | 无版本标记 |

## 修复步骤

### 1. `supplements.json` BSON key override

添加 `JavaScriptActionParameter.actionParameterType` → `ParameterType`：
```json
"JavaScriptActionParameter.actionParameterType": "ParameterType"
```

### 2. `supplements.json` extra_properties

给 `JavaScriptAction` 加旧版 `parameters`（PartList）字段：
```json
"JavaScriptAction": [{"name": "parameters", "kind": "PartList"}]
```

### 3. `supplements.json` part_list_version2_fields

添加版本标记 2：
```json
"JavaScriptAction.parameters"
```

### 4. 重新运行 codegen

```bash
go run ./cmd/modelsdk-codegen
```

### 5. 更新 `execCreateJavaScriptAction`

写参时用：
- `AddParameters(jsaParam)` 替代 `AddActionParameters(jsaParam)`
- `genCA.NewBasicParameterType()` 包装类型，调用 `SetActionParameterType(bpt)`
- 参数名首字母小写（匹配 Studio Pro 约定）

### 6. 更新 nanoflow builder

`addCallJavaScriptActionActionGen` 中 `SetParameterQualifiedName` 的参数名首字母小写：

```go
argName := strings.ToLower(arg.Name[:1]) + arg.Name[1:]
mapping.SetParameterQualifiedName(actionQN + "." + argName)
```

### 7. 更新 `describeJavaScriptActionGen`

describe 函数从 `ParametersItems()`（旧版）先读，再回退到 `ActionParametersItems()`（新版），并解码 `BasicParameterType` 包装：

```go
params := jsa.ParametersItems()
if len(params) == 0 {
    params = jsa.ActionParametersItems()
}
```

## 关键教训

1. **BSON key 版本差异** — Mendix 新版 SDK 用 `ActionParameters`、`ActionParameterType`，但 mxbuild 11.6.6 只读旧版 `Parameters`、`ParameterType`
2. **`BasicParameterType` 包装** — 旧版参数类型必须用 `BasicParameterType` 包装，新版可用裸类型
3. **参数名大小写** — Studio Pro 约定 JS Action 参数名首字母小写（`text` 非 `Text`），nanoflow 调用引用必须一致
4. **调试方法** — `mxcli bson dump --type javascriptaction --object "Module.Name"` 直接查看 BSON 结构对比
