# Migrate sdk/mpr Write Path to modelsdk

将 `sdk/mpr.Writer` 的写路径迁移到 `modelsdk/mpr.WriteTransaction` 的完整指南。每次有新 backend write 方法需要实现或修复时使用。

## When to Use

- 实现新的 `MprBackend` 写方法
- 修复现有写方法报 **SQLITE_READONLY_DBMOVED (1544)** 错误
- 遇到写操作原子性问题（文件已写但 ContentsHash 未更新）
- 用户要求"用 modelsdk 替换旧 sdk"

## Why Migrate

`sdk/mpr.Writer.updateUnit()` v2 分支末尾调用 `updateTransactionID()`：

```go
// sdk/mpr/writer_units.go
_, err := w.reader.db.Exec(`UPDATE _Transaction SET LastTransactionID = ?`, newID)
return err  // 硬链接 MPR 文件返回 SQLITE_READONLY_DBMOVED (1544)
```

`modelsdk/mpr.WriteTransaction` 完全不写 `_Transaction` 表，且使用 temp→rename 两阶段写，崩溃安全。两个 Writer 共享同一个 `*sql.DB`（通过 `NewWriterFromDB` 注入），不存在双连接竞争。

---

## 关键决策：先判断元素类型

**迁移前必须先判断目标元素在 MPR 中是"独立 Unit"还是"DM 嵌入对象"。**

| 元素类型 | Unit 表中有独立行？ | 正确模式 |
|---------|-----------------|---------|
| Microflow / Nanoflow / Page / Layout / Snippet / Workflow | ✅ 是 | 模式 A 或 B |
| Module / ModuleSettings / Folder | ✅ 是 | 模式 A 或 B |
| Enumeration / Constant / JavaAction / ImageCollection | ✅ 是 | 模式 A 或 B |
| Security$ProjectSecurity / Security$ModuleSecurity | ✅ 是 | 模式 A |
| **Entity / Attribute / Association / CrossAssociation** | ❌ 否（嵌入 DomainModel BSON） | **模式 C** |

错误地对嵌入对象调用 `DeleteUnit(entityID)` 会返回 `"unit not found in database"`。

---

## 模式 A：msdkWrite（有 gen 覆盖的顶层 Unit）

适用于安全设置、模块、常量等——modelsdk 已生成对应 Go 类型。
`msdkWrite` 辅助函数已封装读取-解码-变更-编码-写入全流程：

```go
// mdl/backend/mpr/security_project_modelsdk.go
func (b *MprBackend) setXxxViaModelsdk(unitID model.ID, value string) error {
    return b.msdkWrite(unitID, func(elem element.Element) error {
        typed, ok := elem.(*msdkxxx.YourType)
        if !ok {
            return fmt.Errorf("unexpected type %T (want *YourType)", elem)
        }
        typed.SetXxx(value)
        return nil
    })
}
```

`msdkWrite` 内部流程（`security_project_modelsdk.go:20`）：
1. `msdkWriter.Reader().GetRawUnitBytes(unitID)` — 读原始 BSON
2. `codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))` — 解码为 typed element
3. 调用 `mutateFn(elem)` — 变更（SetXxx 自动标脏）
4. `(&codec.Encoder{}).Encode(elem)` — 只重新序列化变更字段
5. `BeginWriteTransaction().WriteUnit().Commit()` — 原子写入
6. `b.reader.InvalidateCache()` — 使 sdk/mpr reader 缓存失效

**imports 示例：**

```go
import (
    "fmt"
    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/modelsdk/element"
    msdkxxx "github.com/mendixlabs/mxcli/modelsdk/gen/<domain>"
)
```

---

## 模式 B：msdkWriteRaw（使用旧 sdk/mpr 序列化器）

适用于 Microflow、Page、DomainModel 等——旧序列化器已有完整 BSON 构建逻辑，
无需迁移序列化，只替换写路径（绕过 `updateTransactionID`）：

