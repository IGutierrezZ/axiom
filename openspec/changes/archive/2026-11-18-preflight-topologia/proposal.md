# Propuesta: preflight y sellado derivados de la topología (C-01-b)

## Intención

El cuestionario de sesión SDD pregunta hoy tres cosas (ritmo, almacén de artefactos y estrategia de PR) aunque Axiom puede deducir dos de ellas, y el sellado del kickoff admite combinaciones de topología, ritmo y roles que luego rechaza o que el usuario ya descartó:

- El almacén: el usuario decidió que en Axiom es siempre `hybrid` («la pregunta engram / openspec / both debe desaparecer porque en Axiom siempre será both»). Los almacenes `engram` y `none` dejan sin las paradas de tres salidas, la publicación de compuertas y el relevo (`--hand-off`).
- El ritmo en `multirepo`: el usuario decidió (2026-11-11) que la topología lo fija. En `multirepo` el cambio se sella siempre `checkpointed` y el ritmo no se pregunta.
- Los roles en `monorepo-embedded`: el usuario decidió que un monorepo solo admite `fullstack` («los roles son para multirepo»), asumiendo la pérdida de `tasks.<rol>.md`, de la parada `role-apply` por rol y de las ramas por rol en monorepo.

El cambio quita del cuestionario lo deducible, hace que la topología decida el ritmo y los roles, y deja el esquema del cuestionario preparado para que otros cambios añadan o retiren grupos (C-02-c retira la estrategia de PR; C-06-b puede añadir uno) sin romper a los demás.

Éxito: en `multirepo` no se pregunta ni ritmo ni almacén y el kickoff se sella `checkpointed`; en `monorepo-embedded` se pregunta el ritmo (más lo que conserve la entrega Git) y el sellado solo admite `fullstack`; el almacén es `hybrid` en ambos.

## Alcance

### Dentro del alcance

1. **Almacén `hybrid` fijo.**
   - El cuestionario deja de preguntar el almacén; el bloque derivado de la sesión dice siempre `Artifact store: hybrid`.
   - `axiom workspace validate` rechaza un `artifact_store` declarado `engram` o `none` en `openspec/config.yaml`, con un mensaje que explica la migración (quitar la clave o declarar `hybrid`) y por qué (sin paradas de tres salidas, publicación ni relevo). `openspec` declarado o la ausencia de declaración se aceptan.
   - La carga (`workspace.Locate`) no se rompe con esos valores.
   - La doctrina (sección «Artifact Store Mode» y `persistence-contract.md`) pasa a decir `hybrid` fijo.
2. **Marcadores por nombre de grupo.** El marcador numerado de cada pregunta se sustituye por uno nominal, `<prefijo> pace:` y `<prefijo> pr-strategy:` (identificadores estables en minúsculas, con el prefijo vigente y el heredado). Transición: el hook y el plugin siguen aceptando los marcadores numerados heredados (1 → `pace`, 2 → `artifacts`, 3 → `pr-strategy`) hasta que el usuario ejecute `axiom sync`; una respuesta de almacén heredada se acepta y se ignora.
3. **Grupos por topología.**
   - El texto proyectado sigue siendo global; indica al agente que lea `workspace.topology` de `axiom.yaml` antes de preguntar.
   - `multirepo`: solo `pr-strategy`; el ritmo se deriva `interactive`.
   - `monorepo-embedded`: `pace` y `pr-strategy`.
   - El hook lee la topología con `workspace.Locate(payload.CWD)`, calcula los grupos requeridos y valida el significado, no la presentación: tolera e ignora grupos extra o heredados (en `multirepo` un ritmo contestado se ignora y el bloque dice `interactive`).
4. **Cuestionario vacío.** Si la topología no requiere ningún grupo (posible en `multirepo` tras C-02-c, P10), el hook suprime el cuestionario: permite el despacho y antepone el bloque enteramente derivado, sin exigir autoridad de transcripción. Mientras haya al menos un grupo requerido se conservan todas las protecciones de autoridad (un hijo no despacha, el texto del modelo no crea autoridad, el payload queda enlazado a la transcripción).
5. **Sellado por topología** (`axiom sdd kickoff seal`):
   - `multirepo`: rechaza `--execution-style continuous` y `--from-session-pace auto` con un mensaje que nombra la topología; sin estilo explícito deriva `checkpointed` sin error.
   - `monorepo-embedded`: rechaza cualquier `--role` distinto de `fullstack` con un mensaje que nombra la topología; sin `--role` sella `fullstack`.
   - La regla de #249 (`continuous` solo con `[fullstack]`) se conserva como red de seguridad. El retrosellado `--infer` queda exento.
   - El chequeo vive en la CLI (`runSDDKickoffSeal`, tras resolver el workspace); `ParseSealArgs` sigue pura.
