#Requires -Version 5.1
<#
.SYNOPSIS
    Axiom source installer for Windows.

.DESCRIPTION
    Installs Axiom from source with Go. Official Windows binary distribution
    and Scoop are temporarily unavailable until public-trust Authenticode signing
    is enforced. Accepted channels: stable (default), beta, nightly. Needs Go
    1.25.10 or newer. It also runs piped (irm <raw script URL> | iex), the
    one-line install documented in the README; parameters live inside Main so no
    top-level param block breaks that form.

.EXAMPLE
    .\install.ps1

.EXAMPLE
    .\install.ps1 -Method go -Channel beta

.EXAMPLE
    # The legacy binary method fails closed with source-install guidance.
    .\install.ps1 -Method binary

.EXAMPLE
    # Also start the Axiom dashboard server when you sign in to Windows.
    .\install.ps1 -Autostart

.EXAMPLE
    # Install without creating any shortcut.
    .\install.ps1 -NoShortcut

.NOTES
    After a verified install on Windows the script creates an "Axiom" shortcut in
    the Start menu that runs "axiom ui --detach --app" (starts the local dashboard
    server in the background and opens it in an application window).
    -Autostart also adds a shortcut to the Startup folder ("axiom ui --detach
    --no-browser"). -NoShortcut skips both. Shortcut failures only warn.
#>

$ErrorActionPreference = "Stop"

# Ensure UTF-8 output so Unicode characters render correctly on all terminals.
$null = & chcp 65001 2>$null
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch {}

$GITHUB_OWNER = "IGutierrezZ"
$GITHUB_REPO = "axiom"
$BINARY_NAME = "axiom"
$WINDOWS_DISTRIBUTION_HOLD = "Windows binary distribution and Scoop are temporarily unavailable until publicly trusted Authenticode signing is enforced."
$STABLE_SOURCE_COMMAND = "go install github.com/IGutierrezZ/axiom/v3/cmd/axiom@latest"

function Write-Info    { param([string]$Message) Write-Host "[info]    $Message" -ForegroundColor Blue }
function Write-Success { param([string]$Message) Write-Host "[ok]      $Message" -ForegroundColor Green }
function Write-Warn    { param([string]$Message) Write-Host "[warn]    $Message" -ForegroundColor Yellow }
function Write-Err     { param([string]$Message) Write-Host "[error]   $Message" -ForegroundColor Red }
function Write-Step    { param([string]$Message) Write-Host "`n==> $Message" -ForegroundColor Cyan }

function Stop-WithError {
    param([string]$Message)
    Write-Err $Message
    # throw, not exit: under "irm | iex" the script runs in the caller's session
    # and exit would close the terminal before the error can be read.
    throw $Message
}

function Show-Banner {
    Write-Host ""
    Write-Host "     _             _                     " -ForegroundColor Cyan
    Write-Host "    / \   __  __ (_)  ___   _ __ ___     " -ForegroundColor Cyan
    Write-Host "   / _ \  \ \/ / | | / _ \ | '_ ` _ \    " -ForegroundColor Cyan
    Write-Host "  / ___ \  >  <  | || (_) || | | | | |   " -ForegroundColor Cyan
    Write-Host " /_/   \_\/_/\_\ |_| \___/ |_| |_| |_|   " -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Axiom - Ecosystem, Frameworks, Workflows" -ForegroundColor DarkGray
    Write-Host ""
}

function Get-Platform {
    Write-Step "Detecting platform"
    if (-not [Environment]::Is64BitOperatingSystem) {
        Stop-WithError "32-bit Windows is not supported."
    }
    $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
    Write-Success "Platform: Windows ($arch)"
}

function Test-Prerequisites {
    Write-Step "Checking prerequisites"
    if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
        Stop-WithError "$WINDOWS_DISTRIBUTION_HOLD Go 1.25.10 or newer is required to install from source. Install it with: winget install --id GoLang.Go -e (or from https://go.dev/dl/), open a new terminal so PATH is refreshed, then run this installer again or run: $STABLE_SOURCE_COMMAND"
    }
    Write-Success "Go is available"
}

