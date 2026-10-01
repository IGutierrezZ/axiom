# Paridad de Kiro (`kiro-ide`) en ámbito proyecto

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/kiro-workspace-parity.md`.
> Espejo de recuperación en Engram: topic `odd/kiro-workspace-parity/tasks`, proyecto `axiom` (pendiente: Engram no está disponible en esta sesión).
> Creado a mano siguiendo `odd/tasks/workspace-isolated-sync.md`: el binario actual no expone `axiom odd` (`axiom odd --help` responde "comando 'odd' no reconocido").

## Objetivo

Que `axiom setup --agent kiro-ide` y `axiom sync --agent kiro-ide --scope workspace` dejen en el repositorio todo lo que Kiro necesita (steering, agentes, skills, MCP y hook de skill-registry) sin tocar `~/.kiro`, con tests que lo garanticen y documentación para usarlo.

## Problema

1. No hay tests que fijen que las rutas de Kiro en ámbito workspace cuelgan del workspace y no del HOME (G3).
2. Un comentario de `internal/components/golden_test.go` cita el steering heredado `gentle-ai.md` (G6).
3. `docs/kiro.md` no documenta el ámbito proyecto y `.gitignore` no excluye los artefactos generados con rutas de máquina (G4).
4. La TUI no preselecciona Kiro al detectarlo ni conoce su directorio de skills en Agent Builder (G5).
5. Kiro no tiene hook que refresque el skill-registry al iniciar sesión, como Claude y Codex (G1).

## Alcance autorizado

- PR1 (`feat/kiro-workspace-parity`): tests en `internal/cli`, comentario de `internal/components/golden_test.go`, `.gitignore`, `docs/kiro.md`, `internal/tui/model.go` y sus tests.
- PR2 (`feat/kiro-skill-registry-hook`, apilado sobre PR1): hook de skill-registry en el adapter de Kiro, el inyector SDD, rutas de backup/verificación y desinstalación.

## Restricciones

- Todo en español (docs, ODD, commits y PRs).
- Prohibido ejecutar `axiom setup|install|sync` reales; los tests se ejecutan con el HOME aislado.
- No se commitea nada bajo `.kiro/` en estos PRs.
- Cada PR ≤400 líneas cambiadas.

## Decisiones

- **G2 no se implementa (opción a).** Las skills propias de `skills/*` no se espejan en `.kiro/skills/`: Kiro ya carga `AGENTS.md` como steering y su índice `axiom:skills-index` lista cada skill con su ruta. Un espejo duplicaría ~15 skills en un directorio ignorado, generaría deriva y exigiría un paso post-sync nuevo. Claude tampoco las espeja.
- **`.kiro/steering/` y `.kiro/hooks/` se versionan.** Su contenido no depende de la máquina (sin rutas absolutas; el hook usa `--cwd .`), así Kiro carga el orquestador al clonar sin ejecutar setup. Solo se ignoran `.kiro/agents/`, `.kiro/skills/` y `.kiro/settings/`. Tras el merge, el usuario genera los ficheros con `axiom sync --agent kiro-ide --scope workspace` y los commitea en un PR aparte.

---

## Tareas

- [x] **T1 · G3: tests del ámbito workspace para Kiro** (`bcec8be5`)
  - `TestComponentInjectionDirScopedWorkspaceSafeguard` incluye `model.AgentKiroIDE` entre los agentes que resuelven a `workspaceDir`.
  - Nuevo `TestComponentPathsWorkspaceScopedKiroStaysInWorkspace` (sdd, skills, engram, context7, persona).
  - Nuevo `TestSyncBackupTargetsScopedWorkspaceKiroStaysInWorkspace`.
  - Sin fugas: con `APPDATA` y `XDG_CONFIG_HOME` bajo home, ninguna ruta de Kiro cae bajo home.

- [x] **T2 · G6: comentario obsoleto del steering de Kiro** (`58e5a8ba`)
  - `internal/components/golden_test.go`: `~/.kiro/steering/gentle-ai.md` → `~/.kiro/steering/axiom.md`.

- [x] **T3 · G4 + `.gitignore`** (`85ac7180`)
  - Bloque Kiro en `.gitignore` (`.kiro/agents/`, `.kiro/skills/`, `.kiro/settings/`).
  - Sección `## Ámbito proyecto (workspace)` en `docs/kiro.md`.

- [x] **T4 · G5: TUI** (`4d1ada3d`)
  - Kiro en `detectedAgentIDs` y `agentBuilderSkillsDir` de `internal/tui/model.go`, con tests.
  - El helper de test `makeDetectionWithAgents` no emitía `kiro-ide`; se añadió a su lista de agentes conocidos.

- [ ] **T5 · G1: hook de skill-registry (PR2)**
  - `.kiro/hooks/axiom-skill-registry.json` con trigger `SessionStart` y `axiom skill-registry refresh --quiet --no-gitignore --cwd .`.
  - Declarado en backup/verificación, eliminado al desinstalar y documentado.

---

## Verificación ejecutable

Todos los tests con el HOME aislado (`$env:HOME` y `$env:USERPROFILE` en `%TEMP%\axiom-kiro-parity-home`).

1. **T1:** `go test ./internal/cli/ -run "Kiro|WorkspaceSafeguard|ScopedWorkspace" -count=1` → `ok (11.0s)`; los 5 subtests de cada test nuevo en verde.
2. **T2:** `go test ./internal/components/ -run TestGoldenSDD_Kiro -count=1` → `ok`.
3. **T3:** `git check-ignore .kiro/agents/x.md .kiro/skills/x .kiro/settings/mcp.json` → lista las tres; `git check-ignore .kiro/steering/axiom.md .kiro/hooks/x.json` → código 1.
4. **T4:** `go test ./internal/tui/ -run "PreselectedAgents|AgentBuilderSkillsDir" -count=1` → `ok`; `go test ./internal/tui/ -count=1` → `ok`.
5. **Global:** `go build ./...`, `go vet ./internal/cli/ ./internal/tui/ ./internal/components/`, `go run ./internal/gofmtcheck` y `git diff --check origin/main...HEAD` sin errores. `go test ./internal/components/ ./internal/tui/ -count=1` → `ok`.
6. **`go test ./internal/cli/` completo:** no termina en local (timeout a 10m, 25m y 1h20m) por los tests de review, que fallan de forma intermitente por el presupuesto de tiempo (`operation_timeout`) de sus subprocesos git. Ejecutado aparte, `TestNegotiatedBoundStatusResumesSameCaptureAfterManagedAssetsConverge/reviewer` pasa 2/2 en la rama y 2/2 en `origin/main`. Los 753 tests de `internal/cli` que no son de review: 720 en verde y 3 en rojo (`TestCodeGraphGuidanceSyncStepRestoresSymlinkAfterInstallerFailure`, `TestCodeGraphGuidanceSyncStepPreservesBrokenSymlinkChain`, `TestRunSyncMigratesLegacyManagedPiCodeGraphSelection`), que fallan igual en `origin/main` (Windows sin privilegio de symlink; wiring de CodeGraph). Preexistentes, no se tocan.
