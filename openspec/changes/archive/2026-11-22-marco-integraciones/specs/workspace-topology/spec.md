# Delta para workspace-topology

> Parte del estado vigente: el resto de requisitos NO cambia. Esta delta añade a `axiom.yaml` el bloque opcional `integrations:` (C-07-b, capacidad `integration-plugins`). Es solo ADDED: no se modifica ningún requisito existente. Un binario anterior ignora la clave de primer nivel desconocida (REQ-24.17).

## ADDED Requirements

### Requirement: Bloque `integrations:` Opcional y Estricto de `axiom.yaml` (REQ-43.19)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `integrations:`, un mapeo `<plugin>: <mapeo de configuración>`. SOLO DEBE admitir identificadores presentes en el catálogo de manifiestos (REQ-43.4) y, dentro de cada uno, solo las claves de configuración de su manifiesto (REQ-43.3). Son error de carga, en `ParseConfig`, `LoadConfig` y `axiom workspace validate`: un plugin desconocido, una clave desconocida, un valor del bloque nulo o que no es un mapeo, una clave duplicada un valor de plugin nulo o que no es un mapeo, un valor de tipo distinto del declarado en el esquema del manifiesto, un valor fuera del enumerado y una clave obligatoria ausente (esta última solo si `enabled` no es `false`). Dentro de cada plugin, además de las claves del manifiesto, se admite la clave reservada `enabled` (booleano estricto, por defecto `true`); un plugin con `enabled: false` no configura nada y no recibe eventos. Los valores por defecto del manifiesto se aplican al proyectar el bloque. Los errores se acumulan todos, con línea y alternativas (con el catálogo vacío, «plugins admitidos: ninguno en esta versión»). `integrations: {}` es válido y no configura nada. El mensaje DEBE incluir la línea y la lista de claves o plugins admitidos, con el mismo formato que el resto de bloques. Sin el bloque, NO DEBE cambiar ningún comportamiento (REQ-43.5). Ningún secreto DEBE poder declararse en este bloque: los secretos no son claves de configuración del manifiesto (REQ-43.3), así que una clave con el nombre de un secreto es una clave desconocida.

(Placement: section "Bloque `workflow:` Opcional de `axiom.yaml` con el Carril del Proyecto (REQ-27.1)", a continuación)

#### Scenario: Bloque válido
- **DADO** el doble registrado en el catálogo y `integrations: { <doble>: { <clave válida>: valor } }`
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y `WorkspaceConfig` proyecta el bloque

#### Scenario: Clave desconocida
- **DADO** `integrations: { <doble>: { clave_inexistente: 1 } }`
- **CUANDO** se ejecuta `ParseConfig`, `LoadConfig` o `axiom workspace validate`
- **ENTONCES** falla con un error que nombra `clave_inexistente`, su línea y las claves admitidas

#### Scenario: Plugin desconocido
- **DADO** `integrations: { azure: {} }` sin ese plugin en el catálogo
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra `azure`, su línea y los plugins admitidos

#### Scenario: Bloque nulo o no mapeo
- **DADO** `integrations:` sin valor, o `integrations: [x]`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error tipado que indica que debe ser un mapeo

#### Scenario: Clave duplicada
- **DADO** dos entradas `<doble>:` iguales bajo `integrations:`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra la clave duplicada y su línea

#### Scenario: Plugin desactivado
- **DADO** `integrations: { <doble>: { enabled: false } }` sin la clave obligatoria del manifiesto
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y el plugin no figura entre los activos

#### Scenario: Obligatoria ausente, tipo y enumerado
- **DADO** un plugin activo sin su clave obligatoria, otro con un valor de tipo incorrecto y otro con un valor fuera del enumerado
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error por cada problema, con su línea y las alternativas

#### Scenario: Bloque vacío
- **DADO** `integrations: {}`
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y no configura ningún plugin

#### Scenario: Un secreto no se puede declarar en `axiom.yaml`
- **DADO** un manifiesto cuyo secreto se llama `token` y `integrations: { <doble>: { token: abc } }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla como clave desconocida y el mensaje no contiene `abc`

### Requirement: `Locate` Expone el Error del Bloque sin Rechazar y no Bloquea el Archivado (REQ-43.20)

`workspace.Locate` NO DEBE rechazar por un bloque `integrations:` inválido: DEBE exponerlo como `Declarations.IntegrationsErr`, igual que `git:`, `workflow:` y `sdd:`. El bloque `integrations:` NO DEBE bloquear el archivado (`axiom sdd status` no lo cuenta entre las declaraciones inválidas que bloquean). `axiom init --force` DEBE tratar `integrations:` como bloque propio del usuario y no sobrescribirlo (`declares()` conoce `Integrations`).

#### Scenario: `Locate` con bloque inválido
- **DADO** un `axiom.yaml` con `integrations:` inválido
- **CUANDO** se llama a `workspace.Locate`
- **ENTONCES** no devuelve error por ello
- **Y** `Declarations.IntegrationsErr` contiene el error de REQ-43.19

#### Scenario: Archivado no bloqueado
- **DADO** un `axiom.yaml` con `integrations:` inválido y un cambio listo para archivar
- **CUANDO** se consulta `axiom sdd status`
- **ENTONCES** la fase de archivado no queda bloqueada por esa declaración

#### Scenario: `init --force` conserva el bloque
- **DADO** un `axiom.yaml` con un bloque `integrations:`
- **CUANDO** se ejecuta `axiom init --force`
- **ENTONCES** el bloque `integrations:` se conserva sin cambios

### Requirement: Plantilla y Manual Documentan `integrations:` (REQ-43.21)

`axiom.example.yaml` DEBE incluir un bloque `integrations:` comentado y `docs/manual-de-inicio.md` (sección 6) DEBE incluir un apartado `integrations:` que describa el bloque, el catálogo de plugins, que los secretos nunca se escriben en `axiom.yaml` y los verbos `axiom integrations`. Las pruebas de contrato (`example_config_test.go`, `docs_contract_test.go`) DEBEN fijar ambos.

#### Scenario: Plantilla y manual cubiertos por las pruebas de contrato
- **CUANDO** se ejecutan las pruebas de contrato de la plantilla y del manual
- **ENTONCES** comprueban la presencia del bloque comentado y del apartado, y fallan si se eliminan
