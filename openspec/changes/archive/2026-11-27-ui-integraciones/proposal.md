# Propuesta: UI de integraciones en el dashboard (C-07-h)

## Intención

Hoy la integración con un gestor de trabajo (Azure DevOps, por ahora el único plugin) solo se configura a mano: el bloque `integrations.<plugin>` de `axiom.yaml` se edita en el YAML, el PAT se guarda con `axiom integrations secret set`, la identidad se resuelve con `axiom integrations connect` y el arranque desde un work item exige `axiom change start` en la terminal. El catálogo de Ajustes (C-06-e) excluye `integrations` a propósito (RF-TOP-21, RF-TOP-22, RF-WEB-18, RF-WEB-19) a la espera de este cambio, y el plan (P11, README rev. 11) fija que `integrations.<plugin>` sea editable desde la UI y se registre en ese catálogo.

Además, el servidor del dashboard responde con `Access-Control-Allow-Origin: *`, contesta 200 a cualquier `OPTIONS` y no fija una CSP general (`internal/dashboard/server.go:219-234`). Antes de que el navegador maneje un token, esa superficie debe cerrarse.

El cambio llega ahora porque sus dependencias ya están en `main`: el marco de integraciones y el manifiesto (C-07-b), el adaptador ADO con `discovery` (C-07-c), el arranque desde un WI (C-07-d), los vínculos, la outbox y las imputaciones de `integrations.yaml` v2 (C-07-e), las plantillas de rama en el catálogo (C-07-f), los actores (C-07-a) y el catálogo con su escritor `yaml.Node` (C-06-e).

Éxito significa: la persona activa y configura Azure DevOps solo desde la UI, con el resultado validado por el decodificador estricto; el PAT no aparece en ninguna respuesta HTTP ni en `axiom.yaml`; la vista del cambio muestra los vínculos, la outbox y los actores; y una petición con `Origin` o `Host` ajeno a una ruta que muta, o a cualquier ruta `/api/integrations*`, se rechaza.

## Alcance

### Dentro del alcance

- **Unidad 1, seguridad (primer commit, antes de manejar tokens).**
  - CSP general fijada en `withHeaders`: `default-src 'self'`, `script-src 'self'`, `img-src 'self' data:`, `font-src 'self'`, `connect-src 'self'`, `object-src 'none'`, `base-uri 'none'`, `form-action 'self'`, `frame-ancestors 'none'`. La CSP de `/api/explainer` sigue fijándose en su handler y prevalece, sin duplicarse (RF-WEB-15 no cambia).
  - `style-src`: se prefiere `'self'` estricta si refactorizar los estilos inline es acotado (`index.html:97, 620, 621, 628`; `core/increment.js:90`; `screens/incrementos.js:173`; `projects.js:45, 211`; `core/icons.js:70`). Si no lo es, `'unsafe-inline'` solo en `style-src`, con el riesgo residual (inyección de CSS) declarado. Lo decide `sdd-design` con evidencia.
  - El script inline de tema (`index.html:9-23`) pasa a un fichero propio cargado de forma síncrona en `<head>`, antes de `tokens.css`; `assets_test.go` se ajusta.
  - Se retiran todas las cabeceras `Access-Control-*` y el 200 genérico a `OPTIONS`, que pasa a responder sin cabeceras CORS (código exacto, en `sdd-design`).
  - Validación de `Host` local (además de `Origin`) en todas las rutas que mutan y en todas las rutas `/api/integrations*`, también en GET (DNS rebinding).
- **Unidad 2, catálogo.** Ámbito nuevo `integrations[<plugin>].<clave>` (más `enabled`) en `internal/settings`, derivado del manifiesto del plugin (`manifest.Lookup`, `ConfigField`: tipo, requerido, enum, formato, mín./máx.) sin literales duplicados. Se retira la exclusión de `integrations` en `document.go`, `patch.go` (incluido `canonicalBlocks` y el análisis de claves con guion en el id del plugin) y en las pruebas que la fijaban. El PAT nunca es clave del catálogo. La escritura sigue pasando por `PUT /api/settings` y `settings.Write`, con revalidación por `workspace.ParseConfig`.
- **Unidad 3, API de integraciones.**
  - `GET /api/integrations`: manifiesto, valores, estado del secreto (solo `source`: `env`, `keyring`, `missing`, `keyring_unavailable`, nunca el valor), identidad persistida y descubrimiento bajo demanda.
  - «Guardar PAT»: endpoint propio (POST, solo JSON, cuerpo acotado, petición local) que guarda con `secrets.Default().Set` y responde solo `source`. Eliminar PAT: endpoint propio. En Linux sin almacén escribible (`ErrNoWritableStore`), la respuesta y la UI guían a la variable de entorno derivada del manifiesto (para ADO, `AXIOM_INTEGRATION_AZURE_DEVOPS_PAT`); si una variable de entorno tapa el almacén, la UI lo indica con `source`.
  - «Probar conexión»: misma semántica que `axiom integrations connect`; resuelve y muestra la identidad y la persiste en `~/.axiom/integration-identities.json` (local, no versionado). Con `IdentitySelectionError`, la UI muestra los candidatos y reenvía la elección.
  - «Descubrir»: rellena los desplegables con `discovery` (C-07-c).
  - Toda ruta nueva queda clasificada en el canario de secretos (`coveredRoutes` o `excludedRoutes` con motivo).
