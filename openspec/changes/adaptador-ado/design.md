# Diseño: adaptador Azure DevOps — cliente REST, conexión, identidad y descubrimiento (C-07-c)

> Fase `design` del cambio `adaptador-ado`. Rutas relativas a la raíz del worktree. Castellano de España; identificadores, códigos, esquemas y tokens en inglés. Entradas: `proposal.md`, `exploration.md` y las specs delta `azure-devops-adapter` (REQ-45.1 a REQ-45.16), `integration-plugins` (REQ-45.17 y MODIFIED REQ-43.4) y `workspace-topology` (MODIFIED REQ-43.21). Código de partida: `main` en `a8cb13b4` (marco C-07-b fusionado). Sin dependencias externas nuevas: solo biblioteca estándar.

## Enfoque técnico

Enfoque 1 de la propuesta: un paquete `internal/integrations/azuredevops` con un cliente REST propio, dos puertos opcionales en `internal/integrations` y dos verbos genéricos. Las piezas, de dentro hacia fuera:

1. **Manifiesto como dato en la hoja `manifest`** (`manifest.AzureDevOps()`), que alimenta a la vez el catálogo cerrado y el `Manifest()` del plugin.
2. **Ensamblado de plugins fuera del núcleo**: el paquete nuevo `internal/integrations/builtin` construye la lista cerrada (`azuredevops.New(...)`) y expone `builtin.DefaultRegistry()`. `internal/integrations` deja de tener lista propia y ofrece `ResolveRegistry(builtins)`, que respeta la costura `SetDefaultRegistryForTest`.
3. **Cliente REST** con un único `do()` que concentra URL, `api-version=7.1`, autenticación `Basic`, timeout por intento, ritmo por `Retry-After`, reintentos acotados, clasificación de respuestas y saneado de errores, con un reloj sustituible.
4. **Operaciones**: lectura de work items y lotes, `PATCH` con `test /rev` obligatorio, repositorios con caché, descubrimiento e identidad.
5. **Puertos opcionales** `IdentityResolver` y `Discoverer` en `internal/integrations`, con funciones de invocación (`integrations.ResolveIdentity`, `integrations.Discover`) que detectan el puerto por aserción de tipo, acotan y rastrean los secretos y sanean el error.
6. **Almacén de identidad** en el paquete hoja nuevo `internal/integrations/identitystore`: `~/.axiom/integration-identities.json`, esquema versionado, indexado por plugin y organización, escritura atómica 0600.
7. **Verbos** `axiom integrations connect` y `discover` en `internal/cli`, genéricos sobre los puertos.
8. **Doble de Azure DevOps** en `internal/integrations/azuredevops/adotest` (servidor `httptest` programable, reloj manual e instalación en las costuras), como `integrationstest` en C-07-b.

`Deliver` existe, pero es un error permanente sin efectos: el manifiesto solo escucha `change.start`, que nada emite todavía, así que los verbos que despachan siguen en coste cero (REQ-43.5).

## 1. Decisiones de arquitectura

### D-01. Alta del plugin sin ciclo de importaciones

**Elección.**

- El manifiesto vive como **función de datos** en la hoja: `internal/integrations/manifest/azuredevops.go` define `const AzureDevOpsID = "azure-devops"` y `func AzureDevOps() Manifest`, que devuelve un valor nuevo en cada llamada. `builtinCatalog = []Manifest{AzureDevOps()}`.
- `azuredevops.Plugin.Manifest()` devuelve `manifest.AzureDevOps()`. `NewRegistry` sigue comparando con `reflect.DeepEqual` contra `manifest.Lookup`, que devuelve un `Clone()`; `Clone` conserva los `nil` (`manifest.go:348-369`), así que la igualdad se cumple por construcción y la comprobación sigue protegiendo contra una divergencia futura.
- `internal/integrations/registry.go` elimina `builtinPlugins` y `DefaultRegistry()` y añade:

  ```go
  // ResolveRegistry devuelve el registro instalado por SetDefaultRegistryForTest o, si no hay
  // ninguno, NewRegistry(builtins...). Lo llama el ensamblado de plugins (internal/integrations/builtin).
  func ResolveRegistry(builtins []Plugin) (*Registry, error)
  ```

  `SetDefaultRegistryForTest` no cambia de nombre ni de contrato: sustituye el resultado de `builtin.DefaultRegistry()`.
- Paquete nuevo `internal/integrations/builtin` (importa `integrations` y `azuredevops`):

  ```go
  // Plugins devuelve la lista cerrada de plugins compilados en esta versión, en el orden del catálogo.
  func Plugins() []integrations.Plugin { return []integrations.Plugin{azuredevops.New(azuredevops.Options{})} }
  // DefaultRegistry es el registro de producción (REQ-43.4).
  func DefaultRegistry() (*integrations.Registry, error) { return integrations.ResolveRegistry(Plugins()) }
  ```

- `internal/cli/integrations_events.go:32` y los verbos nuevos usan `builtin.DefaultRegistry()`.
- La prueba de importaciones (`internal/integrations/imports_test.go`) añade tres reglas: (a) el paquete raíz `internal/integrations` no importa, ni directa ni transitivamente, `internal/integrations/azuredevops` ni `internal/integrations/builtin`. Se recorre el grafo de paquetes del módulo desde el directorio raíz del paquete, sin recursión de directorio y analizando solo ficheros que no son de prueba. (b) Fuera de `internal/integrations/builtin` y de `internal/integrations/azuredevops/adotest`, ningún fichero que no sea de prueba importa `azuredevops`: el ensamblado tiene un único punto. (c) `identitystore` solo importa la biblioteca estándar, `internal/system` e `internal/components/filemerge`. Una prueba de no vacuidad, como `TestImportsCheckerDetectsViolations`, fija el recorrido transitivo.

**Alternativas descartadas.**

- *Registro por `init()` en `azuredevops`* (patrón `database/sql`): el contenido de `DefaultRegistry()` dependería del grafo de enlazado, y en las pruebas de `internal/integrations` el registro saldría vacío con el catálogo lleno. El fallo de olvidar el import en blanco sería silencioso.
- *Manifiesto en un subpaquete hoja (`manifest/azuredevopsmanifest`) importado por `manifest`*: lo prohíben `TestLeafImportsOnlyStandardLibrary` y la regla «manifest solo importa la biblioteca estándar», porque la ruta del módulo contiene un punto.
- *Registrar el catálogo desde fuera (inyección en `manifest`)*: `workspace` valida `axiom.yaml` con `manifest.Catalog()` sin importar el adaptador, así que el catálogo por defecto debe estar completo sin ningún import. Inyectarlo rompería REQ-43.4 en cualquier binario o prueba que no enlace el adaptador.
- *Mantener `integrations.DefaultRegistry()` con un parámetro variádico*: las llamadas antiguas `DefaultRegistry()` seguirían compilando y devolverían un registro vacío en silencio. Al retirar el símbolo, cada llamada se revisa de forma consciente.

