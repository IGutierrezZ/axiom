# Delta para change-branch-lifecycle

> Parte del estado vigente: el resto de requisitos (REQ-25.3, REQ-25.16, REQ-25.46, REQ-25.49 y REQ-25.50 incluidos) NO cambia. Esta delta añade un comando de solo lectura, `axiom branch scope`, que calcula qué aporta el cambio en cada repositorio de código, y amplía REQ-25.1 y REQ-25.2 (que hoy fijan el conjunto de verbos y no admiten `scope`). No modifica `ensure`, `status`, `done`, el plan materializado, `kickoff.yaml` ni `gates.yaml`. Los identificadores REQ-25.51 a REQ-25.54 son los siguientes libres tras REQ-25.50. Decisión del usuario del 2026-11-24: versión simple, una rama por repositorio (la del cambio que registra `ensure`) y un dominio corto de motivos de `unverifiable`.

## Fuera de alcance

- Registro del SHA de la punta de las ramas y localización de ramas borradas por SHA o por mensaje (tarea futura). Una rama borrada es un límite documentado: se declara `unverifiable` con `branch_missing`.
- Vigencia u obsolescencia de las rondas de verificación conjunta.
- El carril ODD (el comando no es aplicable).
- Verify o juicio en el archivado.
- `pruebas.md` (C-06-c).
- Contención de ramas de rol y políticas de rol `deferred` u `optional`.

## MODIFIED Requirements

### Requirement: Familia de Comandos `axiom branch` (REQ-25.1)

La CLI DEBE exponer la familia de primer nivel `axiom branch` con exactamente los subcomandos `ensure`, `status`, `scope`, `done` y `guard`. La familia DEBE ser neutral respecto al carril: `ensure`, `status` y `done` DEBEN aceptar `--change <nombre>` y `--lane sdd|odd` (por defecto `sdd`) y NO DEBEN depender de que exista un cambio SDD; `scope` DEBE aceptar `--change <nombre>` y solo opera sobre un cambio SDD con kickoff sellado (REQ-25.51). Ningún subcomando de `axiom branch` DEBE ejecutar `git push` ni `git commit`, crear un PR ni realizar operaciones remotas salvo la solicitada por `--fetch` (REQ-25.10), que `scope` no admite. La familia NO DEBE exponer subcomandos `pr`, `push` ni ningún comprobador para la CI. Toda ejecución de git del motor DEBE pasar por un único punto, de modo que la ausencia de operaciones remotas sea verificable con un observador instalado en ese punto y con una prueba de arquitectura que cubra el código de `internal/branch/**` y de `internal/gitcommon`. La operación de subir documentación del cambio es el verbo `axiom sdd publish`, que NO pertenece a esta familia, usa un puerto de publicación propio al que `axiom branch` no puede llegar y no altera este invariante (REQ-27.11 de `sdd-docs-publication`).
(Previously: «exactamente los subcomandos `ensure`, `status`, `done` y `guard`»; `scope` no existía y `--change` y `--lane` solo se enumeraban para `ensure`, `status` y `done`)

#### Scenario: La ayuda lista los cinco verbos
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch --help`
- **ENTONCES** la salida lista `ensure`, `status`, `scope`, `done` y `guard`
- **Y** no lista `pr`, `push` ni ningún comprobador para la CI

#### Scenario: Un subcomando no listado no existe
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch` seguido de `push`, de `pr` o del verbo retirado `check`
- **ENTONCES** el comando se rechaza como no reconocido con código de salida distinto de cero

#### Scenario: Ningún subcomando publica ni abre PR
- **DADO** un repositorio con un remoto configurado y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecutan `ensure`, `status`, `scope` y `done` sin `--fetch`
- **ENTONCES** el observador no registra ninguna invocación de `git push`, `git commit`, `git fetch`, `git pull` ni `git ls-remote`

