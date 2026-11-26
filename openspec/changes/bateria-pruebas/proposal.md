# Propuesta: batería de pruebas de aceptación (`pruebas.md`) generada en sdd-tasks

Cambio `bateria-pruebas` (C-06-c del plan maestro, `docs/plan/backlog-2026-11/06-ui-human-loop.md:119-180`). Ola 11, en paralelo con C-07-d.

## Intención

Los equipos de un multirepo necesitan un listado de pruebas de aceptación para probar en local, desarrollo o preproducción. Ese listado debe cubrir todos los requisitos y escenarios del cambio con el menor número de casos, poder revisarse antes de implementar y verse en la UI.

Hoy no existe nada parecido. Las tareas citan requisitos con `Cubre:`, pero nadie comprueba que cada escenario Given/When/Then tenga una prueba manual o automatizable, ni en qué entornos se ejecuta.

El cambio añade un artefacto opcional, `pruebas.md`, que sdd-tasks escribe después del design. Su cobertura la comprueba Go de forma determinista. El modelo aplica la minimización y Go solo comprueba e informa.

## Alcance

### Dentro del alcance

1. **Declaración de proyecto.**
   - Nuevo bloque de primer nivel en `axiom.yaml`: `acceptance_tests: { enabled: bool, environments: [string] }`.
   - Se registra en `internal/workspace/declarations.go` con su par bloque/error y un accesor que nunca asume un valor (patrón de `ExplainerHTML()`).
   - El decodificador es estricto: claves cerradas; `enabled: true` exige `environments` no vacío y sin duplicados.
   - `workspace.Validate` rechaza `enabled: true` fuera de `multirepo`, en el paso 3b, junto a la regla de roles de RF-TOP-04.
   - Se conserva en `hub/init.go` `declares()` y se documenta en `axiom.example.yaml`, `docs/manual-de-inicio.md` y `docs/openspec-config.md`.
2. **Formato de `pruebas.md`.**
   - Una tabla markdown parseable de forma determinista.
   - Columnas: `ID` (`P-NN` estable), `Cubre`, `Rol`, `Tipo`, `Prioridad`, `Entornos`, `Precondiciones`, `Pasos` y `Resultado esperado`. Las cabeceras se localizan por nombre.
   - `Cubre` es una lista de `ID` o de `ID#<nombre exacto del escenario>`. Los IDs se reconocen solo con el puerto `internal/reqid`: `REQ-n.m`, `RF-<DOM>-<NN>`, provisionales y `sdd.id_pattern` del proyecto.
   - `Tipo` toma uno de dos valores: `manual` o `automatizable`.
   - `Prioridad` toma uno de tres valores: `alta`, `media` o `baja`.
   - `Entornos` debe ser un subconjunto de `acceptance_tests.environments`.
   - `Rol` debe ser un rol sellado del cambio.
   - El separador de listas dentro de una celda lo fija la spec. La propuesta es `;`, porque los nombres de escenario pueden llevar comas.
3. **Paquete `internal/acceptancetests`.**
   - Expone `Check`: parsea `pruebas.md` y lo cruza con los requisitos y escenarios ADDED y MODIFIED de las specs delta del cambio. REMOVED y RENAMED quedan fuera.
   - Informa de los escenarios cubiertos (M/M), de los requisitos cubiertos y del número de casos.
   - Un escenario solo se cubre si se cita explícitamente con `ID#escenario`. Un requisito se cubre con su ID suelto o con cualquiera de sus escenarios.
   - Son hallazgos que hacen fallar la comprobación:
     - un requisito o un escenario sin cubrir;
     - un requisito de la delta sin ID citable;
     - un ID o un escenario citado que no existe en la delta;
     - un nombre de escenario ambiguo dentro de un requisito;
     - un `P-NN` duplicado o mal formado;
     - un tipo, una prioridad, un entorno o un rol fuera de su dominio;
     - una tabla ausente o con una forma no válida.
   - Reutiliza `internal/isoevidence`: exporta sus ayudantes de tabla y de líneas (`splitLines`, `isTableLine` y `splitCells`) y usa `Scenarios` y `AffectedFromDelta`, sin duplicarlos.
