#!/usr/bin/env bash
# scripts/perf-bench.sh
#
# 手动触发的端到端性能测试，覆盖 microflow / entity / page / workflow 的
# 冷启动（首次）与热启动（多次重复）时间分布。
#
# 用法:
#   ./scripts/perf-bench.sh                  # 使用默认项目 (expr-checker)
#   ./scripts/perf-bench.sh --large          # 使用 corpus-b（较大，更真实）
#   ./scripts/perf-bench.sh --runs 12        # 热启动重复次数（默认 8）
#   ./scripts/perf-bench.sh --cold 3         # 冷启动重复次数（默认 3）
#
# 依赖: go（用于 go run）或预构建的 bin/mxcli

set -euo pipefail

# ── 参数解析 ──────────────────────────────────────────────────
USE_LARGE=false
RUNS=8
COLD_RUNS=3

while [[ $# -gt 0 ]]; do
  case "$1" in
    --large) USE_LARGE=true ;;
    --runs)  RUNS="$2";      shift ;;
    --cold)  COLD_RUNS="$2"; shift ;;
    -h|--help)
      sed -n '2,14p' "$0" | sed 's/^# \?//'
      exit 0 ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
  shift
done

# ── 准备 mxcli 二进制 ─────────────────────────────────────────
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MXCLI="$REPO_ROOT/bin/mxcli"

if [[ ! -x "$MXCLI" ]]; then
  echo "正在构建 mxcli..."
  (cd "$REPO_ROOT" && go build -o bin/mxcli ./cmd/mxcli)
fi

# ── 选择源项目 ────────────────────────────────────────────────
if $USE_LARGE; then
  SRC="$REPO_ROOT/testdata/corpus-b"
  MPR_NAME="app.mpr"
  PROJECT_LABEL="corpus-b (179 MB, 4 153 mxunit)"
  MODULE="BenchModule"
else
  SRC="$REPO_ROOT/testdata/expr-checker"
  MPR_NAME="minimal.mpr"
  PROJECT_LABEL="expr-checker (22 MB, 轻量)"
  MODULE="MyFirstModule"
fi

# ── 创建测试目录（hardlink 加速，不复制文件内容）─────────────
TESTDIR=$(mktemp -d /tmp/mxcli-perf-XXXXXX)
trap 'rm -rf "$TESTDIR"' EXIT

echo "准备测试项目 → $TESTDIR  (hardlink 复制)"
cp "$SRC/$MPR_NAME" "$TESTDIR/"

# mprcontents 用 hardlink 避免复制大量文件
if [[ -d "$SRC/mprcontents" ]]; then
  mkdir -p "$TESTDIR/mprcontents"
  find "$SRC/mprcontents" -type f | while read -r f; do
    rel="${f#$SRC/}"
    mkdir -p "$TESTDIR/$(dirname "$rel")"
    ln "$f" "$TESTDIR/$rel" 2>/dev/null || cp "$f" "$TESTDIR/$rel"
  done
fi

MPR="$TESTDIR/$MPR_NAME"

# ── 工具函数 ──────────────────────────────────────────────────
ms_now()  { date +%s%3N; }

# 执行单条 MDL，返回耗时(ms)。stdout/stderr 均静默。
run_one() {
  local mdl="$1"
  local t0 t1
  t0=$(ms_now)
  "$MXCLI" -p "$MPR" -c "$mdl" >/dev/null 2>&1
  t1=$(ms_now)
  echo $((t1 - t0))
}

# bench_series <label> <n> <mdl with IDX placeholder>
# 运行 n 次，每次用序号替换 IDX，打印统计。
bench_series() {
  local label="$1" n="$2" mdl_tpl="$3"
  local total=0 min=99999 max=0 raw=()

  for i in $(seq 1 "$n"); do
    local mdl="${mdl_tpl//IDX/$i}"
    local t
    t=$(run_one "$mdl")
    raw+=("${t}ms")
    total=$((total + t))
    (( t < min )) && min=$t
    (( t > max )) && max=$t
  done

  local avg=$((total / n))
  printf "  %-44s  min=%4dms  avg=%4dms  max=%4dms\n" \
    "$label" "$min" "$avg" "$max"
  printf "    样本: [%s]\n" "$(IFS=', '; echo "${raw[*]}")"
}

