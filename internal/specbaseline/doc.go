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
//   - compose.go: la composición de MODIFIED, REMOVED y RENAMED por
//     identificador, sin escribir nada: devuelve los bytes compuestos, la traza
//     y los avisos.
//
// Limitaciones documentadas: los encabezados setext no se reconocen (R33), y la
// detección de vallas de código (unas 40 líneas) duplica la del motor nativo
// para no tocar código compartido con upstream (D-08); la unificación queda
// para un cambio posterior.
package specbaseline
