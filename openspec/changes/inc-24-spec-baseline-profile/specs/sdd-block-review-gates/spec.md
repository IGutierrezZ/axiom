# Delta para Compuertas de Revisión por Bloque (spec, design, tasks, apply) (INC-24)

> **Incremento:** `inc-24-spec-baseline-profile` · **Tipo:** delta sobre la capacidad viva `sdd-block-review-gates`

Cambios de este delta: con `technical_context.maintained` declarado, la compuerta `design` (REQ-21.9) evalúa además el impacto previsto en el contexto mantenido y la compuerta `apply` por rol (REQ-21.11) evalúa además el impacto final. La mecánica de las compuertas (comandos de registro y ledger) no cambia.

## MODIFIED Requirements

### Requirement: Compuerta de `design` (REQ-21.9)

En modalidad "con paradas", al completarse la fase `design` el orquestador DEBE detenerse y evaluar explícitamente si el diseño satisface todos los requerimientos del spec y si respeta las directrices de arquitectura, las tecnologías declaradas en `axiom.yaml` y los patrones existentes del repositorio. El orquestador NO DEBE iniciar la fase `tasks` mientras esta compuerta no reciba una aprobación explícita del usuario. Con `technical_context.maintained` declarado, la compuerta DEBE además presentar el delta de contexto con el impacto previsto y evaluar si es coherente con el diseño; si el delta falta, DEBE señalar esa ausencia explícitamente como parte de la presentación (REQ-24.36).
(Previously: la compuerta no evaluaba el impacto previsto en el contexto técnico mantenido)

#### Scenario: Diseño conforme aprobado habilita tasks

- **DADO** un `design.md` que cubre todos los requerimientos del spec aprobado y respeta la arquitectura declarada
- **CUANDO** se presenta la compuerta de `design` y el usuario la aprueba
- **ENTONCES** el orquestador inicia la fase `tasks`

#### Scenario: Desviación arquitectónica se señala explícitamente en la compuerta

- **DADO** un `design.md` que introduce una tecnología o patrón no declarado en `axiom.yaml` ni presente en el repositorio
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** el orquestador señala explícitamente esa desviación como parte de la presentación de la compuerta
- **Y** no inicia `tasks` hasta recibir una aprobación explícita

#### Scenario: El impacto previsto en el contexto mantenido se presenta en la compuerta

- **DADO** `technical_context.maintained` declarado y un diseño con su delta de contexto
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** la presentación incluye el delta de contexto y evalúa su coherencia con el diseño
- **Y** no inicia `tasks` hasta recibir una aprobación explícita

#### Scenario: La ausencia del delta de contexto se señala en la compuerta

- **DADO** `technical_context.maintained` declarado y un diseño sin delta de contexto
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** el orquestador señala explícitamente que falta el delta de contexto
- **Y** no asume «Sin impacto»

#### Scenario: Sin contexto mantenido la compuerta no cambia

- **DADO** un `axiom.yaml` sin `technical_context.maintained`
- **CUANDO** se presenta la compuerta de `design`
- **ENTONCES** la presentación no incluye ni exige delta de contexto

#### Scenario: Las plantillas del orquestador prescriben la presentación del delta en la compuerta de diseño

- **DADO** los textos de las plantillas de orquestador que describen la compuerta `design`
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican presentar el delta de contexto, y señalar su ausencia, cuando el status proyecta `technicalContext.maintained`

---

### Requirement: Compuerta de `apply` por Rol (REQ-21.11)

En modalidad "con paradas", al finalizar la implementación de cada rol activo (incluido el rol único `fullstack`), el orquestador DEBE detenerse y evaluar: conformidad con el diseño asignado a ese rol, cumplimiento de las funcionalidades de la `spec` asignadas a ese rol, y estándares de calidad (compilación limpia, pruebas unitarias ejecutadas, cobertura y estilo). El orquestador NO DEBE marcar el rol como concluido, ni permitir que otro rol pendiente inicie su propia fase `apply`, mientras esta compuerta no reciba una aprobación explícita. Con `technical_context.maintained` declarado, la compuerta DEBE además presentar el delta de contexto actualizado y evaluar que el impacto final refleja lo implementado por ese rol.
(Previously: la compuerta no evaluaba el impacto final en el contexto técnico mantenido)

#### Scenario: Rol concluido y aprobado habilita el siguiente rol pendiente

- **DADO** un cambio con los roles `core` y `web`, donde `core` termina su implementación primero
- **CUANDO** se presenta la compuerta de `apply` del rol `core` y el usuario la aprueba
- **ENTONCES** el orquestador marca `core` como concluido y habilita el inicio de la fase `apply` del rol `web`

#### Scenario: Incumplimiento de calidad bloquea la conclusión del rol

- **DADO** un rol cuya implementación no supera sus pruebas unitarias o no cumple la funcionalidad de spec asignada
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** el orquestador la rechaza en el criterio de calidad correspondiente y no marca el rol como concluido
- **Y** ese rol no puede disparar el aviso de último rol (REQ-21.13) hasta que su compuerta se apruebe

#### Scenario: El impacto final en el contexto mantenido se presenta en la compuerta

- **DADO** `technical_context.maintained` declarado y un rol que termina su implementación con el delta de contexto actualizado
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** la presentación incluye el delta de contexto actualizado y evalúa que refleja lo implementado por el rol
- **Y** no marca el rol como concluido hasta recibir una aprobación explícita

#### Scenario: Un impacto final no reflejado en el delta puede rechazarse

- **DADO** un rol cuya implementación cambia algo documentado en el contexto mantenido mientras el delta declara «Sin impacto»
- **CUANDO** se presenta la compuerta de `apply` de ese rol
- **ENTONCES** el orquestador señala la discrepancia como parte de la presentación
- **Y** el usuario puede rechazar la compuerta para que se remedie el delta antes de reintentar el avance (REQ-21.12)

#### Scenario: Las plantillas del orquestador prescriben la presentación del delta en la compuerta de apply

- **DADO** los textos de las plantillas de orquestador que describen la compuerta `apply` por rol
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** indican presentar el delta de contexto actualizado cuando el status proyecta `technicalContext.maintained`

---

