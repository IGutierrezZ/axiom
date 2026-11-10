# Delta para Ciclo de Vida del Archivado SDD (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** delta sobre la capacidad viva `sdd-archive-lifecycle`.
> **Idioma:** castellano de España; identificadores y códigos en inglés.

Cambios de este delta:

- REQ-27.14 (nuevo): tras archivar, los documentos que toca el archivado se publican con `axiom sdd publish --change <x> --archive` sobre la rama de archivado, con la misma lista cerrada de rutas y las mismas negativas de `sdd-docs-publication`.
- REQ-21.16, REQ-21.17, REQ-25.39 y el resto no cambian: la precondición de integración, el contenido del archivado y el PR documental aparte siguen igual; la PR y el merge siguen siendo de la persona.

## ADDED Requirements

### Requirement: Publicación de la Documentación Tras el Archivado (REQ-27.14)

Una vez completado `archive`, la documentación que toca el archivado DEBE poder publicarse con `axiom sdd publish --change <x> --archive` (capacidad `sdd-docs-publication`) sobre la rama donde se archivó (el indicador `--archive` selecciona la rama de archivado como rama esperada de `publish`): en `monorepo-embedded`, la rama de archivado `<prefijo>/<cambio>-archive` que prepara `axiom branch ensure --change <x> --archive` (REQ-25.45 de `change-branch-lifecycle`); en el repositorio de specs de un `multirepo`, su rama de integración, donde se archiva in situ. La publicación DEBE incluir únicamente rutas de la lista cerrada de documentación (REQ-27.5): la carpeta archivada del cambio bajo `openspec/changes/archive/`, el borrado del directorio `openspec/changes/<x>/`, las especificaciones vivas de `openspec/specs/` que integró el archivado, `openspec/INDEX.md` y las rutas de baseline o de contexto técnico mantenido a las que el archivado integró su delta (REQ-24.55). NO DEBE incluir código ni rutas de `technical_context.reference`. En un cambio `continuous` con `docs_publication: auto` el orquestador DEBE ejecutar la publicación al cerrar el archivado, sin preguntar (REQ-27.13); en un cambio sin ese campo DEBE ofrecer `axiom sdd publish --change <x> --archive` como siguiente paso, y en un cambio `checkpointed` DEBE ofrecerlo igualmente, sin parada adicional. `archive` NO DEBE ejecutar `git commit` ni `git push` por sí mismo: la publicación es una orden aparte. Una negativa de `publish` (rama equivocada, almacén sin ficheros, remoto inexistente) es informativa y NO revierte el archivado. La PR documental contra la integración y su merge siguen siendo decisión de la persona (REQ-25.39).

#### Scenario: Monorepo publica la rama de archivado

- **DADO** un monorepo con el cambio `x` archivado en la rama `inc/x-archive`, con `openspec/changes/archive/<fecha>-x/` creado, `openspec/changes/x/` eliminado y las especificaciones vivas actualizadas
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** `origin/inc/x-archive` contiene un commit con esas rutas de documentación
- **Y** ningún fichero de código fuente aparece en el commit

#### Scenario: Multirepo publica la integración del repositorio de specs

- **DADO** un multirepo con el cambio archivado in situ en la rama de integración del repositorio de specs
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive` en ese repositorio
- **ENTONCES** se sube su rama de integración a `origin` con los documentos archivados

#### Scenario: La baseline y el contexto mantenido integrados se publican

- **DADO** un archivado que integró su delta en `docs/requisitos.md` (baseline declarada) y `docs/arquitectura.md` (`technical_context.maintained`)
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** ambas rutas figuran en el commit
- **Y** una ruta de `technical_context.reference` modificada en el mismo árbol no figura en el commit

#### Scenario: Cambio continuo autorizado publica al archivar

- **DADO** un cambio `continuous` con `docs_publication: auto` cuyo archivado acaba de completarse
- **CUANDO** el orquestador cierra el archivado
- **ENTONCES** ejecuta `axiom sdd publish --change x --archive` sin preguntar

#### Scenario: Cambio continuo sin autorización ofrece la publicación

- **DADO** un cambio `continuous` sin `docs_publication` cuyo archivado acaba de completarse
- **CUANDO** el orquestador cierra el archivado
- **ENTONCES** no ejecuta `publish` y ofrece `axiom sdd publish --change x --archive` como siguiente paso

#### Scenario: Publicar con `--archive` fuera de la rama de archivado se rechaza

- **DADO** un monorepo con el cambio `x` archivado y el repositorio desplegado en la rama del cambio `inc/x`
- **CUANDO** se ejecuta `axiom sdd publish --change x --archive`
- **ENTONCES** se rechaza con `wrong_branch` nombrando `inc/x-archive` como rama esperada
- **Y** no se confirma ni se sube nada

#### Scenario: Sin `--archive` desde la rama de archivado se rechaza nombrando la orden correcta

- **DADO** un monorepo con el cambio `x` archivado y el repositorio desplegado en `inc/x-archive`
- **CUANDO** se ejecuta `axiom sdd publish --change x` sin `--archive`
- **ENTONCES** se rechaza con `wrong_branch` nombrando `inc/x` como rama esperada
- **Y** el mensaje incluye la orden correcta `axiom sdd publish --change x --archive`
- **Y** no se confirma ni se sube nada

#### Scenario: `archive` no publica por sí mismo

- **DADO** un observador instalado en el único punto del producto que ejecuta git
- **CUANDO** se ejecuta `archive` con éxito
- **ENTONCES** el observador no registra `commit` ni `push`

#### Scenario: Una negativa de `publish` no revierte el archivado

- **DADO** un cambio archivado con un commit de código sin subir en la rama de archivado
- **CUANDO** `publish` se niega con `code_commits_unpublished`
- **ENTONCES** el cambio sigue archivado en `openspec/changes/archive/`
- **Y** el orquestador presenta la negativa como informativa con la orden de push
