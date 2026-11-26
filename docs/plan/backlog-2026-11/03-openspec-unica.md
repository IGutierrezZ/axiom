# 03 · OpenSpec única: spec viva, integración desde ODD e identificadores funcionales

> Sección escrita por la sesión de análisis 3 (OpenSpec única). Rutas verificadas contra `main` en `5f6b05b4`. Engram: `analisis-backlog/openspec-unica` y sus subtemas `decision-iso-sin-odd`, `enmienda-carril`, `decision-ids-funcionales`, `decision-ids-declarados`, `decision-migrar-axiom`, `decision-migracion-iso` y `decision-conservar-ids-cliente`.

## Origen

El backlog original planteaba tres temas, en palabras del usuario:

1. **Migrar a OpenSpec, sin integrar otras specs vivas.** «Se empezaron a hacer cambios para poder adoptar specs vivas de otra índole que no fueran OpenSpec, pero creo que es liar la manta. Lo más fácil y mejor es ir todo a OpenSpec, así que hay que diseñar un onboarding de las specs que vengan para migrarlas a formato OpenSpec, cumpliendo ISO, etc.»
2. **ODD archiva en OpenSpec al terminar.** «Está bien que ODD tenga sus propios flujos y archivos y no vaya por OpenSpec, pero al terminar esos cambios tienen que integrarse en la spec viva, que es OpenSpec. Aquí ISO no aplica, porque ISO obliga a SDD.»
3. **Identificadores de funcionalidad (tipo RF-USU-01) y trazabilidad.** Incluirlos si no existen, y que los incrementos y los bugs se relacionen con ese ID para que todo quede trazado. Si hay integración con un tercero, enviarlo también allí.

### Estado en `main` (5f6b05b4)

