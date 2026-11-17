# 04 · Consumo por agentes: contexto técnico y skills heredadas

> Sección escrita por la sesión de análisis 4. Rutas verificadas contra `main` en `5f6b05b4` (2026-11-17). El análisis original se hizo sobre un `main` anterior (2026-11-11); las diferencias encontradas al reverificar están en «Correcciones a la consolidación».

## Origen

Dos temas del backlog original, con la misma pregunta de fondo: la pieza existe, pero ¿algún agente o skill la lee de verdad y en el momento adecuado?

1. **Contextos técnicos genéricos.** «Ver cómo integrar contextos técnicos de manera genérica, que puedan leerse y usarse y mantenerlos en el archivado. Creo que está hecho, pero le damos una vuelta por si faltan cosas o no está bien enlazado, y luego no tiene utilidad porque los agentes y las skills no saben leerlo ni cuándo.» Veredicto del análisis: **hay que darle una vuelta**.
2. **Heredar skills.** «Comprobar que las skills que se traen se puedan utilizar. No sé si hace falta algo que las revise y compruebe que están correctas y que las van a poder leer los distintos adapters.» Veredicto del análisis: **interesa**.

### Estado de partida (verificado en `5f6b05b4`)

**Contexto técnico.** La infraestructura existe y tiene tests:

- Declaración y parser: `TechnicalContextConfig` y `ContextMode` en `internal/workspace/declarations.go:14` y `:20`; `decodeTechnicalContext` en `:98`; validación contra disco `checkContextPaths` en `internal/workspace/validator.go:189`.
- Proyección: `projectTechnicalContext` en `internal/sddstatus/workspace_declarations.go:57`, único llamador en `:32` (`decorateWorkspaceDeclarations`, invocado desde `internal/sddstatus/status.go:531` en todo status, haya cambio o no).
- Delta `context-impact.md`: parser `techcontext.ParseDelta` en `internal/techcontext/delta.go:86`; guarda de archivado en `internal/sddstatus/workspace_declarations.go:118` y `:173-178`; integración con `axiom archive context <check|prepare|verify|show>` (`internal/cli/archive_context.go:22`).

Mapa de lectores en `main`:

| Fase | ¿Lee el contexto técnico? | Dónde |
|---|---|---|
| sdd-explore | Sí (`entry`; `paths` bajo demanda) | `internal/assets/skills/sdd-explore/SKILL.md:93` |
| sdd-propose | **No**. Su paso 3 solo lee `openspec/specs/` | `internal/assets/skills/sdd-propose/SKILL.md:67-73` |
| sdd-spec, sdd-tasks | No, por decisión explícita | `internal/assets/skills/_shared/project-baseline-convention.md:17` |
| sdd-design | Sí, y redacta el delta | `internal/assets/skills/sdd-design/SKILL.md:61`, `:67-71` |
| sdd-apply | Sí, y mantiene el delta | `internal/assets/skills/sdd-apply/SKILL.md:80`, `:337` |
| sdd-verify | Sí, con aviso no bloqueante | `internal/assets/skills/sdd-verify/SKILL.md:38`, `:45` |
| sdd-archive | Rutas del status; aplica el delta | `internal/assets/skills/sdd-archive/SKILL.md:123-137` |
| ODD | **No lee al explorar**. El paso 2 es genérico; solo el paso 7 (cierre) obliga a actualizar la doc mantenida y da el locator | `internal/components/agentguidance/routing.go:72` y `:77` |

La exploración es opcional para propose: `internal/assets/claude/sdd-orchestrator-workflow.md:187` (`sdd-propose | exploration (optional)`). Si se salta explore, propose escribe «Affected Areas», «Risks» y «Dependencies» sin contexto técnico.

**Skills heredadas.** Solo se validan las skills embebidas y solo en un test:

