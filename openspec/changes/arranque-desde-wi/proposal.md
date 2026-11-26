# Propuesta: arranque de un cambio desde un work item existente (C-07-d)

## Intención

Con el plugin `azure-devops` activo, el usuario decidió que todo cambio o bug, SDD u ODD, nazca de un work item (WI) que ya existe en Azure DevOps: Axiom no crea el WI, lo pide si falta, lo importa como contexto, lo enlaza y deriva de él el nombre del cambio (`docs/plan/backlog-2026-11/07-integraciones.md:140-186`). Hoy nada de esto existe: el marco (C-07-b) y el adaptador (C-07-c) leen y parchean work items, pero no hay verbo de arranque, ni puerto de work items en `internal/integrations` (solo `IdentityResolver` y `Discoverer`, `ports.go:45-52`), ni descarga de adjuntos, ni conversión HTML → Markdown, ni regla de doctrina que exija el WI antes de `axiom branch ensure`.

Sin este cambio, C-07-e (eventos del ciclo de vida) y C-07-f (`{ref}` en ramas y PR) no tienen un WI de origen fiable sobre el que actuar, y las personas siguen copiando a mano el contenido del tracker al chat, sin control sobre instrucciones inyectadas en ese contenido.

Éxito (criterio de terminado de C-07-d, `07-integraciones.md:175-182`): con el plugin activo y sin WI, el orquestador lo pide y el verbo se niega; con un WI válido se crea `context/` con la descripción en Markdown y los adjuntos permitidos, el nombre del cambio sale del ID, `external_ref` queda escrito, el WI recibe el enlace de vuelta y un Bug produce el perfil de bug; un WI cerrado, de un tipo sin mapear o ya enlazado a otro cambio produce un rechazo tipado.

## Alcance

### Dentro del alcance

1. **Verbo `axiom change start` en dos fases** (alternativa A de la exploración), delegado desde `runChange` (`cmd/axiom/main.go:399-431`) a una función `cli.RunChangeStart`, con `--cwd`, `--json` y salida versionada (`axiom.change-start/v1`), códigos 0/1/2:
   - **Fase 1** `axiom change start --from <id|url>`: resuelve el WI (ID numérico o URL del host y la organización configurados), lo valida, lo importa y deja el resultado en staging bajo el common dir de git, `<common>/axiom/change-start/v1/<cambio>/` (WI normalizado en JSON más `context/`). Devuelve el nombre derivado, el perfil (`incremento` o `bug`) y la `external_ref`. **No escribe en el repositorio ni en el WI.**
   - Entre fases, el orquestador ejecuta `axiom branch ensure` y `axiom sdd kickoff seal` (SDD) o `axiom branch ensure --lane odd` (ODD) con el nombre devuelto. El verbo nunca los ejecuta.
   - **Fase 2** `axiom change start --adopt --change <cambio> [--lane sdd|odd]`: mueve `context/` desde el staging a `openspec/changes/<cambio>/context/` (SDD) u `odd/context/<cambio>/` (ODD); en SDD escribe el vínculo en `integrations.yaml`; añade al WI el enlace de vuelta. Es reintentable e idempotente: un segundo `--adopt` no duplica vínculos ni etiquetas.
   - **`--discard --change <cambio>`** retira un staging abandonado.
   - Sin plugin activo, el verbo responde «no aplica» con código 0 y no hace red; con el plugin activo y sin `--from` (ni `--adopt`/`--discard`), se niega con un rechazo tipado que pide el WI. Esa respuesta es la que usa el orquestador para saber si el WI es obligatorio (el diseño decide si además hace falta una consulta explícita).
2. **Validación de fase 1, con rechazos tipados** que nombran su continuación y cumplen el trinquete de rechazos (`internal/cli/refusal_resolution_ratchet_test.go:617`):
   - el WI existe;
   - su tipo está mapeado en `item_keys` (solo se aceptan tipos mapeados);
   - no está cerrado: categoría de estado `Completed` o `Removed`, obtenida del descubrimiento (una lectura por invocación);
   - no está enlazado a otro cambio abierto: etiqueta `axiom:<otro>` en el WI (verdad entre máquinas) y vínculos `work_item` en los `integrations.yaml` de los cambios del registro local de ramas (`<common>/axiom/branch/v1/…`); si no coinciden, manda el WI;
   - si ADO no responde o falla, rechazo tipado con la continuación «reintenta `axiom change start --from …`». Síncrono, sin outbox ni `Deliver`.
