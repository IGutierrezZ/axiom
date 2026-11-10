# Delta para Topología del Workspace (carril-proyecto-publicacion-docs)

> **Cambio:** `carril-proyecto-publicacion-docs` · **Tipo:** delta sobre la capacidad viva `workspace-topology`.
> **Idioma:** castellano de España; identificadores y claves en inglés.

Cambios de este delta:

- REQ-2.1: el esquema acepta además el bloque opcional de primer nivel `workflow` (REQ-27.1).
- REQ-27.1 (nuevo): bloque `workflow` con la clave `lane` (`odd | sdd | ask`), análogo a REQ-25.26.

## ADDED Requirements

### Requirement: Bloque `workflow:` Opcional de `axiom.yaml` con el Carril del Proyecto (REQ-27.1)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `workflow:` con la clave opcional `lane`, cuyos valores válidos son `odd`, `sdd` y `ask`. Sin el bloque, o sin la clave, el carril efectivo del proyecto vale `sdd`. El bloque DEBE decodificarse de forma estricta por lista cerrada de claves, con el mismo patrón que `git:` (REQ-25.26): una clave desconocida dentro de `workflow:` y un valor de `lane` fuera de los tres admitidos DEBEN reportarse como error accionable que nombre la clave (`workflow.lane` o la clave desconocida) y los valores admitidos, tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. Un `workflow:` que no sea un mapa también DEBE reportarse como error. Las claves desconocidas de primer nivel fuera de los bloques reconocidos se siguen ignorando (REQ-24.17 de `spec-baseline-declaration`): un binario anterior a este cambio DEBE cargar un `axiom.yaml` con el bloque `workflow:` sin error y sin interpretarlo. El carril es política del proyecto y se versiona; la elección del modo de trabajo personal NO DEBE declararse en `axiom.yaml`. El valor por defecto `sdd` rige en `monorepo-embedded` y en `multirepo`; en `multirepo` el carril `odd` sigue estando permitido.

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

## MODIFIED Requirements

### Requirement: Estructura del Esquema axiom.yaml (REQ-2.1)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`)

Además DEBE aceptar los bloques opcionales de primer nivel `baseline` (REQ-24.13), `technical_context` (REQ-24.26), `git` (REQ-25.26), `repositories` (REQ-25.27) y `workflow` (REQ-27.1): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas, perfiles y valores). Las claves desconocidas fuera de esos bloques se siguen ignorando.
(Previously: el esquema no contemplaba el bloque opcional `workflow`)

#### Scenario: El bloque `workflow` es opcional y se valida cuando está presente

- **DADO** un `axiom.yaml` con los campos obligatorios y, en una variante, además `workflow: {lane: ask}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` sobre ambas variantes
- **ENTONCES** las dos son `COMPLIANT`
- **Y** con `workflow: {lane: x}` la validación falla nombrando `workflow.lane`
