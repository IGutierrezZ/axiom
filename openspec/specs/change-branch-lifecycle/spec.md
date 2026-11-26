# Especificación de Requerimientos: Ciclo de Vida de las Ramas y Worktrees de un Cambio por Topología (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** capacidad nueva
> **Idioma:** Español (castellano peninsular)
> **Reapertura (2026-10-17):** la spec se reabrió por las decisiones D16 a D19 y las lecturas del design aceptadas (G-1, G-4 a G-7 y G-9, A-1 y A-3). D16 sustituye a D15, que queda retirada: en monorepo el archive se hace en un PR documental aparte. D17 elimina la capa C (el comprobador de CI y el invariante de la rama de integración). D18 define la rama de archivado. D19 enmienda la guarda D.

## Propósito

Definir cómo crea, consulta, cierra y protege Axiom las ramas (y, opcionalmente, los worktrees) de un cambio, de forma determinista y común a todos los runtimes. Hasta ahora la política de ramas era prosa de agente («crea una rama si estás en la rama por defecto»), el worktree era opcional y llegaba tarde (en apply), las specs se escribían «siempre en main» y no existía configuración ni guarda mecánica. Esta capacidad convierte esa doctrina en un motor de la CLI, la familia `axiom branch ensure|status|done|guard`, que:

- crea la rama del cambio y, según la preferencia personal, su worktree **al nombrar el cambio y antes de cualquier escritura en su carpeta**;
- decide el modelo de ramas por la **topología** y por el **número de roles del cambio que tocan cada repositorio**;
- crea, cuando el PR del cambio ya está fusionado, la **rama de archivado** donde se hace el archive y que se entrega en un PR documental aparte;
- impide trabajar en la rama equivocada mediante tres capas de guarda (escritores Go, `axiom sdd status` y hook opcional);
- se comporta igual en Claude Code, Codex, OpenCode, Pi y el resto de runtimes, sin depender de primitivas propias de uno de ellos.

**Principio rector:** las ramas se crean siempre; los worktrees son una preferencia personal. La política del proyecto (rama de integración, prefijo, repositorios) se versiona en `axiom.yaml`; cómo trabaja cada persona en su máquina (modo de trabajo y raíz de los worktrees) vive en el hub personal.

**Resultado esperado:** una sola orden deja creada la rama del cambio y, según la preferencia, su worktree, y devuelve rutas absolutas que todas las órdenes posteriores usan con `--cwd`; ninguna escritura de Axiom en la carpeta de un cambio ocurre fuera de la rama que le corresponde; y los cambios ya fusionados que aún no se han archivado se muestran como pendientes de archivar en `axiom branch status` y en el dashboard, sin bloquear nada.

## Términos

| Término | Definición |
|---|---|
| Rama de integración | Rama a la que se integra el trabajo. Valor de `git.integration_branch` (por defecto `main`), con posible valor propio por repositorio (RF-BRA-03). |
| Rama principal | Valor de `git.main_branch`, declarado solo en el bloque `git:` (sin valor por repositorio ni por defecto). Su ausencia, o un valor igual a la rama de integración, significa una sola etapa (RF-BRA-03, RF-BRA-28). |
| Modo de promoción | Valor `branch`, `release` o `ask`: desde qué rama sale la rama del cambio y contra cuál se confirma su fusión en `done`. Se resuelve por repositorio y por cambio (RF-BRA-28). |
| Prefijo | Valor de `git.branch_prefix` (por defecto `feat`), con posible valor propio por repositorio (RF-BRA-03). |
| Rama del cambio | `<prefijo>/<cambio>`. |
| Rama de rol | `<prefijo>/<cambio>-<rol>`. Solo existe cuando dos o más roles del cambio tocan el mismo repositorio (RF-BRA-04). |
| Rama de seguimiento | La rama del cambio cuando existen ramas de rol sobre ese repositorio: los PR de rol se dirigen contra ella. |
| Rama de archivado | `<prefijo>/<cambio>-archive`. Solo existe en los repositorios con rama de cambio (`monorepo-embedded`), la crea `axiom branch ensure --change <c> --archive` desde la rama de integración una vez fusionado el PR del cambio, y en ella se ejecuta `archive` (RF-BRA-13). No existe en `multirepo`. |
| Cambio pendiente de archivar | Cambio activo presente en el árbol de la rama de integración de un repositorio con rama de cambio, es decir, ya fusionado y aún sin archivar. Se excluyen por nombre los sumideros `e2e-cumulative` y `cumulative-<rol>` (RF-BRA-21) y las fusiones intermedias, es decir, los cambios en fusión intermedia (RF-BRA-26). |
| Raíz git | Directorio de nivel superior del repositorio git que contiene una ruta (`git rev-parse --show-toplevel`), con su git common dir (`git rev-parse --git-common-dir`). |
| Repositorio con rama de cambio | En `monorepo-embedded`, cada repositorio git tocado por el cambio (incluido el del planning home). En `multirepo`, cada repositorio de código tocado por algún rol del cambio. |
| Repositorio de specs en multirepo | En `multirepo`, el repositorio que contiene el planning home (`planning-home-resolution`). Se trabaja siempre in situ, en su rama de integración. |
| Repositorio de código en multirepo | Cualquier otro repositorio declarado por los roles en `multirepo`. |
| Roles del cambio sobre un repositorio | Intersección entre los roles sellados en el `kickoff.yaml` del cambio y los roles de `axiom.yaml` cuyo `repositories` resuelve a esa raíz git. El rol reservado `fullstack` cuenta como un único rol que toca todos los repositorios que declara algún rol de `axiom.yaml` y, en `monorepo-embedded`, el repositorio del planning home (supuesto S-3). |
| Modo de trabajo | Preferencia personal por proyecto: `worktree` (por defecto), `in_place` o `ask`. Vive en el hub, no en `axiom.yaml`. |
| Raíz de worktrees | Directorio personal donde se crean los worktrees. Por defecto `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`, fuera de cualquier repositorio. |
| Registro compartido | Instantánea de política escrita una sola vez por `kickoff seal` en `kickoff.yaml` (RF-KOF-06). |
| Registro local | Estado local no versionado, en el git common dir del repositorio del planning home (RF-BRA-15). |
| Instantánea | Entradas de política del registro compartido: integración, prefijo, base por repositorio y nombres de rama derivados. |
| Veredicto de rama | Respuesta única a «¿esta rama es la que corresponde a este cambio en este repositorio según su papel?» (RF-BRA-19). |
| Carril | `sdd` (por defecto) u `odd`. |

### Tokens y etiquetas del JSON

Todo valor de estado o de clasificación del JSON es un **token estable en inglés** acompañado, en un campo aparte, de una **etiqueta legible en castellano**. Los tokens no se traducen ni cambian con el idioma; los mensajes y las etiquetas están en castellano (RF-BRA-02).

| Token | Etiqueta | Dónde |
|---|---|---|
| `not_started` | no iniciado | estado de un cambio sin ramas (RF-BRA-16) |
| `correct` | correcto | veredicto de rama (RF-BRA-19) |
| `not_applicable` | no aplicable | veredicto de rama (RF-BRA-19) |
| `ancestry` | ascendencia | capa 1 de la detección de fusión (RF-BRA-17) |
| `equivalent_tree` | árbol equivalente | capa 2 de la detección de fusión (RF-BRA-17) |
| `pr_merged` | PR fusionado | capa 3 de la detección de fusión (RF-BRA-17) |
| `forced` | forzado | confirmación con `--force` (RF-BRA-18) |

### Códigos tipados

Los resultados que no son «listo» se identifican por un código estable, nunca por el texto del mensaje.

| Código | Significado |
|---|---|
| `decision_required` | El modo de trabajo o el modo de promoción es `ask` y no hay respuesta registrada: no se ha creado nada (RF-BRA-08, RF-BRA-29). |
| `dirty_worktree` | Cambios sin confirmar impiden la operación (RF-BRA-09, RF-BRA-18). |
| `wrong_branch` | La rama actual no es la que corresponde (RF-BRA-19). |
| `specs_repo_not_on_integration` | En multirepo, el repositorio de specs no está en su rama de integración (RF-BRA-05). |
| `base_not_found` | No existe la rama de integración, ni en local ni como referencia remota local (RF-BRA-10). |
| `fetch_failed` | `--fetch` fue solicitado y falló (RF-BRA-10). |
| `path_too_long` | La ruta de un worktree supera el límite admitido (RF-BRA-11). |
| `invalid_name` | El nombre del cambio, el prefijo o la rama indicados en la invocación no son válidos (RF-BRA-03, RF-BRA-11). |
| `config_invalid` | `axiom.yaml` o el hub personal no se pueden leer, o su política es inválida, y impiden planificar (RF-BRA-03). |
| `git_failed` | Una invocación de git falló por una causa que ningún otro código cubre; el JSON nombra la operación y resume el error (RF-BRA-02). |
| `plan_conflict` | Una invocación contradice el plan ya materializado del cambio: un modo de trabajo o de promoción distinto del materializado (RF-BRA-08, RF-BRA-12, RF-BRA-13, RF-BRA-29). |
| `branch_checked_out_elsewhere` | La rama ya está desplegada en un worktree que el registro no conoce (RF-BRA-07). |
| `not_a_git_repository` | Una ruta declarada no pertenece a ningún repositorio git; se omite con aviso (RF-BRA-06). |
| `merge_not_confirmed` | Ninguna capa confirma la fusión de la rama (RF-BRA-17). |
| `change_not_integrated` | `ensure --archive`: la base local no contiene el cambio ya fusionado; sugiere `--fetch` (RF-BRA-13). |
| `archive_branch_not_applicable` | `ensure --archive` en una topología o carril sin rama de archivado (RF-BRA-13). |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| RF-BRA-01 | Familia `axiom branch` | Cuatro verbos de primer nivel, neutrales respecto al carril; ni `push` ni PR; sin operaciones remotas verificables salvo `--fetch` o el autofetch encendido (RF-BRA-10); `axiom autofetch` es un verbo aparte (RF-BRA-39) |
| RF-BRA-02 | Contrato de salida común | JSON con rutas absolutas, `--cwd`, tokens en inglés con etiqueta en castellano y código de salida «listo = 0» |
| RF-BRA-03 | Resolución de la política | Excepción por cambio > repositorio > `git:` > defecto |
| RF-BRA-04 | Modelo de ramas por número de roles | 1 rol: `<prefijo>/<cambio>`; 2 o más: seguimiento más ramas de rol; rama de archivado aparte |
| RF-BRA-05 | Papel del repositorio según topología | Repositorio de specs in situ en multirepo; código con rama de cambio |
| RF-BRA-06 | Raíz git de cada repositorio | Subcarpetas, `.`, repositorios compartidos y rutas no git |
| RF-BRA-07 | `ensure`: plan e idempotencia | Mismo plan, mismo resultado, sin duplicar |
| RF-BRA-08 | `ensure`: modo de trabajo | `worktree`, `in_place` y `ask` tipado; el checkout principal no se toca |
| RF-BRA-09 | `ensure` en `in_place` con árbol sucio | Rechazo sin modificar el repositorio |
| RF-BRA-10 | Base sin red | Sin `fetch` implícito con el autofetch apagado; encendido, fetch implícito acotado que avisa y sigue si falla; `base.source`, `sha` y `fetched` |
| RF-BRA-11 | Rutas de worktree y longitud | `<raíz>/<cambio>[-<rol>\|-archive]/<repo>`; rechazo por longitud (220 en Windows) |
| RF-BRA-12 | Excepciones por cambio | `--prefix`, `--branch`, `--base`; registradas y sin contradicción |
| RF-BRA-13 | Rama de archivado y `ensure --archive` | `<prefijo>/<cambio>-archive` desde la integración; `change_not_integrated` sin fetch implícito salvo con el autofetch encendido |
| RF-BRA-14 | Carril ODD | Misma política para `odd/tasks/` |
| RF-BRA-15 | Registro local | Rutas absolutas y respuesta a `ask` en el git common dir |
| RF-BRA-16 | `status` | Estado de ramas y worktrees del cambio, solo con referencias locales |
| RF-BRA-17 | `done`: detección de fusión por capas | `ancestry`, `equivalent_tree`, `pr_merged` |
| RF-BRA-18 | `done`: seguridad y eliminación de worktree y rama local | Elimina worktree y rama local tras fusión confirmada; rechazo con árbol sucio o fusión no confirmada salvo `--force`; nunca toca ramas remotas |
| RF-BRA-19 | Veredicto único de rama | Una sola consulta, por papel del repositorio y contexto (ordinario o de archivado) |
| RF-BRA-20 | Guarda A: escritores Go | Todo escritor de la carpeta del cambio se niega fuera de rama |
| RF-BRA-21 | Cambios pendientes de archivar | Visibles en `status` y en el dashboard, sin fusiones intermedias (RF-BRA-26); sin código de fallo ni integración con la CI |
| RF-BRA-22 | `guard`: hook opcional | Bloquea documentos de Axiom y avisa sin bloquear en el resto; lee HEAD sin lanzar git; fail-open |
| RF-BRA-23 | Instalación del hook por `axiom sync` | Claude Code, Codex y OpenCode; sin hooks de git |
| RF-BRA-24 | Skills y doctrina delegan en el verbo | Retirada de ODD-5.x y del Paso 2c |
| RF-BRA-25 | Dogfooding en este repositorio | `git:` declarado (`integration_branch: main`, `branch_prefix: inc`) y documentación remitida a `axiom branch` |

## No cubre esta capacidad

- `push` ni creación de PR desde `axiom branch`: siguen siendo decisiones de la persona.
- Hooks de git (`pre-commit`, `pre-push`, etc.) en cualquier repositorio, incluidos los de terceros.
- Operaciones remotas por defecto: ninguna orden hace `fetch`, `pull`, `push` ni consulta remota sin una opción explícita (RF-BRA-10); el interruptor personal de autofetch encendido cuenta como opción explícita y persistente, y solo autoriza `fetch` y el avance por fast-forward de RF-BRA-39, nunca `pull`, `push` ni `prune`.
- Un comprobador para la CI ni un código de fallo por cambios activos en la rama de integración: con el archive en un PR documental aparte, es normal que la rama de integración contenga cambios activos de forma temporal (RF-BRA-21).
- Un escritor Go del documento ODD `odd/tasks/<feature>.md`: ODD sigue siendo doctrina de agente y queda cubierto solo por la doctrina y por la guarda D (RF-BRA-14).
- Edición de `axiom.yaml` desde el dashboard (`local-web-dashboard`).
- Cobertura del hook en Pi: no hay canal demostrado y la capa D es de mejor esfuerzo (RF-BRA-23).
- La eliminación de ramas remotas: `done` elimina el worktree y la rama local de un cambio ya fusionado, pero nunca toca ramas remotas (RF-BRA-18).
- La migración de cambios preexistentes: se eximen, no se migran (RF-KOF-07 de `sdd-lane-kickoff`).

---

## 1. Capacidad: `change-branch-lifecycle`

El motor de ramas planifica de forma pura (topología, política, roles y preferencias personales producen un plan determinista) y ejecuta el plan con git de forma idempotente. Las guardas consultan el mismo veredicto.

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

### Requirement: Contrato de Salida Común con Rutas Absolutas (RF-BRA-02)

`ensure`, `status`, `scope` y `done` DEBEN admitir `--json` y `--cwd <ruta>`. La salida JSON DEBE incluir un estado tipado y, cuando no sea «listo», uno de los códigos tipados de esta capacidad. Todo valor de estado o de clasificación del JSON DEBE ser un token estable en inglés (`not_started`, `correct`, `not_applicable`, `ancestry`, `equivalent_tree`, `pr_merged`, `forced`, etc.) acompañado, en un campo aparte, de una etiqueta legible en castellano; los tokens NO DEBEN traducirse. Toda ruta de la salida (raíz git de cada repositorio, planning home, worktrees, raíz de worktrees) DEBE ser absoluta. El código de salida DEBE ser `0` únicamente cuando la operación ha dejado el estado pedido; todo rechazo, todo `decision_required` y todo error DEBEN terminar con código distinto de cero. Para `scope`, que es de solo lectura, «el estado pedido» es que el alcance se haya calculado: el código `0` se devuelve aunque haya repositorios `unverifiable`, que son información del resultado y no un fallo (RF-BRA-35). Los mensajes legibles DEBEN estar en español y nombrar la continuación. Una configuración ilegible o inválida DEBE terminar con `config_invalid` y una invocación de git que falle por una causa no cubierta por otro código con `git_failed`, que DEBE nombrar la operación y resumir el error. Ninguna fase posterior a `ensure` DEBE depender del directorio de trabajo heredado: las órdenes posteriores reciben `--cwd` con una ruta tomada de este JSON.
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
- **Y** una repetición de la orden, ya resuelta la causa, completa el plan sin duplicar nada (RF-BRA-07)

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

### Requirement: Resolución de la Política de Ramas (RF-BRA-03)

La política de cada repositorio DEBE resolverse con esta precedencia, de mayor a menor: (1) las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`, RF-BRA-12); (2) la entrada del repositorio en `repositories:` de `axiom.yaml`, si existe; (3) el bloque `git:` de `axiom.yaml`; (4) los valores por defecto: rama de integración `main` y prefijo `feat`. La política incorpora además la plantilla de nombre de rama (`branch_template`), con su propia precedencia de plantilla efectiva, rol (`roles.<id>.branch_template`) > repositorio (`repositories[].branch_template`) > `git:` (`git.branch_template`), definida en RF-BRA-44 y sin valor por defecto: su ausencia en todos los niveles significa los nombres de siempre. Sin bloque `git:`, `integration_branch` DEBE valer `main` y `branch_prefix` DEBE valer `feat`; sin `repositories:`, ningún repositorio tiene ajustes propios y todos heredan los valores del workspace. La política incorpora además la rama principal (`git.main_branch`, solo en `git:` y sin valor por repositorio ni por defecto: su ausencia significa una sola etapa) y el modo de promoción por defecto (`promotion`, con herencia `repositories:` > `git:`; el valor vacío hereda); el modo efectivo de cada repositorio y cambio se resuelve en RF-BRA-28. Los nombres de rama resultantes DEBEN ser nombres de referencia git válidos: un prefijo o nombre inválido indicado en la invocación (`--prefix`, `--branch`, `--change`) DEBE rechazarse con `invalid_name`, y un valor inválido (incluida una plantilla `branch_template` inválida) o una configuración ilegible en `axiom.yaml` o en el hub personal con `config_invalid`, en ambos casos sin crear nada. Una variable de plantilla sin valor y un nombre renderizado inválido se rechazan con los códigos tipados de RF-BRA-45.
(Previously: la política no incluía la rama principal ni el modo de promoción)
(Previously: la política no incluía la plantilla de nombre de rama ni sus rechazos tipados)

#### Scenario: Sin configuración se aplican los valores por defecto
- **DADO** un `axiom.yaml` sin bloque `git:` ni `repositories:`
- **CUANDO** se planifica el cambio `x`
- **ENTONCES** la rama de integración es `main` y la rama del cambio es `feat/x`

#### Scenario: El bloque `git:` sustituye los valores por defecto
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y `git.branch_prefix: inc`
- **CUANDO** se planifica el cambio `x`
- **ENTONCES** la rama de integración es `develop` y la rama del cambio es `inc/x`

#### Scenario: Un repositorio con ajustes propios prevalece sobre `git:`
- **DADO** un `axiom.yaml` con `git.integration_branch: main` y una entrada de `repositories:` para `repo-backend` con rama de integración `release`
- **CUANDO** se planifica un cambio que toca `repo-backend` y `repo-frontend`
- **ENTONCES** la base de `repo-backend` es `release` y la de `repo-frontend` es `main`

#### Scenario: La excepción por cambio prevalece sobre todo lo demás
- **DADO** la política anterior y la invocación `ensure --change x --prefix fix`
- **CUANDO** se planifica el cambio
- **ENTONCES** la rama del cambio es `fix/x` en todos los repositorios del plan

#### Scenario: Un prefijo inválido de la invocación se rechaza sin efectos
- **DADO** un `ensure` con `--prefix "mi prefijo//"`
- **CUANDO** se ejecuta
- **ENTONCES** se rechaza con el código tipado `invalid_name`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Un prefijo inválido de `axiom.yaml` se rechaza como configuración
- **DADO** un `axiom.yaml` con `git.branch_prefix: "inc//"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado `config_invalid`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: La política hereda la rama principal y el modo de promoción
- **DADO** un `axiom.yaml` con `git.main_branch: master`, `git.promotion: branch` y una entrada de `repositories:` para `repo-backend` con `promotion: release`
- **CUANDO** se resuelve la política de `repo-backend` y de `repo-frontend`
- **ENTONCES** ambos tienen la rama principal `master`
- **Y** el modo por defecto de `repo-backend` es `release` y el de `repo-frontend` es `branch`

#### Scenario: Sin `main_branch` la política no inventa una rama principal
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y sin `git.main_branch`
- **CUANDO** se resuelve la política
- **ENTONCES** la rama principal queda vacía y el repositorio es de una sola etapa

#### Scenario: La plantilla de `git:` entra en la política y nombra la rama del cambio
- **DADO** un `axiom.yaml` con `git.branch_prefix: inc` y `git.branch_template: "{prefix}/{ref}-{slug}"`, y el cambio `us-2525-alta-de-usuarios` con work item `2525`
- **CUANDO** se resuelve la política y se planifica el cambio
- **ENTONCES** la política declara esa plantilla y la rama del cambio es `inc/2525-alta-de-usuarios`

