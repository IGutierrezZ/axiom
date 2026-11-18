# Delta para workspace-topology

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta añade el bloque opcional de primer nivel `delivery:` de `axiom.yaml` con la clave `split_hint_lines` (decisión P5), que activa un aviso único y no bloqueante de cambio grande. La capacidad `workspace-topology` es la dueña del esquema de `axiom.yaml`, como ya lo es de `git:` (REQ-25.26) y de `workflow:` (REQ-27.1); la lectura de la clave y el aviso se consumen desde la doctrina (`sdd-orchestrator-assets` y `organic-agent-trigger-rules`). Los requisitos nuevos usan el grupo de numeración REQ-32.

## MODIFIED Requirements

### Requirement: Estructura del Esquema axiom.yaml (REQ-2.1)

El motor de configuración DEBE deserializar y validar el archivo `axiom.yaml`, exigiendo los campos obligatorios:
- `workspace.name` (cadena no vacía)
- `workspace.topology` (uno de: `monorepo-embedded`, `multirepo`)
- `workspace.specs_repository` (ruta relativa al directorio de especificaciones)
- `roles` (mapa con al menos 1 rol definido, conteniendo `name` y una lista de `repositories`)

Además DEBE aceptar los bloques opcionales de primer nivel `technical_context` (REQ-24.26), `git` (REQ-25.26), `repositories` (REQ-25.27), `workflow` (REQ-27.1) y `delivery` (REQ-32.1): su ausencia no altera la validación y, cuando están presentes, se validan de forma estricta (claves desconocidas, rutas y valores). Un bloque `baseline` NO es un bloque admitido: se rechaza con el error tipado de REQ-30.21. Las claves desconocidas fuera de esos bloques se siguen ignorando.
(Previously: la lista de bloques opcionales de primer nivel era `technical_context`, `git`, `repositories` y `workflow`)

#### Scenario: El bloque `workflow` es opcional y se valida cuando está presente

- **DADO** un `axiom.yaml` con los campos obligatorios y, en una variante, además `workflow: {lane: ask}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` sobre ambas variantes
- **ENTONCES** las dos son `COMPLIANT`
- **Y** con `workflow: {lane: x}` la validación falla nombrando `workflow.lane`

#### Scenario: El bloque `delivery` es opcional y se valida cuando está presente

- **DADO** un `axiom.yaml` con los campos obligatorios y, en una variante, además `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom workspace validate` sobre ambas variantes
- **ENTONCES** las dos son `COMPLIANT`
- **Y** con `delivery: {split_hint_lines: 0}` la validación falla nombrando `delivery.split_hint_lines`

#### Scenario: El bloque `baseline` ya no es un bloque admitido

- **DADO** un `axiom.yaml` con los campos obligatorios y un bloque `baseline:`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la validación falla con el error tipado de REQ-30.21 y no lo trata como clave desconocida ignorable

## ADDED Requirements

### Requirement: Bloque `delivery:` Opcional de `axiom.yaml` con el Umbral del Aviso de Cambio Grande (REQ-32.1)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `delivery:` con la clave opcional `split_hint_lines`, un entero positivo (mayor o igual que 1) que fija el número de líneas a partir del cual el orquestador avisa de que un cambio es grande (REQ-32.2). Sin el bloque, o sin la clave, el aviso está desactivado: no hay umbral por defecto y ninguna superficie lo sustituye por un número. El valor `0` NO es una forma de desactivar el aviso: se rechaza; la forma de desactivarlo es no declarar la clave. El bloque DEBE decodificarse de forma estricta por lista cerrada de claves, con el mismo patrón y el mismo canal de error que `git:` (REQ-25.26) y `workflow:` (REQ-27.1), tanto al cargar la configuración como en `axiom workspace validate`, con código de salida `1`. DEBEN reportarse como error accionable que nombre la clave: una clave desconocida o repetida dentro de `delivery:`, un `delivery:` que no sea un mapa, y un valor de `split_hint_lines` que no sea un entero, que sea `0` o negativo, o que sea nulo (`delivery.split_hint_lines`). Las claves desconocidas de primer nivel fuera de los bloques reconocidos se siguen ignorando: un binario anterior a este cambio DEBE cargar un `axiom.yaml` con el bloque `delivery:` sin error y sin interpretarlo. `axiom init --force` DEBE preservar un bloque `delivery:` ya declarado, igual que los demás bloques declarados. La clave es política del proyecto y se versiona. El manual de inicio (`docs/manual-de-inicio.md`) DEBE documentar la clave, su dominio y que su ausencia desactiva el aviso.

#### Scenario: Sin bloque el aviso está desactivado

- **DADO** un `axiom.yaml` sin `delivery:`
- **CUANDO** se resuelve la declaración del proyecto
- **ENTONCES** `split_hint_lines` no tiene valor y el aviso está desactivado
- **Y** `axiom workspace validate` no reporta ningún error por ello

#### Scenario: Bloque vacío o sin clave mantiene el aviso desactivado

