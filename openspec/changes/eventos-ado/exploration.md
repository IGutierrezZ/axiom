## Exploración: eventos-ado (C-07-e) — Tasks por rol, estimación, resumen de la spec e imputación en Azure DevOps

### Estado actual

**Marco de integraciones (`internal/integrations`).**
- Puerto `Plugin{Manifest, Deliver}` en `plugin.go:55-60`; `Delivery` (evento, config, instantánea de vínculos, secretos) en `plugin.go:19-24`. Puertos opcionales por aserción de tipo: identidad y descubrimiento (`ports.go:44-52`) y work items `WorkItemSource` (`workitems.go:118-125`: `ParseItemRef`, `ReadItem`, `StateCategory`, `DownloadAttachment`, `LinkBack`). **No existe ningún puerto de escritura de Tasks, estimación, transición, asignación ni descripción.**
- Catálogo cerrado de eventos (`manifest/events.go:9-15`): `change.start`, `gate.spec.approved`, `gate.design.approved`, `role.start`, `role.apply.approved`, `archive`. Catálogo cerrado de capacidades (`manifest/capabilities.go:7-17`): `items.create`, `estimation`, `time`, `items.transition`, `assignment`, `relations`, `vcs_links`… ya declaradas como nombres, ninguna usada.
- Manifiesto ADO (`manifest/azuredevops.go:14-39`): capacidades `items.read, discovery, tags, attachments`; **solo escucha `change.start`**; `RefAlias: ado`; config `organization, project, team, item_keys` (tipos `string|bool|int|string_list`, `manifest.go:51-54`; único formato `type_prefix_pairs`, `format.go:13`); secreto `pat`. `Deliver` devuelve `Permanent(ErrLifecycleUnsupported)` sin efectos (`azuredevops/plugin.go:36-41`).
- Despachador (`dispatcher.go:148-250`): solo entrega a plugins cuyo manifiesto escucha el evento (`dispatcher.go:106-121`); cola durable antes de intentar; intento inmediato de 10 s; `MaxAttempts=10`. Clave de idempotencia por evento (`event.go:76-93`): `gate.*/<cambio>/<huella>`, `role.apply.approved/<cambio>/<rol>/<huella>`, `role.start/<cambio>/<rol>`, `archive/<cambio>/<carpeta>`; huella = `d`+16 hex del digest del artefacto u `o<n>`. `Event` (`event.go:24-34`) **no lleva carga útil**: solo tipo, cambio, rol, compuerta, actor, fecha, digest.
- Vínculos `integrations.yaml` (`links.go:19-38`, esquema `axiom.integrations/v1`, lectura estricta `KnownFields`): `links[]` (`Link{plugin, kind, id, url, role, source, recorded_at}`, `plugin.go:33-41`) y `time_entries[]` (`TimeEntry{plugin, role, item_id, minutes, date, source, recorded_at}`, `plugin.go:44-52`, «Nadie la produce aún (C-07-e)»). Identidad de vínculo `(plugin,kind,id,role)`, de imputación `(plugin,source,item_id)` (`links.go:151-152,178-202`). **`TimeEntry` no tiene campo de actor ni de estimación**; hay `Role` en `Link`, así que el vínculo de Task por rol cabe hoy (`kind: task`, `role: back`). `LinkWorkItem`/`WorkItemLinks` (`links.go:271-293`) ya existen para `kind: work_item` (C-07-d).
- Fusión solo si `PolicyFor(...).PersistOutcome` (`event.go:110-124`): `archive` nunca fusiona (carpeta archivada inmutable).