#### Scenario: Una plantilla inválida de `axiom.yaml` se rechaza como configuración
- **DADO** un `axiom.yaml` con `git.branch_template: "{prefix}/{desconocida}"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado `config_invalid` que nombra `git.branch_template`
- **Y** no se crea ninguna rama ni worktree


### Requirement: Modelo de Ramas por Número de Roles que Tocan Cada Repositorio (RF-BRA-04)

El modelo de ramas de cada repositorio con rama de cambio DEBE depender de cuántos roles del cambio lo tocan (roles sellados en el `kickoff.yaml` del cambio intersecados con los roles de `axiom.yaml` cuyos repositorios resuelven a esa raíz git):

- **Un rol** (lo habitual, también `fullstack`): solo existe la rama `<prefijo>/<cambio>`; el rol trabaja en ella y su PR se dirige a la rama de integración.
- **Dos o más roles** (solo posible en `multirepo`: en `monorepo-embedded` el único rol es `fullstack`, RF-TOP-04 de `workspace-topology`, y por tanto nunca hay rama de rol): `<prefijo>/<cambio>` es la rama de seguimiento y cada rol saca `<prefijo>/<cambio>-<rol>` desde ella. Con plantillas de nombre de rama (RF-BRA-44), el nombre de la rama de seguimiento sale de la plantilla efectiva del repositorio o de `git:` y el de la rama de rol, de `roles.<id>.branch_template` o, si no se declara, de `<rama de seguimiento>-<rol>`; la rama de seguimiento NO PUEDE usar `{role}` ni `{task_ref}` (RF-BRA-45). Hay un PR de cada rol contra la rama de seguimiento y un PR final a la rama de integración que lleva el cambio (código y artefactos), sin el archive. Se RECOMIENDA el merge commit, sin imponerlo.
- **Rama de archivado** (solo en repositorios con rama de cambio): una vez fusionado en la rama de integración el PR del cambio, el archive se hace en `<prefijo>/<cambio>-archive`, que sale de la rama de integración (RF-BRA-13) y se entrega en un PR documental aparte contra la rama de integración. NO existe en `multirepo`.

El modelo DEBE evaluarse por repositorio: en un mismo cambio un repositorio puede tener rama de rol y otro no. Un rol cuyo nombre sea `archive` colisionaría con la rama de archivado y DEBE rechazarse con `invalid_name` al planificar las ramas de rol (supuesto S-15). `ensure` NO DEBE publicar la rama de seguimiento: DEBE avisar de que la persona la sube al remoto antes del primer PR de rol. La rama destino propuesta para cada PR (rol contra seguimiento, cambio contra integración, archivado contra integración) DEBE poder obtenerse del plan (`status`, RF-BRA-16) para que `sdd-apply` la proponga en lugar de preguntarla.
(Previously: el PR final a la rama de integración llevaba el archive y no existía rama de archivado; los escenarios de ramas de rol, de la colisión con `archive` y del destino de los PR se ejecutaban sobre un monorepo con los roles `core` y `qa`, y ahora se ejecutan sobre `multirepo` o sobre el rol `fullstack`)
(Previously: los nombres de la rama de seguimiento y de la rama de rol eran siempre `<prefijo>/<cambio>` y `<prefijo>/<cambio>-<rol>`)

#### Scenario: Un rol, rama única
- **DADO** un monorepo con el rol único `fullstack` sellado en el kickoff del cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** existe solo la rama `feat/x`
- **Y** no existe ninguna rama `feat/x-<rol>`

#### Scenario: Dos roles sobre el mismo repositorio crean seguimiento y ramas de rol
- **DADO** un multirepo con los roles `core` y `qa` sellados en el kickoff del cambio `x`, ambos con repositorios que resuelven a la misma raíz git de código `repo-backend`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** en `repo-backend` existe la rama de seguimiento `feat/x` desde la base
- **Y** existen `feat/x-core` y `feat/x-qa`, ambas creadas desde `feat/x`
- **Y** la salida avisa de que la rama de seguimiento debe publicarla la persona antes del primer PR de rol
- **Y** no se ejecuta ningún `git push`

#### Scenario: En multirepo cada repositorio de código con un rol usa la rama del cambio
- **DADO** un multirepo con el rol `backend` sobre `repo-backend` y el rol `frontend` sobre `repo-frontend`, ambos sellados en el kickoff del cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** `repo-backend` y `repo-frontend` reciben cada uno la rama `feat/x` desde su rama de integración
- **Y** ninguno recibe rama de rol ni rama de archivado

#### Scenario: Dos roles que comparten un repositorio de código en multirepo
- **DADO** un multirepo con los roles `api` y `jobs` sobre el mismo `repo-backend` y el rol `frontend` sobre `repo-frontend`
- **CUANDO** se ejecuta `ensure --change x` con los tres roles sellados
- **ENTONCES** `repo-backend` recibe `feat/x`, `feat/x-api` y `feat/x-jobs`
- **Y** `repo-frontend` recibe solo `feat/x`

#### Scenario: Un rol sellado que no toca ningún repositorio declarado no crea ramas
- **DADO** un rol sellado en el kickoff que no figura en `axiom.yaml` y que no es `fullstack`
- **CUANDO** se planifica el cambio
- **ENTONCES** ese rol no contribuye a ningún repositorio ni a ninguna rama
- **Y** la salida incluye un aviso que nombra el rol

#### Scenario: Un rol llamado `archive` colisiona con la rama de archivado
- **DADO** un multirepo con los roles `core` y `archive` sellados sobre el mismo repositorio de código
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando el rol `archive` y la colisión con el nombre reservado `feat/x-archive`
- **Y** no se crea ninguna rama de rol

#### Scenario: La rama de archivado no pertenece al modelo de roles
- **DADO** un monorepo con el rol único `fullstack` y las ramas `feat/x` y `feat/x-archive`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de `feat/x` es la rama de integración
- **Y** el de la rama de archivado `feat/x-archive` es la rama de integración
- **Y** el plan no contiene ninguna rama de rol



#### Scenario: Con plantilla en un solo rol la rama del cambio sigue siendo única
- **DADO** un multirepo con el rol `backend` sobre `repo-backend` y `roles.backend.branch_template: "feature/{task_ref}"` con Task `1235`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** `repo-backend` recibe solo la rama `feature/1235`, sin rama de seguimiento ni de rol

#### Scenario: Dos roles con plantilla de rol propia
- **DADO** un repositorio compartido por los roles `api` y `jobs`, con `repositories[].branch_template: "{prefix}/{ref}"`, `roles.api.branch_template: "feature/API-{task_ref}"` y `roles.jobs.branch_template: "feature/JOBS-{task_ref}"`, con Tasks `11` y `12`
- **CUANDO** se ejecuta `ensure --change x`, después `ensure --change x --role api` y después `ensure --change x --role jobs`
- **ENTONCES** la primera crea la rama de seguimiento `<prefijo>/<ref>`
- **Y** la segunda crea `feature/API-11` desde la rama de seguimiento y la tercera `feature/JOBS-12` desde la misma

### Requirement: Papel del Repositorio según la Topología (RF-BRA-05)

El tratamiento de cada repositorio DEBE depender de su papel:

- **`monorepo-embedded`**: el repositorio del planning home es un repositorio con rama de cambio y recibe el modelo de RF-BRA-04, con worktree según el modo de trabajo.
- **`multirepo`, repositorio de specs**: se trabaja SIEMPRE in situ, en su rama de integración. `ensure` NO DEBE crear en él ramas del cambio, ramas de rol, ramas de archivado ni worktrees. `ensure` NO DEBE hacer checkout en él: si no está en su rama de integración, o está en una rama de rol del cambio, DEBE rechazar con `specs_repo_not_on_integration`, nombrando la rama actual y la continuación.
- **`multirepo`, repositorios de código**: cada repositorio tocado por algún rol del cambio recibe el modelo de RF-BRA-04 desde su propia rama de integración.

Git no admite dos worktrees sobre la misma rama, por lo que la integración del repositorio de specs solo puede trabajarse in situ.

#### Scenario: El repositorio de specs de un multirepo no recibe ramas ni worktrees
- **DADO** un multirepo con el repositorio de specs en su rama `main`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el repositorio de specs conserva `main` desplegada, sin rama `feat/x` ni worktree
- **Y** el JSON lo lista como trabajado in situ con su raíz git absoluta y su rama de integración

#### Scenario: Repositorio de specs fuera de su rama de integración
- **DADO** un multirepo cuyo repositorio de specs está desplegado en `feat/otro`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `specs_repo_not_on_integration` nombrando `feat/otro` y `main`
- **Y** no se hace checkout ni se crea ninguna rama

#### Scenario: Repositorio de specs en una rama de rol del cambio
- **DADO** un multirepo cuyo repositorio de specs está en `feat/x-core`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `specs_repo_not_on_integration`

#### Scenario: En monorepo embebido el repositorio del planning home recibe la rama del cambio
- **DADO** un monorepo `monorepo-embedded` con un rol
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el repositorio del planning home recibe la rama `feat/x` y, en modo `worktree`, un worktree

---

### Requirement: Raíz Git Real de Cada Repositorio (RF-BRA-06)

El motor DEBE obtener la raíz git de cada repositorio declarado en `roles.<rol>.repositories[].path` con las consultas de git (`rev-parse --show-toplevel` y `--git-common-dir`), nunca suponiendo un directorio `.git` literal. Una ruta que sea una subcarpeta de un repositorio, el propio `.`, o un repositorio compartido por varios roles DEBE resolverse a una única raíz git deduplicada, con un único conjunto de ramas y worktrees por raíz. Una ruta que no pertenezca a ningún repositorio git DEBE omitirse sin abortar el resto del plan: el JSON DEBE listarla como omitida con el código `not_a_git_repository` y un mensaje que nombre la ruta y el rol. Una ruta declarada que no exista DEBE reportarse con el mismo tratamiento de omisión y mensaje.

#### Scenario: Una subcarpeta resuelve a la raíz git del repositorio que la contiene
- **DADO** un rol con `repositories: [{path: services/api}]` dentro de un repositorio cuya raíz es la carpeta maestra
- **CUANDO** se planifica el cambio
- **ENTONCES** el repositorio del plan es la raíz git, no `services/api`

#### Scenario: Dos roles sobre rutas del mismo repositorio se deduplican
- **DADO** dos roles con rutas `app/web` y `app/api`, ambas dentro del mismo repositorio
- **CUANDO** se planifica el cambio
- **ENTONCES** el plan contiene un único repositorio
- **Y** los dos roles se cuentan como roles del cambio sobre ese repositorio (RF-BRA-04)

#### Scenario: Ruta que no es un repositorio git se omite con mensaje
- **DADO** un rol cuya ruta `docs/` existe pero no pertenece a ningún repositorio git
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** los demás repositorios del plan se procesan con normalidad
- **Y** el JSON lista `docs/` como omitido con el código `not_a_git_repository` y un mensaje en español que nombra la ruta y el rol

#### Scenario: Entrada `.` equivale a la raíz del proyecto
- **DADO** un perfil con un rol único cuyo repositorio es `.`
- **CUANDO** se planifica el cambio
- **ENTONCES** el repositorio del plan es la raíz git del propio proyecto

---

### Requirement: `ensure` Planifica de Forma Determinista y es Idempotente (RF-BRA-07)

`axiom branch ensure` DEBE producir un plan determinista a partir de la topología, `git:`, `repositories:`, los roles del kickoff y las preferencias personales, y ejecutarlo con git. Una segunda ejecución con los mismos argumentos y el mismo estado DEBE no cambiar nada (ni ramas, ni worktrees, ni registro) y devolver el mismo JSON, salvo `branches[].createdFrom` y `branches[].createdFromRef` (RF-BRA-27), que se emiten solo en la ejecución que crea la rama y no en las siguientes. El comando DEBE crear solo lo que falta: si la rama ya existe, DEBE reutilizarla sin moverla ni reiniciarla; si no existe en local pero existe su referencia remota local, DEBE adoptarla (RF-BRA-27); si ya existe su worktree registrado, DEBE devolver su ruta. Si la rama ya está desplegada en un worktree que el registro local no conoce, DEBE rechazar con `branch_checked_out_elsewhere` nombrando la ruta. Cuando todavía no hay kickoff sellado (el cambio se nombra antes del sellado), `ensure` DEBE planificar solo para el repositorio del planning home la rama `<prefijo>/<cambio>`, que es idéntica en ambos modelos de RF-BRA-04 (en `multirepo` ese repositorio se trabaja in situ, RF-BRA-05, y no se crea ninguna rama), y una ejecución posterior, ya con el kickoff sellado, DEBE completar de forma incremental las ramas de rol y los repositorios de código que correspondan sin alterar lo ya creado (supuesto S-1). `ensure --change <c> --archive` (RF-BRA-13) es un plan aparte con las mismas garantías de determinismo e idempotencia: no altera las ramas del cambio ya creadas.
(Previously: los escenarios de ejecución previa y posterior al sellado y de `--archive` suponían un monorepo con los roles `core` y `qa` y la rama `feat/x-core`; ahora el de roles se ejecuta sobre `multirepo` y los de monorepo sobre el rol `fullstack`)


#### Scenario: Segunda ejecución sin cambios
- **DADO** un cambio ya preparado por `ensure`
- **CUANDO** se ejecuta de nuevo `ensure --change x --json` con los mismos argumentos
- **ENTONCES** el JSON es idéntico al de la primera ejecución, salvo `createdFrom` y `createdFromRef`, que solo aparecen en la ejecución que creó la rama
- **Y** `git worktree list` y `git branch --list` no han cambiado

#### Scenario: La rama existente se reutiliza sin moverla
- **DADO** una rama `feat/x` ya existente con commits propios y sin worktree
- **CUANDO** se ejecuta `ensure --change x` en modo `worktree`
- **ENTONCES** se crea el worktree sobre `feat/x`
- **Y** el commit al que apunta `feat/x` no cambia

#### Scenario: Rama desplegada en un worktree desconocido
- **DADO** una rama `feat/x` desplegada en un worktree que el registro local no conoce
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `branch_checked_out_elsewhere` nombrando la ruta del worktree existente
- **Y** no se crea ningún otro worktree

#### Scenario: Ejecución previa al sellado y posterior al sellado
- **DADO** un cambio nombrado sin `kickoff.yaml`, en un multirepo con los roles `core` y `qa` sobre el mismo repositorio de código `repo-backend`
- **CUANDO** se ejecuta `ensure --change x` antes del sellado y de nuevo después de sellar los dos roles
- **ENTONCES** la primera ejecución no crea ninguna rama, porque el repositorio de specs de un multirepo se trabaja in situ en su rama de integración (RF-BRA-05)
- **Y** la segunda conserva lo ya creado y crea en `repo-backend` `feat/x`, `feat/x-core` y `feat/x-qa` (las de rol desde `feat/x`)

#### Scenario: En monorepo la ejecución posterior al sellado no añade ramas
- **DADO** un cambio nombrado sin `kickoff.yaml`, en un monorepo
- **CUANDO** se ejecuta `ensure --change x` antes del sellado y de nuevo después de sellar el rol `fullstack`
- **ENTONCES** ambas ejecuciones dejan solo la rama `feat/x`
- **Y** la segunda no cambia nada

#### Scenario: Se rechaza un nombre de cambio que no es válido
- **DADO** un nombre de cambio con caracteres inválidos como componente de rama git
- **CUANDO** se ejecuta `ensure --change "a b:c"`
- **ENTONCES** se rechaza con `invalid_name`
- **Y** no se crea nada

#### Scenario: `--archive` no altera las ramas del cambio
- **DADO** un monorepo con la rama `feat/x` ya creada y su PR fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea `feat/x-archive` y el commit al que apunta `feat/x` no cambia

#### Scenario: La rama publicada se adopta en lugar de crearse desde la base
- **DADO** una rama `feat/x` inexistente en local y `refs/remotes/origin/feat/x` presente
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** `feat/x` se crea desde `origin/feat/x`
- **Y** una segunda ejecución con los mismos argumentos devuelve el mismo JSON sin cambiar nada


---

### Requirement: `ensure`: Modo de Trabajo `worktree`, `in_place` y `ask` (RF-BRA-08)

`ensure` DEBE leer el modo de trabajo del hub personal (`~/.axiom/workspaces.json`) y NUNCA de `axiom.yaml`. Sin preferencias guardadas para el proyecto, estén o no registrado, el modo DEBE ser `worktree`.

- `worktree`: DEBE crear la rama desde la base y un worktree en la raíz de worktrees (RF-BRA-11). NUNCA DEBE modificar el HEAD, el índice ni el árbol de trabajo del checkout principal.
- `in_place`: DEBE crear (o seleccionar) la rama y hacer checkout de ella en el propio repositorio, sujeto a RF-BRA-09.
- `ask`: DEBE devolver el resultado tipado `decision_required` sin crear nada, sin leer la entrada estándar y sin interactuar; quien invoca (skill o dashboard) presenta la pregunta y repite la orden con la respuesta (`--mode worktree|in_place`, supuesto S-2). La respuesta DEBE registrarse por cambio en el registro local (RF-BRA-15) y las ejecuciones posteriores para ese mismo cambio, incluida la de `--archive`, DEBEN reutilizarla sin volver a preguntar. Una respuesta registrada NO modifica la preferencia personal del hub.

Un `--mode` distinto del ya materializado para el cambio DEBE rechazarse con `plan_conflict`. `decision_required` y `plan_conflict` NO son exclusivos del modo de trabajo: el modo de promoción (RF-BRA-28 y RF-BRA-29) usa el mismo mecanismo y el mismo esquema de pregunta, con una pregunta distinta por repositorio, y cuando faltan ambas respuestas la del modo de trabajo se pregunta primero.
(Previously: `decision_required` y `plan_conflict` se describían solo para el modo de trabajo)

#### Scenario: Sin preferencias el modo es `worktree`
- **DADO** un proyecto sin preferencias guardadas en el hub
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se crea la rama y un worktree en la raíz de worktrees por defecto

#### Scenario: El checkout principal no se toca en modo `worktree`
- **DADO** el checkout principal desplegado en `main` con un fichero modificado sin confirmar
- **CUANDO** se ejecuta `ensure --change x` en modo `worktree`
- **ENTONCES** el HEAD, el índice y el árbol de trabajo del checkout principal son idénticos a los de antes
- **Y** el checkout principal sigue en `main`

#### Scenario: Modo `in_place` hace checkout en el propio repositorio
- **DADO** un proyecto en modo `in_place` con el árbol limpio
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el repositorio queda desplegado en `feat/x`
- **Y** no se crea ningún worktree

#### Scenario: Modo `ask` devuelve `decision_required` sin crear nada
- **DADO** un proyecto en modo `ask` y sin respuesta registrada para el cambio
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el estado es `decision_required` y el código de salida es distinto de cero
- **Y** no se crea ninguna rama, worktree ni registro
- **Y** el comando no espera entrada alguna

#### Scenario: La respuesta a `ask` se registra por cambio
- **DADO** un proyecto en modo `ask`
- **CUANDO** se ejecuta `ensure --change x --mode worktree` y después `ensure --change x` sin `--mode`
- **ENTONCES** la segunda ejecución no devuelve `decision_required` y reutiliza `worktree`
- **Y** para el cambio `y`, sin respuesta registrada, `ensure --change y` devuelve `decision_required`
- **Y** la preferencia personal del hub sigue siendo `ask`

#### Scenario: La rama de archivado reutiliza la respuesta registrada
- **DADO** un proyecto en modo `ask` y el cambio `x` con la respuesta `worktree` registrada y su PR fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** no devuelve `decision_required` y crea el worktree de la rama de archivado

#### Scenario: Un modo distinto del materializado se rechaza
- **DADO** un cambio materializado como `worktree`
- **CUANDO** se ejecuta `ensure --change x --mode in_place`
- **ENTONCES** se rechaza con `plan_conflict` y no se modifica nada

#### Scenario: Con ambas decisiones pendientes se pregunta primero el modo de trabajo
- **DADO** un proyecto con el modo de trabajo `ask` en el hub y `git.promotion: ask` con dos etapas, sin respuestas
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el `decision_required` pregunta por el modo de trabajo y ofrece `worktree` e `in_place`
- **Y** tras responderlo, la siguiente ejecución pregunta por el modo de promoción con `branch` y `release`

---

### Requirement: `ensure` en `in_place` Rechaza un Árbol Sucio (RF-BRA-09)

En modo `in_place`, `ensure` DEBE rechazar con `dirty_worktree` cuando el repositorio cuya rama va a cambiar tenga cambios sin confirmar (ficheros rastreados modificados o ficheros sin seguimiento no ignorados), sin modificar el repositorio y nombrando los ficheros afectados y la continuación (confirmar, guardar o usar el modo `worktree`). Si el repositorio ya está desplegado en la rama esperada, `ensure` DEBE ser un no-op aunque el árbol esté sucio, porque no cambia de rama.

#### Scenario: Árbol sucio rechaza el cambio de rama
- **DADO** un proyecto en modo `in_place` con un fichero rastreado modificado
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `dirty_worktree` nombrando el fichero
- **Y** el repositorio sigue en su rama original y el fichero conserva su modificación

#### Scenario: Un fichero sin seguimiento también cuenta
- **DADO** un proyecto en modo `in_place` con un fichero nuevo sin seguimiento y no ignorado
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `dirty_worktree`

#### Scenario: Ya en la rama esperada con el árbol sucio es un no-op
- **DADO** un proyecto en modo `in_place` ya desplegado en `feat/x`, con cambios sin confirmar
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la orden termina con código `0` sin modificar nada

---

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

### Requirement: Rutas de Worktree y Límite de Longitud (RF-BRA-11)

La ruta del worktree de cada rama DEBE ser `<raíz de worktrees>/<cambio>[-<rol>|-archive]/<repo>`, donde el sufijo `-<rol>` solo aparece si existe rama de rol para ese repositorio (RF-BRA-04), el sufijo `-archive` solo aparece en el worktree de la rama de archivado (RF-BRA-13) y `<repo>` es el último segmento de la ruta (`path`) del repositorio, tal como la declara `repositories:` o el rol (supuesto S-4); para el repositorio cuya ruta es `.`, `<repo>` es el nombre del directorio de su raíz git, y si dos repositorios del plan tuvieran el mismo último segmento, `ensure` DEBE rechazar con `invalid_name` nombrando ambos, antes de crear nada (supuesto S-17). La raíz de worktrees DEBE ser la preferencia personal del hub o, sin ella, `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`, fuera de cualquier repositorio. El planificador DEBE validar, antes de crear nada, que el nombre del cambio es válido a la vez como componente de rama git y como nombre de directorio en Windows, y que la longitud de cada ruta de worktree no supera el límite admitido: el de la plataforma menos un margen de 40 caracteres, es decir, 220 caracteres en Windows sin rutas largas habilitadas (260 menos 40). Si no se cumple, DEBE rechazar con `invalid_name` o `path_too_long`, nombrando la ruta, su longitud y el máximo, sin crear ramas ni worktrees.
(Previously: `<repo>` era el nombre declarado en `repositories:` o el del directorio de la raíz git, la ruta no contemplaba la rama de archivado y el límite solo exigía «margen» sin cifra; el escenario del sufijo de rol se ejecutaba sobre un monorepo con los roles `core` y `qa` y ahora sobre `multirepo`)

#### Scenario: Ruta sin sufijo de rol con una sola rama
- **DADO** un monorepo con un rol y modo `worktree`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el worktree está en `<raíz de worktrees>/x/<repo>`
- **Y** la ruta no contiene sufijo de rol

#### Scenario: Ruta con sufijo de rol cuando hay rama de rol
- **DADO** un multirepo con los roles `core` y `qa` sobre el mismo repositorio de código
- **CUANDO** se ejecuta `ensure --change x` con ambos sellados
- **ENTONCES** el worktree de la rama de seguimiento está en `<raíz de worktrees>/x/<repo>`
- **Y** los de las ramas de rol están en `<raíz de worktrees>/x-core/<repo>` y `<raíz de worktrees>/x-qa/<repo>`

#### Scenario: Ruta de la rama de archivado
- **DADO** un monorepo en modo `worktree` y el cambio `x` fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** el worktree de la rama de archivado está en `<raíz de worktrees>/x-archive/<repo>`

#### Scenario: `<repo>` es el último segmento de la ruta declarada
- **DADO** un multirepo con un repositorio declarado con `path: services/repo-backend`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el worktree de ese repositorio está en `<raíz de worktrees>/x/repo-backend`

#### Scenario: Dos repositorios con el mismo último segmento se rechazan
- **DADO** dos repositorios del plan declarados con `path: a/api` y `path: b/api`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando ambas rutas
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Raíz de worktrees por defecto fuera de cualquier repositorio
- **DADO** una carpeta maestra `<P>/axiom` sin preferencia de raíz en el hub
- **CUANDO** se calcula la raíz de worktrees
- **ENTONCES** es `<P>/axiom-wt`, que no está dentro de ningún repositorio del proyecto

#### Scenario: Ruta demasiado larga en Windows
- **DADO** una raíz de worktrees personalizada y un nombre de cambio cuyo worktree tendría 221 caracteres
- **CUANDO** se ejecuta `ensure --change <nombre largo>`
- **ENTONCES** se rechaza con `path_too_long` nombrando la ruta, su longitud (221) y el máximo (220)
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Una ruta de exactamente 220 caracteres se admite
- **DADO** una raíz de worktrees personalizada y un nombre de cambio cuyo worktree tendría exactamente 220 caracteres
- **CUANDO** se ejecuta `ensure --change <nombre>`
- **ENTONCES** se crea el worktree

#### Scenario: Nombre con carácter inválido en Windows
- **DADO** un nombre de cambio con un carácter no admitido en nombres de directorio de Windows (por ejemplo `:` o `?`)
- **CUANDO** se ejecuta `ensure --change`
- **ENTONCES** se rechaza con `invalid_name` antes de crear nada



---
### Requirement: Excepciones por Cambio `--prefix`, `--branch` y `--base` (RF-BRA-12)

`ensure` DEBE admitir `--prefix`, `--branch` y `--base` como excepciones por cambio a la política del proyecto. `--prefix` sustituye el prefijo; `--branch` sustituye el nombre completo de la rama del cambio (la rama de seguimiento si hay ramas de rol, y las ramas de rol derivan como `<--branch>-<rol>`) y se aplica a todos los repositorios del plan, que `--repo` PUEDE acotar; `--base` sustituye la base de todos los repositorios del plan. Las excepciones DEBEN registrarse en el registro local y, cuando el cambio se selle, reflejarse en la instantánea del `kickoff.yaml` (RF-KOF-06) para que ejecuciones posteriores y las guardas usen los mismos nombres. Una invocación posterior cuyas excepciones contradigan las ya materializadas DEBE rechazarse con `plan_conflict`. Con `--archive` (RF-BRA-13) solo se admite `--prefix`, que fija el prefijo de la rama de archivado sin contradecir el del cambio; `--branch` y `--base` se rechazan con `plan_conflict` (supuesto S-16). Sin `--prefix`, la rama de archivado deriva de la rama del cambio registrada como `<rama del cambio>-archive`.
(Previously: las excepciones no contemplaban la rama de archivado)

#### Scenario: `--branch` fija el nombre de la rama del cambio
- **DADO** un monorepo con un rol
- **CUANDO** se ejecuta `ensure --change x --branch inc/inc-25-x`
- **ENTONCES** la rama del cambio es `inc/inc-25-x` y no `feat/x`

#### Scenario: `--branch` con ramas de rol deriva los nombres
- **DADO** un cambio con los roles `core` y `qa` sobre el mismo repositorio
- **CUANDO** se ejecuta `ensure --change x --branch inc/x`
- **ENTONCES** la rama de seguimiento es `inc/x` y las ramas de rol son `inc/x-core` e `inc/x-qa`

#### Scenario: `--base` fija la base
- **DADO** un repositorio con una rama `release/2`
- **CUANDO** se ejecuta `ensure --change x --base release/2`
- **ENTONCES** `base.ref` es `release/2` en el JSON

#### Scenario: Una excepción posterior contradictoria se rechaza
- **DADO** un cambio materializado con `--branch inc/x`
- **CUANDO** se ejecuta `ensure --change x --branch feat/x`
- **ENTONCES** se rechaza con `plan_conflict`
- **Y** no se crea ninguna rama nueva

#### Scenario: Una repetición idéntica de la excepción es idempotente
- **DADO** un cambio materializado con `--branch inc/x`
- **CUANDO** se repite `ensure --change x --branch inc/x`
- **ENTONCES** la orden termina con código `0` sin modificar nada

#### Scenario: `--prefix` fija el prefijo de la rama de archivado
- **DADO** un cambio `x` materializado con la rama `feat/x` y fusionado
- **CUANDO** se ejecuta `ensure --change x --archive --prefix doc`
- **ENTONCES** la rama de archivado es `doc/x-archive`
- **Y** la rama del cambio sigue siendo `feat/x` sin `plan_conflict`

#### Scenario: La rama de archivado deriva de la rama del cambio registrada
- **DADO** un cambio materializado con `--branch inc/inc-25-x` y fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** la rama de archivado es `inc/inc-25-x-archive`

#### Scenario: `--branch` y `--base` no se admiten con `--archive`
- **DADO** un cambio fusionado
- **CUANDO** se ejecuta `ensure --change x --archive --branch doc/x` o `ensure --change x --archive --base release/2`
- **ENTONCES** se rechaza con `plan_conflict`
- **Y** no se crea ninguna rama ni worktree

---

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
### Requirement: Carril ODD con la Misma Política (RF-BRA-14)

`ensure --lane odd --change <feature>` DEBE aplicar la misma política que SDD, incluida la plantilla de nombre de rama: la rama de la feature se nombra con la plantilla efectiva (`git.branch_template`, o `repositories[].branch_template` en un repositorio de código incluido con `--repo`) y su nombre se congela en el registro local (RF-BRA-15) en el primer `ensure`. El carril ODD no tiene roles ni Tasks, por lo que `{role}` y `{task_ref}` NO resuelven y su uso se rechaza con el código tipado de RF-BRA-45; `roles.<id>.branch_template` no se aplica nunca. Como no hay `kickoff.yaml`, los valores salen así: `{ref}` del vínculo `work_item` de `integrations.yaml` (que ODD también escribe al adoptar) o, si aún no existe, del staging de `change start`; `{kind}` del perfil del staging o del documento adoptado y, si no hay, `feature`; `{slug}` según RF-BRA-42. Una plantilla con `{ref}` y sin work item en ODD se rechaza con el código de variable sin valor de RF-BRA-45. Con estas particularidades: el documento ODD es `<workspaceRoot>/odd/tasks/<feature>.md`, en el repositorio del planning home. En `monorepo-embedded`, la rama es `<prefijo>/<feature>` con worktree según el modo; en `multirepo`, el repositorio de specs se trabaja in situ en su rama de integración (RF-BRA-05) y los repositorios de código se incluyen solo bajo demanda con `--repo <ruta>`, porque ODD no divide en roles y cada repositorio incluido recibe solo la rama `<prefijo>/<feature>`. En el carril ODD no existe `kickoff.yaml` ni instantánea ni rama de archivado: las únicas protecciones que se aplican son la propia política de `ensure` y la guarda D (RF-BRA-22), que bloquea las escrituras del agente en `odd/tasks/` fuera de la rama que corresponde, porque no hay escritor Go del documento ODD.
(Previously: la rama de la feature era siempre `<prefijo>/<feature>`)

#### Scenario: ODD en monorepo prepara rama y worktree
- **DADO** un monorepo `monorepo-embedded` en modo `worktree`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se crea la rama `feat/mi-feature` y su worktree
- **Y** el JSON indica `<worktree>/odd/tasks/mi-feature.md` como ubicación del documento ODD

#### Scenario: ODD en multirepo trabaja in situ en el repositorio de specs
- **DADO** un multirepo con el repositorio de specs en su rama de integración
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** no se crea rama ni worktree en el repositorio de specs
- **Y** el JSON indica `<repositorio de specs>/odd/tasks/mi-feature.md`

#### Scenario: Un repositorio de código se incluye con `--repo`
- **DADO** un multirepo con `repo-backend`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature --repo repo-backend`
- **ENTONCES** `repo-backend` recibe solo la rama `feat/mi-feature`, sin rama de rol

