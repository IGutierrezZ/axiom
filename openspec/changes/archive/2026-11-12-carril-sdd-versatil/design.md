# Design: Evidencias ISO y correcciones del SDD (carril-sdd-versatil)

> Fase `design` · rama `inc/carril-sdd-versatil` · 2026-11-12 · entrada: `proposal.md` (con las enmiendas D4/D5, la segunda y la tercera enmienda del 2026-11-12) y las 9 deltas `specs/*/spec.md` (REQ-29.1 a REQ-29.19). Sin contexto técnico mantenido: no hay `context-impact.md`.
> Tercera enmienda incorporada: D4 es «ODD integra en la spec viva; los proyectos ISO solo usan SDD». La doctrina ODD manda todo el trabajo de un proyecto ISO al SDD (REQ-29.18, decisión D16) y `sdd.iso: true` con `workflow.lane: odd` es una declaración contradictoria (REQ-29.1 y REQ-29.2, decisión D18).
> Revisión tras el validador de contrato (2026-11-12): la matriz se exige aunque `apply` no haya terminado (D12), el status ruta a `verify` en ISO (D19), el error de la declaración llega al status con `iso.reason` (D8), hay un punto en Go antes de mover la carpeta (`axiom sdd archive-check`, D13), `archive-compose` reutiliza `changeRootOfDelta` y los huecos H2 a H4 se aplican en las tablas.
> H6 decidido por el usuario (2026-11-12): REQ-29.3 tolera el ADDED ya compuesto por un archivado interrumpido (mismo dominio, nombre y bloque byte a byte); D17 y `archive-check` lo implementan.

## 0. Resumen para quien revisa

El cambio se implementa como **un paquete hoja nuevo, `internal/isoevidence`**, que contiene todos los analizadores y reglas ISO como funciones puras, y **cuatro adaptadores finos** que lo conectan: `internal/sddstatus/iso.go` (estado y motivos tipados), `internal/cli/sdd_archive_check.go` (comprobación previa al movimiento de la carpeta), `internal/cli/sdd_archive_compose_iso.go` (negativa del escritor de archivado) e `internal/cli/sdd_trace.go` (traza). Las dos correcciones del SDD actual (QA diferido y arranque con carpeta) son cambios acotados en `sddstatus`, `kickoff` y `cli`. Nada depende de `internal/specbaseline`, de `axiom sdd integrate` ni del bloque `baseline:`, que P6 retirará.

Qué revisar primero:

1. §2, decisiones D1, D2 y D6: dónde vive cada pieza y cómo bloquea el status sin cambiar claves. D16 y D18 recogen la tercera enmienda (doctrina ISO solo SDD y validación cruzada con `workflow.lane`). D12, D13 y D19 a D21 recogen la revisión del validador: matriz exigida también con tareas pendientes, `archive-check` antes de mover, ruta `verify` obligatoria en ISO, compuerta `spec` del bug y almacenes sin ficheros.
2. §5, contratos: tokens de los motivos tipados, proyección `iso`, formato de la matriz.
3. §11, huecos de producto: seis preguntas con recomendación; ninguna bloquea el design.

Previsión total: **del orden de 4.900 a 5.750 líneas autoradas** (creadas más borradas, sin goldens), por encima de las 1.200 a 1.800 que estimó la propuesta (§10).

---

## 1. Enfoque técnico

| Pieza de la propuesta (§3.1) | Enfoque | Paquetes |
|---|---|---|
| 1. QA diferido no bloquea `verify` | Partición por `gate_policy` en la regla de `verify` y en el aviso del último rol | `sddstatus`, `kickoff`, `cli` |
| 3. Arranque con carpeta e instantánea `sdd.iso` | Resolutor propio de `seal` que crea la carpeta tras validar; bloque `sdd:` en `kickoff.yaml` | `kickoff`, `cli` |
| 4. Declaración ISO e identificador | Bloque `sdd:` en `axiom.yaml` con validación cruzada contra `workflow.lane`; regla REQ-29.3 en `isoevidence` | `workspace`, `isoevidence`, `sddstatus` |
| 2. Escenarios por requisito | Regla REQ-29.10 en `isoevidence`; adelanto en status y negativa en `archive-compose` | `isoevidence`, `sddstatus`, `cli` |
| 5. Cobertura `Cubre:` | Analizador de líneas `Cubre:` y cobertura conjunta | `isoevidence`, `sddstatus` |
| 6. Matriz con evidencia contrastable | Analizador de la sección de matriz, resolución de referencias contra el árbol | `isoevidence`, `sddstatus`, `cli` |
| 7. Trazabilidad | Verbo de solo lectura `axiom sdd trace` sobre `git log --first-parent` | `isoevidence`, `cli` |
| 8. Perfil de bug (tres rutas) | Analizador de `proposal.md`; el artefacto de specs se cumple sin delta | `isoevidence`, `sddstatus` |
| 12. D4 en la doctrina | Con ISO proyectado, todo el trabajo va al SDD sin pregunta de carril ni `odd/tasks/`; los bugs, con el perfil de bug | `agentguidance` |
| 13. Doctrina, guía, goldens | Fuente única nueva `_shared/sdd-iso-convention.md` y párrafos en secciones existentes | `assets`, `agentguidance` |

Principios que rigen todo el design:

- **Opt-in estricto.** Sin instantánea ISO en el cambio, ninguna regla nueva bloquea. Las únicas novedades visibles fuera de ISO son las dos correcciones y el perfil de bug, que solo actúa si `proposal.md` lo declara.
- **El status solo crece.** Ninguna clave existente del status v2 cambia de nombre ni de tipo. Las reglas ISO **estrechan** dependencias (de `ready` o `all_done` a `blocked`) con el patrón que ya usan `verifyDependencyFromRoleGates` y `archiveDependencyFromGovernance`.
- **Validación en Go, formato en prosa.** Las skills describen los formatos; los analizadores viven en Go y producen motivos tipados.
- **Sin dependencia de lo que P6 borra.** No se reutiliza `internal/specbaseline`, ni `axiom sdd integrate`, ni la rama de documento con perfil de `archive-compose`. `changeRootOfDelta` sí se reutiliza: no es maquinaria de baseline, la usa la guarda A (`guardComposeDelta`, REQ-25.19).

---

## 2. Decisiones de arquitectura

### D1. Paquete hoja nuevo `internal/isoevidence`

**Elección:** un paquete nuevo, sin dependencias internas (solo la biblioteca estándar, `go/parser` y `go/ast`), que expone funciones puras: reciben el modelo ya leído (requisitos, escenarios, textos de tareas, texto del informe, un `fs.FS` del árbol) y devuelven hallazgos tipados. `sddstatus` y `cli` lo adaptan.

**Alternativas descartadas:**

- *Dentro de `internal/sddstatus`.* El paquete ya tiene unas 19.000 líneas, y `cli/sdd_trace.go` necesitaría importar `sddstatus` para algo que no es estado.
- *Reutilizar `internal/specbaseline`.* Es la maquinaria de baseline declarada que P6 retira; atarse a ella obliga a P6 a conservarla o a reescribir este cambio.

**Motivo:** las reglas ISO son lógica de dominio pura y muy testeable por tablas; separarlas de la E/S permite probar cada regla sin montar un workspace, y deja a `sddstatus` con un adaptador corto.

### D2. El análisis de specs reutiliza el lector nativo de OpenSpec

**Elección:** el adaptador `sddstatus/iso.go` lee las deltas con `parseOpenSpecDelta` y la baseline con `parseSpecDocument` (ambas en `openspec_archive_compose.go`, sin tocar ese fichero) y entrega a `isoevidence` un modelo `[]isoevidence.Requirement`. `isoevidence` solo extrae los escenarios del texto de cada bloque.

**Alternativas descartadas:**

- *Lector propio en `isoevidence`.* Duplicaría el reconocimiento de encabezados y vallas, y dos lectores pueden discrepar sobre qué es un requisito (riesgo real: el nativo no detecta vallas en la delta y sí en la canónica).
- *Mover el lector nativo a un paquete compartido.* Toca `openspec_archive_compose.go`, que se comparte con upstream.

**Motivo:** una sola definición de «requisito» para componer y para validar. El status y `archive-compose` nunca discrepan sobre qué requisitos tiene una delta.

### D3. Bloque `sdd:` de `axiom.yaml` en un fichero propio

**Elección:** `internal/workspace/sdd.go` con `SDDConfig`, la lista cerrada `sddKeys = []string{"iso"}` y `decodeSDD(node)`, al estilo de `workflow.go`. `Declarations` gana `SDD *SDDConfig` y `SDDErr error`; `joinLoadErrors` suma `SDDErr`, de modo que `LoadConfig` y `axiom workspace validate` fallan con código `1`, igual que con un `workflow:` inválido. `WorkspaceConfig` no gana campo: los consumidores leen `Location.Declarations`.

**Alternativas descartadas:**

- *Reutilizar `decodeStringKeys`.* Valida cadenas; `iso` es booleano y un `iso: "true"` entre comillas debe rechazarse.
- *Proyectar `cfg.SDD` en `WorkspaceConfig`.* Nadie lo necesita y añade un punto más de solape con P6 en `loader.go`.

**Motivo:** mismo patrón que `workflow:` (REQ-27.1), mínimo de líneas en `declarations.go` y `loader.go`, que P6 también toca.

### D4. Instantánea ISO como bloque `sdd:` de primer nivel en `kickoff.yaml`

**Elección:** `Kickoff.SDD *SDDSnapshot yaml:"sdd,omitempty"` con `SDDSnapshot{ISO bool yaml:"iso,omitempty"}`. `seal` solo lo rellena cuando el proyecto declara `iso: true`; en cualquier otro caso queda `nil` y el fichero no lleva ni `sdd:` ni `iso`. La lectura usa `Kickoff.IsISO()`, que es `k.SDD != nil && k.SDD.ISO`.

**Alternativas descartadas:**

- *Campo `iso` dentro del bloque `kickoff:` (`FlowConfig`).* Ese bloque recoge decisiones del cuestionario previo; ISO es política del proyecto copiada, como `git:`.
- *Booleano de primer nivel `iso:`.* La spec habla de «instantánea de `sdd:`», y un bloque deja sitio a futuras claves sin otra migración.

**Motivo:** mismo patrón que la instantánea `git:` (REQ-25.36): bloque aditivo, puntero nulo cuando no aplica. **Compatibilidad comprobada:** `kickoff.ParseKickoff` decodifica con `yaml.Unmarshal` sin `KnownFields` (`internal/kickoff/schema.go`), así que un binario anterior ignora el bloque sin error. No hay riesgo de rechazo.

### D5. `kickoff seal` prepara la carpeta con un resolutor propio

**Elección:** `seal` deja de usar `resolveGovernanceChangeRoot` (que exige que la carpeta exista y que comparten `gate` y `show`) y usa `resolveSealChangeRoot`, que:

1. valida el nombre del cambio con `gitref.ValidBranchComponent` y rechaza `archive` (colisionaría con `changes/archive/`);
2. conserva la negativa de cambio archivado (`kickoff.ArchivedChangeRoot` y `kickoff.RefuseArchivedRoot`);
3. devuelve la ruta y si la carpeta ya existe, sin crearla.

La carpeta se crea con `kickoff.PrepareChangeRoot` **después** de todas las validaciones (argumentos, declaración `sdd:`, guarda de rama y `ValidateForSeal`) y justo antes de `kickoff.Seal`; si `Seal` falla, la función de retirada borra solo los directorios que creó y solo si están vacíos. `--infer` sigue exigiendo una carpeta existente (retrosellado de un cambio con artefactos).

**Alternativas descartadas:**

- *Crear la carpeta dentro de `kickoff.Seal`.* `Seal` no conoce la guarda de rama ni la declaración `sdd:`, y la spec exige no dejar carpeta ante cualquier rechazo.
- *Crear la carpeta al principio y borrarla al fallar.* Abre una ventana en la que otra orden ve una carpeta vacía, y un borrado tardío puede llevarse ficheros ajenos.

**Motivo:** cumple REQ-29.12 (sin carpeta ante `wrong_branch` ni ante `sdd:` inválido, idempotente, carpeta existente intacta) y cierra la travesía de rutas que abre crear un directorio a partir de `--change`. La guarda A ya admite una carpeta inexistente: `VerdictFor` sube hasta el primer directorio existente (`existingDirectory`).

### D6. Motivos tipados `blocked(<token>)` por etapa, con estrechamiento de dependencias

**Elección:** cada hallazgo ISO o de perfil de bug es una línea de `blockedReasons` con la forma `blocked(<token>): <detalle>; <continuación>`, el mismo formato que `blocked(wrong_branch)`. Los tokens forman una lista cerrada (§5.3). Cada hallazgo pertenece a una etapa, y la etapa decide qué dependencia estrecha:

