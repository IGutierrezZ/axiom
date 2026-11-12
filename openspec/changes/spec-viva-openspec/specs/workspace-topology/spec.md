# Delta para workspace-topology

> Parte del estado vigente: la estructura del esquema, las topologías, el bloque `git:`, `repositories:` y `workflow:` y la retirada de `monorepo-decoupled` NO cambian. Esta delta retira el bloque `baseline:` del esquema (se rechaza con error tipado) y actualiza `axiom workspace validate`.

## ADDED Requirements

### Requirement: Un Bloque `baseline:` Residual se Rechaza con Error Tipado y Guía de Migración (REQ-30.21)

`axiom.yaml` NO DEBE admitir un bloque de primer nivel `baseline:`: la spec viva es únicamente `openspec/specs/`. Un `baseline:` residual NO DEBE ignorarse en silencio ni degradar a «sin baseline». DEBE producir una causa tipada que nombre el bloque `baseline:` y la continuación («retira `baseline:` y migra los requisitos con `axiom archive adopt`») en: (a) la carga de `axiom.yaml`; (b) `axiom workspace validate`, que DEBE finalizar con código de salida `1`; (c) `axiom sdd status`, que DEBE marcar el archivado como bloqueado (`dependencies.archive` en `blocked`) con esa causa sin bloquear las dependencias de las fases de planificación ni la resolución del planning home; y (d) el archivado, que DEBE detenerse sin escribir en `openspec/specs/` ni mover el cambio. El status NO DEBE proyectar ninguna clave `baseline`. Un `axiom.yaml` sin `baseline:` NO DEBE verse afectado.

#### Scenario: `axiom workspace validate` rechaza el bloque
- **DADO** un `axiom.yaml` válido que además declara `baseline:` con un documento
- **CUANDO** se ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el bloque `baseline:` se retiró y la continuación `axiom archive adopt`
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: El status bloquea solo el archivado
- **DADO** el mismo `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `dependencies.archive` es `blocked` con una causa que nombra `baseline:` y la continuación
- **Y** las dependencias de las fases de planificación no son `blocked` por esa causa
- **Y** el JSON no contiene la clave `baseline`

#### Scenario: El archivado se detiene sin escribir una segunda spec viva
- **DADO** el mismo `axiom.yaml` y un cambio listo para archivar
- **CUANDO** se solicita el archivado
- **ENTONCES** archive se detiene con el motivo tipado
- **Y** `openspec/specs/` no se modifica y el cambio permanece en `openspec/changes/`

#### Scenario: Un `axiom.yaml` sin `baseline:` no se ve afectado
- **DADO** un `axiom.yaml` válido sin `baseline:`
- **CUANDO** se ejecutan `axiom workspace validate` y `axiom sdd status --json`
- **ENTONCES** no se reporta ninguna causa por `baseline:`

#### Scenario: La carga produce la causa tipada
- **DADO** un `axiom.yaml` con `baseline: {}` vacío
- **CUANDO** se carga la configuración
- **ENTONCES** la carga reporta la causa tipada que nombra `baseline:`
- **Y** no se degrada a «sin baseline»

## MODIFIED Requirements

### Requirement: Subcomando axiom workspace validate (REQ-1.2)

El binario `axiom` DEBE proporcionar el subcomando `axiom workspace validate`, el cual evalúa el archivo `axiom.yaml` del directorio actual (o de la ruta especificada mediante `--path`) y verifica la conformidad de la topología y de los repositorios asociados a los roles. Además DEBE validar los bloques opcionales `technical_context`, `git` y `repositories` de `axiom.yaml`: la sintaxis y la contención de rutas se reportan como errores y las rutas de contexto técnico inexistentes como advertencias que no alteran el resultado. También DEBE reportar como error que un contexto técnico mantenido se declare con un almacén de artefactos sin ficheros (`engram` o `none`, REQ-30.19), DEBE rechazar un bloque `baseline:` residual con el error tipado de REQ-30.21 y DEBE rechazar la topología retirada `monorepo-decoupled` con el mensaje accionable de REQ-25.25.
(Previously: validaba el bloque `baseline` y reportaba el almacén sin ficheros también para una baseline del proyecto; ahora un `baseline:` se rechaza)

#### Scenario: Validación de espacio de trabajo conforme
- **DADO** un directorio con un archivo `axiom.yaml` válido y las carpetas de repositorios existentes
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida reporta un estado `COMPLIANT` indicando la topología detectada y los roles verificados
- **Y** el código de salida del proceso es `0`

#### Scenario: Validación con ruta explícita mediante --path
- **DADO** un espacio de trabajo ubicado en una ruta externa `/tmp/test-workspace`
- **CUANDO** el usuario ejecuta `axiom workspace validate --path /tmp/test-workspace`
- **ENTONCES** la validación se ejecuta sobre el directorio especificado
- **Y** reporta el resultado correspondiente a esa ubicación

#### Scenario: Espacio de trabajo no conforme finaliza con error
- **DADO** un espacio de trabajo donde faltan repositorios obligatorios o el archivo de configuración es inválido
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla de forma explícita en español los motivos de no conformidad
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Bloque `baseline:` residual finaliza con error
- **DADO** un `axiom.yaml` que declara `baseline:`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el bloque se retiró y la continuación `axiom archive adopt`
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Ruta de contexto técnico inexistente solo avisa
- **DADO** un `axiom.yaml` conforme cuyo `technical_context.maintained.paths` incluye una ruta que no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida incluye una advertencia que nombra la ruta y reporta `COMPLIANT`
- **Y** el código de salida del proceso es `0`

#### Scenario: Declaración incompatible con el almacén de artefactos finaliza con error
- **DADO** un `axiom.yaml` con `technical_context.maintained` válido y un `openspec/config.yaml` con `artifact_store: engram`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español que el contexto mantenido exige un almacén de ficheros, y la continuación
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Bloque `git:` o `repositories:` inválido finaliza con error
- **DADO** un `axiom.yaml` con `git.branch_prefix: "a b//"` o con una entrada de `repositories:` cuya ruta no existe
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida detalla en español la clave o la ruta y la regla incumplida
- **Y** el proceso finaliza con código de salida `1`

#### Scenario: Topología retirada finaliza con error accionable
- **DADO** un `axiom.yaml` con `workspace.topology: monorepo-decoupled`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida indica en español que la topología fue retirada y que debe sustituirse por `multirepo`
- **Y** el proceso finaliza con código de salida `1`

---

### Requirement: Estructura del Esquema axiom.yaml (REQ-2.1)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`)

