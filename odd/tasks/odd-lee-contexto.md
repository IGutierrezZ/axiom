# ODD: ODD lee el contexto técnico al explorar (odd-lee-contexto)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/odd-lee-contexto.md`. Espejo Engram: `odd/odd-lee-contexto/tasks`.
> Rama: `inc/odd-lee-contexto` · worktree `C:\repos\axiom-wt\odd-lee-contexto\axiom`. Cambio C-04-b del plan maestro (`docs/plan/backlog-2026-11/04-consumo-agentes.md`), decisión P6 del apartado 6.2 del README.

## Objetivo

Que ODD lea el `entry` del contexto técnico mantenido en el paso 2 «Explore» y siga actualizando la documentación mantenida en el commit de la unidad de trabajo (paso 7), obteniendo los locators con un comando de workspace que no depende de un cambio SDD.

## Problema

El paso 2 de ODD (`internal/components/agentguidance/routing.go:72`) no menciona el contexto técnico: ODD actualiza en el paso 7 documentación que nunca ha leído. Además, el paso 7 toma los locators de `axiom sdd status --json`, que sin `--change` selecciona un cambio activo y, con 0 o varios cambios activos, devuelve un status bloqueado con razones ajenas a ODD («No active OpenSpec changes», «selection is ambiguous»).

## Por qué

Leer antes de escribir: la documentación mantenida se actualiza con conocimiento de lo que dice. El comando propio evita el ruido del status de SDD (P6).

## Decisiones

- **P6 (usuario, 2026-11-17): comando nuevo de workspace.** Adaptador en `cmd/axiom/main.go` e `internal/cli/` que reutiliza `projectTechnicalContext` (`internal/sddstatus/workspace_declarations.go:57`) sin duplicarla; ODD lo usa y no depende de `axiom sdd status --json`.
- «lo segundo me vale» (usuario): ODD no adopta el rigor de SDD; sin `context-impact.md` ni verificación de integración del delta.
- Solo se lee el contexto **mantenido** (criterio de terminado: «lectura del `entry` del contexto técnico mantenido (y las rutas bajo demanda)»).
- **Diseño del comando (de la sesión, no cambia el alcance):** `axiom workspace context [--cwd <dir>] [--json]`.
  - Resuelve el workspace con `workspace.Locate(cwd)` (la misma resolución que `axiom sdd status` sin raíz explícita) y proyecta con `projectTechnicalContext`.
  - JSON: `{"schemaName":"axiom.workspace-context","schemaVersion":1,"workspaceRoot":"<abs>","technicalContext":{…}|null}`; con `technicalContext` de la misma forma que el campo del status (`maintained`/`reference`, `paths` absolutas, `entry` si se declaró).
  - Sin declaración: `technicalContext: null` y código 0.
  - Declaración inválida: falla cerrado (RF-CTX-20): `technicalContext: null`, campo `invalid` con el motivo y la indicación de `axiom workspace validate`, y código distinto de cero.
  - Sin `--json`: texto breve en español con `entry` y `paths` de cada modo, o «sin contexto técnico declarado».
- El paso 7 deja de citar `technical_context.maintained` «in `axiom.yaml`» como fuente del locator y remite al comando nuevo. La frase de RDD del paso 7 (`routing.go:77`, «The native review candidate…») es de C-01-c (apartado 6.3 del README): **no se retira**.
- El test `TestRenderRoutingClosesODDWithTheMaintainedTechnicalContext` prohibía mencionar el contexto técnico fuera del paso 7 (decisión deliberada de inc-24, RF-RUT-14: «El cierre NO DEBE alterar los demás pasos»). Se revierte a propósito por C-04-b; el commit lo explica.
- La spec viva se modifica (RF-RUT-14 y RF-CTX-03) en vez de añadir requisitos nuevos, para no asignar un identificador nuevo antes de C-03-b.

## Alcance autorizado

