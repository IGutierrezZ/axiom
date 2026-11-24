# Delta para sdd-independent-review

> Parte del estado vigente: el resto de requisitos (REQ-28.2, REQ-28.3, REQ-28.5 a REQ-28.11 incluidos) NO cambia. Esta delta añade una verificación conjunta que NO es un cuarto informe ni una fase nueva: es el `verify` global sin rol (REQ-21.15 de `sdd-role-closure-handoff`, que no cambia) con el alcance que calcula `axiom branch scope` (REQ-25.51 de `change-branch-lifecycle`), que el orquestador **ofrece** al presentar la compuerta `integration`. Decisión del usuario del 2026-11-23 (opción B): se ofrece, se documenta y NUNCA bloquea. REQ-28.3 no se modifica: la verificación conjunta queda cubierta por su regla general de no bloqueo, que REQ-28.42 repite de forma expresa. Los identificadores REQ-28.41 y REQ-28.42 son los siguientes libres tras REQ-28.40. Versión simple (decisión del usuario del 2026-11-24): sin vigencia de rondas ni localización de ramas borradas (véase «Fuera de alcance» en `change-branch-lifecycle`).

## RENAMED Requirements

### Requirement: El Orquestador Ofrece la Revisión Independiente en Tres Puntos del Flujo (REQ-28.1) → El Orquestador Ofrece la Revisión Independiente en Cuatro Puntos del Flujo (REQ-28.1)

(Reason: el requisito pasa de tres a cuatro puntos al añadir la oferta de la verificación conjunta en la presentación de `integration`)
(Migration: las referencias al nombre anterior, los pines de texto del orquestador y las pruebas que citen «tres puntos» DEBEN actualizarse; el identificador REQ-28.1 no cambia)

## MODIFIED Requirements

### Requirement: El Orquestador Ofrece la Revisión Independiente en Cuatro Puntos del Flujo (REQ-28.1)

El orquestador DEBE contemplar la revisión independiente en cuatro puntos y solo en ellos: (a) al concluir el `apply` de cada rol activo, incluido el rol único `fullstack`, el `verify` de ese rol y, tras un verify limpio, el juicio; (b) al concluir la fase `spec`, la revisión de cobertura del spec; (c) al concluir la fase `design`, la revisión de cobertura de spec y design; (d) al presentar la compuerta `integration` de un cambio `checkpointed`, la **oferta** de la verificación conjunta cuando el resumen de `axiom branch scope` indica `offerable` (REQ-25.54 de `change-branch-lifecycle`). En los puntos (a), (b) y (c) DEBE aplicar la preferencia del usuario según el estilo sellado (REQ-28.2). El punto (d) es solo una oferta: el orquestador NO DEBE lanzar la verificación conjunta por iniciativa propia, con ninguna preferencia, y NO DEBE añadir una pregunta ni una opción nueva a la preferencia de revisión independiente; la persona decide en la propia presentación de `integration`. La oferta NO DEBE añadir estados, campos de ledger ni puertas: es texto del orquestador. Con la preferencia `none` o cobertura `no`, el orquestador NO DEBE lanzar la revisión por iniciativa propia en los puntos (a), (b) y (c), pero DEBE lanzarla si el usuario la pide de forma explícita, y DEBE atender siempre una petición explícita de verificación conjunta, aunque la oferta no sea `offerable`, declarando los huecos (REQ-28.41).
(Previously: «tres puntos y solo en ellos», sin ninguna oferta en `integration`)

#### Scenario: Un rol que concluye su apply activa el verify según la preferencia
- **DADO** un cambio con el rol `core` y la preferencia `verify` tras el apply
- **CUANDO** `core` concluye su `apply`
- **ENTONCES** el orquestador lanza el `verify` de `core` sobre el diff de `core`
- **Y** no lanza ningún juicio

