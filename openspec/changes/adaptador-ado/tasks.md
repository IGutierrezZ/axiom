# Tareas: adaptador Azure DevOps (C-07-c)

> Fase `tasks` del cambio `adaptador-ado`. Rol único sellado: `fullstack` (`tasks.md`). Rutas relativas a la raíz del worktree (`C:\repos\axiom-wt\adaptador-ado\axiom`). Castellano de España; identificadores en inglés. Entradas: `proposal.md`, `design.md` (D-01 a D-12, siete unidades) y las specs delta `azure-devops-adapter` (REQ-45.1 a REQ-45.16), `integration-plugins` (REQ-45.17, MODIFIED REQ-43.4 y MODIFIED REQ-43.18) y `workspace-topology` (MODIFIED REQ-43.21).
>
> **Correcciones de la spec posteriores al diseño (prevalecen sobre `design.md`):** (1) el scope probable de las **iteraciones** es `vso.work`, no `vso.project` (D-05 y open question 3 del diseño); miembros, proyectos y equipos siguen en `vso.project`. (2) MODIFIED REQ-43.18 enumera exactamente `sync`, `emit`, `secret`, `connect` y `discover`, así que la pregunta abierta 1 del diseño queda resuelta y no hay tarea pendiente de spec.
>
> TDD: `strict_tdd: false` en `openspec/config.yaml`; RED observado antes de GREEN solo en las tareas marcadas `[RED]` (reglas de seguridad, redacción del PAT, `test /rev`, importaciones). Runner: `go test` acotado por paquete o con `-run`. **Prohibido** compilar binarios sueltos de `axiom` (`go build -o`, `go install`): el EDR los borra. `internal/cli` completo es muy lento en Windows: siempre con `-run`. Ninguna prueba llama a Azure DevOps real: `httptest` (`adotest`) y dobles.

## Delivery Forecast

| Field | Value |
|-------|-------|
| Estimated size | XL |
| Estimated changed lines | ~3.500 (código y pruebas; ~55 % pruebas) |
| Delivery strategy | `single-pr`: una PR, un commit por unidad, sin `size:exception` |

### Work Units

Orden obligado: **1 → 2 → {3, 4} → 5 → 6 → 7**. La 1 va primera (el catálogo cambia pruebas existentes); 3 y 4 dependen de la 2; 5 depende de la 4; 6 de la 4 y la 5; la 7 cierra.

| Unit | Goal | Commit | Focused test command | Rollback boundary |
|------|------|--------|----------------------|-------------------|
| 1 | Alta del plugin: manifiesto en la hoja, catálogo, `ResolveRegistry`, `builtin`, plugin con `Deliver` permanente, importaciones, bloque en `workspace`, plantilla y manual | `feat(integrations): alta del plugin azure-devops con manifiesto y registro` | `go test ./internal/integrations/... ./internal/workspace -run 'Integrations\|Example\|Docs'` | `manifest/azuredevops.go`, `builtin`, `azuredevops/plugin.go`, `registry.go` |
| 2 | Cliente REST: `Options`, `Clock`, URL, `do()`, reintentos, clasificación, redacción y `adotest` | `feat(integrations): cliente REST de Azure DevOps con reintentos y redacción` | `go test ./internal/integrations/azuredevops/...` | `azuredevops/client.go`, `errors.go`, `adotest/` |
| 3 | Work items, lotes, `PATCH` con `test /rev` y repositorios con caché | `feat(integrations): work items, PATCH con rev y repositorios de Azure DevOps` | `go test ./internal/integrations/azuredevops -run 'WorkItem\|Patch\|Repos'` | `workitems.go`, `repos.go` |
| 4 | Puertos opcionales en `integrations` y descubrimiento del adaptador | `feat(integrations): puertos opcionales y descubrimiento de Azure DevOps` | `go test ./internal/integrations -run 'Port\|Discover' ./internal/integrations/azuredevops -run 'Discover'` | `ports.go`, `discovery.go` |
| 5 | Identidad del adaptador e `identitystore` | `feat(integrations): identidad del adaptador y fichero de identidad por persona` | `go test ./internal/integrations/identitystore ./internal/integrations/azuredevops -run 'Identity\|Store'` | `identity.go`, `identitystore/` |
| 6 | Verbos `connect` y `discover`, ayuda de familia y global, manual y plantilla con los verbos | `feat(integrations): verbos connect y discover` | `go test ./internal/cli -run 'Integrations' ./cmd/axiom -run 'IntegrationsHelp\|Help'` | `integrations_ports.go`, `integrations.go`, `cmd/axiom/main.go` |
| 7 | Canario de extremo a extremo, ratchet de código muerto y comprobación final | `test(integrations): canario del PAT de Azure DevOps y cierre` | `go test ./internal/cli -run 'ADOCanary'` | `integrations_ado_canary_test.go` |

