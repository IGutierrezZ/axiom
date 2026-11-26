# Revisión de cobertura de la spec — ui-integraciones

Revisión informativa (`_shared/sdd-coverage-review.md`): no bloquea fases ni archivo y no modifica la spec. Dos revisores ciegos de solo lectura (A y B) sobre `proposal.md` y las deltas de `specs/`. El informe es la unión deduplicada de sus huecos.

## Ronda 1 — 2026-11-26

Entradas: `proposal.md`, `specs/local-web-dashboard/spec.md` (prov:WEB:1..11; MODIFIED RF-WEB-07, 10, 18, 19) y `specs/workspace-topology/spec.md` (MODIFIED RF-TOP-21, 22). Ninguno de los dos revisores consultó las specs vivas ni el plan.

Resultado: 22 huecos tras deduplicar (A: 21, B: 22, con gran solape). Ninguno es de producto: todos son concreciones de comportamiento que `sdd-design` puede fijar sin cambiar el alcance.

| # | Tipo | Ubicación | Hueco | Revisores |
|---|---|---|---|---|
| 1 | edge_case | prov:WEB:2 | Falta definir «`Host` local» (lista cerrada `127.0.0.1`, `localhost`, `[::1]` con el puerto del servidor), la comparación de `Origin` (esquema, puerto, `Origin: null`), `Host` ausente o duplicado, y un caso negativo de nombre DNS ajeno que resuelve a bucle local. Aceptar «el `Host` del propio servidor» sin lista cerrada reabre el *rebinding*. | A, B |
| 2 | edge_case | prov:WEB:2 | Sin escenarios de `HEAD`/`OPTIONS` sobre `/api/integrations*`, ni de regresión sobre las rutas que ya mutan (marcas, `worktree-prefs`, `settings/personal`, alta) ni de que una petición local legítima sin `Origin` (curl, pruebas) siga funcionando. | A, B |
| 3 | error | prov:WEB:1 | El código de `OPTIONS` no está fijado, ni hay escenario de *preflight* real desde origen ajeno, ni de POST «simple» (`text/plain`) cruzado rechazado por `Content-Type` y `Origin`. | A, B |
| 4 | edge_case | prov:WEB:1 | La CSP no se exige en respuestas de error (403/404/405/500), 304 ni `HEAD`; faltan cabeceras complementarias (`X-Content-Type-Options: nosniff`, `Referrer-Policy`, `Cache-Control: no-store` en `/api/integrations*`). | A, B |
| 5 | uncovered_requirement | prov:WEB:1 | Ningún escenario verifica que la SPA siga funcionando sin violaciones bajo la CSP (sin `style=` ni manejadores inline), ni que el script de tema externo conserve el comportamiento (tipo MIME, sin parpadeo, preferencia guardada), ni que `style-src` elegido quede fijado por prueba. | A, B |
| 6 | edge_case | prov:WEB:4 | Validación de la entrada del PAT: `Content-Type` distinto de JSON, vacío o solo blancos, saltos de línea, longitud máxima, campos desconocidos, plugin desconocido o inactivo, PAT por *query string* o por GET. | A, B |
| 7 | error | prov:WEB:4 | Eliminar: idempotencia sin PAT guardado, fallo del almacén al eliminar, confirmación explícita previa (el PAT es de ámbito de usuario y plugin, compartido por todos los proyectos). | A, B |
| 8 | error | prov:WEB:4 | Fallos del almacén distintos de `ErrNoWritableStore` (keyring bloqueado o denegado, E/S); guardado correcto pero tapado por la variable de entorno (responder `source: env` y avisar); disponibilidad inmediata del PAT guardado para conectar y descubrir. | A, B |
| 9 | error | prov:WEB:5 | Errores de «Probar conexión»: credencial rechazada (401/403) frente a red, DNS o TLS; configuración incompleta; plugin inactivo o desconocido; mensaje del servicio externo que no refleje el PAT ni `Authorization`. | A, B |
| 10 | edge_case | prov:WEB:5 | Elección de identidad forjada o caducada (no estaba entre los candidatos) rechazada con error tipado; identidad que cambia; fallo al escribir `~/.axiom/integration-identities.json`; dos pruebas simultáneas. | A, B |
| 11 | edge_case | prov:WEB:5, prov:WEB:6 | ¿Se prueba y descubre con lo guardado en `axiom.yaml` o con lo tecleado sin guardar? Riesgo de enviar el PAT a una organización no guardada. | A, B |
| 12 | uncovered_requirement | prov:WEB:5, prov:WEB:6 | El plazo por handler no es verificable: código HTTP del plazo agotado, cancelación real de la llamada de red al cerrar el cliente, sin llamadas colgadas, doble clic y limitación de frecuencia. | A, B |
| 13 | error | prov:WEB:6 | Descubrimiento parcial, listas vacías o muy grandes, valor configurado que ya no aparece en lo descubierto (no debe perderse en silencio), descubrimientos solapados. | A, B |
| 14 | edge_case | prov:WEB:7 | Revisión obsoleta (409) al guardar desde Integraciones y edición concurrente con Ajustes o con un editor externo; `writable: false`; bloque `integrations` inválido o plugin desconocido ya escrito a mano. | A, B |
| 15 | use_case | prov:WEB:7 | Desactivar un plugin con cambios en curso (vínculos, outbox); cero o varios plugins disponibles; campos de lista (tags, tipos, estados) y vaciado con `null`; el PAT nunca en el lote de `PUT /api/settings` ni en `localStorage` (campo `type=password`, `autocomplete=off`, vaciado tras guardar); accesibilidad básica. | A, B |
| 16 | error | prov:WEB:8 | `integrations.yaml` u outbox corruptos, de versión desconocida o inexistentes (error parcial sin romper el detalle); outbox grande (límite y orden); redacción de datos sensibles de las operaciones (solo tipo, estado e intentos); vínculos históricos con el plugin desactivado; rol sin Task; actor ausente (cambios anteriores a C-07-a). | A, B |
| 17 | error | prov:WEB:8 | Lectura concurrente con la CLI drenando la outbox (sin bloquear ni corromper); cambio inexistente (404 tipado) y nombre con *path traversal*. | A, B |
| 18 | edge_case | prov:WEB:9 | Entrada del WI: vacía, caracteres de control, longitud, URL de otro host u organización, WI inexistente o sin permisos, WI ya enlazado, nombre derivado que colisiona, plugin o PAT que desaparecen entre mostrar y enviar; el contenido del WI es de terceros (no confiable). | A, B |
| 19 | error | prov:WEB:9 | Fallos y reanudación: navegador o servidor caídos entre fases (staging huérfano tras recargar), doble envío, `decision_required` de `ensure` desde la UI, rechazos tipados de la fase 2 (`change_not_prepared`, `staging_not_found`, `staging_conflict`, `change_start_busy`, `write_failed`), plazo total frente al `WriteTimeout`. | A, B |
| 20 | use_case | prov:WEB:9 | «Reintentar» y «Descartar» son rutas que mutan: sin ruta definida, sin exigencia de `Host`/`Origin` ni de clasificación en el canario, sin confirmación explícita previa del descarte; estilo de ejecución, modo de worktree, multirepo, ISO y carril con que se crea el cambio desde el dashboard. | A, B |
| 21 | edge_case | prov:WEB:10 | Validación de URL frente a `JavaScript:`, blancos o caracteres de control iniciales, URLs relativas o `//host`, credenciales embebidas, host distinto de la organización configurada; la etiqueta «Abrir en Azure DevOps» contradice una pantalla genérica por plugin. | A, B |
| 22 | uncovered_requirement | prov:WEB:11, RF-TOP-21, RF-TOP-22 | Canario: ejercitar también rutas de error y mensajes del servicio externo y del almacén, cabeceras y logs del servidor. Catálogo: plugin sin manifiesto o con valor huérfano, campos de lista o mapa, claves con `.`, `[` o `-`, obligatoriedad condicional a `enabled`, orden estable, topología. Escritor: tipos no cadena, plugin desconocido intacto junto a uno conocido, comentarios al borrar el bloque, posición del bloque nuevo. | A, B |

### Tratamiento

En `continuous` el padre no relanza `sdd-spec` por un hueco. Este informe se entrega como entrada a `sdd-design`, que fija las concreciones técnicas dentro del alcance (lista de `Host`, código de `OPTIONS`, validación del PAT, plazos, rutas de reintento y descarte, redacción de la outbox, validación de URL) y declara las que deja fuera. Los huecos no resueltos van al resumen final.