#### Scenario: ODD no crea `kickoff.yaml`
- **DADO** un monorepo
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** no se crea ningún fichero bajo `openspec/changes/`


#### Scenario: ODD aplica la plantilla de `git:` y congela el nombre
- **DADO** un monorepo con `git.branch_template: "{kind}/{change}"`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se crea la rama `feature/mi-feature`
- **Y** el registro guarda ese nombre y un `ensure` posterior lo reutiliza aunque cambie la plantilla

#### Scenario: ODD en un repositorio de código usa la plantilla del repositorio
- **DADO** un multirepo con `repositories[].branch_template: "feature/{change}"` para `repo-backend`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature --repo repo-backend`
- **ENTONCES** `repo-backend` recibe la rama `feature/mi-feature`

#### Scenario: ODD sin plantilla conserva el nombre actual
- **DADO** un `axiom.yaml` sin plantillas y el prefijo `feat`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** la rama es `feat/mi-feature`

#### Scenario: ODD toma `{ref}` y `{kind}` del staging
- **DADO** `git.branch_template: "{kind}/{ref}-{slug}"`, el staging de `change start` del cambio `us-2525-alta` con `id` `2525` y perfil bug, y sin vínculo `work_item` todavía
- **CUANDO** se ejecuta `ensure --lane odd --change us-2525-alta`
- **ENTONCES** la rama es `fix/2525-alta`
- **Y** el registro congela ese nombre

#### Scenario: ODD con `{kind}` sin perfil conocido vale `feature`
- **DADO** `git.branch_template: "{kind}/{change}"` y un cambio ODD sin staging ni perfil en el documento
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** la rama es `feature/mi-feature`

#### Scenario: ODD con `{ref}` sin work item se rechaza
- **DADO** `git.branch_template: "{prefix}/{ref}"` y un cambio ODD sin vínculo `work_item` ni staging
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se rechaza con el código de variable sin valor que nombra `ref`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: ODD rechaza `{task_ref}`
- **DADO** `git.branch_template: "feature/{task_ref}"`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se rechaza con el código tipado de RF-BRA-45 que nombra `{task_ref}`
- **Y** no se crea ninguna rama ni worktree

### Requirement: Registro Local en el Git Common Dir (RF-BRA-15)

`ensure` DEBE mantener un registro local por cambio, fuera de cualquier árbol versionado, en el git common dir del repositorio del planning home, obtenido con `git rev-parse --git-common-dir` (nunca asumiendo `.git`). El registro DEBE contener las rutas absolutas de todos los repositorios, ramas (incluida la de archivado) y worktrees del cambio, la respuesta a `ask` del modo de trabajo (RF-BRA-08) y, por cada repositorio con dos etapas, el modo de promoción resuelto y su origen (RF-BRA-28). Los campos del modo de promoción DEBEN ser opcionales: un registro sin ellos sigue siendo válido y un binario anterior los ignora. Cuando el nombre de una rama se obtiene de una plantilla (RF-BRA-44), el primer `ensure` que crea esa rama DEBE congelar el nombre resuelto en el registro, y las ejecuciones posteriores DEBEN leer ese nombre sin volver a renderizar la plantilla, de modo que un cambio posterior de la plantilla en `axiom.yaml` no renombra ni duplica una rama ya creada. El registro es la fuente de verdad del nombre congelado. Si el registro se borra, `ensure` DEBE recuperar el nombre de la instantánea o, si esta no lo guarda, de renderizar la plantilla de la instantánea con las entradas versionadas actuales, y reutilizar la rama existente con ese nombre sin renombrar nada; un error de git distinto de «la rama no existe» al comprobarla se propaga como error y nunca se trata como ausencia. Es una limitación declarada: si la Task de un rol se revinculó a otro `id` después de crear la rama y el registro se perdió, el nombre renderizado puede diferir del de la rama existente; la salida es `--branch` con el nombre existente o restaurar el vínculo anterior. Una rama pendiente (RF-BRA-46) NO DEBE figurar en el registro. El registro es la fuente operativa del modo de promoción en todos los carriles. Debe ser accesible con el mismo contenido desde el checkout principal y desde cualquier worktree enlazado del mismo repositorio. Si el registro se borra, el repositorio NO DEBE verse afectado y `ensure` DEBE reconstruirlo de forma idempotente a partir del estado de git (ramas y worktrees existentes) y, para el modo de promoción, de la instantánea del `kickoff.yaml` si la hay (RF-BRA-28), mientras `status` DEBE informar que el cambio no tiene registro.
(Previously: el registro no guardaba el modo de promoción ni su origen y la reconstrucción no consultaba la instantánea)
(Previously: el registro no congelaba el nombre de las ramas nombradas por plantilla ni leía ese nombre al planificar)
(Previously: la cláusula sobre la coexistencia con el estado local de RDD (`<git common dir>/gentle-ai/`) y sus linajes; RDD se retira (C-01-e) y `ensure` ya no tiene nada que preservar de él.)

#### Scenario: El registro se resuelve con el git common dir
- **DADO** un repositorio cuyo directorio git no se llama `.git` (git dir enlazado)
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro se crea bajo el directorio devuelto por `git rev-parse --git-common-dir`

#### Scenario: El registro es visible desde un worktree enlazado
- **DADO** un cambio preparado por `ensure` en modo `worktree`
- **CUANDO** se ejecuta `axiom branch status --change x --cwd <ruta del worktree>`
- **ENTONCES** el resultado incluye las rutas absolutas de todos los repositorios y worktrees del cambio registrados desde el checkout principal

#### Scenario: Borrar el registro no daña el repositorio
- **DADO** un cambio preparado y su registro borrado
- **CUANDO** se ejecuta `ensure --change x` de nuevo
- **ENTONCES** las ramas y worktrees existentes se reutilizan sin modificarlos
- **Y** el registro se reconstruye con las mismas rutas absolutas

#### Scenario: Registro ausente informado por `status`
- **DADO** un cambio sin registro local
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON indica que el cambio no tiene registro y no falla

#### Scenario: El registro guarda el modo de promoción y su origen por repositorio
- **DADO** un multirepo con dos repositorios de código con dos etapas, uno resuelto por respuesta y otro por la política
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el registro contiene, por cada repositorio, su modo y su origen

#### Scenario: Un registro sin modo de promoción sigue siendo válido
- **DADO** un registro escrito antes de esta capacidad, sin los campos del modo
- **CUANDO** se ejecutan `status` y `ensure --change x`
- **ENTONCES** ambas órdenes lo leen sin error y no lo consideran corrupto

#### Scenario: `ensure` conserva el modo registrado al reescribir el registro
- **DADO** un cambio con el modo `branch` registrado para `repo-backend`
- **CUANDO** se ejecuta `ensure --change x --archive` o de nuevo `ensure --change x`
- **ENTONCES** el modo de `repo-backend` sigue siendo `branch` en el registro

#### Scenario: El primer `ensure` que crea la rama congela su nombre
- **DADO** `repositories[].branch_template: "feature/TASK-{task_ref}"` y la Task `1235` del rol `frontend`
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el registro guarda el nombre `feature/TASK-1235` para esa rama

#### Scenario: Un cambio de la plantilla no renombra una rama ya creada
- **DADO** la rama `feature/TASK-1235` creada y registrada, y después `axiom.yaml` con `repositories[].branch_template: "feat/{task_ref}"`
- **CUANDO** se ejecuta de nuevo `ensure --change x --role frontend`
- **ENTONCES** el plan usa `feature/TASK-1235`, la lee del registro y no crea `feat/1235`

#### Scenario: Registro borrado con ramas de plantilla
- **DADO** el cambio anterior con la rama `feature/TASK-1235` creada y el registro borrado
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** la rama se reutiliza sin modificarla y el registro se reconstruye con el nombre `feature/TASK-1235`

#### Scenario: Task revinculada y registro borrado: limitación declarada
- **DADO** la rama `feature/TASK-1235` creada y registrada, la Task del rol `frontend` revinculada después a `1300` en `integrations.yaml` y el registro borrado
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el nombre se renderiza con el vínculo actual (`feature/TASK-1300`) y no se renombra ni se borra la rama `feature/TASK-1235`
- **Y** la continuación ofrece `--branch feature/TASK-1235` o restaurar el vínculo anterior

#### Scenario: La rama pendiente no figura en el registro
- **DADO** una rama pendiente por `{task_ref}` y un `ensure --change x` sin `--role`
- **CUANDO** termina
- **ENTONCES** el registro no contiene ninguna entrada para esa rama

### Requirement: `status`: Estado de Ramas y Worktrees del Cambio (RF-BRA-16)

`axiom branch status` DEBE informar, para cada repositorio del cambio, su papel (repositorio con rama de cambio, repositorio de specs in situ o repositorio de código), la rama esperada y la rama actual, la ruta absoluta del worktree si existe, si el árbol tiene cambios sin confirmar, el modo de trabajo materializado, la base, y el número de commits que la rama lleva de atraso o de adelanto respecto de su base calculado solo con referencias locales, y la rama destino propuesta para el PR (RF-BRA-04). En un repositorio con dos etapas, la base es la base efectiva de su modo de promoción (RF-BRA-29) y DEBE informar además de ese modo, de la rama principal y, en modo `branch` con la rama del cambio ya fusionada en la integración, de la siguiente PR con destino la rama principal (RF-BRA-30). Cuando existe la rama de archivado, DEBE incluirla con los mismos datos. `--change` es opcional: sin él, `status` lista los cambios registrados y los cambios pendientes de archivar (RF-BRA-21); con él, además informa de si el cambio solicitado está pendiente de archivar. NUNCA DEBE realizar operaciones remotas. DEBE poder ejecutarse desde el checkout principal, desde cualquier worktree del cambio y con `--cwd`. Un cambio sin ramas creadas DEBE informarse como no iniciado (`not_started`), no como error. DEBE listar además las ramas pendientes por `{task_ref}` (RF-BRA-46), con su repositorio, su rol y su plantilla, en un campo aditivo del JSON; una rama pendiente no tiene rama esperada ni actual, y se lista con la orden que la crea (RF-BRA-46). Con `integrations.yaml` ilegible y plugin activo, `status` informa el código del rechazo en la entrada afectada sin fallar la orden entera.
(Previously: `status` no listaba ramas pendientes)
(Previously: `status` no informaba del modo de promoción ni de la siguiente PR, y la base era siempre la de integración)

#### Scenario: Estado de un cambio con rama y worktree
- **DADO** un cambio preparado por `ensure` en modo `worktree`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON incluye, por repositorio, la rama esperada, la rama actual, la ruta absoluta del worktree y si el árbol está limpio

#### Scenario: Atraso calculado solo con referencias locales
- **DADO** una rama `feat/x` con 3 commits menos que `main` y sin acceso a la red
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON informa de 3 commits de atraso respecto de la base
- **Y** no se ejecuta ninguna operación remota

#### Scenario: Cambio no iniciado
- **DADO** un nombre de cambio sin rama ni worktree
- **CUANDO** se ejecuta `axiom branch status --change nuevo --json`
- **ENTONCES** el estado es el token `not_started` con la etiqueta `no iniciado` y el código de salida es `0`

#### Scenario: Rama destino propuesta para el PR
- **DADO** un cambio con las ramas de rol `feat/x-core` y `feat/x-qa` y seguimiento `feat/x`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de cada rama de rol es `feat/x` y el de `feat/x` es la rama de integración

#### Scenario: La rama de archivado figura en el plan
- **DADO** un cambio con la rama de archivado `feat/x-archive` y su worktree
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON incluye la rama `feat/x-archive` con la ruta absoluta de su worktree
- **Y** su destino propuesto de PR es la rama de integración

#### Scenario: `status` sin `--change` lista registrados y pendientes
- **DADO** un proyecto con el cambio `x` registrado y el cambio `y` fusionado y pendiente de archivar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el JSON lista `x` entre los cambios registrados y `y` entre los pendientes de archivar

#### Scenario: La base y el modo de un repositorio con dos etapas figuran en el estado
- **DADO** un cambio en modo `branch` con `main_branch: master` y `integration_branch: develop`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la base del repositorio es `master`
- **Y** el JSON informa del modo `branch` y de la rama principal `master`, además de los campos ya existentes


#### Scenario: `status` lista las ramas pendientes
- **DADO** un multirepo sellado con `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend` y ningún `ensure --role frontend` todavía
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON lista una rama pendiente con el repositorio `repo-frontend`, el rol `frontend` y la plantilla
- **Y** esa rama no figura con rama esperada ni actual

#### Scenario: Sin ramas pendientes el JSON de `status` no cambia
- **DADO** un cambio sin plantillas con `{task_ref}`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON es idéntico al de antes de esta capacidad y no incluye ramas pendientes

### Requirement: `done`: Detección de Fusión por Capas (RF-BRA-17)

`axiom branch done` DEBE determinar, para cada rama del cambio, si ya está fusionada en su destino (la rama de rol en la de seguimiento; la rama del cambio en la de integración, o en la rama principal cuando el repositorio tiene dos etapas y modo `branch`, RF-BRA-31; la rama de archivado en la de integración) con estas capas, de la primera a la última, deteniéndose en la primera que confirme:

1. **Ascendencia** (`ancestry`): la rama es ancestro del destino.
2. **Árbol equivalente** (`equivalent_tree`): la fusión simulada de la rama en el destino (`git merge-tree --write-tree`) produce el árbol del destino. Detecta los merges por squash o rebase aunque el destino haya avanzado después. La capa DEBE omitirse sin error si la versión de git no admite la orden (Q1, a verificar en `sdd-design`).
3. **Evidencia `pr_merged`** (`pr_merged`): el `gates.yaml` del cambio contiene una evidencia `pr_merged` ya verificada. Esta capa NO aplica a la rama de archivado: la evidencia del cambio acredita el PR del cambio y no el PR documental de archivado, de modo que la rama de archivado solo se confirma por las capas 1 y 2. Tampoco aplica cuando el destino es la rama principal en modo `branch`: la evidencia no identifica el destino del PR del cambio y por tanto no acredita la fusión en la rama principal (RF-BRA-31).

Si ninguna capa confirma la fusión, la rama NO se considera fusionada y `done` DEBE exigir `--force` (RF-BRA-18). El JSON DEBE indicar, por rama, qué capa la confirmó con los tokens `ancestry`, `equivalent_tree`, `pr_merged` o `forced`, cada uno con su etiqueta en castellano. La detección DEBE usar solo referencias locales; solo con `--fetch` PUEDE actualizarse el remoto antes de evaluar.
(Previously: el destino de la rama del cambio era siempre la rama de integración y la capa `pr_merged` se aplicaba a todas las ramas salvo la de archivado)

#### Scenario: Fusión real detectada por ascendencia
- **DADO** una rama de rol fusionada con merge commit en la de seguimiento
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `ancestry`

#### Scenario: Merge por squash detectado por la capa `equivalent_tree`
- **DADO** la rama `feat/x` integrada en `main` mediante un squash y `main` con commits posteriores
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `equivalent_tree`
- **Y** el JSON identifica esa capa

#### Scenario: Sin soporte de `merge-tree` la capa se omite
- **DADO** un git que no admite `git merge-tree --write-tree` y una rama integrada con squash
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la capa 2 se omite sin error
- **Y** si no hay evidencia `pr_merged`, `done` exige `--force`

#### Scenario: Evidencia `pr_merged` confirma la fusión
- **DADO** una rama integrada con squash, sin soporte de `merge-tree`, y un `gates.yaml` con evidencia `pr_merged` verificada
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `pr_merged`

#### Scenario: La rama de archivado se confirma por `equivalent_tree` tras un squash
- **DADO** la rama `feat/x-archive` integrada en la rama de integración mediante un squash
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama de archivado se confirma por el token `equivalent_tree`

#### Scenario: La evidencia `pr_merged` no confirma la rama de archivado
- **DADO** la rama `feat/x-archive` sin fusionar, sin confirmación por `equivalent_tree` y un `gates.yaml` con evidencia `pr_merged` del PR del cambio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama de archivado se rechaza con `merge_not_confirmed`

#### Scenario: Nada confirma la fusión
- **DADO** una rama con commits propios sin fusionar, sin confirmación por `equivalent_tree` y sin evidencia
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` para esa rama y se exige `--force`
- **Y** no se elimina su worktree ni su rama local

#### Scenario: La detección no usa la red
- **DADO** un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom branch done --change x` sin `--fetch`
- **ENTONCES** el observador no registra ninguna operación remota

#### Scenario: En modo `branch` la capa `pr_merged` no confirma la fusión en la rama principal
- **DADO** un repositorio con `main_branch: master`, modo `branch`, `feat/x` sin fusionar en `master` y un `gates.yaml` con evidencia `pr_merged`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master`

#### Scenario: En modo `branch` las capas 1 y 2 se evalúan contra la rama principal
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y `feat/x` fusionada en `master` con un squash
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `equivalent_tree` evaluado contra `master`

---

### Requirement: `done`: Seguridad y Eliminación de Worktree y Rama Local (RF-BRA-18)

Para cada rama cuya fusión esté confirmada por la detección por capas de RF-BRA-17, `done` DEBE eliminar su worktree (`git worktree remove`), después eliminar la rama local (también cuando git no la considere fusionada, como ocurre tras un squash, porque la fusión ya está confirmada por las capas) y actualizar el registro local. Si la eliminación del worktree falla, NO DEBE eliminarse la rama. `done` DEBE rechazar la operación sobre una rama cuya fusión no esté confirmada contra el destino que corresponde a su modo (`merge_not_confirmed`, RF-BRA-31, nombrando ese destino) y sobre un worktree con cambios sin confirmar (`dirty_worktree`), salvo `--force`. El cerrojo de la compuerta que deja `axiom sdd gate record` en la carpeta del cambio o en la archivada (`.gates.lock`, un fichero vacío y sin seguimiento, RF-PUB-01) NO DEBE contar como cambio sin confirmar ni aparecer en el rechazo; si es lo único sin confirmar, `done` DEBE retirar el worktree descartándolo. Un `.gates.lock` con contenido sí cuenta como cambio sin confirmar. Con `--force`, DEBE saltarse la exigencia de fusión confirmada y la de árbol limpio, y DEBE informar en el JSON de los cambios sin confirmar descartados y, si la rama no estaba fusionada, de la punta (SHA) que tenía la rama eliminada para poder recuperarla. `done` NUNCA DEBE tocar ramas remotas ni ejecutar operaciones remotas (RF-BRA-01). Cada rama del cambio, incluida la de archivado, DEBE procesarse por separado, sin selector de rol en esta versión: las confirmadas se eliminan y las no confirmadas se conservan y se reportan, y el código de salida DEBE ser distinto de cero si alguna rama no pudo procesarse.

En modo `in_place` no hay worktree que eliminar: `done` DEBE devolver el checkout a la rama de integración (también en modo de promoción `branch`) y, después, eliminar la rama local, y solo si el árbol está limpio. En `in_place`, `--force` solo salta la exigencia de fusión confirmada: NUNCA salta la de árbol limpio, de modo que con cambios sin confirmar `done` rechaza con `dirty_worktree` sin cambiar de rama ni eliminar nada.
(Previously: el `.gates.lock` vacío que deja `gate record` contaba como cambio sin confirmar y `done` se negaba con `dirty_worktree` tras archivar)
(Previously: si quedaba abierto un linaje RDD asociado al worktree, `done` emitía un aviso que nombraba el linaje y `axiom review abandon`; ese aviso desaparece con RDD (C-01-e) y ningún comando `axiom review` existe ya.)

#### Scenario: Eliminación tras una fusión confirmada
- **DADO** una rama `feat/x` fusionada, su worktree con el árbol limpio y una rama remota `origin/feat/x`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el worktree se retira del disco y de `git worktree list`
- **Y** la rama local `feat/x` se elimina
- **Y** la rama remota `origin/feat/x` permanece intacta y no consta ninguna operación remota

#### Scenario: Una rama integrada con squash se elimina tras confirmarse por `equivalent_tree`
- **DADO** `feat/x` integrada en `main` mediante un squash, confirmada por la capa `equivalent_tree` (RF-BRA-17)
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan su worktree y la rama local aunque git no la considere fusionada

#### Scenario: La rama de archivado se elimina como las demás
- **DADO** la rama `feat/x-archive` fusionada en la rama de integración y su worktree con el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan el worktree y la rama local `feat/x-archive`
- **Y** no se toca ninguna rama remota

#### Scenario: Árbol sucio impide la eliminación
- **DADO** una rama fusionada cuyo worktree tiene un fichero modificado sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `dirty_worktree`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: El cerrojo vacío de la compuerta no ensucia el árbol
- **DADO** la rama de archivado fusionada cuyo worktree solo tiene sin confirmar el `.gates.lock` vacío de la carpeta archivada
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se eliminan el worktree y la rama local `feat/x-archive`
- **Y** con otro fichero sin confirmar junto al cerrojo, el rechazo `dirty_worktree` nombra ese fichero y no el cerrojo

#### Scenario: Rama sin fusionar impide la eliminación
- **DADO** una rama con commits propios sin fusionar y sin evidencia, con su worktree limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: `--force` elimina con fusión no confirmada y avisa de lo descartado
- **DADO** una rama sin fusionar y su worktree con cambios sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x --force`
- **ENTONCES** se eliminan el worktree y la rama local
- **Y** el JSON informa de que se descartaron cambios sin confirmar, indica la punta (SHA) de la rama eliminada y la capa de confirmación es el token `forced`
- **Y** no se toca ninguna rama remota

#### Scenario: `done` no emite ningún aviso de linajes
- **DADO** un worktree de una rama fusionada en cuyo git common dir sobrevive un directorio `gentle-ai/` con restos de revisiones antiguas
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la salida no incluye ningún aviso sobre linajes ni cita `axiom review`
- **Y** la eliminación se realiza igual que sin ese directorio

#### Scenario: Ramas de rol y de seguimiento se procesan por separado
- **DADO** `feat/x-core` fusionada en `feat/x` y `feat/x` aún sin fusionar en `main`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el worktree y la rama local `feat/x-core` se eliminan
- **Y** el worktree y la rama local `feat/x` se conservan con `merge_not_confirmed`
- **Y** el código de salida es distinto de cero

#### Scenario: `in_place` con árbol limpio vuelve a la integración y elimina la rama local
- **DADO** un cambio materializado en modo `in_place`, desplegado en `feat/x`, fusionado y con el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio queda desplegado en la rama de integración
- **Y** la rama local `feat/x` se elimina
- **Y** no se toca ninguna rama remota

#### Scenario: `in_place` con árbol sucio se rechaza incluso con `--force`
- **DADO** un cambio en modo `in_place`, desplegado en `feat/x`, con un fichero modificado sin confirmar
- **CUANDO** se ejecuta `axiom branch done --change x` y después `axiom branch done --change x --force`
- **ENTONCES** ambas ejecuciones se rechazan con `dirty_worktree`
- **Y** el repositorio sigue desplegado en `feat/x` con la rama y el fichero intactos

#### Scenario: `in_place` sin fusión confirmada se rechaza
- **DADO** un cambio en modo `in_place`, con el árbol limpio y la rama `feat/x` sin fusionar
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` sin cambiar de rama

#### Scenario: El rechazo en modo `branch` nombra la rama principal
- **DADO** un repositorio con `main_branch: master`, modo `branch` y `feat/x` fusionada solo en la integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el rechazo `merge_not_confirmed` nombra `master` como destino y no `develop`
- **Y** el worktree y la rama local permanecen intactos

#### Scenario: `in_place` en modo `branch` vuelve a la integración al borrar
- **DADO** un cambio en `in_place`, modo `branch`, con `feat/x` fusionada en `master` y el árbol limpio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio queda desplegado en la rama de integración y se elimina la rama local `feat/x`

### Requirement: Veredicto Único de Rama, por Papel del Repositorio (RF-BRA-19)

Las guardas A (RF-BRA-20), B (`blocked(wrong_branch)` de `axiom-sdd-cli-integration`) y D (`guard`, RF-BRA-22) DEBEN consultar un único veredicto: «¿esta rama es la que corresponde a este cambio en este repositorio según su papel?». El veredicto DEBE ser uno de los tokens `correct`, `wrong_branch` o `not_applicable` (RF-BRA-02). La consulta se hace en uno de dos contextos: el **ordinario** y el de **archivado** (la fase archive de `axiom sdd status` y el escritor de `axiom sdd archive-compose`). Las reglas son:

- **Repositorio con rama de cambio, contexto ordinario**: es correcta la rama del cambio o, si existen ramas de rol de ese cambio, cualquiera de ellas; cualquier otra rama (incluida la de integración, la de archivado y un HEAD desacoplado) es `wrong_branch`.
- **Repositorio con rama de cambio, contexto de archivado**: es correcta únicamente la rama de archivado `<prefijo>/<cambio>-archive`; cualquier otra (la del cambio, la de integración, una de rol o una ajena) es `wrong_branch`, y su continuación DEBE ser `axiom branch ensure --cwd <ruta> --change <c> --archive`.
- **Repositorio de specs en multirepo**: es correcta únicamente la rama de integración, también en el contexto de archivado (no hay rama de archivado); una rama de rol del cambio o cualquier otra es `wrong_branch`.
- **Repositorio de código en multirepo**: el veredicto solo se evalúa cuando el consultante es apply (el estado de aplicación de `sdd status` o la guarda D sobre una ruta de ese repositorio); en otras fases es `not_applicable`.
- **Exenciones**: es `not_applicable` para un cambio cuyo `kickoff.yaml` no contiene instantánea (RF-KOF-07), para los sumideros `e2e-cumulative` y `cumulative-<rol>` por nombre, y cuando la ruta no pertenece a ningún repositorio git o git no está disponible (en esos casos la consulta NO DEBE bloquear y DEBE advertirlo).

