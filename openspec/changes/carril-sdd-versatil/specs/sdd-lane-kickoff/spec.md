# Delta para sdd-lane-kickoff

## ADDED Requirements

### Requirement: `kickoff seal` Prepara la Carpeta del Cambio si No Existe (REQ-29.12)

`axiom sdd kickoff seal` DEBE dejar preparada la carpeta del cambio en el planning home (`changes/<cambio>/`) cuando no existe, en lugar de exigir que el orquestador o la persona la cree antes. La carpeta se crea después de superar la guarda de rama (REQ-25.38): ante `wrong_branch` el comando se niega sin crear la carpeta ni escribir `kickoff.yaml`. Si la carpeta ya existe, el comando NO DEBE modificar ni borrar ninguno de sus ficheros. Preparar la carpeta NO crea ningún otro artefacto (`proposal.md`, specs, `design.md` ni `tasks.md`): solo la carpeta y `kickoff.yaml`. Un sellado que se rechaza por cualquier otro motivo (argumentos inválidos, declaración `sdd:` inválida) NO DEBE dejar la carpeta creada. El retro-sellado `seal --infer` (REQ-21.4) conserva su comportamiento. El comando DEBE ser idempotente: repetirlo no cambia la carpeta ni su contenido. El orquestador y la skill de kickoff NO DEBEN pedir que se cree a mano la carpeta antes de sellar.

#### Scenario: Sellar un cambio sin carpeta la crea y escribe `kickoff.yaml`

- **DADO** un cambio `x` cuya carpeta `changes/x/` no existe y el repositorio desplegado en la rama que corresponde
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** existe `changes/x/kickoff.yaml`
- **Y** el comando termina con código de salida `0`
- **Y** no existe ningún otro fichero en `changes/x/`

#### Scenario: Sellar en la rama equivocada no crea la carpeta

- **DADO** un monorepo con la política por defecto, el repositorio desplegado en `main` y el cambio `x` sin carpeta
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza con el código tipado `wrong_branch`
- **Y** no se crea `changes/x/`

#### Scenario: Un sellado rechazado por otro motivo no deja la carpeta

- **DADO** un cambio `x` sin carpeta y un `axiom.yaml` con `sdd:` inválido
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza nombrando la clave o el valor inválido
- **Y** no se crea `changes/x/`

#### Scenario: Una carpeta existente se respeta

- **DADO** un cambio `x` con la carpeta ya creada y un `proposal.md` dentro
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** `proposal.md` permanece sin modificar
- **Y** se escribe `kickoff.yaml` junto a él

#### Scenario: Repetir el sellado es idempotente respecto de la carpeta

- **DADO** un cambio ya sellado cuya carpeta preparó el propio comando
- **CUANDO** se repite el comando con los mismos argumentos
- **ENTONCES** la carpeta y sus ficheros no cambian

#### Scenario: La skill de kickoff no pide crear la carpeta a mano

- **DADO** el texto de la skill y de las plantillas del orquestador que describen el sellado
- **CUANDO** se inspeccionan (pin de texto)
- **ENTONCES** no indican crear la carpeta del cambio antes de ejecutar `axiom sdd kickoff seal`

---

### Requirement: Instantánea de `sdd.iso` en `kickoff.yaml`, Escrita una Sola Vez por `seal` (REQ-29.13)

`axiom sdd kickoff seal` DEBE copiar en `kickoff.yaml`, dentro de la misma única escritura de sellado y con la semántica de sellado único de REQ-25.36, el valor que el proyecto declara en el bloque `sdd:` de `axiom.yaml` para la declaración ISO (clave `iso`, REQ-29.1 de `sdd-iso-evidence`). Es una instantánea, igual que la de política git (REQ-25.36) y que el campo `docs_publication` (REQ-27.4): el valor sigue siendo del proyecto y nadie elige ISO por cambio. Las reglas ISO de ESE cambio (REQ-29.3 a REQ-29.11) DEBEN leerse de su instantánea y NO de `axiom.yaml` en vivo; la declaración de `axiom.yaml` solo es la fuente en el momento de sellar. Un proyecto que no declara `iso`, o lo declara desactivado, sella un cambio no ISO. La serialización DEBE omitir los campos vacíos (`omitempty`), la lectura de `kickoff.yaml` DEBE tolerar su ausencia y un binario anterior DEBE ignorarlo sin error. Un cambio sellado antes de esta versión, sin instantánea de `sdd:`, DEBE contar como no ISO, que es el comportamiento actual. Cambiar `sdd:` en `axiom.yaml` solo afecta a los cambios sellados después: un cambio ya sellado conserva su valor hasta que se archive. `axiom sdd status` PUEDE avisar con una nota no bloqueante de la discrepancia entre la instantánea del cambio y la declaración vigente del proyecto, y NO DEBE bloquear por ella. El retrosellado `axiom sdd kickoff seal --infer` de un cambio preexistente DEBE congelar lo que ya rige ese cambio, como exige REQ-21.4: `iso: false`, sin leer la declaración `sdd:` de `axiom.yaml`. Así ningún cambio en curso pasa a ISO por retrosellarse. Registrar `iso` en el retrosellado NO DEBE contar como la instantánea de política git de REQ-25.36: REQ-25.37 no cambia y el cambio retro-sellado sigue exento de las guardas de rama. Un `sdd:` inválido DEBE rechazar el sellado sin escribir nada (REQ-29.1): nunca se sella «no ISO» por defecto ante una declaración inválida. Un segundo `seal` NO DEBE reescribir la instantánea (REQ-25.36).

