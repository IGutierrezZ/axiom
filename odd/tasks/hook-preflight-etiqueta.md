# ODD: El hook del preflight valida por etiqueta, no por posición (C-01-a)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/hook-preflight-etiqueta.md`.
> Espejo de recuperación en Engram: topic `odd/hook-preflight-etiqueta/tasks`, proyecto `axiom`.
> Plan maestro: `docs/plan/backlog-2026-11/01-poda-friccion.md` (C-01-a) y `README.md` (apartado 6.2).

## Objetivo

Que un preflight SDD bien contestado valga con cualquier orden de opciones y de preguntas, y que todo rechazo diga exactamente qué corregir.

## Problema

`resolveSDDPreflightHookBlock` (`internal/cli/sdd_preflight_hook.go`) compara las etiquetas de cada pregunta por posición y mapea la respuesta a su token por índice. Si el agente presenta «Automatic» antes que «Interactive», el hook deniega aunque el usuario haya contestado bien. Además, `resolveSDDPreflightTranscriptAuthority` descarta la causa (`continue`) y el `deny` siempre es el mensaje genérico, así que el modelo tiene que adivinarla.

## Evidencia

Escaneo de las transcripciones reales de este equipo (`~/.claude/projects/C--repos-axiom/**/*.jsonl`, 2026-11-17): 21 llamadas `AskUserQuestion` de preflight; 20 en orden canónico y 1 con `["Automatic","Interactive"]`. Ninguna etiqueta llevaba el sufijo «(Recommended)»: la inferencia del plan sobre ese sufijo no se reproduce, pero la guía de `AskUserQuestion` lo pide al recomendar una opción, así que se tolera. También aparecen respuestas libres («un pr con size expection»), que deben rechazarse nombrando la pregunta y la etiqueta.

## Decisiones

- Cada pregunta se identifica por su marcador `Axiom SDD preflight N/3:` (o el prefijo heredado), no por su posición en el array; cada respuesta, por su etiqueta, con igualdad de conjuntos entre las opciones ofrecidas y las canónicas. Se mapea etiqueta → token.
- Comparación de etiquetas tolerante a la presentación: espacios exteriores, mayúsculas y el sufijo final «(Recommended)». Paridad con `normalizeSDDPreflightAnswer` del plugin de OpenCode (trim y plegado de mayúsculas).
- El número de preguntas sigue siendo exactamente 3 (C-01-b lo hace variable). La estrategia de PR sigue en el preflight (P10 la retira en C-02-c).
- El `deny` nombra la causa del último preflight reconocido y no válido: pregunta que falta o repetida, etiqueta fuera del dominio, multiselección o respuesta ambigua.
- Se conservan intactos los rechazos de autoridad: hijo, texto del modelo y payload sin transcripción.

## Fuera de alcance

Quitar preguntas del preflight (C-01-b), cambiar el contrato de autoridad, el plugin de OpenCode [COMPARTIDO] y la variante de chat sin UI nativa.

## Tareas

- [x] **T1 · Validación por marcador y etiqueta** — `internal/cli/sdd_preflight_hook.go` y su test. Ruta: inline (1 fichero de código + test, diseño ya cerrado).
  - Aceptar opciones y preguntas en cualquier orden con las etiquetas canónicas y producir el mismo bloque `## SDD Session Preflight`.
  - Tolerar el sufijo «(Recommended)», espacios y mayúsculas.
  - `TestSDDPreflightHookDeniesReorderedCanonicalLabels` pasa a aceptar el orden invertido.
- [x] **T2 · Rechazo con causa concreta** — mismo fichero y test. Ruta: inline.
  - `resolveSDDPreflightTranscriptAuthority` devuelve la causa del último preflight reconocido e inválido y el `deny` la incluye.
  - Una etiqueta fuera del dominio produce un `deny` que nombra la pregunta y la etiqueta.
- [ ] **T3 · Texto proyectado sin exigir orden** — `internal/components/sdd/session_preflight.go:16` y `testdata/golden/sdd-opencode-multi-settings.golden`. Ruta: inline (mecánico).
  - Sustituir «Keep options in the exact order below.» por una frase que diga que el runtime enlaza por etiqueta.

## Criterio de terminado

- Un preflight con las opciones en cualquier orden y etiquetas canónicas produce el mismo bloque que en orden canónico.
- Una etiqueta fuera del dominio produce un `deny` cuyo texto nombra la pregunta y la etiqueta.
- Siguen denegando los casos de autoridad (hijo, texto del modelo, payload sin transcripción).
- `go test ./internal/cli/ -run SDDPreflight` en verde, y los paquetes afectados por T3 en verde.

## Configuración

- TDD: no configurado (`axiom.yaml` sin clave TDD); comprobaciones funcionales ordinarias. Runner: `go test`.
- RDD: `off (decided by default)`; sin revisión nativa.
- Entrega: una sola PR (`exception-ok` si pasa de 400 líneas, con `size:exception`). Previsión: ~250 líneas.

## Progreso

- Worktree `C:\repos\axiom-wt\hook-preflight-etiqueta\axiom`, rama `inc/hook-preflight-etiqueta` desde `a7fb0ed9`.
- Espejo Engram: **pendiente** (el modo automático no pudo revisar `mem_save` porque un hook modifica la llamada).

Plan stop: approved 2026-11-17; implementation continues

## Siguiente paso

T1.
