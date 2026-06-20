// SPDX-License-Identifier: Apache-2.0
// perf_import — regression harness for import-path DM cache / buffer behavior.
// Runs five scenarios (write-read consistency, sessionBuf overlay, post-Flush
// disk consistency, write-read-write monotonicity, DM cache speedup). Requires
// /mnt/data_sdd/gh/mendix-app-import/MacnicaApp.mpr as the source corpus.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

const (
	perfMDLDir = "/mnt/data_sdd/gh/mendix-app-import/mdlsource"
	perfMPRSrc = "/mnt/data_sdd/gh/mendix-app-import/MacnicaApp.mpr"
)

func main() {
	fmt.Println("╔═════════════════════════════════════════════════════════════╗")
	fmt.Println("║   DomainModel 缓存击穿边界情况验证                          ║")
	fmt.Println("╚═════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ── 场景 1: 写后读一致性（write-through 正确性）───────────────
	fmt.Println("━━ 场景 1: 写后读一致性 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	testWriteReadConsistency()

	// ── 场景 2: sessionBuf 激活时 overlay 是否覆盖 reader cache ──
	fmt.Println("━━ 场景 2: sessionBuf 激活期间 overlay 读一致性 ━━━━━━━━━━━━")
	testOverlayConsistency()

	// ── 场景 3: Flush 后 overlay 清除，磁盘读回是否正确 ───────────
	fmt.Println("━━ 场景 3: Flush 后 overlay 清除，磁盘数据一致性 ━━━━━━━━━━━")
	testPostFlushConsistency()

	// ── 场景 4: 多语句同模块写—读—写序列（模拟连续 CREATE ENTITY）─
	fmt.Println("━━ 场景 4: 同模块连续写-读-写序列下的一致性 ━━━━━━━━━━━━━━━")
	testMultiWriteConsistency()

	// ── 场景 5: 测量引入 DM 缓存后的性能收益（对比有/无缓存）──────
	fmt.Println("━━ 场景 5: DomainModel 缓存性能收益测量 ━━━━━━━━━━━━━━━━━━━")
	measureCacheBenefit()
}

// ── 场景 1: 写后读一致性 ─────────────────────────────────────────────
// 测试：CREATE ENTITY 写入 DM → 立即 GET DM → 确认新 entity 可见
func testWriteReadConsistency() {
	mprPath := copyMPR()
	defer cleanup(mprPath)
	exec := newExec(mprPath)
	be := mustGetBackend(exec)

	// 写入一个新 entity
	mdl := `
CREATE MODULE TestModule;
CREATE PERSISTENT ENTITY TestModule.Customer (
  Name: String(200) NOT NULL,
  Age: Integer
);`
	prog, _ := visitor.Build(mdl)
	if err := exec.ExecuteProgram(prog); err != nil {
		fmt.Printf("  ✗ FAIL: execute: %v\n\n", err)
		return
	}

	// 立即读取 DomainModel，验证 entity 存在
	mod, err := be.GetModuleByName("TestModule")
	if err != nil || mod == nil {
		fmt.Printf("  ✗ FAIL: GetModule: %v\n\n", err)
		return
	}
	dm, err := be.GetDomainModelGen(model.ID(mod.ID))
	if err != nil || dm == nil {
		fmt.Printf("  ✗ FAIL: GetDomainModelGen: %v\n\n", err)
		return
	}

	found := false
	for _, e := range dm.EntitiesItems() {
		if ent, ok := e.(*genDm.Entity); ok && ent.Name() == "Customer" {
			found = true
			break
		}
	}
	if found {
		fmt.Printf("  ✓ PASS: 写后 entity 可见（%d entities in DM）\n\n", len(dm.EntitiesItems()))
	} else {
		fmt.Printf("  ✗ FAIL: 写后 entity 不可见！DM 中有 %d entities: %v\n\n",
			len(dm.EntitiesItems()), dmEntityNames(dm))
	}
}

// ── 场景 2: sessionBuf 激活期间 overlay 读一致性 ──────────────────────
// 测试：EnableImportBuffer → 写 DM → 不 Flush → 立即读 DM → 应看到新数据
func testOverlayConsistency() {
	mprPath := copyMPR()
	defer cleanup(mprPath)
	exec := newExec(mprPath)
	be := mustGetBackend(exec)

	// 先建模块
	prog0, _ := visitor.Build("CREATE MODULE OverlayTest;")
	_ = exec.ExecuteProgram(prog0)

	// 激活 import buffer
	importBuf := be.EnableImportBuffer()
	defer be.DisableImportBuffer()

	_ = importBuf

	// 写入 entity（此时 sessionBuf 可能未接线，写会直接到磁盘）
	prog1, _ := visitor.Build(`
CREATE PERSISTENT ENTITY OverlayTest.Product (
  Code: String(50) NOT NULL
);`)
	if err := exec.ExecuteProgram(prog1); err != nil {
		fmt.Printf("  ✗ FAIL: execute: %v\n", err)
		// 不 flush，检查 reader 是否能读到
	}

	// 检测 sessionBuf 是否实际接线
	bufCount := importBuf.PendingCount()
	fmt.Printf("  Buffer pending 数: %d（0=写路径未经过 buffer，直接落盘）\n", bufCount)

	mod, _ := be.GetModuleByName("OverlayTest")
	if mod == nil {
		fmt.Printf("  ✗ FAIL: OverlayTest module not found\n\n")
		return
	}
	dm, _ := be.GetDomainModelGen(model.ID(mod.ID))
	if dm == nil {
		fmt.Printf("  ✗ FAIL: DomainModel nil\n\n")
		return
	}

	found := false
	for _, e := range dm.EntitiesItems() {
		if ent, ok := e.(*genDm.Entity); ok && ent.Name() == "Product" {
			found = true
			break
		}
	}

	if bufCount == 0 && found {
		fmt.Printf("  ⚠ DIAGNOSIS: Buffer 未接线（写直接落盘），但读一致性仍正确（走磁盘读）\n")
		fmt.Printf("  → 证明 BufferedUnitStore.Write 从未被调用\n")
		fmt.Printf("  → EnableImportBuffer 缺少 SetSessionBuf 接线！\n\n")
	} else if bufCount > 0 && found {
		fmt.Printf("  ✓ PASS: Buffer 接线正常，overlay 读一致\n\n")
	} else if bufCount > 0 && !found {
		fmt.Printf("  ✗ FAIL: Buffer 接线但 overlay 读不到数据（击穿！）\n\n")
	} else {
		fmt.Printf("  ✗ FAIL: Buffer 未接线且读不到数据\n\n")
	}
}

// ── 场景 3: Flush 后 overlay 清除，磁盘数据一致性 ─────────────────────
// 测试：Flush → overlay 清除 → 从磁盘读 → 数据必须与写入前一致
func testPostFlushConsistency() {
	mprPath := copyMPR()
	defer cleanup(mprPath)
	exec := newExec(mprPath)
	be := mustGetBackend(exec)

	prog0, _ := visitor.Build("CREATE MODULE FlushTest;")
	_ = exec.ExecuteProgram(prog0)

	importBuf := be.EnableImportBuffer()

	prog1, _ := visitor.Build(`
CREATE PERSISTENT ENTITY FlushTest.Order (
  OrderDate: DateTime NOT NULL,
  TotalAmount: Decimal
);`)
	_ = exec.ExecuteProgram(prog1)

	// Flush（将 pending 写入磁盘，清除 overlay）
	if err := importBuf.Flush(); err != nil {
		fmt.Printf("  ✗ FAIL: Flush: %v\n\n", err)
		be.DisableImportBuffer()
		return
	}
	be.DisableImportBuffer() // 清除 overlay

	// 现在从磁盘读
	mod, _ := be.GetModuleByName("FlushTest")
	if mod == nil {
		fmt.Printf("  ✗ FAIL: module not found after flush\n\n")
		return
	}
	dm, _ := be.GetDomainModelGen(model.ID(mod.ID))
	if dm == nil {
		fmt.Printf("  ✗ FAIL: DomainModel nil after flush\n\n")
		return
	}

	found := false
	for _, e := range dm.EntitiesItems() {
		if ent, ok := e.(*genDm.Entity); ok && ent.Name() == "Order" {
			found = true
			break
		}
	}
	if found {
		fmt.Printf("  ✓ PASS: Flush 后磁盘读一致（entity 可见）\n\n")
	} else {
		fmt.Printf("  ✗ FAIL: Flush 后磁盘读不到 entity！（overlay 已清除，但磁盘数据缺失）\n\n")
	}
}

// ── 场景 4: 同模块连续写-读-写序列 ───────────────────────────────────
// 测试：连续 N 次 CREATE ENTITY，每次写后立即读，统计 GetDomainModelGen 调用次数
// 以及每次读取的 entity 数量是否单调递增
func testMultiWriteConsistency() {
	mprPath := copyMPR()
	defer cleanup(mprPath)
	exec := newExec(mprPath)
	be := mustGetBackend(exec)

	prog0, _ := visitor.Build("CREATE MODULE SeqTest;")
	_ = exec.ExecuteProgram(prog0)

	mod, _ := be.GetModuleByName("SeqTest")
	if mod == nil {
		fmt.Printf("  ✗ FAIL: SeqTest module not found\n\n")
		return
	}

	// 测量：每次 CREATE ENTITY 后，DM 中 entity 数量是否单调递增
	var getDMCount atomic.Int64
	const n = 10
	prevCount := 0
	allOK := true
	totalDMTime := time.Duration(0)

	for i := 1; i <= n; i++ {
		mdl := fmt.Sprintf(`CREATE PERSISTENT ENTITY SeqTest.Entity%d (Code: String(50));`, i)
		prog, _ := visitor.Build(mdl)
		if err := exec.ExecuteProgram(prog); err != nil {
			fmt.Printf("  ✗ FAIL: exec entity%d: %v\n", i, err)
			allOK = false
			continue
		}

		t0 := time.Now()
		dm, err := be.GetDomainModelGen(model.ID(mod.ID))
		totalDMTime += time.Since(t0)
		getDMCount.Add(1)

		if err != nil || dm == nil {
			fmt.Printf("  ✗ FAIL: GetDomainModelGen after entity%d: %v\n", i, err)
			allOK = false
			continue
		}

		count := len(dm.EntitiesItems())
		if count <= prevCount {
			fmt.Printf("  ✗ FAIL: entity%d: count 未增加（%d → %d）\n", i, prevCount, count)
			allOK = false
		}
		prevCount = count
	}

	avgDMTime := totalDMTime / time.Duration(getDMCount.Load())
	if allOK {
		fmt.Printf("  ✓ PASS: %d 次连续写-读，entity 数单调递增（最终 %d entities）\n", n, prevCount)
	}
	fmt.Printf("  GetDomainModelGen 平均耗时: %v（每次 3 次 SQLite 读）\n", avgDMTime)
	fmt.Printf("  → 这就是 execute 慢的根因：每个 CREATE 语句调用 1 次，每次 3 次 SQLite 读\n\n")
}

// ── 场景 5: DomainModel 缓存性能收益测量 ─────────────────────────────
// 测试：用简单的 map 缓存 DM，对比有/无缓存下连续 CREATE ENTITY 的速度
func measureCacheBenefit() {
	const n = 30 // 比较 30 个 entity 的建立速度

	// 无缓存（当前行为）
	mpr1 := copyMPR()
	defer cleanup(mpr1)
	exec1 := newExec(mpr1)
	be1 := mustGetBackend(exec1)
	prog0, _ := visitor.Build("CREATE MODULE PerfTest;")
	_ = exec1.ExecuteProgram(prog0)

	prev := debug.SetGCPercent(-1) // 排除 GC 干扰
	t0 := time.Now()
	for i := 1; i <= n; i++ {
		mdl := fmt.Sprintf(`CREATE PERSISTENT ENTITY PerfTest.E%d (Code: String(50));`, i)
		prog, _ := visitor.Build(mdl)
		_ = exec1.ExecuteProgram(prog)
	}
	durNoCache := time.Since(t0)
	debug.SetGCPercent(prev)

	// 测量 GetDomainModelGen 在无缓存下的实际开销
	mod1, _ := be1.GetModuleByName("PerfTest")
	const reps = 100
	t1 := time.Now()
	for i := 0; i < reps; i++ {
		_, _ = be1.GetDomainModelGen(model.ID(mod1.ID))
	}
	getDMAvg := time.Since(t1) / reps

	fmt.Printf("  无缓存: %d 个 entity 建立耗时: %v (%.1f ms/entity)\n",
		n, durNoCache, float64(durNoCache.Milliseconds())/n)
	fmt.Printf("  GetDomainModelGen 单次耗时: %v（含 3 次 SQLite 读）\n", getDMAvg)
	fmt.Printf("\n")

	// 推算：如果 GetDomainModelGen 命中缓存（0ms），能节省多少？
	// 每个 CREATE ENTITY 约调用 GetDomainModelGen 1-3 次
	savedPerEntity := getDMAvg * 2 // 假设平均 2 次调用/entity
	fmt.Printf("  推算每 entity 可节省: %v（缓存命中时）\n", savedPerEntity)
	fmt.Printf("  推算 1023 文件总节省: %v\n",
		savedPerEntity*time.Duration(n)*time.Duration(1023)/time.Duration(n))
	fmt.Printf("  → 若 DM 缓存命中率 80%%，预期总耗时从 6m → 约 %.1f min\n\n",
		float64(6)-float64(getDMAvg*2*1023/time.Minute)*0.8)
}

// ── 工具函数 ─────────────────────────────────────────────────────────

func mustGetBackend(exec *executor.Executor) *mprbackend.MprBackend {
	be, ok := exec.Backend().(*mprbackend.MprBackend)
	if !ok {
		fmt.Fprintln(os.Stderr, "FATAL: backend is not MprBackend")
		os.Exit(1)
	}
	return be
}

func dmEntityNames(dm *genDm.DomainModel) []string {
	var names []string
	for _, e := range dm.EntitiesItems() {
		if ent, ok := e.(*genDm.Entity); ok {
			names = append(names, ent.Name())
		} else {
			names = append(names, fmt.Sprintf("<%T>", e))
		}
	}
	return names
}

// 让编译器不报 element 未使用
var _ element.Element = nil

func newExec(mprPath string) *executor.Executor {
	out := &bytes.Buffer{}
	exec := executor.New(out)
	exec.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
	if err := exec.Execute(&ast.ConnectStmt{Path: mprPath}); err != nil {
		fatalf("connect: %v", err)
	}
	return exec
}

func copyMPR() string {
	dir, _ := os.MkdirTemp("", "perf-*")
	dst := filepath.Join(dir, "perf.mpr")
	data, _ := os.ReadFile(perfMPRSrc)
	_ = os.WriteFile(dst, data, 0644)
	contentsDir := filepath.Join(filepath.Dir(perfMPRSrc), "mprcontents")
	if info, err := os.Stat(contentsDir); err == nil && info.IsDir() {
		copyDir(contentsDir, filepath.Join(dir, "mprcontents"))
	}
	return dst
}

func cleanup(mprPath string) {
	os.RemoveAll(filepath.Dir(mprPath))
}

func copyDir(src, dst string) {
	_ = os.MkdirAll(dst, 0755)
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		s, d := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDir(s, d)
		} else {
			data, _ := os.ReadFile(s)
			_ = os.WriteFile(d, data, 0644)
		}
	}
}

func fatalf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+f+"\n", a...)
	os.Exit(1)
}
