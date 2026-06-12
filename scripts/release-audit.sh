#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# release-audit.sh — 检查三个发布项目自上一个 tag 以来的代码变更
#
# 用途：辅助决定是否需要为每个组件创建新的 GitHub Release。
#
# 用法：
#   ./scripts/release-audit.sh              # 默认对比 HEAD
#   ./scripts/release-audit.sh <ref>        # 对比任意 ref（如 main）
#
# 输出：结构化 Markdown，含变更摘要 + AI 决策指引。

set -euo pipefail

TARGET_REF="${1:-HEAD}"

# ──────────────── 颜色 ────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ──────────────── 组件定义 ────────────────
# 每个组件：tag 前缀 | 显示名 | 相关路径（空格分隔）
COMPONENTS=(
  "v:mxcli（Launcher）:cmd/mxcli-launcher:install.sh:install.ps1"
  "daemon-v:mxcli-daemon（核心引擎）:cmd/mxcli:mdl:modelsdk:internal:sql:modelsdk.go:Makefile"
  "local-v:mxcli-local（本地运行）:cmd/mxcli-local:cmd/mxcli/docker"
)

# ──────────────── 辅助函数 ────────────────

# 获取某组件的最新 tag
latest_tag() {
  local prefix="$1"
  git tag --list "${prefix}*" --sort=-version:refname 2>/dev/null | head -1 || true
}

# 检查某路径自 tag 以来是否有变更
has_changes_since() {
  local tag="$1"
  shift
  local paths=("$@")
  if [ ${#paths[@]} -eq 0 ]; then
    return 1
  fi
  # 检查每个路径在 tag..ref 之间是否有变更
  for p in "${paths[@]}"; do
    if git diff --quiet "$tag..$TARGET_REF" -- "$p" 2>/dev/null; then
      continue
    else
      return 0  # 有变更
    fi
  done
  return 1  # 无变更
}

# 获取变更摘要（最近 20 条相关 commit 的 subject）
changes_summary() {
  local tag="$1"
  shift
  local paths=("$@")
  local path_args=()
  for p in "${paths[@]}"; do
    path_args+=("--" "$p")
  done
  git log --oneline --no-decorate "${tag}..${TARGET_REF}" "${path_args[@]}" 2>/dev/null | head -20 || true
}

# 计数变更 commit 数
count_changes() {
  local tag="$1"
  shift
  local paths=("$@")
  local path_args=()
  for p in "${paths[@]}"; do
    path_args+=("--" "$p")
  done
  git rev-list --count "${tag}..${TARGET_REF}" "${path_args[@]}" 2>/dev/null || echo "0"
}

# ──────────────── 主流程 ────────────────

echo ""
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  Release Audit${NC}"
echo -e "${BOLD}  目标: ${CYAN}${TARGET_REF}${NC}"
echo -e "${BOLD}  时间: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

OVERALL_HAS_CHANGES=false

for comp_def in "${COMPONENTS[@]}"; do
  IFS=':' read -r prefix display_name paths_str <<< "$comp_def"
  IFS=' ' read -ra paths <<< "$paths_str"

  tag=$(latest_tag "$prefix")
  echo -e "${BOLD}## ${display_name}${NC}"
  echo "  Tag 前缀: ${prefix}*"
  echo ""

  if [ -z "$tag" ]; then
    echo -e "  ${YELLOW}⚠ 未找到匹配 tag${NC}"
    echo ""
    continue
  fi

  echo -e "  最新 tag:  ${CYAN}${tag}${NC}"
  echo "  相关路径: ${paths[*]}"
  echo ""

  n=$(count_changes "$tag" "${paths[@]}")
  if [ "$n" -gt 0 ]; then
    OVERALL_HAS_CHANGES=true
    echo -e "  ${GREEN}✔ 有 ${n} 个 commit 包含相关变更${NC}"
    echo ""
    echo -e "  ${BOLD}变更摘要:${NC}"
    changes_summary "$tag" "${paths[@]}" | while IFS= read -r line; do
      if [ -n "$line" ]; then
        echo "    $line"
      fi
    done
    echo ""
    echo -e "  ${BOLD}AI 决策:${NC}"
    echo -e "  → 需要创建新 tag: ${CYAN}${prefix}${NC}<新版本号>"
    echo -e "  → 工作流将自动触发 GitHub Release"
    echo ""
  else
    echo -e "  ${YELLOW}✗ 无相关变更${NC}"
    echo ""
    echo -e "  ${BOLD}AI 决策:${NC}"
    echo -e "  → 无需创建新 tag"
    echo ""
  fi

  echo "  ---"
  echo ""
done

# ──────────────── 全局建议 ────────────────
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  全局建议${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ "$OVERALL_HAS_CHANGES" = true ]; then
  echo -e "  ${GREEN}✔ 有组件需要发布${NC}"
  echo ""
  echo -e "  推荐执行步骤:"
  echo -e "  1. 审查上述变更摘要，确认版本号（遵循 semver）"
  echo -e "  2. 逐个打 tag 并推送:"
  echo ""
  echo -e "     git tag v<version>              # launcher"
  echo -e "     git tag daemon-v<version>       # daemon"
  echo -e "     git tag local-v<version>        # local"
  echo -e "     git push origin --tags"
  echo ""
  echo -e "  3. 等待 GitHub Actions 完成各 Release"
  echo -e "  4. 验证 Release 页面产物正确"
else
  echo -e "  ${YELLOW}✗ 所有组件均无变更，无需发布${NC}"
fi

echo ""
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
