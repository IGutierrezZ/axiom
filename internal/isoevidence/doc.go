// Package isoevidence reúne las reglas del modo ISO del SDD (capacidad
// sdd-iso-evidence, REQ-29.x) como funciones puras y probadas por tablas.
//
// Es un paquete hoja: solo usa la biblioteca estándar y no importa ningún
// paquete interno ni `os/exec`. Los adaptadores (`internal/sddstatus` y
// `internal/cli`) leen los artefactos, construyen el modelo de este paquete y
// traducen los hallazgos a motivos de bloqueo; aquí no hay E/S de estado ni se
// ejecuta ningún comando.
//
// El paquete se organiza en capas:
//
//   - model.go: Section, Requirement, Stage, Code, Finding y la composición del
//     motivo `blocked(<token>): <detalle>; <continuación>` (REQ-29.16).
//   - fence.go: lectura de líneas con la marca de «dentro de una valla», que
//     comparten todos los analizadores de Markdown.
//   - reqid.go: identificadores `REQ-<n>.<m>` en el nombre de los requisitos
//     (REQ-29.3, con la tolerancia del archivado interrumpido).
//   - scenarios.go: escenarios por requisito (REQ-29.10 y nombres repetidos de
//     REQ-29.6).
//   - bug.go y cubre.go: perfil de bug sin delta, líneas `Cubre:` y cobertura
//     (REQ-29.4, REQ-29.5, REQ-29.14, REQ-29.15 y REQ-29.19).
//   - matrix.go y testref.go: matriz de verificación y resolución de las
//     referencias de prueba contra el árbol (REQ-29.6 a REQ-29.8).
//   - trace.go: traza por `--first-parent` detrás de un puerto de historial
//     (REQ-29.9).
package isoevidence
