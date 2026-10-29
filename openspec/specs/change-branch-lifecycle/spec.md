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
| Rama de integración | Rama a la que se integra el trabajo. Valor de `git.integration_branch` (por defecto `main`), con posible valor propio por repositorio (REQ-25.3). |
| Prefijo | Valor de `git.branch_prefix` (por defecto `feat`), con posible valor propio por repositorio (REQ-25.3). |
| Rama del cambio | `<prefijo>/<cambio>`. |
| Rama de rol | `<prefijo>/<cambio>-<rol>`. Solo existe cuando dos o más roles del cambio tocan el mismo repositorio (REQ-25.4). |
| Rama de seguimiento | La rama del cambio cuando existen ramas de rol sobre ese repositorio: los PR de rol se dirigen contra ella. |
| Rama de archivado | `<prefijo>/<cambio>-archive`. Solo existe en los repositorios con rama de cambio (`monorepo-embedded`), la crea `axiom branch ensure --change <c> --archive` desde la rama de integración una vez fusionado el PR del cambio, y en ella se ejecuta `archive` (REQ-25.45). No existe en `multirepo`. |
| Cambio pendiente de archivar | Cambio activo presente en el árbol de la rama de integración de un repositorio con rama de cambio, es decir, ya fusionado y aún sin archivar. Se excluyen por nombre los sumideros `e2e-cumulative` y `cumulative-<rol>` (REQ-25.20). |
| Raíz git | Directorio de nivel superior del repositorio git que contiene una ruta (`git rev-parse --show-toplevel`), con su git common dir (`git rev-parse --git-common-dir`). |
| Repositorio con rama de cambio | En `monorepo-embedded`, cada repositorio git tocado por el cambio (incluido el del planning home). En `multirepo`, cada repositorio de código tocado por algún rol del cambio. |
| Repositorio de specs en multirepo | En `multirepo`, el repositorio que contiene el planning home (`planning-home-resolution`). Se trabaja siempre in situ, en su rama de integración. |
| Repositorio de código en multirepo | Cualquier otro repositorio declarado por los roles en `multirepo`. |
| Roles del cambio sobre un repositorio | Intersección entre los roles sellados en el `kickoff.yaml` del cambio y los roles de `axiom.yaml` cuyo `repositories` resuelve a esa raíz git. El rol reservado `fullstack` cuenta como un único rol que toca todos los repositorios que declara algún rol de `axiom.yaml` y, en `monorepo-embedded`, el repositorio del planning home (supuesto S-3). |
| Modo de trabajo | Preferencia personal por proyecto: `worktree` (por defecto), `in_place` o `ask`. Vive en el hub, no en `axiom.yaml`. |
| Raíz de worktrees | Directorio personal donde se crean los worktrees. Por defecto `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`, fuera de cualquier repositorio. |
| Registro compartido | Instantánea de política escrita una sola vez por `kickoff seal` en `kickoff.yaml` (REQ-25.36). |
| Registro local | Estado local no versionado, en el git common dir del repositorio del planning home (REQ-25.14). |
| Instantánea | Entradas de política del registro compartido: integración, prefijo, base por repositorio y nombres de rama derivados. |
| Veredicto de rama | Respuesta única a «¿esta rama es la que corresponde a este cambio en este repositorio según su papel?» (REQ-25.18). |
| Carril | `sdd` (por defecto) u `odd`. |

### Tokens y etiquetas del JSON

Todo valor de estado o de clasificación del JSON es un **token estable en inglés** acompañado, en un campo aparte, de una **etiqueta legible en castellano**. Los tokens no se traducen ni cambian con el idioma; los mensajes y las etiquetas están en castellano (REQ-25.2).

| Token | Etiqueta | Dónde |
|---|---|---|
| `not_started` | no iniciado | estado de un cambio sin ramas (REQ-25.15) |
| `correct` | correcto | veredicto de rama (REQ-25.18) |
| `not_applicable` | no aplicable | veredicto de rama (REQ-25.18) |
| `ancestry` | ascendencia | capa 1 de la detección de fusión (REQ-25.16) |
| `equivalent_tree` | árbol equivalente | capa 2 de la detección de fusión (REQ-25.16) |
| `pr_merged` | PR fusionado | capa 3 de la detección de fusión (REQ-25.16) |
| `forced` | forzado | confirmación con `--force` (REQ-25.17) |

### Códigos tipados

Los resultados que no son «listo» se identifican por un código estable, nunca por el texto del mensaje.

| Código | Significado |
|---|---|
| `decision_required` | El modo es `ask` y no hay respuesta registrada: no se ha creado nada (REQ-25.8). |
| `dirty_worktree` | Cambios sin confirmar impiden la operación (REQ-25.9, REQ-25.17). |
| `wrong_branch` | La rama actual no es la que corresponde (REQ-25.18). |
| `specs_repo_not_on_integration` | En multirepo, el repositorio de specs no está en su rama de integración (REQ-25.5). |
| `base_not_found` | No existe la rama de integración, ni en local ni como referencia remota local (REQ-25.10). |
| `fetch_failed` | `--fetch` fue solicitado y falló (REQ-25.10). |
| `path_too_long` | La ruta de un worktree supera el límite admitido (REQ-25.11). |
| `invalid_name` | El nombre del cambio, el prefijo o la rama indicados en la invocación no son válidos (REQ-25.3, REQ-25.11). |
| `config_invalid` | `axiom.yaml` o el hub personal no se pueden leer, o su política es inválida, y impiden planificar (REQ-25.3). |
| `git_failed` | Una invocación de git falló por una causa que ningún otro código cubre; el JSON nombra la operación y resume el error (REQ-25.2). |
| `plan_conflict` | Una invocación contradice el plan ya materializado del cambio (REQ-25.8, REQ-25.12, REQ-25.45). |
| `branch_checked_out_elsewhere` | La rama ya está desplegada en un worktree que el registro no conoce (REQ-25.7). |
| `not_a_git_repository` | Una ruta declarada no pertenece a ningún repositorio git; se omite con aviso (REQ-25.6). |
| `merge_not_confirmed` | Ninguna capa confirma la fusión de la rama (REQ-25.16). |
| `change_not_integrated` | `ensure --archive`: la base local no contiene el cambio ya fusionado; sugiere `--fetch` (REQ-25.45). |
| `archive_branch_not_applicable` | `ensure --archive` en una topología o carril sin rama de archivado (REQ-25.45). |

## Requerimientos de un vistazo

