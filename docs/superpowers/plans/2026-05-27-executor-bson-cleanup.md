# Executor BSON 反模式清除实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 从 `mdl/executor/` 的 12 个文件中完全清除直接 `bson`/`codec` 导入，所有 BSON 操作通过 `ctx.Backend.*` 或 gen-package 封装函数访问。

**Architecture:** 三层修复：(1) 在 `modelsdk/gen/*/` 创建 supplement 文件封装 codec 调用，executor 改为调用 gen-package helpers；(2) 为 `map[string]any`→gen 的 round-trip 模式添加 backend 接口方法；(3) 重定义 `DataGridSpec` 去除 `bson.D` 字段，将 BSON 构建下沉到 mpr 私有函数；最后添加 Go 测试防护门防止回归。

**Tech Stack:** Go、`go/parser`（guard）、`modelsdk/codec`、`modelsdk/gen/*`、`mdl/backend` 接口层

---

## 文件清单

**新增文件：**
- `modelsdk/gen/microflows/supplement_describe.go` — 为缺少 getter 的 microflow gen 类型提供封装函数
- `modelsdk/gen/workflows/supplement_describe.go` — 为 workflow gen 类型提供封装函数
- `modelsdk/gen/texts/supplement_items.go` — Text/Translation BSON 解析辅助
- `modelsdk/gen/pages/supplement_describe.go` — primitive.A/Binary 封装函数（用于 cmd_pages_describe.go）
- `mdl/types/javaaction_desc.go` — JavaParamDesc 共享类型
- `mdl/executor/import_guard_test.go` — 禁止 executor 直接 import bson/codec 的 Go 测试

**修改文件：**
- `mdl/backend/backend.go` — `PageBackend` 增加 decode 方法；`JavaBackend` 增加 `DescribeJavaActionParameters`
- `mdl/backend/mutation.go` — `DataGridSpec` 去除 `bson.D`，新增 `DataGridDataSource` 类型
- `mdl/backend/mpr/pages.go`（或页面相关文件）— 实现新 decode 方法
- `mdl/backend/mpr/datagrid_builder.go` — 接收 `DataGridDataSource`，内部构建 BSON
- `mdl/backend/mpr/javaactions.go` — 实现 `DescribeJavaActionParameters`
- `mdl/backend/mock/mock_page.go` — 添加 decode 方法 Func 字段
- `mdl/backend/mock/mock_java.go` — 添加 `DescribeJavaActionParametersFunc` 字段
- `mdl/executor/cmd_microflows_format_calls_gen.go` — 移除 codec 导入
- `mdl/executor/cmd_microflows_format_data_gen.go` — 移除 codec/bson 导入
- `mdl/executor/cmd_microflows_format_external_gen.go` — 移除 codec 导入
- `mdl/executor/cmd_microflows_format_action_gen.go` — 移除 bson 导入
- `mdl/executor/cmd_workflows_gen.go` — 移除 codec 导入
- `mdl/executor/cmd_javaactions_gen.go` — 移除 codec 导入
- `mdl/executor/cmd_pages_describe_pluggable.go` — 移除 bson/codec 导入
- `mdl/executor/cmd_pages_describe_output.go` — 移除 bson/codec/primitive 导入
- `mdl/executor/cmd_pages_builder_v3.go` — 移除 bson 导入

---

## Task 1：添加 import guard（先写测试，测试应失败）

**Files:**
- Create: `mdl/executor/import_guard_test.go`

> **注意：** 此测试先写，此刻应报告 12 个违规文件。每完成一个后续 Task，运行此测试确认违规数量减少。

- [ ] **Step 1: 写 import guard 测试**

```go
// mdl/executor/import_guard_test.go
package executor

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestNoDirectBSONImportInExecutor 扫描 mdl/executor 中所有非测试 .go 文件，
// 禁止直接导入 bson/codec 相关包。
// 所有 BSON 操作必须通过 ctx.Backend.* 或 gen-package 封装函数进行。
func TestNoDirectBSONImportInExecutor(t *testing.T) {
	forbidden := []string{
		"go.mongodb.org/mongo-driver/bson",
		"go.mongodb.org/mongo-driver/bson/primitive",
		"go.mongodb.org/mongo-driver/bson/bsoncore",
		"github.com/mendixlabs/mxcli/modelsdk/codec",
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if path == bad || strings.HasPrefix(path, bad+"/") {
					t.Errorf("%s: forbidden import %q (use ctx.Backend.* or gen supplement functions instead)", name, path)
				}
			}
		}
	}
}
```

- [ ] **Step 2: 确认测试失败（此时应报告 ~12 个违规）**

```bash
cd mdl/executor && go test -run TestNoDirectBSONImportInExecutor -v .
```

预期：FAIL，显示约 12 个文件的违规 import。

- [ ] **Step 3: Commit**

```bash
git add mdl/executor/import_guard_test.go
git commit -m "test(executor): add import guard – bson/codec forbidden in executor"
```

---

## Task 2：注册 ExpressionBasedCodeActionParameterValue + 修复 format_calls_gen

**Files:**
- Create: `modelsdk/gen/microflows/supplement_code_actions.go`
- Modify: `mdl/executor/cmd_microflows_format_calls_gen.go`

目标：让 `ExpressionBasedCodeActionParameterValue` 被 codec.DefaultRegistry 识别，
从而使 executor 无需直接调用 codec。

- [ ] **Step 1: 写测试——codec 应能解码该类型**

```go
// 在现有文件 modelsdk/gen/gen_format_test.go 中添加（或在 microflows/ 创建新测试文件）
// 文件: modelsdk/gen/expr_based_code_action_test.go
package gen_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"go.mongodb.org/mongo-driver/bson"
)

func TestExpressionBasedCodeActionParameterValue_Decode(t *testing.T) {
	// 构造含 Expression 字段的 BSON 文档
	raw, _ := bson.Marshal(bson.D{
		{Key: "$Type", Value: "Microflows$ExpressionBasedCodeActionParameterValue"},
		{Key: "Expression", Value: "'hello world'"},
	})
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	v, ok := elem.(*genMf.ExpressionBasedCodeActionParameterValue)
	if !ok {
		t.Fatalf("got %T, want *genMf.ExpressionBasedCodeActionParameterValue", elem)
	}
	if got := v.Expression(); got != "'hello world'" {
		t.Errorf("Expression() = %q, want %q", got, "'hello world'")
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./modelsdk/gen/... -run TestExpressionBased -v
```

预期：FAIL（类型未注册，返回 `*element.Base`）

- [ ] **Step 3: 创建 supplement 文件，注册类型并添加 getter**

```go
// modelsdk/gen/microflows/supplement_code_actions.go
package microflows

import (
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

func init() {
	codec.DefaultRegistry.Register(
		"Microflows$ExpressionBasedCodeActionParameterValue",
		func() element.Element { return &ExpressionBasedCodeActionParameterValue{} },
	)
}

// Expression returns the Mendix expression string stored in this parameter value.
func (o *ExpressionBasedCodeActionParameterValue) Expression() string {
	v, _ := codec.ReadBSONFieldString(o.Raw(), "Expression")
	return v
}
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
go test ./modelsdk/gen/... -run TestExpressionBased -v
```