| Etapa | Cuándo se evalúa | Dependencia que estrecha | Efecto en la ruta |
|---|---|---|---|
| `specs` | delta presente (o declaración de bug) | `specs` → `blocked`; `coreReady` → falso | la ruta vuelve a `spec` |
| `tasks` | existen todos los ficheros de tareas | `tasks` → `blocked`; `coreReady` → falso | la ruta vuelve a `tasks` |
| `archive` | siempre que `dependencies.archive` saldría `ready` (`coreReady` y `apply` no bloqueado), también con tareas pendientes | `archive` → `blocked` | con `apply` terminado, `verify` si falta la matriz o es inválida (D19) y `resolve-blockers` si tiene filas `fail`; con `apply` pendiente, la ruta no cambia y el motivo va a `notes` |

**Alternativas descartadas:**

- *Un campo nuevo de motivos estructurados.* Los consumidores ya leen `blockedReasons`; un canal paralelo exigiría cambiar todos los orquestadores.
- *Solo añadir motivos, sin estrechar dependencias.* `nextRecommended` seguiría proponiendo `design` con el identificador ausente, en contra de REQ-29.3.
- *Marcar el artefacto `specs` como `partial`.* Mentiría sobre el estado de los ficheros.

**Motivo:** el contrato de los consumidores prohíbe `apply` con `blockedReasons` no vacío; por eso los motivos de la etapa `archive` van siempre a `notes` y solo pasan a `blockedReasons` cuando, sin ellos, la ruta sería `archive` (mismo patrón que `blockArchiveOnDeclarations`, `workspace_declarations.go:154-169`). El orden de precedencia existente se conserva: `wrong_branch` se sigue anteponiendo al final de `resolveOpenSpecStatus`, y los motivos ISO se añaden después de los de artefactos, gobierno, autoridad de edición, `verify` y archivado.

### D7. Proyección `iso` aditiva y omitida cuando no hay nada que decir

**Elección:** campo nuevo `Status.ISO *ISOProjection json:"iso,omitempty"`, proyectado igual en v2. Solo aparece cuando `axiom.yaml` tiene bloque `sdd:` (válido o no) o cuando el cambio seleccionado es ISO por su instantánea (§5.2).

**Alternativa descartada:** *proyectarlo siempre, como `workflow`.* Cambiaría la salida de todos los cambios que hoy fijan las pruebas de regresión byte a byte (`kickoff_absence_regression_test.go`, `status_v2_clean_break_test.go`).

**Motivo:** REQ-29.2 permite omitir las claves sin bloque; un consumidor anterior no ve ninguna diferencia.

### D8. Una declaración `sdd:` inválida falla cerrada en el status bloqueando solo el archivado

Fuera del status no hay matiz: `LoadConfig`, `axiom workspace validate` y `kickoff seal` (sin `--infer`) fallan con el error (D3 y D18).

**Elección:** el status trata como inválida la declaración en tres casos: `SDDErr` no nulo (bloque inválido o contradictorio con `workflow.lane`, D18), y `ParseErr` no nulo (`axiom.yaml` ilegible: no se puede saber qué declara). En los tres:

- proyecta `iso.project: "invalid"` e `iso.reason` con el error y su continuación (§5.2), **con o sin cambio seleccionado**;
- añade siempre a `notes` la línea `La declaración sdd: de axiom.yaml no es válida: <error>. Corrígela con axiom workspace validate --path <maestra>; mientras tanto ningún cambio puede archivarse.`;
- con cambio seleccionado, estrecha `archive` con `blocked(iso_declaration_invalid)`, que pasa a `blockedReasons` solo cuando la ruta sería `archive` (D6).

Las reglas del cambio siguen leyéndose de su instantánea. Mientras exista la maquinaria de baseline, un `ParseErr` también lo bloquea `declarationCauses`; el archivado cita entonces dos motivos para la misma causa, y la duplicación desaparece cuando P6 retire esa función.

**Alternativa descartada:** *bloquear todas las fases.* Una errata en `axiom.yaml` pararía la planificación y la implementación, en contra del criterio que ya aplica `blockArchiveOnDeclarations` (REQ-24.18).

**Motivo:** cumple «el cambio NO se trata como no ISO» sin inventar el valor: nadie archiva mientras la declaración sea ilegible.

### D9. Perfil de bug: el artefacto de specs se cumple sin delta

**Elección:** en `resolveOpenSpecStatus`, justo después de calcular `artifacts`, si el artefacto `specs` está `missing` (sin delta ni ficheros en `specs/`), se analiza `proposal.md` con `isoevidence.ParseBugDeclaration`:

| Resultado | Efecto |
|---|---|
| Sin `## Informe de bug` ni `## Impacto en la spec` | Nada: estado idéntico al actual |
| Declaración válida (requisito existente o sin impacto) | `artifacts["specs"]` pasa a `done` |
| Declaración incompleta | `specs` sigue `missing`, con un motivo por lo que falta (tabla siguiente) |

| Situación sin delta | Token | El detalle nombra |
|---|---|---|
| `## Informe de bug` sin `Observado:` o sin `Esperado:`, o con valor vacío | `bug_report_incomplete` | la línea que falta o está vacía |
| `## Impacto en la spec` presente y `## Informe de bug` ausente | `bug_report_incomplete` | la sección `## Informe de bug` que falta |
| `## Impacto en la spec` sin la línea literal `Sin impacto en la spec` | `bug_no_impact_incomplete` | la línea literal que falta |
| `## Impacto en la spec` sin `Motivo:` o con motivo vacío | `bug_no_impact_incomplete` | la línea `Motivo:` |

Si faltan cosas de las dos secciones, salen los dos tokens.

Con delta (`specs` en `done` o `partial`), las secciones no se analizan (regla 1 de REQ-29.15).

**Alternativa descartada:** *un artefacto nuevo `bugReport` en el mapa `artifacts`.* Cambia el dominio de claves de un mapa que los consumidores recorren.

**Motivo:** un único punto de cambio; todo lo demás (dependencias, ruta, `coreReady`) se deriva solo. El motivo de una declaración incompleta va por el canal `genuine`, que la ruta `spec` conserva, con el mismo precedente que `openSpecSpecsLayoutReason`.

### D10. Formato de la matriz: sección de nivel 2 con tabla identificada por cabeceras

**Elección:** la matriz es la tabla bajo el encabezado exacto `## Matriz de verificación` de `verify-report.md`. Las columnas se reconocen por su nombre (§5.5), no por su posición. Solo cuenta un encabezado de nivel 2 fuera de vallas. Dos encabezados de nivel 2 son una matriz ambigua e inválida.

Conservar una matriz histórica (REQ-28.8): cuando el verify vigente sustituye a uno anterior, la matriz anterior se mueve dentro de la sección de contexto histórico y su encabezado **baja a nivel 3** con fecha: `### Matriz de verificación (histórica, AAAA-MM-DD)`. Esa regla se escribe en `_shared/sdd-iso-convention.md` y en `sdd-verify/references/report-format.md` (sección *Historical context*), con un pin de texto, para que un informe con historial nunca tenga dos secciones `##` de matriz.

**Alternativas descartadas:**

- *Dos tablas (automáticas y manuales).* Duplica el reconocimiento de cabeceras y complica la unión por escenario.
- *Datos de ejecución dentro de una sola celda.* El subanálisis de celdas libres es frágil.
- *Bloque YAML dentro del informe.* Ilegible para un auditor y ajeno al estilo del informe.

**Motivo:** es la forma más simple de validar sin ambigüedad, legible en cualquier visor de Markdown y sin ninguna línea que empiece por `verdict:` (cada fila empieza por `|`).

### D11. Resolución de referencias con `os.Root` y `go/parser`

**Elección:**

- Raíz del árbol: la carpeta maestra (`Location.Master`) en toda topología (hueco H1, aplicado). En `monorepo-embedded` coincide con la raíz del repositorio; en `multirepo` admite `<repo>/ruta:símbolo`.
- `ruta/fichero:símbolo`: la ruta, relativa y con `/`, se abre con `os.Root` sobre la raíz del árbol, que rechaza rutas absolutas, `..` y enlaces simbólicos que salgan del árbol. Lectura acotada a 4 MiB. El símbolo debe aparecer como identificador completo: ni el carácter anterior ni el siguiente son letra, dígito, `_` o `$`.
- `TestX` (nombre `Test` seguido de nada o de un carácter que no sea minúscula): recorrido único del árbol, omitiendo `.git`, `vendor`, `node_modules` y `testdata`, sobre los `*_test.go`; prefiltro textual `func TestX(` y confirmación con `go/parser` de una `FuncDecl` de nivel superior sin receptor. El índice se construye una sola vez por validación.

**Alternativa descartada:** *`filepath.Join` y comprobación léxica de `..`.* No detecta un enlace simbólico que salga del árbol.

**Motivo:** contrastable (la prueba existe) y seguro (nunca se lee fuera del árbol ni se ejecuta nada).

### D12. La matriz se evalúa siempre que el archivado estaría disponible

**Elección:** en un cambio ISO, `isoevidence.ValidateMatrix` se llama siempre que `resolveDependencies` dejaría `dependencies.archive` en `ready`, es decir, con `coreReady` y `apply` no bloqueado, **también con tareas pendientes**: `status.go:1726-1728` permite el archivado explícito de trabajo incompleto. Si la regla falla (matriz ausente, inválida o con filas `fail`), `archive` pasa a `blocked`:

| Estado de `apply` | Dónde va el motivo | Ruta |
|---|---|---|
| Pendiente (`ready`) | solo `notes` | sin cambios (`apply`) |
| Terminado (`all_done`) | `blockedReasons` | la de D19 |

El resultado se proyecta además en `iso.matrix` (§5.2), que el orquestador usa para decidir si el verify del rol `fullstack` es limpio (REQ-28.6) sin leer el informe.

**Alternativas descartadas:**

- *Evaluar solo con `apply` terminado.* Un cambio ISO con tareas pendientes se archivaría sin matriz (archivado explícito).
- *Poner siempre el motivo en `blockedReasons`.* Prohibiría `apply` durante toda la implementación.

**Motivo:** REQ-29.11 no admite excepciones por tareas pendientes, y el patrón de `blockArchiveOnDeclarations` ya resuelve la convivencia con `apply`.

### D13. El archivado ISO se niega en Go antes de mover la carpeta y antes de componer

**Elección:** una sola función, `sddstatus.ISOArchiveRefusal(workspaceRoot, changeRoot) []isoevidence.Finding`, aplica la regla del archivado ISO: declaración válida (D8), escenarios de todas las deltas (REQ-29.10) y matriz válida y de éxito (REQ-29.11), con independencia del estado de `apply` (D12). La usan tres consumidores, de modo que ninguno discrepa:

| Consumidor | Cuándo | Efecto |
|---|---|---|
| `axiom sdd status` | siempre | estrecha `archive` (D6, D12) |
| `axiom sdd archive-check --cwd <ruta> --change <c> [--json]` (verbo nuevo, de solo lectura) | lo invoca la skill `sdd-archive` antes de componer y antes de cualquier `git mv` | sale `1` con los motivos tipados si hay hallazgos; `0` en un cambio no ISO o sin hallazgos. Cubre los casos sin composición: capacidad nueva que se copia y bug sin delta. Informa además, por dominio, `alreadyComposed` para reanudar un archivado interrumpido (D17) |
| `axiom sdd archive-compose` (modos `--canonical` y `--document`) | al componer | se niega con los motivos antes de leer la canónica y sin escribir; defensa en profundidad |

`archive-compose` determina el cambio con `changeRootOfDelta(loc, deltaPath, domain)`, la misma función que usa la guarda A (`guardComposeDelta`, `sdd_writer_guard.go:59`). En el modo `--canonical` localiza el workspace y toma el dominio de la carpeta de la delta, igual que `guardComposeLooseDelta`. Sin cambio determinable no aplica nada (último escenario de REQ-29.10). El modo `--change --in-place` no se toca: solo existe para la baseline declarada.

La skill `sdd-archive` gana un paso obligatorio en *Archive Readiness*: «run `axiom sdd archive-check --cwd <workspaceRoot> --change <name>` and STOP with `blocked` on a non-zero exit, quoting its reasons, before any composition or move». Un pin de texto lo fija.

**Alternativas descartadas:**

- *Un verbo nuevo `axiom sdd archive` que haga el movimiento.* Cambia el flujo de archivado, que REQ-29.17 obliga a conservar.
- *`archive-compose --check`.* `archive-compose` exige una delta y un dominio, y en un bug sin delta no hay ninguno; y `--change` ya activa el modo `--in-place`.
- *Confiar solo en `dependencies.archive`.* Deja la negativa en manos de la prosa en los casos sin composición.

**Motivo:** el movimiento lo sigue haciendo la skill con `git mv`, pero la decisión de si puede moverse la toma Go. El riesgo residual (que la skill no invoque el verbo) queda acotado por el pin de texto (R4). El nombre del verbo forma parte del hueco H5.

### D14. Traza con un verbo de solo lectura `axiom sdd trace`

**Elección:** `axiom sdd trace --cwd <ruta> --change <c> [--integration-branch <ref>] [--json]`. La lógica vive en `isoevidence/trace.go` detrás de un puerto `History`; la implementación del puerto en `cli` usa `gitcommon.Run`, el único punto de ejecución de git. Algoritmo en §6.4.