6. **Roles en monorepo.**
   - `axiom workspace validate` rechaza en `monorepo-embedded` cualquier rol distinto de `fullstack`, con los pasos de migración; la carga no se rompe.
   - El alta (CLI del hub, detector y dashboard) en monorepo solo escribe `fullstack` y el formulario no ofrece roles. Los roles detectados (`core`, `web`, `qa`) solo se aplican en `multirepo`.
   - El perfil `knowledge`/`spec-only` sigue forzando `multirepo` con el rol `knowledge`, con prueba.
7. **Plugin de OpenCode** (las dos copias): paridad con el hook en marcador nominal, número variable de grupos, enlace etiqueta → token (hoy posicional) y tolerancia de heredados; registro del digest anterior en `opencode_plugin_digests.go`.
8. **Doctrina y goldens**: «Execution Mode» deja de decir «default to Automatic» y pasa a depender de la topología; «Artifact Store Mode» y `persistence-contract.md` dicen `hybrid` fijo; mismos cambios en todos los runtimes y en las reescrituras de `orchestrator.go`; regeneración de los goldens afectados.
9. **Specs vivas**: deltas de las capacidades listadas abajo, sin escenarios de monorepo con roles especializados.

### Fuera del alcance

- La estrategia de PR y el umbral de 400 líneas: los decide C-02-c (P10: la pregunta de estrategia de PR sale del preflight en C-02-c, no aquí). Este cambio conserva el grupo `pr-strategy` y la política de revisión de 400 líneas tal cual.
- La pantalla de ajustes del proyecto (C-06-e).
- La interacción propia de `explainer_html` (C-06-b); este cambio solo deja el esquema de marcadores preparado para ella.
- Cambiar el valor por defecto del almacén en el binario (`sddstatus` sigue devolviendo `openspec` sin declaración).
- Migrar automáticamente `axiom.yaml` u `openspec/config.yaml` existentes: solo se valida y se explica la migración.
- Reescribir `kickoff.yaml` existentes o archivados que incumplan las nuevas reglas: se siguen leyendo igual.

## Capacidades

### Nuevas capacidades

Ninguna.

### Capacidades modificadas

- `sdd-preflight-configuration`:
  - REQ-28.14: deja de fijar «tres grupos canónicos» y los marcadores numerados; pasa a grupos por topología con marcador nominal, transición de heredados y cuestionario vacío suprimido. La interacción de revisión independiente sigue sin reutilizar el prefijo ni los marcadores del cuestionario.
  - REQ-21.5: en `multirepo` la modalidad no se pregunta, se deriva `checkpointed`; en `monorepo-embedded` sigue exigiéndose explícita.
  - REQ-21.6: los roles especializados solo existen en `multirepo`; en `monorepo-embedded` el rol es siempre `fullstack` (sus escenarios de `core`/`web`/`qa` pasan a contexto `multirepo`).
  - REQ-28.12 y REQ-28.13: se conservan como red de seguridad, con la precedencia de los rechazos por topología (mensaje que nombra la topología) y los escenarios de varios roles situados en `multirepo`.
- `sdd-lane-kickoff`: REQ-21.3 solo si su texto supone un cuestionario siempre presentado; la fase de spec lo confirma (posiblemente sin delta).
- `change-branch-lifecycle`: tabla del glosario (`:36`) y escenarios de monorepo con roles `core`/`qa`/`archive` (REQ-25.4 `:267-273`, `:293-303`; REQ-25.11 `:548-552`), que pasan a `multirepo` o a `fullstack`.
- `multi-project-hub`: REQ-2.2 (el alta monorepo escribe solo `fullstack`) y el escenario de REQ-2.1 con roles detectados, acotado a `multirepo`.
- `workspace-topology`: requisito nuevo o modificado (REQ-2.2 «Topología Monorepo Embebido») para «monorepo solo `fullstack`» y para el rechazo de `artifact_store` `engram`/`none`.
- `project-technical-context`: REQ-30.19, escenario «Sin contexto mantenido el almacén no se exige» (`:526-529`): `axiom workspace validate` pasa a reportar `engram`/`none` aunque no haya contexto mantenido; el comportamiento del status y del archivado no cambia.

## Enfoque

Combinación de los enfoques 1 y 3 de la exploración: un único contrato Go de grupos por topología y el hook como autoridad que conoce la topología.

