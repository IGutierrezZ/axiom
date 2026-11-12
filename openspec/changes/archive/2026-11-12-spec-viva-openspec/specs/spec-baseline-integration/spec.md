# Delta para spec-baseline-integration

> Capacidad retirada entera (D5, R1): la integración con perfil de requisito, la edición dirigida de prosa y los identificadores provisionales solo servían a documentos de baseline ajena. Se reubican las reglas que sobreviven: composición nativa, fidelidad de bytes y validación completa en `sdd-archive-lifecycle`; edición dirigida, verificación nativa y traza, restringidas al contexto técnico, en `project-technical-context`. Al archivar, la carpeta `openspec/specs/spec-baseline-integration/` se retira.
> Nota de coordinación con `carril-sdd-versatil` (REQ-29.14 sin destino en este cambio): su delta añade REQ-29.14 (consulta de un requisito existente) y modificaba REQ-24.38 y REQ-24.46 en esta capacidad (su primera enmienda retiró el MODIFIED de REQ-24.46). REQ-29.14 NO tiene destino aquí: su segunda enmienda (R5) la retira y este cambio no redacta ningún sucesor ni la lista entre los REMOVED. REQ-24.38 y REQ-24.46 se reubican en `sdd-archive-lifecycle` (REQ-30.22 y REQ-30.24).
> Supuesto: REQ-24.59 (sustitución de identificadores provisionales) desaparece sin sucesor, porque en OpenSpec el autor elige el `REQ-n.m` definitivo (R2).

## REMOVED Requirements

### Requirement: Composición Nativa Intacta sin Perfil (REQ-24.38)

(Reason: sin perfil de requisito la composición nativa es la única.)
(Motivo: sin perfil de requisito la composición nativa es la única.)
(Migración: REQ-30.22 de `sdd-archive-lifecycle`, que además retira los modos `--document` e `--in-place` de `archive-compose`.)

### Requirement: Reconocimiento de Requisitos e Identidad según el Perfil (REQ-24.39)

(Reason: el perfil de requisito se retira con la baseline ajena.)
(Motivo: el perfil de requisito se retira con la baseline ajena.)
(Migración: la identidad del requisito OpenSpec es el texto exacto de su nombre, que contiene `REQ-n.m`.)

### Requirement: Rechazo de Operaciones sobre IDs Duplicados (REQ-24.40)

(Reason: dependía del perfil de requisito.)
(Motivo: dependía del perfil de requisito.)
(Migración: la unicidad de `REQ-n.m` se comprueba en REQ-30.4 de `odd-living-spec-integration` y en REQ-30.11 de `openspec-adoption`.)

### Requirement: Frontera de Bloque por Nivel y Secciones No Requisito Contiguas (REQ-24.41)

(Reason: describía la frontera de bloque de documentos ajenos con encabezados de nivel variable.)
(Motivo: describía la frontera de bloque de documentos ajenos con encabezados de nivel variable.)
(Migración: ninguna; los requisitos OpenSpec usan `### Requirement:`.)

### Requirement: Formato del Delta, Render por Plantilla y Cuerpo Íntegro sin Encabezados que Cierren el Bloque (REQ-24.42)

(Reason: el render por plantilla era propio del perfil de requisito.)
(Motivo: el render por plantilla era propio del perfil de requisito.)
(Migración: ninguna; el formato de la delta OpenSpec es el nativo, y el cuerpo vallado de REQ-30.1 evita que los encabezados cierren la sección ODD.)

### Requirement: Requisitos ADDED con Ubicación Explícita e ID Definitivo Asignado en Archive (REQ-24.43)

(Reason: los identificadores provisionales y la asignación del definitivo en archive se retiran; el autor elige el `REQ-n.m` (R2).)
(Motivo: los identificadores provisionales y la asignación del definitivo en archive se retiran; el autor elige el `REQ-n.m` (R2).)
(Migración: REQ-30.4 de `odd-living-spec-integration` y REQ-30.12 de `openspec-adoption`.)

### Requirement: MODIFIED, REMOVED y RENAMED con Identidad por ID (REQ-24.44)

(Reason: la identidad por ID era del perfil de requisito.)
(Motivo: la identidad por ID era del perfil de requisito.)
(Migración: la composición nativa empareja por el texto exacto del nombre del requisito (REQ-30.22 de `sdd-archive-lifecycle`).)

### Requirement: Fidelidad de Bytes al Componer (REQ-24.45)

