# Delta para multi-project-hub

> Parte del estado vigente: el registro global, la conmutación, la detección tecnológica (REQ-2.1), los endpoints, el selector de proyectos y las preferencias de worktree (REQ-25.29 a REQ-25.31) NO cambian. Esta delta (a) quita de `axiom init` la rama de baseline del proyecto, (b) hace que el detector proponga orígenes de migración y candidatos a `technical_context` y (c) muestra el informe de adopción tras el alta de proyecto.

## ADDED Requirements

### Requirement: El Detector del Hub Propone Orígenes de Migración y Candidatos a Contexto Técnico (REQ-30.26)

El detector del hub DEBE reconocer, además de `openspec/` y `.openspec/`, los orígenes candidatos de requisitos previos de un directorio y los candidatos a `technical_context` reutilizando el inventario de `axiom archive adopt scan` (REQ-30.9 de `openspec-adoption`), sin escribir ningún fichero ni modificar `axiom.yaml`. El resultado DEBE exponerse al alta de proyecto como un informe con los orígenes, el recuento de unidades y los candidatos a `maintained` y a `reference`. Un directorio sin orígenes candidatos DEBE producir un informe vacío sin error.

#### Scenario: El detector propone orígenes
- **DADO** un directorio con `docs/requisitos.md` y sin `openspec/specs/`
- **CUANDO** se ejecuta el detector del hub
- **ENTONCES** el informe lista `docs/requisitos.md` como origen candidato con su recuento de unidades
- **Y** no se escribe ningún fichero

#### Scenario: El detector propone candidatos a contexto técnico
- **DADO** un directorio con documentación técnica y sin `technical_context:` en `axiom.yaml`
- **CUANDO** se ejecuta el detector
- **ENTONCES** el informe propone candidatos a `maintained` y a `reference`
- **Y** `axiom.yaml` no se modifica

#### Scenario: Sin orígenes candidatos
- **DADO** un directorio sin documentación de requisitos
- **CUANDO** se ejecuta el detector
- **ENTONCES** el informe de migración está vacío y no hay error

---

### Requirement: El Alta de Proyecto Muestra el Informe de Adopción y la Instrucción (REQ-30.27)

Tras el alta de un proyecto (`/api/projects/init`), el dashboard DEBE mostrar el informe de orígenes de migración y de candidatos a `technical_context` (REQ-30.26) junto con la instrucción para lanzar la adopción desde el agente (`axiom archive adopt scan --change <cambio>` y el cambio de adopción). El dashboard NO DEBE ejecutar modelos ni crear el cambio de adopción: solo muestra el informe y la instrucción. Con un informe vacío NO DEBE mostrar la instrucción de adopción.

#### Scenario: Informe e instrucción tras el alta
- **DADO** un alta de proyecto cuyo directorio contiene un origen candidato
- **CUANDO** el alta termina con éxito
- **ENTONCES** el dashboard muestra el informe con el origen y la instrucción de adopción
- **Y** no se crea ningún cambio de adopción ni se ejecuta ningún modelo

#### Scenario: Informe vacío
- **DADO** un alta de proyecto sin orígenes candidatos
- **CUANDO** el alta termina con éxito
- **ENTONCES** el dashboard no muestra la instrucción de adopción

## MODIFIED Requirements

### Requirement: Generación de Andamiaje `axiom.yaml` y Registro Automático (REQ-2.2)

Al ejecutar `axiom init` sobre un directorio:
1. Si NO existe `axiom.yaml`, genera un archivo canónico con el nombre del proyecto, topología seleccionada (`monorepo-embedded`), repositorio de especificaciones (`openspec`), roles detectados (`core` y `qa`) y gobernanza en español.
2. Crea las carpetas base necesarias: `.axiom/inbox/skills/`, `openspec/specs/` y `openspec/changes/`.
3. Registra automáticamente el proyecto en `~/.axiom/workspaces.json` y lo marca como proyecto activo.
4. Si YA existe `axiom.yaml`, no lo sobreescribe; valida su estructura (incluido el bloque opcional `technical_context`; un bloque `baseline:` residual se rechaza con el error tipado de REQ-30.21) y lo registra en el Hub global si no estaba registrado.

Antes de escribir nada, `axiom init` DEBE rechazarse cuando el propio directorio contiene un puntero `.axiom-workspace` cuyo planning home, resuelto con el primitivo de la capacidad `planning-home-resolution` (REQ-24.1), es distinto de `<directorio>/openspec`: crear ese directorio produciría un segundo planning home y la ambigüedad de REQ-24.6 en todos los verbos. La comparación es entre rutas y NO exige que `<directorio>/openspec` exista, porque `init` es quien lo crea; la existencia del repositorio de especificaciones que exige REQ-24.7 (caso 4) es una condición del primitivo y no de esta comparación. El rechazo DEBE aplicarse con cualquier perfil, finalizar con código de salida distinto de cero, no escribir ni registrar nada y nombrar el puntero, el planning home resuelto y la continuación: inicializar el `workspaceRoot` resuelto con `axiom init --path <workspaceRoot>` o montar la topología a mano según la receta documentada (REQ-24.12).