**Motivo.** Es la única opción en la que el catálogo (que lee `workspace`) y la lista de plugins (que lee la CLI) salen de la misma fuente sin efectos de enlazado y con el núcleo libre del adaptador. `builtin_test.go` fija que los identificadores de `builtin.DefaultRegistry()` coinciden exactamente con los de `manifest.Catalog()`. Esta prueba sustituye a `TestDefaultRegistryIsEmptyAndMatchesCatalog`. MODIFIED REQ-43.4 no fija el paquete de `DefaultRegistry()`: vive en `builtin`.

### D-02. Puertos opcionales y tipos de resultado

**Elección.** Fichero nuevo `internal/integrations/ports.go`. `Plugin` no cambia.

```go
const (
	PortIdentity  = "identity"
	PortDiscovery = "discovery"
)

// PortRequest es lo que recibe un puerto opcional: la configuración del plugin y sus secretos.
type PortRequest struct {
	Config  map[string]any // copia de integrations.<plugin>, sin `enabled`
	Secrets SecretSource   // acotada al manifiesto y rastreada para la redacción
}

type IdentityRequest struct {
	PortRequest
	User string // --user: identificador elegido de forma explícita; vacío = vía automática del plugin
}

// TrackerIdentity es la identidad de la persona en el tracker. Es serializable en JSON.
type TrackerIdentity struct {
	Account     string `json:"account"`      // clave de indexado: la organización en Azure DevOps
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Source      string `json:"source"`       // lo define el plugin: connection_data | member_selection
}

type IdentityResolver interface {
	ResolveIdentity(ctx context.Context, req IdentityRequest) (TrackerIdentity, error)
}

type Discoverer interface {
	Discover(ctx context.Context, req PortRequest) (Discovery, error)
}

const DiscoverySchemaV1 = "axiom.integration-discovery/v1"

type Discovery struct {
	Schema     string            `json:"schema"`
	Plugin     string            `json:"plugin"`
	Scope      map[string]string `json:"scope"` // en Azure DevOps: organization, project, team
	ItemTypes  []ItemType        `json:"item_types"`
	Iterations []Iteration       `json:"iterations"`
	Members    []Member          `json:"members"`
}
type ItemType struct {
	Name          string      `json:"name"`
	ReferenceName string      `json:"reference_name"`
	States        []ItemState `json:"states"`
	Fields        []ItemField `json:"fields"`
}
type ItemState struct {
	Name     string `json:"name"`
	Category string `json:"category"` // tal como lo devuelve el tracker, sin normalizar
	Color    string `json:"color,omitempty"`
}
type ItemField struct {
	ReferenceName  string `json:"reference_name"`
	Name           string `json:"name"`
	AlwaysRequired bool   `json:"always_required"`
}
type Iteration struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	StartDate  string `json:"start_date,omitempty"`  // texto de la API, sin reinterpretar
	FinishDate string `json:"finish_date,omitempty"`
	TimeFrame  string `json:"time_frame,omitempty"`  // past | current | future
}
type Member struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// Errores tipados.
var ErrPortUnsupported = errors.New("el plugin no ofrece el puerto")
type PortUnsupportedError struct{ Plugin, Port string } // Is(ErrPortUnsupported)

var ErrIdentitySelectionRequired = errors.New("hace falta elegir el usuario del tracker con --user")
type IdentitySelectionError struct {
	Plugin, Account string
	User            string   // vacío: no se indicó --user; no vacío: no coincide con ningún miembro
	Reason          string   // por qué no sirvió la vía automática (saneado)
	Candidates      []Member // miembros del equipo, con id y nombre visible
} // Is(ErrIdentitySelectionRequired)

// PortError envuelve el fallo de un puerto: Error() es el texto saneado con los secretos
// rastreados; Unwrap() conserva la causa para errors.Is y errors.As.
type PortError struct{ Plugin, Port, Message string; Err error }

// Funciones de invocación: detectan el puerto por aserción de tipo, copian la configuración,
// construyen la fuente acotada (newScopedSecrets) con un secretTracker propio, recuperan un pánico
// como error y sanean el error.
func ResolveIdentity(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store, user string) (TrackerIdentity, error)
func Discover(ctx context.Context, p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store) (Discovery, error)
```

`Discover` fija `Schema = DiscoverySchemaV1` y `Plugin = m.ID` y convierte las listas `nil` en vacías, de modo que el JSON nunca lleva `null` en listas. `ResolveIdentity` rechaza una identidad con `ID` o `Account` vacíos con un error del plugin. `Member` cumple también el papel de candidato.

**Alternativas descartadas.**

- *Ampliar `Plugin`*: lo prohíbe REQ-45.17 y obligaría a cada plugin futuro a implementar puertos que no ofrece.
- *Puertos con resultado `any` y renderizado propio del plugin*: no hay esquema estable para C-07-h ni para un segundo tracker, y el verbo genérico no puede imprimir de forma legible.
- *Construir la fuente de secretos en la CLI*: obligaría a exportar `newScopedSecrets` y el rastreador. Las funciones de invocación lo encapsulan en el mismo paquete que ya lo usa el despachador (`dispatcher.go:331`).

**Motivo.** Los tipos son neutros respecto al tracker. Solo los valores de `Source` y de `Scope` son de Azure DevOps. El paquete `integrations` ya importa `workspace` (`dispatcher.go`), así que recibir `IntegrationPluginConfig` no añade dependencias.

### D-03. Cliente REST: estructura

**Elección.** `internal/integrations/azuredevops/client.go`:

```go
const (
	DefaultEndpoint   = "https://dev.azure.com"
	APIVersion        = "7.1"
	DefaultTimeout    = 30 * time.Second  // por intento, incluida la lectura del cuerpo
	DefaultMaxAttempts = 4                 // 1 + 3 reintentos
	backoffBase       = 1 * time.Second    // 1 s, 2 s, 4 s...
	backoffCap        = 16 * time.Second
	maxRetryAfter     = 60 * time.Second   // techo de espera
	maxResponseBytes  = 16 << 20           // cuerpo de éxito
	maxErrorBodyBytes = 64 << 10           // cuerpo de error leído para clasificar
	maxErrorDetail    = 512                // runas del detalle de error que se incluye
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error // devuelve ctx.Err() si ctx vence antes
}

type Options struct {
	Endpoint    string        // solo pruebas; por defecto DefaultEndpoint (no configurable por axiom.yaml)
	HTTPClient  *http.Client  // por defecto &http.Client{} (DefaultTransport, con proxy de entorno)
	Timeout     time.Duration // por intento; por defecto DefaultTimeout
	MaxAttempts int           // por defecto DefaultMaxAttempts
	Clock       Clock         // por defecto el reloj real
}

type Target struct{ Organization, Project, Team string }

type Client struct {
	endpoint *url.URL
	http     *http.Client // copia del inyectado con CheckRedirect = no seguir
	opts     Options
	target   Target
	secrets  integrations.SecretSource
	state    *orgState    // compartido por organización durante la vida del proceso
}

type orgState struct {
	mu        sync.Mutex
	notBefore time.Time                 // ritmo por Retry-After
	rate      RateLimitInfo             // últimas X-RateLimit-* leídas
	repos     map[string][]Repository   // caché por proyecto; solo éxitos
}
```