(Reason: la regla sigue vigente; solo cambia de capacidad.)
(Motivo: la regla sigue vigente; solo cambia de capacidad.)
(Migración: REQ-30.23 de `sdd-archive-lifecycle`.)

### Requirement: Validación Completa antes de Escribir (REQ-24.46)

(Reason: la regla sigue vigente; solo cambia de capacidad y se reformula sin dominios de prosa.)
(Motivo: la regla sigue vigente; solo cambia de capacidad y se reformula sin dominios de prosa.)
(Migración: REQ-30.24 de `sdd-archive-lifecycle`.)

### Requirement: Resolución de Documento y Perfil en `axiom sdd archive-compose` (REQ-24.47)

(Reason: la resolución de documento y perfil por declaración desaparece con `--document` e `--in-place`.)
(Motivo: la resolución de documento y perfil por declaración desaparece con `--document` e `--in-place`.)
(Migración: REQ-30.22 de `sdd-archive-lifecycle`; `archive-compose` solo admite `--canonical --delta`.)

### Requirement: Un Dominio No Declarado se Rechaza (REQ-24.48)

(Reason: sin dominios declarados en una baseline no hay dominios no declarados.)
(Motivo: sin dominios declarados en una baseline no hay dominios no declarados.)
(Migración: ninguna; una capacidad nueva se crea con ADDED (REQ-30.1 de `odd-living-spec-integration`).)

### Requirement: Precondiciones de la Integración (REQ-24.49)

(Reason: la parte de baseline desaparece; la parte de contexto mantenido sigue vigente.)
(Motivo: la parte de baseline desaparece; la parte de contexto mantenido sigue vigente.)
(Migración: REQ-30.29 de `project-technical-context`.)

### Requirement: Integración Dirigida de Documentos de Prosa (REQ-24.50)

(Reason: los documentos de prosa de baseline ajena se retiran; la edición dirigida sobrevive solo para el contexto.)
(Motivo: los documentos de prosa de baseline ajena se retiran; la edición dirigida sobrevive solo para el contexto.)
(Migración: REQ-30.16 de `project-technical-context`.)

### Requirement: Verificación Nativa de la Integración Dirigida (REQ-24.51)

(Reason: la verificación de documentos de baseline y la sustitución de identificadores provisionales desaparecen; la del contexto sigue vigente.)
(Motivo: la verificación de documentos de baseline y la sustitución de identificadores provisionales desaparecen; la del contexto sigue vigente.)
(Migración: REQ-30.17 de `project-technical-context`.)

### Requirement: Traza de Integración por Entrada (REQ-24.52)

(Reason: la traza de requisitos con perfil y del mapeo de identificadores provisionales desaparece; la del contexto sigue vigente.)
(Motivo: la traza de requisitos con perfil y del mapeo de identificadores provisionales desaparece; la del contexto sigue vigente.)
(Migración: REQ-30.18 de `project-technical-context`.)

### Requirement: Modo Solo Registro para Documentos de Prosa (REQ-24.53)

(Reason: el modo `register` era propio de la baseline ajena.)
(Motivo: el modo `register` era propio de la baseline ajena.)
(Migración: ninguna.)

### Requirement: Lectura de la Baseline por las Fases de Planificación (REQ-24.54)

(Reason: las fases leen la spec viva `openspec/specs/` sin localizador de baseline.)
(Motivo: las fases leen la spec viva `openspec/specs/` sin localizador de baseline.)
(Migración: REQ-30.28 de `sdd-orchestrator-assets`; la lectura del contexto técnico sigue en REQ-24.29 de `project-technical-context`.)

### Requirement: Sustitución de los IDs Provisionales por los Definitivos en Todo el Texto Integrado (REQ-24.59)

(Reason: los identificadores provisionales se retiran; el autor elige el `REQ-n.m` definitivo (R2).)
(Motivo: los identificadores provisionales se retiran; el autor elige el `REQ-n.m` definitivo (R2).)
(Migración: ninguna; REQ-24.55 de `sdd-archive-lifecycle` ya no los menciona.)

### Requirement: Escenarios de la Delta Spec en el Bloque del Proyecto: `include` y `omit` (REQ-24.61)

(Reason: dependía del perfil de requisito de la baseline ajena.)
(Motivo: dependía del perfil de requisito de la baseline ajena.)
(Migración: ninguna; los escenarios OpenSpec se componen con el requisito, y `adopt check` exige al menos uno por requisito migrado (REQ-30.11).)