预期：PASS

- [ ] **Step 5: 更新 executor 文件，移除 codec 调用**

打开 `mdl/executor/cmd_microflows_format_calls_gen.go`，找到函数 `formatCodeActionParameterValueGen`（约行 242）。

当前（两处 `codec.ReadBSONFieldString` 调用）：
```go
case *genMf.ExpressionBasedCodeActionParameterValue:
    if expr, err := codec.ReadBSONFieldString(t.Raw(), "Expression"); err == nil {
        return fmt.Sprintf("expression(%s)", expr)
    }
    return "expression(?)"
// ...
case *element.Base:
    // fallback — ExpressionBasedCodeActionParameterValue 注册前的降级路径
    if expr, err := codec.ReadBSONFieldString(t.Raw(), "Expression"); err == nil {
        return fmt.Sprintf("expression(%s)", expr)
    }
```

替换为（注册后 `*element.Base` fallback 不再触发，可简化）：
```go
case *genMf.ExpressionBasedCodeActionParameterValue:
    if expr := t.Expression(); expr != "" {
        return fmt.Sprintf("expression(%s)", expr)
    }
    return "expression(?)"
```

同时删除 `*element.Base` 中的 Expression fallback 分支（现在已不需要）。

从 import 块中删除：`"github.com/mendixlabs/mxcli/modelsdk/codec"`

- [ ] **Step 6: 验证编译 + import guard**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

预期：`cmd_microflows_format_calls_gen.go` 从违规列表中消失。

- [ ] **Step 7: Commit**

```bash
git add modelsdk/gen/microflows/supplement_code_actions.go \
        mdl/executor/cmd_microflows_format_calls_gen.go
git commit -m "fix(gen): register ExpressionBasedCodeActionParameterValue + add Expression() getter"
```

---

## Task 3：Microflow format_data_gen 的 supplement 封装函数

**Files:**
- Modify: `modelsdk/gen/microflows/supplement_code_actions.go`（扩展，或创建 `supplement_describe.go`）
- Modify: `mdl/executor/cmd_microflows_format_data_gen.go`

`cmd_microflows_format_data_gen.go` 有约 12 处 `codec.ReadBSONFieldString` 调用，读取未暴露为 gen getter 的字段。在 gen supplement 文件中封装这些调用。

- [ ] **Step 1: 在 `modelsdk/gen/microflows/` 新建 supplement 文件**

```go
// modelsdk/gen/microflows/supplement_describe.go
package microflows

import "github.com/mendixlabs/mxcli/modelsdk/codec"

// readField reads a BSON string field from a raw element's bytes.
// Used internally for gen types that do not expose a getter for a field.
func readField(raw []byte, key string) string {
	v, _ := codec.ReadBSONFieldString(raw, key)
	return v
}

// CastActionObjectVariableName reads the ObjectVariableName BSON field.
// This field stores the variable name that the cast result is bound to.
func CastActionObjectVariableName(o *CastAction) string {
	return readField(o.Raw(), "ObjectVariableName")
}

// ConstantRangeLimitExpression reads the LimitExpression BSON field.
func ConstantRangeLimitExpression(o *ConstantRange) string {
	return readField(o.Raw(), "LimitExpression")
}

// ConstantRangeOffsetExpression reads the OffsetExpression BSON field.
func ConstantRangeOffsetExpression(o *ConstantRange) string {
	return readField(o.Raw(), "OffsetExpression")
}

// DatabaseRetrieveSourceXPathConstraint reads the XpathConstraint BSON field.
func DatabaseRetrieveSourceXPathConstraint(o *DatabaseRetrieveSource) string {
	return readField(o.Raw(), "XpathConstraint")
}

// ODataRetrieveSourceAssociationID reads the AssociationId BSON field.
func ODataRetrieveSourceAssociationID(o *ODataRetrieveSource) string {
	return readField(o.Raw(), "AssociationId")
}

// ODataRetrieveSourceXPathConstraint reads the XpathConstraint BSON field.
func ODataRetrieveSourceXPathConstraint(o *ODataRetrieveSource) string {
	return readField(o.Raw(), "XpathConstraint")
}

// ValidationFeedbackActionValidationVariableName reads the ValidationVariableName BSON field.
func ValidationFeedbackActionValidationVariableName(o *ValidationFeedbackAction) string {
	return readField(o.Raw(), "ValidationVariableName")
}

// CreateVariableActionResultVariableName reads the ResultVariableName BSON field
// as fallback when OutputVariableName is empty.
func CreateVariableActionResultVariableName(o *CreateVariableAction) string {
	return readField(o.Raw(), "ResultVariableName")
}
```

> **注意：** 运行 `grep -n "type CastAction\|type ConstantRange\|type DatabaseRetrieveSource\|type ODataRetrieveSource\|type ValidationFeedbackAction\|type CreateVariableAction" modelsdk/gen/microflows/types.go` 确认这些类型名存在于 types.go，并用实际的导出类型名替换上述函数参数中的类型（types.go 中可能有前缀或不同拼写）。

- [ ] **Step 2: 为 supplement 函数写测试**

```go
// modelsdk/gen/microflows/supplement_describe_test.go
package microflows_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"go.mongodb.org/mongo-driver/bson"
)

func TestCastActionObjectVariableName(t *testing.T) {
	raw, _ := bson.Marshal(bson.D{
		{Key: "$Type", Value: "Microflows$CastAction"},
		{Key: "ObjectVariableName", Value: "myVar"},
	})
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, _ := dec.Decode(bson.Raw(raw))
	ca, ok := elem.(*genMf.CastAction)
	if !ok {
		// CastAction の登録を確認
		t.Skipf("CastAction not decoded as expected, got %T — verify type name in types.go", elem)
	}
	if got := genMf.CastActionObjectVariableName(ca); got != "myVar" {
		t.Errorf("got %q, want %q", got, "myVar")
	}
}

func TestConstantRangeLimitExpression(t *testing.T) {
	raw, _ := bson.Marshal(bson.D{
		{Key: "$Type", Value: "Microflows$ConstantRange"},
		{Key: "LimitExpression", Value: "100"},
	})
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, _ := dec.Decode(bson.Raw(raw))
	cr, ok := elem.(*genMf.ConstantRange)
	if !ok {
		t.Skipf("ConstantRange not decoded as expected, got %T", elem)
	}
	if got := genMf.ConstantRangeLimitExpression(cr); got != "100" {
		t.Errorf("got %q, want %q", got, "100")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./modelsdk/gen/microflows/... -run TestCastAction -v
go test ./modelsdk/gen/microflows/... -run TestConstantRange -v
```

如果 Skipped，检查 `types.go` 中的实际类型名并更新函数参数。如果 PASS，继续。

- [ ] **Step 4: 更新 executor，替换所有 codec 调用**

在 `mdl/executor/cmd_microflows_format_data_gen.go` 中，将所有 `codec.ReadBSONFieldString(elem.Raw(), "FieldName")` 替换为对应的 supplement 函数：

