// Package acceptancetests es el núcleo puro de la batería de pruebas de aceptación
// (`pruebas.md`): parsea su tabla (RF-KOF-18 y RF-KOF-19), la cruza con los requisitos y
// escenarios de las specs delta (RF-KOF-20) y reescribe los provisionales de `Cubre` al
// archivar (RF-KOF-23).
//
// El paquete no lee ficheros ni ejecuta órdenes: recibe el texto y el modelo ya leídos por
// el adaptador. Solo depende de internal/reqid y de internal/isoevidence (lector de tablas
// y de escenarios compartido con la matriz ISO).
package acceptancetests
