# Exploración: marcas-pruebas (C-06-d)

Marcas de ejecución por entorno de `pruebas.md`, marcadas en la UI y publicadas con push a la rama de integración del repo de specs (solo multirepo y solo SDD).

## 1. Cómo publica hoy `sddpublish`

- `Publisher.Run` (`internal/sddpublish/publish.go:46`) sigue los pasos: almacén (`:62`), carpeta del cambio en disco (`:74`), destino (`:90`), remoto (`:115`), rango sin subir (`:128-158`), conjunto sucio (`:161`), commit (`:178`), commits de código (`:205-237`), push (`:254`).
- El destino sale de `Engine.PublicationTarget` (`internal/branch/engine/publication.go:30`). En el repo de specs de un multirepo (`RepoSpecsInPlace`) solo permite la rama de integración y marca `InPlace` (`:88-92`); otra rama es `wrong_branch`. La rama es siempre la actual (`:74-86`), y `Request.Branch` ata la publicación a la rama autorizada (`:105`).
- Por tanto `Run` ya publica en la integración del repo de specs sin worktree de cambio: en multirepo los specs se trabajan in situ. La carpeta `changes/<x>` debe existir en disco (`:74-87`); archivada solo con `Archive`.
- Rutas que commitea: toda la documentación sucia de la lista cerrada (`internal/docpaths/docpaths.go:158-173`: carpeta del cambio, archivo, `specs/`, `INDEX.md`, contexto mantenido), salvo estado local con punto o `context/` (`:181`). `pruebas-estado.yaml` entra por estar bajo la carpeta del cambio. **No hay forma de limitar el commit a un fichero**: el commit lleva todas las raíces sucias (`publish.go:192`).
- Rechazos tipados: `internal/sddpublish/result.go:52-64` (`store_without_files`, `change_not_found`, `remote_missing`, `unpublished_range_unknown`, `identity_missing`, `commit_hook_rejected`, `commit_failed`, `unpublished_range_too_large`, `code_commits_unpublished`, `push_non_fast_forward`, `push_hook_rejected`, `remote_unavailable`, `push_failed`) más los de `branch` (`wrong_branch`, `detached_head`, ...). El push se clasifica en `classify.go:73-122`.
- **RF-PUB-05**: ninguna negativa reintenta, hace `fetch` ni fuerza; el commit local se conserva. Un reintento con rebase exige una variante explícita, no el camino por defecto.
- **RF-PUB-07 y su prueba** (`internal/sddpublish/isolation_test.go:204-237`): solo `internal/cli/sdd_publish.go` puede importar `internal/sddpublish`; solo `sddpublish/gitport.go` ejecuta `commit` y `push` (`gitcommon/architecture_test.go:104-150`). **El dashboard no puede importar `sddpublish` ni lanzar git push por su cuenta.** `cmd/axiom/main.go:483,1207` construye el servicio y ya importa `internal/cli` (`:21`).

## 2. Dashboard

