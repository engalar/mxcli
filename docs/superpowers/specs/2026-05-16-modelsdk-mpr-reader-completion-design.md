# modelsdk/mpr.Reader 补全设计文档（PR5 正确路径）

**日期：** 2026-05-16
**状态：** 已批准，待实施
**背景：** 现有 Bridge/Alias/双路径模式均为 workaround，根因是 modelsdk/mpr.Reader 不完整。本文档定义消除所有 workaround 的正确设计。

---

## 问题陈述

当前存在三类 workaround：

| Workaround | 文件 | 根因 |
|---|---|---|
| type sdkReader = sdkmpr.Reader | sdkmpr_bridge.go | modelsdk/mpr.Reader 方法不全 |
| Bridge 函数（sdkSerialize*, sdkPatch*） | sdkmpr_bridge.go, repos/sdk_bridge.go | modelsdk/mpr 缺少这些纯函数 |
| Interface 适配（bsonReader, projectTreeReader） | bson_reader_bridge.go | modelsdk/mpr.Reader 不满足这些接口 |

正确解法：补全 modelsdk/mpr.Reader，使之成为 sdk/mpr.Reader 的真正替代。

---

## 架构目标

### 终态层次

```
executor → backend interface（model.* 类型，暂不动）
              → MprBackend.reader: *modelsdkmpr.Reader   ← 唯一 reader
              → gen/* 类型 → backend 实现层按需转 model.*
              （所有 bridge 文件已删除，sdk/mpr 目录已删除）
```

### 设计原则

1. **modelsdk/mpr.Reader 返回 gen/* 类型**（不是 model.* 类型）
2. **backend.go 实现层**做薄转换（gen/* → model.*），仅针对接口要求 model.* 的方法
3. **executor 层 0 改动**，backend 接口定义 0 改动
4. **全部 bridge 文件删除**

---

## 48 个缺失方法的实现策略

### 标准实现模式（已验证）

```go
func (r *Reader) ListEnumerationsGen() ([]*genEnum.Enumeration, error) {
    units, err := r.listUnitsByType("Enumerations$Enumeration")
    if err != nil { return nil, err }
    var result []*genEnum.Enumeration
    for _, u := range units {
        contents, err := r.resolveContents(u.ID, u.Contents)
        if err != nil { continue }
        obj, err := r.decoder.Decode(bson.Raw(contents))
        if err != nil { continue }
        e, ok := obj.(*genEnum.Enumeration)
        if !ok { continue }
        e.SetID(element.ID(u.ID))
        result = append(result, e)
    }
    return result, nil
}
```

### 方法分组

| 分组 | 方法数 | gen 类型来源 | 难度 |
|---|---|---|---|
| 简单列表（Enum/Const/ScheduledEvent） | 6 | gen/enumerations, gen/constants, gen/scheduledevents | 低 |
| 模块与文件夹（完整 Module） | 3 | gen/projects | 低 |
| Mapping/JsonStructure | 6 | gen/importmappings, gen/exportmappings | 低 |
| Web Services（OData/REST） | 6 | gen/webservices, gen/odatapublish, gen/rest | 低 |
| Navigation/Settings/Security | 7 | gen/navigation, gen/settings, gen/security | 中 |
| Business/Data Services | 4 | gen/businessevents, gen/datatransformers, gen/databaseconnector | 中 |
| AgentEditor | 4 | mdl/types（customblob 模式） | 中 |
| Raw/Widget/JavaScriptAction | 12 | types.RawUnitInfo, types.RawCustomWidgetType 等 | 高 |

### backend.go 转换层设计

只有接口要求 model.* 但 reader 返回 gen/* 的方法需要薄转换：

```go
func (b *MprBackend) ListModules() ([]*model.Module, error) {
    genMods, err := b.reader.ListModules() // 返回 []*genProjects.Module
    if err != nil { return nil, err }
    result := make([]*model.Module, len(genMods))
    for i, gm := range genMods {
        result[i] = &model.Module{
            BaseElement: model.BaseElement{ID: model.ID(gm.ID())},
            Name: gm.Name(),
        }
    }
    return result, nil
}
```

已经返回 gen 类型的 10 个方法（ListDomainModelsGen 等）不需要转换，直接透传。

---

## 实施序列（严格有序）

### Phase 1：补全 modelsdk/mpr.Reader（核心工作）

实现全部 48 个缺失方法，按难度从低到高：

1. **简单列表组**（Enum/Const/ScheduledEvent/Module/Folder）— 约 9 个
2. **Mapping/Web Services 组**— 约 12 个  
3. **Navigation/Settings/Security 组**— 约 7 个
4. **Business/Data Services 组**— 约 4 个
5. **AgentEditor 组**— 约 4 个
6. **Raw/Widget/JavaScriptAction 组**— 约 12 个

文件组织：每个逻辑分组创建独立文件（如 `reader_enumerations.go`、`reader_mappings.go`）

验证：`go build ./... && go test ./...` 全绿

### Phase 2：切换 sdkReader 类型别名

```go
// sdkmpr_bridge.go 中
// Before:
type sdkReader = sdkmpr.Reader
func sdkOpenReader(path string) (*sdkReader, error) {
    return sdkmpr.OpenWithOptions(path, sdkmpr.OpenOptions{ReadOnly: false})
}

// After:
type sdkReader = modelsdkmpr.Reader
func sdkOpenReader(path string) (*sdkReader, error) {
    return modelsdkmpr.Open(path)
}
```

backend.go 中 `b.reader.*` 调用的返回类型改为 gen/*，加转换层。

验证：编译通过，测试全绿

### Phase 3：删除 bridge 文件

```bash
git rm mdl/backend/mpr/sdkmpr_bridge.go
git rm mdl/backend/mpr/repos/sdk_bridge.go
git rm cmd/mxcli/bson_reader_bridge.go
```

将 bsonReader/widgetReader/projectTreeReader 接口中的方法改为直接调用 reader 方法。

### Phase 4：删除 sdk/mpr

```bash
git rm -r sdk/mpr/
```

### Phase 5：清理善后

- 删除 modelsdk.go 注释中的 sdk/mpr 残留引用
- 更新 CLAUDE.md 架构描述
- 更新 migrate-sdk-to-modelsdk SKILL.md

---

## 终态验证标准

```bash
# 1. sdk/mpr 目录不存在
ls sdk/mpr/  # 期望：No such file or directory

# 2. 全局零 sdk/mpr import
grep -r '"github.com/mendixlabs/mxcli/sdk/mpr"' . --include="*.go"  # 期望：空

# 3. 全量编译
go build ./...  # 期望：无错误

# 4. 全量测试
go test ./...   # 期望：全 ok
```

---

## 不动的边界

- **executor 层 0 改动**：ctx.Backend.* 调用不受影响
- **backend 接口 0 改动**：mdl/backend/*.go 接口定义保持 model.* 类型
- **AST / visitor 层 0 改动**
- **model.ID / model.Point 等基础类型保留**

## 不在本设计范围内

- 将 backend 接口从 model.* 改为 gen.*（Stage 5 独立工作）
- 删除 model/* 包（另行设计）
- ~~sdk/versions 包~~ ✅ 已迁移至 modelsdk/version