| ID | Requerimiento | Garantía principal |
|---|---|---|
| REQ-25.1 | Familia `axiom branch` | Cuatro verbos de primer nivel, neutrales respecto al carril; ni `push` ni PR; sin operaciones remotas verificables |
| REQ-25.2 | Contrato de salida común | JSON con rutas absolutas, `--cwd`, tokens en inglés con etiqueta en castellano y código de salida «listo = 0» |
| REQ-25.3 | Resolución de la política | Excepción por cambio > repositorio > `git:` > defecto |
| REQ-25.4 | Modelo de ramas por número de roles | 1 rol: `<prefijo>/<cambio>`; 2 o más: seguimiento más ramas de rol; rama de archivado aparte |
| REQ-25.5 | Papel del repositorio según topología | Repositorio de specs in situ en multirepo; código con rama de cambio |
| REQ-25.6 | Raíz git de cada repositorio | Subcarpetas, `.`, repositorios compartidos y rutas no git |
| REQ-25.7 | `ensure`: plan e idempotencia | Mismo plan, mismo resultado, sin duplicar |
| REQ-25.8 | `ensure`: modo de trabajo | `worktree`, `in_place` y `ask` tipado; el checkout principal no se toca |
| REQ-25.9 | `ensure` en `in_place` con árbol sucio | Rechazo sin modificar el repositorio |
| REQ-25.10 | Base sin red | Sin `fetch` implícito; `base.source`, `sha` y `fetched` |
| REQ-25.11 | Rutas de worktree y longitud | `<raíz>/<cambio>[-<rol>\|-archive]/<repo>`; rechazo por longitud (220 en Windows) |
| REQ-25.12 | Excepciones por cambio | `--prefix`, `--branch`, `--base`; registradas y sin contradicción |
| REQ-25.45 | Rama de archivado y `ensure --archive` | `<prefijo>/<cambio>-archive` desde la integración; `change_not_integrated` sin fetch implícito |
| REQ-25.13 | Carril ODD | Misma política para `odd/tasks/` |
| REQ-25.14 | Registro local | Rutas absolutas y respuesta a `ask` en el git common dir |
| REQ-25.15 | `status` | Estado de ramas y worktrees del cambio, solo con referencias locales |
| REQ-25.16 | `done`: detección de fusión por capas | `ancestry`, `equivalent_tree`, `pr_merged` |
| REQ-25.17 | `done`: seguridad y eliminación de worktree y rama local | Elimina worktree y rama local tras fusión confirmada; rechazo con árbol sucio o fusión no confirmada salvo `--force`; nunca toca ramas remotas |
| REQ-25.18 | Veredicto único de rama | Una sola consulta, por papel del repositorio y contexto (ordinario o de archivado) |
| REQ-25.19 | Guarda A: escritores Go | Todo escritor de la carpeta del cambio se niega fuera de rama |
| REQ-25.20 | Cambios pendientes de archivar | Visibles en `status` y en el dashboard; sin código de fallo ni integración con la CI |
| REQ-25.21 | `guard`: hook opcional | Bloquea documentos de Axiom y avisa sin bloquear en el resto; lee HEAD sin lanzar git; fail-open |
| REQ-25.22 | Instalación del hook por `axiom sync` | Claude Code, Codex y OpenCode; sin hooks de git |
| REQ-25.23 | Skills y doctrina delegan en el verbo | Retirada de ODD-5.x y del Paso 2c |
| REQ-25.24 | Dogfooding en este repositorio | `git:` declarado (`integration_branch: main`, `branch_prefix: inc`) y documentación remitida a `axiom branch` |

## No cubre esta capacidad

- `push` ni creación de PR desde `axiom branch`: siguen siendo decisiones de la persona.
- Hooks de git (`pre-commit`, `pre-push`, etc.) en cualquier repositorio, incluidos los de terceros.
- Operaciones remotas por defecto: ninguna orden hace `fetch`, `pull`, `push` ni consulta remota sin una opción explícita (REQ-25.10).
- Un comprobador para la CI ni un código de fallo por cambios activos en la rama de integración: con el archive en un PR documental aparte, es normal que la rama de integración contenga cambios activos de forma temporal (REQ-25.20).
- Un escritor Go del documento ODD `odd/tasks/<feature>.md`: ODD sigue siendo doctrina de agente y queda cubierto solo por la doctrina y por la guarda D (REQ-25.13).
- Edición de `axiom.yaml` desde el dashboard o la TUI (`local-web-dashboard`).
- Cobertura del hook en Pi: no hay canal demostrado y la capa D es de mejor esfuerzo (REQ-25.22).
- La eliminación de ramas remotas: `done` elimina el worktree y la rama local de un cambio ya fusionado, pero nunca toca ramas remotas (REQ-25.17).
- La migración de cambios preexistentes: se eximen, no se migran (REQ-25.37 de `sdd-lane-kickoff`).

---

## 1. Capacidad: `change-branch-lifecycle`

El motor de ramas planifica de forma pura (topología, política, roles y preferencias personales producen un plan determinista) y ejecuta el plan con git de forma idempotente. Las guardas consultan el mismo veredicto.

### Requirement: Familia de Comandos `axiom branch` (REQ-25.1)

La CLI DEBE exponer la familia de primer nivel `axiom branch` con exactamente los subcomandos `ensure`, `status`, `done` y `guard`. La familia DEBE ser neutral respecto al carril: `ensure`, `status` y `done` DEBEN aceptar `--change <nombre>` y `--lane sdd|odd` (por defecto `sdd`) y NO DEBEN depender de que exista un cambio SDD. Ningún subcomando DEBE ejecutar `git push`, crear un PR ni realizar operaciones remotas salvo la solicitada por `--fetch` (REQ-25.10). La familia NO DEBE exponer subcomandos `pr`, `push` ni ningún comprobador para la CI. Toda ejecución de git del motor DEBE pasar por un único punto, de modo que la ausencia de operaciones remotas sea verificable con un observador instalado en ese punto y con una prueba de arquitectura que cubra el código de `internal/branch/**` y de `internal/gitcommon`.
(Previously: cinco verbos, incluido un comprobador de CI, y la ausencia de operaciones remotas se verificaba con un envoltorio externo de `git`)

#### Scenario: La ayuda lista los cuatro verbos
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch --help`
- **ENTONCES** la salida lista `ensure`, `status`, `done` y `guard`
- **Y** no lista `pr`, `push` ni ningún comprobador para la CI

#### Scenario: Un subcomando no listado no existe
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom branch` seguido de `push`, de `pr` o del verbo retirado `check`
- **ENTONCES** el comando se rechaza como no reconocido con código de salida distinto de cero

#### Scenario: Ningún subcomando publica ni abre PR
- **DADO** un repositorio con un remoto configurado y un observador instalado en el único punto del motor que ejecuta git
- **CUANDO** se ejecutan `ensure`, `status` y `done` sin `--fetch`
- **ENTONCES** el observador no registra ninguna invocación de `git push`, `git fetch`, `git pull` ni `git ls-remote`

#### Scenario: Una prueba de arquitectura fija el único punto de ejecución de git
- **DADO** el código fuente de los paquetes bajo `internal/branch/**` y de `internal/gitcommon`
- **CUANDO** se ejecuta la prueba de arquitectura
- **ENTONCES** ningún paquete lanza procesos ni abre conexiones de red por otra vía que no sea ese único punto de ejecución de git
- **Y** la prueba falla si se añade una ejecución de git fuera de ese punto

#### Scenario: El carril ODD no exige un cambio SDD
- **DADO** un proyecto sin ningún cambio bajo `openspec/changes/`
- **CUANDO** se ejecuta `axiom branch ensure --lane odd --change mi-feature`
- **ENTONCES** la orden se ejecuta sin requerir `kickoff.yaml` ni `proposal.md`

---

### Requirement: Contrato de Salida Común con Rutas Absolutas (REQ-25.2)

`ensure`, `status` y `done` DEBEN admitir `--json` y `--cwd <ruta>`. La salida JSON DEBE incluir un estado tipado y, cuando no sea «listo», uno de los códigos tipados de esta capacidad. Todo valor de estado o de clasificación del JSON DEBE ser un token estable en inglés (`not_started`, `correct`, `not_applicable`, `ancestry`, `equivalent_tree`, `pr_merged`, `forced`, etc.) acompañado, en un campo aparte, de una etiqueta legible en castellano; los tokens NO DEBEN traducirse. Toda ruta de la salida (raíz git de cada repositorio, planning home, worktrees, raíz de worktrees) DEBE ser absoluta. El código de salida DEBE ser `0` únicamente cuando la operación ha dejado el estado pedido; todo rechazo, todo `decision_required` y todo error DEBEN terminar con código distinto de cero. Los mensajes legibles DEBEN estar en español y nombrar la continuación. Una configuración ilegible o inválida DEBE terminar con `config_invalid` y una invocación de git que falle por una causa no cubierta por otro código con `git_failed`, que DEBE nombrar la operación y resumir el error. Ninguna fase posterior a `ensure` DEBE depender del directorio de trabajo heredado: las órdenes posteriores reciben `--cwd` con una ruta tomada de este JSON.
(Previously: el contrato incluía el comprobador de CI y los valores de estado no tenían un token estable con etiqueta separada)

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

---

### Requirement: Resolución de la Política de Ramas (REQ-25.3)