#### Scenario: Cada rol activa su propia oferta
- **DADO** un cambio con los roles `core` y `web` y la preferencia `verify`
- **CUANDO** `core` concluye su `apply` y, más tarde, `web` concluye el suyo
- **ENTONCES** el orquestador lanza un `verify` tras cada uno, cada uno sobre el diff de su rol
- **Y** el verify de `web` no se lanza antes de que `web` concluya

#### Scenario: Con la preferencia none no se lanza nada por iniciativa propia
- **DADO** la preferencia `none` y cobertura `no`
- **CUANDO** concluyen `spec`, `design` y el `apply` de un rol
- **ENTONCES** el orquestador no lanza ningún verify, juicio ni revisión de cobertura por iniciativa propia
- **Y** el flujo continúa según el estilo sellado

#### Scenario: Una petición explícita se atiende con la preferencia none
- **DADO** la preferencia `none`
- **CUANDO** el usuario pide explícitamente un verify del rol `core` ya aplicado
- **ENTONCES** el orquestador lanza el verify de `core` conforme a esta capacidad

#### Scenario: Las ofertas no crean estado de gobierno
- **DADO** un cambio en el que se han ejecutado verify, juicio, cobertura y verificación conjunta
- **CUANDO** se inspeccionan `gates.yaml` y `kickoff.yaml`
- **ENTONCES** no contienen ningún campo, decisión ni puerta nueva atribuible a estas revisiones

#### Scenario: Un rol sin diff propio no produce un verify vacío engañoso
- **DADO** un rol cuyo apply no modificó ningún fichero
- **CUANDO** se aplica la preferencia `verify` tras su apply
- **ENTONCES** el verify informa de que el diff del rol está vacío y no emite un veredicto de cumplimiento técnico sobre nada
- **Y** no se ofrece juicio

#### Scenario: La compuerta integration ofrece la verificación conjunta sin lanzarla
- **DADO** un cambio `checkpointed` con la preferencia `verify-judgment` y `offer` igual a `offerable`
- **CUANDO** el orquestador presenta la compuerta `integration`
- **ENTONCES** ofrece la verificación conjunta en la presentación
- **Y** no la lanza hasta que la persona la acepta
- **Y** no formula ninguna pregunta de preferencia nueva

#### Scenario: Con la preferencia none la oferta de integration no se lanza sola
- **DADO** la preferencia `none` y `offer` igual a `offerable`
- **CUANDO** el orquestador presenta la compuerta `integration`
- **ENTONCES** declara el estado «ofrecida» sin lanzarla por iniciativa propia
- **Y** la lanza si la persona la pide de forma explícita

### Requirement: Verify por Rol con Sección de Cumplimiento Técnico (REQ-28.4)

El `verify` de un rol DEBE inspeccionar únicamente el diff de ese rol y escribir su informe en el `verify_file` que el kickoff asigna a ese rol. Además de la verificación funcional existente (requisitos, pruebas, compilación), el informe DEBE incluir una sección **Cumplimiento técnico** que contraste el diff con las reglas aplicables (REQ-28.5). Cuando no hay hallazgos, la sección DEBE existir y declararlo, con la lista de fuentes consultadas. El verify global multirrol, que se evalúa cuando REQ-21.15 lo habilita, no cambia en cuanto a su habilitación ni a la semántica del veredicto que lee la barrera de `multi-role-fan-out`: la sección nueva es contenido adicional del informe. **Alcance del verify sin rol:** cuando el verify global se ejecuta sin rol en modo conjunto sobre un cambio con instantánea sellada de repositorios, DEBE inspeccionar la **unión del alcance** que devuelve `axiom branch scope` (REQ-25.51 de `change-branch-lifecycle`) y no un alcance definido por el agente; ese alcance solo delimita lo que se inspecciona: la suite completa de pruebas que REQ-21.15 exige no se acota por él. Cuando `axiom branch scope` no es aplicable o falla, el verify sin rol se comporta como antes y su informe declara el motivo; nunca falla por ello ni cambia su veredicto. En un cambio ISO, el informe consolidado (o el del rol único `fullstack`) incluye además la matriz de verify (REQ-29.6 de `sdd-iso-evidence`) como otra sección adicional, que NO escribe líneas `verdict:` y NO altera la sección «Cumplimiento técnico»; los informes por rol de un cambio multirrol NO necesitan matriz. En un cambio con el rol único `fullstack`, el verify del rol y el verify del cambio son el mismo informe y NO se ejecutan dos veces sin motivo. Nota: el verify por rol es un diagnóstico del rol que se ofrece antes de aprobar `role-apply`; NO es la fase `verify` global de REQ-21.15, que sigue evaluándose cuando REQ-21.15 la habilita.
(Previously: el verify global multirrol «NO cambia» sin matiz y no tenía alcance definido; ahora, sin rol y en modo conjunto, su alcance es la unión del de `axiom branch scope`, sin alterar su habilitación ni el veredicto que lee la barrera)

