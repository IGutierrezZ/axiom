# Delta para Ciclo de Vida de las Ramas del Cambio (change-branch-lifecycle)

> **Cambio:** `autofetch-specs` (C-05-d). **Idioma:** castellano de España; identificadores, comandos y claves en inglés.
> El interruptor `autofetch` es la preferencia personal de RF-HUB-07 y RF-HUB-13 (`multi-project-hub`). «Interruptor encendido» y «apagado» se entienden siempre respecto del proyecto actual. La tabla «Requerimientos de un vistazo» y la sección «No cubre esta capacidad» se actualizan al integrar: RF-BRA-01, RF-BRA-10 y RF-BRA-13 dejan de decir «sin operaciones remotas / sin fetch implícito» a secas, y la línea «Operaciones remotas por defecto» pasa a citar la excepción del interruptor.
> `axiom branch status` sigue siendo solo local (RF-BRA-16): no cambia y no incorpora ningún bloque remoto.

## ADDED Requirements

### Requirement: Autofetch del Repositorio de Specs con el Interruptor Personal (RF-BRA-39)

Con el interruptor de autofetch encendido, el sistema MUST ofrecer el verbo propio `axiom autofetch run [--cwd] [--agent] [--json]`, que NO pertenece a la familia `axiom branch`. Con el interruptor apagado, `run` MUST NOT realizar ninguna operación de red ni escribir, MUST informar de que el autofetch está apagado y MUST terminar con código 0. Un proyecto no registrado o un hub ilegible cuentan como apagado (RF-HUB-07). El aviso se emite solo cuando `run` se invoca a mano (sin `--agent`); el hook de sesión (`run --agent`) se comporta exactamente como con el interruptor apagado, sin aviso.

El ciclo de `run` con el interruptor encendido MUST ser este:

1. **Fetch.** Ejecuta `git fetch origin` en el repositorio de specs (en `monorepo-embedded`, el propio repositorio; en `multirepo`, el repositorio de specs in situ). MUST NOT usar `--prune` ni escribir en el remoto. MUST estar acotado por un tiempo máximo de unos 8 segundos (el `run` completo en modo hook, con fetch, lecturas locales y avance, MUST terminar en unos 10 segundos como máximo) y MUST NOT pedir credenciales de forma interactiva. MUST NOT repetir el fetch antes de transcurrido un intervalo mínimo desde el último fetch correcto: dentro del intervalo `run` MUST NOT hacer red.
2. **Avance de la rama de integración.** Avanza la rama de integración local (`git.integration_branch`) a `origin/<integración>` SOLO por fast-forward estricto (`merge --ff-only`) y SOLO si la rama está desplegada en un checkout, el árbol de ese checkout está limpio y la rama local no ha divergido de `origin/<integración>`. Si la rama ya está al día, no hace nada. Si no se cumple alguna condición (árbol sucio, divergencia, rama no desplegada en ningún checkout, operación git en curso (merge, rebase, cherry-pick) o HEAD desacoplado en el checkout de la integración, bloqueo de git u otro error del avance), NO DEBE mover la rama y DEBE avisar con la orden exacta que la persona puede ejecutar. El aviso MUST ser un aviso, no un fallo: el código de salida sigue siendo 0.
3. **Aviso por relevancia.** Una integración local por detrás de `origin/<integración>` que no se ha podido avanzar (árbol sucio, divergencia, rama no desplegada, operación git en curso o HEAD desacoplado en ese checkout) es relevante y se avisa siempre con la orden exacta; si la integración no está por detrás, no hay aviso de no-avance. Además avisa únicamente de (a) commits entrantes en `origin/<integración>` que tocan `openspec/changes/<cambio actual>/`, las specs vivas de las capacidades que el cambio actual declara en su carpeta `specs/`, o `odd/tasks/<feature>.md` del cambio actual, y (b) commits nuevos en `origin/<rama del cambio actual>` que la rama local no tiene. Cada aviso MUST nombrar la rama, el número de commits y su alcance, y MUST NOT listar nada irrelevante. Sin cambio actual resoluble, el ciclo hace el fetch y el avance pero no emite avisos de relevancia. Sin nada relevante, `run` MUST NOT emitir ningún aviso.

Prohibiciones, con el interruptor encendido y en cualquier circunstancia: el sistema MUST NOT avanzar `main_branch` cuando es distinta de `integration_branch`; cuando coinciden (monorepo de una sola etapa, o el `main` del repositorio de specs de un multirepo, que hace de integración), avanzarla por fast-forward estricto ES el avance de la integración que ordena el paso 2; MUST NOT mover la rama de un cambio, la rama de archivado ni la rama de un worktree de feature (ni rebase ni merge ni reset: solo avisar); MUST NOT avanzar ninguna rama local de un repositorio de código; MUST NOT ejecutar `push` ni `prune`.

