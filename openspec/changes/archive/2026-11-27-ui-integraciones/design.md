# Diseño: UI de integraciones en el dashboard (C-07-h)

## Enfoque técnico

El cambio sigue el enfoque 1 de la propuesta: seguridad primero y después el catálogo ampliado, la API, la pantalla generada desde el manifiesto, la pestaña por cambio y el alta desde un work item. Cada unidad es un commit con sus pruebas y su documentación, dentro de una única PR (`single-pr`).

Ideas que vertebran el diseño:

1. **Una sola barrera de red, en el envoltorio del servidor.** `withHeaders` (`internal/dashboard/server.go:219`) pasa a ser la cadena de seguridad de todas las respuestas: CSP general, cabeceras complementarias, retirada de CORS, `405` para `OPTIONS` y la comprobación de `Host` y `Origin` locales en toda petición que muta y en toda ruta `/api/integrations*` (prov:WEB:1, prov:WEB:2). Las comprobaciones que ya hacen algunos handlers (`rejectNonLocal`) se conservan como defensa en profundidad.
2. **Una sola puerta de escritura de `axiom.yaml`.** El bloque `integrations` entra en el catálogo de `internal/settings` como un cuarto ámbito, `integrations[<plugin>].<clave>`, derivado en tiempo de ejecución de `manifest.Catalog()`; se escribe solo por `PUT /api/settings` y `settings.Write` (RF-TOP-21, RF-TOP-22, RF-WEB-10, RF-WEB-19).
3. **El dashboard no reimplementa la lógica de la CLI.** `connect`, `discover` y las fases de `change start` se exponen con un puerto inyectable, `dashboard.IntegrationsGateway`, que en producción implementa `cli.IntegrationsGateway` y se inyecta desde `cmd/axiom/main.go`, igual que `MarksPublisher` (`internal/dashboard/marks_port.go`). El secreto (guardar, eliminar y procedencia) se gestiona directamente con `secrets.Default()`, que el dashboard puede importar.
4. **El secreto solo entra.** El PAT cruza una única petición `POST` y ninguna respuesta, DTO, cabecera ni log lo contiene; las respuestas solo llevan su procedencia (`source`).
5. **Lectura de solo lectura del estado de integración.** La pestaña del cambio lee vínculos, imputaciones, tiempo pendiente, outbox y actores con lectores públicos de `internal/integrations` y `internal/kickoff`; la outbox se expone con un lector nuevo que ya devuelve una vista redactada.

## Decisiones de arquitectura

### D-01: `style-src 'self'` estricta, refactorizando los estilos en línea

**Elección**: la CSP general fija `style-src 'self'` sin `'unsafe-inline'`. Se retiran los ocho usos de estilo en línea inventariados y el script de tema pasa a `/js/theme-init.js` (prov:WEB:1).

| Sitio | Cambio |
|---|---|
| `index.html:97` (`<i data-icon="download" style="color: var(--warn)">`) | clase `icon-warn` en `style.css` |
| `index.html:620`, `:628` (`border-top` y `padding-top`) | clase `form-stack--divided` |
| `index.html:621` (`justify-content: space-between`) | clase `field-row--between` |
| `core/icons.js:70` (`svg.setAttribute('style', …)`) | se elimina la copia del atributo `style`; ya se copian las clases |
| `core/increment.js:90` (`<span style="width:${v}%">`) | `<span data-pct="${v}">`; el ancho lo fija el módulo nuevo `core/cssvars.js` con CSSOM |
| `screens/incrementos.js:173` (`style="--ix-cols:${template}"`) | `data-ix-cols="${template}"`; el módulo fija la variable con `style.setProperty` |
| `projects.js:45` | clase `popover-label--plain` |
| `projects.js:211` | clase `error-banner--inset` |

`core/cssvars.js` exporta `applyCssData(root)`, que recorre `[data-pct]` y `[data-ix-cols]` y asigna `el.style.width` y `el.style.setProperty('--ix-cols', …)`, y `observeCssData(root)`, que instala un `MutationObserver` sobre `document.body` desde `main.js` para aplicarlo a todo nodo insertado por `innerHTML`. La CSP no bloquea las asignaciones CSSOM; sí bloquea el atributo `style` en HTML analizado y `setAttribute('style', …)`. El valor de `data-pct` se acota a 0-100 y el de `data-ix-cols` se acepta solo si cumple `^[0-9a-z().,% -]+$`, para que un atributo manipulado no inyecte CSS.

**Alternativas consideradas**: `'unsafe-inline'` en `style-src` (deja abierta la inyección de CSS, y la refactorización es acotada: ocho sitios, un módulo de unas 30 líneas y seis clases); barras de progreso con clases discretas por tramos (pierde precisión y añade 21 clases); `<progress>` nativo (cambia el aspecto y obliga a rehacer los estilos del tablero).

**Razón**: el inventario es cerrado y pequeño, y la exploración no encontró otros orígenes de estilo en línea (`markdown.js` no emite `style`). Una prueba Go que recorre los assets impide que vuelva a aparecer (`TestNoInlineStyleOrScriptInAssets`).

### D-02: Cadena de seguridad en `withHeaders` y comprobación local central

**Elección**: `withHeaders` aplica, en este orden y para toda petición:

1. Cabeceras: `Content-Security-Policy` general (literal `dashboardCSP`), `X-Content-Type-Options: nosniff` y `Referrer-Policy: no-referrer`; en rutas `/api/integrations*`, además `Cache-Control: no-store`. Se fijan **antes** de `next.ServeHTTP`, así que también las llevan los `403`, `404`, `405`, `500`, `304` y `HEAD` (Go conserva las cabeceras ya fijadas en `http.Error` y en los errores de `http.FileServer`).
2. Comprobación local: si la ruta es `/api/integrations` o empieza por `/api/integrations/`, con cualquier método, o si el método no es `GET`, `HEAD` ni `OPTIONS`, y `isLocalRequest` falla, responde `403` JSON `{"code":"forbidden_origin","error":"…"}` sin leer el cuerpo.
3. `OPTIONS`: responde `405 Method Not Allowed` sin ninguna cabecera `Access-Control-*`.

Se retiran las cuatro cabeceras `Access-Control-*` (prov:WEB:1). `handleExplainer` sigue fijando su CSP con `Set`, que sustituye la general y deja una sola cabecera (RF-WEB-15 sin cambios).

`isLocalRequest` (`server.go:969`) se endurece con una lista cerrada:

| Elemento | Regla |
|---|---|
| `Host` | `127.0.0.1`, `localhost` o `[::1]`, con el puerto enlazado (`s.port`); sin servidor escuchando (`s.port == 0`, pruebas) se acepta cualquier puerto, solo con esos nombres. `Host` vacío se rechaza; `Host` duplicado ya lo rechaza `net/http` con `400`. |
| `Origin` ausente | se acepta (curl, pruebas y navegación del mismo origen). |
| `Origin` presente | exactamente `http://<host local>:<puerto>` con la misma lista; se rechazan `https://…`, `null`, rutas, credenciales y cualquier nombre DNS aunque resuelva a bucle local (defensa frente a *DNS rebinding*). |

