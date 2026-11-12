<!-- Especificación Viva generada a partir de '2026-09-16-inc-13-sdd-commands-axiom-cli-integration' -->

# Especificación de Requerimientos: Integración de Comandos SDD en la CLI axiom (INC-13)

## Propósito

Definir de forma exhaustiva los requerimientos y escenarios de prueba para la exposición nativa de los comandos de ciclo de vida SDD bajo la CLI unificada `axiom` y la actualización de los contratos de invocación en los prompts de agentes.

---

## 1. Capacidad: `axiom-sdd-cli-integration`

Exposición de las operaciones de estado, despacho y presupuesto SDD directamente en el ejecutable `axiom`.

### Requirement: Subcomando axiom sdd status (REQ-13.1)
El binario `axiom` DEBE soportar la invocación `axiom sdd status [change] [flags]` (y su alias `axiom sdd-status`), retornando el informe de estado estructurado de la fase actual con idéntica fidelidad de contrato JSON que el motor v2.

#### Scenario: Consulta de estado mediante axiom sdd status
- **DADO** un repositorio con un cambio SDD activo en `openspec/changes/<change>`
- **CUANDO** el usuario o agente ejecuta `axiom sdd status <change> --json`
- **ENTONCES** la salida estándar emite el JSON del estado del cambio con código de salida `0`
- **Y** los campos `activeChange`, `artifactStatuses` y `nextRecommended` se resuelven correctamente

### Requirement: Subcomando axiom sdd continue (REQ-13.2)
El binario `axiom` DEBE soportar `axiom sdd continue [change]` para ejecutar el cálculo del despachador y emitir la siguiente acción autorizada del ciclo.

#### Scenario: Enrutamiento mediante axiom sdd continue
- **DADO** un cambio con fase de diseño completada y tareas pendientes
- **CUANDO** se ejecuta `axiom sdd continue <change>`
- **ENTONCES** la salida emite la instrucción correspondiente a la siguiente fase autorizada (`sdd-tasks` o `sdd-apply`)

### Requirement: Subcomando axiom sdd attempt tras la retirada de la gobernanza de presupuesto (REQ-13.3)
El binario `axiom` NO DEBE soportar las operaciones `acquire` y `settle` del libro mayor de intentos de ejecución bajo `axiom sdd attempt`, siguiendo la retirada de la gobernanza de *attempts* absorbida de upstream (commit `18fa04fb`). El binario `axiom` DEBE conservar sin cambio de comportamiento la operación `axiom sdd attempt grant`, que registra la autoridad de edición por raíz (`--root`) para el cambio activo.

(Previamente: el binario `axiom` DEBÍA soportar las operaciones `acquire` y `settle` del libro mayor de intentos de ejecución, `axiom sdd attempt acquire` y `axiom sdd attempt settle`.)

#### Scenario: acquire y settle dejan de estar disponibles
- **DADO** un cambio en fase `sdd-apply` tras absorber la tanda F4
- **CUANDO** se invoca `axiom sdd attempt acquire --change <cambio> ...` o `axiom sdd attempt settle --token <token> ...`
- **ENTONCES** el binario rechaza la operación como no reconocida
- **Y** no retorna ningún token de sesión ni registra ningún resultado de presupuesto

#### Scenario: grant sigue emitiendo autoridad de edición sin cambios
- **DADO** un cambio activo con raíces de edición pendientes de autorizar
- **CUANDO** se invoca `axiom sdd attempt grant --root <ruta> --change-instance <token> --request-id <id> --actor <actor> --reason <motivo>`
- **ENTONCES** el sistema registra la autoridad de edición para esa raíz
- **Y** el comportamiento de `grant` no cambia respecto al contrato vigente antes de esta absorción

---

## 2. Capacidad: `orchestrator-prompt-call-alignment`

Instrucciones de los agentes configuradas para utilizar la CLI `axiom`.

### Requirement: Referencia exclusiva a la CLI axiom en prompts (REQ-13.4)
Los prompts de orquestadores, comandos slash y guías de agentes NO DEBEN instruir a ejecutar `gentle-ai`, sino que deben prescribir la sintaxis `axiom sdd ...` (o `axiom sdd-*`).