- `internal/sddstatus/`: función exportada fina sobre `projectTechnicalContext` y tests (incluida la verificación pendiente del README: `axiom sdd status --json` proyecta `technicalContext` con 0 y con varios cambios activos).
- `internal/cli/`: adaptador del comando y tests en proceso.
- `cmd/axiom/main.go`: grupo `workspace` (subcomando `context`) y texto de ayuda.
- `docs/usage.md` (lista de comandos), si lista `workspace validate`.
- `internal/components/agentguidance/routing.go` (pasos 2 y 7) y `routing_test.go`.
- `## Spec viva` de este documento e integración con `axiom archive integrate --odd odd-lee-contexto`.

Fuera de alcance: delta `context-impact.md` en ODD; `axiom archive context`; cambios en SDD y en sus skills (C-04-c es `sdd-propose`); la frase de RDD del paso 7 (C-01-c); goldens (el apartado 4 del README dice que C-04-b no los toca: si un golden falla, se para y se avisa).

## Modo TDD

Desactivado: `openspec/config.yaml` declara `strict_tdd: false` y no hay elección del usuario. Comprobaciones funcionales ordinarias con `go test` troceado por paquetes o `-run` (sin binarios sueltos).

## Tareas

- [x] **T1 · Proyección del contexto sin cambio SDD** (ruta: delegada, un escritor para T1-T3; disparador: 4+ ficheros no triviales en tres paquetes)
  - Función exportada en `internal/sddstatus` que resuelve la ubicación desde un `cwd` y devuelve raíz del workspace, proyección (`projectTechnicalContext`) y error de declaración si la hay.
  - Test de la función: dos modos, un modo, sin declaración, declaración inválida.
  - Test de la verificación pendiente: `Resolve` sin `--change` proyecta `technicalContext` con 0 y con 2 cambios activos (status bloqueado, pero con el campo).
  - Comprobación: `go test ./internal/sddstatus/ -run '<tests nuevos>'` y el paquete entero si cabe.
- [x] **T2 · Comando `axiom workspace context`** (ruta: delegada, mismo escritor)
  - Adaptador en `internal/cli/` (`--cwd`, `--json`), cableado en el grupo `workspace` de `cmd/axiom/main.go`, ayuda y `docs/usage.md`.
  - Tests en proceso: JSON con contexto, sin declaración (`null`, código 0), inválida (código distinto de cero), texto.
  - Comprobación: `go test ./internal/cli/ -run '<tests nuevos>'` y `go test ./cmd/axiom/ -run '<ayuda>'` si hay test de ayuda.
- [x] **T3 · Doctrina ODD: paso 2 lee, paso 7 remite al comando** (ruta: delegada, mismo escritor)
  - Paso 2: leer `technicalContext.maintained.entry` de `axiom workspace context --cwd <repo> --json` primero y `paths` bajo demanda; sin contexto mantenido declarado no lee nada más.
  - Paso 7: el locator sale del comando nuevo, no de `axiom sdd status --json` ni de «`axiom.yaml`»; se conserva la frase de RDD.
  - Reescribir la regla del test: fija la frase nueva en el paso 2 y en el paso 7, idéntica para todos los adaptadores; ningún otro paso (0, 1, 3-6) la menciona.
  - Comprobación: `go test ./internal/components/agentguidance/...`.
- [x] **T4 · Spec viva** (ruta: en línea)
  - MODIFIED RF-RUT-14 (`organic-agent-trigger-rules`) y RF-CTX-03 (`project-technical-context`) en `## Spec viva`; integración con `axiom archive integrate --odd odd-lee-contexto`.
  - Comprobación: la integración termina sin rechazo y repetirla da `already_integrated`.

## Criterio de terminado

El orquestador renderizado para cada runtime incluye en el paso 2 de ODD la lectura del `entry` del contexto técnico mantenido (y las rutas bajo demanda), con la forma exacta de obtener los locators (`axiom workspace context --cwd <repo> --json`); el test de routing fija la nueva frase en el paso 2 y en el paso 7; `go test ./internal/components/agentguidance/...` en verde; el test de `sddstatus` cubre 0 y varios cambios activos.

## Entrega

