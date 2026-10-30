#Requires -Version 5.1
<#
.SYNOPSIS
    Envoltorio local de `axiom branch` para desarrollar Axiom.

.DESCRIPTION
    Una sola implementación de ramas y worktrees: la del producto (`axiom branch`). Este script
    aporta la costumbre local y el verbo `pr`, que el producto deja fuera (la publicación es una
    decisión de la persona):

    new  <slug> [-Prefix p]  axiom branch ensure --cwd <checkout principal> --change <slug> [--prefix p] --json
    pr   <slug>              push de la rama del cambio y PR hacia el prTarget de axiom branch status --json
    done <slug> [-Force]     axiom branch done --cwd <checkout principal> --change <slug> y, después,
                             el avance rápido de main. La rama sale del registro local que escribe
                             `ensure` (con su prefijo); `done` no admite -Prefix. Si no ve ninguna
                             rama del cambio, lo avisa en lugar de dar el cambio por desmantelado

    Los worktrees viven en <carpeta del proyecto>-wt/<slug>/<repositorio> (p. ej. axiom-wt\<slug>\axiom).
    Necesita un `axiom` con el verbo `axiom branch` en el PATH (`go install ./cmd/axiom`).

.USAGE
    scripts/axiom-worktree.ps1 new <slug> [-Prefix feat|inc|fix|refactor]
    scripts/axiom-worktree.ps1 pr <slug>
    scripts/axiom-worktree.ps1 done <slug> [-Force]
#>
param(
    [Parameter(Position = 0, Mandatory = $true)]
    [ValidateSet('new', 'pr', 'done')]
    [string]$Verb,

    [Parameter(Position = 1, Mandatory = $true)]
    [string]$Slug,

    # Sin -Prefix rige el branch_prefix de axiom.yaml (bloque git:).
    [Parameter(Position = 2, Mandatory = $false)]
    [ValidateSet('feat', 'inc', 'fix', 'refactor')]
    [string]$Prefix,

    [Parameter(Mandatory = $false)]
    [switch]$Force
)

$ErrorActionPreference = 'Continue'
# axiom escribe JSON en UTF-8; sin esto las rutas con acentos se leen en la página de códigos OEM.
try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false) } catch { }

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot  = Split-Path -Parent $scriptDir

# El checkout principal es el padre del git common dir: desde un worktree, $repoRoot ya no lo es.
$commonDir = git -C $repoRoot rev-parse --path-format=absolute --git-common-dir 2>$null
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($commonDir)) {
    $cd = (git -C $repoRoot rev-parse --git-common-dir 2>$null)
    if ($cd) {
        $commonDir = (Resolve-Path -LiteralPath (Join-Path $repoRoot $cd.Trim())).Path
    }
}
if ([string]::IsNullOrWhiteSpace($commonDir)) {
    $commonDir = Join-Path $repoRoot '.git'
}
$commonDir = ([string]($commonDir | Select-Object -First 1)).Trim()
$mainRoot  = Split-Path -Parent $commonDir

function Info { param($msg) Write-Host $msg -ForegroundColor Cyan }
function Ok   { param($msg) Write-Host $msg -ForegroundColor Green }
function Warn { param($msg) Write-Host "AVISO: $msg" -ForegroundColor Yellow }
function Fail { param($msg) Write-Host "ERROR: $msg" -ForegroundColor Red; exit 1 }

# --- Validaciones comunes ---
if ($Slug -notmatch '^[a-z0-9]+(-[a-z0-9]+)*$') {
    Fail "Slug inválido '$Slug'. Usa kebab-case en minúsculas (ej. worktree-and-pr-governance)."
}

git -C $repoRoot rev-parse --is-inside-work-tree 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) { Fail "No es un repositorio git: $repoRoot" }