- Reglas en `TestSkillFrontmatterIsLintClean`, `internal/assets/skills_frontmatter_test.go:27`: frontmatter delimitado por `---`, `name` igual al directorio, `description` entrecomillada, de una línea, dentro del presupuesto de longitud y con `Trigger:`, y solo las claves `name`, `description`, `license`, `metadata`, `version`, `user-invocable`, `disable-model-invocation`.
- El registro es tolerante: `LoadSkill` en `internal/skillregistry/registry.go:382` usa el nombre del directorio si no hay frontmatter; `parseFrontmatter` en `:463`; colisiones resueltas sin aviso en `dedupeBySkillName` (`:512`); nombres reservados (`_shared`, `skill-registry`, prefijo `sdd-`) descartados sin aviso (`:30-36`).
- `Manager.Approve` (`internal/autoskill/manager.go:281`) copia la skill del buzón sin validar y no la instala en las carpetas de los runtimes; solo regenera el índice.
- `agentbuilder.Install` (`internal/agentbuilder/installer.go:20`) escribe el `SKILL.md` sin validar (`:26`).
- `axiom doctor` no comprueba skills: registro de comprobaciones en `internal/cli/doctor.go:157-166`; solo `internal/cli/doctor_footprint.go:100` mira `~/.agents/skills`, para detectar restos al desinstalar.
- Pi: Axiom no le instala skills (`SkillsDir` devuelve `""` en `internal/agents/pi/adapter.go:286`); le instala paquetes npm (`managedPackageSources`, `:52-59`). El registro escanea `~/.pi/agent/skills` con el comentario «Keep these source roots in sync with the gentle-pi skill-registry extension» (`internal/skillregistry/registry.go:50`).

## Cambios resultantes

### C-04-a · Bug del minero de autoskill: `SKILL.md` sin frontmatter válido

- **Objetivo.** El minero de autoskill genera un `SKILL.md` que empieza con `# Título`, pone `description:`, `trigger:` y `origin:` en el cuerpo y coloca el `---` después, sin bloque de frontmatter inicial (`internal/autoskill/miner.go:30-34`). Los runtimes no cargan bien esa skill y el registro la indexa con la descripción vacía. Resultado esperado: toda skill minada empieza con un frontmatter válido según las mismas reglas que `TestSkillFrontmatterIsLintClean`.
- **Decisiones.**
  - «Primero el bug del minero (SKILL.md sin frontmatter), después skillcheck» (usuario, al cerrar el orden del tema 2; recogido así en la consolidación).
  - Es un arreglo independiente y puede ir primero (análisis; aceptado por el usuario).
- **Fuera de alcance.** El validador `skillcheck` (C-04-d) y cambiar el flujo de `approve`.
- **Carril propuesto.** ODD: el defecto y la corrección están claros, sin decisiones abiertas.
- **Tamaño.** S (en la práctica XS).
- **Depende de / bloquea a.** No depende de nada. Bloquea a C-04-d: el validador rechazaría de entrada las skills minadas actuales. El README lo coloca en la ola 1.
- **Ficheros y paquetes que toca.**
  - `internal/autoskill/miner.go:30-34` (plantilla `SkillMD` del patrón `pattern-table-tests`) y el resto de plantillas `SkillMD` del mismo fichero.
  - Tests de `internal/autoskill/`.
  - Revisar si hay skills minadas ya versionadas con el formato roto en la raíz `skills/` (el análisis vio `skills/axiom-go-table-tests/SKILL.md`).
- **Criterio de terminado.** Un test de `internal/autoskill` comprueba, para cada patrón del minero, que el `SKILL.md` generado empieza con `---`, que `name` coincide con el directorio y que la `description` contiene `Trigger:`; `go test ./internal/autoskill/...` en verde.
- **Riesgos y avisos.** Mientras no exista C-04-d, las reglas del frontmatter se duplican en el test del minero; C-04-d debe unificarlas.

### C-04-b · ODD lee el contexto técnico al explorar