- El `Plugin` guarda `sessions map[string]*orgState` (con `sync.Mutex`), con clave igual a la organización en minúsculas, y crea en cada llamada un `Client` ligero ligado a la `SecretSource` de esa llamada y al `orgState` de su organización. Así se cumplen a la vez «PAT resuelto en cada llamada» (REQ-45.2) y «caché por proceso» (REQ-45.8). La lectura de `organization`, `project` y `team` desde `Config` exige `string` no vacío y, si no, devuelve `ErrConfig`.
- **URL** (REQ-45.2, escenario de caracteres reservados): el endpoint se analiza una vez y cada petición compone `Path` y `RawPath` con los segmentos `[organización, proyecto?, equipo?, "_apis", ...]`, cada uno escapado con `url.PathEscape`. Nunca se usa `JoinPath` ni concatenación sin escapar, porque `JoinPath` interpretaría la `/` de un nombre como separador. Un `/` o un `?` en la organización viaja como `%2F` o `%3F` dentro del primer segmento y el host no cambia. La consulta se construye con `url.Values` y siempre lleva `api-version=7.1`.
- **Cabeceras**: `Authorization: Basic base64(":"+PAT)`, `Accept: application/json` y `User-Agent: axiom` (genérico, sin datos de la persona).
- **Redirecciones**: no se siguen (`CheckRedirect` devuelve `http.ErrUseLastResponse` sobre una copia del `*http.Client`; el inyectado no se modifica). Azure DevOps redirige al inicio de sesión cuando la credencial no vale, y seguir la redirección convertiría un fallo de autenticación en un 200 con HTML.

**Alternativas descartadas.** `Timeout` global de `http.Client` (no cubre las esperas entre reintentos ni distingue el intento del total). Un cliente por proceso con secreto fijo (incumple REQ-45.2). Reloj con `time.Sleep` directo (las pruebas dormirían; REQ-45.16).

### D-04. Política de reintentos, ritmo y límites de tasa

**Elección.** `do(ctx, request, out)` con un descriptor interno:

```go
type request struct {
	method, contentType string
	segments            []string   // tras la organización, sin escapar
	query               url.Values
	body                []byte
	scope               string     // scope probable del PAT para el error de autenticación
	operation           string     // texto para los errores: "leer el work item 42"
	retryable           bool       // GET, POST workitemsbatch y PATCH con test /rev
	revPatch            bool       // activa la clasificación de conflicto de /rev
	workItemID          int        // para NotFound y RevConflict
}
```

Algoritmo, por petición lógica:

1. Resolver el PAT con `secrets.Secret(ctx, "pat")` **una vez por petición lógica** (los reintentos lo reutilizan). Si falla, se devuelve el error sin HTTP; `*secrets.MissingError` ya nombra `azure-devops/pat` y `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT` (`secrets/store.go:58-62`).
2. **Ritmo**: si `now < state.notBefore`, `Clock.Sleep(notBefore-now)`. Si la espera supera `maxRetryAfter`, se devuelve `RateLimitError{Attempts: 0}` sin enviar nada.
3. Intento `n` (de 1 a `MaxAttempts`), con `context.WithTimeout(ctx, Timeout)` que cubre `Do` y la lectura del cuerpo.
4. Tras cada respuesta se leen `X-RateLimit-Delay` (segundos con decimales), `-Remaining`, `-Limit` (números decimales) y `-Reset` (epoch Unix) en `state.rate`, expuestas con `Client.RateLimit() RateLimitInfo`. `X-RateLimit-Resource` no se lee. Un valor ilegible deja el campo a cero: son informativas y no son un fallo. `Retry-After` (segundos enteros o fecha HTTP con `http.ParseTime`, relativa a `Clock.Now()`) en **cualquier** respuesta, incluida 200, fija `state.notBefore = now + RetryAfter`. `X-RateLimit-Delay` **no** se usa para esperar: informa del retraso que el servidor ya aplicó.
5. Se reintentan solo las peticiones `retryable` y solo ante: **429**, **503** y **error de transporte** (cualquier error de `Do` o de lectura del cuerpo mientras el `ctx` padre siga vivo, incluido el timeout del intento). Espera = `max(backoff(n), RetryAfter)`, con `backoff(n) = min(backoffBase·2^(n-1), backoffCap)`. Si `RetryAfter > maxRetryAfter`, no se reintenta: así se respeta «nunca esperar menos que `Retry-After`» sin quedarse colgado. Sin *jitter*: un solo cliente por persona y pruebas deterministas.
6. Si el `ctx` padre vence en cualquier espera o intento, se devuelve `ctx.Err()` envuelto, sin más peticiones (escenario de cancelación).
7. Agotado el presupuesto: 429 → `RateLimitError{Attempts}` (`ErrRateLimited`); 503 → `APIError{Status: 503, Attempts}` (`ErrUnavailable`); transporte → `TransportError{Attempts}` (`ErrTransport`).
8. Ni 500 ni 502 ni 504 se reintentan (la spec solo nombra 429 y 503). Así el escenario «un fallo no se cachea» de REQ-45.8 ve el 500 en la primera llamada.

**PATCH**: es `retryable`, pero solo ante 429, 503 y transporte. Si la respuesta se pierde y el reintento vuelve con rechazo de `/rev`, el conflicto lleva `AfterRetry: true` y el mensaje advierte que la modificación pudo aplicarse en un intento anterior y que hay que releer el work item. Nunca se reintenta tras un rechazo de `test`, ni tras ningún otro 4xx.

**Alternativas descartadas.** Reintentar 500 y 502 (contradice la spec y oculta fallos reales). Reintentar solo GET (dejaría el `PATCH` frágil ante cortes de red, cuando `test /rev` ya hace seguro el reintento). Esperar por `X-RateLimit-Delay` (Microsoft la describe como el retraso ya aplicado). Tope fijo que recorte `Retry-After` (incumple REQ-45.4).

### D-05. Clasificación de respuestas y errores tipados

Fichero `internal/integrations/azuredevops/errors.go`. Centinelas para `errors.Is`: `ErrAuth`, `ErrNotFound`, `ErrRateLimited`, `ErrUnavailable`, `ErrTransport`, `ErrBadRequest`, `ErrServer`, `ErrInvalidResponse`, `ErrRevConflict`, `ErrMissingRev`, `ErrConfig` y `ErrLifecycleUnsupported`. Tipos con datos: `AuthError{Status, Scope, Operation}`, `NotFoundError{Operation, WorkItemID}`, `RevConflictError{WorkItemID, Rev, AfterRetry}`, `RateLimitError{Attempts, RetryAfter}`, `TransportError{Attempts, Operation, cause}` y `APIError{Status, Attempts, Operation, Detail}`.

| Respuesta | Resultado |
|---|---|
| 2xx con `Content-Type` JSON | se decodifica; JSON inválido o cuerpo > `maxResponseBytes` → `ErrInvalidResponse` (nunca valor vacío) |
| 200 con `Content-Type: text/html` | `AuthError` (página de inicio de sesión), sin interpretar el cuerpo |
| 203 | `AuthError`, sin interpretar el cuerpo |
| 204 | éxito sin cuerpo |
| 3xx (no se siguen) | `AuthError` (redirección al inicio de sesión) |
| 401, 403 | `AuthError` con el `scope` de la petición |
| 404 | `NotFoundError` (nombra el work item si lo hay) |
| 412; o 400/409 con rechazo de `test` en una petición `revPatch` | `RevConflictError` (`ErrRevConflict`) |
| 400 restante | `APIError` (`ErrBadRequest`) |
| 429, 503 | reintento (D-04) |
| otro 4xx | `APIError` |
| 5xx restante | `APIError` (`ErrServer`) |