## Precondiciones y notas de ejecución

- [x] P.1 Sin dependencias nuevas: solo biblioteca estándar. No ejecutar `go get` ni `go mod tidy`.
- [x] P.2 Las pruebas que usan costuras (`SetDefaultRegistryForTest`, `SetCatalogForTest`, `secrets.SetDefaultForTest`) no llaman a `t.Parallel()`. Ninguna prueba espera tiempo real, salvo el timeout (`Timeout: 20ms`, `MaxAttempts: 1`).
- [x] P.3 Cada unidad termina con `gofmt -l` vacío y `go vet` sobre los paquetes tocados antes de su commit. Commits convencionales, sin atribución de IA.
- [ ] P.4 Las specs delta no son tareas de `apply`: las integra `sdd-archive`. Confirmar antes de archivar que `REQ-45` está libre en `main`.

## Phase 1 (unidad 1): Alta del plugin

- [x] 1.1 [RED] Crear `internal/integrations/manifest/azuredevops_test.go`: `AzureDevOps()` valida; capacidades exactas `items.read` y `discovery` (sin capacidades de escritura); evento `change.start`; claves `organization`, `project`, `team` string y obligatorias; secreto `pat` obligatorio; identidad `plugin` con secreto `pat`; la variable es `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`; cada llamada devuelve un valor nuevo. Fallar antes de 1.2. Prueba: `go test ./internal/integrations/manifest -run AzureDevOps`.
  Cubre: REQ-45.1
- [x] 1.2 Crear `internal/integrations/manifest/azuredevops.go` (`AzureDevOpsID`, `AzureDevOps()`, D-11) y modificar `catalog.go` (`builtinCatalog = []Manifest{AzureDevOps()}`) y `catalog_test.go` (catálogo por defecto = exactamente `azure-devops`; la costura lo restaura; el `leaf_test.go` sigue verde). Prueba: `go test ./internal/integrations/manifest`.
  Cubre: REQ-45.1, MOD REQ-43.4
- [x] 1.3 [RED] Modificar `internal/integrations/registry.go`: retirar `builtinPlugins` y `DefaultRegistry()`, añadir `ResolveRegistry(builtins []Plugin)` que respeta `SetDefaultRegistryForTest` (D-01). Actualizar `registry_test.go`, `helpers_test.go` e `integrationstest/fake_test.go` (`ResolveRegistry(nil)`), y el comentario de `event.go`. Prueba: `go test ./internal/integrations -run Registry` y `go test ./internal/integrations/integrationstest`.
  Cubre: MOD REQ-43.4
