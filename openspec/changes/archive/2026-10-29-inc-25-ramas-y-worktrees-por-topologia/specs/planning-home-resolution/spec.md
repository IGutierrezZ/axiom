# Delta para Resolución del Planning Home desde Cualquier Directorio (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** delta sobre la capacidad viva `planning-home-resolution`
> **Idioma:** Español (castellano peninsular)

Cambios de este delta:

- REQ-24.3, REQ-24.5 y REQ-24.7: se retira `monorepo-decoupled` (REQ-25.25 de `workspace-topology`); el enraizado por repositorio de especificaciones rige solo para `multirepo`.
- REQ-24.9: las raíces de los worktrees y repositorios que registra `axiom branch ensure` se proyectan en `allowedEditRoots` (REQ-25.43 de `axiom-sdd-cli-integration`).
- REQ-25.28 (nuevo): resolución del planning home cuando el directorio de partida es un worktree enlazado, también fuera de la carpeta maestra.

> Nota para el archivado: la tabla «Términos» (fila «Repositorio de especificaciones»), la tabla «Requerimientos de un vistazo» (fila REQ-24.3) y el encabezado del documento no son requerimientos y no se actualizan por este delta: DEBEN editarse a mano al archivar para sustituir «`monorepo-decoupled` y `multirepo`» por «`multirepo`».

## MODIFIED Requirements

### Requirement: Enraizado por Topología en Multirepo (REQ-24.3)

Una vez hallada la carpeta maestra (REQ-24.5) y cargado `axiom.yaml`, `workspaceRoot` y el planning home DEBEN derivarse según la topología:

- `monorepo-embedded`: `workspaceRoot` es la carpeta maestra y el planning home es `<workspaceRoot>/openspec` (REQ-24.4).
- `multirepo`: el repositorio de especificaciones es `<maestra>/<specs_repository>`. Si el último segmento de `specs_repository` es `openspec`, el planning home es ese directorio y `workspaceRoot` su padre. En otro caso, el planning home es `<repositorio de especificaciones>/openspec` y `workspaceRoot` es el repositorio de especificaciones.

En todos los casos `openspec/` DEBE permanecer literal bajo `workspaceRoot`, de modo que las rutas `openspec/changes/...` siguen siendo válidas respecto a `workspaceRoot`.
(Previously: el requerimiento se titulaba «Enraizado por Topología en Monorepo Desacoplado y Multirepo» y el enraizado por repositorio de especificaciones se aplicaba también a `monorepo-decoupled`, topología retirada)

#### Scenario: Multirepo canónico
- **DADO** el workspace canónico
- **CUANDO** se resuelve desde `<maestra>`, desde `<maestra>/repo-specs` o desde `<maestra>/repo-backend/src`
- **ENTONCES** el planning home es `<maestra>/repo-specs/openspec` y `workspaceRoot` es `<maestra>/repo-specs` en los tres casos

#### Scenario: Multirepo con `axiom.yaml` en la carpeta maestra
- **DADO** una carpeta maestra con `axiom.yaml` (`multirepo`, `specs_repository: especificacion`) y los directorios `app-monorepo/` y `especificacion/`
- **CUANDO** se resuelve desde la carpeta maestra
- **ENTONCES** `workspaceRoot` es `<maestra>/especificacion` y el planning home es `<maestra>/especificacion/openspec`

#### Scenario: Perfil knowledge con `specs_repository` terminado en `openspec`
- **DADO** un directorio `X` con `axiom.yaml` (`multirepo`, `specs_repository: openspec`) y `X/openspec/`
- **CUANDO** se resuelve desde `X`
- **ENTONCES** el planning home es `X/openspec` y `workspaceRoot` es `X`

#### Scenario: `specs_repository` de varios segmentos terminado en `openspec`
- **DADO** una carpeta maestra con `axiom.yaml` (`multirepo`, `specs_repository: docs/openspec`)
- **CUANDO** se resuelve desde la carpeta maestra
- **ENTONCES** el planning home es `<maestra>/docs/openspec` y `workspaceRoot` es `<maestra>/docs`

