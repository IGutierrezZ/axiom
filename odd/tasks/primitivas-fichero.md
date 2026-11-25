# primitivas-fichero (C-08-c)

Rama: `refactor/primitivas-fichero`. Plan: `docs/plan/backlog-2026-11/08-adelgazamiento.md`, C-08-c.

## Objetivo

Que ningún paquete ajeno a RDD importe `internal/reviewtransaction` salvo en usos clasificados como «parte RDD», para que C-01-e quede en un borrado.

## Problema y porqué

`kickoff`, `sddstatus`, `branch/registry`, `statecoord`, `filecoord`, `integrations`, `app` y tres ficheros de `cli` importan `reviewtransaction` solo para usar el cerrojo de fichero, la publicación atómica, la sincronización de directorio, la apertura física sin seguir enlaces y dos consultas de git. Al borrar RDD se romperían.

## Alcance y restricciones

- Refactor sin cambio de comportamiento: mismos textos de error, mismo formato del fichero de cerrojo y mismas costuras de prueba.
- Paquete neutral nuevo: `internal/fileprim`. `reviewtransaction` pasa a importarlo.
- Las primitivas se mueven con sus pruebas, también las `_windows.go`.
- Fuera de alcance: borrar código de RDD.
- TDD: no configurado para este cambio; comprobaciones funcionales con `go test` por paquetes (sin binarios sueltos).

## Clasificación de usos externos

| Fichero | Símbolos | Clasificación |
|---|---|---|
| `filecoord/lock_backend.go` | `AcquireAuthorityFileLock`, `ErrStoreLockContended` | Neutral: se extrae |
| `branch/registry/registry.go` | cerrojo, `ReplaceFileAtomic`, `SyncReviewDirectory` | Neutral: se extrae |
| `kickoff/ledger.go`, `seal.go`, `seal_test.go` | cerrojo, `ReplaceFileAtomic`, `SyncReviewDirectory`, `PublishFileNoReplace` | Neutral: se extrae |
| `statecoord/statecoord.go` | `AcquireAuthorityFileLock` | Neutral: se extrae |
| `integrations/files.go`, `files_test.go` | cerrojo, `ReplaceFileAtomic`, `SyncReviewDirectory` | Neutral: se extrae (no estaba en el inventario de la 08) |
| `sddstatus/edit_authority_consent.go`, `edit_grant_contention_test.go`, `status_v2_clean_break_test.go` | `PublishFileNoReplace`, cerrojo | Neutral: se extrae |
| `sddstatus/runtime_ledger.go` | cerrojo, publicación, `ErrConcurrentUpdate` del cerrojo | Neutral: se extrae |
| `sddstatus/runtime_ledger.go` | `SnapshotBuilder.ResolveRepositoryRoot` | Neutral (SDD): pasa a `gitcommon` |
| `sddstatus/runtime_ledger.go` | `CompactAuthoritativeStore` (sonda del almacén para hallar el git common dir) | **Parte RDD** según la 08. Aviso para C-01-e: el ledger es de SDD, así que no basta con borrar la sonda; hay que sustituirla por `gitcommon.CommonDir` |
| `cli/sdd_gate_evidence.go` | `SnapshotBuilder.RevisionIsAncestor` | Neutral (SDD): pasa a `gitcommon` |
| `cli/compatibility_transaction_windows.go` | `OpenPhysicalPath` | Neutral (instalación): se extrae |
| `app/state_lock_test.go` | `AcquireAuthorityFileLock` | Neutral: se extrae |
| `cli/install_state_lock_test.go`, `cli/sync_persona_migration_test.go`, `tui/model_test.go`, `components/uninstall/state_lock_test.go` | cerrojo + `RDDModeOn` | Cerrojo: se extrae. `RDDModeOn`: **parte RDD** (lo retira C-01-e) |
| `sddstatus/main_test.go` | `LocalGitCommandTimeout`, `RemoteGitCommandTimeout` | **Parte RDD**: acortan el runner de git de RDD |
| `sddstatus/review_mode_home_test.go`, `review_offer_absence_guard_test.go` | `RDDModeOn`, guarda | **Parte RDD** |
| `kickoff/rdd_boundary_test.go` | guarda de vocabulario | **Parte RDD**: lista C-01-e; se actualiza su lista de imports permitidos |
| `cli/branch_lineage.go`, `branch_lineage_test.go` | `InventoryAuthority`, `AuthorityStatusReport` | **Parte RDD** (inventario de linajes) |
| `app/app.go`, `tui/model.go`, `tui/screens/*review*` | almacén y modo RDD | **Parte RDD** (lista C-01-e) |
| `components/sdd/boundedreview.go` y pruebas, `reviewerprovider/*` | revisor RDD | **Parte RDD** (lista C-01-e) |
| `cli/review*.go`, `cli/sdd_status_*review*_test.go`, `cli/dual_contract_integration_test.go`, `cli/continuation_paths_test.go` | RDD | **Parte RDD** (lista C-01-e) |
| `cli/guard_population_test.go`, `cli/refusal_resolution_ratchet_test.go` | rutas en texto, sin uso de símbolos | **Parte RDD**: escanean el paquete |

