<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Cierre de Último Rol, Aviso Formal y Relevo de Integración (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para el cierre determinista del último rol activo en un cambio SDD, incluyendo el aviso formal al usuario y la transición hacia la fase `verify` global de toda la solución, habilitada al aprobarse todos los `role-apply` del roster (REQ-21.15); el `handoff.md` de integración queda retirado (REQ-21.14).

---

## 1. Capacidad: `sdd-role-closure-handoff`

Al concluir el último rol activo del cambio, el orquestador cierra la fase de implementación con un aviso formal, un relevo de integración y la apertura de la verificación global de toda la solución.

### Requirement: Aviso Formal de Conclusión del Último Rol Activo (REQ-21.13)

Al aprobarse la compuerta de `apply` (REQ-21.11) del último rol pendiente entre los roles activos del cambio —incluido el caso de un único rol `fullstack`, y tanto con la salida 2 como con la salida 3 de REQ-26.1— el orquestador DEBE notificar formalmente al usuario que ha concluido la implementación del último rol activo y que queda abierta la verificación global de la solución (REQ-21.15). El aviso NO DEBE anunciar la generación de ningún relevo y NO DEBE escribir ningún fichero `handoff.md`.
(Previously: el aviso anunciaba que se generaba el relevo de integración para la verificación global)

#### Scenario: Último de varios roles activa el aviso sin relevo

- **DADO** un cambio con los roles `core`, `web` y `qa`, donde `core` y `web` ya están concluidos y `qa` es el único rol pendiente
- **CUANDO** se aprueba la compuerta de `apply` del rol `qa`
- **ENTONCES** el orquestador emite el aviso formal de conclusión del último rol
- **Y** no escribe ningún `handoff.md` en la carpeta del cambio

#### Scenario: Rol único fullstack satisface la condición de último rol de inmediato

- **DADO** un cambio con el rol único `fullstack`
- **CUANDO** se aprueba la compuerta de `apply` de ese único rol
- **ENTONCES** esa misma aprobación satisface la condición de "último rol activo"
- **Y** el orquestador emite el aviso formal inmediatamente, sin esperar ningún otro rol y sin escribir `handoff.md`

#### Scenario: Compuerta rechazada no dispara el cierre aunque sea el único rol pendiente

- **DADO** un cambio en el que el único rol pendiente tiene su compuerta de `apply` rechazada (REQ-21.12)
- **CUANDO** se registra el rechazo
- **ENTONCES** el orquestador no emite el aviso de último rol
- **Y** espera a que esa compuerta se apruebe explícitamente antes de reevaluar la condición de cierre

#### Scenario: Aprobar el último rol con la salida 2 emite el aviso y cede verify

- **DADO** un cambio con el rol único `fullstack` y su trabajo confirmado en la rama
- **CUANDO** se aprueba la compuerta `role-apply:fullstack` con la salida 2
- **ENTONCES** se emite el aviso de último rol
- **Y** la fase `verify` queda cedida a otra persona (REQ-26.7)
- **Y** no se escribe ningún `handoff.md`

---

### Requirement: Generación del `handoff.md` de Integración (REQ-21.14) [SUPERSEDED / DEPRECADO]
> [!CAUTION]
> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.
> Motivo: el usuario decidió (Engram #1811, `product/paradas-tres-salidas`) no implementar el relevo intermedio ni el de integración como artefacto del flujo SDD: el relevo entre personas se resuelve con las tres salidas de las paradas, que dejan lo aprobado en la rama de integración. El último `role-apply` aprobado ya habilita `verify` por sí solo (REQ-21.15).


Junto con el aviso de REQ-21.13, el orquestador DEBE generar un artefacto `handoff.md` canónico, conforme al esquema ya definido para relevos estructurados (frontmatter YAML con `from_phase`, `to_phase`, `from_role`, `to_role`, `timestamp` y `status`, seguido de las cinco secciones descriptivas), con `status: ready`, consolidando los cambios de todos los roles participantes y especificando las instrucciones de prueba para la fase `verify`.

#### Scenario: Handoff de integración consolida todos los roles participantes

- **DADO** el aviso de conclusión del último rol activo de un cambio con varios roles
- **CUANDO** el orquestador genera el relevo de integración
- **ENTONCES** el `handoff.md` resultante declara `from_phase: apply`, `to_phase: verify` y `status: ready`
- **Y** su contenido consolida los artefactos y decisiones de todos los roles participantes, no solo del último rol concluido

---

### Requirement: Ejecución de la Fase `verify` Global tras Aprobarse Todos los `role-apply` (REQ-21.15)

En un cambio con más de un rol y con compuertas (modalidad "con paradas"), la fase `verify` global DEBE habilitarse cuando todas las compuertas `role-apply:<rol>` del roster sellado están aprobadas, y NO DEBE habilitarse mientras falte alguna. No se exige ni se consulta ningún `handoff.md`: un `handoff.md` presente, ausente, `ready`, `blocked` o `needs_clarification` NO DEBE alterar esa condición. Una vez habilitada, la fase `verify` DEBE ejecutar la suite completa de pruebas de la solución (integración, regresión y compuertas de seguridad aplicables) considerando el trabajo consolidado de todos los roles, y DEBE emitir un informe consolidado `verify-report.md`, distinto de los informes individuales `verify-report.<rol>.md` de cada rol. La dependencia `verify` de `axiom sdd status` DEBE reflejar esa condición. Un cambio con un único rol, o en modalidad "continua", NO DEBE verse afectado por este requisito.
(Previously: «Ejecución de la Fase `verify` Global tras el Handoff»: la fase `verify` global exigía un `handoff.md` de integración en `status: ready` y no se iniciaba con `blocked` o `needs_clarification`)

#### Scenario: Verify global emite un informe consolidado distinto de los informes por rol

- **DADO** un cambio con varios roles, todos con `role-apply` aprobado y verificados individualmente
- **CUANDO** se ejecuta la fase `verify` global
- **ENTONCES** se genera `verify-report.md` consolidado, distinto de los `verify-report.<rol>.md` de cada rol
- **Y** el informe refleja la suite completa de pruebas de la solución, no solo la de un rol

#### Scenario: Verify multirrol bloqueado mientras falte algún role-apply aprobado

- **DADO** un cambio con los roles `core` y `web` donde solo `role-apply:core` está aprobado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la dependencia `verify` aparece bloqueada
- **Y** no se enruta hacia `verify`

#### Scenario: Verify multirrol se desbloquea con todos los role-apply aprobados sin handoff.md

- **DADO** un cambio con los roles `core` y `web` con ambos `role-apply` aprobados y sin `handoff.md` en la carpeta del cambio
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** la dependencia `verify` no está bloqueada
- **Y** `verify` es la fase siguiente proyectada

#### Scenario: Un handoff.md bloqueado ya no impide verify

- **DADO** un cambio multirrol con todos los `role-apply` aprobados y un `handoff.md` con `status: blocked` o `needs_clarification`
- **CUANDO** se evalúa si procede iniciar la fase `verify` global
- **ENTONCES** el estado del `handoff.md` no se consulta y `verify` no queda bloqueado por él

#### Scenario: Un role-apply rechazado mantiene verify bloqueado

- **DADO** un cambio multirrol con todos los `role-apply` aprobados salvo uno cuya última decisión es `rejected`
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** está bloqueada mientras ese rol no esté aprobado

#### Scenario: Un cambio de un solo rol o continuo no cambia

- **DADO** un cambio con el rol único `fullstack`, o un cambio sellado con modalidad "continua"
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** el resultado es el mismo que antes de este cambio, sin consultar `handoff.md` ni la antigua política de relevos

