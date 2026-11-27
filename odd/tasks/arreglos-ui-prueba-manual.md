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
- [x] **T2 · Tabla de agentes sin desbordar** — ruta: inline.
  - Causa: la celda del agente es `th[scope="row"]` y heredaba las mayúsculas y el `nowrap` de las cabeceras de columna (275 px). Se le aplica el mismo estilo de fila que a `.eco-doctor` y `.eco-repo-policy`.
- [x] **T3 · Mensaje de `axiom` sin TTY** — ruta: inline.
  - Mensaje en castellano que explica que `axiom` abre el dashboard y ofrece `axiom ui`. Conserva `--version`, `axiom update` y `--help`, que exige RF-TUI-01 (`axiom-tui-branding`), así que no hay delta de spec.
- [x] **T4 · Línea en «Abrir en el editor» y etiquetas de `governance.*`** — ruta: inline.
  - `settings.Entry.Line` (línea del nodo clave, solo en las claves de solo lectura declaradas) → `SettingDTO.line` → `editorHref(configPath, line)` añade `:<línea>` si es un entero positivo. «Copiar ruta» sigue copiando la ruta sin línea.
  - `governanceLabels`: `language` → «Idioma», `shared_memory` → «Memoria compartida»; una clave desconocida conserva su ruta como etiqueta.
  - Cambia el destino que fija RF-WEB-18: delta MODIFIED en `## Spec viva`, integrado con `axiom archive integrate --odd`. Nota en `docs/usage.md`.

## Criterios de aceptación

- Los cuatro requisitos citados muestran su ID en la insignia y el sufijo en el título.
- La tabla de agentes no desborda su caja a 1024 px.
- `axiom` sin TTY explica en castellano que abre la UI y ofrece `axiom ui`, `--help` y `--version`.
- Cada clave de solo lectura declarada abre `axiom.yaml` en su línea; `governance.language` y `governance.shared_memory` tienen nombre legible.

## Spec viva

T4 cambia el destino del enlace «Abrir en el editor» que fija RF-WEB-18 (ahora lleva la línea de la clave) y da nombre legible a las claves conocidas de `governance`. T1, T2 y T3 corrigen la presentación sin cambiar ningún requisito (T3 conserva lo que exige RF-TUI-01).

### Capacidad: local-web-dashboard

```openspec
## MODIFIED Requirements

### Requirement: Pantalla «Ajustes del Proyecto» Generada desde el Catálogo, con Reglas por Topología e ISO y sin Controles de RDD (RF-WEB-18)

El dashboard DEBE ofrecer una pantalla «Ajustes del proyecto» con ruta por *hash* y entrada en la navegación. La pantalla DEBE generarse desde el catálogo que devuelve `GET /api/settings` (RF-WEB-19) sin conocer las claves: cada entrada se pinta con su etiqueta, descripción, tipo, valores permitidos y efecto descrito, y un control acorde (selector para vocabularios cerrados, casilla para booleanos, número, texto o lista). Todo texto del catálogo y del `axiom.yaml` DEBE insertarse como texto, nunca como HTML.

La pantalla DEBE mostrar de forma visible el aviso «Editar solo afecta a cambios nuevos» y comunicar que los cambios ya sellados conservan su instantánea de `kickoff.yaml`; ninguna acción de la pantalla DEBE modificar un `kickoff.yaml` existente. Al guardar, DEBE enviar la revisión leída, mostrar el error del validador (castellano) sin perder lo tecleado cuando se rechaza, y pedir recargar cuando la revisión está obsoleta. Retirar un valor DEBE enviarse como `null`. La pantalla DEBE enviar todos los cambios pendientes en una sola petición (lote atómico, RF-WEB-19). Junto a la ruta del fichero DEBE indicar que lo guardado queda sin confirmar en git, en el árbol de trabajo del repositorio que contiene `axiom.yaml` (en `multirepo`, el de specs), y que confirmarlo es decisión de la persona. Cuando el servidor informa `writable: false` (documento con anclas, alias, `<<` o varios documentos YAML), DEBE deshabilitar los controles de edición y mostrar el motivo.

Las claves con `editable: false` (`technical_context`, `workspace.root`, `skill_roots`, `governance.*`, `workspace.topology`, `workspace.specs_repository`) DEBEN mostrarse sin control de edición, con su valor y la acción «Abrir en el editor»: un enlace `vscode://file/<ruta absoluta de axiom.yaml>:<línea>` etiquetado «Abrir en el editor», que abre el fichero en la línea de la clave cuando está declarada (sin `:<línea>` cuando no lo está), más un botón «Copiar ruta» que copia esa ruta absoluta, sin línea. Las claves conocidas de `governance` DEBEN mostrarse con nombre legible (`language`: «Idioma»; `shared_memory`: «Memoria compartida»); una clave desconocida se muestra con su ruta. El servidor NO DEBE lanzar ningún proceso para abrir el editor.

