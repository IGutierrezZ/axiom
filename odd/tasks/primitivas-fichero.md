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
- [ ] T2 · `SnapshotBuilder` de `sddstatus/runtime_ledger.go` y `cli/sdd_gate_evidence.go` a `gitcommon`. Ruta: inline (<100 líneas).
- [ ] T3 · Comprobar el criterio de terminado (`rg -l 'internal/reviewtransaction"'`) y `go test` de los paquetes tocados.

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