La política de cada repositorio DEBE resolverse con esta precedencia, de mayor a menor: (1) las excepciones por cambio de `ensure` (`--prefix`, `--branch`, `--base`, REQ-25.12); (2) la entrada del repositorio en `repositories:` de `axiom.yaml`, si existe; (3) el bloque `git:` de `axiom.yaml`; (4) los valores por defecto: rama de integración `main` y prefijo `feat`. Sin bloque `git:`, `integration_branch` DEBE valer `main` y `branch_prefix` DEBE valer `feat`; sin `repositories:`, ningún repositorio tiene ajustes propios y todos heredan los valores del workspace. Los nombres de rama resultantes DEBEN ser nombres de referencia git válidos: un prefijo o nombre inválido indicado en la invocación (`--prefix`, `--branch`, `--change`) DEBE rechazarse con `invalid_name`, y un valor inválido o una configuración ilegible en `axiom.yaml` o en el hub personal con `config_invalid`, en ambos casos sin crear nada.
(Previously: todo valor inválido se rechazaba con `invalid_name`, sin distinguir entre invocación y configuración)

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

---

### Requirement: Modelo de Ramas por Número de Roles que Tocan Cada Repositorio (REQ-25.4)

El modelo de ramas de cada repositorio con rama de cambio DEBE depender de cuántos roles del cambio lo tocan (roles sellados en el `kickoff.yaml` del cambio intersecados con los roles de `axiom.yaml` cuyos repositorios resuelven a esa raíz git):

- **Un rol** (lo habitual, también `fullstack`): solo existe la rama `<prefijo>/<cambio>`; el rol trabaja en ella y su PR se dirige a la rama de integración.
- **Dos o más roles**: `<prefijo>/<cambio>` es la rama de seguimiento y cada rol saca `<prefijo>/<cambio>-<rol>` desde ella. Hay un PR de cada rol contra la rama de seguimiento y un PR final a la rama de integración que lleva el cambio (código y artefactos), sin el archive. Se RECOMIENDA el merge commit, sin imponerlo.
- **Rama de archivado** (solo en repositorios con rama de cambio): una vez fusionado en la rama de integración el PR del cambio, el archive se hace en `<prefijo>/<cambio>-archive`, que sale de la rama de integración (REQ-25.45) y se entrega en un PR documental aparte contra la rama de integración. NO existe en `multirepo`.

El modelo DEBE evaluarse por repositorio: en un mismo cambio un repositorio puede tener rama de rol y otro no. Un rol cuyo nombre sea `archive` colisionaría con la rama de archivado y DEBE rechazarse con `invalid_name` al planificar las ramas de rol (supuesto S-15). `ensure` NO DEBE publicar la rama de seguimiento: DEBE avisar de que la persona la sube al remoto antes del primer PR de rol. La rama destino propuesta para cada PR (rol contra seguimiento, cambio contra integración, archivado contra integración) DEBE poder obtenerse del plan (`status`, REQ-25.15) para que `sdd-apply` la proponga en lugar de preguntarla.
(Previously: el PR final a la rama de integración llevaba el archive y no existía rama de archivado)

#### Scenario: Un rol, rama única
- **DADO** un monorepo con el rol único `fullstack` sellado en el kickoff del cambio `x`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** existe solo la rama `feat/x`
- **Y** no existe ninguna rama `feat/x-<rol>`

#### Scenario: Dos roles sobre el mismo repositorio crean seguimiento y ramas de rol
- **DADO** un monorepo con los roles `core` y `qa` sellados en el kickoff del cambio `x`, ambos con repositorios que resuelven a la misma raíz git
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** existe la rama de seguimiento `feat/x` desde la base
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
- **DADO** un monorepo con los roles `core` y `archive` sellados sobre el mismo repositorio
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `invalid_name` nombrando el rol `archive` y la colisión con `feat/x-archive`
- **Y** no se crea ninguna rama de rol

#### Scenario: La rama de archivado no pertenece al modelo de roles
- **DADO** un monorepo con los roles `core` y `qa` y las ramas `feat/x`, `feat/x-core` y `feat/x-qa`
- **CUANDO** se ejecuta `axiom branch status --change x --json`
- **ENTONCES** el destino propuesto del PR de `feat/x` es la rama de integración
- **Y** el de la rama de archivado `feat/x-archive`, cuando existe, es la rama de integración

---

### Requirement: Papel del Repositorio según la Topología (REQ-25.5)

El tratamiento de cada repositorio DEBE depender de su papel:

- **`monorepo-embedded`**: el repositorio del planning home es un repositorio con rama de cambio y recibe el modelo de REQ-25.4, con worktree según el modo de trabajo.
- **`multirepo`, repositorio de specs**: se trabaja SIEMPRE in situ, en su rama de integración. `ensure` NO DEBE crear en él ramas del cambio, ramas de rol, ramas de archivado ni worktrees. `ensure` NO DEBE hacer checkout en él: si no está en su rama de integración, o está en una rama de rol del cambio, DEBE rechazar con `specs_repo_not_on_integration`, nombrando la rama actual y la continuación.
- **`multirepo`, repositorios de código**: cada repositorio tocado por algún rol del cambio recibe el modelo de REQ-25.4 desde su propia rama de integración.

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

### Requirement: Raíz Git Real de Cada Repositorio (REQ-25.6)

El motor DEBE obtener la raíz git de cada repositorio declarado en `roles.<rol>.repositories[].path` con las consultas de git (`rev-parse --show-toplevel` y `--git-common-dir`), nunca suponiendo un directorio `.git` literal. Una ruta que sea una subcarpeta de un repositorio, el propio `.`, o un repositorio compartido por varios roles DEBE resolverse a una única raíz git deduplicada, con un único conjunto de ramas y worktrees por raíz. Una ruta que no pertenezca a ningún repositorio git DEBE omitirse sin abortar el resto del plan: el JSON DEBE listarla como omitida con el código `not_a_git_repository` y un mensaje que nombre la ruta y el rol. Una ruta declarada que no exista DEBE reportarse con el mismo tratamiento de omisión y mensaje.

#### Scenario: Una subcarpeta resuelve a la raíz git del repositorio que la contiene
- **DADO** un rol con `repositories: [{path: services/api}]` dentro de un repositorio cuya raíz es la carpeta maestra
- **CUANDO** se planifica el cambio
- **ENTONCES** el repositorio del plan es la raíz git, no `services/api`

#### Scenario: Dos roles sobre rutas del mismo repositorio se deduplican
- **DADO** dos roles con rutas `app/web` y `app/api`, ambas dentro del mismo repositorio
- **CUANDO** se planifica el cambio
- **ENTONCES** el plan contiene un único repositorio
- **Y** los dos roles se cuentan como roles del cambio sobre ese repositorio (REQ-25.4)

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

### Requirement: `ensure` Planifica de Forma Determinista y es Idempotente (REQ-25.7)

`axiom branch ensure` DEBE producir un plan determinista a partir de la topología, `git:`, `repositories:`, los roles del kickoff y las preferencias personales, y ejecutarlo con git. Una segunda ejecución con los mismos argumentos y el mismo estado DEBE no cambiar nada (ni ramas, ni worktrees, ni registro) y devolver el mismo JSON. El comando DEBE crear solo lo que falta: si la rama ya existe, DEBE reutilizarla sin moverla ni reiniciarla; si ya existe su worktree registrado, DEBE devolver su ruta. Si la rama ya está desplegada en un worktree que el registro local no conoce, DEBE rechazar con `branch_checked_out_elsewhere` nombrando la ruta. Cuando todavía no hay kickoff sellado (el cambio se nombra antes del sellado), `ensure` DEBE planificar solo para el repositorio del planning home la rama `<prefijo>/<cambio>`, que es idéntica en ambos modelos de REQ-25.4, y una ejecución posterior, ya con el kickoff sellado, DEBE completar de forma incremental las ramas de rol y los repositorios de código que correspondan sin alterar lo ya creado (supuesto S-1). `ensure --change <c> --archive` (REQ-25.45) es un plan aparte con las mismas garantías de determinismo e idempotencia: no altera las ramas del cambio ya creadas.
(Previously: no contemplaba el plan de la rama de archivado)

