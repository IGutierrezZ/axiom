# Delta para requirement-identifiers

Este delta (C-03-c) elimina la excepción temporal de REQ-44.4: `adoptcheck` y `archive adopt` dejan de usar el patrón por defecto fijo y obtienen el patrón efectivo del workspace como el resto de consumidores.

## MODIFIED Requirements

### Requirement: Un Único Punto de Parseo, Validación, Formato y Serie que Usan los Consumidores (REQ-44.4)

El parseo del identificador de un título de requisito, la detección de candidatos, la validación, la extracción de serie y número, el formato del siguiente identificador, las notaciones del patrón y el identificador sintético DEBEN residir en un único paquete hoja (`internal/reqid`, solo biblioteca estándar) que no depende de ningún otro paquete de Axiom. `openspecdoc` e `isoevidence` solo pueden importar ese paquete entre los internos. El identificador de un título es el último identificador válido entre paréntesis (`(ID)`); el campo `ID` precalculado de los requisitos de `openspecdoc` desaparece, de modo que ningún consumidor lee un identificador calculado con un patrón por defecto oculto.

Los consumidores `openspecdoc`, `isoevidence`, `sddstatus`, `livespec`, `livingdoc`, `internal/adoption/adoptcheck` y la CLI (`archive integrate`, `archive adopt`, `sdd archive-compose`, `sdd archive-check`, `sdd trace`) DEBEN obtener el patrón efectivo del workspace cuando lo tienen y recibirlo por parámetro u opción cuando no, con el valor por defecto si no se pasa. Ningún consumidor DEBE conservar una expresión regular, una constante ni un `Sprintf` propios del formato del identificador. NO existe ninguna excepción: `axiom archive adopt check` y los textos de `axiom archive adopt` leen `sdd.id_pattern` y miden con el patrón efectivo; un patrón declarado inaccesible o inválido da el mismo error accionable que en el resto de consumidores.
(Previously: excepción deliberada y temporal (A-03-2, REQ-30.31) por la que `internal/adoption/adoptcheck` y `axiom archive adopt` seguían usando el patrón por defecto `REQ-<n>.<m>` hasta C-03-c, sin leer `sdd.id_pattern`.)

**Criterio de terminado ejecutable** (decisión del 2026-11-23): `rg -n 'REQ-[^0-9]' --type go internal cmd -g '!*_test.go'` solo coincide en `internal/reqid` y en comentarios, y la prueba de arquitectura `TestNoRequirementIDFormatOutsideReqid` falla si un literal de texto de un `.go` de producción de `internal/` o `cmd/` contiene `REQ-` seguido de algo que no sea una cifra fuera de `internal/reqid` (los comentarios quedan fuera). Las citas de requisitos de Axiom con cifras (`REQ-21.18`) en mensajes y la plantilla de borrador de `internal/knowledge/sweep.go` quedan fuera de este criterio y para C-03-e.

#### Scenario: Ningún consumidor define el formato

- **DADO** el árbol de código de producción tras este cambio
- **CUANDO** se ejecuta `rg -n 'REQ-[^0-9]' --type go internal cmd -g '!*_test.go'`
- **ENTONCES** solo hay coincidencias en `internal/reqid` y en comentarios

#### Scenario: La prueba de arquitectura vigila los literales de texto

- **DADO** un literal de texto de producción fuera de `internal/reqid` que contiene `REQ-` seguido de una letra, un `<` o un guion
- **CUANDO** se ejecuta `TestNoRequirementIDFormatOutsideReqid`
- **ENTONCES** la prueba falla nombrando el fichero
- **Y** un literal con `REQ-` seguido de una cifra, o un comentario, no la hace fallar

#### Scenario: Todos los consumidores coinciden ante la misma entrada

- **DADO** un proyecto con el patrón `RF-<DOM>-<NN>` y un título `Alta de usuario (RF-USU-01)`
- **CUANDO** lo consultan `openspecdoc`, `isoevidence` y `livespec`
- **ENTONCES** los tres obtienen el mismo identificador `RF-USU-01`, la misma serie `USU` y el mismo número `1`

#### Scenario: El puerto y los consumidores de la hoja respetan las importaciones

- **DADO** los paquetes `internal/reqid`, `internal/openspecdoc` e `internal/isoevidence`
- **CUANDO** se listan sus importaciones
- **ENTONCES** `internal/reqid` solo importa biblioteca estándar
- **Y** `internal/openspecdoc` e `internal/isoevidence` solo importan `internal/reqid` entre los paquetes de Axiom

#### Scenario: Sin patrón pasado, un consumidor usa el defecto

- **DADO** un consumidor invocado sin patrón (por ejemplo, `openspecdoc` sin opciones)
- **CUANDO** procesa un título con `(REQ-30.4)`
- **ENTONCES** reconoce `REQ-30.4`, con el patrón por defecto

#### Scenario: `adoptcheck` con el defecto se comporta como un proyecto por defecto

- **DADO** un proyecto sin `sdd.id_pattern` y un cambio de adopción con identificadores `REQ-<n>.<m>`
- **CUANDO** se ejecuta `axiom archive adopt check`
- **ENTONCES** los identificadores se validan con el patrón por defecto `REQ-<n>.<m>` y los fallos de formato y unicidad se reportan con las reglas de REQ-30.11

#### Scenario: `adoptcheck` y `archive adopt` leen el patrón declarado

- **DADO** un proyecto con `sdd: {id_pattern: "RF-<NNN>"}` y un título de requisito `(RF-001)`
- **CUANDO** se ejecuta `axiom archive adopt check` o `axiom archive adopt`
- **ENTONCES** se mide y se informa con el patrón `RF-<NNN>`, sin fallo de formato por el identificador `RF-001`
- **Y** los textos y la ayuda de `axiom archive adopt` citan el patrón declarado y no la excepción «hasta C-03-c»

#### Scenario: Un patrón declarado inválido da el error accionable en la adopción

- **DADO** un proyecto con un `sdd.id_pattern` que no es un patrón válido
- **CUANDO** se ejecuta `axiom archive adopt check`
- **ENTONCES** falla con el mismo error accionable que el resto de verbos, nombrando el patrón
