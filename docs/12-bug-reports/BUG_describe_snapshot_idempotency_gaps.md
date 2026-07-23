# BUG: describe-snapshot.mdl 幂等性剩余差异

**日期**：2026-06-02  
**发现场景**：`mxcli check describe-snapshot.mdl --references` 反复输出差异  
**严重程度**：Medium（describe 输出不能无损往返，影响 snapshot 可靠性）

---

## 背景

手动验证步骤（历史 CI 测试 `TestHelpdeskGolden_DescribeSnapshot_Idempotent` 已移除）：

```
describe-snapshot.mdl ≡ describe(execute(clean_mpr, describe-snapshot.mdl))
```

本次工作（2026-06-02）已修复以下问题：
- ✅ enum 必须先于引用它的 entity 描述
- ✅ module role 必须先于 entity grant 描述（两遍 pass 执行）
- ✅ `create workflow` → `create or modify workflow`（幂等）
- ✅ `create user role` → `create or modify user role`（幂等）
- ✅ user role 模块角色替换语义（不再追加重复）

**剩余两类差异仍导致测试失败：**

---

## BUG 1：关联路径 WHERE 子句格式不一致

**现象**：

```diff
- where HD.TicketComment_Ticket/HD.Ticket = $Ticket and IsInternal = true
+ where HD.TicketComment_Ticket / HD.Ticket = $Ticket and IsInternal = true

- where HD.EscalationRequest_Ticket/HD.Ticket = $Ticket
+ where HD.EscalationRequest_Ticket / HD.Ticket = $Ticket
```

**根因**：`describe microflow` 输出关联路径时不加空格（`X/Y`），但 MDL 解析器在解析并重新序列化 WHERE 子句时规范化为 `X / Y`（两侧加空格）。两者语义相同但字符串不同。

**影响**：幂等性测试失败；mxcli check 不受影响。

**修复方向**：
- 方案 A：microflow describe 输出时统一在 `/` 两侧加空格（align with parser output）
- 方案 B：幂等性测试在比较前对 WHERE 子句做空格规范化

**相关文件**：
- `mdl/executor/cmd_microflows_describe*.go`（WHERE 子句序列化）
- （幂等性测试与此报告一同归档，FUSE 基础架构已移除）

---

## BUG 2：Workflow describe 格式细节往返不一致

**现象（多处）**：

### 2a. `call workflow` 格式

```diff
- call workflow HD.WF_SUB_ManagerReview comment 'WF_SUB_ManagerReview' with (WorkflowContext = '$WorkflowContext')
+ call workflow HD.WF_SUB_ManagerReview comment 'WF_SUB_ManagerReview' with (WorkflowContext = '$WorkflowContext');
```

原始 describe 无分号，re-describe 有分号。

### 2b. `workflow operation abort` reason 引号转义

```diff
- workflow operation abort $Workflow reason '''Administratively aborted''';
+ workflow operation abort $Workflow reason '''''''Administratively aborted''''''';
```

原始 3 个引号，re-describe 7 个引号。说明 reason 字符串经过了二次转义。

### 2c. `wait for notification` 注释

```diff
- wait for notification -- WaitForManagerAvailable
+ wait for notification -- Wait for notification
```

注释内容不同（原始用节点名，re-describe 用显示名称）。

### 2d. `boundary event` 结构

```diff
- boundary event interrupting timer 'addHours([%CurrentDateTime%], 48)'
```

原始 describe 包含 boundary event，re-describe 后丢失或位置改变。

### 2e. `@position` 注解漂移

```diff
- @position(1010,200)
+ @position(850,200)
```

位置坐标在往返后变化。

### 2f. Workflow `default -> { }; outcomes` 结构

原始 describe 包含 `default -> { };` 和 `outcomes` 关键字，re-describe 格式不同。

**根因**：workflow describe 和 create 路径在格式化细节上不完全对称。涉及：
- 语句末尾分号一致性
- 字符串字面量转义层数
- 节点注释来源（name vs displayName）
- 位置坐标在 create 时是否被 reset

**影响**：幂等性测试失败；实际 BSON 写入正确性待验证。

**修复方向**：
- 统一 `call workflow` 末尾分号处理
- 修复 `workflow operation abort reason` 的字符串转义（应转义一次，不应二次转义）
- 统一 `wait for notification` 注释来源
- 验证 boundary event 在 create or modify 后是否正确保留
- 考虑 `RESET LAYOUT` 对 position 的影响

**相关文件**：
- `mdl/executor/cmd_workflows_gen.go`（workflow describe）
- `mdl/executor/cmd_workflows_write_gen2.go`（workflow create/modify）

---

## 复现步骤（历史方法，FUSE 基础架构已移除）

```bash
# 手动对比 describe-snapshot.mdl 与 mxcli describe 输出
./bin/mxcli describe -p testdata/helpdesk-golden-11.6.6/minimal.mpr > /tmp/current-describe.mdl
diff testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl /tmp/current-describe.mdl
```

---

## 优先级建议

| BUG | 修复难度 | 优先级 |
|-----|---------|--------|
| BUG 1（关联路径空格）| 低（格式统一）| Medium |
| BUG 2a（call workflow 分号）| 低 | Medium |
| BUG 2b（abort reason 转义）| 低-Medium | High（可能影响实际行为）|
| BUG 2c（wait for notification 注释）| 低 | Low |
| BUG 2d（boundary event 丢失）| Medium-High | High（可能影响 BSON 正确性）|
| BUG 2e（position 漂移）| Medium | Low |
| BUG 2f（default/outcomes 格式）| Medium | Medium |