- **Unidad 4, pantalla «Integraciones».** Generada desde el manifiesto, genérica por plugin (sin conocer `azuredevops`): activación, conexión, tipos, eventos, estados, campos, tags y jornada. Las plantillas de rama no se duplican: la pantalla enlaza a «Ajustes del proyecto», donde ya están (C-07-f). Entrada de navegación y ruta por *hash*.
- **Unidad 5, vista por cambio.** Pestaña nueva en el detalle del cambio: WI enlazado, Task por rol con su estimación e imputaciones (`WorkItemLinks`, `TaskLinks`, `TimeEntries`, `PendingTimeFor`), operaciones pendientes de la outbox mediante un lector público nuevo de solo lectura en `internal/integrations`, y actores por fase (`GateRecord.Actor`, `Kickoff.SealedBy`, `TimeEntry.Actor`). También para cambios archivados (`Folder`).
- **Unidad 6, arranque desde un WI.** Campo de ID o URL del WI en el alta de incremento del dashboard, visible solo con un plugin activo. Invoca las fases de `change start` (C-07-d: fase 1, `ensure` con el nombre derivado, `seal`, fase 2) mediante un puerto inyectable (patrón `marks_port.go`), sin reimplementar su lógica. Un rechazo de la fase 1 no deja rama, carpeta ni `kickoff.yaml`; un fallo posterior conserva el *staging* y ofrece reintentar o descartar, y el descarte solo se ejecuta por acción explícita de la persona. Enlace «Abrir en Azure DevOps» desde `Link.URL`, con URL validada (http/https) y `target="_blank" rel="noopener noreferrer"`.

### Fuera del alcance

- Formulario de alta de work items (descartado por el usuario).
- Llamar a Azure DevOps real en las pruebas: se usan dobles (`adotest`) y puertos inyectables.
- Sincronización bidireccional, re-importación o `refresh` del WI.
- Duplicar en «Integraciones» los controles de `branch_template`.
- Cualquier ruta legal del PAT a `axiom.yaml`, a otro fichero versionado o a una respuesta HTTP.
- Cambios en el adaptador ADO, en el formato de `integrations.yaml` o en la semántica de la outbox.

## Capacidades

### Capacidades nuevas

Ninguna. Los requisitos nuevos entran como ADDED en capacidades existentes.

### Capacidades modificadas

- `local-web-dashboard`:
  - MODIFIED RF-WEB-07: la navegación incorpora «Integraciones» y la pestaña de integraciones del detalle del cambio.
  - MODIFIED RF-WEB-10: las escrituras de `integrations.<plugin>` siguen pasando solo por `/api/settings`; el PAT se escribe por su endpoint propio y nunca toca `axiom.yaml`; toda escritura exige `Host` y `Origin` locales.
  - MODIFIED RF-WEB-18: desaparece el escenario «ningún control de `integrations.*`»; la pantalla de Ajustes no pinta el bloque y enlaza a «Integraciones».
  - MODIFIED RF-WEB-19: la respuesta incluye `integrations` y `integrations[<plugin>].<clave>` deja de ser `unknown_key`.
  - A confirmar en `sdd-spec`: RF-WEB-01 (la exploración lo cita) y RF-WEB-26 (canario de marcas), o bien un ADDED propio para el canario de las rutas nuevas.
  - ADDED (`prov:WEB:<n>`): cabeceras de seguridad generales (CSP, sin CORS, `Host` local); pantalla «Integraciones» generada desde el manifiesto; endpoints `/api/integrations*` (estado, PAT, conexión, descubrimiento) sin secretos en ninguna respuesta; pestaña de integraciones por cambio; campo de WI en el alta de incremento con enlace «Abrir en Azure DevOps».