**Verbos y disparo de eventos (`internal/cli`).**
- `axiom integrations` registrado en `cmd/axiom/main.go:294` → `RunIntegrations` (`cli/integrations.go:135-159`): `sync, emit, secret, connect, discover`. `axiom change start` en `main.go:284` (`cli/change_start.go`).
- Emisión: `emitGateIntegrationEvent` tras `AppendGate` (`sdd_gate.go:183`, `integrations_events.go:165-176`) solo con decisión `approved`; `emitRoleStartEvent` desde `branch ensure --role` (`branch.go:323-325` → `branch_role.go:61-73`) **después** de que el motor deje la rama lista; `emitArchiveEvent` desde `sdd publish --archive` (`sdd_publish.go:121`). Siempre con aviso, nunca falla el verbo de origen (RF-INT-10).
- El rechazo de `--role` se valida **antes** del motor, con `branch.Refusal` tipado (`branch_role.go:26-57`, `branch/role_refusal.go:16-57`; códigos `plan_conflict`, `role_not_applicable`, `kickoff_not_sealed`, `role_not_in_roster`). `engine/ensure.go` no conoce el rol (no hay ninguna mención de `integrations` ni de rol en él): el punto natural para «negarse sin Task vinculada» es `validateEnsureRole`, no `ensure.go:79` (la ruta del plan queda desfasada).
- Actor (C-07-a): `defaultActor` (`cli/sdd_actor_identity.go:40-46`) devuelve `"Nombre <email>"` o `"cli"`; `gate record` lo graba en `GateRecord.Actor` (`sdd_gate.go:126-152`); `emit` lo reutiliza (`integrations.go:429-431`).
- **Continuous: no existe hoy ningún verbo determinista al final de spec/design/apply.** En `continuous` `gates.yaml` no tiene compuertas (`sddstatus/governance.go:411-432`) y la doctrina dice «ninguna parada». `emit` de eventos de compuerta exige una decisión `approved` ya registrada («no se fabrican aprobaciones», `integrations.go:462-481`). `gate record` no comprueba el estilo de ejecución, pero grabar compuertas en `continuous` rompe la semántica actual. Lo único determinista en `continuous` es `sdd publish` (solo con `docsPublication: auto`, `sddstatus/publication.go:102-146,180-183`) y `sdd-task-result` (`cli/sdd_task_result.go`, clasifica resultado, no despacha). Además, en monorepo con rol único `fullstack` nada obliga hoy a llamar `ensure --role fullstack`: si la salvaguarda vive solo en `ensure --role`, `continuous` la esquiva.

**Adaptador `azure-devops` (`internal/integrations/azuredevops`).**
- Cliente único con reintentos y `Retry-After` (`client.go:108-117,211-212,348-359`). `UpdateWorkItem(id, rev, ops)` antepone siempre `test /rev` y devuelve `*RevConflictError` sin reintentar (`workitems.go:138-193`); `GetWorkItem` con `$expand=relations` (`workitems.go:47-65`); `GetWorkItems` por lotes (`:76-118`). `LinkBack` añade `axiom:<cambio>` a `System.Tags` con 3 intentos (`linkback.go:25-67`) y rechaza si hay **otras** etiquetas `axiom:*` (`linkback.go:50-52,70-77`).
- **No existe creación de work items** (ningún `POST .../workitems/$Task`), ni constantes de campos de estimación (`workitem_port.go:18-24` solo tiene Type, Title, State, Tags, Description, ReproSteps, AcceptanceCriteria), ni escritura de relaciones, estados o asignación. Un `POST` de creación no es reintentable por diseño (`retryable` solo GET, `workitemsbatch` y PATCH con `test /rev`, `client.go:211`): un fallo tras aplicarse duplicaría la Task.
- Identidad ADO de la persona: `identitystore` (`identitystore/store.go:35-49`, fichero `~/.axiom/integration-identities.json`, índice `plugins[plugin][organización]`, `Get/Put`). `connect` la guarda (RF-ADO-10/11). Es el origen de «asignada a la persona» en `role.start`.
- Doble: `adotest.Server` con `Handle/Reply/Sequence/Requests/Count` (`adotest/server.go:137-180`), `adotest/workitems.go` y `states.go`; `integrationstest/fake.go` para el núcleo. Hermético: solo hosts locales (`server.go:114-135`).
- `item_keys` (`Tipo=Prefijo`) y `RefAlias` (`manifest/azuredevops.go:20,29-32`); `change start` (C-07-d): validación de WI, importación a `context/`, `LinkWorkItem` con `source: change.start`, etiqueta de vuelta.

**IDs funcionales (C-03-b).** Paquete hoja `internal/reqid`: `Pattern.Scan(text)` → válidos y mal formados (`reqid.go:108`), `ScanProvisional`/`ScanProvisionalRefs` (`provisional.go:107,123`), forma `prov:<DOM>:<n>` (`ProvisionalText`, `provisional.go:27`). Las deltas del cambio los cargan en los títulos `### Requirement:`; `livespec/ids.go:218-230` y `livespec/change.go:253-261` ya los extraen. **Ojo:** al aprobar la spec los `ADDED` llevan IDs provisionales; los definitivos solo se asignan al archivar y se guardan en `ids.yaml` (`livespec/change.go:23`, `cli/sdd_archive_check.go:126,267`).

