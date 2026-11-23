## Exploración: adaptador-ado (C-07-c) — Adaptador Azure DevOps: cliente REST, conexión, identidad y descubrimiento

### Estado actual (marco C-07-b, ya fusionado)

- **Puerto.** `integrations.Plugin` tiene solo `Manifest()` y `Deliver(ctx, Delivery) (Outcome, error)` (`internal/integrations/plugin.go:55-60`). No existe interfaz de `discovery`, de identidad ni de conexión: las capacidades `discovery`, `items.read`, `assignment`, `vcs_links`… son etiquetas del manifiesto (`manifest/capabilities.go:6-18`) sin método asociado. C-07-c debe definir esas interfaces opcionales (type assertion o interfaces nuevas en el paquete `integrations`).
- **Errores.** `integrations.Permanent(err)` / `ErrPermanent` marca un fallo sin reintento (`plugin.go:62-81`). `Deliver` debe ser idempotente por `Event.IdempotencyKey`, respetar `ctx` y no incluir secretos en el error.
- **Manifiesto.** `manifest.Manifest{Schema, ID, Title, Capabilities, Events, Config, Secrets, Identity}` (`manifest/manifest.go:78-87`). ID `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`, máx. 32. Claves de config `^[a-z][a-z0-9_]*$`; se rechazan las que parecen secreto (`token`, `pat`, `secret`…, `manifest.go:30-43`) y `enabled` (reservada). Tipos: string, bool, int, string_list. `IdentitySpec{Source: none|git_actor|plugin, Secret}`; `IdentityPlugin` está documentado como «la resuelve el plugin contra el tracker (C-07-c)» (`manifest.go:68-69`).
- **Registro cerrado (dos alta obligatorias).** (1) Añadir el manifiesto a `builtinCatalog` en `manifest/catalog.go:10` (hoy `[]Manifest{}`, «vacío hasta C-07-c»); (2) añadir el plugin a `builtinPlugins` en `registry.go:56` (hoy vacío). `NewRegistry` exige que el manifiesto del plugin sea `reflect.DeepEqual` al del catálogo (`registry.go:25-46`). Hay costuras de prueba `manifest.SetCatalogForTest` y `integrations.SetDefaultRegistryForTest`.
- **Secreto.** `SecretSource.Secret(ctx, name)` acotado a los secretos del manifiesto y rastreado para redacción (`sanitize.go:86-125`). Almacén `secrets.Default()` = cadena env → keyring (`internal/integrations/secrets/{chain,env,keyring,default}.go`; keyring solo Windows/macOS, servicio `"axiom"`, en Linux solo env). Variable de entorno: `manifest.EnvVarName("azure-devops","pat")` = `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT` (`manifest/envname.go:11-14`, ejemplo literal en el comentario). `secrets.Secret` se redacta en `String/Format/JSON/YAML/slog`. Errores del plugin se sanean (`sanitizeText`, tope 512 caracteres) pero **solo** redactan valores resueltos vía `SecretSource`: si C-07-c construye el valor Basic (base64 de `:PAT`) debe rastrearlo/redactarlo él mismo, porque base64 no coincide con el valor rastreado.
- **Validación de `integrations.<plugin>` en `axiom.yaml`.** `internal/workspace/integrations.go:52-153`: plugin desconocido = error de carga («plugins admitidos: …»), duplicado, claves fuera del manifiesto, tipos, obligatorias; `loc.Declarations.IntegrationsErr` lo propagan los verbos. Todo se deriva del catálogo, así que añadir el manifiesto basta; `workspace/example_config_test.go:105-115` tiene una prueba que afirma «catálogo vacío hasta C-07-c» y fallará al añadir el catálogo (hay que actualizarla).
- **Eventos.** Catálogo cerrado en `manifest/events.go` (`change.start` lo emite C-07-d, no emitible aún: `event.go:98`, `cli/integrations.go:383`). Despacho: `dispatcher.go` (`Dispatch`, `attempt` construye `Delivery{Config, Links, Secrets}` en `:331`), plazo inmediato y por operación (`dispatcher.go:91-104`), outbox YAML por cambio (`outbox.go`), vínculos en `integrations.yaml` (`links.go`).
- **Doble en memoria.** `integrationstest.Fake` (`integrationstest/fake.go`): registra entregas, fallos programables, `ReadToken`, `LeakTokenInError`, idempotente por clave; `integrationstest.Install` (`install.go`) instala catálogo, registro y almacén. Patrón a imitar para pruebas del adaptador.
- **Puntos de extensión que C-07-b dejó a C-07-c (comentarios):** `registry.go:54-55`, `manifest/catalog.go:8-9`, `event.go:4`, `manifest/manifest.go:68`, `workspace/example_config_test.go:105,115`. C-07-d: `change.start`. C-07-e: `TimeEntry` (`plugin.go:43`).