| 原调用 | 替换为 |
|--------|--------|
| `codec.ReadBSONFieldString(o.Raw(), "ObjectVariableName")` | `genMf.CastActionObjectVariableName(o)` |
| `codec.ReadBSONFieldString(o.Raw(), "LimitExpression")` | `genMf.ConstantRangeLimitExpression(o)` |
| `codec.ReadBSONFieldString(o.Raw(), "OffsetExpression")` | `genMf.ConstantRangeOffsetExpression(o)` |
| `codec.ReadBSONFieldString(ds.Raw(), "XpathConstraint")` (DatabaseRetrieve) | `genMf.DatabaseRetrieveSourceXPathConstraint(ds)` |
| `codec.ReadBSONFieldString(ds.Raw(), "AssociationId")` | `genMf.ODataRetrieveSourceAssociationID(ds)` |
| `codec.ReadBSONFieldString(o.Raw(), "ValidationVariableName")` | `genMf.ValidationFeedbackActionValidationVariableName(o)` |
| `codec.ReadBSONFieldString(o.Raw(), "ResultVariableName")` | `genMf.CreateVariableActionResultVariableName(o)` |

其他 `codec.ReadBSONFieldString` 调用（如 `AttributeValueCollection`）：按同样模式在 supplement 文件中添加对应函数，再替换。

删除 import：`"github.com/mendixlabs/mxcli/modelsdk/codec"` 和 `"go.mongodb.org/mongo-driver/bson"`

- [ ] **Step 5: 编译 + import guard**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

预期：`cmd_microflows_format_data_gen.go` 从违规列表中消失。

- [ ] **Step 6: Commit**

```bash
git add modelsdk/gen/microflows/supplement_describe.go \
        modelsdk/gen/microflows/supplement_describe_test.go \
        mdl/executor/cmd_microflows_format_data_gen.go
git commit -m "fix(gen): add microflow supplement getters, remove codec from format_data_gen"
```

---

## Task 4：Microflow format_external_gen 的 supplement 封装函数

**Files:**
- Modify: `modelsdk/gen/microflows/supplement_describe.go`（扩展）
- Modify: `mdl/executor/cmd_microflows_format_external_gen.go`

`cmd_microflows_format_external_gen.go` 有约 11 处 `codec.ReadBSONFieldString`，主要在外部集成操作（REST、WebService、Import/ExportXml）。

- [ ] **Step 1: 扩展 supplement 文件，添加外部集成字段封装**

在 `modelsdk/gen/microflows/supplement_describe.go` 末尾追加：

```go
// RestCallActionResultVariableName reads the ResultVariableName BSON field.
func RestCallActionResultVariableName(o *RestCallAction) string {
	return readField(o.Raw(), "ResultVariableName")
}

// ImportMappingCallReturnValueMapping reads the ReturnValueMapping BSON field.
func ImportMappingCallReturnValueMapping(o *ImportMappingCall) string {
	return readField(o.Raw(), "ReturnValueMapping")
}

// RestOperationCallActionEntity reads the Entity BSON field.
func RestOperationCallActionEntity(o *RestOperationCallAction) string {
	return readField(o.Raw(), "Entity")
}

// RestOperationCallActionQueryParameter reads the QueryParameter BSON field
// (legacy alias for Parameter).
func RestOperationCallActionQueryParameter(o *RestOperationParameterMapping) string {
	return readField(o.Raw(), "QueryParameter")
}

// ImportXmlActionOutputVariableName reads OutputVariableName from an OutputMethod element.
// The element is passed as raw bytes since OutputMethod has no stable gen type.
func ImportXmlActionOutputVariableName(raw []byte) string {
	return readField(raw, "OutputVariableName")
}

// ImportXmlActionMappingID reads the MappingId BSON field.
func ImportXmlActionMappingID(o *ImportXmlAction) string {
	return readField(o.Raw(), "MappingId")
}

// ImportXmlActionMappingVariableName reads the MappingVariableName BSON field.
func ImportXmlActionMappingVariableName(o *ImportXmlAction) string {
	return readField(o.Raw(), "MappingVariableName")
}

// ExportXmlActionImportedService reads the ImportedService BSON field.
func ExportXmlActionImportedService(o *ExportXmlAction) string {
	return readField(o.Raw(), "ImportedService")
}

// WebServiceCallReturnValueMapping reads the ReturnValueMapping BSON field.
func WebServiceCallReturnValueMapping(o *WebServiceCallAction) string {
	return readField(o.Raw(), "ReturnValueMapping")
}

// ExecuteDatabaseQueryActionQueryParameter reads the QueryParameter BSON field.
func ExecuteDatabaseQueryActionQueryParameter(o *ExecuteDatabaseQueryAction) string {
	return readField(o.Raw(), "QueryParameter")
}
```

> **注意：** 用 `grep -n "^// ─\|^type " modelsdk/gen/microflows/types.go | grep -A1 "RestCallAction\|ImportMappingCall\|ExportXml"` 确认类型名，并调整上述函数参数中的类型。

- [ ] **Step 2: 更新 executor，替换 codec 调用**

在 `mdl/executor/cmd_microflows_format_external_gen.go` 中，将所有 `codec.ReadBSONFieldString` 替换为对应的 `genMf.Xxx()` 调用（参照 Task 3 的替换表格模式）。

对于 `RestCallAction.ResultHandling` 内嵌元素的字段读取，如果读取的是子元素的字段，用 `genMf.RestCallActionResultVariableName(result)` 其中 `result` 是已解码的子元素。

删除 import：`"github.com/mendixlabs/mxcli/modelsdk/codec"`

- [ ] **Step 3: 编译 + import guard**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

预期：`cmd_microflows_format_external_gen.go` 从违规列表中消失。

- [ ] **Step 4: Commit**

```bash
git add modelsdk/gen/microflows/supplement_describe.go \
        mdl/executor/cmd_microflows_format_external_gen.go
git commit -m "fix(gen): add external action supplement getters, remove codec from format_external_gen"
```

---

## Task 5：Workflow describe 的 supplement 封装函数

**Files:**
- Create: `modelsdk/gen/workflows/supplement_describe.go`
- Modify: `mdl/executor/cmd_workflows_gen.go`

`cmd_workflows_gen.go` 约 12 处 `codec.ReadBSONFieldString`，主要读取 Annotation/Description、Text/Translation/Value 字段和定时器字段。

- [ ] **Step 1: 创建 supplement 文件**

