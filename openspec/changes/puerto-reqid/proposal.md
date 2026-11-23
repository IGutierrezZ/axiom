# Propuesta: puerto único de identificadores de requisito y patrón declarado por proyecto (C-03-a)

## Intención

Hoy el formato `REQ-<n>.<m>` está fijado en cuatro expresiones regulares independientes (`openspecdoc/parse.go:33`, `isoevidence/reqid.go:13-14`, `livespec/registry.go:59`, `livingdoc/indexer.go:17`), más el reformateo `REQ-%d.%d` de `livespec` y varios mensajes de ayuda. Además, el compositor (`openspecdoc/compose.go:310-316`) identifica cada requisito por el texto completo de su título, de modo que cambiar el título de un MODIFIED rompe la identidad del requisito.

El usuario decidió pasar a IDs funcionales tipo `RF-USU-01` (D-03-3), conservar los IDs del cliente al adoptar (D-03-8) y que cada proyecto declare su patrón (A-03-1). Nada de eso es posible mientras cada consumidor tenga su propia regex. Este cambio es el primer paso del «cambio RF» (C-03-a → C-03-b → C-03-c ∥ C-03-d → C-03-e): concentra parseo, validación, formato y serie del ID en un único paquete hoja, deja que el proyecto declare el patrón en `axiom.yaml` y hace que el compositor empareje por ID.

Éxito (criterio de terminado de `docs/plan/backlog-2026-11/03-openspec-unica.md:84-89`):

- `rg 'REQ-' --type go internal cmd -g '!*_test.go'` solo encuentra coincidencias en el paquete del puerto, en el valor por defecto y en comentarios.
- Un proyecto de prueba con patrón `RF-<DOM>-<NN>` integra una delta ODD y archiva un cambio SDD sin errores.
- El compositor aplica un MODIFIED cuyo título cambió pero conserva el ID.
- REQ-29.3 está reescrito.
- `go test ./...` en verde, salvo el fallo conocido de `internal/components/filemerge` en Windows.

## Alcance

### Dentro del alcance

1. **Paquete hoja `internal/reqid`** (solo biblioteca estándar): tipo inmutable de patrón con parseo de la declaración, extracción del ID de un título, detección laxa de candidatos (para distinguir «candidato» de «válido», como hace hoy `isoevidence`), validación, serie, número, formato del siguiente ID y un texto legible del patrón para los mensajes. Un valor por defecto que admite exactamente los `REQ-n.m` vigentes.
2. **Declaración del patrón en el bloque `sdd:` de `axiom.yaml`** (`internal/workspace/sdd.go`): una clave nueva con decodificador estricto; un patrón inválido es error accionable de carga y de `axiom workspace validate` (canal `SDDErr`), nunca silencioso. Sin declaración, rige el valor por defecto. La clave concreta se fija en el diseño.
3. **Migración de todos los consumidores al puerto**, sin regex propia en ninguno:
   - `internal/openspecdoc` (`RequirementID`, `ParseDelta`, `ParseCanonical`).
   - `internal/isoevidence` (`RequirementIDs`, `CheckRequirementIDs`, `NewBaselineIndex`, `ParseCubre`, `ScenarioKey` y textos de continuación), con sus llamadores en `internal/sddstatus/iso.go` e `internal/cli/sdd_trace.go`.
   - `internal/livespec` (registro de series, `NextFree`, `SeriesOf`, `checkIdentifiers`, `nextFree`).
   - `internal/livingdoc` (lectura laxa e ID sintético, separado del espacio de IDs válidos).
   - `internal/adoption/adoptcheck`: solo lo mínimo para compilar con los nuevos tipos de serie; sus reglas no cambian.
   - Mensajes de ayuda de `internal/cli/archive_integrate.go` e `internal/cli/archive_adopt.go`.
4. **Compositor por ID** (`internal/openspecdoc/compose.go`): cuando el título de la delta y el de la canónica contienen un ID válido según el patrón, el emparejamiento es por ID aunque el título cambie; si no, sigue siendo por título exacto. RENAMED sigue exigiendo `Antiguo → Nuevo`. Toda composición que hoy se acepta produce la misma salida byte a byte (REQ-30.22, REQ-30.23).
5. **Reescritura de REQ-29.3** para que pida «un identificador válido según el patrón declarado del proyecto», y alineación de los requisitos vivos que citan ese formato (ver «Capacidades»).
6. **Guías y documentación que citan `REQ-<n>.<m>` como regla** (texto real localizado por la exploración): `internal/assets/skills/_shared/sdd-iso-convention.md:18`, `internal/assets/skills/sdd-spec/SKILL.md:257`, `internal/assets/skills/_shared/sdd-status-contract.md:153`, `docs/manual-de-inicio.md:547,625,630` y `axiom.example.yaml` (bloque `sdd:` comentado con la clave nueva), con sus pruebas de contrato y pines de texto.
7. **Cierre de las dos sugerencias del verify de P6** (`openspec/changes/archive/2026-11-12-spec-viva-openspec/verify-report.md:62-64`):
   - Constante exportada compartida para la etiqueta `Motivo:` (hoy duplicada entre `livespec/odddoc.go:41` y el literal de `isoevidence/bug.go:70`).
   - Prueba tabular de paridad en `livespec` que pase las mismas entradas a `isoevidence.ParseBugDeclaration` y a `parseImpact` y compare marcador y motivo. El marcador ya es compartido (`odddoc.go:40` usa `isoevidence.NoImpactMark`) y la interpretación de `Motivo:` ya convergió (`odddoc.go:349-353` frente a `bug.go:140-148`): falta fijarlo con la prueba.

