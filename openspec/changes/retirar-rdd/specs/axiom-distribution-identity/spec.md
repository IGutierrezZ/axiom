# Delta for axiom-distribution-identity

## MODIFIED Requirements

### Requirement: Taxonomía de interoperabilidad vs. identidad (REQ-20.7)

El sistema DEBE clasificar cada superficie de nombre publicado en exactamente una de dos categorías: interoperabilidad (contrato de cable consumido por otro software, cuyo renombrado rompe a consumidores) o identidad (superficie leída por una persona, renombrable sin romper ningún contrato). El namespace de protocolo bajo `contracts/**` y los marcadores de contenido gestionado que un componente Go analiza de forma literal DEBEN clasificarse como interoperabilidad. El instalador, el tap de Homebrew, las compuertas de release, el nombre de binario citado en la documentación y los nombres de servicio de despliegue DEBEN clasificarse como identidad.
(Previously: el escenario usaba el esquema `contracts/review-integration/v2/schemas/status.schema.json`, que se retira con RDD (C-01-e); se sustituye por el esquema de consentimiento de `contracts/sdd-integration`, que permanece.)

#### Scenario: El namespace de contracts/** se clasifica como interoperabilidad
- **DADO** el fichero `contracts/sdd-integration/v1/schemas/consent.schema.json` con `"$id": "https://gentle-ai.dev/contracts/sdd-integration/v1/schemas/consent.schema.json"`
- **CUANDO** se clasifica esa superficie
- **ENTONCES** se clasifica como interoperabilidad
- **Y** ninguna tanda de identidad la modifica

### Requirement: Namespace de protocolo en contracts/** sin cambios (REQ-20.10)

El sistema NO DEBE modificar el namespace de protocolo bajo `contracts/**`: los valores `$id`, los prefijos de contrato (`axiom.sdd-integration/v1`, `axiom.sdd-integration.consent/v1` y equivalentes) y los esquemas asociados DEBEN permanecer byte a byte idénticos.
(Previously: los prefijos de ejemplo eran `gentle-ai.review-integration/v2` y `gentle-ai.sdd-status/v2`, ya sin ficheros bajo `contracts/**` tras retirar `contracts/review-integration` y `contracts/review-provider-contract` (C-01-e); el escenario hablaba de «agentes y revisores».)

#### Scenario: contracts/** permanece sin cambios
- **DADO** el estado de `contracts/**/*.schema.json`
- **CUANDO** se auditan los esquemas
- **ENTONCES** cada fichero bajo `contracts/**` es byte a byte idéntico
- **Y** los agentes y demás consumidores continúan interoperando con el formato publicado

## ADDED Requirements

### Requirement: La Release No Publica el Bundle `review-provider-contract` (REQ-46.2)

La configuración de GoReleaser, los workflows `release.yml` y `promote-stable-rc.yml`, los scripts `scripts/verify-release-assets.sh` y `scripts/promote-stable-preflight.sh` y la política de `internal/releasepolicy` NO DEBEN producir, esperar, verificar ni promover el artefacto `gentle-ai-review-provider-contract-<semver>.tar.gz`, ni generar el directorio `.goreleaser-provider-contract`, ni exigir la variable `PROVIDER_CONTRACT_SEMVER`. Los directorios `contracts/review-integration` y `contracts/review-provider-contract` NO DEBEN existir; `contracts/sdd-integration` permanece byte a byte idéntico. Una release sin ese artefacto DEBE superar las compuertas de verificación de assets.

#### Scenario: Los assets esperados no incluyen el bundle

- **DADO** el conjunto de assets que `verify-release-assets.sh` y `internal/releasepolicy` esperan para una versión
- **CUANDO** se compara con los artefactos que declara `.goreleaser.yaml`
- **ENTONCES** ninguno de los dos contiene `review-provider-contract` ni `PROVIDER_CONTRACT_SEMVER`
- **Y** ambos listan los mismos artefactos

#### Scenario: Una release sin el bundle supera las compuertas

- **DADO** una release simulada que contiene todos los artefactos `axiom_*` y ningún `gentle-ai-review-provider-contract-*`
- **CUANDO** se ejecutan la verificación de assets y los tests de `internal/update` y `internal/releasepolicy`
- **ENTONCES** la verificación termina sin error

#### Scenario: Los contratos de revisión no existen y el de SDD permanece

- **DADO** el árbol del repositorio tras la retirada
- **CUANDO** se listan los subdirectorios de `contracts/`
- **ENTONCES** solo existe `sdd-integration`
- **Y** sus ficheros son idénticos a los de antes de la retirada
