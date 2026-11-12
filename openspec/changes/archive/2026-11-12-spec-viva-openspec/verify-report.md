# Informe de verificación: Spec viva única en OpenSpec (spec-viva-openspec)

> Fase `verify` · rama `inc/spec-viva-openspec` · HEAD `1166efca` (rebasada sobre `origin/main` c91a0afd) · 2026-11-12 · solo lectura sobre el código; sin binarios sueltos.

## Veredicto

Sin hallazgos CRITICAL ni WARNING: 0 CRITICAL, 0 WARNING, 3 SUGGESTION. La implementación cumple las deltas (REQ-30.1 a REQ-30.32 más las MODIFIED y REMOVED), el diseño y la coherencia con `carril-sdd-versatil`. Siguiente paso recomendado: `sdd-archive`.

## Alcance

- Artefactos leídos: `proposal.md`, `design.md`, `tasks.md`, `specs/` (16 capacidades con delta). Almacén `hybrid`. Strict TDD no declarado para esta verificación.
- Implementación inspeccionada: `internal/{openspecdoc,techcontext,livespec,adoption,adoption/adoptcheck}`, adaptadores `internal/cli/archive_{context,integrate,adopt}.go`, `sdd_integrate_retired.go`, `sdd_archive_compose*.go`, `internal/workspace/{retired,declarations}.go`, consumidores de la retirada (`sddstatus`, `livingdoc`, `knowledge`, `sddpublish`, `docpaths`, `hub`, `dashboard`, `tui`), doctrina (`routing.go`, skills, `_shared/*`) y documentación.

## Progreso observado

67/67 tareas marcadas `[x]` en `tasks.md` (U1 13, U2 15, U3 8, U4 13, U5 6, U6 12). No se ha modificado ninguna casilla. Hay dos desviaciones ya anotadas en `tasks.md`: tarea 4.6 (`Check` vive en `internal/adoption/adoptcheck` por un ciclo de importación) y tarea 4.2 (prueba del enlace simbólico saliente omitida donde la plataforma no permite crear enlaces).

## Comprobaciones

| Comando | Resultado |
|---|---|
| `go vet ./...` | exit 0 |
| `gofmt -l internal/ cmd/` | salida vacía |
| `bash scripts/deadcode-ratchet.sh` | «no new unreachable functions», exit 0 |
| `go test` de `internal/{techcontext,livespec,openspecdoc,adoption/...,hub,dashboard,livingdoc,tui/...,knowledge,sddpublish,docpaths,branch,gitcommon,workspace,sddstatus,isoevidence,assets,components/...}` con `-count=1` | todos `ok`, salvo `internal/components/filemerge` |
| `internal/components/filemerge` | FAIL en `TestWriteFileAtomicFollowsDirectoryJunction` y `TestWriteFileAtomicFollowsRelativeDirectoryJunction` (junction de Windows): fallo de entorno conocido, también en `main` |
| `go test ./internal/cli -run <Archive, SddArchiveCompose, ArchiveCheck, SddIntegrate, Kickoff, Integrate, Guard, Compose, Retired, Iso, ISO, TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign> -count=1 -timeout 45m` | ok (116 s), incluye el ratchet de negativas |
| `go test ./cmd/axiom/... -count=1` | ok (23 s) |
| `go test ./internal/app/ -run 'Documented|Invocation' -count=1` | ok (58 s) |

No ejecutado: el paquete `internal/cli` completo (más de 1 h en local, convención del proyecto) ni `internal/app` completo (fallo conocido `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`). Las pruebas con enlaces simbólicos quedan omitidas en esta máquina. Es una limitación, no una prueba de corrección.

## Cumplimiento frente a la spec

Cada REQ-30.x tiene código de producción y pruebas; la trazabilidad se contrastó por el contenido de las pruebas, no solo por etiqueta.

- REQ-30.1 a 30.8 (integración ODD): `internal/livespec/{odddoc,registry,plan,apply,impact,context,iso,report}.go` con sus pruebas (tabla de `ParseFeatureDoc`, idempotencia, todo o nada, colisiones, impacto, contexto en los cuatro estados de commit, ISO). Adaptador `internal/cli/archive_integrate.go`.
- REQ-30.9 a 30.14 (adopción): `internal/adoption/*`, `internal/adoption/adoptcheck/check.go` (una prueba por regla y por acumulación), `internal/cli/archive_adopt.go`. REQ-30.9 recoge la guarda A de REQ-25.19 (tarea 4.1, `specs/openspec-adoption/spec.md:48` y `:72`).
- REQ-30.15 a 30.20, 30.29, 30.30, 30.32 (contexto técnico): `internal/techcontext/*`, `internal/cli/archive_context.go`; `check` no escribe y `Verify` falla cerrado con declaración o almacén inválidos.
- REQ-30.21 (baseline residual): `internal/workspace/retired.go` y `declarations.go:33,71`; cubierto en carga, `validate` (salida 1), status (solo `archive` bloqueado, sin clave `baseline`), `hub`, `livingdoc`, `knowledge`, `dashboard` y `tui`.
- REQ-30.22 a 30.25: `archive-compose` conserva `--canonical --delta`; `*ComposeModeRetiredError` en `internal/cli/sdd_archive_compose.go:38-46`; `axiom sdd integrate` es `*SDDIntegrateRetiredError` en `internal/cli/sdd_integrate_retired.go` y no escribe.
- REQ-30.26, 30.27: `hub` (`TechDetection.Adoption`, `InitResult.Adoption`) y panel de `projects.js` con prueba de asset.
- REQ-30.28, 30.31: skills y `_shared/*` sin baseline; referencia `sdd-spec/references/openspec-adoption.md`.
- REMOVED (`spec-baseline-declaration`, `spec-baseline-integration`) y MODIFIED (`living-documentation`, `multi-project-hub`, `planning-home-resolution`, `project-technical-context`, `sdd-archive-lifecycle`, `sdd-docs-publication`, `tui-ui-parity`, `workspace-topology`, `axiom-sdd-cli-integration`, `change-branch-lifecycle`, `organic-agent-trigger-rules`): código retirado (`internal/specbaseline/` ya no existe); `.deadcode-baseline.txt` y `.refusal-ratchet-baseline.txt` sin entradas de lo borrado.