## Tareas

- [x] T1 · Crear `internal/fileprim` con cerrojo, publicación, reemplazo, sincronización de directorio y apertura sin seguir enlaces; mover sus pruebas; `reviewtransaction` delega en él; migrar los consumidores neutrales. Ruta: delegada (protección de contexto del padre: ~3.000 líneas de lectura).
- [x] T2 · `SnapshotBuilder` de `sddstatus/runtime_ledger.go` y `cli/sdd_gate_evidence.go` a `gitcommon`. Ruta: inline (<100 líneas).
- [x] T3 · Comprobar el criterio de terminado (`rg -l 'internal/reviewtransaction"'`) y `go test` de los paquetes tocados.

## Criterio de aceptación

- `rg -l 'internal/reviewtransaction"' -g '*.go'` fuera de `reviewtransaction` solo devuelve ficheros de la lista de C-01-e o de la tabla anterior marcados «parte RDD».
- `go test` en verde de `fileprim`, `reviewtransaction`, `kickoff`, `sddstatus`, `branch/registry`, `statecoord`, `filecoord`, `integrations`, `app`, `gitcommon` y `cli` (troceado).

## Progreso y evidencia

### T1 · `internal/fileprim`

- API: `Lock`, `AcquireLock`, `AcquireLockAnchored` (con `AnchorResolver`), `ErrLockContended`, `PreAcquisitionError`, `LockOwner`/`LockSchema` (literal `gentle-ai.review-store-lock/v1`, formato en disco), `PublishNoReplace`, `ReplaceAtomic`, `SyncDirectory`, `SyncDirectoryStrict`, `TolerateUnsupportedDirectorySync`, `OpenNoFollow`; en Windows también `OpenPhysicalPath`. `fileprim` no importa ningún paquete de Axiom.
- Se movieron con `git mv`: `secure_open*`, `publish_immutable_*` y `store_lock_{unix,windows}.go`, con sus pruebas; las pruebas de cerrojo que usan `acquireStoreLock`, `inventoryLock` o `CompactAuthoritativeStore` siguen en `reviewtransaction`.
- `reviewtransaction` delega en `fileprim` y conserva su cerrojo anclado, el de mantenimiento, `storeLockBusyError` (con `ErrConcurrentUpdate` en la cadena) y las costuras `syncReviewDirectory`/`reviewRuntimeGOOS`. Se retiraron `AuthorityFileLock`, `PublishFileNoReplace`, `ReplaceFileAtomic` y `OpenPhysicalPath` de `reviewtransaction`, porque ya no tenían consumidores.
- El error de cerrojo ocupado conserva el texto exacto. Fuera de RDD el centinela que se comprueba es `fileprim.ErrLockContended`, igual a `reviewtransaction.ErrStoreLockContended`.
- CI: `./internal/fileprim` entra en el paso Windows release-blocker de `ci.yml`, y `scripts/darwin-release-blockers.manifest` apunta a `fileprim` en 14 pruebas.
- Ruta: delegada, para proteger el contexto del padre (unas 3.000 líneas de lectura).
- Evidencia:
  - `go build ./...` OK.
  - `go vet` limpio en Windows, Linux y Darwin.
  - `go test` ok en `fileprim`, `kickoff`, `sddstatus`, `branch`, `statecoord`, `filecoord`, `integrations`, `components/uninstall` y `tui`.
  - `go test` ok en `cli -run 'StateLock|Compatibility|PersonaMigration|SDDGate|Lineage'` y en `cli -run 'ReviewMode|LockContention'`.
  - `reviewtransaction`, por shards: 4 fallos de entorno por falta del privilegio de crear symlinks en Windows (`os.Symlink`). Dos de ellos se reprodujeron también en `main`.
  - `app`: falla solo `TestTuiInstallOnThenSyncPreservesAndRefreshesOpenCodeActivation`, por «OpenCode runtime version unavailable». Es de entorno y falla igual en `main`.
- Commit: `b58254ec`.

### T2 · consultas de git a `gitcommon`

- `gitcommon.RepositoryRoot` sustituye a `SnapshotBuilder.ResolveRepositoryRoot` en `sddstatus/runtime_ledger.go`, y `gitcommon.IsAncestor` sustituye a `SnapshotBuilder.RevisionIsAncestor` en `cli/sdd_gate_evidence.go`.
- Desvío del plan: no se mueve el runner de `SnapshotBuilder`. Es el motor git de RDD, de unas 1.900 líneas. Las dos consultas de SDD se reimplementan sobre `gitcommon.Run` y `TopLevel`, que ya existen.
- Cambian solo dos cosas, ambas de diagnóstico:
  - el plazo de git pasa de 120 s a 30 s, el de `gitcommon`;
  - el texto del error, cuando la consulta falla.
