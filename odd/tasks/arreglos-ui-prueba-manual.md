# ODD: arreglos de la prueba manual del dashboard

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/arreglos-ui-prueba-manual.md`.
> Espejo de recuperación en Engram: topic `odd/arreglos-ui-prueba-manual/tasks`, proyecto `axiom`.
> Rama `fix/arreglos-ui-prueba-manual`, worktree `C:\repos\axiom-wt\arreglos-ui-prueba-manual\axiom`.

Tipo: bug

## Objetivo

Corregir los cuatro defectos menores que encontró la prueba manual de `axiom ui` del 2026-11-26 sobre `main` (56e2de995).

## Problema

1. **Visor de specs vivas.** Un requisito cuyo encabezado lleva un sufijo tras el ID (`### Requirement: Título (RF-TOP-05) [SUPERSEDED / DEPRECADO]`, `… (RF-DIS-06) — RETIRADO`) no extrae el ID: no hay insignia y el ID queda dentro del título. Afecta a RF-DIS-06, RF-TUI-03, RF-RCL-02 y RF-TOP-05 (4 de 576).
2. **Ecosistema → Agentes.** Con la ventana a unos 1024 px la tabla mide más que su caja (711 frente a 695 px): la columna Acción se corta y aparece desplazamiento horizontal.
3. **`axiom` sin TTY.** El error sale en inglés, menciona `axiom update` y no dice que `axiom` abre la UI ni que `axiom ui` es la vía sin terminal (`internal/app/app.go`).
4. **Ajustes.** «Abrir en el editor» apunta a `vscode://file/…/axiom.yaml` sin línea en todas las claves de solo lectura; y `governance.language` y `governance.shared_memory` no tienen nombre legible (la etiqueta repite la clave).

## Alcance

- `internal/dashboard/assets/js/core/spec.js` y su prueba de ejecución en `internal/dashboard/knowledge_assets_test.go`.
- `internal/dashboard/assets/style.css` (tabla de agentes).
- `internal/app/app.go` y su prueba.
- `internal/settings` (línea de la clave y etiquetas de `governance.*`), `internal/dashboard/settings_api.go`/`types.go` y `internal/dashboard/assets/js/core/settings.js` con sus pruebas.

Fuera de alcance: que el enlace del editor fije VS Code (decisión de diseño de C-06-e); el desfase «70 archivados» del inicio frente a los 175 de la lista.

## Restricciones

- Sin compilar binarios sueltos: solo `go test` de los paquetes tocados.
- TDD: desactivado (sin configuración de proyecto ni de sesión); se ejecutan las comprobaciones funcionales de cada paquete.

## Tareas

- [x] **T1 · ID con sufijo en el visor de specs** — ruta: inline.
  - `splitRequirementId` en `core/spec.js`: si el título no acaba en el ID, separa un sufijo de estado (raya, guion con espacio o corchete, sin paréntesis) y lo devuelve al título.
- [ ] **T2 · Tabla de agentes sin desbordar** — ruta: inline.
- [ ] **T3 · Mensaje de `axiom` sin TTY** — ruta: inline.
- [ ] **T4 · Línea en «Abrir en el editor» y etiquetas de `governance.*`** — ruta: inline.

## Criterios de aceptación

- Los cuatro requisitos citados muestran su ID en la insignia y el sufijo en el título.
- La tabla de agentes no desborda su caja a 1024 px.
- `axiom` sin TTY explica en castellano que abre la UI y ofrece `axiom ui`, `--help` y `--version`.
- Cada clave de solo lectura declarada abre `axiom.yaml` en su línea; `governance.language` y `governance.shared_memory` tienen nombre legible.

## Impacto en la spec

Sin impacto en la spec
Motivo: corrige la presentación de requisitos y ajustes ya especificados; no cambia ningún comportamiento descrito.

## Progreso

- Worktree creado con `scripts/axiom-worktree.ps1 new arreglos-ui-prueba-manual -Lane odd -Prefix fix`.
- T1: `go test ./internal/dashboard -run 'TestSpecParserAndRouterRuntime|TestEmbeddedModulesParse'` → ok (casos nuevos: retirado, sustituido y paréntesis normal con sufijo). El parser sobre `openspec/specs/` da 633 requisitos, todos con ID; los cuatro retirados, con insignia y el sufijo en el título.

## Commits

| Tarea | Commit |
|---|---|
