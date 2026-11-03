<!-- Especificación Viva generada a partir de '2026-09-22-inc-21-upfront-flow-governance' -->

# Especificación de Requerimientos: Cuestionario Pre-Vuelo de SDD, Modalidad de Avance y Asignación Obligatoria de Roles (INC-21)

## Propósito

Definir de forma rigurosa, ejecutable y verificable los requerimientos funcionales y escenarios BDD para el cuestionario de pre-vuelo que sella, antes de crear la propuesta SDD, la modalidad de avance del cambio (continua o con paradas y reviews por bloque) y la asignación de roles, incluyendo la asignación obligatoria del rol `fullstack` cuando no existe subdivisión de roles especializados.

---

## 1. Capacidad: `sdd-preflight-configuration`

El cuestionario de pre-vuelo sella, antes de crear la propuesta, la modalidad de avance del cambio SDD, su política de relevos y la asignación de roles.

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

### Requirement: Asignación Obligatoria del Rol `fullstack` sin Subdivisión Especializada (REQ-21.6)

Si, al concluir el cuestionario de pre-vuelo, ni el usuario ni el diseño previsto subdividen el trabajo en roles especializados (por ejemplo `core`, `web`, `qa`), el orquestador DEBE asignar obligatoriamente el rol único `fullstack`, con política de compuerta `blocking`, fichero de tareas `tasks.md` y fichero de verificación `verify-report.md`. El orquestador NO DEBE dejar un cambio SDD sin al menos un rol informado en su configuración de kickoff.

#### Scenario: Sin roles declarados, se asigna fullstack por defecto

- **DADO** un cuestionario de pre-vuelo en el que el usuario no declara ningún rol especializado
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador asigna el rol único `fullstack` con política `blocking`, `tasks.md` y `verify-report.md`
- **Y** el cambio queda con exactamente un rol informado

#### Scenario: Un único rol especializado declarado no se combina con fullstack

- **DADO** un cuestionario de pre-vuelo en el que el usuario declara exactamente un rol especializado (por ejemplo, únicamente `core`)
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador respeta ese único rol declarado como el rol activo del cambio
- **Y** no asigna adicionalmente el rol `fullstack` junto al rol ya declarado

#### Scenario: Varios roles especializados declarados tampoco activan fullstack

- **DADO** un cuestionario de pre-vuelo en el que el usuario declara varios roles especializados (por ejemplo `core`, `web` y `qa`)
- **CUANDO** concluye el cuestionario
- **ENTONCES** el orquestador no asigna el rol `fullstack`
- **Y** cada rol declarado conserva su propia política de compuerta
### Requirement: `handoff_policy` Deprecado: se Acepta, se Ignora y se Fija a `none` (REQ-26.9)

El campo `handoff_policy` de `kickoff.yaml` y el flag `--handoff-policy` de `axiom sdd kickoff seal` quedan deprecados y SIN efecto:

- `axiom sdd kickoff seal` NO DEBE exigir `--handoff-policy`. Si se pasa con `none` o `per_checkpoint`, DEBE aceptarse sin error e ignorarse: su valor NO DEBE llegar a `kickoff.yaml`. Un valor fuera de esos dos (supuesto A-6) sigue rechazándose.
- Un `kickoff.yaml` nuevo, del sellado o del retro-sello `seal --infer`, DEBE contener `handoff_policy: none`, con independencia del flag. Es un valor de compatibilidad: los binarios anteriores a este cambio rechazan un kickoff sin la clave, y con `none` su `verify` no consulta `handoff.md`. La retirada de la clave queda para un cambio posterior.
- Un `kickoff.yaml` existente con `handoff_policy: none` o `per_checkpoint` DEBE leerse y validarse sin error; un `kickoff.yaml` sin la clave, también (la validación acepta el valor vacío).
- Ningún comportamiento DEBE depender del valor del campo: en particular, `per_checkpoint` NO DEBE hacer que `verify` consulte `handoff.md` ni que se genere ningún relevo intermedio.
- El texto de `kickoff seal` y `kickoff show` NO DEBE mencionar la política de relevos. El JSON de `kickoff seal`, `kickoff show` y `axiom sdd status` (`governance.kickoff`) PUEDE emitir el valor leído y NO DEBE emitir el campo cuando está vacío.
- El texto de ayuda y los mensajes de error NO DEBEN sugerir `--handoff-policy`.

Supuesto A-6 (a confirmar en design): un valor fuera de `none|per_checkpoint` se rechaza como hoy, porque los orquestadores sincronizados solo pasan esos dos y el rechazo protege de erratas sin riesgo de compatibilidad.

#### Scenario: Sellar sin la política de relevos funciona

- **DADO** un cambio SDD sin `kickoff.yaml`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --execution-style checkpointed` con los roles y sin `--handoff-policy`
- **ENTONCES** el sellado se completa
- **Y** `kickoff.yaml` contiene `handoff_policy: none`

#### Scenario: Un orquestador antiguo que pasa el flag sigue funcionando

- **DADO** un orquestador con instrucciones sincronizadas antes de este cambio
- **CUANDO** ejecuta `axiom sdd kickoff seal ... --handoff-policy per_checkpoint`
- **ENTONCES** el sellado se completa sin error
- **Y** `kickoff.yaml` contiene `handoff_policy: none`, no `per_checkpoint`, y el flag no tiene ningún efecto

#### Scenario: Un binario anterior lee un kickoff nuevo

- **DADO** un `kickoff.yaml` sellado por esta versión, con o sin `seal --infer`
- **CUANDO** un binario anterior a este cambio lo carga y lo valida
- **ENTONCES** la validación tiene éxito, porque la clave está presente con un valor que ese binario reconoce
- **Y** su `verify` no consulta `handoff.md`, porque el valor es `none`

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

- **DADO** un `kickoff.yaml` sin la clave `handoff_policy`
- **CUANDO** se ejecuta `axiom sdd kickoff show --json` y `axiom sdd status --json`
- **ENTONCES** ni `handoffPolicy` ni su equivalente aparecen en el JSON

#### Scenario: El texto de kickoff no menciona la política de relevos

- **DADO** un cambio sellado por esta versión, con `handoff_policy: none`
- **CUANDO** se ejecuta `axiom sdd kickoff seal` o `axiom sdd kickoff show` sin `--json`
- **ENTONCES** el texto no menciona la política de relevos

#### Scenario: Ningún asset nombra el flag como obligatorio

- **DADO** el conjunto de assets del producto (orquestadores, contrato de status, convención de openspec, skills)
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** ninguno prescribe `--handoff-policy` como argumento de `kickoff seal`
- **Y** el mensaje de error que guía a sellar el kickoff no lo incluye

