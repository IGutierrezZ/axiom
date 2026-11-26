# ODD: frontmatter válido en las skills minadas (fix-miner-frontmatter)

> **Documento vivo ODD.** Fichero autoritativo: `odd/tasks/fix-miner-frontmatter.md`. Espejo Engram: `odd/fix-miner-frontmatter/tasks`.
> Rama: `inc/fix-miner-frontmatter` · worktree `C:\repos\axiom-wt\fix-miner-frontmatter\axiom`. Cambio C-04-a del plan maestro (`docs/plan/backlog-2026-11/04-consumo-agentes.md`).

Tipo: bug

## Objetivo

Que toda skill minada por autoskill empiece con un frontmatter válido según las mismas reglas que `TestSkillFrontmatterIsLintClean` (`internal/assets/skills_frontmatter_test.go`).

## Problema

Las tres plantillas `SkillMD` de `internal/autoskill/miner.go` empiezan con `# Título`, ponen `description:`, `trigger:` y `origin:` en el cuerpo y colocan el `---` después: no hay bloque de frontmatter inicial. Los runtimes no cargan bien esas skills y el registro las indexa sin descripción (`AGENTS.md` muestra «—» para `axiom-go-table-tests` y `axiom-idiomatic-error-wrapping`).

## Por qué

C-04-d (`skillcheck`, ola 5) validará las skills minadas con estas reglas; sin este arreglo las rechazaría todas de entrada.

## Decisiones

- **`trigger` y `origin` van bajo `metadata:`** (usuario, 2026-11-17, ante la contradicción entre RF-ASK-07, que los pedía como claves de primer nivel, y la lista de claves permitidas del lint). El disparador también entra en la `description` como `Trigger: …`. RF-ASK-07 se actualiza en este cambio con la `## Spec viva`.
- Forma del frontmatter generado:
  ```yaml
  ---
  name: <nombre de la skill = directorio>
  description: "<resumen>. Trigger: <disparador>."
  metadata:
    origin: mined
    trigger: "<disparador>"
  ---
  ```
  La `description` cumple las reglas del lint: entrecomillada, una línea, <=160 caracteres y con `Trigger:`. Para caber, se acortan los campos `Description` y `Trigger` de las tres propuestas.
- El frontmatter se genera con una función a partir de los campos de la propuesta, no a mano en cada plantilla, para que nombre, descripción y disparador no diverjan.
- Mientras no exista C-04-d, las reglas del lint se duplican en el test del minero (riesgo aceptado en el plan; C-04-d las unifica). No se adelanta trabajo de C-04-d ni se cambia el flujo de `approve`.

## Alcance autorizado

- `internal/autoskill/miner.go` y sus tests en `internal/autoskill/`.
- Skills minadas ya versionadas: `skills/axiom-go-table-tests/SKILL.md`, `skills/axiom-idiomatic-error-wrapping/SKILL.md`, y sus filas de `AGENTS.md`.
- Delta de RF-ASK-07 en `## Spec viva` e integración en `openspec/specs/autoskills/spec.md` con `axiom archive integrate --odd fix-miner-frontmatter`.

Fuera de alcance: `skillcheck`, `internal/assets/skills_frontmatter_test.go`, el flujo de `approve`.

## Modo TDD

Desactivado: `axiom.yaml` no declara TDD y no hay elección del usuario. Comprobaciones funcionales ordinarias con `go test` (sin binarios sueltos).

## Tareas

- [x] **T1 · Frontmatter generado en el minero** (ruta: delegada, un escritor; disparador: código y test no triviales en dos ficheros)
  - Función que compone el frontmatter desde `Name`, `Description` y `Trigger`, y cuerpo con `# Título` tras el bloque.
  - Acortar `Description` y `Trigger` de las tres propuestas para que la `description` quepa en 160 caracteres.
  - Test en `internal/autoskill` que, para cada patrón del minero (los tres), comprueba: empieza con `---`; `name` igual al nombre del directorio de la skill (`Name`); `description` entrecomillada, de una línea, <=160 y con `Trigger:`; solo claves permitidas; `metadata.origin: mined`.
  - Comprobación: `go test ./internal/autoskill/...`.
- [x] **T2 · Skills minadas versionadas** (ruta: en línea, mecánica)
  - Sustituir la cabecera rota de las dos skills de `skills/` por el frontmatter que genera ahora el minero (el cuerpo no cambia) y actualizar sus filas de `AGENTS.md`.
  - Comprobación: el `SKILL.md` versionado coincide con el `SkillMD` del minero (test o comparación).