# Sin un `axiom` que traiga `axiom branch` no hay nada que envolver.
$axiomCmd = Get-Command axiom -ErrorAction SilentlyContinue
if ($axiomCmd) {
    & axiom branch --help 2>&1 | Out-Null
}
if (-not $axiomCmd -or $LASTEXITCODE -ne 0) {
    Fail 'instala un `axiom` con `axiom branch` (`go install ./cmd/axiom`).'
}


switch ($Verb) {

    'new' {
        $ensureArgs = @('--mode', 'worktree')
        if ($Prefix) { $ensureArgs += @('--prefix', $Prefix) }

        # `--mode worktree`: la política local exige un worktree propio y un checkout principal
        # siempre en main, sea cual sea la preferencia personal del hub.
        $lines = & axiom branch ensure --cwd $mainRoot --change $Slug @ensureArgs --json 2>&1
        $exitCode = $LASTEXITCODE
        $text = ($lines | ForEach-Object { "$_" }) -join "`n"
        if ($exitCode -ne 0) {
            Write-Host $text
            Fail "axiom branch ensure rechazó el cambio '$Slug'."
        }
        try {
            $result = $text | ConvertFrom-Json
        } catch {
            Fail "axiom branch ensure no devolvió un JSON legible: $text"
        }

        $worktrees = @()
        foreach ($repository in @($result.repositories)) {
            foreach ($b in @($repository.branches)) {
                if ($b.worktree) { $worktrees += $b.worktree }
            }
        }
        if ($worktrees.Count -eq 0) {
            Fail "axiom branch ensure no devolvió ningún worktree para '$Slug'."
        }
        foreach ($path in $worktrees) { Ok "Worktree: $path" }
        Info 'Trabaja con rutas absolutas dentro de esos directorios (--cwd y git -C).'
        Info "Al verificar, ejecuta: scripts/axiom-worktree.ps1 pr $Slug"
    }

    'pr' {
        $lines = & axiom branch status --cwd $mainRoot --change $Slug --json 2>&1
        $exitCode = $LASTEXITCODE
        $text = ($lines | ForEach-Object { "$_" }) -join "`n"
        if ($exitCode -ne 0) {
            Write-Host $text
            Fail "axiom branch status falló para el cambio '$Slug'."
        }
        try {
            $status = $text | ConvertFrom-Json
        } catch {
            Fail "axiom branch status no devolvió un JSON legible: $text"
        }

        # Rama del cambio, su worktree y la base del PR (`prTarget`) salen del plan del producto.
        $changeBranch = $null
        foreach ($repository in @($status.repositories)) {
            foreach ($b in @($repository.branches)) {
                if ($null -eq $changeBranch -and $b.kind -eq 'change') { $changeBranch = $b }
            }
        }
        if ($null -eq $changeBranch) {
            Fail "axiom branch status no encontró la rama del cambio '$Slug'. Créala con: scripts/axiom-worktree.ps1 new $Slug"
        }
        $branch   = [string]$changeBranch.name
        $wtPath   = [string]$changeBranch.worktree
        $prTarget = [string]$changeBranch.prTarget
        if ([string]::IsNullOrWhiteSpace($wtPath) -or -not (Test-Path -LiteralPath $wtPath)) {
            Fail "La rama $branch no tiene worktree en este equipo (worktree: '$wtPath')."
        }
        if ([string]::IsNullOrWhiteSpace($prTarget)) {
            Fail "axiom branch status no propuso una rama destino (prTarget) para $branch."
        }

        $dirty = git -C $wtPath status --porcelain
        if ($dirty) {
            Fail "El worktree tiene cambios sin commitear. Haz commit de todo antes de crear el PR:`n$dirty"
        }

        Info "Subiendo $branch a origin..."
        git -C $wtPath push -u origin $branch
        if ($LASTEXITCODE -ne 0) {
            Fail "git push falló. Revisa tu conexión y permisos sobre origin."
        }

        $remoteUrl = (git -C $wtPath remote get-url origin 2>$null).Trim()
        $remoteUrl = $remoteUrl -replace '\.git$', ''
        $compareUrl = "$remoteUrl/compare/$prTarget...$branch" + '?expand=1'
        # HOST/OWNER/REPO de origin (https o ssh) para no depender de 'gh repo set-default'.
        $ghRepo = $remoteUrl -replace '^(?:[a-zA-Z][a-zA-Z0-9+.-]*://)?(?:[^@/]+@)?([^/:]+)[:/]', '$1/'

        $gh = Get-Command gh -ErrorAction SilentlyContinue
        if ($gh) {
            gh auth status 2>$null
            if ($LASTEXITCODE -eq 0) {
                Info "Creando el Pull Request mediante gh CLI (base: $prTarget)..."
                Push-Location $wtPath
                try {
                    & gh pr create --repo $ghRepo --base $prTarget --head $branch --fill
                    if ($LASTEXITCODE -eq 0) {
                        Ok "Pull Request creado exitosamente."
                        return
                    }
                }
                finally {
                    Pop-Location
                }
            }
            else {
                Warn "gh está instalado pero no autenticado ('gh auth login'). Abre el PR manualmente:"
            }
        }
        else {
            Warn "gh CLI no está disponible. Abre el PR manualmente en GitHub:"
        }
        Info $compareUrl
    }

    'done' {
        $doneArgs = @()
        if ($Force) { $doneArgs += '--force' }
        if ($Prefix) {
            Warn "-Prefix no se aplica a done: el nombre de la rama sale del registro local que escribió 'ensure'."
        }

        # Antes de desmontar, qué ramas del cambio existen según el plan del producto (solo lectura).
        # Una rama creada fuera de `ensure` no está en el registro y `done` no la ve: sin esta
        # comprobación el script daría por desmantelado un cambio que sigue montado.
        $existing = @()
        $statusLines = & axiom branch status --cwd $mainRoot --change $Slug --json 2>&1
        if ($LASTEXITCODE -eq 0) {
            try {
                $status = (($statusLines | ForEach-Object { "$_" }) -join "`n") | ConvertFrom-Json
                foreach ($repository in @($status.repositories)) {
                    foreach ($b in @($repository.branches)) {
                        if ($b.exists) { $existing += $b.name }
                    }
                }
            } catch {
                Warn "axiom branch status no devolvió un JSON legible; no se puede comprobar qué ramas existen."
                $existing = $null
            }
        } else {
            Warn "axiom branch status falló; no se puede comprobar qué ramas existen."
            $existing = $null
        }

        # `axiom branch done` confirma la fusión y desmonta el worktree y la rama local del plan.
        & axiom branch done --cwd $mainRoot --change $Slug @doneArgs
        if ($LASTEXITCODE -ne 0) {
            Fail "axiom branch done no desmanteló por completo el cambio '$Slug' (revisa el mensaje anterior)."
        }

        # Sincronizar main si estamos en él en el checkout principal
        $currentBranch = (git -C $mainRoot rev-parse --abbrev-ref HEAD 2>$null).Trim()
        if ($currentBranch -eq 'main') {
            Info "Actualizando main local con origin/main..."
            git -C $mainRoot fetch origin main
            git -C $mainRoot merge --ff-only origin/main
            if ($LASTEXITCODE -eq 0) {
                Ok "main local actualizado correctamente."
            } else {
                Warn "main local no se pudo actualizar vía fast-forward. Realiza pull manual."
            }
        }

        git -C $mainRoot worktree prune
        if ($null -ne $existing -and $existing.Count -eq 0) {
            Warn "axiom branch done no encontró ninguna rama del cambio '$Slug': no había nada que desmantelar."
            Warn "Si la rama se creó fuera de 'ensure' (sin registro local), retira a mano con git su worktree y su rama local."
            return
        }
        Ok "Ciclo de trabajo '$Slug' desmantelado y entorno limpio."
    }
}
