# Tareas: Ajustes del proyecto desde el dashboard (C-06-e)

Cambio `ajustes-proyecto`. Rol único `fullstack`. Cada tarea es una unidad de trabajo (código, pruebas y docs) cerrada con un único commit conventional, sin atribución de IA. Toda verificación es localizada: nunca la suite entera (queda para la CI), sin `t.Parallel` con `freePort` ni el hub compartido, y sin compilar binarios sueltos.

## Previsión de entrega

| Campo | Valor |
|-------|-------|
| Tamaño estimado | L |
| Líneas modificadas estimadas | 3.200 – 4.600 (código, pruebas, goldens y docs; sin generados) |

### Unidades de trabajo

| Unidad | Objetivo | Commit | Comando de prueba localizado | Arnés de ejecución | Límite de reversión |
|--------|----------|--------|------------------------------|--------------------|---------------------|
| 1 | Accesores de vocabulario cerrado en `workspace` y `multirole` | `refactor(workspace): exponer los vocabularios cerrados de claves de axiom.yaml` | `go test ./internal/workspace ./internal/multirole ./internal/hub -run 'Promotion\|WorkflowLane\|ExplainerHTML\|GatePolic\|PolicyViolations'` | N/A: refactorización sin comportamiento nuevo; las pruebas existentes fijan los mensajes | Los tres ficheros de `workspace`, `multirole/types.go` e `init_validate.go` |
| 2 | Paquete `internal/settings`: catálogo, lectura y errores tipados | `feat(settings): catálogo de claves de axiom.yaml y lectura de valores` | `go test ./internal/settings -run 'Catalog\|Read\|Document'` | N/A: paquete hoja sin HTTP; pruebas con documentos en memoria y `t.TempDir()` | Ficheros nuevos de `internal/settings` (sin consumidores) |
| 3 | Escritor de round-trip con `restoreLayout`, revisión y validación | `feat(settings): escritura de axiom.yaml con round-trip, revisión y validación` | `go test ./internal/settings -run 'Write\|Patch\|Layout\|Revision\|Engine\|RoundTrip'` | Golden `commented.lane-sdd.golden` y ida y vuelta de `axiom.yaml` y `axiom.example.yaml` reales | `write.go`, `patch.go`, `layout.go` y sus `testdata` |
| 4 | Endpoints `/api/settings`, `isLocalRequest` en `worktree-prefs` y canario | `feat(dashboard): endpoints GET y PUT de /api/settings` | `go test ./internal/dashboard -run 'Settings\|WorktreePrefs\|SecretsCanary\|Classified'` | Servidor de pruebas con `newPrefsServer` y `request`/`requestFrom` (host ajeno ⇒ 403) | `settings_api.go`, rutas en `server.go`, `types.go`, `service.go` y `worktree_prefs.go` |
| 5 | Endpoint personal `/api/settings/personal` (autofetch) | `feat(dashboard): endpoint de preferencias personales con autofetch` | `go test ./internal/dashboard -run 'SettingsPersonal\|WorktreePrefs\|SecretsCanary'` | `newPrefsServer` con `workspaces.json` temporal | Handler, rutas y `Service.hub()` |
| 6 | Pantalla «Ajustes del proyecto» (`#/ajustes`) | `feat(dashboard): pantalla Ajustes del proyecto generada desde el catálogo` | `go test ./internal/dashboard -run 'SettingsAssets\|SettingsRuntime\|Assets\|Server.*Marker'` | `runNodeModules` sobre `core/settings.js` (se omite sin Node) | `core/settings.js`, `screens/ajustes.js`, router, nav, main, shell, icons, `index.html` y CSS |
| 7 | Retirar la política de ramas de Ecosistema y enlazar a Ajustes | `refactor(dashboard): retirar la política de ramas de solo lectura de Ecosistema` | `go test ./internal/dashboard -run 'BranchPolicy\|Ecosystem\|WorktreePrefs\|WorkspaceIsReadOnly\|Assets'` | `runNodeModules` de `core/branches.js` con cuatro funciones | `ecosystem.js`, `branches.js`, pruebas de `assets_branch_test.go` y `server_test.go` |
| 8 | Documentación del flujo | `docs(dashboard): documentar los ajustes del proyecto desde el dashboard` | `go test ./internal/dashboard ./internal/idmigration -run 'Docs\|Residue\|Settings'` | N/A: solo documentación | `docs/usage.md`, `docs/codebase/dashboard.md`, `axiom.example.yaml` |
| 9 | Cierre de verificación agregada | `chore(settings): cerrar la verificación localizada de ajustes-proyecto` (solo si hay correcciones; si no, sin commit) | Todos los comandos localizados anteriores, juntos | Recorrido manual en `axiom ui` solo si la persona lo pide | Correcciones de cierre |