#### Scenario: Segunda ejecución sin cambios
- **DADO** un cambio ya preparado por `ensure`
- **CUANDO** se ejecuta de nuevo `ensure --change x --json` con los mismos argumentos
- **ENTONCES** el JSON es idéntico al de la primera ejecución
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
- **DADO** un cambio nombrado sin `kickoff.yaml`, en un monorepo
- **CUANDO** se ejecuta `ensure --change x` antes del sellado y de nuevo después de sellar dos roles `core` y `qa`
- **ENTONCES** la primera ejecución crea solo `feat/x`
- **Y** la segunda conserva `feat/x` y crea `feat/x-core` y `feat/x-qa` desde ella

#### Scenario: Se rechaza un nombre de cambio que no es válido
- **DADO** un nombre de cambio con caracteres inválidos como componente de rama git
- **CUANDO** se ejecuta `ensure --change "a b:c"`
- **ENTONCES** se rechaza con `invalid_name`
- **Y** no se crea nada

#### Scenario: `--archive` no altera las ramas del cambio
- **DADO** un cambio con las ramas `feat/x` y `feat/x-core` ya creadas y su PR fusionado
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se crea `feat/x-archive` y el commit al que apunta cada una de las ramas anteriores no cambia

---

### Requirement: `ensure`: Modo de Trabajo `worktree`, `in_place` y `ask` (REQ-25.8)

`ensure` DEBE leer el modo de trabajo del hub personal (`~/.axiom/workspaces.json`) y NUNCA de `axiom.yaml`. Sin preferencias guardadas para el proyecto, estén o no registrado, el modo DEBE ser `worktree`.

- `worktree`: DEBE crear la rama desde la base y un worktree en la raíz de worktrees (REQ-25.11). NUNCA DEBE modificar el HEAD, el índice ni el árbol de trabajo del checkout principal.
- `in_place`: DEBE crear (o seleccionar) la rama y hacer checkout de ella en el propio repositorio, sujeto a REQ-25.9.
- `ask`: DEBE devolver el resultado tipado `decision_required` sin crear nada, sin leer la entrada estándar y sin interactuar; quien invoca (skill o dashboard) presenta la pregunta y repite la orden con la respuesta (`--mode worktree|in_place`, supuesto S-2). La respuesta DEBE registrarse por cambio en el registro local (REQ-25.14) y las ejecuciones posteriores para ese mismo cambio, incluida la de `--archive`, DEBEN reutilizarla sin volver a preguntar. Una respuesta registrada NO modifica la preferencia personal del hub.

Un `--mode` distinto del ya materializado para el cambio DEBE rechazarse con `plan_conflict`.

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

---

### Requirement: `ensure` en `in_place` Rechaza un Árbol Sucio (REQ-25.9)

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

### Requirement: Base sin Red y `--fetch` Explícito (REQ-25.10)

`ensure` NO DEBE realizar ninguna operación remota por defecto. La base de cada rama DEBE ser `origin/<integración>` si esa referencia existe en local y, si no, la rama de integración local. El JSON DEBE devolver, para cada repositorio, `base.ref`, `base.sha`, `base.source` (que indica si procede de la referencia remota local o de la rama local) y `fetched: false`. Solo con `--fetch` el comando PUEDE ejecutar un `fetch` previo, y entonces `fetched` DEBE ser `true`. Si `--fetch` falla, `ensure` DEBE terminar con `fetch_failed` sin crear nada (supuesto S-8). Si no existe la rama de integración ni local ni como referencia remota local, DEBE rechazar con `base_not_found`, nombrando la rama esperada. `--base <ref>` sustituye la base (REQ-25.12).

#### Scenario: Sin `--fetch` no hay operaciones remotas
- **DADO** un repositorio con remoto y un observador instalado en el único punto del motor que ejecuta git
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

#### Scenario: Rama de integración inexistente
- **DADO** un `axiom.yaml` con `git.integration_branch: develop` y un repositorio sin `develop` ni `origin/develop`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** se rechaza con `base_not_found` nombrando `develop`

---

### Requirement: Rutas de Worktree y Límite de Longitud (REQ-25.11)

La ruta del worktree de cada rama DEBE ser `<raíz de worktrees>/<cambio>[-<rol>|-archive]/<repo>`, donde el sufijo `-<rol>` solo aparece si existe rama de rol para ese repositorio (REQ-25.4), el sufijo `-archive` solo aparece en el worktree de la rama de archivado (REQ-25.45) y `<repo>` es el último segmento de la ruta (`path`) del repositorio, tal como la declara `repositories:` o el rol (supuesto S-4); para el repositorio cuya ruta es `.`, `<repo>` es el nombre del directorio de su raíz git, y si dos repositorios del plan tuvieran el mismo último segmento, `ensure` DEBE rechazar con `invalid_name` nombrando ambos, antes de crear nada (supuesto S-17). La raíz de worktrees DEBE ser la preferencia personal del hub o, sin ella, `<padre de la carpeta maestra>/<nombre de la carpeta maestra>-wt`, fuera de cualquier repositorio. El planificador DEBE validar, antes de crear nada, que el nombre del cambio es válido a la vez como componente de rama git y como nombre de directorio en Windows, y que la longitud de cada ruta de worktree no supera el límite admitido: el de la plataforma menos un margen de 40 caracteres, es decir, 220 caracteres en Windows sin rutas largas habilitadas (260 menos 40). Si no se cumple, DEBE rechazar con `invalid_name` o `path_too_long`, nombrando la ruta, su longitud y el máximo, sin crear ramas ni worktrees.
(Previously: `<repo>` era el nombre declarado en `repositories:` o el del directorio de la raíz git, la ruta no contemplaba la rama de archivado y el límite solo exigía «margen» sin cifra)

#### Scenario: Ruta sin sufijo de rol con una sola rama
- **DADO** un monorepo con un rol y modo `worktree`
- **CUANDO** se ejecuta `ensure --change x`
- **ENTONCES** el worktree está en `<raíz de worktrees>/x/<repo>`
- **Y** la ruta no contiene sufijo de rol

#### Scenario: Ruta con sufijo de rol cuando hay rama de rol
- **DADO** un monorepo con los roles `core` y `qa` sobre el mismo repositorio
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

### Requirement: Excepciones por Cambio `--prefix`, `--branch` y `--base` (REQ-25.12)

`ensure` DEBE admitir `--prefix`, `--branch` y `--base` como excepciones por cambio a la política del proyecto. `--prefix` sustituye el prefijo; `--branch` sustituye el nombre completo de la rama del cambio (la rama de seguimiento si hay ramas de rol, y las ramas de rol derivan como `<--branch>-<rol>`) y se aplica a todos los repositorios del plan, que `--repo` PUEDE acotar; `--base` sustituye la base de todos los repositorios del plan. Las excepciones DEBEN registrarse en el registro local y, cuando el cambio se selle, reflejarse en la instantánea del `kickoff.yaml` (REQ-25.36) para que ejecuciones posteriores y las guardas usen los mismos nombres. Una invocación posterior cuyas excepciones contradigan las ya materializadas DEBE rechazarse con `plan_conflict`. Con `--archive` (REQ-25.45) solo se admite `--prefix`, que fija el prefijo de la rama de archivado sin contradecir el del cambio; `--branch` y `--base` se rechazan con `plan_conflict` (supuesto S-16). Sin `--prefix`, la rama de archivado deriva de la rama del cambio registrada como `<rama del cambio>-archive`.
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

### Requirement: Rama de Archivado y `ensure --archive` (REQ-25.45)

En los repositorios con rama de cambio (`monorepo-embedded`), el archivado de un cambio DEBE hacerse en una rama propia, la rama de archivado `<prefijo>/<cambio>-archive`, y entregarse en un PR documental aparte, **después** de fusionar el PR del cambio (REQ-25.39). `axiom branch ensure --change <c> --archive` DEBE crearla en el repositorio del planning home, saliendo de la rama de integración con la base de REQ-25.10 (`origin/<integración>` si existe en local y, si no, la rama de integración local), y aplicando el modo de trabajo personal de REQ-25.8 (en modo `worktree`, un worktree en `<raíz de worktrees>/<cambio>-archive/<repo>`; en `in_place`, el checkout de la rama en el propio repositorio, sujeto a REQ-25.9). `--archive` admite `--prefix` (REQ-25.12).