**Alternativas consideradas**: comprobar en cada handler nuevo (deja sin cubrir las rutas que ya mutan sin comprobación, como `/api/projects/add`, `/api/agents/install` o `/api/skills/approve`, que hoy no llaman a `isLocalRequest`); `204` para `OPTIONS` (indistinguible de un éxito para un cliente que no lee cabeceras); pasar `OPTIONS` a cada handler (algunos no distinguen métodos y lo tratarían como `POST`).

**Razón**: un único punto cubre todas las rutas que mutan, también las existentes (prov:WEB:2 dice «toda ruta que muta»), y las cabeceras quedan en todas las respuestas, incluidas las de error. El `405` sin `Allow` sigue el patrón de los `405` actuales del servidor (`http.Error` sin `Allow`).

**Consecuencia para las pruebas**: `Router()` devuelve el `ServeMux` sin envoltorio y lo usan decenas de pruebas con `Host: example.com`. Se conserva tal cual; se añade el método no exportado `handler()`, que devuelve `s.withHeaders(s.mux)`, lo usan `serve()` y las pruebas nuevas de seguridad y del canario, que fijan `Host: 127.0.0.1`.

### D-03: Ámbito `integrations[<plugin>].<clave>` derivado del manifiesto

**Elección**: `internal/settings` añade `scopePlugin` y genera las filas del catálogo al vuelo desde `manifest.Catalog()` (no hay tabla estática):

- Por cada manifiesto: `integrations[<id>].enabled` (`TypeBool`) y una fila por cada `ConfigField`, con este mapeo de tipos: `string` → `TypeString` (o `TypeEnum` si declara `Enum`), `bool` → `TypeBool`, `int` → `TypeInt`, `string_list` → `TypeStringList` (con `Allowed` = `Enum` si lo declara).
- `Entry` gana `Plugin string`, `Format string`, `Min *int` y `Max *int`, y `SettingDTO` los publica (`plugin`, `format`, `min`, `max`). `Section` vale `integrations`, `Label` es la clave, `Description` sale del manifiesto y `Default` describe el `Default` del campo.
- `Required` de la entrada refleja `ConfigField.Required` para la interfaz, pero el `keySpec` se construye con `required: false` y `engineChecked: true`: la obligatoriedad es condicional a `enabled` y la decide el decodificador estricto al revalidar el documento (`decodeIntegrations`). Así, retirar un plugin entero (todas sus claves a `null`) es posible.
- `parseKey` reconoce el prefijo `integrations[`; el selector va del primer `[` al último `].` y se toma íntegro, guiones incluidos. El identificador real del plugin es `azure-devops` (`manifest.AzureDevOpsID`); los escenarios de la spec que dicen `azuredevops` son ilustrativos y las pruebas usan el identificador real. `resolve` devuelve `unknown_key` si el plugin no está en el catálogo, si el campo no está en su `Config` o si coincide con el nombre de un `SecretSpec` (defensa explícita, además de `looksLikeSecret` del manifiesto).
- `apply` crea `integrations` (se añade al final de `canonicalBlocks`, tras `acceptance_tests`) y la entrada del plugin si faltan; al retirar, borra la clave, la entrada del plugin si queda vacía y el bloque si queda vacío. Un mapeo vacío en estilo de flujo (`{}`) pasa a estilo de bloque al añadirle claves.
- `unsupportedConstruct` (`document.go:96`) deja de saltarse `integrations`: un ancla, alias o `<<` dentro del bloque hace el documento no escribible (RF-TOP-22).
- `entries()` añade las filas de integración tras las editables, en el orden del catálogo de manifiestos y de su `Config`, en todas las topologías.

**Alternativas consideradas**: tabla estática de claves de Azure DevOps (duplica literales del plugin y contradice RF-TOP-21); escritor propio del bloque (segunda puerta de escritura de `axiom.yaml`, descartado en la propuesta).

**Razón**: la única fuente de tipo, enumeración, formato y límites es el manifiesto, que también usa el decodificador; `settings` solo importa `integrations/manifest`, hoja de la biblioteca estándar (`TestImportsDirection` sigue en verde, y `settings` ya lo alcanzaba de forma transitiva por `workspace`).

### D-04: Puerto `IntegrationsGateway` inyectado desde `main`

**Elección**: el dashboard declara el puerto y la CLI lo implementa sobre las funciones existentes, sin moverlas:

```go
// internal/dashboard/integrations_port.go
type IntegrationsGateway interface {
	Connect(ctx context.Context, cwd, pluginID, user string) (cli.IntegrationConnectOutcome, error)
	Discover(ctx context.Context, cwd, pluginID string) (integrations.Discovery, error)
	WorkItemApplicability(cwd string) cli.WorkItemApplicability
	StartWorkItem(ctx context.Context, cwd, ref string) (cli.WorkItemStartOutcome, error)
	WorkItemStaged(cwd, change string) (bool, error)
	AdoptWorkItem(ctx context.Context, cwd, change string) (cli.WorkItemAdoptOutcome, error)
	DiscardWorkItem(cwd, change string) (cli.WorkItemDiscardOutcome, error)
}
func (s *Service) SetIntegrationsGateway(g IntegrationsGateway)
```

`cli.IntegrationsGateway` (fichero nuevo `internal/cli/integrations_gateway.go`) envuelve `connectIntegration`, `discoverIntegration`, `resolveWorkItemTracker` y `changestart.Start|Adopt|Discard`, y traduce los rechazos con el catálogo ya existente (`changeStartRefusalText`) a `*cli.WorkItemRefusal{Code, Message, Continuation, Fields}`, cuyo `Error()` devuelve el campo `Message` (no un literal). `main.go` añade `svc.SetIntegrationsGateway(cli.NewIntegrationsGateway())` junto a `SetMarksPublisher`. Sin puerto, las rutas que lo necesitan responden `503` `integrations_gateway_missing`.

**Alternativas consideradas**: mover `connect`, `discover` y el tracker a un paquete nuevo importable (más superficie y riesgo de ciclos con `cli`, que ya los usa); exportar las funciones sueltas de `cli` (más exportados y acoplamiento directo del dashboard).

**Razón**: el trinquete de código muerto ve alcanzables los métodos porque el tipo se convierte a la interfaz en `main`; las pruebas del dashboard usan un doble del puerto y nunca llaman a Azure DevOps. No hay ciclo: `cli` no importa `dashboard`. El gateway no introduce literales de rechazo nuevos en `internal/cli`; si alguno resultara imprescindible, nombrará su resolución o llevará `// refusal:by-design` (`TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`).

### D-05: Plazo por handler con `http.ResponseController` y cancelación por contexto

**Elección**: las rutas que llaman al tracker fijan un presupuesto propio y amplían su plazo de escritura:

