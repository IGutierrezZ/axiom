# Especificación Viva: Identidad de Distribución de Axiom

> **Dominio:** `axiom-distribution-identity`  
> **Versión Canónica:** 1.0.0  
> **Estado:** Vigente  
> **Idioma:** Español (Castellano peninsular)

---

## 1. Capacidad: `axiom-distribution-identity`

Contrato de nombre publicado por clase de superficie: instalador y tap, compuertas de release, workflows de CI, shim de `crosslane`, namespace de protocolo en `contracts/**`, ruta de módulo Go y nombres de servicio de telemetría de despliegue (este último retirado: RF-DIS-06). Distingue interoperabilidad (lo que lee una máquina, no renombrable sin romper consumidores) de identidad (lo que lee un humano, renombrable).

### Requirement: Taxonomía de interoperabilidad vs. identidad (RF-DIS-01)

El sistema DEBE clasificar cada superficie de nombre publicado en exactamente una de dos categorías: interoperabilidad (contrato de cable consumido por otro software, cuyo renombrado rompe a consumidores) o identidad (superficie leída por una persona, renombrable sin romper ningún contrato). El namespace de protocolo bajo `contracts/**` y los marcadores de contenido gestionado que un componente Go analiza de forma literal DEBEN clasificarse como interoperabilidad. El instalador, el tap de Homebrew, las compuertas de release, el nombre de binario citado en la documentación y los nombres de servicio de despliegue DEBEN clasificarse como identidad.
(Previously: el escenario usaba el esquema `contracts/review-integration/v2/schemas/status.schema.json`, que se retira con RDD (C-01-e); se sustituye por el esquema de consentimiento de `contracts/sdd-integration`, que permanece.)

#### Scenario: El namespace de contracts/** se clasifica como interoperabilidad
- **DADO** el fichero `contracts/sdd-integration/v1/schemas/consent.schema.json` con `"$id": "urn:axiom:contracts:sdd-integration:v1:consent"`
- **CUANDO** se clasifica esa superficie
- **ENTONCES** se clasifica como interoperabilidad
- **Y** ninguna tanda de identidad la modifica
(Previously: el escenario citaba el `$id` `https://gentle-ai.dev/contracts/sdd-integration/v1/schemas/consent.schema.json`, retirado al pasar el contrato de consentimiento a la identidad `axiom`.)

### Requirement: Publicación y compuertas de release bajo identidad propia (RF-DIS-02)

Las compuertas de release (`scripts/verify-release-assets.sh`, `scripts/release-preflight.sh`, `scripts/promote-stable-preflight.sh`) y la configuración de GoReleaser DEBEN apuntar a la identidad de distribución de Axiom (`IGutierrezZ/axiom`). Los artefactos de release publicados llevan el prefijo `axiom_{{ .Version }}_*`.

#### Scenario: Los activos publicados usan la identidad de Axiom
- **DADO** una versión taggeada del proyecto
- **CUANDO** el pipeline de release construye y publica los activos
- **ENTONCES** los binarios y paquetes se publican bajo el repositorio de Axiom con el nombre `axiom_*`
- **Y** el manifiesto de procedencia y los checksums firmados atestiguan el repositorio `IGutierrezZ/axiom`

---

### Requirement: Ruta de módulo Go migrada a /v3 (RF-DIS-03)

La ruta de módulo Go DEBE migrar de `github.com/gentleman-programming/gentle-ai/v2` al destino que sigue a upstream, `github.com/IGutierrezZ/axiom/v3`. La migración DEBE ser una única reescritura mecánica aplicada a todos los ficheros que citan la ruta anterior.

#### Scenario: Migración mecánica completa de la ruta de módulo
- **DADO** el módulo declarado como `github.com/gentleman-programming/gentle-ai/v2` en `go.mod`
- **CUANDO** se ejecuta la migración
- **ENTONCES** `go.mod` y todo fichero que importaba la ruta `/v2` pasan a citar `/v3`
- **Y** `go build ./...` compila en verde tras la migración

---

### Requirement: Namespace de protocolo en contracts/** sin cambios (RF-DIS-04)
El sistema NO DEBE modificar el namespace de protocolo bajo `contracts/**` fuera de un cambio de contrato deliberado: los valores `$id` (el del consentimiento es `urn:axiom:contracts:sdd-integration:v1:consent`), los prefijos de contrato (`axiom.sdd-integration/v1`, `axiom.sdd-integration.consent/v1` y equivalentes) y los esquemas asociados DEBEN permanecer byte a byte idénticos a la versión publicada con la identidad `axiom`, cuyo digest fija `internal/consentenvelope/consent_contract_test.go`. Ningún nombre `gentle-ai` DEBE aparecer en un `$id` ni en un prefijo de contrato.

#### Scenario: contracts/** permanece sin cambios
- **DADO** el estado de `contracts/**/*.schema.json`
- **CUANDO** se auditan los esquemas
- **ENTONCES** cada fichero bajo `contracts/**` coincide con el digest fijado en la prueba del contrato
- **Y** ningún `$id` ni prefijo de contrato contiene `gentle-ai`
- **Y** los agentes y demás consumidores continúan interoperando con el formato publicado
(Previously: el `$id` y los prefijos de ejemplo eran los del namespace `gentle-ai` y el escenario exigía identidad byte a byte sin referencia al digest del contrato.)

### Requirement: Binario gentle-ai retirado y shim de crosslane conservado (RF-DIS-05)