Si la base resuelta no contiene ya el cambio fusionado, es decir, si su árbol no incluye la carpeta `changes/<cambio>/` del planning home, `ensure --archive` DEBE rechazar con `change_not_integrated`, sin crear nada, nombrando la base evaluada y sugiriendo `--fetch`. NO hay `fetch` implícito: solo con `--fetch` explícito se actualiza la referencia remota local antes de evaluar (REQ-25.10). `ensure --archive` es idempotente: reutiliza la rama y el worktree existentes sin moverlos, y NO publica la rama. En `multirepo` NO existe rama de archivado: `--archive` DEBE rechazarse con `archive_branch_not_applicable` porque el archive se hace in situ en la rama de integración del repositorio de specs (REQ-25.5); tampoco existe en el carril ODD (`--lane odd`), que no tiene archive. `done` también cubre la rama de archivado (REQ-25.16 y REQ-25.17) y `status` la incluye en el plan (REQ-25.15).

#### Scenario: Crea la rama de archivado desde la integración con el cambio fusionado
- **DADO** un monorepo en modo `worktree`, el cambio `x` fusionado en `origin/main` (referencia local) y la política por defecto
- **CUANDO** se ejecuta `ensure --change x --archive --json`
- **ENTONCES** existe la rama `feat/x-archive` creada desde `origin/main`
- **Y** existe su worktree en `<raíz de worktrees>/x-archive/<repo>`
- **Y** `base.ref` es `origin/main` y `fetched` es `false`

#### Scenario: La base local sin el cambio fusionado se rechaza y sugiere `--fetch`
- **DADO** un monorepo cuya rama de integración local y cuya referencia remota local no contienen `openspec/changes/x/`
- **CUANDO** se ejecuta `ensure --change x --archive`
- **ENTONCES** se rechaza con `change_not_integrated` nombrando la base evaluada
- **Y** el mensaje sugiere `--fetch`
- **Y** no se crea ninguna rama ni worktree
- **Y** no consta ninguna operación remota

#### Scenario: `--fetch` explícito permite alcanzar el cambio fusionado
- **DADO** el mismo monorepo, con el cambio fusionado ya en el remoto pero no en las referencias locales
- **CUANDO** se ejecuta `ensure --change x --archive --fetch`
- **ENTONCES** se ejecuta un `fetch` antes de evaluar la base y el JSON devuelve `fetched: true`
- **Y** se crea la rama `feat/x-archive`

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

---

### Requirement: Carril ODD con la Misma Política (REQ-25.13)

`ensure --lane odd --change <feature>` DEBE aplicar la misma política que SDD, con estas particularidades: el documento ODD es `<workspaceRoot>/odd/tasks/<feature>.md`, en el repositorio del planning home. En `monorepo-embedded`, la rama es `<prefijo>/<feature>` con worktree según el modo; en `multirepo`, el repositorio de specs se trabaja in situ en su rama de integración (REQ-25.5) y los repositorios de código se incluyen solo bajo demanda con `--repo <ruta>`, porque ODD no divide en roles y cada repositorio incluido recibe solo la rama `<prefijo>/<feature>`. En el carril ODD no existe `kickoff.yaml` ni instantánea ni rama de archivado: las únicas protecciones que se aplican son la propia política de `ensure` y la guarda D (REQ-25.21), que bloquea las escrituras del agente en `odd/tasks/` fuera de la rama que corresponde, porque no hay escritor Go del documento ODD.

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

---

### Requirement: Registro Local en el Git Common Dir (REQ-25.14)

`ensure` DEBE mantener un registro local por cambio, fuera de cualquier árbol versionado, en el git common dir del repositorio del planning home, obtenido con `git rev-parse --git-common-dir` (nunca asumiendo `.git`). El registro DEBE contener las rutas absolutas de todos los repositorios, ramas (incluida la de archivado) y worktrees del cambio y la respuesta a `ask` (REQ-25.8). Debe ser accesible con el mismo contenido desde el checkout principal y desde cualquier worktree enlazado del mismo repositorio. Si el registro se borra, el repositorio NO DEBE verse afectado y `ensure` DEBE reconstruirlo de forma idempotente a partir del estado de git (ramas y worktrees existentes), mientras `status` DEBE informar que el cambio no tiene registro. La coexistencia con el estado local actual de RDD (`<git common dir>/gentle-ai/`) se resuelve en `sdd-design` (Q2) y NO DEBE alterar los linajes RDD existentes.

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

---

### Requirement: `status`: Estado de Ramas y Worktrees del Cambio (REQ-25.15)

`axiom branch status` DEBE informar, para cada repositorio del cambio, su papel (repositorio con rama de cambio, repositorio de specs in situ o repositorio de código), la rama esperada y la rama actual, la ruta absoluta del worktree si existe, si el árbol tiene cambios sin confirmar, el modo de trabajo materializado, la base, y el número de commits que la rama lleva de atraso o de adelanto respecto de su base calculado solo con referencias locales, y la rama destino propuesta para el PR (REQ-25.4). Cuando existe la rama de archivado, DEBE incluirla con los mismos datos. `--change` es opcional: sin él, `status` lista los cambios registrados y los cambios pendientes de archivar (REQ-25.20); con él, además informa de si el cambio solicitado está pendiente de archivar. NUNCA DEBE realizar operaciones remotas. DEBE poder ejecutarse desde el checkout principal, desde cualquier worktree del cambio y con `--cwd`. Un cambio sin ramas creadas DEBE informarse como no iniciado (`not_started`), no como error.
(Previously: `--change` era obligatorio, no incluía la rama de archivado ni los cambios pendientes de archivar y el estado de un cambio sin ramas se expresaba como texto en castellano)

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

---

### Requirement: `done`: Detección de Fusión por Capas (REQ-25.16)

`axiom branch done` DEBE determinar, para cada rama del cambio, si ya está fusionada en su destino (la rama de rol en la de seguimiento; la rama del cambio y la rama de archivado en la de integración) con estas capas, de la primera a la última, deteniéndose en la primera que confirme:

1. **Ascendencia** (`ancestry`): la rama es ancestro del destino.
2. **Árbol equivalente** (`equivalent_tree`): la fusión simulada de la rama en el destino (`git merge-tree --write-tree`) produce el árbol del destino. Detecta los merges por squash o rebase aunque el destino haya avanzado después. La capa DEBE omitirse sin error si la versión de git no admite la orden (Q1, a verificar en `sdd-design`).
3. **Evidencia `pr_merged`** (`pr_merged`): el `gates.yaml` del cambio contiene una evidencia `pr_merged` ya verificada. Esta capa NO aplica a la rama de archivado: la evidencia del cambio acredita el PR del cambio y no el PR documental de archivado, de modo que la rama de archivado solo se confirma por las capas 1 y 2.

Si ninguna capa confirma la fusión, la rama NO se considera fusionada y `done` DEBE exigir `--force` (REQ-25.17). El JSON DEBE indicar, por rama, qué capa la confirmó con los tokens `ancestry`, `equivalent_tree`, `pr_merged` o `forced`, cada uno con su etiqueta en castellano. La detección DEBE usar solo referencias locales; solo con `--fetch` PUEDE actualizarse el remoto antes de evaluar.
(Previously: la detección no cubría la rama de archivado y los valores que identificaban la capa eran etiquetas en castellano)

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

---

### Requirement: `done`: Seguridad y Eliminación de Worktree y Rama Local (REQ-25.17)

