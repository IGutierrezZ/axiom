# ODD: comprobación previa a la PR

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/guardas-previas-pr.md`.
> Espejo de recuperación en Engram: topic `odd/guardas-previas-pr/tasks`, proyecto `axiom`.
> Rama `inc/guardas-previas-pr`, worktree `C:\repos\axiom-wt\guardas-previas-pr\axiom`.

## Objetivo

Que las PRs dejen de fallar en la CI por guardas que nadie ejecuta en local: comprobarlas antes de abrir la PR, desde `scripts/axiom-worktree.{ps1,sh} pr`, y evitar desde el origen el código muerto que caza el deadcode ratchet.

## Problema

Entre el 25-nov a las 22:14 y el 27-nov hubo 99 ejecuciones de CI en PRs y 18 fallos (16 de 70 ramas). Unos 13 tienen la misma causa: guardas transversales que no corren en local, porque la regla de pruebas localizadas solo ejecuta los paquetes tocados.

- **Deadcode Ratchet (5):** funciones de producción que el producto no llama. Tres patrones: solo las llaman pruebas (#347, #350, #353), una variante `XForEvent` deja sin llamantes a la original (#323) y una retirada deja huérfanos (#318).
- **Meta-tests que vigilan todo el repositorio desde otro paquete (7):** el trinquete de rechazos de `internal/cli` (4), `TestRepositoryLegacyIDResidues` (2), `TestWriterInventory…`, `TestImportsDirection`, el canario y los estilos del dashboard y `TestToolchainMetaTestsRunInExactlyOneJob`.
- **gofmt (1).**

El deadcode ratchet no se puede ejecutar tal cual en este equipo: compila `deadcode.exe` en un directorio temporal en cada ejecución, y el EDR corporativo marca los ejecutables recién compilados en `%TEMP%`. `axiom.exe`, instalado con `go install` en `%USERPROFILE%\go\bin`, se ejecuta a diario sin problema.

## Por qué

Cada fallo se arregla con un commit pequeño, pero llega tarde: correo de fallo, nueva ronda de CI y otra revisión.

## Alcance

- `scripts/deadcode-ratchet.sh`: reutilizar un `deadcode` v0.49.0 ya instalado en `GOBIN` o `GOPATH/bin`; fuera de la CI, no compilarlo nunca en un directorio temporal.
- `scripts/pre-pr-check.sh` y `scripts/pre-pr-guards.txt`: gofmt de todo el repositorio, los meta-tests transversales por nombre y el deadcode ratchet.
- `scripts/axiom-worktree.ps1` y `scripts/axiom-worktree.sh`: `pr` ejecuta la comprobación del worktree del cambio antes de subir la rama; con fallo, ni sube la rama ni crea la PR.
- `internal/workspace/scripts_axiom_worktree_test.go`: pruebas del enganche.
- `AGENTS.md` y `CLAUDE.md`: reglas contra el código muerto y la comprobación previa.

Fuera de alcance: los fallos de plataforma (Linux frente a Windows), el job de la CI (no cambia) y la ejecución duplicada y cancelada de `PR Validation`.

## Restricciones

- Sin compilar binarios sueltos: solo `go test` de los paquetes tocados y `-run`; la batería completa, en la CI.
- TDD: desactivado (sin configuración de proyecto ni de sesión); se ejecutan las comprobaciones funcionales de cada tarea.
- Entrega `single-pr`, un commit por tarea. Previsión: unas 250 líneas.

## Tareas

- [x] **T1 · El ratchet reutiliza el deadcode instalado** — ruta: inline.
  - `installed_deadcode` busca `deadcode` en GOBIN o en el primer GOPATH/bin y comprueba con `go version -m` que es `golang.org/x/tools` v0.49.0. Sin él: en la CI (`CI` definido) se compila en temporal como antes; fuera, falla con `go install golang.org/x/tools/cmd/deadcode@v0.49.0`.
  - Evidencia: sin instalar, sale con 1 y la orden de instalación; instalado en `%USERPROFILE%\go\bin`, `bash scripts/deadcode-ratchet.sh` sale con 0 en 8 s («no new unreachable functions») y el binario sigue en su sitio después.
- [ ] **T2 · Script de comprobación previa y lista de guardas** — ruta: inline.
- [ ] **T3 · `pr` ejecuta la comprobación antes de subir la rama** — ruta: inline.
- [ ] **T4 · Reglas contra el código muerto para los agentes** — ruta: inline.

## Criterios de aceptación

- `scripts/deadcode-ratchet.sh` en local usa el `deadcode` instalado y, si falta o es de otra versión, falla con la orden exacta de instalación; en la CI se comporta como antes.
- `scripts/pre-pr-check.sh` sale con 0 en `main` y con distinto de 0 si falla gofmt, un meta-test de la lista (o no se ejecuta) o el ratchet.
- `pr` no sube la rama ni crea la PR cuando la comprobación falla; sin `scripts/pre-pr-check.sh` en la rama, avisa y sigue.
- `AGENTS.md` recoge las tres reglas contra el código muerto y `CLAUDE.md` la instalación única de `deadcode`.

## Comprobaciones

- `go test ./internal/workspace -run TestScriptsAxiomWorktree -count=1`
- `bash scripts/pre-pr-check.sh` en el worktree.
- `bash -n` de los scripts tocados.

## Progreso

Documento creado. Siguiente paso: T1.