#### Scenario: El informe de un rol contiene la sección Cumplimiento técnico
- **DADO** el rol `web` con su `apply` concluido y un diff en `web`
- **CUANDO** se ejecuta su verify
- **ENTONCES** `verify-report.web.md` contiene una sección «Cumplimiento técnico»
- **Y** el informe no incluye hallazgos sobre ficheros modificados solo por otro rol

#### Scenario: El rol fullstack escribe en verify-report.md
- **DADO** un cambio con el rol único `fullstack`
- **CUANDO** se ejecuta su verify
- **ENTONCES** el informe se escribe en `verify-report.md`
- **Y** contiene la sección «Cumplimiento técnico»

#### Scenario: Sin hallazgos la sección existe y lo declara
- **DADO** un diff que cumple todas las reglas aplicables
- **CUANDO** se genera el informe
- **ENTONCES** la sección «Cumplimiento técnico» declara que no hay hallazgos
- **Y** enumera las fuentes de reglas contrastadas

#### Scenario: El verify global multirrol no cambia su habilitación
- **DADO** un cambio con los roles `core` y `web`, ambos `blocking`, y todos los `role-apply` aprobados
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** se habilita conforme a REQ-21.15, sin ninguna condición añadida por este requisito
- **Y** el veredicto que lee la barrera de sincronización se obtiene como antes, sin tener en cuenta la sección nueva

#### Scenario: El verify sin rol en modo conjunto usa el alcance de axiom branch scope
- **DADO** un cambio multirrepo con instantánea sellada y el verify global lanzado sin rol en modo conjunto
- **CUANDO** el verify determina qué inspecciona
- **ENTONCES** inspecciona, por repositorio, los commits y ficheros que devuelve `axiom branch scope`
- **Y** no incluye commits ajenos que ese comando excluye

#### Scenario: Si el comando de alcance no es aplicable o falla, el verify lo declara y continúa
- **DADO** un cambio sin instantánea sellada, o una ejecución de `axiom branch scope` que termina con un error tipado
- **CUANDO** se ejecuta el verify sin rol
- **ENTONCES** el informe declara el motivo concreto y que se aplicó el comportamiento anterior
- **Y** el verify no falla por ello ni cambia su veredicto

#### Scenario: El alcance no acota la suite completa de pruebas
- **DADO** un verify sin rol en modo conjunto con un alcance de dos repositorios
- **CUANDO** se ejecutan las pruebas que REQ-21.15 exige
- **ENTONCES** se ejecuta la suite completa y no solo la de los ficheros del alcance
- **Y** el informe indica que el alcance solo delimita lo inspeccionado

#### Scenario: En un cambio ISO el informe consolidado incluye la matriz como sección adicional
- **DADO** un cambio ISO con los roles `core` y `web` y todos los `role-apply` aprobados
- **CUANDO** se genera el `verify-report.md` consolidado
- **ENTONCES** contiene la matriz de verify junto a las demás secciones
- **Y** la barrera lee la misma línea `verdict:` que sin la matriz
- **Y** los informes `verify-report.core.md` y `verify-report.web.md` no necesitan matriz