Para cada rama cuya fusión esté confirmada por la detección por capas de REQ-25.16, `done` DEBE eliminar su worktree (`git worktree remove`), después eliminar la rama local (también cuando git no la considere fusionada, como ocurre tras un squash, porque la fusión ya está confirmada por las capas) y actualizar el registro local. Si la eliminación del worktree falla, NO DEBE eliminarse la rama. `done` DEBE rechazar la operación sobre una rama cuya fusión no esté confirmada (`merge_not_confirmed`) y sobre un worktree con cambios sin confirmar (`dirty_worktree`), salvo `--force`. Con `--force`, DEBE saltarse la exigencia de fusión confirmada y la de árbol limpio, y DEBE informar en el JSON de los cambios sin confirmar descartados y, si la rama no estaba fusionada, de la punta (SHA) que tenía la rama eliminada para poder recuperarla. `done` NUNCA DEBE tocar ramas remotas ni ejecutar operaciones remotas (REQ-25.1). Si queda abierto un linaje RDD asociado al worktree, `done` DEBE emitir un aviso que nombre el linaje y la acción (`axiom review abandon`), sin impedir la operación. Cada rama del cambio, incluida la de archivado, DEBE procesarse por separado, sin selector de rol en esta versión: las confirmadas se eliminan y las no confirmadas se conservan y se reportan, y el código de salida DEBE ser distinto de cero si alguna rama no pudo procesarse.

En modo `in_place` no hay worktree que eliminar: `done` DEBE devolver el checkout a la rama de integración y, después, eliminar la rama local, y solo si el árbol está limpio. En `in_place`, `--force` solo salta la exigencia de fusión confirmada: NUNCA salta la de árbol limpio, de modo que con cambios sin confirmar `done` rechaza con `dirty_worktree` sin cambiar de rama ni eliminar nada.
(Previously: no mencionaba la rama de archivado)

#### Scenario: Eliminación tras una fusión confirmada
- **DADO** una rama `feat/x` fusionada, su worktree con el árbol limpio y una rama remota `origin/feat/x`
- **CUANDO** se ejecuta `axiom branch done --change x`
- **ENTONCES** el worktree se retira del disco y de `git worktree list`
- **Y** la rama local `feat/x` se elimina
- **Y** la rama remota `origin/feat/x` permanece intacta y no consta ninguna operación remota

#### Scenario: Una rama integrada con squash se elimina tras confirmarse por `equivalent_tree`
- **DADO** `feat/x` integrada en `main` mediante un squash, confirmada por la capa `equivalent_tree` (REQ-25.16)
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

#### Scenario: Linaje RDD abierto genera un aviso
- **DADO** un worktree con un linaje RDD abierto
- **CUANDO** se ejecuta `axiom branch done --change x` sobre una rama fusionada
- **ENTONCES** la salida incluye un aviso que nombra el linaje y `axiom review abandon`
- **Y** la eliminación se realiza

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

---

### Requirement: Veredicto Único de Rama, por Papel del Repositorio (REQ-25.18)

Las guardas A (REQ-25.19), B (`blocked(wrong_branch)` de `axiom-sdd-cli-integration`) y D (`guard`, REQ-25.21) DEBEN consultar un único veredicto: «¿esta rama es la que corresponde a este cambio en este repositorio según su papel?». El veredicto DEBE ser uno de los tokens `correct`, `wrong_branch` o `not_applicable` (REQ-25.2). La consulta se hace en uno de dos contextos: el **ordinario** y el de **archivado** (la fase archive de `axiom sdd status` y el escritor de `axiom sdd archive-compose`). Las reglas son:

- **Repositorio con rama de cambio, contexto ordinario**: es correcta la rama del cambio o, si existen ramas de rol de ese cambio, cualquiera de ellas; cualquier otra rama (incluida la de integración, la de archivado y un HEAD desacoplado) es `wrong_branch`.
- **Repositorio con rama de cambio, contexto de archivado**: es correcta únicamente la rama de archivado `<prefijo>/<cambio>-archive`; cualquier otra (la del cambio, la de integración, una de rol o una ajena) es `wrong_branch`, y su continuación DEBE ser `axiom branch ensure --cwd <ruta> --change <c> --archive`.
- **Repositorio de specs en multirepo**: es correcta únicamente la rama de integración, también en el contexto de archivado (no hay rama de archivado); una rama de rol del cambio o cualquier otra es `wrong_branch`.
- **Repositorio de código en multirepo**: el veredicto solo se evalúa cuando el consultante es apply (el estado de aplicación de `sdd status` o la guarda D sobre una ruta de ese repositorio); en otras fases es `not_applicable`.
- **Exenciones**: es `not_applicable` para un cambio cuyo `kickoff.yaml` no contiene instantánea (REQ-25.37), para los sumideros `e2e-cumulative` y `cumulative-<rol>` por nombre, y cuando la ruta no pertenece a ningún repositorio git o git no está disponible (en esos casos la consulta NO DEBE bloquear y DEBE advertirlo).

Los nombres esperados DEBEN obtenerse de la instantánea del `kickoff.yaml` cuando existe (REQ-25.36) y, si el cambio aún no tiene kickoff (se está creando), de la política de `axiom.yaml` (REQ-25.3); el nombre de la rama de archivado se toma del registro local si `ensure --archive` recibió `--prefix` (REQ-25.12). Un `wrong_branch` DEBE incluir la rama actual, la esperada, el repositorio y la continuación exacta: la orden `axiom branch ensure` con `--cwd`, `--change` y, si procede, `--lane` o `--archive`. El veredicto NO consulta si la rama de integración contiene cambios activos: con el archive en un PR documental aparte es normal que los contenga temporalmente (REQ-25.20).
(Previously: lo consultaban cuatro capas, incluida una de CI; el veredicto no distinguía el contexto de archivado y los valores eran etiquetas en castellano)

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
- **DADO** un cambio con los roles `core` y `qa` y el repositorio desplegado en `feat/x-qa`
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

#### Scenario: En el contexto de archivado la rama del cambio, la de integración y las de rol son incorrectas
- **DADO** un cambio con instantánea y el repositorio desplegado, por separado, en `feat/x`, en `main` y en `feat/x-core`
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

---

### Requirement: Guarda A en Todo Escritor Go de la Carpeta del Cambio (REQ-25.19)

Todo escritor Go que cree o modifique ficheros dentro de la carpeta de un cambio DEBE consultar el veredicto de REQ-25.18 antes de escribir y, ante `wrong_branch`, DEBE negarse sin escribir nada y devolver el error tipado con la continuación. Esto incluye como mínimo: la creación de un incremento (el escritor compartido por el dashboard y `axiom change create`), la migración al acumulativo, la creación de handoff (`axiom handoff create` y el servicio del dashboard), `kickoff seal`, el registro de compuertas (`gate record`), el marcador de instancia del consentimiento de autoridad de edición, `axiom sdd integrate` (`integration.yaml`) y `axiom sdd archive-compose` (que consulta el veredicto en el contexto de archivado). El dashboard y `axiom change create` DEBEN ejecutar `ensure` y después escribir con el escritor guardado; el escritor guardado DEBE negarse fuera de rama aunque se invoque sin haber ejecutado `ensure`. La TUI solo lista cambios y no ejecuta `ensure`; cualquier escritura que alcance la carpeta de un cambio pasa por un escritor guardado. La guarda NO DEBE bloquear cambios exentos (REQ-25.18). Cada escritor DEBE tener una prueba propia de negativa.
(Previously: la lista de escritores incluía la creación desde la TUI y `archive-compose` consultaba el veredicto ordinario)

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

---

### Requirement: Cambios Pendientes de Archivar Visibles en `status` y en el Dashboard (REQ-25.20)