- [x] 1.4 Crear `internal/integrations/azuredevops/plugin.go` (`Plugin`, `New(Options)` con `Options` mínima por ahora, `Manifest()` = `manifest.AzureDevOps()`, `Deliver` que devuelve `integrations.Permanent(ErrLifecycleUnsupported)` sin red ni disco, mapa de sesiones por organización en minúsculas, lectura de `Config` con `ErrConfig` ante vacío) con prueba: `Deliver` → `IsPermanent` y cero peticiones. Crear `internal/integrations/builtin/builtin.go` (`Plugins()`, `DefaultRegistry()`) y `builtin_test.go` (los ids de `DefaultRegistry()` coinciden exactamente con `manifest.Catalog()`; sustituye a `TestDefaultRegistryIsEmptyAndMatchesCatalog`). Prueba: `go test ./internal/integrations/azuredevops ./internal/integrations/builtin`.
  Cubre: REQ-45.14, MOD REQ-43.4
- [x] 1.5 [RED] Ampliar `internal/integrations/imports_test.go` con las reglas (a) el paquete raíz no importa transitivamente `azuredevops` ni `builtin`, (b) fuera de `builtin` y `azuredevops/adotest` ningún fichero no de prueba importa `azuredevops`, (c) `identitystore` solo importa biblioteca estándar, `internal/system` y `internal/components/filemerge`, y una prueba de no vacuidad que detecta una violación sembrada en un `TempDir`. La regla (c) se declara ahora y se aplica cuando exista el paquete (unidad 5). Prueba: `go test ./internal/integrations -run Imports`.
  Cubre: REQ-45.1 (arquitectura), MOD REQ-43.4
- [x] 1.6 Cambiar `internal/cli/integrations_events.go:32` a `builtin.DefaultRegistry()` y añadir en `internal/cli/integrations_events_test.go` la prueba de REQ-45.14: `gate record --decision approved` sobre `spec` con y sin el bloque `azure-devops` da stdout, stderr y código idénticos, sin `integrations.yaml` ni outbox. Prueba: `go test ./internal/cli -run 'IntegrationsEvents'`.
  Cubre: REQ-45.14, MOD REQ-43.4
- [x] 1.7 Modificar `internal/workspace/integrations_test.go`: bloque válido `azure-devops` (carga con el catálogo por defecto); `base_url` desconocida rechazada; `pat` rechazada; `jira` desconocido nombra `azure-devops` entre los disponibles. Actualizar `internal/hub/init_baseline_test.go` (solo el comentario de la línea 591). Prueba: `go test ./internal/workspace -run Integrations` y `go test ./internal/hub -run Baseline`.
  Cubre: REQ-45.2 (carga)
- [x] 1.8 Plantilla y manual (parte de MOD REQ-43.21): añadir a `axiom.example.yaml` el bloque comentado `integrations.azure-devops` (`organization`, `project`, `team`) y a `docs/manual-de-inicio.md` el secreto `pat`, la variable, el PAT de una sola organización y los scopes. Ajustar `internal/workspace/example_config_test.go` y `docs_contract_test.go`: la plantilla contiene el bloque; descomentado carga con el catálogo por defecto; nada exige «catálogo vacío». Prueba: `go test ./internal/workspace -run 'Example\|Docs'`.
  Cubre: MOD REQ-43.21
- [x] 1.9 Cierre de la unidad: `gofmt -l internal/integrations internal/cli internal/workspace internal/hub`, `go vet` sobre esos paquetes, `go test ./internal/integrations/...`; commit `feat(integrations): alta del plugin azure-devops con manifiesto y registro`.
  Cubre: REQ-45.1, REQ-45.14, MOD REQ-43.4, MOD REQ-43.21

## Phase 2 (unidad 2): Cliente REST

- [x] 2.1 Crear `internal/integrations/azuredevops/adotest/{server.go,clock.go,install.go}`: servidor `httptest` programable por ruta con registro de peticiones y tiempo virtual, simulación de `rev`, reloj manual (`Sleep` registra, ejecuta gancho opcional, devuelve `ctx.Err()` si vence, avanza sin dormir) e `Install(t, srv)` (catálogo por defecto, registro con el plugin apuntando al servidor, almacén en memoria). Prueba propia mínima en `adotest/server_test.go`. Comando: `go test ./internal/integrations/azuredevops/adotest`.
  Cubre: REQ-45.16