- **Scope probable** (REQ-45.5): work items, lotes, `PATCH`, tipos, estados y campos → `vso.work_write`; miembros, iteraciones, proyectos y equipos → `vso.project`; repositorios → `vso.code`. `connectionData` no tiene scope asociado porque su fallo nunca llega a la persona como error (D-08). Mensaje de `AuthError`: «Azure DevOps rechazó la autenticación (HTTP 401) al <operación>: el PAT debe ser de una sola organización y llevar el scope <scope>; los PAT globales se retiran». No cita fechas de retirada, porque Microsoft publica dos distintas.
- **Rechazo de `test /rev`** (`isRevTestRejection(status, apiErr)`, una sola función con tabla de pruebas): 412 siempre; 400 o 409 cuando el cuerpo JSON de error trae `typeKey` igual, sin distinguir mayúsculas, a `TestOperationFailedException`, o un `message` que contiene `test operation` sin distinguir mayúsculas. Un 400 con otro mensaje (por ejemplo `TF51535: Cannot find field`) es `ErrBadRequest`. La lista de marcas está **no verificada** contra Azure DevOps real (exploración, punto b) y vive en un único sitio para corregirla.
- **Cuerpo de error**: se leen como máximo `maxErrorBodyBytes`. Si es JSON, el detalle es su `message`; si no, el texto. Después se **redacta** (D-06), se aplana a una línea y se recorta a `maxErrorDetail` runas. El error nunca incluye cabeceras, el cuerpo de la petición ni la URL con la consulta. `TransportError` incluye el mensaje de la causa redactado (la URL no contiene el PAT, que viaja en la cabecera) y conserva la causa en `Unwrap`.

**Alternativas descartadas.** Tratar como conflicto cualquier 4xx con `TF…` (convertiría errores de validación en conflictos, en contra del escenario «Un 400 ajeno al test no es conflicto»). Exponer el cuerpo completo (riesgo de fuga y salida ilegible).

### D-06. Redacción del PAT y de su forma `Basic`

**Elección.** Redacción propia del cliente, en un único punto. `do()` construye por petición un `redactor` con el PAT, `base64(":"+PAT)` (`StdEncoding`) y `"Basic "+b64`, ordenados de más largo a más corto. Solo se redactan valores de 4 o más caracteres, como `minTrackedSecretLen`. Todo texto que entra en un error del cliente (detalle del cuerpo y mensaje de causa de transporte) pasa por él antes de guardarse. El cliente nunca vuelca la petición ni las cabeceras, ni registra en `slog`. Como segunda línea, las funciones de invocación de D-02 y el despachador sanean con `sanitizeText` los valores rastreados (el PAT).

**Alternativa descartada.** Registrar el valor codificado en el `secretTracker` del marco mediante una interfaz opcional de `SecretSource`: añade API pública para un caso que el cliente cubre por sí solo, y el rastreador solo actúa sobre errores, no sobre lo que el cliente construye.

### D-07. Work items, lotes, `PATCH` y repositorios

`workitems.go` y `repos.go`:

```go
type WorkItem struct {
	ID        int            `json:"id"`
	Rev       int            `json:"rev"`
	Fields    map[string]any `json:"fields"`
	Relations []Relation     `json:"relations,omitempty"`
	URL       string         `json:"url,omitempty"`
}
type Relation struct {
	Rel        string         `json:"rel"`
	URL        string         `json:"url"`
	Attributes map[string]any `json:"attributes,omitempty"`
}
type PatchOp struct {
	Op    string `json:"op"`    // add | replace | remove | copy | move; "test" lo pone el cliente
	Path  string `json:"path"`
	From  string `json:"from,omitempty"`
	Value any    `json:"value,omitempty"`
}
type Repository struct{ ID, Name string }

func (c *Client) GetWorkItem(ctx context.Context, id int) (WorkItem, error)
func (c *Client) GetWorkItems(ctx context.Context, ids []int) ([]WorkItem, error)
func (c *Client) UpdateWorkItem(ctx context.Context, id, rev int, ops []PatchOp) (WorkItem, error)
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error)
```

- `GetWorkItem`: `GET {org}/{project}/_apis/wit/workitems/{id}?$expand=relations`, con `scope` `vso.work_write`.
- `GetWorkItems`: lista vacía → resultado vacío sin HTTP. Troceo en lotes de **200** (máximo de `workitemsbatch`), con `POST {org}/{project}/_apis/wit/workitemsbatch` y cuerpo `{"ids":[...],"$expand":"Relations","errorPolicy":"Fail"}`. El resultado se reordena según los identificadores pedidos (mapa por `id`): no se confía en el orden de la API. Con `errorPolicy: Fail`, un identificador inexistente es `NotFoundError`, igual que la lectura individual.
- `UpdateWorkItem`: `newPatchBody(rev, ops)` es la **única** forma de construir el cuerpo. Rechaza `rev <= 0` (`ErrMissingRev`), `ops` vacías y cualquier operación `test` sobre `/rev` aportada por quien llama, y antepone siempre `{"op":"test","path":"/rev","value":rev}`. `Content-Type: application/json-patch+json`, `revPatch: true` y `retryable: true` (D-04).
- `ListRepositories`: `GET {org}/{project}/_apis/git/repositories` (`vso.code`). Consulta y escribe en `orgState.repos[project]` bajo el mutex y solo guarda los éxitos. Si dos llamadas concurrentes fallan la caché, las dos piden: se acepta, porque el resultado es idéntico.

`items.read` es la capacidad declarada. `UpdateWorkItem` queda como operación del cliente para C-07-e sin capacidad en el manifiesto, porque ningún evento la invoca en este cambio (REQ-45.1: no se declaran capacidades no entregadas).

### D-08. Descubrimiento e identidad

**Descubrimiento** (`discovery.go`, `Plugin.Discover`). Peticiones secuenciales, en este orden:

1. `GET {org}/{project}/_apis/wit/workitemtypes` → `value[] {name, referenceName}`.
2. Por tipo, en el orden de la API: `GET .../workitemtypes/{type}/states` → `value[] {name, color, category}` (la categoría se copia tal cual) y `GET .../workitemtypes/{type}/fields` → `value[] {referenceName, name, alwaysRequired}`.
3. `GET {org}/{project}/{team}/_apis/work/teamsettings/iterations`. El decodificador acepta `values` y `value` y usa la que venga con elementos, porque la documentación y algunas respuestas difieren. Si no hay ninguna, la lista es vacía y no es un error.
4. Miembros paginados (`listMembers`): `GET {org}/_apis/projects/{project}/teams/{team}/members?$top=100&$skip=n` → `value[] {identity{id, displayName}}`, sin duplicados por `id`. La paginación se detiene con una página de menos de `$top`, con una página llena que no aporta ningún `id` nuevo (servidor que ignora `$skip`) o al llegar a 1.000 páginas. Nunca se leen ni se guardan direcciones de correo (`uniqueName` no se decodifica).