**Resumen HTML de C-06-b.** `resumen-spec.html` es opcional (`governance.kickoff.explainerHtml: yes`), lo escribe el **agente**, lleva CSS y SVG en línea (`assets/skills/_shared/sdd-explainer-html.md:10-14`) y el binario solo mide su frescura (`internal/explainer/explainer.go:1-18`, `MaxBytes` 5 MiB). **No es reutilizable** para `System.Description`: no siempre existe, es no determinista y usa justo el HTML que ADO podría filtrar. La variante simple debe generarla el binario de forma determinista desde el md de la spec.

**Doctrina y pruebas.**
- Skills a tocar: `internal/assets/skills/sdd-design/SKILL.md` (pasos 3-6: añadir propuesta de estimación por rol), `sdd-apply/SKILL.md` (paso 7 «Return Summary»/«Closing the Work», líneas ~217-275), `sdd-archive/SKILL.md` (informe: `## Resumen de Cierre`, línea ~339). Orquestador: `_shared/sdd-orchestrator-sections.md` y `components/agentguidance/routing.go:138-141` (donde C-07-d añadió el texto de `change start`; C-07-e se coloca junto a él). Copias espejo: `internal/assets/skill_copies_parity_test.go`, `shared_skill_files_test.go`; goldens en `testdata/golden/sdd-*.golden` y `components/golden_test.go`, `components/sdd/orchestrator_composition_test.go`.
- Trinquetes: `scripts/deadcode-ratchet.sh` (objetivo `./cmd/axiom`; todo código nuevo debe ser alcanzable desde el despacho); `TestEveryProductionRefusalNamesResolutionOrDeclaresByDesign` (`cli/refusal_resolution_ratchet_test.go:617`; cada rechazo nuevo nombra un `axiom …` o lleva `// refusal:by-design <categoría>:`); canarios de secretos `cli/integrations_canary_test.go:20` y `integrations_ado_canary_test.go:131` (rutas `/api` nuevas); `idmigration/repository_test.go:167` (`TestRepositoryLegacyIDResidues`, escanea `openspec/changes/`: usar `prov:ADO:n` y nunca el prefijo heredado).
- **Specs vivas que esta spec contradice y habrá que MODIFICAR:** RF-ADO-14 «`Deliver` sin efectos del ciclo de vida» (`azure-devops-adapter/spec.md:431-445`, incluido su escenario «Verbos de despacho sin cambios») y la frase de RF-ADO-07 «ningún evento invoca esta operación todavía»; RF-INT-09 (`integration-plugins/spec.md:239-286`, TimeEntry y «ODD no escribe integrations.yaml»); RF-INT-12 (`emit`, `:348`); el comentario de `manifest/azuredevops.go:10-13`.
- Prefijos y último número vivo: `RF-ADO` hasta 19 (`azure-devops-adapter`), `RF-INT` hasta 24 (`integration-plugins`), `RF-ARR` hasta 19 (`change-start-from-work-item`). Otras capacidades probables: `change-branch-lifecycle` (rechazo de `ensure --role`), `sdd-orchestrator-assets` (doctrina), `sdd-archive-lifecycle` (resumen final), `change-actor-identity`. En la spec del cambio, usar `prov:ADO:1…`, `prov:INT:1…`, `prov:ARR:1…`, etc.

### Verificaciones técnicas (solo documentación pública de Microsoft)

