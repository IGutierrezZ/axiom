# Diseño: pantalla «Ajustes del proyecto» basada en un catálogo de claves (C-06-e)

## Enfoque técnico

Se sigue el enfoque 1 de la propuesta: un paquete nuevo `internal/settings` con el catálogo de claves de `axiom.yaml` y un escritor de *round-trip* sobre `yaml.Node`, y endpoints finos en `internal/dashboard` que solo traducen HTTP a llamadas del paquete. La pantalla JS se genera desde el catálogo que devuelve el servidor y no conoce ninguna clave.

- **Dependencias:** `settings → workspace, multirole, reqid, components/filemerge` y `dashboard → settings, hub`. Nada importa `settings` salvo `dashboard`, y `workspace` nunca importa `settings` (se ha comprobado que `multirole` ya importa `workspace` y no al revés, así que no hay ciclo).
- **Validación:** se reutiliza `workspace.ParseConfig(data []byte)` (`internal/workspace/loader.go:17`), que ya opera sobre bytes. Ejecuta `parseLayered` → `readDeclarationBlocks` (`declarations.go:101`), que incluye `decodeGit`, `decodeWorkflow`, `decodeSDD` con `checkISOWorkflow`, `decodeDelivery`, `decodeIntegrations`, `decodeRepositories` y `decodeAcceptanceTests`, más `validateBaseConfig`. No hace falta exportar ningún decodificador. La validación del motor contra el disco es la misma que la de `axiom workspace validate` (`cmd/axiom/main.go:520-546`): `workspace.Validate`, `ValidateDeclarationPaths` y `ValidateRepositoryPolicies`, con `loc.Master` como `baseDir`.
- **Ruta del fichero:** siempre `workspace.Locate(root).ConfigPath` (`locate.go:50`). `Service.GetWorkspace` (`service.go:152`) sigue usando `<raíz>/axiom.yaml` y queda fuera de este cambio.
- **Escritura atómica:** se reutiliza `filemerge.WriteFileAtomic` (`internal/components/filemerge/writer.go:92`). Ya cubre el temporal en el mismo directorio, el `fsync`, la lectura de comprobación, el rechazo de enlaces simbólicos y la conservación del modo, y tiene pruebas de «sin temporal tras un fallo» (`writer_landed_test.go:147`, `writer_test.go:257`).

Cubre RF-TOP-21, RF-TOP-22 y RF-TOP-23 (`specs/workspace-topology/spec.md`) y RF-WEB-07, RF-WEB-10, RF-WEB-18, RF-WEB-19, RF-WEB-20 y RF-WEB-21 (`specs/local-web-dashboard/spec.md`). Las divergencias detectadas con la spec están en «Preguntas abiertas».

## Evidencia de la prueba de *round-trip* de yaml.v3

Antes de decidir se hizo una prueba desechable con `go test` en el *scratchpad* (sin compilar binarios sueltos), con `gopkg.in/yaml.v3 v3.0.1`, la versión de `go.mod`. Cada `axiom.yaml` y `axiom.example.yaml` del repositorio se decodificó a `yaml.Node` y se volvió a codificar con `yaml.NewEncoder` y `SetIndent(2)`. Resultado:

- Se conservan los comentarios de cabecera, de línea y de pie, el comentario de documento con la línea en blanco que lo sigue, el orden de las claves, las comillas dobles, el estilo literal `|` y el estilo sangrado de las secuencias (`- path:`).
- **Se pierden todas las líneas en blanco entre bloques.** Sin corregirlo, el golden de RF-TOP-22 («solo cambia esa línea») falla.
- Si un nodo nuevo `!!str` lleva el valor `123`, se emite con comillas (`"123"`) y no cambia de tipo. Un `!!bool` se emite sin comillas.
- Cambiar solo `Value` en un nodo escalar existente conserva su `Style` (`lane: "sdd"`).

## Decisiones de arquitectura

### Decisión: un paquete `internal/settings` como dueño del catálogo y del escritor

**Elección:** paquete nuevo con cinco ficheros: `catalog.go`, `document.go`, `patch.go`, `layout.go` y `write.go` (más `errors.go`). El dashboard solo traduce entre HTTP y el paquete.
**Alternativas descartadas:** (a) ponerlo en `internal/workspace`, que mezclaría la escritura con un paquete que hoy es de solo lectura (`Locate` lo declara así en `locate.go:111`) y del que dependen decenas de paquetes; (b) ponerlo en `internal/dashboard`, de modo que C-07-h no podría reutilizarlo sin depender del servidor HTTP; (c) un catálogo en JS con un `PUT` genérico (el enfoque 2 de la exploración).
**Razón:** C-07-h reutiliza el catálogo y el escritor (P11). Un paquete hoja con una API pequeña se prueba sin HTTP y deja `workspace` como está.

### Decisión: vocabularios cerrados como funciones exportadas que también usa el validador

**Elección:** se exportan cuatro funciones que devuelven una copia nueva en cada llamada. Las usan **a la vez** el validador existente y el catálogo:

| Función | Fichero | Validador que pasa a usarla |
|---|---|---|
| `workspace.PromotionValues() []string` | `gitpolicy.go` | `validPromotion` (`gitpolicy.go:208`) |
| `workspace.WorkflowLanes() []WorkflowLane` | `workflow.go` | `workflowValidator` (`workflow.go:40`) |
| `workspace.ExplainerHTMLModes() []ExplainerHTMLMode` | `sdd.go` | `sddExplainerHTML` (`sdd.go:120`) |
| `multirole.GatePolicies() []GatePolicy` | `multirole/types.go` | `hub.policyViolations` (`hub/init_validate.go:347`) |