function Get-InstallMethod {
    param([string]$Forced, [string]$Channel)

    if ($Forced -eq "binary") {
        Stop-WithError "$WINDOWS_DISTRIBUTION_HOLD No unsigned binary will be downloaded or executed. Install from source with Go 1.25.10 or newer: $STABLE_SOURCE_COMMAND"
    }
    if ($Channel -eq "beta") {
        Write-Info "Using beta channel - installing $BINARY_NAME from main via go install"
    } else {
        Write-Info "Using source installation via go install"
    }
    return "go"
}

function Install-ViaGo {
    param([string]$Channel = "stable")

    Write-Step "Installing via go install"
    $version = if ($Channel -eq "beta") { "main" } else { "latest" }
    # /v3 is part of the module path, not decoration: Go refuses to resolve a
    # module whose tags are v3.x unless the import path carries the major
    # version suffix. The owner keeps its exact case (IGutierrezZ): Go compares the
    # module path declared in go.mod case-sensitively, so a lowercased owner fails
    # with "module declares its path as ... but was required as ...".
    $goPackage = "github.com/$GITHUB_OWNER/$GITHUB_REPO/v3/cmd/$BINARY_NAME@$version"
    Write-Info "Running: go install $goPackage"

    if ($Channel -eq "beta") {
        Add-GoEnvPattern -Name "GONOSUMDB" -Pattern "github.com/IGutierrezZ/axiom/v3"
        Add-GoEnvPattern -Name "GOPRIVATE" -Pattern "github.com/IGutierrezZ/axiom/v3"
        Add-GoEnvPattern -Name "GONOPROXY" -Pattern "github.com/IGutierrezZ/axiom/v3"
    }

    & go install $goPackage
    if ($LASTEXITCODE -ne 0) {
        Stop-WithError "Failed to install via go install. Make sure Go is properly configured."
    }

    $gobin = & go env GOBIN 2>$null
    if (-not $gobin) {
        $gopath = & go env GOPATH 2>$null
        $gobin = Join-Path $gopath "bin"
    }
    if ($env:PATH -notlike "*$gobin*") {
        Write-Warn "$gobin is not in your PATH"
        Write-Warn "Add it to your PATH environment variable."
    }
    Write-Success "Installed $BINARY_NAME from source via go install"
}

function Add-GoEnvPattern {
    param(
        [string]$Name,
        [string]$Pattern
    )

    $current = [Environment]::GetEnvironmentVariable($Name, "Process")
    if (-not $current) {
        Set-Item -Path "Env:$Name" -Value $Pattern
        return
    }

    $patterns = $current.Split(",", [System.StringSplitOptions]::RemoveEmptyEntries).Trim()
    if ($patterns -contains $Pattern) { return }
    Set-Item -Path "Env:$Name" -Value ("{0},{1}" -f $Pattern, $current)
}

function Test-Installation {
    Write-Step "Verifying installation"
    $gobin = & go env GOBIN 2>$null
    if (-not $gobin) {
        $gopath = & go env GOPATH 2>$null
        $gobin = Join-Path $gopath "bin"
    }
    $binaryPath = Join-Path $gobin "$BINARY_NAME.exe"
    if (-not (Test-Path $binaryPath)) {
        Write-Warn "Could not verify $binaryPath. Check go env GOBIN and go env GOPATH."
        return
    }

    # The binary only honors GENTLE_AI_NO_SELF_UPDATE (internal/app/selfupdate.go);
    # any other name would let this version check trigger a self-update.
    $env:GENTLE_AI_NO_SELF_UPDATE = "1"
    $versionOutput = & $binaryPath --version 2>&1
    Remove-Item Env:GENTLE_AI_NO_SELF_UPDATE -ErrorAction SilentlyContinue
    Write-Success "$BINARY_NAME installed at $binaryPath`: $versionOutput"
    return $binaryPath
}

function New-AxiomShortcut {
    param(
        [string]$Path,
        [string]$Target,
        [string]$Arguments,
        [string]$Description,
        [int]$WindowStyle = 1
    )

    $dir = Split-Path -Parent $Path
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    $shell = New-Object -ComObject WScript.Shell
    $link = $shell.CreateShortcut($Path)
    $link.TargetPath = $Target
    $link.Arguments = $Arguments
    $link.WorkingDirectory = $HOME
    $link.WindowStyle = $WindowStyle
    $link.Description = $Description
    $link.IconLocation = "$Target,0"
    $link.Save()
}