### Fuera del alcance

- IDs provisionales en ADDED y asignación del número definitivo al integrar, y proyección de `requirements` en `status` (C-03-b).
- Adopción que conserva los IDs del cliente y reescritura de las reglas de `axiom archive adopt check` (`RuleFormat`, `RuleUniqueness`, `RuleSeries`) y de `internal/adoption/scan.go` (C-03-c). Por eso tampoco se tocan la guía `internal/assets/skills/sdd-spec/references/openspec-adoption.md`, `docs/manual-de-inicio.md:563` ni los requisitos REQ-30.11, REQ-30.12 y REQ-30.31, que siguen describiendo `REQ-n.m` mientras `adopt check` lo exija (A-03-2).
- `axiom trace <ID>` (C-03-d).
- Migración de las specs de Axiom a IDs funcionales y paso del valor por defecto a `RF-<dominio>-<nº>` (C-03-e).
- Esquemas de ID mixtos de un cliente (`RF-1` junto a `RF-001`): es decisión de C-03-b/C-03-c.
- Cambios en `internal/components/agentguidance/routing.go` y regeneración masiva de goldens (ver «Discrepancias con el plan»).

## Capacidades

### Nuevas capacidades

- `requirement-identifiers`: patrón de identificador de requisito declarado por proyecto en el bloque `sdd:` de `axiom.yaml`; valor por defecto que admite los `REQ-n.m` vigentes; gramática del patrón; qué es un ID válido, su serie y su número; formato del siguiente ID libre de una serie; error accionable ante un patrón inválido; y que el ID es el único criterio de identidad de un requisito cuando su título lo contiene. Prefijo propuesto **REQ-44** (máximo actual en `openspec/specs/`: REQ-43, `integration-plugins`; ningún fichero de `openspec/` ni de `docs/plan/` usa REQ-44 en este árbol).

### Capacidades modificadas

- `sdd-iso-evidence`:
  - REQ-29.1: el bloque `sdd:` deja de tener «una única clave, `iso`» y admite también la clave del patrón (REQ-44.x), con el mismo decodificador estricto.
  - REQ-29.3: RENAMED y MODIFIED; el nombre de todo requisito de la delta de un cambio ISO debe contener un identificador válido según el patrón declarado del proyecto. Las reglas de ambigüedad, repetición, tolerancia del archivado interrumpido y RENAMED se conservan.
  - REQ-29.4 y REQ-29.9: el identificador citado en `Cubre:` y en la cadena de trazabilidad es el de REQ-29.3, no la forma literal `REQ-<n>.<m>`.
- `spec-baseline-integration`: REQ-29.14, la consulta de un requisito existente resuelve por el identificador válido según el patrón.
- `axiom-sdd-cli-integration`: REQ-29.16, el motivo de identificador ausente, ambiguo o repetido se formula según el patrón declarado.
- `odd-living-spec-integration`: REQ-30.4, todo ADDED lleva un identificador válido según el patrón; la colisión y el «siguiente número libre» se calculan por la serie que define el patrón.
- `sdd-archive-lifecycle`: REQ-30.22, la identidad del requisito al componer es su ID cuando el título lo contiene y el texto exacto del nombre si no; se mantiene la fidelidad byte a byte de toda composición hoy aceptada.

Ninguna otra capacidad cambia de requisito: `living-documentation` (REQ-1.1) no fija el formato del ID, y `sdd-orchestrator-assets` (REQ-30.31) y `openspec-adoption` quedan para C-03-c.

## Enfoque

Enfoque recomendado por la exploración (opción B). La clave y la gramática exactas las fija el diseño (decisión delegada, README §6), no el usuario.