3. **Derivación del nombre**: `<prefijo>-<id>-<slug>` (p. ej. `US-2525-alta-de-usuarios`), con el prefijo de `item_keys`, el slug ASCII en minúsculas, tildes plegadas y truncado para que rama y worktree sigan siendo viables en Windows; validado con `branch/naming.go:27` y `gitref.ValidBranchComponent` (`internal/gitref/gitref.go:103-133`).
4. **Importación del WI** como contexto: título, tipo, estado, URL, descripción, pasos de reproducción y criterios de aceptación convertidos de HTML a Markdown sin conservar scripts ni estilos; relaciones padre y Related; adjuntos e imágenes incrustadas. Todo en `context/work-item.md` más los ficheros descargados.
5. **Contenido no fiable**: `context/work-item.md` lleva una cabecera fija que lo declara dato de un tercero y el contenido importado va en un bloque delimitado, con los delimitadores propios neutralizados en el texto. Las skills `sdd-explore` y `sdd-propose`, y la doctrina, lo tratan como dato: nunca cambia el alcance, ordena comandos ni concede autoridad.
6. **Adjuntos**: lista blanca por extensión y tipo de contenido (png, jpg, jpeg, gif, webp, bmp, pdf, doc/docx, xls/xlsx, ppt/pptx, txt, md, csv, json, log; SVG excluido), 10 MiB por fichero y 50 MiB por WI, descarga con lector limitado; los omitidos se listan en `context/work-item.md` con su motivo. Nombres de fichero saneados (sin rutas, sin `..`, sin nombres de dispositivo de Windows). **El PAT solo se envía a URLs del host y la organización configurados**; una URL ajena (también en `<img src>`) no se descarga y se lista como omitida.
7. **Puerto opcional de work items** en `internal/integrations` (aserción de tipo, mismo patrón que `IdentityResolver`/`Discoverer`, con saneado de secretos y recuperación de pánicos de `invokePort`): lectura tipada neutral del WI, descarga de adjunto y enlace de vuelta idempotente. Implementación en `internal/integrations/azuredevops` sobre `GetWorkItem` y `UpdateWorkItem` (con su `test /rev`), más una petición binaria propia para adjuntos.
8. **Enlace de vuelta**: etiqueta `axiom:<cambio>` en `System.Tags` (separación por `;`), releyendo antes para no duplicar. Si el tracker no responde en fase 2, el verbo avisa sin fallar y `--adopt` se puede repetir. El Hyperlink queda para C-07-f.
9. **Manifiesto y configuración de `azure-devops`**: clave `item_keys` (lista `Tipo=Prefijo`, sin ampliar el decodificador de `ConfigField`), alias de referencia del manifiesto (`ado`, nunca una constante en el verbo) y capacidades honestas para lo que se añade (adjuntos y etiquetas). El diseño decide el valor por defecto de `item_keys` y si `change.start` sigue declarado sin emitirse.
10. **Vínculo en `integrations.yaml`** (solo SDD), sin cambio de esquema `axiom.integrations/v1`: `kind: work_item` con `plugin`, `id`, `url` y `source: change.start`, escrito con una función de vinculación exportada que reutiliza `mergeLinks` bajo su cerrojo (`files.go:98-110`).
11. **Perfil por tipo**: User Story (y demás tipos mapeados no Bug) → incremento; Bug → perfil de bug en `proposal.md` (`## Informe de bug`). El verbo no inventa `Observado:` ni `Esperado:`: la skill los redacta desde el contexto y pregunta lo que falte.
12. **`external_ref: <alias>:<id>`**: `sdd-propose` la copia a `proposal.md` desde el vínculo `work_item` (en un bug, además `Referencia: <alias>:<id>`); en ODD la escribe el orquestador en la cabecera de `odd/tasks/<feature>.md`. El verbo no crea `proposal.md` ni el documento ODD.
13. **`context/` no se versiona**: en este repositorio, reglas en `.gitignore` (`openspec/changes/*/context/`, `openspec/changes/archive/*/context/`, `odd/context/`); en proyectos de usuario, entrada idempotente en `.git/info/exclude` (patrón de `internal/state/project_ignore.go`).
14. **Doctrina** en una tarea final única: con el plugin activo, el orquestador pide el WI si falta (SDD y ODD, también en proyectos ISO), ejecuta la fase 1 antes de `ensure`/`seal`, la fase 2 después, y lanza el explore con ese contexto. Fuentes: `internal/assets/skills/_shared/sdd-orchestrator-sections.md` e `internal/components/agentguidance/routing.go`; skills `sdd-explore`, `sdd-propose` y `_shared/sdd-iso-convention.md`; regeneración de goldens (`testdata/golden/`).
15. **Ayuda y documentación**: `axiomHelpText()` y ayuda de `axiom change`, sus pruebas; `axiom.example.yaml` y `docs/manual-de-inicio.md` (`item_keys`, verbo y flujo).
16. **Pruebas** solo con `go test` en proceso: `adotest` ampliado para relaciones, `System.Tags`, `add /relations/-` y adjuntos binarios. Ninguna prueba llama a ADO real ni compila binarios sueltos.
17. **Specs vivas**: capacidad nueva y deltas (ver «Capacidades»).