`Scope = {organization, project, team}`. **Un fallo en cualquier paso hace fallar el descubrimiento entero** (D-12).

**Identidad** (`identity.go`, `Plugin.ResolveIdentity`):

- `User != ""`: `listMembers` y búsqueda por `id` sin distinguir mayúsculas (son GUID). Si coincide → `TrackerIdentity{Source: "member_selection"}`. Si no coincide → `IdentitySelectionError{User, Candidates}`. **No** se consulta `connectionData`.
- `User == ""`: `GET {org}/_apis/connectionData?api-version=7.1`. Es utilizable si `authenticatedUser.id` no está vacío y hay nombre visible (`customDisplayName` o, si no, `providerDisplayName`). En ese caso → `Source: "connection_data"`, sin consultar miembros. Si falla con cualquier error del cliente (no con la cancelación del `ctx`, que se propaga) o no es utilizable → `listMembers` → `IdentitySelectionError{Reason, Candidates}`. Si `listMembers` también falla, se devuelve ese error (por ejemplo `AuthError` con `vso.project`) con la nota de que `connectionData` tampoco sirvió.
- `Account = organization` tal como está en `axiom.yaml`. El almacén la normaliza (D-09).
- El contrato de `connectionData` está **no verificado** con un PAT real: solo lo cubren pruebas con `httptest`.

### D-09. Fichero de identidad por persona

**Elección.** Paquete hoja nuevo `internal/integrations/identitystore`:

```go
const (
	SchemaV1 = "axiom.integration-identities/v1"
	FileName = "integration-identities.json" // junto a workspaces.json
)
type Entry struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Source      string    `json:"source"`
	ResolvedAt  time.Time `json:"resolved_at"` // UTC
}
type File struct {
	Schema  string                      `json:"schema"`
	Plugins map[string]map[string]Entry `json:"plugins"` // plugin → cuenta (organización) → entrada
}
type Store struct{ Path string }

func DefaultPath() (string, error)                        // system.AxiomDir(os.UserHomeDir()) + FileName
func (s Store) Load() (File, error)                        // inexistente → File vacío; ilegible → *CorruptError
func (s Store) Get(plugin, account string) (Entry, bool, error)
func (s Store) Put(plugin, account string, e Entry) error  // Load → sustituir una entrada → escritura atómica 0600
```

- La clave de cuenta se normaliza a minúsculas, porque las organizaciones de Azure DevOps no distinguen mayúsculas. `Put` solo sustituye `Plugins[plugin][account]`, así que las demás organizaciones no cambian. `encoding/json` ordena las claves de los mapas y la salida es determinista.
- Escritura con `filemerge.WriteFileAtomicMode(path, data, 0o600)`, que crea el directorio, publica con renombrado y limpia el temporal en el fallo (`filemerge/writer.go:102`). La llamada pasa por una costura de paquete `var writeAtomic = filemerge.WriteFileAtomicMode` para las pruebas de fallo.
- JSON inválido o `schema` desconocido → `*CorruptError{Path}` (nombra el fichero) y **no se sobrescribe**. La persona lo revisa o lo borra a mano.
- No toca `workspaces.json` ni `internal/hub` (`hub/manager.go` no cambia).

**Alternativas descartadas.**

- *En `internal/hub`* (el plan cita `manager.go`): `hub.Manager` gestiona el catálogo de proyectos con 0644 y su propia escritura. Mezclaría un dato personal del tracker con el catálogo, y C-07-e, que necesita la identidad al entregar, tendría que importar `hub`, cuyo grafo es mucho mayor que el del adaptador.
- *En el paquete `azuredevops`*: el verbo es genérico y el fichero también. Un segundo tracker duplicaría el almacén.
- *Ampliar `HubConfig`*: cambia el esquema de `workspaces.json`, en contra de REQ-45.11.

No hay bloqueo entre procesos: dos `connect` simultáneos pueden perder una actualización. Es el mismo modelo que `hub.Manager` y el riesgo es bajo, porque la acción es manual y puntual.

### D-10. Verbos `connect` y `discover`

Fichero nuevo `internal/cli/integrations_ports.go`. Se registran en `RunIntegrations` (`integrations.go:136-150`), con su ayuda y con el mensaje de subcomando desconocido ampliado («los disponibles son sync, emit, secret, connect y discover»).

```
axiom integrations connect --plugin <id> [--user <id>] [--cwd <ruta>] [--json]
axiom integrations discover --plugin <id> [--cwd <ruta>] [--json]
```

Flujo común (`resolvePortTarget`):

1. Sin `--plugin` → código 2 (uso).
2. `manifest.Lookup`: un plugin desconocido es un error con la lista del catálogo (se reutiliza el texto de `resolveSecretTarget`) → código 1. Se comprueba antes de leer el workspace.
3. `locateIntegrationsWorkspace` (`IntegrationsErr` → 1).
4. Bloque ausente o plugin no declarado → «el plugin <id> no está configurado en `integrations:` de axiom.yaml» → 1. `enabled: false` → «el plugin <id> está desactivado (`enabled: false`)» → 1.
5. `builtin.DefaultRegistry()` y `Get`. Un plugin no registrado es un error → 1.
6. Invocación del puerto (`integrations.ResolveIdentity` o `integrations.Discover`). `ErrPortUnsupported` → «el plugin <id> no ofrece el puerto de identidad|descubrimiento» → 1, sin efectos, porque la aserción va antes de resolver secretos o abrir ficheros.

`connect`:

- **Éxito** → `identitystore.Store{DefaultPath()}.Put(plugin, identity.Account, Entry{..., ResolvedAt: nowUTC()})` y código 0. Salida legible: «Conectado a <plugin> (cuenta <account>) como <display_name> (id <id>), origen <source>. Identidad guardada en <ruta>.» Con `--json`, en stdout:

  ```json
  {"schema":"axiom.integrations-connect/v1","plugin":"azure-devops","account":"contoso","status":"connected",
   "identity":{"id":"…","display_name":"…","source":"connection_data","resolved_at":"…"},"identity_file":"…"}
  ```

- **`IdentitySelectionError`** → no se guarda nada y el código es 1. En modo legible, en stderr: el motivo, la lista `  <id>  <display_name>` y «Repite con `axiom integrations connect --plugin <id> --user <id>`». Con `--json`, en stdout: `{"schema":"axiom.integrations-connect/v1","plugin":…,"account":…,"status":"selection_required","user":"zzz"?,"reason":…,"candidates":[{"id","display_name"}]}`, más una línea de error en stderr. Los candidatos se imprimen desde el campo `Candidates` y no desde el mensaje, que el saneado recorta a 512 caracteres. No hay ninguna pregunta interactiva.
- **Otro error** → `Error: <mensaje saneado>` en stderr y código 1.

`discover`: sin escritura de ficheros. Con `--json`, imprime el `Discovery` tal cual (`axiom.integration-discovery/v1`), un único esquema compartido por el puerto y el verbo. En modo legible:

