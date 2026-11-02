# Delta para Cuestionario Pre-Vuelo de SDD, Modalidad de Avance y Asignación Obligatoria de Roles (paradas-tres-salidas)

> **Cambio:** `paradas-tres-salidas` · **Tipo:** delta sobre la capacidad viva `sdd-preflight-configuration`
> **Idioma:** Español (castellano peninsular); identificadores en inglés.

Cambios de este delta:

- REQ-21.5: el pre-vuelo ya no sella ni pregunta la política de relevos; solo la modalidad de avance (los roles siguen en REQ-21.6).
- REQ-26.9 (nuevo): `handoff_policy` queda deprecado. El flag `--handoff-policy` se acepta y se ignora, deja de ser obligatorio, no se escribe en los kickoffs nuevos y los kickoffs antiguos se siguen leyendo. La retirada completa del campo queda para un cambio posterior.

Nota no normativa: la sección «Propósito» de la capacidad menciona «la política de relevos (sin o con handoffs intermedios)»; queda desfasada y puede corregirse al archivar, porque un delta no modifica esa sección.

## MODIFIED Requirements

### Requirement: Bloqueo de Avance sin Modalidad Sellada (REQ-21.5)

Al entrar en el carril SDD, el orquestador DEBE bloquear la creación de `proposal.md` hasta que se defina explícitamente la modalidad de avance (`continuo` o `con paradas y reviews por bloque`). El orquestador DEBE persistir esa decisión de forma sellada para el cambio activo y NO DEBE volver a preguntarla mientras el cambio permanezca activo, salvo lo indicado en REQ-21.4. El orquestador NO DEBE preguntar ni exigir ninguna política de relevos: la entrega del trabajo a otra persona se decide en cada parada con las tres salidas (REQ-26.1 de `sdd-block-review-gates`), no en el pre-vuelo.
(Previously: «Bloqueo de Avance sin Modalidad y Política de Relevos Selladas»: el orquestador exigía además sellar la política de relevos (`sin handoffs intermedios` o `con handoff.md en cada parada`))

#### Scenario: Modalidad de avance sellada antes de crear la propuesta

- **DADO** que el usuario ha elegido el carril SDD
- **CUANDO** responde explícitamente la modalidad de avance
- **ENTONCES** el orquestador sella la decisión para el cambio
- **Y** procede a crear `proposal.md`
- **Y** no ha formulado ninguna pregunta sobre política de relevos

#### Scenario: Respuesta ambigua o ausente no crea la propuesta ni asume un valor por defecto

- **DADO** que el usuario no responde con una modalidad reconocible entre las opciones válidas
- **CUANDO** el orquestador evalúa la respuesta
- **ENTONCES** no crea `proposal.md`
- **Y** vuelve a presentar la pregunta pendiente hasta obtener una respuesta válida, sin asumir un valor por defecto no confirmado

#### Scenario: El sellado funciona sin política de relevos

- **DADO** un cambio SDD con la modalidad y los roles definidos
- **CUANDO** se ejecuta `axiom sdd kickoff seal` sin `--handoff-policy`
- **ENTONCES** el sellado se completa y se escribe `kickoff.yaml`

## ADDED Requirements

### Requirement: `handoff_policy` Deprecado: se Acepta, se Ignora y no se Escribe (REQ-26.9)

El campo `handoff_policy` de `kickoff.yaml` y el flag `--handoff-policy` de `axiom sdd kickoff seal` quedan deprecados y SIN efecto:

