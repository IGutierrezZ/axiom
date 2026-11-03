# ODD · distribucion-un-comando (P1 de la hoja de ruta de visión de producto)

## Objetivo

Que cualquiera instale Axiom con **un solo comando de terminal** contra el repositorio
público `IGutierrezZ/axiom`, sin clonarlo:

- Unix: `curl -fsSL https://raw.githubusercontent.com/IGutierrezZ/axiom/main/scripts/install.sh | bash`
- Windows: `irm https://raw.githubusercontent.com/IGutierrezZ/axiom/main/scripts/install.ps1 | iex`

## Problema y porqué

Auditoría `docs/audits/2026-11-03-vision-producto.md` (rama `inc/auditoria-vision-producto`),
objetivo O1, huecos G1.1–G1.7. Hallazgo nuevo al mapear, **verificado**:

- `install.ps1` (y la vía `--method go` de `install.sh`) construyen el paquete con el
  propietario en minúsculas (`github.com/igutierrezz/axiom/v3/...`). `go get` falla:
  «module declares its path as: github.com/IGutierrezZ/axiom/v3 but was required as:
  github.com/igutierrezz/axiom/v3». Con el propietario en mayúsculas y minúsculas
  resuelve `v3.5.1`. **Hoy el instalador de Windows no funciona.**
- Existe la etiqueta `v3.5.1` pero la última release publicada en GitHub es `v3.5.0`.

## Decisiones

- **D1 (2026-11-04):** en Windows el canal oficial provisional es `go install` del módulo
  público; no se restauran los targets de Windows en GoReleaser (sigue el gate de
  `docs/release-signing.md`).
- Los identificadores de contrato `gentle-ai.*`, `GENTLE_AI_*`, los nombres de assets de
  release `gentle-ai-*` y el tap de Homebrew de `engram` y `gga` **no se tocan**: son
  contratos o pertenecen de verdad a Gentleman-Programming.

## Alcance autorizado

`scripts/install.sh`, `scripts/install.ps1`, `internal/update/**` (registro, instrucciones,
estrategia de upgrade del propio Axiom) y sus pruebas, `cmd/axiom/main.go` (comentarios),
`README.md`, `docs/quickstart.md`, `docs/platforms.md`, `docs/usage.md`.
Fuera: GoReleaser, políticas de release, `scripts/verify-release-distribution-policy.sh`,
`scripts/promote-stable-preflight.sh` (residuo de CI de release, se anota como seguimiento).

## Configuración

- TDD: **off** (fuente: `axiom.yaml` no lo configura). Comprobaciones funcionales normales.
- RDD: **off** (`axiom review mode status`: decidido por defecto).
- Ejecutor de pruebas: `go test` en proceso. **Prohibido compilar binarios sueltos** (Cortex).
- Entrega: `single-pr` (previsión ≈ 180 líneas, por debajo de 400).

## Tareas

- [ ] **T1 · `install.ps1` instalable con `irm | iex`** (ruta: delegada, escritor único T1–T4; disparador: 2+ ficheros no triviales).
  - Paquete con el propietario tal cual (`IGutierrezZ`), sin `ToLower()`; actualizar la prueba que fija el patrón y añadir una que impida volver a minúsculas.
  - Sin Go: mensaje accionable (`winget install GoLang.Go`, reabrir la terminal y repetir el comando).
  - Mantener el mensaje de retención de Windows y `go install github.com/IGutierrezZ/axiom/v3/cmd/axiom@latest` literal (lo exige `windows_distribution_policy_test.go`); ASCII sin BOM.
  - Comprobación: `go test ./internal/update/... ./internal/workspace/...`.
- [ ] **T2 · `install.sh` sin Homebrew ajeno ni `git`**
  - Quitar `BREW_TAP`, `BREW_FORMULA_REF`, `install_brew`, la autodetección de `brew` y su ayuda; el método por defecto es `binary`.
  - Quitar el requisito de `git`; `curl -fsSL` en cabecera y ayuda.
  - Misma corrección de mayúsculas en `--method go`.
  - Comprobación: `bash -n scripts/install.sh`; `go test ./internal/update/...`.
- [ ] **T3 · Actualización del propio Axiom sin tap ajeno**
  - `strategy.go`: no hacer `brew tap`/trust de Gentleman-Programming para la herramienta propia (sí para `engram` y `gga`); retirar la rama muerta `brew upgrade` del propio Axiom en `strategy.go` e `instructions.go`.
  - `registry.go`: `Owner`/`Repo` a `IGutierrezZ`/`axiom` (el `Name` se mantiene); comentarios de `cmd/axiom/main.go`.
  - Comprobación: `go test ./internal/update/... ./internal/app/... ./internal/installcmd/...`.
- [ ] **T4 · Documentación de instalación**
  - README y quickstart: las dos líneas como camino principal; `git clone` solo en «Contribuir / desarrollo».
  - Alinear `docs/platforms.md` y `docs/usage.md` (Homebrew, `go install`).
  - Comprobación: lectura estructural.
- [ ] **T5 · Release al día** — **requiere permiso explícito del usuario** (etiqueta y publicación son acciones externas). Revisar por qué `v3.5.1` no tiene release y publicar una desde `main` con P1 incluido.
- [ ] **T6 · Cierre**: PR, y al fusionar marcar P1 en el documento de auditoría.

## Criterios de aceptación

- `go get github.com/IGutierrezZ/axiom/v3@latest` resuelve (verificado) y los dos scripts usan esa ruta.
- Ningún camino de instalación o actualización del propio Axiom invoca el tap de otra organización.
- El README muestra las dos líneas de instalación como primera opción.
- Pruebas de `internal/update`, `internal/workspace`, `internal/app` e `internal/installcmd` en verde.

## Progreso y evidencias

- 2026-11-04: mapa de cambios hecho (delegado, solo lectura). Verificado con `go get` en un módulo temporal: minúsculas falla, mayúsculas resuelve `v3.5.1`.

## Seguimientos fuera de alcance

- `scripts/promote-stable-preflight.sh:60` consulta releases de `Gentleman-Programming/gentle-ai`.
- Firma Authenticode con Azure Artifact Signing a nombre de Hiberus para restaurar el binario de Windows.

## Siguiente paso

T1–T4 con un escritor delegado y un commit por tarea.