Si el remoto es inalcanzable o el fetch agota su tiempo, `run` MUST NOT bloquear: deja un aviso, no avanza nada y termina con código 0. `run --json` MUST devolver un resultado tipado con el estado del interruptor, si hubo fetch o se omitió por el intervalo, si la rama de integración avanzó, y la lista de avisos con sus órdenes exactas.

#### Scenario: Con el interruptor apagado no hay red ni escritura

- **DADO** un proyecto registrado con el interruptor apagado y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom autofetch run`
- **ENTONCES** el observador no registra ninguna invocación de `git fetch`, `git pull`, `git push` ni `git ls-remote`
- **Y** ninguna rama ni fichero se modifica
- **Y** la orden informa de que el autofetch está apagado y termina con código 0

#### Scenario: Interruptor encendido, árbol limpio y sin divergencia avanza por fast-forward

- **DADO** el interruptor encendido, la rama de integración desplegada con el árbol limpio y `origin/<integración>` con commits nuevos que la rama local no tiene
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** se ejecuta `git fetch origin` sin `--prune`
- **Y** la rama de integración local queda en el mismo commit que `origin/<integración>` sin crear ningún commit de fusión
- **Y** el resultado indica que la rama avanzó

#### Scenario: Con la rama ya al día no se mueve nada

- **DADO** el interruptor encendido y la rama de integración local igual a `origin/<integración>` tras el fetch
- **CUANDO** se ejecuta `axiom autofetch run`
- **ENTONCES** la rama de integración no cambia y no se emite ningún aviso

#### Scenario: Árbol sucio solo avisa con la orden exacta

- **DADO** el interruptor encendido, la rama de integración desplegada con un fichero modificado sin confirmar y commits nuevos en `origin/<integración>`
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** la rama de integración local no se mueve y el árbol queda intacto
- **Y** el resultado incluye un aviso con la orden exacta que la persona puede ejecutar para avanzar
- **Y** la orden termina con código 0

#### Scenario: Divergencia solo avisa con la orden exacta

- **DADO** el interruptor encendido y una rama de integración local con un commit que no está en `origin/<integración>`, que además tiene commits que la local no tiene
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** la rama de integración local no se mueve
- **Y** el resultado incluye un aviso de divergencia con la orden exacta para resolverla
- **Y** la orden termina con código 0

#### Scenario: Rama de integración no desplegada solo avisa

- **DADO** el interruptor encendido y la rama de integración local sin desplegar en ningún checkout ni worktree, con commits nuevos en `origin/<integración>`
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** la rama no se mueve
- **Y** el resultado incluye un aviso con la orden exacta

#### Scenario: La rama de un cambio nunca se mueve

- **DADO** el interruptor encendido, la rama `inc/x` de un cambio con commits locales o desplegada en un worktree, y commits nuevos en `origin/inc/x`
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** `inc/x` local y el contenido de su worktree no cambian
- **Y** el resultado incluye un aviso de commits nuevos en `origin/inc/x`

#### Scenario: `main_branch` y los repositorios de código no se avanzan

- **DADO** un multirepo con el interruptor encendido, `main_branch` distinta de la rama de integración y commits nuevos en `origin/<main_branch>` y en `origin/<rama>` de un repositorio de código
- **CUANDO** se ejecuta `axiom autofetch run`
- **ENTONCES** ni `main_branch` ni ninguna rama local de los repositorios de código cambian
- **Y** no se ejecuta ningún `git fetch` en los repositorios de código

#### Scenario: Con `main_branch` igual a la integración se avanza por fast-forward

- **DADO** el interruptor encendido, `main_branch` igual a `integration_branch`, desplegada con el árbol limpio y con commits nuevos en `origin/<integración>`
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** esa rama avanza por fast-forward estricto a `origin/<integración>` sin commit de fusión
- **Y** ninguna rama de los repositorios de código cambia

#### Scenario: Integración por detrás con operación git en curso solo avisa

- **DADO** el interruptor encendido, la integración local por detrás de `origin/<integración>` y el checkout de la integración con un rebase en curso o con HEAD desacoplado
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** la rama no se mueve y el resultado incluye un aviso con la orden exacta
- **Y** la orden termina con código 0

#### Scenario: Integración al día no emite aviso de no-avance

- **DADO** el interruptor encendido, la integración igual a `origin/<integración>` y el árbol de su checkout sucio
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** el resultado no incluye ningún aviso de no-avance

#### Scenario: Nunca se hace push ni prune

- **DADO** el interruptor encendido y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom autofetch run`
- **ENTONCES** el observador no registra ninguna invocación de `git push` y el fetch no lleva `--prune`