Los mensajes de error actuales no cambian de texto: la refactorización solo sustituye el `switch` por `slices.Contains`.
**Alternativas descartadas:** (a) repetir en el catálogo las constantes una a una (`PromotionBranch`, `PromotionRelease`…), que obliga a editar el catálogo cuando se añade un valor e incumple RF-TOP-21 («sin editar el catálogo»); (b) exportar las variables de las listas, que son mutables; (c) exportar las listas de claves (`gitKeys`, `sddKeys`…) con un `DeclarationKeys(block)`. La spec no lo exige, y una prueba de coherencia basta: cada clave editable del catálogo, con un valor de ejemplo válido, la acepta `ParseConfig`.
**Razón:** una sola lista para el validador y para el catálogo. Cada función tiene uso de producción doble, así que el trinquete de código muerto (`scripts/deadcode-ratchet.sh`) queda en verde.
**Nota:** el vocabulario de `gate_policy` vive en `multirole` (`types.go:26-35`), no en `workspace`. La spec dice «constantes del esquema de `workspace`»; se deriva de su dueño real (ver «Preguntas abiertas»).

### Decisión: catálogo declarativo en Go con plantillas por ruta, expandido por documento

**Elección:** una tabla `[]keySpec` con plantillas de ruta, que se expanden con el documento leído:

- Claves de bloque: `workspace.name`, `git.integration_branch`, `git.branch_prefix`, `git.main_branch`, `git.promotion`, `workflow.lane`, `delivery.split_hint_lines`, `sdd.iso`, `sdd.id_pattern`, `sdd.explainer_html`, `acceptance_tests.enabled` y `acceptance_tests.environments`.
- Claves por elemento: `repositories[<path>].integration_branch`, `repositories[<path>].branch_prefix`, `repositories[<path>].promotion` y `roles[<id>].gate_policy`.
- Solo lectura: `workspace.topology`, `workspace.specs_repository`, `workspace.root`, `workspace.skill_roots`, `technical_context` (subárbol completo) y `governance.<clave>`, una por cada clave declarada en el mapeo `governance` del fichero, sin lista fija.