```go
// mdl/backend/mpr/mf_page_modelsdk.go
func (b *MprBackend) updateMicroflowViaModelsdk(mf *microflows.Microflow) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    contents, err := b.writer.SerializeMicroflow(mf)
    if err != nil {
        return fmt.Errorf("serialize microflow: %w", err)
    }
    return b.msdkWriteRaw(mf.ID, contents)
}
```

`msdkWriteRaw` 辅助函数（`security_allowed_roles_modelsdk.go:81`）：
直接接受 `[]byte`，用 `WriteTransaction` 写入后使两个 reader 缓存失效。

**创建（Insert）变体：**

```go
func (b *MprBackend) createMicroflowViaModelsdk(mf *microflows.Microflow) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    if mf.ID == "" {
        mf.ID = model.ID(modelsdkmpr.GenerateID())
    }
    mf.TypeName = "Microflows$Microflow"
    contents, err := b.writer.SerializeMicroflow(mf)
    if err != nil {
        return fmt.Errorf("serialize microflow: %w", err)
    }
    return b.msdkWriter.InsertUnit(
        string(mf.ID), string(mf.ContainerID),
        "Documents", "Microflows$Microflow", contents,
    )
}
```

**删除 / 移动（顶层 Unit 才可用）：**

```go
func (b *MprBackend) deleteMicroflowViaModelsdk(id model.ID) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveMicroflowViaModelsdk(mf *microflows.Microflow) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    return b.msdkWriter.UpdateUnitContainer(string(mf.ID), string(mf.ContainerID))
}
```

---

## 模式 C：writeDomainModel（DM 嵌入元素）

**Entity / Attribute / Association / CrossAssociation 必须使用此模式。**
这些元素没有独立 Unit 行——它们是 DomainModel BSON 内的数组元素。
操作时需读取整个 DM、在内存中修改切片、再整体回写：

```go
// mdl/backend/mpr/domainmodel_modelsdk.go
func (b *MprBackend) deleteEntityViaModelsdk(domainModelID, entityID model.ID) error {
    return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
        for i, e := range dm.Entities {
            if e.ID == entityID {
                dm.Entities = append(dm.Entities[:i], dm.Entities[i+1:]...)
                // 同步清理同 DM 内引用该实体的关联
                var kept []*domainmodel.Association
                for _, a := range dm.Associations {
                    if a.ParentID != entityID && a.ChildID != entityID {
                        kept = append(kept, a)
                    }
                }
                dm.Associations = kept
                return nil
            }
        }
        return fmt.Errorf("entity not found: %s", entityID)
    })
}

func (b *MprBackend) addAttributeViaModelsdk(domainModelID, entityID model.ID, attr *domainmodel.Attribute) error {
    return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
        for _, e := range dm.Entities {
            if e.ID == entityID {
                if attr.ID == "" {
                    attr.ID = model.ID(mpr.GenerateID())
                }
                attr.TypeName = "DomainModels$Attribute"
                attr.ContainerID = entityID
                e.Attributes = append(e.Attributes, attr)
                return nil
            }
        }
        return fmt.Errorf("entity not found: %s", entityID)
    })
}
```

`writeDomainModel` 辅助函数（`domainmodel_modelsdk.go:52`）：
1. `b.reader.GetDomainModelByID(dmID)` — 通过 sdk/mpr reader 读取并解析 DM BSON
2. 调用 `mutateFn(dm)` — 修改内存中的 Go 结构体
3. `b.writer.SerializeDomainModel(dm)` — 用旧序列化器重建 DM BSON
4. `b.msdkWriteRaw(dm.ID, contents)` — 写入（同模式 B）

**`writeDomainModel` 已内置 nil 守卫，调用方无需重复检查。**

---

## Step 1: 确认 modelsdk gen 包已覆盖目标 domain（仅模式 A）

```bash
ls modelsdk/gen/<domain>/types.go
grep "YourType" modelsdk/gen/<domain>/types.go
```