#### Scenario: Remoto inalcanzable no bloquea

- **DADO** el interruptor encendido y un remoto inalcanzable
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** la orden termina dentro de la cota de unos 10 segundos (fetch de unos 8 segundos como máximo) con código 0
- **Y** el resultado incluye un aviso de remoto inalcanzable
- **Y** ninguna rama se modifica

#### Scenario: El intervalo mínimo evita repetir el fetch

- **DADO** el interruptor encendido y un fetch correcto realizado hace menos del intervalo mínimo
- **CUANDO** se ejecuta de nuevo `axiom autofetch run --json`
- **ENTONCES** no se ejecuta ningún `git fetch`
- **Y** el resultado indica que el fetch se omitió por el intervalo

#### Scenario: Transcurrido el intervalo el fetch se repite

- **DADO** el interruptor encendido y un último fetch correcto anterior al intervalo mínimo
- **CUANDO** se ejecuta `axiom autofetch run`
- **ENTONCES** se ejecuta `git fetch origin`

#### Scenario: Solo se avisa de lo relevante para el cambio actual

- **DADO** el interruptor encendido, el cambio actual `x` y `origin/<integración>` con tres commits nuevos: uno que toca `openspec/changes/x/`, uno que toca `openspec/specs/<capacidad declarada por x>/spec.md` y uno que solo toca `docs/otra-cosa.md`
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** el resultado incluye avisos solo por los dos primeros commits
- **Y** el tercer commit no figura en ningún aviso

#### Scenario: Un commit entrante sobre el documento ODD del cambio es relevante

- **DADO** el interruptor encendido, la feature ODD `f` y un commit nuevo en `origin/<integración>` que toca `odd/tasks/f.md`
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** el resultado incluye un aviso por ese commit

#### Scenario: Sin nada relevante no hay aviso

- **DADO** el interruptor encendido y commits nuevos en `origin/<integración>` que no tocan nada relevante para el cambio actual
- **CUANDO** se ejecuta `axiom autofetch run`
- **ENTONCES** no se emite ningún aviso

#### Scenario: Proyecto no registrado o hub ilegible en `run`

- **DADO** un proyecto no registrado o un hub ilegible
- **CUANDO** se ejecuta `axiom autofetch run` a mano, y después `axiom autofetch run --agent claude-code`
- **ENTONCES** la primera ejecución informa de autofetch apagado con aviso y la segunda no emite ningún aviso ni salida
- **Y** ninguna hace red ni escribe, y ambas terminan con código 0

#### Scenario: Multirepo opera solo sobre el repositorio de specs

- **DADO** un multirepo con el interruptor encendido
- **CUANDO** se ejecuta `axiom autofetch run --json`
- **ENTONCES** el fetch y el avance afectan solo al repositorio de specs, in situ y sobre su rama de integración
- **Y** los repositorios de código no reciben ninguna operación de git remota

### Requirement: Disparo del Autofetch por Hook de Sesión y por la Doctrina (RF-BRA-40)

Con el interruptor encendido, el autofetch MUST dispararse en tres momentos: (1) al iniciar la sesión, mediante un hook propio de inicio de sesión que `axiom sync` instala en Claude Code y en Codex y que ejecuta `axiom autofetch run --agent <runtime>`; (2) antes de `axiom branch ensure` (RF-BRA-10); (3) antes de presentar cada parada de una puerta de revisión, por doctrina del orquestador, que MUST ejecutar `axiom autofetch run` (sujeto al intervalo mínimo de RF-BRA-39) y relayar sus avisos. En los runtimes sin canal de hook demostrado (Gemini, Cursor, Antigravity, Kiro, Pi y OpenCode por esta vía) el disparo de inicio de sesión MUST ser solo doctrinal.

La orden instalada del hook MUST ser fail-open: MUST terminar siempre con código 0, aunque `axiom` falte o falle; MUST NOT bloquear el arranque de la sesión; y MUST ser inerte fuera de un proyecto Axiom (un `axiom.yaml` propio o un puntero `.axiom-workspace`, en el directorio o en un ancestro, RF-BRA-23) y con el interruptor apagado: no lanza procesos de git, no hace red y no escribe. Los avisos de `run` MUST emitirse por el canal de contexto adicional del runtime, y solo cuando hay algo relevante. La instalación MUST ser idempotente y determinista, conservar las entradas de hooks ajenas y no alterar el hook de `axiom branch guard`; una ejecución posterior de `axiom sync` que ya no deba instalarlo, y la desinstalación, MUST retirarlo. La instalación del hook MUST NOT depender del estado del interruptor (el hook es inerte cuando está apagado) y MUST NOT instalar hooks de git.