#### Scenario: Verificación de sintaxis de comandos en sdd-orchestrator-sections.md
- **DADO** la sección `Native SDD Dispatcher Guard` del activo de orquestación
- **CUANDO** se evalúa el comando de inspección prescrito
- **ENTONCES** el comando inicia con `axiom sdd status` (o `axiom sdd-status`)
- **Y** no contiene ninguna mención a `gentle-ai sdd-status`

#### Scenario: Comandos slash de OpenCode apuntando a axiom
- **DADO** el comando `/sdd-status` en OpenCode
- **CUANDO** el agente ejecuta la tarea descrita en el archivo markdown del comando
- **ENTONCES** invoca `axiom sdd status` con su herramienta bash
### Requirement: `blocked(wrong_branch)` en `axiom sdd status` (REQ-25.42)

Cuando el veredicto de rama (REQ-25.18 de `change-branch-lifecycle`) para el cambio es `wrong_branch`, `axiom sdd status` DEBE proyectar el estado de aplicación como bloqueado con el motivo tipado `blocked(wrong_branch)`, el segundo motivo tipado junto a `blocked(edit_authority_missing)`. El motivo DEBE nombrar el repositorio, la rama actual y la rama esperada, e incluir la continuación exacta: la orden `axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane`. En un repositorio con rama de cambio (`monorepo-embedded`) se evalúa en toda fase: en la fase archive la rama esperada es la rama de archivado `<prefijo>/<cambio>-archive` y la continuación exacta es `axiom branch ensure --cwd <ruta> --change <c> --archive`, de modo que estar en la rama del cambio, en la de integración o en una de rol bloquea; en el resto de fases la rama esperada es la del cambio (o una de rol). En un repositorio de código de un `multirepo` solo se evalúa cuando el estado de aplicación evalúa unidades de trabajo que lo tocan (apply); en el repositorio de specs de un `multirepo` exige la rama de integración, también en la fase archive. La evaluación NO consulta si la rama de integración contiene cambios activos (REQ-25.20 de `change-branch-lifecycle`). NO se proyecta para cambios exentos (REQ-25.37 de `sdd-lane-kickoff`). Cuando concurren `wrong_branch` y `edit_authority_missing`, `wrong_branch` DEBE proyectarse primero, porque las raíces autorizadas dependen de la rama correcta (supuesto S-9). `axiom sdd continue` DEBE reflejar el mismo bloqueo sin autorizar ninguna fase siguiente. La evaluación DEBE ser de solo lectura y, si git no está disponible o la ruta no es un repositorio git, NO DEBE bloquear y DEBE emitir un aviso. El motivo NO lleva envoltorio de consentimiento: la continuación es una orden que el orquestador ejecuta, no una concesión.

#### Scenario: Status en la rama de integración de un monorepo
- **DADO** un cambio `x` con instantánea en el `kickoff.yaml` y el repositorio del planning home desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** el estado de aplicación es `blocked` con el motivo `blocked(wrong_branch)`
- **Y** el motivo nombra la rama actual `main`, la esperada `feat/x` y el repositorio
- **Y** incluye la continuación `axiom branch ensure --cwd <ruta absoluta> --change x`

#### Scenario: Status en la rama del cambio no bloquea
- **DADO** el mismo cambio con el repositorio desplegado en `feat/x`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`

#### Scenario: Se evalúa en toda fase en un monorepo
- **DADO** un cambio en fase `spec` con instantánea y el repositorio desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** se proyecta `blocked(wrong_branch)`

#### Scenario: En la fase archive solo la rama de archivado es correcta
- **DADO** un monorepo con el cambio `x` fusionado en la fase archive y el repositorio desplegado, por separado, en `feat/x`, en `main` y en `feat/x-core`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** en los tres casos se proyecta `blocked(wrong_branch)` con la rama esperada `feat/x-archive`
- **Y** la continuación es `axiom branch ensure --cwd <ruta absoluta> --change x --archive`
- **Y** con el repositorio desplegado en `feat/x-archive` no se proyecta `blocked(wrong_branch)`

#### Scenario: En la fase archive del repositorio de specs de un multirepo se exige la integración
- **DADO** un multirepo en la fase archive con el repositorio de specs desplegado en `feat/x-core`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** se proyecta `blocked(wrong_branch)` con la rama esperada `main` y sin continuación de `--archive`

#### Scenario: Los cambios activos en la rama de integración no bloquean
- **DADO** un cambio `y` fusionado y presente en la rama de integración, y un cambio `x` con instantánea y el repositorio desplegado en `feat/x`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`