| Ruta | Presupuesto (`ctx`) | Plazo de escritura |
|---|---|---|
| `POST /api/integrations/connect`, `POST /api/integrations/discover` | 45 s | presupuesto + 5 s |
| `POST /api/integrations/start` | 120 s | presupuesto + 5 s |

El handler hace `ctx, cancel := context.WithTimeout(r.Context(), budget)` y `http.NewResponseController(w).SetWriteDeadline(time.Now().Add(budget + 5*time.Second))` (el error `ErrNotSupported` de `httptest.ResponseRecorder` se ignora). El vencimiento responde `504` `integration_timeout` en castellano. Si el cliente cierra la conexión, `r.Context()` se cancela y la llamada al puerto se corta, porque `ResolveIdentity`, `Discover` y `changestart.Start` reciben ese contexto. Un cerrojo `TryLock` por tipo de operación (uno para `connect` y `discover`, otro para `start` y `discard`) responde `409` `integration_busy` a un doble clic o a peticiones solapadas. Los presupuestos son campos del `Server` (`integrationBudget`, `workItemBudget`) para que las pruebas los reduzcan.

**Alternativas consideradas**: respuesta asíncrona con sondeo (añade estado y rutas); subir `WriteTimeout` global (alarga todas las rutas); presupuesto de 12 s bajo el `WriteTimeout` de 15 s (corto para un descubrimiento con reintentos de Azure DevOps).

**Razón**: cumple «un plazo propio inferior a la espera de escritura» (prov:WEB:5, prov:WEB:6) para cada ruta sin cambiar las demás. La limitación de frecuencia queda fuera: servidor local de un solo usuario y el cerrojo ya impide el solapamiento.

### D-06: API `/api/integrations*`

**Elección**: rutas nuevas, todas bajo la comprobación local de D-02 y con errores `{"code","error",…}` en castellano:

| Ruta | Método | Respuesta correcta | Errores |
|---|---|---|---|
| `/api/integrations` | `GET` | `IntegrationsDTO` | `404 config_missing`, `405` |
| `/api/integrations/secret` | `POST` | `{plugin, name, source, envVar, shadowedByEnv}` | `400 invalid_request`, `404 unknown_plugin`, `422 unknown_secret`, `422 no_writable_store` (con `envVar`), `500 secret_store_failed`, `405` |
| `/api/integrations/secret/delete` | `POST` | igual que el guardado | `400` (incluido `confirm` distinto de `true`), `404`, `422`, `500`, `405` |
| `/api/integrations/connect` | `POST` | `{identity, resolvedAt, replaced?}` | `409 identity_selection_required` (con `candidates`), `422 identity_user_unknown` (con `candidates`), `422 secret_missing`, `422 plugin_not_configured`, `422 plugin_disabled`, `422 config_invalid`, `422 plugin_not_compiled`, `404 unknown_plugin`, `502 integration_failed`, `504 integration_timeout`, `409 integration_busy`, `500 identity_store_failed`, `503` |
| `/api/integrations/discover` | `POST` | `integrations.Discovery` | los de `connect` salvo los de identidad |
| `/api/integrations/change` | `GET` | `ChangeIntegrationsDTO` (`?name=<cambio>`) | `400 invalid_request`, `404 change_not_found`, `405` |
| `/api/integrations/start` | `POST` | `WorkItemStartResponse` | ver D-08 |
| `/api/integrations/start/discard` | `POST` | `{change, status, warnings}` | `400` (sin `confirm: true`), `409`, `422`, `503` |

Reglas comunes de las rutas `POST`: `Content-Type` debe ser `application/json` (otro valor, `400 invalid_request`); cuerpo acotado con `http.MaxBytesReader` (8 KiB; 16 KiB en `start`); `DisallowUnknownFields`; cualquier parámetro en la *query* se rechaza con `400` (el PAT nunca viaja en la URL).

Validación del guardado del PAT (prov:WEB:4): `plugin` en `manifest.Catalog()`; `name` declarado en `Secrets` del manifiesto; `value` no vacío ni solo blancos, sin caracteres de control (`< 0x20` y `0x7f`, incluidos saltos de línea) y de como mucho `secrets.MaxValueBytes` (4096, constante nueva que sustituye a `maxSecretBytes` de `cli/integrations.go:36`, de modo que CLI y dashboard comparten el límite). El plugin no necesita estar activo: el PAT es de ámbito de usuario. Tras `Set`, se relee con `Get` para responder la procedencia resultante; si es `env`, `shadowedByEnv: true`. `ErrNoWritableStore` responde `422 no_writable_store` con `envVar` (de `secrets.Ref.EnvVar()`, derivada del manifiesto). Cualquier otro fallo del almacén responde `500 secret_store_failed`; su texto pasa por una redacción que sustituye el valor recibido por `[redactado]` antes de salir.

Eliminación: exige `confirm: true` en el cuerpo (la interfaz pide confirmación explícita y explica que el PAT es del usuario y lo comparten todos sus proyectos); es idempotente (sin PAT guardado, responde `200` con `source: missing` o `env`); un fallo del almacén es `500 secret_store_failed`; en sistemas sin almacén responde `422 no_writable_store` con la guía de la variable.

`connect` y `discover` comprueban antes de llamar al puerto que cada secreto `Required` del manifiesto tiene procedencia `env` o `keyring`; si no, `422 secret_missing` con `envVar` y sin llamar al tracker (prov:WEB:5). Siempre operan con lo **guardado** en `axiom.yaml` (el puerto lee `workspace.Locate`), nunca con lo tecleado sin guardar; la interfaz desactiva «Probar conexión» y «Descubrir» mientras haya cambios sin guardar. Los errores del puerto (`*integrations.PortError`) ya salen saneados de valores de secreto; se responden como `502 integration_failed` con su mensaje.

La elección de identidad se reenvía como `user` y la vuelve a resolver el plugin en el servidor: una elección forjada o caducada no coincide con ningún miembro y vuelve como `IdentitySelectionError` con `User`, que se responde `422 identity_user_unknown` sin guardar nada.

**Alternativas consideradas**: `DELETE /api/integrations/secret` (cuerpo en `DELETE` poco interoperable y confirmación incómoda); `GET` para descubrir (cacheable y precargable por el navegador; se prefiere `POST` porque usa el secreto y la red).

**Razón**: cada ruta tiene un único efecto, sus códigos son estables y las respuestas no contienen nada derivado del secreto.

### D-07: Pestaña «Integraciones» del cambio con lector público de la outbox

**Elección**: `internal/integrations` añade:

```go
// OperationView es la vista de solo lectura de una operación de la outbox. No lleva la clave de
// idempotencia, el texto del último error, el actor ni el digest del artefacto.
type OperationView struct {
	Plugin        string
	Event         EventType
	Role, Gate    string
	State         OperationState
	Attempts      int
	EnqueuedAt    time.Time
	LastAttemptAt time.Time
	Reason        string
}

// OutboxView lee la outbox de la carpeta sin cerrojo (la escritura es atómica) y la devuelve
// ordenada por EnqueuedAt, plugin y tipo de evento. Sin fichero devuelve una lista vacía.
func OutboxView(f Folder) ([]OperationView, error)
```