## Fase 1: Cimientos de vocabulario

- [x] 1.1 Escribir antes la prueba (RED) de que cada accesor devuelve una copia nueva por llamada: `PromotionValues()` (`branch, release, ask`) en `internal/workspace/gitpolicy_promotion_test.go`, `WorkflowLanes()` (`odd, sdd, ask`) en `internal/workspace/workflow_test.go`, `ExplainerHTMLModes()` (`ask, always, never`) en `internal/workspace/sdd_test.go` y `GatePolicies()` (`blocking, deferred, optional`) en `internal/multirole/types_test.go`.
- [x] 1.2 Crear `PromotionValues` en `internal/workspace/gitpolicy.go`, `WorkflowLanes` en `internal/workspace/workflow.go` y `ExplainerHTMLModes` en `internal/workspace/sdd.go`; `validPromotion`, `workflowValidator` y `sddExplainerHTML` pasan a `slices.Contains` sobre ellos, sin cambiar el texto de los errores.
- [x] 1.3 Crear `GatePolicies` en `internal/multirole/types.go` y hacer que `policyViolations` de `internal/hub/init_validate.go` lo use, sin cambiar su mensaje.
- [x] 1.4 Verificar que las pruebas existentes de mensajes siguen en verde con el comando de la unidad 1. Cada función exportada tiene así uso de producción doble (validador y, desde la unidad 2, catálogo). Evidencia: commit 71c56d0ff.

Cubre: RF-TOP-21

Recordatorio de CI: trinquete de código muerto (`scripts/deadcode-ratchet.sh`): las cuatro funciones no se exportan solo para pruebas; las consume el validador en esta misma unidad.

## Fase 2: Catálogo y lectura

- [x] 2.1 Escribir antes las pruebas (RED) en `internal/settings/catalog_test.go`: `Allowed` igual a los cuatro accesores; coherencia en los dos sentidos (cada valor permitido lo acepta `workspace.ParseConfig` y `zzz` lo rechaza); filtros (monorepo y topología ilegible sin `repositories[...]`, `acceptance_tests.*` ni `roles[...]`; multirepo con ellos); ISO sin `odd` en `workflow.lane`; claves de solo lectura (`workspace.topology`, `specs_repository`, `root`, `skill_roots`, `technical_context`, `governance.<clave>` declaradas); exclusiones (`integrations`, `artifact_store`, `rdd`, `commit_review`, `docs_publication`); `sdd.id_pattern` con valor por defecto de `reqid.Default().Template()`. Las fixtures usan `RF-<DOM>-<NN>`, nunca `REQ-`.
- [x] 2.2 Crear `internal/settings/errors.go` con `Code` (diez códigos de la tabla del diseño) y `Error{Code, Message, Violations}`, con mensajes en castellano.
- [x] 2.3 Crear `internal/settings/catalog.go`: `ValueType`, `keySpec`, tabla del catálogo, expansión por documento (`repositories[<path>]` normalizado y `roles[<id>]`), filtros de topología e ISO, análisis de rutas con selector hasta el último `].` y valores por defecto desde constantes (`DefaultIntegrationBranch`, `DefaultBranchPrefix`, `DefaultWorkflowLane`, `ExplainerHTMLAsk`, `multirole.PolicyBlocking`, `reqid.Default().Template()`; ningún literal `REQ-`). Efectos explícitos de `sdd.iso` y `sdd.id_pattern` y aviso común «solo afecta a cambios nuevos».
- [x] 2.4 Escribir antes la prueba (RED) en `internal/settings/document_test.go`: `Read` devuelve el valor tal como está escrito en un documento inválido, `Revision` SHA-256, `ConfigPath`, `ISO`, `Topology`; multidocumento, anclas, alias y `<<` dan `Writable=false` con `WriteBlockedReason` que nombra la construcción y la línea; `DocumentErrors` viene de `ParseConfig`; el bloque `integrations` no se recorre.
- [x] 2.5 Crear `internal/settings/document.go`: lectura del fichero, revisión, decodificación con `yaml.NewDecoder` (segunda llamada a `Decode` debe dar `io.EOF`), detección de anclas, alias y `<<`, navegación del árbol y `Read(configPath) (*Snapshot, error)`. Evidencia: commit b6e0fbff6.