- [x] **T3 · Spec viva de RF-ASK-07** (ruta: en línea)
  - Delta MODIFIED en `## Spec viva` e integración con `axiom archive integrate --odd fix-miner-frontmatter`.
  - Comprobación: la integración termina sin rechazo y `openspec/specs/autoskills/spec.md` refleja el nuevo texto.

## Criterio de terminado

Un test de `internal/autoskill` comprueba, para cada patrón del minero, que el `SKILL.md` generado empieza con `---`, que `name` coincide con el directorio y que la `description` contiene `Trigger:`; `go test ./internal/autoskill/...` en verde.

## Entrega

Una sola PR (`single-pr`); previsión, unas 150-250 líneas. Push, PR y fusión los decide el usuario.

## Spec viva

### Capacidad: autoskills

```openspec
## MODIFIED Requirements

### Requirement: Generación Canónica de Borradores de Skill Minada (RF-ASK-07)

Cada patrón detectado por el minero DEBE formularse como un documento canónico `SKILL.md` que incluya:
1. Un bloque de frontmatter al inicio del fichero, delimitado por `---`, con `name` igual al directorio de la skill, `description` entrecomillada en una sola línea que contenga `Trigger:`, y `metadata` con `origin: mined` y `trigger`. No DEBE llevar otras claves de primer nivel que las admitidas para las skills de Axiom.
2. Sección de Propósito y Reglas del patrón en español.
3. Ejemplos de código correctos (*Do*) e incorrectos (*Don't*) basados en la estructura del propio repositorio.

#### Scenario: Generación estructurada de borrador de skill minada
- **DADO** un patrón de arquitectura detectado por el minero
- **CUANDO** se genera la propuesta de skill
- **ENTONCES** el archivo `SKILL.md` resultante sigue estrictamente el formato canónico de skills de Axiom
- **Y** queda redactado en español castellano con ejemplos del repositorio

#### Scenario: Frontmatter válido de la skill minada
- **DADO** un patrón detectado por el minero
- **CUANDO** se genera su `SKILL.md`
- **ENTONCES** el fichero empieza con `---` y su frontmatter declara `name` igual al directorio de la skill
- **Y** la `description` es una cadena entrecomillada de una línea que contiene `Trigger:`
- **Y** `origin: mined` y el disparador figuran bajo `metadata`
```

## Progreso

Plan stop: approved 2026-11-17; implementation continues

- T1 · commit `110d2cd6`. Escritor delegado: `composeSkillMD` en `miner.go`; test `TestMinerFrontmatterEsValido` en `internal/autoskill/miner_frontmatter_test.go` (tres patrones). Descripciones compuestas: 137, 151 y 134 runas.
- T2 · commit `4c8ff64b`. Las dos skills de `skills/` se regeneraron con la salida del minero mediante un test temporal (ya borrado); solo cambia la cabecera. Las filas de `AGENTS.md` coinciden con lo que lee el registro (`axiom skill index list --json`).
- T3 · commit `3de744f5`. `axiom archive integrate --odd fix-miner-frontmatter`: `autoskills: composed`; repetirlo da `already_integrated`.
- RDD: desactivado (`axiom review mode status`: off por defecto); sin revisión nativa.

## Verificación

- `gofmt -l internal/autoskill`: sin salida (escritor).
- `go vet ./internal/autoskill/...`: sin avisos (escritor).
- `go test ./internal/autoskill/...`: ok (escritor); comprobación de la sesión con `go test -count=1 -run TestMiner ./internal/autoskill/`: ok.
- `go test -count=1 ./internal/skillregistry/ ./internal/assets/`: ok.

## Seguimientos

- `axiom archive integrate --odd` quitó la regla `---` que separaba RF-ASK-07 de la sección 4 de `openspec/specs/autoskills/spec.md`: al componer un requisito MODIFIED, el separador final parece contarse como parte del bloque. No se corrige aquí (fuera de alcance).
- C-04-d debe unificar las reglas del frontmatter que ahora duplica `miner_frontmatter_test.go`.

Espejo Engram: pendiente (el modo automático no pudo revisar `mem_save` porque un hook modifica la llamada; se resincroniza cuando sea posible).