| Tema | Resultado | Fuente |
|---|---|---|
| Crear Task | **Verificado:** `POST https://dev.azure.com/{org}/{project}/_apis/wit/workitems/${type}?api-version=7.1`, cuerpo `application/json-patch+json` con ops `add` sobre `/fields/...`; parámetros opcionales `validateOnly`, `bypassRules`, `suppressNotifications`, `$expand`; scope `vso.work_write`. El ejemplo oficial solo crea con `System.Title`. | https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/work-items/create?view=azure-devops-rest-7.1 |
| Enlace padre/hijo | **Verificado en parte:** la respuesta oficial de `update` muestra `rel: System.LinkTypes.Hierarchy-Reverse` con `url` del padre y `attributes.comment`, y el patrón `{"op":"add","path":"/relations/-","value":{"rel":…,"url":…}}`. Que se pueda crear la Task **ya con** esa relación en el POST es el uso conocido de la API, pero la página de `create` no lo ejemplifica. **No verificado en doc oficial.** | https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/work-items/update?view=azure-devops-rest-7.1 |
| Campos `OriginalEstimate`/`RemainingWork`/`CompletedWork` | **No verificado** en esta exploración (no se obtuvo la página de referencia de campos). Nombres tomados del plan; confirmar con `discovery` (campos por tipo Task) y con el proceso Agile/Scrum. | — |
| HTML de `System.Description` | **No verificado:** no hay documentación pública de lista de etiquetas permitidas ni del tratamiento de `style`, `script` o SVG. Solo consta que el campo es de tipo HTML y que por REST el formato por defecto de los campos largos sigue siendo HTML. | búsquedas en learn.microsoft.com (sin página con lista blanca) |
| `ArtifactLink` de rama `vstfs:///Git/Ref/<proyecto>%2F<repo>%2FGB<rama>` | **No verificado:** la documentación pública solo describe el tipo de enlace «Branch» (Git/Branch) y el endpoint `artifactlinktypes`; no da el formato. Recomendación del propio resultado: crear el enlace a mano en un WI de prueba y leer `relations`. (Probablemente, por el launcher, usa **IDs** de proyecto y repo, no nombres; sin confirmar.) Es de C-07-f/h; este cambio no enlaza ramas. | https://learn.microsoft.com/en-us/rest/api/azure/devops/wit/artifact-link-types/list |
| Etiquetas (`System.Tags`) | **Verificado:** ≤400 caracteres, sin separadores `,` ni `;`; **máx. 100 etiquetas por work item** (error `TF401243`); sensibles a mayúsculas; el nombre de etiqueta se comparte en el proyecto (reserva común de 150.000 con las de PR); crear etiquetas nuevas exige el permiso de proyecto «Create tag definition» (por defecto, Contributors). | https://learn.microsoft.com/en-us/azure/devops/boards/queries/add-tags-to-work-items?view=azure-devops |
| Hyperlink por REST | **Verificado:** `{"op":"add","path":"/relations/-","value":{"rel":"Hyperlink","url":"…"}}`. | página `update` 7.1 |
| Markdown en `Description` | No verificado (la página consultada devolvió 404). No se necesita: se usa HTML. | — |

### Áreas afectadas
- `internal/integrations/` — puerto opcional nuevo de «tareas por rol» y de escritura de WI (crear Task hija, leer estimación y `CompletedWork`, ajustar `RemainingWork`, transición, asignación, reemplazar descripción); extender `TimeEntry`/`Link` o el vínculo de Task con estimación y actor (esquema `integrations.yaml`, lectura estricta); posible `Event`/`Delivery` con carga útil.
- `internal/integrations/manifest/azuredevops.go` y `catalog` — capacidades `items.create, relations, estimation, time, items.transition, assignment` y eventos escuchados; nuevas claves de config (jornada, estados por evento) con formato nuevo.
- `internal/integrations/azuredevops/` — `CreateWorkItem` (POST no reintentable, con control de duplicados), campos de horas, PATCH de descripción/estado/asignación, validación de Task (existe, es Task, cuelga del WI); `Deliver` deja de ser «sin efectos».
- `internal/cli/integrations.go` (+ ficheros nuevos) — `integrations tasks create|bind`, verbo de imputación y de resumen; ayuda, esquema de salida, rechazos con resolución nombrada.
- `internal/cli/sdd_gate.go:183`, `integrations_events.go`, `branch_role.go:26-57`, `branch/role_refusal.go` — disparo y rechazo tipado «Task sin vincular».
- `internal/assets/skills/{sdd-design,sdd-apply,sdd-archive}/SKILL.md`, `_shared/sdd-orchestrator-sections.md`, `components/agentguidance/routing.go`, `testdata/golden/*`.
- `openspec/specs/{azure-devops-adapter,integration-plugins,change-branch-lifecycle,sdd-orchestrator-assets,sdd-archive-lifecycle}` — deltas.
- Posible integración con `sdd archive-compose` / informe de cierre para el resumen final.

