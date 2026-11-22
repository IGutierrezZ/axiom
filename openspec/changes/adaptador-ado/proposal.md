# Propuesta: adaptador Azure DevOps — cliente REST, conexión, identidad y descubrimiento (C-07-c)

## Intención

El marco de C-07-b dejó el puerto `Plugin`, el catálogo cerrado de manifiestos, el almacén de secretos y los verbos `axiom integrations`, pero el catálogo y el registro siguen vacíos (`manifest/catalog.go:10`, `registry.go:56`): Axiom no puede hablar todavía con ningún tracker. C-07-d (arranque desde un work item), C-07-e (eventos del ciclo de vida) y C-07-h (pantalla de conexión) necesitan un adaptador real que se conecte con el PAT de cada persona, sepa quién es esa persona en Azure DevOps y conozca el proceso del proyecto (tipos, estados, campos, iteraciones y miembros).

Este cambio añade ese primer adaptador con un único cliente REST robusto, que corrige los fallos del launcher (transporte triplicado, sin reintentos ni timeouts, `catch {}` silenciosos y read-modify-write sin control de concurrencia), sin emitir todavía ninguna escritura del ciclo de vida.

Éxito (criterio de terminado del backlog, `docs/plan/backlog-2026-11/07-integraciones.md:129-135`): con un servidor HTTP de prueba, el adaptador conecta con PAT y resuelve la identidad, devuelve el `discovery` completo, reintenta ante un 429 o un `Retry-After` y rechaza una escritura con `/rev` obsoleto sin pisar datos; el PAT queda en el almacén del sistema y no aparece en ningún log.

## Alcance

### Dentro del alcance

1. **Paquete nuevo `internal/integrations/azuredevops`** con el plugin `azure-devops`.
2. **Cliente REST único** (patrón `ClientOptions` de `internal/autoskill/client.go:19-54`: URL base y `*http.Client` inyectables para `httptest`):
   - `api-version=7.1` en todas las peticiones; URL base `https://dev.azure.com/{organization}` construida desde la configuración (no configurable por `axiom.yaml`: on-premises queda fuera).
   - Autenticación `Basic` con `:{PAT}` obtenido de `SecretSource` en el momento de la llamada.
   - Timeout por petición y respeto de `ctx` en todas las esperas.
   - Límites de tasa: `Retry-After` respetado también en HTTP 200 (se espera antes de la siguiente petición), 429 y 503 con reintentos acotados y retroceso exponencial con tope, errores de red transitorios con el mismo presupuesto; cabeceras `X-RateLimit-*` (`Delay`, `Remaining`, `Reset`, `Limit`) leídas y expuestas, sin usar `X-RateLimit-Resource` para cálculo.
   - Errores tipados y saneados: cuerpo de error acotado, nunca cabeceras ni la petición; 401, 203 con página de inicio de sesión y 403 producen un error de autenticación que nombra el scope probable de la operación; ningún fallo se silencia.
3. **Operaciones del cliente**, listas para que las usen C-07-d y C-07-e:
   - `GET .../_apis/wit/workitems/{id}?$expand=relations`.
   - Lotes con `POST .../_apis/wit/workitemsbatch`, troceados al máximo de la API.
   - `PATCH .../_apis/wit/workitems/{id}` con `application/json-patch+json` y concurrencia optimista obligatoria: la primera operación es siempre `{"op":"test","path":"/rev","value":N}`. Cualquier rechazo de esa prueba (412, o 400/409 con el error de test de ADO) es un conflicto tipado (`ErrRevConflict`): no se reintenta, no se relee para reaplicar y no se pisa nada.
   - Repositorios git con sus GUID (`_apis/git/repositories`) y caché por proceso.
4. **Descubrimiento** (`discovery`): tipos de work item; estados por tipo con su categoría tal como la devuelve la API (`Proposed`, `InProgress`, `Resolved`, `Completed`, `Removed`); campos por tipo; iteraciones del equipo (respuesta `values`, con fechas y `timeFrame`); miembros del equipo paginados con `$top/$skip`, con su `id` y `displayName`. Las sugerencias de asignación salen de esa lista: nunca hay emails en el código.
5. **Identidad de la persona**:
   - Al conectar se intenta `GET _apis/connectionData`. Si responde con un `authenticatedUser` utilizable, esa es la identidad.
   - Si falla o no trae un usuario utilizable, la persona elige su usuario de la lista de miembros del equipo. El verbo admite además una opción para elegirlo explícitamente sin consultar `connectionData`.
   - Se guarda **solo** en `~/.axiom/` de la persona (respeta `AXIOM_STATE_DIR` vía `system.AxiomDir`), en un fichero propio junto a `workspaces.json`, no versionado, con escritura atómica y modo 0600, **indexado por organización** (una entrada por organización ADO, porque una persona puede trabajar con varias). No guarda el PAT ni ningún secreto.
