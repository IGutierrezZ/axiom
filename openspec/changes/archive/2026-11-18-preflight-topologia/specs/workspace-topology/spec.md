# Delta para workspace-topology

> Parte del estado vigente: el resto de requisitos NO cambia. REQ-2.1 (`roles` con al menos un rol) y REQ-2.4 (multirepo con sus roles) se mantienen: la restricción de roles solo afecta a `monorepo-embedded`. La carga de la configuración (`workspace.Locate`) NO rechaza ninguno de los valores que esta delta hace reportar a `axiom workspace validate`.

## MODIFIED Requirements

### Requirement: Topología Monorepo Embebido (REQ-2.2)

En la topología `monorepo-embedded`, el repositorio de especificaciones DEBE apuntar a la raíz del propio monorrepo (`"."`) o a una subcarpeta interna existente (como `openspec` o `specs`). Además, el único rol admitido es `fullstack`: un monorepo no se subdivide en roles especializados, porque los roles especializados (`core`, `web`, `qa`, etc.) pertenecen a `multirepo`. `axiom workspace validate` DEBE reportar como error, en español, cualquier rol de `monorepo-embedded` distinto de `fullstack`, nombrando la topología, el rol y los pasos de migración (dejar solo el rol `fullstack` o declarar la topología `multirepo`), y DEBE finalizar con código de salida `1`. Ese rechazo es solo de validación: la carga de la configuración (`workspace.Locate`) NO DEBE fallar por ello, de modo que `axiom sdd status`, `axiom branch` y el resto de verbos siguen funcionando sobre un `axiom.yaml` existente. La regla NO se aplica a `multirepo`, que admite cualquier conjunto de roles válido.
(Previously: solo fijaba la ubicación del repositorio de especificaciones; un monorepo admitía cualquier conjunto de roles)

#### Scenario: Monorepo embebido con specs en raíz o subdirectorio válido
- **DADO** una configuración con topología `monorepo-embedded` y `specs_repository: "."`
- **Y** la carpeta existe dentro del espacio de trabajo
- **CUANDO** se ejecuta la validación de topología
- **ENTONCES** la topología se marca como válida y conforme

#### Scenario: Monorepo con el rol único `fullstack` es conforme
- **DADO** un `axiom.yaml` con topología `monorepo-embedded`, specs válido y el único rol `fullstack`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** el estado es `COMPLIANT` y no se reporta ningún error de roles

#### Scenario: Monorepo con roles especializados se rechaza con los pasos de migración
- **DADO** un `axiom.yaml` con topología `monorepo-embedded` y los roles `core` y `qa`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error en español que nombra `monorepo-embedded`, los roles `core` y `qa` y la migración (dejar solo `fullstack` o pasar a `multirepo`)
- **Y** el código de salida es `1`

#### Scenario: Monorepo con `fullstack` y otro rol se rechaza
- **DADO** un `axiom.yaml` con topología `monorepo-embedded` y los roles `fullstack` y `web`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta el error de roles nombrando `web`
- **Y** el código de salida es `1`

#### Scenario: La carga no se rompe con un monorepo de roles especializados
- **DADO** el mismo `axiom.yaml` de topología `monorepo-embedded` con los roles `core` y `qa`
- **CUANDO** se resuelve con `workspace.Locate` y se ejecuta `axiom sdd status`
- **ENTONCES** la carga tiene éxito y el status se proyecta sin error causado por los roles

#### Scenario: Multirepo con roles especializados sigue siendo conforme
- **DADO** un `axiom.yaml` con topología `multirepo`, specs válido y los roles `core` y `qa` con sus repositorios existentes
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** no se reporta ningún error de roles

## ADDED Requirements

### Requirement: El Almacén de Artefactos Declarado Solo Admite Ficheros: `engram` y `none` se Reportan (REQ-2.6)

En Axiom el almacén de artefactos de una sesión SDD es siempre `hybrid`. `axiom workspace validate` DEBE reportar como error, en español, un `artifact_store` (clave `artifact_store` o `artifactStore` de `openspec/config.yaml`) declarado `engram` o `none`, con independencia de que exista contexto técnico mantenido. El mensaje DEBE nombrar la clave, el valor declarado, la causa (con esos almacenes no hay paradas de tres salidas, ni publicación de compuertas ni relevo con `--hand-off`) y la migración (retirar la clave o declarar `hybrid`), y la validación DEBE finalizar con código de salida `1`. Un `artifact_store` declarado `openspec` o `hybrid`, o la ausencia de declaración, NO DEBE reportarse. El rechazo es solo de validación: la carga (`workspace.Locate`), `axiom sdd status` (que sin declaración sigue resolviendo `openspec`), la publicación y el archivado NO cambian de comportamiento. Este binario NO reescribe `openspec/config.yaml`.

#### Scenario: `engram` declarado se reporta con la migración
- **DADO** un `openspec/config.yaml` con `artifact_store: engram` y un `axiom.yaml` válido sin contexto técnico mantenido
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta un error en español que nombra `artifact_store`, `engram`, la ausencia de paradas, publicación y relevo, y la migración (retirar la clave o declarar `hybrid`)
- **Y** el código de salida es `1`

#### Scenario: `none` declarado se reporta
- **DADO** un `openspec/config.yaml` con `artifact_store: none`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta el mismo error nombrando `none`
- **Y** el código de salida es `1`

#### Scenario: `openspec`, `hybrid` y la ausencia de declaración se aceptan
- **DADO** tres proyectos válidos: con `artifact_store: openspec`, con `artifact_store: hybrid` y sin `artifact_store` declarado
- **CUANDO** se ejecuta `axiom workspace validate` en cada uno
- **ENTONCES** ninguno reporta error por el almacén

#### Scenario: La carga no se rompe con `engram` o `none`
- **DADO** un proyecto con `artifact_store: engram` declarado
- **CUANDO** se resuelve con `workspace.Locate` y se ejecuta `axiom sdd status --json`
- **ENTONCES** la carga y el status funcionan como antes de este cambio
- **Y** el almacén declarado sigue leyéndose como `engram`
