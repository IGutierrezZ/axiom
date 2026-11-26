# Especificación de Publicación Ligera de Documentación SDD (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** capacidad nueva (especificación completa, no delta). `sdd-archive` la promoverá a `openspec/specs/sdd-docs-publication/spec.md`.
> **Idioma:** castellano de España; identificadores, códigos y tokens en inglés.

## Propósito

Definir el verbo nativo `axiom sdd publish`, que confirma y sube, y solo eso, los documentos de un cambio SDD a la rama que el planificador le asigna, de modo que aprobar una parada, cerrar `tasks` en un kickoff autorizado o archivar deje los documentos disponibles para otra persona sin una PR fusionada. Fija la lista cerrada de rutas de documentación, la política de rama y remoto, las negativas tipadas y el aislamiento de la operación remota respecto de `axiom branch` y de `axiom sdd gate record`.

## Términos

- **Rama asignada:** la rama que `axiom branch ensure` y el planificador de ramas asignan al cambio en el repositorio que corresponda (RF-BRA-04 de `change-branch-lifecycle`): rama del cambio, rama de rol o rama de archivado. En el repositorio de specs de un `multirepo`, su rama de integración (RF-PUB-03).
- **Rutas de documentación** (lista cerrada, RF-PUB-01): las únicas rutas que `publish` puede confirmar.
- **Commit sin subir:** commit de la rama actual que no está contenido en `refs/remotes/origin/<rama>` local o, si esa referencia no existe, en la base de integración local.
- **Negativa informativa:** terminación de `publish` con código de salida distinto de cero y un código tipado, sin efectos remotos y sin anular ninguna decisión ya registrada.

## Requirements

### Requirement: Verbo `axiom sdd publish --change <x>` y Lista Cerrada de Rutas de Documentación (RF-PUB-01)

