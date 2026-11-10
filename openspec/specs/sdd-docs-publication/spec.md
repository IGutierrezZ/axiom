# Especificación de Publicación Ligera de Documentación SDD (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** capacidad nueva (especificación completa, no delta). `sdd-archive` la promoverá a `openspec/specs/sdd-docs-publication/spec.md`.
> **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.

## Propósito

Definir el verbo nativo `axiom sdd publish`, que confirma y sube, y solo eso, los documentos de un cambio SDD a la rama que el planificador le asigna, de modo que aprobar una parada, cerrar `tasks` en un kickoff autorizado o archivar deje los documentos disponibles para otra persona sin una PR fusionada. Fija la lista cerrada de rutas de documentación, la política de rama y remoto, las negativas tipadas y el aislamiento de la operación remota respecto de `axiom branch` y de `axiom sdd gate record`.

## Términos

- **Rama asignada:** la rama que `axiom branch ensure` y el planificador de ramas asignan al cambio en el repositorio que corresponda (REQ-25.4 de `change-branch-lifecycle`): rama del cambio, rama de rol o rama de archivado. En el repositorio de specs de un `multirepo`, su rama de integración (REQ-27.7).
- **Rutas de documentación** (lista cerrada, REQ-27.5): las únicas rutas que `publish` puede confirmar.
- **Commit sin subir:** commit de la rama actual que no está contenido en `refs/remotes/origin/<rama>` local o, si esa referencia no existe, en la base de integración local.
- **Negativa informativa:** terminación de `publish` con código de salida distinto de cero y un código tipado, sin efectos remotos y sin anular ninguna decisión ya registrada.

## Requirements

### Requirement: Verbo `axiom sdd publish --change <x>` y Lista Cerrada de Rutas de Documentación (REQ-27.5)

La CLI DEBE exponer `axiom sdd publish --change <nombre>` con `--cwd` y `--json`, como subcomando de `axiom sdd` y no de `axiom branch`. La orden DEBE operar únicamente sobre las **rutas de documentación**, una lista cerrada formada por: `openspec/changes/<x>/**`; la carpeta de `openspec/changes/archive/**` que corresponde al cambio; `openspec/specs/**`; `openspec/INDEX.md`; y las rutas de baseline y de contexto técnico mantenido declaradas en `axiom.yaml` (`baseline.documents` y `technical_context.maintained`). Las rutas de `technical_context.reference` y `odd/tasks/**` NO DEBEN pertenecer nunca a la lista. Toda ruta que no esté en la lista es **código** a efectos de esta capacidad. El indicador opcional `--archive` selecciona la rama de archivado del cambio como rama asignada que `publish` espera (REQ-27.7 y REQ-27.14): la publicación posterior al archivado DEBE invocarse como `axiom sdd publish --change <x> --archive`. Sin `--change`, o con un nombre de cambio inválido, la orden DEBE rechazarse sin efectos.

#### Scenario: La orden existe en `sdd` y no en `branch`

- **DADO** el binario `axiom`
- **CUANDO** se ejecuta `axiom sdd publish --help` y `axiom branch --help`
- **ENTONCES** la ayuda de `sdd` lista `publish`
- **Y** la ayuda de `branch` no lista `publish` ni `push`

#### Scenario: La lista incluye la baseline y el contexto mantenido declarados

- **DADO** un `axiom.yaml` con `baseline.documents` que declara `docs/requisitos.md` y `technical_context.maintained` que declara `docs/arquitectura.md`
- **CUANDO** `publish` clasifica `docs/requisitos.md` y `docs/arquitectura.md`
- **ENTONCES** ambas son rutas de documentación

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

---

### Requirement: `publish` Confirma Solo las Rutas de Documentación del Cambio (REQ-27.6)

`publish` DEBE confirmar (commit) únicamente las rutas de documentación del cambio que tengan modificaciones, mediante una selección explícita de rutas con semántica «solo estas rutas» (`--only`): el commit NO DEBE contener ninguna ruta de código y NO DEBE alterar ni el árbol de trabajo ni el índice de las rutas que no confirma, incluidas las rutas de código ya preparadas por la persona. El commit DEBE incluir `gates.yaml` y `kickoff.yaml` del cambio cuando estén modificados. Si no hay modificaciones en rutas de documentación, `publish` NO DEBE crear un commit vacío y DEBE continuar con la comprobación y el push de los commits ya existentes (REQ-27.8 y REQ-27.7). `publish` NO DEBE usar `--no-verify`: los hooks de git del usuario se ejecutan.

#### Scenario: El commit contiene solo documentación

