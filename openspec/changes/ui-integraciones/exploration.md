## Exploración: ui-integraciones (C-07-h, UI de integraciones en el dashboard)

### Estado actual

**Servidor (`internal/dashboard/server.go`)**
- `withHeaders` (`:219-234`) fija `Access-Control-Allow-Origin: *`, `Allow-Methods`, `Allow-Headers` y `Expose-Headers`, y responde 200 a todo `OPTIONS`. No hay CSP general. La SPA es del mismo origen, así que ninguna cabecera CORS hace falta; ninguna prueba comprueba `Access-Control` (grep sobre `*_test.go`).
- CSP del explicador: `explainerCSP` (`:1031`) se fija dentro de `handleExplainer` (`:1045`), no en `withHeaders`. Si `withHeaders` la fija antes de `next.ServeHTTP`, el `Set` del handler la pisa sin duplicar.
- Escucha solo en `127.0.0.1` (`:150`). `ReadTimeout` y `WriteTimeout` valen 15 s (`:176-180`).
- `isLocalRequest` (`:969-990`) exige `Host` 127.0.0.1 o localhost en el puerto enlazado y, si hay `Origin`, que sea `http://` y local. Un `Origin` ausente se acepta. `rejectNonLocal` (`settings_api.go:22`) devuelve 403 JSON. Se usa en las escrituras de ajustes, worktree-prefs, marcas, actualización, etc.
- Los GET no pasan por `isLocalRequest`. Quitar ACAO no impide la lectura por DNS rebinding: el origen atacante es el suyo y lee sus propios GET, así que haría falta validar `Host` también en los GET de las rutas nuevas.
- Rutas registradas en `server.go:75-123`; `/` sirve `AssetsFS` embebido con `http.FileServer` (`:126-129`).

**Assets frente a una CSP**
- Un único `<script>` inline: el anti-parpadeo de tema (`index.html:9-23`). Lo fija `assets_test.go:227-236` leyendo `/` (busca `axiom.theme`, `matchMedia(...)`, `dataset.theme`, `try {`, y que esté antes de `tokens.css`). Todo lo demás es `<script type="module" src="/js/main.js">` (`index.html:783`).
- No hay `onclick=` ni handlers inline: la UI delega eventos (`ajustes.js:226-239`). No hay CDN ni URL externa. Las fuentes son `/fonts/*.woff2` y `@font-face` con `url("/fonts/...")` en `style.css:16-30`. El service worker (`sw.js`) no cachea nada ni intercepta peticiones.
- Estilos inline que una CSP sin `'unsafe-inline'` en `style-src` rompería:
  - estáticos: `index.html:97, 620, 621, 628`;
  - dinámicos por plantilla `innerHTML`: `core/increment.js:90` (`style="width:${v}%"`), `screens/incrementos.js:173` (`--ix-cols`), `projects.js:45, 211`;
  - `core/icons.js:70` hace `setAttribute('style', ...)`, que también bloquea la CSP. Las asignaciones CSSOM (`components.js:96-97`, `ta.style.x`) sí se permiten.
- `markdown.js:39` y `update.js:132` generan enlaces `target="_blank" rel="noopener noreferrer"`. Un enlace «Abrir en Azure DevOps» debe seguir ese patrón con URL validada (http/https).
- El iframe del explicador usa `sandbox=""` y `src=/api/explainer` (`detalle-explicacion.js:11`): un `default-src 'self'` lo cubre, porque `frame-src` hereda de `default-src`.

**Ajustes (`internal/settings`)**
- `catalog.go:101-121`: tabla `editableSpecs` de `keySpec`, con tres ámbitos (`scopeBlock`, `scopeRepo`, `scopeRole`). Ya incluye `git.branch_template`, `repositories[].branch_template` y `roles[].branch_template`, así que C-07-f ya está registrado y la dependencia blanda queda resuelta.
- `integrations` está excluido a propósito, en cuatro sitios:
  - `document.go:101` (`unsupportedConstruct` no recorre el bloque, ni anclas ni alias);
  - `patch.go:76` (`CodeUnknownKey`);
  - las pruebas `catalog_test.go:202-219`, `document_test.go:118-128`, `patch_test.go:170`, `write_test.go:265-266`;
  - la spec viva: RF-TOP-21 (`workspace-topology/spec.md:865`), RF-TOP-22 (`:1027`), RF-WEB-18 (`local-web-dashboard/spec.md:604-607`) y RF-WEB-19 (`:629`, `:643`, `:689`).