- `axiom sdd kickoff seal` NO DEBE exigir `--handoff-policy`. Si se pasa con `none` o `per_checkpoint`, DEBE aceptarse sin error, ignorarse y NO escribirse en `kickoff.yaml`. Un valor fuera de esos dos (supuesto A-6) sigue rechazándose.
- Un `kickoff.yaml` nuevo NO DEBE contener la clave `handoff_policy`; tampoco el retro-sello `seal --infer`.
- Un `kickoff.yaml` existente con `handoff_policy: none` o `per_checkpoint` DEBE leerse y validarse sin error; un `kickoff.yaml` sin la clave, también (la validación acepta el valor vacío).
- Ningún comportamiento DEBE depender del valor del campo: en particular, `per_checkpoint` NO DEBE hacer que `verify` consulte `handoff.md` ni que se genere ningún relevo intermedio.
- La salida de `kickoff seal` y `kickoff show` (texto y JSON) y el JSON de `axiom sdd status` (`governance.kickoff`) NO DEBEN emitir el campo cuando está vacío; cuando está presente en un kickoff antiguo, PUEDEN mostrarlo marcado como obsoleto.
- El texto de ayuda y los mensajes de error NO DEBEN sugerir `--handoff-policy`.

Supuesto A-6 (a confirmar en design): un valor fuera de `none|per_checkpoint` se rechaza como hoy, porque los orquestadores sincronizados solo pasan esos dos y el rechazo protege de erratas sin riesgo de compatibilidad.

#### Scenario: Sellar sin la política de relevos funciona

- **DADO** un cambio SDD sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style checkpointed` con los roles y sin `--handoff-policy`
- **ENTONCES** el sellado se completa
- **Y** `kickoff.yaml` no contiene `handoff_policy`

#### Scenario: Un orquestador antiguo que pasa el flag sigue funcionando

- **DADO** un orquestador con instrucciones sincronizadas antes de este cambio
- **CUANDO** ejecuta `axiom sdd kickoff seal ... --handoff-policy per_checkpoint`
- **ENTONCES** el sellado se completa sin error
- **Y** `kickoff.yaml` no contiene `handoff_policy` y el flag no tiene ningún efecto

#### Scenario: Un valor desconocido del flag se rechaza

- **DADO** una ejecución de `axiom sdd kickoff seal` con `--handoff-policy otra`
- **CUANDO** se validan los argumentos
- **ENTONCES** se rechaza como valor no válido y no se escribe `kickoff.yaml`

#### Scenario: Un kickoff antiguo con la clave se lee sin error

- **DADO** un `kickoff.yaml` existente con `handoff_policy: per_checkpoint`
- **CUANDO** se carga y se valida
- **ENTONCES** la carga tiene éxito
- **Y** el valor no altera ninguna proyección de `axiom sdd status`

#### Scenario: Un kickoff sin la clave valida

- **DADO** un `kickoff.yaml` sin `handoff_policy`
- **CUANDO** se valida
- **ENTONCES** la validación acepta el valor vacío

#### Scenario: per_checkpoint no hace que verify consulte handoff.md

- **DADO** un cambio de un solo rol sellado antiguamente con `handoff_policy: per_checkpoint` y sin `handoff.md`
- **CUANDO** se evalúa la dependencia `verify`
- **ENTONCES** es idéntica a la de un cambio sellado con `none`

#### Scenario: La salida de kickoff y status no emite el campo vacío

- **DADO** un cambio sellado sin `handoff_policy`
- **CUANDO** se ejecuta `axiom sdd kickoff show --json` y `axiom sdd status --json`
- **ENTONCES** ni `handoffPolicy` ni su equivalente aparecen en el JSON
- **Y** el texto de `kickoff seal` y `kickoff show` no menciona la política de relevos

#### Scenario: Ningún asset nombra el flag como obligatorio

- **DADO** el conjunto de assets del producto (orquestadores, contrato de status, convención de openspec, skills)
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** ninguno prescribe `--handoff-policy` como argumento de `kickoff seal`
- **Y** el mensaje de error que guía a sellar el kickoff no lo incluye

## RENAMED Requirements

### Requirement: Bloqueo de Avance sin Modalidad y Política de Relevos Selladas (REQ-21.5) → Bloqueo de Avance sin Modalidad Sellada (REQ-21.5)

(Reason: el pre-vuelo deja de sellar la política de relevos; el nombre del requisito debe reflejar que solo se sella la modalidad de avance.)
(Migration: referencias al requisito por su nombre en specs y docs se actualizan; el ID REQ-21.5 se conserva.)