`GET /api/integrations/change?name=<cambio>` valida el nombre con el mismo patrón que el explicador (`validExplainerChangeName`, que rechaza `..`, separadores y `:`), lo localiza con `locateIncrement` (activo, en worktree o archivado) y construye `integrations.Folder{WorkspaceRoot, Root}`, con `WorkspaceRoot` igual al padre del directorio `openspec` que contiene la carpeta. Lee `WorkItemLinks`, `TaskLinks`, `TimeEntries`, `PendingTimeFor` por rol, `OutboxView`, `kickoff.Load` (`SealedBy` y roster) y `kickoff.LoadGates` (`Actor` por puerta). Cada lectura que falla (fichero corrupto o de versión desconocida) añade una frase a `errors` y deja vacía su sección, sin romper la pestaña. La outbox se limita a 100 operaciones con `outboxTruncated`. Un rol del roster sin Task se muestra «sin Task»; un actor ausente (cambios anteriores a C-07-a), «sin registrar». Del tiempo pendiente solo se publican `minutes`, `date` e `itemId`.

El enlace al work item se publica solo si `Link.URL` supera `safeExternalURL` (D-09); su etiqueta es «Abrir en <Title del manifiesto>» («Abrir en Azure DevOps» para ADO, prov:WEB:10), y cae en el id del plugin si su manifiesto ya no está en el catálogo (vínculos históricos).

**Alternativas consideradas**: añadir los datos a `IncrementDetailDTO` (encarece todos los detalles y deja estos datos fuera de la comprobación de `Host` de `/api/integrations*`); exportar `readOutbox` (expone `LastError`, la clave y el actor).

**Razón**: la redacción vive en el paquete dueño de la outbox, de modo que ningún llamante puede filtrar más de lo debido, y la pestaña se carga bajo demanda.

### D-08: Alta desde un work item con las fases de `change start`, sin sellar el kickoff

**Elección**: el alta con work item es `POST /api/integrations/start` con `{ref?, change?, intent?, type?, mode?}`. Exactamente uno de `ref` (primer intento) o `change` (reintento sin fase 1) es obligatorio. Secuencia del servicio (`Service.StartFromWorkItem`):

1. Validación: `ref` recortado, de 1 a 512 caracteres y sin caracteres de control; `change` válido según `changestart.ValidChangeName` (expuesto por el gateway como parte de `WorkItemStaged`); `intent` de como mucho 4 KiB.
2. Con `ref`: `gateway.StartWorkItem(ctx, cwd, ref)` (fase 1). Un rechazo responde `422` con `{code, error, continuation, stage: "start"}` y, por contrato de la fase 1, no deja *staging*, rama, carpeta ni `kickoff.yaml`. Con `change`: `gateway.WorkItemStaged(cwd, change)`; sin *staging*, `409 staging_not_found` sin crear nada.
3. `ensureIncrementBranch(change, mode)` con el nombre derivado y carril SDD, la misma función que el alta actual. `decision_required` responde `409` con la decisión y con `change`, y la interfaz repite con `{change, mode}` (sin repetir la fase 1). Otro rechazo responde `409` con `conflictBody` más `change` y `actions: ["retry","discard"]`.
4. Carpeta del cambio: `writeIncrement` se divide en `prepareIncrementFolder` (guarda `cli.GuardChangeWriter` y `MkdirAll`) y `writeIncrementProposal`. En un reintento, una carpeta ya existente del mismo cambio se reutiliza.
5. Fase 2: `gateway.AdoptWorkItem(ctx, cwd, change)` con `cwd = filepath.Dir(planningHome)` del resultado de `ensure` (el worktree del cambio en modo `worktree`). Sus rechazos (`change_not_prepared`, `staging_not_found`, `staging_conflict`, `change_start_busy`, `write_failed`) responden `409` o `422` con mensaje, continuación, `change` y `actions: ["retry","discard"]`. Sus advertencias con éxito (`backlink pending|conflict`, `staging_not_retired`) se devuelven en `warnings` y no detienen el flujo.
6. `proposal.md` con la plantilla del alta actual si aún no existe. La intención es la que teclea la persona o, si falta, un texto genérico que cita `external_ref`; nunca se copia texto del work item (dato de terceros no confiable).
7. Respuesta `200` `{success, name, path, externalRef, warnings}`.

El descarte es `POST /api/integrations/start/discard` con `{change, confirm: true}` y llama a `gateway.DiscardWorkItem`, que solo retira el *staging* (como `axiom change start --discard`); nunca se ejecuta de forma automática. La interfaz muestra «Reintentar» y «Descartar» tras cualquier fallo posterior a la fase 1, y «Descartar» abre una confirmación explícita.

**El dashboard no sella `kickoff.yaml`**, igual que el alta actual (`CreateIncrement` nunca lo ha sellado): sellar exige que la persona elija el estilo de ejecución, una decisión de gobierno que el dashboard no pregunta. El kickoff lo sella después el orquestador con `axiom sdd kickoff seal --infer` al retomar el cambio, como con cualquier cambio creado desde el dashboard. En este flujo, el paso de «sellado» de prov:WEB:9 se materializa como la creación guardada de la carpeta del cambio (paso 4), que es lo que la fase 2 exige (`folder_missing`). Se declara como desviación en «Huecos de cobertura» (H-20).

Recarga del navegador o caída a mitad: volver a dar el mismo work item reanuda, porque la fase 1 es idempotente (`already_staged`) y `ensure`, la carpeta y la fase 2 también lo son.

**Alternativas consideradas**: ampliar `POST /api/increments` con el work item (mezcla dos contratos de error y no queda bajo la comprobación de `Host` en `GET` de `/api/integrations*`); sellar `checkpointed` con `fullstack` por defecto (decide gobierno por la persona y diverge del alta manual); un listado de *staging* huérfano en la interfaz (fuera de alcance; la reanudación por el mismo work item lo cubre).

**Razón**: se reutiliza la lógica de `changestart` y del alta actual sin duplicarla, con las mismas salidas que el orquestador.

### D-09: Validación de URL externa y etiqueta genérica por plugin

**Elección**: `safeExternalURL(raw string) bool` en el servidor y su equivalente en `core/integrations.js`: el texto no puede tener blancos ni caracteres de control al inicio, al final ni dentro; `url.Parse` sin error; esquema `http` o `https` (comparación sin distinguir mayúsculas); `Host` no vacío; sin `User`. Se rechazan `javascript:`, `data:`, relativas y `//host`. El servidor omite `url` si no la supera; el cliente vuelve a validar y crea el enlace con `a.href = …` y `a.textContent = …`, `target="_blank"` y `rel="noopener noreferrer"`, nunca con `innerHTML`. La etiqueta sale de `Manifest.Title`.