1. **Patrón como plantilla de segmentos.** Recomendación: clave `sdd.id_pattern` (escalar, en inglés como el resto de claves), con literales y tokens: `<DOM>` (código en mayúsculas), `<S>` (serie numérica) y `<N>`/`<NN>`/`<NNN>` (número con anchura mínima), con exactamente un token de número. Defecto `REQ-<S>.<N>`. La serie es todo lo que no es el número (`30` en `REQ-30.4`, `USU` en `RF-USU-01`, serie única en `RF-<N>`). La plantilla es invertible (permite formatear el siguiente ID), compila a una regex interna y da mensajes legibles; se descartan la regex declarada (no invertible, mensajes ilegibles) y el mapa estructurado (más superficie de esquema).
2. **Dirección de dependencias.** `internal/reqid` es hoja. `workspace` lo importa para validar al decodificar y expone el patrón efectivo (el defecto si no hay declaración). Los consumidores que ya tienen el workspace (`livespec`, `sddstatus`, `cli`, `adoptcheck`) lo obtienen de ahí; los que no (`openspecdoc`, `isoevidence`, `livingdoc`) lo reciben por parámetro u opción, con el defecto cuando no se pasa. Sin ciclos.
3. **Mensajes estables.** El texto del patrón por defecto se renderiza exactamente como `REQ-<n>.<m>`, de modo que los mensajes, continuaciones y pines actuales no cambian con el defecto.
4. **Compositor.** Emparejamiento por ID si ambos títulos lo tienen, por título si no. El diseño define el caso de un ID repetido en la canónica y la interacción con la tolerancia D17 del archivado interrumpido.
5. **Specs de este propio cambio.** Los cambios de título de requisitos vivos (REQ-29.3, REQ-30.4) se expresan con RENAMED + MODIFIED, para que el archivado funcione tanto con el compositor actual (por título) como con el nuevo (por ID).
6. **Verificación.** Solo con `go test`, sin compilar binarios sueltos (antivirus del entorno). La prueba de extremo a extremo del criterio de terminado usa un proyecto temporal con `RF-<DOM>-<NN>`: `axiom archive integrate --odd` y archivado SDD con `archive-compose`.
7. **Entrega.** Una sola PR (`single-pr`) con un commit por unidad de trabajo, en este orden orientativo: puerto y pruebas; clave en `axiom.yaml`, plantilla y contrato; `openspecdoc` y compositor; `isoevidence` y `sddstatus`; `livespec`, `adoptcheck` y CLI; `livingdoc`; guías y manual; sugerencias de P6; specs vivas.

## Discrepancias con el plan (hechos, sin resolver nada más)

