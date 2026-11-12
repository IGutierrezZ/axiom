# Delta para sdd-independent-review

> Esta delta no añade ningún requisito REQ-28.x. Declara la excepción del modo ISO a REQ-28.3 (la matriz de verify válida y con éxito sí condiciona el archivado), hace que REQ-28.4 remita a REQ-21.15 en lugar de repetir su condición, y matiza en REQ-28.6 cuándo un verify es limpio en un cambio ISO. Las revisiones siguen siendo informativas y ajenas a RDD; los hallazgos, el juicio y la revisión de cobertura no bloquean nada tampoco en ISO.
> **Supuesto visible para la compuerta `spec`:** la definición de «Verify limpio» del preámbulo de esta capacidad no es un requisito y no se modifica aquí; su matiz ISO consta en el texto de REQ-28.6.

## MODIFIED Requirements

### Requirement: Las Revisiones Son Informativas y No Bloquean (REQ-28.3)

Ni el verify por rol, ni el juicio, ni la revisión de cobertura DEBEN bloquear una fase, una puerta o el archivado. Un hallazgo, sea cual sea su severidad, NO DEBE alterar `nextRecommended`, las dependencias proyectadas por `axiom sdd status`, ni impedir `sdd-archive`. La decisión de actuar sobre un hallazgo es humana: en `checkpointed` se toma en la parada; en `continuous` el orquestador informa en su resumen final y no corrige por su cuenta, salvo las correcciones acotadas del juicio (REQ-28.6). El verify NO DEBE mutar código, tareas ni artefactos de planificación; solo escribe su informe. **Excepción declarada:** en un cambio sellado como ISO (REQ-29.1 de `sdd-iso-evidence` y REQ-29.13 de `sdd-lane-kickoff`), la matriz de verify del informe SÍ condiciona el archivado: `archive` se bloquea mientras la matriz esté ausente, sea inválida o tenga alguna fila fallida (REQ-29.11 de `sdd-archive-lifecycle`). Ese bloqueo procede de la validez de la matriz —evidencia declarada y contrastada con el árbol de trabajo— y nunca de un hallazgo de revisión: los hallazgos, el juicio `ESCALATED` y la revisión de cobertura siguen sin alterar `nextRecommended` ni las dependencias, también en un cambio ISO.
(Previously: ninguna revisión ni informe de verify bloqueaba el archivado en ningún caso; ahora un cambio ISO tiene la excepción de la matriz de verify)

#### Scenario: Un hallazgo crítico no bloquea el archivado

- **DADO** un verify de rol con un hallazgo de cumplimiento técnico de severidad crítica
- **CUANDO** se consulta `axiom sdd status` y se solicita el archivado
- **ENTONCES** `nextRecommended` y las dependencias son idénticas a las de un cambio sin ese hallazgo
- **Y** el archivado no se rechaza por el informe

#### Scenario: Un juicio ESCALATED no bloquea

- **DADO** un juicio del rol que termina con `JUDGMENT: ESCALATED`
- **CUANDO** el orquestador continúa el flujo
- **ENTONCES** el veredicto se presenta como informativo
- **Y** ninguna fase ni puerta queda bloqueada por él

#### Scenario: El verify no modifica el trabajo del rol

- **DADO** un verify de rol que detecta un incumplimiento
- **CUANDO** termina
- **ENTONCES** no ha modificado código, `tasks*.md` ni `spec.md`/`design.md`
- **Y** el único fichero escrito es su informe

#### Scenario: Una revisión de cobertura con huecos no impide iniciar la fase siguiente

- **DADO** un informe de cobertura con huecos abiertos en un cambio `continuous`
- **CUANDO** concluye la revisión
- **ENTONCES** el orquestador continúa con la fase siguiente
- **Y** los huecos quedan en el informe y en el resumen final

#### Scenario: En un cambio ISO una matriz inválida bloquea el archivado sin ningún hallazgo