- [x] 2.2 [RED] Crear `azuredevops/client_test.go` (paquete `azuredevops_test`): URL con segmentos escapados y host intacto con `/` o `?` en la organización; `api-version=7.1`; `Authorization` `Basic base64(":"+PAT)` exacto; `User-Agent: axiom`; PAT resuelto en cada petición; secreto ausente → error `*secrets.MissingError` sin ninguna petición HTTP; redirecciones no seguidas; timeout por intento; cancelación del `ctx` sin más peticiones. Debe fallar antes de 2.3.
  Cubre: REQ-45.2, REQ-45.3
- [x] 2.3 Crear `azuredevops/client.go` (`Options`, `Clock` con reloj real, `Target`, `Client`, `orgState`, `do()` con la secuencia de D-04 y sesiones por organización en `Plugin`). Ampliar `plugin.go` con `Options` completas. Prueba: 2.2 en verde.
  Cubre: REQ-45.2, REQ-45.3
- [x] 2.4 [RED] Crear `azuredevops/retry_test.go`: 429 con `Retry-After` (segundos y fecha HTTP) espera al menos lo indicado; `Retry-After` en 200 fija el ritmo de la siguiente petición; 503 con retroceso 1 s, 2 s, 4 s y tope; `Retry-After` > 60 s no se reintenta; presupuesto agotado → `RateLimitError{Attempts}`, `APIError` 503 o `TransportError` según el caso; `X-RateLimit-*` expuestas con `RateLimit()` y `X-RateLimit-Resource` ignorada; corte de conexión reintentado; 500, 502 y 504 sin reintento. Todo con el reloj manual.
  Cubre: REQ-45.4
- [x] 2.5 Implementar el ritmo (`notBefore`), la lectura de cabeceras y los reintentos en `do()`. Prueba: 2.4 en verde.
  Cubre: REQ-45.4
- [x] 2.6 [RED] Crear `azuredevops/errors_test.go`: tabla de D-05 (401, 403, 203, 200 HTML, 3xx → `AuthError` con scope y sin reintento; 404; 400; otro 4xx; 5xx; 204; JSON inválido y cuerpo > 16 MiB → `ErrInvalidResponse`); mensaje de `AuthError` sin fechas de retirada; cuerpo de error de megabytes acotado a 512 runas; eco del PAT, de su base64 y de `Basic <b64>` en el cuerpo o en la causa de transporte redactados; el error no incluye cabeceras, cuerpo de petición ni consulta de la URL. Tabla `TestIsRevTestRejection` (412; 400/409 con `typeKey` o `message` de `test operation`; 400 ajeno no es rechazo).
  Cubre: REQ-45.5, REQ-45.7 (clasificación), REQ-45.15 (eco)
- [x] 2.7 Crear `azuredevops/errors.go` (centinelas, tipos, clasificación, `isRevTestRejection`, redactor de D-06 con valores de 4 o más caracteres). Scope probable por petición: work items, lotes, `PATCH`, tipos, estados y campos → `vso.work_write`; **iteraciones → `vso.work`** (corrección de la spec); miembros, proyectos y equipos → `vso.project`; repositorios → `vso.code`. Prueba: 2.6 en verde.
  Cubre: REQ-45.5, REQ-45.15
- [x] 2.8 Cierre de la unidad: `gofmt -l internal/integrations`, `go vet ./internal/integrations/...`, `go test ./internal/integrations/azuredevops/...`; commit `feat(integrations): cliente REST de Azure DevOps con reintentos y redacción`.
  Cubre: REQ-45.2, REQ-45.3, REQ-45.4, REQ-45.5, REQ-45.15, REQ-45.16

## Phase 3 (unidad 3): Work items y repositorios

- [x] 3.1 [RED] Crear `azuredevops/workitems_test.go` (lectura y lotes): `GetWorkItem` con `$expand=relations`; 404 → `NotFoundError` con el id; lista vacía sin HTTP; lote de 450 ids → 3 `POST` de 200 como máximo con el orden pedido conservado aunque la API lo desordene; id inexistente con `errorPolicy: Fail` → `NotFoundError`.
  Cubre: REQ-45.6
