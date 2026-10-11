// Package specbaseline compone las delta specs de un cambio sobre los
// documentos de la baseline del proyecto (inc-24, capacidad
// spec-baseline-integration).
//
// Es un paquete solo del fork: la composición nativa de OpenSpec
// (sddstatus.ComposeOpenSpecCanonicalSpecWithOptions) no se modifica y sigue
// siendo el motor de la baseline por defecto, de modo que su comportamiento se
// conserva por construcción (R1, D-08). Este paquete añade el segundo motor, el
// que reconoce los requisitos de un documento mediante el perfil declarado en
// `axiom.yaml` (`baseline.documents[].requirement`).
//
// El paquete se organiza en capas:
//
//   - document.go: modelo del documento por líneas, con la marca de orden de
//     bytes y el final de línea de cada línea (D-12), encabezados ATX fuera de
//     vallas, bloques de requisito y cola contigua de secciones (D-09).
//   - profile.go: el perfil de requisito compilado (patrón `heading`, nivel,
//     identidad, plantilla `render` y tratamiento de escenarios).
//   - delta.go: lectura del delta en formato nativo (D-10).
//   - compose.go: la composición de ADDED, MODIFIED, REMOVED y RENAMED por
//     identificador, con la ubicación explícita de los ADDED, los escenarios
//     `include` u `omit` y la sustitución de IDs provisionales, sin escribir
//     nada: devuelve los bytes compuestos, la traza y los avisos.
//   - ids.go: la correspondencia de IDs provisionales a definitivos de un
//     cambio entero (PlanIDs) y su sustitución por tokens completos (D-11).
//   - contextdelta.go y anchors.go: el delta de contexto `context-impact.md`,
//     su gramática determinista y las anclas por ruta de encabezados (D-16).
//   - gitview.go: la lectura de solo lectura de git, con el repositorio elegido
//     por la ruta del documento y nunca por el proceso (D-15, design §8).
//   - plan.go y trace.go: el plan `integration.yaml`, las huellas del árbol de
//     trabajo, la foto acotada de la suciedad previa y `Check` y `Prepare`, que
//     validan y componen todo el cambio en memoria antes de escribir nada
//     (D-13, REQ-24.46).
//
// Limitaciones documentadas: los encabezados setext no se reconocen (R33), y la
// detección de vallas de código (unas 40 líneas) duplica la del motor nativo
// para no tocar código compartido con upstream (D-08); la unificación queda
// para un cambio posterior.
package specbaseline
