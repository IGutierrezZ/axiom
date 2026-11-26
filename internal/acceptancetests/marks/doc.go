// Package marks es el núcleo puro de las marcas de ejecución por entorno de `pruebas.md`
// (C-06-d): el formato de `pruebas-estado.yaml`, la codificación segura de una marca, su
// validación contra la batería, el resumen por entorno, las entradas pendientes de publicar y
// la fusión de la reproducción. No lee ficheros, no ejecuta git y no conoce la red: quien lo
// usa (status, dashboard y publicador) aporta los bytes y la batería ya leídos.
//
// El fichero es una secuencia YAML de mapas, sin cabecera, de solo añadir por concatenación:
// el contenido previo es siempre prefijo exacto del nuevo.
package marks