4. **Verbo `axiom sdd tests check --cwd <repo> --change <c> [--json]`.**
   - Es de solo lectura. Tiene un subdespachador `tests` como el de `gate` y se registra en `cmd/axiom/main.go`, en su ayuda y en `main_test.go`.
   - Sale con error cuando hay hallazgos, o cuando la función está activa y falta `pruebas.md` después de tasks.
5. **Proyección `acceptanceTests` en `axiom sdd status`.**
   - Sigue el patrón de `sddstatus/explainer.go` y es aditiva: no toca `nextRecommended`, las dependencias ni los digests.
   - Proyecta el estado vivo de `axiom.yaml` (activación y entornos, o el error del bloque) y, si existe `pruebas.md`, su cobertura y el número de hallazgos.
   - Si la función está activa y falta `pruebas.md` después de tasks, emite un aviso en el status, sin bloquear la puerta ni el avance.
   - El contrato se documenta en `_shared/sdd-status-contract.md`.
6. **Digest de la puerta `tasks`.**
   - `pruebas.md` entra en el digest solo cuando existe.
   - Las dos rutas espejo, `cli/sdd_gate_digest.go` (`tasksArtifactPathsForDigest`) y `sddstatus/governance.go` (`governanceArtifactInputs`), resuelven el conjunto de ficheros con una única función compartida.
   - Editarlo cambia el digest. Una puerta `rejected` vuelve a `pending`, reabierta. Una puerta `approved` sigue aprobada, como con `tasks.md` (ver la decisión 1).
7. **Archivado.** `internal/livespec/change.go` reescribe los IDs provisionales también en la columna `Cubre` de `pruebas.md`, como un tercer `changeFileKind`.
8. **Dashboard.**
   - Se añade un campo nuevo en `IncrementDetailDTO` de `GET /api/increments/<n>` con el contenido de `pruebas.md` y su cobertura. La cobertura se calcula con el mismo `acceptancetests.Check`. No hay ruta nueva.
   - La pestaña «Pruebas» es de solo lectura, en `detalle.js`, con un módulo propio (`detalle-pruebas.js`) que reutiliza los patrones de C-06-a y C-06-b.
   - Se actualizan los tests de pines de pestañas.
9. **Doctrina de sdd-tasks.**
   - `internal/assets/skills/sdd-tasks/SKILL.md` escribe `pruebas.md` solo cuando el cambio es SDD y el status proyecta `acceptanceTests` activo.
   - Aplica la minimización por capas:
     1. trazado obligatorio;
     2. particiones de equivalencia y valores límite;
     3. pairwise solo con tres o más parámetros independientes;
     4. fusión de casos con las mismas precondiciones.
   - Después ejecuta `axiom sdd tests check`.
   - En la parada `tasks` presenta `pruebas.md` junto a los ficheros de tareas.
   - La doctrina común va en un `_shared/sdd-acceptance-tests.md` nuevo, con un test de pines.
   - Se actualizan los agentes delgados existentes (`assets/claude/agents/sdd-tasks.md`, `assets/cursor/agents/sdd-tasks.md` y `assets/copilot/agents/sdd-tasks.agent.md`) y el golden `testdata/golden/sdd-claude-agent-sdd-tasks.golden`.
10. **Specs vivas.** Las deltas de las capacidades listadas abajo.

### Fuera del alcance (del plan)

- ODD: nunca genera `pruebas.md`.
- Monorepo: `enabled: true` se rechaza.
- Marcas de ejecución por entorno (C-06-d).
- Integración con gestores de pruebas, como Azure DevOps Test Plans.
- Generar código de test.
- Un generador pairwise determinista en Go: la v1 solo comprueba.
- Sellar `acceptance_tests` en `kickoff.yaml`.
- Editar el texto compartido del orquestador (`_shared/sdd-orchestrator-sections.md`) y sus 10 goldens.
- Editar la skill `sdd-apply`: la toca C-05-d en paralelo. Ver Riesgos.
- Agentes de kiro o kimi para sdd-tasks, porque no existen. El plan está desfasado en ese punto.

## Decisiones tomadas por el coordinador (con motivo)

