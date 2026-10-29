# Delta para Declaración de la Baseline del Proyecto y Guarda contra una Segunda Baseline (INC-25)

> **Incremento:** `inc-25-ramas-y-worktrees-por-topologia` · **Tipo:** delta sobre la capacidad viva `spec-baseline-declaration`
> **Idioma:** Español (castellano peninsular)

Cambio de este delta: REQ-24.13 nombra `monorepo-decoupled` («tres topologías») en su texto normativo y en un escenario; la topología se retira (REQ-25.25 de `workspace-topology`) y el bloque `baseline:` se admite en las dos topologías restantes. Ningún otro requerimiento de la capacidad cambia.

> Nota para el archivado: la prosa no normativa del documento vivo (propósito y términos) no cita `monorepo-decoupled`; la tabla «Requerimientos de un vistazo» (fila REQ-24.13) no menciona topologías y no requiere edición.

## MODIFIED Requirements

### Requirement: Bloque `baseline:` Opcional y Defecto OpenSpec (REQ-24.13)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `baseline:` con la lista `documents`. Sin el bloque, la baseline es la de OpenSpec por defecto y la lectura, la integración y el archivado DEBEN comportarse exactamente como antes de esta capacidad. Con el bloque, la baseline es la del proyecto: los documentos declarados, en su ubicación y formato, y Axiom NO DEBE consultar, crear ni actualizar `<planning home>/specs/` como baseline. El bloque se admite en las dos topologías (`monorepo-embedded` y `multirepo`). Un bloque `baseline:` sin documentos (lista ausente o vacía) NO es válido.
(Previously: el bloque se admitía en las tres topologías, incluida la retirada `monorepo-decoupled`)

#### Scenario: Sin bloque la baseline es OpenSpec
- **DADO** un `axiom.yaml` sin `baseline:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status no contiene la clave `baseline`
- **Y** archive integra los deltas en `<planning home>/specs/<dominio>/spec.md` como antes

#### Scenario: Con bloque la baseline es la del proyecto
- **DADO** un `axiom.yaml` con `baseline.documents` que declara el dominio `requisitos-funcionales`
- **CUANDO** archive integra un cambio con delta para ese dominio
- **ENTONCES** la integración se aplica al documento declarado
- **Y** no se consulta ni se crea `<planning home>/specs/`

#### Scenario: El bloque se admite en las dos topologías
- **DADO** un `axiom.yaml` con `baseline:` válido en `monorepo-embedded` y en `multirepo`
- **CUANDO** se ejecuta `axiom workspace validate` en cada uno
- **ENTONCES** ninguno reporta error por la presencia del bloque

#### Scenario: Bloque sin documentos
- **DADO** un `axiom.yaml` con `baseline: {}` o con `documents: []`
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** se reporta en español que la baseline declarada requiere al menos un documento
- **Y** el código de salida es `1`