- Detalle: `GetIncrementDetail` (`internal/dashboard/service.go:534`) llama a `incrementAcceptanceTests` (`:555`, `:1761`) solo con `kind == "active"`; usa `sddstatus.AcceptanceTestsFor(loc, changeRoot)` (`internal/sddstatus/acceptance_tests.go:92`). DTO en `types.go:165-210`; pestaña en `assets/js/screens/detalle-pruebas.js:41` y `detalle.js:208`.
- Lectura: `locateIncrement` (`service_listing.go:41`) da prioridad a un worktree registrado (`:49`), luego `loc.PlanningHome` del checkout principal (`:55`). En multirepo el repo de specs está in situ, así que lo normal es leer `PlanningHome` en la integración; el riesgo del worktree retirado afecta sobre todo a un registro obsoleto (hay que ignorar `worktreePath` al escribir marcas y exigir `PublicationTarget.InPlace`). Un cambio archivado (`kind archived`) no calcula la batería: las marcas no están disponibles tras archivar.
- Protección POST: `isLocalRequest` (`server.go:966`) solo comprueba Host y Origin contra 127.0.0.1/localhost y el puerto; **no hay token CSRF**. Úsala en las rutas nuevas. Nota: `handleSpecsPull` (`:464`) y `handleIncrementVerify` (`:516`) no la usan. Se puede reutilizar `GET /api/workspace/specs/sync-status` (`service.go:~1640`, hace `git fetch` pasivo) como patrón de estado.
- Canario de secretos: `secrets_canary_test.go:25-77`. `TestEveryDashboardRouteIsClassifiedForTheSecretsCanary` (`:108`) falla si una ruta de `server.go` no está en `coveredRoutes` (GET, recorrida por el canario) ni en `excludedRoutes` con motivo. Rutas nuevas: marcar (POST) y publicar (POST) van a `excludedRoutes` («mutación (POST)»); si se añade un GET de estado de marcas, a `coveredRoutes` o excluida con motivo. El DTO del detalle sigue sin ruta nueva para la lectura.
- RF-WEB-17 (`local-web-dashboard/spec.md:497`) dice «Pestaña de solo lectura, sin ruta nueva»: hay que MODIFICARLO.

## 3. Identidad, commit y digest

- Identidad: `Git.IdentityConfigured` (`ports.go:38`) solo da un booleano; no hay lector de `user.name`/`user.email`. Hay que añadir una lectura mínima (`git config user.name/user.email`) en el puerto de git de `sddpublish` o en el servicio.
- Commit probado: `Git.ResolveCommit(ctx, dir, "HEAD")` (`ports.go:31`) del repo de código; en multirepo es otro repo que el de specs. Decisión pendiente: la UI pide el commit o build como campo editable, con valor por defecto vacío; no se infiere de forma silenciosa.
- `plan_digest`: `kickoff.ArtifactDigest([]string{pruebas.md})` (`internal/kickoff/digest.go:23`), SHA-256 hex con CRLF normalizado; es la misma función que el digest de la puerta `tasks`.

## 4. Status y puerta `integration`

- Proyección: añadir `Marks` (resumen por entorno y caducidad) a `AcceptanceTestsProjection` (`acceptance_tests.go:64`). `decorateAcceptanceTests` (`:187`) ya corre con cambio seleccionado no archivado, y `status_v2.go:274` la copia sin cambios. No toca `nextRecommended`, ni dependencias, ni digests.
- Puerta `integration`: `sdd-archive/SKILL.md:76` (evidencia `pr_merged|deployment|attestation`), `:96` (archivo con rama de archivado en monorepo), `:102` (multirepo in situ). Su presentación informativa nace de RF-GAT-26 (`sdd-block-review-gates/spec.md:932`) y de la doctrina del orquestador `Independent Review`/`integration`.
- Goldens: `testdata/golden` tiene 15 ficheros con `sdd-archive` (por ejemplo `sdd-claude-agent-sdd-archive.golden`, `sdd-claude-cmd-sdd-archive.golden`, `skills-default.json`, `sdd-*-rulesmd`/`agentsmd`, que mencionan el skill y la doctrina del orquestador). Tocar `SKILL.md` o la doctrina obliga a regenerarlos (blando, solape con C-02-c/C-03-b/C-07-e). `archive-report` lleva el resumen en el texto del skill.

## 5. Specs vivas afectadas

- `local-web-dashboard`: último ID `RF-WEB-21`; modificar `RF-WEB-17` (de solo lectura a lectura más marcas); añadir requisitos `prov:WEB:1..n` (pantalla de marcas, endpoint de marca, endpoint de publicación, regla local y de topología).
- `sdd-block-review-gates`: último `RF-GAT-32`; añadir presentación del resumen de marcas en `integration` (informativo, no bloqueante) y modificar `RF-GAT-26` si procede.
- `sdd-docs-publication`: último `RF-PUB-12`; modificar `RF-PUB-01` (lista cerrada con `pruebas-estado.yaml`), `RF-PUB-05` (reintento acotado) y `RF-PUB-07` (el puerto lo alcanza también el publicador de marcas); añadir requisito de publicación de marcas.
- Formato: `### Requirement: <título> (RF-<DOM>-<n>)`; los nuevos se escriben `prov:<DOM>:<n>`.