#### Scenario: Un repositorio de código de un multirepo solo se evalúa en apply
- **DADO** un multirepo, un cambio con instantánea y `repo-backend` desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json` en fase `spec`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`
- **Y** en fase de aplicación con una unidad de trabajo que toca `repo-backend` se proyecta `blocked(wrong_branch)` nombrando `repo-backend`

#### Scenario: Un cambio exento no se bloquea
- **DADO** un cambio con `kickoff.yaml` sin instantánea y el repositorio desplegado en `main`
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`

#### Scenario: `wrong_branch` precede a `edit_authority_missing`
- **DADO** un cambio con instantánea, el repositorio desplegado en `main` y una tarea que toca un repositorio sin autorizar
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** el motivo proyectado es `blocked(wrong_branch)`
- **Y** tras corregir la rama se proyecta `blocked(edit_authority_missing)`

#### Scenario: `axiom sdd continue` refleja el bloqueo
- **DADO** el mismo cambio bloqueado por rama
- **CUANDO** se ejecuta `axiom sdd continue x`
- **ENTONCES** no emite ninguna fase siguiente autorizada
- **Y** reporta `blocked(wrong_branch)` con la misma continuación

#### Scenario: Sin git no se bloquea
- **DADO** un proyecto fuera de un repositorio git
- **CUANDO** se ejecuta `axiom sdd status x --json`
- **ENTONCES** no se proyecta `blocked(wrong_branch)`
- **Y** el status incluye un aviso no bloqueante

#### Scenario: El contrato del status solo crece
- **DADO** el status de un cambio sin bloqueo de rama
- **CUANDO** se compara con el de antes de este incremento
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor

---

### Requirement: Las Raíces Registradas por `ensure` Entran en `allowedEditRoots` (REQ-25.43)

`actionContext.allowedEditRoots` DEBE incluir, además de `workspaceRoot` y las concesiones por cambio, las raíces absolutas de los repositorios y worktrees que `axiom branch ensure` registró para el cambio (en modo `in_place`, la raíz del propio repositorio). Las rutas de las unidades de trabajo de `tasks.md` y `tasks.<rol>.md` DEBEN resolverse respecto del worktree que corresponde al rol (el de la rama de rol para `tasks.<rol>.md` cuando existe, y el del cambio en otro caso) y reducirse a su raíz git. Un cambio cuyas tareas tocan solo sus propios worktrees NO DEBE quedar en `blocked(edit_authority_missing)`. Una raíz no registrada por `ensure` sigue exigiendo la concesión por cambio (`axiom sdd attempt grant`). Sin registro local para el cambio, el comportamiento DEBE ser el anterior a este incremento.

#### Scenario: Las tareas de un cambio en worktree no exigen autorización
- **DADO** un monorepo con el cambio `x` en un worktree registrado y un `tasks.md` cuyas unidades nombran ficheros del repositorio
- **CUANDO** se consulta `axiom sdd status x --json`
- **ENTONCES** `actionContext.allowedEditRoots` incluye la raíz absoluta del worktree
- **Y** `applyState` no es `blocked(edit_authority_missing)`

#### Scenario: La tarea de un rol se resuelve contra el worktree del rol
- **DADO** un cambio con los roles `core` y `qa` y un `tasks.core.md` cuyas unidades nombran `internal/core/a.go`
- **CUANDO** se consulta el status
- **ENTONCES** la ruta se resuelve contra el worktree de `feat/x-core`
- **Y** `allowedEditRoots` incluye la raíz de ese worktree

#### Scenario: Un repositorio no registrado exige concesión
- **DADO** un multirepo y una unidad de trabajo que toca un repositorio de código que `ensure` no registró para el cambio
- **CUANDO** se consulta el status
- **ENTONCES** se proyecta `blocked(edit_authority_missing)` nombrando ese repositorio
- **Y** el repositorio entra en `allowedEditRoots` solo tras `axiom sdd attempt grant`

#### Scenario: Un repositorio de código registrado por `ensure` entra en las raíces
- **DADO** un multirepo con el cambio `x` y `ensure` ejecutado, que registró el worktree de `repo-backend`
- **CUANDO** se consulta el status
- **ENTONCES** `allowedEditRoots` incluye la raíz del worktree de `repo-backend`

#### Scenario: Sin registro el comportamiento es el anterior
- **DADO** un cambio sin registro local
- **CUANDO** se consulta el status
- **ENTONCES** `allowedEditRoots` es `workspaceRoot` más las concesiones por cambio

---

### Requirement: `status` y `continue` Resuelven los Cambios de los Worktrees Registrados (REQ-25.44)

`axiom sdd status` y `axiom sdd continue` DEBEN resolver los cambios activos que viven en los worktrees registrados en el registro local de `axiom branch` (REQ-25.14), de modo que, al ejecutarse desde el checkout principal u otro directorio del proyecto, `axiom sdd status <cambio>` resuelva el cambio en su worktree. En ese caso `workspaceRoot`, `planningHome`, `changeRoot` y `artifactPaths` DEBEN referirse al worktree. Sin nombre de cambio, los candidatos DEBEN incluir los cambios activos de los worktrees registrados con el orden de preferencia vigente. Si un mismo cambio figura en el checkout principal y en un worktree, prevalece el del worktree; si figura en el worktree de la rama del cambio y en el de la rama de archivado, prevalece el de la rama de archivado, porque el cambio ya está fusionado y solo falta archivarlo (REQ-25.45 de `change-branch-lifecycle`). Una entrada del registro cuyo worktree ya no existe DEBE omitirse con un aviso. Con `--cwd` dentro de un worktree, el cambio se resuelve en su propio árbol (REQ-25.28 de `planning-home-resolution`).

#### Scenario: Status desde el checkout principal resuelve el worktree
- **DADO** un proyecto cuyo checkout principal solo contiene `archive/` y un cambio `x` activo en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd status x --json --cwd <checkout principal>`
- **ENTONCES** `changeRoot` es `<worktree>/openspec/changes/x`
- **Y** `workspaceRoot` es la raíz del worktree