- **DADO** un `axiom.yaml` con `delivery: {}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT` y el aviso sigue desactivado

#### Scenario: Valor válido

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT`
- **Y** la configuración resuelta expone `split_hint_lines` con el valor `600`

#### Scenario: Valor cero, negativo, no entero o nulo

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 0}` (y variantes con `-5`, `"mucho"`, `12.5` y `null`)
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra `delivery.split_hint_lines` y la regla incumplida (entero positivo)
- **Y** el código de salida es `1`

#### Scenario: Clave desconocida o repetida dentro de `delivery:`

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600, budget: 400}` (y otro con `split_hint_lines` repetida)
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `budget` como desconocida (y la repetida como repetida)
- **Y** el código de salida es `1`

#### Scenario: `delivery:` que no es un mapa

- **DADO** un `axiom.yaml` con `delivery: 600`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** la salida nombra la clave `delivery` y la regla incumplida
- **Y** el código de salida es `1`

#### Scenario: La carga de la configuración rechaza el valor inválido por el mismo canal

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: -1}`
- **CUANDO** se carga la configuración del workspace
- **ENTONCES** la carga reporta el error de `delivery.split_hint_lines` junto con los de los demás bloques, no un error genérico

#### Scenario: Un binario anterior ignora el bloque

- **DADO** un `axiom.yaml` válido con `delivery: {split_hint_lines: 600}`
- **CUANDO** lo carga el decodificador previo a este cambio (el que ignora las claves desconocidas de primer nivel)
- **ENTONCES** la carga termina sin error
- **Y** el resto de la configuración se lee igual que sin el bloque

#### Scenario: `axiom init --force` preserva el bloque

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom init --force`
- **ENTONCES** el `axiom.yaml` resultante conserva el bloque `delivery:` con su valor

#### Scenario: Ambas topologías admiten el bloque

- **DADO** un `axiom.yaml` de topología `monorepo-embedded` y otro `multirepo`, ambos con `delivery: {split_hint_lines: 600}`
- **CUANDO** el usuario ejecuta `axiom workspace validate`
- **ENTONCES** el resultado es `COMPLIANT` en ambos

#### Scenario: El manual documenta la clave

- **DADO** el manual de inicio
- **CUANDO** se inspecciona (prueba de pin)
- **ENTONCES** documenta `delivery.split_hint_lines`, que es un entero positivo y que su ausencia desactiva el aviso

---

### Requirement: Con `delivery.split_hint_lines` Fijada el Orquestador Avisa una Vez de un Cambio Grande y Nunca Bloquea (REQ-32.2)

Cuando `axiom.yaml` declara `delivery.split_hint_lines` (REQ-32.1) y la previsión de líneas del trabajo la supera, el orquestador DEBE emitir **un único aviso** por cambio (SDD) o por feature (ODD) indicando que el cambio es grande y que la persona puede pedir PRs encadenadas, y DEBE continuar sin esperar respuesta. El aviso NUNCA bloquea ni detiene el avance, no cambia la estrategia de entrega (sigue siendo `single-pr` salvo petición expresa de la persona) y no exige ninguna decisión. La previsión que se compara es la estimación cualitativa de tamaño de `sdd-tasks` en SDD y la previsión del documento de la feature en ODD. El orquestador lee la clave directamente de `axiom.yaml` (no se proyecta en `axiom sdd status`). Sin la clave NO hay aviso ni ningún número de comparación: ni la doctrina ni los orquestadores DEBEN contener un umbral por defecto.

#### Scenario: Con la clave fijada y la previsión superada se avisa una vez

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}` y una previsión de líneas del cambio superior a 600
- **CUANDO** el orquestador evalúa la previsión
- **ENTONCES** emite un único aviso de cambio grande que menciona que se pueden pedir PRs encadenadas
- **Y** continúa con la entrega `single-pr` sin esperar respuesta

#### Scenario: El aviso no se repite en el mismo cambio

- **DADO** un cambio sobre el que ya se emitió el aviso
- **CUANDO** el orquestador vuelve a evaluar la previsión en una fase posterior del mismo cambio
- **ENTONCES** no emite un segundo aviso

#### Scenario: Con la previsión por debajo o igual no hay aviso

- **DADO** un `axiom.yaml` con `delivery: {split_hint_lines: 600}` y una previsión que no supera 600
- **CUANDO** el orquestador evalúa la previsión
- **ENTONCES** no emite ningún aviso

#### Scenario: Sin la clave no hay aviso ni número

- **DADO** un `axiom.yaml` sin `delivery:`
- **CUANDO** el orquestador evalúa la previsión de un cambio de cualquier tamaño
- **ENTONCES** no emite ningún aviso
- **Y** la doctrina que lo describe no contiene ningún número de líneas como umbral por defecto

#### Scenario: El aviso nunca bloquea

- **DADO** un aviso de cambio grande emitido
- **CUANDO** la persona no responde
- **ENTONCES** el trabajo continúa sin parada ni decisión pendiente
- **Y** `sdd-apply` no se detiene por ese motivo
