# Revisión de cobertura de la spec — adaptador-ado

Informativa: no bloquea ninguna fase ni el archivo y no modifica la spec. Revisores ciegos `jd-judge-a` (A) y `jd-judge-b` (B), en paralelo; el informe es la unión deduplicada de sus huecos.

## Ronda 1 — 2026-11-22

Entradas: `proposal.md` y `specs/{azure-devops-adapter,integration-plugins,workspace-topology}/spec.md`. Ambos revisores devolvieron resultado utilizable.

| # | Tipo | Ubicación | Hueco | Revisor |
|---|---|---|---|---|
| 1 | uncovered_requirement | REQ-45.2 / REQ-45.3 | No se dice qué claves son obligatorias: falta el error tipado de configuración cuando `organization`, `project` o `team` faltan o están vacíos al conectar o descubrir (si no, la URL base queda en `https://dev.azure.com/`). | A, B |
| 2 | edge_case | REQ-45.3 | Solo se escapa la organización: faltan `project`, `team` (espacios habituales) y demás segmentos de ruta. | A |
| 3 | edge_case | REQ-45.3 | Redirecciones (3xx a la página de inicio de sesión u otro host): el `http.Client` por defecto las sigue y podría reenviar `Authorization` fuera de `dev.azure.com`. Tampoco se define el PAT válido de otra organización. | A, B |
| 4 | uncovered_requirement / edge_case | REQ-45.4 | `X-RateLimit-Delay` solo se «lee y expone»; la propuesta dice que también provoca espera, y no hay escenario. | A, B |
| 5 | edge_case | REQ-45.4 | `Retry-After` sin tope máximo, sin formato fecha HTTP y sin tratamiento de valores inválidos o negativos. | A, B |
| 6 | edge_case | REQ-45.4 | Sin definir los 5xx distintos de 503 (500, 502, 504), el error tipado al agotar el presupuesto con 503 o con errores de red, si el timeout es reintentable y la distinción explícita entre peticiones seguras y no seguras. | A, B |
| 7 | uncovered_requirement / error | REQ-45.5 | El mapa de scope probable no cubre `connectionData` ni tipos, estados y campos. Faltan 404 de recursos distintos de work item, 200 con cuerpo vacío y el escenario del recordatorio de la retirada de los PAT globales, que el requisito exige. | A, B |
| 8 | edge_case | REQ-45.6 | Lote con ids duplicados, ids que la API omite (inexistentes o sin permiso), orden cuando faltan, tamaño máximo del trozo (200) y fallo de un trozo intermedio. | A, B |
| 9 | edge_case | REQ-45.7 | Sin criterio para reconocer el error de test de ADO en 400/409, sin un 409 ajeno al test, sin 404 ni 401/403 en el PATCH, sin cuerpo de rechazo no parseable y sin PATCH vacío. | A, B |
| 10 | edge_case | REQ-45.8 | La caché de repos no cubre el aislamiento entre organizaciones o proyectos ni la concurrencia. | B |
| 11 | edge_case | REQ-45.9 | La paginación de miembros no cubre un servidor que ignora `$skip` (bucle infinito), un total múltiplo del tamaño de página ni un tope de páginas. Tampoco miembros que son grupos, equipo o proyecto inexistentes (404), tipos sin estados o campos ni salida ordenada y determinista. | A, B |
| 12 | uncovered_requirement | REQ-45.9 | «Las sugerencias de asignación salen de esa lista» no tiene operación ni escenario. | B |
| 13 | edge_case | REQ-45.10 | `authenticatedUser` parcial, equipo sin miembros (lista de candidatos vacía), comparación de `--user` (mayúsculas, displayName o email) y `--user ""`. | B |
| 14 | edge_case | REQ-45.11 | Dos `connect` concurrentes (actualización perdida), versión de esquema futura, normalización de mayúsculas de la organización y directorio de estado inexistente. | A, B |
| 15 | error / edge_case | REQ-45.12 | Sin formato de error con `--json` (candidatos, 401, configuración), PAT ausente, lista de miembros vacía, fallo al guardar la identidad, identidad previa cuando `connectionData` falla y código de salida concreto. | A, B |
| 16 | uncovered_requirement | REQ-45.13 | `discover` no tiene escenarios de `--plugin` desconocido, bloque ausente o desactivado ni PAT ausente: solo se cubren en `connect`. | B |
| 17 | edge_case | REQ-45.15 | Redacción del PAT en un eco del servidor URL-encoded o JSON-escaped, y orden entre truncar y redactar: si se trunca antes, puede quedar un fragmento del PAT sin redactar. | B |
| 18 | uncovered_requirement | REQ-45.1 / REQ-45.14 | «Verbos de despacho sin cambios» solo se prueba con `gate record`; faltan `branch ensure --role` y `sdd publish --archive`. Tampoco se especifica que `emit` rechace `change.start` a mano. | B |
| 19 | edge_case | REQ-43.4 (delta) | Falta el efecto en la salida de los verbos `integrations` existentes con un plugin ya registrado, el mensaje para `integrations.jira` y el escenario del ensamblado fuera de `integrations` (que `cli` y `cmd` usan el registro ensamblado). | B |

Estado: 19 huecos abiertos. En `continuous` no se relanza `sdd-spec`: van al resumen final y la persona decide.