La regla de doctrina de los textos del orquestador (y de las guías `AGENTS.md`, `GEMINI.md` y del manual de inicio) MUST pasar a decir que el `--fetch` exige la autorización explícita de la persona, salvo que la persona haya encendido el autofetch.

#### Scenario: `axiom sync` instala el hook de sesión en Claude Code y Codex

- **DADO** un proyecto con Claude Code y Codex configurados
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** la configuración de cada uno incluye una entrada de inicio de sesión que ejecuta `axiom autofetch run --agent <runtime>`

#### Scenario: Sincronizar dos veces no duplica y conserva lo ajeno

- **DADO** la instalación anterior y un hook de inicio de sesión propio del usuario
- **CUANDO** se ejecuta `axiom sync` por segunda vez
- **ENTONCES** la configuración de cada runtime es byte a byte idéntica a la anterior
- **Y** el hook del usuario y la entrada de `axiom branch guard` permanecen sin alterar

#### Scenario: La desinstalación retira el hook

- **DADO** el hook de inicio de sesión instalado
- **CUANDO** se desinstala Axiom, o `axiom sync` ya no debe instalarlo
- **ENTONCES** la entrada de `axiom autofetch run` desaparece de la configuración del runtime
- **Y** las entradas ajenas permanecen

#### Scenario: El hook termina siempre en 0 y no bloquea

- **DADO** el hook instalado, un remoto inalcanzable o `axiom` ausente del `PATH`
- **CUANDO** el runtime inicia la sesión
- **ENTONCES** la orden del hook termina con código 0
- **Y** el `run` completo termina en unos 10 segundos como máximo (fetch acotado a unos 8 segundos más lecturas locales y avance) y el arranque de la sesión no espera más que esa cota

#### Scenario: El hook es inerte fuera de proyectos Axiom

- **DADO** el hook instalado en la configuración global y un repositorio sin `axiom.yaml` ni puntero `.axiom-workspace` resoluble
- **CUANDO** se inicia la sesión en ese repositorio
- **ENTONCES** el hook no lanza procesos, no hace red, no avisa y no escribe nada

#### Scenario: El hook es inerte con el interruptor apagado

- **DADO** el hook instalado en un proyecto Axiom con el interruptor apagado
- **CUANDO** se inicia la sesión
- **ENTONCES** no se ejecuta ninguna operación de red ni se emite ningún aviso

#### Scenario: Los avisos llegan por el contexto adicional solo si hay algo relevante

- **DADO** el interruptor encendido y un commit entrante relevante para el cambio actual
- **CUANDO** el hook ejecuta `axiom autofetch run --agent claude-code`
- **ENTONCES** el aviso se emite por el canal de contexto adicional del runtime
- **Y** sin nada relevante no se emite ninguna salida de contexto

#### Scenario: Los runtimes sin hook demostrado quedan solo con la doctrina

- **DADO** Gemini, Cursor, Antigravity, Kiro, Pi u OpenCode configurados
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** no se instala ningún hook de inicio de sesión de autofetch en ellos
- **Y** la ejecución termina con normalidad

#### Scenario: La doctrina condiciona la regla de `--fetch` al autofetch

- **DADO** los textos generados para el orquestador, `AGENTS.md`, `GEMINI.md` y el manual de inicio
- **CUANDO** se inspecciona la regla sobre `--fetch`
- **ENTONCES** exige la autorización explícita de la persona salvo que haya encendido el autofetch
- **Y** la doctrina indica ejecutar `axiom autofetch run` antes de presentar cada parada

## MODIFIED Requirements

### Requirement: Familia de Comandos `axiom branch` (RF-BRA-01)