function New-AxiomShortcuts {
    param(
        [string]$BinaryPath,
        [bool]$Autostart
    )

    Write-Step "Creating shortcuts"
    # Shortcuts are optional: any failure here warns and lets the install finish.
    if ($env:OS -ne "Windows_NT") {
        Write-Warn "Shortcuts are only created on Windows. Skipping."
        return $false
    }
    if (-not $BinaryPath) {
        Write-Warn "Axiom binary not verified; skipping shortcuts."
        return $false
    }
    try {
        $startMenu = Join-Path ([Environment]::GetFolderPath("Programs")) "Axiom.lnk"
        New-AxiomShortcut -Path $startMenu -Target $BinaryPath -Arguments "ui --detach --app" -Description "Abre el panel local de Axiom" -WindowStyle 7
        Write-Success "Start menu shortcut: $startMenu"
        if ($Autostart) {
            $startup = Join-Path ([Environment]::GetFolderPath("Startup")) "Axiom.lnk"
            New-AxiomShortcut -Path $startup -Target $BinaryPath -Arguments "ui --detach --no-browser" -Description "Inicia el servidor del panel local de Axiom" -WindowStyle 7
            Write-Success "Startup shortcut: $startup"
        }
        return $true
    } catch {
        Write-Warn ("Could not create shortcuts: {0}" -f $_.Exception.Message)
        return $false
    }
}

function Show-NextSteps {
    param(
        [string]$Channel = "stable",
        [bool]$Shortcuts = $false,
        [bool]$Autostart = $false
    )

    Write-Host ""
    Write-Host "Installation complete!" -ForegroundColor Green
    Write-Host ""
    if ($Channel -eq "beta") {
        Write-Host ('  Run ''$env:AXIOM_CHANNEL = "beta"; {0} install'' to keep using the beta channel' -f $BINARY_NAME) -ForegroundColor Cyan
    } else {
        Write-Host "  Run '$BINARY_NAME' to start the TUI installer" -ForegroundColor Cyan
    }
    if ($Shortcuts) {
        Write-Host "  Open the 'Axiom' shortcut in the Start menu to launch the dashboard" -ForegroundColor Cyan
        Write-Host "  (right-click it to pin it to the taskbar). In Edge or Chrome you can also" -ForegroundColor Cyan
        Write-Host "  install it as an app from the browser menu." -ForegroundColor Cyan
        if ($Autostart) {
            Write-Host "  The dashboard server will also start when you sign in to Windows." -ForegroundColor Cyan
        }
    }
    Write-Host "Docs: https://github.com/$GITHUB_OWNER/$GITHUB_REPO" -ForegroundColor DarkGray
    Write-Host ""
}

function Main {
    [CmdletBinding()]
    param(
        [ValidateSet("auto", "go", "binary")]
        [string]$Method = "auto",

        [ValidateSet("stable", "beta", "nightly")]
        [string]$Channel = $(if ($env:AXIOM_CHANNEL) { $env:AXIOM_CHANNEL } elseif ($env:GENTLE_AI_CHANNEL) { $env:GENTLE_AI_CHANNEL } else { "stable" }),

        [string]$InstallDir = "",

        [switch]$Insecure,

        [switch]$Autostart,

        [switch]$NoShortcut
    )

    Show-Banner
    if ($Insecure) {
        Stop-WithError "$WINDOWS_DISTRIBUTION_HOLD The legacy -Insecure switch cannot bypass this policy. $STABLE_SOURCE_COMMAND"
    }
    if ($InstallDir) {
        Stop-WithError "-InstallDir is unavailable for source installation. Configure go env GOBIN instead."
    }
    if ($Channel -eq "nightly") { $Channel = "beta" }

    $installMethod = Get-InstallMethod -Forced $Method -Channel $Channel
    Get-Platform
    Test-Prerequisites
    if ($installMethod -eq "go") {
        Install-ViaGo -Channel $Channel
    }
    $binaryPath = Test-Installation
    $shortcuts = $false
    if (-not $NoShortcut) {
        $shortcuts = New-AxiomShortcuts -BinaryPath $binaryPath -Autostart ([bool]$Autostart)
    }
    Show-NextSteps -Channel $Channel -Shortcuts $shortcuts -Autostart ($shortcuts -and [bool]$Autostart)
}

Main @args