Los nombres esperados DEBEN obtenerse de la instantánea del `kickoff.yaml` cuando existe (RF-KOF-06) y, para las ramas nombradas por plantilla cuyo nombre no está en la instantánea (las que usan `{task_ref}`), del nombre congelado en el registro local (RF-BRA-15) o, si el registro no lo tiene, de renderizar la plantilla de la instantánea con los vínculos versionados; si el cambio aún no tiene kickoff (se está creando), de la política de `axiom.yaml` (RF-BRA-03). El veredicto NUNCA DEBE recomponer el nombre como `<prefijo>/<cambio>` cuando el cambio usa plantilla, de modo que una rama nombrada por plantilla no se marca como `wrong_branch`. Sin registro, el nombre esperado se obtiene igual que en el plan: el de la instantánea, si no el del registro, si no el renderizado de la plantilla de la instantánea con los vínculos versionados; para las ramas de rol pendientes, el de la plantilla de rol con el `{task_ref}` de ese rol. Un repositorio sin ninguna rama creada y con todas sus ramas pendientes (RF-BRA-46) es `not_applicable` con `Details.reason=awaiting_role`, los roles pendientes y la continuación `axiom branch ensure --cwd <ruta> --change <c> --role <rol>`; si algún rol ya entró, el veredicto se calcula sobre las ramas creadas y las pendientes se informan aparte; el nombre de la rama de archivado se toma del registro local si `ensure --archive` recibió `--prefix` (RF-BRA-12). Un `wrong_branch` DEBE incluir la rama actual, la esperada, el repositorio y la continuación exacta: la orden `axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane` o `--archive`. El veredicto NO consulta si la rama de integración contiene cambios activos: con el archive en un PR documental aparte es normal que los contenga temporalmente (RF-BRA-21).
(Previously: lo consultaban cuatro capas, incluida una de CI; el veredicto no distinguía el contexto de archivado y los valores eran etiquetas en castellano)
(Previously: los nombres esperados salían de la instantánea o de `<prefijo>/<cambio>` y no leían el registro ni conocían las ramas pendientes)

#### Scenario: Rama del cambio en un monorepo es correcta
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en `feat/x`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: La rama de integración es incorrecta para un cambio en monorepo
- **DADO** un cambio con instantánea y el repositorio del planning home desplegado en `main`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `wrong_branch` con rama actual `main` y rama esperada `feat/x`
- **Y** la continuación es la orden exacta de `axiom branch ensure --cwd <ruta> --change x`

#### Scenario: Una rama de rol es correcta cuando hay ramas de rol
- **DADO** un multirepo con los roles `core` y `qa` sobre el mismo repositorio de código y ese repositorio desplegado en `feat/x-qa`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: La rama de archivado es incorrecta en el contexto ordinario
- **DADO** un cambio con instantánea y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `wrong_branch` con rama esperada `feat/x`

#### Scenario: En el contexto de archivado solo la rama de archivado es correcta
- **DADO** un cambio con instantánea y el repositorio desplegado en `feat/x-archive`
- **CUANDO** se consulta el veredicto en el contexto de archivado
- **ENTONCES** es `correct`

#### Scenario: En el contexto de archivado la rama del cambio, la de integración y una ajena son incorrectas
- **DADO** un monorepo con instantánea y el repositorio desplegado, por separado, en `feat/x`, en `main` y en una rama ajena `feat/otra`
- **CUANDO** se consulta el veredicto en el contexto de archivado
- **ENTONCES** es `wrong_branch` en los tres casos con rama esperada `feat/x-archive`
- **Y** la continuación es `axiom branch ensure --cwd <ruta> --change x --archive`

#### Scenario: Un HEAD desacoplado es incorrecto
- **DADO** un cambio con instantánea y el repositorio con HEAD desacoplado
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `wrong_branch` y el detalle indica que el HEAD está desacoplado

#### Scenario: El repositorio de specs de un multirepo exige su rama de integración
- **DADO** un multirepo con un cambio con instantánea y el repositorio de specs desplegado en `feat/x-core`
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `wrong_branch` con rama esperada `main`

#### Scenario: El repositorio de código de un multirepo solo se evalúa en apply
- **DADO** un multirepo con `repo-backend` desplegado en `main` y un cambio con instantánea
- **CUANDO** se consulta el veredicto desde la fase `spec`
- **ENTONCES** es `not_applicable`
- **Y** desde la fase de aplicación de una tarea que toca `repo-backend` es `wrong_branch`

#### Scenario: Un cambio preexistente sin instantánea queda exento
- **DADO** un cambio con `kickoff.yaml` sin instantánea y el repositorio desplegado en `main`
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `not_applicable`

#### Scenario: Los cambios activos en la rama de integración no alteran el veredicto
- **DADO** un cambio `y` fusionado y presente en el árbol de la rama de integración, y un cambio `x` con instantánea en `feat/x`
- **CUANDO** se consulta el veredicto de `x` desplegado en `feat/x`
- **ENTONCES** es `correct`

#### Scenario: Fuera de un repositorio git no se bloquea
- **DADO** un directorio de proyecto que no pertenece a ningún repositorio git
- **CUANDO** se consulta el veredicto
- **ENTONCES** no se bloquea y se emite un aviso



#### Scenario: Una rama nombrada por plantilla es correcta
- **DADO** un cambio con instantánea cuya rama del cambio, nombrada por plantilla, es `inc/2525-alta-de-usuarios`, y el repositorio desplegado en ella
- **CUANDO** se consulta el veredicto en el contexto ordinario
- **ENTONCES** es `correct`

#### Scenario: Una rama con `{task_ref}` se reconoce por el registro
- **DADO** la rama `feature/TASK-1235` creada por `ensure --role frontend`, registrada, con la instantánea que solo guarda su plantilla, y el repositorio desplegado en ella
- **CUANDO** se consulta el veredicto en la fase de aplicación
- **ENTONCES** es `correct`

#### Scenario: La rama de plantilla se reconoce sin registro
- **DADO** el mismo caso con el registro local borrado
- **CUANDO** se consulta el veredicto
- **ENTONCES** es `correct`, porque el nombre se obtiene de renderizar la plantilla de la instantánea con los vínculos versionados

#### Scenario: Una rama pendiente es `not_applicable`
- **DADO** un repositorio de código con su rama pendiente por `{task_ref}` y desplegado en `main`
- **CUANDO** se consulta el veredicto desde la aplicación de una tarea de otro rol
- **ENTONCES** es `not_applicable`
- **Y** el detalle indica `awaiting_role`, nombra el rol pendiente `frontend` y la continuación es `axiom branch ensure --cwd <ruta> --change x --role frontend`

#### Scenario: Con un rol ya dentro el veredicto se calcula sobre las ramas creadas
- **DADO** un repositorio compartido por `core` (rama creada, desplegado en ella) y `qa` (rama pendiente por `{task_ref}`)
- **CUANDO** se consulta el veredicto en la fase de aplicación
- **ENTONCES** es `correct`
- **Y** la rama pendiente de `qa` se informa aparte y no cambia el veredicto

### Requirement: Guarda A en Todo Escritor Go de la Carpeta del Cambio (RF-BRA-20)

Todo escritor Go que cree o modifique ficheros dentro de la carpeta de un cambio DEBE consultar el veredicto de RF-BRA-19 antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir nada y devolver el error tipado con la continuación. Esto incluye como mínimo: la creación de un incremento (el escritor compartido por el dashboard y `axiom change create`), la migración al acumulativo, la creación de handoff (`axiom handoff create` y el servicio del dashboard), `kickoff seal`, el registro de compuertas (`gate record`), el marcador de instancia del consentimiento de autoridad de edición, `axiom archive context prepare` y `axiom archive context verify` (plan y traza del contexto, que consultan el veredicto en el contexto de archivado), `axiom archive adopt scan --change` (inventario de adopción) y `axiom sdd archive-compose` (que consulta el veredicto en el contexto de archivado). El dashboard y `axiom change create` DEBEN ejecutar `ensure` y después escribir con el escritor guardado; el escritor guardado DEBE negarse fuera de rama aunque se invoque sin haber ejecutado `ensure`. Cualquier escritura que alcance la carpeta de un cambio pasa por un escritor guardado. La guarda NO DEBE bloquear cambios exentos (RF-BRA-19). Cada escritor DEBE tener una prueba propia de negativa.
(Previously: la lista incluía `axiom sdd integrate` (`integration.yaml`), retirado; ahora incluye `axiom archive context prepare|verify` y `axiom archive adopt scan --change`)
(Previously: añadía que la TUI solo lista cambios y no ejecuta `ensure`, con un escenario propio.)

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

#### Scenario: Un sumidero acumulativo está exento por nombre
- **DADO** el checkout en la rama de integración
- **CUANDO** se escribe en `openspec/changes/e2e-cumulative` o en `openspec/changes/cumulative-core`
- **ENTONCES** la escritura procede

### Requirement: Cambios Pendientes de Archivar Visibles en `status` y en el Dashboard (RF-BRA-21)

Los cambios pendientes de archivar de un repositorio con rama de cambio, es decir, los cambios activos presentes en el árbol de la rama de integración (directorios bajo `changes/` del planning home distintos de `archive/` y de los sumideros `e2e-cumulative` y `cumulative-<rol>`, excluidos por nombre) que no están en fusión intermedia (RF-BRA-26), DEBEN mostrarse en `axiom branch status` (RF-BRA-16) y en el dashboard (RF-WEB-11 de `local-web-dashboard`). Es información, no una comprobación: NO existe ningún código de fallo por su presencia, NO hay integración con la CI y NO DEBE bloquear ninguna orden ni guarda. La evaluación DEBE hacerse sobre la referencia de integración local de RF-BRA-10 (`origin/<integración>` si existe en local y, si no, la rama local), sin cambiar de rama y sin operaciones remotas; si esa referencia no puede resolverse, la lista se omite con un aviso y sin error. Solo los directorios cuentan como cambios; los ficheros sueltos de `changes/` no lo son. En el repositorio de specs de un multirepo no aplica: allí los cambios activos viven en la rama de integración de forma normal. Cada cambio pendiente DEBE indicar la continuación para archivarlo (`axiom branch ensure --change <c> --archive`, RF-BRA-13).
(Previously: todo cambio activo presente en el árbol de la rama de integración contaba como pendiente de archivar; ahora se excluye el que está en fusión intermedia (RF-BRA-26))

#### Scenario: Un cambio fusionado y sin archivar figura como pendiente
- **DADO** un repositorio con rama de cambio cuya referencia de integración local contiene `openspec/changes/inc-99-ejemplo/`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el JSON lista `inc-99-ejemplo` entre los cambios pendientes de archivar con la continuación `axiom branch ensure --change inc-99-ejemplo --archive`
- **Y** el código de salida es `0`

#### Scenario: Solo `archive/` no genera pendientes
- **DADO** la referencia de integración con `openspec/changes/` que contiene únicamente `archive/`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Los sumideros exentos no son pendientes
- **DADO** la referencia de integración con `openspec/changes/` que contiene `archive/`, `e2e-cumulative` y `cumulative-core`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Un fichero suelto no cuenta como cambio
- **DADO** la referencia de integración con `openspec/changes/` que contiene `archive/` y un fichero `.gitkeep`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Los pendientes no producen código de fallo ni bloquean
- **DADO** un repositorio con dos cambios pendientes de archivar
- **CUANDO** se ejecutan `axiom branch status`, `axiom sdd status` y un escritor guardado en la rama correcta de otro cambio
- **ENTONCES** todas terminan con normalidad y ninguna se bloquea por la presencia de los pendientes

#### Scenario: `status --change` indica que el cambio solicitado está pendiente
- **DADO** el cambio `x` fusionado y presente en la referencia de integración
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON indica que `x` está pendiente de archivar y la continuación `axiom branch ensure --change x --archive`

#### Scenario: El repositorio de specs de un multirepo no tiene pendientes
- **DADO** un multirepo con el repositorio de specs en `main` y un cambio activo en `openspec/changes/`
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar está vacía

#### Scenario: Sin referencia de integración la lista se omite con aviso
- **DADO** un repositorio sin la rama de integración ni local ni como referencia remota local
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el código de salida es `0` y la lista de pendientes se omite con un aviso

#### Scenario: La evaluación no usa la red
- **DADO** un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el observador no registra ninguna operación remota

#### Scenario: Una fusión intermedia no figura como pendiente
- **DADO** la referencia de integración que contiene `openspec/changes/x/` en fusión intermedia (RF-BRA-26) y `openspec/changes/y/` fusionado sin marca de cesión
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** la lista de pendientes de archivar contiene `y` y no contiene `x`
- **Y** el código de salida es `0`

---

### Requirement: `guard`: Hook Opcional que Bloquea los Documentos de Axiom y Avisa en el Resto (RF-BRA-22)

`axiom branch guard` DEBE ser el hook de la capa D que usan los runtimes con hooks. Recibe del runtime la herramienta y la ruta que va a escribir el agente, y su efecto depende de la ruta:

1. **Documentos de Axiom** (la carpeta de un cambio u `odd/tasks/`): DEBE **bloquear** (con el mecanismo de bloqueo del runtime) cuando el veredicto de RF-BRA-19 sea `wrong_branch`, con un mensaje que incluya la continuación de `ensure`. Rige en cualquier proyecto Axiom, con independencia de que haya un cambio registrado (supuesto S-18), y aunque git ignore el documento.
2. **Cualquier otro fichero** de un repositorio con un cambio activo registrado por `ensure` en el registro local (RF-BRA-15) y no cerrado por `done`: DEBE **avisar sin bloquear** cuando la rama del checkout que contiene el fichero no es la que corresponde al cambio. El aviso DEBE incluir la ruta del worktree correcto o, en `in_place`, la rama esperada; con varios cambios activos registrados en el repositorio, nombra los de todos. El repositorio de specs de un `multirepo` queda exento de este aviso. Un fichero que git ignora en su checkout tampoco se avisa: es configuración o resultado local que no viaja con la rama. Un fichero versionado no cuenta como ignorado aunque coincida con un patrón de exclusión.
3. En cualquier otro caso DEBE no hacer nada: sin cambio activo registrado en el repositorio, o fuera de un proyecto Axiom, termina sin bloquear, sin avisar y sin escribir.

DEBE leer la rama actual leyendo `HEAD` del repositorio, sin lanzar git. Mientras las lecturas de ficheros no pidan un aviso (bloqueo, documento de Axiom, sin cambio activo registrado, checkout en la rama de un cambio activo, fuera de un proyecto Axiom), NO DEBE lanzar ningún proceso. Solo cuando las lecturas piden un aviso, y antes de emitirlo, el hook PUEDE lanzar una única consulta local `git check-ignore`, sin red y dentro de su presupuesto de tiempo, para descartar un fichero ignorado; si esa consulta falla, git no está disponible o se agota su límite, el aviso se emite como si el fichero no estuviera ignorado. DEBE ser fail-open: ante cualquier error interno, configuración o registro ilegibles o expiración del presupuesto, NO DEBE bloquear ni avisar. Solo afecta a las escrituras del agente a través del runtime: NUNCA al editor humano ni a las órdenes de la CLI. Es la única capa, junto con la doctrina, que cubre el documento ODD. NO DEBE escribir ningún fichero. El canal no bloqueante por el que cada runtime entrega el aviso se decide en `sdd-design`.
(Previously: no lanzaba ningún proceso en ningún camino y avisaba también de los ficheros ignorados por git)
(Previously: bloqueaba solo las escrituras en la carpeta de un cambio u `odd/tasks/`, no avisaba en otros ficheros y consultaba git para las rutas protegidas)

#### Scenario: Bloquea un documento de Axiom en rama equivocada
- **DADO** un cambio con instantánea y el checkout en la rama de integración
- **CUANDO** el runtime invoca el hook para escribir `openspec/changes/x/proposal.md`
- **ENTONCES** el hook bloquea la escritura
- **Y** el mensaje incluye la continuación `axiom branch ensure --cwd <ruta> --change x`

#### Scenario: Permite un documento de Axiom en la rama correcta
- **DADO** el mismo cambio con el checkout en `feat/x`
- **CUANDO** el runtime invoca el hook para esa escritura
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Avisa sin bloquear en un fichero de código en la rama equivocada
- **DADO** un monorepo con el cambio `x` activo y registrado por `ensure` (worktree en `<raíz de worktrees>/x/<repo>`) y el checkout principal en `main`
- **CUANDO** el runtime invoca el hook para escribir `internal/foo/foo.go` en el checkout principal
- **ENTONCES** el hook no bloquea la escritura
- **Y** emite un aviso que incluye la ruta `<raíz de worktrees>/x/<repo>` como worktree correcto

#### Scenario: Con varios cambios activos el aviso nombra los worktrees de todos
- **DADO** dos cambios activos `x` e `y` registrados en el mismo repositorio y el checkout principal en `main`
- **CUANDO** el runtime invoca el hook para escribir un fichero de código en el checkout principal
- **ENTONCES** el aviso incluye la ruta del worktree de `x` y la de `y`

#### Scenario: En modo `in_place` el aviso nombra la rama esperada
- **DADO** un cambio `x` registrado en modo `in_place` y el repositorio desplegado en `main`
- **CUANDO** el runtime invoca el hook para escribir un fichero de código
- **ENTONCES** el hook no bloquea y el aviso nombra la rama esperada `feat/x`

#### Scenario: Sin aviso si el fichero está en el worktree correcto
- **DADO** el cambio `x` registrado y un fichero de código dentro de su worktree, cuyo `HEAD` es `feat/x`
- **CUANDO** el runtime invoca el hook para escribirlo
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Sin aviso en un fichero ignorado por git
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y un `.gitignore` que excluye `.claude/settings.local.json`
- **CUANDO** el runtime invoca el hook para escribir `.claude/settings.local.json` en el checkout principal
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Un fichero versionado avisa aunque coincida con un patrón de exclusión
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y un fichero versionado que coincide con un patrón de `.gitignore`
- **CUANDO** el runtime invoca el hook para escribirlo en el checkout principal
- **ENTONCES** el hook no bloquea y emite el aviso

#### Scenario: Si la consulta a git falla, el aviso se emite
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y una consulta `git check-ignore` que falla o un git que no está disponible
- **CUANDO** el runtime invoca el hook para escribir un fichero de código en el checkout principal
- **ENTONCES** el hook no bloquea y emite el aviso como si el fichero no estuviera ignorado

#### Scenario: Un documento de Axiom ignorado por git se sigue bloqueando
- **DADO** un proyecto Axiom con el checkout en la rama de integración y un `.gitignore` que excluye la carpeta del cambio `x`
- **CUANDO** el runtime invoca el hook para escribir `openspec/changes/x/proposal.md`
- **ENTONCES** el hook bloquea la escritura sin lanzar ningún proceso

#### Scenario: Sin cambio activo registrado en el repositorio no hace nada con el código
- **DADO** un proyecto Axiom sin ningún cambio activo registrado por `ensure` y el checkout en `main`
- **CUANDO** el runtime invoca el hook para escribir `internal/foo/foo.go`
- **ENTONCES** el hook termina sin bloquear ni avisar

#### Scenario: El repositorio de specs de un multirepo queda exento del aviso
- **DADO** un multirepo con un cambio activo registrado y el repositorio de specs en su rama de integración
- **CUANDO** el runtime invoca el hook para escribir un fichero que no es un documento de Axiom en el repositorio de specs
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Inerte fuera de un proyecto Axiom
- **DADO** un directorio sin `axiom.yaml` ni `.axiom-workspace` que resuelva a uno, en él o en sus ancestros (aunque tenga un `openspec/`)
- **CUANDO** el runtime invoca el hook para cualquier escritura
- **ENTONCES** termina sin bloquear ni avisar
- **Y** no escribe ningún fichero

#### Scenario: Fail-open ante un error interno
- **DADO** un proyecto Axiom con `axiom.yaml` o registro local ilegibles
- **CUANDO** el runtime invoca el hook para escribir en una carpeta de cambio o en un fichero de código
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Sin motivo de aviso no se lanza ningún proceso
- **DADO** un proyecto Axiom y un observador de procesos
- **CUANDO** el runtime invoca el hook para una escritura que se resuelve sin aviso con las lecturas de ficheros: un bloqueo o un documento de Axiom, sin cambio activo registrado, con el checkout en la rama de un cambio activo o fuera de un proyecto Axiom
- **ENTONCES** el observador no registra ninguna invocación de git ni de ningún otro proceso
- **Y** la rama actual se obtiene leyendo `HEAD`

#### Scenario: Antes de avisar, como mucho una consulta `git check-ignore`
- **DADO** el cambio `x` activo y registrado, el checkout principal en `main` y un observador de procesos
- **CUANDO** el runtime invoca el hook para escribir en el checkout principal un fichero de código o un fichero ignorado por git
- **ENTONCES** el hook avisa del fichero de código y no avisa del ignorado
- **Y** en cada escritura el observador registra como mucho una invocación de git, `check-ignore`, local y sin red

#### Scenario: El documento ODD está cubierto
- **DADO** un monorepo con el checkout en la rama de integración y la política por defecto
- **CUANDO** el runtime invoca el hook para escribir `odd/tasks/mi-feature.md`
- **ENTONCES** el hook bloquea con la continuación `axiom branch ensure --lane odd --change mi-feature`

#### Scenario: Solo afecta al agente a través del runtime
- **DADO** un proyecto con el hook instalado
- **CUANDO** una persona guarda un fichero desde su editor o ejecuta una orden de la CLI
- **ENTONCES** el hook no se ejecuta, porque solo lo invoca el runtime en las escrituras del agente y no se instala ningún hook de git ni observador de ficheros

### Requirement: Instalación del Hook por `axiom sync` y Prohibición de Hooks de Git (RF-BRA-23)

`axiom sync` DEBE instalar `axiom branch guard` como hook en los runtimes donde existan hooks demostrados (Claude Code, Codex, OpenCode y VS Code Copilot) de forma idempotente y determinista (misma entrada, mismo resultado), sin duplicar entradas en ejecuciones repetidas y sin alterar las entradas de hooks ajenas. Una ejecución posterior de `axiom sync` que ya no deba instalarlo DEBE retirarlo. NO DEBE instalar hooks de git (`pre-commit`, `pre-push`, ni otros) en ningún repositorio, propio o de terceros. NO DEBE instalarse en Pi, que no tiene canal demostrado. Con el ámbito de instalación `workspace` (el predeterminado) el hook queda en la configuración del proyecto; con `--scope global` queda en la de usuario, que corre en todo repositorio. En ambos casos la inercia de RF-BRA-22 fuera de proyectos Axiom y fuera de repositorios con un cambio activo registrado es condición de esta instalación: el hook no hace nada, no lanza procesos y no escribe en ellos. Un proyecto Axiom es el que tiene un `axiom.yaml` propio o el que se alcanza mediante un puntero `.axiom-workspace`, en el directorio o en un ancestro; un `openspec/` suelto no basta. El formato del hook de Codex se verifica en su documentación oficial al implementar (A-2). En VS Code Copilot el hook se instala solo con el ámbito `workspace`, en un fichero propio de Axiom (`.github/hooks/axiom-branch-guard.json`, con el formato del arnés de Copilot) que no altera los demás ficheros de `.github/hooks`. Como en `preToolUse` de ese arnés cualquier código de salida distinto de 0 deniega la escritura, la orden instalada DEBE terminar con código 0 aunque `axiom` falte o falle, y la decisión DEBE ir solo por la salida estándar.
(Previously: los runtimes con hook eran Claude Code, Codex y OpenCode)
(Previously: la inercia exigida era solo fuera de proyectos Axiom)
(Previously: la inercia se justificaba solo por la configuración global de Claude Code y un `openspec/` suelto valía como marcador de proyecto Axiom)

#### Scenario: Instalación en los runtimes con hooks
- **DADO** un proyecto con Claude Code, Codex, OpenCode y VS Code Copilot configurados
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** cada uno incluye una entrada de hook que invoca `axiom branch guard`

#### Scenario: Sincronizar dos veces no duplica
- **DADO** la instalación anterior
- **CUANDO** se ejecuta `axiom sync` por segunda vez
- **ENTONCES** la configuración de cada runtime es byte a byte idéntica a la anterior

#### Scenario: No se instalan hooks de git
- **DADO** un proyecto con repositorios
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** ningún repositorio contiene hooks de git nuevos

#### Scenario: Las entradas ajenas se conservan
- **DADO** un runtime con hooks propios del usuario
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** los hooks del usuario permanecen sin alterar

#### Scenario: Pi queda fuera
- **DADO** Pi configurado como runtime
- **CUANDO** se ejecuta `axiom sync`
- **ENTONCES** no se instala el hook en Pi y la ejecución termina con normalidad

#### Scenario: En un repositorio ajeno la instalación global es inerte
- **DADO** el hook instalado en la configuración global de Claude Code y un repositorio de terceros sin `axiom.yaml` ni puntero `.axiom-workspace` resoluble (aunque tenga un `openspec/`)
- **CUANDO** el runtime invoca el hook para escribir en ese repositorio
- **ENTONCES** el hook no bloquea, no avisa, no lanza procesos y no escribe nada

#### Scenario: VS Code Copilot recibe un fichero de hooks propio que nunca deniega por fallo
- **DADO** un proyecto con VS Code Copilot y un fichero de hooks del equipo en `.github/hooks/`
- **CUANDO** se ejecuta `axiom sync` con el ámbito `workspace`
- **ENTONCES** `.github/hooks/axiom-branch-guard.json` declara `axiom branch guard --agent vscode-copilot` en `preToolUse` y en `postToolUse` para las herramientas de escritura
- **Y** la orden termina con código 0 aunque `axiom` no esté en el `PATH`
- **Y** el fichero de hooks del equipo queda byte a byte igual

