# Delta para planning-home-resolution

> Parte del estado vigente: el resto de requisitos del primitivo (REQ-24.1 a REQ-24.6 y REQ-24.8 a REQ-24.12, REQ-25.28) NO cambia. Esta delta solo sustituye la referencia a un bloque `baseline:` inválido de REQ-24.7 por la regla de un `baseline:` residual (REQ-30.21 de `workspace-topology`) y de un `technical_context:` inválido (REQ-30.30 de `project-technical-context`).
> Nota de promoción: la cabecera («Incremento: `inc-24-spec-baseline-profile`») y la fila «Planning home» de la tabla de términos del documento vigente mencionan la baseline por defecto; son texto no normativo y se actualizan al archivar para decir que el planning home contiene `changes/` y `specs/`.

## MODIFIED Requirements

### Requirement: Configuración No Resoluble o No Cargable (REQ-24.7)

El primitivo DEBE distinguir cuatro casos:

1. Un puntero cuyo destino no existe, o que no define `config` ni `specs`, DEBE rechazarse nombrando el puntero, el destino y la acción para corregirlo o retirarlo.
2. Un puntero cuyo destino existe pero cuyo `axiom.yaml` no puede leerse como YAML o no supera la validación de la configuración base (nombre, topología y roles) DEBE rechazarse nombrando la causa y `axiom workspace validate` como continuación: la configuración base determina el repositorio de especificaciones, de modo que el planning home es desconocido y no hay resolución ni status que devolver. Un destino legible cuya configuración base es válida y cuyo único defecto es un bloque `baseline:` residual o un bloque `technical_context:` inválido NO DEBE rechazarse: el planning home es determinable y se aplican REQ-30.21 de `workspace-topology` y REQ-30.30 de `project-technical-context`.
3. Un `axiom.yaml` directo (sin puntero) que no puede cargarse NO DEBE rechazarse: el primitivo DEBE conservar la resolución embebida (`workspaceRoot` es el directorio que lo contiene y el planning home es `<raíz>/openspec`) y comunicar un aviso no bloqueante que nombre la causa y `axiom workspace validate`. Que esa configuración impida leer el contexto técnico (fichero ilegible o bloque inválido) deja el status operativo y bloquea solo el archivado (REQ-30.30 de `project-technical-context`).
4. En `multirepo`, una raíz del repositorio de especificaciones (`workspaceRoot`) inexistente DEBE rechazarse nombrando la ruta esperada y la acción para crearla o corregir `specs_repository`. Esa raíz es `<maestra>/<specs_repository>` o, si el último segmento de `specs_repository` es `openspec`, su directorio padre. El planning home (`openspec/`) NO tiene que existir para resolver: el primitivo no lo exige ni lo crea (REQ-24.1).
(Previously: los casos 2 y 3 hablaban de un bloque `baseline:` inválido y de impedir declarar una baseline, y remitían a REQ-24.18; la baseline ajena se retira)

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

#### Scenario: Puntero hacia una configuración legible con un bloque `baseline:` residual
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con configuración base válida y con un bloque `baseline:`
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta del puntero
- **ENTONCES** la resolución se completa sin rechazo y el status devuelve el planning home que determina el puntero
- **Y** el status señala la causa tipada de REQ-30.21 y marca el archivado como bloqueado

#### Scenario: Puntero hacia una configuración legible con un `technical_context:` inválido
- **DADO** un `.axiom-workspace` cuyo destino es un `axiom.yaml` con configuración base válida y con `technical_context.maintained.paths: []`
- **CUANDO** se ejecuta `axiom sdd status --json` desde la carpeta del puntero
- **ENTONCES** la resolución se completa sin rechazo y el status devuelve el planning home que determina el puntero
- **Y** el status señala la causa y marca el archivado como bloqueado (REQ-30.30 de `project-technical-context`)

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
