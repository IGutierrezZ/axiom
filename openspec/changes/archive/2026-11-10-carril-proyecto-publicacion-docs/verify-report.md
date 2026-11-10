# Informe de verificación — carril-proyecto-publicacion-docs

- **Fecha:** 2026-11-10
- **Base verificada:** `main` en `09e75e66` (merge de la PR #235; cabeza de la rama `68a6e659`).
- **Veredicto:** PASS con advertencias (0 CRITICAL, 3 WARNING, 2 SUGGESTION). Nada bloquea el archivado.

## Tareas

28/28 marcadas en `tasks.md`, en cuatro commits de unidad de trabajo: WU-1 `e92d8c5d`, WU-2 `93796b0b`, WU-4 `87237779` y WU-3 `68a6e659`.

## Requisitos → evidencia

Los requisitos añadidos REQ-27.1 a REQ-27.17 y los REQ-2.1, 21.x, 25.x y 26.x que el cambio modifica están cumplidos. Cada uno se traza a código y a pruebas etiquetadas `REQ-` (salvo lo indicado en S-1).

| Ámbito | Requisitos | Evidencia principal |
|---|---|---|
| Carril del proyecto (`workflow.lane`) | REQ-27.1, REQ-2.1 | `internal/workspace/workflow.go`; pruebas de `internal/workspace` |
| Proyección en `axiom sdd status` | REQ-27.2, REQ-27.2b | `internal/sddstatus/workspace_declarations.go`; pruebas `TestRunSDDStatus*` y `TestDualContract_SDDStatus*` |
| Kickoff y `docs_publication` | REQ-27.3, REQ-27.4, REQ-21.1–21.3, REQ-25.35 | `internal/kickoff/schema.go:98`; pruebas de `internal/kickoff` |
| Publicación ligera de documentos | REQ-27.5–27.11, REQ-27.13 | `internal/sddpublish/publish.go`, `internal/docpaths/docpaths.go`; pruebas con remoto bare de `internal/sddpublish` y `internal/cli` (`SDDPublish*`, `PublishIsolation*`) |
| Puertas y cesión | REQ-27.12, REQ-26.1, 26.3–26.6, 26.8 | `internal/cli/sdd_gate_handoff.go:169`; `internal/cli/sdd_publish_gate_test.go` |
| Archivado | REQ-27.14 | `publish --archive` en `internal/sddpublish/publish.go` |
| Adopción de la rama publicada | REQ-27.15, REQ-25.1, 25.7, 25.10, 26.11 | `internal/branch/engine/ensure.go`, `internal/branch/gitcli/ops.go:320`; `internal/branch/engine/ensure_adopt_test.go` |
| Doctrina ODD sin paradas | REQ-27.16, REQ-27.17, REQ-25.40; REQ-26.10 eliminado | `internal/components/agentguidance/routing.go:68`; goldens de `internal/components` |

## Comandos ejecutados

| Comando | Resultado observado |
|---|---|
| `go test ./internal/docpaths/ ./internal/kickoff/ ./internal/agents/capabilitymanifest/` | ok |
| `go test ./internal/sddpublish/ ./internal/workspace/` | ok (89,8 s y 45,9 s) |
| `go vet` de esos cinco paquetes | limpio |
| `rg 'uncommitted_artifacts\|CodeUncommittedArtifacts\|MergeRequired\|MergeNotice'` | 0 apariciones en producción; solo aserciones negativas en `internal/cli/sdd_gate_handoff_test.go:222-230` y `internal/assets/assets_test.go:4280` |

CI de la PR #235 en verde sobre `68a6e659`: Unit Tests (8m23s), Go Format, Claude Network-None Runtime, OpenCode V2 Released SDK Contracts y PR Validation.

## Advertencias

- **W-1:** REQ-25.7 quedó matizado durante apply: `createdFrom` y `createdFromRef` solo aparecen en la ejecución que crea la rama. La spec delta ya lo recoge.
- **W-2:** fallo preexistente en `main` y ajeno al cambio: `internal/components/filemerge` `TestWriteFileAtomicFollows(Relative)DirectoryJunction` (Windows).
- **W-3:** esta verificación no reejecutó `internal/cli` completo, `branch`, `sddstatus`, `agentguidance` ni `assets`. Se apoya en el CI de la PR #235 y en las ejecuciones locales de apply.

## Sugerencias

- **S-1:** las pruebas de REQ-27.1 y REQ-27.3 no llevan la etiqueta `REQ-`. La cobertura existe por nombre y contenido.
- **S-2:** mantener la correspondencia entre los comentarios de los pasos 1 a 7 de `Publisher.Run` y DD-09 si el algoritmo se amplía.

## Desviaciones registradas en apply

- `.codex/AGENTS.md` se actualizó por bloques, porque su marca y su tabla de modelos difieren del golden.
- Los digests del manifiesto de capacidades y el hash de Kilocode se recalcularon por el literal nuevo de la política de selección.
- Los rechazos de la guarda de escritura de `axiom sdd publish` salen como texto; los del `Publisher` salen en JSON con `--json`.

## Contexto técnico

`axiom.yaml` no declara `technical_context`, así que no aplica el delta de contexto.