- `Write` (`write.go:43-123`) parchea el árbol `yaml.Node`, restaura el formato y revalida con `workspace.ParseConfig` más `engineErrors`. Usa revisión SHA-256 y reemplazo atómico, y solo bloquea los errores nuevos. `ParseConfig` ya valida `integrations:` con decodificador estricto: tipos, enum, formato y claves obligatorias (`workspace/integrations.go:52-157`).
- `applyPatch` asume `bloque.campo` con `strings.Cut(key, ".")` (`patch.go:185`), y los ids de plugin llevan guion (`azure-devops`). Hace falta un cuarto ámbito de elemento, `integrations[<plugin>].<clave>`, como `repositories[...]` y `roles[...]`. `canonicalBlocks` (`patch.go:22`) no lista `integrations`.
- API de ajustes: `GET|PUT /api/settings` (`settings_api.go:59-87`) con `settingsMu`, `rejectNonLocal` y cuerpo acotado a 1 MiB. Frontend: `screens/ajustes.js` más `core/settings.js` (`collectChanges`, `controlHtml`, `groupBySection`).

**Marco de integraciones (`internal/integrations`)**
- Manifiesto (`manifest/manifest.go`): `Config []ConfigField` (clave, tipo string/bool/int/string_list, requerido, default, enum, formato, min/max), `Secrets []SecretSpec`, `Events`, `Capabilities`, `Identity`, `RefAlias`. El de ADO (`manifest/azuredevops.go:15-60`) declara organization, project, team, item_keys, task_state_*, work_item_state_archive, req_tag_prefix, req_tag_max, workday_hours y workdays; secreto `pat`; identidad `plugin`.
- Un campo de configuración con nombre de secreto se rechaza (`looksLikeSecret`, `manifest.go:245`), y `decodeIntegrations` trata una clave de secreto como desconocida. El PAT no tiene ruta legal a `axiom.yaml`.
- Secretos (`integrations/secrets`): `Default()` encadena `Env{}` y el keyring de plataforma (`default.go:15-23`). Solo hay almacén propio en Windows y macOS: en Linux `ErrNoWritableStore` (`store.go:37`). `Secret` redacta en toda forma de impresión o serialización (`secret.go`). `Get` devuelve la fuente: `env`, `keyring`, `missing` o `keyring_unavailable`. La variable de entorno prevalece sobre el keyring, así que `Set` puede quedar oculto.
- Verbos CLI: `integrations secret set|status|delete` (`cli/integrations.go:570-667`), `connect` y `discover` (`cli/integrations_ports.go:161-198`). `connectIntegration`, `discoverIntegration` y `resolvePortTarget` no se exportan. `connect` persiste la identidad en `~/.axiom/integration-identities.json` (`identitystore`) y puede devolver `IdentitySelectionError` con candidatos.
- `integrations.yaml` v2 (`links.go`): `LinksDoc{Links, TimeEntries}`. Lectores exportados: `WorkItemLinks`, `TaskLinks` (con `Role`, `EstimateHours`, `StartedAt`, `URL`) y `TimeEntries` (con `Actor`). `PendingTimeFor` lee `.time-pending.yaml`. La outbox (`outbox.go`, `Operation{Key, Plugin, State, Attempts, LastError, Reason, Event}`) solo se lee con `readOutbox`, que no está exportado: falta un lector público de solo lectura. `Folder{WorkspaceRoot, Root}` identifica la carpeta del cambio, también la archivada.
- Importaciones (`imports_test.go`): `manifest` solo usa la biblioteca estándar; `workspace` solo importa `integrations/manifest`; `secrets` no importa `integrations` ni `workspace`; solo `internal/integrations/secrets` importa go-keyring; solo `builtin` y `adotest` importan `azuredevops`. El dashboard no está restringido: ya importa `internal/cli` y `integrations/secrets` (este último en `secrets_canary_test.go`).