**Alternativas descartadas:**

- *Reutilizar `integration.yaml` y la traza de `internal/specbaseline`.* La spec la excluye y P6 la retira.
- *Añadir la traza a `axiom sdd status`.* El status no debe lanzar `git log` en cada consulta, y la traza tiene sentido sobre todo con el cambio ya archivado.

**Motivo:** cumple REQ-29.9 sin estado nuevo, y es el único sitio donde un auditor puede pedir la cadena completa. El nombre del verbo es la recomendación del hueco H5.

### D15. QA diferido: partición por política en el estado y en el aviso

**Elección:**

- `verifyDependencyFromRoleGates` (`governance_handoff.go`) recorre `governance.Kickoff.Config.Roles` y separa bloqueantes de no bloqueantes. `verify` se bloquea solo si falta algún `role-apply` aprobado de un rol `blocking`. Devuelve además notas no bloqueantes que nombran cada rol `deferred` u `optional` sin aprobar (rol, política y estado: `pendiente` o `rechazado`).
- `kickoff` gana `LastBlockingRoleClosed(roster, gates)` y `PendingNonBlockingRoles(roster, gates)`. `lastRoleClosedForGovernance` se sustituye por la primera.
- `cli/sdd_gate.go` emite el aviso de REQ-21.13 solo cuando la compuerta aprobada es de un rol `blocking` y con ella quedan aprobados todos los bloqueantes; el aviso nombra los no bloqueantes pendientes. El `--json` conserva `lastRoleClosed` (ahora «último bloqueante cerrado») y añade `pendingNonBlockingRoles` con `omitempty`.
- `kickoff.HandedOffTo` usa `LastBlockingRoleClosed` para decidir cuándo la salida 2 cede `verify`.
- `kickoff.IntermediateMerge` (`handed_off.go`) considera solo los roles `blocking` al decidir si una fusión tras una salida 2 es intermedia (hueco H4, aplicado). Sin ese ajuste, un rol `deferred` que nunca se aprueba haría que `axiom branch ensure --archive` rechazara el archivado como fusión intermedia. `internal/branch` no cambia: sigue llamando a la misma función.

**Alternativa descartada:** *reutilizar la barrera `multirole.EvaluateBarrier`.* Mira tareas y veredictos, no compuertas; mezclar ambas cosas cambiaría la semántica de REQ-21.15.

**Motivo:** misma partición que la barrera (REQ-2.1) con el mínimo de código. Con todos los roles `blocking`, o con el rol único `fullstack`, el resultado es idéntico al actual.

### D16. Doctrina con fuente única, generada en Go donde ya lo está

**Elección:** la guía detallada (formato de `Cubre:`, matriz, referencias, perfil de bug con sus textos literales y traza) vive en un fichero compartido nuevo, `internal/assets/skills/_shared/sdd-iso-convention.md`. Las skills `sdd-propose`, `sdd-spec`, `sdd-tasks`, `sdd-verify` (y `references/report-format.md`) y `sdd-archive` lo citan con una línea. La doctrina del orquestador cambia **solo dentro de secciones existentes** de `sdd-orchestrator-sections.md` (sin secciones ni marcadores nuevos), así que `orchestrator_no_drift_test.go` y las 12 plantillas no cambian.

La doctrina de D4 (REQ-29.18) se genera en Go, en los puntos que hoy enumeran cómo se entra en SDD, para que ninguno la contradiga:

| Fichero | Texto que cambia | Contenido nuevo |
|---|---|---|
| `agentguidance/routing.go`, paso 0 | Se antepone una condición a la lectura del carril, y la frase «Small, understood work stays small in every lane» gana la excepción ISO | El texto propuesto, abajo |
| `agentguidance/routing.go`, preámbulo y viñeta **Optional SDD** | «SDD is entered only by an explicit request, by the project lane… or by an accepted proposal» | «…by the project lane declared in `axiom.yaml` (including an ISO project, `iso.project: iso`), or by an accepted proposal» |
| `agentguidance/delegation.go`, **Optional SDD trigger** | «Select SDD only after an explicit request, the project lane `workflow.lane: sdd`…» | Se añade «or an ISO project (`iso.project: iso`), for all work»; por ser la fuente única, el cambio llega igual al bloque de routing, a los 12 orquestadores y al prompt de OpenCode |

Texto propuesto para el inicio del paso 0 (inglés, como el resto del bloque; los literales se conservan tal cual):

> First read `axiom sdd status --cwd <repo> --json`. When its `iso` field projects `project: iso`, the project is ISO and only uses SDD: all work, including small, understood work, goes through the SDD lane without asking the lane question, also when `workflow.lane` is `ask`, and no `odd/tasks/<feature-name>.md` is created. A bug in an ISO project always goes through SDD with the bug profile, along one of three routes: when it restores a requirement that already exists, no delta and a `Cubre:` line citing that requirement; when it touches no requirement, no delta and a `## Impacto en la spec` section holding the line `Sin impacto en la spec` and a `Motivo:` line; when the requirement was wrongly specified or missing, a delta, as a normal change. The first two routes also carry a `## Informe de bug` section with `Observado:` and `Esperado:` lines in `proposal.md`. Otherwise, read the project lane from the `workflow` field: …

La frase actual «Small, understood work stays small in every lane and creates no work artifact» pasa a «…in every lane, except in an ISO project (above)». La sección «SDD Change Kickoff and Block Gates» repite la regla en su párrafo **SDD lane selected**. Ningún texto menciona la combinación `iso: true` con `lane: odd`: la rechaza la validación (D18).

**Literal de la política de selección.** `checkRoutingPolicy` (`routing.go:23`) solo admite `capabilitymanifest.SDDSelectionExplicitRequestProjectLaneOrAcceptedProposal` (`explicit_request_project_lane_or_accepted_proposal`). **El literal no cambia:** un proyecto ISO es una declaración del proyecto de la misma familia que el carril (ISO fija el carril SDD), así que la política sigue siendo «petición explícita, carril del proyecto o propuesta aceptada»; cambia solo la prosa que la explica. Cambiar el literal obligaría a regenerar los digests del manifiesto canónico y a migrar la política de todos los adaptadores sin ganar nada. Prueba: el pin de `routing_stops_test.go` sobre `checkRoutingPolicy` sigue igual, y un pin nuevo comprueba que el bloque renderizado con esa misma política nombra la vía ISO.

**Alternativas descartadas:**

- *Repetir la guía en cada skill.* Cinco copias de los textos literales que el carril ODD reutilizará.
- *Una sección de orquestador nueva.* Obliga a añadir el marcador a las 12 plantillas, y aporta poco: el orquestador solo necesita saber qué proyecta el status y qué hacer con los motivos.

**Motivo:** paridad entre runtimes por construcción (todo `_shared/` se proyecta tal cual) y un solo sitio para los pines de texto.

### D17. Unicidad de un ADDED con la tolerancia acotada del archivado interrumpido (REQ-29.3 enmendado, H6)

**Elección:** un ADDED cuyo `REQ-<n>.<m>` ya figura en la baseline bloquea con `iso_requirement_id`, **salvo** que la baseline del mismo dominio contenga un requisito con ese identificador, el mismo nombre y el mismo bloque byte a byte que el bloque ADDED (fila enmendada de REQ-29.3). La comparación reproduce exactamente lo que hace el compositor nativo al insertar un ADDED (`ComposeOpenSpecCanonicalSpecWithOptions`, `segments = … Text: ensureTrailingNewline(r.Text)`): se compara `ensureTrailingNewline(bloque de la delta)` con el texto del segmento de la canónica que devuelve `parseSpecDocument`. No se normaliza nada más. Cualquier diferencia de nombre, de bloque o de dominio sigue bloqueando.

Además se conserva el orden de comprobación, que hace la tolerancia innecesaria en el camino normal:

- `axiom sdd archive-check` se ejecuta **antes** de la primera composición (D13);
- `ISOArchiveRefusal`, que usan `archive-check` y `archive-compose`, **no** evalúa REQ-29.3: solo la declaración, los escenarios y la matriz.

**Archivado interrumpido, de principio a fin.** Si el archivado se interrumpe después de componer el dominio `pagos` y antes de mover la carpeta:

1. `axiom sdd status` no proyecta `iso_requirement_id` para el ADDED ya compuesto, así que la ruta **no vuelve a `spec`**: sigue la de archivado.
2. `axiom sdd archive-check --json` informa, por dominio, `alreadyComposed: true` cuando la canónica ya contiene la delta entera: todo ADDED presente con el mismo nombre y bloque, todo MODIFIED con el bloque de la delta, todo REMOVED ausente (o marcado como superado con `--supersede`) y todo RENAMED con el nombre nuevo presente y el antiguo ausente. La composición nativa es atómica por dominio (fichero temporal más `mv`), así que un dominio nunca queda compuesto a medias.
3. La skill `sdd-archive` compone solo los dominios con `alreadyComposed: false` (volver a componer `pagos` fallaría con «a requirement named … already exists») y termina el movimiento. La convención ISO describe este paso de reanudación.

**Alternativas descartadas:**

- *Tolerar por mismo nombre, sin comparar el bloque.* Dejaría pasar un ADDED que reutiliza el identificador de otro requisito con distinto contenido.
- *Hacer idempotente la composición nativa.* Cambia `openspec_archive_compose.go`, compartido con upstream, y el comportamiento byte a byte de REQ-24.38.

**Motivo:** el usuario aprobó la tolerancia acotada (H6). La comparación byte a byte con la misma normalización que el compositor garantiza que solo se tolera el propio requisito ya compuesto.

### D18. Validación cruzada `sdd.iso` con `workflow.lane` en la carga por capas

**Elección:** la regla vive en `internal/workspace/sdd.go` como `checkISOWorkflow(sdd *SDDConfig, workflow *WorkflowConfig) error` y se llama desde `readDeclarationBlocks` (`declarations.go`) una vez decodificados los dos bloques, solo si ambos son válidos por separado. Rechaza únicamente `sdd.iso: true` con `workflow.lane: odd`; el error envuelve el centinela `ErrISOWorkflowContradiction`, nombra `sdd.iso` y `workflow.lane` y da las dos correcciones (retirar o fijar en `sdd` el carril, o retirar o poner en `false` `iso`). Se guarda en el canal `SDDErr` y deja `Declarations.SDD` en `nil`. `workflow.go` no cambia y `WorkflowErr` sigue en `nil`, de modo que la decodificación de `workflow:` es la de siempre.

Efectos, todos derivados de ese único canal:

| Consumidor | Efecto |
|---|---|
| `LoadConfig`, `ParseConfig`, `axiom workspace validate` | `joinLoadErrors` ya suma `SDDErr`: error accionable y salida `1` |
| `kickoff seal` (sin `--infer`) | Se niega por `SDDErr` antes de crear la carpeta o escribir `kickoff.yaml` (D5) |
| Status, proyección `iso` | `iso.project: "invalid"` y, con cambio seleccionado, `blocked(iso_declaration_invalid)` en el archivado (D8) |
| Status, proyección `workflow` | `projectWorkflow` (`sddstatus/workspace_declarations.go`) devuelve `ask` / `invalid` cuando `errors.Is(SDDErr, ErrISOWorkflowContradiction)`: no asume el carril `odd`, y el orquestador formula la pregunta de carril, que es el fallo cerrado ya existente |

**Alternativas descartadas:**

- *En `decodeWorkflow` o en `workflow.go`.* Mezcla dos bloques en el decodificador de uno y obliga a `workflow:` a conocer `sdd:`, en contra de «REQ-27.1 sin MODIFIED».
- *En `validateBaseConfig`.* Esa validación solo ve la configuración base; los bloques `sdd:` y `workflow:` se leen como declaraciones.
- *Canal de error propio (`CrossErr`).* Añade un campo más a `Declarations` y a `joinLoadErrors`, los dos puntos de solape con P6, para el mismo comportamiento.
- *Marcar `workflow` como inválido para cualquier `SDDErr`.* Una errata en `sdd:` no debe cambiar el carril del proyecto; solo la contradicción lo hace.

**Motivo:** un solo punto de detección y un solo canal de error cubren la carga, la validación, el sellado y el status, igual que los demás bloques de `axiom.yaml`.

### D19. En un cambio ISO, `verify` es obligatorio y el status ruta a él

**Elección:** con `apply` terminado y la regla de la matriz fallando, `resolveOpenSpecStatus` sustituye la ruta ordinaria (`archive` o `resolve-blockers`) por:

| Situación de la matriz | `nextRecommended` | Motivo |
|---|---|---|
| Ausente (`iso_matrix_missing`) o inválida (`iso_matrix_invalid`) | `verify` | en `blockedReasons`, con la continuación «run the `sdd-verify` phase to write or repair `## Matriz de verificación` in `verify-report.md`» |
| Válida con filas `fail` (`iso_matrix_failed`) | `resolve-blockers` | en `blockedReasons`, con la continuación «fix the failing behaviour, then run `sdd-verify` again» |

