# Delta para Publicación de Documentación SDD (sdd-docs-publication)

> **Cambio:** `marcas-pruebas` (C-06-d). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> `axiom sdd publish` ordinario NO cambia de comportamiento: las excepciones solo alcanzan al publicador de marcas.

## ADDED Requirements

### Requirement: Publicación de Marcas en la Integración del Repositorio de Specs, Acotada a `pruebas-estado.yaml` (prov:PUB:1)

El publicador de marcas DEBE confirmar y subir únicamente `pruebas-estado.yaml` del cambio, mediante la lista exacta de rutas del modo acotado (RF-PUB-01 modificado), sobre la rama de integración del repositorio de specs de un `multirepo`. DEBE exigir un destino de publicación en el propio repositorio (`InPlace`) y NO DEBE usar ningún worktree registrado. Cualquier otro documento sucio del repositorio de specs NO DEBE bloquear la publicación ni confirmarse ni alterarse (ni en el árbol de trabajo ni en el índice): DEBE informarse como aviso. Sin identidad git (`user.name`, `user.email`), sin remoto `origin` o con la rama distinta de la integración, DEBE rechazar con los códigos tipados existentes antes de crear ningún commit. Tras agotar el reintento (prov:PUB:2), DEBE devolver un rechazo tipado con etiqueta en castellano y continuación, conservando las entradas en el fichero local. En monorepo NO DEBE publicar nada.

#### Scenario: Solo se confirma `pruebas-estado.yaml`
- **DADO** entradas pendientes y otros documentos modificados del repositorio de specs
- **CUANDO** el publicador de marcas publica
- **ENTONCES** el commit contiene solo `pruebas-estado.yaml` del cambio
- **Y** los demás ficheros siguen modificados e intactos
- **Y** el resultado incluye un aviso que los nombra

#### Scenario: Publica en la integración, no en un worktree
- **DADO** un worktree registrado obsoleto del cambio y el repositorio de specs en su rama de integración
- **CUANDO** el publicador de marcas publica
- **ENTONCES** el commit y el push se hacen en el repositorio de specs in situ, sobre la rama de integración

#### Scenario: Rama distinta de la integración
- **DADO** el repositorio de specs en una rama que no es la de integración
- **CUANDO** el publicador de marcas publica
- **ENTONCES** se rechaza con el código tipado `wrong_branch` y no se crea ningún commit

#### Scenario: Identidad ausente o remoto inexistente
- **DADO** un repositorio de specs sin identidad git, o sin remoto `origin`
- **CUANDO** el publicador de marcas publica
- **ENTONCES** se rechaza con `identity_missing` o `remote_missing` respectivamente y no se crea ningún commit

#### Scenario: Monorepo no publica
- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se solicita publicar marcas
- **ENTONCES** se rechaza como no disponible y no se ejecuta ninguna operación git

### Requirement: Un Único Reintento por Reproducción ante *non-fast-forward* en el Publicador de Marcas (prov:PUB:2)

Cuando el push del publicador de marcas es rechazado por `push_non_fast_forward`, el publicador DEBE hacer un único reintento por reproducción: `fetch` del remoto `origin`, descartar el commit local de marcas, avance rápido a la integración remota, volver a añadir las entradas pendientes de `pruebas-estado.yaml`, nuevo commit y nuevo push. No DEBE usar `rebase`, forzar el push ni omitir hooks, y NO DEBE intentar un segundo reintento. El `fetch` queda autorizado por el mismo clic de confirmación de prov:WEB:3. Si el segundo push también falla, DEBE devolver el rechazo tipado correspondiente (por ejemplo `push_non_fast_forward`) con su continuación. Las entradas pendientes NO DEBEN perderse en ningún caso, ni tras el reintento fallido. El reintento NO DEBE ser accesible desde `axiom sdd publish`.

#### Scenario: El reintento resuelve el rechazo
- **DADO** un remoto cuya integración avanzó con commits de otra persona y entradas pendientes locales
- **CUANDO** el publicador de marcas publica
- **ENTONCES** el primer push es rechazado por *non-fast-forward*
- **Y** se ejecuta `fetch`, se avanza a la integración remota y se vuelven a añadir las entradas
- **Y** el segundo push tiene éxito y el commit remoto contiene las entradas pendientes y las que subió la otra persona

#### Scenario: El reintento falla también
- **DADO** un remoto que rechaza el segundo push por *non-fast-forward*
- **CUANDO** el publicador de marcas publica
- **ENTONCES** devuelve `push_non_fast_forward` con su etiqueta y continuación
- **Y** no hay un tercer intento
- **Y** las entradas pendientes siguen en el fichero local

#### Scenario: Otros rechazos no se reintentan
- **DADO** un hook `pre-push` que rechaza
- **CUANDO** el publicador de marcas publica
- **ENTONCES** devuelve el rechazo tipado de hook y no se ejecuta `fetch` ni un segundo push