**Temas 1 y 2: hechos.** Los resolvió P6 `spec-viva-openspec` (#259 y #260, archivado en `openspec/changes/archive/2026-11-12-spec-viva-openspec/`, 67/67 tareas, `archive-report.md:6`):

- `internal/specbaseline/` ya no existe; lo retiró el commit `1166efca`.
- `baseline:` se rechaza con `workspace.BaselineRetiredError` (`internal/workspace/declarations.go:33`).
- `axiom sdd integrate` devuelve `SDDIntegrateRetiredError` (`internal/cli/sdd_integrate_retired.go:32`).
- Hay verbos nuevos, enrutados en `cmd/axiom/main.go:1774-1778`:
  - `axiom archive integrate --odd <feature>`, en `internal/cli/archive_integrate.go:33`, paquete `internal/livespec`.
  - `axiom archive adopt scan|check`, en `internal/cli/archive_adopt.go:28`, paquetes `internal/adoption` e `internal/adoption/adoptcheck`.
- El contexto técnico se reubicó en `internal/techcontext`.
- El documento ODD lleva su delta en `## Spec viva` (`internal/livespec/odddoc.go:36`, guía en `internal/components/agentguidance/routing.go:77`).
- Un proyecto ISO no puede declarar `workflow.lane: odd` (`internal/workspace/sdd.go:22`, RF-ISO-01).

**Tema 3: pendiente.** De ahí sale todo lo que contiene esta sección.

## Decisiones cerradas con el usuario (literales)

Todas son del 2026-11-12 y las tomó el usuario. Se citan tal como quedaron y se usan en los cambios siguientes:

- **D-03-1:** «un proyecto con iso no va a usar odd, usara sdd». Ya está en código (`internal/workspace/sdd.go:20-22`).
- **D-03-2:** corregir `carril-sdd-versatil` antes de implementarlo: «Sí, antes». Ya se cumplió: D4 invertida, D5 = solo OpenSpec, fuera `identity: id`. Queda un residuo: RF-ISO-03 sigue atado a `REQ-<n>.<m>`. Se trata en C-03-a.
- **D-03-3:** IDs funcionales tipo `RF-USU-01` en lugar de `REQ-<incremento>.<n>`: «IDs funcionales».
- **D-03-4:** «además si un incremento toca un id funcional existente en el incremento tiene que quedar reflejado que se va a tocar ese. si va a crear uno nuevo lo mismo aunque aun no se sepa el numero hasta que se integre.»
- **D-03-5:** el mecanismo de D-03-4 (ID provisional y número asignado al integrar) entra «Solo en el cambio RF». P6 conservó su numeración `REQ-n.m`.
- **D-03-6:** el cambio RF migra también las specs propias de Axiom: «Sí, migrar».
- **D-03-7:** «cumpliendo ISO» en la migración de specs ajenas significa: «lo que quiero es que en la spec viva de open spec que se va a generar tenga los IDs RF- etc...». No hay que validar ninguna norma ISO publicada (como la 29148) ni usar las reglas REQ-29.x como criterio propio de la migración.
- **D-03-8:** si la spec de origen ya trae IDs funcionales (p. ej. `RF-120`): «Conservar los suyos».

Decisiones del análisis que se derivan de las anteriores:

- **A-03-1:** de D-03-8 se deduce que el esquema no puede fijarse en `<TIPO>-<DOM>-<NN>`. Cada proyecto declara su patrón de ID: por defecto `RF-<dominio>-<nº>`, y un proyecto migrado declara el patrón heredado del cliente. Los IDs nuevos continúan la serie del cliente (`RF-121`…). Es una regla solo para el ID, no la maquinaria de perfiles de documento que retiró P6.
- **A-03-2:** no adoptar proyectos reales con `axiom archive adopt` hasta que llegue C-03-c. Hoy `adopt` exige `REQ-n.m` (`internal/adoption/adoptcheck/check.go:147-148`) y obligaría a una segunda migración.
- **A-03-3:** D-03-4 se cumple con la propia delta, sin líneas nuevas escritas a mano.
  - MODIFIED, REMOVED y RENAMED citan el ID existente.
  - ADDED lleva un ID provisional.
  - `status` proyecta la lista de IDs que se modifican y se crean, derivada de la delta.
  - En la conversación se habló de líneas `Modifica:`/`Crea:` en ODD; la derivación desde la delta las hace innecesarias y evita duplicar la información.

## Cambios resultantes

Estos cambios forman el «cambio RF» de la consolidación. Se parten en unidades porque tocan capas distintas y su orden importa: a → b → (c ∥ d) → e.

### C-03-a · Puerto único de identificadores de requisito y patrón declarado por proyecto

- **Objetivo:** hoy el formato `REQ-<n>.<m>` está fijado en al menos cuatro regex independientes y el compositor empareja requisitos por título. Hay que concentrar parseo, validación y formato del ID en un solo punto, con un patrón que declara cada proyecto, y hacer que el compositor identifique cada requisito por su ID. Así los IDs funcionales son posibles sin tocar cada consumidor.
- **Decisiones:**
  - D-03-3, D-03-8 y A-03-1 (usuario y análisis).
  - Análisis: un paquete hoja nuevo (p. ej. `internal/reqid`) expone `Parse`, `Valid`, `Format` y la serie de un ID. Todos los consumidores pasan a usarlo y ninguno conserva su propia regex.
  - Análisis: el patrón se declara en el bloque `sdd:` de `axiom.yaml` (`internal/workspace/sdd.go:17`, hoy `sddKeys = []string{"iso"}`). Tiene decodificador estricto y Axiom no interpreta `openspec/config.yaml`. La clave concreta se decide en el design.
  - Análisis: sin declaración, el patrón por defecto debe admitir los `REQ-n.m` existentes hasta que C-03-e migre las specs de Axiom. Así no se rompe nada entre olas.
  - Análisis: el compositor empareja por ID cuando el título lo tiene, y por título si no. Renombrar el título deja de romper la identidad.
  - Análisis: RF-ISO-03 (`openspec/specs/sdd-iso-evidence/spec.md:244-246`) dice hoy literalmente «DEBE contener un identificador con la forma `REQ-<n>.<m>`». Se reescribe para que pida «un identificador válido según el patrón declarado del proyecto».
- **Fuera de alcance:** IDs provisionales y asignación (C-03-b), adopción (C-03-c), `axiom trace` (C-03-d) y la migración de las specs de Axiom (C-03-e).
- **Carril propuesto:** SDD. Modifica requisitos vivos (RF-ISO-03, la integración ODD REQ-30.x y el compositor) y cambia un contrato que consumen ISO, ODD y adopción.
- **Tamaño:** L.
- **Depende de:** nada de esta sección. **Bloquea a:** C-03-b, C-03-c, C-03-d y C-03-e.
- **Ficheros y paquetes que toca:**
  - `internal/openspecdoc/parse.go:33-36` (`requirementIDInTitle`, `RequirementID`).
  - `internal/openspecdoc/compose.go:33` (emparejado por título).
  - `internal/sddstatus/openspec_archive_compose.go:20` (delega en `openspecdoc`).
  - `internal/isoevidence/reqid.go:13-14,55-58` (`reqCandidate`, `reqValid` y mensajes).
  - `internal/isoevidence/cubre.go:21,159`.
  - `internal/livespec/registry.go:59,140-142` (`reqIDParts`, `NextFree`) y `internal/livespec/plan.go:270-308`.
  - `internal/livingdoc/indexer.go:17,136` (regex e ID sintético `REQ-<dominio>-<n>`).
  - `internal/adoption/adoptcheck/check.go:147-148,177,204` (reutiliza `livespec.SeriesOf`).
  - Mensajes en `internal/cli/archive_integrate.go:106` e `internal/cli/archive_adopt.go:54`.
  - `internal/workspace/sdd.go` y `axiom.example.yaml`. **[COMPARTIDO]: esquema de `axiom.yaml`.**
  - Guías que citan `REQ-<n>.<m>`: `internal/assets/skills/_shared/sdd-iso-convention.md`, `internal/assets/skills/_shared/openspec-convention.md` y `internal/components/agentguidance/routing.go:68,77`. **[COMPARTIDO]: guías de agente.**
  - Goldens en `testdata/golden/` (89 ficheros) que fijen ese texto. **[COMPARTIDO].**
  - Delta de `openspec/specs/sdd-iso-evidence/spec.md` (RF-ISO-03) y de las capacidades de integración ODD y adopción.
- **Criterio de terminado:**
  - `rg 'REQ-' --type go internal cmd -g '!*_test.go'` solo encuentra coincidencias en el paquete del puerto, en el valor por defecto y en comentarios.
  - Un proyecto de prueba con patrón `RF-<DOM>-<NN>` integra una delta ODD y archiva un cambio SDD sin errores.
  - El compositor aplica un MODIFIED cuyo título cambió pero conserva el ID.
  - RF-ISO-03 está reescrito.
  - `go test ./...` en verde (salvo el fallo conocido de `internal/components/filemerge` en Windows, `verify-report.md:23` de P6).
- **Riesgos y avisos:**
  - Cuatro paquetes tienen semánticas distintas: `livingdoc` acepta `REQ-[^)]+` y genera IDs sintéticos; `isoevidence` distingue un ID candidato de uno válido. Hay que conservar los mensajes de error tipados.
  - Las tres sugerencias que dejó el verify de P6 caen en esta zona: falta una prueba de paridad del marcador entre `isoevidence` y `livespec`, y `Motivo:` se interpreta distinto en ODD y en ISO (`openspec/changes/archive/2026-11-12-spec-viva-openspec/verify-report.md:62-64`). Conviene cerrarlas aquí.

### C-03-b · IDs provisionales, número asignado al integrar y declaración de impacto en `status`

- **Objetivo:** cumplir D-03-4. Todo incremento SDD u ODD deja reflejado qué IDs existentes toca y qué IDs nuevos crea. Los nuevos llevan un ID provisional, porque el número definitivo no se conoce hasta la integración. Además desaparece la colisión entre ramas concurrentes: `carril-sdd-versatil` tuvo que renumerar de REQ-28 a REQ-29.
- **Decisiones:**
  - D-03-4 y D-03-5 (usuario).
  - A-03-3 (análisis): la declaración es la delta.
  - Análisis: MODIFIED, REMOVED y RENAMED citan un ID existente, y Go comprueba que exista en `openspec/specs/`. Un ADDED lleva un ID provisional con el dominio ya fijado (p. ej. `RF-USU-N1`; el formato exacto se decide en el design).
  - Análisis: en la integración (archivado SDD y `axiom archive integrate --odd`), Go asigna el siguiente número libre de la serie según el patrón y reescribe el provisional en:
    - la spec viva;
    - las líneas `Cubre:` de las tareas;
    - el documento ODD.

    El mapa provisional→definitivo queda como traza en el cambio archivado o en el documento ODD.
  - Análisis: `axiom sdd status` (y el equivalente ODD) proyecta `requirements: {modifies: [...], creates: [...]}` derivado de la delta. Lo consume el cambio de integraciones (sección 07) para enlazar el work item antes de integrar.
  - Análisis: `ids.go` ya no se puede portar porque P6 lo borró; su lógica (provisional→definitivo, relleno de ceros, `IDMap`) se recupera del historial (`git show 1166efca^:internal/specbaseline/ids.go`) como referencia. P6 sacó del alcance la asignación (`proposal.md:63`) y eliminó el escenario `RF-9001`→`RF-008` (`proposal.md:84`).
- **Fuera de alcance:** el envío del ID a un tercero (sección 07) y `external_ref`, que hoy no existe en Go.
- **Carril propuesto:** SDD. Cambia el contrato de la delta, del archivado y de la integración ODD, y añade campos a `status`.
- **Tamaño:** L.
- **Depende de:** C-03-a. **Bloquea a:** C-03-c (los IDs nuevos de un proyecto migrado continúan la serie) y C-03-d. Para la sección 07 es una dependencia **blanda**: C-07-e solo la necesita si el resumen que sube al WI cita IDs funcionales (P12 del README). Que esta sección «bloqueaba» a la 07 era una suposición del análisis, no una decisión del usuario.
- **Ficheros y paquetes que toca:**
  - `internal/livespec/plan.go:270-308`: hoy exige el ID en el ADDED (`added_without_id`) y sugiere `nextFree` cuando hay colisión. Pasa a aceptar el provisional y a asignarlo.
  - `internal/livespec/registry.go:50-59,140-142`.
  - `internal/openspecdoc` (render y reescritura).
  - El archivado SDD: `internal/sddstatus/openspec_archive_compose.go` y `internal/cli/sdd_archive_compose.go`.
  - `internal/isoevidence/cubre.go` (reescritura de `Cubre:`).
  - `internal/sddstatus` (proyección `requirements`) y `internal/assets/skills/_shared/sdd-status-contract.md`. **[COMPARTIDO].**
  - `internal/assets/skills/sdd-spec/SKILL.md`, `internal/assets/skills/sdd-archive/SKILL.md` y `internal/components/agentguidance/routing.go:77`. **[COMPARTIDO].**
  - Goldens. **[COMPARTIDO].**
- **Criterio de terminado:**
  - Dos ramas concurrentes con un ADDED provisional en la misma capacidad se integran una tras otra sin renumerar a mano y reciben números distintos.
  - Un MODIFIED con un ID inexistente se rechaza con un error tipado.
  - `axiom sdd status --json` muestra `requirements.modifies` y `requirements.creates`.
  - La traza provisional→definitivo se puede leer después de archivar.
- **Riesgos y avisos:**
  - La reescritura debe ser atómica: si falla, ni la spec ni las referencias quedan a medias, igual que hoy `UnappliedDeltaError` nunca compone parcialmente.
  - La integración ODD en `monorepo-embedded` va en la PR de la feature, así que la asignación ocurre en la rama. Si dos PR asignan el mismo número, la segunda en fusionarse debe reintegrar. Hay que definir en el design si la asignación ocurre al integrar en la rama o tras la fusión.

### C-03-c · La adopción conserva los IDs del cliente y genera la spec viva con IDs funcionales

- **Objetivo:** cumplir D-03-7 y D-03-8. `axiom archive adopt` genera la spec viva OpenSpec con IDs funcionales. Si el origen ya trae los suyos, se conservan tal cual, y el proyecto queda declarado con el patrón heredado.
- **Decisiones:**
  - D-03-7, D-03-8 y A-03-1 (usuario y análisis).
  - Análisis: `adopt scan` ya detecta IDs ajenos con `foreignIDPattern` (`internal/adoption/scan.go:34`, que reconoce `RF|RNF|RN|REQ|US|HU|UC|CU|FR|NFR` con dominio opcional). Debe proponer el patrón que se declarará en `axiom.yaml`.
  - Análisis: `adopt check` deja de exigir `(REQ-n.m)` (`internal/adoption/adoptcheck/check.go:147-148`) y la «serie nueva única» (`check.go:160-204`). Pasa a validar contra el patrón declarado: IDs únicos, que respetan el patrón y nunca se reutilizan.
  - Análisis: `Origen: <ruta>#<ancla> (<ID original>)` (`check.go:265`) se mantiene como trazabilidad. Cuando el ID se conserva, el original coincide con el de la spec.
  - Análisis: un requisito del origen sin ID recibe un provisional y su número se asigna al integrar (C-03-b).
- **Fuera de alcance:** validar normas ISO publicadas (D-03-7).
- **Carril propuesto:** SDD. Modifica los requisitos de adopción RF-ADP-01 a RF-ADP-06.
- **Tamaño:** M.
- **Depende de:** C-03-a y C-03-b. **Bloquea a:** la adopción de proyectos reales (A-03-2).
- **Ficheros y paquetes que toca:**
  - `internal/adoption/scan.go`, `internal/adoption/discards.go:22` y `internal/adoption/adoptcheck/check.go`.
  - `internal/cli/archive_adopt.go`.
  - La skill o guía de adopción en `internal/assets/**`. **[COMPARTIDO].**
  - `docs/manual-de-inicio.md` (sección de adopción).
  - La delta de la capacidad de adopción.
- **Criterio de terminado:**
  - Al adoptar un origen con `RF-001`…`RF-120`, la spec viva conserva esos IDs, `axiom.yaml` declara el patrón y un requisito nuevo integrado después recibe `RF-121`.
  - Un origen sin IDs recibe IDs con el patrón por defecto.
  - `adopt check` falla si hay IDs duplicados o fuera del patrón.
- **Riesgos y avisos:** los IDs de cliente pueden no ser uniformes (mezcla de `RF-1` y `RF-001`, prefijos de varios tipos). Si el origen mezcla esquemas, el design debe decidir entre normalizarlos (pedido al humano) o admitir un patrón con alternativas.

### C-03-d · `axiom trace <ID>`: trazabilidad de un requisito de punta a punta

- **Objetivo:** dado un ID, responder qué incrementos (SDD archivados, documentos ODD), tareas (`Cubre:`), bugs y commits de integración lo crearon o tocaron. Hoy `axiom sdd trace` está ligado a un cambio (`internal/cli/sdd_trace.go:38`, `resolveTraceChangeRoot` en `:143`) y no cubre ODD.
- **Decisiones:**
  - Tema 3 del origen (usuario): «que los incrementos y los bugs se relacionen con ese ID para que todo quede trazado».
  - Análisis: la consulta lee los cambios archivados, los documentos `odd/tasks/*.md` integrados y las trazas provisional→definitivo de C-03-b. Tras C-03-e también lee el mapa `REQ-n.m`→nuevo, así que `axiom trace RF-PLH-03` responde con el ID nuevo.
  - Análisis: solo lectura, salida JSON versionada, sin estado nuevo y anclada en el commit de integración `--first-parent`, como el `sdd trace` actual.
- **Fuera de alcance:** el envío al tercero. Este verbo es la fuente que puede consumir la sección 07. **Aviso:** el tema 3 del origen pide «si hay integración con un tercero, enviarlo allí también», y ningún cambio de la sección 07 envía el ID funcional del requisito al WI (C-07-d vincula el cambio con el WI mediante `external_ref`, no los requisitos). Queda como decisión del usuario (ver `revision-03.md`).
- **Carril propuesto:** SDD. Es un verbo nuevo con contrato JSON.
- **Tamaño:** M.
- **Depende de:** C-03-a y C-03-b; del mapa de C-03-e para resolver IDs antiguos. **Bloquea a:** nada obligatorio. Le es útil a la sección 07.
- **Ficheros y paquetes que toca:**
  - `internal/cli/sdd_trace.go` (reutilizar o extraer).
  - `cmd/axiom/main.go` (registro del verbo, junto a `:2240`).
  - `internal/livespec` (lectura de los documentos ODD).
  - El manual.
- **Criterio de terminado:** `axiom trace <ID> --json` devuelve, para un requisito tocado por un cambio SDD y por una feature ODD, ambos incrementos con su commit de integración. Un bug SDD con `Cubre:` a ese ID aparece como bug.
- **Riesgos y avisos:** los documentos ODD antiguos no tienen traza de integración, así que la consulta debe tolerar que falte.

### C-03-e · Migración de las specs propias de Axiom de `REQ-n.m` a IDs funcionales

- **Objetivo:** cumplir D-03-6. Las specs de Axiom pasan al esquema funcional con un mapa versionado, sin reescribir la historia.
- **Decisiones:**
  - D-03-6 (usuario).
  - Análisis:
    1. El agente propone un código de dominio por capacidad y el humano lo aprueba; es la única parte que pide criterio.
    2. Go genera un mapa versionado `REQ-n.m → RF-<DOM>-<NN>`.
    3. Una herramienta Go reescribe mecánicamente, a partir del mapa, las specs vivas, el código, las pruebas, los goldens y los documentos ODD, en un commit propio.
    4. `openspec/changes/archive/` no se reescribe, porque es historia; el mapa resuelve los IDs antiguos (C-03-d).
  - Análisis: dimensión medida en `5f6b05b4`:
    - 553 requisitos (`### Requirement:`) en 71 capacidades de `openspec/specs/`.
    - 407 ficheros de `internal/` citan `REQ-n.m`, casi todos pruebas y pines.
    - 32 ficheros entre `cmd/`, `odd/` y `docs/`.
- **Fuera de alcance:** reescribir los cambios archivados.
- **Carril propuesto:** SDD. Toca las 71 capacidades vivas y el proyecto se declara con el patrón nuevo; la parte de criterio (códigos de dominio) necesita la compuerta `spec`.
- **Tamaño:** XL por volumen, pero mecánico. La herramienta y el mapa suman unas 300-500 líneas con pruebas, y la reescritura toca cientos de ficheros. Va en una sola PR, nunca encadenada. Lleva `size:exception` solo si el check de 400 sigue existiendo cuando se abra; C-02-e lo retira en la ola 5, antes de la ola 9 de este cambio (P13 del README).
- **Depende de:** C-03-a y C-03-b; C-03-d para la resolución de IDs antiguos. **Bloquea a:** nada. Conviene hacerlo antes de que la sección 07 enlace requisitos de Axiom con un tercero.
- **Ficheros y paquetes que toca:**
  - `openspec/specs/**`.
  - `internal/**` (pruebas y pines), incluido `internal/assets/assets_test.go`. **[COMPARTIDO].**
  - `testdata/golden/**`. **[COMPARTIDO].**
  - `odd/tasks/*.md`, `docs/**`, `openspec/INDEX.md` (se regenera con `axiom archive sync`) y `axiom.yaml`. **[COMPARTIDO]: esquema y valor.**
- **Criterio de terminado:**
  - `rg 'REQ-[0-9]+\.[0-9]+'` fuera de `openspec/changes/archive/` y del mapa no encuentra nada.
  - `go test ./...` en verde.
  - `axiom trace RF-PLH-03` resuelve al ID nuevo.
  - `INDEX.md` se regenera sin diferencias inesperadas.
- **Riesgos y avisos:**
  - Es el cambio más conflictivo del backlog por solapes. Cualquier cambio en vuelo que añada o cite `REQ-n.m` choca con él, así que debe ir en una ventana sin otros cambios SDD abiertos, o rebasar estos sobre el mapa.
  - Las cadenas `REQ-` de los mensajes de usuario y de las guías también cambian.

## Descartado

- **Integrar specs vivas en formatos distintos de OpenSpec (D5, KVP y otros) y la frontera del destino del archivado:** lo descartó el usuario («liar la manta»). Ya se retiró en P6 y en la corrección de `carril-sdd-versatil`.
- **Que ODD cumpla evidencias ISO o se use en proyectos ISO:** descartado por D-03-1. Ya está en código (`internal/workspace/sdd.go:20-22`).
- **Que el agente edite `openspec/specs/` a mano al cerrar ODD:** descartado en el análisis. La skill de archivado prohíbe fusionar a mano, y se resolvió con `axiom archive integrate --odd` (P6).
- **Que el cierre ODD genere un mini cambio SDD:** descartado en el análisis porque arrastraría la carpeta del cambio, la rama `-archive` y la compuerta `integration`.
- **Adoptar ya en P6 el ID provisional con número al integrar:** descartado por D-03-5 («Solo en el cambio RF»). Mientras tanto se acepta el riesgo de colisión entre ramas.
- **Mantener `REQ-<incremento>.<n>`:** descartado por D-03-3. Codifica el incremento que creó el requisito, no su dominio, y no sirve para negocio ni para trackers.
- **Validar una norma ISO publicada en la migración:** descartado por D-03-7.
- **Renumerar los IDs del cliente al migrar:** descartado por D-03-8.
- **Líneas `Modifica:`/`Crea:` escritas a mano en ODD:** las sustituye la derivación desde la delta (A-03-3). Si en el design se ve que el humano necesita verlas en el documento antes de escribir la delta, se puede recuperar como texto derivado, nunca como fuente.

## Correcciones a la consolidación

Contrastado con Engram `analisis-backlog/consolidacion` (#2167):

1. **Cifras de la migración.** La consolidación habla de «las 536 specs de Axiom». En `5f6b05b4` son **553 requisitos en 71 capacidades**, y además «specs» se refería a requisitos, no a capacidades.
2. **Pregunta pendiente de comprobar.** La consolidación pedía comprobar «si RF-ISO-03 quedó atado a REQ-n.m y si la asignación de IDs está en un solo punto». Comprobado:
   - RF-ISO-03 **sí está atado** a `REQ-<n>.<m>` (`openspec/specs/sdd-iso-evidence/spec.md:246`).
   - La asignación **no está en un solo punto**: hay al menos cuatro regex independientes (`openspec/specs` vía `internal/openspecdoc/parse.go:33`, `internal/isoevidence/reqid.go:13-14`, `internal/livespec/registry.go:59` e `internal/livingdoc/indexer.go:17`).
   - Por eso existe C-03-a como primer paso del cambio RF.
3. **`ids.go` no se portó.** La consolidación (tema 4, consumo por agentes) dice «si se retira specbaseline, mover contextdelta.go y plan.go». Eso ya se hizo en `internal/techcontext`. Pero la lógica de IDs provisionales de `specbaseline/ids.go` **se borró sin portar** (commit `1166efca`), y C-03-b debe rehacerla tomando el historial como referencia.
4. **El cambio RF es una secuencia, no un único cambio.** La consolidación lo trata como un paso. Aquí se divide en C-03-a → C-03-b → (C-03-c ∥ C-03-d) → C-03-e, porque la migración masiva (C-03-e) solo es segura con el puerto, la asignación y la traza ya fusionados.
5. **Dependencia de la sección 07.** La consolidación dice que integraciones «necesita RF y external_ref». Precisión: `external_ref` **no existe** en Go (solo `Referencia:` en el informe de bug, `internal/assets/skills/_shared/sdd-iso-convention.md:41`) y no es de esta sección: lo introduce C-07-d. Lo que esta sección le entrega es la proyección `requirements` de C-03-b y el verbo `axiom trace` de C-03-d, y la dependencia es blanda.
6. **Batería de pruebas (sección 06).** La consolidación dice que `pruebas.md` va «trazada al ID del requisito». Si llega antes que C-03-a, se trazará a `REQ-n.m` y la reescribirá C-03-e. Conviene que use el puerto de C-03-a.
