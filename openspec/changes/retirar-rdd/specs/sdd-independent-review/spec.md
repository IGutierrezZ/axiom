# Delta for sdd-independent-review

## ADDED Requirements

### Requirement: Juicio Explícito con Dos Jueces Ciegos y Rondas Acotadas (REQ-28.43)

Judgment Day DEBE seleccionarse de forma explícita: por petición de la persona o porque su preferencia de sesión incluye el juicio (`verify-judgment`, REQ-28.2); nunca por iniciativa propia del orquestador ni como efecto de un hito de entrega. Cuando se selecciona, DEBE ejecutar dos jueces distintos, ciegos entre sí y de solo lectura, cuyos resultados se contrastan con un acuerdo o confirmación explícitos antes de dar por corroborado un hallazgo; no se usa ningún refutador adicional. El juicio DEBE permitir como máximo dos rondas de arreglo y dos rejuicios acotados al alcance de los arreglos. Si tras el segundo rejuicio persiste un hallazgo severo confirmado, el veredicto DEBE ser `ESCALATED` y NO DEBE abrirse una tercera ronda. El juicio NO DEBE emitir recibos, ni depender de un estado de revisión nativo, ni invocar comandos `axiom review`, y su veredicto es informativo (REQ-28.3). Sus artefactos (`sdd/{change}/review-ledger`) son los propios de Judgment Day.
(Trasladado desde el requisito «Explicit Judgment Day replacement» de la capacidad `review-findings-ledger`, retirada con RDD (C-01-e, P20), sin su acoplamiento a la revisión 4R, a `review-refuter`, al estado nativo ni a los recibos.)

#### Scenario: El juicio no se lanza sin selección explícita

- **DADO** un rol con verify limpio y una preferencia de sesión `verify` o `none` y sin petición de la persona
- **CUANDO** concluye el `apply` del rol
- **ENTONCES** el orquestador no lanza el juicio
- **Y** no lo sustituye por ninguna otra revisión automática

#### Scenario: Dos jueces ciegos de solo lectura

- **DADO** un juicio seleccionado de forma explícita sobre el diff de un rol
- **CUANDO** se lanzan los jueces
- **ENTONCES** son dos ejecuciones distintas que no ven el veredicto una de otra
- **Y** ninguna puede modificar ficheros ni lanzar otro rol
- **Y** un hallazgo solo se da por corroborado cuando ambos coinciden o se confirma de forma explícita

#### Scenario: El juicio no supera la segunda ronda

- **DADO** un hallazgo severo que persiste tras el segundo rejuicio acotado
- **CUANDO** el orquestador evalúa el presupuesto de rondas
- **ENTONCES** el veredicto es `ESCALATED`
- **Y** no se abre una tercera ronda de arreglo ni de rejuicio

#### Scenario: El juicio no emite recibos ni usa comandos de revisión

- **DADO** un juicio `APPROVED` tras un verify limpio
- **CUANDO** el flujo continúa
- **ENTONCES** no se crea ningún recibo ni consentimiento
- **Y** no se invoca ningún comando `axiom review`
- **Y** el veredicto no condiciona fase, puerta ni archivado

## REMOVED Requirements

### Requirement: Frontera con RDD: Ningún Texto Nuevo Usa su Vocabulario (REQ-28.11)

(Reason: el requisito solo existe para separar las revisiones independientes de RDD (vocabulario, recibos, consentimiento, estado y guardas AST `OfferReviewAfterVerify`/`ReviewCore`); con RDD retirado (C-01-e, P20) no hay frontera que guardar y las guardas se borran con `internal/sddstatus/review_*`.)
(Migration: la ausencia de vocabulario y de comandos de revisión por recibos en la doctrina y las skills pasa a REQ-46.3 de `sdd-orchestrator-assets`; que el juicio no emita recibos pasa a REQ-28.43 de esta capacidad; `sdd-block-review-gates` (REQ-28.10) deja de citar REQ-28.11.)