1. **Reapertura: el mismo trato que `tasks.md`.**
   - Una puerta `approved` es terminal: D-08, `internal/kickoff/machine.go:254`, fijado por `TestEvaluateGatesApprovedGateNeverInvalidatedByDigestChange`. Reabrirla bloquearía a `sdd-apply` contra sus propias casillas de `tasks.md`.
   - Se aplica la intención del plan («se reabre como cualquier artefacto de tareas») y no su letra.
   - `pruebas.md` entra en el digest de `tasks`. Una puerta rechazada se reabre al editarlo y una aprobada sigue aprobada.
   - **Desvío documentado.** El criterio literal «editar tras aprobar reabre la puerta (test)» se sustituye por dos tests:
     - editar `pruebas.md` produce un digest distinto;
     - una puerta `tasks` rechazada pasa a `pending` (reabierta) tras editar solo `pruebas.md`.
2. **Bloque `acceptance_tests` de primer nivel.**
   - Lo fija el plan, y `sdd:` es de claves cerradas y con semántica ISO.
   - Se rechaza fuera de `multirepo` en `workspace.Validate`, no en la carga, como RF-TOP-04. Así `Locate` y el archivado toleran el error.
3. **Sin sellar en `kickoff.yaml` y sin tocar el texto del orquestador.**
   - La activación se lee del estado vivo proyectado en el status, y la parada `tasks` la presentan la skill y el status.
   - Así se evitan las dos zonas de solape duro de la ola (kickoff y doctrina del orquestador) y los 10 goldens.
4. **Con la función activa y sin `pruebas.md`: solo informa.** Hay un aviso en el status y `check` sale en rojo. La puerta y el avance no se bloquean.
5. **Sintaxis de `Cubre`: `ID` o `ID#<escenario>`, solo vía `reqid`.**
   - Hay un único puerto de IDs.
   - Un requisito sin ID es un hallazgo que hace fallar la comprobación. Así la cobertura M/M no lo ignora en silencio.
6. **Cobertura sobre ADDED y MODIFIED.** Es la misma base que `AffectedFromDelta`. REMOVED y RENAMED no llevan escenarios que probar.
7. **Dominios cerrados.**
   - Los entornos deben estar en `environments` y el rol debe estar sellado.
   - `Tipo` toma uno de dos valores: `manual` o `automatizable`.
   - `Prioridad` toma uno de tres valores: `alta`, `media` o `baja`.
8. **También en `continuous`.** Se genera igual. En multirepo el sellado es siempre `checkpointed`, así que el caso es teórico.
9. **Reescribir provisionales en `pruebas.md` al archivar.** Evita citas huérfanas tras integrar el cambio.
10. **Reutilizar `isoevidence`.**
    - Se exportan sus ayudantes en lugar de duplicarlos.
    - Un único `acceptancetests.Check` sirve al verbo, al status y al dashboard. Así toda función exportada nueva es alcanzable desde `cmd/axiom` (ratchet de código muerto).
11. **sdd-tasks escribe y sdd-apply solo refina rutas o URL.** sdd-apply no añade alcance. La regla de refinado vive en la doctrina compartida nueva y no se edita la skill `sdd-apply`.

## Capacidades

### Capacidades nuevas

Ninguna. La batería se reparte entre las capacidades existentes, como C-06-b.

### Capacidades modificadas

- `workspace-topology` (RF-TOP-19):
  - el bloque `acceptance_tests:` es opcional y estricto;
  - `enabled: true` exige entornos no vacíos y sin duplicados;
  - `Validate` rechaza `enabled: true` fuera de `multirepo`;
  - la plantilla y el manual lo documentan.
- `sdd-lane-kickoff` (RF-KOF-17 y siguientes):
  - sdd-tasks genera `pruebas.md` solo en SDD y solo con `acceptanceTests` activo en el status; ODD nunca lo genera;
  - se fijan el formato `P-NN`, sus columnas y sus dominios;
  - el status proyecta el estado vivo de `acceptance_tests` y la cobertura;
  - existe el verbo `axiom sdd tests check`, con el cálculo M/M y sus hallazgos;
  - la activa sin el fichero produce un aviso sin bloqueo;
  - el archivado reescribe los provisionales en `pruebas.md`.
- `sdd-block-review-gates` (RF-GAT-31):
  - `pruebas.md`, cuando existe, entra en el digest de la puerta `tasks` en las dos rutas espejo;
  - una puerta rechazada se reabre al editarlo y una aprobada sigue terminal (D-08);
  - la parada `tasks` presenta `pruebas.md` junto a los ficheros de tareas.
