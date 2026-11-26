# Delta para Topología de Workspace y CLI de Axiom (workspace-topology)

> **Cambio:** `bateria-pruebas` (C-06-c). **Idioma:** castellano de España; identificadores, claves y tokens en inglés.
> RF-TOP-01 a RF-TOP-18 no cambian de texto.

## ADDED Requirements

### Requirement: Bloque `acceptance_tests:` Opcional y Estricto de `axiom.yaml`, Solo en `multirepo` (RF-TOP-19)

`axiom.yaml` PUEDE declarar un bloque de primer nivel `acceptance_tests: { enabled: <bool>, environments: [<cadena>, ...] }`. El decodificador DEBE ser estricto, en `ParseConfig`, `LoadConfig` y `axiom workspace validate`, y DEBE acumular todos los errores con su línea:

- una clave distinta de `enabled` y `environments` es un error de clave desconocida, que lista las admitidas;
- un valor del bloque nulo o que no es un mapeo, y una clave duplicada, son error;
- `enabled` DEBE ser un booleano estricto (`true` o `false`); cualquier otro tipo o valor es error y no se asume nunca un valor;
- `environments` DEBE ser una lista de cadenas no vacías; un elemento no cadena o vacío es error, y también lo es un elemento que contenga `;`, porque `;` separa los entornos de cada caso en `pruebas.md`;
- con `enabled: true`, `environments` DEBE estar presente, no vacío y sin duplicados (comparación exacta, sensible a mayúsculas); su ausencia, una lista vacía o un duplicado son error;
- con `enabled: false`, `environments` es opcional y no se valida más allá del tipo.

`acceptance_tests: { enabled: false }` y la ausencia del bloque son equivalentes: no activan nada. El accesor del bloque (patrón de `ExplainerHTML()`) NO DEBE asumir un valor cuando la declaración es inválida: DEBE devolver el error.

`workspace.Validate` DEBE rechazar `enabled: true` cuando la topología no es `multirepo`, en el paso 3b junto a la regla de roles de RF-TOP-04, con un error que nombra la clave y la topología declarada. `enabled: false` y la ausencia del bloque son válidos en cualquier topología. La regla vive en `Validate` y no en la carga, para que `Locate` y el archivado toleren el error (RF-TOP-20).

#### Scenario: Bloque válido en multirepo
- **DADO** un `axiom.yaml` `multirepo` con `acceptance_tests: { enabled: true, environments: [local, dev, pre] }`
- **CUANDO** se carga la configuración y se ejecuta `workspace.Validate`
- **ENTONCES** ambos terminan sin error
- **Y** el accesor devuelve activación `true` y los entornos `local`, `dev` y `pre` en ese orden

#### Scenario: Sin bloque o con `enabled: false`
- **DADO** un `axiom.yaml` sin `acceptance_tests` o con `acceptance_tests: { enabled: false }`
- **CUANDO** se carga la configuración
- **ENTONCES** la carga termina sin error y la función se considera no activa

#### Scenario: Clave desconocida
- **DADO** `acceptance_tests: { enabled: true, environments: [dev], modo: x }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra `modo`, su línea y las claves admitidas `enabled` y `environments`

#### Scenario: Activo sin entornos o con entornos vacíos
- **DADO** `acceptance_tests: { enabled: true }` o `acceptance_tests: { enabled: true, environments: [] }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que indica que `environments` debe ser no vacío

#### Scenario: Entornos duplicados
- **DADO** `acceptance_tests: { enabled: true, environments: [dev, pre, dev] }`
- **CUANDO** se carga la configuración
- **ENTONCES** falla con un error que nombra `dev` como duplicado

#### Scenario: `enabled` no booleano o bloque no mapeo
- **DADO** `acceptance_tests: { enabled: "sí" }`, `acceptance_tests:` sin valor o `acceptance_tests: [x]`
- **CUANDO** se carga la configuración
- **ENTONCES** cada caso falla con un error tipado y no se asume ningún valor de `enabled`

#### Scenario: Monorepo rechaza `enabled: true`
- **DADO** un `axiom.yaml` `monorepo-embedded` con `acceptance_tests: { enabled: true, environments: [dev] }`
- **CUANDO** se ejecuta `workspace.Validate`
- **ENTONCES** falla con un error que nombra `acceptance_tests` y la topología `monorepo-embedded`

#### Scenario: Monorepo admite `enabled: false` y la ausencia del bloque
- **DADO** un `axiom.yaml` `monorepo-embedded` con `acceptance_tests: { enabled: false }`, y otro sin el bloque
- **CUANDO** se ejecuta `workspace.Validate`
- **ENTONCES** ambos terminan sin error por esta regla

### Requirement: `Locate` Expone el Error del Bloque sin Rechazar, no Bloquea el Archivado y `init --force` lo Conserva; Plantilla y Manual lo Documentan (RF-TOP-20)

`workspace.Locate` NO DEBE rechazar por un bloque `acceptance_tests:` inválido: DEBE exponerlo como `Declarations.AcceptanceTestsErr`, igual que `git:`, `workflow:`, `sdd:` e `integrations:`. El bloque NO DEBE bloquear el archivado (`axiom sdd status` no lo cuenta entre las declaraciones inválidas que bloquean). `axiom init --force` DEBE tratar `acceptance_tests:` como bloque propio del usuario y no sobrescribirlo (`declares()` conoce el bloque). `axiom.example.yaml` DEBE incluir un bloque `acceptance_tests:` comentado con `enabled` y `environments`, y `docs/manual-de-inicio.md` y `docs/openspec-config.md` DEBEN describirlo: claves, que solo se admite `enabled: true` en `multirepo`, que `environments` es el dominio cerrado de la columna `Entornos` de `pruebas.md` y que sin el bloque no cambia ningún comportamiento. Las pruebas de contrato DEBEN fijarlo.

#### Scenario: `Locate` con bloque inválido
- **DADO** un `axiom.yaml` con `acceptance_tests:` inválido
- **CUANDO** se llama a `workspace.Locate`
- **ENTONCES** no devuelve error por ello
- **Y** `Declarations.AcceptanceTestsErr` contiene el error de RF-TOP-19

#### Scenario: Archivado no bloqueado
- **DADO** un `axiom.yaml` con `acceptance_tests:` inválido y un cambio listo para archivar
- **CUANDO** se consulta `axiom sdd status`
- **ENTONCES** la fase de archivado no queda bloqueada por esa declaración

#### Scenario: `init --force` conserva el bloque
- **DADO** un `axiom.yaml` con un bloque `acceptance_tests:`
- **CUANDO** se ejecuta `axiom init --force`
- **ENTONCES** el bloque se conserva sin cambios

#### Scenario: Plantilla y manual cubiertos por las pruebas de contrato
- **CUANDO** se ejecutan las pruebas de contrato de `axiom.example.yaml`, del manual y de `docs/openspec-config.md`
- **ENTONCES** comprueban la presencia del bloque comentado y de su apartado, y fallan si se eliminan