La pantalla DEBE incluir un bloque «Preferencias personales» con el autofetch (RF-WEB-20) y `worktree_mode` y `worktree_root` (estos dos sobre `worktree-prefs`, RF-WEB-10), identificado como personal y no versionado. En `monorepo-embedded` la pantalla NO DEBE mostrar `roles`, `gate_policy`, `repositories` ni `acceptance_tests`; en `multirepo` DEBE mostrar `repositories[]`, `acceptance_tests` y `roles[].gate_policy` con `optional` entre sus opciones. En un proyecto ISO, `workflow.lane` NO DEBE ofrecer `odd`. La pantalla NO DEBE incluir ningún control de RDD (global, por proyecto o por clon), ni `commit_review`, ni `docs_publication`. Tampoco DEBE pintar el bloque `integrations`: DEBE enlazar a «Integraciones» (RF-WEB-33), que es donde se consulta y edita.

(Previously: el enlace «Abrir en el editor» era `vscode://file/<ruta absoluta de axiom.yaml>`, sin línea, y las claves de `governance` se etiquetaban con su ruta.)

#### Scenario: La pantalla se genera desde el catálogo
- **DADO** un catálogo con una entrada de vocabulario cerrado
- **CUANDO** se abre «Ajustes del proyecto»
- **ENTONCES** la entrada se pinta como selector con exactamente los valores que devolvió el servidor
- **Y** se muestran su descripción y su efecto

#### Scenario: Aviso de que solo afecta a cambios nuevos
- **DADO** un proyecto con un cambio ya sellado
- **CUANDO** se abre «Ajustes del proyecto»
- **ENTONCES** se muestra «Editar solo afecta a cambios nuevos»
- **Y** tras guardar `workflow.lane`, el `kickoff.yaml` del cambio sellado no cambia

#### Scenario: Cambiar el carril conserva comentarios y orden
- **DADO** un `axiom.yaml` comentado con `workflow.lane: ask`
- **CUANDO** el usuario elige `sdd` y guarda
- **ENTONCES** el servidor escribe el fichero solo con esa línea cambiada
- **Y** la pantalla muestra la revisión nueva y el valor guardado

#### Scenario: Valor inválido o revisión obsoleta no pierde lo tecleado
- **DADO** un valor rechazado por el validador o una revisión obsoleta
- **CUANDO** el usuario guarda
- **ENTONCES** la pantalla muestra el error en castellano y conserva el valor tecleado
- **Y** en el caso de la revisión obsoleta ofrece recargar

#### Scenario: Claves de solo lectura con «Abrir en el editor» y «Copiar ruta»
- **DADO** la pantalla abierta
- **CUANDO** se inspecciona `workspace.topology`, `technical_context` o `governance.*`
- **ENTONCES** se muestran sin control de edición
- **Y** aparece un enlace «Abrir en el editor» con destino `vscode://file/<ruta absoluta de axiom.yaml>:<línea de la clave>` y un botón «Copiar ruta» que copia la ruta sin línea
- **Y** `governance.language` y `governance.shared_memory` se etiquetan «Idioma» y «Memoria compartida»
- **Y** el servidor no lanza ningún proceso

#### Scenario: Clave de solo lectura sin declarar abre el fichero sin línea
- **DADO** un `axiom.yaml` sin `technical_context`
- **CUANDO** se inspecciona `technical_context`
- **ENTONCES** el enlace «Abrir en el editor» apunta a `vscode://file/<ruta absoluta de axiom.yaml>`, sin `:<línea>`

#### Scenario: Bloque «Preferencias personales»
- **DADO** un proyecto registrado y activo
- **CUANDO** se abre la pantalla
- **ENTONCES** el bloque «Preferencias personales» muestra el autofetch, `worktree_mode` y `worktree_root`
- **Y** indica que no se guardan en `axiom.yaml`