#### Scenario: Con el ámbito global VS Code Copilot no recibe hooks
- **DADO** VS Code Copilot configurado
- **CUANDO** se ejecuta `axiom sync --scope global`
- **ENTONCES** no se escribe ningún fichero de hooks de VS Code Copilot

### Requirement: Las Skills y la Doctrina Delegan en el Verbo (RF-BRA-24)

Las skills `sdd-apply`, `sdd-spec`, `sdd-design`, `sdd-propose`, `sdd-explore` y `sdd-archive`, la convención compartida `_shared` y los orquestadores por runtime DEBEN delegar en `axiom branch` la creación de ramas y worktrees. En concreto: (1) el Paso 2c de `sdd-apply` (cuestionario de worktree y `git worktree add` manual) y las reglas ODD-5.1, ODD-5.2, ODD-5.5 y ODD-5.6 («specs siempre en main», `git commit` y `git push` previos al handoff, `scripts/cleanup-worktree.sh`) DEBEN retirarse y sustituirse por la invocación de `axiom branch ensure`/`done`; (2) `sdd-spec` y `sdd-design` NO DEBEN prescribir trabajar en `main`/`master` ni prohibir ramas y worktrees; (3) `sdd-apply` DEBE proponer la rama destino del PR a partir del plan (RF-BRA-04) en lugar de preguntarla; (4) las skills DEBEN usar las rutas absolutas del JSON de `ensure` con `--cwd`, `git -C` y rutas absolutas, nunca el directorio de trabajo heredado; (5) las rutas y literales de rama de integración de las skills DEBEN referirse a la rama de integración resuelta y no al literal `main`; (6) `sdd-archive` DEBE indicar que, en monorepo, el archive se hace en la rama de archivado creada con `axiom branch ensure --change <c> --archive` una vez fusionado el PR del cambio (RF-ARC-07).

#### Scenario: Retirada del cuestionario de worktree y de ODD-5.x
- **DADO** el texto de `sdd-apply/SKILL.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** no contiene el cuestionario de worktree del Paso 2c, ni `git worktree add`, ni las reglas ODD-5.1, ODD-5.2, ODD-5.5 ni ODD-5.6
- **Y** contiene la invocación de `axiom branch ensure`

#### Scenario: Spec y design ya no imponen `main`
- **DADO** los textos de `sdd-spec/SKILL.md` y `sdd-design/SKILL.md`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no instruyen a operar siempre en `main`/`master` ni prohíben ramas o worktrees
- **Y** remiten a `axiom branch ensure` para la rama de la topología

#### Scenario: `sdd-apply` propone la rama destino del PR
- **DADO** el texto de `sdd-apply/SKILL.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** instruye a proponer la rama destino del PR a partir del plan de `axiom branch status` en lugar de preguntarla

#### Scenario: Rutas absolutas y `--cwd`
- **DADO** los textos de las skills SDD y de `_shared/sdd-phase-common.md`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** instruyen a usar `--cwd`, `git -C` y rutas absolutas tomadas del JSON de `ensure`

#### Scenario: No quedan literales `main` donde debe ir la integración
- **DADO** los textos de las skills SDD y los literales de rama de integración del motor (el valor por defecto de `pr_merged` de `kickoff` y el valor del dashboard)
- **CUANDO** se inspeccionan
- **ENTONCES** las skills se refieren a la rama de integración resuelta
- **Y** el motor usa la rama de integración resuelta de la política en lugar de un literal `main`, que solo permanece como valor por defecto de la política (RF-BRA-03)

#### Scenario: `sdd-archive` remite a la rama de archivado
- **DADO** el texto de `sdd-archive/SKILL.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica crear la rama de archivado con `axiom branch ensure --change <c> --archive` tras fusionar el PR del cambio y ejecutar allí el archive

---

### Requirement: Dogfooding en este Repositorio (RF-BRA-25)

El repositorio de Axiom DEBE declarar en su `axiom.yaml` el bloque `git:` con `integration_branch: main` y `branch_prefix: inc`, y DEBE superar `axiom workspace validate`. `CLAUDE.md`, `AGENTS.md` y `GEMINI.md` DEBEN remitir a `axiom branch` para crear ramas y worktrees y NO DEBEN decir que el worktree se decide en apply. El destino de `scripts/axiom-worktree.ps1` y `.sh` y de `scripts/claude-hooks/guard-primary-checkout.ps1` (envoltorio fino o retirada) se resuelve en `sdd-design` (Q6). Los scripts `scripts/axiom-worktree.ps1` y `.sh` NO DEBEN fijar `main` a mano en la subida de la rama principal local tras `done`: DEBEN tomar la rama de integración resuelta del `repositories[].integrationBranch` del `axiom branch status --json` y, si no hay un JSON legible, caer a `main`; con la configuración actual de este repositorio `new`, `pr` y `done` DEBEN seguir funcionando igual. El producto no añade ninguna exención para cambios preexistentes: un cambio activo ya fusionado, como `inc-23-knowledge-profile` mientras no se archive, aparece como cambio pendiente de archivar (RF-BRA-21) sin bloquear nada.
(Previously: los scripts de dogfooding fijaban `main` a mano tras `done`)

#### Scenario: `axiom.yaml` de este repositorio declara `git:`
- **DADO** el `axiom.yaml` del repositorio de Axiom
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** reporta `COMPLIANT`
- **Y** el bloque `git:` declara `integration_branch: main` y `branch_prefix: inc`

#### Scenario: La documentación remite al verbo
- **DADO** los textos de `CLAUDE.md`, `AGENTS.md` y `GEMINI.md`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** cada uno remite a `axiom branch ensure`
- **Y** ninguno afirma que el worktree se crea en apply

#### Scenario: Un cambio preexistente ya fusionado figura como pendiente de archivar
- **DADO** el cambio `inc-23-knowledge-profile` presente en la referencia de integración de este repositorio sin archivar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** figura entre los cambios pendientes de archivar y el código de salida es `0`

#### Scenario: `done` usa la integración resuelta del `status`
- **DADO** un `status --json` con `repositories[0].integrationBranch: develop` y el checkout principal en `develop`
- **CUANDO** se ejecuta `scripts/axiom-worktree.ps1 done <slug>` o `scripts/axiom-worktree.sh done <slug>`
- **ENTONCES** tras `axiom branch done` el script avanza `develop` con `--ff-only` y no `main`

#### Scenario: Sin JSON legible `done` cae a `main`
- **DADO** un `status --json` ausente o ilegible
- **CUANDO** se ejecuta `done <slug>` en cualquiera de los dos scripts
- **ENTONCES** el script usa `main` como hasta ahora y no falla por ese motivo

#### Scenario: Con la configuración actual los scripts siguen igual
- **DADO** este repositorio con `integration_branch: main` y sin `main_branch`
- **CUANDO** se ejecutan `new`, `pr` y `done` de ambos scripts
- **ENTONCES** su comportamiento y su salida no cambian
### Requirement: Fusión Intermedia de un Cambio no es Integración ni Pendiente de Archivar (RF-BRA-26)

Una **fusión intermedia** es el estado de un cambio cuya carpeta `changes/<cambio>/` está presente en el árbol de la rama de integración porque se fusionó una PR que llevaba una aprobación con cesión de una parada (RF-GAT-11 de `sdd-block-review-gates`) y el trabajo del cambio aún no está completo. El criterio observable es: el registro de puertas integrado (`gates.yaml`) contiene al menos una compuerta aprobada con la marca de cesión y NO están aprobadas todas las compuertas `role-apply:<rol>` del roster sellado (supuesto A-7). Una rama del cambio **publicada** en `origin` (RF-PUB-03 de `sdd-docs-publication`) que aún no se ha fusionado NO es una fusión intermedia ni una rama integrada: la evaluación sigue haciéndose únicamente sobre la referencia de integración local de RF-BRA-10. Un cambio en fusión intermedia:

- NO DEBE figurar entre los cambios pendientes de archivar de `axiom branch status` ni del dashboard (RF-BRA-21), ni ofrecer la continuación de archivado;
- NO DEBE ser aceptado por `ensure --archive` como base que contiene el cambio fusionado (RF-BRA-13), que rechaza con `change_not_integrated`;
- NO DEBE bloquear ninguna orden ni guarda, y la evaluación sigue haciéndose sobre la referencia de integración local de RF-BRA-10, sin cambiar de rama y sin operaciones remotas;
- PUEDE mostrarse como «en curso (fusión intermedia)», sin código de fallo y sin continuación de archivado.

Si el registro de puertas o el kickoff integrados no pueden leerse, el cambio DEBE tratarse como hasta ahora (supuesto A-8), sin error. Un cambio fusionado sin ninguna marca de cesión NO es una fusión intermedia.

#### Scenario: Un cambio con solo spec cedida y fusionada no figura como pendiente de archivar

- **DADO** un monorepo cuya referencia de integración contiene `openspec/changes/x/` con un `gates.yaml` en el que `spec` está aprobada con la marca de cesión y sin ningún `role-apply` aprobado
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` no figura entre los cambios pendientes de archivar
- **Y** el código de salida es `0`

#### Scenario: Un cambio con design cedido y roles sin aprobar sigue siendo intermedio

- **DADO** la referencia de integración con el cambio `x`, con `spec` aprobada con la salida 3, `design` aprobada con la marca de cesión y los roles `core` y `web` sin `role-apply` aprobado
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** `x` no se indica como pendiente de archivar y no se ofrece la continuación `axiom branch ensure --change x --archive`

#### Scenario: Un rol cedido con otro rol pendiente sigue siendo intermedio

- **DADO** la referencia de integración con el cambio `x`, con `role-apply:core` aprobado con la marca de cesión y `role-apply:web` sin aprobar
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` no figura entre los pendientes de archivar

#### Scenario: Al aprobarse todos los role-apply el cambio deja de ser intermedio

- **DADO** la referencia de integración con el cambio `x` cuyo `gates.yaml` conserva la marca de cesión de `design` y donde todos los `role-apply:<rol>` del roster están aprobados
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` figura entre los cambios pendientes de archivar con la continuación `axiom branch ensure --change x --archive`

#### Scenario: Un cambio fusionado sin ninguna marca de cesión se comporta como hasta ahora

- **DADO** la referencia de integración con el cambio `y` cuyo `gates.yaml` no contiene ninguna marca de cesión (o que no tiene `gates.yaml`)
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `y` figura entre los cambios pendientes de archivar, como antes de este cambio

#### Scenario: Un registro integrado ilegible se trata como hasta ahora

- **DADO** la referencia de integración con el cambio `z` cuyo `gates.yaml` o `kickoff.yaml` no se puede leer
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** el código de salida es `0` y `z` se trata como hasta ahora, sin error

#### Scenario: La fusión intermedia no bloquea nada

- **DADO** un repositorio con un cambio en fusión intermedia
- **CUANDO** se ejecutan `axiom branch status`, `axiom sdd status` y un escritor guardado en la rama correcta de otro cambio
- **ENTONCES** todas terminan con normalidad y ninguna se bloquea por la fusión intermedia

#### Scenario: Quien recoge la cesión crea su rama desde la base fusionada

- **DADO** un cambio en fusión intermedia fusionado en la integración
- **CUANDO** otra persona ejecuta `axiom branch ensure --change x`
- **ENTONCES** `ensure` crea la rama del cambio desde la base y la carpeta `changes/x/` con los artefactos aprobados está disponible
- **Y** no se ejecuta ninguna operación remota

#### Scenario: Una rama publicada sin fusionar no es una fusión intermedia

- **DADO** un cambio `x` cuya rama `inc/x` está publicada en `origin` con una compuerta aprobada con cesión, y cuya carpeta `changes/x/` no está en la referencia de integración local
- **CUANDO** se ejecuta `axiom branch status --json`
- **ENTONCES** `x` no se clasifica como fusión intermedia ni como pendiente de archivar
- **Y** el código de salida es `0` y no se ejecuta ninguna operación remota
### Requirement: `ensure` Adopta la Rama Publicada si Existe como Referencia Remota Local (RF-BRA-27)

Si la rama de un cambio (rama del cambio, rama de rol o rama de archivado) NO existe como rama local y SÍ existe la referencia local `refs/remotes/origin/<rama>`, `axiom branch ensure` DEBE crear la rama local desde esa referencia, con `origin/<rama>` como rama de seguimiento (upstream), en lugar de crearla desde la base de integración, y DEBE materializar su worktree o hacer su checkout según el modo de trabajo (RF-BRA-08). El JSON DEBE indicarlo en la entrada de esa rama de `branches[]` con `createdFrom: remote` y `createdFromRef: refs/remotes/origin/<rama>` (tokens estables en inglés, con su etiqueta legible en castellano); `base.ref`, `base.sha` y `base.source` NO cambian de semántica y siguen describiendo la base de integración. La adopción solo usa lecturas locales: NO DEBE ejecutar `fetch`, `pull` ni `ls-remote` por defecto; solo con `--fetch` explícito y autorizado (RF-BRA-10) se ejecuta el `fetch` previo, y entonces la adopción usa la referencia ya actualizada. Si la rama local ya existe, se reutiliza sin moverla (RF-BRA-07), aunque exista la referencia remota. Si no existe la referencia remota, `ensure` DEBE comportarse como hasta ahora (crear desde la base) y el resultado DEBE sugerir `--fetch` (con la salvedad de que requiere autorización de la persona). La adopción NO DEBE alterar el kickoff sellado ni modificar la referencia remota. En `monorepo-embedded`, una cesión publicada (RF-GAT-11 de `sdd-block-review-gates`) ya NO exige una PR fusionada para que otra persona continúe: basta `git fetch` (o `ensure --fetch`) y `ensure`.

#### Scenario: Adopta la rama publicada sin rama local

- **DADO** un monorepo cuyo remoto `origin` tiene `inc/x` con documentos del cambio, la persona ha hecho `git fetch` y no existe la rama local `inc/x`
- **CUANDO** ejecuta `axiom branch ensure --change x --json`
- **ENTONCES** la rama local `inc/x` apunta al SHA de `origin/inc/x`
- **Y** su upstream es `origin/inc/x`
- **Y** la entrada de `inc/x` en `branches[]` indica `createdFrom: remote` y `createdFromRef: refs/remotes/origin/inc/x`
- **Y** `base.source` sigue describiendo la base de integración
- **Y** los artefactos publicados están disponibles en el worktree

#### Scenario: Sin referencia remota se comporta como hasta ahora

- **DADO** un repositorio sin `refs/remotes/origin/inc/x` ni rama local `inc/x`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `inc/x` se crea desde la base de integración
- **Y** el resultado sugiere `--fetch` indicando que requiere autorización de la persona

#### Scenario: Ningún fetch implícito

- **DADO** un observador instalado en el único punto del producto que ejecuta git
- **CUANDO** se ejecuta `ensure --change x` sin `--fetch`, con y sin referencia remota de la rama
- **ENTONCES** el observador no registra `fetch`, `pull`, `push` ni `ls-remote`
- **Y** el JSON devuelve `fetched: false`

#### Scenario: Con `--fetch` se adopta la referencia actualizada

- **DADO** un remoto con `inc/x` que la persona aún no ha traído y la orden `ensure --change x --fetch` autorizada
- **CUANDO** se ejecuta
- **ENTONCES** se ejecuta el `fetch`, `fetched` es `true` y la rama local se crea desde `origin/inc/x`
- **Y** la entrada de `inc/x` en `branches[]` indica `createdFrom: remote`

#### Scenario: Una rama local existente no se mueve

- **DADO** una rama local `inc/x` con commits propios y una referencia `origin/inc/x` distinta
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama local no cambia de commit
- **Y** la entrada de `inc/x` en `branches[]` no indica `createdFrom: remote`

#### Scenario: Adopta también la rama de rol en multirepo

- **DADO** un multirepo con dos roles sobre el mismo repositorio de código y la referencia remota local `inc/x-core` sin rama local
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama local `inc/x-core` se crea desde su referencia remota
- **Y** su entrada en `branches[]` indica `createdFrom: remote` con su `createdFromRef`

#### Scenario: Adopta también la rama de archivado en monorepo

- **DADO** un monorepo y la referencia remota local `inc/x-archive` sin rama local
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** la rama local `inc/x-archive` se crea desde su referencia remota
- **Y** su entrada en `branches[]` indica `createdFrom: remote` con su `createdFromRef`

#### Scenario: En modo `in_place` se adopta y se hace checkout

- **DADO** un proyecto en modo `in_place`, árbol limpio y `origin/inc/x` presente sin rama local
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se crea `inc/x` desde `origin/inc/x` y se hace su checkout en el propio repositorio

#### Scenario: Idempotencia tras adoptar

- **DADO** un cambio ya adoptado
- **CUANDO** se ejecuta de nuevo `ensure --change x --json`
- **ENTONCES** el JSON es idéntico y no cambia ninguna rama ni worktree

#### Scenario: Quien continúa un relevo no necesita una PR fusionada

- **DADO** una compuerta `spec` aprobada con la salida 2 y publicada en `origin/inc/x`, sin PR fusionada
- **CUANDO** otra persona ejecuta `git fetch` y `axiom branch ensure --change x`
- **ENTONCES** obtiene la rama `inc/x` con los documentos aprobados
- **Y** `axiom sdd status` proyecta la fase siguiente cedida

### Requirement: Modo de Promoción por Repositorio y por Cambio (RF-BRA-28)

Cuando un repositorio tiene dos etapas (`git.main_branch` declarada y distinta de la rama de integración resuelta de ese repositorio), cada repositorio con rama de cambio DEBE seguir, para cada cambio, uno de dos modos de promoción:

- **`branch`**: la rama del cambio sale de la rama principal, entra en la rama de integración sin borrarse y después pasa a la rama principal, momento en que `done` la borra (RF-BRA-31).
- **`release`**: la rama del cambio sale de la rama de integración, entra en ella y `done` la borra. El paso de la integración a la rama principal es la release, ajena al cambio.

El modo DEBE resolverse por repositorio, con esta precedencia de mayor a menor: (1) el modo registrado en el registro local del cambio (RF-BRA-15); (2) el modo copiado en la instantánea del `kickoff.yaml` (RF-KOF-06 de `sdd-lane-kickoff`); (3) la respuesta de la invocación, `--promotion` (RF-BRA-29); (4) `promotion` de la entrada del repositorio en `repositories:`; (5) `promotion` del bloque `git:`; (6) el defecto `branch`. El valor `ask` en (4) o en (5) significa que la persona decide al crear la rama: si ningún origen de mayor precedencia fija el modo, `ensure` DEBE devolver `decision_required` (RF-BRA-29).

Un repositorio sin dos etapas, es decir, sin `git.main_branch` o con `git.main_branch` igual a su rama de integración, NO tiene modo de promoción: `promotion` NO tiene efecto, no se pregunta, no se registra ni se copia, y el repositorio se comporta como hasta ahora. El repositorio de specs de un `multirepo` NO tiene modo de promoción porque no recibe ramas de cambio (RF-BRA-05).

`ensure` DEBE persistir, en el registro local, el modo resuelto de cada repositorio con dos etapas y su origen (el token que indica de qué eslabón de la precedencia procede: respuesta de la persona, indicador de la invocación, ajuste del repositorio, ajuste de `git:`, defecto, instantánea o `legacy`). El registro DEBE ser la fuente operativa del modo en todos los carriles, incluido ODD, que no tiene instantánea. Si el registro no tiene el modo y la instantánea sí, `ensure` DEBE respetar el de la instantánea y registrarlo. El modo registrado NO DEBE perderse en una ejecución posterior de `ensure` ni de `--archive`.

El modo NO DEBE aplicarse con carácter retroactivo (origen `legacy`): (a) un registro que contiene la rama del cambio del repositorio y no contiene modo, o (b) una instantánea sellada sin `main_branch` (RF-KOF-06 de `sdd-lane-kickoff`) cuya rama del cambio ya existe en local, DEBEN resolverse como `release` con el origen `legacy`, es decir, como de una sola etapa a efectos de `done` (el destino es la rama de integración); `ensure` DEBE registrarlo y NO DEBE moverse ninguna rama. Un repositorio cuya rama del cambio aún no existe toma el modo que asigne la política vigente.

#### Scenario: Sin `main_branch` nada cambia
- **DADO** un `axiom.yaml` con `git.integration_branch: develop`, sin `git.main_branch` y con `git.promotion: ask`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama `feat/x` sale de `origin/develop` como hasta ahora
- **Y** no se devuelve `decision_required` por el modo de promoción
- **Y** el registro local no contiene modo de promoción para el repositorio

#### Scenario: `main_branch` igual a la integración equivale a una sola etapa
- **DADO** un `axiom.yaml` con `git.integration_branch: main`, `git.main_branch: main` y `git.promotion: release`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama sale de `origin/main`
- **Y** el repositorio no tiene modo de promoción registrado

#### Scenario: Con `main_branch` y sin `promotion` el modo es `branch`
- **DADO** un `axiom.yaml` con `git.main_branch: master`, `git.integration_branch: develop` y ninguna clave `promotion`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto del repositorio es `branch`
- **Y** el registro local lo guarda con el origen `default`

#### Scenario: El registro prevalece sobre la instantánea
- **DADO** un cambio con el modo `release` registrado y la instantánea del `kickoff.yaml` con el modo `branch`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `release`

#### Scenario: La instantánea prevalece sobre `--promotion` y sobre la política
- **DADO** un cambio sin modo registrado, con la instantánea del `kickoff.yaml` con el modo `branch`, y un `axiom.yaml` con `git.promotion: release`
- **CUANDO** se ejecuta `ensure --change x` sin `--promotion`
- **ENTONCES** el modo resuelto es `branch`
- **Y** el registro lo guarda con el origen de instantánea

#### Scenario: `--promotion` prevalece sobre `repositories[].promotion` y `git.promotion`
- **DADO** un `axiom.yaml` con `git.promotion: release` y, para `repo-backend`, `promotion: release`, y un cambio sin modo registrado ni instantánea
- **CUANDO** se ejecuta `ensure --change x --promotion branch`
- **ENTONCES** el modo resuelto de `repo-backend` es `branch`
- **Y** el registro lo guarda con el origen `flag`

#### Scenario: `repositories[].promotion` prevalece sobre `git.promotion`
- **DADO** un `axiom.yaml` con `git.promotion: branch` y, para `repo-backend`, `promotion: release`
- **CUANDO** se resuelve el modo de `repo-backend` y de `repo-frontend` sin registro, instantánea ni `--promotion`
- **ENTONCES** `repo-backend` resuelve `release` y `repo-frontend` resuelve `branch`

#### Scenario: `git.promotion` prevalece sobre el defecto
- **DADO** un `axiom.yaml` con `git.main_branch: master` y `git.promotion: release`
- **CUANDO** se resuelve el modo sin registro, instantánea, `--promotion` ni entrada de repositorio
- **ENTONCES** el modo resuelto es `release`

#### Scenario: El modo registrado sobrevive a una ejecución posterior
- **DADO** un cambio con el modo `release` registrado y la política posterior cambiada a `git.promotion: branch`
- **CUANDO** se ejecuta de nuevo `ensure --change x`
- **ENTONCES** el modo sigue siendo `release`
- **Y** el JSON es idéntico al de la ejecución anterior (RF-BRA-07)

#### Scenario: Con el registro borrado se respeta la instantánea
- **DADO** un cambio sellado con la instantánea con el modo `branch` y su registro local borrado
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `branch` sin preguntar
- **Y** el registro se reconstruye con ese modo

#### Scenario: Con el registro borrado y sin instantánea `ask` vuelve a preguntar
- **DADO** un carril ODD con `git.promotion: ask`, la rama del cambio ya creada y el registro local borrado
- **CUANDO** se ejecuta `ensure --lane odd --change x`
- **ENTONCES** se devuelve `decision_required` por el modo de promoción
- **Y** la rama existente no se mueve

#### Scenario: Un registro con la rama y sin modo se resuelve como `release` legacy
- **DADO** un cambio registrado antes de esta capacidad, con la rama del cambio en el registro y sin modo de promoción, y un `axiom.yaml` que ahora declara `git.main_branch: master` y `git.integration_branch: develop`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama existente se reutiliza sin moverla (RF-BRA-07)
- **Y** el modo resuelto es `release` con el origen `legacy` y se registra
- **Y** a efectos de `done` el destino es la rama de integración

#### Scenario: Una instantánea sin `main_branch` y con la rama ya creada se resuelve como `release` legacy
- **DADO** un cambio sellado antes de esta capacidad (instantánea sin `main_branch`), sin registro y con la rama del cambio ya existente en local, y un `axiom.yaml` que ahora declara dos etapas
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `release` con el origen `legacy`
- **Y** `kickoff.yaml` no se modifica y la rama no se mueve

#### Scenario: Una instantánea sin `main_branch` y sin la rama creada toma la política vigente
- **DADO** un cambio sellado sin `main_branch` en la instantánea, sin registro y sin la rama del cambio en local, con `git.promotion: branch`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el modo resuelto es `branch` con el origen `git` y la rama sale de la rama principal

#### Scenario: En multirepo cada repositorio de código resuelve su propio modo
- **DADO** un multirepo con `git.main_branch: master`, `git.integration_branch: develop`, `git.promotion: branch` y, para `repo-frontend`, `promotion: release`
- **CUANDO** se ejecuta `ensure --change x` con los roles de ambos repositorios sellados
- **ENTONCES** `repo-backend` resuelve `branch` y `repo-frontend` resuelve `release`
- **Y** el registro local guarda un modo por repositorio

#### Scenario: El repositorio de specs de un multirepo no tiene modo
- **DADO** un multirepo con `git.main_branch: master` y `git.promotion: ask`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el repositorio de specs se trabaja in situ en su rama de integración sin modo de promoción
- **Y** no se pregunta por él

---

### Requirement: `ensure`: Base Efectiva, `ask` y `--promotion` (RF-BRA-29)

La base de la rama del cambio de cada repositorio con dos etapas DEBE ser la **base efectiva** del modo resuelto: la rama principal en `branch` y la rama de integración en `release`. La resolución de la base sigue RF-BRA-10 (`origin/<base efectiva>` si existe en local y, si no, la rama local), y un `--base` explícito (RF-BRA-12) la sustituye. Si no existe la base efectiva, ni local ni como referencia remota local, `ensure` DEBE rechazar con `base_not_found` nombrando esa rama. Un repositorio sin dos etapas conserva como base la rama de integración. La rama de archivado (RF-BRA-13) y las ramas de rol (RF-BRA-04) NO cambian de base: la de archivado sale de la rama de integración y las de rol, de su rama de seguimiento.