```
azure-devops — organization contoso, project Fabrikam, team Fabrikam Team
Tipos de work item (2):
  Bug (Microsoft.VSTS.WorkItemTypes.Bug)
    Estados: New [Proposed], Active [InProgress], Resolved [Resolved], Closed [Completed], Removed [Removed]
    Campos (n): System.Title, System.State, …
Iteraciones (2):
  Sprint 1  2026-11-01 → 2026-11-14  current
Miembros (3):
  <id>  <display_name>
```

Códigos de salida, como el resto de `axiom integrations`: 0 éxito, 1 fallo o selección pendiente, 2 uso. El `ctx` es `context.Background()`: los intentos y las esperas ya están acotados (D-04).

**Ayuda global** (`cmd/axiom/main.go:127-129`): se realinean las cinco líneas a la columna de `discover`:

```
  integrations sync      Reintenta las entregas pendientes a los trackers configurados en axiom.yaml (--change, --json)
  integrations emit      Emite a mano un evento del ciclo de vida de un cambio a los plugins configurados (--change, --event)
  integrations secret    Guarda (set, solo por stdin), consulta (status) o borra (delete) un secreto de integración
  integrations connect   Prueba la conexión con el tracker y guarda tu identidad en ~/.axiom (--plugin, --user)
  integrations discover  Imprime el proceso del tracker: tipos, estados, campos, iteraciones y miembros (--plugin, --json)
```

### D-11. Manifiesto y `Deliver`

`manifest.AzureDevOps()`:

- `Schema: SchemaV1`, `ID: "azure-devops"`, `Title: "Azure DevOps"`.
- `Capabilities: [CapItemsRead, CapDiscovery]` y `Events: [EventChangeStart]`.
- `Config`: `organization`, `project` y `team`, las tres `FieldString` y `Required: true`, con descripción. La organización es la de `dev.azure.com/<organización>`.
- `Secrets: [{Name: "pat", Required: true, Description: "PAT de una sola organización con scopes vso.work_write, vso.project y vso.code"}]`.
- `Identity: {Source: IdentityPlugin, Secret: "pat"}`.

`team` es obligatorio porque las iteraciones, los miembros y la caída de identidad lo necesitan. Un valor por defecto («<proyecto> Team») adivinaría un nombre que la persona puede haber cambiado.

`Plugin.Deliver` devuelve `integrations.Permanent(ErrLifecycleUnsupported)` («el plugin azure-devops no entrega eventos del ciclo de vida en esta versión; llegan con C-07-d y C-07-e») sin tocar la red ni el disco. **Efecto en el despacho:** `HasListeners` solo es verdadero para `change.start`, que ningún verbo emite y que `emit` rechaza a mano (`cli/integrations.go:382`). Por eso `gate record`, `branch ensure --role` y `sdd publish --archive` no construyen el evento, no leen ni escriben disco y no cambian su salida (REQ-43.5 y REQ-45.14). Si algo llegara a encolar `change.start`, `Permanent` lo deja abandonado al primer intento, sin bucle de reintentos.

**Alternativa descartada.** Relajar la exigencia de al menos un evento (`manifest.go:176-179`): cambia REQ-43.3 para un estado transitorio que C-07-d resuelve. Declarar `change.start` no cuesta nada y es honesto con el destino del plugin.

### D-12. El descubrimiento falla entero

**Se confirma.** Un descubrimiento parcial presentado como completo llevaría a C-07-h a ofrecer desplegables incompletos (por ejemplo, sin miembros), y a C-07-d o C-07-e a decidir sobre datos que faltan. El fallo nombra la operación y el scope (escenario de 403 en miembros → `vso.project`), lo que basta para corregir el PAT y repetir. **Descartado:** un resultado parcial con `warnings[]`, porque complica el esquema, obliga a cada consumidor a interpretarlo y no tiene ningún consumidor que lo pida.

## 2. Flujo de datos

```
axiom integrations connect|discover --plugin azure-devops
   │
   ├─ manifest.Lookup ─────────────── catálogo cerrado (manifest.AzureDevOps())
   ├─ workspace.Locate ─────────────── integrations.azure-devops {organization, project, team}
   ├─ builtin.DefaultRegistry() ───── integrations.ResolveRegistry(builtin.Plugins())
   └─ integrations.ResolveIdentity | Discover
          │  aserción de tipo (IdentityResolver | Discoverer) → PortUnsupportedError
          │  newScopedSecrets(manifest, secrets.Default(), tracker)
          ▼
      azuredevops.Plugin ── orgState por organización (ritmo, X-RateLimit-*, caché de repos)
          │
          ▼
      Client.do ── PAT (SecretSource, por petición) ── Basic + api-version=7.1
          │   espera por Retry-After ─ intento con timeout ─ clasificación ─ redacción
          ▼
      https://dev.azure.com/{org}/…   (en las pruebas: adotest.Server, httptest)
          │
          ▼
   connect: identitystore.Put(~/.axiom/integration-identities.json, 0600, atómica)
   discover: stdout (axiom.integration-discovery/v1), sin escribir ficheros

Despacho de eventos (sin cambios): gate record | branch ensure --role | sdd publish --archive
   └─ HasListeners(evento) = false para todo salvo change.start → coste cero
```