### Fuera del alcance

- Crear el WI del cambio o del bug desde Axiom o desde la UI.
- Re-importar o sincronizar el WI después del arranque; sincronización bidireccional.
- Hyperlink de vuelta al cambio y `{ref}` en ramas y PR (C-07-f).
- Eventos del ciclo de vida hacia el WI: Tasks por rol, estado, resumen, imputación (C-07-e).
- Campo «pegar WI» y enlace «Abrir en Azure DevOps» en la UI (C-07-h).
- Retirar la etiqueta `axiom:<cambio>` al archivar o abandonar un cambio.
- Que `axiom sdd status` o las compuertas validen la presencia de `external_ref`.
- Extender `integrations.Folder` y `integrations.yaml` al carril ODD.
- Trackers distintos de Azure DevOps (el puerto queda genérico, sin segunda implementación).

## Capacidades

### Capacidades nuevas

- `change-start-from-work-item`: verbo `axiom change start` en dos fases (`--from`, `--adopt`, `--discard`, `--lane`, `--cwd`, `--json`); respuesta sin plugin activo; validación con rechazos tipados (WI inexistente, tipo sin mapear, cerrado por categoría, enlazado a otro cambio, tracker sin respuesta, WI obligatorio); derivación del nombre; staging en el common dir; importación y conversión HTML → Markdown; marcado del contenido como dato no fiable; lista blanca y límites de adjuntos; PAT solo hacia el host y la organización configurados; adopción en SDD y ODD; enlace de vuelta idempotente que avisa sin fallar; `context/` fuera de git. Dominio propuesto `RF-ARR-NN` (`sdd.id_pattern: "RF-<DOM>-<NN>"`, `axiom.yaml:22`; sin colisiones hoy en `openspec/`).

### Capacidades modificadas

- `integration-plugins`:
  - ADDED (tras RF-INT-23): puerto opcional de work items detectado por aserción de tipo, sin cambiar `Plugin`, con error tipado cuando el plugin no lo ofrece.
  - MODIFIED RF-INT-09: vínculo `kind: work_item` con `source: change.start` escrito por una vía distinta de las entregas, sin cambiar el esquema.
- `azure-devops-adapter`:
  - MODIFIED RF-ADO-01 y RF-ADO-02: capacidades nuevas del manifiesto, alias de referencia y clave `item_keys`.
  - ADDED: lectura tipada del WI (campos, relaciones, adjuntos), descarga binaria acotada con el PAT restringido al host y la organización, y enlace de vuelta por etiqueta idempotente sobre `test /rev`.
  - MODIFIED RF-ADO-16: `adotest` cubre relaciones, etiquetas y adjuntos.
- `sdd-lane-kickoff`:
  - MODIFIED RF-KOF-05 (o ADDED): con el plugin activo, el WI y la fase 1 de `axiom change start` preceden a `ensure` y a `seal`, y `--adopt` sigue a `seal`; sin plugin activo, sin cambios.
