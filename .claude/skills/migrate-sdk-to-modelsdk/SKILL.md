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

`modelsdk/mpr.WriteTransaction` 完全不写 `_Transaction` 表，且使用 temp→rename 两阶段写，崩溃安全。

## Step 1: 确认 modelsdk gen 包已覆盖目标 domain

```bash
ls modelsdk/gen/<domain>/types.go
grep "ProjectSecurity\|YourType" modelsdk/gen/<domain>/types.go
```

`init()` 里的 `codec.DefaultRegistry.Register(...)` 确保类型已注册，无需手动初始化。

## Step 2: 在 MprBackend 确认 msdkWriter 字段存在

`mdl/backend/mpr/backend.go` 的 `MprBackend` 应已有：

```go
msdkWriter *modelsdkmpr.Writer
```

`Connect` 同时打开，`Disconnect` 同时关闭（errors.Join），`Wrap` best-effort（log 失败不阻断）。若字段缺失，先按 commit `74d5c7cb`/`2c72c3c6` 的模式补充。

## Step 3: 新建或修改实现文件

新建 `mdl/backend/mpr/<feature>_modelsdk.go`，使用以下六步模板：

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
    "fmt"
    "go.mongodb.org/mongo-driver/bson"
    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/modelsdk/codec"
    msdkxxx "github.com/mendixlabs/mxcli/modelsdk/gen/<domain>"
)

func (b *MprBackend) setXxxViaModelsdk(unitID model.ID, value string) error {
    // 1. nil 守卫（Wrap 路径下 msdkWriter 可能为 nil）
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }

    // 2. 读原始 BSON
    rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
    if err != nil {
        return fmt.Errorf("read unit: %w", err)
    }

    // 3. 解码为 typed element
    elem, err := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))
    if err != nil {
        return fmt.Errorf("decode unit: %w", err)
    }

    // 4. 类型断言（包含实际类型名以便诊断）
    typed, ok := elem.(*msdkxxx.YourType)
    if !ok {
        return fmt.Errorf("unexpected type %T (want *msdkxxx.YourType)", elem)
    }

    // 5. 变更（property.Set 自动标脏，Encoder 只重建变更字段）
    typed.SetXxx(value)

    // 6. 编码 + 事务写
    newBytes, err := (&codec.Encoder{}).Encode(typed)
    if err != nil {
        return fmt.Errorf("encode unit: %w", err)
    }
    wtx, err := b.msdkWriter.BeginWriteTransaction()
    if err != nil {
        return fmt.Errorf("begin write transaction: %w", err)
    }
    if err := wtx.WriteUnit(string(unitID), newBytes); err != nil {
        _ = wtx.Rollback()
        return fmt.Errorf("write unit: %w", err)
    }
    return wtx.Commit()
}
```

## Step 4: 在 backend.go 替换调用

```go
// 修改前
func (b *MprBackend) SetXxx(unitID model.ID, value string) error {
    return b.writer.SetXxx(unitID, value)
}

// 修改后
func (b *MprBackend) SetXxx(unitID model.ID, value string) error {
    return b.setXxxViaModelsdk(unitID, value)
}
```

## Step 5: 写集成测试

测试文件 `mdl/backend/mpr/<feature>_modelsdk_test.go`，用 `t.TempDir()` 创建最小 v1 MPR：

```go
// 最小 SQLite schema（v1 MPR，避免 mprcontents 文件操作）
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
// 插入目标 BSON unit，调用 MprBackend.Connect + SetXxx，读回验证字段值
```

## 验证

```bash
go test ./mdl/backend/mpr/... -run TestYourTest -v
go build ./mdl/backend/mpr/...
go vet ./mdl/backend/mpr/...
```

## 常见陷阱

### UpdateRawUnit 不等于 WriteTransaction

`modelsdk/mpr.Writer.UpdateRawUnit()` 在 v2 路径也调 `updateTransactionID()`（但静默忽略错误）。应始终用 `BeginWriteTransaction().WriteUnit().Commit()`，确保原子性和明确的错误传播。

### BSON 存储值 ≠ 显示名

Mendix 安全级别示例：`Production` 在 BSON 里存的是 `CheckEverything`（见 `sdk/security/security.go:145`）。executor 已处理映射，modelsdk 路径直接接受 executor 传入的常量字符串。

### v2 MPR 变更在 mprcontents

v2 格式（Mendix ≥ 10.18）的实际数据在 `mprcontents/XX/YY/UUID.mxunit`，MPR 文件本身只更新 ContentsHash。用 `find mprcontents -newer MacnicaApp.mpr` 可定位变更的 mxunit 文件。

### 双连接并发写

`sdk/mpr.Writer` 和 `modelsdk/mpr.Writer` 各持一个 `*sql.DB`。顺序写无问题（SQLite 文件锁），高并发场景可能 SQLITE_BUSY。暂时接受此限制，未来考虑共享 `*sql.DB`。

## 下一步迁移优先级

按复杂度排序（已完成的打 ✅）：

- ✅ `SetProjectSecurityLevel` — 单字段，ProjectSecurity unit
- `SetProjectDemoUsersEnabled` — 同 unit，单字段，模式完全相同
- `AddUserRole` / `RemoveUserRole` / `AlterUserRoleModuleRoles` — PartList 操作，同 ProjectSecurity unit
- `ModuleSecurity` 写方法 — 独立 unit，需确认 gen/security 覆盖
- 实体访问规则（EntityAccessRule）— 涉及嵌套 PartList，复杂度较高
- 微流、页面写路径 — 留到 engine 层全面重写时一并处理

长期目标：`MprBackend.writer`（`sdk/mpr.Writer`）字段完全退出，所有写操作走 modelsdk WriteTransaction。