Los cambios pendientes de archivar de un repositorio con rama de cambio, es decir, los cambios activos presentes en el árbol de la rama de integración (directorios bajo `changes/` del planning home distintos de `archive/` y de los sumideros `e2e-cumulative` y `cumulative-<rol>`, excluidos por nombre), DEBEN mostrarse en `axiom branch status` (REQ-25.15) y en el dashboard (REQ-25.34 de `local-web-dashboard`). Es información, no una comprobación: NO existe ningún código de fallo por su presencia, NO hay integración con la CI y NO DEBE bloquear ninguna orden ni guarda. La evaluación DEBE hacerse sobre la referencia de integración local de REQ-25.10 (`origin/<integración>` si existe en local y, si no, la rama local), sin cambiar de rama y sin operaciones remotas; si esa referencia no puede resolverse, la lista se omite con un aviso y sin error. Solo los directorios cuentan como cambios; los ficheros sueltos de `changes/` no lo son. En el repositorio de specs de un multirepo no aplica: allí los cambios activos viven en la rama de integración de forma normal. Cada cambio pendiente DEBE indicar la continuación para archivarlo (`axiom branch ensure --change <c> --archive`, REQ-25.45).
(Previously: `axiom branch check` exigía en CI que la rama de integración no tuviera cambios activos y fallaba con código distinto de cero; el invariante se elimina porque, con el archive en un PR documental aparte, es normal que la integración los tenga temporalmente)

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

---

### Requirement: `guard`: Hook Opcional que Bloquea los Documentos de Axiom y Avisa en el Resto (REQ-25.21)

`axiom branch guard` DEBE ser el hook de la capa D que usan los runtimes con hooks. Recibe del runtime la herramienta y la ruta que va a escribir el agente, y su efecto depende de la ruta:

1. **Documentos de Axiom** (la carpeta de un cambio u `odd/tasks/`): DEBE **bloquear** (con el mecanismo de bloqueo del runtime) cuando el veredicto de REQ-25.18 sea `wrong_branch`, con un mensaje que incluya la continuación de `ensure`. Rige en cualquier proyecto Axiom, con independencia de que haya un cambio registrado (supuesto S-18).
2. **Cualquier otro fichero** de un repositorio con un cambio activo registrado por `ensure` en el registro local (REQ-25.14) y no cerrado por `done`: DEBE **avisar sin bloquear** cuando la rama del checkout que contiene el fichero no es la que corresponde al cambio. El aviso DEBE incluir la ruta del worktree correcto o, en `in_place`, la rama esperada; con varios cambios activos registrados en el repositorio, nombra los de todos. El repositorio de specs de un `multirepo` queda exento de este aviso.
3. En cualquier otro caso DEBE no hacer nada: sin cambio activo registrado en el repositorio, o fuera de un proyecto Axiom, termina sin bloquear, sin avisar y sin escribir.

DEBE leer la rama actual leyendo `HEAD` del repositorio, sin lanzar git ni ningún otro proceso, y DEBE ser fail-open: ante cualquier error interno, configuración o registro ilegibles o expiración de tiempo, NO DEBE bloquear ni avisar. Solo afecta a las escrituras del agente a través del runtime: NUNCA al editor humano ni a las órdenes de la CLI. Es la única capa, junto con la doctrina, que cubre el documento ODD. NO DEBE escribir ningún fichero. El canal no bloqueante por el que cada runtime entrega el aviso se decide en `sdd-design`.
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

#### Scenario: Sin cambio activo registrado en el repositorio no hace nada con el código
- **DADO** un proyecto Axiom sin ningún cambio activo registrado por `ensure` y el checkout en `main`
- **CUANDO** el runtime invoca el hook para escribir `internal/foo/foo.go`
- **ENTONCES** el hook termina sin bloquear ni avisar

#### Scenario: El repositorio de specs de un multirepo queda exento del aviso
- **DADO** un multirepo con un cambio activo registrado y el repositorio de specs en su rama de integración
- **CUANDO** el runtime invoca el hook para escribir un fichero que no es un documento de Axiom en el repositorio de specs
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: Inerte fuera de un proyecto Axiom
- **DADO** un directorio sin `axiom.yaml`, `.axiom-workspace` ni `openspec/` en él ni en sus ancestros
- **CUANDO** el runtime invoca el hook para cualquier escritura
- **ENTONCES** termina sin bloquear ni avisar
- **Y** no escribe ningún fichero

#### Scenario: Fail-open ante un error interno
- **DADO** un proyecto Axiom con `axiom.yaml` o registro local ilegibles
- **CUANDO** el runtime invoca el hook para escribir en una carpeta de cambio o en un fichero de código
- **ENTONCES** el hook no bloquea ni avisa

#### Scenario: El hook no lanza git ni ningún otro proceso
- **DADO** un proyecto Axiom y un observador de procesos
- **CUANDO** el runtime invoca el hook para cualquier escritura, protegida o no
- **ENTONCES** el observador no registra ninguna invocación de git ni de ningún otro proceso
- **Y** la rama actual se obtiene leyendo `HEAD`

#### Scenario: El documento ODD está cubierto
- **DADO** un monorepo con el checkout en la rama de integración y la política por defecto
- **CUANDO** el runtime invoca el hook para escribir `odd/tasks/mi-feature.md`
- **ENTONCES** el hook bloquea con la continuación `axiom branch ensure --lane odd --change mi-feature`

#### Scenario: Solo afecta al agente a través del runtime
- **DADO** un proyecto con el hook instalado
- **CUANDO** una persona guarda un fichero desde su editor o ejecuta una orden de la CLI
- **ENTONCES** el hook no se ejecuta, porque solo lo invoca el runtime en las escrituras del agente y no se instala ningún hook de git ni observador de ficheros

---

### Requirement: Instalación del Hook por `axiom sync` y Prohibición de Hooks de Git (REQ-25.22)

`axiom sync` DEBE instalar `axiom branch guard` como hook en los runtimes donde existan hooks demostrados (Claude Code, Codex y OpenCode) de forma idempotente y determinista (misma entrada, mismo resultado), sin duplicar entradas en ejecuciones repetidas y sin alterar las entradas de hooks ajenas. Una ejecución posterior de `axiom sync` que ya no deba instalarlo DEBE retirarlo. NO DEBE instalar hooks de git (`pre-commit`, `pre-push`, ni otros) en ningún repositorio, propio o de terceros. NO DEBE instalarse en Pi, que no tiene canal demostrado. Dado que la configuración de Claude Code es global y corre en todo repositorio, la inercia de REQ-25.21 fuera de proyectos Axiom y fuera de repositorios con un cambio activo registrado es condición de esta instalación: el hook no hace nada, no lanza procesos y no escribe en ellos. El formato del hook de Codex se verifica en su documentación oficial al implementar (A-2).
(Previously: la inercia exigida era solo fuera de proyectos Axiom)

#### Scenario: Instalación en los tres runtimes
- **DADO** una máquina con Claude Code, Codex y OpenCode configurados
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
- **DADO** el hook instalado en la configuración global de Claude Code y un repositorio de terceros sin marcadores de Axiom
- **CUANDO** el runtime invoca el hook para escribir en ese repositorio
- **ENTONCES** el hook no bloquea, no avisa, no lanza procesos y no escribe nada

---

### Requirement: Las Skills y la Doctrina Delegan en el Verbo (REQ-25.23)

Las skills `sdd-apply`, `sdd-spec`, `sdd-design`, `sdd-propose`, `sdd-explore` y `sdd-archive`, la convención compartida `_shared` y los orquestadores por runtime DEBEN delegar en `axiom branch` la creación de ramas y worktrees. En concreto: (1) el Paso 2c de `sdd-apply` (cuestionario de worktree y `git worktree add` manual) y las reglas ODD-5.1, ODD-5.2, ODD-5.5 y ODD-5.6 («specs siempre en main», `git commit` y `git push` previos al handoff, `scripts/cleanup-worktree.sh`) DEBEN retirarse y sustituirse por la invocación de `axiom branch ensure`/`done`; (2) `sdd-spec` y `sdd-design` NO DEBEN prescribir trabajar en `main`/`master` ni prohibir ramas y worktrees; (3) `sdd-apply` DEBE proponer la rama destino del PR a partir del plan (REQ-25.4) en lugar de preguntarla; (4) las skills DEBEN usar las rutas absolutas del JSON de `ensure` con `--cwd`, `git -C` y rutas absolutas, nunca el directorio de trabajo heredado; (5) las rutas y literales de rama de integración de las skills DEBEN referirse a la rama de integración resuelta y no al literal `main`; (6) `sdd-archive` DEBE indicar que, en monorepo, el archive se hace en la rama de archivado creada con `axiom branch ensure --change <c> --archive` una vez fusionado el PR del cambio (REQ-25.39).

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
- **Y** el motor usa la rama de integración resuelta de la política en lugar de un literal `main`, que solo permanece como valor por defecto de la política (REQ-25.3)