```go
// modelsdk/gen/workflows/supplement_describe.go
package workflows

import "github.com/mendixlabs/mxcli/modelsdk/codec"

func readField(raw []byte, key string) string {
	v, _ := codec.ReadBSONFieldString(raw, key)
	return v
}

// AnnotationDescription reads the Description BSON field from an Annotation.
func AnnotationDescription(o *Annotation) string {
	return readField(o.Raw(), "Description")
}

// AnnotationText reads the Text BSON field (fallback for Description).
func AnnotationText(o *Annotation) string {
	return readField(o.Raw(), "Text")
}

// AnnotationTextTranslation reads the Translation BSON field.
func AnnotationTextTranslation(raw []byte) string {
	return readField(raw, "Translation")
}

// AnnotationTextValue reads the Value BSON field (second fallback).
func AnnotationTextValue(raw []byte) string {
	return readField(raw, "Value")
}

// WorkflowParameterEntity reads the Entity BSON field (used when EntityQualifiedName is empty).
func WorkflowParameterEntity(o *WorkflowParameter) string {
	return readField(o.Raw(), "Entity")
}

// ExclusiveSplitOutcomeValue reads the Value BSON field.
func ExclusiveSplitOutcomeValue(o *ExclusiveSplitOutcome) string {
	return readField(o.Raw(), "Value")
}

// AnnotationRawName reads the Name BSON field from a raw element bytes.
func AnnotationRawName(raw []byte) string { return readField(raw, "Name") }

// AnnotationRawCaption reads the Caption BSON field.
func AnnotationRawCaption(raw []byte) string { return readField(raw, "Caption") }

// AnnotationRawMicroflow reads the Microflow BSON field.
func AnnotationRawMicroflow(raw []byte) string { return readField(raw, "Microflow") }

// TimerActivityDelay reads the Delay BSON field.
func TimerActivityDelay(raw []byte) string { return readField(raw, "Delay") }

// TimerActivityFirstExecutionTime reads the FirstExecutionTime BSON field.
func TimerActivityFirstExecutionTime(raw []byte) string {
	return readField(raw, "FirstExecutionTime")
}
```

> **注意：** 用 `grep -n "^type Annotation\|^type WorkflowParameter\|^type ExclusiveSplitOutcome" modelsdk/gen/workflows/types.go` 确认这些类型名。

- [ ] **Step 2: 写测试**

```go
// modelsdk/gen/workflows/supplement_describe_test.go
package workflows_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	"go.mongodb.org/mongo-driver/bson"
)

func TestAnnotationDescription(t *testing.T) {
	raw, _ := bson.Marshal(bson.D{
		{Key: "$Type", Value: "Workflows$Annotation"},
		{Key: "Description", Value: "test annotation"},
	})
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, _ := dec.Decode(bson.Raw(raw))
	a, ok := elem.(*genWf.Annotation)
	if !ok {
		t.Skipf("Annotation not decoded, got %T", elem)
	}
	if got := genWf.AnnotationDescription(a); got != "test annotation" {
		t.Errorf("got %q, want %q", got, "test annotation")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./modelsdk/gen/workflows/... -run TestAnnotation -v
```

- [ ] **Step 4: 更新 executor，替换 codec 调用**

在 `mdl/executor/cmd_workflows_gen.go` 中，将所有 `codec.ReadBSONFieldString` 替换为对应的 `genWf.Xxx()` 或 `genWf.XxxRaw(elem.Raw())` 调用。

对于多字段 fallback 模式（尝试多个 BSON 字段名）：
```go
// 旧
if v, err := codec.ReadBSONFieldString(elem.Raw(), "Description"); err == nil { ... }
if v, err := codec.ReadBSONFieldString(elem.Raw(), "Text"); err == nil { ... }

// 新
if v := genWf.AnnotationDescription(a); v != "" { ... }
if v := genWf.AnnotationText(a); v != "" { ... }
```

删除 import：`"github.com/mendixlabs/mxcli/modelsdk/codec"`

- [ ] **Step 5: 编译 + import guard**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

- [ ] **Step 6: Commit**

```bash
git add modelsdk/gen/workflows/supplement_describe.go \
        modelsdk/gen/workflows/supplement_describe_test.go \
        mdl/executor/cmd_workflows_gen.go
git commit -m "fix(gen): add workflow supplement getters, remove codec from cmd_workflows_gen"
```

---

## Task 6：Text 解析 supplement + 修复 format_action_gen

**Files:**
- Create: `modelsdk/gen/texts/supplement_items.go`
- Modify: `mdl/executor/cmd_microflows_format_action_gen.go`

`cmd_microflows_format_action_gen.go` 行 800-815 的 `readTextItemsFromRaw` 用 `bson.Unmarshal` 解析 Texts$Text 文档中的 Items 数组，提取 LanguageCode + Text 翻译对。

- [ ] **Step 1: 创建 texts supplement 文件**

```go
// modelsdk/gen/texts/supplement_items.go
package texts

import (
	"go.mongodb.org/mongo-driver/bson"
)

// TranslationPair holds a language code and its translated text string.
type TranslationPair struct {
	LanguageCode string
	Text         string
}

// ReadTranslationPairs extracts (LanguageCode, Text/Value) pairs from
// the raw BSON bytes of a Texts$Text document.
// Returns nil if raw is empty or cannot be parsed.
func ReadTranslationPairs(raw []byte) []TranslationPair {
	if len(raw) == 0 {
		return nil
	}
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	itemsRaw, ok := doc["Items"]
	if !ok {
		return nil
	}
	arr, ok := itemsRaw.(bson.A)
	if !ok {
		return nil
	}
	var pairs []TranslationPair
	for _, item := range arr {
		m, ok := item.(bson.M)
		if !ok {
			continue
		}
		lang, _ := m["LanguageCode"].(string)
		text, _ := m["Text"].(string)
		if text == "" {
			text, _ = m["Value"].(string)
		}
		if lang != "" {
			pairs = append(pairs, TranslationPair{LanguageCode: lang, Text: text})
		}
	}
	return pairs
}
```

- [ ] **Step 2: 写测试**

```go
// modelsdk/gen/texts/supplement_items_test.go
package texts_test

import (
	"testing"

	genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	"go.mongodb.org/mongo-driver/bson"
)

func TestReadTranslationPairs(t *testing.T) {
	raw, _ := bson.Marshal(bson.D{
		{Key: "$Type", Value: "Texts$Text"},
		{Key: "Items", Value: bson.A{
			bson.D{{Key: "LanguageCode", Value: "en_US"}, {Key: "Text", Value: "Hello"}},
			bson.D{{Key: "LanguageCode", Value: "nl_NL"}, {Key: "Text", Value: "Hallo"}},
		}},
	})
	pairs := genTexts.ReadTranslationPairs(raw)
	if len(pairs) != 2 {
		t.Fatalf("got %d pairs, want 2", len(pairs))
	}
	if pairs[0].LanguageCode != "en_US" || pairs[0].Text != "Hello" {
		t.Errorf("pair[0] = %+v", pairs[0])
	}
}

func TestReadTranslationPairs_Empty(t *testing.T) {
	if got := genTexts.ReadTranslationPairs(nil); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./modelsdk/gen/texts/... -run TestReadTranslationPairs -v
```

预期：PASS

- [ ] **Step 4: 更新 executor**

在 `mdl/executor/cmd_microflows_format_action_gen.go` 中，找到 `readTextItemsFromRaw` 函数（约行 800）：

```go
// 旧的 readTextItemsFromRaw 函数（约50行，使用 bson.Unmarshal + bson.M + bson.A）
func readTextItemsFromRaw(raw []byte) []translationPair {
    // ... bson.Unmarshal logic ...
}
```

替换为调用 supplement：
```go
func readTextItemsFromRaw(raw []byte) []translationPair {
	pairs := genTexts.ReadTranslationPairs(raw)
	result := make([]translationPair, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, translationPair{LanguageCode: p.LanguageCode, Text: p.Text})
	}
	return result
}
```