Cubre: RF-TOP-21

Verificación: `go test ./internal/settings -run 'Catalog|Read|Document'`.

Recordatorios de CI: `internal/idmigration/residues.yaml` prohíbe `REQ-` en Go de producción y en fixtures; ninguna exportación solo para pruebas.

## Fase 3: Escritor

- [x] 3.1 Escribir antes las pruebas (RED) en `internal/settings/layout_test.go`: ida y vuelta byte a byte, sin cambios, de los `axiom.yaml` y `axiom.example.yaml` reales del repositorio (rutas relativas a la raíz del módulo); líneas repetidas (`- path: "."`); CRLF y BOM conservados.
- [x] 3.2 Crear `internal/settings/layout.go` con `restoreLayout(original, encoded []byte) []byte` (LCS de líneas, líneas en blanco, línea en blanco ante clave de primer nivel nueva, CRLF y BOM).
- [x] 3.3 Escribir antes las pruebas (RED) en `internal/settings/patch_test.go` y crear `internal/settings/patch.go`: conversión por tipo; cambio de escalar conservando `Style`; creación al final del mapeo con cadena en `DoubleQuotedStyle`; bloque nuevo en orden canónico `workspace, git, repositories, workflow, sdd, delivery, acceptance_tests`; `repositories[<p>].<campo>` creado y borrado (entrada y bloque); `null` borra la clave y el bloque vacío (`delivery`), y `workspace.name` con `null` da `invalid_value`; `roles[x].gate_policy` en minúsculas sin borrar nunca el mapeo del rol; clave repetida da `unsupported_document`.
- [x] 3.4 Escribir antes las pruebas (RED) en `internal/settings/write_test.go` con `testdata/commented.yaml` y `commented.lane-sdd.golden` (solo cambia una línea; cabecera, línea, pie y blancos conservados), y crear `internal/settings/write.go`: `Write` con revisión (`invalid_request` si falta, `revision_mismatch` si difiere), parche por lotes, serialización con `SetIndent(2)`, `restoreLayout`, `Changed:false` sin escribir si no hay cambio, `ParseConfig` completo (`validation_failed` con una violación por línea), `engineErrors(cfg, master)` (`Validate`, `ValidateDeclarationPaths`, `ValidateRepositoryPolicies`) con diferencia contra el original (los previos vuelven como `Warnings`), costura `beforeReplace`, nueva comprobación de revisión y `filemerge.WriteFileAtomic(path, nuevos, 0o644)`.
- [x] 3.5 Cubrir en `write_test.go`: `git.promotion: siempre` con el mensaje del validador; ISO en las dos direcciones y en un mismo lote válido (`workflow.lane: odd` más `sdd.iso: false`); claves desconocidas (`unknown_key`, incluidas `integrations.*`), de solo lectura (`key_not_editable`) y no aplicables (`key_not_applicable`); cada rechazo comprueba los bytes del fichero sin cambios y el directorio sin temporales; cambio externo entre validación y reemplazo con `beforeReplace` (sin `t.Parallel`, modifica una variable de paquete); error previo de `Validate` (rol `core` en monorepo) no bloquea y vuelve como aviso, y un error nuevo (entrada de `repositories` hacia una ruta inexistente) bloquea. Evidencia: commit 717a013b7.

Cubre: RF-TOP-22, RF-TOP-23

Verificación: `go test ./internal/settings -run 'Write|Patch|Layout|Revision|Engine|RoundTrip'`.