#### Scenario: Un rol con herramienta de análisis no disponible informa la limitación
- **DADO** una comprobación de estilo declarada cuyo ejecutable no está disponible
- **CUANDO** se genera la sección «Cumplimiento técnico»
- **ENTONCES** declara la comprobación como no ejecutada con su motivo
- **Y** no la presenta como superada

## ADDED Requirements

### Requirement: La Verificación Conjunta es el Verify Sin Rol sobre la Unión del Alcance por Repositorio (REQ-28.41)

La verificación conjunta DEBE ser el `verify` global sin rol (REQ-21.15 de `sdd-role-closure-handoff`) y NO un informe ni una fase adicional. Su alcance DEBE ser la **unión por repositorio** de lo que devuelve `axiom branch scope` para el cambio, y DEBE verificar **solo commits**: NO DEBE incluir el árbol de trabajo ni los ficheros sin seguimiento. Los commits ajenos que `axiom branch scope` excluya DEBEN listarse como excluidos, y un aviso `foreign_commits_possible` DEBE reproducirse como aviso; nunca se incluyen en silencio.

El resultado DEBE escribirse en el informe consolidado `verify-report.md` (el `verify_file` del rol `fullstack` cuando es el único rol) como una sección propia **«Verificación conjunta»**, fechada, **una por ronda**. La sección DEBE añadirse sin sobrescribir la matriz ISO ni ningún otro contenido; una verificación posterior añade una ronda nueva y NO borra las anteriores. Cada ronda DEBE indicar, por repositorio, la rama, el `tip` verificado (SHA de la punta de la rama) y el resultado (`ok`, `empty`, `unverifiable` o `exempt`), y DEBE declarar los huecos: cada repositorio `unverifiable` con su motivo (`branch_missing`, `base_missing` o `fork_point_undetermined`), y cada rama sin fusionar en la integración. Un repositorio `empty` o `exempt` se declara como tal y no cuenta como hueco. La ronda NO es limpia mientras tenga algún hueco. El verify NO DEBE ejecutar `fetch` ni rellenar un hueco por su cuenta.

La sección NO DEBE escribir líneas `verdict:`, NO DEBE modificar la línea `**Estado:**` del informe ni alterar el veredicto que lee la barrera de `multi-role-fan-out`, la sección «Cumplimiento técnico» ni la matriz ISO. El verify conjunto NO DEBE mutar código, tareas ni artefactos de planificación; solo escribe su informe.

**Petición explícita.** Cuando la persona pide la verificación conjunta de forma explícita, el orquestador DEBE atenderla siempre, aunque el resumen de `axiom branch scope` no sea `offerable` (ramas pendientes, repositorios `unverifiable`): se ejecuta sobre lo verificable y la ronda declara los huecos. Cuando todos los repositorios son `empty` o `exempt`, no se escribe ninguna ronda y la respuesta lo declara. Cuando el resultado de `axiom branch scope` difiere entre la presentación y la ejecución, la ejecución usa el resultado recalculado.

#### Scenario: Camino feliz, la ronda cubre la unión de los repositorios
- **DADO** un cambio con los repositorios de código `api` y `web`, ambos con commits propios fusionados en la integración
- **CUANDO** se ejecuta la verificación conjunta
- **ENTONCES** `verify-report.md` contiene una sección «Verificación conjunta» fechada con una entrada por `api` y por `web`
- **Y** cada entrada indica rama, `tip` y resultado

#### Scenario: Un commit ajeno excluido no entra en la verificación
- **DADO** un cambio en modo `release` y un commit ajeno `F` llegado a `develop` después de la bifurcación
- **CUANDO** se ejecuta la verificación conjunta
- **ENTONCES** el informe no contiene hallazgos ni ficheros procedentes de `F`
- **Y** si `axiom branch scope` lista commits excluidos o el aviso `foreign_commits_possible`, la sección los reproduce