在 import 块添加 `genTexts "github.com/mendixlabs/mxcli/modelsdk/gen/texts"`，删除 `"go.mongodb.org/mongo-driver/bson"`。

（`translationPair` 是 executor 内部类型，保留不变。）

- [ ] **Step 5: 编译 + import guard**

```bash
go build ./mdl/executor/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

- [ ] **Step 6: Commit**

```bash
git add modelsdk/gen/texts/supplement_items.go \
        modelsdk/gen/texts/supplement_items_test.go \
        mdl/executor/cmd_microflows_format_action_gen.go
git commit -m "fix(gen): add texts.ReadTranslationPairs, remove bson from format_action_gen"
```

---

## Task 7：Page decode backend 方法（修复 describe_pluggable + describe_output）

**Files:**
- Modify: `mdl/backend/backend.go`
- Create/Modify: `mdl/backend/mpr/page_decode.go`（新文件，避免搞乱大文件）
- Modify: `mdl/backend/mock/mock_page.go`
- Modify: `mdl/executor/cmd_pages_describe_pluggable.go`
- Modify: `mdl/executor/cmd_pages_describe_output.go`

`decodeMicroflowQNFromSource(ds map[string]any)` 和 `decodeNanoflowQNFromSource` 接受页面 widget 的原始属性 map，通过 `bson.Marshal` + codec decode 转换为 gen 类型。`decodeMicroflowClientAction` 同理。这些函数必须移到 backend。

- [ ] **Step 1: 在 backend 接口中添加方法**

打开 `mdl/backend/backend.go`，找到 `PageBackend` 接口，添加：

```go
type PageBackend interface {
    // ... 现有方法不变 ...

    // DecodeMicroflowQNFromDataSource extracts the microflow qualified name
    // from a raw plugin-widget datasource property map.
    DecodeMicroflowQNFromDataSource(ds map[string]any) string

    // DecodeNanoflowQNFromDataSource extracts the nanoflow qualified name.
    DecodeNanoflowQNFromDataSource(ds map[string]any) string

    // DecodeMicroflowClientAction decodes a raw client-action property map
    // to a gen-typed MicroflowClientAction. Returns nil if not convertible.
    DecodeMicroflowClientAction(action map[string]any) element.Element
}
```

注意：返回 `element.Element` 而不是具体的 `*genPg.MicroflowClientAction`，避免在 backend 接口中引入 gen/pages 依赖（backend 接口文件应保持最小导入）。executor 可以 type-assert 结果。

- [ ] **Step 2: 在 mock 中添加 Func 字段**

打开 `mdl/backend/mock/mock_page.go`，添加：

```go
type MockBackend struct {
    // ... 现有字段不变 ...
    DecodeMicroflowQNFromDataSourceFunc func(ds map[string]any) string
    DecodeNanoflowQNFromDataSourceFunc  func(ds map[string]any) string
    DecodeMicroflowClientActionFunc     func(action map[string]any) element.Element
}

func (m *MockBackend) DecodeMicroflowQNFromDataSource(ds map[string]any) string {
    if m.DecodeMicroflowQNFromDataSourceFunc != nil {
        return m.DecodeMicroflowQNFromDataSourceFunc(ds)
    }
    return ""
}

func (m *MockBackend) DecodeNanoflowQNFromDataSource(ds map[string]any) string {
    if m.DecodeNanoflowQNFromDataSourceFunc != nil {
        return m.DecodeNanoflowQNFromDataSourceFunc(ds)
    }
    return ""
}

func (m *MockBackend) DecodeMicroflowClientAction(action map[string]any) element.Element {
    if m.DecodeMicroflowClientActionFunc != nil {
        return m.DecodeMicroflowClientActionFunc(action)
    }
    return nil
}
```

- [ ] **Step 3: 实现 mpr 端**

创建 `mdl/backend/mpr/page_decode.go`：

```go
package mpr

import (
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	"go.mongodb.org/mongo-driver/bson"
)

func (b *MprBackend) DecodeMicroflowQNFromDataSource(ds map[string]any) string {
	return decodeQNFromSourceMap(ds, "MicroflowSource", "MicroflowQualifiedName")
}

func (b *MprBackend) DecodeNanoflowQNFromDataSource(ds map[string]any) string {
	return decodeQNFromSourceMap(ds, "NanoflowSource", "NanoflowQualifiedName")
}

func decodeQNFromSourceMap(ds map[string]any, expectedType, qnField string) string {
	raw, err := bson.Marshal(ds)
	if err != nil {
		return ""
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(raw))
	if err != nil {
		return ""
	}
	qn, _ := codec.ReadBSONFieldString(elem.Raw(), qnField)
	return qn
}

func (b *MprBackend) DecodeMicroflowClientAction(action map[string]any) element.Element {
	raw, err := bson.Marshal(action)
	if err != nil {
		return nil
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(raw))
	if err != nil {
		return nil
	}
	return elem
}
```

> **注意：** 打开 `mdl/executor/cmd_pages_describe_pluggable.go` 行 18-55，查看 `decodeMicroflowQNFromSource` 和 `decodeNanoflowQNFromSource` 的实际逻辑，确保 mpr 实现与 executor 当前实现完全一致（只是位置移动）。

- [ ] **Step 4: 写接口符合性检查**

在 `mdl/backend/mpr/page_decode.go` 末尾添加：

```go
var _ interface {
    DecodeMicroflowQNFromDataSource(map[string]any) string
    DecodeNanoflowQNFromDataSource(map[string]any) string
    DecodeMicroflowClientAction(map[string]any) element.Element
} = (*MprBackend)(nil)
```

- [ ] **Step 5: 更新 executor 文件**

**`cmd_pages_describe_pluggable.go`：**

找到函数 `decodeMicroflowQNFromSource` 和 `decodeNanoflowQNFromSource`（行 18-55），删除这两个私有函数，改为调用：

```go
// 旧
qn := decodeMicroflowQNFromSource(dsMap)

// 新
qn := ctx.Backend.DecodeMicroflowQNFromDataSource(dsMap)
```

删除 import：`"go.mongodb.org/mongo-driver/bson"` 和 `"github.com/mendixlabs/mxcli/modelsdk/codec"`

**`cmd_pages_describe_output.go`：**

找到函数 `decodeMicroflowClientAction`（行 1320），删除该私有函数，改为：

```go
// 旧
action := decodeMicroflowClientAction(rawMap)

// 新
action := ctx.Backend.DecodeMicroflowClientAction(rawMap)
```

删除 import：`"go.mongodb.org/mongo-driver/bson"`、`"github.com/mendixlabs/mxcli/modelsdk/codec"` 和 `"go.mongodb.org/mongo-driver/bson/primitive"`

- [ ] **Step 6: 编译 + import guard**

```bash
go build ./mdl/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

预期：两个 describe 文件从违规列表消失。

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/backend.go \
        mdl/backend/mpr/page_decode.go \
        mdl/backend/mock/mock_page.go \
        mdl/executor/cmd_pages_describe_pluggable.go \
        mdl/executor/cmd_pages_describe_output.go