6. **Puertos opcionales en `internal/integrations`**: interfaces que un plugin implementa si las ofrece (resolución de identidad y descubrimiento), detectadas por aserción de tipo. El puerto `Plugin` (`Manifest`, `Deliver`) no cambia.
7. **Alta en el registro cerrado**: manifiesto `azure-devops` en `builtinCatalog` y plugin en `builtinPlugins`, sin ciclo de importaciones (ver «Enfoque»).
   - Configuración en `integrations.azure-devops` de `axiom.yaml` (parametrización de proyecto, sin secretos ni datos personales): `organization`, `project` y `team`.
   - Secreto `pat` (variable `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`, keyring en Windows/macOS, solo entorno en Linux), `identity: {source: plugin, secret: pat}`.
   - Capacidades honestas para lo que entrega este cambio (`items.read`, `discovery`); C-07-d y C-07-e amplían el manifiesto.
   - `Deliver` no emite escrituras del ciclo de vida en este cambio (ver «Enfoque», punto 5).
8. **Verbos nuevos**, coherentes con `axiom integrations secret --plugin <id>` (`internal/cli/integrations.go:89-120`) y genéricos sobre los puertos opcionales:
   - `axiom integrations connect --plugin azure-devops [--user <id>] [--cwd <ruta>] [--json]`: prueba la conexión, resuelve y guarda la identidad. Sin `connectionData` utilizable y sin `--user`, lista los miembros del equipo y termina con código distinto de cero pidiendo repetir con `--user` (sin prompt interactivo, para que sea determinista y usable por agentes).
   - `axiom integrations discover --plugin azure-devops [--cwd <ruta>] [--json]`: imprime el descubrimiento con un esquema versionado.
   - Un plugin sin el puerto correspondiente produce un error tipado. Ayuda de `integrations` y pruebas de ayuda ampliadas (`cmd/axiom/integrations_help_test.go:25`, `internal/cli/integrations_test.go:169`).
9. **Documentación**: bloque comentado de `integrations.azure-devops` en `axiom.example.yaml` y apartado en `docs/manual-de-inicio.md` (claves, secreto `pat`, scopes mínimos del PAT, PAT de una sola organización y los verbos `connect` y `discover`). Se actualiza la prueba que afirma el «catálogo vacío hasta C-07-c» (`internal/workspace/example_config_test.go:104-116`).
10. **Scopes mínimos documentados** por operación: `vso.work_write` (work items), `vso.project` (proyectos, equipos, miembros e iteraciones) y `vso.code` (repositorios). Ante un fallo de autorización el error nombra el scope probable; no se bloquea nada más.
11. **Pruebas solo con `go test` y `httptest`** (sin compilar binarios sueltos): servidor de prueba que simula ADO, reloj y esperas sustituibles para los reintentos, y canario del PAT.
12. **Specs vivas**: capacidad nueva y deltas (ver «Capacidades»).

### Fuera del alcance

- Entra ID en el binario: el binario usa PAT; Entra queda para el MCP del agente (C-07-g).
- Azure DevOps Server on-premises.
- Builds, pipelines, despliegues y el informe de uso de Copilot.
- Crear work items (`POST .../workitems/${type}`): lo añaden C-07-d/C-07-e cuando lo necesiten.
- Tasks por rol, estimación, resumen, imputación, enlace de rama (`ArtifactLink` `vstfs:///Git/Ref/...`) y cualquier evento del ciclo de vida hacia ADO (C-07-e).
- Verbo `axiom change start` y emisión de `change.start` (C-07-d).
- UI de conexión e integraciones (C-07-h).
- Verificar `connectionData` con un PAT real: ninguna prueba ni sesión usa credenciales reales.
- Validar el formato del PAT por longitud o firma.
- Cambios en guías de agente, skills, doctrina y goldens.