#### Scenario: Rama borrada se declara como hueco
- **DADO** un repositorio `web` que `axiom branch scope` declara `unverifiable` con motivo `branch_missing`
- **CUANDO** se genera la ronda
- **ENTONCES** declara `web` como no verificado con ese motivo
- **Y** la ronda no es limpia por ese hueco
- **Y** no se ha ejecutado ningún `fetch`

#### Scenario: Petición explícita con huecos se ejecuta y los declara
- **DADO** un cambio con una rama sin fusionar y otro repositorio `unverifiable`
- **CUANDO** la persona pide de forma explícita la verificación conjunta
- **ENTONCES** se ejecuta sobre lo verificable y no se rechaza la petición
- **Y** la ronda declara la rama pendiente de fusionar y el repositorio no verificable

#### Scenario: Solo commits, sin árbol de trabajo
- **DADO** un cambio con un fichero modificado sin commitear y otro sin seguimiento en uno de los repositorios
- **CUANDO** se ejecuta la verificación conjunta
- **ENTONCES** la verificación no inspecciona esos ficheros
- **Y** el informe declara que verifica solo commits

#### Scenario: El informe existente con matriz ISO se conserva y se añade una ronda
- **DADO** un cambio ISO con un `verify-report.md` que ya contiene la matriz, «Cumplimiento técnico», una línea `**Estado:**` y una ronda anterior de «Verificación conjunta»
- **CUANDO** se ejecuta una verificación conjunta nueva
- **ENTONCES** se añade una ronda fechada nueva y la anterior se conserva
- **Y** la matriz, las demás secciones y la línea `**Estado:**` quedan intactas
- **Y** el informe no gana ninguna línea `verdict:`

#### Scenario: Sin commits propios no se escribe ronda
- **DADO** un cambio cuyos repositorios son todos `empty` o `exempt`
- **CUANDO** la persona pide la verificación conjunta de forma explícita
- **ENTONCES** no se añade ninguna sección «Verificación conjunta»
- **Y** la respuesta declara que no hay commits propios que verificar

### Requirement: La Verificación Conjunta Nunca Bloquea (REQ-28.42)

La verificación conjunta NO DEBE bloquear ni condicionar la compuerta `integration`, `axiom sdd gate record`, el archivado, `nextRecommended` ni las dependencias de `axiom sdd status`; su ausencia, su declinación y cualquier hallazgo o hueco del informe son informativos, igual que el resto de revisiones independientes (REQ-28.3). La evidencia de REQ-21.16 de `sdd-archive-lifecycle` no cambia. En un cambio `continuous` NO existe la compuerta `integration` y el orquestador NO DEBE añadir parada, pregunta ni oferta nueva por esta capacidad.

#### Scenario: La verificación conjunta no bloquea integration
- **DADO** una compuerta `integration` pendiente y ninguna verificación conjunta hecha
- **CUANDO** se registra `axiom sdd gate record --gate integration --decision approved` con su evidencia actual
- **ENTONCES** el registro se acepta
- **Y** `nextRecommended` y las dependencias de `axiom sdd status` son idénticos a los de un cambio con verificación conjunta hecha

#### Scenario: Un hallazgo de la verificación conjunta no bloquea
- **DADO** una verificación conjunta con un hallazgo crítico
- **CUANDO** se consulta `axiom sdd status` y se solicita el archivado
- **ENTONCES** `nextRecommended` y las dependencias no cambian
- **Y** el archivado no se rechaza por el informe, salvo la excepción de la matriz ISO de REQ-28.3

#### Scenario: En continuous no hay oferta ni parada nuevas
- **DADO** un cambio `continuous` con el rol `fullstack`
- **CUANDO** concluye el flujo
- **ENTONCES** el orquestador no presenta ninguna parada ni oferta de verificación conjunta
