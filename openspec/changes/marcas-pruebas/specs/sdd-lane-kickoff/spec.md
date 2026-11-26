# Delta para Arranque del Carril SDD (sdd-lane-kickoff)

> **Cambio:** `marcas-pruebas` (C-06-d). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.

## ADDED Requirements

### Requirement: Formato de `pruebas-estado.yaml` de Solo Añadir y Caducidad por `plan_digest` (prov:KOF:1)

El estado de ejecución de la batería DEBE vivir en `pruebas-estado.yaml`, en la carpeta del cambio y separado de `pruebas.md`. Cada entrada DEBE tener `case` (`P-NN`), `environment`, `result` (`ok|ko|blocked|n/a`), `by` (identidad git `user.name` y `user.email`), `at` (fecha y hora), `commit` (opcional, vacío por defecto), `plan_digest` y `note` (acotada). El fichero es de solo añadir: ninguna operación del producto DEBE editar ni borrar entradas, y una corrección es una entrada nueva; para cada par caso-entorno cuenta como vigente la última entrada que no esté caducada. `plan_digest` DEBE ser `kickoff.ArtifactDigest` de `pruebas.md` (el mismo digest que la compuerta `tasks`, con finales de línea CRLF normalizados). Una entrada cuyo `plan_digest` difiere del actual DEBE considerarse caducada: se cuenta aparte y NUNCA como vigente. Un fichero ausente equivale a ninguna marca; un fichero ilegible o inválido DEBE reportarse como hallazgo informativo, sin bloquear nada. Este fichero NO DEBE formar parte de ningún digest de compuerta ni de `kickoff.yaml`.

#### Scenario: Entrada con todos los campos
- **DADO** una marca `P-01`, `pre`, `ok`
- **CUANDO** se escribe en `pruebas-estado.yaml`
- **ENTONCES** la entrada contiene `case`, `environment`, `result`, `by`, `at`, `commit` vacío, `plan_digest` y `note`

#### Scenario: Caducidad al cambiar `pruebas.md`
- **DADO** una marca con el `plan_digest` actual
- **CUANDO** se modifica `pruebas.md`
- **ENTONCES** la marca se considera caducada y no cuenta como vigente

#### Scenario: Finales de línea no caducan
- **DADO** una marca y un `pruebas.md` cuyos finales de línea pasan de LF a CRLF sin otro cambio
- **CUANDO** se recalcula el digest
- **ENTONCES** el `plan_digest` coincide y la marca sigue vigente

#### Scenario: Gana la última entrada vigente
- **DADO** una entrada `ko` y otra posterior `ok` para el mismo caso y entorno, ambas vigentes
- **CUANDO** se calcula el estado
- **ENTONCES** el par caso-entorno cuenta como `ok`

#### Scenario: Fichero ausente
- **DADO** un cambio sin `pruebas-estado.yaml`
- **CUANDO** se calcula el estado
- **ENTONCES** no hay marcas y no hay error

#### Scenario: Fichero inválido
- **DADO** un `pruebas-estado.yaml` con YAML inválido
- **CUANDO** se calcula el estado
- **ENTONCES** se reporta un hallazgo informativo
- **Y** `nextRecommended`, las dependencias y los digests no cambian

## MODIFIED Requirements

### Requirement: El Status Proyecta `acceptanceTests` de Forma Aditiva y Avisa sin Bloquear la Ausencia de `pruebas.md` (RF-KOF-22)

`axiom sdd status --json` DEBE proyectar la clave aditiva `acceptanceTests` con el estado vivo de `axiom.yaml`: la activación y los entornos, o el error del bloque si es inválido (sin asumir un valor), y, si existe `pruebas.md`, el resultado de `acceptancetests.Check`: número de casos, escenarios y requisitos cubiertos (M/M) y número de hallazgos. Si la función está activa, `tasks` está terminado y falta `pruebas.md`, DEBE emitir un aviso informativo. Con `enabled: false` o sin el bloque, DEBE proyectar la función como no activa y NO DEBE emitir ningún aviso; omitir la clave `acceptanceTests` equivale a proyectarla como no activa, para que el JSON del status de un proyecto sin el bloque no cambie.

Cuando existe `pruebas.md` y `pruebas-estado.yaml` (prov:KOF:1) con entradas, la proyección DEBE añadir de forma aditiva el resumen de marcas: por cada entorno, el número de casos con marca vigente `ok` sobre el total de casos aplicables a ese entorno, y el número de marcas caducadas aparte. Sin `pruebas-estado.yaml`, o en un proyecto `monorepo-embedded`, la clave de marcas NO DEBE proyectarse. Las tres presentaciones del resumen (status, presentación de `integration` y `archive-report`) DEBEN salir del mismo cálculo.
(Previously: la proyección no incluía ningún resumen de marcas)

La proyección y el aviso NO DEBEN alterar `nextRecommended`, las dependencias, los motivos de bloqueo ni los digests, y NO DEBEN bloquear la compuerta ni el avance. Ninguna clave existente del status DEBE cambiar de nombre, tipo ni valor. El contrato publicado (`_shared/sdd-status-contract.md`) DEBE documentar la clave y el resumen de marcas.

#### Scenario: Función activa con `pruebas.md`
- **DADO** la función activa y un `pruebas.md` con 2 hallazgos
- **CUANDO** se ejecuta `axiom sdd status --change <c> --json`
- **ENTONCES** `acceptanceTests` proyecta activación, entornos, cobertura y 2 hallazgos

#### Scenario: Aviso por ausencia sin bloqueo
- **DADO** la función activa, `tasks` terminado y sin `pruebas.md`
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` incluye un aviso
- **Y** `nextRecommended` y las dependencias son idénticos a los del mismo cambio con `pruebas.md` ausente y la función no activa

#### Scenario: Función no activa
- **DADO** `enabled: false` o sin el bloque
- **CUANDO** se consulta el status
- **ENTONCES** no hay aviso y los digests no cambian

#### Scenario: Declaración inválida
- **DADO** un bloque `acceptance_tests` inválido
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` proyecta el error del bloque
- **Y** no se asume activación

#### Scenario: Resumen por entorno
- **DADO** 5 casos aplicables a `pre` con 5 marcas vigentes `ok`, y 2 casos aplicables a `local` con 1 marca vigente `ok` y 1 marca caducada
- **CUANDO** se consulta el status
- **ENTONCES** el resumen proyecta `pre` con 5 de 5 `ok` y `local` con 1 de 2 `ok` y 1 marca caducada

#### Scenario: Sin marcas no hay clave
- **DADO** un cambio con `pruebas.md` y sin `pruebas-estado.yaml`
- **CUANDO** se consulta el status
- **ENTONCES** `acceptanceTests` no incluye el resumen de marcas y el resto del JSON no cambia

#### Scenario: Las marcas no alteran el enrutado
- **DADO** el mismo cambio con y sin `pruebas-estado.yaml`
- **CUANDO** se consulta el status
- **ENTONCES** `nextRecommended`, las dependencias, los motivos de bloqueo y los digests son idénticos