Cuando el puntero del propio directorio no puede resolverse (por ejemplo, un puntero roto o un `axiom.yaml` apuntado ilegible, REQ-24.7), el primitivo rechaza la resolución y `axiom init` DEBE rechazarse con ese mismo error, que nombra la causa y la continuación, con código de salida distinto de cero y sin escribir ni registrar nada.

`axiom init` NO DEBE rechazarse cuando el puntero resuelve a `<directorio>/openspec` (exista ya ese directorio o no), cuando el propio directorio no contiene puntero (aunque lo contenga un directorio ancestro: el marcador más cercano gana, REQ-24.2) ni cuando no hay ningún marcador: en esos casos se comporta como describen los puntos 1 a 4.

(Previously: el punto 2 omitía `openspec/specs/` e `openspec/INDEX.md` cuando `axiom.yaml` declaraba una baseline del proyecto y el punto 4 validaba también el bloque `baseline`; la baseline ajena se retira y `openspec/specs/` se crea siempre)

#### Scenario: Inicialización exitosa de proyecto virgen
- **DADO** un directorio limpio sin archivos de Axiom
- **CUANDO** se ejecuta `axiom init --name "MiProyecto"`
- **ENTONCES** se crea `axiom.yaml` con la estructura canónica
- **Y** se crean las carpetas `openspec/specs/`, `openspec/changes/` y `.axiom/inbox/skills/`
- **Y** el proyecto queda registrado en `~/.axiom/workspaces.json`

#### Scenario: Inicialización sobre un `axiom.yaml` con `baseline:` residual
- **DADO** un directorio con un `axiom.yaml` que declara `baseline:` con al menos un documento
- **CUANDO** se ejecuta `axiom init`
- **ENTONCES** el comando se rechaza con el error tipado de REQ-30.21 y la continuación `axiom archive adopt`
- **Y** el `axiom.yaml` existente no se sobreescribe ni se registra el proyecto

#### Scenario: Inicialización rechazada en una carpeta maestra con puntero hacia otro planning home
- **DADO** el workspace canónico de la capacidad `planning-home-resolution`, cuya carpeta maestra `<maestra>` contiene `.axiom-workspace` (`config: repo-specs/axiom.yaml`) y no contiene `openspec/`
- **CUANDO** se ejecuta `axiom init --path <maestra>`, con o sin `--profile=knowledge`
- **ENTONCES** el comando se rechaza con código de salida distinto de cero
- **Y** el mensaje nombra el puntero, el planning home resuelto `<maestra>/repo-specs/openspec` y la continuación `axiom init --path <maestra>/repo-specs` o la receta manual documentada
- **Y** no se crea `<maestra>/openspec/`, ni `axiom.yaml`, ni `.axiom/inbox/skills/`, y `~/.axiom/workspaces.json` no se modifica

#### Scenario: Un puntero propio que no puede resolverse rechaza la inicialización
- **DADO** un directorio con un `.axiom-workspace` cuyo `config` apunta a un fichero inexistente, o a un `axiom.yaml` que no es YAML válido
- **CUANDO** se ejecuta `axiom init --path <directorio>`
- **ENTONCES** el comando se rechaza con código de salida distinto de cero y con el error del primitivo de resolución, que nombra la causa y la continuación
- **Y** no se crea ningún fichero ni directorio y `~/.axiom/workspaces.json` no se modifica

#### Scenario: Un puntero que resuelve a `<directorio>/openspec` no se rechaza aunque ese directorio no exista todavía
- **DADO** una carpeta `M` con `.axiom-workspace` (`config: axiom.yaml`), un `axiom.yaml` `monorepo-embedded` y sin el directorio `M/openspec`
- **CUANDO** se ejecuta `axiom init --path M`
- **ENTONCES** no se rechaza, porque el planning home que resuelve el puntero es `M/openspec` y la comparación no exige que exista
- **Y** la inicialización se comporta como describen los puntos 1 a 4 y crea `M/openspec/`

#### Scenario: Un puntero en un directorio ancestro no impide inicializar
- **DADO** el mismo workspace canónico y el directorio `<maestra>/repo-backend`, que no contiene `.axiom-workspace` propio
- **CUANDO** se ejecuta `axiom init --path <maestra>/repo-backend`
- **ENTONCES** no se rechaza por el puntero de `<maestra>` y la inicialización se comporta como antes de este incremento