El sistema NO DEBE compilar ni publicar el binario `gentle-ai`: el paquete `cmd/gentle-ai` y el build `gentle-ai-deprecated` de `.goreleaser.yaml` se retiraron por decisión de producto (ODD `seguimientos-restos-upstream`). El shim ejecutable `gentle-ai` de `scripts/crosslane` ya no existe en el repositorio.

#### Scenario: El release publica solo el binario canónico
- **DADO** la configuración de release `.goreleaser.yaml`
- **CUANDO** se resuelven sus builds
- **ENTONCES** el único build es el que compila `./cmd/axiom` como `axiom`
- **Y** ningún build, archivo ni compuerta de release referencia `cmd/gentle-ai` ni un binario `gentle-ai`
(Previously: el requisito obligaba a conservar el shim `gentle-ai` de `scripts/crosslane/battery.go` y `host.go`; esos ficheros ya no existen y ningún artefacto del producto usa el nombre `gentle-ai`.)

### Requirement: Nombres de servicio de telemetría de despliegue (RF-DIS-06) — RETIRADO

> **Retirado:** la telemetría y su despliegue se eliminaron por decisión de producto (ODD `retirada-telemetria-y-restos-upstream`). `deploy/telemetry/` ya no existe y Axiom no envía telemetría. Se conserva el identificador RF-DIS-06 y el texto original solo por trazabilidad; el requisito no vincula.

Los nombres de servicio de despliegue de telemetría (`deploy/telemetry/axiom-telemetry.service`, `deploy/telemetry/axiom-telemetry-backup.service` y paneles de Grafana asociados) DEBEN resolverse hacia la identidad de distribución de Axiom.

#### Scenario: Los nombres de servicio de telemetría se renombran
- **DADO** las definiciones bajo `deploy/telemetry/`
- **CUANDO** se despliega el servicio
- **ENTONCES** los ficheros de unidad y paneles de Grafana citan la identidad de Axiom

---

### Requirement: Raíz de respaldos resuelta exclusivamente a través de internal/backup (RF-DIS-07)

Todo escritor de producción que cree o localice el directorio raíz de respaldos DEBE resolverlo invocando la función de resolución canónica exportada de `internal/backup` (`backup.BackupRootFor(homeDir)`), y NO DEBE construir esa ruta mediante un literal `filepath.Join` propio. Los respaldos que sobrevivan bajo `~/.gentle-ai/backups` DEBEN moverse una sola vez, en el arranque, a la raíz de respaldos de Axiom sin pisar los existentes y reescribiendo `root_dir` y `snapshot_path` de su manifiesto; `ListBackups` NO DEBE leer ninguna segunda raíz.

#### Scenario: Un escritor de producción resuelve la raíz a través del paquete
- **DADO** cualquier componente de backup, sync, run, upgrade o uninstall
- **CUANDO** necesita el directorio raíz de respaldos
- **ENTONCES** lo obtiene invocando la función de resolución canónica de `internal/backup`
(Previously: los respaldos de `~/.gentle-ai/backups` seguían siendo legibles por `ListBackups` sin migración obligatoria.)

### Requirement: Guarda ejecutable contra rutas literales de raíz de respaldos (RF-DIS-08)

El sistema DEBE mantener una prueba automatizada que analice los ficheros de producción de `internal/` y que falle si alguno construye la raíz de respaldos mediante literales directos en lugar de invocar la función de resolución canónica de `internal/backup`.

#### Scenario: La guarda pasa cuando todos los escritores usan la función canónica
- **DADO** que todos los escritores de producción vigilados resuelven la raíz a través de `internal/backup`
- **CUANDO** se ejecuta `go test ./cmd/axiom/...`
- **ENTONCES** la guarda `TestUserStateRootsResolveThroughOwningPackage` pasa sin señalar ningún literal
### Requirement: La Release No Publica el Bundle `review-provider-contract` (RF-DIS-09)

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
### Requirement: Procedencia, firma y CI de release con identidad axiom (RF-DIS-10)

El archivo de procedencia de release DEBE llamarse `axiom-release-provenance-v1.tar.gz` y sus esquemas DEBEN ser `axiom.release-provenance/v1` y `axiom.release-provenance/local-build`. Las claves de firma del runner DEBEN llamarse `axiom-release.key` y `axiom-release.pub`, el ejecutable de CI en Windows `axiom.exe` y las imágenes de los e2e `axiom-e2e-*`. `.goreleaser.yaml`, `internal/releasepolicy`, `internal/releaseprovenance` y los scripts de verificación y promoción NO DEBEN esperar ni producir ningún nombre `gentle-ai`. `scripts/promote-stable-preflight.sh` DEBE consultar la existencia de la release estable en el repositorio propio (`$GITHUB_REPOSITORY`).

#### Scenario: La procedencia y la firma usan la identidad axiom
- **DADO** una release simulada con los artefactos `axiom_*`
- **CUANDO** se ejecutan la verificación de assets y las pruebas de `internal/releasepolicy` e `internal/releaseprovenance`
- **ENTONCES** esperan `axiom-release-provenance-v1.tar.gz` con el esquema `axiom.release-provenance/v1`
- **Y** ningún fichero de política ni script de release contiene un nombre de artefacto `gentle-ai`

#### Scenario: La promoción consulta el repositorio propio
- **DADO** `GITHUB_REPOSITORY=IGutierrezZ/axiom`
- **CUANDO** `promote-stable-preflight.sh` comprueba si la release estable ya existe
- **ENTONCES** la consulta usa ese propietario y nombre de repositorio