- **Objetivo.** Hoy ODD actualiza documentación mantenida que nunca ha leído: el paso 2 «Explore» (`internal/components/agentguidance/routing.go:72`) no menciona el contexto técnico y solo el paso 7 «Close» (`:77`) obliga a actualizar la doc. Resultado esperado: ODD lee el `entry` del contexto técnico al explorar y actualiza la documentación mantenida en el mismo commit de la unidad de trabajo, como hoy.
- **Decisiones.**
  - «lo segundo me vale» (usuario, ante la pregunta «¿quieres que ODD lea y mantenga el contexto técnico con el mismo rigor que SDD (delta + `integrate verify`), o te basta con que lo lea al explorar y actualice la documentación en el commit, como ahora?»). Es decir: ODD NO adopta el rigor de SDD; sin `context-impact.md` ni verificación de integración del delta.
  - ODD lee el `entry` del contexto técnico al explorar (paso 2) y actualiza la doc mantenida en el mismo commit de la unidad de trabajo (paso 7, como hoy) (análisis, aceptado por el usuario).
  - Las rutas se obtienen sin necesitar un cambio SDD: «comando de workspace que proyecte los locators sin cambio SDD» (análisis, aceptado por el usuario). **Ver aviso:** al reverificar en `5f6b05b4`, el paso 7 ya remite a `technicalContext.maintained.entry` y `paths` de `axiom sdd status --json`, y `axiom sdd status` proyecta el contexto aunque no se pase `--change`. La sesión que implante esto debe decidir con el usuario si el comando nuevo sigue haciendo falta o basta con `axiom sdd status --json`.
- **Fuera de alcance.** Delta `context-impact.md` en ODD; integración del delta con `axiom archive context`; cambios en SDD; la sección `## Spec viva` y `axiom archive integrate --odd`, que ya existen.
- **Carril propuesto.** ODD: cambio pequeño de texto de guía y, si se mantiene, un comando fino que reutiliza `projectTechnicalContext`.
- **Tamaño.** S.
- **Depende de / bloquea a.** No depende de otros cambios. Antes de empezar hay que cerrar con el usuario la decisión pendiente P6 del README (si el comando de workspace sigue haciendo falta). El README lo coloca en la ola 2. Comparte `routing.go` y `routing_test.go` con C-01-c, C-02-c, C-02-d, C-03-a, C-03-b, C-05-d y los de la 07: no puede ir en paralelo con ninguno de ellos.
- **Ficheros y paquetes que toca.**
  - [COMPARTIDO] `internal/components/agentguidance/routing.go:72` (paso 2) y `:77` (paso 7). El paso 7 sigue diciendo `technical_context.maintained` «in `axiom.yaml`» además del locator del status; la convención dice que los locators salen del status, no de `axiom.yaml`.
  - [COMPARTIDO] `internal/components/agentguidance/routing_test.go:206` (`TestRenderRoutingClosesODDWithTheMaintainedTechnicalContext`): hoy exige que ningún otro paso, incluido «2. **Explore.**», mencione el contexto técnico (`:260-268`) y que `technical_context` aparezca exactamente 2 veces (`:271`). Hay que reescribir esa regla a propósito.
  - [COMPARTIDO] Goldens o pines de texto del orquestador renderizado para todos los runtimes, si los hay para la sección de ODD.
  - Si se mantiene el comando: `cmd/axiom/main.go` (grupo `workspace`) y un adaptador en `internal/cli/` que reutilice `projectTechnicalContext` (`internal/sddstatus/workspace_declarations.go:57`), sin duplicarla.
- **Criterio de terminado.** El orquestador renderizado para cada runtime incluye en el paso 2 de ODD la lectura del `entry` del contexto técnico mantenido (y las rutas bajo demanda), con la forma exacta de obtener los locators; el test de routing fija la nueva frase en el paso 2 y en el paso 7; `go test ./internal/components/agentguidance/...` en verde.
- **Riesgos y avisos.**
  - `axiom sdd status --json` sin `--change` selecciona un cambio activo (en el `main` de Axiom devolvió `changeName: "e2e-cumulative"`). Por lectura de código, con 0 o con varios cambios activos `resolveStatus` devuelve un status bloqueado sin error (`internal/sddstatus/status.go:620` y `:624`, cubierto por `status_test.go:77-90`), y `Resolve` lo decora igualmente con `technicalContext` (`status.go:531`). Es decir, los locators saldrían también en esos casos, pero dentro de un status bloqueado con razones ajenas a ODD («No active OpenSpec changes», «selection is ambiguous»). No está probado en ejecución: la sesión hija debe comprobarlo con un test antes de cerrar P6. Ese ruido es el argumento para conservar un comando de workspace.
  - La prohibición actual del test es una decisión anterior deliberada (inc-24): revertirla debe quedar explicada en el commit.

