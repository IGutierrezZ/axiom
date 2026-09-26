# ODD: Desacople de Gentle AI, Eliminación de Fallbacks y Canonicidad Axiom

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/desacople-gentle-ai-y-canonicidad-axiom.md`.
> Espejo de recuperación en Engram: topic `odd/desacople-gentle-ai-y-canonicidad-axiom/tasks`, proyecto `axiom`.

## Objetivo

Desacoplar de forma exhaustiva y canónica el ecosistema de Axiom respecto a Gentle AI en cuatro ejes coordinados:
1. Eliminar toda fuga de comandos e instrucciones en mensajes al usuario (CLI y TUI), asegurando que cualquier sugerencia guíe exclusivamente hacia `axiom`.
2. Eliminar completamente la capa de fallbacks hacia atrás (variables de entorno `GENTLE_AI_*`, rutas heredadas `~/.gentle-ai/` y el binario shim `cmd/gentle-ai`), aplicando la premisa de diseño de que los entornos operan de forma limpia y directa contra Axiom ("si algo falla, se arregla en origen").
3. Migrar formalmente los esquemas de datos y contratos de protocolo (`*.sdd-status`, `*.review-integration`, `*.review-transaction`, marcadores de plantillas) a la identidad canónica `axiom.*`, adaptando esquemas, fixtures y validadores de versión.
4. Renombrar canónicamente las skills de `skills/` y sus referencias de activación (de `gentle-ai-*` a `axiom-*`), preservando intactos los metadatos de autoría (`author: gentleman-programming`) y los identificadores de código Go internos.

## Problema

1. **Fugas de usuario (Punto 1):** Diversos flujos de error y continuación en `internal/sddstatus/status.go`, `internal/sddtaskresult/handoff.go` y la TUI en `internal/tui/model.go` todavía ordenan al usuario ejecutar `gentle-ai sdd-status`, `gentle-ai sdd-continue` o `gentle-ai review mode`, generando confusión sobre el binario canónico.
2. **Sobrecarga de fallbacks innecesarios (Punto 2):** El sistema mantiene una bifurcación constante para leer variables `GENTLE_AI_*` si no existen las de `AXIOM_*`, así como para inspeccionar directorios legados `~/.gentle-ai/`. Esta redundancia oculta errores de configuración en lugar de forzar entornos limpios y consistentes.
3. **Identidad de contratos no migrada (Punto 3):** Los protocolos de comunicación interna entre agentes y despachadores siguen atados al prefijo `gentle-ai.` en sus constantes de esquema, firmas hash de transacción y marcadores en templates (como `<!-- gentle-ai:sdd-... -->`).
4. **Skills con nombres legados (Punto 4):** Las carpetas y descriptores de skills como `gentle-ai-bench` y `gentle-ai-collab-perfect` mantienen prefijos antiguos en el índice y en los archivos `SKILL.md`, desalineados con el catálogo de Axiom.

## Alcance Autorizado

- `internal/sddstatus/` e `internal/sddtaskresult/`: Corrección de textos de error, sugerencias de comando y prefijos de fallo (`AXIOM_SDD_FAILURE `).
- `internal/tui/`: Corrección de mensajes de pantalla, desinstalación y configuración en TUI.
- `internal/system/env.go`: Retirada de fallbacks automáticos hacia variables `GENTLE_AI_*`. Soporte estricto para variables canónicas `AXIOM_*`.
- `internal/system/user_paths.go` e `internal/state/state.go`: Eliminación de rutas de búsqueda y métodos legados `~/.gentle-ai/`. Operación exclusiva sobre `~/.axiom/`.
- `cmd/gentle-ai/`: Retirada o inhabilitación estricta del binario pasarela.
- `internal/reviewtransaction/` y `contracts/`: Migración de contratos a `axiom.*` (`axiom.review-integration/v2`, `axiom.review-transaction/v1`, `axiom.sdd-status/v1` o `v2`).
- `skills/`: Renombrado de directorios (`gentle-ai-bench` → `axiom-bench`, `gentle-ai-collab-perfect` → `axiom-collab-perfect`), atributo `name:` en frontmatter y referencias de activación en `AGENTS.md`.

## Restricciones

- **No tocar metadatos de autoría:** Los campos de frontmatter YAML `metadata.author: gentleman-programming` (o autores originales de skills) NO se deben modificar.
- **No alterar identificadores de código Go no contractuales:** No renombrar tipos, structs, constantes o enums internos de Go que no afecten directamente a los contratos de serialización o cadenas de visualización acordadas.
- **Aislamiento en Worktree:** La implementación técnica de código de este ODD se ejecutará en un worktree dedicado (`scripts/axiom-worktree.ps1 new desacople-gentle-ai`) una vez autorizado el inicio.
- **Idioma Obligatorio:** Español (castellano peninsular con tuteo profesional) para reportes, descripciones de cambio y mensajes de usuario.

---

## Tareas

- [x] **T1 · Saneamiento de mensajes de terminal y UI (Punto 1)**
  - [x] Sustituir en `internal/sddstatus/status.go` todas las cadenas de instrucción (`Run gentle-ai sdd-status...`, `Run gentle-ai sdd-continue...`, `rerun gentle-ai sdd-status...`) por sus equivalentes con `axiom`.
  - [x] Actualizar en `internal/sddtaskresult/handoff.go` la síntesis del comando de continuación para emitir `axiom sdd status ...`.
  - [x] Corregir en `internal/tui/model.go` y pantallas los mensajes de error de RDD (`Retry with axiom review mode enable...`) e instrucciones de Homebrew (`brew uninstall axiom`).

- [x] **T2 · Supresión total de capas de fallback (Punto 2)**
  - [x] Limpiar `internal/system/env.go` eliminando la resolución en cascada hacia variables `GENTLE_AI_*` (`Getenv` responderá únicamente a las claves `AXIOM_*` solicitadas).
  - [x] Eliminar en `internal/system/user_paths.go` los métodos `LegacyDir`, `LegacyStatePath`, `LegacyCacheDir` y `LegacyBinDir`.
  - [x] Limpiar en `internal/state/state.go` y `cmd/axiom/main.go` cualquier lectura o advertencia sobre `~/.gentle-ai/backups/`.
  - [x] Retirar o convertir en fallo inmediato sin redirección el paquete `cmd/gentle-ai`.

- [x] **T3 · Migración formal de contratos, schemas y marcadores de plantilla (Punto 3)**
  - [x] Actualizar en `internal/sddstatus/` los contratos de estado a `axiom.sdd-status/v1` (o `v2`) y `axiom.sdd-integration.consent/v1`.
  - [x] Actualizar en `internal/sddtaskresult/` el prefijo `HandoffPrefix = "AXIOM_SDD_FAILURE "` y el schema `axiom.sdd-task-result-failure/v1`.
  - [x] Migrar en `internal/reviewtransaction/` las constantes de schema a `axiom.review-transaction/v1`, `axiom.review-targeted-validation-request/v1` y los hashes criptográficos asociados.
  - [x] Renombrar los marcadores en plantillas e instrucciones de `<!-- gentle-ai:sdd-... -->` a `<!-- axiom:sdd-... -->`.
  - [x] Actualizar las definiciones JSON de schemas en `contracts/` y sus fixtures para validar los nuevos contratos.

- [x] **T4 · Renombrado de skills y actualización de referencias (Punto 4)**
  - [x] Renombrar carpetas de skills en `skills/`:
    - `skills/gentle-ai-bench/` → `skills/axiom-bench/`
    - `skills/gentle-ai-collab-perfect/` → `skills/axiom-collab-perfect/`
    - Actualizar `name: axiom-branch-pr` en `skills/branch-pr/SKILL.md`
    - Actualizar `name: axiom-chained-pr` en `skills/chained-pr/SKILL.md`
  - [x] En los ficheros `SKILL.md`, actualizar el valor `name:` y los textos descriptivos donde se mencione la invocación de la skill, asegurando que `metadata.author: gentleman-programming` quede intacto.
  - [x] Actualizar el índice de skills en `AGENTS.md` reflejando las nuevas rutas y nombres canónicos.

- [ ] **T5 · Actualización de pruebas unitarias, golden files y verificación**
  - [ ] Actualizar los tests unitarios afectados por la eliminación de fallbacks en `internal/system/env_test.go` y `user_paths_test.go`.
  - [ ] Actualizar las aserciones de tests y golden files en `internal/sddstatus/`, `internal/sddtaskresult/`, `internal/reviewtransaction/` y `testdata/golden/`.
  - [ ] Ejecutar `go test ./...` y `go vet ./...` para certificar que el repositorio compila limpiamente y todas las suites pasan en verde.