- `local-web-dashboard` (RF-WEB-17). **Añadida por esta fase sobre la lista del coordinador:**
  - la pestaña «Pruebas» es de solo lectura y el detalle del incremento gana el campo de la batería, sin ruta nueva;
  - el precedente es C-06-b, que fijó su pestaña en RF-WEB-16;
  - si el coordinador prefiere no tocar esta spec, el comportamiento de la UI quedaría sin requisito.

## Enfoque

Se sigue el enfoque 1 de la exploración: superficie mínima en lo compartido.

**Flujo.** El flujo va del bloque de `axiom.yaml` al estado vivo del status. De ahí pasa a la skill sdd-tasks, que escribe `pruebas.md`, y a `acceptancetests.Check`. Un único cálculo sirve al CLI, al status y al dashboard.

**Digest.** El conjunto de ficheros del digest de `tasks` se centraliza en una función compartida por las dos rutas espejo, para que no diverjan.

**Compatibilidad.**
- Sin el bloque o con `enabled: false` nada cambia: no hay `pruebas.md`, el digest es el mismo y no hay aviso.
- Los cambios en curso sin `pruebas.md` conservan su digest.

**Pruebas.** Las pruebas end-to-end usan fixtures multirepo (`workspace/workspacetest/fixtures.go`). Este repositorio es `monorepo-embedded` y no puede activar la función con su propio `axiom.yaml`.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/workspace/declarations.go`, `types.go`, `loader.go`, `validator.go`, `acceptance_tests.go` (nuevo) | Modificado/Nuevo | Bloque, decodificador estricto, accesor y regla de topología en `Validate` |
| `internal/hub/init.go` | Modificado | `declares()` conserva el bloque con `--force` |
| `internal/isoevidence` | Modificado | Exporta los ayudantes de tabla y de líneas |
| `internal/acceptancetests` (nuevo) | Nuevo | Parser de `pruebas.md` y `Check` de cobertura |
| `internal/cli/sdd_tests.go` (nuevo), `cmd/axiom/main.go`, `main_test.go` | Nuevo/Modificado | Verbo `sdd tests check` |
| `internal/cli/sdd_gate_digest.go`, `internal/sddstatus/governance.go`, `status.go`, `status_v2.go`, `acceptance_tests.go` (nuevo) | Modificado/Nuevo | Digest espejo y proyección `acceptanceTests` |
| `internal/livespec/change.go` | Modificado | Reescritura de provisionales en `pruebas.md` |
| `internal/dashboard/types.go`, `service.go`, `assets/index.html`, `js/core/router.js`, `js/screens/detalle.js`, `detalle-pruebas.js` (nuevo) y tests de assets | Modificado/Nuevo | Campo del DTO y pestaña «Pruebas» |
| `internal/assets/skills/sdd-tasks/SKILL.md`, `_shared/sdd-acceptance-tests.md` (nuevo), `_shared/sdd-status-contract.md`, los agentes `sdd-tasks` (claude, cursor y copilot) y `testdata/golden/sdd-claude-agent-sdd-tasks.golden` | Modificado/Nuevo | Doctrina de generación y test de pines |
| `axiom.example.yaml`, `docs/manual-de-inicio.md`, `docs/openspec-config.md` | Modificado | Documentación de la clave |
| `openspec/specs/{workspace-topology,sdd-lane-kickoff,sdd-block-review-gates,local-web-dashboard}` | Delta | Requisitos nuevos |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Desvío respecto del criterio literal de reapertura | Alta (es deliberado) | Se documenta en la propuesta y en la spec, y dos tests lo sustituyen (decisión 1) |
| Las dos rutas espejo del digest divergen | Media | Una única función compartida y un test que compare el digest de `gate record` con el del status |
| Ratchet de código muerto: un ayudante exportado solo lo usan los tests | Media | Todo pasa por `acceptancetests.Check`, alcanzable desde el CLI, el status y el dashboard |
| Nombres de escenario libres, con comas, `\|` o duplicados fuera de ISO | Media | Separador `;`, hallazgo por ambigüedad y nombre exacto; la gramática la cierra la spec |
| La regla «sdd-apply solo refina» no está en la skill `sdd-apply` | Media | Vive en `_shared/sdd-acceptance-tests.md`; enlazarla desde `sdd-apply` queda pendiente tras C-05-d |
| Solape duro con C-02-c en `sdd-tasks` y con otros cambios en `sddstatus`, `sdd_gate_digest.go` y el dashboard | Media | Ola 11 sin solape duro con C-07-d; quien fusione segundo rebasa |
| Un `_shared/` nuevo puede exigir registrarlo en los manifiestos de assets o en las proyecciones | Baja | Seguir el precedente de `_shared/sdd-explainer-html.md` y su test |
| Calidad de la minimización y más tokens en sdd-tasks | Media | Go comprueba la cobertura de forma determinista; la minimización es responsabilidad del modelo en la v1 |
| Sin dogfooding: el repositorio es monorepo | Alta | Fixtures multirepo en los tests de `workspace`, `sddstatus`, `cli` y `dashboard` |
| Se amplía la lista de capacidades (`local-web-dashboard`) | Baja | Lo decide el coordinador; ver Capacidades |

## Plan de reversión

1. Revertir la PR única del cambio con `git revert`. No hay migraciones de datos ni cambios en `kickoff.yaml`.
2. Un binario anterior ignora el bloque `acceptance_tests` de un `axiom.yaml`, porque las claves de primer nivel no registradas se ignoran. Los ficheros `pruebas.md` ya escritos quedan como documentos inertes.
3. Al salir `pruebas.md` del digest, una puerta `tasks` rechazada que lo incluía vuelve a `pending`; una aprobada no cambia, por D-08. Se avisa a quien tenga cambios multirepo abiertos con la función activa.
4. Si solo falla la UI o la doctrina, se pueden revertir aparte sus commits de unidad (pestaña, o skill y golden) sin tocar el núcleo de cobertura.

## Dependencias

- C-06-a fusionado: el detalle del incremento y `readArtifactFiles`.
- C-01-b fusionado: los roles solo en multirepo (RF-TOP-04 en `Validate`).
- C-03-a y C-03-b fusionados: puerto `reqid` con `RF-<DOM>-<NN>`, provisionales y `sdd.id_pattern`.
- C-06-b fusionado: el patrón de proyección y pestaña que se reutiliza.
- Bloquea a C-06-d (marcas de ejecución).

## Criterio de terminado

- [ ] Con `acceptance_tests.enabled: true` en un fixture multirepo, la doctrina de sdd-tasks pide escribir `pruebas.md` con IDs `P-NN`, y un test de pines lo fija.
- [ ] `axiom sdd tests check` falla si algún requisito o escenario ADDED/MODIFIED de las specs del cambio no está referenciado, e informa de los escenarios M/M, de los requisitos cubiertos y del número de casos, con texto y `--json`.
- [ ] `check` falla con cada uno de estos hallazgos, y hay un test por hallazgo:
  - un requisito sin ID;
  - un ID o un escenario inexistente;
  - un entorno no declarado;
  - un rol no sellado;
  - un tipo o una prioridad fuera de su dominio;
  - un `P-NN` duplicado.
- [ ] En un monorepo, `enabled: true` se rechaza en `workspace.Validate`. `enabled: true` con `environments` vacío o duplicado se rechaza al decodificar.
- [ ] Con `enabled: false` o sin bloque, el status no proyecta aviso, el digest de `tasks` no cambia y la doctrina no genera nada.
- [ ] En ODD nunca se genera: la doctrina lo condiciona a SDD y un test de pines lo fija.
- [ ] Editar `pruebas.md` cambia el digest de la puerta `tasks`, y es el mismo en `gate record` y en el status.
- [ ] Una puerta `tasks` rechazada pasa a `pending` (reabierta) tras editar solo `pruebas.md`. Este test sustituye al criterio literal del plan; ver la decisión 1.
- [ ] `axiom sdd status` proyecta `acceptanceTests` (activación, entornos, cobertura y aviso por ausencia) sin alterar `nextRecommended` ni las dependencias.
- [ ] `GET /api/increments/<n>` devuelve la batería y su cobertura, y la pestaña «Pruebas» la muestra en solo lectura. Los tests de assets y el canario de secretos siguen en verde sin rutas nuevas.
- [ ] Al archivar, los provisionales de `Cubre` en `pruebas.md` se reescriben a IDs definitivos.
- [ ] El ratchet de código muerto, el de deriva del orquestador y los goldens siguen en verde. Solo cambia el golden del agente sdd-tasks de claude.