#### Scenario: Sin nombre de cambio se consideran los worktrees registrados
- **DADO** un proyecto con un único cambio activo `x`, que vive en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd status --json --cwd <checkout principal>`
- **ENTONCES** el cambio activo resuelto es `x`

#### Scenario: El worktree prevalece sobre el checkout principal
- **DADO** un cambio presente en el checkout principal y en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status corresponde al del worktree

#### Scenario: El worktree de la rama de archivado prevalece sobre el de la rama del cambio
- **DADO** el cambio `x` presente en el worktree de `feat/x` y en el de `feat/x-archive`, ambos registrados
- **CUANDO** se ejecuta `axiom sdd status x --json --cwd <checkout principal>`
- **ENTONCES** `changeRoot` es `<worktree de feat/x-archive>/openspec/changes/x`

#### Scenario: Un worktree desaparecido se omite con aviso
- **DADO** una entrada del registro local cuyo worktree no existe
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status se devuelve sin ese cambio
- **Y** incluye un aviso que nombra la ruta ausente

#### Scenario: `continue` resuelve igual que `status`
- **DADO** el cambio `x` en un worktree registrado
- **CUANDO** se ejecuta `axiom sdd continue x --cwd <checkout principal>`
- **ENTONCES** la siguiente fase autorizada se calcula sobre el cambio del worktree
### Requirement: `axiom sdd status` Proyecta el Carril Efectivo del Proyecto (REQ-27.2)

`axiom sdd status` DEBE incluir, en el JSON v1 y en el v2 y con o sin cambio seleccionado, un campo opcional `workflow` con el carril efectivo (`lane`: `odd`, `sdd` o `ask`) y su origen (`source`), con los tokens estables en inglés `declared`, `default`, `no_workspace` e `invalid`. El origen `declared` indica que `axiom.yaml` declara `workflow.lane`; `default` que `axiom.yaml` es válido y no lo declara (carril `sdd`); `no_workspace` que no hay `axiom.yaml` (repositorio no gobernado por Axiom); e `invalid` que el bloque `workflow` no es válido. Con `no_workspace` o `invalid` el carril proyectado DEBE ser `ask`, nunca `sdd`: la ausencia de configuración o una configuración inválida NO DEBEN activar SDD automático. El campo DEBE estar presente con `changeName: null` cuando no hay cambio seleccionado, con el mismo criterio que `technicalContext`, y NO DEBE alterar ningún otro campo del contrato ni de las rutas proyectadas. Un consumidor anterior que ignore los campos desconocidos DEBE seguir funcionando. El contrato publicado (`sdd-status-contract.md`) DEBE documentar el campo y la regla de fallo cerrado hacia `ask`.
(Previously: el campo se proyectaba con el mismo criterio que `baseline` y `technicalContext`; la baseline ajena se retira)

#### Scenario: Carril declarado con cambio seleccionado

- **DADO** un `axiom.yaml` con `workflow: {lane: odd}` y un cambio SDD activo
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` en el formato v2
- **ENTONCES** el JSON contiene `workflow` con `lane: odd` y `source: declared`

