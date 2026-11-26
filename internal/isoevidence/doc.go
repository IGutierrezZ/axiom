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
//     motivo `blocked(<token>): <detalle>; <continuación>` (RF-CLI-11).
//   - fence.go: lectura de líneas con la marca de «dentro de una valla», que
//     comparten todos los analizadores de Markdown.
//   - reqid.go: identificadores de requisito (patrón del proyecto, por defecto `REQ-<n>.<m>`) en el nombre de los requisitos
//     (RF-ISO-03, con la tolerancia del archivado interrumpido).
//   - scenarios.go: escenarios por requisito (RF-ARC-09 y nombres repetidos de
//     RF-ISO-06).
//   - bug.go y cubre.go: perfil de bug sin delta, líneas `Cubre:` y cobertura
//     (RF-ISO-04, RF-ISO-05, RF-BAS-01, RF-CLI-10 y RF-ISO-10).
//   - matrix.go y testref.go: matriz de verificación y resolución de las
//     referencias de prueba contra el árbol (RF-ISO-06 a RF-ISO-08).
//   - trace.go: traza por `--first-parent` detrás de un puerto de historial
//     (RF-ISO-09).
package isoevidence