**C-07-d y C-07-a**
- `axiom change start --from` (`cli/change_start.go`, `internal/changestart`): fase 1 (`Start`), fase 2 (`--adopt`) y `--discard`. `workItemTracker` y `resolveWorkItemTracker` están en `cli`, sin exportar. La salida JSON trae `external_ref` y `work_item.url`. `LinkWorkItem` escribe el vínculo `work_item` con `URL` en `integrations.yaml`.
- El plugin construye la URL del work item en `azuredevops/workitem_port.go:147` (`.../{project}/_workitems/edit/{id}`). Para «Abrir en Azure DevOps» sirve `Link.URL` de `WorkItemLinks`, sin reconstruirla.
- El dashboard crea cambios con `Service.CreateIncrement` (`service.go:1073`) y `ensureIncrementBranch` (`service_branch.go:33`). El flujo con work item exige fase 1, luego `ensure` con el nombre derivado, `seal`, fase 2 y después los artefactos. Hoy el dashboard no lo hace.
- Actores: `GateRecord.Actor` en `gates.yaml` (`kickoff/types.go:259`), leído con `kickoff.LoadGates` (ya lo usa `service.go:421`), `Kickoff.SealedBy` (`:206`) vía `kickoff.Load`, y `TimeEntry.Actor` en `integrations.yaml`. No existe un registro de «quién inició o cerró un rol» aparte de la compuerta `role-apply` y de la imputación `role.close/`.
- Detalle del cambio: `GetIncrementDetail` (`service.go:547-600`) rellena `IncrementDetailDTO`; las pestañas son paneles `inc-panel-*` (`index.html:245-246`) más un módulo `detalle-*.js` (`detalle.js:13-14, 207-209`), y `INCREMENT_TABS` en `router.js:13`.

**Barandillas de pruebas**
- Canario de secretos (`dashboard/secrets_canary_test.go`): toda ruta nueva de `server.go` debe estar en `coveredRoutes` (GET recorrido con el centinela) o en `excludedRoutes` con motivo. `TestEveryDashboardRouteIsClassifiedForTheSecretsCanary` falla si no.
- `TestImportsDirection` y `TestAdapterAssemblyImportRules` (`integrations/imports_test.go`): no impiden que dashboard o settings importen `manifest` o `secrets`.
- `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` (`cli/refusal_resolution_ratchet_test.go`) escanea `internal/cli`, `fileprim` y `sddstatus`: cualquier rechazo nuevo en `cli` nombra su resolución o lleva `// refusal:by-design <forma>: <motivo>`. El dashboard no está en el escaneo.
- Trinquete de código muerto (`scripts/deadcode-ratchet.sh`, CI `deadcode-ratchet`): todo exportado nuevo debe alcanzarse desde `main`.
- Pruebas de assets que fijan la estructura: `assets_settings_test.go`, `assets_branch_test.go` (formulario de Ajustes y entrada de navegación), `assets_test.go` (anti-parpadeo), `server_test.go:444-458` (módulos servidos como `text/javascript`).
- Provisionales: `prov:WEB:<n>` (dominio `local-web-dashboard`), con el último RF-WEB vigente el 26 (`RF-WEB-26`, canario de marcas). Requisitos existentes a MODIFICAR: RF-WEB-01 (escrituras de `axiom.yaml` solo por `PUT /api/settings`), RF-WEB-07 (navegación), RF-WEB-10, RF-WEB-18 (escenario «Ningún control de ... `integrations.*`»), RF-WEB-19 (la respuesta no contiene `integrations`; `unknown_key` para `integrations.*`), RF-WEB-26; y en `workspace-topology`: RF-TOP-21 y RF-TOP-22. Relacionados sin cambio: RF-INT-13/15/16 (almacén y GET), RF-INT-23 (puertos), RF-ADO-10 a 13 y RF-ADO-15 (PAT sin salida).