**Alternativas consideradas**: comparar el host con la organización configurada (la URL la construye el propio plugin y `links.go` ya exige `http(s)` sin `userinfo` al escribirla; se deja fuera); etiqueta fija «Abrir en Azure DevOps» (contradice la pantalla genérica).

**Razón**: doble validación barata y una sola fuente para la etiqueta.

### D-10: Pantalla «Integraciones» genérica sobre el catálogo de ajustes

**Elección**: `screens/integraciones.js` pinta un bloque por manifiesto de `GET /api/integrations` y reutiliza `controlHtml` y `collectChanges` de `core/settings.js` con las entradas de sección `integrations`, de modo que tipo, enumeración, obligatoriedad, mínimos y máximos salen del catálogo. Guardar envía un único `PUT /api/settings` con la revisión leída; un `409` ofrece recargar y conserva lo tecleado; con `writable: false` los controles se deshabilitan con el motivo; `documentErrors` se muestran arriba. El PAT usa `type="password"` y `autocomplete="off"`, se vacía tras guardar y nunca entra en el lote de `PUT /api/settings` ni en `localStorage`. Todo texto del manifiesto, de la configuración y del tracker se inserta con `escapeHtml` o `textContent`.

«Descubrir» muestra un panel de solo lectura con tipos (y sus estados), iteraciones y miembros, y añade a cada campo de texto libre un `<datalist>` con los nombres de tipo y de estado descubiertos (como mucho 500 opciones por lista, con aviso si se recortan). Son sugerencias, no restricciones: un valor configurado que no aparece en lo descubierto se conserva y se marca «no encontrado en el tracker». Los campos con `Enum` siguen siendo selectores del manifiesto. Desactivar un plugin es un `enabled: false`; la pantalla avisa de que las operaciones pendientes de la outbox quedan pendientes (`plugin_not_configured`). Las plantillas de rama no se duplican: hay un enlace a `#/ajustes`. «Ajustes del proyecto» omite la sección `integrations` y enlaza a `#/integraciones` (RF-WEB-18).

**Alternativas consideradas**: asociar cada campo a una lista concreta del descubrimiento (exige una pista nueva en el esquema del manifiesto, fuera de alcance); un formulario propio sin el catálogo (duplica validación).

**Razón**: la pantalla no conoce ningún plugin y la escritura sigue la única puerta.

## Flujo de datos

```
Navegador (SPA, mismo origen)
   │  fetch /api/...
   ▼
withHeaders ── cabeceras (CSP, nosniff, Referrer-Policy, no-store en /api/integrations*)
   │         ── Host/Origin locales (muta o /api/integrations*) ──✗──► 403 forbidden_origin
   │         ── OPTIONS ──► 405
   ▼
ServeMux
   ├─ GET  /api/integrations ─► manifest.Catalog() + settings.Read() + secrets.Default().Get()
   │                            + identitystore.Load() + gateway.WorkItemApplicability()
   ├─ PUT  /api/settings ─────► settings.Write ─► axiom.yaml (único escritor)
   ├─ POST /api/integrations/secret[/delete] ─► secrets.Default().Set|Delete ─► keyring
   ├─ POST /api/integrations/connect|discover ─► IntegrationsGateway (cli) ─► plugin ─► tracker
   │                                              └─► ~/.axiom/integration-identities.json
   ├─ GET  /api/integrations/change ─► integrations.{WorkItemLinks,TaskLinks,TimeEntries,
   │                                   PendingTimeFor,OutboxView} + kickoff.{Load,LoadGates}
   └─ POST /api/integrations/start ─► fase 1 ─► ensure ─► carpeta ─► fase 2 ─► proposal.md
```

Alta desde un work item:

```
ref ──► gateway.StartWorkItem ──rechazo──► 422 (sin rastro)
          │ staged(change)
          ▼
        ensureIncrementBranch(change, mode) ──decision_required──► 409 {decision, change}
          │ planningHome                    ──otro rechazo──────► 409 {change, actions}
          ▼
        prepareIncrementFolder ──fallo──► 500 {change, actions}
          ▼
        gateway.AdoptWorkItem(Dir(planningHome)) ──rechazo──► 409/422 {change, actions}
          ▼                                       ──aviso───► warnings
        writeIncrementProposal ─► 200 {name, externalRef, warnings}

actions = Reintentar {change, mode?} | Descartar {change, confirm: true}
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/dashboard/server.go` | Modificar | `withHeaders` (D-02), `dashboardCSP`, `handler()`, `isLocalRequest` endurecido, rutas nuevas |
| `internal/dashboard/integrations_port.go` | Crear | Interfaz `IntegrationsGateway` y `SetIntegrationsGateway` |
| `internal/dashboard/integrations_api.go` | Crear | Handlers de `/api/integrations*`, presupuestos, cerrojos y mapeo de errores |
| `internal/dashboard/service_integrations.go` | Crear | Estado de plugins, secreto, vista por cambio, `safeExternalURL` |
| `internal/dashboard/service_workitem.go` | Crear | `StartFromWorkItem` y descarte (D-08) |
| `internal/dashboard/service_branch.go` | Modificar | División de `writeIncrement` en `prepareIncrementFolder` y `writeIncrementProposal` |
| `internal/dashboard/types.go` | Modificar | DTO de integraciones, de la vista por cambio y del alta; `SettingDTO` con `plugin`, `format`, `min`, `max` |
| `internal/dashboard/settings_api.go` | Modificar | Publicar los campos nuevos de `Entry` |
| `internal/dashboard/secrets_canary_test.go` | Modificar | Clasificación de rutas, uso de `handler()` con `Host` local y cobertura de `/api/integrations` y `/api/integrations/change` |
| `internal/dashboard/security_headers_test.go` | Crear | Pruebas de prov:WEB:1 y prov:WEB:2 |
| `internal/dashboard/integrations_api_test.go`, `workitem_start_test.go`, `change_integrations_test.go`, `integrations_assets_test.go` | Crear | Pruebas por unidad |
| `internal/dashboard/assets_test.go`, `assets_settings_test.go`, `server_test.go` | Modificar | Script de tema externo, enlace a «Integraciones», módulos servidos |
| `internal/dashboard/assets/index.html` | Modificar | Script de tema por `src`, clases en lugar de `style`, vista y navegación «Integraciones», panel de pestaña, campo de work item en el alta |
| `internal/dashboard/assets/js/theme-init.js` | Crear | Script de tema síncrono |
| `internal/dashboard/assets/js/core/cssvars.js` | Crear | `applyCssData` y `observeCssData` (D-01) |
| `internal/dashboard/assets/js/core/integrations.js` | Crear | Cliente de la API, `safeExternalURL`, textos de procedencia |
| `internal/dashboard/assets/js/screens/integraciones.js` | Crear | Pantalla (D-10) |
| `internal/dashboard/assets/js/screens/detalle-integraciones.js` | Crear | Pestaña del cambio (D-07) |
| `internal/dashboard/assets/js/core/router.js`, `main.js`, `shell.js`, `screens/detalle.js`, `screens/ajustes.js`, `screens/incrementos.js`, `core/increment.js`, `core/icons.js`, `projects.js`, `style.css` | Modificar | Rutas, pestaña, navegación, estilos en línea retirados, alta con work item |
| `internal/settings/catalog.go`, `patch.go`, `document.go` | Modificar | Ámbito `integrations[<plugin>].<clave>` (D-03) |
| `internal/settings/catalog_test.go`, `document_test.go`, `patch_test.go`, `write_test.go` | Modificar | Las pruebas de exclusión pasan a probar la inclusión |
| `internal/settings/integrations_test.go` | Crear | Coherencia con el decodificador y casos del ámbito nuevo |
| `internal/integrations/outbox.go` | Modificar | `OperationView` y `OutboxView` (D-07) |
| `internal/integrations/outbox_view_test.go` | Crear | Orden, redacción y lectura sin fichero |
| `internal/integrations/secrets/store.go` | Modificar | `MaxValueBytes` |
| `internal/cli/integrations_gateway.go` | Crear | `IntegrationsGateway` y tipos de resultado (D-04) |
| `internal/cli/integrations_gateway_test.go` | Crear | Envolturas sobre `adotest` y el doble de `integrationstest` |
| `internal/cli/integrations.go` | Modificar | `maxSecretBytes` pasa a `secrets.MaxValueBytes` |
| `cmd/axiom/main.go` | Modificar | `svc.SetIntegrationsGateway(cli.NewIntegrationsGateway())` |