Precedencia con las compuertas: la ruta `verify` sustituye a `await-gate` solo cuando la compuerta abierta es `integration`, porque la matriz forma parte del PR del cambio y debe existir antes de esa fusión. Las compuertas `spec`, `design`, `tasks` y `role-apply:<rol>` conservan su precedencia. `verify` ya es un valor admitido de `nextRecommended` (`status_v2.go:331`), así que el contrato no cambia.

La sección «Dependency Graph» de `sdd-orchestrator-sections.md` (fuente única) añade, sin tocar la regla general: «In an ISO change (`iso.change: iso`), verify is mandatory before archive: status routes to `verify` while the verification matrix is missing or invalid, and archive stays blocked until it is valid and successful. Every other change keeps verification opt-in.» En `continuous`, el orquestador encadena `verify` porque sigue `nextRecommended`, y la cadena no se detiene.

**Alternativa descartada:** *dejar `resolve-blockers` con un motivo que nombre `sdd-verify`.* En `continuous` el orquestador se detiene ante `resolve-blockers`; la cadena ISO quedaría parada siempre al final.

**Motivo:** sin esta ruta, nadie produce la matriz en ISO, porque la doctrina actual presenta `verify` como opcional.

### D20. Compuerta `spec` de un bug sin delta: juzga `proposal.md`

**Elección (hueco H3, aplicado):** en un cambio `checkpointed` declarado como bug sin delta, la huella de la compuerta `spec` se calcula sobre `proposal.md`, que contiene el `## Informe de bug` y, en su caso, la marca `Sin impacto en la spec`. El cálculo se añade en los dos sitios que ya calculan huellas: `governanceArtifactInputs` (`sddstatus/governance.go`) y el del CLI (`internal/cli/sdd_gate_digest.go`), con el mismo criterio de «bug sin delta» que D9. Con delta, la compuerta juzga las specs como hoy.

**Alternativa descartada:** *no abrir la compuerta `spec` en un bug.* Se saltaría en silencio y desaparecería una revisión registrada que interesa a un auditor ISO.

**Motivo:** REQ-29.1 exige que un cambio ISO tenga las mismas compuertas; un bug no debe perder la suya.

### D21. Almacenes sin ficheros: fallo cerrado en ISO

**Elección (hueco H2, aplicado):** el perfil de bug y las reglas ISO solo se evalúan en el resolutor OpenSpec (almacenes `openspec` e `hybrid`). Si un cambio con instantánea ISO se resuelve con almacén `engram` o `none`, `decorateISO` lo detecta (lee la instantánea con `kickoff.Load` cuando el resolutor no la evaluó) y estrecha `archive` con `blocked(iso_store_unsupported)`, cuya continuación es declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`.

**Alternativa descartada:** *tratar el cambio como no ISO.* Archivaría sin evidencia un cambio sellado como ISO.

**Motivo:** fallo cerrado, coherente con D8.

---

## 3. Flujo de datos

### 3.1 Status de un cambio (resolutor OpenSpec)

```
axiom sdd status <c>
   │
   ├─ workspace.Locate ─────────────── Declarations{SDD, SDDErr, ...}
   │
   └─ resolveOpenSpecStatus
        ├─ artifacts (ficheros)
        ├─ [D9] specs == missing ──► isoevidence.ParseBugDeclaration(proposal.md)
        │                              ├─ válida      → artifacts.specs = done
        │                              └─ incompleta  → blocked(bug_*)
        ├─ loadGovernance ──► Kickoff (instantánea: IsISO())
        ├─ evaluateISOChange (solo si IsISO)
        │     ├─ baseline index  ◄── parseSpecDocument(openspec/specs/*/spec.md)
        │     ├─ delta model     ◄── parseOpenSpecDelta(changes/<c>/specs/*/spec.md)
        │     ├─ etapa specs:  CheckRequirementIDs, CheckScenarios
        │     ├─ etapa tasks:  ParseCubre(tasks*.md) → CheckCoverage / CheckBug
        │     └─ etapa archive (archive saldría ready, también con tareas pendientes):
        │           ISOArchiveRefusal → ParseMatrix(verify-report.md) → ValidateMatrix(árbol = maestra)
        ├─ coreReady &&= !specsBlocked && !tasksBlocked
        ├─ resolveDependencies → [D15] verify por gate_policy
        │                      → archive por gobierno → archive por ISO
        ├─ [D19] apply all_done y matriz ausente/inválida → nextRecommended = verify
        └─ blockedReasons (genuine + ISO si la ruta sería archive) ; notes (ISO de archive) ; wrong_branch delante
   │
   └─ decorateWorkspaceDeclarations (sin cambios, salvo projectWorkflow de D18)
   └─ decorateISO(status, location)          [nuevo, desde Resolve]
        ├─ iso.project (iso | not_iso | invalid) e iso.reason
        ├─ nota de discrepancia instantánea ≠ proyecto
        ├─ nota de declaración inválida, con o sin cambio
        ├─ invalid + cambio → archive blocked(iso_declaration_invalid)
        └─ [D21] cambio ISO con almacén engram/none → archive blocked(iso_store_unsupported)
```

### 3.2 Sellado

```
kickoff seal --change x
   ├─ ParseSealArgs
   ├─ resolveSealChangeRoot ── nombre válido, no archivado, ¿existe?
   ├─ (sin --infer) Declarations.SDDErr ≠ nil (incluida la contradicción con workflow.lane) ──► rechazo, nada escrito
   ├─ ToKickoff + applySealBranchPolicy ── wrong_branch ──► rechazo, nada escrito
   ├─ SDD = &SDDSnapshot{ISO:true} solo si sdd.iso == true (nunca con --infer)
   ├─ ValidateForSeal
   ├─ PrepareChangeRoot (si no existía) ──┐
   └─ kickoff.Seal ── error ──► rollback ◄┘ (solo directorios propios y vacíos)
```

### 3.3 Archivado y traza

```
sdd-archive (skill)
   ├─ status: dependencies.archive ── blocked ──► STOP
   ├─ axiom sdd archive-check ── ISOArchiveRefusal ── hallazgos ──► salida 1 ──► STOP (antes de componer o mover)
   ├─ archive-compose ── changeRootOfDelta → ¿cambio ISO? ──► ISOArchiveRefusal ── hallazgos ──► rechazo sin escribir
   └─ git mv a changes/archive/<fecha>-<c>/ ; PR documental ; merge

axiom sdd trace --change c
   ├─ ref = git.integration_branch (instantánea o axiom.yaml)
   ├─ commit de código   ◄── git log --first-parent --diff-filter=A <ref> -- changes/<c>/verify-report.md (+ tasks)
   ├─ commit de archivo  ◄── git ls-tree <ref> changes/archive/ → <fecha>-<c> → git log --first-parent --diff-filter=A
   ├─ contraste pr_merged ◄── gates.yaml + git merge-base --is-ancestor
   └─ cadena REQ → Cubre → fila ◄── artefactos archivados (isoevidence)