La CLI DEBE exponer la familia de primer nivel `axiom branch` con exactamente los subcomandos `ensure`, `status`, `scope`, `done` y `guard`. La familia DEBE ser neutral respecto al carril: `ensure`, `status` y `done` DEBEN aceptar `--change <nombre>` y `--lane sdd|odd` (por defecto `sdd`) y NO DEBEN depender de que exista un cambio SDD; `scope` DEBE aceptar `--change <nombre>` y solo opera sobre un cambio SDD con kickoff sellado (RF-BRA-35). Ningún subcomando de `axiom branch` DEBE ejecutar `git push` ni `git commit`, crear un PR ni realizar operaciones remotas salvo la solicitada por `--fetch` o la que autoriza el interruptor de autofetch encendido (RF-BRA-10), que `scope` no admite. La familia NO DEBE exponer subcomandos `pr`, `push` ni ningún comprobador para la CI; el verbo `axiom autofetch` (RF-BRA-39) NO pertenece a esta familia y no amplía sus cinco subcomandos. Toda ejecución de git del motor DEBE pasar por un único punto, de modo que la ausencia de operaciones remotas sea verificable con un observador instalado en ese punto y con una prueba de arquitectura que cubra el código de `internal/branch/**` y de `internal/gitcommon`. La operación de subir documentación del cambio es el verbo `axiom sdd publish`, que NO pertenece a esta familia, usa un puerto de publicación propio al que `axiom branch` no puede llegar y no altera este invariante (RF-PUB-07 de `sdd-docs-publication`).
(Previously: la única operación remota permitida era la solicitada por `--fetch`; el verbo `axiom autofetch` no existía)

#### Scenario: La ayuda lista los cinco verbos
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch --help`
- **ENTONCES** la salida lista `ensure`, `status`, `scope`, `done` y `guard`
- **Y** no lista `pr`, `push` ni ningún comprobador para la CI
- **Y** no lista `autofetch`

#### Scenario: Un subcomando no listado no existe
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch` seguido de `push`, de `pr` o del verbo retirado `check`
- **ENTONCES** el comando se rechaza como no reconocido con código de salida distinto de cero

#### Scenario: Ningún subcomando publica ni abre PR
- **DADO** un repositorio con un remoto configurado, el interruptor apagado y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecutan `ensure`, `status`, `scope` y `done` sin `--fetch`
- **ENTONCES** el observador no registra ninguna invocación de `git push`, `git commit`, `git fetch`, `git pull` ni `git ls-remote`

#### Scenario: Con el interruptor encendido tampoco se publica
- **DADO** un repositorio con un remoto configurado, el interruptor encendido y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecutan `ensure`, `status`, `scope` y `done`
- **ENTONCES** el observador no registra ninguna invocación de `git push`, `git commit`, `git pull` ni `git ls-remote`

#### Scenario: `scope` no admite `--fetch`
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch scope --change x --fetch`
- **ENTONCES** la orden se rechaza como uso incorrecto con código de salida distinto de cero
- **Y** no se ha ejecutado ninguna operación de red

#### Scenario: `scope` no hace red con el interruptor encendido
- **DADO** el interruptor encendido y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom branch scope --change x`
- **ENTONCES** el observador no registra ninguna invocación de git remota

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

### Requirement: Base sin Red y `--fetch` Explícito (RF-BRA-10)

`ensure` NO DEBE realizar ninguna operación remota salvo que la persona la haya pedido de forma explícita: con `--fetch` en la orden o con el interruptor de autofetch encendido (RF-HUB-07, RF-HUB-13), que cuenta como opción explícita y persistente. Con el interruptor apagado y sin `--fetch`, ninguna orden de `axiom branch` hace red. La base de cada rama DEBE ser `origin/<base efectiva>` si esa referencia existe en local y, si no, la rama local de la base efectiva, donde la base efectiva es la rama de integración (en los repositorios de una sola etapa y en modo `release`) o la rama principal (en los repositorios con dos etapas en modo `branch`, RF-BRA-29); si la propia rama del cambio no existe en local pero existe su referencia remota local, la rama se adopta desde ella (RF-BRA-27). El JSON DEBE devolver, para cada repositorio, `base.ref`, `base.sha`, `base.source` (que indica si la base procede de la referencia remota local o de la rama local; la rama del cambio adoptada desde el remoto se indica aparte, en `branches[].createdFrom: remote` y `createdFromRef`, RF-BRA-27) y `fetched`, que DEBE ser `false` si no hubo fetch y `true` si lo hubo. Con `--fetch`, el comando PUEDE ejecutar un `fetch` previo, y entonces `fetched` DEBE ser `true`. Si `--fetch` falla, `ensure` DEBE terminar con `fetch_failed` sin crear nada (supuesto S-8). Con el interruptor encendido y sin `--fetch`, `ensure` DEBE ejecutar el fetch de forma implícita, antes de resolver la base, en todos los repositorios del plan (también los de código, y el repositorio de specs in situ de un multirepo, donde solo hay fetch y `ensure` no avanza nada), con un tiempo máximo de unos 8 segundos por repositorio, omitiendo sin aviso (`fetched: false`) los repositorios sin remoto `origin`, también cuando se pasa `--base <ref>` (el fetch implícito se hace igual), sin respetar el intervalo mínimo de RF-BRA-39 y sin avanzar ninguna rama local; el fetch implícito se aplica también a `ensure --archive` (RF-BRA-13) y NO a `done`. Si el fetch implícito falla, `ensure` NO DEBE rechazar: DEBE avisar del fallo, continuar con la base local resuelta como si no hubiera fetch y devolver `fetched: false` para ese repositorio. Si no existe la base efectiva ni local ni como referencia remota local, DEBE rechazar con `base_not_found`, nombrando la rama esperada. `--base <ref>` sustituye la base (RF-BRA-12). El `--fetch` explícito sigue exigiendo la autorización de la persona, salvo que haya encendido el autofetch.
(Previously: la única vía de red era `--fetch`; no existía el interruptor ni el fetch implícito, y `fetched` solo era `true` con `--fetch`)