## Interfaces y contratos

```go
// internal/cli/integrations_gateway.go
type IntegrationsGateway struct{}
func NewIntegrationsGateway() *IntegrationsGateway

type IntegrationConnectOutcome struct {
	Identity   integrations.TrackerIdentity
	ResolvedAt string
	Replaced   *identitystore.Entry
}
type WorkItemApplicability struct {
	Applicable bool
	Plugin     string // id del plugin que aporta el puerto de work items
	RefAlias   string
	Reason     string // por qué no aplica, en castellano
}
type WorkItemStartOutcome struct {
	Change, ExternalRef, Status string // staged | already_staged
	Warnings                   []string
}
type WorkItemAdoptOutcome struct {
	Change, Status string // adopted | already_adopted
	Warnings       []string
}
type WorkItemDiscardOutcome struct {
	Change, Status string
	Warnings       []string
}
type WorkItemRefusal struct {
	Code, Message, Continuation string
	Fields                      map[string]string
}
func (r *WorkItemRefusal) Error() string { return r.Message }
```

```go
// internal/dashboard/types.go (extracto)
type IntegrationsDTO struct {
	ConfigPath     string                  `json:"configPath"`
	Revision       string                  `json:"revision"`
	Writable       bool                    `json:"writable"`
	WriteBlocked   string                  `json:"writeBlockedReason,omitempty"`
	DocumentErrors []string                `json:"documentErrors"`
	Plugins        []IntegrationPluginDTO  `json:"plugins"`
	WorkItems      WorkItemApplicabilityDTO `json:"workItems"`
}
type IntegrationPluginDTO struct {
	Manifest   manifest.Manifest        `json:"manifest"`
	Enabled    bool                     `json:"enabled"`
	Declared   bool                     `json:"declared"`
	Entries    []SettingDTO             `json:"entries"`   // las de sección integrations de este plugin
	Secrets    []SecretStatusDTO        `json:"secrets"`   // {name, source, envVar, required}
	Identities []IdentityDTO            `json:"identities"` // {account, id, displayName, source, resolvedAt}
	IdentityError string                `json:"identityError,omitempty"`
}
```

`ChangeIntegrationsDTO` lleva `change`, `archived`, `workItems[]` (`plugin`, `id`, `url?`, `linkLabel`, `recordedAt`), `roles[]` (`role`, `plugin`, `taskId`, `url?`, `estimateHours?`, `startedAt?`, `timeEntries[]` con `minutes`, `date` y `actor`, `pending?` con `minutes`, `date` e `itemId`), `outbox[]` (los campos de `OperationView`), `outboxTruncated`, `actors` (`sealedBy`, `gates[]` con `gate`, `decision`, `actor` y `recordedAt`) y `errors[]`.

## Huecos de cobertura

Concreción de los 22 huecos de `coverage-review.spec.md` (ronda 1). «Resuelto» significa que el diseño fija el comportamiento y su prueba; «parcial» o «fuera», que queda una parte sin cubrir con su motivo.