### Zonas afectadas
- `internal/dashboard/server.go`: `withHeaders` (CSP y sin ACAO), rutas nuevas, Host y Origin en las rutas de integraciones.
- Ficheros nuevos de dashboard (p. ej. `integrations_api.go`, `service_integrations.go`, `types.go`): DTOs, servicio y puerto inyectable.
- `internal/settings/{catalog,document,patch}.go`: ámbito `integrations[<plugin>].<clave>` derivado del manifiesto y retirada de la exclusión.
- `internal/cli`: exportar `connectIntegration`, `discoverIntegration` y el tracker de work items, o moverlos a un paquete que el dashboard pueda importar sin ciclos.
- `internal/integrations`: lector público de la outbox, de solo lectura.
- `assets/index.html` (script de tema, vista nueva, navegación), `assets/js/screens/integraciones.js` y `core/integrations.js` (nuevos), `detalle.js`, `detalle-integraciones.js` (nuevo), `router.js`, `main.js`, `shell.js`, `style.css`.
- Specs: `local-web-dashboard`, `workspace-topology`.
- Pruebas que cambian de sentido: `catalog_test.go`, `document_test.go`, `patch_test.go`, `write_test.go`, `assets_test.go`, `secrets_canary_test.go` (clasificar rutas nuevas).

### Enfoques

1. **Seguridad primero, luego catálogo ampliado y pantalla generada del manifiesto (recomendado)**
   - Commit 1: CSP general en `withHeaders` y sin `Access-Control-*`. `default-src 'self'`; `script-src 'self'`; `style-src 'self' 'unsafe-inline'`; `img-src 'self' data:`; `font-src 'self'`; `connect-src 'self'`; `object-src 'none'`; `base-uri 'none'`; `form-action 'self'`; `frame-ancestors 'none'`. Se mueve el script de tema a `/js/theme-init.js`, síncrono en `<head>`, y se ajusta `assets_test.go`.
   - Además: la CSP de `/api/explainer` queda en su handler; `OPTIONS` responde 405 o 204 sin cabeceras CORS; y pruebas de Host y Origin ajenos (DNS rebinding) sobre las rutas nuevas.
   - Después: nuevo ámbito en el catálogo (`integrations[<plugin>].<clave>` y `enabled`), generado de `manifest.Lookup`. Tipos: string, bool, int y string_list → `TypeString`, `TypeBool`, `TypeInt`, `TypeStringList`; con enum desde `ConfigField.Enum` y validación por el decodificador estricto (`engineChecked`).
   - Rutas nuevas: `GET /api/integrations` (manifiesto, valores, estado del secreto sin valor, identidad y descubrimiento bajo demanda); `POST /api/integrations/secret` (PAT por JSON, solo `Content-Type: application/json`, 4 KiB máx., `rejectNonLocal`, respuesta `{source}`); `DELETE`; `POST /api/integrations/connect` y `.../discover`; `GET` de vista por cambio.
   - Pros: cumple el criterio de terminado y las barandillas. El PAT cruza solo el POST y se guarda por `secrets.Default().Set`, que no lo devuelve.
   - Contras: toca la spec de topología (RF-TOP-21/22) y las pruebas que prohibían `integrations`.
   - Esfuerzo: alto, dentro de las 1.000-1.400 líneas previstas.

2. **Pantalla propia que escribe `integrations` con un escritor aparte (sin tocar el catálogo de Ajustes)**
   - Pros: no cambia RF-TOP-21/22 ni sus pruebas.
   - Contras: duplica el round-trip, la revisión y la validación; contradice la decisión del plan (escribir por el catálogo de Ajustes y `settings.Write`); crea dos puertas de escritura de `axiom.yaml`.
   - Esfuerzo: medio-alto, con más riesgo de deriva.

3. **CSP con hash del script inline y `'unsafe-inline'` también en scripts de tema**
   - Variante del commit 1: mantener el `<script>` inline con `script-src 'sha256-...'` calculado al arrancar a partir de `index.html`.
   - Pros: no mueve el script ni cambia `assets_test.go`.
   - Contras: frágil ante cualquier retoque del script; el hash se calcula en tiempo de ejecución. Se prefiere mover el script a un fichero propio.
   - Esfuerzo: bajo.