#### Scenario: Monorepo sin claves de multirepo
- **DADO** un proyecto `monorepo-embedded`
- **CUANDO** se abre la pantalla
- **ENTONCES** no aparecen `roles`, `gate_policy`, `repositories` ni `acceptance_tests`

#### Scenario: Multirepo con `gate_policy` incluido `optional`
- **DADO** un proyecto `multirepo`
- **CUANDO** se abre la pantalla
- **ENTONCES** aparecen `repositories[]`, `acceptance_tests` y `roles[].gate_policy`
- **Y** `gate_policy` ofrece `blocking`, `deferred` y `optional`

#### Scenario: ISO no ofrece `odd`
- **DADO** un proyecto con `sdd.iso: true`
- **CUANDO** se abre la pantalla
- **ENTONCES** `workflow.lane` no ofrece `odd`
- **Y** si un cliente envía `odd`, el servidor lo rechaza (RF-WEB-19)

#### Scenario: Ningún control de RDD ni de integraciones, con enlace a «Integraciones»
- **DADO** la pantalla abierta en cualquier topología
- **CUANDO** se recorren todos sus controles
- **ENTONCES** no existe ningún control de RDD, de `commit_review` ni de `docs_publication`
- **Y** no se pinta ningún control de `integrations.*`, sino un enlace a «Integraciones»

#### Scenario: Lo guardado queda sin confirmar en git
- **DADO** la pantalla abierta con la ruta de `axiom.yaml`
- **CUANDO** se inspecciona junto a esa ruta
- **ENTONCES** se indica que los cambios guardados quedan sin confirmar en git y que confirmarlos es decisión de la persona
- **Y** guardar no ejecuta ninguna operación de git

#### Scenario: Documento no editable deshabilita los controles
- **DADO** un `axiom.yaml` con anclas, alias, `<<` o varios documentos YAML
- **CUANDO** se abre la pantalla
- **ENTONCES** los controles de edición están deshabilitados y se muestra el motivo informado por el servidor

#### Scenario: Texto hostil se muestra como texto
- **DADO** un `workspace.name` con `<script>` en su valor
- **CUANDO** se abre la pantalla
- **ENTONCES** el valor aparece literal y no se interpreta como HTML
```

## Progreso

- Worktree creado con `scripts/axiom-worktree.ps1 new arreglos-ui-prueba-manual -Lane odd -Prefix fix`.
- T1: `go test ./internal/dashboard -run 'TestSpecParserAndRouterRuntime|TestEmbeddedModulesParse'` → ok (casos nuevos: retirado, sustituido y paréntesis normal con sufijo). El parser sobre `openspec/specs/` da 633 requisitos, todos con ID; los cuatro retirados, con insignia y el sufijo en el título.
- T2: medido en el navegador a 1024×800 con la regla inyectada: la tabla pasa de 711 px a 695 px (cabe en su caja) y la columna Agente, de 275 px a 146 px. `go test ./internal/dashboard -run 'TestEcosystemAgentsBlock|TestEcosystem|Platform'` → ok.
- T3: la prueba que exige `axiom ui` falló con el mensaje antiguo y pasa con el nuevo: `go test ./internal/app -run 'TestRunArgsNoArgument|TestRunArgsExplicitCommandsBypassTTYBoundary'` → ok; `go vet` y `gofmt -l` limpios. `TestBuiltBinaryClosedStdinRefusesBeforeUI` se omite en Windows (compila un binario); lo cubre la CI en Linux.
- T4: `go test ./internal/settings` → ok (líneas de las siete claves de solo lectura, 0 sin declarar, etiquetas de `governance`); `go test ./internal/dashboard -run 'Settings|Integrations|Spec|Living|Archive|TestEmbeddedModulesParse'` → ok (`TestSettingsRuntimeControlsAndChanges` con Node: `:16`, sin línea con 0, -1, `'7'`, 1,5 y `undefined`); `go vet` y `gofmt -l` limpios. `axiom archive integrate --odd arreglos-ui-prueba-manual --check` y después sin `--check` → `local-web-dashboard` compuesta (RF-WEB-18, un escenario más: 3186 en `INDEX.md`).

## Commits

| Tarea | Commit |
|---|---|
