# Cross-Adapter BSON Decode Sharing Analysis

## Problem

当前 4 个 adapter 独立解码同一批页面：

```
PageAdapter:      LoadUnit(43 pages) → TypeName check → nodeForElement
PageRefAdapter:   LoadUnit(43 pages) → bson.Unmarshal → walkWidgetTree
WidgetInstanceAdapter: RawUnitSource(43 pages) → bson.Unmarshal → walkRawWidgets
DataContainerAdapter:  RawUnitSource(43 pages) → bson.Unmarshal → walkWidgets (×2)
```

**43 pages × 4 adapters = 172 redundant `LoadUnit` + 129 redundant `bson.Unmarshal`**

pprof 显示 GC 占 32%、PageAdapter 占 21%——主要来自重复的 BSON map 分配。

## 约束

1. **SRP**: 不能合并 adapter（每个 adapter 一个关注点）
2. **OCP**: 方案不应要求修改现有 adapter 接口
3. **DIP**: 高层应依赖抽象，不应依赖低层 BSON 细节

## 方案对比

### 方案 A: 共享 RawUnit 缓存（推荐）

在 `IndexManager` 或 `RawUnitSource` 层添加一个 `decodeCache map[string]map[string]any`，按 unit ID 缓存已解码的 BSON map。

流程：
```
Pre-build phase（在 BuildAll 之前）:
  for each page unit:
    cache[unitID] = bson.Unmarshal(raw)

BuildAll:
  PageAdapter:      直接读 cache[unitID]（已存在）
  PageRefAdapter:   直接读 cache[unitID]（命中）
  DataContainerAdapter: 直接读 cache[unitID]（命中）
```

```go
// 新增：BsonDocCache 接口 + 实现
type BsonDocCache interface {
    Get(unitID string) map[string]any  // 返回已解码 map
    Preload(units []RawUnit)           // 批量预解码
}

// 每个 adapter 通过依赖注入获得 cache（DIP）
type PageRefAdapter struct {
    Model *modelsdk.Model
    DocCache BsonDocCache  // 新增：注入
}
```

**优点**:
- 完全保留 SRP（adapter 仍是单领域）
- OCP 兼容（现有 adapter 代码不改，只加一个字段）
- 复用已解码的 BSON map，减少 66% 的 bson.Unmarshal 调用
- 预解码可并行化（goroutine per unit）

**代价**:
- 增加 ~43 page × ~50 widget/widget = ~2150 个 map 的内存驻留
- 需要统一所有 adapter 使用同一个 map（目前 PageAdapter 用 typed path，其他用 raw map）

### 方案 B: 合并 RawUnitSource 预解码

修改 `ModelsdkUnitSource.Units()` 在返回前预解码所有 unit 的 BSON，在 `RawUnit` 上附加缓存字段：

```go
type cachedRawUnit struct {
    id       string
    typeName string
    raw      []byte
    decoded  map[string]any  // 缓存已解码 map，nil 表示未解码
}
```

```go
func (s *ModelsdkUnitSource) Units() []RawUnit {
    for each unit:
        load, decode raw → cachedRawUnit{raw:, decoded: nil}
}

// 新增全局方法
func DecodeUnit(unit RawUnit) map[string]any {
    if cu, ok := unit.(*cachedRawUnit); ok && cu.decoded != nil {
        return cu.decoded
    }
    doc := bson.Unmarshal(raw)
    if cu, ok := unit.(*cachedRawUnit); ok {
        cu.decoded = doc
    }
    return doc
}
```

**优点**: 无侵入性，RawUnit 接口不变，adapter 代码不改
**缺点**: 需要 RawUnit 实现类型判断，map 生命周期不明确

### 方案 C: IndexManager 级联缓存

在 `IndexManager` 上添加一个 `preDecode` 阶段，在 BuildAll 之前运行。所有依赖 raw BSON 的 adapter 从解码的 map 读取。

```go
type IndexManager struct {
    graph    *Graph
    adapters map[string]IndexAdapter
    deltaLog *DeltaLog
    docCache map[string]map[string]any  // unitID → decoded
}

func (m *IndexManager) PreDecodePages(source RawUnitSource) {
    for _, unit := range source.Units() {
        if isPage(unit) {
            var doc map[string]any
            bson.Unmarshal(unit.Raw(), &doc)
            m.docCache[unit.ID()] = doc
        }
    }
}
```

Adapter 通过 `IndexManager.DocCache()` 获取。

**优点**: 对 adapter 无侵入（只加一个公共 getter）
**缺点**: 引入 `IndexManager` 对 BSON 解码的依赖，违反分层

## 推荐方案：A

理由：
1. 保持 SRP（adapter 不负责解码）
2. 依赖注入（DocCache 通过构造字段传入，DIP）
3. 可测试性（mock DocCache）
4. 渐进式（先从 DataContainerAdapter 开始，再扩展到 PageRefAdapter 和 WidgetInstanceAdapter）

### 实施步骤

1. 定义接口：
```go
type BsonDocCache interface {
    Get(unitID string) map[string]any
    Set(unitID string, doc map[string]any)
}
```

2. 实现 `mapBsonDocCache`（基于 `sync.Map`）

3. 在 `cmd_graph.go` 的 `buildGraph` 中创建 cache，注册给所有 page-walking adapter

4. 各 adapter 的 `Build()`: 
   - 先检查 cache 是否命中
   - 若命中则直接用，跳过 `bson.Unmarshal`
   - 若不命中则解码并存入 cache

5. 预期收益：页面 BSON 解码从 4 次→1 次，减少 ~75% 解码开销