| Hueco | Estado | Concreción |
|---|---|---|
| H-01 (prov:WEB:2) | Resuelto | Lista cerrada de `Host` y regla de `Origin` de D-02; `null`, `https`, nombre DNS local y `Host` vacío se rechazan; `Host` duplicado lo rechaza `net/http`. |
| H-02 (prov:WEB:2) | Resuelto | Pruebas de `HEAD` y `OPTIONS` sobre `/api/integrations*` con `Host` ajeno (`403`), de regresión sobre `POST /api/increments/marks`, `PUT /api/projects/active/worktree-prefs`, `PUT /api/settings/personal` y `POST /api/increments`, y de petición local sin `Origin` atendida. |
| H-03 (prov:WEB:1) | Resuelto | `OPTIONS` responde `405` sin `Access-Control-*`; un `POST` simple `text/plain` desde origen ajeno cae en `403` por `Origin`, y las rutas nuevas exigen además `application/json` (`400`). |
| H-04 (prov:WEB:1) | Resuelto | Cabeceras fijadas antes del handler (también en `403`, `404`, `405`, `500`, `304` y `HEAD`); `nosniff` y `Referrer-Policy: no-referrer` en todo; `Cache-Control: no-store` en `/api/integrations*`. |
| H-05 (prov:WEB:1) | Parcial | Prueba Go que recorre los assets y falla con `style=`, `setAttribute('style'` o `<script>` con cuerpo; prueba del MIME de `theme-init.js` y de su orden en `<head>`; `style-src 'self'` fijado por prueba. Falta una prueba en navegador real (el repositorio no tiene ejecutor de JS): queda como comprobación manual en `verify`. |
| H-06 (prov:WEB:4) | Resuelto | Validación del PAT de D-06 (tipo de contenido, vacío, control, longitud, campos desconocidos, plugin o secreto desconocido, *query string* y `GET`). El plugin inactivo no impide guardar (ámbito de usuario). |
| H-07 (prov:WEB:4) | Resuelto | Eliminación idempotente, `500 secret_store_failed` ante fallo del almacén y `confirm: true` obligatorio más confirmación en la interfaz. |
| H-08 (prov:WEB:4) | Resuelto | `500 secret_store_failed` para fallos distintos de `ErrNoWritableStore`; `shadowedByEnv` cuando la variable tapa el almacén; `secrets.Default()` se consulta en cada llamada, así que el PAT guardado está disponible al instante. |
| H-09 (prov:WEB:5) | Parcial | Tabla de errores de D-06. Credencial rechazada y fallo de red comparten `502 integration_failed` con el mensaje saneado del plugin, que ya los distingue en el texto; separar códigos exigiría cambiar el contrato de `PortError` (fuera de alcance). |
| H-10 (prov:WEB:5) | Resuelto | La elección se vuelve a resolver en el servidor (`422 identity_user_unknown`); `replaced` informa del cambio de identidad; `500 identity_store_failed` si falla el fichero; cerrojo `409 integration_busy` para pruebas simultáneas. |
| H-11 (prov:WEB:5, prov:WEB:6) | Resuelto | Siempre con lo guardado en `axiom.yaml`; la interfaz desactiva los botones con cambios sin guardar. |
| H-12 (prov:WEB:5, prov:WEB:6) | Parcial | `504 integration_timeout`, cancelación por `r.Context()`, `SetWriteDeadline` y cerrojo anti doble clic (D-05). La limitación de frecuencia queda fuera: servidor local de un único usuario. |
| H-13 (prov:WEB:6) | Resuelto | Listas vacías: panel con estado vacío; listas grandes: tope de 500 opciones con aviso; valor configurado ausente: se conserva y se marca; solapamiento: cerrojo. Un descubrimiento parcial no existe en el contrato del puerto (devuelve todo o error). |
| H-14 (prov:WEB:7) | Resuelto | `409 revision_mismatch` con recarga y conservación de lo tecleado; `writable: false` deshabilita; un bloque inválido escrito a mano se ve en `documentErrors` y se corrige con un lote válido. |
| H-15 (prov:WEB:7) | Resuelto | Desactivar con aviso sobre la outbox; cero plugins: estado vacío; varios: un bloque por manifiesto; listas con un valor por línea y `null` para vaciar; PAT `type="password"`, `autocomplete="off"`, vaciado tras guardar, nunca en el lote ni en `localStorage`; etiquetas `label for` y `aria-live` en los avisos. |
| H-16 (prov:WEB:8) | Resuelto | Errores parciales en `errors[]`; outbox limitada a 100 y ordenada; vista redactada (`OperationView`); vínculos históricos con plugin desactivado; «sin Task» y «sin registrar». |
| H-17 (prov:WEB:8) | Resuelto | Lectura sin cerrojo sobre escrituras atómicas; `404 change_not_found`; nombre validado contra *path traversal*. |
| H-18 (prov:WEB:9) | Resuelto | Validación de la entrada de D-08; URL de otra organización, WI inexistente, sin permisos, ya enlazado o nombre que colisiona: rechazos tipados de la fase 1; plugin o PAT que desaparecen: aplicabilidad recalculada en el servidor; el contenido del WI nunca se copia en `proposal.md` ni en la interfaz. |
| H-19 (prov:WEB:9) | Parcial | Reanudación por el mismo WI, cerrojo anti doble envío, `decision_required` con `{change, mode}`, rechazos de la fase 2 mapeados y presupuesto de 120 s. Fuera: listado en la interfaz del *staging* huérfano tras recargar (lo cubre la reanudación por el mismo WI). |
| H-20 (prov:WEB:9) | Resuelto con desviación | Rutas de reintento y descarte definidas, bajo `Host` y `Origin`, clasificadas en el canario y con confirmación. Carril SDD, modo de worktree por `ensure` y su `decision_required`, multirepo e ISO como el alta actual. **Desviación declarada**: el dashboard no sella `kickoff.yaml` (D-08); el kickoff se sella después con `--infer`. La spec habla de «sellado del kickoff»; conviene ajustar su redacción al archivar o aceptar esta lectura en `verify`. |
| H-21 (prov:WEB:10) | Resuelto | `safeExternalURL` (D-09) en servidor y cliente; etiqueta «Abrir en <Title>». Fuera: comparar el host con la organización (la URL la construye el plugin y se valida al escribirla). |
| H-22 (prov:WEB:11, RF-TOP-21, RF-TOP-22) | Resuelto | Canario sobre el envoltorio completo y respuestas de las rutas `POST` con un doble que intenta filtrar el centinela; catálogo: plugin desconocido, valor huérfano, listas, claves con `-`, obligatoriedad condicional, orden estable y ambas topologías; escritor: tipos no cadena, comentarios al borrar el bloque y posición del bloque nuevo. El servidor no registra cuerpos de petición (sin logs nuevos). |

## Estrategia de pruebas

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitaria (`internal/settings`) | Ámbito nuevo, coherencia con el decodificador, round-trip del bloque | `go test` con manifiestos de prueba (`manifest.SetCatalogForTest`) y ficheros dorados |
| Unitaria (`internal/integrations`) | `OutboxView` | Fichero de outbox en `t.TempDir()` |
| Integración (`internal/dashboard`) | Cabeceras, guarda local, API, vista por cambio, alta con WI | `handler()` con `httptest`, doble de `IntegrationsGateway`, `integrationstest.Install` y `secrets.SetDefaultForTest`; sin `t.Parallel()` en las que tocan estado de paquete, `freePort` o el hub |
| Integración (`internal/cli`) | Gateway sobre la lógica real | `adotest` (servidor HTTP doble de Azure DevOps) y el plugin `fake` con `LeakTokenInError`; nunca Azure DevOps real |
| Assets | Estructura de HTML y JS | Pruebas Go de cadenas (no hay ejecutor de JS) |
| Barandillas | Importaciones, rechazos, canario | `TestImportsDirection`, `TestAdapterAssemblyImportRules`, `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign`, `TestEveryDashboardRouteIsClassifiedForTheSecretsCanary` |

Pruebas en ROJO planificadas por requisito: prov:WEB:1 (`TestSecurityHeadersOnEveryResponse`, `TestNoAccessControlHeaders`, `TestOptionsIsMethodNotAllowed`, `TestExplainerKeepsItsOwnCSPOnce`, `TestNoInlineStyleOrScriptInAssets`, `TestThemeInitScriptIsExternalAndFirst`); prov:WEB:2 (`TestLocalGuardRejectsForeignHostAndOrigin`, `TestLocalGuardAcceptsLocalWithoutOrigin`, `TestLocalGuardCoversExistingMutatingRoutes`); prov:WEB:3 y prov:WEB:4 (`TestIntegrationsAPIStatus*`, `TestIntegrationsAPISecret*`); prov:WEB:5 y prov:WEB:6 (`TestIntegrationsAPIConnect*`, `TestIntegrationsAPIDiscover*`, `TestIntegrationsAPITimeout`); prov:WEB:7 (`TestIntegrationsScreenAssets*`); prov:WEB:8 (`TestChangeIntegrations*`, `TestOutboxView*`); prov:WEB:9 y prov:WEB:10 (`TestWorkItemStart*`, `TestSafeExternalURL`); prov:WEB:11 (canario); RF-TOP-21 y RF-TOP-22 (`TestIntegrationsCatalog*`, `TestIntegrationsPatch*`, `TestIntegrationsWrite*`); RF-WEB-18 y RF-WEB-19 (`TestSettings*Integrations*`).