git commit -m "fix(backend): add page decode methods, remove bson from describe_pluggable/output"
```

---

## Task 8：JavaAction backend 方法（修复 cmd_javaactions_gen）

**Files:**
- Create: `mdl/types/javaaction_desc.go`
- Modify: `mdl/backend/backend.go`
- Create/Modify: `mdl/backend/mpr/javaactions_describe.go`
- Modify: `mdl/backend/mock/mock_java.go`
- Modify: `mdl/executor/cmd_javaactions_gen.go`

`cmd_javaactions_gen.go` 直接用 `codec.DecodeChild` 处理 `BasicParameterType.Type` 子元素，并处理 `CodeActions$` vs `JavaActions$` 命名空间不匹配问题。

- [ ] **Step 1: 定义共享类型**

```go
// mdl/types/javaaction_desc.go
package types

// JavaParamDesc describes a single parameter of a Java/JavaScript action.
type JavaParamDesc struct {
	Name     string // parameter name
	Category string // "basic", "entity", "enum", "list", "object"
	TypeName string // e.g. "String", "Integer", "Module.MyEntity"
	IsList   bool   // true for list parameters
}
```

- [ ] **Step 2: 添加 backend 接口方法**

在 `mdl/backend/backend.go` 的 `JavaBackend` 接口中添加：

```go
type JavaBackend interface {
    // ... 现有方法不变 ...

    // DescribeJavaActionParameters returns a typed description of the action's
    // parameter types, handling the CodeActions$/JavaActions$ namespace mismatch
    // and unregistered sub-types internally.
    DescribeJavaActionParameters(id model.ID) ([]types.JavaParamDesc, error)
}
```

- [ ] **Step 3: mock stub**

在 `mdl/backend/mock/mock_java.go` 添加：

```go
DescribeJavaActionParametersFunc func(id model.ID) ([]types.JavaParamDesc, error)

func (m *MockBackend) DescribeJavaActionParameters(id model.ID) ([]types.JavaParamDesc, error) {
    if m.DescribeJavaActionParametersFunc != nil {
        return m.DescribeJavaActionParametersFunc(id)
    }
    return nil, fmt.Errorf("MockBackend.DescribeJavaActionParameters not configured")
}
```

- [ ] **Step 4: 实现 mpr 端**

创建 `mdl/backend/mpr/javaactions_describe.go`。实现逻辑：将 executor 中 `cmd_javaactions_gen.go` 的参数类型格式化逻辑（约行 200-280）整体移入此文件，封装为 `DescribeJavaActionParameters`：

```go
// mdl/backend/mpr/javaactions_describe.go
package mpr

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
)

func (b *MprBackend) DescribeJavaActionParameters(id model.ID) ([]types.JavaParamDesc, error) {
	// 1. 通过 reader 获取 JavaAction 的原始 BSON 字节
	raw, err := b.reader.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	// 2. 用 codec 解码，处理 CodeActions$ 命名空间
	//    （将 "CodeActions$" 视同 "JavaActions$"）
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(raw)
	if err != nil {
		return nil, err
	}
	// 3. 提取 ActionParameters / Parameters 字段（双轨兼容）
	//    将 executor 中原有的参数读取逻辑移至此处
	// ... (copy the parameter extraction logic from cmd_javaactions_gen.go)
	return describeParamsFromElement(elem), nil
}

// describeParamsFromElement extracts JavaParamDesc list from a decoded JavaAction element.
// Copy the parameter extraction logic from cmd_javaactions_gen.go into this function.
// Steps:
//  1. Run: grep -n "codec\.DecodeChild\|codec\.ReadBSONFieldString\|ActionParameters\|Parameters" mdl/executor/cmd_javaactions_gen.go
//  2. Identify the function(s) that build the parameter description list (typically formatJavaActionParamsGen or similar)
//  3. Copy those functions here, renaming to unexported helpers
//  4. Return []types.JavaParamDesc built from each parameter element
// codec.DecodeChild and codec.ReadBSONFieldString are ALLOWED here (in mpr package).
func describeParamsFromElement(elem element.Element) []types.JavaParamDesc {
	// Use codec.DecodeChild to get the Parameters or ActionParameters list:
	params, _ := codec.DecodeChildren(elem.Raw(), "ActionParameters")
	if len(params) == 0 {
		params, _ = codec.DecodeChildren(elem.Raw(), "Parameters")
	}
	result := make([]types.JavaParamDesc, 0, len(params))
	for _, p := range params {
		name, _ := codec.ReadBSONFieldString(p.Raw(), "Name")
		// Decode the nested "Type" child element to get category/type name:
		typeElem, _ := codec.DecodeChild(p.Raw(), "Type")
		category, typeName := describeJavaParamType(typeElem)
		result = append(result, types.JavaParamDesc{
			Name:     name,
			Category: category,
			TypeName: typeName,
		})
	}
	return result
}

// describeJavaParamType reads category + type name from a BasicParameterType's Type child.
// Handles CodeActions$BasicType, CodeActions$EntityType, CodeActions$EnumerationType, etc.
func describeJavaParamType(elem element.Element) (category, typeName string) {
	if elem == nil {
		return "unknown", ""
	}
	typStr, _ := codec.ReadBSONFieldString(elem.Raw(), "$Type")
	switch {
	case strings.Contains(typStr, "StringType"):
		return "basic", "String"
	case strings.Contains(typStr, "IntegerType"):
		return "basic", "Integer"
	case strings.Contains(typStr, "LongType"):
		return "basic", "Long"
	case strings.Contains(typStr, "BooleanType"):
		return "basic", "Boolean"
	case strings.Contains(typStr, "DecimalType"):
		return "basic", "Decimal"
	case strings.Contains(typStr, "DateTimeType"):
		return "basic", "DateTime"
	case strings.Contains(typStr, "EntityType"):
		name, _ := codec.ReadBSONFieldString(elem.Raw(), "Entity")
		return "entity", name
	case strings.Contains(typStr, "EnumerationType"):
		name, _ := codec.ReadBSONFieldString(elem.Raw(), "Enumeration")
		return "enum", name
	default:
		return "unknown", typStr
	}
}
```

> **实现注意：** 打开 `mdl/executor/cmd_javaactions_gen.go`，找到所有 `codec.ReadBSONFieldString` 和 `codec.DecodeChild` 调用（约行 200-280），将它们整体搬到 `describeParamsFromElement` 中。backend/mpr 包可以合法使用 `codec` 包。

- [ ] **Step 5: 更新 executor**

在 `cmd_javaactions_gen.go` 中：
1. 找到使用 `codec.DecodeChild` 和 `codec.ReadBSONFieldString` 的参数格式化函数
2. 将这些函数的调用改为 `params, err := ctx.Backend.DescribeJavaActionParameters(id)`
3. 用 `params` 中的类型信息直接格式化输出
4. 删除 `codec` import

- [ ] **Step 6: 编译 + import guard**

```bash
go build ./mdl/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

- [ ] **Step 7: Commit**