#### Scenario: El sellado copia `iso` en `kickoff.yaml`

- **DADO** un `axiom.yaml` con `sdd: {iso: true}`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...` desde la rama del cambio
- **ENTONCES** `kickoff.yaml` contiene `iso` activo junto a la instantánea de política git y en la misma escritura
- **Y** el proyecto sigue siendo la fuente: no existe ninguna opción por cambio para elegir otro valor

#### Scenario: Sin declaración se sella un cambio no ISO

- **DADO** un `axiom.yaml` sin `sdd:`
- **CUANDO** se sella un cambio
- **ENTONCES** `kickoff.yaml` no contiene `iso` activo
- **Y** `axiom sdd status` proyecta el cambio como no ISO

#### Scenario: Activar `iso` a mitad no exige la matriz a un cambio ya sellado

- **DADO** un cambio `x` sellado sin `iso` y después el proyecto activa `iso` en `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status x --json` en la fase `archive` con un `verify-report.md` libre
- **ENTONCES** no se proyecta ningún motivo de matriz, cobertura ni identidad ISO para `x`
- **Y** un cambio sellado después sí es ISO

#### Scenario: Desactivar `iso` a mitad sigue exigiendo la matriz a un cambio ya sellado

- **DADO** un cambio `x` sellado con `iso` y después el proyecto retira `iso` de `axiom.yaml`
- **CUANDO** se ejecuta `axiom sdd status x --json` en la fase `archive` con una matriz inválida
- **ENTONCES** el archivado sigue bloqueado con el motivo tipado de matriz (REQ-29.11)

#### Scenario: Un cambio sellado antes de esta versión cuenta como no ISO

- **DADO** un `kickoff.yaml` sin instantánea de `sdd:` y un `axiom.yaml` actual con `sdd: {iso: true}`
- **CUANDO** se ejecutan `axiom sdd status` y `axiom sdd continue` sobre ese cambio
- **ENTONCES** el cambio se comporta como no ISO, sin errores
- **Y** no se proyecta ningún bloqueo de las reglas ISO

#### Scenario: La discrepancia entre la instantánea y el proyecto es una nota no bloqueante

- **DADO** un cambio sellado con `iso` y un `axiom.yaml` que ya no declara `iso`
- **CUANDO** se ejecuta `axiom sdd status <cambio> --json`
- **ENTONCES** el status incluye una nota que nombra la clave, el valor sellado y el valor vigente del proyecto
- **Y** ninguna dependencia queda bloqueada por esa nota

#### Scenario: Un segundo `seal` no reescribe la instantánea

- **DADO** un cambio sellado sin `iso` y un `axiom.yaml` que ahora declara `iso`
- **CUANDO** se ejecuta de nuevo `kickoff seal` desde la rama del cambio
- **ENTONCES** termina con éxito indicando «Kickoff ya estaba sellado»
- **Y** `kickoff.yaml` conserva los valores de la primera escritura

#### Scenario: El retrosellado congela `iso: false`

- **DADO** un cambio preexistente con artefactos, sin kickoff sellado, y un `axiom.yaml` con `sdd: {iso: true}`
- **CUANDO** se ejecuta `axiom sdd kickoff seal --infer --change x`
- **ENTONCES** el cambio queda sellado con `iso: false`, sin leer la declaración `sdd:`
- **Y** `axiom sdd status x --json` lo proyecta como no ISO
- **Y** el retrosellado no escribe la instantánea de política git (REQ-25.37) ni cambia el rol que ya infería

#### Scenario: El retrosellado sigue exento de las guardas de rama

- **DADO** el cambio retro-sellado del escenario anterior y el repositorio desplegado en la rama de integración
- **CUANDO** se consulta el veredicto de rama
- **ENTONCES** es `not_applicable`
- **Y** el campo `iso` registrado no se interpreta como instantánea de política git

#### Scenario: Un `sdd:` inválido rechaza el sellado

- **DADO** un `axiom.yaml` con `sdd:` inválido
- **CUANDO** se ejecuta `axiom sdd kickoff seal --change x ...`
- **ENTONCES** se rechaza nombrando la clave o el valor inválido y no se escribe `kickoff.yaml`
- **Y** no se sella el cambio como no ISO por defecto

#### Scenario: Un binario anterior ignora el campo

- **DADO** un `kickoff.yaml` con `iso` activo
- **CUANDO** lo decodifica el esquema previo a este cambio
- **ENTONCES** la decodificación termina sin error