- **DADO** un cambio ISO cuyo informe de verify no contiene ningún hallazgo y cuya matriz tiene un escenario sin fila
- **CUANDO** se solicita el archivado
- **ENTONCES** `archive` se bloquea con el motivo tipado de matriz (REQ-29.11 de `sdd-archive-lifecycle`)
- **Y** el motivo no cita ningún hallazgo de revisión

#### Scenario: En un cambio ISO un hallazgo crítico con la matriz válida no bloquea

- **DADO** un cambio ISO con una matriz válida y con éxito y un verify con un hallazgo de cumplimiento técnico de severidad crítica
- **CUANDO** se consulta `axiom sdd status` y se solicita el archivado
- **ENTONCES** `nextRecommended` y las dependencias son idénticas a las de un cambio ISO sin ese hallazgo
- **Y** el archivado no se rechaza por el informe

#### Scenario: En un cambio ISO un juicio ESCALATED no bloquea

- **DADO** un cambio ISO con una matriz válida y con éxito y un juicio que termina con `JUDGMENT: ESCALATED`
- **CUANDO** el orquestador continúa el flujo
- **ENTONCES** el veredicto se presenta como informativo
- **Y** `archive` no queda bloqueado por él

#### Scenario: Fuera de ISO la matriz no condiciona el archivado

- **DADO** un cambio no ISO cuyo informe de verify no contiene matriz
- **CUANDO** se solicita el archivado
- **ENTONCES** el archivado no se rechaza por el informe

---

### Requirement: Verify por Rol con Sección de Cumplimiento Técnico (REQ-28.4)

El `verify` de un rol DEBE inspeccionar únicamente el diff de ese rol y escribir su informe en el `verify_file` que el kickoff asigna a ese rol. Además de la verificación funcional existente (requisitos, pruebas, compilación), el informe DEBE incluir una sección **Cumplimiento técnico** que contraste el diff con las reglas aplicables (REQ-28.5). Cuando no hay hallazgos, la sección DEBE existir y declararlo, con la lista de fuentes consultadas. El verify global multirrol, que se evalúa cuando REQ-21.15 lo habilita, NO cambia, y la semántica del veredicto que lee la barrera de `multi-role-fan-out` tampoco: la sección nueva es contenido adicional del informe. En un cambio ISO, el informe consolidado (o el del rol único `fullstack`) incluye además la matriz de verify (REQ-29.6 de `sdd-iso-evidence`) como otra sección adicional, que NO escribe líneas `verdict:` y NO altera la sección «Cumplimiento técnico»; los informes por rol de un cambio multirrol NO necesitan matriz. En un cambio con el rol único `fullstack`, el verify del rol y el verify del cambio son el mismo informe y NO se ejecutan dos veces sin motivo. Nota: el verify por rol es un diagnóstico del rol que se ofrece antes de aprobar `role-apply`; NO es la fase `verify` global de REQ-21.15, que sigue evaluándose cuando REQ-21.15 la habilita.
(Previously: el verify global multirrol se describía como evaluado «tras todos los `role-apply` aprobados», lo que repetía la condición de REQ-21.15 y divergía de ella cuando un rol `deferred` u `optional` seguía pendiente; ahora remite a REQ-21.15 sin repetirla, y se añade la matriz ISO como sección adicional)

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

#### Scenario: El verify global multirrol no cambia

- **DADO** un cambio con los roles `core` y `web`, ambos `blocking`, y todos los `role-apply` aprobados
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** se habilita conforme a REQ-21.15, sin ninguna condición añadida por este requisito
- **Y** el veredicto que lee la barrera de sincronización se obtiene como antes, sin tener en cuenta la sección nueva

#### Scenario: El verify global se evalúa con REQ-21.15 aunque un rol deferred siga pendiente

- **DADO** un cambio con `core` (`blocking`) aprobado y `e2e` (`deferred`) sin aprobar
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** el resultado es el que REQ-21.15 determina (no bloqueada)
- **Y** este requisito no añade ninguna exigencia sobre `e2e`

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

---

### Requirement: El Juicio Solo se Ofrece o Lanza Tras un Verify Limpio del Rol (REQ-28.6)

