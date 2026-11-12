# Delta para sdd-docs-publication

> Parte del estado vigente: el resto de requisitos (REQ-27.6 en adelante) NO cambia. Esta delta solo retira `baseline.documents` de la lista cerrada de rutas de documentación de REQ-27.5, porque la baseline ajena se retira; `openspec/specs/**` ya figuraba en la lista.

## MODIFIED Requirements

### Requirement: Verbo `axiom sdd publish --change <x>` y Lista Cerrada de Rutas de Documentación (REQ-27.5)

La CLI DEBE exponer `axiom sdd publish --change <nombre>` con `--cwd` y `--json`, como subcomando de `axiom sdd` y no de `axiom branch`. La orden DEBE operar únicamente sobre las **rutas de documentación**, una lista cerrada formada por: `openspec/changes/<x>/**`; la carpeta de `openspec/changes/archive/**` que corresponde al cambio; `openspec/specs/**`; `openspec/INDEX.md`; y las rutas de contexto técnico mantenido declaradas en `axiom.yaml` (`technical_context.maintained`). Las rutas de `technical_context.reference` y `odd/tasks/**` NO DEBEN pertenecer nunca a la lista. Toda ruta que no esté en la lista es **código** a efectos de esta capacidad. El indicador opcional `--archive` selecciona la rama de archivado del cambio como rama asignada que `publish` espera (REQ-27.7 y REQ-27.14): la publicación posterior al archivado DEBE invocarse como `axiom sdd publish --change <x> --archive`. Sin `--change`, o con un nombre de cambio inválido, la orden DEBE rechazarse sin efectos.
(Previously: la lista incluía también las rutas de `baseline.documents`; la baseline ajena se retira)

#### Scenario: La orden existe en `sdd` y no en `branch`

- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd publish --help` y `axiom branch --help`
- **ENTONCES** la ayuda de `sdd` lista `publish`
- **Y** la ayuda de `branch` no lista `publish` ni `push`

#### Scenario: La lista incluye el contexto mantenido declarado

- **DADO** un `axiom.yaml` con `technical_context.maintained` que declara `docs/arquitectura.md`
- **CUANDO** `publish` clasifica `docs/arquitectura.md`
- **ENTONCES** es una ruta de documentación

#### Scenario: Las rutas de referencia y `odd/tasks` no son documentación

- **DADO** un `axiom.yaml` con `technical_context.reference` que declara `docs/referencia.md`
- **CUANDO** `publish` clasifica `docs/referencia.md` y `odd/tasks/mi-feature.md`
- **ENTONCES** ambas son código a efectos de esta capacidad
- **Y** ninguna puede confirmarse con `publish`

#### Scenario: Una carpeta archivada de otro cambio no es del cambio

- **DADO** el cambio `x` y una carpeta `openspec/changes/archive/2026-11-01-y/`
- **CUANDO** `publish --change x` clasifica sus rutas
- **ENTONCES** la carpeta archivada de `y` no pertenece a la lista de `x`

#### Scenario: Sin `--change` la orden se rechaza

- **DADO** un repositorio con cambios SDD
- **CUANDO** se ejecuta `axiom sdd publish` sin `--change`
- **ENTONCES** se rechaza con un error que nombra el argumento obligatorio
- **Y** no se confirma ni se sube nada