#### Scenario: Sin `--fetch` y con el interruptor apagado no hay operaciones remotas
- **DADO** un repositorio con remoto, el interruptor apagado y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el observador no registra ninguna invocación remota (`fetch`, `pull`, `push`, `ls-remote`)
- **Y** el JSON devuelve `fetched: false`

#### Scenario: La base es la referencia remota local si existe
- **DADO** un repositorio con `origin/main` y `main` locales en commits distintos
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `origin/main`, `base.sha` es el de `origin/main` y `base.source` indica la referencia remota local

#### Scenario: Sin referencia remota la base es la rama local
- **DADO** un repositorio sin remoto
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `main`, `base.sha` es el de `main` y `base.source` indica la rama local
- **Y** `fetched` es `false`

#### Scenario: `--fetch` es explícito
- **DADO** un repositorio con remoto
- **CUANDO** se ejecuta `ensure --change x --fetch`
- **ENTONCES** se ejecuta un `fetch` antes de resolver la base y el JSON devuelve `fetched: true`

#### Scenario: `--fetch` fallido no deja efectos
- **DADO** un remoto inalcanzable
- **CUANDO** se ejecuta `ensure --change x --fetch`
- **ENTONCES** se rechaza con `fetch_failed`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Con el interruptor encendido `ensure` hace fetch implícito
- **DADO** el interruptor encendido y un repositorio con remoto cuyo `origin/main` tiene commits que las referencias locales no tienen
- **CUANDO** se ejecuta `ensure --change x --json` sin `--fetch`
- **ENTONCES** se ejecuta un `fetch` antes de resolver la base
- **Y** el JSON devuelve `fetched: true`
- **Y** `base.sha` es el de `origin/main` actualizado

#### Scenario: El fetch implícito alcanza también a los repositorios de código sin mover sus ramas
- **DADO** un multirepo con el interruptor encendido y repositorios de código con commits nuevos en su remoto
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** cada repositorio del plan recibe un `fetch` y su JSON devuelve `fetched: true`
- **Y** ninguna rama local preexistente de los repositorios de código cambia de commit

#### Scenario: El fetch implícito incluye el repositorio de specs in situ de un multirepo

- **DADO** un multirepo con el interruptor encendido y commits nuevos en el remoto del repositorio de specs
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el repositorio de specs recibe un `fetch` y su JSON devuelve `fetched: true`
- **Y** ninguna rama de ese repositorio se avanza desde `ensure`

#### Scenario: Un repositorio sin remoto `origin` se omite sin aviso

- **DADO** el interruptor encendido y un repositorio del plan sin remoto `origin`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** ese repositorio no recibe `fetch`, devuelve `fetched: false` y no se emite ningún aviso por ello

#### Scenario: Con `--base <ref>` el fetch implícito se hace igual

- **DADO** el interruptor encendido y un repositorio con remoto
- **CUANDO** se ejecuta `ensure --change x --base <ref> --json`
- **ENTONCES** se ejecuta un `fetch` antes de resolver la base y `fetched` es `true`

#### Scenario: El fetch implícito tiene tiempo máximo por repositorio

- **DADO** el interruptor encendido y un remoto que no responde en un repositorio del plan
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el fetch de ese repositorio se corta en unos 8 segundos, se avisa del fallo y `fetched` es `false`

#### Scenario: El fetch implícito no respeta el intervalo mínimo
- **DADO** el interruptor encendido y un fetch correcto realizado hace menos del intervalo mínimo
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se ejecuta igualmente un `fetch` y `fetched` es `true`