`ensure` DEBE aceptar el indicador `--promotion <modo>` y `--promotion <repositorio>=<modo>`, este último repetible, donde `<modo>` es `branch` o `release`. La forma sin repositorio se aplica a todos los repositorios del plan con dos etapas; la forma con repositorio, solo al nombrado. `ask` NO es una respuesta válida de `--promotion`. Un indicador presente pero vacío o un `<modo>` inválido DEBEN rechazarse como uso inválido de la orden sin crear nada. Un `--promotion <repositorio>=<modo>` cuyo `<repositorio>` no nombre ningún repositorio del plan DEBE rechazarse con el código tipado `invalid_name`, sin crear nada, nombrando el repositorio pedido y la lista de nombres válidos del plan. Un `--promotion` que alcance un repositorio sin dos etapas NO tiene efecto y DEBE avisar sin bloquear. Dos indicadores con valores distintos para el mismo repositorio DEBEN rechazarse como uso inválido.

Cuando el modo de un repositorio con dos etapas resuelva a `ask` sin respuesta, `ensure` DEBE devolver el resultado tipado `decision_required`, sin crear nada ni interactuar, con el mismo esquema y mecanismo que el de RF-BRA-08: una pregunta, las respuestas `branch` y `release`, cada una con su etiqueta y su efecto en castellano y la invocación exacta que repite la orden con la respuesta. La pregunta DEBE ser de **un repositorio por llamada**, en el orden del plan, nombrando el repositorio en el detalle del resultado; con varios repositorios pendientes las preguntas son secuenciales. Cada invocación propuesta DEBE conservar las respuestas ya dadas en la orden (`--mode`, `--prefix`, `--branch`, `--base`, `--repo` y los `--promotion` previos). Cuando también falta el modo de trabajo (RF-BRA-08), esa pregunta va primero y las de promoción después. La respuesta DEBE registrarse por cambio y repositorio en el registro local (RF-BRA-15) y las ejecuciones posteriores del cambio, incluida la de `--archive`, DEBEN reutilizarla sin volver a preguntar.

Un modo pedido (por `--promotion`) distinto del registrado, o distinto del copiado en la instantánea cuando no hay registro, DEBE rechazarse con `plan_conflict`, sin crear nada y sin rebasar ni mover la rama: la rama ya sale de una base concreta. Un modo pedido igual al registrado es un no-op. Con la rama ya creada y sin registro ni instantánea, el modo pedido se acepta como declaración, se registra y NO mueve la rama.

#### Scenario: En modo `branch` la rama sale de la rama principal
- **DADO** un repositorio con `git.main_branch: master`, `git.integration_branch: develop`, `git.promotion: branch` y las referencias locales `origin/master` y `origin/develop`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** la rama `feat/x` se crea desde `origin/master`
- **Y** `base.ref` es `origin/master`
- **Y** el destino propuesto del PR de `feat/x` es `develop`

#### Scenario: En modo `release` la rama sale de la integración
- **DADO** el mismo repositorio con `git.promotion: release`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** la rama `feat/x` se crea desde `origin/develop`
- **Y** `base.ref` es `origin/develop`

#### Scenario: Sin referencia remota la base efectiva es la rama principal local
- **DADO** un repositorio sin remoto, con `master` y `develop` locales y modo `branch`
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** `base.ref` es `master` y `base.source` indica la rama local

#### Scenario: Base efectiva inexistente
- **DADO** un repositorio con `develop` pero sin `master` ni `origin/master`, y modo `branch`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `base_not_found` nombrando `master`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: `--base` prevalece sobre la base efectiva
- **DADO** el repositorio en modo `branch` y la invocación `ensure --change x --base origin/develop`
- **CUANDO** se ejecuta
- **ENTONCES** la rama se crea desde `origin/develop`

#### Scenario: `ask` devuelve `decision_required` con `branch` y `release`
- **DADO** un repositorio con `git.main_branch: master`, `git.integration_branch: develop`, `git.promotion: ask` y sin respuesta registrada
- **CUANDO** se ejecuta `ensure --change x --json`
- **ENTONCES** el estado es `decision_required` y el código de salida es distinto de cero
- **Y** las respuestas ofrecidas son `branch` y `release`, cada una con etiqueta, efecto e invocación
- **Y** el detalle nombra el repositorio
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: Repetir con `--promotion` crea la rama en la base correcta
- **DADO** el resultado `decision_required` del escenario anterior
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** la rama `feat/x` se crea desde `origin/develop`
- **Y** el modo `release` queda registrado con el origen de respuesta
- **Y** una ejecución posterior de `ensure --change x` sin `--promotion` no vuelve a preguntar

#### Scenario: `--promotion branch` crea la rama desde la rama principal
- **DADO** el mismo repositorio con `ask` sin respuesta
- **CUANDO** se ejecuta `ensure --change x --promotion branch`
- **ENTONCES** la rama `feat/x` se crea desde `origin/master`

#### Scenario: Un modo distinto del registrado se rechaza con `plan_conflict`
- **DADO** un cambio con la rama ya creada y el modo `branch` registrado
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** se rechaza con `plan_conflict`
- **Y** la rama no se mueve y el registro no cambia

#### Scenario: El mismo modo que el registrado es un no-op
- **DADO** un cambio con la rama creada y el modo `branch` registrado
- **CUANDO** se ejecuta `ensure --change x --promotion branch`
- **ENTONCES** el resultado es idéntico al de la ejecución anterior (RF-BRA-07)

#### Scenario: Un modo distinto del de la instantánea se rechaza
- **DADO** un cambio sin modo registrado, con la instantánea con el modo `branch`
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** se rechaza con `plan_conflict`

#### Scenario: `ask` no es una respuesta válida
- **DADO** un repositorio con dos etapas
- **CUANDO** se ejecuta `ensure --change x --promotion ask`
- **ENTONCES** se rechaza como uso inválido de la orden
- **Y** no se crea ninguna rama ni worktree

#### Scenario: Un `--promotion` vacío se rechaza
- **DADO** un repositorio con dos etapas
- **CUANDO** se ejecuta `ensure --change x --promotion ""`
- **ENTONCES** se rechaza como uso inválido de la orden y no se crea nada

#### Scenario: En multirepo las preguntas son secuenciales y conservan las respuestas
- **DADO** un multirepo con `git.promotion: ask`, los repositorios de código `repo-backend` y `repo-frontend` con roles sellados, el modo de trabajo del hub en `ask` y ninguna respuesta
- **CUANDO** se ejecuta `ensure --change x` y se responde cada pregunta con la invocación propuesta
- **ENTONCES** primero se pregunta por el modo de trabajo y después por `repo-backend` y por `repo-frontend`, una pregunta por llamada
- **Y** la invocación propuesta en cada pregunta repite las respuestas ya dadas
- **Y** tras la última respuesta se crean las ramas de ambos repositorios, cada una en la base de su modo

#### Scenario: `--promotion <repositorio>=<modo>` fija modos distintos por repositorio
- **DADO** un multirepo con dos repositorios de código con dos etapas
- **CUANDO** se ejecuta `ensure --change x --promotion repo-backend=branch --promotion repo-frontend=release`
- **ENTONCES** `repo-backend` sale de su rama principal y `repo-frontend` de su rama de integración

#### Scenario: Un repositorio inexistente en el plan se rechaza
- **DADO** un multirepo sin el repositorio `repo-otro`
- **CUANDO** se ejecuta `ensure --change x --promotion repo-otro=branch`
- **ENTONCES** se rechaza con `invalid_name` nombrando `repo-otro` y listando los nombres válidos del plan
- **Y** no se crea ninguna rama ni worktree

#### Scenario: `--promotion` sobre un repositorio de una sola etapa avisa y no actúa
- **DADO** un repositorio sin `git.main_branch`
- **CUANDO** se ejecuta `ensure --change x --promotion release`
- **ENTONCES** la rama sale de la integración como hasta ahora
- **Y** el resultado avisa de que `--promotion` no tiene efecto, sin bloquear

#### Scenario: La rama de archivado y las de rol no cambian de base
- **DADO** un repositorio en modo `branch` con la rama `feat/x` fusionada en la integración
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** `feat/x-archive` sale de la rama de integración, no de la principal

---

### Requirement: `status`: Siguiente PR hacia la Rama Principal en Modo `branch` (RF-BRA-30)

En un repositorio con dos etapas y modo `branch`, `axiom branch status` DEBE informar de la **siguiente PR** de la rama del cambio. Mientras la rama del cambio no esté fusionada en la rama de integración, el destino propuesto del PR (`prTarget`) DEBE ser la rama de integración. Cuando git, solo con referencias locales y sin ninguna operación remota, confirme que la rama del cambio está fusionada en la rama de integración, por ascendencia o por árbol equivalente (RF-BRA-17, capas 1 y 2), `status` DEBE presentar la siguiente PR con la rama del cambio como cabeza y la rama principal como base, y el destino propuesto pasa a ser la rama principal. Si la comprobación no concluye (por ejemplo, la versión de git no admite la capa de árbol equivalente y la rama no es ascendiente), `status` NO DEBE fingir certeza: DEBE mantener la rama de integración como destino y avisar de que no pudo confirmar la fusión.

Con modo `release` o con una sola etapa, el destino propuesto es siempre la rama de integración y no hay siguiente PR. `status` DEBE exponer, además, de forma aditiva por repositorio, el modo de promoción resuelto y la rama principal; los campos nuevos NO DEBEN alterar los ya existentes. Si el modo del repositorio es `ask` sin respuesta (sin registro ni instantánea), `status` NO DEBE calcular la etapa: DEBE avisar de que el modo está pendiente de decisión y de que `ensure` lo preguntará, sin fallar y sin inventar un modo.

#### Scenario: Antes de fusionar en la integración el destino es la integración
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y la rama `feat/x` sin fusionar en `develop`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de `feat/x` es `develop`
- **Y** no hay siguiente PR
- **Y** el JSON informa del modo `branch` y de la rama principal `master`

#### Scenario: Tras fusionar con merge commit en la integración aparece la siguiente PR
- **DADO** el mismo repositorio con `feat/x` fusionada en `develop` con un merge commit
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la siguiente PR tiene como cabeza `feat/x` y como base `master`
- **Y** el destino propuesto es `master`
- **Y** no consta ninguna operación remota

#### Scenario: Tras fusionar con squash se detecta por árbol equivalente
- **DADO** `feat/x` integrada en `develop` mediante un squash y `develop` con commits posteriores
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la siguiente PR tiene como base `master`

#### Scenario: Si la comprobación no concluye se mantiene la integración con un aviso
- **DADO** un git sin soporte de `git merge-tree --write-tree` y `feat/x` integrada en `develop` con un squash
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto sigue siendo `develop`
- **Y** el resultado avisa de que no se pudo confirmar la fusión en la integración
- **Y** el código de salida es `0`

#### Scenario: En modo `release` no hay siguiente PR
- **DADO** un repositorio con dos etapas, modo `release` y `feat/x` fusionada en `develop`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto es `develop` y no hay siguiente PR

#### Scenario: Sin `main_branch` el resultado no cambia
- **DADO** un repositorio sin `git.main_branch` y `feat/x` fusionada en la integración
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto es la integración, no hay siguiente PR y los campos existentes del JSON no cambian

#### Scenario: Modo `ask` sin respuesta se informa como pendiente
- **DADO** un cambio con la rama creada, `git.promotion: ask`, sin registro ni instantánea
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el JSON no incluye siguiente PR ni etapa
- **Y** el resultado avisa de que el modo está pendiente y de que `ensure` lo preguntará
- **Y** el código de salida es `0`

---

### Requirement: `done`: Destino de la Confirmación de Fusión según el Modo de Promoción (RF-BRA-31)

`done` DEBE confirmar la fusión de la rama del cambio contra el destino que corresponde a su modo: en un repositorio con dos etapas y modo `branch`, la **rama principal**; en modo `release` o con una sola etapa, la **rama de integración**, como hasta ahora. Las ramas de rol se siguen confirmando contra su rama de seguimiento y la rama de archivado contra la rama de integración, en cualquier modo (RF-BRA-13).

En modo `branch`, la confirmación contra la rama principal DEBE usar solo las capas de ascendencia (`ancestry`) y de árbol equivalente (`equivalent_tree`) de RF-BRA-17, o `--force`. La evidencia `pr_merged` NO acredita la fusión en la rama principal, porque el `gates.yaml` identifica el PR del cambio y no su destino: en modo `branch`, la capa `pr_merged` NO se usa. En la rama de integración y en modo `release`, las capas siguen como en RF-BRA-17.

El rechazo `merge_not_confirmed` DEBE nombrar el destino real contra el que se evaluó. En modo `branch`, una rama fusionada solo en la rama de integración NO DEBE borrarse. La tolerancia a squash se mantiene: una rama integrada con squash en la rama principal se confirma por `equivalent_tree` o, si esa capa no concluye, con `--force`. Con el modo `ask` pendiente (sin registro ni instantánea), `done` DEBE confirmar contra la rama principal, el destino más exigente, porque borrar una rama es irreversible; `--force` sigue siendo la salida. El repositorio de specs de un `multirepo` se omite porque no tiene ramas de cambio (RF-BRA-05). El resto del comportamiento de `done` (RF-BRA-18) no cambia.

#### Scenario: En modo `branch` una rama fusionada solo en la integración no se borra
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `branch` y `feat/x` fusionada en `develop` pero no en `master`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master` como destino
- **Y** el worktree y la rama local `feat/x` permanecen intactos

#### Scenario: En modo `branch` una fusión en la rama principal con merge commit se confirma por ascendencia
- **DADO** `feat/x` fusionada en `master` con un merge commit
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `ancestry`
- **Y** se eliminan el worktree y la rama local

#### Scenario: En modo `branch` un squash en la rama principal se confirma por árbol equivalente
- **DADO** `feat/x` integrada en `develop` y después en `master` mediante squashes, y `master` con commits posteriores
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma por el token `equivalent_tree`
- **Y** se eliminan el worktree y la rama local

#### Scenario: En modo `branch` un squash que no concluye exige `--force`
- **DADO** un git sin soporte de `git merge-tree --write-tree` y `feat/x` integrada en `master` con un squash
- **CUANDO** se ejecuta `axiom branch done --change x` y después `axiom branch done --change x --force`
- **ENTONCES** la primera se rechaza con `merge_not_confirmed` nombrando `master`
- **Y** la segunda elimina la rama con la capa `forced` e informa de la punta (SHA)

#### Scenario: En modo `branch` la evidencia `pr_merged` no confirma la fusión en la rama principal
- **DADO** `feat/x` fusionada solo en `develop`, sin confirmación por `equivalent_tree` en `master` y un `gates.yaml` con evidencia `pr_merged` del PR del cambio
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master`

#### Scenario: En modo `release` la rama se borra al fusionarla en la integración
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, modo `release` y `feat/x` fusionada en `develop`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma contra `develop` y se eliminan el worktree y la rama local

#### Scenario: Sin `main_branch` el destino sigue siendo la integración
- **DADO** un repositorio sin `git.main_branch` y `feat/x` fusionada en la integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** la rama se confirma y se elimina como hasta ahora, incluida la capa `pr_merged`

#### Scenario: En multirepo `done` omite el repositorio de specs y evalúa cada repositorio con su modo
- **DADO** un multirepo con `repo-backend` en modo `branch` fusionado en su rama principal y `repo-frontend` en modo `release` fusionado en su integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el repositorio de specs se omite sin rama ni worktree que eliminar
- **Y** se eliminan las ramas de `repo-backend` y de `repo-frontend`

