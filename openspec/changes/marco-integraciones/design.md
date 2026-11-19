# Diseño: marco de plugins de integración (C-07-b)

> Fase `design` del cambio `marco-integraciones`. Rutas relativas a la raíz del worktree. Castellano de España; identificadores, códigos, esquemas y tokens en inglés. Entradas: `proposal.md`, `explore.md` y las decisiones del usuario del 2026-11-18. La fase `spec` corre en paralelo: este documento no cita números de requisito nuevos (REQ-43.x) y deja a `spec` su redacción.

## 0. Decisión B-1 (resuelta): la cola de `archive` vive en la carpeta archivada

**Decisión del usuario (2026-11-19): opción B.** La operación de `archive` se encola de forma duradera en `.integrations-outbox.yaml`, en la raíz de la carpeta archivada `openspec/changes/archive/<fecha>-<cambio>/`, igual que el resto de eventos en la carpeta ordinaria. Las opciones descartadas eran A (intento inmediato sin cola) y C (cola de archivados en el git common dir).

Hechos que obligan a modificar reglas existentes (comprobados en el código):

1. `axiom sdd publish --archive` se ejecuta **después** del movimiento a la carpeta archivada y la resuelve con `publishChangeRoot` (`internal/cli/sdd_publish.go:126-132`).
2. REQ-21.18 (`openspec/specs/sdd-archive-lifecycle/spec.md:70-86`) congela todo lo que hay bajo `openspec/changes/archive/`, y `kickoff.RefuseArchivedRoot` (`internal/kickoff/archived.go:66-86`) lo aplica a todo escritor de gobernanza.
3. `docpaths.LocalState` y `LocalStateGlob` solo excluyen los ficheros con punto de la raíz de la carpeta **ordinaria** (`internal/docpaths/docpaths.go:173-189`); la prueba `docpaths_test.go:115` fija hoy que `archive/2026-01-01-x/.gates.lock` **no** es estado local, de modo que `sdd publish --archive` lo confirmaría.

Consecuencias de diseño (D-15, D-17 y D-18):

- **Inmutabilidad (MODIFIED REQ-21.18).** Lo inmutable es el contenido **versionado** del incremento archivado. El estado local con punto de la raíz de su carpeta, que nunca se versiona, no lo reabre. La única escritura nueva permitida es la de la cola de integraciones.
- **Publicación (MODIFIED REQ-27.5).** El estado local con punto de la raíz de la carpeta archivada **del cambio** tampoco es documentación: `sdd publish --archive` y las publicaciones posteriores no lo confirman. Nada que hoy se clasifique como documentación cambia, salvo esos ficheros con punto de la raíz de esa carpeta. Los ficheros con punto anidados y las carpetas archivadas de otros cambios conservan su clasificación, y `git status` no cambia.
- **Guarda acotada.** `kickoff.RefuseArchivedRoot` no cambia. Una guarda hermana, `kickoff.RefuseArchivedWriteExceptLocalState`, autoriza bajo `archive/` **solo** el nombre de la cola de integraciones y sus temporales en la raíz exacta de una carpeta archivada. `integrations.yaml` y cualquier otro nombre siguen rechazados, de modo que la regla «el evento `archive` no reescribe `integrations.yaml`» queda impuesta por mecanismo y no solo por política.
- **`sync`** recorre también las carpetas archivadas. La clave sigue siendo `archive/<cambio>/<carpeta archivada>`.

### 0.1 Requisitos MODIFIED que debe llevar la spec

La fase `spec` debe añadir dos bloques MODIFIED completos, con el texto de los bloques vigentes y estos cambios:

**Capacidad `sdd-archive-lifecycle`, «Sellado Inmutable Post-Archive y Gestión Exclusiva vía Bug o Nuevo Incremento (REQ-21.18)»**

> Una vez archivado un incremento, su alcance queda formalmente congelado. El sistema NO DEBE reabrir un incremento ya archivado para aplicar un ajuste, un comportamiento anómalo o una regresión detectados con posterioridad. Cualquier corrección posterior DEBE gestionarse mediante un ticket de bug independiente o un nuevo incremento de evolución que referencie la especificación viva ya sellada, nunca modificando directamente el contenido versionado bajo `openspec/changes/archive/`. El estado local de la carpeta archivada, es decir, los ficheros cuyo nombre empieza por punto situados directamente en su raíz, que nunca se versionan, no forma parte del contenido del incremento y su escritura no lo reabre. El único escritor de Axiom autorizado a escribir estado local en la raíz de una carpeta archivada es la cola de salida de integraciones (`.integrations-outbox.yaml` y sus temporales de escritura atómica). Ningún otro verbo DEBE escribir bajo `openspec/changes/archive/`, e `integrations.yaml` archivado NO DEBE modificarse.
>
> (Previously: prohibía modificar cualquier contenido bajo `openspec/changes/archive/`, sin distinguir el contenido versionado del estado local con punto de la raíz de la carpeta archivada.)

Se conservan sus dos escenarios («Intento de reabrir un incremento archivado se rechaza» y «Regresión post-archive se gestiona mediante bug o nuevo incremento») y se añaden dos más:

- **La cola de integraciones escribe estado local en la carpeta archivada.** DADO un incremento archivado y un plugin configurado que escucha `archive`, CUANDO `sdd publish --archive` termina sin rechazo y la entrega falla, ENTONCES existe `.integrations-outbox.yaml` en la raíz de la carpeta archivada, Y ningún fichero versionado de la carpeta archivada cambia, incluido `integrations.yaml`.
- **Ningún otro escritor abre la excepción.** DADO un incremento archivado, CUANDO un verbo de Axiom intenta escribir en su carpeta archivada cualquier fichero distinto de la cola de integraciones y sus temporales, incluidos `integrations.yaml`, `gates.yaml` u otro fichero con punto, ENTONCES se rechaza con el mensaje de REQ-21.18.

**Capacidad `sdd-docs-publication`, «Verbo `axiom sdd publish --change <x>` y Lista Cerrada de Rutas de Documentación (REQ-27.5)»**