Las rutas por elemento se analizan así: el prefijo es `repositories[` o `roles[`, y el selector va desde el primer `[` hasta el último `].`, de modo que un `]` dentro de la ruta no rompe el análisis. En `repositories[<path>]` el selector se compara con la ruta normalizada (`\` → `/` y `path.Clean`). Los candidatos son la unión de las entradas de `repositories:` y las rutas de los repositorios de los roles, ordenada.

**Filtros por documento:**

- En `monorepo-embedded`, o con la topología ilegible, que se trata como embebida igual que en `Location.GoverningTopology` (`locate.go:64`), no se emiten `repositories[...]`, `acceptance_tests.*` ni `roles[...]`.
- En un proyecto ISO (escalar `sdd.iso` con `!!bool` `true`), `workflow.lane` publica `Allowed` sin `odd`.

**Valores por defecto mostrados:** salen de constantes, nunca de literales. Son `DefaultIntegrationBranch`, `DefaultBranchPrefix`, `DefaultWorkflowLane`, `ExplainerHTMLAsk`, `multirole.PolicyBlocking` y `reqid.Default().Template()`. Esta última evita cualquier literal `REQ-` en Go de producción (`internal/idmigration/residues.yaml` lo prohíbe).

**Efectos descritos explícitos:**

- `sdd.iso`: todo el trabajo pasa por SDD y `verify` es obligatorio antes de archivar.
- `sdd.id_pattern`: solo los requisitos nuevos usan la plantilla; los existentes no se renombran automáticamente.

Toda entrada editable lleva el aviso común «solo afecta a cambios nuevos».

**Alternativas descartadas:** derivar etiquetas y tipos por reflexión de las etiquetas `yaml` de `WorkspaceConfig`. Los bloques estrictos llevan `yaml:"-"` (`types.go:83-89`) y no hay sitio para la descripción ni para el efecto.
**Razón:** el catálogo declara lo que la UI necesita y los vocabularios los aporta el esquema.

### Decisión: valores del GET leídos del `yaml.Node`, no de `WorkspaceConfig`

**Elección:** `settings.Read` recorre el árbol YAML para obtener cada valor tal como está escrito, junto con la topología y el carácter ISO. `ParseConfig` se ejecuta aparte solo para rellenar `DocumentErrors`.
**Alternativa descartada:** usar `Locate().Config`, que es `nil` cuando el documento es inválido, y los valores ya resueltos con defectos.
**Razón:** con un `axiom.yaml` inválido escrito a mano, la pantalla debe mostrar el valor real y dejar que la persona lo corrija. El escritor acepta el parche si el documento **resultante** supera `ParseConfig`, así que corregir una clave inválida desde la UI funciona.

### Decisión: parche por lotes, atómico y con `null` para retirar

**Elección:** el cuerpo del `PUT` es `{"revision": "<sha256>", "changes": {"<clave>": <valor>|null, ...}}`. Todos los cambios se aplican sobre el mismo árbol y se validan juntos.
**Alternativas descartadas:** una clave por petición, que no permite pasar de forma válida de `lane: odd` a `sdd.iso: true` (cualquiera de los dos pasos sueltos choca con `checkISOWorkflow`), y JSON Patch, que es más complejo y no aporta nada.
**Razón:** RF-TOP-22 valida el documento completo. Los cambios que dependen entre sí se envían juntos.

La comprobación de tipo y de vocabulario de cada clave se hace contra el vocabulario **sin filtrar por ISO**. La contradicción ISO la rechaza `checkISOWorkflow` sobre el documento resultante, con su mensaje (`ErrISOWorkflowContradiction`). Así, desactivar ISO y elegir `odd` en el mismo lote es válido.

### Decisión: operaciones sobre el árbol y dónde se insertan los nodos nuevos

**Elección:**

- **Cambiar un escalar existente:** se sustituyen `Value` y `Tag` y se conserva `Style`; `Tag` es `!!str`, `!!bool` o `!!int` según el tipo del catálogo.
- **Crear una clave en un mapeo existente:** se añade al final del mapeo. La cadena nueva usa `DoubleQuotedStyle`, la convención de la plantilla de alta (`hub/init.go`, `%q`); el booleano y el entero van sin comillas.
- **Crear un bloque de primer nivel:** se inserta tras la última clave de primer nivel que lo precede en el orden canónico `workspace, git, repositories, workflow, sdd, delivery, acceptance_tests`. Si no hay ninguna, va tras `workspace` y, si tampoco existe `workspace`, al final. El orden coincide con el `axiom.yaml` del repositorio y deja los bloques nuevos antes de `roles` y `governance` en los ficheros que genera el alta.
- **`repositories[<p>].<campo>` sin entrada:** se crea `{path: <p>, <campo>: <valor>}` al final de la secuencia, y se crea el bloque si falta. Al retirar el último campo de ajuste de una entrada, la entrada se borra; al quedar la secuencia vacía, se borra el bloque.
- **Retirar (`null`):** se borra el par clave-valor. Si el mapeo del bloque queda vacío, se borra el bloque. Es obligatorio: los decodificadores rechazan `git:`, `workflow:` y `sdd:` nulos (`gitpolicy.go:104`, `workflow.go:57`, `sdd.go:52`). `workspace.name` es `Required` y no admite `null` (`invalid_value`).
- **`roles[<id>].gate_policy`:** se escribe en minúsculas canónicas. Al retirarlo, el rol vuelve a `blocking` y su mapeo no se borra nunca.
- **Clave repetida en el mapeo que se edita:** rechazo con `unsupported_document`, porque no se sabe cuál editar.

**Alternativa descartada:** añadir siempre al final del documento, que en los ficheros del alta deja `git:` detrás de `governance:`.

### Decisión: restaurar el formato tras serializar (líneas en blanco, CRLF y BOM)

**Elección:** `layout.go` define `restoreLayout(original, encoded []byte) []byte`:

1. Detecta en el original el BOM UTF-8 y si los finales de línea son `\r\n`.
2. Divide el original en líneas sin finales de línea y anota cuántas líneas en blanco preceden a cada línea con contenido y cuántas quedan al final.
3. Alinea por LCS (programación dinámica de O(n·m); un `axiom.yaml` tiene cientos de líneas) las líneas con contenido del original con las del codificado, comparadas sin espacios finales.
4. Emite el codificado con las líneas en blanco originales antes de cada línea alineada. Una línea nueva sin alinear que abre una clave de primer nivel (columna 0 y no es un comentario) lleva una línea en blanco delante si la anterior no lo es.
5. Vuelve a aplicar CRLF y BOM si el original los tenía.

**Alternativas descartadas:** (a) aceptar la pérdida de líneas en blanco, que reformatea todo el fichero en `git diff` e incumple el golden; (b) edición quirúrgica de texto por posiciones de `yaml.Node`: `yaml.v3` da `Line` y `Column` de inicio pero no de fin, y no sirve para listas ni bloques nuevos; (c) otra librería YAML (`goccy/go-yaml`), que añade una dependencia y obliga a revalidar sus diferencias con los decodificadores de yaml.v3.
**Razón:** la serialización sigue siendo de yaml.v3, que garantiza un YAML correcto, y el posproceso solo reinserta líneas en blanco. Un error de alineación como mucho mueve una línea en blanco: nunca cambia datos, y la revalidación posterior lo confirma.

### Decisión: rechazo de anclas, alias, `<<` y multidocumento

**Elección:** `document.go` decodifica con `yaml.NewDecoder` y exige que la segunda llamada a `Decode` devuelva `io.EOF`. Si no, rechaza el documento: `yaml.Unmarshal` solo lee el primer documento y la reescritura **borraría** los siguientes. Después recorre el árbol y rechaza cualquier nodo con `Anchor != ""`, `Kind == AliasNode` o una clave con `Tag == "!!merge"` (`<<`). El rechazo es `unsupported_document` y nombra la construcción y su línea. El GET lo informa como `writable: false` y `writeBlockedReason`, y la UI deshabilita los controles.
**Alternativa descartada:** rechazar solo cuando la construcción afecta a la clave editada. yaml.v3 conserva anclas y alias al reescribir, pero un cambio en el nodo anclado se propagaría en silencio a sus alias.

### Decisión: la validación del motor bloquea solo los errores que introduce el parche

**Elección:**

1. `ParseConfig(nuevos)` debe pasar **entero**. Cualquier error es `validation_failed`, con una violación por línea del `errors.Join`.
2. Las comprobaciones contra el disco se hacen con la función `engineErrors(cfg, master)`, que reúne las `Errors` de `Validate`, `ValidateDeclarationPaths` y `ValidateRepositoryPolicies`. Se ejecutan sobre el documento original (si `ParseConfig(original)` pasa) y sobre el nuevo. Solo bloquean los errores nuevos, que no estaban antes. Los que ya existían vuelven como `warnings` en la respuesta.

**Alternativa descartada:** exigir que `Validate` pase entero. El propio dashboard tiene proyectos de prueba que no lo pasan: `policyHeader` (`server_test.go:529`) declara el rol `core` en `monorepo-embedded`, y `validator.go:126-137` lo rechaza. En `multirepo` basta con no tener clonado un repositorio de un rol para fallar `Validate`. Con la regla estricta, esos proyectos no podrían cambiar ni el carril desde la UI.
**Razón:** los mensajes de `Validate` no llevan números de línea (citan rutas y roles), así que son estables y la diferencia de conjuntos es fiable. Un parche que estropea el conjunto se sigue rechazando (RF-TOP-22).
**Divergencia:** el texto de RF-TOP-22 («si la validación falla, el fichero NO DEBE modificarse») no distingue los errores previos. Ver «Preguntas abiertas».

### Decisión: revisión SHA-256 con nueva comprobación antes del reemplazo atómico

**Elección:** la revisión es el hexadecimal del SHA-256 de los bytes del fichero tal como están en disco. Secuencia de `Write`:

1. Leer el fichero y calcular `rev0`.
2. Comparar con `ExpectedRevision`: si falta, `invalid_request`; si no coincide, `revision_mismatch`.
3. Parsear, aplicar el parche, serializar, restaurar el formato y validar.
4. Si los bytes resultantes son iguales al original, devolver `Changed: false` sin escribir.
5. Ejecutar `beforeReplace(path)`, una variable de paquete que en producción no hace nada; es la costura de la prueba.
6. Releer el fichero y volver a calcular su revisión; si difiere de `rev0`, `revision_mismatch` sin escribir.
7. `filemerge.WriteFileAtomic(path, nuevos, 0o644)`; el modo existente se conserva.

En el dashboard, `Service.settingsMu` serializa los `PUT` del mismo proceso.
**Alternativa descartada:** un temporal propio, nueva comprobación y `rename`. Duplicaría la lógica endurecida de `filemerge` (enlaces simbólicos, *junctions* de Windows, `fsync` del directorio y lectura de comprobación), y la ventana entre la comprobación y el `rename` seguiría existiendo: ningún sistema operativo ofrece un `rename` condicional.
**Razón:** la ventana que queda es el tiempo de escribir y sincronizar el temporal (milisegundos), igual que con la alternativa, y no se deja ningún temporal porque la nueva comprobación va antes de crearlo.

### Decisión: errores tipados en `settings` y su correspondencia HTTP en `dashboard`

**Elección:** `settings.Error{Code, Message, Violations}`, con el mensaje en castellano. El dashboard traduce cada código a su estado HTTP con una sola función:

| `Code` | HTTP | Cuándo |
|---|---|---|
| `invalid_request` | 400 | JSON mal formado, `revision` ausente, `changes` vacío |
| `config_missing` | 404 | `Locate` sin `ConfigPath` o con un puntero roto |
| `revision_mismatch` | 409 | revisión esperada distinta, o cambio externo antes del reemplazo |
| `unsupported_document` | 422 | anclas, alias, `<<`, multidocumento, clave repetida |
| `unknown_key` | 422 | clave que no está en el catálogo (incluidas `integrations.*`) |
| `key_not_editable` | 422 | clave de solo lectura (`workspace.topology`…) |
| `key_not_applicable` | 422 | clave de multirepo en monorepo, rol o repositorio inexistente |
| `invalid_value` | 422 | tipo incorrecto, fuera de vocabulario, `null` en `Required` |
| `validation_failed` | 422 | `ParseConfig` o errores nuevos del motor; lleva `violations` |
| `write_failed` | 500 | fallo de `WriteFileAtomic` (el mensaje dice si el reemplazo llegó a producirse, `WriteResult.Changed`) |

El cuerpo de error es `{"code", "error", "violations"?}`, en la línea de `refusalBody` (`server.go:936`).
**Alternativa descartada:** 400 para toda validación, como `projects/init`. La spec fija 422 (RF-WEB-19).

### Decisión: `/api/settings/personal` sobre `LookupAutofetch` y `SetAutofetch` con la raíz del servicio

**Elección:** el GET llama a `hubManager.LookupAutofetch(root)`; con `registered == false` responde 404 con el mensaje de `projectNotRegisteredError` (`worktree_prefs.go:24`). El PUT con `{"autofetch": true|false}` llama a `hubManager.SetAutofetch(root, v)`; `errors.Is(err, hub.ErrProjectNotRegistered)` también es 404. La inicialización perezosa del hub de `activeProjectRecord` (`worktree_prefs.go:65-71`) pasa a un método `Service.hub() (*hub.Manager, error)` que comparten las dos rutas. La respuesta es `{"autofetch": bool, "path": root}`.
**Alternativa descartada:** leer `rec.Autofetch` de `activeProjectRecord`. Funciona, pero la spec (RF-WEB-20) nombra `LookupAutofetch`, y `branch/engine/prefs.go:45` ya lo usa con la misma semántica `sameProject`.

### Decisión: `isLocalRequest` antes de leer el cuerpo en todas las escrituras

**Elección:** en `PUT /api/settings`, `PUT /api/settings/personal` y `PUT /api/projects/active/worktree-prefs` (`worktree_prefs.go:128`, hoy sin la comprobación), lo primero es `if !s.isLocalRequest(r) { 403 }`, con el mismo cuerpo que `server.go:313`. Los GET no cambian.
**Razón:** cierra el hueco de la exploración. Las pruebas actuales de `worktree-prefs` ya usan `request(...)` con `Host: 127.0.0.1` (`server_test.go:75`) y no se rompen.

### Decisión: «Abrir en el editor» como enlace `vscode://file/` más «Copiar ruta»

**Elección:** el GET devuelve `configPath`, la ruta absoluta del sistema operativo. `core/settings.js` construye `editorHref(configPath)`: cambia `\` por `/`, quita la `/` inicial, aplica `encodeURI` y antepone `vscode://file/`. Por ejemplo, `vscode://file/C:/repos/x/axiom.yaml` o `vscode://file/home/u/x/axiom.yaml`. Cada fila de solo lectura muestra el enlace «Abrir en el editor» y un botón «Copiar ruta», que usa `copyText` (`core/components.js:83`) con su propio aviso, «Ruta copiada al portapapeles». No se reutiliza `bindCopyButtons`, cuyo aviso dice «Orden copiada». El servidor no lanza ningún proceso.
**Alternativa descartada:** abrir el editor desde el servidor con `exec`. Supone lanzar un proceso, lo que la decisión del orquestador (P15) excluye.

### Decisión: la pantalla JS se divide en lógica pura (`core/settings.js`) y DOM (`screens/ajustes.js`)

**Elección:**

- `core/settings.js` contiene funciones puras: `groupBySection`, `controlHtml(entry)`, que elige el control según `type` y `allowed` (`select`, casilla, `number`, `text`, lista de una línea por elemento) y escapa todo texto con `escapeHtml`; `readOnlyHtml(entry, configPath)`; `collectChanges(entries, values)`, que solo envía lo que difiere y manda `null` para lo vaciado; `editorHref`, y `errorView(err)`, con el mensaje, las violaciones y la acción de recargar si es `revision_mismatch`.
- `screens/ajustes.js` carga los datos, pinta, guarda sin perder lo tecleado al fallar, y gestiona el bloque «Preferencias personales»: el autofetch, y el modo y la raíz de worktree, que se **trasladan** desde `screens/ecosystem.js` con los mismos IDs de DOM.

**Razón:** es el patrón existente: `branches.js` y `ecosystem.js` separan así la lógica del DOM, y `runNodeModules` (`assets_branch_test.go:41`) solo ejecuta en Node módulos de `js/core/`.
**Alternativa descartada:** duplicar el editor de worktree en Ajustes y en Ecosistema, con dos lugares para editar la misma preferencia.

### Decisión: `GET /api/workspace` no cambia

**Elección:** se conservan `branchPolicy` y `repositories` en `WorkspaceDTO` (`types.go:20-23`) y `branchPolicyView` (`service_branch.go:174`). En JS se retiran `branchPolicyRows` y `repoPolicyHtml` de `core/branches.js` (junto con el import de `tableWrap`, que solo usaba `repoPolicyHtml`) y su uso en `screens/ecosystem.js:19,143,145`.
**Razón:** RF-WEB-01 sigue exigiendo esos campos y RF-WEB-21 exige no quitar campos de los que dependan otras pantallas.

## Flujo de datos

```
GET /api/settings
  Server.handleSettings ─→ Service.GetSettings
      workspace.Locate(root) ─→ ConfigPath, Master (sin ConfigPath: 404 config_missing)
      settings.Read(ConfigPath)
          os.ReadFile ─→ sha256 = revision
          yaml.NewDecoder ─→ yaml.Node (multidocumento, anclas, alias y << ⇒ writable=false)
          topología e ISO leídos del nodo ─→ catálogo filtrado ─→ Entries con su valor
          workspace.ParseConfig(bytes) ─→ DocumentErrors (solo informativo)
  ←─ 200 SettingsDTO {configPath, revision, topology, iso, writable, entries[]}

PUT /api/settings {revision, changes}
  isLocalRequest? no ─→ 403 (sin leer el cuerpo)
  decodificar JSON ─→ 400 invalid_request
  Service.settingsMu.Lock
  Locate ─→ settings.Write{ConfigPath, Master, ExpectedRevision, Patch}
      leer y calcular rev0 ≟ expected ──────────────── no ─→ 409 revision_mismatch
      parsear y comprobar construcciones ──────────── no ─→ 422 unsupported_document
      resolver claves con el catálogo (tipo, vocabulario, aplicabilidad) ─→ 422
      aplicar al árbol ─→ Encoder(SetIndent 2) ─→ restoreLayout
      iguales al original ─→ 200 changed=false (sin escribir)
      ParseConfig(nuevos) ─────────────────────────── error ─→ 422 validation_failed
      engineErrors(nuevo) − engineErrors(original) ─ ≠∅ ─→ 422 validation_failed
      beforeReplace; releer y calcular ≟ rev0 ───────── no ─→ 409 revision_mismatch
      filemerge.WriteFileAtomic ───────────────────── error ─→ 500 write_failed
  settings.Read de nuevo ─→ 200 {settings: SettingsDTO, changed, warnings}

GET|PUT /api/settings/personal
  (PUT: isLocalRequest) ─→ Service.hub() ─→ LookupAutofetch(root) | SetAutofetch(root, v)
  no registrado ─→ 404 · valor no booleano ─→ 400 · correcto ─→ 200 {autofetch, path}

Pantalla #/ajustes
  loadSettings ─→ GET /api/settings + GET /api/settings/personal + GET worktree-prefs
  guardar ─→ collectChanges ─→ PUT /api/settings {revision, changes}
      409 ─→ aviso con «Recargar» · 422 ─→ violaciones, conserva lo tecleado
```

## Cambios de ficheros

| Fichero | Acción | Descripción |
|---|---|---|
| `internal/settings/catalog.go` | Crear | `ValueType`, `keySpec`, la tabla del catálogo, la expansión por documento (repositorios y roles), los filtros de topología e ISO y el análisis de rutas de clave |
| `internal/settings/document.go` | Crear | Lectura, revisión, decodificación con detección de multidocumento, anclas, alias y `<<`, navegación del árbol y `Read` |
| `internal/settings/patch.go` | Crear | Conversión de valores por tipo, cambio e inserción de escalares y listas, borrado con limpieza de bloques y orden canónico de inserción |
| `internal/settings/layout.go` | Crear | `restoreLayout`: LCS de líneas, líneas en blanco, CRLF y BOM |
| `internal/settings/write.go` | Crear | `Write`: revisión, parche, serialización, `ParseConfig`, diferencia de `engineErrors`, `beforeReplace`, nueva comprobación y `filemerge.WriteFileAtomic` |
| `internal/settings/errors.go` | Crear | `Code` y `Error` |
| `internal/settings/*_test.go`, `internal/settings/testdata/*.yaml` | Crear | Pruebas del paquete y goldens (ver «Estrategia de pruebas») |
| `internal/workspace/gitpolicy.go`, `workflow.go`, `sdd.go` | Modificar | `PromotionValues`, `WorkflowLanes` y `ExplainerHTMLModes`, que pasan a usar sus validadores |
| `internal/multirole/types.go` | Modificar | `GatePolicies()` |
| `internal/hub/init_validate.go` | Modificar | `policyViolations` usa `multirole.GatePolicies()`; el mensaje no cambia |
| `internal/dashboard/settings_api.go` | Crear | `handleSettings`, `handleSettingsPersonal`, `Service.GetSettings`, `Service.PutSettings`, `Service.GetPersonalSettings`, `Service.SetPersonalAutofetch` y la correspondencia de `settings.Code` con HTTP |
| `internal/dashboard/server.go` | Modificar | `HandleFunc("/api/settings", …)` y `HandleFunc("/api/settings/personal", …)`. Van aquí con literal de cadena, porque `registeredRoutes` (`secrets_canary_test.go:78`) solo analiza `server.go`. Se actualiza el comentario de `handleWorkspace` (`server.go:433-435`) |
| `internal/dashboard/worktree_prefs.go` | Modificar | `isLocalRequest` en el `PUT`; la inicialización perezosa pasa a `Service.hub()` |
| `internal/dashboard/service.go` | Modificar | Campo `settingsMu sync.Mutex` |
| `internal/dashboard/types.go` | Modificar | `SettingsDTO`, `SettingDTO`, `SettingsPutRequest`, `SettingsPutResponse`, `PersonalSettingsDTO` y `settingsErrorBody` (camelCase, como `branchPolicy` y `pendingArchive`) |
| `internal/dashboard/settings_api_test.go` | Crear | Pruebas HTTP (ver «Estrategia de pruebas») |
| `internal/dashboard/assets_settings_test.go` | Crear | Estructura de la pantalla y ejecución en Node de `core/settings.js` |
| `internal/dashboard/secrets_canary_test.go` | Modificar | `/api/settings` y `/api/settings/personal` en `coveredRoutes` (los dos son GET de lectura) |
| `internal/dashboard/assets_branch_test.go` | Modificar | `TestBranchPolicyIsShownReadOnlyInEcosystem` → prueba de que Ecosistema enlaza a Ajustes sin pintar la política; se borra `TestBranchPolicyRuntime`; `TestWorktreePrefsEditorInEcosystem` → «InSettings» (`view-ajustes`, `screens/ajustes.js`); se actualiza el comentario de cabecera |
| `internal/dashboard/server_test.go` | Modificar | El mapa de marcadores (`:445-447`) cambia `/js/screens/ecosystem.js` por `/js/screens/ajustes.js`; las referencias de comentario (`:418`, `:440`, `:607`) pasan a RF-WEB-10; `TestWorkspaceIsReadOnly` se conserva |
| `internal/dashboard/assets/js/core/settings.js` | Crear | Lógica pura de la pantalla |
| `internal/dashboard/assets/js/screens/ajustes.js` | Crear | DOM, guardado y preferencias personales; recibe las funciones de worktree de `ecosystem.js` |
| `internal/dashboard/assets/js/screens/ecosystem.js` | Modificar | Retira `branchPolicyRows`, `repoPolicyHtml` y las funciones y el enlace del formulario de worktree; la tarjeta Topología enlaza a `#/ajustes` |
| `internal/dashboard/assets/js/core/branches.js` | Modificar | Retira `branchPolicyRows`, `repoPolicyHtml` y el import de `tableWrap`; la cabecera pasa a cuatro funciones |
| `internal/dashboard/assets/js/core/router.js` | Modificar | `'ajustes'` en `SIMPLE_ROUTES` y en el comentario de rutas |
| `internal/dashboard/assets/js/main.js` | Modificar | `VIEW_BY_ROUTE.ajustes`, `initSettings()`, y `loadSettings()` en `loadAllData` |
| `internal/dashboard/assets/js/shell.js` | Modificar | `ROUTE_LABELS.ajustes = 'Ajustes del proyecto'` |
| `internal/dashboard/assets/js/core/icons.js` | Modificar | Icono `settings` (trazo SVG en línea, sin recursos externos) |
| `internal/dashboard/assets/index.html` | Modificar | Entrada de navegación en «Plataforma», `<section id="view-ajustes">` con el aviso «Editar solo afecta a cambios nuevos», el contenedor del catálogo y la tarjeta de preferencias personales (la de worktree se traslada desde `view-ecosistema` con los mismos IDs); enlace a Ajustes en el *popover* de proyecto |
| `internal/dashboard/assets/style.css` | Modificar (mínimo) | Solo las clases que falten; se reutilizan `card`, `field-grid`, `field`, `eco-note` y `eco-pad` |
| `internal/dashboard/assets/sw.js` | Sin cambios | No guarda nada en caché por diseño (`sw.js:1-8`); ver «Preguntas abiertas» |
| `docs/usage.md` | Modificar | Subsección «Ajustes del proyecto desde el dashboard»: claves, solo lectura, «solo cambios nuevos», endpoints y sus códigos, petición local y autofetch |
| `docs/codebase/dashboard.md` | Modificar | Invariante: `axiom.yaml` solo se escribe por `/api/settings` mediante `internal/settings`; toda escritura exige `isLocalRequest` |
| `axiom.example.yaml` | Modificar | Nota en la cabecera: las claves del proyecto se editan también desde «Ajustes del proyecto» (`axiom ui`), que conserva los comentarios y valida antes de escribir |

## Interfaces y contratos

```go
package settings

type ValueType string

const (
	TypeString     ValueType = "string"
	TypeBool       ValueType = "bool"
	TypeInt        ValueType = "int"
	TypeEnum       ValueType = "enum"
	TypeStringList ValueType = "stringList"
	TypeYAML       ValueType = "yaml" // solo lectura: subárbol serializado (technical_context)
)

// Entry es una clave concreta del catálogo para un documento.
type Entry struct {
	Key            string    // "git.promotion", "repositories[repo-a].promotion", "roles[web].gate_policy"
	Section        string    // workspace, git, repositories, workflow, delivery, sdd, acceptance_tests, roles, technical_context, governance
	Label          string
	Description    string
	Effect         string
	Type           ValueType
	Allowed        []string // enum, ya filtrado por ISO
	Editable       bool
	ReadOnlyReason string
	Required       bool
	Default        string // descripción del valor por defecto del motor
	Declared       bool
	Value          any // string, bool, int, []string o nil
}

type Snapshot struct {
	ConfigPath         string
	Revision           string
	Topology           workspace.TopologyType
	ISO                bool
	Writable           bool
	WriteBlockedReason string
	DocumentErrors     []string
	Entries            []Entry
}

func Read(configPath string) (*Snapshot, error)

// Patch: valor nil = retirar. Tipos dinámicos admitidos: string, bool,
// float64 entero o int, []any o []string de cadenas.
type Patch map[string]any

type WriteRequest struct {
	ConfigPath       string
	Master           string
	ExpectedRevision string
	Patch            Patch
}

type WriteResult struct {
	Changed  bool
	Revision string
	Warnings []string // errores del motor que ya existían antes del parche
}

func Write(req WriteRequest) (*WriteResult, error)

type Code string

type Error struct {
	Code       Code
	Message    string // castellano
	Violations []string
}

func (e *Error) Error() string
```

```go
package workspace

func PromotionValues() []string               // branch, release, ask
func WorkflowLanes() []WorkflowLane           // odd, sdd, ask
func ExplainerHTMLModes() []ExplainerHTMLMode // ask, always, never
```

```go
package multirole

func GatePolicies() []GatePolicy // blocking, deferred, optional
```

Contrato HTTP (JSON en camelCase):

```
GET  /api/settings
  200 {"configPath","revision","topology","iso","writable","writeBlockedReason"?,
       "documentErrors":[], "entries":[{"key","section","label","description","effect",
       "type","allowed"?,"editable","readOnlyReason"?,"required","default"?,"declared","value"}]}
  404 {"code":"config_missing","error"}
PUT  /api/settings  {"revision":"<hex>","changes":{"workflow.lane":"sdd","git.promotion":null}}
  200 {"settings":<GET>,"changed":bool,"warnings":[]}
  400|403|404|409|422|500 {"code","error","violations"?}
  POST|PATCH|DELETE ⇒ 405
GET  /api/settings/personal          200 {"autofetch":bool,"path"} | 404 {"error"}
PUT  /api/settings/personal {"autofetch":bool}
  200 | 400 (JSON inválido o no booleano) | 403 | 404 | 405 en otros métodos
```

Ni el GET ni el PUT leen ni serializan el bloque `integrations`: no está en la tabla del catálogo y `Read` no recorre ningún nodo que no sea de una entrada. `technical_context` es el único subárbol que se serializa entero.

## Estrategia de pruebas

Las pruebas son localizadas: `go test ./internal/settings` y `go test ./internal/dashboard -run 'Settings|WorktreePrefs|BranchPolicy|Ecosystem|SecretsCanary|Classified'`, con `go test ./internal/workspace ./internal/multirole ./internal/hub -run '<pruebas tocadas>'`. La batería completa queda para la CI. No se compila ningún binario suelto.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitaria `settings` (catálogo) | `Allowed` igual a `PromotionValues`, `WorkflowLanes`, `ExplainerHTMLModes` y `GatePolicies`; coherencia en los dos sentidos (cada valor permitido lo acepta `ParseConfig` y `zzz` lo rechaza); filtros de monorepo y multirepo; ISO sin `odd`; solo lectura; exclusiones (`integrations`, `artifact_store`, `rdd`, `commit_review`, `docs_publication`); `id_pattern` por defecto desde `reqid.Default()` | Tablas con documentos en memoria. Las *fixtures* usan `RF-<DOM>-<NN>`, nunca `REQ-`, para no tocar `residues.yaml` |
| Unitaria `settings` (escritor) | Golden: `testdata/commented.yaml` (cabecera, línea, pie, líneas en blanco y comentarios entre bloques) frente a `commented.lane-sdd.golden`, donde solo cambia una línea; `null` borra la clave (`git.main_branch` junto a otras claves de `git`, porque `delivery` solo tiene una clave); el bloque vacío se borra (`delivery`); bloque nuevo insertado en el orden canónico; entrada de `repositories` creada y borrada; `roles[x].gate_policy`; anclas, alias, `<<`, multidocumento y clave repetida rechazados; `git.promotion: siempre` con el mensaje del validador; ISO en las dos direcciones y en un mismo lote válido; claves de solo lectura, desconocidas y no aplicables; CRLF y BOM conservados; parche sin cambios no escribe | `t.TempDir()`. Cada rechazo comprueba los bytes del fichero sin cambios y el listado del directorio sin temporales |
| Unitaria `settings` (revisión) | Revisión que coincide; revisión obsoleta; cambio externo entre la validación y el reemplazo; revisión ausente | La costura `beforeReplace` reescribe el fichero. Esa prueba **no** usa `t.Parallel` porque modifica una variable de paquete |
| Unitaria `settings` (motor) | Un error previo de `Validate` (rol `core` en monorepo) no bloquea y vuelve como aviso; un error nuevo (entrada de `repositories` hacia una ruta inexistente) bloquea | `t.TempDir()` con carpetas reales |
| Unitaria `layout` | Ida y vuelta de `axiom.yaml` y `axiom.example.yaml` del repositorio idéntica byte a byte sin cambios; líneas repetidas (`- path: "."`) | Se leen los ficheros reales con rutas relativas a la raíz del módulo |
| Unitaria `workspace`, `multirole` y `hub` | Los validadores refactorizados conservan sus mensajes | Las pruebas existentes (`gitpolicy_promotion_test.go`, `workflow_test.go`, `sdd_test.go` y las de `init_validate`) más una prueba por función exportada |
| Integración `dashboard` | GET con catálogo, revisión y ruta; sin `integrations` en el cuerpo; PUT 200, 400, 403, 404, 409, 422 y 405; multirepo con `workspacetest.Canonical` escribe `repo-specs/axiom.yaml` y no crea `axiom.yaml` en la maestra ni en el código; un `kickoff.yaml` sellado no cambia (`snapshotTree` antes y después, salvo `axiom.yaml`); `/api/settings/personal` GET, PUT, 400, 403 y 404, con `workspaces.json` actualizado y el árbol del proyecto intacto; `PUT worktree-prefs` 403 sin escribir el hub y GET normal | `newPrefsServer` (hub temporal propio), `request` y `requestFrom` con `Host` ajeno. Sin `freePort` ni el hub compartido, así que estas pruebas pueden usar `t.Parallel` |
| Canario | Las rutas nuevas quedan clasificadas y el GET no devuelve el secreto | `secrets_canary_test.go` existente con las rutas en `coveredRoutes`. `TestNoReadOnlyDashboardRouteReturnsAnIntegrationSecret` usa `NewService` con el hub por defecto y ya no es paralela: no se cambia |
| Assets JS | Ruta `ajustes` en `SIMPLE_ROUTES`, `VIEW_BY_ROUTE` y `ROUTE_LABELS`; entrada de navegación; aviso «Editar solo afecta a cambios nuevos»; `ecosystem.js` y `branches.js` sin `branchPolicyRows` ni `repoPolicyHtml` y con enlace a `#/ajustes`; editor de worktree en `view-ajustes`; ningún control de RDD ni de `integrations` (`index.html` sigue sin `integration_branch`) | Comprobaciones de cadenas como en `assets_branch_test.go` |
| Ejecución JS | `controlHtml` con vocabulario cerrado como `select` con exactamente `allowed`; escapado de `<script>`; `collectChanges` envía solo lo cambiado y `null` al vaciar; `editorHref` con rutas de Windows y POSIX; `errorView` con `revision_mismatch` ofrece recargar | `runNodeModules(t, []string{"api","icons","components","settings"}, …)`; se omite sin Node |

## Matriz de amenazas

No aplica: el cambio no toca enrutado de agentes, órdenes de shell, subprocesos, automatización de VCS o PR, clasificación de ficheros ejecutables ni integración de procesos. El enlace `vscode://` lo resuelve el navegador y el servidor no lanza procesos.

| Frontera | Aplicabilidad | Motivo |
|---|---|---|
| Rutas con aspecto de documentación | N/A | No se clasifica ni se ejecuta ningún fichero |
| Selección de repositorio git | N/A | No se ejecuta git; la ruta sale de `Locate().ConfigPath` |
| Estado del commit | N/A | No se hace commit; el cambio queda en el árbol de trabajo |
| Estado del push | N/A | No se publica nada |
| Órdenes de PR | N/A | No hay automatización de PR |

**Superficie HTTP de escritura (fuera de la matriz, cubierta por pruebas):** `isLocalRequest` antes de leer el cuerpo en los tres `PUT` (CSRF y DNS rebinding, `server.go:957-983`); el canario de secretos cubre las rutas nuevas; el catálogo excluye `integrations.*`; la revisión impide pisar ediciones externas, y la escritura atómica nunca deja un fichero a medias.

## Migración y despliegue

No requiere migración. No cambia ningún esquema ni decodificador, y los `kickoff.yaml` sellados no se tocan. Las escrituras de la UI quedan sin confirmar en el árbol de trabajo del repositorio que contiene `axiom.yaml`; en `multirepo`, el repositorio de specs. Confirmarlas con git es decisión de la persona, y la pantalla lo indica junto a la ruta. Para revertir basta con revertir la PR, y una escritura concreta se deshace con `git checkout -- axiom.yaml`.

## Preguntas abiertas

Ninguna bloquea el diseño. Son divergencias entre la spec escrita en paralelo y el código o el diseño, que conviene ajustar en la spec antes de las tareas:

- [x] **RF-TOP-22, regla de validación del motor:** el diseño bloquea solo los errores de `Validate` que introduce el parche y devuelve como avisos los que ya existían. Con la lectura literal de la spec, un proyecto con un repositorio sin clonar, o con un rol especializado en monorepo, no podría editar nada desde la UI. Propuesta: añadir a RF-TOP-22 un escenario de «error previo del motor no bloquea».
- [x] **RF-WEB-18 y RF-WEB-21, caché del *service worker*:** `sw.js` no guarda nada en caché por diseño (`sw.js:1-8`: «una copia obsoleta sería peor que un fallo de red»). La cláusula no se puede cumplir sin cambiar esa política. Propuesta: quitarla de la spec; el diseño no toca `sw.js`.
- [x] **RF-WEB-21, «Proyectos»:** no existe una pantalla Proyectos con política de ramas. `js/projects.js` es el selector o *popover* y no usa `branchPolicyRows`, que solo usa `screens/ecosystem.js`. El diseño añade un enlace «Ajustes del proyecto» en el *popover*. Propuesta: reformular el escenario («el selector de proyecto ofrece un enlace a Ajustes»).
- [x] **RF-WEB-01:** su texto vivo dice «ninguna ruta del servidor DEBE modificar `axiom.yaml` (RF-WEB-10)», lo que contradice el RF-WEB-10 nuevo. Hay que añadir RF-WEB-01 a los MODIFIED, limitando la frase a `/api/workspace`.
- [x] **RF-TOP-21, origen de `gate_policy`:** su vocabulario está en `multirole`, no en `workspace`. El diseño lo deriva de `multirole.GatePolicies()`.
- [x] **RF-TOP-22, escenario «`delivery.split_hint_lines` y otras claves en `delivery`»:** no es posible, porque `deliveryKeys` solo tiene `split_hint_lines` (`delivery.go:19`). Las pruebas usan `git` para «retirar una clave conserva las demás» y `delivery` para «bloque vacío se borra».
- [x] **RF-TOP-23, «justo antes del `rename`»:** la nueva comprobación se hace justo antes de `filemerge.WriteFileAtomic`, no entre su temporal y su `rename`. La ventana es la misma en la práctica y no deja temporales (ver la decisión correspondiente).

**Resolución (2026-11-26):** el gatekeeper relanzó una vez la spec con corrección acotada, y las siete divergencias quedaron recogidas en `specs/` (RF-WEB-01 pasa a MODIFIED; RF-TOP-22 añade el escenario del error previo; se quita la caché del *service worker*; se reformula «Proyectos»; `gate_policy` sale de `multirole`; el escenario de `delivery` pasa a `git`; la ventana residual queda declarada en RF-TOP-23).