# ─────────────────────────────────────────────────────────────
echo ""
echo "╔════════════════════════════════════════════════════════╗"
echo "║         mxcli 端到端性能基准   —  手动验证脚本         ║"
echo "╚════════════════════════════════════════════════════════╝"
echo "  项目  : $PROJECT_LABEL"
echo "  模块  : $MODULE"
echo "  热启动: ${RUNS} 次 / 冷启动: ${COLD_RUNS} 次"
echo ""

# ══════════════════════════════════════════════════════════════
# A. 冷启动  — OS page cache 新鲜（或已清除）
# ══════════════════════════════════════════════════════════════
echo "┌─ A. 冷启动（首次访问，page cache 冷）─────────────────┐"

# 尝试清 cache；没权限则给出提示
if sudo sh -c 'sync && echo 3 > /proc/sys/vm/drop_caches' 2>/dev/null; then
  echo "│  (page cache 已清除)"
else
  echo "│  ⚠ 无 root 权限，无法清 cache；冷启动数据仅供参考"
fi
echo "│"

bench_series "A1 create microflow (冷)" "$COLD_RUNS" \
  "create or modify microflow ${MODULE}.ACT_Cold_IDX () returns Nothing begin return; end;"

bench_series "A2 create entity (冷)" "$COLD_RUNS" \
  "create or modify persistent entity ${MODULE}.ColdEnt_IDX (Name: String(200));"

echo "└───────────────────────────────────────────────────────┘"

# ══════════════════════════════════════════════════════════════
# B. 热启动  — page cache 已热，重复执行
# ══════════════════════════════════════════════════════════════
echo ""
echo "┌─ B. 热启动（page cache 热，重复 ${RUNS} 次）──────────┐"
echo "│"
echo "│  ── B1. Microflow ─────────────────────────────────── │"

bench_series "create () returns Nothing" "$RUNS" \
  "create or modify microflow ${MODULE}.ACT_MfNoop_IDX () returns Nothing begin return; end;"

bench_series "create (\$P: String) returns String" "$RUNS" \
  "create or modify microflow ${MODULE}.ACT_MfStr_IDX (\$P: String) returns String begin return \$P; end;"

bench_series "create with log + return Boolean" "$RUNS" \
  "create or modify microflow ${MODULE}.ACT_MfLog_IDX (\$Msg: String) returns Boolean begin log info node 'Bench' \$Msg; return true; end;"

echo "│"
echo "│  ── B2. Entity ────────────────────────────────────── │"

bench_series "create 1 attr (String)" "$RUNS" \
  "create or modify persistent entity ${MODULE}.EntA_IDX (Name: String(200));"

bench_series "create 4 attrs (mixed types)" "$RUNS" \
  "create or modify persistent entity ${MODULE}.EntB_IDX (Name: String(200), Age: Integer, Active: Boolean default false, Score: Decimal);"

bench_series "alter entity add attr" "$RUNS" \
  "alter entity ${MODULE}.EntA_1 add (Tag: String(50));"

echo "│"
echo "│  ── B3. Page ──────────────────────────────────────── │"

# Page 依赖 entity，确保 EntA_1 存在（A2 或 B2 冷启动已创建）
bench_series "create page (dataview, empty)" "$RUNS" \
  "create or modify page ${MODULE}.PageA_IDX (title: 'Page IDX', layout: Atlas_Core.Atlas_Default) begin content: dataview (entity: ${MODULE}.EntA_1) begin end; end;"

echo "│"
echo "│  ── B4. Workflow ──────────────────────────────────── │"

bench_series "create workflow (1 usertask)" "$RUNS" \
  "create or modify workflow ${MODULE}.WF_IDX begin usertask Step1 (caption: 'Step IDX', allowed roles: []) begin end; end;"

echo "│"
echo "│  ── B5. DESCRIBE / 只读路径 ───────────────────────── │"

bench_series "describe microflow" "$RUNS" \
  "describe microflow ${MODULE}.ACT_MfNoop_1;"

bench_series "describe entity" "$RUNS" \
  "describe entity ${MODULE}.EntB_1;"

bench_series "show microflows in module" "$RUNS" \
  "show microflows in ${MODULE};"

echo "│"
echo "│  ── B6. 混合脚本（单次 -c，多条语句）──────────────── │"

bench_series "entity + microflow (单次调用)" "$RUNS" \
  "create or modify persistent entity ${MODULE}.ComboEnt_IDX (Val: Integer); create or modify microflow ${MODULE}.ACT_Combo_IDX () returns Nothing begin return; end;"

echo "└───────────────────────────────────────────────────────┘"
echo ""
echo "  完成。如需更大项目，请加 --large 参数重新运行。"
echo ""