#### Scenario: `scope` no admite `--fetch`
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch scope --change x --fetch`
- **ENTONCES** la orden se rechaza como uso incorrecto con código de salida distinto de cero
- **Y** no se ha ejecutado ninguna operación de red

#### Scenario: Una prueba de arquitectura fija el único punto de ejecución de git
- **DADO** el código fuente de los paquetes bajo `internal/branch/**` y de `internal/gitcommon`
- **CUANDO** se ejecuta la prueba de arquitectura
- **ENTONCES** ningún paquete lanza procesos ni abre conexiones de red por otra vía que no sea ese único punto de ejecución de git
- **Y** la prueba falla si se añade una ejecución de git fuera de ese punto

#### Scenario: El carril ODD no exige un cambio SDD
- **DADO** un proyecto sin ningún cambio bajo `openspec/changes/`
- **CUANDO** se ejecuta `axiom branch ensure --lane odd --change mi-feature`
- **ENTONCES** la orden se ejecuta sin requerir `kickoff.yaml` ni `proposal.md`

#### Scenario: El verbo de publicación no pertenece a la familia
- **DADO** el binario `axiom`
- **CUANDO** se comparan `axiom branch --help` y `axiom sdd --help`
- **ENTONCES** `publish` figura solo en la ayuda de `axiom sdd`
- **Y** el puerto de publicación no es importado por `internal/branch/**`

### Requirement: Contrato de Salida Común con Rutas Absolutas (REQ-25.2)

`ensure`, `status`, `scope` y `done` DEBEN admitir `--json` y `--cwd <ruta>`. La salida JSON DEBE incluir un estado tipado y, cuando no sea «listo», uno de los códigos tipados de esta capacidad. Todo valor de estado o de clasificación del JSON DEBE ser un token estable en inglés (`not_started`, `correct`, `not_applicable`, `ancestry`, `equivalent_tree`, `pr_merged`, `forced`, etc.) acompañado, en un campo aparte, de una etiqueta legible en castellano; los tokens NO DEBEN traducirse. Toda ruta de la salida (raíz git de cada repositorio, planning home, worktrees, raíz de worktrees) DEBE ser absoluta. El código de salida DEBE ser `0` únicamente cuando la operación ha dejado el estado pedido; todo rechazo, todo `decision_required` y todo error DEBEN terminar con código distinto de cero. Para `scope`, que es de solo lectura, «el estado pedido» es que el alcance se haya calculado: el código `0` se devuelve aunque haya repositorios `unverifiable`, que son información del resultado y no un fallo (REQ-25.51). Los mensajes legibles DEBEN estar en español y nombrar la continuación. Una configuración ilegible o inválida DEBE terminar con `config_invalid` y una invocación de git que falle por una causa no cubierta por otro código con `git_failed`, que DEBE nombrar la operación y resumir el error. Ninguna fase posterior a `ensure` DEBE depender del directorio de trabajo heredado: las órdenes posteriores reciben `--cwd` con una ruta tomada de este JSON.
(Previously: el contrato enumeraba `ensure`, `status` y `done`, y el código `0` no tenía una definición para un verbo de solo lectura; antes de eso incluía el comprobador de CI y los valores de estado no tenían un token estable con etiqueta separada)

#### Scenario: Todas las rutas son absolutas
- **DADO** un proyecto `monorepo-embedded` con modo `worktree`
- **CUANDO** se ejecuta `axiom branch ensure --change x --json`
- **ENTONCES** cada ruta del JSON es absoluta
- **Y** el JSON incluye, por cada repositorio, su raíz git, su rama y la ruta de su worktree

#### Scenario: Un rechazo termina con código distinto de cero y código tipado
- **DADO** un proyecto en modo `in_place` con el árbol sucio
- **CUANDO** se ejecuta `axiom branch ensure --change x --json`
- **ENTONCES** el código de salida es distinto de cero
- **Y** el JSON contiene el código tipado `dirty_worktree` y la continuación en español

#### Scenario: Los valores del JSON son tokens en inglés con etiqueta en castellano
- **DADO** un nombre de cambio sin ramas ni worktrees
- **CUANDO** se ejecuta `axiom branch status --change nuevo --json`
- **ENTONCES** el estado es el token `not_started`
- **Y** un campo aparte contiene la etiqueta `no iniciado`

#### Scenario: Una configuración ilegible se informa con `config_invalid`
- **DADO** un `axiom.yaml` ilegible o con `git.branch_prefix` inválido
- **CUANDO** se ejecuta `axiom branch ensure --change x --json`
- **ENTONCES** el código tipado es `config_invalid` y el código de salida es distinto de cero
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Un fallo de git no cubierto se informa con `git_failed`
- **DADO** una invocación de git que falla por una causa que ningún otro código cubre
- **CUANDO** se ejecuta `axiom branch ensure --change x --json`
- **ENTONCES** el código tipado es `git_failed` y el JSON nombra la operación y resume el error
- **Y** una repetición de la orden, ya resuelta la causa, completa el plan sin duplicar nada (REQ-25.7)

#### Scenario: `--cwd` fija el directorio de partida
- **DADO** un worktree de un cambio y un directorio de trabajo ajeno al proyecto
- **CUANDO** se ejecuta `axiom branch status --change x --cwd <ruta absoluta del worktree> --json`
- **ENTONCES** el resultado corresponde al proyecto del `--cwd` y no al directorio de trabajo

#### Scenario: `scope` admite `--json` y `--cwd` con rutas absolutas
- **DADO** un cambio con kickoff sellado y un directorio de trabajo ajeno al proyecto
- **CUANDO** se ejecuta `axiom branch scope --change x --cwd <ruta absoluta> --json`
- **ENTONCES** el resultado corresponde al proyecto del `--cwd`
- **Y** toda ruta del JSON es absoluta y todo valor de estado es un token en inglés con su etiqueta en castellano

#### Scenario: `scope` con un repositorio `unverifiable` termina con código 0
- **DADO** un cambio con un repositorio `unverifiable` por `branch_missing`
- **CUANDO** se ejecuta `axiom branch scope --change x --json`
- **ENTONCES** el código de salida es `0`
- **Y** el repositorio figura como `unverifiable` con su motivo

## ADDED Requirements

### Requirement: `axiom branch scope`: Alcance del Cambio por Repositorio, de Solo Lectura y con Contrato Versionado (REQ-25.51)

`axiom branch scope --change <cambio> --json` DEBE calcular, para cada repositorio de código de la instantánea sellada del cambio, qué aporta el cambio en ese repositorio, usando **una única rama por repositorio**: la rama del cambio que registra `axiom branch ensure`. DEBE emitir un contrato JSON versionado propio (campo `schema` con su versión; distinto del de `axiom branch status`). Cada entrada de repositorio DEBE informar de: identificador del repositorio; rama del cambio; base del modo de promoción (REQ-25.46: `main_branch` en modo `branch`, `integration_branch` en modo `release`) y rama de integración; punto de partida; rango de commits y lista de commits propios; lista de ficheros cambiados por esos commits, **sin** los de `openspec/changes/<cambio>/`; si la rama del cambio está fusionada en la rama de integración (sí o no); y un resultado tipado (REQ-25.53). El contrato DEBE incluir además un resumen a nivel de cambio con la oferta de verificación conjunta (REQ-25.54).

La orden DEBE ser de solo lectura: NO DEBE ejecutar `fetch` ni ningún acceso a red, NO DEBE crear ni borrar ramas ni worktrees, NO DEBE modificar referencias, índice ni árbol de trabajo, y NO DEBE escribir en `kickoff.yaml`, `gates.yaml` ni en los documentos del cambio. Solo inspecciona commits: el árbol de trabajo, el índice y los ficheros sin seguimiento NO DEBEN entrar nunca en el resultado. El repositorio de especificaciones in situ de `multirepo` DEBE aparecer con resultado `exempt` y sin cálculo.

**Rechazos y salida.** La orden DEBE rechazar con un error tipado y salida `1`, antes de calcular nada y sin mutar el repositorio: cambio del carril `odd` o kickoff sin instantánea de repositorios, con `scope_not_applicable`; cambio inexistente o sin `kickoff.yaml`, con `kickoff_not_sealed`; configuración, kickoff o instantánea ilegibles, con `config_invalid`; y un fallo de git que ningún otro código cubra, con `git_failed`, que no se degrada a un resultado por repositorio. Un uso incorrecto (falta `--change`, nombre inválido, bandera desconocida o `--fetch`) DEBE terminar con código `2`. El código de salida DEBE ser `0` cuando el alcance se calculó, aunque algún repositorio sea `unverifiable` (REQ-25.2).

**Topologías.** En `monorepo-embedded` la única entrada de código es el repositorio del propio proyecto, con `openspec/changes/<cambio>/` excluida de la lista de ficheros. Un cambio cuya instantánea solo contiene el repositorio de specs in situ DEBE producir un resultado calculado con todas sus entradas `exempt` y la oferta `nothing_to_verify`, sin rechazo.

#### Scenario: Camino feliz, un repositorio con commits propios
- **DADO** un cambio `x` en un repositorio de código con la rama `feat/x` y tres commits propios que modifican `a.go` y `b.go`
- **CUANDO** se ejecuta `axiom branch scope --change x --json`
- **ENTONCES** la entrada del repositorio contiene la rama `feat/x`, su base, la rama de integración, el punto de partida y el rango de esos tres commits
- **Y** la lista de ficheros es exactamente `a.go` y `b.go`
- **Y** el resultado es `ok` y el código de salida es `0`

#### Scenario: La carpeta del cambio no entra en la lista de ficheros
- **DADO** un commit propio que modifica `a.go` y `openspec/changes/x/tasks.md`
- **CUANDO** se calcula el alcance
- **ENTONCES** la lista de ficheros contiene `a.go` y ningún fichero de `openspec/changes/x/`

#### Scenario: Multirepo con specs in situ
- **DADO** un cambio `multirepo` con los repositorios de código `api` y `web` y el repositorio de specs in situ
- **CUANDO** se ejecuta `axiom branch scope --change x --json`
- **ENTONCES** hay una entrada calculada por cada uno de `api` y `web`
- **Y** el repositorio de specs aparece con resultado `exempt` y sin rango ni ficheros

#### Scenario: La orden no cambia nada
- **DADO** un repositorio con ramas, índice y worktrees conocidos, y un worktree con cambios con y sin stage y un fichero sin seguimiento
- **CUANDO** se ejecuta `axiom branch scope --change x --json`
- **ENTONCES** las referencias, el índice y los worktrees son idénticos antes y después
- **Y** no se ha ejecutado ningún `fetch` ni se ha abierto ninguna conexión de red
- **Y** ninguno de esos ficheros no confirmados figura en la lista de ficheros

#### Scenario: El carril ODD no admite el comando
- **DADO** un cambio del carril `odd`, que no tiene instantánea sellada
- **CUANDO** se ejecuta `axiom branch scope --change x --json`
- **ENTONCES** la orden se rechaza con `scope_not_applicable` y salida distinta de cero
- **Y** no se ha modificado ninguna referencia ni fichero

### Requirement: `scope`: Punto de Partida Estable en Modo `release` y Exclusión de Commits Ajenos (REQ-25.52)

El punto de partida del alcance DEBE ser el punto de bifurcación de la rama del cambio respecto de la base de su modo, y DEBE ser **estable** antes y después de que la rama se fusione en la rama de integración: un commit ajeno que llegue a la integración después de la bifurcación NUNCA DEBE entrar en el alcance.

- **Modo `branch`:** el punto de partida es el de bifurcación respecto de `main_branch`, que todavía no contiene el cambio.
- **Modo `release`:** el punto de partida DEBE seguir siendo el punto de bifurcación tras fusionar la rama en la integración, con commit de fusión, con squash o con avance rápido. El cálculo ingenuo (el ancestro común entre la integración y la rama ya fusionada) deja el alcance vacío y NO DEBE presentarse como un alcance válido: cuando el punto de bifurcación no pueda determinarse, el repositorio DEBE ser `unverifiable` con motivo `fork_point_undetermined`.

El alcance DEBE contener solo los commits que aporta el cambio. Los commits ajenos que llegaron a la rama del cambio por una fusión de la rama de integración DEBEN excluirse del rango, de los commits y de los ficheros **cuando pueda distinguirse cuáles son**, y el contrato DEBE listarlos como excluidos. Cuando no pueda distinguirse, el alcance se calcula sobre el rango disponible y la entrada DEBE llevar el aviso `foreign_commits_possible`; los commits ajenos nunca se incluyen en silencio.

#### Scenario: Commit ajeno en develop, modo release, rama sin fusionar
- **DADO** un repositorio en modo `release` donde `feat/x` sale de `develop`, después llega a `develop` un commit ajeno `F` que modifica `otro.go`, y `feat/x` aún no está fusionada
- **CUANDO** se calcula el alcance
- **ENTONCES** el rango y los ficheros contienen solo los commits y ficheros de `feat/x`
- **Y** ni `F` ni `otro.go` aparecen

#### Scenario: Modo release, rama ya fusionada, mismo alcance
- **DADO** el mismo repositorio con `feat/x` ya fusionada en `develop` mediante un commit de fusión posterior a `F`
- **CUANDO** se calcula el alcance
- **ENTONCES** el resultado es el mismo conjunto de commits y de ficheros que antes de fusionar
- **Y** ni `F` ni `otro.go` aparecen

#### Scenario: Modo release, punto de bifurcación no determinable
- **DADO** un repositorio en modo `release` con `feat/x` fusionada por avance rápido y sin forma de determinar su punto de bifurcación
- **CUANDO** se calcula el alcance
- **ENTONCES** la entrada es `unverifiable` con motivo `fork_point_undetermined`
- **Y** no se presenta un rango vacío como alcance válido

#### Scenario: Commits ajenos no distinguibles se avisan
- **DADO** un repositorio en modo `branch` donde la rama del cambio contiene commits de `develop` que no pueden distinguirse de los propios
- **CUANDO** se calcula el alcance
- **ENTONCES** la entrada declara el aviso `foreign_commits_possible`
- **Y** el alcance no presenta esos commits como propios en silencio

### Requirement: `scope`: Resultado Tipado por Repositorio y Motivos de `unverifiable` (REQ-25.53)

Cada entrada de repositorio DEBE llevar exactamente un resultado del dominio cerrado `ok`, `empty`, `unverifiable` o `exempt`, y NUNCA DEBE omitirse un repositorio de la instantánea sin una entrada que lo declare.

- `ok`: la rama del cambio existe, tiene commits propios y el alcance se calculó.
- `empty`: la rama del cambio existe y no aporta commits propios, o todos sus commits propios tocan únicamente `openspec/changes/<cambio>/`.
- `unverifiable`: el alcance no puede calcularse con certeza. El motivo DEBE pertenecer a este dominio cerrado, y no puede haber otro:
  - `branch_missing`: la rama del cambio no existe en local ni como `origin/<rama>` ya descargada. El mensaje DEBE decir «la rama ya no está o falta un fetch autorizado». La orden NO DEBE ejecutar `fetch`.
  - `base_missing`: la base del modo o la rama de integración no existen en local ni como referencia remota ya descargada.
  - `fork_point_undetermined`: no puede fijarse el punto de partida (REQ-25.52).
- `exempt`: el repositorio de especificaciones in situ (REQ-25.51).

Cuando la rama solo existe como `origin/<rama>` ya descargada, la entrada DEBE declarar que usó esa referencia. Un repositorio `unverifiable` NO DEBE impedir el cálculo del resto. Una rama borrada tras fusionarse es un límite documentado de esta capacidad: se declara `unverifiable` con `branch_missing` y la doctrina recomienda no borrar las ramas del cambio hasta haber hecho la verificación conjunta.

#### Scenario: Rama borrada o ausente
- **DADO** un cambio `multirepo` cuya instantánea nombra la rama `feat/x` del repositorio `web`, que no existe en local ni como `origin/feat/x` descargada, y un repositorio `api` con la rama presente
- **CUANDO** se ejecuta `axiom branch scope --change x --json`
- **ENTONCES** `web` es `unverifiable` con motivo `branch_missing` y el mensaje «la rama ya no está o falta un fetch autorizado»
- **Y** `api` se calcula con normalidad
- **Y** no se ha ejecutado ningún `fetch` ni cambiado ninguna referencia

#### Scenario: Rama solo como referencia remota ya descargada
- **DADO** una rama `feat/x` ausente en local pero presente como `origin/feat/x` ya descargada
- **CUANDO** se calcula el alcance
- **ENTONCES** la entrada declara que usó `origin/feat/x`
- **Y** el resultado no es `unverifiable`

#### Scenario: Base o integración ausentes
- **DADO** un repositorio cuya rama de integración no existe en local ni descargada
- **CUANDO** se calcula el alcance
- **ENTONCES** el repositorio es `unverifiable` con motivo `base_missing`
- **Y** no se ha ejecutado ningún `fetch`

#### Scenario: Rama sin commits propios
- **DADO** una rama del cambio recién creada en esta máquina sin commits propios, en modo `branch`
- **CUANDO** se calcula el alcance
- **ENTONCES** el resultado es `empty` y no `ok` con un rango vacío

#### Scenario: Ningún repositorio se omite
- **DADO** una instantánea con tres repositorios de código y el de specs in situ, uno de ellos `unverifiable`
- **CUANDO** se calcula el alcance
- **ENTONCES** el JSON contiene cuatro entradas, cada una con su resultado tipado
- **Y** el código de salida es `0`

### Requirement: `scope`: Fusión en la Integración y Resumen con la Oferta (REQ-25.54)

El indicador de fusión de cada repositorio DEBE comprobar que la rama del cambio está fusionada en la **rama de integración** (también en modo `branch`, donde la siguiente PR hacia la rama principal no cuenta; REQ-25.50), usando solo las capas de ascendencia (`ancestry`) y de árbol equivalente (`equivalent_tree`) de REQ-25.16, sin red y sin la capa `pr_merged`. El resultado de cada repositorio DEBE declarar si está fusionado (sí o no).

El resumen del cambio DEBE incluir un campo `offer` con exactamente uno de los valores `unverifiable`, `pending_merge`, `nothing_to_verify` u `offerable`, con esta precedencia cuando coincidan varios: `unverifiable` (algún repositorio es `unverifiable`) > `pending_merge` (algún repositorio de código `ok` no está fusionado en la integración) > `nothing_to_verify` (ningún repositorio es `ok`: todos `empty` o `exempt`) > `offerable` (todos los repositorios `ok` están fusionados). El resumen DEBE listar los repositorios que motivan un valor distinto de `offerable` con su rama y su motivo. Los repositorios `empty` y `exempt` no son pendientes. El resumen es información de solo lectura: ningún otro verbo DEBE depender de él.

#### Scenario: Oferta posible con todo fusionado
- **DADO** un cambio con todas las ramas fusionadas en `develop` y un repositorio `ok`
- **CUANDO** se calcula el alcance
- **ENTONCES** `offer` es `offerable`

#### Scenario: Pendiente de fusionar
- **DADO** un cambio con una rama sin fusionar en `develop` y ningún repositorio `unverifiable`
- **CUANDO** se calcula el alcance
- **ENTONCES** `offer` es `pending_merge` y el resumen nombra el repositorio y la rama

#### Scenario: Nada que verificar
- **DADO** un cambio con todos los repositorios de código `empty`
- **CUANDO** se calcula el alcance
- **ENTONCES** `offer` es `nothing_to_verify`

#### Scenario: Modo release con la rama fusionada
- **DADO** un repositorio en modo `release` con `feat/x` fusionada en `develop`, aunque no en la rama principal
- **CUANDO** se calcula el alcance
- **ENTONCES** la rama figura como fusionada y `offer` es `offerable`

#### Scenario: Un repositorio no verificable domina la oferta
- **DADO** un cambio con un repositorio `unverifiable` y otra rama sin fusionar
- **CUANDO** se calcula el alcance
- **ENTONCES** `offer` es `unverifiable`
- **Y** el resumen lista ambos casos con su motivo