Además DEBE aceptar los bloques opcionales de primer nivel `technical_context` (REQ-24.26), `git` (REQ-25.26), `repositories` (REQ-25.27) y `workflow` (REQ-27.1): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas y valores). Un bloque `baseline` NO es un bloque admitido: se rechaza con el error tipado de REQ-30.21. Las claves desconocidas fuera de esos bloques se siguen ignorando.
(Previously: aceptaba también el bloque opcional `baseline` (REQ-24.13) y validaba sus perfiles; la baseline ajena se retira)

#### Scenario: El bloque `workflow` es opcional y se valida cuando está presente

- **DADO** un `axiom.yaml` con los campos obligatorios y, en una variante, además `workflow: {lane: ask}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` sobre ambas variantes
- **ENTONCES** las dos son `COMPLIANT`
- **Y** con `workflow: {lane: x}` la validación falla nombrando `workflow.lane`

#### Scenario: El bloque `baseline` ya no es un bloque admitido

- **DADO** un `axiom.yaml` con los campos obligatorios y un bloque `baseline:`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la validación falla con el error tipado de REQ-30.21 y no lo trata como clave desconocida ignorable

---

### Requirement: Bloque `workflow:` Opcional de `axiom.yaml` con el Carril del Proyecto (REQ-27.1)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `workflow:` con la clave opcional `lane`, cuyos valores válidos son `odd`, `sdd` y `ask`. Sin el bloque, o sin la clave, el carril efectivo del proyecto vale `sdd`. El bloque DEBE decodificarse de forma estricta por lista cerrada de claves, con el mismo patrón que `git:` (REQ-25.26): una clave desconocida dentro de `workflow:` y un valor de `lane` fuera de los tres admitidos DEBEN reportarse como error accionable que nombre la clave (`workflow.lane` o la clave desconocida) y los valores admitidos, tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. Un `workflow:` que no sea un mapa también DEBE reportarse como error. Las claves desconocidas de primer nivel fuera de los bloques reconocidos se siguen ignorando (con la excepción de `baseline`, que se rechaza con el error tipado de REQ-30.21): un binario anterior a este cambio DEBE cargar un `axiom.yaml` con el bloque `workflow:` sin error y sin interpretarlo. El carril es política del proyecto y se versiona; la elección del modo de trabajo personal NO DEBE declararse en `axiom.yaml`. El valor por defecto `sdd` rige en `monorepo-embedded` y en `multirepo`; en `multirepo` el carril `odd` sigue estando permitido.
(Previously: las claves desconocidas de primer nivel se ignoraban citando REQ-24.17 de `spec-baseline-declaration`, que se retira; `baseline` deja de ser un bloque reconocido y se rechaza)

#### Scenario: Sin bloque el carril efectivo es `sdd`

- **DADO** un `axiom.yaml` sin `workflow:`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `sdd` con origen por defecto

#### Scenario: Bloque con `lane: odd`

- **DADO** un `axiom.yaml` con `workflow: {lane: odd}`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `odd` con origen declarado

#### Scenario: Bloque con `lane: ask`

- **DADO** un `axiom.yaml` con `workflow: {lane: ask}`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `ask` con origen declarado

#### Scenario: Bloque vacío completa con el valor por defecto

- **DADO** un `axiom.yaml` con `workflow: {}`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** el carril efectivo es `sdd` y no se reporta ningún error

#### Scenario: Valor desconocido de `lane`

- **DADO** un `axiom.yaml` con `workflow: {lane: agil}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `workflow.lane`, el valor `agil` y los valores admitidos `odd`, `sdd` y `ask`
- **Y** el código de salida es `1`

#### Scenario: Clave desconocida dentro de `workflow:`

- **DADO** un `axiom.yaml` con `workflow: {lane: sdd, publish: true}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `publish` como desconocida
- **Y** el código de salida es `1`

#### Scenario: `workflow:` que no es un mapa

- **DADO** un `axiom.yaml` con `workflow: sdd`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `workflow` y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: Un binario anterior ignora el bloque

- **DADO** un `axiom.yaml` válido con `workflow: {lane: ask}`
- **CUANDO** lo carga el decodificador previo a este cambio (el que ignora las claves desconocidas de primer nivel)
- **ENTONCES** la carga termina sin error
- **Y** el resto de la configuración se lee igual que sin el bloque

#### Scenario: Multirepo admite `odd`

- **DADO** un `axiom.yaml` de topología `multirepo` con `workflow: {lane: odd}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT`
- **Y** el código de salida es `0`

#### Scenario: La plantilla y el manual documentan el bloque

- **DADO** la plantilla `axiom.example.yaml` y el manual de inicio
- **CUANDO** se inspeccionan (prueba de pin)
- **ENTONCES** ambos documentan `workflow.lane`, sus tres valores y el valor por defecto `sdd`