## 3. Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/integrations/manifest/azuredevops.go` | Crear | `AzureDevOpsID` y `AzureDevOps()` (D-11) |
| `internal/integrations/manifest/catalog.go` | Modificar | `builtinCatalog = []Manifest{AzureDevOps()}`; comentario |
| `internal/integrations/manifest/catalog_test.go` | Modificar | El catálogo por defecto contiene exactamente `azure-devops`; la costura lo restaura |
| `internal/integrations/manifest/azuredevops_test.go` | Crear | Validación, capacidades exactas, sin capacidades de escritura, variable `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT` |
| `internal/integrations/registry.go` | Modificar | Retira `builtinPlugins` y `DefaultRegistry`; añade `ResolveRegistry` (D-01) |
| `internal/integrations/registry_test.go`, `helpers_test.go` | Modificar | `ResolveRegistry`; la prueba del registro por defecto pasa a `builtin` |
| `internal/integrations/imports_test.go` | Modificar | Reglas transitivas (a), (b) y (c) de D-01 y prueba de no vacuidad |
| `internal/integrations/event.go` | Modificar | Comentario de la línea 4 (los adaptadores reales ya existen) |
| `internal/integrations/ports.go` (+ `ports_test.go`) | Crear | Puertos, tipos, errores y funciones de invocación (D-02) |
| `internal/integrations/builtin/builtin.go` (+ `builtin_test.go`) | Crear | `Plugins()` y `DefaultRegistry()`; identificadores iguales al catálogo |
| `internal/integrations/identitystore/store.go` (+ `store_test.go`) | Crear | Fichero de identidad (D-09) |
| `internal/integrations/integrationstest/fake_test.go` | Modificar | `DefaultRegistry()` → `ResolveRegistry(nil)` |
| `internal/integrations/azuredevops/plugin.go` | Crear | `Plugin`, `New(Options)`, `Manifest`, `Deliver`, sesiones por organización, lectura de `Config` |
| `internal/integrations/azuredevops/client.go` | Crear | `Client`, `Options`, `Clock`, URL, `do()`, ritmo y reintentos (D-03, D-04) |
| `internal/integrations/azuredevops/errors.go` | Crear | Centinelas, tipos, clasificación, `isRevTestRejection`, redactor (D-05, D-06) |
| `internal/integrations/azuredevops/workitems.go`, `repos.go` | Crear | D-07 |
| `internal/integrations/azuredevops/discovery.go`, `identity.go` | Crear | D-08 |
| `internal/integrations/azuredevops/*_test.go` | Crear | Paquete externo `azuredevops_test`, con `adotest` |
| `internal/integrations/azuredevops/adotest/{server.go,clock.go,install.go}` | Crear | Servidor ADO simulado y programable, reloj manual, `Install(t, srv)` (catálogo por defecto + registro con el plugin apuntando al servidor + almacén en memoria) |
| `internal/cli/integrations.go` | Modificar | Ayuda de la familia, `switch` con `connect` y `discover`, mensaje de subcomando desconocido |
| `internal/cli/integrations_events.go` | Modificar | `builtin.DefaultRegistry()` |
| `internal/cli/integrations_ports.go` (+ `integrations_ports_test.go`) | Crear | Verbos (D-10) |
| `internal/cli/integrations_ado_canary_test.go` | Crear | Canario de extremo a extremo (REQ-45.15) |
| `internal/cli/integrations_events_test.go` | Modificar | REQ-45.14: verbos de despacho idénticos con `azure-devops` configurado |
| `cmd/axiom/main.go`, `cmd/axiom/integrations_help_test.go` | Modificar | Ayuda global de cinco verbos (D-10) |
| `internal/workspace/integrations_test.go` | Modificar | Bloque válido; `base_url` desconocida; `pat` rechazada; `jira` nombra `azure-devops` |
| `internal/workspace/example_config_test.go`, `docs_contract_test.go` | Modificar | MODIFIED REQ-43.21 |
| `axiom.example.yaml`, `docs/manual-de-inicio.md` | Modificar | Bloque `integrations.azure-devops`, secreto, scopes, PAT de una organización y verbos |
| `internal/hub/init_baseline_test.go` | Modificar | Solo el comentario de la línea 591 («catálogo vacío»); la prueba sigue válida (`azure` no es `azure-devops`) |

`internal/hub/manager.go` no cambia. No hay goldens afectados: la ayuda se cubre con pruebas de contenido.

## 4. Interfaces y contratos públicos

- `axiom.integration-discovery/v1`: `Discovery` (D-02). Es la salida de `discover --json`.
- `axiom.integrations-connect/v1`: documento del verbo (D-10).
- `axiom.integration-identities/v1`: fichero de identidad (D-09).
- Cliente para C-07-d y C-07-e: `GetWorkItem`, `GetWorkItems`, `UpdateWorkItem`, `ListRepositories`, `RateLimit()` y los errores de D-05.
- Contrato de secretos sin cambios: `pat` → `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT` y keyring en el servicio `axiom`.

## 5. Estrategia de pruebas

Solo `go test` por paquete, sin compilar binarios (Cortex XDR) y sin red real: todo va contra `adotest.Server` (`httptest`). Reloj manual: `Sleep` registra la duración, ejecuta un gancho opcional (para cancelar el `ctx` «durante» la espera), devuelve `ctx.Err()` si vence y avanza el tiempo virtual sin dormir. Ninguna prueba espera tiempo real, salvo el escenario de timeout, que usa `Timeout: 20ms` y `MaxAttempts: 1`. Las pruebas que usan costuras de paquete (`SetDefaultRegistryForTest`, `SetCatalogForTest`, `secrets.SetDefaultForTest`) no llaman a `t.Parallel()`. Pruebas de tabla con `t.Run` cuando hay variantes.

| Capa | Qué | Cómo |
|---|---|---|
| Unitaria `manifest` | REQ-45.1: validación, capacidades exactas, sin capacidades de escritura, variable de entorno; catálogo por defecto `[azure-devops]`; restauración de la costura | Tabla sobre `AzureDevOps()` y `Catalog()` |
| Unitaria `integrations` | `ResolveRegistry` con y sin costura; REQ-45.17: aserción de tipo, `PortUnsupportedError` y error saneado con un doble que filtra el secreto; `Discover` fija esquema y listas no nulas | Dobles locales en `ports_test.go` |
| Importaciones | Reglas (a), (b) y (c) de D-01; el recorrido transitivo detecta una violación sembrada en un `TempDir` | `go/parser`, solo `ImportsOnly` |
| `builtin` | MODIFIED REQ-43.4: `DefaultRegistry()` contiene exactamente `azure-devops` y coincide con `Catalog()` | Prueba directa |
| Cliente | REQ-45.2 (PAT por petición, secreto ausente sin HTTP), REQ-45.3 (`api-version`, `Basic` exacto, timeout, cancelación, escape de la organización y host intacto), REQ-45.4 (429 con `Retry-After`, `Retry-After` en 200, 503 con retroceso y tope, presupuesto agotado con intentos, `X-RateLimit-*` expuestas, `Resource` ignorada, corte de conexión), REQ-45.5 (401, 403, 203 y HTML con scope, sin reintento; 500 con cuerpo de megabytes acotado; eco del PAT y de su base64 redactado; JSON inválido) | `adotest.Server` con respuestas programadas por ruta y registro de peticiones con su tiempo virtual |
| Work items | REQ-45.6 (`$expand`, lote de 450 → 3 `POST` de 200 como máximo y orden conservado, 404, lote vacío sin HTTP); REQ-45.7 (`PATCH` vigente → rev 8; obsoleto → una sola petición y ningún cambio; 412, 400 y 409 → conflicto; 400 ajeno → `ErrBadRequest`; corte tras aplicar → conflicto con `AfterRetry` y una sola aplicación; construcción sin `rev` rechazada); REQ-45.8 (caché, fallo no cacheado) | El servidor simula `rev` y aplica `test` |
| Descubrimiento | REQ-45.9: completo, categorías `InProgress` y `Removed` intactas, paginación con 250 miembros y página de 100, sin repetidos, servidor que ignora `$skip`, iteraciones vacías y en `values` o `value`, 403 en miembros → fallo entero con `vso.project`; ningún correo en la salida | `adotest` |
| Identidad | REQ-45.10: `connectionData` válido (sin consulta de miembros), 401, `authenticatedUser` vacío, `--user` válido (sin `connectionData`), `--user` desconocido | Registro de peticiones de `adotest` |
| `identitystore` | REQ-45.11: primera escritura (modo 0600 solo fuera de Windows), dos organizaciones, sustitución, mayúsculas normalizadas, fallo de escritura (costura `writeAtomic`: el fichero anterior queda byte a byte y no aparece ningún fichero nuevo en el directorio; la limpieza del temporal la fijan las pruebas de `filemerge`, `writer_test.go:268`), JSON corrupto sin sobrescribir, `workspaces.json` intacto | `t.TempDir()` y `AXIOM_STATE_DIR` |
| CLI | REQ-45.12, REQ-45.13 y REQ-45.17: todos sus escenarios, con `RunIntegrations` en proceso; «sin escritura» por instantánea del workspace y de `AXIOM_STATE_DIR` antes y después; plugin sin puertos con `integrationstest.Fake` | `adotest.Install`, `t.Setenv` |
| CLI despacho | REQ-45.14: `gate record --decision approved` sobre `spec` con y sin el bloque → stdout, stderr y código idénticos, sin `integrations.yaml` ni outbox; `Deliver` → `IsPermanent` y cero peticiones | Arnés de `integrations_events_test.go` |
| Canario | REQ-45.15: PAT canario en el entorno; `connect` con y sin `--user`, `discover`, y servidores en modo 401, 403, 429 (presupuesto agotado) y 500; ni el canario ni su base64 en stdout, stderr, errores, fichero de identidad, `axiom.yaml` ni salida de `discover` | `integrations_ado_canary_test.go` |
| Ayuda | `axiom --help` con las cinco líneas y sin verbos inventados; `integrations --help` y `connect`/`discover --help` | `cmd/axiom/integrations_help_test.go` y `internal/cli/integrations_test.go` |
| Documentación | MODIFIED REQ-43.21: la plantilla contiene el bloque comentado; descomentado, carga con el catálogo por defecto; el manual menciona `pat`, la variable, los tres scopes, `integrations connect` e `integrations discover`; nada exige «catálogo vacío» | `example_config_test.go`, `docs_contract_test.go` |