#### Scenario: Carril por defecto sin bloque

- **DADO** un `axiom.yaml` válido sin `workflow:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `workflow` indica `lane: sdd` y `source: default`

#### Scenario: Sin cambio seleccionado el campo sigue presente

- **DADO** un `axiom.yaml` con `workflow: {lane: ask}` y ningún cambio seleccionado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `changeName` es `null`
- **Y** `workflow` indica `lane: ask` y `source: declared`

#### Scenario: Presente en el formato v1 y en el v2

- **DADO** el mismo proyecto con `workflow: {lane: sdd}`
- **CUANDO** se ejecuta `axiom sdd status --json` en el formato v1 y en el v2
- **ENTONCES** ambos contienen `workflow` con los mismos valores de `lane` y `source`

#### Scenario: Sin `axiom.yaml` el carril proyectado es `ask`

- **DADO** un repositorio sin `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `workflow` indica `lane: ask` y `source: no_workspace`

#### Scenario: Bloque inválido proyecta `ask` y su origen

- **DADO** un `axiom.yaml` con `workflow: {lane: agil}`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `workflow` indica `lane: ask` y `source: invalid`
- **Y** el resto del JSON se proyecta como hasta ahora

#### Scenario: El resto del contrato no cambia

- **DADO** un cambio con `technicalContext` y `governance` proyectados
- **CUANDO** se ejecuta `axiom sdd status --json` antes y después de añadir `workflow` a `axiom.yaml`
- **ENTONCES** los demás campos son idénticos

#### Scenario: El contrato publicado documenta el campo

- **DADO** `sdd-status-contract.md`
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** documenta `workflow`, sus tres valores de `lane`, los cuatro orígenes y que `no_workspace` e `invalid` proyectan `ask`
### Requirement: Subcomando `axiom sdd publish` en el Contrato del Binario (REQ-27.2b)

El binario `axiom` DEBE soportar la invocación `axiom sdd publish --change <nombre> [--cwd <ruta>] [--json]` como subcomando de `axiom sdd`, con el comportamiento de la capacidad `sdd-docs-publication`. La ayuda y la lista de subcomandos de `axiom sdd` DEBEN incluirlo.

#### Scenario: La ayuda de `sdd` lista `publish`

- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd --help`
- **ENTONCES** la salida lista `publish` junto a los subcomandos ya existentes, que siguen listados

#### Scenario: `publish` sin cambio se rechaza con ayuda

- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd publish` sin `--change`
- **ENTONCES** termina con código de salida distinto de cero y un mensaje que nombra `--change`
### Requirement: Preparación del Cambio con Bug sin Delta (con Requisito Existente o sin Impacto en la Spec), Aceptando el Formato Actual (REQ-29.15)