El orquestador DEBE ofrecer o lanzar el juicio de un rol únicamente cuando el verify de ese rol, en su estado vigente, es limpio. Con un verify no limpio, el orquestador NO DEBE lanzarlo por iniciativa propia, DEBE informar de que el juicio no se lanza por no ser limpio el verify, y DEBE aceptar una petición explícita del usuario de lanzarlo igualmente (supuesto A-2: la petición explícita de un usuario informado prevalece). Con la preferencia `verify-judgment` y un verify no limpio, el juicio simplemente no se lanza y el flujo continúa. En un cambio ISO cuyo informe vigente del rol contiene la matriz de verify (el rol único `fullstack`), una matriz ausente, inválida o con alguna fila fallida (REQ-29.6 a REQ-29.8 de `sdd-iso-evidence`) cuenta como comprobación obligatoria fallida: el verify no es limpio y el juicio no se lanza por iniciativa propia. En un cambio multirrol, la matriz vive en el informe consolidado y no afecta a si el verify de un rol es limpio. Antes de lanzar el juicio, el orquestador DEBE informar del coste previsto (dos jueces ciegos y hasta dos rondas con el actor de arreglos) en una línea; en `continuous` la información no detiene el flujo.
(Previously: «limpio» no consideraba la matriz de verify de un cambio ISO)

#### Scenario: Verify limpio habilita el juicio

- **DADO** la preferencia `verify-judgment` y un verify del rol sin hallazgos críticos ni comprobaciones fallidas
- **CUANDO** termina el verify
- **ENTONCES** el orquestador informa del coste y lanza el juicio sobre el diff del rol

#### Scenario: Verify con hallazgo crítico no lanza el juicio

- **DADO** la preferencia `verify-judgment` y un verify con un hallazgo de severidad crítica
- **CUANDO** termina el verify
- **ENTONCES** el orquestador no lanza el juicio
- **Y** informa de que no lo lanza porque el verify no es limpio

#### Scenario: Un verify con solo avisos y sugerencias es limpio

- **DADO** un verify cuyo único contenido de hallazgos son avisos y sugerencias
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** lo trata como limpio y lo lanza

#### Scenario: Un verify parcial por herramienta no disponible no es limpio

- **DADO** un verify en el que las pruebas no pudieron ejecutarse
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** no lo lanza por iniciativa propia
- **Y** informa del motivo

#### Scenario: Petición explícita con verify no limpio

- **DADO** un verify no limpio y la petición explícita del usuario de lanzar el juicio
- **CUANDO** el orquestador la atiende
- **ENTONCES** lanza el juicio avisando de que el verify no era limpio

#### Scenario: Sin verify previo no hay juicio por iniciativa del orquestador

- **DADO** una preferencia que no incluye verify
- **CUANDO** concluye el `apply` del rol
- **ENTONCES** el orquestador no lanza el juicio por iniciativa propia

#### Scenario: En ISO una matriz inválida hace que el verify no sea limpio

- **DADO** un cambio ISO con el rol único `fullstack`, la preferencia `verify-judgment` y un verify sin hallazgos cuya matriz tiene un escenario sin fila
- **CUANDO** termina el verify
- **ENTONCES** el verify no es limpio
- **Y** el orquestador no lanza el juicio e informa del motivo

#### Scenario: En ISO una fila fallida hace que el verify no sea limpio

- **DADO** un cambio ISO con el rol único `fullstack` y una matriz válida con una fila de resultado `fail`
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** no lo lanza por iniciativa propia
- **Y** informa de que la matriz tiene una fila fallida

#### Scenario: En ISO una matriz válida y con éxito no impide un verify limpio

- **DADO** un cambio ISO con el rol único `fullstack`, una matriz válida y con éxito y un verify sin hallazgos críticos ni comprobaciones fallidas
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** lo trata como limpio

#### Scenario: Fuera de ISO la matriz no interviene en «limpio»

- **DADO** un cambio no ISO sin matriz y un verify sin hallazgos críticos ni comprobaciones fallidas
- **CUANDO** el orquestador decide si lanza el juicio
- **ENTONCES** lo trata como limpio