## Capacidades

### Nuevas capacidades

- `azure-devops-adapter`: plugin `azure-devops` y su manifiesto; cliente REST 7.1 (timeouts, `Retry-After` también en 200, 429/503 con reintentos acotados, `X-RateLimit-*`, errores tipados y saneados con scope probable); lectura de work items y lotes; `PATCH` con `test /rev` obligatorio y conflicto tipado sin reintento; repositorios con caché; descubrimiento (tipos, estados con categoría, campos, iteraciones y miembros); identidad por `connectionData` con caída a la elección entre miembros; fichero de identidad por persona en `~/.axiom/` indexado por organización; verbos `axiom integrations connect` y `discover`; garantía de que el PAT (y su forma `Basic` en base64) no aparece en salida, logs, errores ni ficheros. Prefijo propuesto **REQ-45**: el máximo en `openspec/specs` es REQ-43 (`integration-plugins`) y la exploración de `verify-conjunto` (C-02-b, en vuelo) contempla REQ-44. Antes de archivar se confirma en `main` el siguiente prefijo libre.

### Capacidades modificadas

- `integration-plugins`:
  - MODIFIED REQ-43.4: `DefaultRegistry()` y el catálogo por defecto dejan de estar vacíos y contienen exactamente `azure-devops`; el escenario «Registro por defecto vacío» se sustituye.
  - ADDED (familia REQ-45): puertos opcionales de identidad y descubrimiento detectados por aserción de tipo, sin cambiar `Plugin`; error tipado cuando el plugin no ofrece el puerto.
- `workspace-topology`:
  - MODIFIED REQ-43.21: la plantilla y el manual documentan `integrations.azure-devops` (claves, secreto y scopes) en lugar de «catálogo vacío hasta C-07-c».

## Enfoque

Enfoque 1 de la exploración: un paquete `azuredevops` con cliente propio, puertos opcionales y verbos mínimos. Se descartan un cliente REST genérico reutilizable (abstracción prematura con un solo tracker) y dejar la identidad para C-07-h (incumple el criterio de terminado).