- [x] 3.2 Crear `azuredevops/workitems.go` (`WorkItem`, `Relation`, `GetWorkItem`, `GetWorkItems`). Prueba: 3.1 en verde.
  Cubre: REQ-45.6
- [x] 3.3 [RED] Crear `azuredevops/patch_test.go`: `newPatchBody` es la única forma de construir el cuerpo y antepone `test /rev`; rechaza `rev <= 0` (`ErrMissingRev`), ops vacías y un `test` sobre `/rev` aportado por quien llama; PATCH con rev vigente → rev 8; con rev obsoleto → una sola petición y ningún cambio; 412, 400 y 409 de `test` → `RevConflictError`; 400 ajeno → `ErrBadRequest`; corte tras aplicar y reintento rechazado → conflicto con `AfterRetry` y una sola aplicación; nunca se reintenta tras un 4xx.
  Cubre: REQ-45.7
- [x] 3.4 Implementar `PatchOp`, `UpdateWorkItem` y `newPatchBody` en `workitems.go` (`application/json-patch+json`, `revPatch`, `retryable`). Prueba: 3.3 en verde. No se declara capacidad de escritura en el manifiesto.
  Cubre: REQ-45.7
- [x] 3.5 [RED] Crear `azuredevops/repos_test.go` y `repos.go` (`Repository`, `ListRepositories` con caché por proyecto en `orgState`): segunda llamada sin petición; un fallo (500) no se cachea y la siguiente repite; caché por organización y proceso. Prueba: `go test ./internal/integrations/azuredevops -run Repos`.
  Cubre: REQ-45.8
- [x] 3.6 Cierre de la unidad: `gofmt -l internal/integrations`, `go vet ./internal/integrations/azuredevops/...`, `go test ./internal/integrations/azuredevops/...`; commit `feat(integrations): work items, PATCH con rev y repositorios de Azure DevOps`.
  Cubre: REQ-45.6, REQ-45.7, REQ-45.8

## Phase 4 (unidad 4): Puertos y descubrimiento

- [x] 4.1 [RED] Crear `internal/integrations/ports_test.go` con dobles locales: aserción de tipo detecta `IdentityResolver` y `Discoverer`; plugin sin puerto → `PortUnsupportedError` (`errors.Is(ErrPortUnsupported)`) antes de resolver secretos; el error se sanea con un doble que filtra el secreto y conserva la causa con `errors.As`; un pánico se recupera como error; `Discover` fija `Schema` y `Plugin` y devuelve listas no nulas (sin `null` en JSON); `ResolveIdentity` rechaza `ID` o `Account` vacíos.
  Cubre: REQ-45.17 (núcleo)
- [x] 4.2 Crear `internal/integrations/ports.go` (puertos, tipos `Discovery`, `ItemType`, `ItemState`, `ItemField`, `Iteration`, `Member`, `TrackerIdentity`, errores tipados, `ResolveIdentity` y `Discover` de D-02, `Plugin` sin cambios). Prueba: 4.1 en verde y `go test ./internal/integrations -run Imports`.
  Cubre: REQ-45.17
- [x] 4.3 [RED] Crear `azuredevops/discovery_test.go`: descubrimiento completo (tipos, estados con categoría copiada tal cual incluidas `InProgress` y `Removed`, campos, iteraciones, miembros); paginación con 250 miembros y página de 100 sin repetidos; servidor que ignora `$skip` termina; iteraciones vacías y en `values` o en `value`; un 403 en miembros hace fallar todo el descubrimiento con `vso.project`; un 403 en iteraciones nombra **`vso.work`**; ningún correo (`uniqueName`) en la salida.
  Cubre: REQ-45.9