Recordatorio de CI: ninguna exportación solo para pruebas (`beforeReplace` es variable no exportada).

## Fase 4: Endpoints del dashboard

- [x] 4.1 Escribir antes las pruebas (RED) en `internal/dashboard/settings_api_test.go` con `newPrefsServer`, `request` y `requestFrom`: GET con catálogo, `revision`, `configPath` y sin `integrations` ni secretos en el cuerpo; PUT 200, 400, 403 (host ajeno, sin leer el cuerpo), 404 (`config_missing`), 409, 422 y 405 en POST, PATCH y DELETE; multirepo con `workspacetest.Canonical` escribe `repo-specs/axiom.yaml` y no crea `axiom.yaml` en la maestra ni en el código; un `kickoff.yaml` sellado no cambia (`snapshotTree` antes y después, salvo `axiom.yaml`); `PUT worktree-prefs` con host ajeno da 403 sin escribir el hub y el GET sigue normal.
- [x] 4.2 Añadir a `internal/dashboard/types.go` `SettingsDTO`, `SettingDTO`, `SettingsPutRequest`, `SettingsPutResponse` y `settingsErrorBody` (camelCase), y el campo `settingsMu sync.Mutex` a `internal/dashboard/service.go`.
- [x] 4.3 Crear `internal/dashboard/settings_api.go`: `handleSettings`, `Service.GetSettings`, `Service.PutSettings` (bajo `settingsMu`, ruta de `workspace.Locate(root).ConfigPath`, `Master` de `loc.Master`) y la función única que traduce `settings.Code` a HTTP (400, 404, 409, 422, 500; cuerpo `{code, error, violations?}`). `isLocalRequest` va lo primero en el PUT.
- [x] 4.4 Registrar `HandleFunc("/api/settings", …)` con literal de cadena en `internal/dashboard/server.go` (la guarda `registeredRoutes` solo analiza ese fichero) y actualizar el comentario de `handleWorkspace`.
- [x] 4.5 Añadir `isLocalRequest` al inicio del `PUT` de `internal/dashboard/worktree_prefs.go` (antes de leer el cuerpo, mismo cuerpo de 403 que `server.go`).
- [x] 4.6 Añadir `/api/settings` a `coveredRoutes` en `internal/dashboard/secrets_canary_test.go` y comprobar que `TestNoReadOnlyDashboardRouteReturnsAnIntegrationSecret` sigue en verde sin cambiar su estructura. Evidencia: commit af2d8507c.

Cubre: RF-WEB-18, RF-WEB-19, RF-WEB-10, RF-WEB-01

Verificación: `go test ./internal/dashboard -run 'Settings|WorktreePrefs|SecretsCanary|Classified'`.

Recordatorios de CI: canario de secretos (`secrets_canary_test.go`); trinquete de código muerto (`settings.Read` y `settings.Write` quedan con consumidor de producción en esta unidad); fixtures con `RF-`, no `REQ-`; las pruebas con `newPrefsServer` pueden usar `t.Parallel`, pero nunca con `freePort` ni con el hub compartido.

## Fase 5: Preferencias personales

- [x] 5.1 Escribir antes las pruebas (RED) en `internal/dashboard/settings_api_test.go`: `GET /api/settings/personal` devuelve `{autofetch, path}`; `PUT {"autofetch": true|false}` actualiza `workspaces.json` y no toca el árbol del proyecto; 400 con JSON inválido o valor no booleano; 403 con host ajeno; 404 con el mensaje de `projectNotRegisteredError`; 405 en otros métodos.
- [x] 5.2 Extraer a `Service.hub() (*hub.Manager, error)` la inicialización perezosa de `activeProjectRecord` en `internal/dashboard/worktree_prefs.go`, de modo que la comparten worktree-prefs y las rutas nuevas, sin cambiar el comportamiento previo.
- [x] 5.3 Añadir en `internal/dashboard/settings_api.go` `handleSettingsPersonal`, `Service.GetPersonalSettings` (`LookupAutofetch(root)`) y `Service.SetPersonalAutofetch` (`SetAutofetch(root, v)`, 404 con `errors.Is(err, hub.ErrProjectNotRegistered)`), con `PersonalSettingsDTO` en `types.go` y el registro de `/api/settings/personal` en `server.go`.
- [x] 5.4 Añadir `/api/settings/personal` a `coveredRoutes` de `internal/dashboard/secrets_canary_test.go` (solo el GET). Evidencia: commit aae568291.