```bash
git add mdl/types/javaaction_desc.go \
        mdl/backend/backend.go \
        mdl/backend/mpr/javaactions_describe.go \
        mdl/backend/mock/mock_java.go \
        mdl/executor/cmd_javaactions_gen.go
git commit -m "fix(backend): add DescribeJavaActionParameters, remove codec from javaactions_gen"
```

---

## Task 9：cmd_pages_describe.go 的 primitive 类型封装

**Files:**
- Create: `modelsdk/gen/pages/supplement_describe.go`
- Modify: `mdl/executor/cmd_pages_describe.go`

`cmd_pages_describe.go` 使用 5 处 `primitive.A` 和 `primitive.Binary` 类型断言，用于在 DESCRIBE PAGE 输出中提取数组值或二进制 ID。

- [ ] **Step 1: 查看实际使用**

```bash
grep -n "primitive\." mdl/executor/cmd_pages_describe.go
```

记录每处断言的上下文（在哪个函数、处理什么字段）。

- [ ] **Step 2: 在 gen/pages 创建 supplement，封装 primitive 操作**

```go
// modelsdk/gen/pages/supplement_describe.go
package pages

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BSONArrayFromAny 将 BSON 数组（primitive.A）转换为 []any。
// 用于从 raw 元素中提取数组字段而无需在 executor 直接使用 primitive 包。
func BSONArrayFromAny(v any) ([]any, bool) {
	arr, ok := v.(primitive.A)
	if !ok {
		return nil, false
	}
	return []any(arr), true
}

// BSONBinaryHex 将 primitive.Binary 转换为十六进制字符串（用于显示 ID）。
func BSONBinaryHex(v any) (string, bool) {
	b, ok := v.(primitive.Binary)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%x", b.Data), true
}
```

（根据 `grep` 输出的实际使用场景增减函数。如果 `primitive.Binary` 只是用来提取 ID bytes，则只需 `BSONBinaryBytes`。）

- [ ] **Step 3: 更新 executor**

将 `cmd_pages_describe.go` 中所有 `primitive.A`/`primitive.Binary` 断言改为调用 `genPg.BSONArrayFromAny(v)` 和 `genPg.BSONBinaryHex(v)`。

删除 `"go.mongodb.org/mongo-driver/bson/primitive"` import（如果 `bson` 也被用到，按需处理）。

- [ ] **Step 4: 编译 + import guard**

```bash
go build ./mdl/...
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

预期：`cmd_pages_describe.go` 从违规列表消失。

- [ ] **Step 5: Commit**

```bash
git add modelsdk/gen/pages/supplement_describe.go \
        mdl/executor/cmd_pages_describe.go
git commit -m "fix(gen): add pages supplement for primitive types, clean cmd_pages_describe"
```

---

## Task 10：DataGridSpec 重构（Batch 2）

**Files:**
- Modify: `mdl/backend/mutation.go`
- Modify: `mdl/backend/mpr/datagrid_builder.go`
- Modify: `mdl/executor/cmd_pages_builder_v3.go`

这是改动最大的 Task。将 `DataGridSpec` 的 `DataSourceBSON bson.D` 和 `HeaderWidgetsBSON []bson.D` 替换为类型化的 Go struct，把 BSON 构建函数从 executor 移到 backend。

- [ ] **Step 1: 在 mutation.go 中定义新类型**

在 `mdl/backend/mutation.go` 中，在 `DataGridSpec` 定义之前添加：

```go
// DataGridDataSource describes the data source for a DataGrid2 widget.
// All fields are resolved (IDs, names) — no raw BSON.
type DataGridDataSource struct {
    Type string // "parameter", "database", "microflow", "nanoflow", "association", "selection"

    // For "parameter": the resolved entity ID and variable name
    EntityID    string // bsonutil-format binary ID
    EntityName  string // qualified name, e.g. "MyMod.Order"
    SourceVar   string // parameter variable name

    // For "database": XPath constraint and sort
    XPath     string
    SortItems []DataGridSortItem

    // For "microflow" / "nanoflow": resolved flow ID
    FlowID   string // bsonutil-format binary ID
    FlowName string // qualified name

    // For "association": resolved association and entity
    AssocID       string
    AssocName     string
    DestEntityID  string
    DestEntityName string
    IsCurrentObjectMode bool // true when ContextVariable was empty

    // For "selection": source widget ID
    SourceWidgetID string
}

// DataGridSortItem is a sort specification for database-sourced DataGrid2.
type DataGridSortItem struct {
    AttributeQN string // fully-qualified attribute path
    Ascending   bool
}