- **DADO** un cambio con `spec.md` modificado y un fichero de código `internal/x.go` modificado y sin confirmar
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** el commit creado contiene `spec.md` y ningún otro fichero
- **Y** `internal/x.go` sigue modificado en el árbol de trabajo

#### Scenario: El índice de la persona queda intacto

- **DADO** un fichero de código preparado en el índice (`git add`) y un `design.md` modificado
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** el commit contiene solo `design.md`
- **Y** el fichero de código sigue preparado en el índice, sin confirmar

#### Scenario: `gates.yaml` se confirma junto a los artefactos

- **DADO** un `gates.yaml` recién actualizado por `gate record` y sin confirmar
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** el commit incluye `gates.yaml`

#### Scenario: Sin cambios de documentación no hay commit vacío

- **DADO** un cambio sin modificaciones en rutas de documentación y con un commit de documentación aún sin subir
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** no se crea ningún commit nuevo
- **Y** se sube el commit existente según REQ-27.7

#### Scenario: Los hooks no se omiten

- **DADO** un repositorio con un hook de git que se ejecuta al confirmar
- **CUANDO** se ejecuta `publish`
- **ENTONCES** el hook se ejecuta y la orden no usa `--no-verify`

---

### Requirement: Política de Rama y Remoto: Solo la Rama Asignada, a `origin`, sin Forzar (REQ-27.7)

`publish` DEBE subir únicamente la rama asignada al cambio, con una referencia explícita `<rama>:<rama>` hacia el remoto `origin`, sin `--force`, sin `--force-with-lease` y sin `--no-verify`, y NO DEBE ejecutar `fetch`, `pull` ni `ls-remote`. `publish` DEBE negarse, sin confirmar ni subir, en cualquiera de estos casos: HEAD desacoplado; la rama actual distinta de la rama asignada que corresponde a la operación (`wrong_branch`, el mismo código y la misma guarda que REQ-25.19 de `change-branch-lifecycle`, que nombra la rama esperada y la orden correcta, con o sin `--archive`); la rama actual es la rama de integración de un repositorio de código (`monorepo-embedded` o repositorio de código de un `multirepo`); o el almacén de artefactos es `engram` o `none` (REQ-27.10). La única excepción a la negativa por rama de integración es el **repositorio de specs de un `multirepo`**, donde la rama asignada al cambio es su rama de integración (REQ-25.5) aunque esta se llame `main`: ahí `publish` sube esa rama. Tras un push correcto, `publish` DEBE devolver, también en `--json`, la rama, el remoto, el SHA subido y la lista de rutas confirmadas.

#### Scenario: Sube la rama del cambio con referencia explícita

- **DADO** un monorepo con la rama `inc/x` asignada, el remoto `origin` (repositorio *bare*) y documentación modificada
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** `origin` contiene `inc/x` en el commit de la rama local
- **Y** el push se hizo con `inc/x:inc/x` sin `--force`

#### Scenario: Nunca sube `main`

- **DADO** un monorepo con el repositorio desplegado en `main` (la integración)
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** se rechaza con un código tipado de rama de integración no publicable
- **Y** `origin/main` no cambia y no se crea ningún commit

#### Scenario: HEAD desacoplado se rechaza

- **DADO** un repositorio con HEAD desacoplado
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** se rechaza con un código tipado de HEAD desacoplado
- **Y** no se crea ningún commit ni se sube nada

#### Scenario: Una rama equivocada se rechaza con `wrong_branch`

- **DADO** el cambio `x` con rama asignada `inc/x` y el repositorio desplegado en `inc/otro`
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** se rechaza con `wrong_branch` nombrando `inc/x` como rama esperada
- **Y** no se confirma ni se sube nada

#### Scenario: El repositorio de specs de un multirepo sube su integración

- **DADO** un `multirepo` con el repositorio de specs en su rama de integración `main` y documentación del cambio modificada
- **CUANDO** se ejecuta `publish --change x` en ese repositorio
- **ENTONCES** se confirma la documentación y se sube `main:main` a `origin`
- **Y** no se rechaza por ser la rama de integración

#### Scenario: Un repositorio de código de un multirepo rechaza su integración

- **DADO** un `multirepo` y un repositorio de código desplegado en su rama de integración
- **CUANDO** se ejecuta `publish --change x` con ese repositorio como destino
- **ENTONCES** se rechaza con el código tipado de rama de integración no publicable
- **Y** no se sube nada

#### Scenario: Ninguna operación remota distinta del push

