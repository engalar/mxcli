# SPDX-License-Identifier: Apache-2.0
# release-audit.ps1 — 检查三个发布项目自上一个 tag 以来的代码变更
# 用法: .\scripts\release-audit.ps1 [[-TargetRef] <string>]

param([string]$TargetRef = "HEAD")

$Components = @(
    [PSCustomObject]@{ Prefix = "v"; Display = "mxcli (Launcher)"; Paths = @("cmd/mxcli-launcher", "install.sh", "install.ps1") }
    [PSCustomObject]@{ Prefix = "daemon-v"; Display = "mxcli-daemon (Core Engine)"; Paths = @("cmd/mxcli", "mdl", "modelsdk", "internal", "sql", "modelsdk.go", "Makefile") }
    [PSCustomObject]@{ Prefix = "local-v"; Display = "mxcli-local (Local Runtime)"; Paths = @("cmd/mxcli-local", "cmd/mxcli/docker") }
)

function Get-LatestTag($prefix) {
    $tags = git tag --list "${prefix}*" --sort=-version:refname 2>$null
    if ($tags) { return $tags[0] }
    return $null
}

function Get-ChangeCount($tag, $paths) {
    $pathArgs = [System.Collections.ArrayList]@()
    foreach ($p in $paths) { [void]$pathArgs.Add("--"); [void]$pathArgs.Add($p) }
    $count = git rev-list --count "${tag}..${TargetRef}" @pathArgs 2>$null
    if ($count -eq $null) { return 0 }
    return [int]$count
}

function Get-ChangeSummary($tag, $paths, $maxCount) {
    $pathArgs = [System.Collections.ArrayList]@()
    foreach ($p in $paths) { [void]$pathArgs.Add("--"); [void]$pathArgs.Add($p) }
    return git log --oneline --no-decorate "-$maxCount" "${tag}..${TargetRef}" @pathArgs 2>$null
}

# --- Main ---
Write-Host "`n===== Release Audit =====" -ForegroundColor Cyan
Write-Host "Target: $TargetRef" -ForegroundColor Cyan
Write-Host "Time:   $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Cyan
Write-Host ""

$hasChanges = $false

foreach ($comp in $Components) {
    $tag = Get-LatestTag $comp.Prefix
    Write-Host "--- $($comp.Display) ---" -ForegroundColor White

    if (-not $tag) {
        Write-Host "  [WARN] No tag found for prefix $($comp.Prefix)*" -ForegroundColor Yellow
        Write-Host ""
        continue
    }

    Write-Host "  Latest tag:  $tag" -ForegroundColor Cyan
    $count = Get-ChangeCount $tag $comp.Paths

    if ($count -gt 0) {
        $hasChanges = $true
        Write-Host "  [CHANGED] $count commits in: $($comp.Paths -join ', ')" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Summary (top 20):"
        $summary = Get-ChangeSummary $tag $comp.Paths 20
        foreach ($line in $summary) { if ($line) { Write-Host "    $line" } }
        Write-Host ""
        Write-Host "  => RECOMMEND: git tag $($comp.Prefix)<new-version> && git push origin --tags" -ForegroundColor Cyan
        Write-Host ""
    } else {
        Write-Host "  [NO CHANGE] No relevant changes since $tag" -ForegroundColor Yellow
        Write-Host ""
    }
}

Write-Host "===== Summary =====" -ForegroundColor Cyan
if ($hasChanges) {
    Write-Host "  [ACTION REQUIRED] One or more components need a new release." -ForegroundColor Green
    Write-Host "  Steps:"
    Write-Host "    1. Review commit summaries above"
    Write-Host "    2. Determine version bumps (semver)"
    Write-Host "    3. Create and push tags:"
    Write-Host "       git tag v<version>         # launcher"
    Write-Host "       git tag daemon-v<version>  # daemon"
    Write-Host "       git tag local-v<version>   # local"
    Write-Host "       git push origin --tags"
    Write-Host "    4. Wait for GitHub Actions to complete"
} else {
    Write-Host "  [NO ACTION] No components need a release." -ForegroundColor Yellow
}
Write-Host ""