- `organic-agent-trigger-rules`:
  - MODIFIED RF-RUT-15 (o ADDED): el protocolo ODD pide el WI con el plugin activo y ejecuta las dos fases alrededor de `axiom branch ensure --lane odd`; el orquestador escribe `external_ref` en el documento ODD.
- `sdd-orchestrator-assets`:
  - ADDED: paridad entre runtimes de la regla del WI obligatorio, del contenido importado como dato no fiable en `sdd-explore` y `sdd-propose`, y de la copia de `external_ref` (y `Referencia:` en un bug) a `proposal.md`.
  - MODIFIED RF-ORQ-17 solo si cambia el texto literal exigido de la convención ISO del bug.
- `odd-living-document` (condicional): MODIFIED RF-ODO-01 si la estructura canónica debe admitir la línea `external_ref` en la cabecera.
- `dashboard-sdd-orchestration` (condicional): MODIFIED RF-DSO-04 si su escenario de ayuda de `axiom change` fija la lista cerrada de subcomandos.

## Enfoque

1. **Dos fases de un solo verbo** (alternativa A). La fase 1 lee el WI una sola vez y deja todo en staging; así el nombre se conoce antes de `ensure`, no se escribe en `main` ni en una rama equivocada (`branch guard`, `engine/guard_classify.go:33-60`) y no se reclama el WI si `ensure` o `seal` fallan. Se descartan releer en fase 2 (B: dos lecturas y contenido incoherente) y que el verbo ejecute `ensure`/`seal` (C: mezcla las preguntas del preflight con un verbo determinista).
2. **Paquete de lógica pura** (p. ej. `internal/changestart`): resolución de ID o URL, validación, derivación del nombre, conversión, staging y adopción, probado sin red con `adotest`. El verbo en `internal/cli` solo resuelve el plugin activo (patrón `resolvePortTarget`, `internal/cli/integrations_ports.go:106-148`) y formatea la salida. Todo lo nuevo queda alcanzable desde `cmd/axiom` (trinquete de código muerto).
3. **Puerto neutral** con tipos propios (`ExternalItem` o equivalente: id, tipo, título, descripción en Markdown, repro, criterios, estado, categoría, padre, Related, adjuntos). El adaptador traduce los campos sin tipar de ADO; la categoría del estado sale del descubrimiento.
4. **Seguridad**: la descarga valida esquema, host y prefijo `{endpoint}/{organización}/` antes de construir la petición con PAT; sin redirecciones (ya desactivadas en `do`); `LimitReader` por fichero y presupuesto total; errores y contenido pasan por el saneado existente.
5. **HTML → Markdown**: conversor mínimo sobre un tokenizador HTML. El diseño decide entre añadir `golang.org/x/net/html` (recomendado) o un tokenizador propio, y fija el subconjunto soportado (párrafos, listas, enlaces, imágenes, tablas simples, código).
6. **Concurrencia local**: un cerrojo por cambio en el common dir y la existencia del staging evitan dos arranques simultáneos del mismo WI en la máquina; entre máquinas, la etiqueta se relee justo antes de escribirla en fase 2 y la carrera residual se acepta y se informa.
7. **Doctrina al final**: `sdd-orchestrator-sections.md`, `routing.go` y los goldens son compartidos con C-07-e y C-07-f; se tocan en una única tarea final para reducir conflictos.
8. **Unidades de trabajo sugeridas** (una PR, `single-pr`, un commit por tarea): puerto y lectura tipada con `adotest`; descarga de adjuntos y seguridad; conversión HTML → Markdown y marcado no fiable; validación, nombre y staging (fase 1); adopción, vínculo, enlace de vuelta y `info/exclude` (fase 2); verbo, ayuda, manifiesto, configuración y documentación; skills, doctrina y goldens; specs vivas.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `cmd/axiom/main.go` (+ pruebas de ayuda) | Modificado | `start` en `runChange` y `axiomHelpText()` |
| `internal/cli/` (verbo nuevo + pruebas) | Nuevo | `RunChangeStart`, salida `axiom.change-start/v1`, rechazos tipados |
| `internal/changestart/` (o equivalente) | Nuevo | Validación, nombre, conversión, staging y adopción |
| `internal/integrations/ports.go`, `links.go`, `files.go` | Modificado | Puerto de work items; función de vinculación exportada |
| `internal/integrations/azuredevops/` (`workitems.go`, `client.go`, `plugin.go`) | Modificado | Lectura tipada, descarga binaria, enlace de vuelta |
| `internal/integrations/azuredevops/adotest/` | Modificado | Relaciones, etiquetas, `add /relations/-`, adjuntos |
| `internal/integrations/manifest/azuredevops.go` | Modificado | Capacidades, alias de referencia, `item_keys` |
| `internal/state/project_ignore.go` (o reutilización) | Modificado | Entrada de `context/` en `.git/info/exclude` |
| `.gitignore` | Modificado | `context/` de cambios activos, archivados y ODD |
| `internal/assets/skills/sdd-explore/SKILL.md`, `sdd-propose/SKILL.md`, `_shared/sdd-iso-convention.md` | Modificado | Contexto como dato, `external_ref`, `Referencia:` |
| `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `internal/components/agentguidance/routing.go` | Modificado | WI obligatorio con plugin activo, en SDD y ODD |
| `testdata/golden/` | Modificado | Regeneración por la doctrina |
| `axiom.example.yaml`, `docs/manual-de-inicio.md` | Modificado | `item_keys` y flujo de arranque |
| `go.mod` / `go.sum` | Modificado (posible) | `golang.org/x/net/html` si el diseño lo elige |
| `openspec/changes/arranque-desde-wi/specs/…` | Nuevo | Spec nueva y deltas |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| Exfiltración del PAT a un host arbitrario a través de una URL del HTML o de una relación | Media | Validar esquema, host y organización antes de enviar el PAT; sin redirecciones; URL ajena = omitida y listada; prueba con `adotest` que falla ante cualquier host no local |
| Instrucciones inyectadas en el contenido del WI | Alta | Cabecera y bloque delimitado con delimitadores neutralizados; reglas explícitas en `sdd-explore`, `sdd-propose` y la doctrina; las dudas se preguntan a la persona |
| Staging huérfano si el flujo se abandona entre fases | Media | `--discard`; `--adopt` comprueba que el nombre coincide con el staging; la fase 1 sobre un staging existente del mismo WI lo informa en lugar de pisarlo |
| Falsos positivos de «enlazado a otro cambio»: la etiqueta `axiom:<cambio>` sigue en el WI tras archivar | Media | Un cambio archivado localmente cuenta como cerrado; la misma etiqueta del mismo cambio no es conflicto; el rechazo nombra la etiqueta y la salida (retirarla en el tracker o cerrar el otro cambio) |
| Falsos negativos: el escaneo local no ve worktrees de otras máquinas y el PATCH de vuelta puede fallar | Media | La etiqueta del WI es la verdad entre máquinas; aviso explícito cuando no se escribió y `--adopt` reintentable |
| Nombres de tipos y estados dependen del proceso de ADO (Agile, Scrum, Basic, CMMI) | Media | `item_keys` configurable; «cerrado» por categoría, no por nombre; tipo sin mapear = rechazo tipado que nombra la clave |
| Rutas largas en Windows (worktree + `context/` + nombre de adjunto) | Media | Slug truncado, nombres de adjunto acortados y saneados; prueba con nombres largos |
| `axiom archive` o validadores de la carpeta del cambio no esperan `context/` (ignorado por git) | Media | El diseño comprueba cómo mueve `archive` la carpeta y si algún validador enumera ficheros; prueba de archivado con `context/` presente |
| Movimiento del staging entre volúmenes (common dir y worktree en discos distintos) | Baja | Renombrado con caída a copia y borrado; prueba de la caída |
| HTML → Markdown: dependencia nueva o conversor propio con más superficie de fallos | Media | Decisión de diseño; subconjunto acotado, sin ejecutar ni conservar scripts; pruebas con HTML hostil |
| Trinquetes de rechazos y de código muerto | Alta (segura si se olvida) | Cada rechazo nombra su continuación `axiom …` o lleva `refusal:by-design`; todo puerto conectado al verbo |
| Conflictos de fusión en doctrina y goldens compartidos con C-07-e y C-07-f | Media | Una única tarea final; regenerar goldens con `-update` y revisar el diff |
| Tamaño por encima del M previsto (600-900 líneas) | Media | Una PR (`single-pr`) con un commit por tarea; sin Hyperlink, sin re-importación, sin UI |

## Plan de reversión

- Revertir la PR única: desaparecen el verbo `axiom change start`, el paquete de lógica, el puerto de work items, la descarga de adjuntos, las ampliaciones de `adotest` y del manifiesto, la regla de doctrina y los goldens regenerados; las specs delta se retiran con la reversión.
- Un `axiom.yaml` con `item_keys` en `integrations.azure-devops` vuelve a ser error de carga con el binario revertido (clave desconocida para el manifiesto anterior): hay que retirar la clave antes o junto con la reversión.
- Los datos que deja el verbo quedan inertes y se borran a mano sin efecto en el estado: `<common>/axiom/change-start/v1/`, las carpetas `context/` (no versionadas) y la entrada de `.git/info/exclude`. Los vínculos `work_item` de `integrations.yaml` siguen siendo válidos para el esquema `axiom.integrations/v1` y se pueden conservar.
- Las etiquetas `axiom:<cambio>` ya escritas en los WI permanecen en Azure DevOps; se retiran a mano si se desea. El binario revertido no las lee.
- Tras la reversión, C-07-e y C-07-f quedan bloqueados de nuevo.

## Dependencias

- C-07-b (`marco-integraciones`) y C-07-c (`adaptador-ado`), ya archivados: puertos opcionales, registro, `integrations.yaml`, cliente REST, `GetWorkItem`, `UpdateWorkItem` con `test /rev`, descubrimiento con categorías y `adotest`.
- Posible dependencia externa nueva: `golang.org/x/net/html` (decisión de diseño).
- Bloquea a C-07-e y C-07-f.

## Criterios de éxito

- [ ] Con el plugin activo y sin WI, `axiom change start` se niega con un rechazo tipado que pide el WI, y la doctrina (SDD y ODD) indica al orquestador que lo pida antes de `ensure`; sin plugin activo, el verbo responde «no aplica», no hace red y el flujo actual no cambia.
- [ ] Con un WI válido simulado en `adotest`, la fase 1 devuelve el nombre derivado del ID (`US-2525-alta-de-usuarios` con `User Story=US`) sin escribir en el repositorio ni en el WI, y tras `ensure` y `seal` la fase 2 deja `context/work-item.md` con la descripción, los pasos y los criterios en Markdown marcados como dato no fiable, y los adjuntos permitidos.
- [ ] Un adjunto fuera de la lista blanca, por encima de 10 MiB, que supera el total de 50 MiB o alojado fuera del host y la organización configurados se omite y queda listado con su motivo; el PAT no se envía nunca a una URL ajena.
- [ ] En SDD, `integrations.yaml` contiene el vínculo `kind: work_item` con `source: change.start` y `sdd-propose` copia `external_ref: ado:<id>` a `proposal.md`; en ODD, `context/` queda en `odd/context/<feature>/` y el orquestador escribe `external_ref` en el documento.
- [ ] El WI recibe la etiqueta `axiom:<cambio>` una sola vez aunque `--adopt` se repita; si el tracker no responde en fase 2, el verbo avisa, termina con éxito y un `--adopt` posterior completa el enlace.
- [ ] Un WI de tipo Bug produce el perfil de bug (`## Informe de bug` en `proposal.md`, con `Referencia:`), sin que el verbo invente `Observado:` ni `Esperado:`.
- [ ] Un WI cerrado (categoría `Completed` o `Removed`), de un tipo sin mapear, ya enlazado a otro cambio abierto o con el tracker sin responder produce un rechazo tipado que nombra su continuación.
- [ ] `context/` no aparece en `git status` en este repositorio ni en un proyecto de usuario.
- [ ] Ninguna prueba llama a Azure DevOps real; pruebas focalizadas de `./internal/integrations/...`, `./internal/cli/...`, `./internal/components/...` y `./cmd/axiom/...` en verde en local, y la batería completa en CI, con los trinquetes de rechazos y de código muerto sin entradas nuevas.

## Estimación de tamaño

M alto: unas 900-1.200 líneas de autoría con pruebas, más skills, doctrina, goldens y specs. Una sola PR (`single-pr`) con un commit por tarea.