`init()` 里的 `codec.DefaultRegistry.Register(...)` 确保类型已注册，无需手动初始化。

## Step 2: 确认 MprBackend.msdkWriter 字段存在

`mdl/backend/mpr/backend.go` 的 `MprBackend` 应已有：

```go
msdkWriter modelsdkmpr.UnitWriter
```

`Connect` 同时打开，`Disconnect` 同时置 nil，`Wrap` best-effort（log 失败不阻断）。
两个 Writer 共享同一 `*sql.DB`（`NewWriterFromDB` 注入），已无双连接问题。

## Step 3: 新建实现文件，选择对应模式

新建 `mdl/backend/mpr/<feature>_modelsdk.go`，根据元素类型选模式 A / B / C。

## Step 4: 在 backend.go 替换调用

```go
// 修改前
func (b *MprBackend) DeleteEntity(domainModelID, entityID model.ID) error {
    return b.writer.DeleteEntity(domainModelID, entityID)
}

// 修改后
func (b *MprBackend) DeleteEntity(domainModelID, entityID model.ID) error {
    return b.deleteEntityViaModelsdk(domainModelID, entityID)
}
```

## Step 5: 写集成测试

测试文件 `mdl/backend/mpr/<feature>_modelsdk_test.go`，用 `t.TempDir()` 创建最小 v1 MPR：

```go
db, _ := sql.Open("sqlite", mprPath)
db.Exec(`
    CREATE TABLE _MetaData (_FormatVersion INTEGER, _ProductVersion TEXT,
                            _BuildVersion TEXT, _SchemaHash TEXT);
    INSERT INTO _MetaData VALUES (1, '10.18.0', '10.18.0.0', '');
    CREATE TABLE _Transaction (LastTransactionID TEXT);
    INSERT INTO _Transaction VALUES ('00000000-0000-0000-0000-000000000000');
    CREATE TABLE Unit (UnitID BLOB PRIMARY KEY NOT NULL, ContainerID BLOB,
                       ContainmentName TEXT, TreeConflict LONG,
                       ContentsHash TEXT, ContentsConflicts TEXT, Contents BLOB);
`)
db.Close()

b := New()
b.Connect(mprPath)
defer b.Disconnect()
// 插入目标 BSON unit，调用方法，读回验证字段值
```

## 验证

```bash
go test ./mdl/backend/mpr/... -run TestYourTest -v
go build ./mdl/backend/mpr/...
go vet ./mdl/backend/mpr/...
```

---

## 常见陷阱

### ❌ DomainModel 嵌入元素 ≠ Unit 表独立行

Entity / Attribute / Association / CrossAssociation **没有** Unit 表行。
直接调用 `msdkWriter.DeleteUnit(entityID)` 总是返回 `"unit not found in database"`。

**正确做法**：一律走模式 C（`writeDomainModel`）。判断方法：
```bash
# 如果旧 sdk/mpr 实现里有 GetDomainModelByID + 修改切片 + updateDomainModel，
# 说明是嵌入元素，必须用模式 C。
grep -n "GetDomainModelByID" sdk/mpr/writer_domainmodel.go
```

### UpdateRawUnit 不等于 WriteTransaction

`modelsdk/mpr.Writer.UpdateRawUnit()` 在 v2 路径也调 `updateTransactionID()`（静默忽略错误）。
应始终用 `BeginWriteTransaction().WriteUnit().Commit()`，确保原子性和明确的错误传播。
唯一例外：ALTER PAGE 的 `UpdateRawUnit` 调用（通过 backend 接口透传，已知可接受）。

### BSON 存储值 ≠ 显示名

Mendix 安全级别示例：`Production` 在 BSON 里存的是 `CheckEverything`（见 `sdk/security/security.go:145`）。executor 已处理映射，modelsdk 路径直接接受 executor 传入的常量字符串。

### v2 MPR 变更在 mprcontents