### Enfoques

**A. Disparo en `continuous` (bifurcación real).**

| Opción | Pros | Contras | Esfuerzo |
|---|---|---|---|
| A1. Verbos dedicados deterministas (`integrations tasks create`, `integrations spec-summary`, `integrations time propose/log`) que la doctrina invoca al final de cada fase en ambos estilos; `gate record` en checkpointed solo informa/encadena | Un único camino para `checkpointed` y `continuous`; las horas y estimaciones viajan en la invocación; reintento y vínculo manual ya son verbos | Duplica el origen del hecho respecto del catálogo de eventos; dependencia de la doctrina (el agente podría omitirlo) → la barrera dura queda en `ensure --role` | M |
| A2. Extender `emit` para aceptar `gate.spec/design.approved` y `role.apply.approved` en `continuous` cuando la fase está completa (huella de `gateArtifactDigest`) | Un solo catálogo y un solo despachador; la idempotencia reutiliza huellas | `emit` hoy «no fabrica aprobaciones» (RF-INT-12); semántica de «aprobado» sin compuerta; exige modificar RF-INT-12 y estado; las horas necesitan carga útil | M-L |
| A3. Registrar compuertas aprobadas en `continuous` | Reutiliza todo el camino actual | Cambia el contrato de `gates.yaml` en `continuous` (sin compuertas por diseño, `governance.go:411-432`); riesgo en `status`/`nextRecommended`/archivo | L, arriesgado |

Recomendación: **A1 con barrera en `ensure --role` y, en `checkpointed`, `gate record` sin lógica nueva más allá de informar el siguiente paso**. Es la única que no toca el contrato de compuertas y resuelve la excepción síncrona (P8) con el mismo verbo de reintento. Marcar en la spec que `continuous` se apoya en doctrina + barrera mecánica del `ensure`.

**B. Dónde viven las horas y la estimación (`integrations.yaml`).**
- B1. Extender `TimeEntry` con `actor` y `hours` (o minutos) y añadir `kind: task` con campo `estimate_hours` en el vínculo. Pros: un solo fichero versionado, el resumen final lo lee sin red. Contras: cambio de esquema con lectura estricta (binarios antiguos rechazarían el fichero); hay que decidir si sube a `axiom.integrations/v2`.
- B2. Guardar solo el vínculo y leer siempre de ADO. Pros: sin cambio de esquema. Contras: el resumen final de `continuous` depende de red y no es reproducible tras archivar. 
- Recomendación: B1, con `source` distinto por imputación (la identidad `(plugin,source,item_id)` hoy impediría dos imputaciones de la misma Task con la misma clave).

**C. Transporte de los IDs funcionales (P12).** Reglas del tracker sin personalizar el proceso:

| Opción | Pros | Contras |
|---|---|---|
| C1. Etiquetas `System.Tags` una por ID | Filtrables; ya hay `LinkBack` y `splitTags` | Hasta decenas de etiquetas por WI frente al **límite de 100**; contaminan la reserva compartida del proyecto; los `prov:` cambian al archivar; `foreignAxiomTags` trata cualquier otra `axiom:*` como conflicto, así que **no usar el prefijo `axiom:`** |
| C2. Una sola sección «Requisitos funcionales» dentro del HTML de `Description` | Sin límites ni etiquetas; viaja con el resumen e idempotente | No son «aparte»: no filtrable ni consultable |
| C3. Un `Hyperlink` por ID/portal | Aparte de la descripción | Exige URL real; ruido de enlaces; no aplica |
| C4. Campo estándar (p. ej. `System.History`) | — | Es comentario: fuera de alcance del plan |

Recomendación: **C2 como base (siempre) + una única etiqueta agregada configurable opcional** (p. ej. `req-count:N` no; mejor un prefijo configurable distinto de `axiom:`), y dejar C1 por ID solo bajo configuración con tope. Pregunta abierta real: ver abajo.

**D. Resumen de la spec en `Description`.** D1: renderizador Go determinista md→HTML simple (`h2/h3/p/ul/li/code`, sin `style`/SVG/CSS), con IDs y requisitos; D2: reutilizar `resumen-spec.html`. D2 descartada (opcional, agente, SVG/CSS). D1 es idempotente por construcción y no depende de la verificación pendiente del filtrado. Recomendada.