### C-04-c · `sdd-propose` lee el `entry` del contexto técnico solo sin exploración

- **Objetivo.** La exploración es opcional para propose (`internal/assets/claude/sdd-orchestrator-workflow.md:187`). Cuando se salta, «Affected Areas», «Risks» y «Dependencies» de la propuesta se escriben sin contexto técnico. Resultado esperado: propose lee el `entry` solo cuando no recibe exploración.
- **Decisiones.**
  - «Sí, condicional» (usuario, ante la recomendación de que propose lea el `entry` solo cuando no reciba exploración).
  - La profundidad del contexto técnico sigue siendo cosa de design (análisis; el usuario preguntó «¿o debería ser cosa más de design realmente?» y aceptó la lectura condicional).
  - Con exploración, propose no lee nada nuevo; spec y tasks siguen sin leer (análisis, aceptado).
  - Se descarta hacer explore obligatorio antes de propose (ver «Descartado»).
- **Fuera de alcance.** Lectura de `paths` además del `entry`; cambios en spec, tasks o design.
- **Carril propuesto.** ODD: un párrafo de skill, una línea de convención y la ampliación de un test.
- **Tamaño.** S (en la práctica XS).
- **Depende de / bloquea a.** Independiente. La consolidación agrupaba C-04-b y C-04-c como «contexto técnico (S)», pero el README los separa (una PR por cambio): C-04-c va en la ola 3. Comparte la skill `sdd-propose` con C-07-d y `assets_test.go` (solape blando) con C-02-c, C-03-e y C-05-d.
- **Ficheros y paquetes que toca.**
  - [COMPARTIDO] `internal/assets/skills/sdd-propose/SKILL.md:67-73` (paso 3 «Read Existing Specs»).
  - [COMPARTIDO] `internal/assets/skills/_shared/project-baseline-convention.md:17` («Only `sdd-explore`, `sdd-design`, `sdd-apply` and `sdd-verify` are required to read the technical context; the other phases are not.»): añadir la excepción condicional de propose.
  - [COMPARTIDO] `internal/assets/assets_test.go:2194` (`TestReadingPhaseSkillsReadTheTechnicalContextEntry`): ampliar a `sdd-propose` con la condición.
  - Hash o digest de assets embebidos, si algún test fija el contenido de las skills.
- **Criterio de terminado.** `sdd-propose/SKILL.md` indica leer el `entry` de cada modo proyectado en `technicalContext` solo cuando no se recibió exploración, remitiendo a la convención; la convención recoge la excepción; el test de fases lectoras cubre propose; `go test ./internal/assets/...` en verde.
- **Riesgos y avisos.** La redacción debe dejar claro el disparador («no exploration was received»), para que propose no lo lea siempre y duplique lo que design lee después.

### C-04-d · Validador de skills `skillcheck`: puerta, `doctor` y `axiom skill check`