Comprobación: `go test ./internal/integrations/... ./internal/cli/... ./internal/workspace/... ./internal/hub/... ./cmd/axiom/...` en verde y `go test ./...` sin regresiones.

## 6. Matriz de amenazas

N/A: no hay enrutamiento, órdenes de shell, subprocesos, automatización de VCS o de PR, clasificación de ficheros ejecutables ni integración de procesos. El único límite nuevo es HTTP saliente hacia Azure DevOps, cubierto por D-03 a D-06: credencial solo en la cabecera y resuelta del almacén, sin redirecciones, redacción del PAT y de su base64, errores acotados, URL con segmentos escapados y host fijo, e identidad en un fichero 0600 sin secretos.

## 7. Trazabilidad

| Requisito | Elemento de diseño |
|---|---|
| REQ-45.1 | D-11, D-01; `manifest/azuredevops.go` |
| REQ-45.2 | D-11 (claves), D-03 (URL no configurable y escapada; organización, proyecto y equipo), D-04 paso 1 (PAT por petición; secreto ausente) |
| REQ-45.3 | D-03, D-04 (timeout por intento, `ctx`, inyección) |
| REQ-45.4 | D-04 |
| REQ-45.5 | D-05, D-06 |
| REQ-45.6 | D-07 |
| REQ-45.7 | D-07 (`newPatchBody`), D-05 (`isRevTestRejection`), D-04 (reintento solo ante transporte, 429 y 503) |
| REQ-45.8 | D-07, `orgState.repos` |
| REQ-45.9 | D-08, D-12 |
| REQ-45.10 | D-08 |
| REQ-45.11 | D-09 |
| REQ-45.12 | D-10 (`connect`) |
| REQ-45.13 | D-10 (`discover`) |
| REQ-45.14 | D-11 |
| REQ-45.15 | D-06, D-10 (candidatos desde campos), canario de §5 |
| REQ-45.16 | `adotest`, reloj manual, §5 |
| REQ-45.17 | D-02, D-10 paso 6 |
| MODIFIED REQ-43.4 | D-01 |
| MODIFIED REQ-43.21 | §3 (`axiom.example.yaml`, manual y pruebas de contrato) |

## 8. Unidades de trabajo propuestas (para `tasks`)

Una sola PR (`single-pr`) con un commit por unidad. Cada unidad lleva su código, sus pruebas y sus docs.

| # | Unidad | Requisitos | Líneas estimadas (con pruebas) |
|---|---|---|---|
| 1 | Alta del plugin: manifiesto en la hoja, catálogo, `ResolveRegistry`, `builtin`, plugin con `Deliver` permanente, reglas de importaciones, pruebas del «catálogo vacío» actualizadas, pruebas del bloque en `workspace`, prueba de despacho sin cambios, y en plantilla y manual el bloque `azure-devops` con secreto y scopes | 45.1, 45.2 (carga), 45.14, MOD 43.4, MOD 43.21 (parte) | ~450 |
| 2 | Cliente REST: `Options`, `Clock`, URL, `do()`, ritmo y reintentos, clasificación, redacción; `adotest` (servidor y reloj) | 45.2 (secreto), 45.3, 45.4, 45.5, 45.15 (eco), 45.16 | ~750 |
| 3 | Work items, lotes, `PATCH` con `/rev` y repositorios con caché | 45.6, 45.7, 45.8 | ~450 |
| 4 | Puertos opcionales en `integrations` y descubrimiento del adaptador | 45.17 (núcleo), 45.9 | ~600 |
| 5 | Identidad del adaptador e `identitystore` | 45.10, 45.11 | ~450 |
| 6 | Verbos `connect` y `discover`, ayuda de familia y global, y manual y plantilla con los verbos | 45.12, 45.13, 45.17 (CLI), MOD 43.21 (resto) | ~600 |
| 7 | Canario de extremo a extremo | 45.15 | ~200 |

Total estimado: unas 3.500 líneas de autoría, alrededor del 55 % pruebas, por encima de las 1.000-1.300 de la propuesta. El grueso de la diferencia está en las pruebas de reintentos, de clasificación y del canario que exige la spec. Orden obligado: la 1 primero (el cambio de catálogo repercute en pruebas existentes); la 3 y la 4 dependen de la 2; la 5 depende de la 4 (`listMembers` y los puertos); la 6 depende de la 4 y la 5; la 7 cierra.

## 9. Migración y despliegue

Sin migración. Un `axiom.yaml` sin `integrations.azure-devops` no cambia de comportamiento. El fichero de identidad nace en el primer `connect`. Reversión: la de la propuesta. Antes de revertir hay que retirar el bloque `integrations.azure-devops`, porque con el binario revertido vuelve a ser un error de carga.

## 10. Preguntas abiertas

- [ ] **Hueco de spec (bloquea `verify` y `archive`, no esta fase):** el REQ-43.18 vivo (`openspec/specs/integration-plugins/spec.md:489`) exige «exactamente los subcomandos `sync`, `emit` y `secret`» y su escenario de ayuda global solo nombra esos tres. REQ-45.12 añade `connect` y `discover`, pero el delta de `integration-plugins` no modifica REQ-43.18: al archivar, la spec viva se contradiría. Hace falta un MODIFIED REQ-43.18 en `specs/integration-plugins/spec.md` (fase `spec`). Este diseño ya implementa los cinco subcomandos.
- [ ] **No verificado con Azure DevOps real:** `connectionData` con PAT, el código y el cuerpo del rechazo de `test /rev` (marcas de `isRevTestRejection`) y si las iteraciones llegan en `values` o en `value`. Todo está aislado en un punto y cubierto con `httptest`.
- [ ] **Scope de iteraciones:** la spec nombra `vso.project` para las iteraciones, pero la referencia REST 7.1 de *Iterations - List* indica `vso.work`. El diseño sigue la spec. Si se confirma `vso.work`, basta cambiar el `scope` de esa petición y el texto del manual.