1. **Cliente.** Un solo `do()` concentra autenticación, timeout, espera por `Retry-After`/`X-RateLimit-Delay`, reintentos y saneado de errores. Solo se reintentan peticiones seguras: lecturas, lotes de lectura y el `PATCH` protegido por `test /rev` (si un `PATCH` aplicado pierde la respuesta, el reintento falla por `/rev` y se informa como conflicto: nunca se pisa). Las esperas usan un reloj sustituible para probar reintentos sin dormir.
2. **Redacción del PAT.** El sanitizador de C-07-b solo redacta valores resueltos vía `SecretSource` (`sanitize.go:86-125`), y la cabecera `Authorization: Basic` lleva el base64 de `:PAT`, que no coincide con el valor rastreado. El cliente nunca vuelca cabeceras ni peticiones en errores o logs, y además el valor codificado se redacta (registrándolo en el rastreo o con redacción propia; el diseño elige). Una prueba canario busca el PAT y su base64 en stdout, stderr, errores, el fichero de identidad, `axiom.yaml` y la salida de `discover`.
3. **Registro sin ciclo de importaciones.** `azuredevops` importa `integrations`; el registro no puede importar `azuredevops`. El manifiesto vive como dato en el paquete hoja `manifest` (o en un subpaquete hoja que este importa), y la lista de plugins compilados se ensambla fuera de `integrations` (un paquete de ensamblado que importan `cli`/`cmd`) o se inyecta en él; `NewRegistry` sigue exigiendo igualdad con el catálogo. El diseño fija el mecanismo y una prueba de importaciones lo protege.
4. **Identidad.** `connect` lee `integrations.azure-devops`, obtiene el PAT del almacén, intenta `connectionData` y, si no sirve, recurre a los miembros del equipo. La entrada guardada lleva el `id` de ADO, el `displayName`, el origen (`connection_data` o `member_selection`) y la fecha. Escritura atómica con `filemerge.WriteFileAtomicMode(..., 0o600)` en un fichero nuevo bajo `system.AxiomDir`, sin tocar el esquema de `workspaces.json`. El nombre del fichero, su esquema versionado y el paquete exacto los fija el diseño.
5. **`Deliver` en C-07-c.** El validador de manifiestos exige al menos un evento (`manifest.go:176-179`). El manifiesto declara solo `change.start`, que ningún verbo emite hasta C-07-d y que `emit` rechaza a mano. Así ningún evento del ciclo de vida llega al plugin y, con el bloque configurado, los verbos actuales siguen en coste cero (REQ-43.5: ningún plugin activo escucha sus eventos). Si `Deliver` se invocara, devuelve un error permanente tipado sin efectos. Alternativa que el diseño puede preferir: relajar la exigencia de un evento (cambia REQ-43.3).
6. **Verbos genéricos.** `connect` y `discover` despachan por los puertos opcionales del plugin indicado con `--plugin`, de modo que un futuro tracker los reutilice sin verbos nuevos. Errores de configuración (`IntegrationsErr`, bloque ausente o plugin desactivado) son error tipado del verbo, no aviso.
7. **Unidades de trabajo sugeridas** (una PR, `single-pr`, un commit por tarea): cliente y límites de tasa; work items, `/rev` y repositorios; descubrimiento; puertos opcionales, manifiesto y alta en el registro; identidad y fichero en `~/.axiom/`; verbos y ayuda; documentación, canario y specs vivas.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/integrations/azuredevops/` | Nuevo | Cliente REST, work items, descubrimiento, identidad y `Plugin` |
| `internal/integrations/plugin.go` | Modificado | Puertos opcionales de identidad y descubrimiento |
| `internal/integrations/registry.go` y ensamblado de plugins | Modificado / Nuevo | Alta de `azure-devops` sin ciclo de importaciones |
| `internal/integrations/manifest/catalog.go` | Modificado | Manifiesto `azure-devops` en `builtinCatalog` |
| `internal/hub/` (junto a `manager.go`) o paquete equivalente | Nuevo | Fichero de identidad por persona en `~/.axiom/`, indexado por organización |
| `internal/cli/integrations.go` (+ pruebas) | Modificado | Verbos `connect` y `discover`, ayuda |
| `cmd/axiom/integrations_help_test.go` | Modificado | Lista cerrada de subcomandos |
| `internal/workspace/example_config_test.go`, `docs_contract_test.go` | Modificado | Fin del «catálogo vacío»; contrato del bloque `azure-devops` |
| `axiom.example.yaml`, `docs/manual-de-inicio.md` | Modificado | Bloque comentado, scopes y verbos |
| `openspec/changes/adaptador-ado/specs/{azure-devops-adapter,integration-plugins,workspace-topology}/spec.md` | Nuevo | Spec de la capacidad nueva y deltas |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| **No verificado:** que `_apis/connectionData` responda con PAT (endpoint no documentado; Microsoft dice que la API de perfiles exige Entra). Ninguna prueba usa credenciales reales | Alta | Flujo con caída a la elección entre miembros y opción `--user` explícita; el contrato de `connectionData` se prueba solo contra `httptest` y se documenta como «no verificado» |
| PAT globales retirados con dos fechas contradictorias en la misma página de Microsoft (2026-12-01 y 2027-01-27); la primera cae en 9 días | Alta | Documentar PAT de una sola organización con scopes mínimos; el error de autenticación lo recuerda. No se intenta detectar si un PAT es global |
| Fuga del PAT en la cabecera `Authorization: Basic` (base64 no coincide con el valor rastreado por el sanitizador) | Media | No volcar nunca cabeceras ni peticiones; redactar también el valor codificado; canario sobre el PAT y su base64 |
| Ciclo de importaciones `integrations` ↔ `azuredevops` por `builtinPlugins` | Media | Manifiesto como dato en el paquete hoja y ensamblado de plugins fuera de `integrations`; prueba de importaciones |
| `example_config_test.go:104-116` y REQ-43.4/43.21 afirman el «catálogo vacío»; fallan al añadir el manifiesto | Alta (segura) | Actualizarlos en la misma unidad de trabajo que el alta en el catálogo, con sus deltas MODIFIED |
| Código HTTP de `/rev` obsoleto no documentado (412 según fuentes no oficiales) | Media | Tratar como conflicto tipado cualquier rechazo de la operación `test` (412, o 400/409 con el error de test); nunca reintentar ni reaplicar |
| El validador exige al menos un evento y el plugin no debe emitir en C-07-c | Baja | Declarar solo `change.start` (no emitible hasta C-07-d) y `Deliver` permanente sin efectos; o relajar la regla en diseño |
| Scopes: `vso.work_write` no basta para miembros (`vso.project`) ni repos (`vso.code`) | Media | Documentar scopes mínimos por operación; el error de autorización nombra el scope probable |
| Categoría de estado `In Progress` (backlog) frente a `InProgress` (API); `Removed` sin ejemplo oficial | Baja | Exponer el valor de la API tal cual; pruebas con ambos |
| En Windows el modo 0600 no restringe por ACL | Baja | El fichero de identidad no contiene secretos; el PAT va al keyring |
| Tamaño por encima del M (800-1.100 líneas con pruebas) | Media | Una sola PR (`single-pr`), commit por tarea; sin cliente genérico ni creación de work items |
| Colisión del prefijo REQ-45 con otros cambios en vuelo | Baja | Confirmar el siguiente prefijo libre en `main` antes de archivar y renumerar si hace falta |

## Plan de reversión

- Revertir la PR única: desaparecen `internal/integrations/azuredevops`, el manifiesto del catálogo, el alta en el registro, los puertos opcionales y los verbos `connect` y `discover`. Las specs delta se retiran con la reversión.
- Un `axiom.yaml` que declare `integrations.azure-devops` vuelve a ser error de carga en `ParseConfig`/`LoadConfig`/`workspace validate` con el binario revertido (plugin desconocido para el catálogo vacío; `Locate` lo expone sin rechazar, REQ-43.20): hay que retirar el bloque antes o junto con la reversión.
- El fichero de identidad en `~/.axiom/` queda inerte y se puede borrar a mano. El PAT del keyring se retira antes con `axiom integrations secret delete --plugin azure-devops --name pat` o, después, desde el gestor de credenciales (servicio `axiom`).
- Sin migración de datos: este cambio no escribe en ADO ningún evento del ciclo de vida ni en las carpetas de cambio.
- Tras la reversión, C-07-d, C-07-e y C-07-h quedan bloqueados de nuevo.

## Dependencias

- C-07-b (`marco-integraciones`, ya archivado): puerto, catálogo, registro, almacén de secretos y verbos `axiom integrations`.
- Sin dependencias externas nuevas: solo `net/http` de la biblioteca estándar.
- Bloquea a C-07-d, C-07-e y C-07-h.

## Criterios de éxito

- [ ] Con un servidor `httptest` que simula ADO, `axiom integrations connect --plugin azure-devops` conecta con PAT y resuelve la identidad por `connectionData`; con `connectionData` fallido o sin usuario, lista los miembros y, con `--user`, guarda la identidad elegida. La identidad queda en `~/.axiom/` indexada por organización, con escritura atómica, y nunca en el repositorio.
- [ ] `axiom integrations discover --plugin azure-devops --json` devuelve el descubrimiento completo: tipos, estados por tipo con su categoría, campos por tipo, iteraciones y miembros del equipo (paginados).
- [ ] El cliente reintenta ante un 429 y un 503 con un número acotado de intentos, respeta `Retry-After` (también en HTTP 200 antes de la siguiente petición) y `ctx`, y aplica timeout a cada petición.
- [ ] Un `PATCH` con `/rev` obsoleto se rechaza con `ErrRevConflict`, sin reintento y sin que el servidor de prueba registre ninguna modificación.
- [ ] El PAT sale solo del almacén de secretos (keyring o `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`); un canario del PAT y de su base64 no aparece en stdout, stderr, errores, logs, el fichero de identidad, `axiom.yaml` ni la salida de `discover`.
- [ ] `integrations.azure-devops` con `organization`, `project` y `team` carga sin error; una clave desconocida es error de carga; con el bloque configurado, los verbos que despachan eventos no cambian su salida ni crean ficheros (ningún evento emitible llega al plugin).
- [ ] Un fallo de autorización (401, 203 con inicio de sesión o 403) produce un error tipado que nombra el scope probable, sin silenciarse.
- [ ] `go test ./internal/integrations/... ./internal/cli/... ./internal/workspace/... ./internal/hub/... ./cmd/axiom/...` en verde y `go test ./...` sin regresiones.

## Estimación de tamaño

M alto: unas 1.000-1.300 líneas de autoría con pruebas, más docs y specs. Una sola PR (`single-pr`) con un commit por tarea.
