# mprutil — Mendix MPR BSON 探索与分析库

## 用途

高效探索、分析和对比 Mendix `.mpr` 项目的 Python 库。支持 MPR v1（SQLite）和 v2（mprcontents/）两种格式。

常见场景：

- **崩溃调查**：定位 mx check KeyNotFoundException 的根源——找到缺失的 `$ID` 定义或悬挂指针
- **BSON 结构对比**：对比不同项目或版本中同类型 Unit 的结构差异
- **引用追踪**：追踪 ByIdRef 指针的引用链，检测悬挂引用
- **元数据提取**：批量提取 nanoflow/microflow 的返回变量、活动类型等信息

## 架构

```
mprutil/
├── domain/              # 领域模型（纯数据类，无外部依赖）
│   ├── unit.py          # Unit, UnitId — 存储单元与 UUID 转换
│   ├── document.py      # BsonDocument, BsonElement — 零拷贝文档模型
│   └── reference.py     # ByIdRef, ByNameRef — 引用模型
├── infrastructure/      # 基础设施（文件 I/O、解析）
│   ├── bson_parser.py   # 零拷贝 BSON 解析器
│   ├── mpr_base.py      # MprReader 协议接口
│   ├── mpr_v1.py        # SQLite 后端
│   └── mpr_v2.py        # 文件系统后端
├── application/          # 应用服务
│   ├── explorer.py      # 导航、搜索、字符串提取
│   ├── analyzer.py      # $ID→$Type 映射、引用追踪
│   └── differ.py        # BSON 逐字段对比
└── __main__.py           # CLI 入口
```

### 依赖方向

```
application → domain ← infrastructure
```

- `domain` 无外部依赖
- `infrastructure` 依赖 `domain`
- `application` 依赖 `domain` + `infrastructure`（注入）

## 快速开始

```bash
cd /mnt/data_sdb/mxcli
pip install -e mprutil/  # 可选，直接用亦可
```

### CLI

```bash
# 项目概览（unit 类型分布）
python -m mprutil /tmp/minimal.mpr info

# 列出所有有返回变量的 nanoflow
python -m mprutil /tmp/minimal.mpr nanos

# 列出所有有返回变量的 microflow
python -m mprutil /tmp/minimal.mpr micros

# 提取所有 BSON 元素类型
python -m mprutil /tmp/minimal.mpr types

# 显示指定 Unit 的详细信息
python -m mprutil /tmp/minimal.mpr dump <unit-id>

# 检测悬挂引用（潜在崩溃根因）
python -m mprutil /tmp/minimal.mpr refs

# 提取 nanoflow 中所有 $ID→$Type 映射
python -m mprutil /tmp/minimal.mpr elements
```

### Python API

```python
from mprutil import open_mpr, UnitExplorer, ReferenceAnalyzer, BsonDiffer

with open_mpr("project.mpr") as reader:
    # 导航
    explorer = UnitExplorer(reader)

    # 按类型搜索
    for u in explorer.find_by_type("Microflows$Nanoflow"):
        info = explorer.describe(u)
        if info.get("return_variable"):
            print(f"{info['name']} → ${info['return_variable']}")

    # 提取可读字符串
    strings = explorer.extract_strings_containing(u, {"$Variable", "Return"})

    # 引用追踪
    analyzer = ReferenceAnalyzer(reader)

    # 提取 $ID→$Type 映射
    elements = analyzer.extract_element_types(u.raw_bytes)

    # 检测悬挂引用
    dangling = analyzer.find_dangling_refs(u)

    # BSON 对比
    u1 = reader.get_unit("unit-id-a")
    u2 = reader.get_unit("unit-id-b")
    diff = BsonDiffer.diff_units(u1, u2)
    for entry in diff.structural_diffs:
        print(f"  {entry.kind}: {entry.path}")
```

## 关键概念

### UnitId — UUID 与 GUID LE 互转

Mendix BSON 使用 Windows GUID 字节序（前 3 部分 Little Endian）。

```python
uuid_str = UnitId.guid_le_to_uuid(raw_16_bytes)   # bytes → "xxxxxxxx-xxxx-..."
raw_bytes = UnitId.uuid_to_guid_le(uuid_str)       # "..." → 16 字节检索用
```

### BsonDocument — 零拷贝解析

解析时只记录每个字段的 `(type, key, offset)` 三元组，value 引用原始 buffer。不拷贝数据，只在取值时解码。

### BsonElement — 按类型取值

```python
elem.as_string()    # BSON STRING
elem.as_guid()      # BSON BINARY UUID → hex string
elem.as_int32()     # BSON INT32
elem.as_document()  # BSON EMBEDDED DOC → BsonDocument
elem.as_array()     # BSON ARRAY → list
```

## 崩溃调查工作流

```python
# 1. 从 mx check 输出获取崩溃 UUID
crash_uuid = "495aea6a-021f-4a9b-ae33-98f832d62b66"

# 2. 找到包含该 UUID 的所有 Unit
target, referrers = analyzer.trace_refs(crash_uuid)

# 3. 提取定义该 UUID 的 Unit 中所有元素
elements = analyzer.extract_element_types(target.raw_bytes)

# 4. 检查定义该 UUID 的元素类型
for el_id, tp in elements:
    if el_id == crash_uuid_hex:
        print(f"Crash element: {tp}")
        break
else:
    print("UUID referenced but never defined → DANGLING REF")
    # 找到哪里引用了它
    for ref in analyzer.collect_refs_in_unit(target.raw_bytes):
        if "495aea6a" in str(ref):
            print(f"  Referenced by: {ref}")
```