Una sola PR (`single-pr`) con etiqueta `type:feat`; previsión, unas 300-450 líneas (si pasa de 400, `size:exception`). Push, PR y fusión los decide el usuario.

## Spec viva

### Capacidad: organic-agent-trigger-rules

```openspec
## MODIFIED Requirements

### Requirement: El cierre de ODD mantiene al día el contexto técnico mantenido (RF-RUT-14)

Las instrucciones de enrutamiento gestionadas que proyectan el protocolo ODD DEBEN incluir, en sus pasos de exploración y de cierre, estas reglas mínimas, en un texto idéntico para todos los proyectos y adaptadores:

1. Lectura del contexto técnico: el paso de exploración DEBE indicar que, si el proyecto declara un contexto técnico mantenido, se obtienen los localizadores con `axiom workspace context --cwd <repo> --json`, se abre primero `technicalContext.maintained.entry` y se consultan los demás `technicalContext.maintained.paths` solo bajo demanda. Sin contexto mantenido declarado, el paso NO DEBE exigir ninguna lectura adicional. La regla NO DEBE depender de `axiom sdd status --json` ni de un cambio SDD.
2. Contexto técnico: el cierre DEBE indicar que, si el proyecto declara un contexto técnico mantenido y una unidad de trabajo cambia algo que ese contexto documenta, la documentación afectada se actualiza en el mismo commit de esa unidad de trabajo, con los localizadores leídos en la exploración (la salida de `axiom workspace context --cwd <repo> --json`). La regla NO DEBE exigir nada cuando el proyecto no declara contexto mantenido y NO DEBE autorizar la modificación de documentación declarada como de referencia (`technical_context.reference`).
3. Spec viva: el cierre DEBE indicar que, cuando el trabajo cambia o debe revisar la spec viva, la delta OpenSpec se escribe en la sección `## Spec viva` del documento `odd/tasks/<feature>.md`, que todo bug (señalado con `Tipo: bug`) aporta una `## Spec viva` con contenido o el marcador de la sección `## Impacto en la spec` (la línea `Sin impacto en la spec` más una línea `Motivo:` no vacía, el mismo contrato que el carril SDD; el bug ODD no exige `## Informe de bug`) y que se ejecuta `axiom archive integrate --odd <feature>` en el commit de la unidad de trabajo que cierra la feature, sin paradas (RF-RUT-18). La integración en `openspec/specs/` es la única escritura permitida fuera del documento ODD y del código.
4. Remisión: el cierre NO DEBE contener el valor del carril ni la condición ISO de ningún proyecto; la negativa de la integración en proyectos ISO la emite el verbo (RF-OLS-07 de `odd-living-spec-integration`).

Estas reglas NO DEBEN alterar los demás pasos del protocolo ODD ni el carril SDD. La proyección DEBE seguir siendo determinista: el mismo texto en cada renderizado. Estas reglas son la extensión de `project-technical-context` y de `odd-living-spec-integration` al carril ODD, que no tiene archive.
(Previously: solo el paso de cierre mencionaba el contexto técnico, con el localizador de `axiom sdd status --json`, y la exploración no lo leía)

#### Scenario: La exploración lee la entrada del contexto mantenido

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el paso de exploración del protocolo ODD indica obtener los localizadores con `axiom workspace context --cwd <repo> --json`, abrir primero `technicalContext.maintained.entry` y consultar `technicalContext.maintained.paths` solo bajo demanda
- **Y** la regla indica que sin contexto mantenido declarado no se lee nada más

#### Scenario: La regla figura en el paso de cierre del protocolo ODD

- **DADO** un adaptador registrado con un manifiesto de capacidades válido
- **CUANDO** se renderizan sus instrucciones de enrutamiento gestionadas
- **ENTONCES** el paso de cierre del protocolo ODD incluye la regla de actualizar el contexto técnico mantenido en el mismo commit de la unidad de trabajo
- **Y** la regla indica que solo aplica cuando el proyecto declara un contexto técnico mantenido

#### Scenario: El cierre remite a los localizadores de la exploración

- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de cierre
- **ENTONCES** remite a los localizadores leídos en la exploración con `axiom workspace context --cwd <repo> --json`
- **Y** no cita `axiom sdd status --json` como localizador del contexto técnico

#### Scenario: El cierre manda integrar la spec viva y declarar el impacto de los bugs

- **DADO** las instrucciones de enrutamiento renderizadas
- **CUANDO** se inspecciona (pin de texto) el paso de cierre
- **ENTONCES** indica la sección `## Spec viva`, el marcador de `## Impacto en la spec` para los bugs sin delta (la línea literal `Sin impacto en la spec` y una línea `Motivo:`) y la ejecución de `axiom archive integrate --odd <feature>`
- **Y** no contiene ninguna parada ni el texto de las tres salidas

#### Scenario: La regla no exige nada sin contexto mantenido

- **DADO** un proyecto sin `technical_context.maintained`
- **CUANDO** el orquestador explora y cierra una unidad de trabajo bajo ODD
- **ENTONCES** la regla no exige ninguna lectura ni actualización de documentación de contexto
- **Y** la exploración y el cierre proceden como antes

#### Scenario: La documentación de referencia no se modifica

- **DADO** un proyecto con `technical_context.reference` declarado
- **CUANDO** una unidad de trabajo bajo ODD cambia algo que esa documentación describe
- **ENTONCES** la regla no autoriza a modificarla

#### Scenario: El renderizado sigue siendo determinista e idéntico entre proyectos

- **DADO** una sincronización de adaptador y dos proyectos, uno ISO y otro no
- **CUANDO** se renderizan repetidamente las instrucciones de enrutamiento gestionadas
- **ENTONCES** el texto es idéntico en cada renderizado y entre proyectos
- **Y** los pasos del protocolo ODD distintos de la exploración y el cierre, y el carril SDD, no mencionan el contexto técnico
```

### Capacidad: project-technical-context

```openspec
## MODIFIED Requirements

### Requirement: Proyección Aditiva de `technicalContext` en el Status v2 (RF-CTX-03)

Cuando `axiom.yaml` declara `technical_context:` y la declaración es válida, el status v2 DEBE incluir el campo opcional `technicalContext` con, por cada modo declarado (`maintained`, `reference`), `paths` (rutas absolutas) y, si se declaró, `entry` (ruta absoluta). Un modo no declarado NO DEBE aparecer y, sin declaración, el campo NO DEBE aparecer. El campo se proyecta con o sin cambio seleccionado, también cuando el status queda bloqueado por no haber cambios activos o por una selección ambigua, y NO DEBE alterar `artifacts`, `artifactPaths` ni `contextFiles`: el delta de contexto no se proyecta en el status.

La misma proyección DEBE obtenerse sin cambio SDD con `axiom workspace context [--cwd <dir>] [--json]`, que resuelve el workspace desde `--cwd` y devuelve un JSON con `schemaName` `axiom.workspace-context`, `schemaVersion` 1, `workspaceRoot` y `technicalContext` (la misma forma que el campo del status, o `null` sin declaración). Sin declaración el comando DEBE terminar con código 0. Con una declaración inválida DEBE fallar cerrado (RF-CTX-20): `technicalContext` `null`, un campo `invalid` con el motivo y la indicación de ejecutar `axiom workspace validate`, y código distinto de cero.
(Previously: la proyección solo existía en el status v2 y no constaba su presencia con cero o varios cambios activos)

#### Scenario: Ambos modos proyectados
- **DADO** un `technical_context:` con los dos modos
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `technicalContext` contiene `maintained` y `reference`, cada uno con `paths` absolutos y su `entry` absoluta

#### Scenario: Un solo modo
- **DADO** un `technical_context:` solo con `reference`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** `technicalContext` contiene únicamente `reference`

#### Scenario: Sin declaración
- **DADO** un `axiom.yaml` sin `technical_context:`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el JSON no contiene `technicalContext`

#### Scenario: El estado de artefactos no cambia
- **DADO** un cambio con delta de contexto presente y `technical_context.maintained` declarado
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** las claves de `artifacts` y de `artifactPaths` son las mismas que sin delta de contexto