#### Scenario: `sdd-archive` remite a la rama de archivado
- **DADO** el texto de `sdd-archive/SKILL.md`
- **CUANDO** se inspecciona (pin de texto)
- **ENTONCES** indica crear la rama de archivado con `axiom branch ensure --change <c> --archive` tras fusionar el PR del cambio y ejecutar allí el archive

---

### Requirement: Dogfooding en este Repositorio (REQ-25.24)

El repositorio de Axiom DEBE declarar en su `axiom.yaml` el bloque `git:` con `integration_branch: main` y `branch_prefix: inc`, y DEBE superar `axiom workspace validate`. `CLAUDE.md`, `AGENTS.md` y `GEMINI.md` DEBEN remitir a `axiom branch` para crear ramas y worktrees y NO DEBEN decir que el worktree se decide en apply. El destino de `scripts/axiom-worktree.ps1` y `.sh` y de `scripts/claude-hooks/guard-primary-checkout.ps1` (envoltorio fino o retirada) se resuelve en `sdd-design` (Q6). El producto no añade ninguna exención para cambios preexistentes: un cambio activo ya fusionado, como `inc-23-knowledge-profile` mientras no se archive, aparece como cambio pendiente de archivar (REQ-25.20) sin bloquear nada.
(Previously: el prefijo lo determinaba `sdd-design` y la activación de un comprobador de CI en este repositorio exigía archivar antes `inc-23-knowledge-profile`)

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

---

## Supuestos confirmados en la compuerta de spec (2026-10-15)

Los supuestos son decisiones mínimas tomadas para que los requerimientos sean comprobables. El usuario los confirmó en la compuerta de spec del 2026-10-15; S-6 se corrigió según D3. Q1, Q2 y Q6 de la propuesta pertenecen a `sdd-design` y se citan como dependencias.

> **Nota de reapertura (2026-10-17):** la spec se reabrió por D16 a D19. D15 queda retirada y S-12 se resuelve ahora por D16. S-4 se actualiza por la lectura G-5 y S-5 queda sin objeto por D17. Los supuestos S-15 a S-18 se añaden en la reapertura.

| ID | Supuesto | Estado | Dónde afecta |
|---|---|---|---|
| S-1 | La propuesta no indica cómo conocer los roles cuando `ensure` se ejecuta al nombrar el cambio, antes de `kickoff seal`. `ensure` es incremental: antes del sellado planifica solo `<prefijo>/<cambio>` en el repositorio del planning home (idéntica en ambos modelos) y tras el sellado completa ramas de rol y repositorios de código. | Confirmado | REQ-25.7 |
| S-2 | La respuesta a `decision_required` de `ask` se da con la opción `--mode worktree\|in_place` de `ensure`. | Confirmado | REQ-25.8 |
| S-3 | `fullstack` no existe en `axiom.yaml`. Cuenta como un único rol que toca todos los repositorios declarados por algún rol y, en monorepo, el del planning home. | Confirmado | Términos, REQ-25.4 |
| S-4 | `<repo>` es el último segmento de la ruta (`path`) del repositorio (G-5), y `<workspace>-wt` usa el nombre de la carpeta maestra. | Confirmado (actualizado por G-5) | REQ-25.11 |
| S-5 | Retirado por D17: existía para fijar el mecanismo con el que el comprobador de CI evaluaba un árbol desacoplado, y el comprobador se ha eliminado. | Sin objeto (D17) | — |
| S-6 | Resuelto según D3: `done` elimina el worktree y la rama local una vez confirmada la fusión por capas (REQ-25.16); se niega si la rama no está fusionada o si el árbol está sucio, salvo `--force`; en `in_place` devuelve el checkout a la rama de integración y elimina la rama local solo con el árbol limpio (`--force` no salta esa exigencia); nunca toca ramas remotas; procesa cada rama por separado, incluida la de archivado, sin selector de rol en esta versión. | Confirmado (corregido, D3) | REQ-25.17 |
| S-7 | Si el cambio ya tiene `kickoff.yaml` pero sin instantánea queda exento; si aún no tiene `kickoff.yaml` se aplica la política de `axiom.yaml`. | Confirmado | REQ-25.18 |
| S-8 | `--fetch` fallido termina con `fetch_failed` sin crear nada. | Confirmado | REQ-25.10 |
| S-9 | `blocked(wrong_branch)` se evalúa antes de `blocked(edit_authority_missing)`. | Confirmado | `axiom-sdd-cli-integration` |
| S-10 | La pregunta de `ask` del carril ODD es una preferencia de entorno personal, no una pregunta de gobernanza, y no contraviene REQ-21.2. | Confirmado | `sdd-lane-kickoff` |
| S-11 | Si el directorio de partida es un worktree enlazado sin marcador ascendente, el planning home se resuelve desde el checkout principal del mismo clon. | Confirmado | `planning-home-resolution` |
| S-12 | Resuelto por D16 (sustituye a la decisión anterior): en monorepo el archive se hace en un PR documental aparte, en la rama de archivado `<prefijo>/<cambio>-archive` (D18), una vez fusionado el PR del cambio. REQ-21.16 vuelve a su forma original con dos cambios (rama de integración configurada y sin `monorepo-decoupled`) y la precondición incluye la evidencia `pr_merged` del PR del cambio. Archivar en otra rama se rechaza con `blocked(wrong_branch)` y la continuación `ensure --archive`. `multirepo` no cambia. | Confirmado (D16) | `sdd-archive-lifecycle` |
| S-13 | `repositories:` identifica cada entrada por `path` y admite `integration_branch` y `branch_prefix` como ajustes propios por repositorio. | Confirmado (incluye `branch_prefix` por repositorio) | `workspace-topology` |
| S-14 | La TUI se cubre como escenario de paridad en `local-web-dashboard`; `tui-ui-parity` no figura en las capacidades modificadas de la propuesta. | Confirmado | `local-web-dashboard` |
| S-15 | Un rol llamado `archive` colisionaría con la rama de archivado `<prefijo>/<cambio>-archive` y se rechaza con `invalid_name` al planificar las ramas de rol. | Propuesto en la reapertura (2026-10-17) | REQ-25.4 |
| S-16 | Con `--archive` solo se admite `--prefix`; `--branch` y `--base` se rechazan con `plan_conflict`; sin `--prefix` la rama de archivado deriva de la rama del cambio registrada como `<rama del cambio>-archive`. Una base «contiene el cambio fusionado» cuando su árbol incluye `changes/<cambio>/` del planning home. | Propuesto en la reapertura (2026-10-17) | REQ-25.12, REQ-25.45 |
| S-17 | Para el repositorio cuya ruta es `.`, `<repo>` es el nombre del directorio de su raíz git; dos repositorios del plan con el mismo último segmento se rechazan con `invalid_name`. | Propuesto en la reapertura (2026-10-17) | REQ-25.11 |
| S-18 | D19: el bloqueo de documentos de Axiom (carpeta del cambio y `odd/tasks/`) rige en todo proyecto Axiom aunque no haya cambio registrado, para cubrir el carril ODD antes de `ensure`; el aviso en otros ficheros solo rige en repositorios con un cambio activo registrado; el repositorio de specs de un multirepo queda exento del aviso pero no del bloqueo de documentos. | Propuesto en la reapertura (2026-10-17) | REQ-25.21 |

### Dependencias de diseño

| ID | Dependencia | Dónde afecta |
|---|---|---|
| Q1 | Versión mínima de git para `git merge-tree --write-tree` (2.38). | REQ-25.16 (design) |
| Q2 | Coexistencia del registro local con `<git common dir>/gentle-ai/`. | REQ-25.14 (design) |
| Q6 | Destino de los scripts locales y de `guard-primary-checkout.ps1`. | REQ-25.24 (design) |