- **Objetivo.** Las skills que no vienen embebidas (minadas, aprobadas desde el buzón, generadas por `agentbuilder`, copiadas a mano en las carpetas de proyecto o usuario) no las valida nadie, y `axiom doctor` no mira skills. Resultado esperado: una única fuente de reglas que se aplica en el build, al traer una skill y bajo demanda.
- **Decisiones.**
  - «Validador, puerta y doctor» (usuario, eligiendo el alcance completo frente a «solo doctor» y «solo control en approve»).
  - Paquete Go `skillcheck` reutilizado por (a) el test de assets, como misma fuente de verdad; (b) una puerta en `axiom skill approve`, el minero y `agentbuilder`; (c) una comprobación «skills» en `axiom doctor`; y (d) un comando nuevo `axiom skill check [--json]` (análisis, aceptado por el usuario).
  - Comprobaciones: frontmatter al inicio y parseable; `name` igual al directorio; `description` con `Trigger:` y longitud; claves permitidas; referencias relativas dentro de la skill resolubles; nombres reservados y sombras o colisiones entre fuentes; alcance por adapter (la skill está en una carpeta que ese runtime carga); frescura del registro (huella frente a caché) (análisis, aceptado por el usuario).
  - Pi: «Dejar a gentle-pi» (usuario). Axiom no instala skills en Pi; `skillcheck` y `doctor` solo emiten un aviso informativo para Pi («las skills llegan por gentle-pi, fuera de Axiom»), sin bloquear.
- **Fuera de alcance.** Instalar skills en Pi; que `approve` instale la skill en las carpetas de los runtimes (hueco detectado, no decidido); retirar `skills/branch-pr/` (ver «Descartado»); límite de tamaño total de una skill (hueco detectado, no decidido).
- **Carril propuesto.** ODD (recomendación del análisis; el usuario no eligió carril). Las reglas ya existen en un test y el usuario cerró el alcance; no queda ambigüedad que justifique proposal, spec y design. Si el contrato de `axiom skill check --json` debe ser estable para terceros, SDD sería defendible: es la decisión pendiente P9 del README y la toma el usuario, no la sesión hija.
- **Tamaño.** M (~300-500 líneas con tests).
- **Depende de / bloquea a.** Depende de C-04-a (las skills minadas actuales no pasarían). El README lo coloca en la ola 5.
- **Ficheros y paquetes que toca.**
  - Nuevo `internal/skillcheck/` (no existe en `5f6b05b4`).
  - `internal/assets/skills_frontmatter_test.go:27`: pasa a usar `skillcheck`.
  - `internal/autoskill/manager.go:281` (`Approve`) y `internal/autoskill/miner.go`: puerta antes de promocionar y al generar.
  - `internal/agentbuilder/installer.go:20-26`: puerta antes de escribir.
  - `internal/cli/doctor.go:157-166` (registro de comprobaciones) y `internal/doctor/doctor.go:10` (IDs de comprobación): nueva comprobación «skills».
  - `cmd/axiom/main.go:375` (subcomando `skill`) y `:1481` (`runSkillGroup`): nuevo `check`.
  - `internal/skillregistry/registry.go:30-36`, `:382`, `:463`, `:512`: fuente de las rutas por runtime y de las reglas de sombra y reservados (leer, no necesariamente cambiar; dejar de descartar en silencio se resuelve informando desde `skillcheck`).
  - [COMPARTIDO] Documentación de comandos (`docs/`) si lista los subcomandos de `axiom skill` y las comprobaciones de `doctor`.
- **Criterio de terminado.**
  - `TestSkillFrontmatterIsLintClean` delega en `skillcheck` y sigue en verde para las 25 skills embebidas.
  - `axiom skill approve` rechaza una skill del buzón con frontmatter inválido y explica qué falló.
  - `axiom doctor` informa de skills con frontmatter roto, referencias rotas, colisiones o sombras y registro desactualizado; en Pi solo avisa.
  - `axiom skill check --json` devuelve un informe por skill y sale con código distinto de cero si hay errores.
  - `go test ./internal/skillcheck/... ./internal/autoskill/... ./internal/agentbuilder/... ./internal/cli/...` en verde (ojo: `internal/cli` ya no se ejecuta entero de una vez; trocear por paquetes o filtros).