La CLI DEBE exponer `axiom sdd publish --change <nombre>` con `--cwd` y `--json`, como subcomando de `axiom sdd` y no de `axiom branch`. La orden DEBE operar únicamente sobre las **rutas de documentación**, una lista cerrada formada por: `openspec/changes/<x>/**`; la carpeta de `openspec/changes/archive/**` que corresponde al cambio; `openspec/specs/**`; `openspec/INDEX.md`; y las rutas de contexto técnico mantenido declaradas en `axiom.yaml` (`technical_context.maintained`). Las rutas de `technical_context.reference` y `odd/tasks/**` NO DEBEN pertenecer nunca a la lista. Tampoco pertenece a la lista el **estado local** del cambio: los ficheros cuyo nombre empieza por punto situados directamente en la raíz de `openspec/changes/<x>/` o en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<x>/`). `publish` NUNCA DEBE confirmarlos, ni con `--archive` ni en publicaciones posteriores. Los ficheros con punto de subcarpetas siguen siendo documentación. Toda ruta que no esté en la lista es **código** a efectos de esta capacidad. El indicador opcional `--archive` selecciona la rama de archivado del cambio como rama asignada que `publish` espera (RF-PUB-03 y RF-ARC-08): la publicación posterior al archivado DEBE invocarse como `axiom sdd publish --change <x> --archive`. Sin `--change`, o con un nombre de cambio inválido, la orden DEBE rechazarse sin efectos.
(Previously: la lista incluía todo `openspec/changes/<x>/**` y la carpeta archivada del cambio sin excluir explícitamente el estado local con punto de sus raíces. La exclusión de la carpeta ordinaria solo existía en la implementación y la carpeta archivada no tenía ninguna.)

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

### Requirement: `publish` Confirma Solo las Rutas de Documentación del Cambio (RF-PUB-02)

`publish` DEBE confirmar (commit) únicamente las rutas de documentación del cambio que tengan modificaciones, mediante una selección explícita de rutas con semántica «solo estas rutas» (`--only`): el commit NO DEBE contener ninguna ruta de código y NO DEBE alterar ni el árbol de trabajo ni el índice de las rutas que no confirma, incluidas las rutas de código ya preparadas por la persona. El commit DEBE incluir `gates.yaml` y `kickoff.yaml` del cambio cuando estén modificados. Si no hay modificaciones en rutas de documentación, `publish` NO DEBE crear un commit vacío y DEBE continuar con la comprobación y el push de los commits ya existentes (RF-PUB-04 y RF-PUB-03). `publish` NO DEBE usar `--no-verify`: los hooks de git del usuario se ejecutan.

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
- **Y** se sube el commit existente según RF-PUB-03

#### Scenario: Los hooks no se omiten

- **DADO** un repositorio con un hook de git que se ejecuta al confirmar
- **CUANDO** se ejecuta `publish`
- **ENTONCES** el hook se ejecuta y la orden no usa `--no-verify`

---

### Requirement: Política de Rama y Remoto: Solo la Rama Asignada, a `origin`, sin Forzar (RF-PUB-03)

`publish` DEBE subir únicamente la rama asignada al cambio, con una referencia explícita `<rama>:<rama>` hacia el remoto `origin`, sin `--force`, sin `--force-with-lease` y sin `--no-verify`, y NO DEBE ejecutar `fetch`, `pull` ni `ls-remote`. `publish` DEBE negarse, sin confirmar ni subir, en cualquiera de estos casos: HEAD desacoplado; la rama actual distinta de la rama asignada que corresponde a la operación (`wrong_branch`, el mismo código y la misma guarda que RF-BRA-20 de `change-branch-lifecycle`, que nombra la rama esperada y la orden correcta, con o sin `--archive`); la rama actual es la rama de integración de un repositorio de código (`monorepo-embedded` o repositorio de código de un `multirepo`); o el almacén de artefactos es `engram` o `none` (RF-PUB-06). La única excepción a la negativa por rama de integración es el **repositorio de specs de un `multirepo`**, donde la rama asignada al cambio es su rama de integración (RF-BRA-05) aunque esta se llame `main`: ahí `publish` sube esa rama. Tras un push correcto, `publish` DEBE devolver, también en `--json`, la rama, el remoto, el SHA subido y la lista de rutas confirmadas.

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

### Requirement: Negativa Informativa ante Commits de Código sin Subir (RF-PUB-04)

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

### Requirement: Negativas Tipadas sin Reintento y sin Estado a Medias (RF-PUB-05)

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

### Requirement: Disponibilidad según el Almacén de Artefactos (RF-PUB-06)

`publish` solo es real cuando hay ficheros de documentación que subir: DEBE estar disponible únicamente con el almacén de artefactos `openspec` o `hybrid`. Con el almacén `engram` o `none` DEBE rechazarse con un código tipado que nombre el almacén, sin confirmar ni subir nada. Esta disponibilidad es la misma que la de la salida 2 de las paradas (RF-GAT-12 de `sdd-block-review-gates`).

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
- **ENTONCES** se confirma y se sube según RF-PUB-02 y RF-PUB-03

---

### Requirement: Aislamiento de la Operación Remota (RF-PUB-07)

La capacidad de subir una rama a un remoto DEBE vivir en un puerto de publicación propio, separado del puerto `Git` y del puerto de escritura del motor de ramas, y solo `axiom sdd publish` DEBE poder alcanzarlo. `axiom branch ensure`, `status`, `done` y `guard`, `axiom archive` y `axiom sdd gate record` NO DEBEN alcanzar ese puerto ni ejecutar `git commit` ni `git push`. La ejecución de git del puerto de publicación DEBE pasar por el único punto de ejecución de git del producto. Una prueba de arquitectura o de dependencias DEBE fijar este aislamiento, y el invariante «ningún subcomando de `axiom branch` publica» (RF-BRA-01 de `change-branch-lifecycle`) DEBE seguir verificado por el observador del único punto de ejecución.

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

### Requirement: Publicación Automática en el Estilo `continuous` Solo con Autorización Sellada (RF-PUB-08)

En un cambio sellado con `execution_style: continuous`, el orquestador DEBE publicar los documentos con `axiom sdd publish`, sin preguntar, exactamente en cuatro momentos: una vez al terminar la fase `spec`, una vez al terminar la fase `design`, una vez al cerrar la fase `tasks` y una vez al archivar (RF-ARC-08), y solo si `kickoff.yaml` registra la autorización de publicación del cambio (`docs_publication: auto`, RF-KOF-10 de `sdd-lane-kickoff`). Un kickoff sellado con `docs_publication: auto` antes de que existieran los momentos de `spec` y `design` los hereda sin volver a preguntar ni a sellar: la autorización sigue siendo publicar solo documentos. Un kickoff sin ese campo (incluidos los sellados antes de este cambio) NO DEBE publicar nada de forma automática: el orquestador DEBE ofrecer `axiom sdd publish --change <x>` como siguiente paso y esperar la decisión de la persona. En el estilo `continuous` NO DEBE existir ninguna parada adicional por la publicación (RF-GAT-01 de `sdd-block-review-gates`). Una negativa de `publish` en este flujo es informativa: el orquestador la presenta y no interrumpe el encadenado de fases.
(Previously: con la autorización sellada se publicaba solo en dos momentos, al cerrar `tasks` y al archivar)

#### Scenario: Con autorización sellada se publica al cerrar `tasks`

- **DADO** un cambio `continuous` con `docs_publication: auto` en su `kickoff.yaml` y la fase `tasks` recién completada
- **CUANDO** el orquestador cierra la fase
- **ENTONCES** ejecuta `axiom sdd publish --change <x>` sin preguntar
- **Y** continúa con la fase siguiente sin presentar ninguna parada

#### Scenario: Con autorización sellada se publica al archivar

- **DADO** un cambio `continuous` con `docs_publication: auto` y el archivado completado
- **CUANDO** el orquestador cierra el archivado
- **ENTONCES** ejecuta `axiom sdd publish --change <x> --archive` sobre la rama de archivado (RF-ARC-08)

#### Scenario: Sin el campo no se publica de forma automática

- **DADO** un cambio `continuous` cuyo `kickoff.yaml` no contiene `docs_publication`
- **CUANDO** el orquestador cierra la fase `tasks`
- **ENTONCES** no ejecuta `publish`
- **Y** ofrece `axiom sdd publish --change <x>` como siguiente paso

#### Scenario: Con autorización sellada se publica al terminar `spec` y al terminar `design`

- **DADO** un cambio `continuous` con `docs_publication: auto` y la fase `spec` o la fase `design` recién completada
- **CUANDO** el orquestador cierra la fase
- **ENTONCES** ejecuta `axiom sdd publish --change <x>` sin preguntar
- **Y** continúa con la fase siguiente sin presentar ninguna parada

#### Scenario: Un kickoff sellado antes hereda los momentos nuevos

- **DADO** un cambio `continuous` sellado con `docs_publication: auto` antes de que se publicara tras `spec` y `design`
- **CUANDO** el orquestador cierra la fase `spec`
- **ENTONCES** ejecuta `axiom sdd publish --change <x>` sin preguntar
- **Y** no vuelve a preguntar ni a sellar el kickoff

#### Scenario: No se publica tras `apply`

- **DADO** un cambio `continuous` con `docs_publication: auto`
- **CUANDO** concluye la fase `apply`
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

### Requirement: `sdd publish --archive` Confirmado Emite `archive` (RF-PUB-09)

`axiom sdd publish --change <x> --archive` DEBE despachar el evento `archive` (RF-INT-06) cuando el resultado de la publicación NO es un rechazo, es decir, cuando el archivado queda confirmado. La operación DEBE encolarse de forma duradera en `.integrations-outbox.yaml` en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<x>/`, RF-INT-19), con la clave de idempotencia `archive/<cambio>/<carpeta archivada>` (RF-INT-07), e intentar la entrega inmediata acotada. Un resultado rechazado NO DEBE emitir. Sin `--archive`, NO DEBE emitir. La emisión NUNCA DEBE alterar el resultado, el código de salida ni la salida de `publish` (RF-INT-10), y NO DEBE impedir ni retrasar la publicación ya confirmada.

(Placement: after RF-PUB-08)

#### Scenario: Archivado confirmado
- **DADO** el doble configurado y una publicación de archivado confirmada
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** el doble recibe `archive` con la clave `archive/x/<fecha>-x`
- **Y** el resultado y el código de salida de `publish` son los de siempre

#### Scenario: Publicación rechazada
- **DADO** una publicación que termina con una negativa tipada (RF-PUB-05)
- **CUANDO** se ejecuta `publish --archive`
- **ENTONCES** el doble no recibe ningún evento y no se crea ninguna cola

#### Scenario: Sin `--archive`
- **DADO** el doble configurado
- **CUANDO** se ejecuta `publish --change x` sin `--archive`
- **ENTONCES** el doble no recibe ningún evento

#### Scenario: Fallo de la integración deja la cola en la carpeta archivada
- **DADO** el doble configurado para fallar
- **CUANDO** se ejecuta `publish --archive` con archivado confirmado
- **ENTONCES** `publish` termina con su código habitual y stderr avisa de la operación pendiente y de la orden `axiom integrations sync`
- **Y** la operación consta en `openspec/changes/archive/<fecha>-x/.integrations-outbox.yaml`

#### Scenario: Una publicación posterior no confirma la cola
- **DADO** una cola pendiente en la carpeta archivada
- **CUANDO** se repite `publish --change x --archive`
- **ENTONCES** el commit, si lo hay, no incluye `.integrations-outbox.yaml`

### Requirement: El Evento `archive` no Reescribe `integrations.yaml` (RF-PUB-10)

El despacho de `archive` NO DEBE modificar `integrations.yaml` de la carpeta archivada ni ningún otro fichero versionado: `integrations.yaml` archivado solo se lee, para dar los vínculos al plugin. Los vínculos o imputaciones que devuelva el plugin al entregar `archive` se descartan con un aviso por stderr y NO se persisten en este cambio. La única escritura permitida es la cola local de la carpeta archivada (RF-ARC-03 modificado), de modo que la carpeta archivada ya publicada no quede sucia en ficheros versionados. La regla se impone por mecanismo, no solo por política: la guarda de escritura bajo `archive/` rechaza cualquier nombre que no sea la cola y sus temporales.

(Placement: after RF-PUB-09)

#### Scenario: El plugin devuelve vínculos en `archive`
- **DADO** un doble que devuelve un vínculo al recibir `archive`
- **CUANDO** se confirma `publish --archive`
- **ENTONCES** `integrations.yaml` de la carpeta archivada no cambia y `git status` no muestra cambios en ficheros versionados
- **Y** stderr avisa de que los vínculos devueltos se han descartado

#### Scenario: Intento de escribir `integrations.yaml` archivado
- **DADO** una carpeta archivada
- **CUANDO** el marco de integraciones intenta escribir `integrations.yaml` en ella
- **ENTONCES** la escritura se rechaza con el mensaje de RF-ARC-03 y no se modifica ningún fichero

#### Scenario: Entrega fallida de `archive`
- **DADO** el doble configurado para fallar
- **CUANDO** se confirma `publish --archive`
- **ENTONCES** la operación queda pendiente solo en `.integrations-outbox.yaml` de la carpeta archivada, que no se versiona
### Requirement: El Status Nativo Proyecta la Publicación Pendiente (RF-PUB-11)

`axiom sdd status --json` DEBE proyectar, para el cambio seleccionado, un campo opcional `publication` en la **raíz** del status (junto a `gateQuestion`, no dentro de `governance`), con `omitempty`: cuando no hay estado que proyectar, el JSON del status NO DEBE cambiar ni un byte y la clave `publication` NO DEBE aparecer. El cálculo DEBE ser de solo lectura y sin red: NO DEBE ejecutar `fetch`, `pull`, `ls-remote`, `commit`, `add` ni `push`, y solo DEBE usar referencias locales (`refs/remotes/origin/<rama>`, `refs/remotes/origin/<integración>` y, como último recurso, la rama de integración local). La proyección NO DEBE alterar `nextRecommended`, las dependencias ni los motivos de bloqueo del status.

**Condición (a), según el caso.** No se lee nada de git si no se cumple (a):

- **Cambio activo `checkpointed`:** existe al menos una compuerta aprobada cuya clave no es `integration`.
- **Cambio activo `continuous`:** el kickoff registra `docs_publication: auto`; al menos uno de los artefactos `spec`, `design` o `tasks` está completo; no hay tareas de `apply` completadas; y el informe de `verify` no está hecho. Las dos últimas condiciones excluyen la ventana de `apply` y `verify` (RF-PUB-08: no se publica tras `apply`).
- **Cambio archivado:** solo si es `continuous` con `docs_publication: auto`; entonces la publicación se calcula sobre la rama de archivado y su `invocation` DEBE incluir `--archive` (RF-ARC-08). Un cambio `checkpointed` archivado NO DEBE proyectar nada (el archivado se ofrece a la persona).

**Condición (b).** Con (a) cumplida, `publication` se proyecta solo si hay algo que `publish` haría: documentación sin confirmar (rutas de la lista cerrada de RF-PUB-01, con el estado local con punto excluido) o al menos un commit sin subir (mismo rango que RF-PUB-04). La ausencia de `refs/remotes/origin/<rama>` (primera publicación) es solo un **motivo** (`remote_branch_missing`) y NO basta por sí sola: sin documentación sin confirmar y sin commits sin subir, `publish` respondería `up_to_date` y NO DEBE proyectarse nada.

**Estados.** El campo lleva `state` con uno de tres valores, `reasons` (cuando procede), `remote`, `branch`, `prTarget` (informativo, vacío en la integración in situ) y `archive` (en el momento de archivado). Se aplica la primera regla que se cumpla:

| Hechos | `state` | `reasons` | Campos propios |
|--------|---------|-----------|----------------|
| Negativa del cálculo del destino de publicación o fallo de lectura de git | `unavailable` | — | código (`git_failed` en una lectura de git, o el código de la negativa) |
| Ni `origin/<integración>` ni la integración local existen | `unavailable` | — | `unpublished_range_unknown` |
| Documentación sin confirmar | `pending` | `docs_uncommitted` y, si procede, `branch_ahead`, `remote_branch_missing` y `code_commits` | `invocation` |
| Más de 500 commits sin subir | `code_unpublished` | `range_too_big` | `pushCommand` |
| Algún commit sin subir toca rutas que no son documentación (código) | `code_unpublished` | `code_commits` y, si procede, `remote_branch_missing` | `pushCommand` y `codeCommits` (SHA cortos) |
| Solo commits de documentación sin subir | `pending` | `branch_ahead` y, si procede, `remote_branch_missing` | `invocation` |
| Nada que confirmar ni subir | (campo ausente) | | |

- `pending`: la `invocation` DEBE ser idéntica, byte a byte, a `gateQuestion.publication.invocation` (RF-GAT-15) cuando ambas aparecen; en el momento de archivado es esa invocación más `--archive`.
- `code_unpublished` (estado informativo, decisión de producto D1): `publish` se negaría siempre con `code_commits_unpublished` (RF-PUB-04). Lleva `pushCommand` (`git -C <raíz> push origin <rama>:<rama>`, con las rutas citadas) solo para la persona y NO lleva `invocation`. Nadie DEBE ejecutarlo salvo la persona.
- `unavailable`: informativo; lleva el código y nunca hace fallar el status.
- Con documentación sin confirmar y commits de código a la vez, el estado es `pending` (con el motivo `code_commits`) para que `publish` confirme en local; el status siguiente proyecta `code_unpublished`.

**Exclusiones.** NO DEBE proyectarse `publication` cuando: no hay cambio seleccionado; el almacén de artefactos es `engram` o `none`; no se cumple (a) (sin aprobaciones, solo la compuerta `integration`, `continuous` sin `docs_publication: auto` o en la ventana de `apply`/`verify`); o el repositorio no tiene el remoto `origin` configurado (no hay publicación posible). La configuración de `origin` se lee con una consulta de solo lectura de la configuración; no se deduce de la ausencia de referencias remotas, porque un remoto añadido y aún no traído sí permite publicar.

**Multirepo.** En el repositorio de specs de un `multirepo`, la rama de publicación es su rama de integración in situ (RF-PUB-03) y el adelanto se mide contra `origin/<integración>`.

**Una sola lista de rutas.** La clasificación de una ruta como documentación o código DEBE usar la misma lista cerrada de RF-PUB-01 que `publish`, compuesta en un único punto y sin duplicarla. `kickoff.yaml` y `gates.yaml` son documentación; el estado local con punto de la raíz de la carpeta del cambio no lo es. Las carpetas sin seguimiento se evalúan fichero a fichero, no como una sola entrada de directorio. El comportamiento de `axiom sdd publish` NO DEBE cambiar.

#### Scenario: Compuerta aprobada con documentación sin confirmar

- **DADO** un cambio `checkpointed` con la compuerta `spec` aprobada y ficheros de `openspec/changes/<x>/` sin confirmar
- **CUANDO** se ejecuta `axiom sdd status --change <x> --json`
- **ENTONCES** `publication.state` es `pending` con el motivo `docs_uncommitted`
- **Y** su `invocation` es idéntica a `gateQuestion.publication.invocation`
- **Y** la clave está en la raíz del status, no dentro de `governance`
- **Y** `nextRecommended` y las dependencias no cambian

#### Scenario: Varios motivos a la vez

- **DADO** una compuerta aprobada, documentación sin confirmar, commits sin subir en la rama y sin `refs/remotes/origin/<rama>`
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending`
- **Y** `reasons` contiene `docs_uncommitted`, `branch_ahead` y `remote_branch_missing`

#### Scenario: Tras un `publish` correcto deja de proyectarse

- **DADO** el primer escenario y un `axiom sdd publish --change <x>` correcto
- **CUANDO** se vuelve a consultar el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Rama por delante de origin solo con commits de documentación

- **DADO** una compuerta aprobada, ningún fichero de documentación sin confirmar y commits sin subir que solo tocan rutas de documentación
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con el motivo `branch_ahead`
- **Y** no se proyecta `code_unpublished`

#### Scenario: Primera publicación con algo que subir

- **DADO** una compuerta aprobada, `origin` configurado, ninguna `refs/remotes/origin/<rama>` y al menos un commit de documentación sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con los motivos `branch_ahead` y `remote_branch_missing`
- **Y** el status no falla

#### Scenario: Sin referencia remota y sin nada que subir no se proyecta

- **DADO** una compuerta aprobada, ninguna `refs/remotes/origin/<rama>`, sin documentación sin confirmar y sin commits sin subir
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication` (`publish` respondería `up_to_date`)

#### Scenario: Repositorio sin remoto `origin` no proyecta

- **DADO** una compuerta aprobada, documentación sin confirmar y ningún remoto `origin` configurado
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Commits de código sin subir proyectan un estado informativo aparte

- **DADO** una compuerta `role-apply:<rol>` aprobada, toda la documentación confirmada y commits que tocan código sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `code_unpublished` con el motivo `code_commits`
- **Y** lleva `pushCommand` (`git -C <raíz> push origin <rama>:<rama>`) y `codeCommits`, y no lleva `invocation`

#### Scenario: Documentación sin confirmar prevalece sobre el código sin subir

- **DADO** una compuerta aprobada, documentación sin confirmar y commits de código sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con los motivos `docs_uncommitted` y `code_commits` y con `invocation`
- **Y** tras el `publish` que confirma la documentación, el status proyecta `code_unpublished`

#### Scenario: Más de 500 commits sin subir

- **DADO** una compuerta aprobada, la documentación confirmada y más de 500 commits sin subir
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `code_unpublished` con el motivo `range_too_big` y `pushCommand`

#### Scenario: Una negativa del destino o un fallo de git es `unavailable`

- **DADO** una compuerta aprobada y un cálculo del destino de publicación que se niega, o una lectura de git que falla
- **CUANDO** se consulta el status
- **ENTONCES** el status termina correctamente con `publication.state` `unavailable` y el código (el de la negativa o `git_failed`)

#### Scenario: Rango sin subir desconocido es `unavailable`

- **DADO** una compuerta aprobada, `origin` configurado, y ni `origin/<integración>` ni la integración local existen
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `unavailable` con el código `unpublished_range_unknown`

#### Scenario: Almacén `engram` o `none` no proyecta

- **DADO** un cambio con aprobaciones registradas y almacén `engram` o `none`
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Sin aprobaciones ni momentos cumplidos no proyecta

- **DADO** un cambio `checkpointed` sin ninguna compuerta aprobada, o un cambio `continuous` sin `docs_publication: auto`, con documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: La compuerta `integration` no cuenta

- **DADO** un cambio `checkpointed` cuya única aprobación es la de `integration` y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Un cambio `continuous` con `auto` proyecta al cumplirse un momento

- **DADO** un cambio `continuous` con `docs_publication: auto`, el artefacto `spec` completo, sin tareas completadas ni informe de `verify`, y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con la `invocation` de `publish`

#### Scenario: En `continuous`, durante `apply` o `verify` no se proyecta

- **DADO** un cambio `continuous` con `docs_publication: auto`, con `tasks` cerrada, al menos una tarea de `apply` completada o el informe de `verify` hecho, y documentación sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication` (RF-PUB-08)

#### Scenario: Archivado `continuous` con `auto` lleva `--archive`

- **DADO** un cambio archivado `continuous` con `docs_publication: auto` y documentación archivada sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` y `archive` es verdadero
- **Y** su `invocation` es la de publicación más `--archive`

#### Scenario: Archivado `checkpointed` no proyecta

- **DADO** un cambio archivado `checkpointed` con documentación archivada sin confirmar
- **CUANDO** se consulta el status
- **ENTONCES** no hay clave `publication`

#### Scenario: Multirepo mide contra la integración del repositorio de specs

- **DADO** un `multirepo` con una compuerta aprobada y el repositorio de specs en su rama de integración con commits de documentación sin subir respecto de `origin/<integración>`
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` y su rama es la de integración del repositorio de specs

#### Scenario: Una carpeta sin seguimiento con solo estado local no cuenta

- **DADO** una compuerta aprobada y la carpeta `openspec/changes/<x>/` sin seguimiento que solo contiene `.gates.lock`
- **CUANDO** se consulta el status
- **ENTONCES** no hay documentación sin confirmar y, sin commits sin subir, no hay clave `publication`

#### Scenario: Una carpeta sin seguimiento con `kickoff.yaml` es documentación

- **DADO** una compuerta aprobada y la carpeta `openspec/changes/<x>/` sin seguimiento que contiene `kickoff.yaml` además de `.gates.lock`
- **CUANDO** se consulta el status
- **ENTONCES** `publication.state` es `pending` con el motivo `docs_uncommitted`
- **Y** `kickoff.yaml` y `gates.yaml` cuentan como documentación mientras el fichero con punto no

#### Scenario: Paridad con `publish`

- **DADO** un cambio con documentación sin confirmar
- **CUANDO** se comparan las rutas que el status considera documentación sin confirmar con las que `axiom sdd publish` confirma
- **ENTONCES** son las mismas, porque ambos usan la misma lista cerrada de RF-PUB-01
- **Y** tras un `publish` correcto el status no proyecta nada
- **Y** el comportamiento, las negativas y los resultados de `publish` son los que ya fijan RF-PUB-01 a RF-ARC-08

#### Scenario: El cálculo no toca la red ni escribe

- **DADO** el observador del punto único de git de la prueba
- **CUANDO** se calcula `publication` en cualquiera de los escenarios anteriores
- **ENTONCES** el observador no registra `fetch`, `pull`, `ls-remote`, `commit`, `add` ni `push`
- **Y** el estado del repositorio no cambia
- **Y** la prueba de aislamiento de `sddpublish` sigue en verde

#### Scenario: `gate record` sigue sin ejecutar git

- **DADO** una aprobación registrada con `axiom sdd gate record`
- **CUANDO** el registro termina
- **ENTONCES** devuelve la orden de publicación sin ejecutarla (RF-GAT-11)
- **Y** la publicación pendiente solo se proyecta después en el status

---

### Requirement: El Orquestador Resuelve la Publicación Pendiente Antes de la Fase Siguiente (RF-PUB-12)

Cuando el status proyecta `publication` con `state: pending`, la doctrina del orquestador DEBE ordenar ejecutar su `invocation` exactamente una vez antes de lanzar la fase siguiente y exactamente una vez al reanudar el cambio; no se repite dentro de la misma transición ni de la misma reanudación. Su negativa es informativa: el orquestador la presenta, no anula ninguna aprobación registrada y no interrumpe el encadenado de fases ni la parada de la compuerta (RF-GAT-15, RF-PUB-08). Con `state: code_unpublished`, el orquestador DEBE presentar su `pushCommand` a la persona y NO DEBE ejecutarlo ni ejecutar `publish` por ese motivo. Con `state: unavailable`, DEBE presentar su código como información. El campo NUNCA DEBE cambiar `nextRecommended`. La doctrina NO DEBE hacer que `gate record` ejecute git (RF-GAT-11, RF-PUB-07). Esta regla completa el párrafo «When to publish» del orquestador sin cambiar RF-GAT-15 ni RF-PUB-08.

#### Scenario: Se ejecuta la invocación una vez antes de la fase siguiente

- **DADO** un status con `publication.state: pending` tras aprobar una compuerta
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** ejecuta la `invocation` proyectada exactamente una vez antes de lanzarla

#### Scenario: Se ejecuta una vez al reanudar

- **DADO** una sesión reanudada cuyo status proyecta `publication.state: pending`
- **CUANDO** el orquestador retoma el cambio
- **ENTONCES** ejecuta la `invocation` exactamente una vez antes de continuar

#### Scenario: La negativa es informativa

- **DADO** un status con `publication.state: pending` y un `publish` que se niega con un código tipado
- **CUANDO** el orquestador recibe la negativa
- **ENTONCES** la presenta como información sin anular la aprobación registrada
- **Y** continúa la cadena o mantiene la parada según el estilo sellado, sin repetir `publish` en esa transición

#### Scenario: `code_unpublished` presenta el `pushCommand` sin ejecutarlo

- **DADO** un status con `publication.state: code_unpublished`
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** presenta el `pushCommand` a la persona
- **Y** no ejecuta `git push` ni `axiom sdd publish`
- **Y** continúa con la fase siguiente

#### Scenario: `unavailable` se presenta como información

- **DADO** un status con `publication.state: unavailable` y un código
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** presenta el código como información y continúa

#### Scenario: El campo nunca cambia el enrutado

- **DADO** un status con `publication` en cualquiera de sus tres estados
- **CUANDO** el orquestador consulta `nextRecommended`
- **ENTONCES** es el mismo que sin el campo

#### Scenario: Sin proyección no se publica por esta regla

- **DADO** un status sin clave `publication`
- **CUANDO** el orquestador va a lanzar la fase siguiente
- **ENTONCES** no ejecuta `publish` por esta regla