### CLI actual

- `axiom integrations` solo tiene `sync`, `emit` y `secret set|status|delete` (`internal/cli/integrations.go:41-49, 136-150`; despacho en `cmd/axiom/main.go:326`). **No hay verbo de conexión** (`connect`/`login`): hay que añadirlo. `secret set` lee solo por stdin (máx. 4 KiB, se niega con terminal, `integrations.go:101-108`), y `secret status` no imprime nunca el valor. Hay una prueba que cierra la lista de subcomandos: `cmd/axiom/integrations_help_test.go:25` (`sync`/`emit`/`secret`) y `cli/integrations_test.go:169`; hay que ampliarlas.

### Hub / `~/.axiom/`

- `hub.Manager` solo gestiona `workspaces.json` bajo `system.AxiomDir(home)` (= `AXIOM_STATE_DIR` o `~/.axiom`, `internal/system/user_paths.go:10-15`). Escritura atómica casera: `MkdirAll(0755)`, temporal `.tmp.<nanos>`, `WriteFile(0644)`, `Rename` con reintento remove+rename para Windows (`manager.go:91-117`). `RWMutex` por proceso, sin bloqueo entre procesos. El README declara `manager.go` como fichero duro de C-07-c (README.md:198).
- Para identidad (dato personal): mejor un fichero **nuevo** (p. ej. `~/.axiom/integrations-identity.json`) con permisos 0600 y escritura atómica reutilizando `filemerge.WriteFileAtomicMode(path, content, 0o600)` (`internal/components/filemerge/writer.go:102`, ya usado para ficheros con OAuth). Alternativa: ampliar `HubConfig` (mezcla identidad con el catálogo, 0644). En Windows el modo 0600 no se aplica realmente (ACL heredada del perfil).

### Patrones de cliente HTTP en el repo

- `internal/update/github.go:20`: `var httpClient = &http.Client{Timeout: 5s}` global, `NewRequestWithContext` (`:209`); URL parametrizable por función, no por inyección de cliente.
- `internal/autoskill/client.go:19-54`: `ClientOptions{BaseURL, HTTPClient}`, recorte de `/`, timeout 15 s por defecto, `NewRequestWithContext`; es el patrón a seguir (base URL y `*http.Client` inyectables para `httptest`).
- No hay ningún cliente con reintentos ni lectura de `Retry-After`: se escribe desde cero.

### Investigación externa (hechos vs suposiciones)