### Recomendación
Enfoque 1, en este orden de commits: (1) CSP y retirada de CORS con pruebas, el script de tema fuera de línea y `style-src 'unsafe-inline'` documentado como deuda acotada (7 sitios); (2) catálogo por manifiesto, MODIFIED de RF-TOP-21/22, de RF-WEB-18/19 y de las pruebas de exclusión; (3) API (secreto, conectar, descubrir, estado) con canario y pruebas de Host y Origin; (4) pantalla «Integraciones»; (5) vista por cambio en `detalle.js` con lector público de outbox; (6) campo de ID o URL del work item y enlace «Abrir en Azure DevOps». La API del PAT usa un POST que nunca devuelve el valor, y sus respuestas solo llevan `source` (`env`, `keyring`, `missing`, `keyring_unavailable`).

### Riesgos
- **CSP frente a estilos inline:** 4 estáticos y ~4 dinámicos más `setAttribute('style')`. Sin `'unsafe-inline'` en `style-src` la UI se rompe; con él, el riesgo residual es inyección de CSS. Una refactorización a clases y CSSOM es opcional y ampliaría el alcance.
- **Dirección de importaciones:** `settings` solo debe importar `manifest` (hoja), no `integrations`; el dashboard puede importar `secrets` y `integrations`. El adaptador ADO solo se alcanza por el registro (`builtin`) y la UI debe tratar el plugin de forma genérica (por manifiesto), no por `azuredevops`.
- **Dónde vive el PAT:** `secrets.Default()` (keyring en Windows y macOS; en Linux, variable de entorno). `Set` puede quedar tapado por una variable de entorno ya definida; la UI debe mostrar `source`. En Linux, «Conectar» no puede guardar: la UI explica `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`.
- **`WriteTimeout` de 15 s** frente a un timeout de 30 s por intento de ADO y reintentos acotados: `connect` y `discover` pueden cortarse. Hace falta límite por handler o respuesta asíncrona.
- **Editar `integrations` en `axiom.yaml`:** quitar la exclusión obliga a decidir qué hacer con anclas o alias dentro del bloque (hoy se ignoran). `canonicalBlocks` no lo lista (se inserta tras `workspace`).
- **DNS rebinding en GET:** quitar ACAO no la cubre. Exigir `Host` local en todas las rutas `/api/integrations*`, no solo en las que mutan.
- **Estado sin lector público:** la outbox no tiene lector exportado; añadirlo sin sobreexponer escritura, y que el trinquete de código muerto lo vea usado.
- **Exportar desde `cli`:** cada rechazo nuevo en `cli` debe nombrar su resolución o llevar `refusal:by-design`.
- **Dependencias del plan:** C-07-e y C-07-f ya están en `main` (`integrations.yaml` v2, `branch_template` en el catálogo). C-07-d y C-07-a también.

### Decisiones abiertas de producto
1. «Probar conexión»: `connect` guarda la identidad en `~/.axiom/integration-identities.json` como efecto lateral. ¿La prueba debe persistir o solo resolver y mostrar? Propuesta por defecto: «Conectar» guarda (y pide elegir usuario si hay `selection_required`); «Probar» resuelve sin guardar.
2. Campo de ID o URL al arrancar: ¿se integra en el modal de creación de cambios (`CreateIncrement`, con fase 1, `ensure`, `seal` y fase 2) o es una acción previa que solo importa el work item? Propuesta: integrarlo en la creación, solo con plugin activo, y usar el nombre derivado.
3. Plantillas de rama en «Integraciones»: ya viven en Ajustes (`git.branch_template`, por repositorio y por rol). Propuesta: enlazarlas desde Integraciones en lugar de duplicar controles.

### Listo para propuesta
Sí. Faltan solo las tres decisiones de arriba, que no bloquean `sdd-propose`. Hay que fijar en el spec los MODIFIED de RF-WEB-01, 07, 10, 18, 19 y 26 y de RF-TOP-21 y 22, y los provisionales `prov:WEB:<n>` para los requisitos nuevos.