#### Scenario: Proyección con cero o varios cambios activos
- **DADO** un `technical_context:` válido y ningún cambio activo, o dos cambios activos sin `--change`
- **CUANDO** se ejecuta `axiom sdd status --json`
- **ENTONCES** el status queda bloqueado
- **Y** `technicalContext` se proyecta igualmente

#### Scenario: Proyección sin cambio SDD
- **DADO** un `technical_context:` válido
- **CUANDO** se ejecuta `axiom workspace context --cwd <dir> --json`
- **ENTONCES** el JSON contiene `workspaceRoot` y `technicalContext` con la misma forma que el campo del status
- **Y** el código de salida es 0

#### Scenario: El comando falla cerrado ante una declaración inválida
- **DADO** un `technical_context:` inválido
- **CUANDO** se ejecuta `axiom workspace context --cwd <dir> --json`
- **ENTONCES** `technicalContext` es `null` y `invalid` indica el motivo y `axiom workspace validate`
- **Y** el código de salida es distinto de cero
```

## Progreso

Plan stop: approved 2026-11-17; implementation continues

- T1 · commit `c9b96547`. Escritor delegado (T1-T3, un único escritor). `ResolveWorkspaceTechnicalContext(cwd)` y `WorkspaceTechnicalContext` en `internal/sddstatus/workspace_context.go`; reutiliza `projectTechnicalContext`; `invalid` sale de `ParseErr` y `TechnicalContextErr`. Tests en `workspace_context_test.go`, incluida la verificación pendiente del README: `Resolve` sin cambio con 0 y con 2 cambios activos queda bloqueado y proyecta `technicalContext`.
- T2 · commit `7a811ab0`. `RunWorkspaceContext` en `internal/cli/workspace_context.go`, `case "context"` y ayuda en `cmd/axiom/main.go`. `docs/usage.md` no lista `workspace validate`; el comando se documenta en `docs/manual-de-inicio.md`, junto al párrafo del contexto técnico. Códigos: 0 sin declaración, 1 inválida o sin workspace, 2 banderas incorrectas.
- T3 · commit `32a99d7d`. Paso 2 con la frase de lectura y paso 7 remitiendo a los localizadores del paso 2; la frase de RDD del paso 7 queda intacta (C-01-c). Se reescriben `TestRenderRoutingClosesODDWithTheMaintainedTechnicalContext` y `TestRenderRoutingClosesODDWithTheMaintainedContextLocator`, que también fijaba el texto anterior del paso 7. Sin cambios en goldens.
- T4 · commit de cierre `docs(spec)`. `axiom archive integrate --odd odd-lee-contexto`: `organic-agent-trigger-rules: composed`, `project-technical-context: composed`; repetirlo da `already_integrated`.
- RDD: desactivado (off por defecto); sin revisión nativa.

## Verificación

- `gofmt -l` en los directorios tocados: sin salida (escritor, antes de cada commit).
- `go test ./internal/sddstatus/ -run 'WorkspaceTechnicalContext|WithoutChangeProjectsTechnicalContext'`: ok; `go test ./internal/sddstatus/`: ok (escritor).
- `go vet ./cmd/axiom/ ./internal/cli/`: limpio; `go test ./internal/cli/ -run 'TestWorkspaceContext'`: ok; `go test ./cmd/axiom/ -run 'TestAxiomHelp|Workspace'`: ok (escritor).
- `go test ./internal/components/agentguidance/...`: ok (escritor y comprobación del padre).
- `go test ./internal/components/ -run Golden`, `go test ./internal/cli/ -run 'Routing|Golden'`, `go test ./internal/assets/`: ok (escritor).
- `axiom archive integrate --odd odd-lee-contexto`: compuesto; segunda ejecución `already_integrated`.
- Aviso: la integración quita una regla `---` de separación en `openspec/specs/project-technical-context/spec.md` (el mismo defecto que anotó C-04-a al componer un MODIFIED). No se corrige aquí: fuera de alcance.