```

---

## 4. Cambios de ficheros

Tamaños: líneas autoradas previstas (creadas más borradas), sin goldens. Pieza = número de §3.1 de la propuesta.

| Fichero | Acción | Pieza | Descripción | Líneas |
|---|---|---|---|---|
| `internal/workspace/sdd.go` | Crear | 4 | `SDDConfig`, `sddKeys`, `decodeSDD` (booleano estricto), `ErrISOWorkflowContradiction`, `checkISOWorkflow` | 80 |
| `internal/workspace/sdd_test.go` | Crear | 4 | Tabla: válido, clave desconocida, `quizás`, nulo, lista, comillas, duplicada; matriz de combinaciones `iso` × `lane` (`odd`, `sdd`, `ask`, sin clave, sin bloque); `workspace validate` con salida `1` | 170 |
| `internal/workspace/declarations.go` | Modificar | 4 | Campos `SDD`/`SDDErr`, una rama en `readDeclarationBlocks` y la llamada a `checkISOWorkflow` tras decodificar `sdd:` y `workflow:` (**solape P6**) | 10 |
| `internal/sddstatus/workspace_declarations.go` | Modificar | 4 | `projectWorkflow`: `ask` / `invalid` con la contradicción (**solape P6**) | 4 |
| `internal/workspace/loader.go` | Modificar | 4 | `SDDErr` en `joinLoadErrors` y en el comentario de `ParseConfig` (**solape P6**) | 4 |
| `internal/hub/init.go` y su prueba | Modificar | 4 | `declares()` cuenta `SDD` y `SDDErr`, para que `axiom init --force` no pierda `sdd:` (**solape P6**) | 27 |
| `internal/kickoff/types.go` | Modificar | 3 | `Kickoff.SDD *SDDSnapshot`, `SDDSnapshot`, `IsISO()` | 18 |
| `internal/kickoff/seal.go` | Modificar | 3 | `PrepareChangeRoot(changeRoot) (rollback func(), err error)` | 35 |
| `internal/kickoff/machine.go` | Modificar | 1 | `LastBlockingRoleClosed`, `PendingNonBlockingRoles` | 45 |
| `internal/kickoff/handed_off.go` | Modificar | 1 | `HandedOffTo` con `LastBlockingRoleClosed`; `IntermediateMerge` solo con roles `blocking` (H4) | 11 |
| `internal/kickoff/*_test.go` | Modificar | 1, 3 | Instantánea `omitempty`, lectura sin clave, decodificación del esquema previo, partición por política, `IntermediateMerge` con un rol `deferred` pendiente, `PrepareChangeRoot` con retirada | 240 |
| `internal/branch/engine/writer_inventory_test.go` | Modificar | 3 | `{"internal/kickoff", "PrepareChangeRoot"}` en `writerCalls`; `runSDDKickoffSeal` lo cubre con `applySealBranchPolicy` | 3 |
| `internal/cli/sdd_kickoff.go` | Modificar | 3 | `resolveSealChangeRoot`, lectura de `sdd:`, creación y retirada, `iso` en texto y JSON | 75 |
| `internal/cli/sdd_kickoff_test.go` | Modificar | 3 | Invierte `TestRunSDDKickoffSealOnNonexistentChangeNamesResolvedPath` (líneas 239-253) y añade los escenarios de REQ-29.12 y REQ-29.13 | 200 |
| `internal/cli/sdd_gate.go` | Modificar | 1 | Aviso al cerrar el último bloqueante, con no bloqueantes pendientes | 35 |
| `internal/cli/sdd_gate_handoff.go` | Modificar | 1 | `pendingNonBlockingRoles` en el JSON | 6 |
| `internal/cli/sdd_gate_test.go` | Modificar | 1 | Escenarios de REQ-21.13 | 90 |
| `internal/cli/sdd_gate_digest.go` | Modificar | 8 | Huella de la compuerta `spec` sobre `proposal.md` en un bug sin delta (D20, H3) | 25 |
| `internal/sddstatus/governance_handoff.go` | Modificar | 1 | `verifyDependencyFromRoleGates` por `gate_policy`, con notas | 40 |
| `internal/sddstatus/governance.go` | Modificar | 1, 8 | `lastRoleClosedForGovernance` → bloqueantes; huella `spec` sobre `proposal.md` en un bug sin delta (D20) | 43 |
| `internal/sddstatus/role_gates_dependency_test.go` | Modificar | 1 | Escenarios de REQ-21.15 | 90 |
| `internal/sddstatus/governance_test.go` | Modificar | 8 | Compuerta `spec` de un bug sin delta: se abre, se reabre al cambiar `proposal.md` y coincide con la huella del CLI | 60 |
| `internal/isoevidence/doc.go` | Crear | 2, 4-8 | Documentación del paquete y sus capas | 30 |
| `internal/isoevidence/model.go` | Crear | 2, 4-8 | `Requirement`, `Section`, `Finding`, `Code`, `Stage` | 70 |
| `internal/isoevidence/reqid.go` | Crear | 4 | Extracción de `REQ-<n>.<m>`, regla REQ-29.3 y tolerancia del archivado interrumpido | 125 |
| `internal/isoevidence/scenarios.go` | Crear | 2 | Escenarios fuera de vallas, REQ-29.10, nombres repetidos | 70 |
| `internal/isoevidence/bug.go` | Crear | 8 | `ParseBugDeclaration` (REQ-29.15) | 120 |
| `internal/isoevidence/cubre.go` | Crear | 5, 8 | `ParseCubre`, cobertura (REQ-29.4), bug con requisito (REQ-29.5), contradicción (REQ-29.19) | 130 |
| `internal/isoevidence/matrix.go` | Crear | 6 | `ParseMatrix` y `ValidateMatrix` (REQ-29.6 a 29.8 y 29.19) | 260 |
| `internal/isoevidence/testref.go` | Crear | 6 | Resolución de referencias (REQ-29.7) | 140 |
| `internal/isoevidence/trace.go` | Crear | 7 | Puerto `History` y `DeriveTrace` (REQ-29.9) | 150 |
| `internal/isoevidence/*_test.go` | Crear | 2, 4-8 | Tablas por regla; árbol en `t.TempDir()`; puerto `History` simulado | 1.030 |
| `internal/sddstatus/iso.go` | Crear | 2, 4-8 | Adaptador: índice de baseline, modelo de la delta, lectura de tareas e informe, etapas, `ISOProjection` con `reason`, `decorateISO` (notas, `ParseErr`, almacén sin ficheros), ruta `verify` (D19), `ISOArchiveRefusal` | 340 |
| `internal/sddstatus/status.go` | Modificar | 2, 4-8 | Campo `ISO`; llamadas en `resolveOpenSpecStatus` (perfil de bug, etapas, `coreReady`, dependencias, ruta `verify`) y `decorateISO` en `Resolve` | 55 |
| `internal/sddstatus/status_v2.go` | Modificar | 4 | `ISO` en `StatusV2Projection` | 6 |
| `internal/sddstatus/iso_test.go` | Crear | 2, 4-8 | Escenarios de status de REQ-29.1 a 29.19, archivado explícito con tareas pendientes, ruta `verify`, `iso.reason` sin cambio seleccionado, `ParseErr`, almacén `engram` | 460 |
| `internal/cli/sdd_archive_check.go` | Crear | 2, 4, 6, 8 | Verbo `sdd archive-check` de solo lectura sobre `ISOArchiveRefusal`, con `alreadyComposed` por dominio, texto y JSON (D13, D17) | 95 |
| `internal/cli/sdd_archive_check_test.go` | Crear | 2, 4, 6, 8 | Cambio no ISO (salida 0), ISO válido, matriz ausente con tareas pendientes, bug sin delta, capacidad nueva sin composición, declaración inválida, `alreadyComposed` con ADDED, MODIFIED, REMOVED (también con `--supersede`) y RENAMED | 160 |
| `internal/cli/sdd_archive_compose.go` | Modificar | 2, 6 | Llamada a la comprobación ISO antes de leer la canónica (**solape P6**) | 8 |
| `internal/cli/sdd_archive_compose_iso.go` | Crear | 2, 6 | Cambio de la delta con `changeRootOfDelta` (modo `--canonical`, como `guardComposeLooseDelta`) y negativa con `ISOArchiveRefusal` | 40 |
| `internal/cli/sdd_archive_compose_iso_test.go` | Crear | 2, 6 | ISO con y sin escenarios, matriz inválida, sin contexto, salida byte a byte | 130 |
| `internal/cli/sdd_trace.go` | Crear | 7 | Verbo `sdd trace`, puerto `History` sobre `gitcommon.Run`, texto y JSON | 120 |
| `internal/cli/sdd_trace_test.go` | Crear | 7 | Repos git reales en `t.TempDir()`: squash, merge y rebase; discrepancia; ausencia; sin estado escrito | 260 |
| `cmd/axiom` (enrutado de `sdd trace` y `sdd archive-check` y su ayuda) | Modificar | 6, 7 | Registro de los dos verbos junto a los demás `sdd` | 15 |
| `internal/components/agentguidance/routing.go` | Modificar | 12 | D4 en el paso 0 del bloque ODD (condición ISO antes de la pregunta de carril, también el trabajo pequeño, sin `odd/tasks/`, tres rutas del bug), en el preámbulo y en la viñeta **Optional SDD** (**solape P6**) | 14 |
| `internal/components/agentguidance/delegation.go` | Modificar | 12 | Proyecto ISO como vía de entrada en **Optional SDD trigger** (fuente única de la delegación) | 2 |
| `internal/assets/skills/_shared/sdd-iso-convention.md` | Crear | 13 | Fuente única de formatos ISO, perfil de bug, matriz histórica de nivel 3, `archive-check`, reanudación de un archivado interrumpido y traza | 170 |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md` | Modificar | 1, 3, 6, 12, 13 | Párrafos en «Dependency Graph» (verify obligatorio en ISO, D19), «SDD Change Kickoff and Block Gates» y «SDD Independent Review» | 35 |
| `internal/assets/skills/_shared/persistence-contract.md` | Modificar | 6 | Excepción ISO junto a la regla general | 4 |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Modificar | 4-8 | Proyección `iso` (con `reason`), ruta `verify` en ISO y tabla de tokens | 60 |
| `internal/assets/skills/sdd-{propose,spec,tasks,verify,archive,onboard}/SKILL.md` y `sdd-verify/references/report-format.md` | Modificar | 3, 5, 6, 8, 13 | Remisión a la convención; sin carpeta a mano; matriz sin `verdict:`; matriz histórica de nivel 3; paso obligatorio `archive-check` en `sdd-archive` y composición solo de los dominios con `alreadyComposed: false` | 90 |
| Pines de texto en `internal/assets/*_test.go` y `internal/components/agentguidance/*_test.go` | Modificar | 6, 12, 13 | REQ-29.17 y REQ-29.18, `archive-check` en `sdd-archive`, verify obligatorio en ISO, política de selección intacta con la vía ISO renderizada | 180 |
| `docs/manual-de-inicio.md`, `axiom.example.yaml` | Modificar | 13 | Declaración ISO, perfil de bug, traza | 50 |
| `testdata/golden/`, `AGENTS.md`, `GEMINI.md`, `.codex/AGENTS.md` | Regenerar | 13 | Con `-update`, nunca a mano | (no cuentan) |

`internal/multirole/barrier.go`, `internal/specbaseline/`, `internal/branch/` e `internal/workspace/workflow.go` **no cambian**.

---

## 5. Contratos

### 5.1 Tipos de configuración y de instantánea

```go
// internal/workspace/sdd.go
// SDDConfig es el bloque `sdd:` de axiom.yaml (REQ-29.1).
type SDDConfig struct {
	ISO bool
}

var sddKeys = []string{"iso"}

// decodeSDD rechaza: un nodo que no es mapeo (incluido el nulo), claves fuera de
// sddKeys o repetidas, y un `iso` que no es un escalar !!bool (true o false).
// Mensajes en castellano que nombran la clave o el valor y los admitidos.
func decodeSDD(node *yaml.Node) (*SDDConfig, error)

// En Declarations:
//   SDD    *SDDConfig // nil sin bloque, con bloque inválido o con la contradicción de D18
//   SDDErr error

// ErrISOWorkflowContradiction identifica la declaración contradictoria de REQ-29.1:
// sdd.iso: true junto a workflow.lane: odd. errors.Is lo reconoce dentro de SDDErr.
var ErrISOWorkflowContradiction = errors.New("sdd.iso: true es incompatible con workflow.lane: odd")

// checkISOWorkflow devuelve un error que envuelve ErrISOWorkflowContradiction solo
// con sdd.iso true y workflow.lane odd; nil con cualquier otra combinación, también
// sin alguno de los dos bloques. El mensaje nombra las dos claves y las dos correcciones.
func checkISOWorkflow(sdd *SDDConfig, workflow *WorkflowConfig) error
```

Mensaje previsto (castellano, una sola línea):

> `declaración contradictoria: sdd.iso es true y workflow.lane es odd, pero un proyecto ISO tramita todo el trabajo por el carril SDD; retira workflow.lane: odd (o fíjalo en sdd), o retira sdd.iso (o ponlo en false)`

```go
// internal/kickoff/types.go
// SDDSnapshot es la instantánea de la declaración `sdd:` de axiom.yaml al sellar
// (REQ-29.13). Solo existe en un cambio ISO: con iso falso no se escribe.
type SDDSnapshot struct {
	ISO bool `yaml:"iso,omitempty"`
}

// En Kickoff:
//   SDD *SDDSnapshot `yaml:"sdd,omitempty"`

// IsISO dice si el cambio se selló como ISO. Sin bloque, o con iso falso, es no ISO.
func (k Kickoff) IsISO() bool { return k.SDD != nil && k.SDD.ISO }
```

Ejemplo de `kickoff.yaml` de un cambio ISO (el resto, igual que hoy):

```yaml
schema: axiom.sdd-kickoff/v1
change: cobro-iva
# ...
git:
  integration_branch: main
  # ...
sdd:
  iso: true
```

```go
// internal/kickoff/seal.go
// PrepareChangeRoot crea changeRoot y sus padres ausentes, uno a uno, y devuelve una
// función que borra, en orden inverso, solo los que creó y solo si están vacíos. Si
// changeRoot ya existe no crea nada y la retirada no hace nada.
func PrepareChangeRoot(changeRoot string) (rollback func(), err error)
```

```go
// internal/kickoff/machine.go
// LastBlockingRoleClosed: hay al menos un rol blocking y todos tienen su role-apply aprobado.
func LastBlockingRoleClosed(roster []multirole.RoleAssignment, gates []GateState) bool

// RoleGateStatus describe un rol no bloqueante para avisos y notas.
type RoleGateStatus struct {
	Role   string
	Policy multirole.GatePolicy
	Status string // "pending" | "rejected"
}

// PendingNonBlockingRoles: roles deferred u optional cuyo role-apply no está aprobado.
func PendingNonBlockingRoles(roster []multirole.RoleAssignment, gates []GateState) []RoleGateStatus
```

### 5.2 Proyección `iso` del status (v1 interno y v2 público)

```go
// ISOProjection es la proyección aditiva de REQ-29.2. Se omite cuando axiom.yaml no
// tiene bloque `sdd:` y el cambio seleccionado no es ISO.
type ISOProjection struct {
	// Project: "iso" (sdd.iso: true), "not_iso" (false o bloque sin la clave) o
	// "invalid" (bloque inválido, contradictorio con workflow.lane (D18), o
	// axiom.yaml ilegible con bloque presente).
	Project string `json:"project"`
	// Reason: solo con Project "invalid"; el error de SDDErr o de ParseErr y su
	// continuación (`axiom workspace validate --path <maestra>` o la corrección de
	// D18). Presente con o sin cambio seleccionado.
	Reason string `json:"reason,omitempty"`
	// Change: "iso" | "not_iso"; leído de la instantánea. Ausente sin cambio seleccionado.
	Change string `json:"change,omitempty"`
	// Matrix: solo en un cambio ISO con apply terminado.
	Matrix *ISOMatrixProjection `json:"matrix,omitempty"`
}

type ISOMatrixProjection struct {
	// State: "valid" | "invalid" | "missing" | "not_required" (bug sin impacto sin matriz).
	State      string `json:"state"`
	Successful bool   `json:"successful"` // valid (o not_required) y sin filas fail
	Findings   int    `json:"findings"`   // número de hallazgos; el detalle va en blockedReasons
}
```

Ejemplos:

```json
"iso": { "project": "iso" }
"iso": { "project": "not_iso", "change": "iso", "matrix": { "state": "valid", "successful": true, "findings": 0 } }
"iso": { "project": "invalid", "reason": "declaración contradictoria: sdd.iso es true y workflow.lane es odd, …; retira workflow.lane: odd (o fíjalo en sdd), o retira sdd.iso (o ponlo en false)" }
```

`iso` también aparece, con `project: "invalid"`, cuando `axiom.yaml` es ilegible (`ParseErr`): no se puede saber si declara el bloque, y la proyección falla cerrada (D8).

Nota de discrepancia (REQ-29.13), en `notes`, solo si el proyecto es válido y difiere de la instantánea:

> `La instantánea ISO del cambio (sdd.iso: true) difiere de la declaración vigente del proyecto (sdd.iso: false); las reglas del cambio siguen su instantánea.`

### 5.3 Motivos tipados

Forma: `blocked(<token>): <detalle>; <continuación>`. El detalle nombra el dominio, la sección, el requisito, la fila o el fichero que corresponda. Lista cerrada:

| Token | REQ | Ámbito | Etapa | Continuación |
|---|---|---|---|---|
| `bug_report_incomplete` | 29.15 | todo proyecto | specs | completar `Observado:` o `Esperado:`, o añadir `## Informe de bug` en `proposal.md` |
| `bug_no_impact_incomplete` | 29.15 | todo proyecto | specs | añadir la línea `Sin impacto en la spec` o un `Motivo:` no vacío en `## Impacto en la spec` |
| `iso_declaration_invalid` | 29.1, 29.2 | cambio seleccionado con `sdd:` inválido o contradictorio con `workflow.lane` | archive | `axiom workspace validate --path <maestra>`; en la contradicción, el texto de la corrección de D18 |
| `iso_requirement_id` | 29.3 | ISO | specs | añadir o corregir `REQ-<n>.<m>` en el nombre; si el requisito existe sin él, un RENAMED en la delta |
| `iso_requirement_scenarios` | 29.10, 29.6 | ISO | specs | añadir un `#### Scenario:` al requisito; renombrar un escenario repetido |
| `iso_coverage` | 29.4 | ISO | tasks | añadir `Cubre: REQ-x` en alguna tarea |
| `iso_cubre_unknown` | 29.4, 29.5 | ISO | tasks | corregir el identificador o retirarlo |
| `iso_bug_requirement` | 29.5 | ISO | tasks | citar el requisito existente con `Cubre:`, o tramitar como cambio normal con delta si no tiene escenarios |
| `iso_bug_contradiction` | 29.19 | ISO | tasks | retirar el `Cubre:` o retirar la marca y tramitarlo como bug con requisito existente |
| `iso_matrix_missing` | 29.6, 29.11 | ISO | archive | ejecutar la fase `sdd-verify` para escribir `## Matriz de verificación` en `verify-report.md` (ruta `verify`, D19) |
| `iso_matrix_invalid` | 29.6 a 29.8 | ISO | archive | ejecutar `sdd-verify` para corregir la fila nombrada (ruta `verify`, D19) |
| `iso_matrix_failed` | 29.11 | ISO | archive | corregir el comportamiento y repetir `sdd-verify` |
| `iso_store_unsupported` | 29.1, 29.11 (D21, H2) | cambio ISO con almacén `engram` o `none` | archive | declarar `artifact_store: openspec` o `hybrid` en `openspec/config.yaml` |

Reglas comunes:

- Un motivo por hallazgo; varios hallazgos producen varios motivos (REQ-29.16).
- Los motivos de las etapas `specs` y `tasks` van al canal `genuine`.
- Los de la etapa `archive` van siempre a `notes`, y además a `blockedReasons` solo cuando, sin ellos, la ruta sería `archive` (D6, D12).
- Fuera de ISO, ninguno de los tokens `iso_*` aparece ni en `blockedReasons` ni en `notes`.
- `blocked(wrong_branch)` sigue el primero; con la rama equivocada se evalúa igual el ISO, pero la ruta queda en `resolve-blockers` como hoy.
- `axiom sdd continue` refleja los motivos sin cambios: deriva del mismo status.

### 5.4 API de `internal/isoevidence`

```go
package isoevidence

type Section string // "ADDED" | "MODIFIED" | "REMOVED" | "RENAMED"

// Requirement es un requisito leído por el adaptador con el lector nativo.
type Requirement struct {
	Domain  string
	Section Section // vacío para un requisito de la baseline
	Name    string  // en un RENAMED, el nombre nuevo
	OldName string  // solo en un RENAMED
	Text    string  // bloque completo, para extraer escenarios
}

type Stage string // "specs" | "tasks" | "archive"

type Code string // los tokens de §5.3

type Finding struct {
	Code         Code
	Stage        Stage
	Subject      string // requisito, fila, fichero o identificador afectado
	Detail       string
	Continuation string
}

// Reason compone la línea de blockedReasons.
func (f Finding) Reason() string

// Identificadores (REQ-29.3, D17).
func RequirementIDs(name string) (valid []string, malformed []string)
func CheckRequirementIDs(delta, baseline []Requirement) []Finding

// Escenarios (REQ-29.10 y nombres repetidos de REQ-29.6).
func Scenarios(blockText string) []string
func CheckScenarios(delta []Requirement) []Finding

// Perfil de bug (REQ-29.15).
type BugKind string // "none" | "existing_requirement" | "no_impact"
type BugDeclaration struct {
	Kind    BugKind
	Valid   bool
	Missing []string // líneas o secciones que faltan, en orden de aparición esperado
	Motivo  string
}
func ParseBugDeclaration(proposal string) BugDeclaration
func (d BugDeclaration) Findings() []Finding // bug_report_incomplete / bug_no_impact_incomplete

// Cubre (REQ-29.4, 29.5, 29.19).
type CubreLine struct {
	File string
	Line int
	IDs  []string // tal como se escribieron, también los mal formados
}
func ParseCubre(file, text string) []CubreLine
type BaselineIndex map[string][]Requirement // por REQ-<n>.<m>
func NewBaselineIndex(baseline []Requirement) BaselineIndex
func CheckCoverage(delta []Requirement, index BaselineIndex, cubre []CubreLine) []Finding
func CheckExistingRequirementBug(index BaselineIndex, cubre []CubreLine) []Finding
func CheckNoImpactBug(cubre []CubreLine) []Finding

// Escenarios afectados (término de la spec): par (requisito, escenario).
type ScenarioKey struct{ RequirementID, Scenario string }
func AffectedFromDelta(delta []Requirement) []ScenarioKey
func AffectedFromBaseline(index BaselineIndex, ids []string) []ScenarioKey

// Matriz (REQ-29.6 a 29.8 y 29.19).
type Matrix struct {
	Present bool
	Rows    []MatrixRow
	Issues  []Finding // problemas de forma: ambigua, cabeceras, celdas
}
type MatrixRow struct {
	Line                                       int
	RequirementID, Scenario, Type, Reference   string
	Command, ExitCode, Owner, Date, Rationale  string
	Result, Commit                             string
}
func ParseMatrix(report string) Matrix

type MatrixState string // "valid" | "invalid" | "missing" | "not_required"
type MatrixResult struct {
	State      MatrixState
	Successful bool
	Findings   []Finding
}
func ValidateMatrix(m Matrix, affected []ScenarioKey, noImpactBug bool, tree TreeResolver) MatrixResult

// Referencias de prueba (REQ-29.7).
type TreeResolver interface {
	ResolvePathSymbol(relPath, symbol string) (ResolveOutcome, error)
	ResolveGoTest(name string) (bool, error)
}
type ResolveOutcome string // "resolved" | "file_missing" | "symbol_missing" | "outside_tree"
func NewTreeResolver(root string) (TreeResolver, error) // basado en os.Root

// Traza (REQ-29.9).
type History interface {
	FirstParentAdded(ctx context.Context, ref, path string) ([]string, error) // hashes, del más antiguo al más reciente
	TreeHas(ctx context.Context, commit, path string) (bool, error)
	TreeEntries(ctx context.Context, ref, dir string) ([]string, error)
	IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error)
}
type TraceInput struct {
	Change, Ref        string
	PlanningHomePrefix string // ruta del planning home relativa a la raíz git, con «/»
	EvidenceCommit     string // vacío sin evidencia pr_merged
}
type CommitRef struct {
	Commit string
	Found  bool
}
type Trace struct {
	CodeCommit, ArchiveCommit CommitRef
	ArchiveFolder             string
	Discrepancy               string // vacío si coincide o no hay evidencia
}
func DeriveTrace(ctx context.Context, h History, in TraceInput) (Trace, error)
```

### 5.5 Formato de la matriz (`verify-report.md`)

```markdown
## Matriz de verificación

| Requisito | Escenario | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado | Commit |
|---|---|---|---|---|---|---|---|---|---|---|
| REQ-30.1 | Cobro con IVA general | unitario | `internal/pagos/cobro_test.go:TestCobraIVA` | `go test ./internal/pagos -run TestCobraIVA -count=1` | 0 | | | | pass | 1a2b3c4 |
| REQ-30.1 | Cobro exento | integración | `TestCobroExento` | `go test ./internal/pagos -run TestCobroExento -count=1` | 0 | | | | pass | |
| REQ-30.2 | Recibo impreso | manual | acta-12 | | | Ana Pérez | 2026-11-12 | La impresora fiscal no tiene emulador | pass | |
```

| Regla | Detalle |
|---|---|
| Sección | Encabezado exacto `## Matriz de verificación`, nivel 2, fuera de vallas; termina en el siguiente `#` o `##`. Más de uno: `iso_matrix_invalid` (ambigua). |
| Cabeceras | Por nombre, sin importar el orden ni las mayúsculas. Obligatorias: Requisito, Escenario, Tipo, Referencia, Comando, Salida, Responsable, Fecha, Justificación, Resultado. Opcional: Commit (nunca se valida). |
| Celdas | Se recortan los espacios y las comillas invertidas que envuelven el valor; `\|` es una barra literal. Una celda que no aplica queda vacía o con `-`/`—`. |
| Tipo | `unitario`, `integración`, `e2e` o `manual`, exactos. |
| Resultado | `pass` o `fail`, exactos. |
| Fila automática | Referencia resoluble (D11); Comando no vacío; Salida entera; `pass` exige Salida `0`. |
| Fila manual | Responsable no vacío; Fecha `AAAA-MM-DD` válida; Justificación no vacía. Referencia opcional y nunca resuelta; Comando y Salida no se exigen. |
| Completitud | Una fila por escenario afectado; ninguna fila fuera de los afectados; ninguna repetida. |
| Bug sin impacto | Matriz ausente o sin filas: `not_required`, de éxito. Cualquier fila: inválida. |
| Sin `verdict:` | Cada fila empieza por `|`; la sección no contiene ninguna línea que empiece por `verdict:`. |

### 5.6 Formatos de `proposal.md` y de las tareas

```markdown
## Informe de bug

Observado: el total omite el IVA reducido.
Esperado: el total aplica el IVA de cada línea.
Referencia: JIRA-1234

## Impacto en la spec

Sin impacto en la spec
Motivo: el defecto estaba en el script de despliegue, no en un requisito.
```

| Regla | Detalle |
|---|---|
| Encabezados | Texto exacto de nivel 2, fuera de vallas, en cualquier lugar de `proposal.md`. La sección termina en el siguiente `#` o `##`. |
| Líneas del informe | La línea, sin el espacio inicial, empieza por la etiqueta; el valor va en la misma línea. Cuenta la primera aparición de cada etiqueta. |
| Marca | Una línea igual a `Sin impacto en la spec` tras recortar solo el espacio inicial y final; sensible a mayúsculas y sin texto añadido. Las demás líneas no vacías de la sección se ignoran. |
| Orden de evaluación | Con delta, las secciones se ignoran. Sin delta y con `## Impacto en la spec`, ruta sin impacto. Sin delta y sin ella, ruta con requisito existente. |

Línea `Cubre:` en `tasks.md` o `tasks.<rol>.md`: una línea propia, con sangría o con viñeta opcionales, fuera de vallas, con la forma `Cubre: REQ-x[, REQ-y ...]`. La cobertura es conjunta (unión de todos los ficheros de tareas), así que no hace falta asociar cada línea a su tarea; el fichero y la línea se conservan para los motivos.

```markdown
- [ ] 2.1 Calcular el IVA por línea.
  Cubre: REQ-30.1, REQ-30.2
```

---

## 6. Detalle de los analizadores y de la traza

### 6.1 Identificadores (REQ-29.3, REQ-29.14)

- Un token es candidato si empieza por `REQ-` en límite de palabra (`\bREQ-[0-9A-Za-z.]*[0-9A-Za-z]`). Es válido si encaja exactamente con `^REQ-\d+\.\d+$`; si no, es mal formado (`REQ-30`, `REQ-x.1`).
- Por nombre: 0 válidos (con o sin mal formados) es ausente; más de un válido distinto es ambiguo; el mismo válido repetido cuenta como uno.
- ADDED: válido, no repetido entre los ADDED de la delta (todos los dominios) y ausente de la baseline, salvo la tolerancia del archivado interrumpido: mismo dominio, mismo nombre y mismo bloque tras `ensureTrailingNewline` (REQ-29.3 enmendado, D17).
- MODIFIED y REMOVED: el nombre debe llevar uno válido; si no, la continuación es el RENAMED.
- RENAMED: el nombre nuevo lleva uno válido; si el antiguo lo lleva, debe coincidir; si lo añade, se aplica la misma unicidad que a un ADDED.
- Índice de la baseline (REQ-29.14): `openspec/specs/*/spec.md` del planning home, leídos con `parseSpecDocument`; un requisito sin identificador no entra en el índice. Una consulta devuelve las coincidencias; varias con el mismo identificador cuentan como `iso_bug_requirement` («ambiguo en la baseline»).

### 6.2 Cobertura y bug

- Identificadores de la delta: los de ADDED, MODIFIED, REMOVED y el nuevo de cada RENAMED.
- Cobertura: cada identificador de la delta aparece en algún `Cubre:`. Un identificador citado que no está en la delta ni en el índice (también uno mal formado) produce `iso_cubre_unknown`, una vez por identificador.
- Bug con requisito existente: sin ningún `Cubre:`, `iso_bug_requirement` («ningún requisito citado»); citado con cero escenarios, `iso_bug_requirement` con la continuación de cambio normal.
- Bug sin impacto: cada `Cubre:` produce `iso_bug_contradiction` (fichero e identificador).
- Todo esto se evalúa solo cuando existen todos los ficheros de tareas del cambio (`kickoff.RoleTaskFiles` o `tasks.md`).

### 6.3 Status: dónde se engancha cada regla

En `resolveOpenSpecStatus` (`status.go`), en este orden:

1. tras construir `artifacts`: perfil de bug (D9);
2. tras `loadGovernance`: `isoEval := evaluateISOChange(...)` si `governance != nil && governance.Kickoff.IsISO()`; etapas `specs` y `tasks`;
3. `coreReady` añade `&& !isoEval.blocksCore()`;
4. tras `resolveDependencies`: `dependencies.Specs` o `dependencies.Tasks` a `blocked` si su etapa tiene hallazgos;
5. tras `archiveDependencyFromGovernance`: si el cambio es ISO y `dependencies.archive` sigue `ready` (también con tareas pendientes), `ISOArchiveRefusal` y `archiveDependencyFromISO`, que solo estrecha de `ready` a `blocked`;
6. tras `resolveNextRecommended`: con `apply` terminado y matriz ausente o inválida, la ruta pasa a `verify` (D19), también sobre `await-gate` de la compuerta `integration`;
7. los motivos de las etapas `specs` y `tasks` se añaden a `blockedReasons.genuine` después de los existentes; los de `archive` van a `notes` y, si la ruta sería `archive` (es decir, con `apply` terminado), también a `blockedReasons`.

`decorateISO` se llama desde `Resolve`, justo después de `decorateWorkspaceDeclarations`, para que `workspace_declarations.go` solo cambie en `projectWorkflow` (D18); si añade algo, regenera las instrucciones de fase como hace aquella.

El resolutor de Engram no evalúa ni el perfil de bug ni las reglas ISO; un cambio ISO en un almacén sin ficheros se bloquea en `decorateISO` con `iso_store_unsupported` (D21).

### 6.4 Traza por `--first-parent` (REQ-29.9)

| Paso | Orden git | Resultado |
|---|---|---|
| Rama | `git.integration_branch` de la instantánea; si no hay, la de `axiom.yaml` (por defecto `main`); `--integration-branch` la sustituye | `<ref>` validada con `gitref.ValidRefName` |
| Commit de código | `git log --first-parent --reverse --diff-filter=A --format=%H <ref> -- <ph>/changes/<c>/verify-report.md`; del primero al último, el primero en el que `git ls-tree` también encuentra un `tasks*.md` | el commit que trae la carpeta con sus `tasks.md` y `verify-report.md` |
| Carpeta archivada | `git ls-tree --name-only <ref> <ph>/changes/archive/` filtrado con `^\d{4}-\d{2}-\d{2}-<c>$` | `<fecha>-<c>` o ausente |
| Commit de archivado | `git log --first-parent --reverse --diff-filter=A --format=%H <ref> -- <ph>/changes/archive/<fecha>-<c>/kickoff.yaml` (o el primer fichero de la carpeta) | el primero |
| Contraste | registro `approved` de la compuerta `integration` con `evidence_kind: pr_merged` en `gates.yaml` (carpeta activa o archivada); `git merge-base --is-ancestor <código> <evidencia>` | `Discrepancy` si no es ancestro; nunca error |
| Cadena | `isoevidence` sobre los artefactos de la carpeta (archivada o activa): REQ de la delta → `Cubre:` → filas de la matriz | tabla en texto o JSON |

Con squash y con rebase, el commit que trae `verify-report.md` está en la línea `--first-parent`. Con merge, el commit de merge compara contra su primer padre y por eso aparece en el `git log` con ruta. La salida JSON usa el esquema `axiom.sdd-trace/v1`. No escribe nada.

---

## 7. Estrategia de pruebas

TDD según la configuración del proyecto; cada pieza escribe sus pruebas antes del código. Sin binarios sueltos: todo con `go test` acotado por paquete y en proceso.

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitaria | `decodeSDD`: válido, `quizás`, `"true"` entre comillas, nulo, lista, clave desconocida (`archive`, `design_triggers`), duplicada | Tabla en `workspace/sdd_test.go` |
| Unitaria | `checkISOWorkflow` en la carga por capas: solo `iso: true` con `lane: odd` falla, con las dos claves y la corrección; `iso: true` con `sdd`, `ask`, sin clave y sin bloque, e `iso: false` o sin `sdd:` con `odd`, son válidos; `workflow` se resuelve igual | Tabla en `workspace/sdd_test.go` con `ParseConfig` y `ReadDeclarations` |
| Integración | La contradicción en `axiom workspace validate` (salida `1`), en `kickoff seal` (sin `kickoff.yaml` ni carpeta) y en el status (`iso.project: invalid`, `workflow` en `ask` / `invalid`) | `cli` y `sddstatus` |
| Unitaria | Instantánea: `omitempty`, lectura sin clave, decodificación con el esquema previo (struct sin `SDD`), `IsISO` | Tabla en `kickoff` |
| Unitaria | `PrepareChangeRoot`: crea padres, retirada solo de lo propio y vacío, carpeta existente | `t.TempDir()` |
| Unitaria | `LastBlockingRoleClosed` y `PendingNonBlockingRoles`: todos `blocking`, mezcla, sin bloqueantes, rechazado | Tabla en `kickoff` |
| Unitaria | `RequirementIDs` y `CheckRequirementIDs`: cada fila de la tabla de REQ-29.3, incluida la tolerancia (mismo dominio, nombre y bloque) y sus negativos (un byte distinto, otro nombre, otro dominio, falta solo el salto de línea final) | Tabla en `isoevidence` |
| Integración | Archivado interrumpido: tras componer un dominio con el compositor real, el status no proyecta `iso_requirement_id` ni ruta a `spec`, `archive-check` informa `alreadyComposed: true` para ese dominio y `false` para el resto, y el resto se compone y se mueve | `sddstatus` y `cli` |
| Integración | Archivado ISO con tareas pendientes: `archive` bloqueado, motivo solo en `notes`, ruta `apply` sin cambios (D12) | `sddstatus` |
| Integración | Ruta `verify` con `apply` terminado y matriz ausente o inválida, también con la compuerta `integration` abierta; `resolve-blockers` con filas `fail` (D19) | `sddstatus` |
| Integración | `iso.reason` y nota con y sin cambio seleccionado para `sdd:` inválido, contradicción y `ParseErr`; `iso_store_unsupported` con almacén `engram` (D8, D21) | `sddstatus` |
| Integración | `archive-check`: salida 0 fuera de ISO, salida 1 con matriz ausente, bug sin delta y capacidad nueva sin composición | `cli` |
| Integración | Compuerta `spec` de un bug sin delta: misma huella en status y en `gate record` (D20) | `sddstatus` y `cli` |
| Unitaria | `declares()` de `hub/init.go` con solo `sdd:` declarado; `IntermediateMerge` con un rol `deferred` pendiente | `hub` y `kickoff` |
| Unitaria | `Scenarios` y `CheckScenarios`: escenario dentro de una valla, ADDED/MODIFIED sin escenarios, REMOVED/RENAMED exentos, repetidos | Tabla |
| Unitaria | `ParseBugDeclaration`: cada escenario de REQ-29.15, incluidos el punto final y las minúsculas | Tabla |
| Unitaria | `ParseCubre` y las tres comprobaciones: unión por rol, varios identificadores, mal formados, baseline sin delta, contradicción | Tabla |
| Unitaria | `ParseMatrix` y `ValidateMatrix`: cada escenario de REQ-29.6 a 29.8 y 29.19, histórico de nivel 3, ambigua, `\|` en el comando | Tabla con un `TreeResolver` simulado |
| Unitaria | `NewTreeResolver`: forma general, `TestX`, subcadena, absoluta, `..`, enlace simbólico que sale del árbol, fichero mayor del límite, Python y TypeScript | `t.TempDir()`; el caso del enlace se omite si el sistema no permite crearlo |
| Unitaria | `DeriveTrace`: squash, merge y rebase simulados en el puerto, discrepancia, ausencia de carpeta | Puerto `History` simulado |
| Integración | Status: escenarios de REQ-29.1, 29.2, 29.3, 29.4, 29.5, 29.13, 29.15, 29.16, 29.19, REQ-21.15; continuidad del contrato (sin bloque `sdd:` el JSON es idéntico byte a byte) | Workspaces en `t.TempDir()` con los ayudantes de `status_fixture_helpers_test.go`; las pruebas de regresión existentes no cambian |
| Integración | `kickoff seal`: la prueba de `sdd_kickoff_test.go:239-253` se invierte; `wrong_branch` e `sdd:` inválido sin carpeta; idempotencia; carpeta existente; `--infer` sin `iso` | Igual que las pruebas actuales del paquete `cli` |
| Integración | `gate record`: aviso de REQ-21.13 con `deferred` y `optional`; sin segundo aviso | `cli` |
| Integración | `archive-compose`: rechazo ISO sin escribir, salida byte a byte de una composición aceptada, sin contexto de cambio | `cli` |
| Integración | `sdd trace` con git real: tres repos (squash, merge y rebase), PR del cambio y PR documental; ningún fichero nuevo en el árbol | Repos en `t.TempDir()` con `gitcommon.Run`, como las pruebas de `branch`; se omiten con `testing.Short()` si el paquete ya lo hace |
| Activos | Pines de REQ-29.17 y REQ-29.18: textos literales, tres rutas del bug descritas, excepción ISO, sin `verdict:`, matriz histórica de nivel 3, sin carpeta a mano, `archive-check` obligatorio en `sdd-archive`, verify obligatorio en ISO en «Dependency Graph», condición ISO antes de la pregunta de carril, también para el trabajo pequeño, sin `odd/tasks/` con ISO, ninguna regla de precedencia entre `sdd.iso` y `workflow.lane: odd`, política `explicit_request_project_lane_or_accepted_proposal` intacta, paridad entre runtimes y entre proyectos ISO y no ISO | Pruebas de `internal/assets` y `agentguidance`; goldens con `-update` y nueva pasada sin él |
| Arquitectura | `isoevidence` no importa `os/exec` ni paquetes internos | Prueba de importaciones del paquete |

---

## 8. Matriz de amenazas

El cambio ejecuta git (traza), lee ficheros cuyo nombre viene de la entrada (referencias de prueba) y crea directorios a partir de `--change`.

| Frontera | Casos adversarios mínimos | Aplicabilidad | Respuesta del design | Pruebas RED previstas |
|---|---|---|---|---|
| Rutas tipo documentación | `requirements.txt`, `CMakeLists.txt`, Markdown/MDX ejecutable, `README.sh` | Aplicable: la matriz registra comandos y las referencias apuntan a ficheros arbitrarios | Los comandos de la matriz y los ficheros referenciados son datos: nunca se ejecutan ni se interpretan; solo se leen, acotados a 4 MiB | Fila con un comando que crearía un fichero centinela y referencia a un `README.sh`: tras validar, el centinela no existe y la referencia resuelve solo por texto |
| Selección del repositorio git | `git -C`, rutas relativas, rutas absolutas | Aplicable: `sdd trace` ejecuta git | `gitcommon.Run` con `dir` = raíz git del workspace resuelto desde `--cwd`, nunca el directorio del proceso; las rutas de las órdenes se construyen con el planning home relativo y el nombre del cambio ya validado; la referencia, validada con `gitref` | Trace lanzado desde un directorio ajeno con `--cwd` al repo; nombre de cambio con `:(`, `*` o `..` rechazado antes de llamar a git; referencia con `..` rechazada |
| Estado del commit | índice preparado, `commit -a`, índice vacío | N/A: ninguna pieza hace commits; la traza solo usa `log`, `ls-tree` y `merge-base` | — | — (la prueba de «sin estado escrito» de REQ-29.9 cubre la ausencia de escrituras) |
| Estado del push | rama con seguimiento, primer push, refspec explícito | N/A: ninguna pieza hace push | — | — |
| Órdenes de PR | `--head` explícito, prefijo de entorno, órdenes compuestas | N/A: ninguna pieza abre PR | — | — |
| Rutas derivadas de la entrada (fila añadida) | `--change ../x`, `--change archive`, `C:\x:Sym`, `\\srv\x:Sym`, `../x_test.go:TestX`, enlace simbólico hacia fuera | Aplicable: `seal` crea directorios y la matriz abre ficheros | `gitref.ValidBranchComponent` y `archive` reservado antes de crear; `os.Root` para las referencias | Un caso por entrada de la columna anterior: rechazo sin crear nada y referencia no resuelta con el motivo «fuera del árbol» |

---

## 9. Coordinación con P6 (`spec-viva-openspec`)

P6 retirará el bloque `baseline:` y su maquinaria después de que este cambio se fusione.

### 9.1 Piezas que dependen de código que P6 borrará

**Ninguna, por diseño.** Este cambio no llama a `internal/specbaseline`, ni a `axiom sdd integrate`, ni a `composeDocumentMode` con perfil, ni a `declarationCauses`. Sí reutiliza `changeRootOfDelta` (`sdd_archive_compose.go`), que no es maquinaria de baseline: la usa la guarda A (`guardComposeDelta`, `sdd_writer_guard.go:59`, REQ-25.19). **P6 debe conservarla** (o moverla junto a la guarda) aunque retire el modo documento con perfil. Lo demás es vecindad en ficheros que ambos tocan:

| Fichero | Qué toca este cambio | Qué tocará P6 | Conflicto esperado |
|---|---|---|---|
| `internal/workspace/declarations.go` | Dos campos (`SDD`, `SDDErr`) en `Declarations`, una rama en `readDeclarationBlocks` y la llamada a la validación cruzada `checkISOWorkflow` (D18) tras decodificar `sdd:` y `workflow:` | Retira `Baseline`, `BaselineErr`, `decodeBaseline` y sus tipos | Textual, de pocas líneas, en el struct y en `readDeclarationBlocks` |
| `internal/workspace/workflow.go` | **Nada**: la validación cruzada vive en `sdd.go` | — | Ninguno |
| `internal/sddstatus/workspace_declarations.go` | Un caso en `projectWorkflow` para la contradicción (D18) | Retira `projectBaseline`, `declarationCauses` y la línea de `integrate prepare` | Textual, de una línea, en `projectWorkflow` |
| `internal/components/agentguidance/routing.go` | Condición ISO al principio del paso 0 del bloque ODD, y la vía ISO en el preámbulo y en la viñeta **Optional SDD** | Localizador de G6.6 en el bloque ODD | Textual, en la misma cadena del paso 0 si P6 la reescribe; además, P6 diseña la integración de ODD en la spec viva y debe respetar que su texto no regule los proyectos ISO, que ya no pasan por ODD |
| `internal/components/agentguidance/delegation.go` | Proyecto ISO como vía de entrada en **Optional SDD trigger** | Desconocido: solo si P6 cambia las vías de entrada en SDD | Textual, de una línea |
| `internal/workspace/loader.go` | `SDDErr` en `joinLoadErrors` | Retira `BaselineErr` de la misma lista | Una línea |
| `internal/hub/init.go` | `SDD` y `SDDErr` en `declares()` | Retira `Baseline` y `BaselineErr` de la misma expresión | Una línea |
| `internal/cli/sdd_archive_compose.go` | Una llamada de 5 a 8 líneas antes de leer la canónica; reutiliza `changeRootOfDelta` | Retira el modo documento con perfil y el modo `--in-place` | Mecánico; la lógica ISO vive en `sdd_archive_compose_iso.go`; `changeRootOfDelta` se conserva (la usa la guarda A) |
| `internal/assets/skills/sdd-archive/SKILL.md` | Bug sin delta y negativa ISO | Retira la rama de baseline del proyecto | Textual |
| `internal/assets/skills/_shared/sdd-status-contract.md` | Proyección `iso` y tokens | Retira la proyección `baseline` | Textual |

### 9.2 Regla del apply

En la PR de este cambio y en la de P6, **quien llegue segundo rebasa**. Para que el rebase sea mecánico:

- Las líneas nuevas de `declarations.go`, `loader.go`, `hub/init.go`, `workspace_declarations.go`, `routing.go` y `delegation.go` van en unidades de trabajo con su propio commit, separadas del resto de su pieza.
- En `workspace_declarations.go` solo cambia `projectWorkflow` (D18); la proyección ISO se engancha desde `Resolve`.
- `workflow.go` no se toca: la validación cruzada vive en `sdd.go`.
- La lógica ISO de `archive-compose` vive en un fichero propio.

---

## 10. Previsión de tamaño

Líneas autoradas (creadas más borradas), sin goldens ni copias versionadas. Las deltas de `specs/` ya están escritas y no cuentan. Los grupos suman exactamente las filas de §4, con los huecos H1 a H4 aplicados.

| Grupo (filas de §4) | Líneas |
|---|---|
| `workspace` (bloque `sdd:`, validación cruzada) y `hub/init.go` | 295 |
| `kickoff` (instantánea, `PrepareChangeRoot`, partición por política, `IntermediateMerge`) e inventario de escritores | 352 |
| `cli` de `kickoff seal` y `gate record` (aviso, huella `spec` de un bug) | 431 |
| `sddstatus` de gobierno (verify por política, compuerta `spec` de un bug) | 233 |
| `internal/isoevidence` (código 1.095, pruebas 1.030) | 2.125 |
| `sddstatus/iso.go`, enganches en `status.go` y `status_v2.go`, pruebas | 861 |
| `sdd archive-check` (con `alreadyComposed`) | 255 |
| Negativa en `archive-compose` | 178 |
| `sdd trace` y registro de verbos en `cmd/axiom` | 395 |
| `agentguidance` (routing y delegación) | 16 |
| Convención, secciones del orquestador, contratos, skills, pines, manual y ejemplo | 589 |
| **Total** | **~5.730** |

Margen razonable: **de 4.900 a 5.750 líneas**, según cuánto compriman las tablas de pruebas. La tolerancia de H6 suma unas 130 (regla y pruebas en `isoevidence`, `alreadyComposed` en `archive-check`, reanudación en la convención y la skill). Respecto de la versión anterior de este design (~4.890), suman las correcciones del validador: `archive-check` (+180), ruta `verify` y `iso.reason` (+100), H3 (+145), H4 (+35), `hub/init.go` (+27) y doctrina y pines adicionales (+65); restan unas 15 por reutilizar `changeRootOfDelta`. Es unas tres veces la estimación de la propuesta (1.200 a 1.800), que no contaba con la resolución de referencias contra el árbol, la traza con git real, las tres rutas de bug ni el punto de comprobación previo al archivado. No cambia la estrategia (una sola PR con `size:exception`), pero sí el tiempo de revisión: tasks debería ordenar los commits para que se revisen por pieza (§9.2 y el orden de §5 de la propuesta).

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: High
```

---

## 11. Huecos de producto

Ninguno bloquea el design. H1 a H5 están **aplicados** con la recomendación (decisiones y tablas de §4, §5.3 y §10); si el usuario elige otra opción, cambia solo la pieza indicada. H6 lo decidió el usuario el 2026-11-12 y está aplicado, con la enmienda de REQ-29.3.

| ID | Pregunta | Recomendación | Estado en el design | Pieza afectada |
|---|---|---|---|---|
| H1 | En `multirepo`, las pruebas viven en los repositorios de código, fuera del árbol del repositorio de specs. ¿Contra qué árbol se resuelven las referencias de la matriz? | La carpeta maestra (`Location.Master`) en toda topología: en monorepo coincide con la raíz del repositorio y en multirepo admite `<repo>/ruta:símbolo` sin salir de la maestra. Si se rechaza, ISO queda inutilizable en multirepo. | Aplicado (D11) | 6 |
| H2 | Con almacén `engram` o `none` no hay ficheros de delta, tareas ni informe que analizar. ¿Qué pasa con un cambio ISO o con un bug sin delta en esos almacenes? | Fallar cerrado: un cambio ISO con almacén sin ficheros bloquea el archivado con `iso_store_unsupported`; el perfil de bug sigue siendo solo del resolutor OpenSpec (`openspec` e `hybrid`). | Aplicado (D21, §5.3) | 2, 4 a 8 |
| H3 | Un bug sin delta no tiene artefacto de specs, así que la compuerta `spec` de un cambio `checkpointed` no se abre nunca y se salta en silencio. ¿Debe haber parada `spec` para un bug? | Sí: la compuerta `spec` juzga `proposal.md`, con la misma huella en `sddstatus/governance.go` y en `cli/sdd_gate_digest.go`. | Aplicado (D20, §4) | 8 |
| H4 | `kickoff.IntermediateMerge` exige el `role-apply` de **todos** los roles cuando hubo una salida 2; con la pieza 1, un archivado con un rol `deferred` pendiente se rechazaría como fusión intermedia. ¿Se alinea con la política? | Sí: considerar solo los roles `blocking`, igual que `verify`, sin tocar `internal/branch`. | Aplicado (D15, §4) | 1 |
| H5 | La spec exige derivar la traza y negar el archivado en Go, pero no nombra ninguna orden. ¿Qué superficies las exponen? | Dos verbos nuevos de solo lectura: `axiom sdd trace --cwd --change [--integration-branch] [--json]` y `axiom sdd archive-check --cwd --change [--json]`, citados en la convención ISO y en `sdd-archive`. | Aplicado (D13, D14) | 6, 7, 8 |
| H6 | Si un archivado ISO se interrumpe después de componer un ADDED y antes de mover la carpeta, `axiom sdd status` proyecta `iso_requirement_id` (el identificador ya está en la baseline) y ruta a `spec`, aunque la continuación correcta sea terminar el archivado. ¿Se enmienda REQ-29.3 para tolerarlo? | Sí, acotado: un ADDED no choca cuando la baseline del mismo dominio contiene un requisito con el mismo identificador, **el mismo nombre y el mismo bloque byte a byte** que el bloque ADDED (es esta delta ya compuesta). | Aplicado por decisión del usuario (2026-11-12): fila de REQ-29.3 enmendada con dos escenarios nuevos; D17 y `alreadyComposed` de `archive-check` | 4 |

---

## 12. Migración y despliegue

No hay migración de datos. Todo es aditivo y opt-in:

- Un proyecto sin `sdd:` y un cambio sin instantánea ISO producen el mismo status que hoy, salvo las dos correcciones (pieza 1 solo cambia algo con roles no bloqueantes; pieza 3 solo cambia el caso que hoy falla).
- Los cambios activos y archivados no se tocan. El retrosellado no escribe `sdd:` (REQ-29.13).
- Activar ISO: añadir `sdd: {iso: true}` a `axiom.yaml`; afecta solo a los cambios sellados después.

### Reversión

| Vía | Cómo | Efecto |
|---|---|---|
| Configuración | Retirar `sdd:` de `axiom.yaml` | Los cambios nuevos dejan de ser ISO; los ya sellados con `sdd.iso` siguen exigiendo su evidencia hasta archivarse |
| Código | `git revert` del PR, o volver a la versión publicada anterior | Un binario anterior ignora `sdd:` en `axiom.yaml` (claves de primer nivel desconocidas, `loader.go`) y en `kickoff.yaml` (decodificación no estricta, comprobado en D4) |
| Doctrina | `axiom sync` con el binario revertido | Se reproyectan las skills y la convención anteriores; `sdd-iso-convention.md` deja de instalarse |
| Datos | — | Un bug sin delta en curso no es completo para el binario anterior: hay que añadirle una delta o terminarlo antes de revertir |

Señales para revertir: un proyecto no ISO bloqueado por un motivo `iso_*`; un `verify` todavía bloqueado por un rol `deferred`; un archivado ISO que pase con la matriz inválida; una referencia existente rechazada; una diferencia en el JSON del status de un proyecto sin `sdd:`.

---

## 13. Riesgos

| ID | Riesgo | Prob. | Mitigación |
|---|---|---|---|
| R1 | Tamaño muy por encima de la previsión de la propuesta (§10) | Alta | Una sola PR con `size:exception`, como ya está decidido; commits por pieza y orden de revisión en tasks |
| R2 | Conflictos de rebase con P6 en `declarations.go`, `loader.go`, `hub/init.go`, `workspace_declarations.go`, `routing.go`, `delegation.go` y `sdd_archive_compose.go`, y riesgo de que P6 retire `changeRootOfDelta` | Media | §9: cambios mínimos en esos ficheros, en commits propios |
| R3 | Una prueba existente que enumere los ficheros de `_shared/` o fije el JSON del status cambie con `iso` | Media | `iso` es `omitempty` y solo aparece con `sdd:`; si una prueba enumera `_shared/`, se actualiza en el mismo commit que añade la convención |
| R4 | El movimiento de la carpeta lo sigue haciendo la skill con `git mv`: la negativa en Go de `archive-check` solo actúa si la skill lo invoca | Baja | Paso obligatorio en *Archive Readiness* fijado con un pin de texto; `dependencies.archive` bloqueada y `archive-compose` como segunda y tercera barrera |
| R5 | Falsos rechazos de referencias en lenguajes no Go (resolución textual) | Media | Supuesto aceptado por la spec; forma general y criterio de identificador completo documentados con ejemplos |
| R6 | La traza elige mal el commit de código con rebase si `verify-report.md` se añadió antes que las tareas | Baja | Se busca el primer commit con ambos ficheros; contraste con `pr_merged`; prueba con los tres modos de fusión |
| R7 | El cambio de significado de `lastRoleClosed` en el JSON de `gate record` con roles no bloqueantes | Baja | Con roles todos `blocking` no cambia; el campo nuevo `pendingNonBlockingRoles` hace explícito el caso |
| R9 | Con la contradicción `iso: true` y `lane: odd`, `LoadConfig` falla en todo consumidor que lo use, no solo en el status y el sellado | Media | Es el mismo comportamiento que ya tiene un `workflow:` inválido (fallo cerrado deseado por REQ-29.1); el mensaje nombra las dos claves y la corrección, y `Locate` sigue cargando la configuración base |
| R10 | Archivado ISO interrumpido entre componer y mover: un byte distinto entre la canónica compuesta y la delta (por ejemplo, si alguien edita a mano la canónica o la delta entre medias) hace que la tolerancia no aplique y el status vuelva a pedir `spec` | Baja | Es el fallo cerrado que pide la spec; la composición nativa reproduce el bloque tal cual, con la única normalización del salto de línea final, así que sin ediciones manuales la tolerancia siempre aplica; prueba con el compositor real |
| R11 | Con `ParseErr`, el archivado cita dos motivos para la misma causa (`declarationCauses` e `iso_declaration_invalid`) hasta que P6 retire la primera | Baja | Aceptado: ambos nombran el mismo error y la misma continuación |
| R8 | Coste del índice `TestX` en repositorios grandes | Baja | Un solo recorrido por validación, con prefiltro textual y directorios omitidos; solo se construye si alguna fila usa el atajo |

---

## 14. Preguntas abiertas

- [ ] H1 a H5 (§11): confirmación del usuario en la compuerta `design` (ya aplicados con la recomendación).
- [x] H6 (§11): decidido por el usuario el 2026-11-12; REQ-29.3 enmendado.
- [ ] Ninguna pregunta técnica bloquea tasks.