> La CLI DEBE exponer `axiom sdd publish --change <nombre>` con `--cwd` y `--json`, como subcomando de `axiom sdd` y no de `axiom branch`. La orden DEBE operar únicamente sobre las **rutas de documentación**, una lista cerrada formada por: `openspec/changes/<x>/**`; la carpeta de `openspec/changes/archive/**` que corresponde al cambio; `openspec/specs/**`; `openspec/INDEX.md`; y las rutas de contexto técnico mantenido declaradas en `axiom.yaml` (`technical_context.maintained`). Las rutas de `technical_context.reference` y `odd/tasks/**` NO DEBEN pertenecer nunca a la lista. Tampoco pertenece a la lista el **estado local** del cambio: los ficheros cuyo nombre empieza por punto situados directamente en la raíz de `openspec/changes/<x>/` o en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<x>/`). `publish` NUNCA DEBE confirmarlos, ni con `--archive` ni en publicaciones posteriores. Los ficheros con punto de subcarpetas siguen siendo documentación. Toda ruta que no esté en la lista es **código** a efectos de esta capacidad. El indicador opcional `--archive` selecciona la rama de archivado del cambio como rama asignada que `publish` espera (REQ-27.7 y REQ-27.14): la publicación posterior al archivado DEBE invocarse como `axiom sdd publish --change <x> --archive`. Sin `--change`, o con un nombre de cambio inválido, la orden DEBE rechazarse sin efectos.
>
> (Previously: la lista incluía todo `openspec/changes/<x>/**` y la carpeta archivada del cambio sin excluir explícitamente el estado local con punto de sus raíces. La exclusión de la carpeta ordinaria solo existía en la implementación y la carpeta archivada no tenía ninguna.)

Se conservan sus cinco escenarios y se añaden dos más:

- **El estado local de la raíz no se publica.** DADO `openspec/changes/x/.integrations-outbox.yaml` y `openspec/changes/archive/2026-11-19-x/.integrations-outbox.yaml`, CUANDO `publish --change x` y `publish --change x --archive` clasifican sus rutas, ENTONCES ninguno de los dos es documentación, Y ninguna publicación los confirma.
- **Un fichero con punto anidado sigue siendo documentación.** DADO `openspec/changes/archive/2026-11-19-x/specs/.nota`, CUANDO `publish --change x --archive` lo clasifica, ENTONCES es documentación del cambio.

Requisitos ADDED que también afectan a `archive` (la spec los redacta en `integration-plugins`): `archive` se encola de forma duradera en la carpeta archivada; `sync` y `emit --event archive` cubren las carpetas archivadas; y el evento `archive` nunca fusiona vínculos en `integrations.yaml`.

## 1. Enfoque técnico

Se añade un núcleo de integraciones en cuatro paquetes con dependencias en una sola dirección, siguiendo los patrones del repositorio:

```
internal/integrations/manifest      hoja pura (solo biblioteca estándar)
        ▲            ▲
        │            └──────────── internal/workspace (valida `integrations.<plugin>`)
        │
internal/integrations/secrets  ───► manifest, github.com/zalando/go-keyring
        ▲
internal/integrations          ───► manifest, secrets, workspace, kickoff (guarda D-18), reviewtransaction (tras costuras)
        ▲
internal/integrations/integrationstest (doble)   internal/cli (adaptadores de verbo)
```

- **Puerto y registro** (patrón de `internal/branch/ports.go` y `internal/agents/registry.go`): `Plugin` es una interfaz de dos métodos; `NewRegistry(plugins...)` rechaza duplicados y manifiestos inválidos; `DefaultRegistry()` construye la lista cerrada en compilación, vacía en este cambio.
- **Manifiesto** serializable en JSON (`axiom.integration-manifest/v1`) en un paquete hoja con el catálogo cerrado de manifiestos. `workspace` lo importa para validar el bloque `integrations:` sin cambiar ninguna firma (`ParseConfig`, `LoadConfig`, `Locate`).
- **Ficheros por cambio** (decisión 3): `integrations.yaml` versionado (vínculos e imputaciones) y `.integrations-outbox.yaml` local con punto, que se borra al quedar vacía. Escritura con temporal en la misma carpeta y `ReplaceFileAtomic` + `SyncReviewDirectory`, como `kickoff/ledger.go`. El **cerrojo** no vive en la carpeta del cambio sino en el git common dir (D-08), para que una cola vacía deje el árbol limpio.
- **Despacho**: un adaptador fino por verbo (`internal/cli/integrations_events.go`), una sola llamada en cada enganche. Encola de forma duradera y hace un intento inmediato acotado; el verbo origen nunca falla por la integración y su stdout no cambia.
- **Secretos**: puerto `secrets.Store`, adaptadores de entorno y keyring y una cadena entorno → keyring; tipo `Secret` redactado; los valores resueltos durante una entrega se registran en memoria para limpiar cualquier error antes de persistirlo o imprimirlo (D-12).
- **Verbos**: `axiom integrations sync|emit|secret`, más `--role` en `axiom branch ensure` (decisión 1) y la emisión desde `sdd gate record` y `sdd publish --archive` (decisión 2), con la cola de `archive` en la carpeta archivada (B-1, opción B).

## 2. Decisiones de arquitectura

### D-01. Paquete hoja `internal/integrations/manifest` con el catálogo cerrado

**Elección**: tipos `Manifest`, `ConfigField`, `SecretSpec`, `IdentitySpec`, `EventType` y `Capability`, sus listas cerradas, `Validate`, `EnvVarName` y el catálogo cerrado (`Catalog`, `Lookup`) viven en un paquete que solo importa la biblioteca estándar. `workspace` lo importa; nadie de `manifest` importa nada del módulo.
**Alternativas rechazadas**: validar solo la forma en `workspace` y las claves al cargar el registro (la clave desconocida deja de ser error de carga: incumple el criterio de terminado); pasar los esquemas por parámetro a `ParseConfig` (cambia la firma de `ParseConfig`, `LoadConfig`, `Locate` y `ReadDeclarations` y todos sus llamadores).
**Justificación**: es la única opción que hace que una clave desconocida sea error de carga en `ParseConfig`/`LoadConfig`/`workspace validate` sin tocar firmas ni crear ciclos. Una prueba de arquitectura fija que `manifest` solo importa la biblioteca estándar (§7).

### D-02. El catálogo de manifiestos es la fuente; el plugin debe coincidir con él

**Elección**: `manifest.Catalog()` devuelve la lista cerrada (vacía en este cambio). `NewRegistry` exige que `plugin.Manifest()` sea válido, que su `ID` esté en el catálogo y que sea igual (`reflect.DeepEqual`) a la entrada del catálogo. Una prueba fija que los identificadores de `Catalog()` y de `DefaultRegistry()` coinciden.
**Alternativas rechazadas**: que `workspace` lea los manifiestos del registro (obligaría a `workspace` a importar `internal/integrations` y crearía el ciclo); dos listas independientes sin comprobación (derivan).
**Justificación**: `workspace` valida contra datos puros; el registro garantiza que el código del plugin y el manifiesto publicado no divergen.

### D-03. Costuras de prueba exportadas con restauración automática

**Elección**: `manifest.SetCatalogForTest(t, manifests...)` e `integrations.SetDefaultRegistryForTest(t, plugins...)`, con la firma `t interface{ Helper(); Cleanup(func()) }` del precedente `internal/components/engram/verify.go:70-80`. `integrationstest.Install(t, plugins...)` instala ambos y un almacén de secretos en memoria (`secrets.SetDefaultForTest`). Las pruebas que las usan no llaman a `t.Parallel()` (estado de paquete); se documenta en la cabecera de `integrationstest`.
**Alternativas rechazadas**: variable de paquete no exportada (no la alcanzan las pruebas de `internal/cli`, `internal/workspace` y `internal/dashboard`); inyectar el registro en cada adaptador CLI (multiplica parámetros en `runSDDGateRecord`, solape con C-06-f y C-07-e).
**Justificación**: precedente existente en el repositorio, restauración garantizada con `Cleanup` y cero superficie en producción más allá de funciones con sufijo `ForTest`.

### D-04. Puerto `Plugin` mínimo: `Manifest` y `Deliver`

**Elección**: `Plugin{ Manifest() manifest.Manifest; Deliver(ctx, Delivery) (Outcome, error) }`. La resolución de identidad se declara en el manifiesto (`identity`) y el método, si C-07-c lo necesita, llega como interfaz opcional (`IdentityResolver`) comprobada con aserción de tipo, sin romper plugins existentes.
**Alternativas rechazadas**: `Handle` por evento con un método por tipo (acopla el puerto al catálogo de eventos: cada evento nuevo rompe todos los plugins); `ResolveIdentity` en el puerto ya (método sin llamador en este cambio).
**Justificación**: un solo punto de entrega facilita la cola (una operación = una llamada) y la serialización futura fuera de proceso.

### D-05. Bloque estricto `integrations:` en `workspace` con canal de error propio

**Elección**: `internal/workspace/integrations.go` con `decodeIntegrations(node)`, siguiendo `sdd.go` y `decodeStringKeys`: bloque nulo o que no es mapeo → error; plugin desconocido → error con línea y plugins admitidos; dentro de cada plugin, solo la clave reservada `enabled` y las claves del manifiesto, sin duplicados, con tipo estricto, enumerado y obligatorias. `Declarations.Integrations`/`IntegrationsErr`; `joinLoadErrors` incluye `IntegrationsErr`; `parseLayered` proyecta a `WorkspaceConfig.Integrations` (`yaml:"-"`); `hub/init.go` `declares()` lo conoce; `sddstatus.invalidDeclarationErrors` no cambia (no bloquea el archivado, como `git`, `workflow` y `sdd`).
**Alternativas rechazadas**: decodificar con `KnownFields` (yaml.v3 no lo ofrece en `Node.Decode`, `declarations.go:46-48`).
**Justificación**: mismo régimen que los cinco bloques existentes; `Locate` no rechaza (D-03 de `workspace`), así que el despacho comprueba `IntegrationsErr` explícitamente (D-11).

### D-06. Dos ficheros por cambio: vínculos versionados y cola local con punto

**Elección** (decisión 3): `changes/<c>/integrations.yaml` (`axiom.integrations/v1`, versionado, lo confirma `sdd publish` como cualquier documento sin punto) y `changes/<c>/.integrations-outbox.yaml` (`axiom.integrations-outbox/v1`, estado local que `docpaths.LocalState` ya excluye de la publicación). Tras el archivado, la cola de `archive` vive con el mismo nombre en la raíz de la carpeta archivada (B-1, D-17 y D-18); `integrations.yaml` archivado solo se lee. La cola se **borra** cuando queda vacía, así que el árbol solo está sucio mientras hay operaciones pendientes o abandonadas: `ensure` en `in_place` (REQ-25.9) y `done` (REQ-25.17) se niegan exactamente en ese caso, también sobre la rama de archivado, lo que protege la cola (§12). No se modifica ninguna de esas reglas.
**Alternativas rechazadas**: un solo fichero versionado con la cola dentro (cada `sync` ensucia la carpeta ya publicada y genera conflictos entre clones); ignorar la cola con `.git/info/exclude` (precedente en `internal/state/project_ignore.go`, pero `done` eliminaría el worktree con operaciones pendientes y se perderían).
**Justificación**: es el régimen de `.gates.lock` y `.axiom-instance` (ficheros con punto en la raíz de la carpeta del cambio) con un ciclo de vida que no deja residuos.

### D-07. Escritura atómica con las primitivas de `reviewtransaction` tras costuras

**Elección**: `internal/integrations/files.go` declara las variables de paquete `acquireIntegrationsLock = reviewtransaction.AcquireAuthorityFileLock`, `replaceIntegrationsFile = reviewtransaction.ReplaceFileAtomic`, `syncIntegrationsDir = reviewtransaction.SyncReviewDirectory`, `sleepBeforeLockRetry = time.Sleep` y `randomInt63n = rand.Int63n`, y una copia local del reintento con retroceso exponencial y jitter de `kickoff/ledger.go:149-190` (200 intentos, 2 ms a 100 ms). Temporales `.integrations-*.yaml` y `.integrations-outbox-*.yaml` en la misma carpeta (con punto: estado local si un fallo los deja).
**Alternativas rechazadas**: exportar el reintento de `kickoff` (acopla dos dominios; C-01-e consolidará ambas copias al extraer las primitivas); `internal/filecoord` (sin uso en producción, `explore.md` §1.5).
**Justificación**: la extracción de C-01-e es un cambio de una línea por variable.

### D-08. El cerrojo vive en el git common dir, no en la carpeta del cambio

**Elección**: `<git common dir>/axiom/integrations/v1/locks/<cambio>.lock`, resuelto sin procesos con `gitcommon.GitRootFromFiles(changeRoot)` y `gitcommon.CommonDirFromFiles` (precedente: el registro de ramas en `<common>/axiom/branch/v1/`, `internal/branch/registry/registry.go:45`). Si la carpeta del cambio no está en un repositorio git, el cerrojo cae a `<changeRoot>/.integrations.lock` (sin git no hay árbol sucio que proteger). Esa caída **no** se aplica a una carpeta archivada: allí el único nombre autorizado es la cola (D-18), así que sin git el despacho de `archive` no encola, avisa y deja la orden `emit` (en la práctica no ocurre: la rama de archivado exige git). La carpeta archivada `<fecha>-<c>` comparte el cerrojo `<c>.lock` con la ordinaria, porque la clave del cerrojo es el nombre del cambio.
**Alternativas rechazadas**: `.integrations.lock` en la carpeta del cambio como `.gates.lock` (el fichero del cerrojo nunca se borra: `storeLock.release` solo lo trunca, `internal/reviewtransaction/store_lock.go:329-346`, y borrarlo tras liberar abre una carrera de dos poseedores; quedaría un fichero sin seguimiento permanente que haría negarse a `done` aunque la cola esté vacía); compartir `.gates.lock` (mismo residuo y acopla el cerrojo al ledger); cerrojo por existencia con creación exclusiva (un proceso muerto deja un cerrojo huérfano y se aparta de la primitiva que fija la propuesta).
**Justificación**: con el cerrojo fuera del árbol, «borrar la cola al quedar vacía» deja de verdad el árbol limpio. El common dir es compartido por todos los worktrees, así que dos worktrees del mismo cambio también se serializan.

### D-09. Clave de idempotencia y huella

**Elección**: la clave identifica la operación lógica; la operación encolada es el par (plugin, clave).

| Evento | Clave | Huella |
|---|---|---|
| `gate.spec.approved` | `gate.spec.approved/<cambio>/<huella>` | `d` + 16 primeros hex de `GateRecord.ArtifactDigest`; si el digest está vacío, `o<n>`, con `n` el ordinal (desde 1) de esa aprobación entre las aprobaciones de la misma compuerta en `gates.yaml` |
| `gate.design.approved` | `gate.design.approved/<cambio>/<huella>` | ídem |
| `role.apply.approved` | `role.apply.approved/<cambio>/<rol>/<huella>` | ídem, sobre la compuerta `role-apply:<rol>` |
| `role.start` | `role.start/<cambio>/<rol>` | sin huella (decisión 1) |
| `archive` | `archive/<cambio>/<carpeta archivada>` | nombre `<fecha>-<cambio>` de la carpeta |
| `change.start` | `change.start/<cambio>` | solo catálogo (C-07-d) |

Reaprobar un artefacto cambiado produce otro digest y otra clave (operación nueva); repetir `gate record` sobre el mismo artefacto, un `sync` o un `ensure --role` repetido dan la misma clave. El encolado deduplica por (plugin, clave) mientras la operación está en la cola.
**Entrega al menos una vez**: tras una entrega con éxito la operación sale de la cola y no queda registro local de claves entregadas (guardarlo exigiría escribir `integrations.yaml` en cada entrega o mantener la cola viva, y la decisión 3 limita el fichero versionado a vínculos e imputaciones). Por eso la clave viaja en el evento y el **contrato del puerto exige que el plugin sea idempotente por clave** (el adaptador real de C-07-c la usará contra el tracker; el doble en memoria la aplica). La spec debe decir «no duplica la operación pendiente; la entrega es al menos una vez con clave estable».
**Alternativas rechazadas**: huella por marca de tiempo (cada reintento sería nuevo); solo ordinal (reaprobar sin cambios generaría operación nueva).
**Justificación**: el digest ya lo calcula `gate record` (`sdd_gate_digest.go:40-57`), es estable entre máquinas y distingue «artefacto cambiado» de «reintento».

### D-10. Política de reintento y de abandono

**Elección**:
- Intento inmediato: plazo total de **10 s** por despacho (`context.WithTimeout`), compartido por los plugins del evento, que reciben el contexto y deben respetarlo.
- `sync`: plazo de **30 s** por operación.
- Errores: transitorio por defecto; `integrations.Permanent(err)` marca uno permanente (configuración inválida, 4xx de autorización). Un pánico del plugin se recupera y cuenta como transitorio.
- `MaxAttempts = 10` (incluido el inmediato). Al alcanzarlo, o con un error permanente, la operación pasa a `abandoned` y queda en la cola, visible.
- `sync` reintenta las `pending`; `--retry-abandoned` reactiva las abandonadas (pone `attempts` a 0); `--discard <clave>` (repetible) retira una operación concreta, pendiente o abandonada. Sin retroceso temporal entre intentos: no hay demonio y `sync` lo lanza una persona o la doctrina.
- `sync` trata las colas de la carpeta ordinaria y de la carpeta archivada del cambio (D-15), con la misma política.
- `sync` termina con 0 solo si no queda ninguna operación (ni pendiente ni abandonada) en las colas tratadas.
- Una operación cuyo plugin ya no está configurado o activo queda pendiente con el motivo `plugin_not_configured` y no cuenta un intento.
**Alternativas rechazadas**: reintento infinito (una cola atascada sin señal); abandono silencioso que borra (pérdida de datos); verbo propio de inspección (fuera del alcance de la propuesta).
**Justificación**: acotado, visible y recuperable sin editar ficheros a mano.

### D-11. Despacho: coste cero, aviso y nunca fallo del verbo origen

**Elección**: `internal/cli/integrations_events.go` expone `dispatchIntegrationEvent(loc workspace.Location, changeRoot string, build func() integrations.Event)`:
1. `loc.Declarations.IntegrationsErr != nil` → una línea `Aviso:` en `integrationWarnings` (variable `io.Writer = os.Stderr`, propia, como `actorIdentityWarnings`) que nombra el error, el evento no despachado y las órdenes `axiom workspace validate` y `axiom integrations emit`; vuelve.
2. Ningún plugin activo del bloque escucha `ev.Type` → vuelve **sin leer ni escribir disco, sin resolver la identidad git y sin construir el evento** (por eso `build` es diferido).
3. Guarda A sobre `changeRoot` (`guardChangeWriter`) en contexto `ordinary`, o `archive` para la carpeta archivada; si rechaza → aviso y vuelve.
4. `Dispatcher.Dispatch` y, por cada operación no entregada, un aviso con la ruta de la cola y la orden `axiom integrations sync --cwd <ws> --change <c>`.
Ningún error sale del adaptador. El texto de los avisos ya pasa por la limpieza de secretos (D-12).
**Alternativas rechazadas**: puerto notificador dentro del motor de `ensure` (acopla el motor y su solape duro con C-05-d, C-07-e y C-07-f); fallar el verbo si la integración falla (contradice la propuesta).
**Justificación**: una sola llamada por verbo minimiza conflictos y garantiza que proyectos sin bloque `integrations` no ven ningún efecto ni coste.

### D-12. Secretos: cadena entorno → keyring, tipo redactado y limpieza por rastreo

**Elección**:
- `secrets.Store.Get(ctx, Ref) (Secret, Source, error)`; `WritableStore` añade `Set` y `Delete`.
- `Env`: lee `manifest.EnvVarName(plugin, secreto)` = `AXIOM_INTEGRATION_` + id del plugin en mayúsculas con `-`→`_` + `_` + nombre del secreto en mayúsculas (`azure-devops` + `pat` → `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`). La validación del catálogo rechaza dos pares que den la misma variable.
- `Keyring`: servicio `axiom`, cuenta `<plugin>/<secreto>`, sobre `github.com/zalando/go-keyring`.
- `Chain`: entorno no vacío gana; si no, keyring. `ErrNotFound` del keyring → `*MissingError` (nombra `axiom integrations secret set` y la variable). Cualquier otro error del keyring (sin D-Bus, plataforma no soportada) → `*KeyringUnavailableError{Ref, EnvVar, Cause}`; **nunca** cae a un fichero en texto plano. `Set` y `Delete` solo van al keyring.
- `Secret` guarda el valor en un campo no exportado; `String`, `GoString`, `Format`, `MarshalJSON`, `MarshalYAML`, `MarshalText` y `LogValue` devuelven `[REDACTED]`; el valor solo sale con `Reveal()`.
- El despachador entrega al plugin un `SecretSource` acotado a los secretos de **su** manifiesto y con rastreo: cada valor resuelto se apunta en memoria y `sanitize` reemplaza sus apariciones (≥ 4 caracteres) por `[REDACTED]` en todo texto de error antes de escribir la cola, imprimir un aviso o devolver JSON. El texto también se recorta a 512 caracteres y se aplana a una línea.
- `secret set` lee el valor **solo** de stdin (máximo 4 KiB, quita un único salto de línea final, rechaza vacío). Si stdin es un terminal (`go-isatty`, ya dependencia directa) se niega y sugiere canalizarlo (`Read-Host -MaskInput | axiom integrations secret set ...` o `printf '%s' ... |`): no hay prompt sin eco en este cambio, porque exigiría una dependencia nueva (`golang.org/x/term` no está en `go.mod`).
**Alternativas rechazadas**: fichero cifrado propio (gestión de claves sin keyring); valor por argumento o bandera (visible en la lista de procesos y en el historial); confiar solo en que los plugins no envuelvan secretos en errores (sin red de seguridad).
**Justificación**: defensa en profundidad comprobable con la prueba canario (§7).

### D-13. `--role` en `axiom branch ensure`: validación en el adaptador y códigos tipados

**Elección** (decisión 1): bandera opcional `--role <id>` en `internal/cli/branch.go`, validada **antes** de llamar a `engine.Default().Ensure`, en este orden, cada fallo como `branch.NewResult("ensure").Refused(...)` (salida 1, texto y `--json` `axiom.branch-result/v1` sin cambio de forma):

| Caso | Código | Constructor (`internal/branch/role_refusal.go`) |
|---|---|---|
| `--role` en blanco | uso, salida 2 (como `--prefix` en blanco) | — |
| `--role` con `--archive` | `plan_conflict` (existente) | `RoleArchiveConflict` |
| `--role` con `--lane odd` | `role_not_applicable` (nuevo) | `RoleNotApplicable` |
| sin `kickoff.yaml` sellado | `kickoff_not_sealed` (nuevo) | `KickoffNotSealed` |
| rol fuera del roster sellado | `role_not_in_roster` (nuevo) | `RoleNotInRoster` (detalle `roster`) |

El `kickoff.yaml` se busca, en solo lectura, en el planning home que devuelve `engine.Default().Status(ctx, dir, change, sdd)` cuando el cambio está registrado y, si no, en el de `workspace.Locate(dir)`. La pertenencia compara con `kickoff.RoleApplyGate(r.Role) == kickoff.RoleApplyGate(role)`, la misma forma canónica (minúsculas) que `gate record`. Tras un resultado con `Status == ready`, el adaptador emite `role.start` sobre `<result.PlanningHome>/changes/<c>` con la clave `role.start/<c>/<rol>`. Con cualquier otro estado no emite. Los tres códigos nuevos y sus etiquetas van a `internal/branch/types.go` y `labels.go`; `internal/branch/engine/ensure.go` no cambia ni conoce el rol. El dashboard, que llama al motor directamente, no emite.
**Alternativas rechazadas**: errores de uso con salida 2 para todos los casos (los del roster dependen del estado del cambio, no de la sintaxis, y deben ser tipados para la doctrina de C-07-e); validar después de `ensure` (incumple «antes de cualquier efecto»).
**Justificación**: rechazo tipado y sin efectos, motor intacto y contrato provisional que heredan C-07-e y C-07-f.

### D-14. Enganche en `gate record`: una llamada tras `emitActorIdentityWarning`

**Elección**: en `runSDDGateRecord` (`internal/cli/sdd_gate.go:155`), justo después de `emitActorIdentityWarning(actorWarning)` y antes de `resolveGatePublish`, una sola línea `emitGateIntegrationEvent(loc, changeRoot, parsed, record)`. Solo con `Decision == approved`: `spec` → `gate.spec.approved`, `design` → `gate.design.approved`, `role-apply:<rol>` → `role.apply.approved` (rol extraído con el helper nuevo `kickoff.RoleFromGate(key) (string, bool)`, que reutiliza `roleApplyGatePrefix`); `tasks` e `integration` no emiten. El actor es `record.Actor` (identidad git de C-07-a o `--actor`); la huella, la de D-09 (el ordinal se lee del ledger ya escrito). `axiom.sdd-gate-record/v1` no cambia. Como `gate record` no confirma nada, el `integrations.yaml` que escriba el despacho lo recoge el `sdd publish` que la orden ya calcula a continuación.
**Justificación**: en ese punto la decisión ya está en `gates.yaml` y el resto del verbo no depende del despacho.

### D-15. `sdd publish --archive` y `emit --event archive`

**Elección** (B-1, opción B): tras `publisher.Run`, si `--archive` y el resultado no es `refused`, el adaptador llama a `dispatchIntegrationEvent` sobre la carpeta archivada resuelta por `publishChangeRoot`, con `DispatchOptions{Durable: true, PersistOutcome: false}`, guarda A en contexto `archive` y clave `archive/<c>/<fecha>-<c>`. La operación se encola en `<carpeta archivada>/.integrations-outbox.yaml` (escritura autorizada solo por la guarda de D-18), se intenta de inmediato y, si falla, queda para `sync`. `integrations.yaml` archivado solo se lee, para dar los vínculos al plugin; los vínculos que devuelva la entrega de `archive` se descartan con un aviso, y la guarda de D-18 impide de todos modos escribirlo. stdout, JSON y código de salida de la publicación no cambian. Como la cola es estado local (D-17), una publicación posterior no la confirma.
`emit --event archive` exige que exista la carpeta archivada (`kickoff.ArchivedChangeRoot`) y hace lo mismo. `sync` sin `--change` recorre `changes/*/` y las carpetas `changes/archive/<fecha>-<c>/` que tengan cola; con `--change` trata la ordinaria y la archivada más reciente, si existen, cada una con la guarda A de su contexto (`ordinary` o `archive`).
**Alternativas rechazadas**: opción A (intento sin cola) y C (cola en el git common dir), por decisión del usuario; despachar antes del commit de `publish` (la carpeta ya está archivada y el evento saldría antes de saber si la publicación se rechaza).
**Justificación**: `archive` tiene la misma durabilidad que el resto de eventos y el contenido versionado del incremento archivado no cambia.

### D-17. El estado local con punto de la raíz de la carpeta archivada no se publica

**Elección**: `docpaths.Set.LocalState(rel)` devuelve verdadero también para un fichero cuyo nombre empieza por punto situado **directamente** en la raíz de la carpeta archivada **de ese cambio** (`<p>/changes/archive/<nombre>`, con `<nombre>` que cumple el `archiveName` ya existente). `LocalStateGlob() string` pasa a `LocalStateGlobs() []string`, con `<p>/changes/<x>/.*` y `<p>/changes/archive/[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]-<x>/.*`. Ambos patrones van como exclusiones `:(top,exclude,glob)` en `Commit` (`internal/sddpublish/gitport.go:50-54`, `261-270`; el puerto ya acepta varias); su único llamador, `publish.go:193`, pasa la lista. Los nombres de cambio válidos (letras, dígitos, `.`, `_`, `-`; `gitref.ValidBranchComponent`) no contienen metacaracteres de glob.
Qué **no** cambia: los ficheros con punto de subcarpetas siguen siendo documentación; las carpetas archivadas de otros cambios siguen fuera de la lista (`Root`); `Candidates()` no cambia; `git status` y las comprobaciones de árbol sucio de `ensure` y `done` no usan `docpaths`, así que nada que hoy esté limpio pasa a estar sucio. La prueba `docpaths_test.go:115` cambia su expectativa (`archive/2026-01-01-x/.gates.lock` pasa a ser estado local) y se añaden los casos anidado y de otro cambio.
**Alternativas rechazadas**: `.git/info/exclude` para la cola archivada (dejaría de proteger la cola en `done`, D-06); un glob `archive/*/.*` (cubriría carpetas archivadas de otros cambios).
**Justificación**: es el MODIFIED REQ-27.5 de §0.1 y aplica a la carpeta archivada la misma regla que ya tiene la ordinaria.

### D-18. Guarda hermana para el único escritor autorizado bajo `archive/`

**Elección**: `kickoff.RefuseArchivedRoot` no cambia, y siguen pasando por ella todos los escritores actuales. Nueva función en `internal/kickoff/archived.go`:

```go
// IntegrationsOutboxFile es el único fichero que Axiom puede escribir en una carpeta archivada
// (MODIFIED REQ-21.18). internal/integrations usa esta constante; no hay un segundo literal.
const IntegrationsOutboxFile = ".integrations-outbox.yaml"

// RefuseArchivedWriteExceptLocalState rechaza escribir name en changeRoot salvo cuando changeRoot
// NO está archivada (delega en RefuseArchivedRoot) o cuando es exactamente la raíz de una carpeta
// archivada (<workspace>/openspec/changes/archive/<AAAA-MM-DD>-<cambio>, sin anidar) y name es
// IntegrationsOutboxFile o un temporal ".integrations-outbox-*.yaml" de su escritura atómica.
// Cualquier otro nombre (integrations.yaml, gates.yaml, otro fichero con punto) o ruta con
// separador se rechaza con el mismo mensaje que RefuseArchivedRoot.
func RefuseArchivedWriteExceptLocalState(workspaceRoot, changeRoot, name string) error
```

`internal/integrations` llama a esta guarda antes de cada escritura en una carpeta del cambio: el temporal, el reemplazo y el borrado de la cola, y la escritura de `integrations.yaml`, que en una carpeta archivada siempre se rechaza. Así la excepción no queda en manos de la política del evento. Ningún otro paquete la llama; una prueba de arquitectura fija que solo `internal/integrations` la importa.
**Alternativas rechazadas**: un parámetro de excepción en `RefuseArchivedRoot` (lo expone a todos sus llamadores); una lista de nombres en `internal/integrations` (la regla de inmutabilidad dejaría de vivir en `kickoff`).
**Justificación**: excepción mínima, en un solo sitio y comprobable en tabla.

### D-16. Verbos `axiom integrations`

**Elección**: `internal/cli/integrations.go` con `RunIntegrations(args []string, stdin io.Reader, stdout, stderr io.Writer) int` y alta `case "integrations"` en `cmd/axiom/main.go`; salida 0 éxito, 1 rechazo, error o algo pendiente, 2 error de uso (convención de `axiom branch`). Bloque `integrations` inválido → error (salida 1), no aviso, porque son verbos explícitos de integración. Detalle de sintaxis y salidas en §5.6.
**Justificación**: superficie mínima que cubre la recuperación (`sync`), el respaldo (`emit`) y el ciclo de vida del secreto (`secret`).

## 3. Flujo de datos

### 3.1 Despacho desde un verbo (`gate record` aprobado, `ensure --role`)

```
verbo (escritura propia ya hecha)
  │
  └─► dispatchIntegrationEvent(loc, changeRoot, build)
        │ IntegrationsErr? ──sí──► Aviso (stderr) y fin
        │ ¿algún plugin activo escucha el evento? ──no──► fin (sin E/S)
        │ guarda A (ordinary) ──rechaza──► Aviso y fin
        ▼
      Dispatcher.Dispatch(ctx 10 s)
        ├─ cerrojo <common>/axiom/integrations/v1/locks/<c>.lock
        ├─ leer .integrations-outbox.yaml (o vacía)
        ├─ encolar (plugin, clave) si no existe ─► escribir cola (atómico)   ← duradero
        ├─ por cada operación del evento:
        │     plugin.Deliver(ctx, Delivery{Event, Config, Links, SecretSource})
        │       ├─ éxito ─► fusionar Outcome en integrations.yaml (atómico) ─► retirar operación
        │       └─ fallo ─► attempts++, last_error saneado, ¿abandoned?
        ├─ cola vacía ─► borrar .integrations-outbox.yaml ; si no ─► escribir (atómico)
        └─ liberar cerrojo
      Report ─► avisos por stderr (saneados); stdout del verbo intacto
```

Orden de escrituras ante una caída: primero `integrations.yaml`, después la cola. Una caída entre ambas reentrega la operación (al menos una vez) y la fusión de vínculos es idempotente.

### 3.2 `axiom integrations sync`

```
sync [--change c]
  ├─ IntegrationsErr? ─► error, salida 1
  ├─ carpetas: --change (ordinaria y archivada más reciente, si existen), o cada changes/<x>/
  │            y cada changes/archive/<fecha>-<x>/ con .integrations-outbox.yaml
  └─ por carpeta: guarda A (ordinary | archive) ─► cerrojo <x>.lock ─► descartar (--discard)
                 ─► reactivar (--retry-abandoned) ─► Deliver pendientes (30 s c/u)
                 ─► fusionar vínculos (solo carpeta ordinaria) ─► reescribir o borrar cola
                    (en la archivada, cada escritura pasa por RefuseArchivedWriteExceptLocalState)
  ─► informe (texto o axiom.integrations-sync/v1) ─► salida 0 solo si no queda nada
```

### 3.1.b `sdd publish --archive`

```
publish --archive ─► Publisher.Run (commit y push; la cola no existe aún o es estado local excluido)
   └─ resultado no refused ─► dispatchIntegrationEvent(loc, <carpeta archivada>, archive)
        ├─ leer integrations.yaml archivado (solo lectura) para Delivery.Links
        ├─ encolar en <carpeta archivada>/.integrations-outbox.yaml (guarda D-18)
        ├─ intento inmediato ─► éxito: retirar y borrar la cola ; fallo: queda para sync
        └─ vínculos devueltos ─► descartados con aviso (PersistOutcome: false)
```

### 3.3 Secretos en una entrega

```
plugin ─► SecretSource.Secret(ctx, "pat") ─► ¿declarado en su manifiesto? ─no─► error
                                           └► Chain: env AXIOM_INTEGRATION_<P>_<S> ─► keyring axiom/<p>/<s>
                                              └► Secret (redactado) + valor apuntado en el rastreador
error del plugin ─► sanitize(err, rastreador) ─► cola / aviso / JSON
```

## 4. Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/integrations/manifest/manifest.go` | Crear | Tipos del manifiesto, `SchemaV1`, `Validate`, `ValidationError` |
| `internal/integrations/manifest/catalog.go` | Crear | Catálogo cerrado (vacío), `Catalog`, `Lookup`, validación de catálogo (ids y variables de entorno únicos), `SetCatalogForTest` |
| `internal/integrations/manifest/events.go` | Crear | `EventType`, lista cerrada `Events()` |
| `internal/integrations/manifest/capabilities.go` | Crear | `Capability`, lista cerrada `Capabilities()` |
| `internal/integrations/manifest/envname.go` | Crear | `EnvVarName(plugin, secret)` |
| `internal/integrations/manifest/*_test.go` | Crear | Validación en tabla, JSON de ida y vuelta, prueba de hoja (solo biblioteca estándar) |
| `internal/integrations/plugin.go` | Crear | `Plugin`, `Delivery`, `Outcome`, `Link`, `TimeEntry`, `SecretSource`, `Permanent`, `ErrPermanent` |
| `internal/integrations/event.go` | Crear | `Event`, constructores de clave y huella (D-09), política por evento |
| `internal/integrations/registry.go` | Crear | `Registry`, `NewRegistry`, `ErrDuplicatePlugin`, `DefaultRegistry`, `SetDefaultRegistryForTest` |
| `internal/integrations/files.go` | Crear | Costuras de `reviewtransaction`, cerrojo en el common dir con reintento, escritura atómica |
| `internal/integrations/links.go` | Crear | `integrations.yaml`: esquema, lectura estricta, fusión idempotente, validación de enlaces |
| `internal/integrations/outbox.go` | Crear | `.integrations-outbox.yaml`: esquema, lectura estricta, encolado con deduplicación, borrado al vaciarse |
| `internal/integrations/dispatcher.go` | Crear | `Dispatcher`, `Dispatch`, `DispatchOptions`, `Report` |
| `internal/integrations/sync.go` | Crear | `Sync`, `SyncOptions`, `SyncReport` |
| `internal/integrations/sanitize.go` | Crear | Rastreador de secretos y `sanitize` |
| `internal/integrations/*_test.go` | Crear | Registro, cola, vínculos, despacho, sync, saneado, arquitectura de importaciones |
| `internal/integrations/secrets/secret.go` | Crear | Tipo `Secret` redactado |
| `internal/integrations/secrets/store.go` | Crear | `Ref`, `Source`, `Store`, `WritableStore`, errores tipados |
| `internal/integrations/secrets/env.go`, `keyring.go`, `chain.go` | Crear | Adaptadores y cadena |
| `internal/integrations/secrets/default.go` | Crear | `Default()` (cadena entorno → keyring), `SetDefaultForTest`, `Memory` (almacén en memoria para pruebas) |
| `internal/integrations/secrets/*_test.go` | Crear | Redacción en todos los verbos de `fmt`/JSON/YAML/slog, precedencia, `keyring.MockInit()`, keyring no disponible |
| `internal/integrations/integrationstest/fake.go`, `install.go` | Crear | Doble en memoria con fallos programables, idempotente por clave; `Install(t, ...)` |
| `internal/workspace/integrations.go` | Crear | `IntegrationsConfig`, `IntegrationPluginConfig`, `decodeIntegrations` |
| `internal/workspace/integrations_test.go` | Crear | Tabla de errores de carga con línea y claves admitidas |
| `internal/workspace/declarations.go` | Modificar | Campos `Integrations`/`IntegrationsErr`; rama en `readDeclarationBlocks` |
| `internal/workspace/loader.go` | Modificar | `joinLoadErrors` incluye `IntegrationsErr`; proyección en `parseLayered` |
| `internal/workspace/types.go` | Modificar | `WorkspaceConfig.Integrations` con `yaml:"-"` |
| `internal/workspace/docs_contract_test.go`, `example_config_test.go` | Modificar | Frases fijas del manual y de la plantilla para `integrations:` |
| `internal/hub/init.go` (+ prueba) | Modificar | `declares()` conoce `Integrations`/`IntegrationsErr` |
| `internal/sddstatus/*_test.go` | Modificar | Prueba de que `IntegrationsErr` no bloquea el archivado |
| `internal/kickoff/types.go` (+ prueba) | Modificar | `RoleFromGate(GateKey) (string, bool)` |
| `internal/branch/types.go`, `labels.go` | Modificar | Códigos `role_not_applicable`, `kickoff_not_sealed`, `role_not_in_roster` y etiquetas |
| `internal/branch/role_refusal.go` (+ prueba) | Crear | Constructores de los rechazos de `--role` |
| `internal/cli/branch.go` (+ pruebas) | Modificar | Bandera `--role`, validación previa, emisión de `role.start`, ayuda de `ensure` |
| `internal/cli/sdd_gate.go` | Modificar | Una línea tras `emitActorIdentityWarning` |
| `internal/cli/sdd_publish.go` | Modificar | Emisión duradera de `archive` sobre la carpeta archivada (D-15) |
| `internal/kickoff/archived.go` (+ prueba) | Modificar | `IntegrationsOutboxFile` y `RefuseArchivedWriteExceptLocalState` (D-18); `RefuseArchivedRoot` intacta |
| `internal/docpaths/docpaths.go`, `docpaths_test.go` | Modificar | `LocalState` cubre la raíz de la carpeta archivada del cambio; `LocalStateGlob` → `LocalStateGlobs` (D-17); expectativa de `docpaths_test.go:115` invertida |
| `internal/sddpublish/publish.go`, `gitport_test.go` | Modificar | Pasa `LocalStateGlobs()` a `Commit`; prueba de que la cola archivada queda sin seguimiento |
| `internal/cli/integrations_events.go` | Crear | Adaptador de despacho, `integrationWarnings`, `emitGateIntegrationEvent`, `emitRoleStartEvent`, `emitArchiveEvent` |
| `internal/cli/integrations.go` | Crear | `RunIntegrations`: `sync`, `emit`, `secret set|status|delete`, ayudas |
| `internal/cli/integrations_*_test.go` | Crear | E2E del criterio de terminado con el doble, canarios de secretos |
| `internal/sddpublish/publish_test.go` | Modificar | `integrations.yaml` se publica; la cola y sus temporales no, ni en la carpeta ordinaria ni en la archivada (`publish --archive` repetido con cola pendiente) |
| `internal/dashboard/secrets_canary_test.go` | Crear | Canario sobre las rutas GET de solo lectura y guarda de clasificación de rutas |
| `cmd/axiom/main.go` | Modificar | `case "integrations"` y tres líneas en `axiomHelpText` |
| `cmd/axiom/integrations_help_test.go` | Crear | Ayuda global y de los subcomandos |
| `cmd/axiom/branch_help_test.go` | Modificar | Ayuda de `ensure` con `--role` |
| `axiom.example.yaml` | Modificar | Bloque `integrations:` comentado |
| `docs/manual-de-inicio.md` | Modificar | Apartado `integrations:` en la sección 6 |
| `go.mod`, `go.sum` | Modificar | `github.com/zalando/go-keyring` y transitivas |
| `internal/branch/engine/ensure.go` | Sin cambios | El motor no conoce el rol (decisión 1) |

## 5. Interfaces y contratos

### 5.1 Manifiesto (`internal/integrations/manifest`)

```go
package manifest

const SchemaV1 = "axiom.integration-manifest/v1"

type EventType string

const (
	EventChangeStart        EventType = "change.start" // solo catálogo: lo dispara C-07-d
	EventGateSpecApproved   EventType = "gate.spec.approved"
	EventGateDesignApproved EventType = "gate.design.approved"
	EventRoleStart          EventType = "role.start"
	EventRoleApplyApproved  EventType = "role.apply.approved"
	EventArchive            EventType = "archive"
)

type Capability string

const (
	CapItemsRead       Capability = "items.read"
	CapItemsTransition Capability = "items.transition"
	CapItemsCreate     Capability = "items.create"
	CapRelations       Capability = "relations"
	CapAssignment      Capability = "assignment"
	CapEstimation      Capability = "estimation"
	CapTime            Capability = "time"
	CapTags            Capability = "tags"
	CapAttachments     Capability = "attachments"
	CapVCSLinks        Capability = "vcs_links"
	CapDiscovery       Capability = "discovery"
)

type FieldType string

const (
	FieldString     FieldType = "string"
	FieldBool       FieldType = "bool"
	FieldInt        FieldType = "int"
	FieldStringList FieldType = "string_list"
)

type IdentitySource string

const (
	IdentityNone     IdentitySource = "none"      // el plugin no necesita identidad de persona
	IdentityGitActor IdentitySource = "git_actor" // usa Event.Actor (identidad git de C-07-a)
	IdentityPlugin   IdentitySource = "plugin"    // la resuelve el plugin contra el tracker (C-07-c)
)

type Manifest struct {
	Schema       string         `json:"schema"`
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Capabilities []Capability   `json:"capabilities"`
	Events       []EventType    `json:"events"`
	Config       []ConfigField  `json:"config"`
	Secrets      []SecretSpec   `json:"secrets"`
	Identity     IdentitySpec   `json:"identity"`
}

type ConfigField struct {
	Key         string    `json:"key"`
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Description string    `json:"description"`
	Default     any       `json:"default,omitempty"`
	Enum        []string  `json:"enum,omitempty"`
}

type SecretSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type IdentitySpec struct {
	Source IdentitySource `json:"source"`
	Secret string         `json:"secret,omitempty"` // secreto con el que el plugin resuelve la identidad
}

func (m Manifest) Validate() error            // *ValidationError con todos los problemas
func Events() []EventType                     // lista cerrada, orden estable
func Capabilities() []Capability              // lista cerrada, orden estable
func Catalog() []Manifest                     // copia profunda
func Lookup(id string) (Manifest, bool)
func EnvVarName(pluginID, secretName string) string
func SetCatalogForTest(t interface{ Helper(); Cleanup(func()) }, manifests ...Manifest)
```

Reglas de `Validate`:

- `Schema == SchemaV1`; `ID` con `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`, como máximo 32 caracteres.
- `Capabilities` y `Events`: al menos una, sin repetidos, dentro de las listas cerradas.
- `Config`: claves `^[a-z][a-z0-9_]*$` (≤ 64), únicas, distintas de la reservada `enabled` y de los nombres de `Secrets`; ninguna clave cuyo segmento (separado por `_`) sea `token`, `secret`, `secrets`, `password`, `passwd`, `pwd`, `pat`, `credential`, `credentials` o `apikey`, ni que contenga el par `api_key`: los secretos se declaran en `Secrets`, nunca como configuración. `Type` dentro de la lista; `Default` del tipo declarado; `Enum` solo en `string` y `string_list`.
- `Secrets`: nombres `^[a-z][a-z0-9_]*$`, únicos.
- `Identity.Source` dentro de la lista; `Identity.Secret`, si está, nombra un secreto declarado.
- Catálogo: identificadores únicos y `EnvVarName` único para todo par (plugin, secreto).

Ejemplo serializado (el doble de pruebas):

```json
{
  "schema": "axiom.integration-manifest/v1",
  "id": "fake",
  "title": "Plugin de prueba en memoria",
  "capabilities": ["items.read", "items.transition"],
  "events": ["gate.spec.approved", "gate.design.approved", "role.start", "role.apply.approved", "archive"],
  "config": [{"key": "project", "type": "string", "required": true, "description": "Proyecto del tracker"}],
  "secrets": [{"name": "token", "description": "Token de acceso", "required": true}],
  "identity": {"source": "git_actor"}
}
```

### 5.2 Puerto y tipos de entrega (`internal/integrations`)

```go
package integrations

type EventType = manifest.EventType

type Event struct {
	Type           EventType `json:"type" yaml:"type"`
	Change         string    `json:"change" yaml:"change"`
	Role           string    `json:"role,omitempty" yaml:"role,omitempty"`
	Gate           string    `json:"gate,omitempty" yaml:"gate,omitempty"`                       // spec, design, role-apply:<rol>
	Actor          string    `json:"actor" yaml:"actor"`                                         // C-07-a: identidad git, --actor o "cli"
	OccurredAt     time.Time `json:"occurred_at" yaml:"occurred_at"`
	ArtifactDigest string    `json:"artifact_digest,omitempty" yaml:"artifact_digest,omitempty"`
	Fingerprint    string    `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	IdempotencyKey string    `json:"idempotency_key" yaml:"idempotency_key"`
}

type SecretSource interface {
	Secret(ctx context.Context, name string) (secrets.Secret, error) // solo secretos del manifiesto del plugin
}

type Delivery struct {
	Event   Event
	Config  map[string]any // valores de integrations.<plugin> (copia), sin `enabled`
	Links   []Link         // instantánea de integrations.yaml en el momento de la entrega
	Secrets SecretSource
}

type Outcome struct {
	Links       []Link
	TimeEntries []TimeEntry
}

type Plugin interface {
	Manifest() manifest.Manifest
	// Deliver entrega un evento. DEBE ser idempotente por Event.IdempotencyKey (entrega al
	// menos una vez), respetar ctx y no incluir valores de secretos en el error.
	Deliver(ctx context.Context, d Delivery) (Outcome, error)
}

var ErrPermanent = errors.New("fallo permanente de la integración")
func Permanent(err error) error // envuelve err con ErrPermanent

var ErrDuplicatePlugin = errors.New("plugin de integración duplicado")
func NewRegistry(plugins ...Plugin) (*Registry, error)
func (r *Registry) Get(id string) (Plugin, bool)
func (r *Registry) IDs() []string
func DefaultRegistry() (*Registry, error) // lista cerrada; vacía en este cambio
func SetDefaultRegistryForTest(t interface{ Helper(); Cleanup(func()) }, plugins ...Plugin)

type DispatchOptions struct {
	Durable        bool // encola antes de intentar (todos los eventos, incluido archive)
	PersistOutcome bool // fusiona vínculos en integrations.yaml (falso para archive)
}

type Dispatcher struct {
	Registry  *Registry
	Config    *workspace.IntegrationsConfig
	Secrets   secrets.Store
	Now       func() time.Time
	Immediate time.Duration // 10 s
	PerOp     time.Duration // 30 s, sync
}

func (d *Dispatcher) Dispatch(ctx context.Context, changeRoot string, ev Event, opts DispatchOptions) Report
func (d *Dispatcher) Sync(ctx context.Context, changeRoot string, opts SyncOptions) (SyncReport, error)
```

### 5.3 `integrations.yaml` (`axiom.integrations/v1`, versionado)

```yaml
schema: axiom.integrations/v1
change: marco-integraciones
links:
  - plugin: fake
    kind: work_item          # ^[a-z][a-z0-9_]*$, definido por el plugin
    id: "12345"              # no vacío, ≤ 128, sin caracteres de control
    url: https://tracker.example/items/12345   # opcional; http(s), sin userinfo
    role: ""                 # opcional
    source: gate.spec.approved/marco-integraciones/d0123456789abcdef
    recorded_at: 2026-11-18T22:30:00Z
time_entries:                # imputaciones; nadie las produce aún (C-07-e)
  - plugin: fake
    role: fullstack
    item_id: "12346"
    minutes: 90
    date: 2026-11-18
    source: role.apply.approved/marco-integraciones/fullstack/d...
    recorded_at: 2026-11-18T22:30:00Z
```

Lectura estricta (esquema exacto, claves desconocidas rechazadas). Fusión idempotente: un vínculo se identifica por (plugin, kind, id, role) y una imputación por (plugin, source, item_id); repetir la fusión no cambia el fichero. Un `Outcome` vacío no crea el fichero. Un enlace con `userinfo` en la URL se rechaza (error permanente de la operación) para que ninguna credencial llegue a disco versionado.

### 5.4 `.integrations-outbox.yaml` (`axiom.integrations-outbox/v1`, local)

```yaml
schema: axiom.integrations-outbox/v1
change: marco-integraciones
operations:
  - key: gate.spec.approved/marco-integraciones/d0123456789abcdef
    plugin: fake
    state: pending           # pending | abandoned
    attempts: 2
    enqueued_at: 2026-11-18T22:30:00Z
    last_attempt_at: 2026-11-18T22:31:00Z
    last_error: "timeout del tracker"   # saneado, ≤ 512 caracteres, una línea
    reason: ""               # plugin_not_configured cuando aplica
    event: { type: gate.spec.approved, change: marco-integraciones, gate: spec, actor: "Nombre <correo>", occurred_at: ..., artifact_digest: ..., fingerprint: d0123456789abcdef, idempotency_key: ... }
```

Nunca guarda secretos ni configuración del plugin (se leen de `axiom.yaml` y del almacén en cada intento). Se borra cuando `operations` queda vacía. Existe en la raíz de la carpeta ordinaria y, tras el archivado, en la raíz de la carpeta archivada, con el mismo esquema; en esta última solo hay operaciones `archive`.

### 5.5 Bloque `integrations:` de `axiom.yaml`

```yaml
integrations:
  <plugin>:            # id del catálogo de manifiestos de esta versión
    enabled: true      # reservada, booleano estricto, por defecto true
    <clave>: <valor>   # solo claves del manifiesto, con su tipo
```

Proyección en `workspace`:

```go
type IntegrationsConfig struct {
	Plugins []IntegrationPluginConfig // en orden de declaración
}

type IntegrationPluginConfig struct {
	ID      string
	Enabled bool
	Values  map[string]any // string, bool, int o []string según el manifiesto; con los Default aplicados
	Line    int
}

func (c *IntegrationsConfig) Active() []IntegrationPluginConfig
```

Errores de carga (todos acumulados con `errors.Join`, con línea y alternativas): bloque nulo o que no es mapeo; plugin desconocido («plugins admitidos: …» o «ninguno en esta versión»); valor de plugin nulo o que no es mapeo; clave desconocida («claves admitidas: enabled, …»); clave duplicada; tipo incorrecto; valor fuera del enumerado; obligatoria ausente (si `enabled` no es `false`). `integrations: {}` es válido y no configura nada.

### 5.6 Verbos

```
axiom integrations sync   [--cwd <ruta>] [--change <c>] [--retry-abandoned] [--discard <clave>]... [--json]
axiom integrations emit   [--cwd <ruta>] --change <c> --event <tipo> [--role <rol>] [--actor <id>] [--json]
axiom integrations secret set    --plugin <id> --name <secreto>          (valor por stdin)
axiom integrations secret status --plugin <id> [--name <secreto>] [--json]
axiom integrations secret delete --plugin <id> --name <secreto>
axiom branch ensure ... [--role <rol>]
```

- `emit`: `change.start` se rechaza (no se emite a mano en esta versión). Los eventos de compuerta exigen que la última decisión de esa compuerta en `gates.yaml` sea `approved` (no se fabrican aprobaciones) y toman de ese registro el digest, la huella y, si no se da `--actor`, el actor; `role.apply.approved` y `role.start` exigen `--role` del roster sellado; `archive` exige la carpeta archivada del cambio y se encola en ella (D-15). Salida 0 si se entregó todo o si ningún plugin activo escucha el evento (con un mensaje que lo dice); 1 si algo queda pendiente o abandonado. JSON `axiom.integrations-emit/v1`: `{schema, change, event, key, delivered[], pending[], abandoned[], nothing_to_do}`.
- `sync`: JSON `axiom.integrations-sync/v1`: `{schema, changes: [{change, delivered[], pending[], abandoned[], discarded[]}], remaining}`; cada operación con `key`, `plugin`, `attempts`, `last_error` (saneado), `reason`.
- `secret status`: por cada secreto del manifiesto, `source` ∈ `env`, `keyring`, `missing`, `keyring_unavailable`; nunca el valor ni su longitud. JSON `axiom.integrations-secret-status/v1`.
- `secret delete`: idempotente (sin secreto, informa y sale con 0); recuerda que una variable de entorno no se borra desde Axiom.
- Ayuda global (`axiomHelpText`, bloque de gobernanza):

```
  integrations sync    Reintenta las entregas pendientes a los trackers configurados en axiom.yaml (--change, --json)
  integrations emit    Emite a mano un evento del ciclo de vida de un cambio a los plugins configurados (--change, --event)
  integrations secret  Guarda (set, solo por stdin), consulta (status) o borra (delete) un secreto de integración
```

## 6. Política de coste cero

Sin bloque `integrations`, con `integrations: {}` o sin ningún plugin activo que escuche el evento, ningún verbo lee ni escribe `integrations.yaml`, la cola ni el cerrojo, ni resuelve la identidad git para el evento. Los goldens no cambian porque las únicas salidas nuevas son avisos por un writer propio y solo con bloque configurado. Una prueba fija que, sin bloque, la carpeta del cambio y el directorio `<common>/axiom/integrations` no existen tras `gate record`, `ensure --role` y `publish --archive`.

## 7. Estrategia de pruebas

Todo con `go test` (sin compilar binarios sueltos: el antivirus del entorno los borra). TDD según la configuración del proyecto que resuelva `sdd-apply`.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unidad | `manifest.Validate`, catálogo, `EnvVarName`, colisiones | Tabla; JSON de ida y vuelta estable |
| Unidad | Hoja pura y dirección de importaciones | `go/parser` sobre los ficheros no de prueba: `manifest` solo biblioteca estándar; `workspace` no importa `internal/integrations` salvo `manifest`; `internal/branch/**` no importa `internal/integrations`; solo `secrets` importa `go-keyring` |
| Unidad | `Registry`: duplicados, manifiesto inválido, ausente del catálogo, distinto del catálogo | Tabla con plugins de prueba |
| Unidad | `decodeIntegrations`: cada error con línea y alternativas; tipos; enumerado; obligatorias; `enabled: false`; `{}` | Tabla con YAML en línea y `SetCatalogForTest` |
| Unidad | Cola: deduplicación por (plugin, clave), estados, `MaxAttempts`, permanente, pánico, borrado al vaciarse, lectura estricta | Doble + reloj fijo + costuras de fichero |
| Unidad | Vínculos: fusión idempotente, URL con userinfo rechazada, `Outcome` vacío sin fichero, `archive` sin persistencia | Tabla |
| Unidad | Cerrojo en el common dir (clon y worktree enlazado) y caída a la carpeta sin git; contención con reintento | Repositorio temporal con `git init` y `git worktree add` |
| Unidad | `Secret` redactado en `%v %+v %#v %s %q %x`, JSON, YAML, texto y slog | Tabla |
| Unidad | Cadena: entorno gana, keyring con `keyring.MockInit()`, `MissingError`, `KeyringUnavailableError` sin caída a texto plano | `MockInit` / `MockInitWithError` (nombres por verificar, §9) |
| Unidad | `sanitize`: el canario incrustado en un error del plugin sale como `[REDACTED]` | Doble que falla con el secreto en el mensaje |
| Unidad | `kickoff.RoleFromGate`; constructores de rechazo de `--role`; etiquetas de los códigos nuevos | Tabla |
| Integración (`internal/cli`) | Criterio de terminado: el doble instalado y configurado recibe `gate.spec.approved`, `gate.design.approved` y `role.apply.approved` (`gate record`), `role.start` (`ensure --role`), `archive` (`publish --archive`) y el evento de `emit`; vínculos en `integrations.yaml` |
| Integración | `archive` duradero: con el doble fallando, `publish --archive` sale como sin integración, la cola queda en la raíz de la carpeta archivada, `integrations.yaml` archivado no cambia (byte a byte) y ningún otro fichero versionado cambia; `sync` (con y sin `--change`) la entrega y borra la cola; un `publish --archive` repetido con la cola pendiente no la confirma (`up_to_date` o commit sin ella); vínculos devueltos por `archive` descartados con aviso | `RunSDDPublish` y `RunIntegrations` en proceso sobre un repositorio temporal con carpeta archivada |
| Unidad | `RefuseArchivedWriteExceptLocalState`: carpeta ordinaria (delega), raíz archivada + cola o temporal (acepta), raíz archivada + `integrations.yaml`, `gates.yaml`, `.gates.lock` u otro fichero con punto (rechaza), subcarpeta de la archivada (rechaza), `archive/` sin carpeta de cambio (rechaza), nombre con separador o `..` (rechaza); prueba de arquitectura: solo `internal/integrations` la usa | Tabla en `internal/kickoff` |
| Unidad | `docpaths.LocalState`/`LocalStateGlobs`: raíz archivada del cambio (sí), fichero con punto anidado (no), carpeta archivada de otro cambio (no), raíz ordinaria (sí, como hoy) | Tabla en `internal/docpaths` |
| Integración | Rama de archivado: con la cola pendiente, `branch done` sobre la rama de archivado se niega con `dirty_worktree` nombrando la cola; con la cola vacía (borrada) procede | Motor de ramas con repositorio temporal |
| Regresión | Lectores de carpetas archivadas en disco (`sddstatus`, `livingdoc`, listados del dashboard) ignoran `.integrations-outbox.yaml` | Fixture con la cola en una carpeta archivada | `integrationstest.Install`, repositorio temporal, `RunBranch`, `RunSDDGate`, `RunSDDPublish`, `RunIntegrations` en proceso |
| Integración | Doble fallando: verbo origen con salida 0, aviso por stderr, operación pendiente; `sync` la entrega y la retira; con algo pendiente, `sync` sale con 1; `--retry-abandoned` y `--discard` | ídem |
| Integración | Idempotencia: `ensure --role` repetido y `gate record` repetido sobre el mismo artefacto → una sola operación pendiente; reaprobar un artefacto cambiado → operación nueva | ídem |
| Integración | Bloque inválido: aviso y ningún despacho en verbos de compuerta, error en `integrations sync/emit`, error de carga en `workspace validate` | ídem |
| Integración | `--role` con `--archive`, con `--lane odd`, sin kickoff y fuera del roster: rechazo tipado y ningún efecto (sin rama, worktree ni registro nuevos) | Estado de git antes y después |
| Integración | Contratos intactos: `axiom.sdd-gate-record/v1` y el JSON de `sdd publish` byte a byte con y sin integración | Comparación con el resultado sin bloque |
| Integración | Coste cero (§6) | Sin bloque `integrations` |
| Canario | Valor distintivo en el almacén en memoria y en `AXIOM_INTEGRATION_FAKE_TOKEN`: tras cada flujo no aparece en `axiom.yaml`, en ningún fichero de la carpeta del cambio (incluidos los de punto), en el cerrojo, en stdout/stderr ni en ninguna salida `--json`; `secret set` con el canario por stdin tampoco lo imprime | Recorrido de ficheros y búferes |
| Canario | Ningún GET del dashboard devuelve el canario | `internal/dashboard/secrets_canary_test.go`: `httptest` sobre las rutas GET de solo lectura (`/api/health`, `/api/projects`, `/api/workspace`, `/api/increments`, `/api/increments/<c>`, `/api/roles`, `/api/handoffs`, `/api/skills`, `/api/archive/specs`); una guarda lee el registro de rutas de `server.go` y falla si aparece una ruta nueva sin clasificar como «cubierta» o «excluida, con motivo» |
| Ayuda | `axiomHelpText`, `axiom integrations --help` y subcomandos, `branch ensure --help` con `--role` | `cmd/axiom/integrations_help_test.go`, `branch_help_test.go` |
| Documentación | Frases fijas del manual y de la plantilla | `docs_contract_test.go`, `example_config_test.go` |

Comando de verificación: `go test ./internal/integrations/... ./internal/workspace/... ./internal/cli/... ./internal/kickoff/... ./internal/branch/... ./internal/docpaths/... ./internal/sddpublish/... ./internal/hub/... ./internal/sddstatus/... ./internal/dashboard/... ./cmd/axiom/...` y `go test ./...` sin regresiones.

## 8. Matriz de amenazas

El cambio no añade enrutado, órdenes de shell, automatización de VCS o PR ni clasificación de ejecutables propios. Sí introduce una frontera de proceso indirecta (el keyring de macOS invoca `/usr/bin/security` dentro de `go-keyring`) y ficheros nuevos en la carpeta del cambio que `sdd publish` clasifica.

| Frontera | Casos adversariales mínimos | Aplicabilidad | Respuesta del diseño | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas de tipo documentación | `integrations.yaml`, `.integrations-outbox.yaml`, temporales `.integrations-*.yaml` | Aplicable | `integrations.yaml` es documentación del cambio (se publica); la cola y los temporales son estado local en la raíz de la carpeta ordinaria (sin cambios) y en la raíz de la carpeta archivada del cambio (D-17); los ficheros con punto anidados y los de carpetas archivadas de otros cambios conservan su clasificación; el cerrojo no está en el árbol (D-08) | `TestPublishCommitsIntegrationsLinks`, `TestPublishExcludesIntegrationsOutbox` y `TestPublishArchiveExcludesArchivedOutbox` en `internal/sddpublish`; `TestLocalStateArchivedRoot` en `internal/docpaths` |
| Selección de repositorio git | `git -C`, rutas relativas y absolutas | N/A: el núcleo no ejecuta git; el common dir se resuelve leyendo ficheros (`gitcommon.CommonDirFromFiles`) a partir de la carpeta del cambio | — | — |
| Estado del commit | preparado, `commit -a`, índice vacío | Aplicable: `sdd publish` pasa a excluir dos globs en lugar de uno (D-17) | La cola archivada sin seguimiento, o preparada a mano por la persona, nunca entra en el commit de `publish --archive` y queda tal como estaba (sin seguimiento o preparada), como hoy con `.gates.lock` en la carpeta ordinaria (`gitport_test.go:205-233`) | `TestCommitExcludesArchivedLocalState` (sin seguimiento y preparada) en `internal/sddpublish/gitport_test.go` |
| Estado del push | rama con seguimiento, primer push, refspec | N/A: ningún código nuevo sube | — | — |
| Órdenes de PR | `--head`, prefijo de entorno, órdenes compuestas | N/A: no hay automatización de PR | — | — |

Riesgo de proceso fuera de la matriz: en macOS, `go-keyring` lanza `security`; si pasara el secreto como argumento sería visible en la lista de procesos. Se verifica en la unidad 3 antes de fijar la versión (§9); si la versión pasa el valor por argumento, se eleva al usuario antes de seguir.

## 9. Dependencia `github.com/zalando/go-keyring`: estado de verificación

**No verificado.** El módulo no está en la caché local (`go env GOMODCACHE`) y consultar el proxy de módulos o el repositorio es una operación remota que esta fase no tiene autorizada. Lo que sigue procede de conocimiento previo y **debe comprobarse en la unidad 3**, con autorización explícita del usuario para la descarga (`go get`):

| Punto | Expectativa (sin verificar) | Cómo verificarlo |
|---|---|---|
| Versión estable a fijar | serie `v0.2.x` | `go list -m -versions github.com/zalando/go-keyring` |
| Windows puro Go (sin cgo) | `github.com/danieljoos/wincred` sobre `golang.org/x/sys/windows` (ya en `go.mod`, v0.47.0) | `go list -deps` con `GOOS=windows CGO_ENABLED=0 go build ./internal/integrations/secrets/...` dentro de `go test` (`go vet` cruzado) |
| Linux | `github.com/godbus/dbus/v5` (Secret Service); sin D-Bus en CI devuelve error, que el diseño traduce a `KeyringUnavailableError` | prueba con `MockInitWithError` |
| macOS | invoca `/usr/bin/security`; comprobar que el valor viaja por stdin (`security -i`) y no por argumento | leer `keyring_darwin.go` de la versión fijada |
| Errores | `keyring.ErrNotFound`, `keyring.ErrUnsupportedPlatform`, `keyring.ErrSetDataTooBig` | `go doc github.com/zalando/go-keyring` |
| Límite de tamaño | Windows ≈ 2.560 bytes por credencial | ídem; el límite propio de 4 KiB de `secret set` se ajusta por debajo si hace falta |
| Pruebas | `keyring.MockInit()` y `keyring.MockInitWithError(err)` | ídem |
| Licencia | MIT | `LICENSE` del módulo |

## 10. Migración y despliegue

No hay migración de datos. Sin bloque `integrations`, nada cambia (§6). Un `axiom.yaml` con `integrations:` sigue cargando en binarios anteriores (REQ-24.17). En este cambio el catálogo está vacío, así que cualquier plugin declarado es un error de carga hasta C-07-c: la plantilla lo deja comentado y el manual lo dice. Reversión según `proposal.md`; además, borrar el directorio `<common>/axiom/integrations/` es inocuo.

D-17 cambia una clasificación existente: los ficheros con punto de la raíz de una carpeta archivada del cambio dejan de publicarse. Los ficheros ya confirmados no se tocan (`publish` solo confirma rutas sucias y nunca borra), y hoy ningún verbo crea ficheros con punto en una carpeta archivada (REQ-21.18). Al revertir la PR, una `.integrations-outbox.yaml` archivada pendiente pasaría a ser documentación publicable para el binario anterior: antes de revertir, `axiom integrations sync` o `--discard`, o borrar la cola a mano.

## 11. Preguntas abiertas

- [x] **B-1** resuelta por el usuario (2026-11-19): opción B (§0).
- [ ] La spec (en paralelo) debe añadir los dos MODIFIED de §0.1 (REQ-21.18 en `sdd-archive-lifecycle` y REQ-27.5 en `sdd-docs-publication`) y los ADDED de `archive` duradero, `sync`/`emit` sobre carpetas archivadas y `archive` sin escritura de `integrations.yaml`. La propuesta decía «solo requisitos ADDED»: con la opción B deja de ser cierto y conviene anotarlo en la spec.
- [ ] La spec (en paralelo) debe reflejar también: entrega al menos una vez con clave estable y deduplicación solo de lo pendiente (D-09); cerrojo fuera del árbol (D-08); códigos `role_not_applicable`, `kickoff_not_sealed`, `role_not_in_roster` y `plan_conflict` para `--role` (D-13); `emit` no fabrica aprobaciones (§5.6).
- [ ] Autorización del usuario para descargar `go-keyring` y sus transitivas en la unidad 3 (§9).
- [ ] No bloqueante: la cuenta del keyring es `<plugin>/<secreto>` por usuario de la máquina. Si C-07-c necesita un PAT por organización, puede añadir una clave de configuración que sufije la cuenta sin romper los secretos ya guardados (sin sufijo = cuenta actual).
- [ ] No bloqueante: residuo preexistente de `.gates.lock` en la carpeta del cambio, que hace negarse a `done` y a `ensure` en `in_place` tras cualquier `gate record` (no lo introduce este cambio; D-08 evita repetirlo).

## 12. Riesgos de diseño

| Riesgo | Mitigación |
|---|---|
| Estado de paquete en las costuras de prueba (catálogo, registro, almacén por defecto) | Restauración con `Cleanup`; sin `t.Parallel()` en esas pruebas; documentado en `integrationstest` |
| Plugin que no respeta el contexto bloquea el verbo más de 10 s | El despachador abandona la espera al vencer el plazo (la llamada sigue en su goroutine, sin escribir nada después) y registra el intento como fallido |
| Cerrojo en el common dir no compartido entre clones distintos | Cada clon tiene su cola: es el comportamiento esperado (la idempotencia por clave cubre los duplicados entre máquinas) |
| Cola en `in_place` impide cambiar de rama mientras haya pendientes | Aceptado por el usuario: protege la cola; el aviso nombra `sync` y `--discard` |
| **`done` sobre la rama de archivado** con la cola de `archive` pendiente | La cola es un fichero sin seguimiento y no ignorado en el árbol de la rama de archivado. `axiom branch done` se niega con `dirty_worktree` (REQ-25.17): siempre en `in_place`, y en `worktree` salvo `--force`. Con `--force` en `worktree`, `done` elimina el worktree y **descarta la cola** (el aviso de `dirty_worktree` ya lista el fichero). En `in_place`, el siguiente `ensure` que cambie de rama se niega igual (REQ-25.9). Mitigación: el aviso de `publish --archive` nombra `sync` antes de `done`; el manual lo documenta; la prueba de §7 fija la negativa. No cambia ninguna regla |
| Excepción a la inmutabilidad del archivado (MODIFIED REQ-21.18) | Lista cerrada de un solo nombre en `kickoff`, limitada a la raíz exacta de la carpeta archivada, y prueba de arquitectura de que solo `internal/integrations` la llama (D-18) |
| Lectores en disco de carpetas archivadas que recorren ficheros | Prueba de regresión de §7; los lectores basados en árboles de git no ven la cola porque nunca se confirma |
| Solape en `sdd_gate.go`, `branch.go` y `sdd_publish.go` con C-05-d, C-06-f, C-07-e y C-07-f | Una línea por enganche; la lógica vive en `integrations_events.go` |

## 13. Plan de commits por unidad de trabajo

Una sola PR (`single-pr`, `size:exception`). Estimación de líneas de autoría (código y pruebas; docs aparte). El orden difiere de la propuesta en un punto: los secretos van antes que la cola porque el despachador consume `secrets.Store` y `SecretSource`.

| # | Commit (Conventional Commits) | Ficheros | Líneas |
|---|---|---|---|
| 1 | `feat(integrations): manifiesto, puerto y registro cerrado con doble en memoria` | `internal/integrations/manifest/*`, `plugin.go`, `event.go`, `registry.go`, `integrationstest/*` y sus pruebas (incluida la de hoja y la de dirección de importaciones) | ~430 |
| 2 | `feat(workspace): bloque estricto integrations en axiom.yaml` | `internal/workspace/integrations.go`, `declarations.go`, `loader.go`, `types.go`, `internal/hub/init.go`, pruebas de `workspace`, `hub` y `sddstatus`; `axiom.example.yaml`, `docs/manual-de-inicio.md`, `docs_contract_test.go`, `example_config_test.go` | ~330 + docs |
| 3 | `feat(integrations): almacén de secretos con entorno y keyring` | `internal/integrations/secrets/*`, `sanitize.go`, `go.mod`, `go.sum` (requiere autorización de descarga y la verificación de §9) | ~300 |
| 4 | `feat(sdd): estado local con punto en la raíz de la carpeta archivada` | `internal/docpaths/docpaths.go` (`LocalState`, `LocalStateGlobs`) y su prueba (expectativa de la línea 115 invertida y casos nuevos); `internal/sddpublish/publish.go` y pruebas de `publish_test.go`/`gitport_test.go`; `internal/kickoff/archived.go` (`IntegrationsOutboxFile`, `RefuseArchivedWriteExceptLocalState`) y su prueba en tabla | ~190 |
| 5 | `feat(integrations): cola local, vínculos y despachador con reintento` | `files.go`, `links.go`, `outbox.go`, `dispatcher.go`, `sync.go` (incluidas las carpetas archivadas y la guarda de D-18) y pruebas; pruebas de clasificación de `integrations.yaml` en `internal/sddpublish` | ~560 |
| 6 | `feat(cli): verbos axiom integrations sync, emit y secret` | `internal/cli/integrations.go`, `integrations_events.go` (adaptador), `cmd/axiom/main.go`, `cmd/axiom/integrations_help_test.go`, pruebas | ~380 |
| 7 | `feat(cli): eventos desde gate record, ensure --role y publish --archive` | `internal/cli/sdd_gate.go`, `branch.go`, `sdd_publish.go`, `internal/kickoff/types.go`, `internal/branch/types.go`, `labels.go`, `role_refusal.go`, `cmd/axiom/branch_help_test.go`, e2e del criterio de terminado (incluido `archive` duradero), prueba de `done` sobre la rama de archivado, regresión de lectores de carpetas archivadas, canarios de `internal/cli` y `internal/dashboard/secrets_canary_test.go` | ~460 |

La unidad 4 va antes que la cola porque el despachador necesita la guarda de D-18 para escribir en la carpeta archivada, y `publish` debe excluir esa cola antes de que exista. Total estimado: ~2.650 líneas de autoría con pruebas, por encima de las 1.800-2.300 de la propuesta (unas 290 más) por la opción B de B-1. Sigue siendo una PR con `size:exception`. Las specs delta las escribe la fase `spec` en la carpeta del cambio y `sdd-archive` las integra en `openspec/specs/`: no son una unidad de `apply` (la unidad 7 de la propuesta queda absorbida por ese flujo).

No se adelanta nada de C-07-c a C-07-h: sin adaptador real, sin `axiom change start`, sin Tasks por rol, sin plantillas de rama, sin MCP ni UI.