Cubre: RF-WEB-20, RF-WEB-10

Verificación: `go test ./internal/dashboard -run 'SettingsPersonal|WorktreePrefs|SecretsCanary'`.

Recordatorios de CI: canario de secretos y literal de cadena de la ruta en `server.go`.

## Fase 6: Pantalla JS

- [x] 6.1 Escribir antes las pruebas (RED) de ejecución en `internal/dashboard/assets_settings_test.go` con `runNodeModules(t, []string{"api","icons","components","settings"}, …)` (se omite sin Node): `controlHtml` con vocabulario cerrado como `select` con exactamente `allowed`; escapado de `<script>`; `collectChanges` envía solo lo cambiado y `null` al vaciar; `editorHref` con rutas de Windows y POSIX (`vscode://file/C:/repos/x/axiom.yaml`); `errorView` con `revision_mismatch` ofrece recargar.
- [x] 6.2 Crear `internal/dashboard/assets/js/core/settings.js` con `groupBySection`, `controlHtml`, `readOnlyHtml`, `collectChanges`, `editorHref` y `errorView`, funciones puras que escapan todo texto con `escapeHtml`.
- [x] 6.3 Crear `internal/dashboard/assets/js/screens/ajustes.js`: carga de `GET /api/settings`, `/api/settings/personal` y worktree-prefs; guardado por lote sin perder lo tecleado al fallar (409 con «Recargar», 422 con violaciones); enlace «Abrir en el editor» y botón «Copiar ruta» (aviso «Ruta copiada al portapapeles» con `copyText`); bloque «Preferencias personales» con autofetch y el modo y la raíz de worktree trasladados desde `screens/ecosystem.js` con los mismos IDs de DOM.
- [x] 6.4 Cablear la ruta: `'ajustes'` en `SIMPLE_ROUTES` y comentario de `internal/dashboard/assets/js/core/router.js`; `VIEW_BY_ROUTE.ajustes`, `initSettings()` y `loadSettings()` en `loadAllData` de `internal/dashboard/assets/js/main.js`; `ROUTE_LABELS.ajustes = 'Ajustes del proyecto'` en `internal/dashboard/assets/js/shell.js`; icono `settings` (SVG en línea) en `internal/dashboard/assets/js/core/icons.js`.
- [x] 6.5 Editar `internal/dashboard/assets/index.html`: entrada de navegación en «Plataforma», `<section id="view-ajustes">` con el aviso «Editar solo afecta a cambios nuevos», contenedor del catálogo y tarjeta de preferencias personales (la de worktree sale de `view-ecosistema`); y las clases mínimas que falten en `internal/dashboard/assets/style.css`.
- [x] 6.6 Pruebas de estructura en `internal/dashboard/assets_settings_test.go`: ruta en `SIMPLE_ROUTES`, `VIEW_BY_ROUTE` y `ROUTE_LABELS`; entrada de navegación; aviso; editor de worktree dentro de `view-ajustes`; ningún control de RDD ni de `integrations` (`index.html` sin `integration_branch`). Actualizar el mapa de marcadores de `internal/dashboard/server_test.go` (`:445-447`) de `/js/screens/ecosystem.js` a `/js/screens/ajustes.js` y las referencias de comentario (`:418`, `:440`, `:607`) a RF-WEB-10.

Cubre: RF-WEB-07, RF-WEB-18, RF-WEB-10

Verificación: `go test ./internal/dashboard -run 'SettingsAssets|SettingsRuntime|Assets|Server.*Marker'`.

Recordatorios de CI: cadenas JS comprobadas en `server_test.go:447` y `assets_branch_test.go`; el `sw.js` no cambia.

## Fase 7: Ecosistema y selector de proyecto