## 6. Trinquetes

- Código muerto (`scripts/deadcode-ratchet.sh`): todo lo exportado nuevo debe tener uso real desde `cmd/axiom` o `cli`.
- Rechazos (`internal/cli/refusal_resolution_ratchet_test.go:617`; audita `internal/cli`, `fileprim`, `sddstatus`): cualquier rechazo nuevo en esos paquetes debe nombrar su resolución. Los códigos nuevos de `sddpublish` no están auditados por ese trinquete, pero deben llevar continuación.
- `internal/idmigration/residues.yaml`: solo si algún fixture nuevo usa `REQ-`; evitarlos (usar `RF-`).
- Catálogo de settings: no se añade clave de `axiom.yaml`; se reutiliza `acceptance_tests.environments`.
- Aislamiento RF-PUB-07: actualizar `TestIsolationOnlyTheCLIVerbImportsSddpublish` solo si el publicador de marcas vive en otro fichero de `internal/cli`.

## 7. Enfoques

**A (recomendado): publicador de marcas inyectado en el dashboard desde `internal/cli`.**
- `sddpublish` gana un modo acotado: `Request.Only` (lista exacta de ficheros, solo `pruebas-estado.yaml`) y reintento acotado ante `push_non_fast_forward` solo para ese modo.
- `internal/cli/sdd_publish.go` exporta un constructor que `cmd/axiom/main.go` entrega al servicio del dashboard mediante una interfaz del propio dashboard. El dashboard no importa `sddpublish` ni ejecuta git push.
- Reintento por reproducción, no `rebase`: ante rechazo, `fetch` del remoto, descartar el commit local de marcas, avance rápido, volver a añadir las entradas pendientes, commit y push; si falla de nuevo, rechazo claro. Evita conflictos de texto en un YAML que solo se añade.

| | Esfuerzo | Riesgo | Notas |
|---|---|---|---|
| A | M | medio | Reutiliza rechazos tipados y el único punto de git; toca RF-PUB-05/07 |
| B: el dashboard lanza `axiom sdd publish --only ...` como subproceso | M | medio-alto | Respeta el aislamiento al pie de la letra, pero depende del binario en ejecución, de parsear JSON y exige el mismo cambio en `Request`; más difícil de probar |

## 8. Riesgos

- **Push solo tras el clic:** dos pasos, `POST marcar` (solo escribe local, añade) y `POST publicar` (con `confirm` explícito); la UI muestra antes remoto, rama y entradas pendientes. No hay token CSRF, así que la garantía es la comprobación de `isLocalRequest` más el cuerpo `confirm`.
- **Concurrencia entre personas:** reintento por reproducción (arriba). Dentro del proceso: mutex por cambio y escritura atómica (`internal/fileprim`).
- **Publicar arrastra otros documentos sucios:** `Only` evita commitear la carpeta entera; hay que decidir si un resto sucio en el repo de specs es rechazo o aviso.
- **Digest vs. CRLF:** usar `kickoff.ArtifactDigest` para que no caduquen marcas por finales de línea.
- **Monorepo:** `EffectiveAcceptanceTests(topology)` (`acceptance_tests.go:98`) ya controla la activación; la pantalla y las rutas deben responder con un estado «no disponible», no con 404 silencioso.
- **Cambio archivado o pruebas.md ausente:** sin pantalla de marcas.
- **Solape de ola 13:** `server.go`, `service.go`, goldens y `sdd-archive` son duros/blandos con otros cambios (README §4); la mezcla se resuelve al fusionar.

## Recomendación

Enfoque A. Cuestiones abiertas para la propuesta: origen del commit probado; si un repo de specs con otros documentos sucios rechaza o avisa; si `n/a` y agente-solo-`local` se validan en el servidor.
