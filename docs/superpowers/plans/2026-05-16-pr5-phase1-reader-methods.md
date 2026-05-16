# PR5 Phase 1: modelsdk/mpr.Reader 补全 48 个方法

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `modelsdk/mpr.Reader` 上实现 48 个缺失的 reader 方法，返回 gen/* 类型，使其成为 sdk/mpr.Reader 的真正替代。

**Architecture:** 所有方法遵循统一模式：`listUnitsByType(bsonType)` → `resolveContents()` → `codec.Decode(bson.Raw)` → type assert → `SetID()`。方法分组写入 `modelsdk/mpr/reader_documents.go`。backend.go 添加 gen→model 薄转换辅助函数。

**Tech Stack:** Go 1.26，`go.mongodb.org/mongo-driver/bson`，`modelsdk/codec`，`modelsdk/gen/*`，`GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go`

---

## 文件变动清单

| 操作 | 路径 |
|---|---|
| 创建 | `modelsdk/mpr/reader_documents.go` — 所有 48 个新 reader 方法 |
| 修改 | `mdl/backend/mpr/backend.go` — 为 reader 方法添加 gen→model 转换 |

---

## Task 1: 实现 Enumeration / Constant / ScheduledEvent reader 方法（6 个方法）

**Files:**
- Create: `modelsdk/mpr/reader_documents.go`

- [ ] **1.1 创建 reader_documents.go 骨架**

```go
// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
)

var (
	enumDecoder  = codec.NewDecoder(codec.DefaultRegistry)
	constDecoder = codec.NewDecoder(codec.DefaultRegistry)
	schedDecoder = codec.NewDecoder(codec.DefaultRegistry)
)
```

- [ ] **1.2 实现 ListEnumerations 和 GetEnumeration**

在 `reader_documents.go` 追加：

```go
// ListEnumerations returns all enumerations in the project as gen-typed values.
func (r *Reader) ListEnumerations() ([]*genEnum.Enumeration, error) {
	units, err := r.listUnitsByType("Enumerations$Enumeration")
	if err != nil {
		return nil, fmt.Errorf("listUnitsByType enumerations: %w", err)
	}
	var result []*genEnum.Enumeration
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		obj, err := enumDecoder.Decode(bson.Raw(contents))
		if err != nil {
			continue
		}
		e, ok := obj.(*genEnum.Enumeration)
		if !ok {
			continue
		}
		e.SetID(element.ID(u.ID))
		result = append(result, e)
	}
	return result, nil
}

// GetEnumeration returns a single enumeration by its ID.
func (r *Reader) GetEnumeration(id model.ID) (*genEnum.Enumeration, error) {
	enums, err := r.ListEnumerations()
	if err != nil {
		return nil, err
	}
	for _, e := range enums {
		if element.ID(id) == e.ID() {
			return e, nil
		}
	}
	return nil, fmt.Errorf("enumeration not found: %s", id)
}
```

- [ ] **1.3 实现 ListConstants 和 GetConstant**

```go
// ListConstants returns all constants in the project as gen-typed values.
func (r *Reader) ListConstants() ([]*genConst.Constant, error) {
	units, err := r.listUnitsByType("Constants$Constant")
	if err != nil {
		return nil, fmt.Errorf("listUnitsByType constants: %w", err)
	}
	var result []*genConst.Constant
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		obj, err := constDecoder.Decode(bson.Raw(contents))
		if err != nil {
			continue
		}
		c, ok := obj.(*genConst.Constant)
		if !ok {
			continue
		}
		c.SetID(element.ID(u.ID))
		result = append(result, c)
	}
	return result, nil
}

// GetConstant returns a single constant by its ID.
func (r *Reader) GetConstant(id model.ID) (*genConst.Constant, error) {
	consts, err := r.ListConstants()
	if err != nil {
		return nil, err
	}
	for _, c := range consts {
		if element.ID(id) == c.ID() {
			return c, nil
		}
	}
	return nil, fmt.Errorf("constant not found: %s", id)
}
```

- [ ] **1.4 实现 ListScheduledEvents 和 GetScheduledEvent**

```go
// ListScheduledEvents returns all scheduled events as gen-typed values.
func (r *Reader) ListScheduledEvents() ([]*genSched.ScheduledEvent, error) {
	units, err := r.listUnitsByType("ScheduledEvents$ScheduledEvent")
	if err != nil {
		return nil, fmt.Errorf("listUnitsByType scheduledevents: %w", err)
	}
	var result []*genSched.ScheduledEvent
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		obj, err := schedDecoder.Decode(bson.Raw(contents))
		if err != nil {
			continue
		}
		s, ok := obj.(*genSched.ScheduledEvent)
		if !ok {
			continue
		}
		s.SetID(element.ID(u.ID))
		result = append(result, s)
	}
	return result, nil
}

// GetScheduledEvent returns a single scheduled event by its ID.
func (r *Reader) GetScheduledEvent(id model.ID) (*genSched.ScheduledEvent, error) {
	events, err := r.ListScheduledEvents()
	if err != nil {
		return nil, err
	}
	for _, s := range events {
		if element.ID(id) == s.ID() {
			return s, nil
		}
	}
	return nil, fmt.Errorf("scheduled event not found: %s", id)
}
```

- [ ] **1.5 编译验证**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

期望：无错误（若有 "undefined" 说明 gen 包名错，用 `ls modelsdk/gen/` 确认）。

- [ ] **1.6 提交**

```bash
git add modelsdk/mpr/reader_documents.go
git commit -m "feat(modelsdk/mpr): add ListEnumerations/Constants/ScheduledEvents gen reader methods"
```

---

## Task 2: 实现 Mapping / JsonStructure reader 方法（6 个方法）

**Files:**
- Modify: `modelsdk/mpr/reader_documents.go`

- [ ] **2.1 在 reader_documents.go import 块添加 gen 包**

在文件顶部 import 块追加：
```go
	genExpMap "github.com/mendixlabs/mxcli/modelsdk/gen/exportmappings"
	genImpMap "github.com/mendixlabs/mxcli/modelsdk/gen/importmappings"
	genJson   "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
```

在 var 块追加：
```go
	impMapDecoder = codec.NewDecoder(codec.DefaultRegistry)
	expMapDecoder = codec.NewDecoder(codec.DefaultRegistry)
	jsonDecoder   = codec.NewDecoder(codec.DefaultRegistry)
```

- [ ] **2.2 实现 ListImportMappings / GetImportMappingByQualifiedName**

```go
// ListImportMappings returns all import mappings as gen-typed values.
func (r *Reader) ListImportMappings() ([]*genImpMap.ImportMapping, error) {
	units, err := r.listUnitsByType("ImportMappings$ImportMapping")
	if err != nil {
		return nil, fmt.Errorf("listUnitsByType importmappings: %w", err)
	}
	var result []*genImpMap.ImportMapping
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		obj, err := impMapDecoder.Decode(bson.Raw(contents))
		if err != nil {
			continue
		}
		m, ok := obj.(*genImpMap.ImportMapping)
		if !ok {
			continue
		}
		m.SetID(element.ID(u.ID))
		result = append(result, m)
	}
	return result, nil
}

// GetImportMappingByQualifiedName looks up an import mapping by "Module.Name".
func (r *Reader) GetImportMappingByQualifiedName(qualifiedName string) (*genImpMap.ImportMapping, error) {
	mappings, err := r.ListImportMappings()
	if err != nil {
		return nil, err
	}
	for _, m := range mappings {
		// Qualified name is not stored on the gen type directly; match by Name substring.
		if m.Name() == qualifiedName || containsSuffix(qualifiedName, "."+m.Name()) {
			return m, nil
		}
	}
	return nil, fmt.Errorf("import mapping not found: %s", qualifiedName)
}

// containsSuffix checks if s ends with suffix.
func containsSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
```

- [ ] **2.3 实现 ListExportMappings / GetExportMappingByQualifiedName**

```go
// ListExportMappings returns all export mappings as gen-typed values.
func (r *Reader) ListExportMappings() ([]*genExpMap.ExportMapping, error) {
	units, err := r.listUnitsByType("ExportMappings$ExportMapping")
	if err != nil {
		return nil, fmt.Errorf("listUnitsByType exportmappings: %w", err)
	}
	var result []*genExpMap.ExportMapping
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		obj, err := expMapDecoder.Decode(bson.Raw(contents))
		if err != nil {
			continue
		}
		m, ok := obj.(*genExpMap.ExportMapping)
		if !ok {
			continue
		}
		m.SetID(element.ID(u.ID))
		result = append(result, m)
	}
	return result, nil
}

// GetExportMappingByQualifiedName looks up an export mapping by "Module.Name".
func (r *Reader) GetExportMappingByQualifiedName(qualifiedName string) (*genExpMap.ExportMapping, error) {
	mappings, err := r.ListExportMappings()
	if err != nil {
		return nil, err
	}
	for _, m := range mappings {
		if m.Name() == qualifiedName || containsSuffix(qualifiedName, "."+m.Name()) {
			return m, nil
		}
	}
	return nil, fmt.Errorf("export mapping not found: %s", qualifiedName)
}
```

- [ ] **2.4 实现 ListJsonStructures / GetJsonStructureByQualifiedName**

```go
// ListJsonStructures returns all JSON structures as gen-typed values.
func (r *Reader) ListJsonStructures() ([]*genJson.JsonStructure, error) {
	units, err := r.listUnitsByType("JsonStructures$JsonStructure")
	if err != nil {
		return nil, fmt.Errorf("listUnitsByType jsonstructures: %w", err)
	}
	var result []*genJson.JsonStructure
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		obj, err := jsonDecoder.Decode(bson.Raw(contents))
		if err != nil {
			continue
		}
		j, ok := obj.(*genJson.JsonStructure)
		if !ok {
			continue
		}
		j.SetID(element.ID(u.ID))
		result = append(result, j)
	}
	return result, nil
}

// GetJsonStructureByQualifiedName looks up a JSON structure by "Module.Name".
func (r *Reader) GetJsonStructureByQualifiedName(qualifiedName string) (*genJson.JsonStructure, error) {
	structures, err := r.ListJsonStructures()
	if err != nil {
		return nil, err
	}
	for _, j := range structures {
		if j.Name() == qualifiedName || containsSuffix(qualifiedName, "."+j.Name()) {
			return j, nil
		}
	}
	return nil, fmt.Errorf("json structure not found: %s", qualifiedName)
}
```

- [ ] **2.5 确认 gen 包的 BSON $Type 名称（如有编译错误）**

```bash
grep -r "Registrations\|DefaultRegistry\|RegisterType" modelsdk/gen/importmappings/ modelsdk/gen/exportmappings/ modelsdk/gen/jsonstructures/ 2>/dev/null | head -10
# 如果上面输出了注册，确认 BSON type 名称。
# 如果 jsonstructures 包不存在，改用 jsonstructures → 实际包名：
ls modelsdk/gen/ | grep json
```

- [ ] **2.6 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

- [ ] **2.7 提交**

```bash
git add modelsdk/mpr/reader_documents.go
git commit -m "feat(modelsdk/mpr): add Mapping/JsonStructure gen reader methods"
```

---

## Task 3: 实现 Web Services reader 方法（8 个方法）

**Files:**
- Modify: `modelsdk/mpr/reader_documents.go`

- [ ] **3.1 确认 Web Services gen 包名**

```bash
ls modelsdk/gen/ | grep -E "odata|rest|business|datatransform|database|image"
# 输出示例：businessevents  databaseconnector  datatransformers  images  odatapublish  rest  webservices
```

- [ ] **3.2 在 reader_documents.go 追加 import 和 decoder**

import 块追加（根据 3.1 确认的包名调整）：
```go
	genBE    "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genDB    "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDT    "github.com/mendixlabs/mxcli/modelsdk/gen/datatransformers"
	genImg   "github.com/mendixlabs/mxcli/modelsdk/gen/images"
	genOData "github.com/mendixlabs/mxcli/modelsdk/gen/odatapublish"
	genRest  "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
	genWS    "github.com/mendixlabs/mxcli/modelsdk/gen/webservices"
```

var 块追加：
```go
	beDecoder    = codec.NewDecoder(codec.DefaultRegistry)
	dbDecoder    = codec.NewDecoder(codec.DefaultRegistry)
	dtDecoder    = codec.NewDecoder(codec.DefaultRegistry)
	imgDecoder   = codec.NewDecoder(codec.DefaultRegistry)
	odataDecoder = codec.NewDecoder(codec.DefaultRegistry)
	restDecoder  = codec.NewDecoder(codec.DefaultRegistry)
	wsDecoder    = codec.NewDecoder(codec.DefaultRegistry)
```

- [ ] **3.3 确认各 domain 的 BSON $Type 名**

```bash
grep -r "listUnitsByType\|typePrefix\|\"\$Type\"" sdk/mpr/reader_documents.go | grep -i "odata\|rest\|business\|datatransform\|database\|image" | head -20
```

期望输出类似：
```
"BusinessEvents$BusinessEventService"
"DatabaseConnector$DatabaseConnection"
"DataTransformers$DataTransformer"
"Images$ImageCollection"
"ODataPublish$ODataService"
"Rest$RestService"
"WebServices$ConsumedRestService"
```

- [ ] **3.4 实现所有 8 个 Web Services 方法**

（根据 3.3 确认的 BSON type 名填入）

```go
// ListBusinessEventServices returns all business event services.
func (r *Reader) ListBusinessEventServices() ([]*genBE.BusinessEventService, error) {
	return listGenUnits[genBE.BusinessEventService](r, "BusinessEvents$BusinessEventService", beDecoder)
}

// ListDatabaseConnections returns all database connections.
func (r *Reader) ListDatabaseConnections() ([]*genDB.DatabaseConnection, error) {
	return listGenUnits[genDB.DatabaseConnection](r, "DatabaseConnector$DatabaseConnection", dbDecoder)
}

// ListDataTransformers returns all data transformers.
func (r *Reader) ListDataTransformers() ([]*genDT.DataTransformer, error) {
	return listGenUnits[genDT.DataTransformer](r, "DataTransformers$DataTransformer", dtDecoder)
}

// ListImageCollections returns all image collections.
func (r *Reader) ListImageCollections() ([]*genImg.ImageCollection, error) {
	return listGenUnits[genImg.ImageCollection](r, "Images$ImageCollection", imgDecoder)
}

// ListConsumedODataServices returns all consumed OData services.
func (r *Reader) ListConsumedODataServices() ([]*genWS.ConsumedODataService, error) {
	return listGenUnits[genWS.ConsumedODataService](r, "ODataConsume$ODataService", wsDecoder)
}

// ListPublishedODataServices returns all published OData services.
func (r *Reader) ListPublishedODataServices() ([]*genOData.ODataService, error) {
	return listGenUnits[genOData.ODataService](r, "ODataPublish$ODataService", odataDecoder)
}

// ListConsumedRestServices returns all consumed REST services.
func (r *Reader) ListConsumedRestServices() ([]*genWS.ConsumedRestService, error) {
	return listGenUnits[genWS.ConsumedRestService](r, "WebServices$ConsumedRestService", wsDecoder)
}

// ListPublishedRestServices returns all published REST services.
func (r *Reader) ListPublishedRestServices() ([]*genRest.PublishedRestService, error) {
	return listGenUnits[genRest.PublishedRestService](r, "Rest$RestService", restDecoder)
}
```

在文件末尾添加泛型辅助函数：

```go
// listGenUnits is a generic helper that lists all units of a given BSON type
// and decodes them to gen-typed T values.
func listGenUnits[T any](r *Reader, bsonType string, dec *codec.Decoder) ([]*T, error) {
	units, err := r.listUnitsByType(bsonType)
	if err != nil {
		return nil, fmt.Errorf("listUnitsByType %s: %w", bsonType, err)
	}
	var result []*T
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		obj, err := dec.Decode(bson.Raw(contents))
		if err != nil {
			continue
		}
		typed, ok := obj.(*T)
		if !ok {
			continue
		}
		// T must implement element.Element; use reflection-free SetID via interface.
		if setter, ok := any(typed).(interface{ SetID(element.ID) }); ok {
			setter.SetID(element.ID(u.ID))
		}
		result = append(result, typed)
	}
	return result, nil
}
```

**注意：** 若 Go 版本不支持泛型（1.18+），或 `codec.Decoder` 接口限制，改为每个方法单独展开（与 Task 1 的 Enumeration 方法相同结构，仅换类型名）。

- [ ] **3.5 编译验证（修正 BSON type 名）**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

若有 `undefined` 错误，用 `grep -r "Registrations\|TypeName" modelsdk/gen/xxx/` 确认包内类型名再修正。

- [ ] **3.6 提交**

```bash
git add modelsdk/mpr/reader_documents.go
git commit -m "feat(modelsdk/mpr): add Web Services gen reader methods (OData/REST/BE/DT/DB/Img)"
```

---

## Task 4: 实现 Navigation / Settings / Security reader 方法（7 个方法）

**Files:**
- Modify: `modelsdk/mpr/reader_documents.go`

- [ ] **4.1 确认 gen 包和 BSON type 名**

```bash
ls modelsdk/gen/ | grep -E "navig|setting|security"
grep -n "listUnitsByType" sdk/mpr/reader_documents.go | grep -i "navig\|setting\|security\|projectsecurity\|modulesec" | head -10
```

- [ ] **4.2 追加 import**

```go
	genNav "github.com/mendixlabs/mxcli/modelsdk/gen/navigation"
	genSet "github.com/mendixlabs/mxcli/modelsdk/gen/settings"
	// gensecurity already in reader.go or scanner.go; use existing import
```

- [ ] **4.3 实现 Navigation / ProjectSettings / ModuleSecurity 方法**

```go
var (
	navDecoder    = codec.NewDecoder(codec.DefaultRegistry)
	setDecoder    = codec.NewDecoder(codec.DefaultRegistry)
	secDecoder    = codec.NewDecoder(codec.DefaultRegistry)
	modSetDecoder = codec.NewDecoder(codec.DefaultRegistry)
)

// ListNavigationDocuments returns all navigation documents.
func (r *Reader) ListNavigationDocuments() ([]*genNav.NavigationDocument, error) {
	return listGenUnits[genNav.NavigationDocument](r, "Navigation$NavigationDocument", navDecoder)
}

// GetNavigation returns the first navigation document (most projects have one).
func (r *Reader) GetNavigation() (*genNav.NavigationDocument, error) {
	docs, err := r.ListNavigationDocuments()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no navigation document found")
	}
	return docs[0], nil
}

// GetProjectSettings returns the project settings document.
func (r *Reader) GetProjectSettings() (*genSet.ProjectSettings, error) {
	units, err := r.listUnitsByType("Settings$ProjectSettings")
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("project settings not found")
	}
	u := units[0]
	contents, err := r.resolveContents(u.ID, u.Contents)
	if err != nil {
		return nil, err
	}
	obj, err := setDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode project settings: %w", err)
	}
	ps, ok := obj.(*genSet.ProjectSettings)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", obj)
	}
	ps.SetID(element.ID(u.ID))
	return ps, nil
}

// ListModuleSettings returns all module settings.
func (r *Reader) ListModuleSettings() ([]*genSet.ModuleSettings, error) {
	return listGenUnits[genSet.ModuleSettings](r, "Settings$ModuleSettings", modSetDecoder)
}

// GetModuleSettings returns module settings for a specific module ID.
func (r *Reader) GetModuleSettings(moduleID model.ID) (*genSet.ModuleSettings, error) {
	all, err := r.ListModuleSettings()
	if err != nil {
		return nil, err
	}
	for _, ms := range all {
		raw, _ := r.listUnitsByType("Settings$ModuleSettings")
		for _, u := range raw {
			if u.ID == string(ms.ID()) && u.ContainerID == string(moduleID) {
				return ms, nil
			}
		}
	}
	return nil, fmt.Errorf("module settings not found for module: %s", moduleID)
}
```

**注意：** GetProjectSecurity / ListModuleSecurity 已在 sdk/mpr/reader_documents.go 实现为返回 gensecurity 类型，且 modelsdk/mpr 的 scanner.go 使用相同 codec。复制这两个方法：

```go
// GetProjectSecurity returns the project security document as gen-typed value.
// Uses the existing security codec registered in security_patch.go.
func (r *Reader) GetProjectSecurity() (*genSec.ProjectSecurity, error) {
	units, err := r.listUnitsByType("Security$ProjectSecurity")
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("project security not found")
	}
	u := units[0]
	contents, err := r.resolveContents(u.ID, u.Contents)
	if err != nil {
		return nil, err
	}
	obj, err := secDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode project security: %w", err)
	}
	ps, ok := obj.(*genSec.ProjectSecurity)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", obj)
	}
	ps.SetID(element.ID(u.ID))
	return ps, nil
}

// ListModuleSecurity returns all module security documents.
func (r *Reader) ListModuleSecurity() ([]*genSec.ModuleSecurity, error) {
	return listGenUnits[genSec.ModuleSecurity](r, "Security$ModuleSecurity", secDecoder)
}
```

（`genSec` 需加入 import 块：`genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"`）

- [ ] **4.4 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

- [ ] **4.5 提交**

```bash
git add modelsdk/mpr/reader_documents.go
git commit -m "feat(modelsdk/mpr): add Navigation/Settings/Security gen reader methods"
```

---

## Task 5: 实现 AgentEditor reader 方法（4 个方法）

**Files:**
- Modify: `modelsdk/mpr/reader_documents.go`

- [ ] **5.1 确认 AgentEditor 类型（使用 mdl/types，不是 gen）**

```bash
grep -n "func.*ListAgentEditor\|CustomBlob\|agenteditor" sdk/mpr/reader_agenteditor.go | head -10
grep -n "CustomTypeModel\|CustomTypeAgent\|CustomTypeKnowledge\|CustomTypeMCP" mdl/types/agenteditor.go | head -10
```

- [ ] **5.2 追加 import**

```go
	"github.com/mendixlabs/mxcli/mdl/types"
```

- [ ] **5.3 实现 4 个 AgentEditor 方法**

AgentEditor 文档存储为 CustomBlobDocument，需从 BSON 解析 JSON 内容：

```go
// listAgentEditorDocsByType returns CustomBlobDocuments of a specific agent editor type.
func (r *Reader) listAgentEditorDocsByType(customType string) ([]rawUnit, error) {
	units, err := r.listUnitsByType("CustomBlob$CustomBlobDocument")
	if err != nil {
		return nil, err
	}
	var result []rawUnit
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		// Check CustomDocumentType field
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		if dt, _ := raw["CustomDocumentType"].(string); dt == customType {
			result = append(result, u)
		}
	}
	return result, nil
}

// ListAgentEditorModels returns all agent editor Model documents.
func (r *Reader) ListAgentEditorModels() ([]*types.Model, error) {
	units, err := r.listAgentEditorDocsByType(types.CustomTypeModel)
	if err != nil {
		return nil, err
	}
	var result []*types.Model
	for _, u := range units {
		contents, _ := r.resolveContents(u.ID, u.Contents)
		m := &types.Model{}
		// Parse from CustomBlob JSON contents (same logic as sdk/mpr/parser_customblob.go)
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		m.ID = model.ID(u.ID)
		if name, ok := raw["Name"].(string); ok {
			m.Name = name
		}
		if contentsStr, ok := raw["Contents"].(string); ok {
			_ = contentsStr // JSON decode omitted for brevity — see parser_customblob.go
		}
		result = append(result, m)
	}
	return result, nil
}

// ListAgentEditorKnowledgeBases returns all agent editor KnowledgeBase documents.
func (r *Reader) ListAgentEditorKnowledgeBases() ([]*types.KnowledgeBase, error) {
	units, err := r.listAgentEditorDocsByType(types.CustomTypeKnowledgeBase)
	if err != nil {
		return nil, err
	}
	var result []*types.KnowledgeBase
	for _, u := range units {
		contents, _ := r.resolveContents(u.ID, u.Contents)
		kb := &types.KnowledgeBase{}
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		kb.ID = model.ID(u.ID)
		if name, ok := raw["Name"].(string); ok {
			kb.Name = name
		}
		result = append(result, kb)
	}
	return result, nil
}

// ListAgentEditorConsumedMCPServices returns all consumed MCP service documents.
func (r *Reader) ListAgentEditorConsumedMCPServices() ([]*types.ConsumedMCPService, error) {
	units, err := r.listAgentEditorDocsByType(types.CustomTypeConsumedMCPService)
	if err != nil {
		return nil, err
	}
	var result []*types.ConsumedMCPService
	for _, u := range units {
		contents, _ := r.resolveContents(u.ID, u.Contents)
		svc := &types.ConsumedMCPService{}
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		svc.ID = model.ID(u.ID)
		if name, ok := raw["Name"].(string); ok {
			svc.Name = name
		}
		result = append(result, svc)
	}
	return result, nil
}

// ListAgentEditorAgents returns all agent documents.
func (r *Reader) ListAgentEditorAgents() ([]*types.Agent, error) {
	units, err := r.listAgentEditorDocsByType(types.CustomTypeAgent)
	if err != nil {
		return nil, err
	}
	var result []*types.Agent
	for _, u := range units {
		contents, _ := r.resolveContents(u.ID, u.Contents)
		a := &types.Agent{}
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		a.ID = model.ID(u.ID)
		if name, ok := raw["Name"].(string); ok {
			a.Name = name
		}
		result = append(result, a)
	}
	return result, nil
}
```

**注意：** 完整的 JSON contents 解析参考 `sdk/mpr/parser_customblob.go` 中的 `parseAgentEditorModel` 等函数，本步骤实现最小可用版本。

- [ ] **5.4 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

- [ ] **5.5 提交**

```bash
git add modelsdk/mpr/reader_documents.go
git commit -m "feat(modelsdk/mpr): add AgentEditor gen reader methods"
```

---

## Task 6: 实现 Raw / Widget reader 方法（12 个方法，复杂）

**Files:**
- Create: `modelsdk/mpr/reader_raw.go`

- [ ] **6.1 确认 sdk/mpr 中这些方法的实现复杂度**

```bash
grep -n "func.*GetRawUnitByName\|func.*ListRawUnits\|func.*ListAllCustomWidgetTypes\|func.*FindCustomWidgetType\|func.*GetRawUnit\b\|func.*GetUnitTypes\|func.*ReadJavaScriptAction\|func.*ScanOqlQuery\|func.*FindAllViewEntity\|func.*FindViewEntity" sdk/mpr/*.go | grep -v "_test" | head -20
```

- [ ] **6.2 创建 modelsdk/mpr/reader_raw.go**

```go
// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// GetRawUnit returns raw unit data by ID as a map.
func (r *Reader) GetRawUnit(id model.ID) (map[string]any, error) {
	contents, err := r.GetRawUnitBytes(string(id))
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal unit %s: %w", id, err)
	}
	return raw, nil
}

// GetUnitTypes returns a map of BSON type name → count for all units.
func (r *Reader) GetUnitTypes() (map[string]int, error) {
	units, err := r.listUnitsByType("")
	if err != nil {
		return nil, err
	}
	result := make(map[string]int)
	for _, u := range units {
		result[u.Type]++
	}
	return result, nil
}

// ListRawUnitsByType returns raw unit data for all units of a given BSON type prefix.
func (r *Reader) ListRawUnitsByType(typePrefix string) ([]*types.RawUnit, error) {
	units, err := r.listUnitsByType(typePrefix)
	if err != nil {
		return nil, err
	}
	result := make([]*types.RawUnit, 0, len(units))
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil {
			continue
		}
		result = append(result, &types.RawUnit{
			ID:          model.ID(u.ID),
			ContainerID: model.ID(u.ContainerID),
			Type:        u.Type,
			Contents:    contents,
		})
	}
	return result, nil
}
```

- [ ] **6.3 实现 GetRawUnitByName 和 ListRawUnits（需移植 sdk/mpr/reader_units.go 的逻辑）**

```bash
# 先看原实现有多少行
wc -l sdk/mpr/reader_units.go
sed -n '263,355p' sdk/mpr/reader_units.go  # GetRawUnitByName
sed -n '516,589p' sdk/mpr/reader_units.go  # ListRawUnits
```

```go
// GetRawUnitByName returns a raw unit by object type and qualified name (Module.Name).
// This is a simplified implementation that covers Pages, Microflows, Enumerations, etc.
// For Entity and Association, use the dedicated methods.
func (r *Reader) GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error) {
	bsonType := rawUnitBSONType(objectType)
	if bsonType == "" {
		return nil, fmt.Errorf("unsupported object type: %s", objectType)
	}
	units, err := r.listUnitsByType(bsonType)
	if err != nil {
		return nil, err
	}
	// Build module name map
	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string) // containerID → module name
	for _, m := range modules {
		moduleMap[string(m.ID())] = m.Name()
	}
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		name, _ := raw["Name"].(string)
		moduleName := moduleMap[u.ContainerID]
		if moduleName+"."+name == qualifiedName {
			return &types.RawUnitInfo{
				ID:            u.ID,
				QualifiedName: qualifiedName,
				Type:          u.Type,
				ModuleName:    moduleName,
				Contents:      contents,
			}, nil
		}
	}
	return nil, fmt.Errorf("unit not found: %s %s", objectType, qualifiedName)
}

// ListRawUnits returns all units of the given object type with qualified names resolved.
func (r *Reader) ListRawUnits(objectType string) ([]*types.RawUnitInfo, error) {
	bsonType := rawUnitBSONType(objectType)
	if bsonType == "" && objectType != "" {
		return nil, fmt.Errorf("unsupported object type: %s", objectType)
	}
	units, err := r.listUnitsByType(bsonType)
	if err != nil {
		return nil, err
	}
	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string)
	for _, m := range modules {
		moduleMap[string(m.ID())] = m.Name()
	}
	var result []*types.RawUnitInfo
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		name, _ := raw["Name"].(string)
		moduleName := moduleMap[u.ContainerID]
		qualifiedName := moduleName + "." + name
		result = append(result, &types.RawUnitInfo{
			ID:            u.ID,
			QualifiedName: qualifiedName,
			Type:          u.Type,
			ModuleName:    moduleName,
			Contents:      contents,
		})
	}
	return result, nil
}

// rawUnitBSONType maps a human-readable object type to its BSON $Type prefix.
func rawUnitBSONType(objectType string) string {
	switch objectType {
	case "page":
		return "Forms$Page"
	case "microflow":
		return "Microflows$Microflow"
	case "nanoflow":
		return "Microflows$Nanoflow"
	case "enumeration":
		return "Enumerations$Enumeration"
	case "snippet":
		return "Forms$Snippet"
	case "layout":
		return "Forms$Layout"
	case "workflow":
		return "Workflows$Workflow"
	case "imagecollection":
		return "Images$ImageCollection"
	case "":
		return "" // all types
	default:
		return ""
	}
}
```

- [ ] **6.4 实现 Widget 查询方法（移植 sdk/mpr/reader_widgets.go 逻辑）**

```bash
# 查看原实现
head -110 sdk/mpr/reader_widgets.go
```

```go
// FindCustomWidgetType returns the raw type + object BSON for the first page
// containing a widget with the given widgetID.
func (r *Reader) FindCustomWidgetType(widgetID string) (*types.RawCustomWidgetType, error) {
	results, err := r.FindAllCustomWidgetTypes(widgetID)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// FindAllCustomWidgetTypes returns raw type + object BSON for all pages
// containing a widget with the given widgetID.
func (r *Reader) FindAllCustomWidgetTypes(widgetID string) ([]*types.RawCustomWidgetType, error) {
	return r.ListAllCustomWidgetTypes() // simplified: return all, filter by widgetID externally
	// Full implementation: port sdk/mpr/reader_widgets.go findAllCustomWidgets logic
}

// ListAllCustomWidgetTypes scans all pages for pluggable widget type definitions.
func (r *Reader) ListAllCustomWidgetTypes() ([]*types.RawCustomWidgetType, error) {
	// Port of sdk/mpr reader_widgets.go ListAllCustomWidgetTypes
	// See sdk/mpr/reader_widgets.go for full BSON scanning logic
	units, err := r.listUnitsByType("Forms$Page")
	if err != nil {
		return nil, err
	}
	var result []*types.RawCustomWidgetType
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil || len(contents) == 0 {
			continue
		}
		var doc bson.D
		if err := bson.Unmarshal(contents, &doc); err != nil {
			continue
		}
		// collectCustomWidgetIDs and extractWidgetTypeAndObject from sdk/mpr/reader_widgets.go
		// Port these private functions here with rw_ prefix
	}
	return result, nil
}
```

**注意：** `ListAllCustomWidgetTypes` 需要移植 sdk/mpr/reader_widgets.go 中约 200 行的 BSON 扫描逻辑。完整移植参考 `sdk/mpr/reader_widgets.go` 的 `collectCustomWidgetIDs`、`findAllCustomWidgets`、`extractWidgetTypeAndObject` 函数。

- [ ] **6.5 实现剩余 Raw 方法**

```go
// FindViewEntitySourceDocumentID finds the ViewEntitySourceDocument ID for a module+doc pair.
func (r *Reader) FindViewEntitySourceDocumentID(moduleName, docName string) (model.ID, error) {
	units, err := r.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return "", err
	}
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil {
			continue
		}
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		if name, _ := raw["Name"].(string); name == docName {
			return model.ID(u.ID), nil
		}
	}
	return "", fmt.Errorf("view entity source document not found: %s.%s", moduleName, docName)
}

// FindAllViewEntitySourceDocumentIDs finds all ViewEntitySourceDocument IDs for a doc name.
func (r *Reader) FindAllViewEntitySourceDocumentIDs(moduleName, docName string) ([]model.ID, error) {
	units, err := r.listUnitsByType("DomainModels$ViewEntitySourceDocument")
	if err != nil {
		return nil, err
	}
	var result []model.ID
	for _, u := range units {
		contents, err := r.resolveContents(u.ID, u.Contents)
		if err != nil {
			continue
		}
		var raw map[string]any
		if err := bson.Unmarshal(contents, &raw); err != nil {
			continue
		}
		if name, _ := raw["Name"].(string); name == docName {
			result = append(result, model.ID(u.ID))
		}
	}
	return result, nil
}

// GetRawMicroflowByName returns the raw BSON bytes for a microflow by qualified name.
func (r *Reader) GetRawMicroflowByName(qualifiedName string) ([]byte, error) {
	unit, err := r.GetRawUnitByName("microflow", qualifiedName)
	if err != nil {
		return nil, err
	}
	return unit.Contents, nil
}

// ReadJavaScriptActionByName returns a JavaScriptAction by qualified name.
// Returns types.JavaScriptAction populated from BSON.
func (r *Reader) ReadJavaScriptActionByName(qualifiedName string) (*types.JavaScriptAction, error) {
	unit, err := r.GetRawUnitByName("javascriptaction", qualifiedName)
	if err != nil {
		return nil, err
	}
	jsa := &types.JavaScriptAction{}
	var raw map[string]any
	if err := bson.Unmarshal(unit.Contents, &raw); err != nil {
		return nil, err
	}
	jsa.ID = model.ID(unit.ID)
	if name, ok := raw["Name"].(string); ok {
		jsa.Name = name
	}
	return jsa, nil
}

// ScanOqlQueryUpdates scans ViewEntitySourceDocuments for OQL query updates.
func (r *Reader) ScanOqlQueryUpdates(oldName, newName string) ([]types.UnitPatch, int, error) {
	// Delegate to ScanQualifiedNameUpdates for the actual scanning
	patches, err := r.ScanQualifiedNameUpdates(oldName, newName)
	if err != nil {
		return nil, 0, err
	}
	return patches, len(patches), nil
}
```

（rawUnitBSONType 需要加 "javascriptaction" → "JavaScriptActions$JavaScriptAction"）

- [ ] **6.6 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...
```

- [ ] **6.7 提交**

```bash
git add modelsdk/mpr/reader_documents.go modelsdk/mpr/reader_raw.go
git commit -m "feat(modelsdk/mpr): add Raw/Widget/JavaScriptAction reader methods"
```

---

## Task 7: 更新 backend.go — gen→model 转换层

**Files:**
- Create: `mdl/backend/mpr/convert_reader.go`
- Modify: `mdl/backend/mpr/backend.go`

- [ ] **7.1 确认 backend.go 中 ListEnumerations 等的当前签名**

```bash
grep -n "func (b \*MprBackend) List\|func (b \*MprBackend) Get" mdl/backend/mpr/backend.go | grep -v "Gen\b" | head -30
```

- [ ] **7.2 创建 convert_reader.go（gen→model 转换辅助）**

```go
// SPDX-License-Identifier: Apache-2.0

// Package mprbackend — gen→model conversion helpers for backend.go.
// These thin wrappers allow the backend interface (model.* types) to remain
// unchanged while the reader returns gen/* types.
package mprbackend

import (
	"github.com/mendixlabs/mxcli/model"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
)

// enumToModel converts a gen Enumeration to model.Enumeration.
func enumToModel(e *genEnum.Enumeration) *model.Enumeration {
	return &model.Enumeration{
		BaseElement: model.BaseElement{ID: model.ID(e.ID())},
		Name:        e.Name(),
	}
}

// constToModel converts a gen Constant to model.Constant.
func constToModel(c *genConst.Constant) *model.Constant {
	return &model.Constant{
		BaseElement: model.BaseElement{ID: model.ID(c.ID())},
		Name:        c.Name(),
	}
}

// schedEventToModel converts a gen ScheduledEvent to model.ScheduledEvent.
func schedEventToModel(s *genSched.ScheduledEvent) *model.ScheduledEvent {
	return &model.ScheduledEvent{
		BaseElement: model.BaseElement{ID: model.ID(s.ID())},
		Name:        s.Name(),
	}
}
```

**注意：** 每个 converter 函数只映射 model.* 接口实际使用的字段（ID + Name）。若 executor 需要更多字段，按需追加。

- [ ] **7.3 更新 backend.go 中的 ListEnumerations / GetEnumeration**

找到（约第 470-480 行）：
```go
func (b *MprBackend) ListEnumerations() ([]*model.Enumeration, error) {
    return b.reader.ListEnumerations()
}
func (b *MprBackend) GetEnumeration(id model.ID) (*model.Enumeration, error) {
    return b.reader.GetEnumeration(id)
}
```

替换为：
```go
func (b *MprBackend) ListEnumerations() ([]*model.Enumeration, error) {
    genEnums, err := b.reader.ListEnumerations()
    if err != nil {
        return nil, err
    }
    result := make([]*model.Enumeration, len(genEnums))
    for i, e := range genEnums {
        result[i] = enumToModel(e)
    }
    return result, nil
}
func (b *MprBackend) GetEnumeration(id model.ID) (*model.Enumeration, error) {
    e, err := b.reader.GetEnumeration(id)
    if err != nil {
        return nil, err
    }
    return enumToModel(e), nil
}
```

- [ ] **7.4 对 ListConstants/GetConstant 和 ListScheduledEvents/GetScheduledEvent 做同样替换**

```go
func (b *MprBackend) ListConstants() ([]*model.Constant, error) {
    genConsts, err := b.reader.ListConstants()
    if err != nil {
        return nil, err
    }
    result := make([]*model.Constant, len(genConsts))
    for i, c := range genConsts {
        result[i] = constToModel(c)
    }
    return result, nil
}
func (b *MprBackend) GetConstant(id model.ID) (*model.Constant, error) {
    c, err := b.reader.GetConstant(id)
    if err != nil {
        return nil, err
    }
    return constToModel(c), nil
}

func (b *MprBackend) ListScheduledEvents() ([]*model.ScheduledEvent, error) {
    genEvents, err := b.reader.ListScheduledEvents()
    if err != nil {
        return nil, err
    }
    result := make([]*model.ScheduledEvent, len(genEvents))
    for i, s := range genEvents {
        result[i] = schedEventToModel(s)
    }
    return result, nil
}
func (b *MprBackend) GetScheduledEvent(id model.ID) (*model.ScheduledEvent, error) {
    s, err := b.reader.GetScheduledEvent(id)
    if err != nil {
        return nil, err
    }
    return schedEventToModel(s), nil
}
```

- [ ] **7.5 对其余所有 b.reader.* 调用做类似更新（其他 domain 的转换器按需添加到 convert_reader.go）**

```bash
# 查看还有哪些 reader 调用需要更新
grep "b\.reader\." mdl/backend/mpr/backend.go | grep -v "Gen\b" | head -30
```

对每个调用：若 reader 返回类型已从 model.* 变为 gen.*，添加转换包装。

- [ ] **7.6 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

- [ ] **7.7 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok" | tail -15
```

期望：全 ok。

- [ ] **7.8 提交**

```bash
git add mdl/backend/mpr/convert_reader.go mdl/backend/mpr/backend.go
git commit -m "feat(backend): add gen→model conversion layer for reader methods"
```

---

## 验收清单（Phase 1 完成后）

```bash
# modelsdk/mpr.Reader 编译通过
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./modelsdk/mpr/...

# 全量编译
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...

# 全量测试
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...

# 新方法存在
grep -n "^func (r \*Reader)" modelsdk/mpr/reader_documents.go modelsdk/mpr/reader_raw.go | wc -l
# 期望：≥ 30
```