// DataGridColumnWidget holds a pre-resolved filter widget specification.
type DataGridColumnWidget struct {
    WidgetID   string
    FilterName string
    FilterType string
    Attributes []string
}
```

替换 `DataGridSpec.DataSourceBSON bson.D` → `DataSource DataGridDataSource`  
替换 `DataGridSpec.HeaderWidgetsBSON []bson.D` → `ColumnWidgets []DataGridColumnWidget`

```go
// 更新后的 DataGridSpec
type DataGridSpec struct {
    DataSource    DataGridDataSource
    Columns       []DataGridColumnSpec
    ColumnWidgets []DataGridColumnWidget
    PagingOverrides map[string]string
    SelectionMode   string
}
```

- [ ] **Step 2: 更新 mpr 后端的 DataGrid 构建函数**

在 `mdl/backend/mpr/datagrid_builder.go` 中，找到 `buildDataGrid2WidgetDoc` 函数（使用 `spec.DataSourceBSON`）。

将 `spec.DataSourceBSON` 的使用改为调用新的私有函数 `buildDataSourceBSONFromSpec(spec.DataSource)`：

```go
// buildDataSourceBSONFromSpec 将 DataGridDataSource 转换为 BSON 文档。
// 逻辑来自原先在 executor cmd_pages_builder_v3.go 中的 buildDataGridDataSourceBSON()。
func buildDataSourceBSONFromSpec(ds DataGridDataSource) (bson.D, string) {
    switch ds.Type {
    case "parameter":
        return bson.D{
            {Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
            {Key: "$Type", Value: "Forms$DataViewSource"},
            {Key: "EntityRef", Value: bson.D{
                {Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
                {Key: "$Type", Value: "Forms$IndirectEntityRef"},
                {Key: "Steps", Value: bson.A{int32(1)}},
            }},
            {Key: "ForceFullObjects", Value: false},
            {Key: "SourceVariable", Value: ds.SourceVar},
        }, ds.EntityName

    case "database":
        sortItems := bson.A{int32(1)}
        for _, s := range ds.SortItems {
            sortItems = append(sortItems, bson.D{
                {Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
                {Key: "$Type", Value: "CustomWidgets$GridSortItem"},
                {Key: "AttributePath", Value: s.AttributeQN},
                {Key: "Ascending", Value: s.Ascending},
            })
        }
        return bson.D{
            {Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
            {Key: "$Type", Value: "CustomWidgets$CustomWidgetXPathSource"},
            {Key: "EntityRef", Value: bson.D{/* entity ref BSON */}},
            {Key: "XPathConstraint", Value: ds.XPath},
            {Key: "SortBar", Value: bson.D{
                {Key: "$Type", Value: "CustomWidgets$GridSortBar"},
                {Key: "Items", Value: sortItems},
            }},
        }, ds.EntityName

    // ... case "microflow", "nanoflow", "association", "selection" ...
    // (copy logic from executor buildDataGridDataSourceBSON, only the BSON assembly part)
    }
    return nil, ""
}
```

> **实现注意：** 打开 `mdl/executor/cmd_pages_builder_v3.go` 行 1919-2112（`buildDataGridDataSourceBSON` 完整函数），将每个 case 分支的 BSON 组装代码（不含 resolver 调用）整体复制到 `buildDataSourceBSONFromSpec` 对应 case 中。resolver 调用结果已在 `DataGridDataSource` 的字段中。

同样处理 `ColumnWidgets []DataGridColumnWidget`：找到 `buildWidgetBSON` 的 BSON 组装部分，移入 backend 私有函数 `buildColumnWidgetBSONFromSpec(w DataGridColumnWidget) bson.D`。

- [ ] **Step 3: 更新 executor**

在 `mdl/executor/cmd_pages_builder_v3.go` 中：

1. **删除** `buildDataGridDataSourceBSON`、`buildWidgetBSON`、`buildMinimalClientTemplate`、`buildMinimalAppearance` 等 4 个函数

2. **替换** executor 调用这些函数的地方（约在 `buildDataGridV3` 函数中）：

```go
// 旧（在 executor）：
dataSrcBSON, entityName, err := pb.buildDataGridDataSourceBSON(s.DataSource)
spec.DataSourceBSON = dataSrcBSON

// 新（在 executor）：
ds, entityName, err := pb.resolveDataGridDataSource(s.DataSource)
// resolveDataGridDataSource() 只做 ID 解析，不构建 BSON：
spec.DataSource = ds
```

3. 新增 executor 私有函数 `resolveDataGridDataSource(*ast.DataSourceV3) (DataGridDataSource, string, error)`，将原 `buildDataGridDataSourceBSON` 中的 **resolver 调用**（`pb.paramScope`、`pb.entityCache`、`pb.resolveEntityByName` 等）保留在 executor，只把**最终 ID 和名称**填入 `DataGridDataSource` 字段。

4. 删除 `"go.mongodb.org/mongo-driver/bson"` import。

- [ ] **Step 4: 编译 + 所有测试**

```bash
go build ./...
go test ./... 2>&1 | head -50
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

- [ ] **Step 5: Commit**

```bash
git add mdl/backend/mutation.go \
        mdl/backend/mpr/datagrid_builder.go \
        mdl/executor/cmd_pages_builder_v3.go
git commit -m "fix(backend): replace DataGridSpec bson.D fields with typed structs, move BSON to mpr"
```

---

## Task 11：验收——import guard + 整体测试

- [ ] **Step 1: 检查 import guard 的剩余违规数量**

```bash
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v 2>&1 | grep "forbidden import"
```

预期：仅剩 2 个文件（`cmd_diff_local.go` 和 `flowbuilder_raw_setter_gen.go`）——这两个是 Batch 3 的范围，在本计划中**有意延迟**。其余 10 个违规文件全部清除。

- [ ] **Step 2: 将 2 个剩余文件添加到 guard 的豁免注释**

在 `mdl/executor/import_guard_test.go` 中，更新测试，将 2 个已知延迟文件排除：

```go
// known Batch-3 files: diff/raw-setter investigation pending
batchThreeFiles := map[string]bool{
    "cmd_diff_local.go":           true,
    "flowbuilder_raw_setter_gen.go": true,
}

for _, e := range entries {
    name := e.Name()
    if e.IsDir() || strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
        continue
    }
    if batchThreeFiles[name] {
        continue // Batch 3 deferred — see docs/superpowers/specs/2026-05-27-executor-bson-cleanup-design.md
    }
    // ... 其余检查不变
}
```

- [ ] **Step 3: import guard 应完全通过**

```bash
go test ./mdl/executor/... -run TestNoDirectBSONImportInExecutor -v
```

预期：PASS（2 个 Batch 3 文件被豁免）。

- [ ] **Step 4: 跑完整测试套件**

```bash
make test
```

预期：所有测试通过（无新增失败）。

- [ ] **Step 5: 最终验证**

```bash
# 检查 10 个应清除的文件已清除
for f in cmd_microflows_format_calls_gen.go cmd_microflows_format_data_gen.go \
          cmd_microflows_format_external_gen.go cmd_microflows_format_action_gen.go \
          cmd_workflows_gen.go cmd_javaactions_gen.go \
          cmd_pages_describe_pluggable.go cmd_pages_describe_output.go \
          cmd_pages_describe.go cmd_pages_builder_v3.go; do
  count=$(grep -c '"go.mongodb.org/mongo-driver/bson\|modelsdk/codec' mdl/executor/$f 2>/dev/null || echo 0)
  echo "$f: $count violations"
done
```

预期：所有文件输出 `0 violations`。

- [ ] **Step 6: Commit**

```bash
git add mdl/executor/import_guard_test.go
git commit -m "chore(executor): 10/12 BSON violations cleared; guard updated with Batch-3 allowlist"
```

---

## Batch 3 调研备忘（第二阶段，本计划不实现）

以下文件推迟到第二阶段处理，需先做调研再设计：

| 文件 | 当前行为 | 调研问题 |
|-----|---------|---------|
| `cmd_diff_local.go` | 读 BSON 字节比较状态 | `RawUnitBackend.GetRawUnitBytes()` 是否足够？diff 格式化是否可走 backend？ |
| `flowbuilder_raw_setter_gen.go` | codec 写 microflow 活动字段 | 是否需要 `MicroflowMutator` 模式类比 PageMutator？ |

---

## 执行顺序总结

```
Task 1（Guard）→ Task 2（ExpressionBased）→ Task 3（format_data）→ Task 4（format_external）
→ Task 5（Workflow describe）→ Task 6（Text items + format_action）→ Task 7（Page decode backend）
→ Task 8（JavaAction backend）→ Task 9（pages_describe primitive）→ Task 10（DataGridSpec）
→ Task 11（验收 + allowlist）
```

**覆盖的 10 个文件：**
1. `cmd_microflows_format_calls_gen.go` — Task 2
2. `cmd_microflows_format_data_gen.go` — Task 3
3. `cmd_microflows_format_external_gen.go` — Task 4
4. `cmd_workflows_gen.go` — Task 5
5. `cmd_microflows_format_action_gen.go` — Task 6
6. `cmd_pages_describe_pluggable.go` — Task 7
7. `cmd_pages_describe_output.go` — Task 7
8. `cmd_javaactions_gen.go` — Task 8
9. `cmd_pages_describe.go` — Task 9
10. `cmd_pages_builder_v3.go` — Task 10

**延迟到 Batch 3 的 2 个文件（guard allowlist）：**
- `cmd_diff_local.go` — 需调研 diff 操作是否走 RawUnitBackend
- `flowbuilder_raw_setter_gen.go` — 需调研是否引入 MicroflowMutator 模式

每个 Task 完成后运行 import guard 验证进度。不要跳跃执行——每个 Task 都依赖前一个 Task 编译通过。