## Conformidad con el diseño

- Paquetes y dependencias como en §1: `openspecdoc` es hoja; `sddstatus` conserva el envoltorio de composición; ningún paquete nuevo importa `specbaseline`.
- CLI: `case "integrate"`, `"adopt"` y `"context"` cableados en `cmd/axiom/main.go:1774-1778`; la ayuda de `axiom sdd integrate` indica la retirada (`cmd/axiom/main.go:165`).
- Errores tipados presentes: `workspace.BaselineRetiredError` (`retired.go`), `cli.SDDIntegrateRetiredError`, `cli.ComposeModeRetiredError`, `livespec.ISOProjectError` (`internal/livespec/iso.go:13`; `RefuseISOProject` rechaza antes de leer el documento).
- `changeRootOfDelta` se conserva (`internal/cli/sdd_archive_compose.go:182`) y lo usan `sdd_archive_compose_iso.go:43` y `sdd_writer_guard.go:59`.
- `docpaths.New` sin parámetro `baseline` (`internal/docpaths/docpaths.go:37`).
- Desviaciones declaradas y aceptables: D2 («extraer en paralelo» en lugar de «delegar»; la pregunta abierta de `design.md:254` queda sin objeto al borrarse `specbaseline` en U6) y la ubicación de `adoptcheck` (tarea 4.6).

## Coherencia con carril-sdd-versatil

- Piezas ISO intactas: pasan `internal/workspace`, `internal/sddstatus` (incluido `iso_test.go`), `internal/isoevidence` y las pruebas ISO y de archive-compose de `internal/cli`. El paso 0 ISO de `internal/components/agentguidance/routing.go:68` se conserva y el paso 7 incorpora el marcador (`routing.go:77`).
- Marcador compartido: el literal `Sin impacto en la spec` y la etiqueta `Motivo:` coinciden en ambos carriles (`internal/isoevidence/bug.go:11` y `internal/livespec/odddoc.go:37-38`), con comparación exacta tras recortar espacios en ambos.

## Referencias vivas a la baseline

`rg` sobre `internal`, `cmd`, skills, `docs/manual-de-inicio.md`, `docs/openspec-config.md`, `axiom.example.yaml`, `AGENTS.md`, `GEMINI.md` y `.codex`: las únicas apariciones de `baseline:`, `specbaseline` y `sdd integrate` en código de producción son el rechazo tipado, sus comentarios y continuaciones, y la prosa de retirada del manual. Ninguna skill ni golden contiene `axiom sdd integrate`, `baseline.documents`, `integration: directed` ni `--in-place`. Las specs vivas de `openspec/specs/` aún mencionan la baseline: es lo esperado hasta que el archivado aplique las deltas REMOVED y MODIFIED.

## Hallazgos

### CRITICAL

Ninguno.

### WARNING

Ninguno.

### SUGGESTION

1. **Contrato del marcador duplicado sin prueba de paridad.** `internal/livespec/odddoc.go:37-38` redefine el literal que `internal/isoevidence/bug.go:11` declara «contrato compartido». Ninguna prueba compara ambos carriles; un cambio en uno no rompería al otro. Valorar una prueba de paridad o reutilizar `isoevidence.NoImpactMark`.
2. **Divergencia menor en `Motivo:`.** `internal/livespec/odddoc.go:346` toma la primera línea `Motivo:` con texto (`&& impact.Reason == ""`), de modo que una `Motivo:` vacía seguida de otra con texto es válida en ODD; `isoevidence.labelValue` toma la primera aunque esté vacía y la declara incompleta. Caso límite sin impacto práctico probable.
3. **Pregunta abierta residual en `design.md:254`.** La casilla de D2 sigue sin marcar aunque U6 la dejó sin objeto; conviene cerrarla al archivar.

## Resumen

Verificado: especificación frente a implementación, diseño, coherencia con carril, ausencia de referencias vivas y la batería de pruebas pedida. Solo falla `filemerge` por junctions de Windows (fallo de entorno conocido). Sin trabajo pendiente para apply; recomendado `sdd-archive`.