- [x] 4.4 Crear `azuredevops/discovery.go` (`Plugin.Discover`, `listMembers` reutilizable para la unidad 5, D-08 y D-12). Prueba: 4.3 en verde.
  Cubre: REQ-45.9
- [x] 4.5 Cierre de la unidad: `gofmt -l internal/integrations`, `go vet ./internal/integrations/...`, `go test ./internal/integrations -run 'Port|Discover|Imports'` y `go test ./internal/integrations/azuredevops -run Discover`; commit `feat(integrations): puertos opcionales y descubrimiento de Azure DevOps`.
  Cubre: REQ-45.9, REQ-45.17

## Phase 5 (unidad 5): Identidad y fichero de identidad

- [x] 5.1 [RED] Crear `azuredevops/identity_test.go`: `connectionData` válido → `Source: connection_data` sin consultar miembros; 401 de `connectionData` o `authenticatedUser` vacío → `IdentitySelectionError` con candidatos; `User` válido (sin distinguir mayúsculas) → `member_selection` sin consultar `connectionData`; `User` desconocido → `IdentitySelectionError{User}`; si `listMembers` también falla se devuelve ese error con la nota de `connectionData`; la cancelación del `ctx` se propaga.
  Cubre: REQ-45.10
- [x] 5.2 Crear `azuredevops/identity.go` (`Plugin.ResolveIdentity`, D-08). Prueba: 5.1 en verde.
  Cubre: REQ-45.10
- [x] 5.3 [RED] Crear `internal/integrations/identitystore/store_test.go`: primera escritura (modo 0600 solo fuera de Windows); dos organizaciones coexisten; sustitución de una entrada; organización normalizada a minúsculas; fallo de escritura con la costura `writeAtomic` deja el fichero anterior byte a byte y sin ficheros nuevos; JSON corrupto o `schema` desconocido → `*CorruptError` sin sobrescribir; `workspaces.json` intacto; inexistente → `File` vacío. Usa `t.TempDir()` y `AXIOM_STATE_DIR`.
  Cubre: REQ-45.11
- [x] 5.4 Crear `internal/integrations/identitystore/store.go` (`SchemaV1`, `FileName`, `Entry`, `File`, `Store`, `DefaultPath`, `Load`, `Get`, `Put`, `CorruptError`, `writeAtomic = filemerge.WriteFileAtomicMode`). Prueba: 5.3 en verde y `go test ./internal/integrations -run Imports` (regla (c)).
  Cubre: REQ-45.11
- [x] 5.5 Cierre de la unidad: `gofmt -l internal/integrations`, `go vet ./internal/integrations/...`, `go test ./internal/integrations/identitystore ./internal/integrations/azuredevops -run 'Identity|Store'`; commit `feat(integrations): identidad del adaptador y fichero de identidad por persona`.
  Cubre: REQ-45.10, REQ-45.11

## Phase 6 (unidad 6): Verbos `connect` y `discover`

- [x] 6.1 [RED] Crear `internal/cli/integrations_ports_test.go` (`RunIntegrations` en proceso con `adotest.Install`): `connect` sin `--plugin` → 2; plugin desconocido → 1 con lista del catálogo antes de leer el workspace; plugin no configurado o `enabled: false` → 1; éxito → código 0, salida legible y con `--json` `axiom.integrations-connect/v1`, identidad guardada con 0600; `IdentitySelectionError` → 1, nada guardado, candidatos desde `Candidates` (legible en stderr y `status: selection_required` en `--json`); `connect --user` válido; `discover` legible y `--json` (`axiom.integration-discovery/v1`) sin escribir ficheros (instantánea del workspace y de `AXIOM_STATE_DIR` antes y después); plugin sin puertos (`integrationstest.Fake`) → 1 con `ErrPortUnsupported` y sin efectos.
  Cubre: REQ-45.12, REQ-45.13, REQ-45.17 (CLI)