1. **Contrato de grupos en Go.** La tabla `sddPreflightGroups` (`internal/cli/sdd_preflight_hook.go`) pasa a estar indexada por identificador estable (`pace`, `pr-strategy`; `artifacts` solo como heredado) y una función devuelve los grupos requeridos para una topología. El bloque derivado se construye siempre con las mismas seis líneas: los valores no preguntados salen de la topología (`Pace: interactive` en `multirepo`, `Artifact store: hybrid` siempre). `validSDDPreflightHookBlock` se ajusta a esos dominios.
2. **Hook.** Lee la topología con `workspace.Locate(payload.CWD)`. Reconoce marcadores nominales y numerados heredados, exige exactamente un marcador por grupo requerido, tolera e ignora los grupos no requeridos o heredados, y enlaza por etiqueta (base de C-01-a). Con cero grupos requeridos permite el despacho con el bloque derivado sin consultar la transcripción. El texto proyectado global no conoce la topología; incluye la orden de leerla y las dos variantes de preguntas.
3. **Texto proyectado.** `sddSessionPreflightBody` y la variante sin interfaz nativa (`session_preflight.go:16`, `:58-69`) se reescriben con marcadores nominales y la sintaxis de respuesta por grupo presente; `validateSDDSessionPreflightProjection` sigue exigiendo igualdad exacta con el nuevo bloque canónico y deja de rechazar solo lo que ya no aplica.
4. **OpenCode.** El plugin replica el contrato. Si no puede leer la topología, acepta cualquier subconjunto no vacío de grupos conocidos y deja la decisión por topología al hook/binario; el diseño decide si puede leerla.
5. **Sellado.** En `runSDDKickoffSeal` (`internal/cli/sdd_kickoff.go`), tras `resolveSealChangeRoot` y antes de `ValidateForSeal`, una comprobación por topología con `loc.Config.Workspace.Topology`: rechaza `continuous`/`auto` en `multirepo` (deriva `checkpointed` sin estilo) y roles ≠ `fullstack` en `monorepo-embedded`. `--infer` no pasa por ella.
6. **Validador.** `internal/workspace/validator.go`: el paso de roles (`:99-116`) añade la regla de monorepo y los errores de almacén (`:268-285`) añaden el rechazo incondicional de `engram`/`none`. Ninguna de las dos reglas entra en `Locate`.
7. **Alta.** `planConfig` (`internal/hub/init_validate.go:174-205`) y `buildAxiomYaml` (`internal/hub/init.go:355-388`) aplican roles detectados o del formulario solo en `multirepo`; `projectform.js`/`projects.js` ocultan el constructor de roles en monorepo y `buildRoles` devuelve `fullstack`.
8. **Doctrina, goldens y specs** según el alcance.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/components/sdd/session_preflight.go` | Modificado | Texto del cuestionario, variante sin UI y validación de la proyección |
| `internal/components/sdd/orchestrator.go`, `inject.go` | Modificado | Reescrituras de doctrina en runtimes de reserva y anclas del cuestionario |
| `internal/components/sdd/opencode_plugin_digests.go` | Modificado | Digest de la revisión sustituida del plugin |
| `internal/cli/sdd_preflight_hook.go` | Modificado | Grupos por topología, marcador nominal, heredados, cuestionario vacío y bloque derivado |
| `internal/cli/sdd_kickoff.go` | Modificado | Comprobación de sellado por topología |
| `internal/kickoff/` (`args.go`, `schema.go`) | Modificado (mínimo) | Mensajes y ayuda; `ParseSealArgs` sigue pura y la regla #249 se conserva |
| `internal/workspace/validator.go` | Modificado | Monorepo solo `fullstack`; `artifact_store` `engram`/`none` rechazado |
| `internal/hub/` (`init_validate.go`, `init.go`, `detector.go`) | Modificado | Alta monorepo solo `fullstack` |
| `internal/dashboard/assets/js/core/projectform.js`, `projects.js` | Modificado | Formulario sin roles en monorepo |
| `internal/assets/opencode/plugins/`, `plugins-v2/sdd-task-result-artifacts.ts` | Modificado | Paridad con el hook |
| `internal/assets/*/sdd-orchestrator*.md`, `internal/assets/skills/_shared/sdd-orchestrator-sections.md`, `persistence-contract.md` | Modificado | Execution Mode por topología; almacén `hybrid` fijo |
| `testdata/golden/*` (11 ficheros según la exploración) | Regenerado | Proyecciones del cuestionario y de la doctrina |
| Tests de `internal/cli`, `internal/components/sdd`, `internal/assets`, `internal/sddtaskresult`, `internal/workspace`, `internal/hub`, `internal/dashboard` | Modificado | Marcadores, número de grupos, almacén y roles |
| `openspec/specs/{sdd-preflight-configuration,sdd-lane-kickoff,change-branch-lifecycle,multi-project-hub,workspace-topology,project-technical-context}` | Delta | Según «Capacidades modificadas» |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| Monorepos existentes con roles especializados (el detector crea `core`/`web`/`qa` en cualquier topología) fallan `axiom workspace validate` | Media | La carga no se rompe; el error nombra la topología y los pasos de migración (dejar solo `fullstack` o pasar a `multirepo`); los `kickoff.yaml` existentes se siguen leyendo |
| Proyectos con `artifact_store: engram`/`none` declarado empiezan a fallar la validación | Baja | Mensaje con la migración; `Locate`, status y archivado no cambian |
| Instalaciones sin `axiom sync` proyectan el texto antiguo con marcadores numerados y la pregunta de almacén | Alta (transitoria) | Hook y plugin aceptan los heredados y la respuesta de almacén se ignora; validar el significado y no la presentación |
| Texto proyectado global con dos variantes de preguntas: el agente puede no leer la topología y preguntar de más o de menos | Media | El hook es la autoridad: tolera grupos extra y deniega con causa si falta uno requerido |
| El cuestionario vacío relaja la autoridad de transcripción | Media | Solo con cero grupos requeridos según la topología leída por el propio hook; con al menos un grupo se conservan todas las protecciones; pruebas de ambos casos |
| El plugin de OpenCode no conoce la topología | Media | Acepta subconjuntos no vacíos de grupos conocidos y la decisión por topología recae en hook/binario; el diseño decide si puede leerla |
| Goldens y tests con literales del cuestionario (≈40 puntos, 11 goldens) | Alta (esfuerzo) | Regeneración con `-update` y revisión del diff; el tamaño puede superar 400 líneas (`size:exception` aceptado) |
| El bloque de sesión dice `hybrid` mientras `axiom sdd status` sigue resolviendo `openspec` sin declaración, y la doctrina dice a las fases que lean el almacén que les inyecta el status | Media | Para ficheros `openspec` ≡ `hybrid` en el binario; el diseño fija qué valor transmite el orquestador a las fases y lo deja explícito en la doctrina |
| La regla de monorepo y la de multirepo dejan REQ-28.12/28.13 inalcanzables por la vía normal | Baja | Se conservan como red de seguridad con escenarios que lo explicitan |

## Plan de reversión

- Revertir la PR única del cambio: restaura el texto de tres grupos, el hook con marcadores numerados, la doctrina y los goldens. Como el hook acepta los marcadores heredados durante la transición, un despliegue parcial (binario nuevo con texto antiguo o al revés) no deniega preflights válidos.
- Tras revertir, ejecutar `axiom sync` para reproyectar el texto anterior; el digest añadido en `opencode_plugin_digests.go` se elimina con la reversión.
- No hay migraciones de datos: no se reescribe ningún `axiom.yaml`, `openspec/config.yaml` ni `kickoff.yaml`, así que no hay estado que restaurar.

## Dependencias

- C-01-a (hook por etiqueta): según la exploración, su cambio ya está en el código base (el texto ya no exige el orden de las opciones); el plugin de OpenCode aún enlaza por posición y lo corrige este cambio.
- P10 resuelta (2026-11-17): la estrategia de PR sale del preflight en C-02-c; este cambio deja el hook preparado para el cuestionario vacío.
- Bloquea a C-02-c (mismo bloque del preflight y del hook) y a C-06-c; por inferencia a C-06-b, C-06-e y C-07-d.

## Criterios de éxito

- [ ] En un `multirepo`, el preflight no muestra la pregunta de ritmo ni la de almacén; el bloque derivado dice `Pace: interactive` y `Artifact store: hybrid`, y el kickoff se sella `checkpointed`.
- [ ] `kickoff seal --execution-style continuous` (y `--from-session-pace auto`) en `multirepo` se rechaza con un mensaje que nombra la topología.
- [ ] En un `monorepo-embedded`, el preflight solo pregunta el ritmo y la estrategia de PR, el almacén es `hybrid` y el sellado solo admite `fullstack`.
- [ ] `axiom workspace validate` rechaza un monorepo con roles distintos de `fullstack` y un `artifact_store` `engram`/`none`, con mensajes que explican cómo migrar; `workspace.Locate` sigue cargando ambos.
- [ ] El alta monorepo (CLI y dashboard) solo escribe `fullstack`; el perfil `knowledge`/`spec-only` sigue forzando `multirepo` con `knowledge` (prueba).
- [ ] El hook acepta marcadores nominales y heredados, ignora grupos extra y la respuesta de almacén heredada, y con cero grupos requeridos permite el despacho con el bloque derivado; los casos de autoridad (hijo, texto del modelo, payload sin transcripción) siguen denegando cuando hay grupos requeridos.
- [ ] El plugin de OpenCode tiene paridad con el hook y su digest anterior está registrado.
- [ ] Las specs vivas no contienen escenarios de monorepo con roles especializados.
- [ ] `go test ./internal/kickoff/... ./internal/cli/... ./internal/workspace/... ./internal/hub/... ./internal/components/...` y `go test ./internal/components/ -run TestGolden` en verde.