#### Scenario: El fetch implícito fallido avisa y sigue con la base local
- **DADO** el interruptor encendido y un remoto inalcanzable, con `origin/main` presente en las referencias locales
- **CUANDO** se ejecuta `ensure --change x --json` sin `--fetch`
- **ENTONCES** la orden no se rechaza y crea las ramas sobre la base local resuelta
- **Y** el resultado incluye un aviso del fallo del fetch
- **Y** `fetched` es `false`

#### Scenario: El `--fetch` explícito fallido conserva el rechazo con el interruptor encendido
- **DADO** el interruptor encendido y un remoto inalcanzable
- **CUANDO** se ejecuta `ensure --change x --fetch`
- **ENTONCES** se rechaza con `fetch_failed`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: `done` no hace fetch implícito con el interruptor encendido
- **DADO** el interruptor encendido y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom branch done --change x` sin `--fetch`
- **ENTONCES** el observador no registra ninguna invocación de `git fetch`

#### Scenario: Rama de integración inexistente
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y un repositorio sin `develop` ni `origin/develop`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `base_not_found` nombrando `develop`

#### Scenario: La rama del cambio publicada se indica con `createdFrom: remote`
- **DADO** un repositorio con `refs/remotes/origin/inc/x` y sin rama local `inc/x`, y el interruptor apagado
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** la entrada de `inc/x` en `branches[]` tiene `createdFrom: remote` y `createdFromRef: refs/remotes/origin/inc/x`
- **Y** `base.source` y `base.ref` siguen describiendo la base de integración
- **Y** `fetched` es `false`

#### Scenario: En modo `branch` la base es la referencia remota local de la rama principal
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch`, el interruptor apagado, y `origin/master` y `origin/develop` presentes en local
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `origin/master`, `base.sha` es el de `origin/master` y `base.source` indica la referencia remota local
- **Y** `fetched` es `false`

### Requirement: Rama de Archivado y `ensure --archive` (RF-BRA-13)

En los repositorios con rama de cambio (`monorepo-embedded`), el archivado de un cambio DEBE hacerse en una rama propia, la rama de archivado `<prefijo>/<cambio>-archive`, y entregarse en un PR documental aparte, **después** de fusionar el PR del cambio (RF-ARC-07). `axiom branch ensure --change <c> --archive` DEBE crearla en el repositorio del planning home, saliendo de la rama de integración con la base de RF-BRA-10 (`origin/<integración>` si existe en local y, si no, la rama de integración local), y aplicando el modo de trabajo personal de RF-BRA-08 (en modo `worktree`, un worktree en `<raíz de worktrees>/<cambio>-archive/<repo>`; en `in_place`, el checkout de la rama en el propio repositorio, sujeto a RF-BRA-09). `--archive` admite `--prefix` (RF-BRA-12). La rama de archivado sale siempre de la rama de integración y espera solo a que el cambio esté fusionado en ella, en cualquier modo de promoción: NUNCA espera a la fusión en la rama principal ni usa la base efectiva de RF-BRA-29.

Si la base resuelta no contiene ya el cambio fusionado, es decir, si su árbol no incluye la carpeta `changes/<cambio>/` del planning home, o si la incluye pero el cambio está en fusión intermedia (RF-BRA-26), `ensure --archive` DEBE rechazar con `change_not_integrated`, sin crear nada, nombrando la base evaluada y sugiriendo `--fetch` cuando la carpeta falta y el interruptor de autofetch está apagado; con el interruptor encendido el fetch ya se ha intentado y el mensaje NO DEBE sugerir `--fetch` como remedio. Cuando el rechazo se debe a una fusión intermedia, el mensaje DEBE indicarlo y NO DEBE sugerir `--fetch` como remedio, porque la base ya contiene la versión integrada. NO hay `fetch` implícito con el interruptor apagado: solo con `--fetch` explícito se actualiza la referencia remota local antes de evaluar (RF-BRA-10); con el interruptor encendido, `ensure --archive` hace el fetch implícito de RF-BRA-10. `ensure --archive` es idempotente: reutiliza la rama y el worktree existentes sin moverlos, y NO publica la rama. En `multirepo` NO existe rama de archivado: `--archive` DEBE rechazarse con `archive_branch_not_applicable` porque el archive se hace in situ en la rama de integración del repositorio de specs (RF-BRA-05); tampoco existe en el carril ODD (`--lane odd`), que no tiene archive. `done` también cubre la rama de archivado (RF-BRA-17 y RF-BRA-18) y `status` la incluye en el plan (RF-BRA-16).
(Previously: no había fetch implícito en ningún caso y `--fetch` se sugería siempre que la carpeta del cambio faltaba en la base)

