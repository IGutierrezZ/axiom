# ODD: El tablero de Incrementos muestra el trabajo ODD y descarta lo ya integrado

> Documento vivo ODD. Fichero autoritativo: `odd/tasks/tablero-trabajo-odd.md`. Worktree `C:\repos\axiom-wt\tablero-trabajo-odd\axiom`, rama `inc/tablero-trabajo-odd`. Espejo Engram: `odd/tablero-trabajo-odd/tasks`.

## Objetivo

Que el tablero de Incrementos del dashboard (`GET /api/increments`) muestre también las *features* ODD en curso en cada worktree y no presente como trabajo en curso lo que ya está integrado en `origin/main`.

## Problema y porqué

- Con 9 worktrees abiertos, el tablero solo mostraba 1 incremento: solo lista cambios SDD (`openspec/changes/<n>`), y 8 de los 9 worktrees son trabajo ODD (`odd/tasks/<f>.md`).
- El registro de ramas (`.git/axiom/branch/v1/`) guarda todos esos worktrees en el carril `sdd`, aunque sean ODD (`scripts/axiom-worktree.ps1` no pasa `--lane odd`), así que el carril del registro no sirve para distinguirlos: hay que mirar el contenido.
- Un worktree cuya rama ya está contenida en `origin/main` y no tiene cambios sin commit (hoy `bloqueos-prueba-multirepo-r2`) no es trabajo en curso; mostrarlo como activo sería información errónea.

## Decisiones

- **Detección ODD por contenido:** para cada worktree registrado (carriles `sdd` y `odd`) que exista en disco, una *feature* ODD activa es un `odd/tasks/<f>.md` que difiere de la base de integración (`git diff --name-only <base>...HEAD` más los cambios sin commit). Si alguno se llama como el slug del worktree, solo ese; si no, los añadidos en la rama; si no, los modificados.
- **Lo ya integrado:** si la rama del worktree es ancestro de la base y no tiene cambios sin commit, no aporta ninguna *feature* activa y el listado añade un aviso en `X-Axiom-Notes` («worktree ya integrado en origin/main; ciérralo con `axiom branch done`»). Los documentos ODD del checkout principal se listan como integrados (tipo `archived`).
- **Fase ODD en las columnas existentes:** sin tareas → `explore`; con tareas pendientes → `apply`; todas marcadas y sin integrar → `verify`; integrado → `archive`.
- **Identificación:** `change_type: "odd"` en `IncrementSummaryDTO`, chip «ODD» y un filtro «ODD» en el tablero.
- **Sin red:** la base es `refs/remotes/origin/<integración>` o la rama local, como `resolveBase`; no se hace `fetch`.

## Alcance autorizado

`internal/dashboard` (servicio, tipos, assets JS/HTML y tests), y `internal/branch/engine` si hace falta un helper de listado o de ancestro. Sin cambios en la TUI salvo lo necesario para que la prueba de paridad siga siendo correcta.

## Restricciones

- TDD: desactivado (no hay configuración de TDD en `axiom.yaml` ni elección explícita); se ejecutan las comprobaciones funcionales ordinarias.
- Nada de binarios sueltos: las pruebas, en proceso con `go test` (Cortex XDR borra `axiom.exe`).
- ~400 líneas por tarea como heurística orientativa, no como límite.

## Tareas

- [x] **T1 · Backend: detección de *features* ODD y de worktrees ya integrados** (`internal/dashboard`, `internal/branch/engine`). Parser de `odd/tasks/*.md` (título `# ODD…`, línea `Estado`, casillas `- [ ]`/`- [x]` y `### [x]`), listado por worktree, fase, aviso de integrado y documentos del checkout principal como integrados. Tests Go con repos temporales. Ruta: delegada (writer único; toca 2+ ficheros no triviales).
- [ ] **T2 · Frontend: chip y filtro ODD** (`internal/dashboard/assets`). Chip «ODD» en tarjeta y lista, filtro «ODD» con contador, subtítulo que no diga solo SDD. Tests de assets existentes ampliados. Ruta: delegada (mismo writer).
- [ ] **T3 · Verificación manual** contra el repositorio real: el listado muestra los 8 worktrees ODD activos, `carril-sdd-versatil` como SDD y avisa de `bloqueos-prueba-multirepo-r2` como integrado. Ruta: inline.

## Criterios de aceptación

- `GET /api/increments` devuelve las *features* ODD activas de cada worktree con `change_type: "odd"`, rama, ruta y progreso de tareas.
- Un worktree integrado y limpio no aparece como activo y genera un aviso.
- Los cambios SDD se listan igual que antes.
- `go test ./internal/dashboard/... ./internal/branch/...` en verde.

## Comprobaciones aplicables

- `go test ./internal/dashboard/... ./internal/branch/... -count=1`
- `go vet ./internal/dashboard/... ./internal/branch/...`

## Progreso y evidencias

- 2026-11-05: worktree creado con `scripts/axiom-worktree.ps1 new tablero-trabajo-odd`. Mapa del listado actual hecho (endpoint `handleIncrements` → `GetIncrements` → `registeredIncrements` / `inspectIncrement`; frontend `screens/incrementos.js`, `core/increment.js`).

- 2026-11-05 · T1 hecha (ruta: delegada, writer único; dispara el disparador de escritura por tocar 2+ ficheros no triviales). `internal/branch/engine/odd_worktrees.go` (`ODDWorktrees`, `SelectODDDocs`), `internal/dashboard/odd_listing.go` (parser, fase, listado, archivados del checkout principal), `GetIncrements` los agrega y `IncrementSummaryDTO` gana `title`.
  - Regla de «integrado» elegida: la rama (`HEAD` del worktree) es ancestro de la base **y** no hay cambios sin confirmar en el worktree **y** la punta de la rama difiere de la de la base. Un worktree recién creado cumple las dos primeras pero tiene la misma punta, así que no se avisa. Límite conocido: una rama integrada por avance rápido deja la misma punta que la base y no se distingue de una recién creada (no se avisa); la integración con commit de fusión sí se detecta. Los worktrees sucios (incluidos ficheros sin seguimiento ajenos a `odd/tasks`) se consideran en curso.
  - Los carriles `sdd` y `odd` del registro se enumeran los dos y se deduplica por raíz de worktree; el aviso de worktree desaparecido solo lo emite este listado para el carril `odd` (el del carril `sdd` ya lo da `RegisteredChanges`).
  - Parser: título `# ODD: …`/`# ODD · …`, casillas `- [ ]`/`- [x]` sin sangría y encabezados `### [x]`; las casillas con sangría solo cuentan si no hay ninguna de primer nivel.
  - Paridad TUI: la prueba `TestTUIListsTheSameChangesAsGetIncrements` compara solo los cambios SDD (se ignora `change_type: "odd"`).
  - Comprobaciones: `go test ./internal/dashboard/... ./internal/branch/... -count=1` → ok (dashboard, branch, engine, gitcli, registry); `go vet ./internal/dashboard/... ./internal/branch/...` → sin salida; `gofmt -l internal/` → sin salida.

## Siguiente paso

T2 (frontend: chip y filtro ODD) y después T3.
