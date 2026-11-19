# Delta para sdd-docs-publication

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta (C-07-b, capacidad `integration-plugins`, decisión B-1 opción B) hace dos cosas: (1) excluye de la lista de rutas de documentación el estado local con punto de la raíz de la carpeta del cambio y de la carpeta archivada del cambio (MODIFIED REQ-27.5), y (2) añade la emisión duradera del evento `archive` al publicar el archivado (ADDED REQ-43.26 y REQ-43.27). El resultado, el código de salida y el contrato JSON de `axiom sdd publish` NO cambian.

## MODIFIED Requirements

### Requirement: Verbo `axiom sdd publish --change <x>` y Lista Cerrada de Rutas de Documentación (REQ-27.5)

La CLI DEBE exponer `axiom sdd publish --change <nombre>` con `--cwd` y `--json`, como subcomando de `axiom sdd` y no de `axiom branch`. La orden DEBE operar únicamente sobre las **rutas de documentación**, una lista cerrada formada por: `openspec/changes/<x>/**`; la carpeta de `openspec/changes/archive/**` que corresponde al cambio; `openspec/specs/**`; `openspec/INDEX.md`; y las rutas de contexto técnico mantenido declaradas en `axiom.yaml` (`technical_context.maintained`). Las rutas de `technical_context.reference` y `odd/tasks/**` NO DEBEN pertenecer nunca a la lista. Tampoco pertenece a la lista el **estado local** del cambio: los ficheros cuyo nombre empieza por punto situados directamente en la raíz de `openspec/changes/<x>/` o en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<x>/`). `publish` NUNCA DEBE confirmarlos, ni con `--archive` ni en publicaciones posteriores. Los ficheros con punto de subcarpetas siguen siendo documentación. Toda ruta que no esté en la lista es **código** a efectos de esta capacidad. El indicador opcional `--archive` selecciona la rama de archivado del cambio como rama asignada que `publish` espera (REQ-27.7 y REQ-27.14): la publicación posterior al archivado DEBE invocarse como `axiom sdd publish --change <x> --archive`. Sin `--change`, o con un nombre de cambio inválido, la orden DEBE rechazarse sin efectos.
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

## ADDED Requirements

### Requirement: `sdd publish --archive` Confirmado Emite `archive` (REQ-43.26)

`axiom sdd publish --change <x> --archive` DEBE despachar el evento `archive` (REQ-43.6) cuando el resultado de la publicación NO es un rechazo, es decir, cuando el archivado queda confirmado. La operación DEBE encolarse de forma duradera en `.integrations-outbox.yaml` en la raíz de la carpeta archivada del cambio (`openspec/changes/archive/<fecha>-<x>/`, REQ-43.28), con la clave de idempotencia `archive/<cambio>/<carpeta archivada>` (REQ-43.7), e intentar la entrega inmediata acotada. Un resultado rechazado NO DEBE emitir. Sin `--archive`, NO DEBE emitir. La emisión NUNCA DEBE alterar el resultado, el código de salida ni la salida de `publish` (REQ-43.10), y NO DEBE impedir ni retrasar la publicación ya confirmada.

(Placement: after REQ-27.13)

#### Scenario: Archivado confirmado
- **DADO** el doble configurado y una publicación de archivado confirmada
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** el doble recibe `archive` con la clave `archive/x/<fecha>-x`
- **Y** el resultado y el código de salida de `publish` son los de siempre

#### Scenario: Publicación rechazada
- **DADO** una publicación que termina con una negativa tipada (REQ-27.9)
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

### Requirement: El Evento `archive` no Reescribe `integrations.yaml` (REQ-43.27)

El despacho de `archive` NO DEBE modificar `integrations.yaml` de la carpeta archivada ni ningún otro fichero versionado: `integrations.yaml` archivado solo se lee, para dar los vínculos al plugin. Los vínculos o imputaciones que devuelva el plugin al entregar `archive` se descartan con un aviso por stderr y NO se persisten en este cambio. La única escritura permitida es la cola local de la carpeta archivada (REQ-21.18 modificado), de modo que la carpeta archivada ya publicada no quede sucia en ficheros versionados. La regla se impone por mecanismo, no solo por política: la guarda de escritura bajo `archive/` rechaza cualquier nombre que no sea la cola y sus temporales.

(Placement: after REQ-43.26)

#### Scenario: El plugin devuelve vínculos en `archive`
- **DADO** un doble que devuelve un vínculo al recibir `archive`
- **CUANDO** se confirma `publish --archive`
- **ENTONCES** `integrations.yaml` de la carpeta archivada no cambia y `git status` no muestra cambios en ficheros versionados
- **Y** stderr avisa de que los vínculos devueltos se han descartado

#### Scenario: Intento de escribir `integrations.yaml` archivado
- **DADO** una carpeta archivada
- **CUANDO** el marco de integraciones intenta escribir `integrations.yaml` en ella
- **ENTONCES** la escritura se rechaza con el mensaje de REQ-21.18 y no se modifica ningún fichero

#### Scenario: Entrega fallida de `archive`
- **DADO** el doble configurado para fallar
- **CUANDO** se confirma `publish --archive`
- **ENTONCES** la operación queda pendiente solo en `.integrations-outbox.yaml` de la carpeta archivada, que no se versiona
