// Package reqtrace responde, sin escribir nada, qué incrementos (cambios SDD archivados y
// documentos ODD integrados) crearon, modificaron, retiraron, renombraron o cubrieron un
// requisito (C-03-d, `axiom trace <ID>`).
//
// Es un paquete de dominio sin E/S propia: lee ficheros a través de io/fs.FS (de solo
// lectura por definición) y consulta git solo a través del puerto isoevidence.History. No
// importa os, os/exec ni internal/gitcommon: la solo-lectura del verbo (prov:trace:10) queda
// fijada por arquitectura (imports_test.go) y no por la disciplina de quien lo modifica.
package reqtrace

// Schema identifica el documento JSON de `axiom trace --json`. El esquema es versionado:
// añadir un campo opcional es compatible; quitar o cambiar uno exige `axiom.trace/v2`.
const Schema = "axiom.trace/v1"
