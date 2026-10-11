# Spike: estructura de la baseline del proyecto de referencia (inc-24, C8)

> Análisis de solo lectura del 2026-10-04 sobre la spec viva del proyecto de referencia multirepo (repo de specs del cliente, copia de trabajo local).
> Solo recuentos: no se ha copiado prosa ni títulos, solo tokens de ID como `RF-123`.
> La copia de trabajo usa `core.autocrlf=true`: el índice git guarda LF y la copia de trabajo, CRLF.
> Las cifras previas de la conversación (512 encabezados RF y 496 IDs) usaban un patrón sin anclar. Las correctas son 511 encabezados y 490 IDs distintos.

## Resultados por documento

| Documento | Líneas | EOL | BOM | Encabezados h1/h2/h3/h4 | IDs en encabezados |
|---|---|---|---|---|---|
| `00_Glosario.md` | 364 | CRLF | no | 1/16/0/0 | — |
| `00_Resumen_Ejecutivo.md` | 189 | CRLF | no | 1/8/2/0 | — |
| `01_Requisitos_Funcionales.md` | 14.447 | CRLF | no | 1/35/914/119 | `RF` en h3: 511 |
| `02_Requisitos_No_Funcionales.md` | 640 | CRLF | no | 1/16/42/0 | `NFR` en h3: 34 |
| `03_Modelo_Datos.md` | 4.761 | CRLF | sí | 1/30/77/105 | `US` en h3: 8; `BUG` en h3: 1 |
| `04_Flujos_Negocio.md` | 2.372 | CRLF | sí | 1/26/27/0 | — |
| `05_Interfaces_Usuario.md` | 3.983 | CRLF | sí | 1/32/96/68 | — |
| `06_Integraciones.md` | 731 | CRLF | no | 1/14/41/0 | `US` en h3: 8; `BUG` en h3: 1 |
| `07_Seguridad.md` | 644 | CRLF | no | 1/16/30/0 | `US` en h3: 4 |
| `README.md`, `REGISTRO_BUGS.md`, `REGISTRO_INCREMENTOS.md` | 63/14/40 | CRLF | no | solo h1/h2 | — |
| `context/` (50 ficheros) | 5.543 | CRLF (50) | 1 fichero | 50/361/231/0 | `US`: 5 |

## Documentos de requisitos (01 y 02)

| Medida | 01 (`RF`, h3) | 02 (`NFR`, h3) |
|---|---|---|
| Encabezados de requisito | 511 | 34 |
| IDs distintos | 490 | 34 |
| IDs duplicados | 21, todos con título distinto | 0 |
| Copias idénticas (título y cuerpo) | 0 | 0 |
| Líneas por bloque (mín/mediana/p90/máx) | 7/15/35/297 | 13/14/15/17 |
| h4 dentro de bloques de requisito | 119 (todos los h4 del documento) | 0 |
| h3 que no son requisito | 403 | 8 |
| … inmediatamente después de un requisito | 97 | 0 |
| … entre dos requisitos | 12 | 0 |
| Transiciones de ID descendentes | 23 | 0 |
| Textos de encabezado repetidos | 232 | 0 |

IDs duplicados en 01, dos apariciones cada uno y con títulos distintos: `RF-254`, `RF-395` a `RF-408` y `RF-420` a `RF-425`.

ID máximo en encabezados de 01: `RF-622`. La numeración tiene huecos: 511 encabezados frente a un máximo de 622. Comprobación posterior de solo lectura sobre los 10 incrementos en curso del flujo antiguo: citan 25 IDs `RF`. De ellos, 19 existen en la baseline y 6 quedan fuera de la secuencia (valores de 5.454 a 6.237). Ningún incremento en curso reserva números de la secuencia de la baseline.

## Implicaciones para la rebanada C

1. **Los duplicados son colisiones, no copias.** Los 21 IDs repetidos tienen títulos distintos y casi todos caen en dos rangos consecutivos. Es el patrón de dos flujos que asignaron los mismos IDs siguientes. Confirma Q7: rechazar operaciones sobre IDs ambiguos, exigir normalización previa y asignar los IDs definitivos al archivar.
2. **Frontera de bloque.** Todos los h4 caen dentro de bloques RF, así que la frontera "siguiente encabezado de nivel ≤ 3" los incluye bien. Pero hay 97 h3 que no son requisito justo después de un RF, 12 de ellos entre dos RF. Pueden ser subsecciones del requisito escritas a nivel h3 o secciones independientes, y un MODIFIED no las sustituiría. El diseño debe decidir cómo tratarlas: aviso, verificación o regla configurable.
3. **Orden.** Hay 23 transiciones de ID descendentes: los requisitos no están ordenados por ID. Un ADDED no puede insertarse "en orden de ID" de forma fiable, así que hay que definir la posición; por ejemplo, al final de la sección h2 de destino declarada.
4. **02 es directamente compatible** con un perfil `^### (NFR-\d+)`: no tiene duplicados ni h3 intercalados.
5. **Documentos que no son requisito.** 03, 06 y 07 ya contienen secciones h3 tituladas con el ID de la historia o bug que las introdujo. La integración actual del proyecto añade una sección por cambio, y eso da un ancla estable para E2 (crear o editar la sección del cambio), más robusta que editar prosa existente. En 04 no se repiten títulos, así que las anclas por título son viables. En 05 se repiten 15, así que no lo son.
6. **Bytes.** Todos los documentos de la copia de trabajo están en CRLF, y 03, 04 y 05 llevan BOM. Compose y E2 deben preservar EOL y BOM (C2, R6). Git normaliza a LF al confirmar.
7. **Contexto técnico.** 50 ficheros, 5.543 líneas en CRLF y 40 títulos repetidos dentro de un mismo fichero. El delta de contexto debe anclar por fichero y sección, y tolerar títulos repetidos.