#### Scenario: `axiom sdd publish` ordinario sigue sin reintentar
- **DADO** el mismo remoto adelantado
- **CUANDO** se ejecuta `axiom sdd publish --change x`
- **ENTONCES** termina con *non-fast-forward* sin `fetch` ni segundo push

## MODIFIED Requirements

### Requirement: Verbo `axiom sdd publish --change <x>` y Lista Cerrada de Rutas de Documentación (RF-PUB-01)

La CLI DEBE exponer `axiom sdd publish --change <nombre>` con `--cwd` y `--json`, como subcomando de `axiom sdd` y no de `axiom branch`. La orden DEBE operar únicamente sobre las **rutas de documentación**, una lista cerrada formada por: `openspec/changes/<x>/**`; la carpeta de `openspec/changes/archive/**` que corresponde al cambio; `openspec/specs/**`; `openspec/INDEX.md`; y las rutas de contexto técnico mantenido declaradas en `axiom.yaml` (`technical_context.maintained`). Las rutas de `technical_context.reference` y `odd/tasks/**` NO DEBEN pertenecer nunca a la lista. Tampoco pertenece a la lista el **estado local** del cambio: los ficheros cuyo nombre empieza por punto situados directamente en la raíz de `openspec/changes/<x>/` o en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<x>/`). `publish` NUNCA DEBE confirmarlos, ni con `--archive` ni en publicaciones posteriores. Los ficheros con punto de subcarpetas siguen siendo documentación. Toda ruta que no esté en la lista es **código** a efectos de esta capacidad. El indicador opcional `--archive` selecciona la rama de archivado del cambio como rama asignada que `publish` espera (RF-PUB-03 y RF-ARC-08): la publicación posterior al archivado DEBE invocarse como `axiom sdd publish --change <x> --archive`. Sin `--change`, o con un nombre de cambio inválido, la orden DEBE rechazarse sin efectos.

La publicación admite un modo de **lista exacta de rutas** (`Request.Only`), reservado al publicador de marcas (prov:PUB:1) y NO accesible desde `axiom sdd publish`: la lista DEBE validarse contra la lista cerrada y solo puede contener `pruebas-estado.yaml` del cambio; cualquier otra ruta, aunque sea de documentación, DEBE rechazarse antes de ejecutar git. En ese modo, el commit contiene solo las rutas de la lista exacta.
(Previously: no existía un modo de lista exacta; la lista cerrada era la única forma de seleccionar rutas)

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

#### Scenario: El estado local de la raíz no se publica

- **DADO** `openspec/changes/x/.integrations-outbox.yaml` y `openspec/changes/archive/2026-11-19-x/.integrations-outbox.yaml`
- **CUANDO** `publish --change x` y `publish --change x --archive` clasifican sus rutas
- **ENTONCES** ninguno de los dos es documentación
- **Y** ninguna publicación los confirma

#### Scenario: Un fichero con punto anidado sigue siendo documentación

- **DADO** `openspec/changes/archive/2026-11-19-x/specs/.nota`
- **CUANDO** `publish --change x --archive` lo clasifica
- **ENTONCES** es documentación del cambio

#### Scenario: La lista exacta admite solo `pruebas-estado.yaml`
- **DADO** una petición con lista exacta que contiene `openspec/changes/x/pruebas-estado.yaml`
- **CUANDO** se valida contra la lista cerrada
- **ENTONCES** se acepta y el commit contiene solo ese fichero

#### Scenario: La lista exacta rechaza otras rutas
- **DADO** una petición con lista exacta que contiene `openspec/changes/x/design.md` o `internal/x.go`
- **CUANDO** se valida
- **ENTONCES** se rechaza antes de ejecutar ninguna operación git

#### Scenario: El verbo no ofrece la lista exacta
- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd publish --help`
- **ENTONCES** la ayuda no lista ningún indicador de lista exacta de rutas

### Requirement: Negativas Tipadas sin Reintento y sin Estado a Medias (RF-PUB-05)

`publish` DEBE distinguir con un código tipado estable en inglés, acompañado de una etiqueta legible en castellano y de la continuación para la persona, al menos estas negativas: push rechazado por no ser *fast-forward*; hook de git que rechaza el commit o el push; identidad de autor de git ausente; HEAD desacoplado; rama de integración no publicable; `wrong_branch`; `code_commits_unpublished`; almacén sin ficheros; y remoto `origin` inexistente (`remote_missing`) o inalcanzable. Ante cualquiera de ellas la orden DEBE terminar con código de salida distinto de cero y NO DEBE reintentar, ejecutar `fetch`, forzar el push ni omitir hooks. La comprobación del remoto `origin` DEBE preceder al commit: con `remote_missing` NO se crea ningún commit. Si el commit local de documentación ya se creó y el push falla (p. ej. *non-fast-forward* o hook de push), el commit DEBE conservarse y la salida DEBE informar de ello y de la orden de push para la persona. Una negativa NO DEBE modificar `gates.yaml`, `kickoff.yaml` ni ningún otro registro del cambio.