#### Scenario: La topología retirada no se resuelve
- **DADO** una carpeta maestra con `axiom.yaml` (`monorepo-decoupled`, `specs_repository: especificacion`)
- **CUANDO** se resuelve desde la carpeta maestra con un puntero
- **ENTONCES** la resolución se rechaza conforme a REQ-25.25 de `workspace-topology`, nombrando `multirepo`

---

### Requirement: Deducción de la Carpeta Maestra desde el Repositorio de Especificaciones (REQ-24.5)

Cuando el marcador hallado es un `axiom.yaml` sin puntero en el directorio `D`, la carpeta maestra DEBE determinarse según la topología. En `monorepo-embedded` es siempre `D`: la deducción por sufijo no se aplica, porque en un proyecto embebido cuyo directorio se llame igual que `specs_repository` (por ejemplo `W/specs` con `specs_repository: specs`) movería `workspaceRoot` al directorio padre y rompería la invariancia exacta de REQ-24.4. En `multirepo` se determina por este orden: (1) si `specs_repository` es vacío o `.`, `D`; (2) si `D` contiene un directorio con la ruta `specs_repository`, `D`; (3) si la ruta de `D` termina con los segmentos de `specs_repository`, `D` sin ese sufijo; (4) en otro caso, `D`. La comparación de segmentos DEBE tratar `/` y `\` como equivalentes. Cuando el marcador es un puntero, la carpeta maestra es el directorio que lo contiene. El campo `workspace.root` NO DEBE intervenir en la resolución.
(Previously: la deducción por orden se aplicaba a `monorepo-decoupled` y `multirepo`)

#### Scenario: Desde el repositorio de especificaciones se deduce la carpeta maestra
- **DADO** el workspace canónico
- **CUANDO** se resuelve desde `<maestra>/repo-specs`
- **ENTONCES** la carpeta maestra es `<maestra>`
- **Y** el planning home y `workspaceRoot` coinciden con los obtenidos desde `<maestra>`

#### Scenario: `specs_repository` de varios segmentos
- **DADO** un `axiom.yaml` `multirepo` en `<maestra>/repos/specs` con `specs_repository: repos/specs`
- **CUANDO** se resuelve desde `<maestra>/repos/specs`
- **ENTONCES** la carpeta maestra es `<maestra>`

#### Scenario: En `monorepo-embedded` no se aplica la deducción por sufijo
- **DADO** un proyecto `monorepo-embedded` cuyo directorio es `W/specs`, con `axiom.yaml` (`specs_repository: specs`) y `W/specs/openspec/`
- **CUANDO** se resuelve desde `W/specs`
- **ENTONCES** la carpeta maestra y `workspaceRoot` son `W/specs` y el planning home es `W/specs/openspec`
- **Y** la carpeta maestra no es `W`

#### Scenario: El directorio que contiene `specs_repository` es la carpeta maestra
- **DADO** un directorio `W/openspec` con `axiom.yaml` (`multirepo`, `specs_repository: openspec`) que contiene a su vez `W/openspec/openspec/`
- **CUANDO** se resuelve desde `W/openspec`
- **ENTONCES** la carpeta maestra es `W/openspec`
- **Y** el planning home es `W/openspec/openspec`

#### Scenario: Sin coincidencia de sufijo la carpeta maestra es el directorio del `axiom.yaml`
- **DADO** un `axiom.yaml` en `D` con `specs_repository: especificacion` y un directorio `D/especificacion`
- **CUANDO** se resuelve desde `D`
- **ENTONCES** la carpeta maestra es `D`

#### Scenario: `workspace.root` no altera la resolución
- **DADO** un proyecto `monorepo-embedded` cuyo `axiom.yaml` declara `workspace.root: ".."`
- **CUANDO** se resuelve desde su raíz
- **ENTONCES** `workspaceRoot` y el planning home son los mismos que sin ese campo

---

### Requirement: Configuración No Resoluble o No Cargable (REQ-24.7)

El primitivo DEBE distinguir cuatro casos:

1. Un puntero cuyo destino no existe, o que no define `config` ni `specs`, DEBE rechazarse nombrando el puntero, el destino y la acción para corregirlo o retirarlo.
2. Un puntero cuyo destino existe pero cuyo `axiom.yaml` no puede leerse como YAML o no supera la validación de la configuración base (nombre, topología y roles) DEBE rechazarse nombrando la causa y `axiom workspace validate` como continuación: la configuración base determina el repositorio de especificaciones, de modo que el planning home es desconocido y no hay resolución ni status que devolver. Un destino legible cuya configuración base es válida y cuyo único defecto es un bloque `baseline:` o `technical_context:` inválido NO DEBE rechazarse: el planning home es determinable y se aplica REQ-24.18.
3. Un `axiom.yaml` directo (sin puntero) que no puede cargarse NO DEBE rechazarse: el primitivo DEBE conservar la resolución embebida (`workspaceRoot` es el directorio que lo contiene y el planning home es `<raíz>/openspec`) y comunicar un aviso no bloqueante que nombre la causa y `axiom workspace validate`. Que esa configuración impida declarar una baseline (fichero ilegible o bloque inválido) deja el status operativo y bloquea solo el archivado (REQ-24.18).
4. En `multirepo`, una raíz del repositorio de especificaciones (`workspaceRoot`) inexistente DEBE rechazarse nombrando la ruta esperada y la acción para crearla o corregir `specs_repository`. Esa raíz es `<maestra>/<specs_repository>` o, si el último segmento de `specs_repository` es `openspec`, su directorio padre. El planning home (`openspec/`) NO tiene que existir para resolver: el primitivo no lo exige ni lo crea (REQ-24.1).
(Previously: el caso 4 se aplicaba a `monorepo-decoupled` y `multirepo`)

#### Scenario: Puntero sin `config` ni `specs`
- **DADO** un `.axiom-workspace` vacío
- **CUANDO** se resuelve desde su carpeta
- **ENTONCES** se rechaza indicando que el puntero debe definir `config` o `specs`

#### Scenario: Destino del puntero con configuración inválida
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con `workspace.topology: "desconocida"`
- **CUANDO** se resuelve desde la carpeta del puntero
- **ENTONCES** se rechaza nombrando la topología inválida y `axiom workspace validate`

#### Scenario: Puntero hacia un `axiom.yaml` que no es YAML válido
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con error de sintaxis YAML
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta del puntero
- **ENTONCES** se rechaza con código de salida distinto de cero nombrando la causa y `axiom workspace validate`, porque el planning home es desconocido
- **Y** no se devuelve ningún status y no se escribe nada

#### Scenario: Puntero hacia una configuración legible con un bloque `baseline:` inválido
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con configuración base válida y con `baseline.documents[0].requirement.identity: nombre`
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta del puntero
- **ENTONCES** la resolución se completa sin rechazo y el status devuelve el planning home que determina el puntero
- **Y** el status señala la causa y marca el archivado como bloqueado (REQ-24.18)

#### Scenario: `axiom.yaml` directo no cargable conserva la resolución embebida
- **DADO** un proyecto con `openspec/` y un `axiom.yaml` sin la sección `roles`
- **CUANDO** se ejecuta `axiom sdd status --json` desde su raíz
- **ENTONCES** `workspaceRoot` es la raíz del proyecto y `planningHome.path` es `<raíz>/openspec`
- **Y** el status incluye un aviso no bloqueante que nombra la causa y `axiom workspace validate`

#### Scenario: Repositorio de especificaciones declarado e inexistente
- **DADO** una carpeta maestra con `axiom.yaml` (`multirepo`, `specs_repository: repo-specs`) y sin el directorio `repo-specs`
- **CUANDO** se resuelve desde la carpeta maestra
- **ENTONCES** se rechaza nombrando `<maestra>/repo-specs` y la acción para crearlo o corregir `specs_repository`

#### Scenario: `specs_repository` terminado en `openspec` con el planning home aún sin crear
- **DADO** una carpeta maestra `M` con `axiom.yaml` (`multirepo`, `specs_repository: openspec`) y sin el directorio `M/openspec`
- **CUANDO** se resuelve desde `M`
- **ENTONCES** no se rechaza: `workspaceRoot` es `M` y el planning home es `M/openspec`
- **Y** `M/openspec` no se crea

---

### Requirement: Semántica de `workspaceRoot`, `planningHome` y Autoridad de Edición en el Status (REQ-24.9)

El status v2 DEBE reportar `actionContext.workspaceRoot` igual al `workspaceRoot` resuelto, `planningHome.path` igual a la ruta absoluta del directorio `openspec` resuelto, `planningHome.mode` igual a `repo-local`, y `changeRoot` y `artifactPaths` bajo `planningHome.path`. `actionContext.allowedEditRoots` DEBE partir de `workspaceRoot` y DEBE incluir además las raíces de los worktrees y repositorios que `axiom branch ensure` registró para el cambio (capacidad `axiom-sdd-cli-integration`, REQ-25.43); los repositorios de código hermanos que no estén registrados por `ensure` NO DEBEN autorizarse implícitamente y siguen el flujo de autorización por cambio existente (`axiom sdd-attempt grant`). Las autorizaciones por cambio, el registro de ejecución y el rechazo de cambios archivados DEBEN operar respecto a `workspaceRoot` y al repositorio git que lo contiene.
(Previously: `allowedEditRoots` partía solo de `workspaceRoot` y las concesiones por cambio, sin raíces registradas por `ensure`)

#### Scenario: Valores del status en el workspace canónico
- **DADO** el workspace canónico con el cambio `inc-99-ejemplo`
- **CUANDO** se ejecuta `axiom sdd status inc-99-ejemplo --json` desde `<maestra>`
- **ENTONCES** `planningHome.path` es `<maestra>/repo-specs/openspec`, `planningHome.mode` es `repo-local` y `changeRoot` es `<maestra>/repo-specs/openspec/changes/inc-99-ejemplo`
- **Y** `actionContext.allowedEditRoots` es `["<maestra>/repo-specs"]`

#### Scenario: Un repositorio de código hermano no registrado exige autorización por cambio
- **DADO** un `tasks.md` cuya unidad de trabajo nombra un fichero de `<maestra>/repo-backend` que `ensure` no registró para el cambio
- **CUANDO** se consulta el status del cambio
- **ENTONCES** `applyState` es `blocked` con un motivo `edit_authority_missing` que nombra `<maestra>/repo-backend`
- **Y** el repositorio no figura en `allowedEditRoots` hasta que se ejecuta `axiom sdd-attempt grant`

#### Scenario: Un repositorio registrado por `ensure` figura en `allowedEditRoots`
- **DADO** el workspace canónico, el cambio `inc-99-ejemplo` y `axiom branch ensure` ejecutado para él, que registró el worktree de `repo-backend`
- **CUANDO** se consulta el status del cambio
- **ENTONCES** `actionContext.allowedEditRoots` incluye `<maestra>/repo-specs` y la raíz absoluta del worktree de `repo-backend`

#### Scenario: Los cambios archivados siguen sellados
- **DADO** un cambio en `<maestra>/repo-specs/openspec/changes/archive/2026-10-04-inc-98-previo`
- **CUANDO** se ejecuta `axiom sdd gate record --cwd <maestra> --change inc-98-previo --gate spec --decision approved`
- **ENTONCES** la operación se rechaza por tratarse de un cambio archivado

## RENAMED Requirements

### Requirement: Enraizado por Topología en Monorepo Desacoplado y Multirepo (REQ-24.3) → Enraizado por Topología en Multirepo (REQ-24.3)

(Reason: el título nombraba la topología retirada `monorepo-decoupled`; el requerimiento conserva su identidad REQ-24.3 y su sustancia para `multirepo`, y su texto se modifica en la sección MODIFIED de este delta bajo el nombre nuevo.)
(Migration: las referencias por identidad `REQ-24.3` (incluidas las de `workspace-topology`, REQ-2.4, y `sdd-archive-lifecycle`) siguen siendo válidas; solo cambia el título. La fila REQ-24.3 de la tabla «Requerimientos de un vistazo» debe actualizarse a mano al archivar.)

## ADDED Requirements

### Requirement: Resolución desde un Worktree Enlazado, también Fuera de la Carpeta Maestra (REQ-25.28)

Cuando el directorio de partida está dentro de un worktree enlazado de git (por ejemplo, el creado por `axiom branch ensure` en la raíz de worktrees, fuera de la carpeta maestra), el primitivo DEBE resolver así, sin escribir nada (REQ-24.1):

1. Si el worktree contiene su propio marcador (`.axiom-workspace`, `axiom.yaml` o `openspec`) en el directorio de partida o en un ancestro **dentro del propio worktree**, se aplica REQ-24.2 sin cambios: gana el marcador más cercano. En `monorepo-embedded` esto hace que `workspaceRoot` y el planning home sean los del propio worktree, donde viven los artefactos del cambio de su rama.
2. Si la ascensión no halla ningún marcador y el directorio de partida está dentro de un worktree enlazado, el primitivo DEBE resolver como lo haría desde el checkout principal del mismo clon (el worktree principal que lista git), de modo que un worktree de un repositorio de código de un `multirepo` resuelva el mismo planning home que su checkout principal (supuesto S-11). Si tampoco desde el checkout principal se halla marcador, se mantiene la resolución por `directorio de partida` de REQ-24.2.
3. La ubicación del worktree fuera de la carpeta maestra NO DEBE usarse para deducir la carpeta maestra.

`axiom workspace validate` ejecutado desde un worktree enlazado DEBE evaluar las rutas de los repositorios de los roles respecto de la carpeta maestra resuelta conforme a esta regla y NO DEBE reportar como inexistentes los repositorios solo porque el worktree esté fuera de la carpeta maestra. Todas las órdenes posteriores a `ensure` reciben `--cwd` con una ruta absoluta del JSON de `ensure` (REQ-25.2), y las rutas `openspec/...` se resuelven contra `actionContext.workspaceRoot` (REQ-24.11).

#### Scenario: Un worktree de un monorepo embebido resuelve su propio planning home
- **DADO** un monorepo `monorepo-embedded` con un worktree de la rama `feat/x` en `<raíz de worktrees>/x/<repo>`, con su `axiom.yaml` y su `openspec/`
- **CUANDO** se resuelve con `--cwd <raíz de worktrees>/x/<repo>/internal`
- **ENTONCES** `workspaceRoot` es `<raíz de worktrees>/x/<repo>` y el planning home es `<raíz de worktrees>/x/<repo>/openspec`
- **Y** no es el del checkout principal

#### Scenario: Un worktree de un repositorio de código de un multirepo resuelve el planning home de la carpeta maestra
- **DADO** el workspace canónico y un worktree de `repo-backend` en `<raíz de worktrees>/x/repo-backend`, fuera de `<maestra>`, sin marcador propio
- **CUANDO** se ejecuta `axiom sdd status --json --cwd <raíz de worktrees>/x/repo-backend`
- **ENTONCES** `planningHome.path` es `<maestra>/repo-specs/openspec`, igual que desde `<maestra>/repo-backend`
- **Y** no se ha escrito ningún fichero

#### Scenario: Sin marcador ni siquiera en el checkout principal
- **DADO** un worktree enlazado cuyo checkout principal tampoco tiene marcadores ascendentes
- **CUANDO** se resuelve desde el worktree
- **ENTONCES** `workspaceRoot` es el directorio de partida y el origen es `directorio de partida`

#### Scenario: La validación desde un worktree externo no da falsos errores
- **DADO** un multirepo conforme y un worktree de `repo-backend` fuera de la carpeta maestra
- **CUANDO** se ejecuta `axiom workspace validate --path <ruta del worktree>`
- **ENTONCES** la salida reporta `COMPLIANT` conforme a la carpeta maestra resuelta
- **Y** no reporta como inexistente ningún repositorio de los roles por estar el worktree fuera de la carpeta maestra

#### Scenario: La ubicación del worktree no mueve la carpeta maestra
- **DADO** un worktree enlazado en un directorio cuyo nombre coincide con los segmentos de `specs_repository`
- **CUANDO** se resuelve desde él
- **ENTONCES** la carpeta maestra no se deduce de esa ubicación