#### Scenario: Con `ask` pendiente `done` confirma contra la rama principal
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop`, `git.promotion: ask`, sin registro ni instantánea, y `feat/x` fusionada solo en `develop`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** se rechaza con `merge_not_confirmed` nombrando `master`
- **Y** la rama se conserva
- **Y** con `--force` se elimina con la capa `forced` e informa de la punta (SHA)

#### Scenario: Las ramas de rol y de archivado conservan su destino
- **DADO** un repositorio en modo `branch` con `feat/x-core` fusionada en `feat/x` y `feat/x-archive` fusionada en la integración
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** `feat/x-core` se confirma contra `feat/x` y `feat/x-archive` contra la rama de integración

---

### Requirement: Contratos de Integración y Publicación que No Cambian con el Modo (RF-BRA-32)

El modo de promoción NO DEBE alterar la semántica de la rama de integración ni los destinos de otros verbos, que siguen refiriéndose a la **primera etapa**:

- `IntegrationBranch` conserva su significado: la rama a la que se dirige el PR de la rama del cambio mientras no esté fusionada en ella (RF-BRA-03).
- El destino de publicación de `axiom sdd publish` (`PublicationTarget`) y el del relevo (`HandOff`) siguen siendo los del plan estático, es decir, la rama de integración, y NO adoptan la siguiente PR de `status` (RF-BRA-30).
- La rama de archivado (RF-BRA-13) sale de la rama de integración y `ensure --archive` solo espera a que el cambio esté fusionado en ella, nunca en la rama principal.
- Los repositorios de specs de `multirepo` y las ramas de rol no cambian.

#### Scenario: `IntegrationBranch` no cambia de significado
- **DADO** un repositorio con `main_branch: master`, `integration_branch: develop` y modo `branch`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** `integrationBranch` del repositorio es `develop`

#### Scenario: La publicación y el relevo siguen apuntando a la integración
- **DADO** un repositorio en modo `branch` con `feat/x` ya fusionada en `develop`, de modo que `status` presenta la siguiente PR hacia `master`
- **CUANDO** se calculan el destino de `axiom sdd publish` y el del relevo
- **ENTONCES** ambos siguen siendo los de la primera etapa, calculados con la rama de integración

#### Scenario: El archivado sigue esperando solo a la integración
- **DADO** un repositorio en modo `branch` con el cambio `x` fusionado en `develop` y aún no en `master`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea `feat/x-archive` desde `develop` sin devolver `change_not_integrated`

#### Scenario: El archivado se rechaza si el cambio no está en la integración aunque esté en la rama principal
- **DADO** un repositorio en modo `branch` cuya base `develop` no contiene `openspec/changes/x/`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `change_not_integrated` nombrando `develop`

### Requirement: `ensure --role` Validado contra el Roster Sellado (RF-BRA-33)

`axiom branch ensure` DEBE aceptar la bandera opcional `--role <id>`. Solo es válida en el carril `sdd`, es incompatible con `--archive` y DEBE referirse a un rol presente en el kickoff sellado del cambio. En caso contrario DEBE rechazar con un error tipado y código distinto de cero ANTES de mutar nada (ni ramas, ni worktrees, ni registro), con los códigos `plan_conflict` (con `--archive`), `role_not_applicable` (fuera del carril `sdd`), `kickoff_not_sealed` (sin kickoff sellado) y `role_not_in_roster` (rol fuera del roster, con el detalle de los roles sellados); un `--role` en blanco es error de uso (salida 2). Además, cuando el cambio tiene un vínculo `kind: work_item` en `integrations.yaml` y hay activo un plugin con la capacidad `items.create` (RF-INT-02), DEBE rechazar con el código tipado `task_not_bound` si el rol no tiene vínculo `kind: task`, también ANTES de mutar nada y tras las comprobaciones anteriores; la resolución DEBE nombrar `axiom integrations tasks create --change <cambio>` y `axiom integrations tasks bind --change <cambio> --role <rol>=<id>`. Un cambio sin vínculo `kind: work_item`, o sin plugin activo con `items.create`, NO DEBE recibir `task_not_bound`. Sin `--role`, el comportamiento y el JSON de `ensure` NO DEBEN cambiar. El motor de `ensure` DEBE recibir el rol de `--role` únicamente para crear las ramas pendientes de ese rol (RF-BRA-46). Con `--role`, el plan, las ramas creadas y el JSON del motor NO DEBEN diferir de los de la misma invocación sin la bandera, salvo en las ramas de ese rol cuya plantilla usa `{task_ref}` y que estaban pendientes: esas se crean y dejan de listarse como pendientes. Sin plantillas con `{task_ref}`, la invariancia se mantiene íntegra.

(Placement: after RF-BRA-27)

(Previously: sin la comprobación `task_not_bound`; el rechazo del rol solo dependía del carril, de `--archive` y del roster sellado)
(Previously: el motor no conocía el rol y, con `--role`, el plan, las ramas y el JSON nunca diferían de los de la invocación sin la bandera)

#### Scenario: Rol sellado
- **DADO** un kickoff sellado con el rol `backend` y sin plantillas con `{task_ref}`
- **CUANDO** se ejecuta `axiom branch ensure --change x --role backend`
- **ENTONCES** `ensure` termina con estado `ready` y el mismo JSON que sin `--role`

#### Scenario: Rol ausente del kickoff
- **DADO** un kickoff sellado con el rol `backend`
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** se rechaza con un error tipado que nombra los roles sellados
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: Incompatible con `--archive`
- **CUANDO** se ejecuta `ensure --change x --archive --role backend`
- **ENTONCES** se rechaza con un error tipado y no se muta nada

#### Scenario: Fuera del carril `sdd`
- **CUANDO** se ejecuta `ensure --lane odd --change f --role backend`
- **ENTONCES** se rechaza con un error tipado y no se muta nada

#### Scenario: Sin kickoff sellado
- **DADO** un cambio sin `kickoff.yaml`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se rechaza con un error tipado que indica que el cambio no tiene roles sellados y no se muta nada

#### Scenario: El motor recibe el rol solo para las ramas pendientes
- **DADO** el rol `frontend` con una rama pendiente por `{task_ref}` y el rol `backend` sin plantilla con `{task_ref}`
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** se crea la rama pendiente de `frontend`
- **Y** el plan de las demás ramas es idéntico al de `ensure --change x` sin la bandera

#### Scenario: Con `--role` el plan solo difiere en las ramas pendientes de ese rol
- **DADO** un plan con la rama pendiente de `frontend` y la de `backend`
- **CUANDO** se comparan `ensure --change x` y `ensure --change x --role frontend`
- **ENTONCES** la única diferencia es la creación de la rama de `frontend`
- **Y** la rama de `backend` sigue pendiente

#### Scenario: Rol sin Task vinculada con tracker activo
- **DADO** un cambio con vínculo `kind: work_item`, un plugin activo con `items.create` y el rol `backend` sin vínculo `kind: task`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se rechaza con `task_not_bound` y código distinto de cero
- **Y** la resolución nombra `axiom integrations tasks create --change x` y `axiom integrations tasks bind --change x --role backend=<id>`
- **Y** no se crea ninguna rama, worktree ni registro y el doble no recibe ningún evento

#### Scenario: Con la Task vinculada la barrera se levanta
- **DADO** el rol `backend` con vínculo `kind: task`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** `ensure` termina con estado `ready` y el mismo JSON que sin `--role`

#### Scenario: El rechazo de roster precede a la barrera
- **DADO** un rol fuera del roster sellado en un cambio con tracker activo
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el código del rechazo es `role_not_in_roster`, no `task_not_bound`

#### Scenario: Cambio sin vínculo de WI
- **DADO** un cambio sin vínculo `kind: work_item` y un plugin activo
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** no se rechaza por `task_not_bound`

#### Scenario: Sin plugin activo con `items.create`
- **DADO** un cambio con vínculo `kind: work_item` y sin plugin activo (por ejemplo con `integrations.azure-devops.enabled: false`)
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** la salida y el JSON coinciden con los de antes de este cambio y no se rechaza por `task_not_bound`

### Requirement: `ensure --role` Listo Emite `role.start` Idempotente (RF-BRA-34)

Tras un `ensure --role <id>` con estado `ready`, el adaptador CLI DEBE despachar el evento `role.start` (RF-INT-06) con la clave de idempotencia exacta `role.start/<cambio>/<rol>` (RF-INT-07). NO DEBE emitir con un estado distinto de `ready`, sin `--role`, ni desde el dashboard (que llama al motor directamente). Un `ensure --role` repetido NO DEBE duplicar la operación mientras está pendiente (deduplicación por plugin y clave); tras una entrega con éxito la operación sale de la cola, así que la repetición vuelve a despachar la misma clave y la idempotencia recae en el plugin (RF-INT-07). La emisión nunca hace fallar el verbo (RF-INT-10).

(Placement: after RF-BRA-33)

#### Scenario: Primera entrada al rol
- **DADO** el doble configurado y el rol `backend` sellado
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** el doble recibe `role.start` con la clave `role.start/x/backend`

#### Scenario: `ensure` repetido
- **DADO** `role.start/x/backend` ya entregado
- **CUANDO** se repite `ensure --change x --role backend`
- **ENTONCES** `ensure` no cambia nada (idempotente)
- **Y** el evento se vuelve a despachar con la misma clave `role.start/x/backend`, y el plugin, que es idempotente por clave (entrega al menos una vez, RF-INT-07), no duplica el efecto

#### Scenario: Operación pendiente no se duplica
- **DADO** `role.start/x/backend` pendiente en la outbox por un fallo
- **CUANDO** se repite `ensure --change x --role backend`
- **ENTONCES** la outbox sigue teniendo una sola operación con esa clave

#### Scenario: Sin `--role` no se emite
- **DADO** el doble configurado
- **CUANDO** se ejecuta `ensure --change x` sin `--role`
- **ENTONCES** el doble no recibe ningún evento

#### Scenario: Estado distinto de `ready`
- **DADO** un `ensure --role` que termina en `decision_required`
- **CUANDO** el verbo termina
- **ENTONCES** no se emite ningún evento

#### Scenario: El dashboard no emite
- **DADO** el doble configurado
- **CUANDO** el dashboard da de alta un incremento llamando al motor
- **ENTONCES** el doble no recibe ningún evento
### Requirement: `axiom branch scope`: Alcance del Cambio por Repositorio, de Solo Lectura y con Contrato Versionado (RF-BRA-35)

`axiom branch scope --change <cambio> --json` DEBE calcular, para cada repositorio de código de la instantánea sellada del cambio, qué aporta el cambio en ese repositorio, usando **una única rama por repositorio**: la rama del cambio que registra `axiom branch ensure`. DEBE emitir un contrato JSON versionado propio (campo `schema` con su versión; distinto del de `axiom branch status`). Cada entrada de repositorio DEBE informar de: identificador del repositorio; rama del cambio; base del modo de promoción (RF-BRA-28: `main_branch` en modo `branch`, `integration_branch` en modo `release`) y rama de integración; punto de partida; rango de commits y lista de commits propios; lista de ficheros cambiados por esos commits, **sin** los de `openspec/changes/<cambio>/`; si la rama del cambio está fusionada en la rama de integración (sí o no); y un resultado tipado (RF-BRA-37). El contrato DEBE incluir además un resumen a nivel de cambio con la oferta de verificación conjunta (RF-BRA-38).

La orden DEBE ser de solo lectura: NO DEBE ejecutar `fetch` ni ningún acceso a red, NO DEBE crear ni borrar ramas ni worktrees, NO DEBE modificar referencias, índice ni árbol de trabajo, y NO DEBE escribir en `kickoff.yaml`, `gates.yaml` ni en los documentos del cambio. Solo inspecciona commits: el árbol de trabajo, el índice y los ficheros sin seguimiento NO DEBEN entrar nunca en el resultado. El repositorio de especificaciones in situ de `multirepo` DEBE aparecer con resultado `exempt` y sin cálculo.

**Rechazos y salida.** La orden DEBE rechazar con un error tipado y salida `1`, antes de calcular nada y sin mutar el repositorio: cambio del carril `odd` o kickoff sin instantánea de repositorios, con `scope_not_applicable`; cambio inexistente o sin `kickoff.yaml`, con `kickoff_not_sealed`; configuración, kickoff o instantánea ilegibles, con `config_invalid`; y un fallo de git que ningún otro código cubra, con `git_failed`, que no se degrada a un resultado por repositorio. Un uso incorrecto (falta `--change`, nombre inválido, bandera desconocida o `--fetch`) DEBE terminar con código `2`. El código de salida DEBE ser `0` cuando el alcance se calculó, aunque algún repositorio sea `unverifiable` (RF-BRA-02).

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

### Requirement: `scope`: Punto de Partida Estable en Modo `release` y Exclusión de Commits Ajenos (RF-BRA-36)

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

### Requirement: `scope`: Resultado Tipado por Repositorio y Motivos de `unverifiable` (RF-BRA-37)

Cada entrada de repositorio DEBE llevar exactamente un resultado del dominio cerrado `ok`, `empty`, `unverifiable` o `exempt`, y NUNCA DEBE omitirse un repositorio de la instantánea sin una entrada que lo declare.

- `ok`: la rama del cambio existe, tiene commits propios y el alcance se calculó.
- `empty`: la rama del cambio existe y no aporta commits propios, o todos sus commits propios tocan únicamente `openspec/changes/<cambio>/`.
- `unverifiable`: el alcance no puede calcularse con certeza. El motivo DEBE pertenecer a este dominio cerrado, y no puede haber otro:
  - `branch_missing`: la rama del cambio no existe en local ni como `origin/<rama>` ya descargada. El mensaje DEBE decir «la rama ya no está o falta un fetch autorizado». La orden NO DEBE ejecutar `fetch`.
  - `base_missing`: la base del modo o la rama de integración no existen en local ni como referencia remota ya descargada.
  - `fork_point_undetermined`: no puede fijarse el punto de partida (RF-BRA-36).
- `exempt`: el repositorio de especificaciones in situ (RF-BRA-35).

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

### Requirement: `scope`: Fusión en la Integración y Resumen con la Oferta (RF-BRA-38)

El indicador de fusión de cada repositorio DEBE comprobar que la rama del cambio está fusionada en la **rama de integración** (también en modo `branch`, donde la siguiente PR hacia la rama principal no cuenta; RF-BRA-32), usando solo las capas de ascendencia (`ancestry`) y de árbol equivalente (`equivalent_tree`) de RF-BRA-17, sin red y sin la capa `pr_merged`. El resultado de cada repositorio DEBE declarar si está fusionado (sí o no).

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

### Requirement: `ensure --role` Registra el Inicio del Rol la Primera Vez que Queda Listo (RF-BRA-41)

Cuando el cambio tiene el vínculo `kind: task` del rol en `integrations.yaml` y un `ensure --role <id>` termina con estado `ready`, el adaptador CLI DEBE registrar `started_at` (instante actual, RFC 3339 en UTC) en ese vínculo SOLO si aún no lo tiene; un `started_at` ya registrado NO DEBE sobrescribirse, de modo que el reloj de la imputación parte de la primera entrada del rol. El registro DEBE hacerse con la fusión de vínculos bajo el mismo cerrojo que las demás escrituras de `integrations.yaml` (RF-INT-09). NO DEBE registrarse con un estado distinto de `ready`, sin `--role`, sin vínculo de Task del rol ni desde el dashboard. Un fallo al escribir `started_at` NO DEBE hacer fallar el verbo: avisa con un aviso tipado y `time propose` lo rechazará después con `role_not_started` (RF-INT-10). Sin plugin activo con el puerto de Tasks, la salida y los ficheros de `ensure --role` NO DEBEN cambiar.

#### Scenario: Primera entrada al rol
- **DADO** el vínculo `kind: task` del rol `fullstack` sin `started_at`
- **CUANDO** se ejecuta `axiom branch ensure --change x --role fullstack` y termina en `ready`
- **ENTONCES** el vínculo guarda `started_at` con el instante actual

#### Scenario: Reentrada al rol
- **DADO** un vínculo con `started_at` igual a `2026-11-23T09:00:00Z`
- **CUANDO** se repite `ensure --change x --role fullstack`
- **ENTONCES** `started_at` sigue siendo `2026-11-23T09:00:00Z`

#### Scenario: Estado distinto de `ready`
- **DADO** un `ensure --role` que termina en `decision_required`
- **CUANDO** el verbo termina
- **ENTONCES** el vínculo no recibe `started_at`

#### Scenario: Sin `--role`
- **DADO** un vínculo `kind: task` sin `started_at`
- **CUANDO** se ejecuta `ensure --change x` sin `--role`
- **ENTONCES** `integrations.yaml` no cambia

#### Scenario: El dashboard no registra el inicio
- **DADO** un vínculo `kind: task` sin `started_at`
- **CUANDO** el dashboard da de alta un incremento llamando al motor
- **ENTONCES** `integrations.yaml` no cambia

#### Scenario: Fallo al escribir el inicio
- **DADO** un `integrations.yaml` que no se puede escribir
- **CUANDO** se ejecuta `ensure --change x --role fullstack`
- **ENTONCES** el verbo termina con el resultado de `ensure` y un aviso tipado
- **Y** `time propose` posterior rechaza con `role_not_started`

#### Scenario: Sin plugin activo
- **DADO** un workspace sin plugin activo con el puerto de Tasks
- **CUANDO** se ejecuta `ensure --change x --role fullstack`
- **ENTONCES** la salida y los ficheros coinciden con los de antes de este cambio

### Requirement: Plantilla de Nombre de Rama: Motor y Variables (RF-BRA-42)

El sistema DEBE poder renderizar el nombre de una rama a partir de una plantilla de texto (`branch_template`) que contiene variables entre llaves. El renderizado DEBE ser una función pura, sin lectura de ficheros, de git ni de la red: recibe la plantilla y los valores ya resueltos, y devuelve el nombre o un rechazo tipado. Las variables admitidas son exactamente estas siete, y cualquier otra DEBE rechazarse:

| Variable | Valor |
|---|---|
| `{prefix}` | El `branch_prefix` efectivo, incluido el que fija `--prefix` en la invocación. |
| `{kind}` | `fix` cuando se conoce que el cambio tiene perfil bug; si no, `feature`. |
| `{change}` | El nombre del cambio. |
| `{slug}` | El nombre del cambio sin el prefijo del ID de work item (`us-2525-alta-de-usuarios` produce `alta-de-usuarios`). Se quita el prefijo `^<letras>-<ref>-` (letras ASCII, el `{ref}` conocido y un guion), sin distinguir mayúsculas, y solo cuando `{ref}` es conocido. Es igual a `{change}` si no hay `{ref}` conocido, si el nombre no casa con ese prefijo o si al quitarlo quedaría vacío (`us-2525`): `{slug}` nunca es vacío. |
| `{role}` | El identificador del rol al que se atribuye la rama. |
| `{ref}` | El `id` del vínculo `kind: work_item` (por ejemplo `2525`), nunca la referencia externa completa (`ado:2525`). |
| `{task_ref}` | El `id` del vínculo `kind: task` del rol al que se atribuye la rama. |

Cada valor sustituido, salvo `{prefix}`, DEBE ser un componente de rama válido para git (letras ASCII, dígitos, `.`, `_` y `-`); `{prefix}` PUEDE contener `/` y sigue las reglas del prefijo (RF-BRA-03). Un valor que no sea seguro para git NO DEBE sanearse ni sustituirse en silencio: la resolución se rechaza con un error tipado que nombra la variable y el valor (RF-BRA-03 de esta capacidad, RF-BRA-45). El nombre resultante DEBE ser un nombre de referencia git válido.

#### Scenario: Plantilla con prefijo, referencia y slug
- **DADO** la plantilla `{prefix}/{ref}-{slug}`, el prefijo `inc`, el cambio `us-2525-alta-de-usuarios` y el vínculo `work_item` con `id` `2525`
- **CUANDO** se renderiza el nombre de la rama del cambio
- **ENTONCES** el nombre es `inc/2525-alta-de-usuarios`

#### Scenario: `{ref}` es el id del vínculo y no la referencia externa
- **DADO** un cambio con la referencia externa `ado:2525` y el vínculo `work_item` con `id` `2525`
- **CUANDO** se renderiza `{prefix}/{ref}`
- **ENTONCES** el nombre contiene `2525` y no contiene `ado:`

#### Scenario: `{kind}` vale `feature` por defecto y `fix` con perfil bug
- **DADO** la plantilla `{kind}/{change}` y el cambio `x`
- **CUANDO** se renderiza sin perfil bug conocido y de nuevo con perfil bug conocido
- **ENTONCES** los nombres son `feature/x` y `fix/x`

#### Scenario: `{slug}` coincide con `{change}` sin work item
- **DADO** la plantilla `{slug}` y el cambio `plantilla-ramas` sin work item
- **CUANDO** se renderiza
- **ENTONCES** el nombre es `plantilla-ramas`

#### Scenario: `{slug}` no distingue mayúsculas al quitar el prefijo
- **DADO** la plantilla `{slug}`, el `{ref}` `2525` y el cambio `US-2525-alta-de-usuarios`
- **CUANDO** se renderiza
- **ENTONCES** el nombre es `alta-de-usuarios`

#### Scenario: `{slug}` nunca queda vacío
- **DADO** la plantilla `{prefix}/{slug}`, el `{ref}` `2525` y el cambio `us-2525`
- **CUANDO** se renderiza
- **ENTONCES** `{slug}` vale `us-2525` y el nombre es `<prefijo>/us-2525`

#### Scenario: `{slug}` sin prefijo que case vale `{change}`
- **DADO** la plantilla `{slug}`, el `{ref}` `2525` y el cambio `alta-de-usuarios`
- **CUANDO** se renderiza
- **ENTONCES** el nombre es `alta-de-usuarios`

#### Scenario: `{prefix}` admite barras
- **DADO** el prefijo `equipo/inc` y la plantilla `{prefix}/{change}`
- **CUANDO** se renderiza para el cambio `x`
- **ENTONCES** el nombre es `equipo/inc/x`

#### Scenario: `{task_ref}` y `{role}` en una rama de rol
- **DADO** la plantilla `feature/TASK-{task_ref}`, el rol `frontend` y su Task con `id` `1235`
- **CUANDO** se renderiza el nombre de la rama atribuida a `frontend`
- **ENTONCES** el nombre es `feature/TASK-1235`

#### Scenario: Un valor no seguro para git no se sanea
- **DADO** la plantilla `feature/{task_ref}` y una Task con `id` `T#12`
- **CUANDO** se renderiza
- **ENTONCES** se rechaza con un error tipado que nombra `task_ref` y el valor `T#12`
- **Y** no se devuelve ningún nombre

#### Scenario: El nombre resultante inválido se rechaza
- **DADO** la plantilla `{prefix}/{change}/` y el cambio `x`
- **CUANDO** se renderiza
- **ENTONCES** se rechaza con `invalid_name`

### Requirement: Origen de los Valores de las Variables de la Plantilla (RF-BRA-43)

`ensure` y el cálculo de la instantánea (RF-KOF-06 de `sdd-lane-kickoff`) DEBEN compartir el mismo cálculo de valores y obtener cada uno de su fuente, con este orden de precedencia:

- `{ref}`: el valor congelado en la instantánea del kickoff si existe (RF-KOF-26 de `sdd-lane-kickoff`); si no, el `id` del vínculo `kind: work_item` de `integrations.yaml`; si el vínculo aún no existe, el `id` del work item del staging de `change start`, que se lee del git common dir y que la fase de adopción retira después del seal.
- `{kind}`: el valor congelado en la instantánea si existe; si no hay instantánea (el cambio se está creando), `fix` si el staging o el vínculo de `change start` indican el perfil bug y, si no, `feature`; en un cambio sellado con plantilla pero sin `kind` en la instantánea, `feature` (RF-KOF-26).
- `{task_ref}`: el `id` del vínculo `kind: task` del rol al que se atribuye la rama.
- `{change}`, `{slug}`, `{prefix}` y `{role}` salen del nombre del cambio, del `id` del work item, de la política y del rol atribuido.

Cuando hay un plugin de work items activo y `integrations.yaml` no se puede leer, el sistema DEBE fallar cerrado con `integrations_links_invalid` si la plantilla usa cualquiera de `{ref}`, `{slug}`, `{kind}` o `{task_ref}` y su valor no está congelado en la instantánea: no produce ningún nombre con valores incompletos, igual que la barrera `task_not_bound` (RF-BRA-33). Un valor ausente NUNCA DEBE sustituirse por una cadena vacía.

#### Scenario: `{ref}` del staging en el primer `ensure`, antes de la adopción
- **DADO** un `axiom.yaml` con `git.branch_template: "{prefix}/{ref}-{slug}"`, el prefijo `inc`, el staging de `change start` del cambio `us-2525-alta-de-usuarios` con `id` `2525` y sin vínculo `work_item` todavía en `integrations.yaml`
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios`
- **ENTONCES** la rama del cambio es `inc/2525-alta-de-usuarios`
- **Y** no se rechaza por `{ref}` sin valor

#### Scenario: `{ref}` del vínculo cuando existe
- **DADO** el vínculo `work_item` con `id` `2525` en `integrations.yaml` y sin staging
- **CUANDO** se resuelve `{ref}`
- **ENTONCES** vale `2525`

#### Scenario: `{kind}` del perfil bug
- **DADO** un staging o vínculo de `change start` con perfil bug y la plantilla `{kind}/{change}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama del cambio es `fix/x`

#### Scenario: `{ref}` sin work item ni staging se rechaza
- **DADO** la plantilla `{prefix}/{ref}`, un cambio sin vínculo `work_item` ni staging
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado de variable sin valor (RF-BRA-45) que nombra `ref`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: `integrations.yaml` ilegible con plugin activo falla cerrado
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible y una plantilla con `{ref}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con un error tipado con continuación
- **Y** no se crea ninguna rama ni worktree ni se escribe el registro

#### Scenario: `integrations.yaml` ilegible con `{slug}` o `{kind}` sin congelar falla cerrado
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible, una instantánea sin `kind` ni `ref` y la plantilla `{kind}/{slug}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `integrations_links_invalid`
- **Y** no se crea ninguna rama ni worktree ni se escribe el registro

#### Scenario: `integrations.yaml` ilegible no impide usar valores congelados
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible y una instantánea con `kind` y `ref` congelados para la plantilla `{kind}/{ref}-{slug}`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama se nombra con los valores congelados y no se rechaza

#### Scenario: `{task_ref}` del rol sin Task se rechaza
- **DADO** la plantilla `feature/TASK-{task_ref}` para el rol `backend` y ese rol sin vínculo `kind: task`
- **CUANDO** se intenta resolver el nombre de su rama
- **ENTONCES** se rechaza con el código tipado de variable sin valor que nombra `task_ref` y el rol `backend`

### Requirement: Plantilla Efectiva por Rol, Repositorio y Proyecto, y Precedencia de Nombres (RF-BRA-44)

La plantilla efectiva de cada rama DEBE resolverse con la precedencia rol (`roles.<id>.branch_template`) > repositorio (`repositories[].branch_template`) > proyecto (`git.branch_template`). Sin plantilla declarada en ningún nivel, el nombre de cada rama DEBE ser idéntico, byte a byte, al que producía el sistema antes de esta capacidad: `<prefijo>/<cambio>` para la rama del cambio, `<rama del cambio>-<rol>` para la de rol y `<rama del cambio>-archive` para la de archivado, y la instantánea, el registro y el JSON de `ensure` NO DEBEN cambiar.

La atribución de cada rama a un rol y la plantilla que le corresponde son:

- **Rama del cambio de un repositorio de código tocado por un solo rol** (incluido `fullstack`): se atribuye a ese rol; plantilla efectiva rol > repositorio > proyecto, con `{role}` y `{task_ref}` de ese rol.
- **Rama de seguimiento de un repositorio tocado por dos o más roles**: no se atribuye a ningún rol; plantilla efectiva repositorio > proyecto.
- **Rama de rol de un repositorio tocado por dos o más roles**: se atribuye a ese rol; usa su `roles.<id>.branch_template` si la declara y, si no, `<nombre de la rama de seguimiento>-<rol>`.
- **Rama que lleva `openspec/`** (la del cambio en `monorepo-embedded`, la del repositorio de specs en `multirepo`): sin rol; plantilla efectiva repositorio > proyecto, sin las plantillas de rol.
- **Rama de archivado**: sigue derivándose de la rama del cambio ya resuelta (`<rama del cambio>-archive`); no tiene plantilla propia. La rama del cambio ya resuelta es la que sale, por este orden, del registro, de la instantánea o del renderizado, y todos los consumidores del archivado (el plan de archivado, `branch done`, los cambios pendientes de archivar y la autocaptura) DEBEN obtenerla del mismo cálculo. Con plantilla, `--prefix` en `ensure --archive` NO cambia esa derivación; sin plantilla, el resultado es el de siempre.

Para cada rama, la precedencia de su nombre DEBE ser, de mayor a menor: (1) la excepción por cambio `--branch` (RF-BRA-12), que solo sustituye el nombre de la rama del cambio (la de seguimiento en un repositorio con ramas de rol) y es válida también con `--role`, con la misma semántica; si esa rama estaba pendiente por `{task_ref}`, `--branch` le da nombre y deja de estar pendiente, de modo que se crea en ese `ensure`; (2) el nombre de la instantánea del kickoff; (3) el nombre congelado en el registro local (RF-BRA-15); (4) la plantilla efectiva, de la instantánea o, si el cambio no tiene instantánea, de la política; (5) el nombre por defecto. La precedencia se aplica por rama: una instantánea sin nombre para una rama (la que usa `{task_ref}`) deja pasar al registro y, si este no lo tiene, a la plantilla. `--prefix` solo alimenta `{prefix}` y los nombres por defecto: si la plantilla no usa `{prefix}`, no tiene efecto y no da aviso. Las ramas de rol sin plantilla de rol siguen siendo `<rama de seguimiento>-<rol>`. El nombre de la carpeta del worktree NO DEBE cambiar: sigue usando el nombre del cambio. La colisión de nombres DEBE detectarse en el planificador sobre todos los nombres resueltos del repositorio en ese plan, incluidos los congelados del registro; al entrar un rol, su nombre renderizado se compara con los ya registrados del mismo repositorio. Si dos ramas del mismo repositorio resuelven al mismo nombre, el sistema DEBE rechazar con `invalid_name` nombrando ambas ramas y el repositorio, sin crear nada.

#### Scenario: Sin plantilla, nombres idénticos a los actuales
- **DADO** un `axiom.yaml` sin `branch_template` en ningún nivel, el prefijo `feat` y dos roles `core` y `qa` sobre el mismo repositorio de código
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** las ramas son `feat/x`, `feat/x-core` y `feat/x-qa`
- **Y** el JSON y el registro coinciden con los de antes de esta capacidad

#### Scenario: El rol prevalece sobre el repositorio y este sobre el proyecto
- **DADO** `git.branch_template: "{prefix}/{change}"`, `repositories[].branch_template: "feature/{ref}"` para `repo-backend` y `roles.backend.branch_template: "feature/TASK-{task_ref}"` con `backend` como único rol sobre `repo-backend`
- **CUANDO** se resuelve la plantilla efectiva de la rama del cambio de `repo-backend` y de `repo-frontend` (con el rol `frontend` único sobre él y sin plantilla de rol)
- **ENTONCES** la de `repo-backend` es la del rol `backend`
- **Y** la de `repo-frontend` es la del proyecto

#### Scenario: Cada repositorio de un multirepo puede nombrar su rama de forma distinta
- **DADO** un multirepo con `roles.backend.branch_template: "feature/{task_ref}"` y `roles.frontend.branch_template: "feature/TASK-{task_ref}"`, ambos con su Task `1235` vinculada y cada uno sobre su repositorio
- **CUANDO** se ejecuta `ensure --change x --role backend` y después `ensure --change x --role frontend`
- **ENTONCES** `repo-backend` recibe `feature/1235` y `repo-frontend` recibe `feature/TASK-1235`

#### Scenario: La rama de rol sin plantilla de rol deriva de la de seguimiento
- **DADO** `repositories[].branch_template: "{prefix}/{ref}-{slug}"` para el repositorio compartido por los roles `core` y `qa`, y ninguno con plantilla de rol, con `ref` `2525` y slug `alta`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** la rama de seguimiento es `<prefijo>/2525-alta`
- **Y** las ramas de rol son `<prefijo>/2525-alta-core` y `<prefijo>/2525-alta-qa`

#### Scenario: `--branch` prevalece sobre la plantilla
- **DADO** `git.branch_template: "{prefix}/{ref}-{slug}"` y la invocación `ensure --change x --branch hotfix/x`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `hotfix/x`

#### Scenario: `--prefix` alimenta `{prefix}`
- **DADO** `git.branch_template: "{prefix}/{change}"` y la invocación `ensure --change x --prefix fix`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `fix/x`

#### Scenario: `--prefix` no tiene efecto si la plantilla no usa `{prefix}`
- **DADO** `git.branch_template: "feature/{change}"` y la invocación `ensure --change x --prefix fix`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `feature/x`
- **Y** no se emite ningún aviso por el prefijo

#### Scenario: `--branch` prevalece sobre el nombre de la instantánea
- **DADO** un cambio sellado cuya instantánea nombra la rama del cambio `inc/2525-alta-de-usuarios` y la invocación `ensure --change us-2525-alta-de-usuarios --branch hotfix/alta`
- **CUANDO** se planifica
- **ENTONCES** la rama del cambio es `hotfix/alta`

#### Scenario: `--branch` en un repositorio con ramas de rol nombra solo la de seguimiento
- **DADO** un repositorio compartido por los roles `core` y `qa` y la invocación `ensure --change x --branch hotfix/x`
- **CUANDO** se planifica
- **ENTONCES** la rama de seguimiento es `hotfix/x`
- **Y** las ramas de rol siguen siendo `hotfix/x-core` y `hotfix/x-qa`

#### Scenario: `--branch` con `--role` da nombre a la rama pendiente y la crea
- **DADO** `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend`, el rol `frontend` único sobre él con su Task vinculada y la invocación `ensure --change x --role frontend --branch hotfix/front`
- **CUANDO** se ejecuta
- **ENTONCES** `repo-frontend` recibe la rama `hotfix/front` y esa rama deja de estar pendiente
- **Y** el nombre queda registrado

#### Scenario: Instantánea sin nombre y registro con nombre
- **DADO** una instantánea que solo guarda la plantilla `feature/TASK-{task_ref}` de `repo-frontend` y un registro que congela `feature/TASK-1235`, con la plantilla de `axiom.yaml` cambiada después
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** el nombre es `feature/TASK-1235`, el del registro
- **Y** no se renderiza la plantilla de `axiom.yaml`