`axiom sdd status` DEBE considerar cumplido el artefacto de specs del cambio cuando existe al menos una delta de spec, o cuando el cambio está declarado como bug sin delta. Esta ruta está disponible en todo proyecto, sea o no ISO. Un bug sin delta se declara en su `proposal.md` con una sección fija `## Informe de bug`, que es la propia declaración (no hay front matter), y admite dos variantes:

| Variante | Qué declara | Qué más lleva |
|---|---|---|
| Bug con requisito existente | Restaura un requisito que ya existe en la baseline. | `Cubre:` en las tareas con el requisito existente; en ISO se añaden las comprobaciones de REQ-29.5 de `sdd-iso-evidence`. |
| Bug sin impacto en la spec | No toca ningún requisito. | La sección `## Impacto en la spec` con la marca de abajo; sin delta y sin `Cubre:`; en ISO rige REQ-29.19 de `sdd-iso-evidence`. |

**Formato de `## Informe de bug`** (contrato fijo): un encabezado de nivel 2 con ese texto exacto y, en su contenido, una línea `Observado:` (comportamiento observado) y una línea `Esperado:` (comportamiento esperado), ambas obligatorias y con texto no vacío tras los dos puntos, y una línea `Referencia:` opcional (referencia externa). Cada línea empieza por su etiqueta y lleva su valor en la misma línea (supuesto visible). Falta de `Observado:` o de `Esperado:`, o valor vacío, deja el informe incompleto: no cuenta como declaración de bug.

**Formato de la marca de impacto** (contrato fijo, que otros carriles reutilizan tal cual): un encabezado de nivel 2 `## Impacto en la spec` cuyo contenido es la línea literal `Sin impacto en la spec` y una línea `Motivo: <una línea>` con texto no vacío. La línea `Sin impacto en la spec` se compara de forma exacta, sensible a mayúsculas y sin más texto en la línea, ignorando solo el espacio en blanco inicial y final (supuesto visible). La marca solo es válida junto a un `## Informe de bug` completo.

**Reglas de evaluación**, por orden de precedencia:

1. **Con delta.** Un cambio con delta se evalúa como cambio normal aunque declare ser un bug o lleve la sección `## Impacto en la spec`: la sección se ignora, sin aviso (supuesto visible).
2. **Sin delta y con la sección `## Impacto en la spec`.** Se evalúa como bug sin impacto en la spec: el artefacto de specs se cumple solo si el `## Informe de bug` está completo y la sección contiene la línea literal y un `Motivo:` no vacío. Si falta el informe, o está incompleto, o falta la línea literal, o falta el motivo o está vacío, el artefacto NO se cumple, el estado proyecta la fase `spec` pendiente y el motivo nombra qué falta (supuesto visible: una declaración incompleta nunca cuenta como declaración).
3. **Sin delta y sin esa sección.** Si hay un `## Informe de bug` completo, se evalúa como bug con requisito existente y el artefacto se cumple; si no hay informe, o está incompleto, el artefacto NO se cumple y el estado proyecta la fase `spec` pendiente, nombrando la línea que falta cuando el informe existe.

Un bug sin delta llega a la fase `archive` sin que la ausencia de delta la bloquee, y su archivado no integra ningún requisito en la baseline porque no hay nada que componer. Un bug cuyo requisito estaba mal especificado o faltaba NO es un bug sin delta: se tramita como cambio normal, con delta. Un cambio en el formato actual (`proposal.md` y delta) DEBE producir el mismo estado que antes de este cambio, sin migración. La proyección es solo aditiva: ninguna clave existente del status v2 cambia de nombre, tipo ni valor.

#### Scenario: El formato actual produce el mismo estado

- **DADO** un cambio con `proposal.md` y una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido y el estado coincide con el anterior a este cambio

#### Scenario: Un bug con requisito existente queda preparado fuera de ISO

- **DADO** un cambio no ISO con un `## Informe de bug` completo (`Observado:` y `Esperado:`) en su `proposal.md`, sin delta y sin la sección `## Impacto en la spec`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido sin exigir delta

