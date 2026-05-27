# mxcli install script for Windows — idempotent.
# Usage: irm https://raw.githubusercontent.com/engalar/mxcli/main/install.ps1 | iex
#
# Optional: set $env:MXCLI_INSTALL_DIR before running to override install location.

$ErrorActionPreference = "Stop"
$Repo = "engalar/mxcli"

# ── Detect architecture ──────────────────────────────────────────────────────
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Error "32-bit Windows is not supported."
    exit 1
}

# ── Fetch latest release tag ─────────────────────────────────────────────────
$ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"
try {
    $Release = Invoke-RestMethod -Uri $ApiUrl -Headers @{ "User-Agent" = "mxcli-installer" }
    $Latest = $Release.tag_name
} catch {
    Write-Error "Could not fetch latest release tag from GitHub: $_"
    exit 1
}
if (-not $Latest) {
    Write-Error "Could not parse latest release tag."
    exit 1
}

# ── Idempotent version check ─────────────────────────────────────────────────
$MxcliCmd = Get-Command mxcli -ErrorAction SilentlyContinue
if ($MxcliCmd) {
    try {
        $VersionOutput = & mxcli version 2>$null | Select-Object -First 1
        $Current = ($VersionOutput -split "\s+")[-1]
    } catch {
        $Current = ""
    }
    if ($Current -eq $Latest) {
        Write-Host "✅ mxcli $Current is already up to date."
        exit 0
    }
    Write-Host "Updating mxcli $Current → $Latest"
} else {
    Write-Host "Installing mxcli $Latest"
}

# ── Determine install directory ───────────────────────────────────────────────
$InstallDir = if ($env:MXCLI_INSTALL_DIR) {
    $env:MXCLI_INSTALL_DIR
} else {
    "$env:LOCALAPPDATA\mxcli"
}
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# ── Add to user PATH (idempotent) ─────────────────────────────────────────────
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
    Write-Host "  Added $InstallDir to user PATH"
}

# ── Download launcher binary ──────────────────────────────────────────────────
$BinName = "mxcli-windows-$Arch.exe"
$BinUrl = "https://github.com/$Repo/releases/download/$Latest/$BinName"
$Dest = Join-Path $InstallDir "mxcli.exe"
$Tmp = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "mxcli-install-$([System.Guid]::NewGuid()).exe")

Write-Host "  Downloading launcher (windows/$Arch) from GitHub..."
try {
    Invoke-WebRequest -Uri $BinUrl -OutFile $Tmp -UseBasicParsing
} catch {
    Write-Error "Download failed: $_"
    exit 1
}

# Atomic install
Move-Item -Force $Tmp $Dest

Write-Host ""
Write-Host "✅ mxcli $Latest installed to $Dest"
Write-Host "   The daemon (~20 MB) will be downloaded automatically on first use."
Write-Host ""
Write-Host "   Run: mxcli version"
Write-Host "   NOTE: Restart your terminal for PATH changes to take effect."
