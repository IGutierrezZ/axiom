# Delta para change-branch-lifecycle

> Parte del estado vigente: el resto de requisitos (REQ-25.18, REQ-25.20 y siguientes) NO cambia. Esta delta solo sustituye `axiom sdd integrate` (`integration.yaml`) de la lista de escritores guardados de REQ-25.19, porque ese subcomando se retira (REQ-30.25 de `sdd-archive-lifecycle`) y su contexto pasa a `axiom archive context` (REQ-30.15 de `project-technical-context`).

## MODIFIED Requirements

### Requirement: Guarda A en Todo Escritor Go de la Carpeta del Cambio (REQ-25.19)

Todo escritor Go que cree o modifique ficheros dentro de la carpeta de un cambio DEBE consultar el veredicto de REQ-25.18 antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir nada y devolver el error tipado con la continuación. Esto incluye como mínimo: la creación de un incremento (el escritor compartido por el dashboard y `axiom change create`), la migración al acumulativo, la creación de handoff (`axiom handoff create` y el servicio del dashboard), `kickoff seal`, el registro de compuertas (`gate record`), el marcador de instancia del consentimiento de autoridad de edición, `axiom archive context prepare` y `axiom archive context verify` (plan y traza del contexto, que consultan el veredicto en el contexto de archivado), `axiom archive adopt scan --change` (inventario de adopción) y `axiom sdd archive-compose` (que consulta el veredicto en el contexto de archivado). El dashboard y `axiom change create` DEBEN ejecutar `ensure` y después escribir con el escritor guardado; el escritor guardado DEBE negarse fuera de rama aunque se invoque sin haber ejecutado `ensure`. La TUI solo lista cambios y no ejecuta `ensure`; cualquier escritura que alcance la carpeta de un cambio pasa por un escritor guardado. La guarda NO DEBE bloquear cambios exentos (REQ-25.18). Cada escritor DEBE tener una prueba propia de negativa.
(Previously: la lista incluía `axiom sdd integrate` (`integration.yaml`), retirado; ahora incluye `axiom archive context prepare|verify` y `axiom archive adopt scan --change`)

#### Scenario: Cada escritor se niega fuera de rama
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en la rama de integración
- **CUANDO** se invoca, por separado, cada escritor de la lista
- **ENTONCES** cada uno se niega con `wrong_branch` y la continuación de `ensure`
- **Y** ningún fichero de la carpeta del cambio se crea ni se modifica

#### Scenario: Cada escritor escribe en la rama correcta
- **DADO** el mismo cambio con el repositorio desplegado en `feat/x`
- **CUANDO** se invoca cada escritor que opera en el contexto ordinario
- **ENTONCES** escribe con normalidad

#### Scenario: `archive-compose` solo escribe en la rama de archivado
- **DADO** un cambio fusionado y el repositorio desplegado, por separado, en `feat/x` y en `feat/x-archive`
- **CUANDO** se invoca `axiom sdd archive-compose`
- **ENTONCES** se niega con `wrong_branch` y la continuación `axiom branch ensure --cwd <ruta> --change x --archive` en `feat/x`
- **Y** escribe con normalidad en `feat/x-archive`

#### Scenario: `axiom archive context prepare` solo escribe en la rama de archivado
- **DADO** un cambio fusionado y el repositorio desplegado, por separado, en `feat/x` y en `feat/x-archive`
- **CUANDO** se invoca `axiom archive context prepare --change x`
- **ENTONCES** se niega con `wrong_branch` y la continuación `axiom branch ensure --cwd <ruta> --change x --archive` en `feat/x`
- **Y** escribe con normalidad en `feat/x-archive`

#### Scenario: El dashboard y `axiom change create` ejecutan `ensure` y después escriben
- **DADO** un monorepo en modo `worktree` y el checkout principal en `main`
- **CUANDO** se crea el incremento `x` desde el dashboard o con `axiom change create`
- **ENTONCES** se ejecuta `ensure` para `x` antes de escribir
- **Y** el escritor guardado escribe `proposal.md` en el worktree de `feat/x`

#### Scenario: El escritor guardado invocado sin `ensure` en rama equivocada se niega
- **DADO** un monorepo con la política por defecto y el checkout en `main`
- **CUANDO** se invoca directamente el escritor guardado de creación del incremento `x` sin haber ejecutado `ensure`
- **ENTONCES** se niega con `wrong_branch` y la continuación `axiom branch ensure --change x`
- **Y** no se crea la carpeta `openspec/changes/x`

#### Scenario: La TUI solo lista
- **DADO** la TUI abierta sobre un proyecto
- **CUANDO** el usuario consulta el ciclo de vida de incrementos
- **ENTONCES** la TUI lista los cambios sin ejecutar `ensure` ni escribir en ninguna carpeta de cambio

#### Scenario: Un sumidero acumulativo está exento por nombre
- **DADO** el checkout en la rama de integración
- **CUANDO** se escribe en `openspec/changes/e2e-cumulative` o en `openspec/changes/cumulative-core`
- **ENTONCES** la escritura procede