- [x] 6.2 Crear `internal/cli/integrations_ports.go` (`resolvePortTarget`, `connect`, `discover`, D-10) y modificar `internal/cli/integrations.go` (`switch`, ayuda de la familia y mensaje de subcomando desconocido con «sync, emit, secret, connect y discover»). Prueba: 6.1 en verde y `go test ./internal/cli -run Integrations`.
  Cubre: REQ-45.12, REQ-45.13, REQ-45.17, MOD REQ-43.18
- [x] 6.3 [RED] Modificar `cmd/axiom/integrations_help_test.go`: `axiom --help` con las cinco líneas de `integrations` alineadas (D-10) y sin verbos inventados; `integrations --help` y `connect|discover --help`. Después modificar `cmd/axiom/main.go` (líneas 127-129). Prueba: `go test ./cmd/axiom -run 'IntegrationsHelp|Help'`.
  Cubre: MOD REQ-43.18
- [x] 6.4 Manual y plantilla (resto de MOD REQ-43.21): `docs/manual-de-inicio.md` documenta `axiom integrations connect` y `discover` (opciones, códigos de salida, fichero `~/.axiom/integration-identities.json`) y que el PAT de iteraciones necesita **`vso.work`** (cubierto por `vso.work_write`); ajustar `docs_contract_test.go` para exigir la mención de `pat`, la variable, los scopes y ambos verbos. Prueba: `go test ./internal/workspace -run 'Example|Docs'`.
  Cubre: MOD REQ-43.21
- [x] 6.5 Cierre de la unidad: `gofmt -l internal/cli cmd/axiom internal/workspace`, `go vet ./internal/cli/... ./cmd/axiom/...`, `go test ./internal/cli -run Integrations` y `go test ./cmd/axiom -run 'IntegrationsHelp|Help'`; commit `feat(integrations): verbos connect y discover`.
  Cubre: REQ-45.12, REQ-45.13, REQ-45.17, MOD REQ-43.18, MOD REQ-43.21

## Phase 7 (unidad 7): Canario y cierre

- [x] 7.1 [RED] Crear `internal/cli/integrations_ado_canary_test.go` (`TestADOCanary...`): PAT canario en el entorno; `connect` con y sin `--user`, `discover` y servidores en modo 401, 403, 429 (presupuesto agotado) y 500; ni el canario ni su base64 ni `Basic <b64>` aparecen en stdout, stderr, errores, fichero de identidad, `axiom.yaml` ni salida de `discover`. Prueba: `go test ./internal/cli -run 'ADOCanary'`.
  Cubre: REQ-45.15
- [x] 7.2 Ratchet de código muerto: ejecutar `bash scripts/deadcode-ratchet.sh` y corregir cualquier función exportada sin llamador (por ejemplo `RateLimit()`, `UpdateWorkItem`, `GetWorkItems`, `ListRepositories`: dar un llamador real o ajustar a no exportada, y solo como último recurso `--update` con justificación). Si el entorno no puede instalar la herramienta, dejarlo registrado como no ejecutado y delegarlo a la CI.
  Cubre: calidad (CI)
- [x] 7.3 Comprobación final por paquetes (nunca `go test ./...` completo ni binarios sueltos): `go test ./internal/integrations/...`, `go test ./internal/workspace/... ./internal/hub/...`, `go test ./cmd/axiom -run 'IntegrationsHelp|Help'`, `go test ./internal/cli -run 'Integrations|ADOCanary|GateRecord|BranchEnsure|SDDPublish'`, `go vet` y `gofmt -l` sobre `internal/integrations internal/cli internal/workspace internal/hub cmd/axiom`. Confirmar que no hay goldens afectados (la ayuda se cubre con pruebas de contenido) y que `internal/hub/manager.go` no cambió.
  Cubre: REQ-45.1 a REQ-45.17, MOD REQ-43.4, MOD REQ-43.18, MOD REQ-43.21
- [x] 7.4 Commit `test(integrations): canario del PAT de Azure DevOps y cierre`.
  Cubre: REQ-45.15

## Registro de remediación

(vacío)