#### Scenario: Crea la rama de archivado desde la integración con el cambio fusionado
- **DADO** un monorepo en modo `worktree`, el cambio `x` fusionado en `origin/main` (referencia local), la política por defecto y el interruptor apagado
- **CUANDO** se ejecuta `ensure --change x --archive --json`
- **ENTONCES** existe la rama `feat/x-archive` creada desde `origin/main`
- **Y** existe su worktree en `<raíz de worktrees>/x-archive/<repo>`
- **Y** `base.ref` es `origin/main` y `fetched` es `false`

#### Scenario: La base local sin el cambio fusionado se rechaza y sugiere `--fetch`
- **DADO** un monorepo con el interruptor apagado cuya rama de integración local y cuya referencia remota local no contienen `openspec/changes/x/`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `change_not_integrated` nombrando la base evaluada
- **Y** el mensaje sugiere `--fetch`
- **Y** no se crea ninguna rama ni worktree
- **Y** no consta ninguna operación remota

#### Scenario: Una fusión intermedia se rechaza con `change_not_integrated`
- **DADO** un monorepo cuya base contiene `openspec/changes/x/` en fusión intermedia (por ejemplo, solo `spec` y `design` aprobadas y cedidas)
- **CUANDO** se ejecuta `ensure --change x --archive` con el interruptor apagado
- **ENTONCES** se rechaza con `change_not_integrated` nombrando la base evaluada
- **Y** el mensaje indica que la versión integrada es una fusión intermedia y no sugiere `--fetch` como remedio
- **Y** no se crea ninguna rama ni worktree
- **Y** no consta ninguna operación remota

#### Scenario: Con todos los role-apply aprobados e integrados, `--archive` procede
- **DADO** un monorepo cuya base contiene `openspec/changes/x/` con todos los `role-apply` aprobados, aunque alguna puerta anterior lleve marca de cesión
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea la rama de archivado como en el primer escenario

#### Scenario: `--fetch` explícito permite alcanzar el cambio fusionado
- **DADO** el mismo monorepo, con el cambio fusionado ya en el remoto pero no en las referencias locales
- **CUANDO** se ejecuta `ensure --change x --archive --fetch`
- **ENTONCES** se ejecuta un `fetch` antes de evaluar la base y el JSON devuelve `fetched: true`
- **Y** se crea la rama `feat/x-archive`

#### Scenario: Con el interruptor encendido `ensure --archive` hace fetch implícito
- **DADO** el interruptor encendido y un monorepo con el cambio fusionado ya en el remoto pero no en las referencias locales
- **CUANDO** se ejecuta `ensure --change x --archive --json` sin `--fetch`
- **ENTONCES** se ejecuta un `fetch` antes de evaluar la base y el JSON devuelve `fetched: true`
- **Y** se crea la rama `feat/x-archive`

#### Scenario: Con el interruptor encendido el rechazo no sugiere `--fetch`
- **DADO** el interruptor encendido, un fetch implícito correcto y un monorepo cuyo remoto tampoco contiene `openspec/changes/x/`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `change_not_integrated` nombrando la base evaluada
- **Y** el mensaje no sugiere `--fetch` como remedio
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Multirepo no tiene rama de archivado
- **DADO** un multirepo con el repositorio de specs en su rama de integración
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `archive_branch_not_applicable`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: El carril ODD no tiene rama de archivado
- **DADO** un monorepo
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature --archive`
- **ENTONCES** se rechaza con `archive_branch_not_applicable`

#### Scenario: Modo `in_place` hace checkout de la rama de archivado
- **DADO** un proyecto en modo `in_place` con el árbol limpio y el cambio `x` fusionado en la base
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** el repositorio queda desplegado en `feat/x-archive`
- **Y** no se crea ningún worktree

#### Scenario: Modo `in_place` con árbol sucio se rechaza
- **DADO** un proyecto en modo `in_place` con un fichero modificado sin confirmar y el cambio `x` fusionado en la base
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `dirty_worktree` sin cambiar de rama

#### Scenario: Idempotencia y ausencia de publicación
- **DADO** la rama de archivado ya creada por `ensure --archive`
- **CUANDO** se repite `ensure --change x --archive`
- **ENTONCES** el JSON es idéntico, el commit de la rama no cambia y no se ejecuta ningún `git push`

#### Scenario: Con dos etapas en modo `branch` el archivado espera solo a la integración
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y el cambio `x` fusionado en `develop` pero no en `master`
- **CUANDO** se ejecuta `ensure --change x --archive --json`
- **ENTONCES** `feat/x-archive` se crea desde `origin/develop` y `base.ref` es `origin/develop`
- **Y** no se devuelve `change_not_integrated`
