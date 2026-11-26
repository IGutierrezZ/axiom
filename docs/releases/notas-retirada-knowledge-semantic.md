# Notas de migración: retirada de `axiom knowledge` y `axiom semantic`

Esta versión retira el análisis semántico propio de Axiom. Los agentes consultan el código con CodeGraph directamente, por su MCP y por su CLI (`codegraph explore "<consulta>"`), como ya hacían. Desaparecen:

- `axiom knowledge sweep`, `axiom knowledge crawl` y `axiom knowledge query`;
- `axiom semantic status`, `axiom semantic symbols`, `axiom semantic inspect` y `axiom semantic reindex`;
- la pantalla «Código y grafo» del dashboard (`axiom ui`), sus rutas `/api/semantic/*` y la fila del estado del código en Inicio.

El perfil `knowledge` de `axiom init` (`--profile knowledge` o `--spec-only`) no cambia, salvo que su `axiom.yaml` ya no lleva `governance.semantic_analysis`.

> **Esta versión no limpia lo que dejaron las versiones anteriores.** Si tienes restos, haz la limpieza de abajo **antes** de instalarla.

## Qué limpiar a mano

1. **`governance.semantic_analysis` en `axiom.yaml`.** Ya no tiene efecto. Bórrala de cada proyecto que la declare; mientras siga, `axiom workspace validate`, `axiom doctor` y el dashboard avisan de que se ignora.
2. **Planes de crawling.** Borra la carpeta `.axiom/knowledge/` de cada proyecto donde ejecutaste `axiom knowledge crawl --plan`. Guardaba `crawl-job.json` y no la lee nadie. Las especificaciones que el barrido o el crawling escribieron en `openspec/specs/` son tuyas: no hay que tocarlas.
3. **Scripts y alias.** Sustituye las llamadas a `axiom knowledge` o `axiom semantic`, que ahora responden como un comando desconocido. Para reindexar CodeGraph, usa su CLI en la raíz del proyecto: `codegraph init` la primera vez y `codegraph index` después.

## Qué pasa con una instalación anterior

- Un `axiom.yaml` con `governance.semantic_analysis` sigue cargando; solo produce el aviso del punto 1.
- CodeGraph no cambia: lo que Axiom instala y registra de CodeGraph cuando lo eliges (MCP e instrucciones) sigue igual.

## Acciones de quien publica

- Pegar este texto en las notas de la primera versión que incluya el cambio.