v2 格式（Mendix ≥ 10.18）的实际数据在 `mprcontents/XX/YY/UUID.mxunit`，MPR 文件本身只更新 ContentsHash。用 `find mprcontents -newer MacnicaApp.mpr` 可定位变更的 mxunit 文件。

### WriteTransaction 提交后两个缓存都要失效

`WriteTransaction.Commit()` 只失效 modelsdk reader 缓存。
`msdkWriteRaw` 额外调用 `b.reader.InvalidateCache()` 失效 sdk/mpr reader 缓存。
直接调用 `msdkWriter.InsertUnit/DeleteUnit/UpdateUnitContainer` 的方法不经过 `msdkWriteRaw`，
必要时需手动调用 `b.reader.InvalidateCache()`。

---

## 当前迁移状态

**`MprBackend.writer *mpr.Writer` 字段已完全退役（2025-05）。** 所有写路径已走 modelsdk WriteTransaction。

### 写路径：已完成（✅）

- ✅ 安全：ProjectSecurity / ModuleSecurity / ModuleRoles / AllowedRoles
- ✅ 模块与文件夹：CreateModule（gen repo）/ CreateFolder（gen repo）/ UpdateModule / UpdateModuleSettings / DeleteModule / MoveFolder
- ✅ 枚举与常量：Enumeration / Constant CRUD
- ✅ 微流 / 纳流 / 页面 / 布局 / 片段 / 工作流：Create + Update + Delete + Move
- ✅ 域模型：Entity / Attribute / Association / CrossAssociation CRUD（gen-native）
- ✅ 服务文档：JavaAction / DBConn / DataTransformer / Mappings / JsonStructure / BusinessEvent / OData / REST / ImageCollection / AgentEditor 系列
- ✅ BSON scan 函数迁到 Reader 方法：ScanQualifiedNameUpdates / ScanRenameReferences / FindRenameTarget / ScanOqlQueryUpdates
- ✅ BSON patch 函数变 standalone：PatchNavigationProfile / PatchReconcileMemberAccesses
- ✅ ReferenceService → 用 `*sdkmpr.Reader` 替代 `*sdkmpr.Writer`

### 剩余 `sdk/mpr` import（不含 b.writer 业务逻辑）

| 文件 | 原因 | 退役条件 |
|------|------|---------|
| `backend.go` | `mpr.Reader`（主读路径）、`Wrap(writer *mpr.Writer)` | Reader 方法全迁移后可考虑 |
| `navigation_modelsdk.go` | `mpr.NavigationProfileSpec` 类型、`mpr.PatchNavigationProfile` 函数 | 移到 types 包 |
| `security_entity_access_modelsdk.go` | `mpr.EntityMemberAccess`、`mpr.EntityAccessRevocation` 类型 | 移到 types 包 |
| `refs_modelsdk.go` | `mpr.RenameHit` 类型（return type） | 移到 types 包 |
| `create_services_modelsdk.go` | Serialize* 系列（DBConn、DataTransformer、Mappings 等复杂序列化器） | 逐域 gen-native 重写 |
| `update_services_modelsdk.go` | `SerializeImageCollection` | gen-native 重写 |
| `settings_modelsdk.go` | `SerializeProjectSettings`（复杂 RawParts 合并逻辑） | gen-native 重写 |
| `agenteditor_modelsdk.go` | `SerializeAgentEditor*`（CustomBlobDocument + JSON 编码） | gen-native 重写 |
| `modules_modelsdk.go` | 已删除（✅ Module → gen repo；ModuleSecurity/ModuleSettings → inline BSON） | — |
| `datagrid_builder.go` | `sdk/widgets`（pluggable widget 模板引擎） | widget engine 迁移 |
| `widget_builder.go` | `sdk/widgets`（widget tree 构建） | widget engine 迁移 |

长期目标：全部 `sdk/mpr` import 退出 `mdl/backend/mpr/`，只保留 `sdk/mpr.Reader` 作为主读路径。