- **Riesgos y avisos.**
  - Endurecer el registro (dejar de aceptar skills sin frontmatter) puede romper skills de usuarios que hoy funcionan; por eso `skillcheck` informa y solo bloquea en las puertas de entrada (`approve`, minero, `agentbuilder`), no en la carga del registro.
  - Aviso de «alcance por adapter»: varios runtimes comparten carpeta (`.agents/skills`, `~/.copilot/skills`); la comprobación debe razonar por carpeta, no por runtime.
  - Memoria del proyecto: no compilar binarios sueltos de `axiom` (el EDR los borra); probar el comando con `go test` en proceso.

## Descartado

- **ODD con el rigor de SDD** (delta `context-impact.md` y verificación de integración en ODD). Descartado por el usuario («lo segundo me vale»): basta con leer al explorar y actualizar en el commit.
- **Explore obligatorio antes de propose cuando hay contexto técnico declarado.** Descartado por el usuario al elegir «Sí, condicional»: obliga a explorar incluso en cambios pequeños, la misma fricción que otras secciones intentan quitar.
- **Que propose lea siempre el contexto técnico.** Descartado (análisis, aceptado): duplica la carga que design ya hace y lleva la propuesta hacia detalles de diseño.
- **Validador solo en `doctor`.** Descartado por el usuario: diagnostica, pero no impide que entren skills rotas.
- **Validador solo como puerta en `approve`.** Descartado por el usuario: no cubre las skills copiadas a mano en `.claude/skills` y similares.
- **Que Axiom instale skills en Pi.** Descartado por el usuario («Dejar a gentle-pi»): duplicaría lo que hace `gentle-pi` con riesgo de choques.
- **Huecos detectados que no se elevaron a cambio** (sin decisión del usuario; quedan fuera de este plan):
  - `## Registro de remediación` no tiene lector máquina: solo sirve para que cambie la huella de la compuerta y la reabra. Se acepta como lector humano.
  - Un campo tipo `context_read` en el sobre de retorno de las fases lectoras, para dejar constancia de qué contexto se leyó.
  - Un aviso en la validación de `axiom.yaml` cuando `technical_context.maintained` no declara `entry` (sin `entry` no se precarga nada).
  - `branch-pr` está retirada (`internal/components/skills/retired.go:27-29`) pero sigue existiendo `skills/branch-pr/` en la raíz del repositorio.

## Correcciones a la consolidación

Contrastado con Engram `analisis-backlog/consolidacion` (revisión del 2026-11-13) y con `main` en `5f6b05b4`:

1. **«Si se retira specbaseline, mover contextdelta.go y plan.go» está obsoleto.** `internal/specbaseline` ya se retiró (commit `1166efca`). El parser del delta vive ahora en `internal/techcontext/delta.go:86` (`ParseDelta`), la guarda de archivado sigue en `internal/sddstatus/workspace_declarations.go:173-178` y `axiom sdd integrate` se sustituyó por `axiom archive context <check|prepare|verify|show>` (`internal/cli/archive_context.go:22`). No queda nada que mover.
2. **«Comando de workspace para las rutas» necesita revisarse antes de implantarse.** Cuando se analizó, el paso 7 de ODD solo citaba `axiom.yaml`. En `5f6b05b4` el paso 7 (`internal/components/agentguidance/routing.go:77`) ya remite a `technicalContext.maintained.entry` y `paths` de `axiom sdd status --json`, y el status proyecta el contexto sin `--change`. La decisión del usuario se mantiene literal en C-04-b, pero el hueco real que queda es que el paso 2 no lee el contexto; el comando nuevo solo hace falta si `axiom sdd status` sin cambio no es fiable con 0 o varios cambios activos.
3. **Líneas desplazadas.** La frase de la convención está en `project-baseline-convention.md:17` (no `:18`), la fila de propose en `sdd-orchestrator-workflow.md:187` (no `:186`) y la compuerta que presenta el delta en `sdd-orchestrator-sections.md:90` (no `:88`). La referencia `sdd-archive/references/project-baseline-integration.md` pasó a `sdd-archive/references/technical-context-integration.md`.
4. El resto de lo consolidado para «Consumo por agentes» coincide con lo decidido: orden minero → `skillcheck`, `skillcheck` M que en Pi solo avisa, y contexto técnico S en paralelo.