- **DADO** un observador instalado en el único punto del producto que ejecuta git
- **CUANDO** se ejecuta `publish --change x` con éxito
- **ENTONCES** el observador registra un único `push` y ningún `fetch`, `pull` ni `ls-remote`
- **Y** el `push` no lleva `--force` ni `--no-verify`

#### Scenario: La salida `--json` informa de lo subido

- **DADO** un push correcto
- **CUANDO** se lee la salida `--json`
- **ENTONCES** contiene en campos separados la rama, el remoto, el SHA subido y las rutas confirmadas

---

### Requirement: Negativa Informativa ante Commits de Código sin Subir (REQ-27.8)

Antes de subir, `publish` DEBE calcular los commits sin subir de la rama contra `refs/remotes/origin/<rama>` local o, si esa referencia no existe, contra la base de integración local (`origin/<integración>` si existe y si no, la rama de integración local), solo con lecturas locales. Si alguno de esos commits modifica una ruta que no pertenece a la lista de rutas de documentación, `publish` DEBE negarse sin subir y sin deshacer el commit de documentación que ya hubiera creado, con el código tipado `code_commits_unpublished`, nombrando los commits y las rutas de código, e incluir la orden de push exacta (`git push origin <rama>:<rama>`) para que la persona decida. La negativa es informativa: NO DEBE considerarse un fallo de la aprobación de una parada ni de ningún otro registro previo.

#### Scenario: Un commit de código sin subir impide el push

- **DADO** la rama `inc/x` con un commit local sin subir que modifica `internal/x.go` y documentación modificada
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** termina con el código `code_commits_unpublished` y código de salida distinto de cero
- **Y** nombra el commit y `internal/x.go` e incluye la orden `git push origin inc/x:inc/x`
- **Y** `origin` no recibe ningún commit

#### Scenario: El commit de documentación creado se conserva

- **DADO** el escenario anterior
- **CUANDO** `publish` se niega
- **ENTONCES** el commit de documentación creado en esta ejecución sigue en la rama local
- **Y** el árbol de trabajo no pierde cambios

#### Scenario: Solo commits de documentación sin subir se suben

- **DADO** la rama `inc/x` con dos commits sin subir que solo tocan rutas de documentación
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** ambos commits se suben en el mismo push

#### Scenario: Sin referencia remota se compara con la base de integración

- **DADO** una rama `inc/x` nueva, sin `refs/remotes/origin/inc/x`, con un commit de código desde la base de integración
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** se niega con `code_commits_unpublished`

#### Scenario: El cálculo no usa la red

- **DADO** un observador instalado en el único punto del producto que ejecuta git
- **CUANDO** `publish` calcula los commits sin subir
- **ENTONCES** el observador no registra `fetch`, `pull` ni `ls-remote` antes del push

---

### Requirement: Negativas Tipadas sin Reintento y sin Estado a Medias (REQ-27.9)

`publish` DEBE distinguir con un código tipado estable en inglés, acompañado de una etiqueta legible en castellano y de la continuación para la persona, al menos estas negativas: push rechazado por no ser *fast-forward*; hook de git que rechaza el commit o el push; identidad de autor de git ausente; HEAD desacoplado; rama de integración no publicable; `wrong_branch`; `code_commits_unpublished`; almacén sin ficheros; y remoto `origin` inexistente (`remote_missing`) o inalcanzable. Ante cualquiera de ellas la orden DEBE terminar con código de salida distinto de cero y NO DEBE reintentar, ejecutar `fetch`, forzar el push ni omitir hooks. La comprobación del remoto `origin` DEBE preceder al commit: con `remote_missing` NO se crea ningún commit. Si el commit local de documentación ya se creó y el push falla (p. ej. *non-fast-forward* o hook de push), el commit DEBE conservarse y la salida DEBE informar de ello y de la orden de push para la persona. Una negativa NO DEBE modificar `gates.yaml`, `kickoff.yaml` ni ningún otro registro del cambio.

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

---

### Requirement: Disponibilidad según el Almacén de Artefactos (REQ-27.10)

`publish` solo es real cuando hay ficheros de documentación que subir: DEBE estar disponible únicamente con el almacén de artefactos `openspec` o `hybrid`. Con el almacén `engram` o `none` DEBE rechazarse con un código tipado que nombre el almacén, sin confirmar ni subir nada. Esta disponibilidad es la misma que la de la salida 2 de las paradas (REQ-26.6 de `sdd-block-review-gates`).

#### Scenario: Almacén engram rechaza la publicación

- **DADO** un cambio con el almacén declarado `engram`
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** se rechaza con el código tipado de almacén sin ficheros nombrando `engram`
- **Y** no se confirma ni se sube nada