- `workspace-topology`:
  - MODIFIED RF-TOP-21: el catálogo incluye `integrations[<plugin>].<clave>` derivado del manifiesto; el PAT nunca es clave.
  - MODIFIED RF-TOP-22: el escritor de *round-trip* edita el bloque `integrations` con las mismas garantías (rechazo de anclas, alias y `<<`, revalidación completa, revisión y escritura atómica).
- Sin cambio de requisito previsto (lo confirma `sdd-spec`): `integration-plugins` (RF-INT-13/15/16 y RF-INT-23; el lector público de la outbox es interno), `azure-devops-adapter` (RF-ADO-10 a 13 y RF-ADO-15) y `change-start-from-work-item` (el dashboard reutiliza las mismas fases).

## Enfoque

Enfoque 1 de la exploración: seguridad primero, después el catálogo ampliado y una pantalla generada desde el manifiesto. Se descartan un escritor aparte para `integrations` (duplica round-trip y validación y abre una segunda puerta de escritura de `axiom.yaml`) y una CSP con hash del script inline (frágil).

1. **Commit por unidad, en el orden del alcance** (1 seguridad, 2 catálogo, 3 API, 4 pantalla, 5 vista por cambio, 6 arranque desde WI), con pruebas y documentación en el mismo commit. Entrega `single-pr`.
2. **Dirección de importaciones:** `internal/settings` importa solo `integrations/manifest` (hoja); el dashboard puede importar `integrations`, `secrets` y `cli`, y trata el plugin de forma genérica.
3. **Lógica sin duplicar:** `connect`, `discover` y las fases de `change start` se reutilizan exportándolas desde `internal/cli` o moviéndolas a un paquete importable sin ciclos; lo decide `sdd-design`. Los rechazos nuevos en `cli` nombran su resolución o llevan `refusal:by-design`.
4. **Tiempo de respuesta:** `connect` y `discover` pueden superar el `WriteTimeout` de 15 s; `sdd-design` fija un plazo por handler o una respuesta asíncrona.
5. **El secreto solo entra:** el PAT cruza una única petición POST hacia el servidor y se guarda en el almacén del sistema; ninguna respuesta, log ni DTO lo contiene (`Secret` ya se redacta al imprimirse o serializarse).

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/dashboard/server.go` | Modificado | `withHeaders` (CSP, sin CORS, `OPTIONS`), validación de `Host`, rutas nuevas |
| `internal/dashboard/` (API, servicio, DTO y puerto de integraciones) | Nuevo | Endpoints `/api/integrations*`, vista por cambio, puerto de `change start` |
| `internal/dashboard/service.go`, `service_branch.go` | Modificado | Alta de incremento con WI; detalle del cambio |
| `internal/dashboard/secrets_canary_test.go`, `assets_test.go`, `assets_settings_test.go`, `server_test.go` | Modificado | Clasificación de rutas; script de tema; cadenas JS |
| `internal/dashboard/assets/` (`index.html`, script de tema nuevo, `screens/integraciones.js`, `core/integrations.js`, `detalle.js`, `detalle-integraciones.js`, `router.js`, `main.js`, `shell.js`, `style.css` y los ficheros con estilos inline) | Nuevo/Modificado | Pantalla, pestaña, navegación y CSP |
| `internal/settings/` (`catalog.go`, `document.go`, `patch.go` y sus pruebas) | Modificado | Ámbito `integrations[<plugin>].<clave>` y retirada de la exclusión |
| `internal/integrations/` | Modificado | Lector público de solo lectura de la outbox |
| `internal/cli/` (`integrations_ports.go`, `change_start.go`) | Modificado | Exportar o trasladar `connect`, `discover` y el tracker de WI |
| `openspec/specs/local-web-dashboard`, `openspec/specs/workspace-topology` | Modificado | Deltas en el cambio, integrados al archivar |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| La CSP rompe la UI por estilos inline o por el script de tema | Media | Inventario cerrado en la exploración; `sdd-design` elige `style-src` estricta o `'unsafe-inline'` con el riesgo declarado; prueba de cabeceras y de assets |
| Fuga del PAT por una ruta, un DTO o un log | Baja | Canario de secretos con todas las rutas nuevas; el endpoint responde solo `source`; tipo `Secret` redactado |
| CSRF o DNS rebinding sobre los endpoints de integración | Media | `Host` y `Origin` locales en las rutas que mutan y en todas las `/api/integrations*`, con pruebas explícitas de `Origin` y `Host` ajenos |
| `connect` o `discover` superan los 15 s de `WriteTimeout` | Media | Plazo por handler o respuesta asíncrona (`sdd-design`) |
| Una variable de entorno tapa el PAT guardado, o en Linux no hay almacén | Media | La UI muestra `source` y guía a la variable de entorno |
| Editar `integrations` con anclas o alias en el bloque | Baja | Mismo rechazo que los demás bloques (RF-TOP-22) |
| Ciclos de importación al reutilizar lógica de `cli` | Media | Puerto inyectable; `TestImportsDirection` y `TestAdapterAssemblyImportRules` |
| Exportados nuevos sin uso de producción | Media | Trinquete de código muerto: solo se exporta lo que usa el dashboard |
| Un fallo a mitad del arranque desde WI deja *staging* huérfano | Media | Mismas salidas que el orquestador: reintentar o descartar por acción explícita |
| Tamaño (L, unas 1.000-1.400 líneas) | Media | Un commit por unidad, revisable por separado dentro de la PR única |

## Plan de reversión

- Revertir la PR única (*merge commit*): desaparecen las rutas `/api/integrations*`, la pantalla, la pestaña y el campo de WI; el catálogo vuelve a excluir `integrations` y los deltas no se integran en la spec viva.
- La unidad de seguridad puede revertirse por separado (`git revert` de su commit) si rompe un cliente legítimo, aunque deja la superficie del PAT sin cerrar: en ese caso deben revertirse también las unidades 3 y 4.
- `axiom.yaml` está versionado: cualquier escritura de la UI se deshace con `git checkout` o `git revert`; el escritor nunca deja un documento inválido ni a medias.
- El PAT guardado en el almacén del sistema y la identidad de `~/.axiom/integration-identities.json` son locales: se retiran con `axiom integrations secret delete` y borrando la entrada de identidad.

## Dependencias

- Ya en `main`: C-07-a, C-07-b, C-07-c, C-07-d, C-07-e, C-07-f y C-06-e.
- Sin decisiones de producto abiertas: el orquestador fijó «Probar conexión», el campo de WI, las plantillas de rama, el catálogo, la vista por cambio y el alcance excluido.
- Decisiones técnicas que pasan a `sdd-design`: `style-src` estricta o con `'unsafe-inline'`; código de `OPTIONS`; exportar o trasladar `connect`, `discover` y el tracker; plazo de `connect` y `discover`.

## Restricciones de CI

- Ningún secreto viaja al navegador ni a ficheros versionados.
- Canario de secretos con todas las rutas nuevas clasificadas.
- Trinquete de código muerto y trinquete de rechazos (`refusal_resolution_ratchet_test.go`).
- `TestImportsDirection` y `TestAdapterAssemblyImportRules`.
- Pruebas localizadas con `-run` sobre los paquetes tocados; la batería completa, en CI. Sin binarios sueltos.

## Criterios de éxito

- [ ] Azure DevOps se activa y configura solo desde la UI, y `axiom.yaml` resultante valida con el decodificador estricto.
- [ ] El PAT no aparece en ninguna respuesta HTTP, en `axiom.yaml` ni en otro fichero versionado (canario de secretos en verde con las rutas nuevas).
- [ ] Toda respuesta del dashboard lleva la CSP general y ninguna cabecera `Access-Control-*`; `/api/explainer` conserva su CSP propia.
- [ ] Una petición con `Origin` ajeno a una ruta que muta se rechaza, y una con `Host` ajeno a cualquier ruta `/api/integrations*`, también en GET, se rechaza.
- [ ] «Probar conexión» muestra la identidad resuelta y, si hay varios candidatos, permite elegir.
- [ ] En Linux sin almacén escribible, la UI explica la variable de entorno y no simula un guardado.
- [ ] El catálogo expone `integrations[<plugin>].<clave>` derivado del manifiesto y nunca el PAT.
- [ ] La vista del cambio muestra el WI enlazado, las Tasks por rol con estimación e imputaciones, la outbox pendiente y los actores por fase.
- [ ] Con plugin activo, el alta de incremento acepta un ID o URL de WI y crea el cambio con el nombre derivado; sin plugin, el alta no cambia.
- [ ] El enlace «Abrir en Azure DevOps» usa `Link.URL` validada.