#### Scenario: La referencia del informe es opcional

- **DADO** un `## Informe de bug` con `Observado:` y `Esperado:` y sin `Referencia:`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el informe se considera completo

#### Scenario: Un informe sin `Esperado:` o con valor vacío está incompleto

- **DADO** un cambio sin delta con `## Informe de bug` que contiene `Observado:` con texto y `Esperado:` vacío (o sin la línea `Esperado:`)
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase `spec` pendiente
- **Y** el motivo nombra la línea `Esperado:`

#### Scenario: Un cambio sin delta y sin declaración de bug no está preparado

- **DADO** un cambio con `proposal.md`, sin delta y sin `## Informe de bug`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase pendiente

#### Scenario: Un cambio con delta que declara ser un bug se trata como cambio normal

- **DADO** un cambio con `## Informe de bug` que además contiene una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se evalúa como un cambio con delta
- **Y** la cobertura `Cubre:` de ISO, si aplica, se calcula sobre los requisitos de esa delta

#### Scenario: Un bug sin impacto en la spec queda preparado en cualquier proyecto

- **DADO** un cambio, ISO o no, con `## Informe de bug` completo y una sección `## Impacto en la spec` con las líneas `Sin impacto en la spec` y `Motivo: el defecto estaba en el script de despliegue`, sin delta y sin `Cubre:`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs se considera cumplido sin exigir delta

#### Scenario: La marca sin `Motivo:` no prepara el cambio

- **DADO** un cambio sin delta con `## Informe de bug` completo y `## Impacto en la spec` que contiene solo `Sin impacto en la spec`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase `spec` pendiente
- **Y** el motivo nombra la línea `Motivo:` que falta

#### Scenario: La marca con motivo vacío no prepara el cambio

- **DADO** la misma declaración con la línea `Motivo:` sin texto
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido
- **Y** el motivo nombra la línea `Motivo:` vacía

#### Scenario: La sección sin la línea literal no prepara el cambio

- **DADO** un cambio sin delta con `## Informe de bug` completo y `## Impacto en la spec` con `Impacto: ninguno` y `Motivo: ninguno`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido
- **Y** el motivo nombra la línea literal `Sin impacto en la spec` que falta

#### Scenario: El texto de la marca debe ser exactamente el literal

- **DADO** una sección `## Impacto en la spec` con `Sin impacto en la spec.` (con punto final) o con `sin impacto en la spec`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la línea no se reconoce como la marca y el artefacto de specs no se considera cumplido

#### Scenario: `## Impacto en la spec` sin `## Informe de bug` no prepara el cambio

- **DADO** un cambio sin delta con `## Impacto en la spec` válida y sin `## Informe de bug`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el artefacto de specs no se considera cumplido y se proyecta la fase `spec` pendiente
- **Y** el motivo nombra la sección `## Informe de bug` que falta

#### Scenario: La marca junto a una delta se ignora y se evalúa como cambio con delta

- **DADO** un cambio con `## Informe de bug`, `## Impacto en la spec` válida y una delta de spec
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se evalúa como un cambio con delta y el artefacto de specs se considera cumplido por la delta
- **Y** la sección `## Impacto en la spec` no genera ningún motivo

#### Scenario: Un bug sin delta llega al archivado sin integrar nada

- **DADO** un bug sin delta, con requisito existente o sin impacto en la spec, con sus fases completadas y el resto de precondiciones de archivado satisfechas
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json` en la fase `archive`
- **ENTONCES** la ausencia de delta no bloquea el archivado
- **Y** el archivado no modifica ningún documento de la baseline

#### Scenario: El informe del bug va en `proposal.md`

- **DADO** un bug sin delta
- **CUANDO** se inspeccionan los artefactos del cambio
- **ENTONCES** `proposal.md` contiene la sección `## Informe de bug` con el comportamiento observado y el esperado
- **Y** no existe ningún otro artefacto de informe de bug ni front matter que lo declare

#### Scenario: El contrato del status solo crece

- **DADO** el status de un cambio con delta
- **CUANDO** se compara con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor

---

### Requirement: Motivos de Bloqueo Tipados de ISO, Cobertura, Escenarios y Matriz en `status` (REQ-29.16)

