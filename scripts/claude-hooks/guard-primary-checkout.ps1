#Requires -Version 5.1
<#
.SYNOPSIS
    Guarda de hooks de Claude Code que protege el checkout principal de Axiom.

.DESCRIPTION
    Política local para desarrollar este repositorio (no es comportamiento del producto):
    el checkout principal (p. ej. C:\repos\axiom) está siempre en `main` y solo recibe
    fast-forwards; todo cambio nace en un worktree creado con scripts/axiom-worktree.ps1.

    Lee por stdin el JSON del hook (hook_event_name, tool_name, tool_input, cwd) y actúa así:

    - PreToolUse con Write, Edit o NotebookEdit: si la ruta de destino cae dentro del
      checkout principal (y no dentro de su `.claude\`, que es configuración local no
      versionada), la deniega con exit 2 y explica cómo crear el worktree.
    - PostToolUse con Bash o PowerShell: si el checkout principal ha quedado fuera de `main`
      (rama distinta o HEAD desacoplado), o si `main` tiene commits que no están en
      `origin/main`, devuelve el aviso a Claude con exit 2 y la forma de corregirlo.
    - Cualquier otro evento o herramienta: no hace nada.

    La comparación de rutas incluye el separador final: `C:\repos\axiom-wt\...` NO se
    considera dentro de `C:\repos\axiom`. Admite rutas `C:\...`, `C:/...` y `/c/...`.

    La guarda falla en abierto: si git no responde, el cwd no es un repositorio, el JSON
    es ilegible o hay cualquier excepción, termina con exit 0 sin bloquear nada.

.NOTES
    Registro (en el `.claude/settings.json` local del checkout principal, no versionado).
    Ajusta la ruta absoluta al checkout principal de cada máquina:

    {
      "hooks": {
        "PreToolUse": [
          {
            "matcher": "Write|Edit|NotebookEdit",
            "hooks": [
              { "type": "command", "command": "powershell -NoProfile -ExecutionPolicy Bypass -File \"C:/repos/axiom/scripts/claude-hooks/guard-primary-checkout.ps1\"" }
            ]
          }
        ],
        "PostToolUse": [
          {
            "matcher": "Bash|PowerShell",
            "hooks": [
              { "type": "command", "command": "powershell -NoProfile -ExecutionPolicy Bypass -File \"C:/repos/axiom/scripts/claude-hooks/guard-primary-checkout.ps1\"" }
            ]
          }
        ]
      }
    }

    Códigos de salida:
      0  permitido (o estado desconocido: fallo en abierto)
      2  bloqueado; el motivo se escribe por stderr y Claude Code lo devuelve al agente
#>

$ErrorActionPreference = 'Continue'
# Claude Code lee stderr como UTF-8; sin esto los acentos salen en la página de códigos OEM.
try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false) } catch { }

function Write-Reason([string]$Message) {
    [Console]::Error.WriteLine($Message)
}

# Convierte /c/..., C:/... y rutas relativas (respecto a $Base) en una ruta absoluta
# de Windows normalizada y sin separador final.
function ConvertTo-NormalPath([string]$Path, [string]$Base) {
    if ([string]::IsNullOrWhiteSpace($Path)) { return $null }
    $p = $Path.Trim()
    if ($p -match '^/([a-zA-Z])(/.*)?$') {
        $rest = $Matches[2]
        if ([string]::IsNullOrEmpty($rest)) { $rest = '/' }
        $p = $Matches[1].ToUpper() + ':' + $rest
    }
    $p = $p.Replace('/', '\')
    if (-not [System.IO.Path]::IsPathRooted($p)) {
        if ([string]::IsNullOrWhiteSpace($Base)) { return $null }
        $p = Join-Path $Base $p
    }
    return [System.IO.Path]::GetFullPath($p).TrimEnd('\')
}

# Verdadero si $Path es $Root o está dentro de él (el prefijo incluye el separador final).
function Test-InsidePath([string]$Path, [string]$Root) {
    if ([string]::IsNullOrEmpty($Path) -or [string]::IsNullOrEmpty($Root)) { return $false }
    return $Path.Equals($Root, [StringComparison]::OrdinalIgnoreCase) -or
        $Path.StartsWith($Root + '\', [StringComparison]::OrdinalIgnoreCase)
}

try {
    try {
        $reader = New-Object System.IO.StreamReader([Console]::OpenStandardInput(), [System.Text.Encoding]::UTF8)
        $raw = $reader.ReadToEnd()
    } catch {
        exit 0
    }
    if ([string]::IsNullOrWhiteSpace($raw)) { exit 0 }

    try {
        $data = $raw | ConvertFrom-Json
    } catch {
        exit 0
    }
    if ($null -eq $data) { exit 0 }

    $eventName = [string]$data.hook_event_name
    $tool  = [string]$data.tool_name
    $cwd   = ConvertTo-NormalPath ([string]$data.cwd) (Get-Location).Path
    if ([string]::IsNullOrEmpty($cwd)) { $cwd = (Get-Location).Path }

    $isPre  = ($eventName -eq 'PreToolUse')  -and ($tool -match '^(Write|Edit|NotebookEdit)$')
    $isPost = ($eventName -eq 'PostToolUse') -and ($tool -match '^(Bash|PowerShell)$')
    if (-not ($isPre -or $isPost)) { exit 0 }

    # Raíz del checkout principal = padre del git common dir (<principal>\.git).
    $commonRaw = git -C $cwd rev-parse --path-format=absolute --git-common-dir 2>$null
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($commonRaw)) { exit 0 }
    $commonDir = ConvertTo-NormalPath ([string]($commonRaw | Select-Object -First 1)) $cwd
    if ([string]::IsNullOrEmpty($commonDir)) { exit 0 }
    if (-not ([System.IO.Path]::GetFileName($commonDir)).Equals('.git', [StringComparison]::OrdinalIgnoreCase)) { exit 0 }
    $primary = [System.IO.Path]::GetDirectoryName($commonDir)
    if ([string]::IsNullOrEmpty($primary)) { exit 0 }
    $primary = $primary.TrimEnd('\')

    if ($isPre) {
        $ti = $data.tool_input
        if ($null -eq $ti) { exit 0 }
        $target = [string]$ti.file_path
        if ([string]::IsNullOrWhiteSpace($target)) { $target = [string]$ti.notebook_path }
        $targetPath = ConvertTo-NormalPath $target $cwd
        if ([string]::IsNullOrEmpty($targetPath)) { exit 0 }

        if ((Test-InsidePath $targetPath $primary) -and -not (Test-InsidePath $targetPath (Join-Path $primary '.claude'))) {
            Write-Reason ("Bloqueado: '$targetPath' está dentro del checkout principal '$primary'. " +
                "El checkout principal está siempre en main y no se edita; todo cambio (código, openspec/ y odd/tasks/) " +
                "se hace en un worktree propio. Crea uno con: " +
                "powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/axiom-worktree.ps1 new <slug> [-Prefix feat|inc|fix|refactor] " +
                "y edita con rutas absolutas dentro de C:\repos\axiom-wt\<slug> (o usa git -C). " +
                "Nunca uses git checkout -b ni EnterWorktree en el checkout principal.")
            exit 2
        }
        exit 0
    }

    # PostToolUse: comprobar el estado del checkout principal.
    $branch = git -C $primary symbolic-ref --quiet --short HEAD 2>$null
    $code = $LASTEXITCODE
    if ($code -eq 0 -and -not [string]::IsNullOrWhiteSpace($branch)) {
        $branch = ([string]($branch | Select-Object -First 1)).Trim()
    } elseif ($code -eq 1) {
        $branch = '(HEAD desacoplado)'
    } else {
        exit 0
    }

    if ($branch -ne 'main') {
        Write-Reason ("El checkout principal '$primary' ha quedado en '$branch' y debe estar siempre en main. " +
            "Restáuralo con: git -C `"$primary`" switch main " +
            "(si había trabajo en curso, tras volver a main muévelo a un worktree, p. ej. " +
            "git -C `"$primary`" worktree add ..\axiom-wt\<slug> <rama>) y continúa en el worktree. " +
            "Para cambios nuevos usa scripts/axiom-worktree.ps1 new <slug>.")
        exit 2
    }

    $ahead = git -C $primary rev-list --count origin/main..main 2>$null
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($ahead)) { exit 0 }
    $aheadCount = 0
    if (-not [int]::TryParse(([string]($ahead | Select-Object -First 1)).Trim(), [ref]$aheadCount)) { exit 0 }
    if ($aheadCount -gt 0) {
        Write-Reason ("La rama main del checkout principal '$primary' tiene $aheadCount commit(s) que no están en origin/main. " +
            "Está prohibido commitear directamente en main: mueve esos commits a una rama en un worktree " +
            "(git -C `"$primary`" worktree add ..\axiom-wt\<slug> -b <rama>, y después restablece main a origin/main " +
            "cuando los commits estén a salvo) y abre el PR con scripts/axiom-worktree.ps1 pr <slug>.")
        exit 2
    }
    exit 0
} catch {
    exit 0
}