**E. Estados/asignación por evento.** Config del plugin por evento con valor por defecto (`role.start → Active` + asignación con la identidad guardada; `role.apply.approved → Tech Review`; `archive → transición del WI`), validada contra `Discovery`/`StateCategory`. Si el estado configurado no existe en el proceso (p. ej. «Tech Review» no es estándar en Agile/Scrum), rechazo tipado con la lista de estados; nunca tocar estados por defecto sin config.

### Recomendación
Entrega única `eventos-ado` en carril SDD, orden de commits: (1) núcleo: puerto de escritura de Tasks, vínculo con estimación/actor, rechazo `task_not_bound` en `validateEnsureRole`; (2) adaptador ADO: POST Task hija con control de duplicados, horas con `test /rev`, descripción, transición/asignación, doble `adotest` ampliado; (3) verbos `integrations tasks create|bind`, resumen de la spec, propuesta/imputación de horas (jornada configurable); (4) doctrina (design/apply/archive, orquestador, routing) y goldens; (5) deltas de spec con `prov:ADO:n`, `prov:INT:n`, y modificación de RF-ADO-14, RF-ADO-07 y RF-INT-09/12. El cambio es L y puede partirse en commits de tarea; no hay motivo para dos PR (regla de una PR por cambio).

### Riesgos
- **Escritura de Task no idempotente:** un POST con respuesta perdida duplica la Task. Mitigar leyendo antes los hijos del WI (título/etiqueta de rol) y registrando el vínculo; `bind` valida existencia, tipo Task y padre.
- **`continuous` no tiene disparador mecánico:** depende de la doctrina; la barrera dura es `ensure --role`, pero un cambio `fullstack` en monorepo puede no llamarlo. Decidir si basta con la doctrina o si `apply` en continuous debe verificar Task vinculada.
- **IDs `prov:` obsoletos tras archivar:** el WI mostraría IDs provisionales; hay que decidir si el evento `archive` los actualiza (archive no persiste vínculos, pero puede llamar al plugin) o si se acepta la deriva.
- **Esquema de `integrations.yaml` estricto:** añadir campos rompe binarios antiguos; coordinar versión.
- **Contradicción con specs vivas** (RF-ADO-14, RF-ADO-07, RF-INT-09/12): hay que modificarlas, no solo añadir.
- **Sin verificar:** filtrado de HTML en `System.Description`, creación de Task con relación en el mismo POST, nombres exactos de campos de horas en el proceso del cliente, existencia de «Tech Review». Mitigación: pruebas herméticas con `adotest` (nunca API real) y rechazos tipados con `discovery`.
- **Horas:** reloj recortado a jornada (8 h, lun-vie) es sugerencia; `CompletedWork` puede tener horas manuales en ADO: leer siempre antes de sumar y usar `test /rev`; un conflicto de `rev` no se reaplica.
- **PAT/canario:** rutas `/api` nuevas deben pasar `integrations_ado_canary_test.go`; scope `vso.work_write` ya cubre crear/actualizar.
- **Trinquetes:** código nuevo alcanzable desde `cmd/axiom`; cada rechazo con resolución nombrada.
- **ODD:** sin Tasks; los verbos nuevos deben rechazar el carril odd con rechazo tipado (como `RoleNotApplicable`).

### Preguntas de producto abiertas (reales)
1. **Transporte de IDs funcionales (P12 dejó la elección al diseño):** ¿vale C2 (sección dentro de `Description`) como «por separado», o el usuario exige un elemento filtrable por separado (etiquetas)? Afecta a la spec.
2. **IDs provisionales:** ¿se reenvían los IDs definitivos al archivar (evento `archive` con lectura de `ids.yaml`)?
3. **`continuous` en monorepo `fullstack`:** ¿es aceptable que la barrera «Task vinculada» solo se aplique si el cambio llama `ensure --role`, o `apply` debe comprobarla siempre?

### Listo para propuesta
Sí. Orquestador: lanzar `sdd-propose` con la recomendación A1+B1+C2+D1, resolviendo antes las tres preguntas anteriores (o dejándolas como supuestos explícitos), y pedir en el diseño la confirmación hermética de los campos de horas y del filtrado HTML con el doble `adotest`.