- El resultado de las consultas no cambia: raíz canónica con contención por `pathidentity`, y ancestro sí/no/error según el código de salida 0/1/otro.
- La sonda `CompactAuthoritativeStore` de `runtime_ledger.go` queda como «parte RDD» y es el único uso de `reviewtransaction` que le queda al fichero.
- Ruta: inline (menos de 100 líneas).
- Evidencia:
  - `go test` ok en `gitcommon`, que incluye las pruebas nuevas de `RepositoryRoot` e `IsAncestor`.
  - `go test` ok en `sddstatus`.
  - `go test` ok en `cli -run 'SDDGate|Gate|Runtime|EditGrant'`.
- Commit: `398f4142`.

### T3 · criterio de terminado

- `rg -l 'internal/reviewtransaction"' -g '*.go'` fuera de `reviewtransaction` devuelve solo ficheros clasificados como «parte RDD» en la tabla:
  - `app/app.go`
  - `cli/branch_lineage*`, `cli/review*`, `cli/sdd_status_*review*_test.go`, `cli/continuation_paths_test.go`, `cli/dual_contract_integration_test.go`, `cli/guard_population_test.go` y `cli/refusal_resolution_ratchet_test.go`
  - `components/sdd/*review*`
  - `reviewerprovider/adapter_minimality_guard_test.go`
  - `kickoff/rdd_boundary_test.go`
  - `sddstatus/main_test.go`, `review_mode_home_test.go` y `review_offer_absence_guard_test.go`
  - `tui/model.go`, `tui/review_store_reset_test.go` y `tui/screens/*review*`
- Además, en cuatro ficheros de prueba el único símbolo que sigue usándose es `RDDModeOn`: `cli/install_state_lock_test.go`, `cli/sync_persona_migration_test.go`, `components/uninstall/state_lock_test.go` y `tui/model_test.go`.
- En `sddstatus/runtime_ledger.go` el único símbolo que sigue usándose es `CompactAuthoritativeStore`.
- No hay ningún fichero de producción neutral que importe `reviewtransaction`.

## Qué queda fuera de RDD (para C-01-e)

- `internal/fileprim` queda fuera de RDD: cerrojo, publicación, reemplazo, sincronización de directorio y apertura sin enlaces. C-01-e no lo toca.
- `internal/gitcommon` queda fuera de RDD: `RepositoryRoot` e `IsAncestor`. C-01-e no lo toca.
- C-01-e tiene dos trabajos pendientes, de distinto tipo:
  - **Sustituir, no borrar:** la sonda `CompactAuthoritativeStore` de `sddstatus/runtime_ledger.go` debe reemplazarse por `gitcommon.CommonDir`, porque el ledger es de SDD.
  - **Retirar:** `RDDModeOn` de las cuatro pruebas citadas en T3.

## Siguiente paso

Abrir la PR y fusionarla con merge commit.

## Corrección tras la CI

- La CI de la PR falló en dos ratchets de `cli`:
  - `TestEveryRegisteredGuardPopulationDeclarationMatchesProduction`;
  - `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`.
- Causa: los dos analizan `reviewtransaction` y `sddstatus`, y el código que pasó a `fileprim` había salido de su cobertura.
- Corrección:
  - se añade `internal/fileprim` a los directorios que analizan los dos ratchets;
  - los baselines solo renombran rutas, y no se congela ninguna frase nueva.
- `fileprim.ErrReparsePoint` pasa a nombrar su salida y lleva la anotación `refusal:by-design`. Antes, `OpenPhysicalPath` devolvía el texto de RAR (`errUnsafeRARAuthorityPath`), que también nombraba la salida. Dentro de RDD, `openRARPathNoFollow` sigue traduciendo este error a `errUnsafeRARAuthorityPath`.
- Evidencia: `go test ./internal/cli -run 'GuardPopulation|Refusal|Ratchet'` ok, y `go test ./internal/fileprim` ok.
- Second CI failure, the dead-code ratchet:
  - `fileprim.NewLockBusyError` had no callers, so it is deleted.
  - `SnapshotBuilder.RevisionIsAncestor` (RDD) had no consumer once `gitcommon.IsAncestor` replaced it, so it is deleted together with its test.
  - 27 functions in `reviewtransaction/verification_contract.go` go into `.deadcode-baseline.txt`. On `main` they were reachable only through reflection (`deadcode -whylive`: "reachable only through reflection"), because SDD code converted a `SnapshotBuilder` value to an interface. This PR removed that value; nothing real calls those functions. C-01-e deletes them along with the package.
  - Evidence: `scripts/deadcode-ratchet.sh` reports "no new unreachable functions".
