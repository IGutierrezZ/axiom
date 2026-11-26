# ODD: MCP de Azure DevOps para el agente (C-07-g)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/mcp-ado.md`.
> Espejo de recuperación en Engram: topic `odd/mcp-ado/tasks`, proyecto `axiom`.
> Rama `inc/mcp-ado`, worktree `C:\repos\axiom-wt\mcp-ado\axiom`.
> Plan maestro: `docs/plan/backlog-2026-11/07-integraciones.md`, apartado C-07-g. Carril ODD, tamaño S.

## Objetivo

Que el agente consulte Azure DevOps (contexto, duplicados, ayuda a la redacción) por el servidor MCP oficial con Entra, y que nunca escriba en Azure DevOps en los eventos del ciclo de vida, que son del binario (REST con PAT, C-07-c/d/e).

## Problema y porqué

- C-07-c, C-07-d y C-07-e dejaron lo determinista en el binario: adaptador REST, puerto `WorkItemSource`, outbox y verbos `axiom integrations`.
- El agente no tiene hoy una vía de lectura de Azure DevOps, y nada le impide escribir por su cuenta en los momentos que ya cubre el binario (la contradicción de KVP: el `CLAUDE.md` prohíbe HTTP mientras el lanzador usa REST).

## Alcance

1. `internal/versions`: versión fijada del paquete `@azure-devops/mcp` (con comentario de renovate), como `Context7MCP`.
2. `internal/components/mcp`: `azuredevops.go` junto a `context7.go`, con las entradas por estrategia de agente y `InjectAzureDevOps(homeDir, targetDir, adapter, organization)`.
   - Servidor local por `npx` con la versión fijada: `@azure-devops/mcp@<v> <organización> -d core work work-items search`.
   - Sin `--authentication`: el valor por defecto del servidor es `interactive` (inicio de sesión de Microsoft, Entra). No se escribe ningún PAT ni secreto.
   - La organización se valida (letras, dígitos y guiones, sin empezar por guion) para que no pueda inyectar argumentos.
3. `internal/cli` (`sync.go` y `run.go`): en el caso del componente MCP (`ComponentContext7`), si el ámbito es workspace y el `axiom.yaml` del workspace tiene activo el plugin `azure-devops`, se inyecta además la entrada `azure-devops` con su `organization`.
4. Doctrina: regla de frontera agente/binario en `internal/assets/skills/_shared/sdd-orchestrator-sections.md` (SDD) y en `internal/components/agentguidance/routing.go` (ODD), más goldens.

## Decisiones

- **Solo ámbito workspace.** El plugin se activa por proyecto (`integrations.azure-devops` en `axiom.yaml`) y la organización es del proyecto; escribirla en la configuración de usuario la filtraría a todos los proyectos. En ámbito de usuario no se inyecta nada.
- **Va con el componente MCP.** La entrada se escribe en el mismo paso que Context7 (el componente MCP del catálogo), sin componente nuevo en la selección.
- **Servidor local, no remoto.** `https://mcp.dev.azure.com/{org}` solo admite Entra con un registro de app propio para Claude Code y Cursor; el local con `interactive` funciona sin registro.
- **Sin retirada.** Desactivar el plugin no borra la entrada ya escrita (fuera de alcance S); se anota como limitación.
- **Sin `## Spec viva`.** El plan lo clasifica como componente de inyección y texto de guía, sin spec de comportamiento nueva.

## Restricciones

- No se llama a Azure DevOps real. No se escriben credenciales.
- Pruebas localizadas: `./internal/components/mcp`, `./internal/versions`, `./internal/components/agentguidance`, las pruebas de doctrina de `./internal/components/sdd` y los goldens afectados de `./internal/components`.
- TDD: desactivado (sin configuración de proyecto ni de sesión); se ejecutan las comprobaciones funcionales.

## Tareas

- [x] **T1 · Versión fijada y entradas MCP de ADO (`internal/versions`, `internal/components/mcp`)** — ruta: inline.
  - `versions.AzureDevOpsMCP` con comentario de renovate y alta en `TestPinnedVersionsAreDefined`.
  - `azuredevops.go`: overlays por estrategia (Claude `.mcp.json`, OpenCode `mcp` local, Gemini/Cursor `mcpServers`, Codex TOML) e `InjectAzureDevOps`.
  - Pruebas: entrada por agente, idempotencia, convivencia con Context7, organización inválida rechazada, ámbito de usuario sin escritura.
- [x] **T2 · Inyección condicionada al plugin activo (`internal/cli/sync.go`, `run.go`)** — ruta: inline.
  - Resolver la organización desde `workspace.Locate(s.workspaceDir)` → `Declarations.Integrations.Active()`.
  - Pruebas: con plugin activo en workspace se escribe; sin plugin o en ámbito de usuario, nada cambia.
- [x] **T3 · Doctrina de frontera agente/binario y goldens** — ruta: inline.
  - Párrafo en `sdd-orchestrator-sections.md` y línea en la sección ODD de `routing.go`.
  - Pruebas de paridad y regeneración de goldens.

## Criterios de aceptación

- Con el plugin activo y ámbito workspace, cada agente soportado en ese ámbito tiene la entrada `azure-devops` con la versión fijada y la organización del proyecto.
- Sin plugin, o en ámbito de usuario, nada cambia.
- La guía (SDD y ODD) contiene la regla: el agente lee por MCP y nunca escribe en Azure DevOps en los eventos del ciclo de vida.

## Progreso

- Worktree creado con `scripts/axiom-worktree.ps1 new mcp-ado -Lane odd`.
- T1: `go test ./internal/versions ./internal/components/mcp -run 'AzureDevOps|TestPinnedVersions'` → ok (6 pruebas, 6 agentes); `gofmt -l` y `go vet` limpios.
- T2: `go test ./internal/cli -run 'TestRunInstallWorkspaceInjectsAzureDevOpsMCPOnlyWithThePlugin|TestAzureDevOpsMCPOrganization|Context7'` → ok; `go vet ./internal/cli` limpio.
- T3: `go test ./internal/components/agentguidance ./internal/components/sdd ./internal/assets ./internal/components/mcp` → ok; goldens regenerados (`go test ./internal/components -run 'TestGoldenSDD_|TestGoldenCombined_Claude' -update`, 10 ficheros) y `go test ./internal/components -run TestGolden` → ok. Nota en `docs/manual-de-inicio.md`.

## Commits

| Tarea | Commit |
|---|---|
| T1 | `e3dfd9f0` |
| T2 | `b7e14cc0` |
| T3 | commit `docs(sdd)` que cierra la rama (sin `## Spec viva`: no hay `axiom archive integrate`) |

## Siguiente paso

Abrir la PR.