- [ ] 7.1 Reescribir en `internal/dashboard/assets_branch_test.go` `TestBranchPolicyIsShownReadOnlyInEcosystem` (Ecosistema enlaza a `#/ajustes` sin pintar la política), borrar `TestBranchPolicyRuntime`, renombrar `TestWorktreePrefsEditorInEcosystem` a «InSettings» (`view-ajustes`, `screens/ajustes.js`) y actualizar el comentario de cabecera (RED).
- [ ] 7.2 Retirar `branchPolicyRows`, `repoPolicyHtml` y el import de `tableWrap` de `internal/dashboard/assets/js/core/branches.js` (la cabecera pasa a cuatro funciones); retirar su uso y las funciones y el enlace del formulario de worktree en `internal/dashboard/assets/js/screens/ecosystem.js`; la tarjeta Topología enlaza a `#/ajustes`.
- [ ] 7.3 Añadir el enlace «Ajustes del proyecto» al popover de proyecto de `internal/dashboard/assets/index.html`, y confirmar que `GET /api/workspace` conserva `branchPolicy` y `repositories` (`TestWorkspaceIsReadOnly` en `internal/dashboard/server_test.go` queda sin cambios).

Cubre: RF-WEB-21, RF-WEB-01

Verificación: `go test ./internal/dashboard -run 'BranchPolicy|Ecosystem|WorktreePrefs|WorkspaceIsReadOnly|Assets'`.

Recordatorio de CI: cadenas JS de `server_test.go:447` y `assets_branch_test.go`; el trinquete de código muerto no se ve afectado, ya que `WorkspaceDTO` y `branchPolicyView` no cambian.

## Fase 8: Documentación

- [ ] 8.1 `docs/usage.md`: subsección «Ajustes del proyecto desde el dashboard» con claves editables, claves de solo lectura, «solo afecta a cambios nuevos», endpoints con sus códigos, petición local y autofetch.
- [ ] 8.2 `docs/codebase/dashboard.md`: invariante de que `axiom.yaml` solo se escribe por `/api/settings` mediante `internal/settings` y de que toda escritura exige `isLocalRequest`.
- [ ] 8.3 `axiom.example.yaml`: nota en la cabecera (solo comentario) sobre «Ajustes del proyecto» en `axiom ui`; comprobar que la ida y vuelta de la tarea 3.1 sigue idéntica con ese comentario.

Cubre: RF-WEB-10, RF-WEB-18, RF-TOP-21, RF-TOP-22

Verificación: `go test ./internal/settings -run RoundTrip` y `go test ./internal/dashboard ./internal/idmigration -run 'Docs|Residue|Settings'`.

Recordatorio de CI: fixtures y documentación sin `REQ-` nuevos (`internal/idmigration/residues.yaml`).

## Fase 9: Cierre de verificación

- [ ] 9.1 Ejecutar los comandos localizados agregados, sin la suite completa: `go test ./internal/settings`; `go test ./internal/workspace ./internal/multirole ./internal/hub -run 'Promotion|WorkflowLane|ExplainerHTML|GatePolic|PolicyViolations'`; `go test ./internal/dashboard -run 'Settings|WorktreePrefs|BranchPolicy|Ecosystem|SecretsCanary|Classified|WorkspaceIsReadOnly|Assets'`. Registrar el resultado observado de cada uno.
- [ ] 9.2 Comprobar manualmente, sin ejecutar CI completa: ningún `REQ-` nuevo fuera de `residues.yaml`, ninguna exportación sin consumidor de producción (trinquete de código muerto), `kickoff.yaml`/`gates.yaml` sin tocar y sin binarios sueltos compilados.
- [ ] 9.3 Confirmar la cobertura de requisitos: RF-TOP-21, RF-TOP-22, RF-TOP-23, RF-WEB-01, RF-WEB-07, RF-WEB-10, RF-WEB-18, RF-WEB-19, RF-WEB-20 y RF-WEB-21 aparecen en alguna línea `Cubre:` y en alguna prueba que pasa.

Cubre: RF-TOP-21, RF-TOP-22, RF-TOP-23, RF-WEB-01, RF-WEB-07, RF-WEB-10, RF-WEB-18, RF-WEB-19, RF-WEB-20, RF-WEB-21