**a. `connectionData` con PAT — NO VERIFICADO oficialmente.**
- Microsoft documenta que «PATs work with most Azure DevOps REST APIs. Some APIs, including the Organizations, Profiles, and PAT Lifecycle Management APIs, require Microsoft Entra tokens» (https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate). `connectionData` no figura en la referencia REST 7.1 de learn.microsoft.com (búsqueda sin resultados); es un endpoint no documentado.
- Fuentes comunitarias (no oficiales) lo usan con Basic `:{PAT}`: `GET https://dev.azure.com/{org}/_apis/connectionData?api-version=7.1` y leen `authenticatedUser.providerDisplayName`, `authenticatedUser.properties.Account.$value` (https://github.com/pingdotgg/t3code/pull/24170, cuyo propio texto admite que «the central PAT-only success scenario is still unverified»). En Azure DevOps Server on-prem algunos clientes reportan que `7.1` da 400 y exige `7.1-preview` (https://github.com/stablyai/orca/issues/12695); fuera de alcance (on-prem excluido).
- Campos esperados (suposición, hay que validarla con un PAT real): `authenticatedUser.id` (GUID), `providerDisplayName`, `properties.Account.$value`, `subjectDescriptor`; `authorizedUser`, `instanceId`. Sin comprobación con el tenant real no se debe fijar el contrato.
- La verificación real exige un PAT de una organización de prueba: es una **pregunta para el usuario** (ver más abajo). Mientras tanto el diseño debe tener el *fallback* ya previsto: la persona elige su usuario en la lista de miembros del equipo. Ojo: `teams/{t}/members` pide scope `vso.profile` o `vso.project`, no `vso.work_write` (https://learn.microsoft.com/en-us/rest/api/azure/devops/core/teams/get-team-members-with-extended-properties?view=azure-devops-rest-7.1), así que el PAT debe llevar también `vso.project` (y `vso.work`/`vso.work_write` para el resto).

**b. Endpoints REST 7.1 (todos con `api-version=7.1`).**
- Tipos: `GET {org}/{project}/_apis/wit/workitemtypes` (scope `vso.work`).
- Estados con categoría: `GET {org}/{project}/_apis/wit/workitemtypes/{type}/states` → `[{name,color,category}]`; ejemplo oficial con categorías `Proposed`, `InProgress`, `Resolved`, `Completed` (https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/work-item-type-states/list?view=azure-devops-rest-7.1). Nota: el backlog escribe «In Progress» con espacio; el valor de la API es `InProgress`; `Removed` consta en la lista de categorías del backlog pero no en el ejemplo (por confirmar).
- Campos por tipo: `GET .../workitemtypes/{type}/fields` (no fetch específico; verificar en implementación).
- Iteraciones del equipo: `GET {org}/{project}/{team}/_apis/work/teamsettings/iterations` → `values[] {id,name,path,attributes{startDate,finishDate,timeFrame}}`; `$timeframe=current` es el único filtro (https://learn.microsoft.com/en-us/rest/api/azure/devops/work/iterations/list?view=azure-devops-rest-7.1). Ojo: la respuesta usa `values`, no `value`.
- Miembros: `GET {org}/_apis/projects/{p}/teams/{t}/members?$top&$skip` → `value[] {isTeamAdmin, identity{id,displayName,uniqueName,descriptor,…}}`; `uniqueName` marcado «deprecated» (mismo enlace anterior). Paginado con `$top/$skip`.
- Proyectos/equipos: `_apis/projects`, `_apis/projects/{p}/teams` (scope `vso.project`; sin fetch específico).
- Lotes: `POST .../_apis/wit/workitemsbatch` (no fetch específico).
- Update: `PATCH .../_apis/wit/workitems/{id}`, `Content-Type: application/json-patch+json`, `[{"op":"test","path":"/rev","value":N}, …]`; scope `vso.work_write`; parámetros `bypassRules`, `suppressNotifications`, `validateOnly` (https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/work-items/update?view=azure-devops-rest-7.1). **Código ante `/rev` obsoleto: no documentado por Microsoft.** Una referencia no oficial dice 412 (no confirmado); se debe tratar cualquier 4xx con cuerpo `TF…` como conflicto y fijar el contrato con una prueba de integración real. Suposición, no hecho.
- Repos git: `GET {org}/{project}/_apis/git/repositories` (scope `vso.code`, no fetch; confirmar).

**c. Límites de tasa** (https://learn.microsoft.com/en-us/azure/devops/integrate/concepts/rate-limits?view=azure-devops): hecho verificado. Cabeceras `Retry-After` (segundos, RFC 6585), `X-RateLimit-Resource` (informativa, no usar para cálculo), `X-RateLimit-Delay` (segundos con milésimas; es la única que se envía cuando ya se retrasa), `-Limit`, `-Remaining`, `-Reset` (epoch Unix), `-Cost`. Texto literal: «Honor the Retry-After header … The response still returns HTTP 200, so retry logic isn't required.» Bloqueo: HTTP 429 con mensaje `TF400733`. Retardos de hasta 30 s por petición; umbral 200 TSTU en ventana deslizante de 5 min. 503: no documentado en esa página (tratarlo como transitorio con retroceso exponencial: decisión de diseño).

**d. ArtifactLink de rama.** Documentado: el tipo de enlace «Branch» (toolType `Git`, artifactType `Branch`) y que `rel: "ArtifactLink"` con URL `vstfs:///…` (https://learn.microsoft.com/en-us/azure/devops/boards/queries/link-type-reference?view=azure-devops y `GET _apis/wit/artifactlinktypes`). **El formato exacto `vstfs:///Git/Ref/{projectId}%2F{repoId}%2FGB{branch}` no consta en la documentación**: sigue siendo evidencia solo del launcher (`adoWorkflowService.ts:1144-1185`, fuera de este repo). Usa IDs (GUID) de proyecto y repo, no nombres, y la rama sin `refs/heads/`. Esto lo consume C-07-e (la capacidad `vcs_links`); C-07-c solo necesita exponer `ListRepositories` (con ID) y la caché.

**e. PAT globales.** Dos fechas en la misma página de Microsoft: en el aviso «All existing global PATs stop working on January 27, 2027» y en las buenas prácticas «Global PATs stop working on December 1, 2026» (https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate). Contradicción interna, la fecha prudente es 2026-12-01 (dentro de 9 días respecto a hoy, 2026-11-22). Recomendación oficial: PAT de una sola organización y mínimos scopes; el formato es de 84 caracteres con firma `AZDO` en posiciones 76-80 (validación opcional, no obligatoria). El PAT no se debe validar por longitud estricta.

### Zonas afectadas

- `internal/integrations/azuredevops/` (nuevo): cliente HTTP, `Plugin`, `connect`, `discovery`, identidad.
- `internal/integrations/manifest/catalog.go` — añadir `azure-devops` a `builtinCatalog`.
- `internal/integrations/registry.go` — añadir a `builtinPlugins` (ojo: evitar ciclo de importación: `azuredevops` importa `integrations`; el registro importaría `azuredevops`; resolver con registro por función `init`/lista inyectada desde `cmd` o `cli`).
- `internal/integrations/plugin.go` — nuevas interfaces opcionales (`Discoverer`, `IdentityResolver`).
- `internal/hub/` — fichero de identidad por persona (nuevo, junto a `manager.go`).
- `internal/cli/integrations.go` y `cmd/axiom/main.go` — nuevo verbo `connect`/`discover`; ampliar las pruebas de ayuda.
- `internal/workspace/example_config_test.go:105-115` — actualizar la prueba «catálogo vacío».
- `openspec/specs/integration-plugins/spec.md` — delta (nuevos REQ-44.x o ampliación de REQ-43).

### Enfoques

1. **Un paquete `azuredevops` con cliente propio + interfaces opcionales + verbo `connect`/`discover` mínimo (recomendado).**
   - Pros: cumple el alcance del backlog; el cliente queda aislado y probable con `httptest`; el registro cerrado no cambia de forma.
   - Contras: hay que cuidar el ciclo de importación del registro.
   - Esfuerzo: Medio (~900-1.100 líneas con pruebas).
2. **Cliente genérico `internal/integrations/restclient` reutilizable + adaptador fino.**
   - Pros: reutilizable por futuros adaptadores.
   - Contras: abstracción prematura; hay un único tracker; sube el tamaño.
   - Esfuerzo: Medio-Alto.
3. **Solo cliente + `discovery`, dejando identidad/verbo a C-07-h.**
   - Pros: más pequeño.
   - Contras: incumple el criterio de terminado («conecta con PAT y resuelve la identidad») y C-07-d/e no tendrían identidad.
   - Esfuerzo: Bajo.

### Recomendación

Enfoque 1. Propuesta de alcance (≈1.000 líneas con pruebas):
- `client.go` (~250): `Client{BaseURL, HTTPClient, Org}`, `ClientOptions` como en `autoskill`; timeout por defecto 30 s; `do()` con `Basic` desde `secrets.Secret` (el valor codificado se añade al `secretTracker`/se redacta él mismo); reintentos acotados (p. ej. 4) para 429/503 y red, honrando `Retry-After` y `X-RateLimit-Delay` (también en 200) con tope y respeto de `ctx`; cuerpo de error acotado y saneado; `Idempotent` solo para GET y PATCH con `test /rev`.
- `discovery.go` (~200): tipos, estados (con `category`), campos, iteraciones, miembros, proyectos/equipos; paginación `$top/$skip`; caché de repos por proceso.
- `workitems.go` (~120): GET con `$expand=relations`, `workitemsbatch`, PATCH con `op: test /rev` (error tipado `ErrRevConflict`; nada se pisa).
- `identity.go` (~120): `ResolveIdentity` con `connectionData`; si falla, selección entre miembros; persistencia en el hub.
- `plugin.go` (~80): manifiesto (id `azure-devops`, capacidades `items.read`, `discovery`, `assignment`; eventos de `Deliver` mínimos; secreto `pat`; config `organization`, `project`, `team`; `Identity{Source: plugin, Secret: pat}`) y `Deliver` que, en C-07-c, solo valida la conexión (la lógica por evento es C-07-e).
- hub (~80) y CLI `connect` (~100) + pruebas (≈ el 50% del total).

### Riesgos

- `connectionData` con PAT sin verificación oficial; hace falta un PAT real de prueba (ver pregunta 1).
- Contrato 412/400 de `/rev` obsoleto no documentado: implementarlo defensivo y fijarlo con prueba contra el servidor real.
- Redacción del PAT: el sanitizador solo redacta valores resueltos por `SecretSource`; el encabezado `Authorization` en base64 no coincide. Debe registrarse también en el rastreador o no volcar nunca cuerpo/cabeceras.
- Ciclo de importaciones `integrations` ↔ `azuredevops` por `builtinPlugins`.
- Las dos fechas de retirada de PAT globales (2026-12-01 y 2027-01-27) son contradictorias en la propia Microsoft; la primera cae en 9 días.
- El scope `vso.work_write` no basta para miembros (`vso.project`/`vso.profile`) ni repos (`vso.code`): hay que declarar los scopes mínimos del PAT.
- Windows: 0600 no restringe de verdad; el PAT va al keyring, la identidad (no secreta) al fichero.
- Cortex XDR: pruebas solo con `go test` en proceso y `httptest`; no compilar binarios sueltos.

### Preguntas abiertas (solo las decide el usuario; no decididas aquí)

1. **PAT real de prueba:** ¿puede el usuario facilitar (o ejecutar él) una comprobación de `connectionData` con un PAT de una organización de prueba para cerrar «no verificado»? Si no, ¿el flujo por defecto es directamente «elegir usuario de la lista de miembros» o intenta primero `connectionData` y cae?
2. **Dónde viven `organization/project/team`:** ¿en `integrations.azure-devops` de `axiom.yaml` (versionado, común al proyecto) como propone el manifiesto, y la identidad solo en `~/.axiom/`?
3. **Identidad por persona:** ¿un único usuario ADO por persona o uno por organización? (afecta a la clave del fichero).
4. **Scopes del PAT que se pide a la persona** (`vso.work_write` + `vso.project` + `vso.code`): ¿se admite documentarlo y avisar si falta alguno?
5. **Nombre del verbo:** `axiom integrations connect|discover` frente a otro (el backlog delega los nombres exactos en la sesión).

### Listo para propuesta

Sí, con las preguntas 1-2 resueltas o con el *fallback* de miembros aceptado como camino por defecto.