`axiom sdd status` DEBE proyectar, con un motivo tipado que nombre el elemento afectado y la continuación, los bloqueos de un cambio ISO (sellado con `iso`, REQ-29.13 de `sdd-lane-kickoff`; las reglas se leen de su instantánea y no de `axiom.yaml` en vivo) definidos en `sdd-iso-evidence` y `sdd-archive-lifecycle`: identificador `REQ-<n>.<m>` ausente, ambiguo o repetido en el nombre de los requisitos de la delta (REQ-29.3), cobertura `Cubre:` y `Cubre:` a un requisito inexistente (REQ-29.4), requisito de un bug con requisito existente inexistente, sin citar o sin escenarios (REQ-29.5), `Cubre:` en un bug sin impacto en la spec (REQ-29.19), requisito ADDED o MODIFIED sin escenarios (REQ-29.10), y matriz de verify ausente, inválida o con filas fallidas en la fase `archive` (REQ-29.6 a REQ-29.8 y REQ-29.11). Fuera de un cambio ISO, ninguna de esas reglas DEBE proyectar ni bloqueo ni aviso. `axiom sdd continue` DEBE reflejar cualquier bloqueo sin autorizar la fase siguiente. Los nuevos motivos son solo aditivos: ninguna clave existente del status v2 cambia, y el orden de precedencia de los motivos existentes (por ejemplo, `wrong_branch` antes de `edit_authority_missing`) se conserva. Cuando concurren varios bloqueos, el status DEBE proyectar todos con su continuación, no solo el primero.

#### Scenario: Un bloqueo ISO nombra el elemento y la continuación

- **DADO** un cambio ISO y un requisito de la delta sin cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de cobertura nombrando el requisito
- **Y** incluye la continuación (añadir un `Cubre:` en alguna tarea)

#### Scenario: Un requisito sin escenarios se proyecta antes del archivado

- **DADO** un cambio ISO y una delta con un ADDED sin escenarios
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de escenarios nombrando el dominio, la sección y el requisito

#### Scenario: Una matriz inválida bloquea la dependencia `archive`

- **DADO** un cambio ISO en la fase `archive` con una matriz con un escenario sin fila
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** la dependencia `archive` aparece bloqueada con el motivo tipado de matriz que nombra el escenario
- **Y** incluye la continuación (completar la matriz del `verify-report.md`)

#### Scenario: Fuera de ISO la cobertura y la matriz no proyectan nada

- **DADO** un cambio no ISO, requisitos sin `Cubre:` y un `verify-report.md` libre
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** no se proyecta ni bloqueo ni aviso de cobertura o de matriz

#### Scenario: `axiom sdd continue` refleja el bloqueo ISO

- **DADO** un cambio ISO bloqueado por cobertura
- **CUANDO** se ejecuta `axiom sdd continue <cambio>`
- **ENTONCES** no emite ninguna fase siguiente autorizada
- **Y** reporta el mismo motivo tipado

#### Scenario: Varios bloqueos concurrentes se proyectan todos

- **DADO** un cambio ISO con un requisito ADDED sin identificador `REQ-<n>.<m>` en su nombre y otro requisito sin cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyectan los motivos de identificador y de cobertura, cada uno con su continuación

#### Scenario: Un `Cubre:` junto a la marca de bug sin impacto se proyecta como motivo tipado

- **DADO** un cambio ISO declarado como bug sin impacto en la spec y una tarea con `Cubre: REQ-13.1`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** se proyecta el motivo tipado de declaración contradictoria, con el fichero de tareas, el requisito y la continuación

#### Scenario: La precedencia de los motivos existentes se conserva

- **DADO** un cambio ISO bloqueado por `wrong_branch` y por cobertura
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** `wrong_branch` se proyecta con la precedencia que ya tenía
- **Y** el motivo de cobertura se proyecta además

#### Scenario: El contrato del status solo crece

- **DADO** el status de un cambio no ISO
- **CUANDO** se compara con el de antes de este cambio
- **ENTONCES** ninguna clave existente cambia de nombre, tipo ni valor
