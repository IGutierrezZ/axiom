# Delta para sdd-lane-kickoff

> Parte del estado vigente: el resto de requisitos NO cambia. REQ-21.3 (entrada al cuestionario de pre-vuelo) se mantiene tal cual: su texto remite a la capacidad `sdd-preflight-configuration` y no fija el número ni el contenido de los grupos, que ahora depende de la topología (REQ-28.15 de esa capacidad). Esta delta solo mueve a `multirepo` el escenario de la instantánea que usaba un monorepo con roles especializados.

## MODIFIED Requirements

### Requirement: Instantánea de Política Git en `kickoff.yaml`, Escrita una Sola Vez por `seal` (REQ-25.36)

`axiom sdd kickoff seal` DEBE escribir en `kickoff.yaml`, junto al resto de la configuración sellada y en la misma única escritura, la instantánea de política git del cambio: la rama de integración y el prefijo resueltos, la base (nombre de rama, no SHA) de cada repositorio y los nombres de rama derivados del cambio (rama del cambio, por repositorio con dos o más roles las ramas de rol y, en los repositorios con rama de cambio, la rama de archivado `<prefijo>/<cambio>-archive`), calculados con la política de `axiom.yaml`, las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`) y los roles que se sellan. La instantánea contiene entradas de política, nunca resultados: NO DEBE incluir SHAs, rutas absolutas ni rutas de worktree, que son personales o cambian. La instantánea se escribe una sola vez: conserva la semántica de sellado único, en la que gana la primera publicación, y NINGUNA otra orden, incluida `ensure`, DEBE modificarla. Un segundo `seal` desde la rama del cambio NO DEBE reescribir `kickoff.yaml`: DEBE terminar con éxito, devolver el kickoff ya sellado e indicar que no escribió (texto «Kickoff ya estaba sellado»; `wrote: false` en `--json`). Un segundo `seal` desde otra rama DEBE rechazarse con `wrong_branch` (veredicto de REQ-25.18 de `change-branch-lifecycle`) sin modificar nada. La lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarla sin error.
(Previously: decía que «un segundo `seal` se rechaza como hoy», pero el sellado único nunca ha rechazado un segundo `seal` desde la rama del cambio: devuelve el ganador sin reescribir, como fijan `internal/kickoff/seal.go` y sus pruebas; el escenario de la instantánea se ejecutaba sobre un monorepo con los roles `core` y `qa`, y ahora se divide en un monorepo con `fullstack` y un `multirepo` con roles especializados)

#### Scenario: El sellado escribe la instantánea
- **DADO** un monorepo con `git.integration_branch: main`, `git.branch_prefix: feat` y el rol único `fullstack`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...` desde la rama del cambio
- **ENTONCES** `kickoff.yaml` contiene la integración `main`, el prefijo `feat`, la base `main` del repositorio y los nombres `feat/x` y la rama de archivado `feat/x-archive`
- **Y** no contiene ninguna rama de rol
- **Y** no contiene SHAs ni rutas absolutas

#### Scenario: El sellado en multirepo escribe las ramas de rol y ninguna rama de archivado
- **DADO** un multirepo con `git.integration_branch: main`, `git.branch_prefix: feat` y los roles `core` y `qa` sobre el mismo repositorio de código `repo-backend`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...` desde la rama que corresponde (REQ-25.38)
- **ENTONCES** `kickoff.yaml` contiene la integración `main`, el prefijo `feat`, la base `main` de `repo-backend` y, para `repo-backend`, los nombres `feat/x`, `feat/x-core` y `feat/x-qa`
- **Y** no contiene rama de archivado para `repo-backend`
- **Y** no contiene SHAs ni rutas absolutas

#### Scenario: La instantánea refleja las excepciones por cambio
- **DADO** un cambio materializado con `ensure --change x --branch inc/x`
- **CUANDO** se sella el cambio
- **ENTONCES** la instantánea nombra la rama del cambio `inc/x`

#### Scenario: La instantánea no se reescribe
- **DADO** un cambio sellado con instantánea
- **CUANDO** se ejecutan de nuevo, desde la rama del cambio, `kickoff seal` con otra configuración y después `axiom branch ensure --change x`
- **ENTONCES** el segundo `seal` termina con éxito, indica «Kickoff ya estaba sellado» y devuelve la configuración del primero
- **Y** `kickoff.yaml` no se modifica en ninguna de las dos órdenes

#### Scenario: Un segundo sellado desde otra rama se rechaza
- **DADO** un cambio sellado con instantánea cuya rama del cambio es `feat/x`
- **CUANDO** se ejecuta de nuevo `kickoff seal` desde `main`
- **ENTONCES** la orden se rechaza con `wrong_branch`, nombrando la rama actual `main` y la esperada `feat/x`
- **Y** `kickoff.yaml` no se modifica

#### Scenario: La lectura tolera la ausencia de instantánea
- **DADO** un `kickoff.yaml` sin instantánea
- **CUANDO** se carga con `ParseKickoff`
- **ENTONCES** la carga tiene éxito y la instantánea queda vacía

#### Scenario: Un binario anterior ignora la instantánea
- **DADO** un `kickoff.yaml` con instantánea
- **CUANDO** lo lee un binario anterior a esta capacidad
- **ENTONCES** carga el resto de la configuración sellada sin error