- El plan cita `internal/components/agentguidance/routing.go:68,77` como afectado. En el árbol actual `routing.go` no contiene `REQ-<n>.<m>` (solo `REQ-27.x` en comentarios y prosa de carril).
- El plan prevé unos 89 goldens afectados. En `testdata/golden/` ningún fichero fija `REQ-<n>.<m>` ni `REQ-n.m`; si algún golden cambia, será solo como efecto de editar las guías.
- El plan cita `internal/assets/skills/_shared/openspec-convention.md`; tampoco contiene `REQ-<n>.<m>`.
- El plan habla de «tres sugerencias» del verify de P6 y enumera dos; el marcador ya está compartido y la interpretación de `Motivo:` ya convergió. Queda la constante de `Motivo:` y la prueba de paridad.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/reqid/` | Nuevo | Patrón, parseo, candidatos, validación, serie, número, formato y texto legible |
| `internal/workspace/sdd.go`, `declarations.go` | Modificado | Clave del patrón en `sdd:`, decodificación estricta, patrón efectivo |
| `internal/openspecdoc/parse.go`, `compose.go` | Modificado | Extracción del ID por el puerto; emparejamiento por ID |
| `internal/sddstatus/iso.go`, `openspec_archive_compose.go` | Modificado | Pasan el patrón del workspace |
| `internal/isoevidence/reqid.go`, `cubre.go`, `bug.go` | Modificado | Puerto en lugar de regex; constante `Motivo:` exportada |
| `internal/livespec/registry.go`, `plan.go`, `odddoc.go` | Modificado | Series por patrón; constante compartida; prueba de paridad |
| `internal/livingdoc/indexer.go` | Modificado | Lectura laxa por el puerto; ID sintético fuera del espacio válido |
| `internal/adoption/adoptcheck/check.go` | Modificado (mínimo) | Tipos de serie nuevos; reglas sin cambios |
| `internal/cli/archive_integrate.go`, `archive_adopt.go`, `sdd_trace.go` | Modificado | Mensajes y llamadas con el patrón |
| `internal/assets/skills/_shared/sdd-iso-convention.md`, `_shared/sdd-status-contract.md`, `sdd-spec/SKILL.md` | Modificado | Regla del ID según el patrón declarado |
| `docs/manual-de-inicio.md`, `axiom.example.yaml` y sus pruebas de contrato | Modificado | Documentación de la clave y del patrón |
| `openspec/changes/puerto-reqid/specs/{requirement-identifiers,sdd-iso-evidence,spec-baseline-integration,axiom-sdd-cli-integration,odd-living-spec-integration,sdd-archive-lifecycle}/spec.md` | Nuevo | Spec de la capacidad nueva y deltas |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| Los cambios de firma en `isoevidence` y `livespec` arrastran muchas pruebas y fixtures | Alta | Mantener el defecto byte a byte y un texto legible que renderice `REQ-<n>.<m>`; opciones con defecto en `openspecdoc` para no romper llamadores |
| El compositor por ID altera alguna composición aceptada hoy, o choca con la tolerancia D17 | Media | Prueba de no regresión de REQ-30.22/30.23; el diseño fija el caso de ID repetido en la canónica |
| El ID sintético de `livingdoc` (`REQ-<dominio>-<n>`) coincide con un patrón declarado | Media | Generarlo fuera del espacio de IDs válidos de cualquier patrón |
| Las tablas de resumen y glosario fuera de los bloques de requisito (`sdd-iso-evidence/spec.md:17,37`, `odd-living-spec-integration/spec.md:30`) no se editan con una delta y quedarían citando `REQ-<n>.<m>` | Media | El diseño decide si se ajustan al archivar o se dejan como descripción del defecto |
| Patrón leído de `axiom.yaml` en vivo frente a instantánea en `kickoff.yaml` (como `sdd.iso`, REQ-29.13): un cambio de patrón con cambios en curso alteraría su validación | Media | Decisión de diseño; elevar solo si cambia el alcance. C-03-e es el primer cambio que alterará el patrón de un proyecto |
| Colisión de la familia REQ-44 con otro cambio de la ola 6 (C-02-b) | Media | Confirmar en `main` el siguiente prefijo libre antes de archivar y renumerar si hace falta |
| Superficies compartidas: esquema de `axiom.yaml`, guías de agente, `livespec`/`openspecdoc`/`isoevidence` (C-03-b, C-06-e y otras olas) | Media | Cambios acotados por unidad de trabajo; C-03-b rebasa sobre este |
| La plantilla no expresa esquemas mixtos de cliente | Baja | Fuera de alcance; queda para C-03-b/C-03-c |
| Un golden cambia por efecto de editar las guías | Baja | Regenerar solo los afectados y revisarlos |

## Plan de reversión

- Revertir la PR única: desaparecen `internal/reqid`, la clave del patrón y el emparejamiento por ID; los consumidores vuelven a sus regex. Las specs delta se retiran con la reversión.
- Sin migración de datos: las specs de Axiom siguen en `REQ-n.m` y el defecto las admitía.
- Un `axiom.yaml` que ya declare el patrón dejaría de cargar con el binario revertido, porque `sdd:` se decodifica con lista cerrada de claves: hay que retirar la clave antes de revertir. Ningún proyecto la declara hoy.
- Las composiciones archivadas con el compositor nuevo son ficheros ordinarios y siguen siendo válidas.

## Dependencias

- No depende de ningún otro cambio de la sección 03. Bloquea a C-03-b, C-03-c, C-03-d y C-03-e. Arista con C-06-e, que registrará la clave del patrón en su catálogo, y con C-06-c, que debe trazar por este puerto.
- Sin dependencias externas nuevas.

## Criterios de éxito

- [ ] `rg 'REQ-' --type go internal cmd -g '!*_test.go'` solo coincide en `internal/reqid`, en el valor por defecto y en comentarios.
- [ ] Un proyecto temporal con patrón `RF-<DOM>-<NN>` integra una delta ODD (`axiom archive integrate --odd`) y archiva un cambio SDD (`archive-compose`) sin errores; la colisión propone el siguiente libre de su serie.
- [ ] El compositor aplica un MODIFIED cuyo título cambió pero conserva el ID; las composiciones hoy aceptadas siguen siendo idénticas byte a byte.
- [ ] Un patrón inválido o una clave desconocida en `sdd:` son error accionable en la carga y en `axiom workspace validate`.
- [ ] Sin declaración, todos los mensajes, continuaciones y pines actuales siguen igual.
- [ ] REQ-29.3 reescrito y REQ-29.1, 29.4, 29.9, 29.14, 29.16, 30.4 y 30.22 alineados.
- [ ] Constante compartida de `Motivo:` y prueba de paridad entre `isoevidence` y `livespec` en verde.
- [ ] `go test ./...` en verde, salvo el fallo conocido de `internal/components/filemerge` en Windows.

## Estimación de tamaño

L: unas 1.200-1.700 líneas de autoría con pruebas, más guías y specs. Una sola PR (`single-pr`) con un commit por unidad de trabajo.