## Matriz de amenazas

El cambio no ejecuta procesos ni automatiza commits, *push* ni PR, pero el alta con WI prepara ramas y worktrees mediante `ensure`, así que aplica la selección de repositorio.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta del diseño | Pruebas en ROJO |
|---|---|---|---|---|
| Rutas con aspecto de documentación | `requirements.txt`, Markdown ejecutable, `README.sh` | N/A: no se clasifican ni ejecutan ficheros | — | — |
| Selección del repositorio git | `git -C`, rutas relativas, rutas absolutas | Aplicable | `ensure` recibe la ruta absoluta del proyecto activo (`filepath.Abs`); la fase 2 recibe `filepath.Dir(planningHome)` del resultado de `ensure`, nunca el directorio de trabajo del proceso; el nombre del cambio sale de la fase 1 y se valida con `ValidChangeName` | `TestWorkItemStartAdoptsInTheEnsureWorktree` (modo `worktree`: la fase 2 recibe la raíz del worktree) y `TestWorkItemStartRejectsInvalidChangeName` (`../x`, `a/b`, absoluto) |
| Estado del commit | índice, `commit -a`, índice vacío | N/A: el dashboard no hace commits | — | — |
| Estado del *push* | rama de seguimiento, primer *push*, *refspec* | N/A: no hay *push* | — | — |
| Comandos de PR | `--head`, prefijo de entorno, comandos compuestos | N/A: no hay PR | — | — |

## Plan de unidades de trabajo

Cada unidad es un commit (Conventional Commits) con sus pruebas y documentación, en este orden. Las pruebas son localizadas; la batería completa corre en CI.

| # | Unidad | Ficheros principales | Pruebas focalizadas | Líneas estimadas |
|---|---|---|---|---|
| 1 | Seguridad (primer commit, antes de manejar tokens) | `server.go` (D-02), `assets/index.html`, `js/theme-init.js`, `js/core/cssvars.js`, `core/icons.js`, `core/increment.js`, `screens/incrementos.js`, `projects.js`, `main.js`, `style.css`, `security_headers_test.go`, `assets_test.go` | `go test ./internal/dashboard -run 'TestSecurityHeaders\|TestNoAccessControl\|TestOptionsIs\|TestExplainerKeeps\|TestLocalGuard\|TestNoInlineStyle\|TestThemeInit\|TestIndexHTMLShell\|TestAssetsAreServedWithTheirContentType\|TestExplainer\|TestSettingsPutFromAForeignHost\|TestWorktreePrefsPutFromAForeignHost'` | 350 |
| 2 | Catálogo | `internal/settings/{catalog,patch,document}.go` y pruebas, `types.go`, `settings_api.go`, `screens/ajustes.js`, `core/settings.js`, `assets_settings_test.go` | `go test ./internal/settings -run 'TestIntegrations\|TestCatalog\|TestPatch\|TestWrite\|TestRead\|TestLayout'`; `go test ./internal/dashboard -run 'TestSettings'`; `go test ./internal/integrations -run 'TestImportsDirection'` | 450 |
| 3 | API | `integrations_port.go`, `integrations_api.go`, `service_integrations.go`, `types.go`, `server.go`, `secrets/store.go`, `cli/integrations.go`, `cli/integrations_gateway.go`, `cmd/axiom/main.go`, `secrets_canary_test.go`, `integrations_api_test.go`, `cli/integrations_gateway_test.go` | `go test ./internal/dashboard -run 'TestIntegrationsAPI\|TestEveryDashboardRouteIsClassified\|TestNoReadOnlyDashboardRoute'`; `go test ./internal/cli -run 'TestIntegrationsGateway\|TestEveryProductionRefusal\|TestIntegrationsSecret'`; `go test ./internal/integrations -run 'TestImportsDirection\|TestAdapterAssemblyImportRules'` | 950 |
| 4 | Pantalla «Integraciones» | `screens/integraciones.js`, `core/integrations.js`, `index.html`, `core/router.js`, `main.js`, `shell.js`, `style.css`, `integrations_assets_test.go` | `go test ./internal/dashboard -run 'TestIntegrationsScreen\|TestRouterDefinesEveryRoute\|TestIndexHTMLShell\|TestJSModuleImportsResolve\|TestAssetsAreServedWithTheirContentType'` | 500 |
| 5 | Vista por cambio | `integrations/outbox.go`, `outbox_view_test.go`, `service_integrations.go`, `types.go`, `screens/detalle-integraciones.js`, `screens/detalle.js`, `core/router.js`, `index.html`, `change_integrations_test.go` | `go test ./internal/integrations -run 'TestOutboxView'`; `go test ./internal/dashboard -run 'TestChangeIntegrations\|TestRouterDefinesEveryRoute\|TestJSModuleImportsResolve\|TestNoReadOnlyDashboardRoute\|TestEveryDashboardRouteIsClassified'` | 550 |
| 6 | Alta con WI | `service_workitem.go`, `service_branch.go`, `integrations_api.go`, `cli/integrations_gateway.go`, `screens/incrementos.js`, `core/branches.js`, `index.html`, `workitem_start_test.go`, `cli/integrations_gateway_test.go`, `secrets_canary_test.go` | `go test ./internal/dashboard -run 'TestWorkItemStart\|TestSafeExternalURL\|TestCreateIncrement_\|TestEveryDashboardRouteIsClassified'`; `go test ./internal/cli -run 'TestIntegrationsGatewayWorkItem\|TestEveryProductionRefusal\|TestChangeStart'` | 700 |

Estimación total: unas 3.500 líneas, de las que cerca de la mitad son pruebas. Supera la previsión de 1.000-1.400 de la propuesta, que no contaba las pruebas ni la retirada de estilos en línea; la entrega sigue siendo `single-pr` con un commit por unidad.

## Migración y despliegue

No hay migración de datos. `axiom.yaml` solo cambia cuando la persona guarda desde la interfaz. Efectos visibles al desplegar:

- Un cliente que llamara a la API desde otro origen deja de recibir CORS y, si muta, recibe `403`; la SPA es del mismo origen y no se ve afectada.
- Un `axiom.yaml` con anclas, alias o `<<` dentro de `integrations` pasa a mostrarse no escribible en Ajustes (antes ese bloque se ignoraba).
- Las imágenes con URL externa dentro de Markdown renderizado quedan bloqueadas por `img-src 'self' data:` (hoy no hay ninguna en los assets).

Reversión: revertir la PR; la unidad 1 puede revertirse sola, pero entonces deben revertirse también las unidades 3, 4 y 6 (propuesta, plan de reversión).

## Preguntas abiertas

- [ ] La redacción de prov:WEB:9 habla de «sellado del kickoff»; el diseño mantiene el comportamiento del alta actual (sin sellar) y lo declara como desviación (H-20). Hay que confirmarlo en `verify` o ajustar la redacción del requisito al archivar. No bloquea `sdd-tasks`.