**Excepción acotada.** Solo el publicador de marcas (prov:PUB:1), y únicamente ante `push_non_fast_forward`, DEBE hacer el único reintento por reproducción de prov:PUB:2, con un `fetch` autorizado por el mismo clic de confirmación; en ese reintento NO DEBE forzar el push ni omitir hooks. `axiom sdd publish` NO DEBE reintentar ni ejecutar `fetch` en ningún caso.
(Previously: ninguna negativa admitía reintento ni `fetch`, sin excepción)

#### Scenario: Un push rechazado por no ser fast-forward no se reintenta

- **DADO** un remoto cuya `inc/x` avanzó con commits que la rama local no contiene
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** termina con el código tipado de *non-fast-forward* y código de salida distinto de cero
- **Y** no se ejecuta `fetch` ni un segundo `push`
- **Y** el commit local de documentación se conserva

#### Scenario: Un hook que rechaza produce una negativa tipada

- **DADO** un hook `pre-push` que termina con error
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** termina con el código tipado de hook rechazado
- **Y** la orden no reintenta con `--no-verify`

#### Scenario: Identidad de autor ausente

- **DADO** un repositorio sin `user.name` ni `user.email` y documentación modificada
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** termina con el código tipado de identidad ausente, nombrando las claves de git que faltan
- **Y** no se crea ningún commit

#### Scenario: Remoto inexistente

- **DADO** un repositorio sin remoto `origin`
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** termina con el código tipado de remoto inexistente (`remote_missing`)
- **Y** no se crea ningún commit, porque la comprobación del remoto precede al commit

#### Scenario: Una negativa no modifica los registros

- **DADO** cualquiera de las negativas anteriores
- **CUANDO** termina `publish`
- **ENTONCES** `gates.yaml` y `kickoff.yaml` tienen el mismo contenido que antes de la orden, salvo el que `publish` confirmó en un commit ya creado

#### Scenario: La excepción es solo del publicador de marcas
- **DADO** un *non-fast-forward* en el publicador de marcas
- **CUANDO** publica
- **ENTONCES** hace un único `fetch` y un único segundo push según prov:PUB:2
- **Y** ninguna otra negativa ni otro camino de publicación reintenta

### Requirement: Aislamiento de la Operación Remota (RF-PUB-07)

La capacidad de subir una rama a un remoto DEBE vivir en un puerto de publicación propio, separado del puerto `Git` y del puerto de escritura del motor de ramas, y solo `axiom sdd publish` y el publicador de marcas (prov:PUB:1) DEBEN poder alcanzarlo. El publicador de marcas DEBE implementarse en `internal/cli` e inyectarse en el dashboard mediante una interfaz del propio dashboard: el dashboard NO DEBE importar `sddpublish` ni ejecutar `git commit`, `git push` ni `git fetch`. `axiom branch ensure`, `status`, `done` y `guard`, `axiom archive` y `axiom sdd gate record` NO DEBEN alcanzar ese puerto ni ejecutar `git commit` ni `git push`. La ejecución de git del puerto de publicación DEBE pasar por el único punto de ejecución de git del producto. Una prueba de arquitectura o de dependencias DEBE fijar este aislamiento, y el invariante «ningún subcomando de `axiom branch` publica» (RF-BRA-01 de `change-branch-lifecycle`) DEBE seguir verificado por el observador del único punto de ejecución.
(Previously: solo `axiom sdd publish` podía alcanzar el puerto de publicación)

#### Scenario: `ensure`, `done` y `status` no publican

- **DADO** un repositorio con remoto y un observador instalado en el único punto de ejecución de git
- **CUANDO** se ejecutan `axiom branch ensure`, `status` y `done` (incluido `ensure --archive`)
- **ENTONCES** el observador no registra ningún `push` ni `commit`

#### Scenario: `gate record` no ejecuta git

- **DADO** un observador instalado en el único punto de ejecución de git
- **CUANDO** se ejecuta `axiom sdd gate record --decision approved`, con o sin el indicador de la salida 2
- **ENTONCES** el observador no registra ningún `commit`, `push`, `fetch`, `pull` ni `ls-remote`

#### Scenario: Una prueba de dependencias fija el aislamiento

- **DADO** el código fuente del producto
- **CUANDO** se ejecuta la prueba de arquitectura o de dependencias
- **ENTONCES** el paquete del puerto de publicación solo es importado por `internal/cli` (el verbo `axiom sdd publish` y el publicador de marcas) y sus pruebas
- **Y** la prueba falla si `internal/branch/**`, el código de `gate record` o `internal/dashboard` lo importan

#### Scenario: El dashboard no ejecuta git
- **DADO** el código fuente del dashboard
- **CUANDO** se ejecuta la prueba de arquitectura
- **ENTONCES** el dashboard no invoca `commit`, `push` ni `fetch` por su cuenta y solo usa la interfaz inyectada
