# Delta para spec-baseline-declaration

> Capacidad retirada entera: la decisión D5 fija OpenSpec como único formato de spec viva y R1 retira la maquinaria de baseline ajena (`baseline:`, `internal/specbaseline`, `axiom sdd integrate`), que ningún proyecto real usa. Los requisitos que siguen vigentes se reubican (en `project-technical-context`, `workspace-topology`, `sdd-archive-lifecycle` o `living-documentation`); los demás desaparecen sin sucesor. Al archivar, la carpeta `openspec/specs/spec-baseline-declaration/` se retira.
> Supuesto sobre REQ-24.23 y REQ-24.25: se retiran sin sucesor porque no hay baseline externa que proteger; su contrapartida es el rechazo tipado de REQ-30.21.

## REMOVED Requirements

### Requirement: Bloque `baseline:` Opcional y Defecto OpenSpec (REQ-24.13)

(Reason: el bloque `baseline:` se retira por D5; OpenSpec es el único formato de spec viva.)
(Motivo: el bloque `baseline:` se retira por D5; OpenSpec es el único formato de spec viva.)
(Migración: un `axiom.yaml` con `baseline:` falla cerrado con el error tipado de REQ-30.21 de `workspace-topology`; los requisitos previos se migran con `axiom archive adopt` (REQ-30.9 y REQ-30.11 de `openspec-adoption`).)

### Requirement: Documentos por Dominio y Mapeo Dominio a Documento (REQ-24.14)

(Reason: sin baseline del proyecto no hay documentos por dominio; la spec viva es `openspec/specs/<capacidad>/spec.md`.)
(Motivo: sin baseline del proyecto no hay documentos por dominio; la spec viva es `openspec/specs/<capacidad>/spec.md`.)
(Migración: ninguna; el mapeo capacidad a fichero de OpenSpec es el implícito.)

### Requirement: Perfil de Requisito (REQ-24.15)

(Reason: el perfil de requisito describía formatos de documento ajenos; OpenSpec fija el formato del requisito y su identidad `REQ-n.m`.)
(Motivo: el perfil de requisito describía formatos de documento ajenos; OpenSpec fija el formato del requisito y su identidad `REQ-n.m`.)
(Migración: los identificadores originales de los requisitos migrados se conservan en `Origen:` (REQ-30.10 de `openspec-adoption`).)

### Requirement: Documentos de Prosa y Modo de Integración (REQ-24.16)

(Reason: los modos `compose`, `directed` y `register` solo existían para documentos de baseline ajena.)
(Motivo: los modos `compose`, `directed` y `register` solo existían para documentos de baseline ajena.)
(Migración: la edición dirigida sobrevive únicamente para el contexto técnico mantenido (REQ-30.16 de `project-technical-context`).)

### Requirement: Validación de la Declaración por el Loader y por `axiom workspace validate` (REQ-24.17)

(Reason: el bloque `baseline:` ya no se valida, se rechaza.)
(Motivo: el bloque `baseline:` ya no se valida, se rechaza.)
(Migración: REQ-30.21 de `workspace-topology` rechaza el bloque en el loader y en `axiom workspace validate`; la validación de `technical_context` es REQ-24.27 de `project-technical-context`.)

### Requirement: Una Declaración Inválida Falla Cerrada (REQ-24.18)

(Reason: la parte de baseline desaparece con el bloque; la parte de `technical_context` sigue vigente.)
(Motivo: la parte de baseline desaparece con el bloque; la parte de `technical_context` sigue vigente.)
(Migración: REQ-30.30 de `project-technical-context` conserva el fallo cerrado de un `technical_context:` inválido y REQ-30.21 de `workspace-topology` el de un `baseline:` residual.)

### Requirement: Proyección de `baseline` en el Status v2 (REQ-24.19)

(Reason: el status deja de proyectar `baseline`.)
(Motivo: el status deja de proyectar `baseline`.)
(Migración: el status solo proyecta `technicalContext` (REQ-24.28 de `project-technical-context`); un `baseline:` residual se proyecta como causa de bloqueo del archivado (REQ-30.21).)

### Requirement: Contrato del Status v2 Solo Aditivo (REQ-24.20)

(Reason: la regla protegía la incorporación de `baseline` y `technicalContext`; la retirada de `baseline` no introduce campos nuevos obligatorios.)
(Motivo: la regla protegía la incorporación de `baseline` y `technicalContext`; la retirada de `baseline` no introduce campos nuevos obligatorios.)
(Migración: la regla aditiva de `technicalContext` vive en REQ-24.28 de `project-technical-context`; las referencias a REQ-24.20 de `carril-sdd-versatil` se actualizan a REQ-24.28.)

### Requirement: Frontera de Configuración entre `axiom.yaml` y `openspec/config.yaml` (REQ-24.21)

(Reason: la frontera sigue vigente pero sin la mención de `baseline`.)
(Motivo: la frontera sigue vigente pero sin la mención de `baseline`.)
(Migración: REQ-30.32 de `project-technical-context`.)

### Requirement: Consulta Única de Baseline del Proyecto (REQ-24.22)

(Reason: sin baseline del proyecto no hay nada que consultar; el planning home es siempre el de OpenSpec.)
(Motivo: sin baseline del proyecto no hay nada que consultar; el planning home es siempre el de OpenSpec.)
(Migración: ninguna; los consumidores usan el primitivo de `planning-home-resolution`.)

### Requirement: Guarda en las Operaciones de la CLI, el Dashboard y la TUI (REQ-24.23)

(Reason: las guardas protegían `openspec/specs/` e `INDEX.md` frente a una baseline ajena; con la baseline retirada no hay segunda spec viva que evitar.)
(Motivo: las guardas protegían `openspec/specs/` e `INDEX.md` frente a una baseline ajena; con la baseline retirada no hay segunda spec viva que evitar.)
(Migración: el rechazo tipado de REQ-30.21 de `workspace-topology` impide que un `baseline:` residual haga escribir una segunda spec viva; `living-documentation` y `multi-project-hub` retiran sus ramas de baseline.)

### Requirement: Guarda en las Skills Creadoras de Baseline (REQ-24.24)

(Reason: las skills creadoras de baseline ajena no se distribuyen sin baseline.)
(Motivo: las skills creadoras de baseline ajena no se distribuyen sin baseline.)
(Migración: REQ-30.28 de `sdd-orchestrator-assets` exige skills sin ramas de baseline.)

### Requirement: Las Lecturas de Catálogo Informan de que la Baseline es Externa (REQ-24.25)

(Reason: no existe baseline externa que informar.)
(Motivo: no existe baseline externa que informar.)
(Migración: REQ-1.1 y REQ-4.1 de `living-documentation` quedan sin el informe de baseline externa.)

### Requirement: Clave `scenarios` por Documento con Perfil de Requisito (REQ-24.60)

(Reason: dependía del perfil de requisito de la baseline ajena.)
(Motivo: dependía del perfil de requisito de la baseline ajena.)
(Migración: ninguna; en OpenSpec el escenario es `#### Scenario:` y `adopt check` exige al menos uno por requisito migrado (REQ-30.11).)

### Requirement: Baseline del Proyecto y Contexto Mantenido Exigen un Almacén de Artefactos de Ficheros (REQ-24.62)

(Reason: la parte de baseline desaparece con el bloque; la parte de contexto mantenido sigue vigente.)
(Motivo: la parte de baseline desaparece con el bloque; la parte de contexto mantenido sigue vigente.)
(Migración: REQ-30.19 de `project-technical-context`.)
