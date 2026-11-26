# Revisión de cobertura · design · ajustes-proyecto

## Ronda 1 — 2026-11-26

Revisores: `jd-judge-a` (A) y `jd-judge-b` (B), a ciegas y en solo lectura. Entradas: `proposal.md`, las dos specs (ya corregidas por el gatekeeper) y `design.md`. Ninguno de los dos encontró citas a código inexistente: los símbolos nuevos (`GatePolicies`, `PromotionValues`, etc.) forman parte del cambio. El informe recoge la unión sin duplicados y es informativo.

| # | Tipo | Ubicación | Hueco | Revisor |
|---|---|---|---|---|
| 1 | error | design#valores del GET · validación del motor | `axiom.yaml` sintácticamente inválido, o que `ParseConfig(original)` no pasa: falta el código del GET y del PUT, y cómo se calcula el conjunto de errores «previos» sin un original válido. El GET promete poder corregirlo desde la UI. | A, B |
| 2 | error | design#/api/settings/personal | Sin proyecto activo, o con fallo del hub ajeno a «no registrado»: falta el código. Un cuerpo `{}` o sin la clave `autofetch` no debe guardar `false` en silencio; la spec pide 400. | A, B |
| 3 | error | design#Flujo de datos (pantalla) | Fallo parcial de las tres cargas (settings, personal, worktree-prefs): proyecto no registrado (404 en el bloque personal) con el catálogo bien, o `config_missing`. No se define si se degrada solo el bloque o falla toda la pantalla. | A, B |
| 4 | edge_case | design#restaurar el formato | `restoreLayout` puede duplicar líneas en blanco dentro de escalares literales `\|` o plegados `>` y alterar su valor. Falta una prueba con líneas en blanco dentro de un literal. | B |
| 5 | edge_case | design#escritura atómica | `axiom.yaml` como enlace simbólico o *junction*, fichero de solo lectura o bloqueado: el GET dice `writable: true` y el PUT falla con `write_failed` 500. | A, B |
| 6 | edge_case | design#rechazo de anclas… | Una clave repetida se rechaza solo al escribir; el GET no la refleja en `writable: false`. | A |
| 7 | edge_case | design#parche por lotes | Valores de tipo correcto pero dominio inválido (`split_hint_lines` igual a 0 o negativo, `""`, `[]` frente a `null` en `environments`, entornos duplicados): ¿`invalid_value` o `validation_failed`? ¿`[]` escribe o retira? | A, B |
| 8 | edge_case | design#operaciones sobre el árbol | `repositories[<p>]` con un `<p>` desconocido: ¿se crea o es `key_not_applicable`, como pide RF-WEB-19? Retirar el último campo de una ruta que también usan los roles; normalización de `..` y rutas absolutas. | B |
| 9 | edge_case | design#«Abrir en el editor» | `encodeURI` no codifica `#`, `?` ni `&`: la ruta queda truncada en el `href`. | B |
| 10 | edge_case | design#ruta del fichero | En multirepo, `GET /api/workspace` sigue leyendo `<raíz>/axiom.yaml` y puede quedar desfasado respecto de Ajustes tras guardar. | A |
| 11 | uncovered_requirement | design#Estrategia de pruebas | Falta una prueba del `GET /api/settings/personal` no local (los GET siguen abiertos, RF-WEB-10). | A |

**Tratamiento (continuous):** el orquestador no relanza el design. Los huecos 2, 4 y 9 contradicen afirmaciones de la propia spec o del design (400 en un cuerpo mal formado, «nunca cambia datos», `href` con la ruta absoluta), así que se pasan a `sdd-apply` como comprobaciones de conformidad con pruebas. El resto queda para el resumen final.