#### Scenario: Almacén none rechaza la publicación

- **DADO** un cambio con el almacén declarado `none`
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** se rechaza con el mismo código tipado nombrando `none`

#### Scenario: Almacén hybrid publica

- **DADO** un cambio con el almacén `hybrid` y documentación modificada
- **CUANDO** se ejecuta `publish --change x`
- **ENTONCES** se confirma y se sube según REQ-27.6 y REQ-27.7

---

### Requirement: Aislamiento de la Operación Remota (REQ-27.11)

La capacidad de subir una rama a un remoto DEBE vivir en un puerto de publicación propio, separado del puerto `Git` y del puerto de escritura del motor de ramas, y solo `axiom sdd publish` DEBE poder alcanzarlo. `axiom branch ensure`, `status`, `done` y `guard`, `axiom archive` y `axiom sdd gate record` NO DEBEN alcanzar ese puerto ni ejecutar `git commit` ni `git push`. La ejecución de git del puerto de publicación DEBE pasar por el único punto de ejecución de git del producto. Una prueba de arquitectura o de dependencias DEBE fijar este aislamiento, y el invariante «ningún subcomando de `axiom branch` publica» (REQ-25.1 de `change-branch-lifecycle`) DEBE seguir verificado por el observador del único punto de ejecución.

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
- **ENTONCES** el paquete del puerto de publicación solo es importado por la implementación de `axiom sdd publish` y sus pruebas
- **Y** la prueba falla si `internal/branch/**` o el código de `gate record` lo importan

---

### Requirement: Publicación Automática en el Estilo `continuous` Solo con Autorización Sellada (REQ-27.13)

En un cambio sellado con `execution_style: continuous`, el orquestador DEBE publicar los documentos con `axiom sdd publish`, sin preguntar, exactamente en dos momentos: una vez al cerrar la fase `tasks` y una vez al archivar (REQ-27.14), y solo si `kickoff.yaml` registra la autorización de publicación del cambio (`docs_publication: auto`, REQ-27.4 de `sdd-lane-kickoff`). Un kickoff sin ese campo (incluidos los sellados antes de este cambio) NO DEBE publicar nada de forma automática: el orquestador DEBE ofrecer `axiom sdd publish --change <x>` como siguiente paso y esperar la decisión de la persona. En el estilo `continuous` NO DEBE existir ninguna parada adicional por la publicación (REQ-21.7 de `sdd-block-review-gates`). Una negativa de `publish` en este flujo es informativa: el orquestador la presenta y no interrumpe el encadenado de fases.

#### Scenario: Con autorización sellada se publica al cerrar `tasks`

- **DADO** un cambio `continuous` con `docs_publication: auto` en su `kickoff.yaml` y la fase `tasks` recién completada
- **CUANDO** el orquestador cierra la fase
- **ENTONCES** ejecuta `axiom sdd publish --change <x>` sin preguntar
- **Y** continúa con la fase siguiente sin presentar ninguna parada

#### Scenario: Con autorización sellada se publica al archivar

- **DADO** un cambio `continuous` con `docs_publication: auto` y el archivado completado
- **CUANDO** el orquestador cierra el archivado
- **ENTONCES** ejecuta `axiom sdd publish --change <x> --archive` sobre la rama de archivado (REQ-27.14)

#### Scenario: Sin el campo no se publica de forma automática

- **DADO** un cambio `continuous` cuyo `kickoff.yaml` no contiene `docs_publication`
- **CUANDO** el orquestador cierra la fase `tasks`
- **ENTONCES** no ejecuta `publish`
- **Y** ofrece `axiom sdd publish --change <x>` como siguiente paso

#### Scenario: No se publica entre otras fases

- **DADO** un cambio `continuous` con `docs_publication: auto`
- **CUANDO** concluyen las fases `spec`, `design` o `apply`
- **ENTONCES** el orquestador no ejecuta `publish`

#### Scenario: Una negativa no interrumpe el encadenado

- **DADO** un cambio `continuous` con `docs_publication: auto` y un commit de código sin subir
- **CUANDO** `publish` se niega con `code_commits_unpublished`
- **ENTONCES** el orquestador presenta la negativa con la orden de push
- **Y** encadena la fase siguiente

#### Scenario: Almacén sin ficheros no publica

- **DADO** un cambio `continuous` con `docs_publication: auto` y almacén `engram`
- **CUANDO** el orquestador cierra la fase `tasks`
- **ENTONCES** no ejecuta `publish`