#### Scenario: Con plantilla, `--prefix` no cambia la rama de archivado
- **DADO** la rama del cambio `inc/2525-alta-de-usuarios` resuelta por plantilla y registrada
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios --archive --prefix fix`
- **ENTONCES** la rama de archivado es `inc/2525-alta-de-usuarios-archive`

#### Scenario: Los consumidores del archivado usan la rama del cambio resuelta
- **DADO** un cambio pendiente de archivar con rama del cambio `inc/2525-alta-de-usuarios` por plantilla, sin instantánea completa ni registro
- **CUANDO** el plan de archivado, `branch done` y `status` (cambios pendientes de archivar) calculan el nombre de su rama
- **ENTONCES** los tres obtienen `inc/2525-alta-de-usuarios` y su archivado `inc/2525-alta-de-usuarios-archive`, del mismo cálculo
- **Y** sin plantilla el resultado es idéntico, byte a byte, al de antes de esta capacidad

#### Scenario: Un nombre ya registrado de otro rol colisiona al entrar un rol
- **DADO** el rol `core` con su rama `feature/7` registrada en un repositorio compartido y el rol `qa` cuya plantilla de rol renderiza `feature/7`
- **CUANDO** se ejecuta `ensure --change x --role qa`
- **ENTONCES** se rechaza con `invalid_name` nombrando ambas ramas y el repositorio
- **Y** no se crea ninguna rama

#### Scenario: El worktree conserva el nombre del cambio
- **DADO** una plantilla que produce la rama `inc/2525-alta-de-usuarios` para el cambio `us-2525-alta-de-usuarios` en modo `worktree`
- **CUANDO** se ejecuta `ensure`
- **ENTONCES** la carpeta del worktree usa el nombre del cambio y no el de la rama

#### Scenario: La rama de archivado deriva de la del cambio con plantilla
- **DADO** una rama del cambio `inc/2525-alta-de-usuarios` resuelta por plantilla
- **CUANDO** se ejecuta `ensure --change us-2525-alta-de-usuarios --archive`
- **ENTONCES** la rama de archivado es `inc/2525-alta-de-usuarios-archive`

#### Scenario: Dos ramas del mismo repositorio con el mismo nombre se rechazan
- **DADO** un repositorio compartido por los roles `core` y `qa` cuyos `roles.core.branch_template` y `roles.qa.branch_template` son `feature/{ref}` y producen el mismo nombre
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando ambas ramas y el repositorio
- **Y** no se crea ninguna rama

### Requirement: Rechazos Tipados de la Resolución de Plantillas (RF-BRA-45)

Todo rechazo de la resolución de plantillas DEBE ser un rechazo tipado de `branch` con código, mensaje y continuación, DEBE producirse antes de mutar nada (ni ramas, ni worktrees, ni registro) y NO DEBE sustituir el valor ausente o inválido por un respaldo silencioso. Los casos son:

- **Variable sin valor** al resolver (`{ref}` sin work item ni staging, `{task_ref}` sin Task del rol, integraciones ilegibles con plugin activo): código tipado propio de variable sin valor, cuyo nombre exacto fija el diseño; la continuación DEBE nombrar la variable, el rol cuando proceda y la orden o la clave de `axiom.yaml` que lo resuelve.
- **Nombre renderizado inválido** o valor de variable no seguro para git: `invalid_name`, nombrando la variable, el valor y la clave de plantilla.
- **Rama compartida por dos o más roles**: la rama de seguimiento de un repositorio tocado por dos o más roles, o la rama que lleva `openspec/`, cuya plantilla efectiva use `{role}` o `{task_ref}`, DEBE rechazarse al planificar con el código `branch_template_not_applicable` (motivo `shared_branch` u `openspec_branch`) que nombra el repositorio y la clave `repositories[].branch_template` como continuación; no existe rama ni nombre alternativo. El mismo rechazo se produce en `axiom sdd kickoff seal`, que planifica, propaga el rechazo y no escribe `kickoff.yaml` (para no sellar una instantánea irresoluble), en `ensure` (SDD y ODD) y en `axiom workspace validate` solo para lo detectable de forma estática (la rama con `openspec/`, RF-TOP-24 de `workspace-topology`).
- **Carril ODD**: `{role}` y `{task_ref}` NO resuelven en ODD, que no tiene roles ni Tasks (RF-BRA-14): su uso en la plantilla aplicable DEBE rechazarse con `branch_template_not_applicable` (motivo `odd_lane`) que nombra la variable y la clave.
- **Nombre que choca con una rama reservada**: un nombre renderizado igual a la rama de integración o a `main_branch` del repositorio DEBE rechazarse con `invalid_name` en el `ensure`, sin crear nada. La colisión con la rama de otro cambio activo queda fuera de esta capacidad como limitación declarada: una plantilla sin variable propia del cambio (`{change}`, `{slug}`, `{ref}` o `{task_ref}`) se rechaza al cargar (RF-TOP-24), que es la mitigación.
- **Plantilla inválida en `axiom.yaml`** (variable desconocida, llaves mal formadas): se rechaza al cargar con `config_invalid` (RF-TOP-24 de `workspace-topology`).

Los códigos nuevos DEBEN figurar con su etiqueta y su continuación en el conjunto de rechazos de producción, de modo que el trinquete de resolución de rechazos los cubra.

#### Scenario: Variable sin valor con continuación
- **DADO** la plantilla `feature/TASK-{task_ref}` y el rol `backend` sin Task vinculada
- **CUANDO** se ejecuta `ensure --change x --role backend` con la barrera `task_not_bound` inactiva
- **ENTONCES** se rechaza con el código tipado de variable sin valor
- **Y** la continuación nombra `task_ref`, el rol `backend` y `axiom integrations tasks bind --change x --role backend=<id>`
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: Nombre renderizado inválido
- **DADO** la plantilla `{prefix}/{change}.lock` y el cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` que nombra la clave de la plantilla y el nombre resultante
- **Y** no se crea nada

#### Scenario: Rama de seguimiento con `{task_ref}`
- **DADO** un repositorio compartido por los roles `core` y `qa` cuya plantilla efectiva de seguimiento es `repositories[].branch_template: "feature/TASK-{task_ref}"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado de rama compartida
- **Y** la continuación nombra el repositorio y `repositories[].branch_template`
- **Y** no se crea ninguna rama

#### Scenario: Rama de seguimiento con `{role}`
- **DADO** un repositorio compartido por dos roles con `git.branch_template: "{prefix}/{role}/{change}"` como plantilla efectiva de seguimiento
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con el código tipado de rama compartida y no se crea nada

#### Scenario: Plantilla con `{role}` en el carril ODD
- **DADO** `git.branch_template: "{prefix}/{role}/{change}"`
- **CUANDO** se ejecuta `ensure --lane odd --change mi-feature`
- **ENTONCES** se rechaza con un código tipado que nombra `{role}` y `git.branch_template`
- **Y** no se crea ninguna rama ni worktree

#### Scenario: El seal rechaza la rama compartida con `{role}` o `{task_ref}`
- **DADO** un repositorio compartido por los roles `core` y `qa` con `git.branch_template: "feature/{task_ref}"` como plantilla de seguimiento
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con `branch_template_not_applicable` (motivo `shared_branch`) y la continuación nombra `repositories[].branch_template`
- **Y** no se escribe `kickoff.yaml`

#### Scenario: La rama con `openspec/` rechaza `{role}` en `ensure`
- **DADO** un `monorepo-embedded` con `git.branch_template: "{prefix}/{role}/{change}"`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `branch_template_not_applicable` (motivo `openspec_branch`) y no se crea nada

#### Scenario: Un nombre igual a la rama de integración se rechaza
- **DADO** `git.integration_branch: develop` y la plantilla `{change}` con el cambio `develop`
- **CUANDO** se ejecuta `ensure --change develop`
- **ENTONCES** se rechaza con `invalid_name` que nombra la rama de integración
- **Y** no se crea ninguna rama

#### Scenario: Un nombre igual a `main_branch` se rechaza
- **DADO** `git.main_branch: master` y la plantilla `{change}` con el cambio `master`
- **CUANDO** se ejecuta `ensure --change master`
- **ENTONCES** se rechaza con `invalid_name` que nombra `main_branch`
- **Y** no se crea ninguna rama

#### Scenario: Cada rechazo lleva continuación
- **DADO** el conjunto de rechazos de producción de `branch`
- **CUANDO** se ejecuta el trinquete de resolución de rechazos
- **ENTONCES** los códigos nuevos de plantilla tienen etiqueta y continuación no vacía

### Requirement: Diferido de las Ramas con `{task_ref}` a la Entrada del Rol (RF-BRA-46)

Una rama cuya plantilla efectiva usa `{task_ref}` DEBE estar **pendiente** hasta que entre el rol al que se atribuye: NO DEBE crearse ni nombrarse en un `ensure` sin `--role`, ni en un `ensure --role <otro rol>`, ni al arrancar el cambio. DEBE crearse en el `ensure --change <c> --role <rol>` de su rol, después de la barrera `task_not_bound` (RF-BRA-33) y de la resolución de `{task_ref}`. El motor de `ensure` DEBE recibir el rol de `--role` solo para ese fin. Una rama cuya plantilla no usa `{task_ref}`, y toda rama sin plantilla, DEBE crearse en el mismo momento en que se crea hoy. Una rama pendiente DEBE listarse como pendiente, con su repositorio, su rol y su plantilla, tanto en la salida de `ensure` como en la de `status` (RF-BRA-16), mediante un campo aditivo del JSON; un binario anterior ignora ese campo. `ensure --role` DEBE ser idempotente: repetirlo no vuelve a crear la rama ni cambia su nombre. La entrada de un rol NO DEBE crear ni nombrar ramas pendientes de otro rol.

`status` NUNCA DEBE mutar nada. Una rama pendiente se lista con su plantilla, su rol y la orden que la crea (`axiom branch ensure --cwd <ruta> --change <c> --role <rol>`), y sin nombre esperado si aún no se puede resolver. Con un plugin de work items activo y `integrations.yaml` ilegible, `status` DEBE informar el código del rechazo (`integrations_links_invalid`) en la entrada afectada, sin fallar la orden entera. La base, el destino de PR y la resolución del modo de promoción de una rama diferida, incluida la pregunta `ask` (RF-BRA-28 y RF-BRA-29), se resuelven en el `ensure --role` que la crea, con la misma regla que si se creara al arrancar el cambio. El worktree de una rama diferida solo se materializa al crearla: mientras tanto, un repositorio sin ninguna rama creada no tiene worktree.

#### Scenario: La rama con `{task_ref}` nace al entrar el rol
- **DADO** un multirepo con `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend`, el rol `frontend` único sobre él y su Task `1235` vinculada, y el cambio ya sellado
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** `repo-frontend` recibe la rama `feature/TASK-1235` desde su base
- **Y** el nombre queda registrado

#### Scenario: Antes de la entrada del rol la rama está pendiente
- **DADO** el mismo multirepo y un `ensure --change x` sin `--role` posterior al sellado
- **CUANDO** termina
- **ENTONCES** la rama de `repo-frontend` no existe ni se ha nombrado
- **Y** el JSON de `ensure` y el de `status` listan una rama pendiente con el repositorio `repo-frontend`, el rol `frontend` y la plantilla `feature/TASK-{task_ref}`
- **Y** las ramas sin `{task_ref}` de los demás repositorios sí se han creado

#### Scenario: La entrada de un rol no toca las ramas pendientes de otro
- **DADO** los roles `backend` y `frontend` con Task vinculada, cada uno con su plantilla con `{task_ref}`
- **CUANDO** se ejecuta `ensure --change x --role backend`
- **ENTONCES** se crea solo la rama de `backend`
- **Y** la de `frontend` sigue pendiente

#### Scenario: Repetir `ensure --role` es idempotente
- **DADO** la rama `feature/TASK-1235` creada por `ensure --change x --role frontend`
- **CUANDO** se repite la misma orden
- **ENTONCES** la rama se reutiliza sin modificarla y su nombre no cambia

#### Scenario: La barrera `task_not_bound` precede a la creación
- **DADO** un plugin activo con `items.create`, un vínculo `work_item` y el rol `frontend` sin Task
- **CUANDO** se ejecuta `ensure --change x --role frontend`
- **ENTONCES** se rechaza con `task_not_bound` y no se crea la rama pendiente

#### Scenario: Segundo clon con `ensure --role`, sin registro y con la rama ya publicada
- **DADO** un cambio sellado con `repositories[].branch_template: "feature/TASK-{task_ref}"` para `repo-frontend`, la Task `1235` del rol `frontend` vinculada y versionada en `integrations.yaml`, la rama `feature/TASK-1235` ya publicada en el remoto por otra persona, y un segundo clon sin registro ni staging
- **CUANDO** se ejecuta `ensure --change x --role frontend` en el segundo clon
- **ENTONCES** el nombre es `feature/TASK-1235`, obtenido de la plantilla de la instantánea y de los vínculos versionados
- **Y** se adopta la rama publicada (RF-BRA-27) sin crear otra ni modificarla
- **Y** el registro se reconstruye con ese nombre

#### Scenario: `status` no muta y lista la rama pendiente con su orden
- **DADO** una rama pendiente de `repo-frontend` para el rol `frontend` cuyo `{task_ref}` aún no se puede resolver
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la entrada pendiente incluye la plantilla, el rol `frontend` y la orden `axiom branch ensure --cwd <ruta> --change x --role frontend`, sin nombre esperado
- **Y** no se crea ninguna rama, worktree ni registro

#### Scenario: `status` con `integrations.yaml` ilegible informa el código sin fallar
- **DADO** un plugin de work items activo, un `integrations.yaml` ilegible y una rama pendiente por `{task_ref}`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** la entrada afectada informa `integrations_links_invalid`
- **Y** el resto del estado se devuelve y la orden no falla

#### Scenario: La promoción `ask` de una rama diferida se resuelve al crearla
- **DADO** `git.main_branch: master`, `git.promotion: ask` y una rama pendiente de `repo-frontend`
- **CUANDO** se ejecuta `ensure --change x` y después `ensure --change x --role frontend`
- **ENTONCES** la primera no pregunta por el modo de promoción de `repo-frontend` ni crea su worktree
- **Y** la segunda resuelve la base, el destino de PR y el modo (incluida la pregunta `ask`) con la misma regla que si la rama se hubiera creado al arrancar

#### Scenario: Sin `{task_ref}` nada se difiere
- **DADO** plantillas sin `{task_ref}` o sin plantillas
- **CUANDO** se ejecuta `ensure --change x` y después `ensure --change x --role backend`
- **ENTONCES** ambas invocaciones producen el mismo plan, las mismas ramas y el mismo JSON
- **Y** el JSON no lista ramas pendientes

## Supuestos confirmados en la compuerta de spec (2026-10-15)

Los supuestos son decisiones mínimas tomadas para que los requerimientos sean comprobables. El usuario los confirmó en la compuerta de spec del 2026-10-15; S-6 se corrigió según D3. Q1, Q2 y Q6 de la propuesta pertenecen a `sdd-design` y se citan como dependencias.

> **Nota de reapertura (2026-10-17):** la spec se reabrió por D16 a D19. D15 queda retirada y S-12 se resuelve ahora por D16. S-4 se actualiza por la lectura G-5 y S-5 queda sin objeto por D17. Los supuestos S-15 a S-18 se añaden en la reapertura.

| ID | Supuesto | Estado | Dónde afecta |
|---|---|---|---|
| S-1 | La propuesta no indica cómo conocer los roles cuando `ensure` se ejecuta al nombrar el cambio, antes de `kickoff seal`. `ensure` es incremental: antes del sellado planifica solo `<prefijo>/<cambio>` en el repositorio del planning home (idéntica en ambos modelos) y tras el sellado completa ramas de rol y repositorios de código. | Confirmado | RF-BRA-07 |
| S-2 | La respuesta a `decision_required` de `ask` se da con la opción `--mode worktree\|in_place` de `ensure`. | Confirmado | RF-BRA-08 |
| S-3 | `fullstack` no existe en `axiom.yaml`. Cuenta como un único rol que toca todos los repositorios declarados por algún rol y, en monorepo, el del planning home. | Confirmado | Términos, RF-BRA-04 |
| S-4 | `<repo>` es el último segmento de la ruta (`path`) del repositorio (G-5), y `<workspace>-wt` usa el nombre de la carpeta maestra. | Confirmado (actualizado por G-5) | RF-BRA-11 |
| S-5 | Retirado por D17: existía para fijar el mecanismo con el que el comprobador de CI evaluaba un árbol desacoplado, y el comprobador se ha eliminado. | Sin objeto (D17) | — |
| S-6 | Resuelto según D3: `done` elimina el worktree y la rama local una vez confirmada la fusión por capas (RF-BRA-17); se niega si la rama no está fusionada o si el árbol está sucio, salvo `--force`; en `in_place` devuelve el checkout a la rama de integración y elimina la rama local solo con el árbol limpio (`--force` no salta esa exigencia); nunca toca ramas remotas; procesa cada rama por separado, incluida la de archivado, sin selector de rol en esta versión. | Confirmado (corregido, D3) | RF-BRA-18 |
| S-7 | Si el cambio ya tiene `kickoff.yaml` pero sin instantánea queda exento; si aún no tiene `kickoff.yaml` se aplica la política de `axiom.yaml`. | Confirmado | RF-BRA-19 |
| S-8 | `--fetch` fallido termina con `fetch_failed` sin crear nada. | Confirmado | RF-BRA-10 |
| S-9 | `blocked(wrong_branch)` se evalúa antes de `blocked(edit_authority_missing)`. | Confirmado | `axiom-sdd-cli-integration` |
| S-10 | La pregunta de `ask` del carril ODD es una preferencia de entorno personal, no una pregunta de gobernanza, y no contraviene RF-KOF-02. | Confirmado | `sdd-lane-kickoff` |
| S-11 | Si el directorio de partida es un worktree enlazado sin marcador ascendente, el planning home se resuelve desde el checkout principal del mismo clon. | Confirmado | `planning-home-resolution` |
| S-12 | Resuelto por D16 (sustituye a la decisión anterior): en monorepo el archive se hace en un PR documental aparte, en la rama de archivado `<prefijo>/<cambio>-archive` (D18), una vez fusionado el PR del cambio. RF-ARC-01 vuelve a su forma original con dos cambios (rama de integración configurada y sin `monorepo-decoupled`) y la precondición incluye la evidencia `pr_merged` del PR del cambio. Archivar en otra rama se rechaza con `blocked(wrong_branch)` y la continuación `ensure --archive`. `multirepo` no cambia. | Confirmado (D16) | `sdd-archive-lifecycle` |
| S-13 | `repositories:` identifica cada entrada por `path` y admite `integration_branch` y `branch_prefix` como ajustes propios por repositorio. | Confirmado (incluye `branch_prefix` por repositorio) | `workspace-topology` |
| S-14 | La TUI se cubre como escenario de paridad en `local-web-dashboard`; `tui-ui-parity` no figura en las capacidades modificadas de la propuesta. | Confirmado | `local-web-dashboard` |
| S-15 | Un rol llamado `archive` colisionaría con la rama de archivado `<prefijo>/<cambio>-archive` y se rechaza con `invalid_name` al planificar las ramas de rol. | Propuesto en la reapertura (2026-10-17) | RF-BRA-04 |
| S-16 | Con `--archive` solo se admite `--prefix`; `--branch` y `--base` se rechazan con `plan_conflict`; sin `--prefix` la rama de archivado deriva de la rama del cambio registrada como `<rama del cambio>-archive`. Una base «contiene el cambio fusionado» cuando su árbol incluye `changes/<cambio>/` del planning home y el cambio no está en fusión intermedia (RF-BRA-26). | Propuesto en la reapertura (2026-10-17); enmendado por RF-BRA-13 y RF-BRA-26 (paradas-tres-salidas) | RF-BRA-12, RF-BRA-13 |
| S-17 | Para el repositorio cuya ruta es `.`, `<repo>` es el nombre del directorio de su raíz git; dos repositorios del plan con el mismo último segmento se rechazan con `invalid_name`. | Propuesto en la reapertura (2026-10-17) | RF-BRA-11 |
| S-18 | D19: el bloqueo de documentos de Axiom (carpeta del cambio y `odd/tasks/`) rige en todo proyecto Axiom aunque no haya cambio registrado, para cubrir el carril ODD antes de `ensure`; el aviso en otros ficheros solo rige en repositorios con un cambio activo registrado; el repositorio de specs de un multirepo queda exento del aviso pero no del bloqueo de documentos. | Propuesto en la reapertura (2026-10-17) | RF-BRA-22 |
| A-7 | Un cambio con todos los `role-apply` aprobados deja de ser fusión intermedia aunque la última puerta lleve marca de cesión (la fase `verify` cedida no hace intermedia la fusión del trabajo ya implementado). | Confirmado (paradas-tres-salidas) | RF-BRA-26 |
| A-8 | Un cambio fusionado sin ninguna marca de cesión en su registro de puertas (incluidos los cambios preexistentes y los sellados en modalidad continua) se comporta exactamente como hasta ahora. | Confirmado (paradas-tres-salidas) | RF-BRA-26 |

### Dependencias de diseño

| ID | Dependencia | Dónde afecta |
|---|---|---|
| Q1 | Versión mínima de git para `git merge-tree --write-tree` (2.38). | RF-BRA-17 (design) |
| Q2 | Coexistencia del registro local con `<git common dir>/gentle-ai/`. | RF-BRA-15 (design) |
| Q6 | Destino de los scripts locales y de `guard-primary-checkout.ps1`. | RF-BRA-25 (design) |


## Supuestos confirmados en la compuerta de spec (2026-10-15)

Los supuestos son decisiones mínimas tomadas para que los requerimientos sean comprobables. El usuario los confirmó en la compuerta de spec del 2026-10-15; S-6 se corrigió según D3. Q1, Q2 y Q6 de la propuesta pertenecen a `sdd-design` y se citan como dependencias.

> **Nota de reapertura (2026-10-17):** la spec se reabrió por D16 a D19. D15 queda retirada y S-12 se resuelve ahora por D16. S-4 se actualiza por la lectura G-5 y S-5 queda sin objeto por D17. Los supuestos S-15 a S-18 se añaden en la reapertura.

| ID | Supuesto | Estado | Dónde afecta |
|---|---|---|---|
| S-1 | La propuesta no indica cómo conocer los roles cuando `ensure` se ejecuta al nombrar el cambio, antes de `kickoff seal`. `ensure` es incremental: antes del sellado planifica solo `<prefijo>/<cambio>` en el repositorio del planning home (idéntica en ambos modelos) y tras el sellado completa ramas de rol y repositorios de código. | Confirmado | RF-BRA-07 |
| S-2 | La respuesta a `decision_required` de `ask` se da con la opción `--mode worktree\|in_place` de `ensure`. | Confirmado | RF-BRA-08 |
| S-3 | `fullstack` no existe en `axiom.yaml`. Cuenta como un único rol que toca todos los repositorios declarados por algún rol y, en monorepo, el del planning home. | Confirmado | Términos, RF-BRA-04 |
| S-4 | `<repo>` es el último segmento de la ruta (`path`) del repositorio (G-5), y `<workspace>-wt` usa el nombre de la carpeta maestra. | Confirmado (actualizado por G-5) | RF-BRA-11 |
| S-5 | Retirado por D17: existía para fijar el mecanismo con el que el comprobador de CI evaluaba un árbol desacoplado, y el comprobador se ha eliminado. | Sin objeto (D17) | — |
| S-6 | Resuelto según D3: `done` elimina el worktree y la rama local una vez confirmada la fusión por capas (RF-BRA-17); se niega si la rama no está fusionada o si el árbol está sucio, salvo `--force`; en `in_place` devuelve el checkout a la rama de integración y elimina la rama local solo con el árbol limpio (`--force` no salta esa exigencia); nunca toca ramas remotas; procesa cada rama por separado, incluida la de archivado, sin selector de rol en esta versión. | Confirmado (corregido, D3) | RF-BRA-18 |
| S-7 | Si el cambio ya tiene `kickoff.yaml` pero sin instantánea queda exento; si aún no tiene `kickoff.yaml` se aplica la política de `axiom.yaml`. | Confirmado | RF-BRA-19 |
| S-8 | `--fetch` fallido termina con `fetch_failed` sin crear nada. | Confirmado | RF-BRA-10 |
| S-9 | `blocked(wrong_branch)` se evalúa antes de `blocked(edit_authority_missing)`. | Confirmado | `axiom-sdd-cli-integration` |
| S-10 | La pregunta de `ask` del carril ODD es una preferencia de entorno personal, no una pregunta de gobernanza, y no contraviene RF-KOF-02. | Confirmado | `sdd-lane-kickoff` |
| S-11 | Si el directorio de partida es un worktree enlazado sin marcador ascendente, el planning home se resuelve desde el checkout principal del mismo clon. | Confirmado | `planning-home-resolution` |
| S-12 | Resuelto por D16 (sustituye a la decisión anterior): en monorepo el archive se hace en un PR documental aparte, en la rama de archivado `<prefijo>/<cambio>-archive` (D18), una vez fusionado el PR del cambio. RF-ARC-01 vuelve a su forma original con dos cambios (rama de integración configurada y sin `monorepo-decoupled`) y la precondición incluye la evidencia `pr_merged` del PR del cambio. Archivar en otra rama se rechaza con `blocked(wrong_branch)` y la continuación `ensure --archive`. `multirepo` no cambia. | Confirmado (D16) | `sdd-archive-lifecycle` |
| S-13 | `repositories:` identifica cada entrada por `path` y admite `integration_branch` y `branch_prefix` como ajustes propios por repositorio. | Confirmado (incluye `branch_prefix` por repositorio) | `workspace-topology` |
| S-14 | La TUI se cubre como escenario de paridad en `local-web-dashboard`; `tui-ui-parity` no figura en las capacidades modificadas de la propuesta. | Confirmado | `local-web-dashboard` |
| S-15 | Un rol llamado `archive` colisionaría con la rama de archivado `<prefijo>/<cambio>-archive` y se rechaza con `invalid_name` al planificar las ramas de rol. | Propuesto en la reapertura (2026-10-17) | RF-BRA-04 |
| S-16 | Con `--archive` solo se admite `--prefix`; `--branch` y `--base` se rechazan con `plan_conflict`; sin `--prefix` la rama de archivado deriva de la rama del cambio registrada como `<rama del cambio>-archive`. Una base «contiene el cambio fusionado» cuando su árbol incluye `changes/<cambio>/` del planning home y el cambio no está en fusión intermedia (RF-BRA-26). | Propuesto en la reapertura (2026-10-17); enmendado por RF-BRA-13 y RF-BRA-26 (paradas-tres-salidas) | RF-BRA-12, RF-BRA-13 |
| S-17 | Para el repositorio cuya ruta es `.`, `<repo>` es el nombre del directorio de su raíz git; dos repositorios del plan con el mismo último segmento se rechazan con `invalid_name`. | Propuesto en la reapertura (2026-10-17) | RF-BRA-11 |
| S-18 | D19: el bloqueo de documentos de Axiom (carpeta del cambio y `odd/tasks/`) rige en todo proyecto Axiom aunque no haya cambio registrado, para cubrir el carril ODD antes de `ensure`; el aviso en otros ficheros solo rige en repositorios con un cambio activo registrado; el repositorio de specs de un multirepo queda exento del aviso pero no del bloqueo de documentos. | Propuesto en la reapertura (2026-10-17) | RF-BRA-22 |
| A-7 | Un cambio con todos los `role-apply` aprobados deja de ser fusión intermedia aunque la última puerta lleve marca de cesión (la fase `verify` cedida no hace intermedia la fusión del trabajo ya implementado). | Confirmado (paradas-tres-salidas) | RF-BRA-26 |
| A-8 | Un cambio fusionado sin ninguna marca de cesión en su registro de puertas (incluidos los cambios preexistentes y los sellados en modalidad continua) se comporta exactamente como hasta ahora. | Confirmado (paradas-tres-salidas) | RF-BRA-26 |

### Dependencias de diseño

| ID | Dependencia | Dónde afecta |
|---|---|---|
| Q1 | Versión mínima de git para `git merge-tree --write-tree` (2.38). | RF-BRA-17 (design) |
| Q2 | Coexistencia del registro local con `<git common dir>/gentle-ai/`. | RF-BRA-15 (design) |
| Q6 | Destino de los scripts locales y de `guard-primary-checkout.ps1`. | RF-BRA-25 (design) |
